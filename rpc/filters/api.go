// Copyright 2026 Sonic Operations Ltd
// This file is part of the Sonic Client
//
// Sonic is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Sonic is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Sonic. If not, see <http://www.gnu.org/licenses/>.

package filters

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/eth"
	"github.com/0xsoniclabs/sonic/rpc/exec"
)

const (
	// feedQueueSize is the queue of the consumers that fill the polled filters.
	feedQueueSize = 4096
	// subscriptionQueueSize is the queue of a consumer behind one subscription.
	subscriptionQueueSize = 256
	// headsCacheSize is the number of newHeads notifications kept for the
	// subscriptions that are behind.
	headsCacheSize = 16
	// maxFilters bounds the polled filters of the node. Any client may install
	// one, and a filter lives until it is not polled for FilterTimeout.
	maxFilters = 100_000
	// journalLimit bounds the events kept for the polled filters of one kind. A
	// filter whose events were dropped before its client polled them is
	// removed, as if it had expired.
	journalLimit = 1 << 20
)

type filterKind int

const (
	logsFilter filterKind = iota
	blocksFilter
	pendingTxsFilter
)

// filter is a polled filter. The events are kept once, in the journal of the
// filter's kind; the filter remembers up to where its client has them, and its
// criteria are applied when the client polls. So an event costs the same
// whatever the number of filters.
type filter struct {
	kind     filterKind
	crit     Criteria
	mode     logsMode
	fullTxs  bool
	lastPoll time.Time
	cursor   uint64 // position in the journal of the kind
}

// API serves the filter and subscription methods of the eth_ namespace. One
// instance serves eth_ and ftm_ alike: a filter installed through one is
// polled through the other.
type API struct {
	env   *exec.Env
	chain *dispatch.Chain
	core  core.Core
	cfg   dispatch.Config

	mu         sync.Mutex
	filters    map[string]*filter
	counts     [3]int // the filters of each kind
	blocks     journal[common.Hash]
	logs       journal[*types.Log]
	txs        journal[*types.Transaction]
	maxFilters int

	headsMu sync.Mutex
	heads   *lru.Cache[common.Hash, json.RawMessage] // newHeads notifications by block hash
}

// New creates the API.
func New(env *exec.Env) *API {
	return &API{
		env: env, chain: env.Chain, core: env.Core, cfg: env.Cfg, filters: map[string]*filter{},
		blocks:     newJournal[common.Hash](journalLimit),
		logs:       newJournal[*types.Log](journalLimit),
		txs:        newJournal[*types.Transaction](journalLimit),
		maxFilters: maxFilters,
		heads:      lru.NewCache[common.Hash, json.RawMessage](headsCacheSize),
	}
}

// Register registers the methods under the namespace "eth" or "ftm".
func (a *API) Register(d *dispatch.Dispatcher, ns string) {
	m := func(name string) string { return ns + "_" + name }
	dispatch.Add1(d, m("newFilter"), a.newFilter, "criteria")
	dispatch.Add0(d, m("newBlockFilter"), a.newBlockFilter)
	dispatch.Add1(d, m("newPendingTransactionFilter"), a.newPendingTransactionFilter, "fullTx?")
	dispatch.Add1(d, m("getFilterChanges"), a.getFilterChanges, "id")
	dispatch.Add1(d, m("getFilterLogs"), a.getFilterLogs, "id")
	dispatch.Add1(d, m("uninstallFilter"), a.uninstallFilter, "id")
	dispatch.Add1(d, m("getLogs"), a.getLogs, "criteria")

	dispatch.AddSub0(d, ns, "newHeads", a.newHeads)
	dispatch.AddSub1(d, ns, "logs", a.logsSubscription, "criteria")
	dispatch.AddSub1(d, ns, "newPendingTransactions", a.newPendingTransactions, "fullTx?")
}

