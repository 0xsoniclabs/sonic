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
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
)

const (
	// maxFilters bounds the polled filters of the node (I17).
	maxFilters = 100_000
	// journalLimit bounds the events journaled of each kind (I17).
	journalLimit = 1 << 20
	// sweepsPerTimeout is how often within a filter timeout the filters not
	// polled for that long are removed.
	sweepsPerTimeout = 5
)

var (
	errFilterNotFound = errors.New("filter not found")
	errTooManyFilters = fmt.Errorf("too many filters, the limit is %d", maxFilters)
)

// kind is what a polled filter reports.
type kind int

const (
	blocksKind kind = iota
	transactionsKind
	logsKind
	kindCount
)

// filter is one polled filter: its criteria and the position up to which
// its client polled.
type filter struct {
	kind             kind
	criteria         Criteria
	receivesLogs     bool
	fullTransactions bool
	position         uint64
	deadline         time.Time
}

// polledFilters are the polled filters of the node and the journals of
// their events. The filters are guarded by the mutex, which no method holds
// while it calls the node (P4); each journal has its own.
type polledFilters struct {
	timeout      time.Duration
	hashes       *journal[common.Hash]
	logs         *journal[*types.Log]
	transactions *journal[*types.Transaction]

	mutex   sync.Mutex
	filters map[string]*filter
	counts  [kindCount]int
}

func newPolledFilters(timeout time.Duration, limit int) *polledFilters {
	return &polledFilters{
		timeout:      timeout,
		hashes:       newJournal[common.Hash](limit),
		logs:         newJournal[*types.Log](limit),
		transactions: newJournal[*types.Transaction](limit),
		filters:      map[string]*filter{},
	}
}

func (polled *polledFilters) end(kind kind) uint64 {
	switch kind {
	case blocksKind:
		return polled.hashes.end()
	case transactionsKind:
		return polled.transactions.end()
	}
	return polled.logs.end()
}

// install adds a filter that reports the events after the current ones.
func (polled *polledFilters) install(added *filter) (string, error) {
	polled.mutex.Lock()
	defer polled.mutex.Unlock()
	if len(polled.filters) >= maxFilters {
		return "", errTooManyFilters
	}
	added.position = polled.end(added.kind)
	added.deadline = time.Now().Add(polled.timeout)
	identifier := dispatch.NewID()
	polled.filters[identifier] = added
	polled.counts[added.kind]++
	return identifier, nil
}

// uninstall removes a filter and tells whether there was one.
func (polled *polledFilters) uninstall(identifier string) bool {
	polled.mutex.Lock()
	defer polled.mutex.Unlock()
	return polled.remove(identifier)
}

func (polled *polledFilters) remove(identifier string) bool {
	removed, found := polled.filters[identifier]
	if found {
		delete(polled.filters, identifier)
		polled.counts[removed.kind]--
	}
	return found
}

// find returns a copy of a filter.
func (polled *polledFilters) find(identifier string) (filter, bool) {
	polled.mutex.Lock()
	defer polled.mutex.Unlock()
	found, ok := polled.filters[identifier]
	if !ok {
		return filter{}, false
	}
	return *found, true
}

// changes is what a poll takes out of the journal of its filter's kind.
type changes struct {
	filter       filter
	hashes       []common.Hash
	logs         []*types.Log
	transactions []*types.Transaction
}

// poll takes the events of a filter since its last poll and moves it on. A
// filter whose events left the journal is removed, as if it had expired.
func (polled *polledFilters) poll(identifier string) (changes, error) {
	polled.mutex.Lock()
	defer polled.mutex.Unlock()
	found, ok := polled.filters[identifier]
	if !ok {
		return changes{}, errFilterNotFound
	}
	taken := changes{filter: *found}
	var end uint64
	var lost bool
	switch found.kind {
	case blocksKind:
		taken.hashes, end, lost = polled.hashes.since(found.position)
	case transactionsKind:
		taken.transactions, end, lost = polled.transactions.since(found.position)
	default:
		taken.logs, end, lost = polled.logs.since(found.position)
	}
	if lost {
		polled.remove(identifier)
		return changes{}, errFilterNotFound
	}
	found.position = end
	found.deadline = time.Now().Add(polled.timeout)
	return taken, nil
}

