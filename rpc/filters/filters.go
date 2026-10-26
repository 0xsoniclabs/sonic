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

// Package filters serves logs, polled filters and subscriptions, under the
// eth namespace and its ftm mirror.
package filters

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/format"
)

const (
	// maxFilters bounds the polled filters of the node.
	maxFilters = 100000
	// journalLimit bounds the events journaled per kind.
	journalLimit = 1 << 20
	// headsQueue is the queue of a newHeads subscription, in blocks.
	headsQueue = 1024
)

// kind is the kind of a polled filter.
type kind int

const (
	blocksKind kind = iota
	txsKind
	logsKind
)

var kindNames = [...]string{blocksKind: "block_filters", txsKind: "pending_tx_filters", logsKind: "log_filters"}

var errFilterNotFound = errors.New("filter not found")

// filter is one polled filter. It holds the position in its kind's journal up
// to which its client polled; its criteria are applied when it polls.
type filter struct {
	kind     kind
	pos      uint64
	lastPoll time.Time
	crit     FilterCriteria // logs
	silent   bool           // a log filter of pending logs, which never arrive
	fullTx   bool           // pending transactions as whole transactions
}

// System holds the polled filters and the feeds of the subscriptions.
type System struct {
	env  *exec.Env
	logs core.LogIndex
	d    *dispatch.Dispatcher

	mu      sync.Mutex
	filters map[string]*filter

	blocks *journal[common.Hash]
	txs    *journal[*types.Transaction]
	events *journal[*types.Log]

	// heads carries the newHeads notifications, each built once per block
	// and shared by every subscription.
	heads *chain.Hub[json.RawMessage]

	headRuns, poolRuns, newHeadsRuns atomic.Int64
}

// Register adds the filter methods and subscriptions under each namespace;
// one system serves all of them, so a filter installed under one is polled
// under another. The log index serves the indexed log queries.
func Register(d *dispatch.Dispatcher, env *exec.Env, logs core.LogIndex, namespaces ...string) *System {
	s := &System{
		env:     env,
		logs:    logs,
		d:       d,
		filters: make(map[string]*filter),
		blocks:  newJournal[common.Hash](journalLimit),
		txs:     newJournal[*types.Transaction](journalLimit),
		events:  newJournal[*types.Log](journalLimit),
		heads:   chain.NewHub[json.RawMessage](headsQueue),
	}
	for _, ns := range namespaces {
		add := func(name string, m dispatch.Method) { d.Add(ns+"_"+name, m) }
		add("getLogs", dispatch.Func(s.getLogs))
		add("newFilter", dispatch.Func(s.newFilter))
		add("newBlockFilter", dispatch.Func(s.newBlockFilter))
		add("newPendingTransactionFilter", dispatch.Func(s.newPendingTransactionFilter))
		add("getFilterChanges", dispatch.Func(s.getFilterChanges))
		add("getFilterLogs", dispatch.Func(s.getFilterLogs))
		add("uninstallFilter", dispatch.Func(s.uninstallFilter))
		d.AddSubscription(ns, "newHeads", dispatch.Func(func(ctx context.Context, _ dispatch.None) (any, error) {
			return s.subscribeNewHeads(ctx, ns)
		}))
		d.AddSubscription(ns, "logs", dispatch.Func(func(ctx context.Context, p criteriaArg) (any, error) {
			return s.subscribeLogs(ctx, ns, p.Crit)
		}))
		d.AddSubscription(ns, "newPendingTransactions", dispatch.Func(func(ctx context.Context, p pendingFilterArg) (any, error) {
			return s.subscribePendingTransactions(ctx, ns, p.FullTx != nil && *p.FullTx)
		}))
	}
	d.Loop("filters head", s.headLoop)
	d.Loop("filters pool", s.poolLoop)
	d.Loop("filters newHeads", s.newHeadsLoop)
	d.Loop("filters expiry", s.expiryLoop)
	return s
}

type criteriaArg struct {
	Crit FilterCriteria `rpc:"crit"`
}

