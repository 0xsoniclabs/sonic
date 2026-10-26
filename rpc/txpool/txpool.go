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

// Package txpool serves the txpool namespace: the content of the pool.
package txpool

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/format"
)

type handlers struct {
	environment *exec.Environment
	pool        core.Pool
}

// Register adds the methods of the namespace.
func Register(dispatcher *dispatch.Dispatcher, environment *exec.Environment, pool core.Pool) {
	handler := &handlers{environment: environment, pool: pool}
	dispatcher.Add("txpool_content", handler.content)
	dispatcher.Add("txpool_contentFrom", handler.contentFrom, dispatch.Required[common.Address]("address"))
	dispatcher.Add("txpool_status", handler.status)
	dispatcher.Add("txpool_inspect", handler.inspect)
}

// byNonce lists the transactions of an account by their nonce.
func byNonce[Entry any](transactions types.Transactions, entry func(*types.Transaction) Entry) map[string]Entry {
	listed := make(map[string]Entry, len(transactions))
	for _, transaction := range transactions {
		listed[fmt.Sprintf("%d", transaction.Nonce())] = entry(transaction)
	}
	return listed
}

// byAccount lists the transactions of each account by their nonce.
func byAccount[Entry any](accounts map[common.Address]types.Transactions, entry func(*types.Transaction) Entry) map[string]map[string]Entry {
	listed := make(map[string]map[string]Entry, len(accounts))
	for account, transactions := range accounts {
		listed[account.Hex()] = byNonce(transactions, entry)
	}
	return listed
}

// pendingForm returns the JSON form of a transaction in the pool, priced at
// the base fee of the head.
func (handler *handlers) pendingForm(ctx context.Context) (func(*types.Transaction) *format.Transaction, error) {
	head, err := handler.environment.Latest(ctx)
	if err != nil {
		return nil, err
	}
	baseFee, chainID := exec.BaseFee(head), handler.environment.ChainID()
	return func(transaction *types.Transaction) *format.Transaction {
		return format.PendingTransaction(transaction, baseFee, chainID)
	}, nil
}

func (handler *handlers) content(ctx context.Context, _ dispatch.Arguments) (any, error) {
	form, err := handler.pendingForm(ctx)
	if err != nil {
		return nil, err
	}
	pending, queued := handler.pool.PoolContent()
	return map[string]map[string]map[string]*format.Transaction{
		"pending": byAccount(pending, form),
		"queued":  byAccount(queued, form),
	}, nil
}

func (handler *handlers) contentFrom(ctx context.Context, arguments dispatch.Arguments) (any, error) {
	form, err := handler.pendingForm(ctx)
	if err != nil {
		return nil, err
	}
	pending, queued := handler.pool.PoolContentFrom(dispatch.Argument[common.Address](arguments, 0))
	return map[string]map[string]*format.Transaction{
		"pending": byNonce(pending, form),
		"queued":  byNonce(queued, form),
	}, nil
}

func (handler *handlers) status(context.Context, dispatch.Arguments) (any, error) {
	pending, queued := handler.pool.PoolStats()
	return map[string]hexutil.Uint{"pending": hexutil.Uint(pending), "queued": hexutil.Uint(queued)}, nil
}

func (handler *handlers) inspect(context.Context, dispatch.Arguments) (any, error) {
	pending, queued := handler.pool.PoolContent()
	return map[string]map[string]map[string]string{
		"pending": byAccount(pending, summary),
		"queued":  byAccount(queued, summary),
	}, nil
}

// summary is the one-line form of a transaction in txpool_inspect.
func summary(transaction *types.Transaction) string {
	if recipient := transaction.To(); recipient != nil {
		return fmt.Sprintf("%s: %v wei + %v gas × %v wei", recipient.Hex(), transaction.Value(), transaction.Gas(), transaction.GasPrice())
	}
	return fmt.Sprintf("contract creation: %v wei + %v gas × %v wei", transaction.Value(), transaction.Gas(), transaction.GasPrice())
}
