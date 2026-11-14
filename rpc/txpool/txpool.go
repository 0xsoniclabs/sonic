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

// Package txpool serves the txpool namespace of the RPC module: the content
// of the node's transaction pool.
package txpool

import (
	"context"
	"fmt"
	"math/big"
	"strconv"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/format"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

// Pool is what the namespace asks the node's pool.
type Pool interface {
	PoolContent() (pending, queued map[common.Address]types.Transactions)
	PoolContentFrom(address common.Address) (pending, queued types.Transactions)
	PoolStats() (pending, queued int)
}

type handlers struct {
	pool    Pool
	chainID *big.Int
}

// Register adds the methods of the namespace to the dispatcher.
func Register(dispatcher *dispatch.Dispatcher, environment *exec.Environment, pool Pool) {
	h := &handlers{pool: pool, chainID: environment.ChainID()}
	dispatch.Add(dispatcher, "txpool_content", h.content)
	dispatch.Add(dispatcher, "txpool_contentFrom", h.contentFrom)
	dispatch.Add(dispatcher, "txpool_status", h.status)
	dispatch.Add(dispatcher, "txpool_inspect", h.inspect)
}

// content answers the pending and queued transactions of every sender by
// their nonces.
func (h *handlers) content(context.Context, struct{}) (map[string]map[string]map[string]*format.Transaction, error) {
	pending, queued := h.pool.PoolContent()
	return map[string]map[string]map[string]*format.Transaction{
		"pending": bySender(pending, h.form),
		"queued":  bySender(queued, h.form),
	}, nil
}

type addressParams struct {
	Address common.Address
}

// contentFrom answers the pending and queued transactions of one sender by
// their nonces.
func (h *handlers) contentFrom(_ context.Context, params addressParams) (map[string]map[string]*format.Transaction, error) {
	pending, queued := h.pool.PoolContentFrom(params.Address)
	return map[string]map[string]*format.Transaction{
		"pending": byNonce(pending, h.form),
		"queued":  byNonce(queued, h.form),
	}, nil
}

// status answers how many transactions are pending and queued.
func (h *handlers) status(context.Context, struct{}) (map[string]hexutil.Uint, error) {
	pending, queued := h.pool.PoolStats()
	return map[string]hexutil.Uint{"pending": hexutil.Uint(pending), "queued": hexutil.Uint(queued)}, nil
}

// inspect answers the content of the pool with each transaction in one line.
func (h *handlers) inspect(context.Context, struct{}) (map[string]map[string]map[string]string, error) {
	pending, queued := h.pool.PoolContent()
	return map[string]map[string]map[string]string{
		"pending": bySender(pending, summary),
		"queued":  bySender(queued, summary),
	}, nil
}

// form is a transaction of the pool, which is in no block.
func (h *handlers) form(transaction *types.Transaction) *format.Transaction {
	return format.NewPendingTransaction(transaction, h.chainID)
}

func summary(transaction *types.Transaction) string {
	if to := transaction.To(); to != nil {
		return fmt.Sprintf("%s: %v wei + %v gas × %v wei", to.Hex(), transaction.Value(), transaction.Gas(), transaction.GasPrice())
	}
	return fmt.Sprintf("contract creation: %v wei + %v gas × %v wei", transaction.Value(), transaction.Gas(), transaction.GasPrice())
}

func bySender[Form any](content map[common.Address]types.Transactions, form func(*types.Transaction) Form) map[string]map[string]Form {
	result := make(map[string]map[string]Form, len(content))
	for sender, transactions := range content {
		result[sender.Hex()] = byNonce(transactions, form)
	}
	return result
}

func byNonce[Form any](transactions types.Transactions, form func(*types.Transaction) Form) map[string]Form {
	result := make(map[string]Form, len(transactions))
	for _, transaction := range transactions {
		result[strconv.FormatUint(transaction.Nonce(), 10)] = form(transaction)
	}
	return result
}