type idArg struct {
	ID string `rpc:"id"`
}

func (s *System) getLogs(ctx context.Context, p criteriaArg) (any, error) {
	return s.queryLogs(ctx, p.Crit)
}

// install adds a filter unless the node holds the most it may. The filter
// starts at the end of its journal, under the lock that trims journals.
func (s *System) install(f *filter) (string, error) {
	id := dispatch.NewID()
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.filters) >= maxFilters {
		return "", fmt.Errorf("too many filters, the limit is %d", maxFilters)
	}
	switch f.kind {
	case blocksKind:
		f.pos = s.blocks.end()
	case txsKind:
		f.pos = s.txs.end()
	case logsKind:
		f.pos = s.events.end()
	}
	f.lastPoll = time.Now()
	s.filters[id] = f
	s.d.Metrics().Subscriptions(kindNames[f.kind]).Inc()
	return id, nil
}

func (s *System) newFilter(ctx context.Context, p criteriaArg) (any, error) {
	if err := checkParameterLimit(s.env.Config().LogQueryParameterLimit, p.Crit.Addresses, p.Crit.Topics); err != nil {
		return nil, err
	}
	r, err := classifyRange(p.Crit)
	if err != nil {
		return nil, err
	}
	return s.install(&filter{kind: logsKind, crit: p.Crit, silent: r == pendingLogs})
}

func (s *System) newBlockFilter(context.Context, dispatch.None) (any, error) {
	return s.install(&filter{kind: blocksKind})
}

type pendingFilterArg struct {
	FullTx *bool `rpc:"fullTx"`
}

func (s *System) newPendingTransactionFilter(ctx context.Context, p pendingFilterArg) (any, error) {
	return s.install(&filter{kind: txsKind, fullTx: p.FullTx != nil && *p.FullTx})
}

func (s *System) uninstallFilter(ctx context.Context, p idArg) (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, found := s.filters[p.ID]
	if found {
		s.removeLocked(p.ID, f)
	}
	return found, nil
}

func (s *System) removeLocked(id string, f *filter) {
	delete(s.filters, id)
	s.d.Metrics().Subscriptions(kindNames[f.kind]).Dec()
}

// poll takes the position of a filter and reads what its journal holds after
// it, under the lock; a filter whose events were dropped is removed. The
// filter's criteria are applied by the caller, outside the lock.
func (s *System) poll(id string) (*filter, []common.Hash, []*types.Transaction, []*types.Log, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, found := s.filters[id]
	if !found {
		return nil, nil, nil, nil, errFilterNotFound
	}
	var (
		hashes []common.Hash
		txs    []*types.Transaction
		logs   []*types.Log
		next   uint64
		ok     bool
	)
	switch f.kind {
	case blocksKind:
		hashes, next, ok = s.blocks.since(f.pos)
	case txsKind:
		txs, next, ok = s.txs.since(f.pos)
	case logsKind:
		logs, next, ok = s.events.since(f.pos)
	}
	if !ok {
		s.removeLocked(id, f)
		return nil, nil, nil, nil, errFilterNotFound
	}
	f.pos = next
	f.lastPoll = time.Now()
	return f, hashes, txs, logs, nil
}

// getFilterChanges answers what happened since the filter's last poll: block
// hashes, pending transactions (null when there are none, as the reference
// answered) or logs.
func (s *System) getFilterChanges(ctx context.Context, p idArg) (any, error) {
	f, hashes, txs, logs, err := s.poll(p.ID)
	if err != nil {
		return []any{}, err
	}
	switch f.kind {
	case blocksKind:
		return format.Hashes(hashes), nil
	case txsKind:
		var result []any
		chainID := s.env.ChainID()
		for _, tx := range txs {
			if f.fullTx {
				result = append(result, format.NewPendingTransaction(tx, tx.GasPrice(), chainID))
			} else {
				result = append(result, tx.Hash())
			}
		}
		return result, nil
	}
	if f.silent {
		return []*types.Log{}, nil
	}
	return format.Logs(filterLogs(logs, f.crit.FromBlock, f.crit.ToBlock, f.crit.Addresses, f.crit.Topics)), nil
}