// has tells whether any filter of a kind is installed, so that a feed
// journals nothing for no one.
func (polled *polledFilters) has(kind kind) bool {
	polled.mutex.Lock()
	defer polled.mutex.Unlock()
	return polled.counts[kind] > 0
}

// dropKinds removes every filter of the kinds, whose feed lost events (I18).
func (polled *polledFilters) dropKinds(kinds ...kind) {
	polled.mutex.Lock()
	defer polled.mutex.Unlock()
	for identifier, installed := range polled.filters {
		for _, dropped := range kinds {
			if installed.kind == dropped {
				polled.remove(identifier)
			}
		}
	}
}

// sweep removes the filters not polled within the timeout and drops the
// events every remaining filter has polled.
func (polled *polledFilters) sweep(now time.Time) {
	polled.mutex.Lock()
	defer polled.mutex.Unlock()
	oldest := [kindCount]uint64{polled.hashes.end(), polled.transactions.end(), polled.logs.end()}
	for identifier, installed := range polled.filters {
		if now.After(installed.deadline) {
			polled.remove(identifier)
			continue
		}
		oldest[installed.kind] = min(oldest[installed.kind], installed.position)
	}
	polled.hashes.trim(oldest[blocksKind])
	polled.transactions.trim(oldest[transactionsKind])
	polled.logs.trim(oldest[logsKind])
}

// followHeads journals the block hashes and logs of the head feed while a
// filter of their kind is installed. Its first consumer is taken when it is
// made. A feed that dropped its consumer
// removes the filters of both kinds and is taken again.
func (polled *polledFilters) followHeads(heads *chain.Hub[core.HeadEvent]) func(ctx context.Context) error {
	next := heads.Subscribe()
	return func(ctx context.Context) error {
		consumer := takeConsumer(&next, heads)
		defer consumer.Close()
		for {
			select {
			case <-ctx.Done():
				return nil
			case event, open := <-consumer.Events():
				if !open {
					polled.dropKinds(blocksKind, logsKind)
					return fmt.Errorf("head feed of the polled filters dropped: %w", consumer.Err())
				}
				if polled.has(blocksKind) {
					polled.hashes.append(event.Block.Hash)
				}
				if polled.has(logsKind) {
					polled.logs.append(event.Logs...)
				}
			}
		}
	}
}

// followPool journals the transactions entering the pool while a filter of
// them is installed.
func (polled *polledFilters) followPool(pool *chain.Hub[evmcore.NewTxsNotify]) func(ctx context.Context) error {
	next := pool.Subscribe()
	return func(ctx context.Context) error {
		consumer := takeConsumer(&next, pool)
		defer consumer.Close()
		for {
			select {
			case <-ctx.Done():
				return nil
			case event, open := <-consumer.Events():
				if !open {
					polled.dropKinds(transactionsKind)
					return fmt.Errorf("pool feed of the polled filters dropped: %w", consumer.Err())
				}
				if polled.has(transactionsKind) {
					polled.transactions.append(event.Txs...)
				}
			}
		}
	}
}

// takeConsumer returns the consumer taken when the loop was made, so that
// no event after the registration is missed, and a new one on a restart.
func takeConsumer[Event any](next **chain.Consumer[Event], hub *chain.Hub[Event]) *chain.Consumer[Event] {
	consumer := *next
	*next = nil
	if consumer == nil {
		consumer = hub.Subscribe()
	}
	return consumer
}

// expire sweeps the filters several times within a timeout.
func (polled *polledFilters) expire(ctx context.Context) error {
	ticker := time.NewTicker(polled.timeout / sweepsPerTimeout)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case now := <-ticker.C:
			polled.sweep(now)
		}
	}
}