// Start starts the loops that feed and expire the polled filters. The feeds
// are followed from here on: a filter installed after Start returned misses no
// event. A feed loop restarted after a panic follows them again.
func (a *API) Start(d *dispatch.Dispatcher) {
	heads := a.chain.SubscribeHeads(feedQueueSize)
	txs := a.chain.SubscribeTxs(feedQueueSize)
	d.Loop("filters-feed", func(ctx context.Context) {
		if heads == nil {
			heads = a.chain.SubscribeHeads(feedQueueSize)
			txs = a.chain.SubscribeTxs(feedQueueSize)
		}
		defer func() {
			heads.Close()
			txs.Close()
			heads, txs = nil, nil
		}()
		a.feed(ctx, &heads, &txs)
	})
	d.Loop("filters-expiry", a.expire)
}

// -- feeding the polled filters --

// feed hands the events of the feeds to the journals. It subscribes again to a
// feed it fell behind.
func (a *API) feed(ctx context.Context, heads **dispatch.HeadConsumer, txs **dispatch.Consumer[evmcore.NewTxsNotify]) {
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-(*heads).Events():
			if !ok {
				log.Warn("RPC filters fell behind the head; some events were lost")
				*heads = a.chain.SubscribeHeads(feedQueueSize)
				continue
			}
			a.onHead(ev)
		case ev, ok := <-(*txs).Events():
			if !ok {
				log.Warn("RPC filters fell behind the pending transactions; some events were lost")
				*txs = a.chain.SubscribeTxs(feedQueueSize)
				continue
			}
			a.onTxs(ev)
		}
	}
}

// onHead journals the hash and the logs of a new head, if a filter wants them.
func (a *API) onHead(ev core.HeadEvent) {
	if ev.Block == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.counts[blocksFilter] > 0 {
		a.blocks.append(ev.Block.Hash)
	}
	if a.counts[logsFilter] > 0 && len(ev.Logs) > 0 {
		a.logs.append(ev.Logs...)
	}
}

// onTxs journals the transactions entering the pool, if a filter wants them.
func (a *API) onTxs(ev evmcore.NewTxsNotify) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.counts[pendingTxsFilter] > 0 {
		a.txs.append(ev.Txs...)
	}
}

// end returns the position after the newest event of the journal of a kind.
func (a *API) end(kind filterKind) uint64 {
	switch kind {
	case blocksFilter:
		return a.blocks.end()
	case pendingTxsFilter:
		return a.txs.end()
	}
	return a.logs.end()
}

// remove removes a filter. The caller holds the lock.
func (a *API) remove(id string, f *filter) {
	delete(a.filters, id)
	a.counts[f.kind]--
}

// expire drops the filters that were not polled within the timeout.
func (a *API) expire(ctx context.Context) {
	timeout := a.cfg.FilterTimeout
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	ticker := time.NewTicker(max(timeout/30, 10*time.Millisecond))
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			a.removeExpired(now, timeout)
		}
	}
}

// removeExpired removes the filters that were not polled within the timeout,
// and drops from each journal the events every remaining filter has polled.
func (a *API) removeExpired(now time.Time, timeout time.Duration) {
	a.mu.Lock()
	defer a.mu.Unlock()
	polled := [3]uint64{logsFilter: a.logs.end(), blocksFilter: a.blocks.end(), pendingTxsFilter: a.txs.end()}
	for id, f := range a.filters {
		if now.Sub(f.lastPoll) > timeout {
			a.remove(id, f)
			continue
		}
		polled[f.kind] = min(polled[f.kind], f.cursor)
	}
	a.logs.trim(polled[logsFilter])
	a.blocks.trim(polled[blocksFilter])
	a.txs.trim(polled[pendingTxsFilter])
}

// install adds a filter and returns its id. It receives the events from here
// on.
func (a *API) install(f *filter) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.filters) >= a.maxFilters {
		return "", fmt.Errorf("too many filters, the limit is %d", a.maxFilters)
	}
	f.lastPoll = time.Now()
	f.cursor = a.end(f.kind)
	id := dispatch.NewFilterID()
	a.filters[id] = f
	a.counts[f.kind]++
	return id, nil
}

