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
	"time"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/format"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// feedBuffer is how many events of the chain the polled filters may trail.
const feedBuffer = 1024

// sweepsPerTimeout is how often per filter timeout the expired filters are
// removed, so that a filter lives at most a fifth longer than its timeout.
const sweepsPerTimeout = 5

// newFilter installs a filter of the logs of new blocks that the criteria
// select.
func (h *handlers) newFilter(_ context.Context, params logsParams) (dispatch.ID, error) {
	if err := checkParameterLimit(h.environment.Config().LogQueryParameterLimit, params.Criteria); err != nil {
		return "", err
	}
	mined, err := params.Criteria.receivesMined()
	if err != nil {
		return "", err
	}
	return h.filters.install(&filter{kind: logsKind, criteria: params.Criteria, mined: mined})
}

// newBlockFilter installs a filter of the hashes of new blocks.
func (h *handlers) newBlockFilter(context.Context, struct{}) (dispatch.ID, error) {
	return h.filters.install(&filter{kind: blocksKind})
}

type fullParams struct {
	Full *bool
}

// newPendingTransactionFilter installs a filter of the transactions that
// enter the pool, as hashes or in full.
func (h *handlers) newPendingTransactionFilter(_ context.Context, params fullParams) (dispatch.ID, error) {
	return h.filters.install(&filter{kind: transactionsKind, full: params.Full != nil && *params.Full})
}

type idParams struct {
	ID dispatch.ID
}

// getFilterChanges answers the events a filter received since it last
// polled: block hashes, logs, or transactions, the latter null if none, as
// the legacy handler answers them.
func (h *handlers) getFilterChanges(_ context.Context, params idParams) (any, error) {
	found, events, err := h.filters.poll(params.ID)
	if err != nil {
		return nil, err
	}
	switch found.kind {
	case blocksKind:
		if events.blocks == nil {
			return []common.Hash{}, nil
		}
		return events.blocks, nil
	case logsKind:
		if !found.mined {
			return answered(nil), nil
		}
		return answered(found.criteria.matching(events.logs)), nil
	}
	var transactions []any
	for _, transaction := range events.transactions {
		if found.full {
			transactions = append(transactions, format.NewPendingTransaction(transaction, h.environment.ChainID()))
		} else {
			transactions = append(transactions, transaction.Hash())
		}
	}
	return transactions, nil
}

// getFilterLogs answers the logs of the served blocks that the criteria of
// a logs filter select.
func (h *handlers) getFilterLogs(ctx context.Context, params idParams) ([]*types.Log, error) {
	found, err := h.filters.get(params.ID)
	if err != nil || found.kind != logsKind {
		return nil, errFilterNotFound
	}
	return h.getLogs(ctx, logsParams{Criteria: found.criteria})
}

// uninstallFilter removes a filter and tells whether there was one.
func (h *handlers) uninstallFilter(_ context.Context, params idParams) (bool, error) {
	return h.filters.uninstall(params.ID), nil
}

// startJournals subscribes to the served heads and the pool's transactions
// at once, so that no event that comes after the registration is missed, and
// journals them in loops of the dispatcher.
func (h *handlers) startJournals(dispatcher *dispatch.Dispatcher) {
	heads := subscriber(h.environment.SubscribeHeads)
	dispatcher.Loop("polled filters of heads", func(ctx context.Context) {
		follow(ctx, heads, func(event core.HeadNotify) {
			h.filters.addBlock(event.Block.Hash, event.Logs)
		}, func() { h.filters.drop(blocksKind, logsKind) })
	})
	transactions := subscriber(h.environment.SubscribeTransactions)
	dispatcher.Loop("polled filters of the pool", func(ctx context.Context) {
		follow(ctx, transactions, func(event evmcore.NewTxsNotify) {
			h.filters.addTransactions(event.Txs)
		}, func() { h.filters.drop(transactionsKind) })
	})
}

// subscriber subscribes a consumer at once, which it answers first, and new
// ones after. Only one loop calls it.
func subscriber[Event any](subscribe func(buffer int) *chain.Consumer[Event]) func() *chain.Consumer[Event] {
	first := subscribe(feedBuffer)
	return func() *chain.Consumer[Event] {
		if consumer := first; consumer != nil {
			first = nil
			return consumer
		}
		return subscribe(feedBuffer)
	}
}

// follow hands the events of a consumer to take until the context ends. A
// consumer that ends would leave a gap, so the filters it fed are dropped
// and a new consumer is subscribed.
func follow[Event any](ctx context.Context, subscribe func() *chain.Consumer[Event], take func(Event), drop func()) {
	for ctx.Err() == nil {
		followOne(ctx, subscribe(), take, drop)
	}
}

func followOne[Event any](ctx context.Context, consumer *chain.Consumer[Event], take func(Event), drop func()) {
	defer drop()
	defer consumer.Unsubscribe()
	for {
		select {
		case <-ctx.Done():
			return
		case <-consumer.Ended():
			log.Warn("RPC polled filters lost events of their feed and are dropped", "err", consumer.Err())
			return
		case event := <-consumer.Events():
			take(event)
		}
	}
}

// expireFilters removes the expired filters until the context ends.
func (h *handlers) expireFilters(ctx context.Context) {
	ticker := time.NewTicker(h.environment.Config().FilterTimeout / sweepsPerTimeout)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.filters.expire()
		}
	}
}
