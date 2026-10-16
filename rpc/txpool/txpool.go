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

// Package txpool implements the txpool_ namespace: a view of the transaction
// pool of the node.
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
	"github.com/0xsoniclabs/sonic/rpc/eth"
)

type api struct {
	chain *dispatch.Chain
	core  core.Core
}

// Register registers txpool_content, txpool_contentFrom, txpool_status and
// txpool_inspect.
func Register(d *dispatch.Dispatcher, chain *dispatch.Chain) {
	a := &api{chain: chain, core: chain.Core()}
	dispatch.Add0(d, "txpool_content", a.content)
	dispatch.Add1(d, "txpool_contentFrom", a.contentFrom, "address")
	dispatch.Add0(d, "txpool_status", a.status)
	dispatch.Add0(d, "txpool_inspect", a.inspect)
}

func (a *api) chainID() *big.Int { return new(big.Int).SetUint64(a.core.ChainInfo().ChainID) }

func (a *api) baseFee() *big.Int {
	if header := a.chain.Head().Header; header != nil {
		return header.BaseFee
	}
	return nil
}

// content returns the transactions contained within the transaction pool.
func (a *api) content(context.Context) (map[string]map[string]map[string]*eth.RPCTransaction, error) {
	content := map[string]map[string]map[string]*eth.RPCTransaction{
		"pending": make(map[string]map[string]*eth.RPCTransaction),
		"queued":  make(map[string]map[string]*eth.RPCTransaction),
	}
	pending, queue := a.core.PoolContent()

	baseFee, chainID := a.baseFee(), a.chainID()
	// Flatten the pending transactions
	for account, txs := range pending {
		dump := make(map[string]*eth.RPCTransaction)
		for _, tx := range txs {
			dump[fmt.Sprintf("%d", tx.Nonce())] = eth.NewRPCPendingTransaction(tx, baseFee, chainID)
		}
		content["pending"][account.Hex()] = dump
	}
	// Flatten the queued transactions
	for account, txs := range queue {
		dump := make(map[string]*eth.RPCTransaction)
		for _, tx := range txs {
			dump[fmt.Sprintf("%d", tx.Nonce())] = eth.NewRPCPendingTransaction(tx, baseFee, chainID)
		}
		content["queued"][account.Hex()] = dump
	}
	return content, nil
}

// contentFrom returns the transactions of one account contained within the
// transaction pool.
func (a *api) contentFrom(_ context.Context, addr common.Address) (map[string]map[string]*eth.RPCTransaction, error) {
	content := make(map[string]map[string]*eth.RPCTransaction, 2)
	pending, queue := a.core.PoolContentFrom(addr)
	baseFee, chainID := a.baseFee(), a.chainID()

	// Build the pending transactions
	dump := make(map[string]*eth.RPCTransaction, len(pending))
	for _, tx := range pending {
		dump[fmt.Sprintf("%d", tx.Nonce())] = eth.NewRPCPendingTransaction(tx, baseFee, chainID)
	}
	content["pending"] = dump

	// Build the queued transactions
	dump = make(map[string]*eth.RPCTransaction, len(queue))
	for _, tx := range queue {
		dump[fmt.Sprintf("%d", tx.Nonce())] = eth.NewRPCPendingTransaction(tx, baseFee, chainID)
	}
	content["queued"] = dump

	return content, nil
}

// status returns the number of pending and queued transaction in the pool.
func (a *api) status(context.Context) (map[string]hexutil.Uint, error) {
	pending, queue := a.core.PoolStats()
	return map[string]hexutil.Uint{
		"pending": hexutil.Uint(pending),
		"queued":  hexutil.Uint(queue),
	}, nil
}

// inspect retrieves the content of the transaction pool and flattens it into an
// easily inspectable list.
func (a *api) inspect(context.Context) (map[string]map[string]map[string]string, error) {
	content := map[string]map[string]map[string]string{
		"pending": make(map[string]map[string]string),
		"queued":  make(map[string]map[string]string),
	}
	pending, queue := a.core.PoolContent()

	// Define a formatter to flatten a transaction into a string
	var format = func(tx *types.Transaction) string {
		if to := tx.To(); to != nil {
			return fmt.Sprintf("%s: %v wei + %v gas × %v wei", tx.To().Hex(), tx.Value(), tx.Gas(), tx.GasPrice())
		}
		return fmt.Sprintf("contract creation: %v wei + %v gas × %v wei", tx.Value(), tx.Gas(), tx.GasPrice())
	}
	// Flatten the pending transactions
	for account, txs := range pending {
		dump := make(map[string]string)
		for _, tx := range txs {
			dump[fmt.Sprintf("%d", tx.Nonce())] = format(tx)
		}
		content["pending"][account.Hex()] = dump
	}
	// Flatten the queued transactions
	for account, txs := range queue {
		dump := make(map[string]string)
		for _, tx := range txs {
			dump[fmt.Sprintf("%d", tx.Nonce())] = format(tx)
		}
		content["queued"][account.Hex()] = dump
	}
	return content, nil
}