// newFilter installs a log filter. The bounds take the combinations the
// reference took; a filter of pending logs, which the reference installed
// but never fed, reports nothing.
func (handler *handlers) newFilter(_ context.Context, arguments dispatch.Arguments) (any, error) {
	criteria := dispatch.Argument[Criteria](arguments, 0)
	if err := checkParameterLimit(handler.environment.Config().LogQueryParameterLimit, criteria); err != nil {
		return nil, err
	}
	receives, err := receivesLogs(criteria)
	if err != nil {
		return nil, err
	}
	return handler.polled.install(&filter{kind: logsKind, criteria: criteria, receivesLogs: receives})
}

// receivesLogs checks the bounds of a log filter or subscription, as the
// reference's SubscribeLogs did, and tells whether mined logs reach it.
func receivesLogs(criteria Criteria) (bool, error) {
	from, until := bound(criteria.FromBlock), bound(criteria.ToBlock)
	latest, pending := int64(chain.LatestBlockNumber), int64(chain.PendingBlockNumber)
	switch {
	case from == pending && until == pending:
		return false, nil
	case from == latest && until == latest:
		return true, nil
	case from >= 0 && until >= 0 && until >= from:
		return true, nil
	case from >= latest && until == pending:
		return false, nil
	case from >= 0 && until == latest:
		return true, nil
	}
	return false, errors.New("invalid from and to block combination: from > to")
}

func (handler *handlers) newBlockFilter(context.Context, dispatch.Arguments) (any, error) {
	return handler.polled.install(&filter{kind: blocksKind})
}

func (handler *handlers) newPendingTransactionFilter(_ context.Context, arguments dispatch.Arguments) (any, error) {
	full := dispatch.ArgumentOr(arguments, 0, false)
	return handler.polled.install(&filter{kind: transactionsKind, fullTransactions: full})
}

func (handler *handlers) uninstallFilter(_ context.Context, arguments dispatch.Arguments) (any, error) {
	return handler.polled.uninstall(dispatch.Argument[string](arguments, 0)), nil
}

// getFilterChanges answers what happened since the filter's last poll, with
// the filter's criteria applied now, outside any lock (I17).
func (handler *handlers) getFilterChanges(_ context.Context, arguments dispatch.Arguments) (any, error) {
	taken, err := handler.polled.poll(dispatch.Argument[string](arguments, 0))
	if err != nil {
		return nil, err
	}
	switch taken.filter.kind {
	case blocksKind:
		if taken.hashes == nil {
			return []common.Hash{}, nil
		}
		return taken.hashes, nil
	case transactionsKind:
		return handler.pendingTransactions(taken.transactions, taken.filter.fullTransactions), nil
	}
	return matchingLogs(taken.logs, taken.filter), nil
}

// pendingTransactions answers the hashes of transactions, or their pool
// forms, each priced at its own gas price (F7). No transaction is null, as
// in the reference.
func (handler *handlers) pendingTransactions(transactions []*types.Transaction, full bool) []any {
	var answered []any
	chainID := handler.environment.ChainID()
	for _, transaction := range transactions {
		answered = append(answered, pendingForm(transaction, full, chainID))
	}
	return answered
}

// matchingLogs answers copies of the logs the filter's criteria match, with
// no null topics (P8).
func matchingLogs(logs []*types.Log, polled filter) []*types.Log {
	answered := []*types.Log{}
	if !polled.receivesLogs {
		return answered
	}
	for _, entry := range logs {
		if polled.criteria.matches(entry) {
			copied := *entry
			if copied.Topics == nil {
				copied.Topics = []common.Hash{}
			}
			answered = append(answered, &copied)
		}
	}
	return answered
}

// getFilterLogs answers the logs of a log filter's criteria, as eth_getLogs.
func (handler *handlers) getFilterLogs(ctx context.Context, arguments dispatch.Arguments) (any, error) {
	found, ok := handler.polled.find(dispatch.Argument[string](arguments, 0))
	if !ok || found.kind != logsKind {
		return nil, errFilterNotFound
	}
	head, err := handler.environment.Latest(ctx)
	if err != nil {
		return nil, err
	}
	return handler.findLogs(head, found.criteria)
}