// logsMode tells which logs a log filter or subscription receives.
type logsMode int

const (
	minedLogs logsMode = iota
	// pendingLogs and minedAndPendingLogs are accepted but never deliver: the
	// node has no pending logs.
	pendingLogs
	minedAndPendingLogs
)

// modeOf classifies the block range of a log filter. A range that is neither
// mined nor pending is refused.
func modeOf(crit Criteria) (logsMode, error) {
	var from, to dispatch.BlockNumber
	if crit.FromBlock == nil {
		from = dispatch.LatestBlockNumber
	} else {
		from = dispatch.BlockNumber(crit.FromBlock.Int64())
	}
	if crit.ToBlock == nil {
		to = dispatch.LatestBlockNumber
	} else {
		to = dispatch.BlockNumber(crit.ToBlock.Int64())
	}

	// only interested in pending logs
	if from == dispatch.PendingBlockNumber && to == dispatch.PendingBlockNumber {
		return pendingLogs, nil
	}
	// only interested in new mined logs
	if from == dispatch.LatestBlockNumber && to == dispatch.LatestBlockNumber {
		return minedLogs, nil
	}
	// only interested in mined logs within a specific block range
	if from >= 0 && to >= 0 && to >= from {
		return minedLogs, nil
	}
	// interested in mined logs from a specific block number, new logs and pending logs
	if from >= dispatch.LatestBlockNumber && to == dispatch.PendingBlockNumber {
		return minedAndPendingLogs, nil
	}
	// interested in logs from a specific block number to new mined blocks
	if from >= 0 && to == dispatch.LatestBlockNumber {
		return minedLogs, nil
	}
	return 0, errors.New("invalid from and to block combination: from > to")
}

// -- polled filters --

func (a *API) newFilter(_ context.Context, crit Criteria) (string, error) {
	if err := checkParameterLimit(a.cfg, crit.Addresses, crit.Topics); err != nil {
		return "", err
	}
	mode, err := modeOf(crit)
	if err != nil {
		return "", err
	}
	return a.install(&filter{kind: logsFilter, crit: crit, mode: mode})
}

// newBlockFilter creates a filter that collects the hashes of the new blocks.
func (a *API) newBlockFilter(context.Context) (string, error) {
	return a.install(&filter{kind: blocksFilter})
}

// newPendingTransactionFilter creates a filter that collects the transactions
// entering the pool, as hashes or, if fullTx is set, in full.
func (a *API) newPendingTransactionFilter(_ context.Context, fullTxs *bool) (string, error) {
	return a.install(&filter{kind: pendingTxsFilter, fullTxs: fullTxs != nil && *fullTxs})
}

// uninstallFilter removes the filter with the id.
func (a *API) uninstallFilter(_ context.Context, id string) (bool, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	f, found := a.filters[id]
	if found {
		a.remove(id, f)
	}
	return found, nil
}

// getFilterLogs returns the logs of the filter. It is an error if the filter is
// not there or is not a log filter.
func (a *API) getFilterLogs(ctx context.Context, id string) ([]*types.Log, error) {
	a.mu.Lock()
	f, found := a.filters[id]
	var crit Criteria
	if found {
		crit = f.crit
	}
	a.mu.Unlock()

	if !found || f.kind != logsFilter {
		return nil, errFilterNotFound
	}
	q, err := a.newQuery(crit)
	if err != nil {
		return nil, err
	}
	logs, err := q.Logs(ctx)
	if err != nil {
		return nil, err
	}
	return returnLogs(logs), nil
}

