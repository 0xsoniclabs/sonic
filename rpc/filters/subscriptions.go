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

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/format"
)

// subscriptionBuffer is how many events a subscription may trail its feed
// before it is dropped with its connection.
const subscriptionBuffer = 128

// registerSubscriptions adds the kinds of subscriptions of the eth
// namespace, and starts the loop that forms each new head once for all
// newHeads subscriptions.
func (h *handlers) registerSubscriptions(dispatcher *dispatch.Dispatcher) {
	dispatch.AddSubscription(dispatcher, "eth", "newHeads", h.newHeads)
	dispatch.AddSubscription(dispatcher, "eth", "logs", h.logs)
	dispatch.AddSubscription(dispatcher, "eth", "newPendingTransactions", h.newPendingTransactions)
	heads := subscriber(h.environment.SubscribeHeads)
	dispatcher.Loop("head forms", func(ctx context.Context) {
		follow(ctx, heads, h.formHead, func() { h.headForms.DropAll(chain.ErrFeedLost) })
	})
}

// formHead forms a head as newHeads notifications carry it, with the
// receipt fields of the block, when a subscription takes it.
func (h *handlers) formHead(event core.HeadNotify) {
	if h.headForms.Consumers() == 0 {
		return
	}
	receipts, _ := h.environment.ReceiptsAt(h.environment.Head(), event.Block.NumberU64())
	h.headForms.Publish(event.Block.ToJson(receipts))
}

// newHeads notifies each new head.
func (h *handlers) newHeads(_ context.Context, subscription *dispatch.Subscription, _ struct{}) error {
	produce(subscription, h.headForms.Subscribe(subscriptionBuffer), func(form *evmcore.EvmHeaderJson) error {
		return subscription.Notify(form)
	})
	return nil
}

// logs notifies each log of a new block that the criteria select, one at a
// time. Bounds that name the pending block are accepted and notify nothing.
func (h *handlers) logs(_ context.Context, subscription *dispatch.Subscription, params logsParams) error {
	if err := checkParameterLimit(h.environment.Config().LogQueryParameterLimit, params.Criteria); err != nil {
		return err
	}
	mined, err := params.Criteria.receivesMined()
	if err != nil || !mined {
		return err
	}
	produce(subscription, h.environment.SubscribeHeads(subscriptionBuffer), func(event core.HeadNotify) error {
		for _, log := range params.Criteria.matching(event.Logs) {
			if err := subscription.Notify(log); err != nil {
				return err
			}
		}
		return nil
	})
	return nil
}

// newPendingTransactions notifies each transaction that enters the pool, as
// its hash or in full, one at a time.
func (h *handlers) newPendingTransactions(_ context.Context, subscription *dispatch.Subscription, params fullParams) error {
	full := params.Full != nil && *params.Full
	produce(subscription, h.environment.SubscribeTransactions(subscriptionBuffer), func(event evmcore.NewTxsNotify) error {
		for _, transaction := range event.Txs {
			var payload any = transaction.Hash()
			if full {
				payload = format.NewPendingTransaction(transaction, h.environment.ChainID())
			}
			if err := subscription.Notify(payload); err != nil {
				return err
			}
		}
		return nil
	})
	return nil
}

// produce notifies the events of a consumer until the subscription ends or
// fails to notify. A consumer that ends has lost events, so the subscription
// is dropped with its connection.
func produce[Event any](subscription *dispatch.Subscription, consumer *chain.Consumer[Event], notify func(Event) error) {
	started := subscription.Go(func(ctx context.Context) {
		defer consumer.Unsubscribe()
		for {
			select {
			case <-ctx.Done():
				return
			case <-consumer.Ended():
				subscription.Drop(consumer.Err())
				return
			case event := <-consumer.Events():
				if err := notify(event); err != nil {
					return
				}
			}
		}
	})
	if !started {
		consumer.Unsubscribe()
	}
}
