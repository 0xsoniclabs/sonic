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
// Package txpool serves the txpool namespace.
package txpool

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/format"
)

type api struct {
	env  *exec.Env
	pool core.Pool
}

// Register adds the txpool methods over the pool.
func Register(d *dispatch.Dispatcher, env *exec.Env, pool core.Pool) {
	a := &api{env: env, pool: pool}
	d.Add("txpool_content", dispatch.Func(a.content))
	d.Add("txpool_contentFrom", dispatch.Func(a.contentFrom))
	d.Add("txpool_status", dispatch.Func(a.status))
	d.Add("txpool_inspect", dispatch.Func(a.inspect))
}

// byNonce lists the transactions of one account by nonce, each in the form
// encode gives it.
func byNonce[T any](txs types.Transactions, encode func(*types.Transaction) T) map[string]T {
	dump := make(map[string]T, len(txs))
	for _, tx := range txs {
		dump[fmt.Sprintf("%d", tx.Nonce())] = encode(tx)
	}
	return dump
}

func (a *api) pending(ctx context.Context) func(*types.Transaction) *format.Transaction {
	baseFee, chainID := new(big.Int), a.env.ChainID()
	if fee := a.env.Latest(ctx).Block.Header().BaseFee; fee != nil {
		baseFee = fee
	}
	return func(tx *types.Transaction) *format.Transaction {
		return format.NewPendingTransaction(tx, baseFee, chainID)
	}
}

func (a *api) content(ctx context.Context, _ dispatch.None) (any, error) {
	pending, queued := a.pool.PoolContent()
	encode := a.pending(ctx)
	content := map[string]map[string]map[string]*format.Transaction{
		"pending": make(map[string]map[string]*format.Transaction, len(pending)),
		"queued":  make(map[string]map[string]*format.Transaction, len(queued)),
	}
	for account, txs := range pending {
		content["pending"][account.Hex()] = byNonce(txs, encode)
	}
	for account, txs := range queued {
		content["queued"][account.Hex()] = byNonce(txs, encode)
	}
	return content, nil
}

type addressArg struct {
	Address common.Address `rpc:"address"`
}

func (a *api) contentFrom(ctx context.Context, p addressArg) (any, error) {
	pending, queued := a.pool.PoolContentFrom(p.Address)
	encode := a.pending(ctx)
	return map[string]map[string]*format.Transaction{
		"pending": byNonce(pending, encode),
		"queued":  byNonce(queued, encode),
	}, nil
}

func (a *api) status(ctx context.Context, _ dispatch.None) (any, error) {
	pending, queued := a.pool.PoolStats()
	return map[string]hexutil.Uint{
		"pending": hexutil.Uint(pending),
		"queued":  hexutil.Uint(queued),
	}, nil
}

// inspect flattens the pool into one line per transaction.
func (a *api) inspect(ctx context.Context, _ dispatch.None) (any, error) {
	pending, queued := a.pool.PoolContent()
	content := map[string]map[string]map[string]string{
		"pending": make(map[string]map[string]string, len(pending)),
		"queued":  make(map[string]map[string]string, len(queued)),
	}
	for account, txs := range pending {
		content["pending"][account.Hex()] = byNonce(txs, describe)
	}
	for account, txs := range queued {
		content["queued"][account.Hex()] = byNonce(txs, describe)
	}
	return content, nil
}

func describe(tx *types.Transaction) string {
	if to := tx.To(); to != nil {
		return fmt.Sprintf("%s: %v wei + %v gas × %v wei", to.Hex(), tx.Value(), tx.Gas(), tx.GasPrice())
	}
	return fmt.Sprintf("contract creation: %v wei + %v gas × %v wei", tx.Value(), tx.Gas(), tx.GasPrice())
}