// getFilterChanges returns what happened since the filter was last polled.
// Pending transaction and block filters answer hashes (or transactions), log
// filters logs. A filter whose events were dropped from the journal before
// this poll is removed and not found, as if it had expired.
func (a *API) getFilterChanges(_ context.Context, id string) (any, error) {
	a.mu.Lock()
	f, found := a.filters[id]
	if !found {
		a.mu.Unlock()
		return []any{}, errFilterNotFound
	}
	f.lastPoll = time.Now()
	var (
		hashes []common.Hash
		txs    []*types.Transaction
		logs   []*types.Log
		kept   bool
	)
	switch f.kind {
	case blocksFilter:
		hashes, kept = a.blocks.since(f.cursor)
	case pendingTxsFilter:
		txs, kept = a.txs.since(f.cursor)
	case logsFilter:
		logs, kept = a.logs.since(f.cursor)
	}
	f.cursor = a.end(f.kind)
	if !kept {
		a.remove(id, f)
	}
	kind, crit, mode, fullTxs := f.kind, f.crit, f.mode, f.fullTxs
	a.mu.Unlock()

	// The events are read without the lock: the journal never writes them again.
	if !kept {
		return []any{}, errFilterNotFound
	}
	switch kind {
	case blocksFilter:
		return returnHashes(hashes), nil
	case pendingTxsFilter:
		return a.pendingTransactionChanges(txs, fullTxs), nil
	}
	if mode != minedLogs {
		return returnLogs(nil), nil
	}
	return returnLogs(filterLogs(logs, crit.FromBlock, crit.ToBlock, crit.Addresses, crit.Topics)), nil
}

var errFilterNotFound = errors.New("filter not found")

// pendingTransactionChanges serves the transactions a filter receives. The
// whole transactions carry no block fields and are priced by their own gas
// price. With nothing new the answer is null, as it always was.
func (a *API) pendingTransactionChanges(txs []*types.Transaction, fullTxs bool) any {
	var result []any
	chainID := new(big.Int).SetUint64(a.core.ChainInfo().ChainID)
	for _, tx := range txs {
		if fullTxs {
			result = append(result, eth.NewRPCPendingTransaction(tx, tx.GasPrice(), chainID))
		} else {
			result = append(result, tx.Hash())
		}
	}
	return result
}

// getLogs returns the logs matching the criteria that are stored in the chain.
func (a *API) getLogs(ctx context.Context, crit Criteria) ([]*types.Log, error) {
	q, err := a.newQuery(crit)
	if err != nil {
		return nil, err
	}
	logs, err := q.Logs(ctx)
	if err != nil {
		return nil, err
	}
	return returnLogs(logs), nil
}

// -- subscriptions --

// begin creates a subscription on the connection of the request.
func begin(ctx context.Context, kind string) (*dispatch.Notifier, *dispatch.Subscription, error) {
	notifier, ok := dispatch.NotifierFrom(ctx)
	if !ok {
		return nil, nil, dispatch.ErrNotificationsUnsupported
	}
	sub, err := notifier.NewSubscription(ctx, kind)
	if err != nil {
		return nil, nil, err
	}
	return notifier, sub, nil
}

// run starts the producer of a subscription. The feeds were installed by the
// caller before: the producer only forwards.
func run(notifier *dispatch.Notifier, sub *dispatch.Subscription, producer func()) (string, error) {
	if !notifier.Go(func() {
		defer sub.Close()
		producer()
	}) {
		sub.Close()
		return "", errors.New("server is shutting down")
	}
	return sub.ID(), nil
}

// fellBehind ends the connection of a client whose subscription cannot keep up.
func fellBehind(notifier *dispatch.Notifier, what string) {
	notifier.Abort(fmt.Errorf("subscription to %s fell behind", what))
}