// getFilterLogs runs the query of a log filter again.
func (s *System) getFilterLogs(ctx context.Context, p idArg) (any, error) {
	s.mu.Lock()
	f, found := s.filters[p.ID]
	var crit FilterCriteria
	if found {
		crit = f.crit
	}
	s.mu.Unlock()
	if !found || f.kind != logsKind {
		return nil, errFilterNotFound
	}
	return s.queryLogs(ctx, crit)
}

// dropKind removes every filter of the kinds: their feed lost events.
func (s *System) dropKind(kinds ...kind) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, f := range s.filters {
		for _, k := range kinds {
			if f.kind == k {
				s.removeLocked(id, f)
			}
		}
	}
}

// headLoop journals the block hashes and logs of new blocks. A restarted
// run, and a run whose feed dropped it, lost events: the filters of these
// kinds are removed, and new ones get every event after.
func (s *System) headLoop(ctx context.Context) {
	if s.headRuns.Add(1) > 1 {
		s.dropKind(blocksKind, logsKind)
	}
	for {
		heads := s.env.Chain().Heads().Subscribe()
		dropped := s.relayHeads(ctx, heads)
		heads.Close()
		if !dropped {
			return
		}
		log.Warn("Polled block and log filters fell behind the head feed; removing them")
		s.dropKind(blocksKind, logsKind)
	}
}

func (s *System) relayHeads(ctx context.Context, heads *chain.Consumer[core.HeadEvent]) bool {
	for {
		select {
		case ev := <-heads.Events():
			s.blocks.add(ev.Block.Hash)
			if len(ev.Logs) > 0 {
				s.events.add(ev.Logs...)
			}
		case <-heads.Dropped():
			return true
		case <-ctx.Done():
			return false
		}
	}
}

// poolLoop journals the transactions entering the pool, as headLoop the
// blocks.
func (s *System) poolLoop(ctx context.Context) {
	if s.poolRuns.Add(1) > 1 {
		s.dropKind(txsKind)
	}
	for {
		txs := s.env.Chain().Txs().Subscribe()
		dropped := func() bool {
			defer txs.Close()
			for {
				select {
				case batch := <-txs.Events():
					s.txs.add(batch...)
				case <-txs.Dropped():
					return true
				case <-ctx.Done():
					return false
				}
			}
		}()
		if !dropped {
			return
		}
		log.Warn("Polled pending transaction filters fell behind the pool feed; removing them")
		s.dropKind(txsKind)
	}
}

// expiryLoop removes the filters not polled within the filter timeout, and
// trims every journal to what a live filter has not polled.
func (s *System) expiryLoop(ctx context.Context) {
	timeout := s.env.Config().FilterTimeout
	ticker := time.NewTicker(max(timeout/10, time.Second))
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.expire(time.Now(), timeout)
		case <-ctx.Done():
			return
		}
	}
}

// expire removes the filters not polled since now-timeout and trims the
// journals.
func (s *System) expire(now time.Time, timeout time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	oldest := [3]uint64{}
	live := [3]bool{}
	for id, f := range s.filters {
		if now.Sub(f.lastPoll) > timeout {
			s.removeLocked(id, f)
			continue
		}
		if !live[f.kind] || f.pos < oldest[f.kind] {
			oldest[f.kind] = f.pos
		}
		live[f.kind] = true
	}
	trim := func(k kind, end uint64) uint64 {
		if live[k] {
			return oldest[k]
		}
		return end
	}
	s.blocks.trim(trim(blocksKind, s.blocks.end()))
	s.txs.trim(trim(txsKind, s.txs.end()))
	s.events.trim(trim(logsKind, s.events.end()))
}

