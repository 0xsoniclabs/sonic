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
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/format"
)

// headerQueue is how many headers a newHeads subscription may lag behind.
const headerQueue = 1024

// headers builds the newHeads notification of a block once, with the
// block's receipts, and hands it to every newHeads subscription (I16).
type headers struct {
	environment *exec.Environment
	hub         *chain.Hub[*evmcore.EvmHeaderJson]
}

// follow builds the notification of every new block while a subscription
// wants it. A head feed that dropped its consumer ends every newHeads
// subscription, which missed blocks (I18), and is taken again.
func (built *headers) follow(heads *chain.Hub[core.HeadEvent]) func(ctx context.Context) error {
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
					built.hub.DropAll(consumer.Err())
					return fmt.Errorf("head feed of the newHeads subscriptions dropped: %w", consumer.Err())
				}
				if built.hub.Len() > 0 {
					built.hub.Publish(built.notification(ctx, event.Block))
				}
			}
		}
	}
}

// notification is the header of a block with the root and bloom of its
// receipts, as the reference built it; receipts that cannot be read leave
// both out.
func (built *headers) notification(ctx context.Context, block *evmcore.EvmBlock) *evmcore.EvmHeaderJson {
	head := built.environment.View(ctx, chain.Head{Number: block.NumberU64(), Block: block})
	receipts, _ := built.environment.ReceiptsAt(head, block.NumberU64())
	return block.Header().ToJson(receipts)
}

// subscribe serves eth_subscribe: the feed of a subscription is taken before
// its id is answered, so it misses nothing after the answer.
func (handler *handlers) subscribe(namespace string) dispatch.Handler {
	return func(ctx context.Context, arguments dispatch.Arguments) (any, error) {
		call, ok := dispatch.CallFrom(ctx)
		if !ok {
			return nil, dispatch.ErrNotificationsUnsupported
		}
		kind := dispatch.Argument[string](arguments, 0)
		argument := dispatch.ArgumentOr(arguments, 1, json.RawMessage(nil))
		switch kind {
		case "newHeads":
			return handler.subscribeHeads(call, namespace, argument)
		case "logs":
			return handler.subscribeLogs(call, namespace, argument)
		case "newPendingTransactions":
			return handler.subscribeTransactions(call, namespace, argument)
		}
		return nil, dispatch.SubscriptionNotFound(namespace, kind)
	}
}

var (
	noParameters                  []dispatch.Parameter
	logsParameters                = []dispatch.Parameter{dispatch.Required[Criteria]("criteria")}
	pendingTransactionsParameters = []dispatch.Parameter{dispatch.Optional[bool]("fullTransactions")}
)

// subscriptionArguments decodes what a subscription takes after its name,
// with the errors of a method's parameters.
func subscriptionArguments(argument json.RawMessage, parameters []dispatch.Parameter) (dispatch.Arguments, error) {
	raw := json.RawMessage("[]")
	if argument != nil {
		raw = append(append(json.RawMessage("["), argument...), ']')
	}
	return dispatch.DecodeParameters(raw, parameters)
}

func (handler *handlers) subscribeHeads(call *dispatch.Call, namespace string, argument json.RawMessage) (any, error) {
	if _, err := subscriptionArguments(argument, noParameters); err != nil {
		return nil, err
	}
	subscription, err := call.Subscribe(namespace, "newHeads")
	if err != nil {
		return nil, err
	}
	relay(subscription, handler.headers.hub.Subscribe(), func(ctx context.Context, header *evmcore.EvmHeaderJson) error {
		return subscription.Notify(ctx, header)
	})
	return subscription.ID, nil
}

// subscribeLogs notifies the logs of new blocks the criteria match, one by
// one, as the reference did.
func (handler *handlers) subscribeLogs(call *dispatch.Call, namespace string, argument json.RawMessage) (any, error) {
	arguments, err := subscriptionArguments(argument, logsParameters)
	if err != nil {
		return nil, err
	}
	criteria := dispatch.Argument[Criteria](arguments, 0)
	if err := checkParameterLimit(handler.environment.Config().LogQueryParameterLimit, criteria); err != nil {
		return nil, err
	}
	receives, err := receivesLogs(criteria)
	if err != nil {
		return nil, err
	}
	subscription, err := call.Subscribe(namespace, "logs")
	if err != nil {
		return nil, err
	}
	relay(subscription, handler.environment.Heads().Subscribe(), func(ctx context.Context, event core.HeadEvent) error {
		if !receives {
			return nil
		}
		for _, log := range event.Logs {
			if criteria.matches(log) {
				if err := subscription.Notify(ctx, log); err != nil {
					return err
				}
			}
		}
		return nil
	})
	return subscription.ID, nil
}

// subscribeTransactions notifies the transactions entering the pool, one by
// one, as hashes or, asked for, whole and priced at their own gas price.
func (handler *handlers) subscribeTransactions(call *dispatch.Call, namespace string, argument json.RawMessage) (any, error) {
	arguments, err := subscriptionArguments(argument, pendingTransactionsParameters)
	if err != nil {
		return nil, err
	}
	full := dispatch.ArgumentOr(arguments, 0, false)
	subscription, err := call.Subscribe(namespace, "newPendingTransactions")
	if err != nil {
		return nil, err
	}
	chainID := handler.environment.ChainID()
	relay(subscription, handler.environment.Transactions().Subscribe(), func(ctx context.Context, event evmcore.NewTxsNotify) error {
		for _, transaction := range event.Txs {
			if err := subscription.Notify(ctx, pendingForm(transaction, full, chainID)); err != nil {
				return err
			}
		}
		return nil
	})
	return subscription.ID, nil
}

func pendingForm(transaction *types.Transaction, full bool, chainID *big.Int) any {
	if full {
		return format.PendingTransaction(transaction, transaction.GasPrice(), chainID)
	}
	return transaction.Hash()
}

// relay hands the events of a consumer to a subscription until either ends.
// A consumer dropped by its feed lost events and ends the subscription with
// its connection (I18).
func relay[Event any](subscription *dispatch.Subscription, consumer *chain.Consumer[Event], notify func(context.Context, Event) error) {
	subscription.Go(func(ctx context.Context) {
		defer consumer.Close()
		for {
			select {
			case <-ctx.Done():
				return
			case event, open := <-consumer.Events():
				if !open {
					subscription.Fail(consumer.Err())
					return
				}
				if notify(ctx, event) != nil {
					return
				}
			}
		}
	})
}