// newHeads notifies the header of each new block.
func (a *API) newHeads(ctx context.Context) (string, error) {
	notifier, sub, err := begin(ctx, "newHeads")
	if err != nil {
		return "", err
	}
	// Installed before returning, so no block between the request and the
	// response of the subscription is missed.
	heads := a.chain.SubscribeHeads(subscriptionQueueSize)
	id, err := run(notifier, sub, func() {
		defer heads.Close()
		for {
			select {
			case <-sub.Done():
				return
			case ev, ok := <-heads.Events():
				if !ok {
					fellBehind(notifier, "new heads")
					return
				}
				head, err := a.newHeadsJSON(ev.Block)
				if err != nil {
					return
				}
				if err := sub.Notify(head); err != nil {
					return
				}
			}
		}
	})
	if err != nil {
		heads.Close()
	}
	return id, err
}

// newHeadsJSON returns the newHeads notification of a block. It is encoded
// once per block for all subscriptions.
func (a *API) newHeadsJSON(block *evmcore.EvmBlock) (json.RawMessage, error) {
	a.headsMu.Lock()
	defer a.headsMu.Unlock()
	if head, ok := a.heads.Get(block.Hash); ok {
		return head, nil
	}
	receipts, rerr := a.core.Receipts(block.NumberU64())
	head, err := json.Marshal(block.ToJson(receipts))
	if err != nil {
		return nil, err
	}
	if rerr == nil {
		// A header without its receipts is not kept: the next subscription
		// tries again.
		a.heads.Add(block.Hash, head)
	}
	return head, nil
}

// logsSubscription notifies the logs that match the criteria.
func (a *API) logsSubscription(ctx context.Context, crit Criteria) (string, error) {
	notifier, ok := dispatch.NotifierFrom(ctx)
	if !ok {
		return "", dispatch.ErrNotificationsUnsupported
	}
	if err := checkParameterLimit(a.cfg, crit.Addresses, crit.Topics); err != nil {
		return "", err
	}
	mode, err := modeOf(crit)
	if err != nil {
		return "", err
	}
	_, sub, err := begin(ctx, "logs")
	if err != nil {
		return "", err
	}
	if mode != minedLogs {
		// Pending logs are accepted but never delivered.
		return run(notifier, sub, func() { <-sub.Done() })
	}
	heads := a.chain.SubscribeHeads(subscriptionQueueSize)
	id, err := run(notifier, sub, func() {
		defer heads.Close()
		for {
			select {
			case <-sub.Done():
				return
			case ev, ok := <-heads.Events():
				if !ok {
					fellBehind(notifier, "logs")
					return
				}
				if len(ev.Logs) == 0 {
					continue
				}
				for _, l := range filterLogs(ev.Logs, crit.FromBlock, crit.ToBlock, crit.Addresses, crit.Topics) {
					if err := sub.Notify(l); err != nil {
						return
					}
				}
			}
		}
	})
	if err != nil {
		heads.Close()
	}
	return id, err
}

// newPendingTransactions notifies each transaction entering the pool: its hash,
// or the whole transaction if fullTx is set.
func (a *API) newPendingTransactions(ctx context.Context, fullTx *bool) (string, error) {
	notifier, sub, err := begin(ctx, "newPendingTransactions")
	if err != nil {
		return "", err
	}
	txs := a.chain.SubscribeTxs(subscriptionQueueSize)
	chainID := new(big.Int).SetUint64(a.core.ChainInfo().ChainID)
	id, err := run(notifier, sub, func() {
		defer txs.Close()
		for {
			select {
			case <-sub.Done():
				return
			case ev, ok := <-txs.Events():
				if !ok {
					fellBehind(notifier, "pending transactions")
					return
				}
				// To keep the original behaviour, a single transaction is sent
				// in one notification.
				for _, tx := range ev.Txs {
					var payload any = tx.Hash()
					if fullTx != nil && *fullTx {
						payload = eth.NewRPCPendingTransaction(tx, tx.GasPrice(), chainID)
					}
					if err := sub.Notify(payload); err != nil {
						return
					}
				}
			}
		}
	})
	if err != nil {
		txs.Close()
	}
	return id, err
}