// newHeadsLoop builds the newHeads notification of each block once: the
// block's receipts and its encoded header, shared by every subscription. A
// restarted run, and a run the feed dropped, drop the subscriptions.
func (s *System) newHeadsLoop(ctx context.Context) {
	if s.newHeadsRuns.Add(1) > 1 {
		s.heads.DropAll()
	}
	for {
		heads := s.env.Chain().Heads().Subscribe()
		dropped := func() bool {
			defer heads.Close()
			for {
				select {
				case ev := <-heads.Events():
					if s.heads.Len() == 0 {
						continue
					}
					head := s.env.View(ctx, &chain.Head{Block: ev.Block})
					receipts, err := s.env.ReceiptsAt(head, head.Number())
					if err != nil && !exec.IsMissing(err) {
						log.Warn("Receipts of a new head not readable", "block", head.Number(), "err", err)
					}
					enc, err := json.Marshal(ev.Block.ToJson(receipts))
					if err != nil {
						log.Error("New head not encodable", "block", head.Number(), "err", err)
						continue
					}
					s.heads.Publish(enc)
				case <-heads.Dropped():
					return true
				case <-ctx.Done():
					return false
				}
			}
		}()
		if !dropped {
			return
		}
		s.heads.DropAll()
	}
}

func notifier(ctx context.Context) (dispatch.Notifier, error) {
	n, ok := dispatch.NotifierFrom(ctx)
	if !ok {
		return nil, dispatch.ErrNotificationsUnsupported
	}
	return n, nil
}

// relay installs a consumer of a hub for a new subscription before the id is
// returned, and delivers what it receives until the subscription or its
// connection ends. A consumer the hub dropped ends the subscription and its
// connection, so that the client learns of the gap.
func relay[T any](ctx context.Context, ns, kind string, hub *chain.Hub[T], deliver func(dispatch.Subscription, T)) (any, error) {
	n, err := notifier(ctx)
	if err != nil {
		return nil, err
	}
	sub, err := n.Subscribe(ns, kind)
	if err != nil {
		return nil, err
	}
	consumer := hub.Subscribe()
	n.Go(func(ctx context.Context) {
		defer consumer.Close()
		for {
			select {
			case ev := <-consumer.Events():
				deliver(sub, ev)
			case <-consumer.Dropped():
				sub.Drop(fmt.Sprintf("%s subscription %s fell behind its feed", kind, sub.ID()))
				return
			case <-sub.Done():
				return
			case <-ctx.Done():
				return
			}
		}
	})
	return sub.ID(), nil
}

func (s *System) subscribeNewHeads(ctx context.Context, ns string) (any, error) {
	return relay(ctx, ns, "newHeads", s.heads, func(sub dispatch.Subscription, header json.RawMessage) {
		sub.Notify(header)
	})
}

func (s *System) subscribeLogs(ctx context.Context, ns string, crit FilterCriteria) (any, error) {
	if err := checkParameterLimit(s.env.Config().LogQueryParameterLimit, crit.Addresses, crit.Topics); err != nil {
		return nil, err
	}
	r, err := classifyRange(crit)
	if err != nil {
		return nil, err
	}
	return relay(ctx, ns, "logs", s.env.Chain().Heads(), func(sub dispatch.Subscription, ev core.HeadEvent) {
		if r == pendingLogs {
			return
		}
		for _, l := range filterLogs(ev.Logs, crit.FromBlock, crit.ToBlock, crit.Addresses, crit.Topics) {
			sub.Notify(l)
		}
	})
}

func (s *System) subscribePendingTransactions(ctx context.Context, ns string, fullTx bool) (any, error) {
	chainID := s.env.ChainID()
	return relay(ctx, ns, "newPendingTransactions", s.env.Chain().Txs(), func(sub dispatch.Subscription, txs []*types.Transaction) {
		for _, tx := range txs {
			if fullTx {
				sub.Notify(format.NewPendingTransaction(tx, tx.GasPrice(), chainID))
			} else {
				sub.Notify(tx.Hash())
			}
		}
	})
}
