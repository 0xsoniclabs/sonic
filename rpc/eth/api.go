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

// Package eth implements the eth_ namespace, which the module also serves as
// ftm_. The handlers keep the order of checks, the wording of errors and the
// shape of results of the node's earlier RPC handlers.
package eth

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
)

// noUncles is the number of uncles of every block.
const noUncles = 0

// API holds the state shared by the handlers of the namespace.
type API struct {
	env   *exec.Env
	chain *dispatch.Chain
	core  core.Core
	cfg   dispatch.Config
}

// Register registers the handlers on the dispatcher under the namespace "eth"
// or "ftm".
func Register(d *dispatch.Dispatcher, env *exec.Env, ns string) {
	a := &API{env: env, chain: env.Chain, core: env.Core, cfg: env.Cfg}
	m := func(name string) string { return ns + "_" + name }

	dispatch.Add0(d, m("chainId"), a.chainID)
	dispatch.Add0(d, m("blockNumber"), a.blockNumber)
	dispatch.Add0(d, m("syncing"), a.syncing)
	dispatch.Add0(d, m("etherbase"), a.zeroAddress)
	dispatch.Add0(d, m("coinbase"), a.zeroAddress)
	dispatch.Add0(d, m("hashrate"), a.hashrate)

	dispatch.Add2(d, m("getBlockByNumber"), a.getBlockByNumber, "number", "fullTx")
	dispatch.Add2(d, m("getBlockByHash"), a.getBlockByHash, "hash", "fullTx")
	dispatch.Add1(d, m("getHeaderByNumber"), a.getHeaderByNumber, "number")
	dispatch.Add1(d, m("getHeaderByHash"), a.getHeaderByHash, "hash")
	dispatch.Add1(d, m("getBlockReceipts"), a.getBlockReceipts, "blockNrOrHash")
	dispatch.Add1(d, m("getBlockTransactionCountByNumber"), a.getBlockTransactionCountByNumber, "blockNr")
	dispatch.Add1(d, m("getBlockTransactionCountByHash"), a.getBlockTransactionCountByHash, "blockHash")
	dispatch.Add2(d, m("getUncleByBlockNumberAndIndex"), a.getUncleByBlockNumberAndIndex, "blockNr", "index")
	dispatch.Add2(d, m("getUncleByBlockHashAndIndex"), a.getUncleByBlockHashAndIndex, "blockHash", "index")
	dispatch.Add1(d, m("getUncleCountByBlockNumber"), a.getUncleCountByBlockNumber, "blockNr")
	dispatch.Add1(d, m("getUncleCountByBlockHash"), a.getUncleCountByBlockHash, "blockHash")

	dispatch.Add2(d, m("getTransactionByBlockNumberAndIndex"), a.getTransactionByBlockNumberAndIndex, "blockNr", "index")
	dispatch.Add2(d, m("getTransactionByBlockHashAndIndex"), a.getTransactionByBlockHashAndIndex, "blockHash", "index")
	dispatch.Add2(d, m("getRawTransactionByBlockNumberAndIndex"), a.getRawTransactionByBlockNumberAndIndex, "blockNr", "index")
	dispatch.Add2(d, m("getRawTransactionByBlockHashAndIndex"), a.getRawTransactionByBlockHashAndIndex, "blockHash", "index")
	dispatch.Add1(d, m("getTransactionByHash"), a.getTransactionByHash, "hash")
	dispatch.Add1(d, m("getRawTransactionByHash"), a.getRawTransactionByHash, "hash")
	dispatch.Add1(d, m("getTransactionReceipt"), a.getTransactionReceipt, "hash")

	registerState(d, a, m)
	registerCalls(d, a, m)
	registerPrices(d, a, m)
	registerSend(d, a, m)

	dispatch.Add1(d, m("getRules"), a.getRules, "block")
	dispatch.Add0(d, m("currentEpoch"), a.currentEpoch)
	dispatch.Add0(d, m("config"), a.config)
}

// -- constants and liveness --

func (a *API) chainID(context.Context) (*hexutil.Big, error) {
	return (*hexutil.Big)(new(big.Int).SetUint64(a.core.ChainInfo().ChainID)), nil
}

func (a *API) blockNumber(context.Context) (hexutil.Uint64, error) {
	return hexutil.Uint64(a.chain.Latest()), nil
}

func (a *API) zeroAddress(context.Context) (common.Address, error) {
	return common.Address{}, nil
}

func (a *API) hashrate(context.Context) (hexutil.Uint64, error) { return 0, nil }

// syncing reports false once the node has caught up, and the sync progress
// otherwise.
func (a *API) syncing(context.Context) (any, error) {
	progress := a.core.Progress()
	blockTime := time.Unix(0, int64(progress.CurrentBlockTime))
	// Not syncing if the synchronization already completed; the limit must be
	// much larger than the longest emit interval.
	if time.Since(blockTime) <= 90*time.Minute {
		return false, nil
	}
	return map[string]any{
		"startingBlock":    hexutil.Uint64(0), // back-compatibility
		"currentEpoch":     hexutil.Uint64(progress.CurrentEpoch),
		"currentBlock":     hexutil.Uint64(progress.CurrentBlock),
		"currentBlockHash": progress.CurrentBlockHash.Hex(),
		"currentBlockTime": hexutil.Uint64(progress.CurrentBlockTime),
		"highestBlock":     hexutil.Uint64(progress.HighestBlock),
		"highestEpoch":     hexutil.Uint64(progress.HighestEpoch),
		"pulledStates":     hexutil.Uint64(0), // back-compatibility
		"knownStates":      hexutil.Uint64(0), // back-compatibility
	}, nil
}

// -- block lookup, as the node's backend answered it --

// blockAt returns the block with the number, nil if there is none.
func (a *API) blockAt(head dispatch.Head, n uint64) (*evmcore.EvmBlock, error) {
	if n > head.Number {
		return nil, nil
	}
	block, err := a.core.Block(n)
	if errors.Is(err, core.ErrNotFound) {
		return nil, nil
	}
	return block, err
}

// headerAt returns the header of the block with the number, nil if there is none.
func (a *API) headerAt(head dispatch.Head, n uint64) (*evmcore.EvmHeader, error) {
	if n > head.Number {
		return nil, nil
	}
	if n == head.Number && head.Header != nil {
		return head.Header, nil
	}
	header, err := a.core.Header(n)
	if errors.Is(err, core.ErrNotFound) {
		return nil, nil
	}
	return header, err
}

// blockByNumber returns the block a number or tag names, nil if there is none.
func (a *API) blockByNumber(head dispatch.Head, bn dispatch.BlockNumber) (*evmcore.EvmBlock, error) {
	n, err := a.chain.Number(head, bn)
	if err != nil {
		return nil, nil
	}
	return a.blockAt(head, n)
}

// headerByNumber returns the header a number or tag names, nil if there is none.
func (a *API) headerByNumber(head dispatch.Head, bn dispatch.BlockNumber) (*evmcore.EvmHeader, error) {
	n, err := a.chain.Number(head, bn)
	if err != nil {
		return nil, nil
	}
	return a.headerAt(head, n)
}

// blockByHash returns the block with the hash, nil if there is none.
func (a *API) blockByHash(head dispatch.Head, hash common.Hash) (*evmcore.EvmBlock, error) {
	n, ok := a.core.BlockNumberByHash(hash)
	if !ok {
		return nil, nil
	}
	return a.blockAt(head, n)
}

// receiptsAt returns the receipts of a block; nil if the block is unknown.
func (a *API) receiptsAt(n uint64) (types.Receipts, error) {
	receipts, err := a.core.Receipts(n)
	if errors.Is(err, core.ErrNotFound) {
		return nil, nil
	}
	return receipts, err
}

func (a *API) bigChainID() *big.Int {
	return new(big.Int).SetUint64(a.core.ChainInfo().ChainID)
}

// minGasPrice is the price the node suggests for new transactions.
func (a *API) minGasPrice(head dispatch.Head) *big.Int { return a.env.MinGasPrice(head) }

// -- blocks and headers --

func (a *API) getBlockByNumber(_ context.Context, number dispatch.BlockNumber, fullTx bool) (*evmcore.EvmBlockJson, error) {
	head := a.chain.Head()
	block, err := a.blockByNumber(head, number)
	if block == nil || err != nil {
		return nil, err
	}
	return a.marshalBlock(block, fullTx)
}

func (a *API) getBlockByHash(_ context.Context, hash common.Hash, fullTx bool) (*evmcore.EvmBlockJson, error) {
	head := a.chain.Head()
	block, err := a.blockByHash(head, hash)
	if block == nil || err != nil {
		return nil, err
	}
	return a.marshalBlock(block, fullTx)
}

func (a *API) marshalBlock(block *evmcore.EvmBlock, fullTx bool) (*evmcore.EvmBlockJson, error) {
	receipts, err := a.receiptsAt(block.NumberU64())
	if err != nil {
		return nil, err
	}
	return RPCMarshalBlock(block, receipts, true, fullTx, a.bigChainID())
}

// The JSON of a header holds the root of its transactions, which only the
// transactions of the block give: the headers are read with their blocks.
func (a *API) getHeaderByNumber(_ context.Context, number dispatch.BlockNumber) (*evmcore.EvmHeaderJson, error) {
	head := a.chain.Head()
	block, err := a.blockByNumber(head, number)
	if block == nil || err != nil {
		return nil, err
	}
	return a.headerWithReceipts(block.Header())
}

func (a *API) getHeaderByHash(_ context.Context, hash common.Hash) (*evmcore.EvmHeaderJson, error) {
	head := a.chain.Head()
	n, ok := a.core.BlockNumberByHash(hash)
	if !ok {
		return nil, fmt.Errorf("block with hash %s not found", hash.String())
	}
	block, err := a.blockAt(head, n)
	if block == nil || err != nil {
		return nil, err
	}
	return a.headerWithReceipts(block.Header())
}

func (a *API) headerWithReceipts(header *evmcore.EvmHeader) (*evmcore.EvmHeaderJson, error) {
	receipts, err := a.receiptsAt(header.Number.Uint64())
	if receipts == nil || err != nil {
		return nil, err
	}
	return header.ToJson(receipts), nil
}

func (a *API) getBlockTransactionCountByNumber(_ context.Context, blockNr dispatch.BlockNumber) (*hexutil.Uint, error) {
	head := a.chain.Head()
	if block, _ := a.blockByNumber(head, blockNr); block != nil {
		n := hexutil.Uint(len(block.Transactions))
		return &n, nil
	}
	return nil, nil
}

func (a *API) getBlockTransactionCountByHash(_ context.Context, blockHash common.Hash) (*hexutil.Uint, error) {
	head := a.chain.Head()
	if block, _ := a.blockByHash(head, blockHash); block != nil {
		n := hexutil.Uint(len(block.Transactions))
		return &n, nil
	}
	return nil, nil
}

// Uncles do not exist: every block has none.

func (a *API) getUncleByBlockNumberAndIndex(_ context.Context, blockNr dispatch.BlockNumber, _ hexutil.Uint) (map[string]any, error) {
	_, err := a.blockByNumber(a.chain.Head(), blockNr)
	return nil, err
}

func (a *API) getUncleByBlockHashAndIndex(_ context.Context, blockHash common.Hash, _ hexutil.Uint) (map[string]any, error) {
	_, err := a.blockByHash(a.chain.Head(), blockHash)
	return nil, err
}

func (a *API) getUncleCountByBlockNumber(_ context.Context, blockNr dispatch.BlockNumber) (*hexutil.Uint, error) {
	if block, _ := a.blockByNumber(a.chain.Head(), blockNr); block != nil {
		n := hexutil.Uint(noUncles)
		return &n, nil
	}
	return nil, nil
}

func (a *API) getUncleCountByBlockHash(_ context.Context, blockHash common.Hash) (*hexutil.Uint, error) {
	if block, _ := a.blockByHash(a.chain.Head(), blockHash); block != nil {
		n := hexutil.Uint(noUncles)
		return &n, nil
	}
	return nil, nil
}

// -- transactions --

func (a *API) getTransactionByBlockNumberAndIndex(_ context.Context, blockNr dispatch.BlockNumber, index hexutil.Uint) (*RPCTransaction, error) {
	if block, _ := a.blockByNumber(a.chain.Head(), blockNr); block != nil {
		return newRPCTransactionFromBlockIndex(block, uint64(index), a.bigChainID()), nil
	}
	return nil, nil
}

func (a *API) getTransactionByBlockHashAndIndex(_ context.Context, blockHash common.Hash, index hexutil.Uint) (*RPCTransaction, error) {
	if block, _ := a.blockByHash(a.chain.Head(), blockHash); block != nil {
		return newRPCTransactionFromBlockIndex(block, uint64(index), a.bigChainID()), nil
	}
	return nil, nil
}

func (a *API) getRawTransactionByBlockNumberAndIndex(_ context.Context, blockNr dispatch.BlockNumber, index hexutil.Uint) (hexutil.Bytes, error) {
	if block, _ := a.blockByNumber(a.chain.Head(), blockNr); block != nil {
		return newRPCRawTransactionFromBlockIndex(block, uint64(index)), nil
	}
	return nil, nil
}

func (a *API) getRawTransactionByBlockHashAndIndex(_ context.Context, blockHash common.Hash, index hexutil.Uint) (hexutil.Bytes, error) {
	if block, _ := a.blockByHash(a.chain.Head(), blockHash); block != nil {
		return newRPCRawTransactionFromBlockIndex(block, uint64(index)), nil
	}
	return nil, nil
}

// txByHash looks a transaction up in the chain; nil if it is unknown.
// Transactions of blocks the module does not serve yet are unknown.
func (a *API) txByHash(head dispatch.Head, hash common.Hash) (tx *types.Transaction, block, index uint64, err error) {
	tx, block, index, err = a.core.TxByHash(hash)
	if errors.Is(err, core.ErrNotFound) || (err == nil && (tx == nil || block > head.Number)) {
		return nil, 0, 0, nil
	}
	if err != nil {
		return nil, 0, 0, err
	}
	return tx, block, index, nil
}

func (a *API) getTransactionByHash(_ context.Context, hash common.Hash) (*RPCTransaction, error) {
	head := a.chain.Head()
	// Try to return an already finalized transaction.
	tx, blockNumber, index, err := a.txByHash(head, hash)
	if err != nil {
		return nil, err
	}
	if tx != nil {
		header, err := a.headerAt(head, blockNumber)
		if header == nil || err != nil {
			return nil, err
		}
		return newRPCTransaction(tx, header.Hash, blockNumber, index, header.BaseFee, a.bigChainID()), nil
	}
	// No finalized transaction, try to retrieve it from the pool.
	if tx := a.core.PoolTransaction(hash); tx != nil {
		return newRPCPendingTransaction(tx, a.minGasPrice(head), a.bigChainID()), nil
	}
	// Transaction unknown, return as such.
	return nil, nil
}

func (a *API) getRawTransactionByHash(_ context.Context, hash common.Hash) (hexutil.Bytes, error) {
	head := a.chain.Head()
	tx, _, _, err := a.txByHash(head, hash)
	if err != nil {
		return nil, err
	}
	if tx == nil {
		if tx = a.core.PoolTransaction(hash); tx == nil {
			// Transaction not found anywhere.
			return nil, nil
		}
	}
	return tx.MarshalBinary()
}

func (a *API) getTransactionReceipt(_ context.Context, hash common.Hash) (map[string]any, error) {
	head := a.chain.Head()
	tx, blockNumber, index, err := a.txByHash(head, hash)
	if tx == nil || err != nil {
		return nil, err
	}
	header, err := a.headerAt(head, blockNumber)
	if header == nil || err != nil {
		return nil, err
	}
	receipts, err := a.receiptsAt(blockNumber)
	if receipts == nil || err != nil {
		return nil, err
	}
	if receipts.Len() <= int(index) {
		return nil, nil
	}
	return formatTxReceipt(a.core.ChainInfo().ChainID, header, tx, index, receipts[index]), nil
}

func (a *API) getBlockReceipts(_ context.Context, ref dispatch.BlockRef) ([]map[string]any, error) {
	head := a.chain.Head()

	var (
		number uint64
		header *evmcore.EvmHeader
	)
	switch {
	case ref.Number != nil:
		n, err := a.chain.Number(head, *ref.Number)
		if err != nil {
			return nil, nil
		}
		number = n
		if header, err = a.headerAt(head, n); header == nil || err != nil {
			return nil, err
		}
	case ref.Hash != nil:
		n, ok := a.core.BlockNumberByHash(*ref.Hash)
		if !ok {
			return nil, fmt.Errorf("block with hash %s not found", ref.Hash.String())
		}
		number = n
		var err error
		if header, err = a.headerAt(head, n); header == nil || err != nil {
			return nil, err
		}
	}

	receipts, err := a.receiptsAt(number)
	if receipts == nil || err != nil {
		return nil, err
	}
	block, err := a.blockAt(head, number)
	if err != nil {
		return nil, err
	}
	chainID := a.core.ChainInfo().ChainID
	result := make([]map[string]any, len(receipts))
	for i, receipt := range receipts {
		if block == nil || i >= len(block.Transactions) || header == nil {
			return nil, fmt.Errorf("no transaction for receipt %d of block %d", i, number)
		}
		result[i] = formatTxReceipt(chainID, header, block.Transactions[i], uint64(i), receipt)
	}
	return result, nil
}

// -- Sonic extensions --

func (a *API) currentEpoch(context.Context) (hexutil.Uint64, error) {
	return hexutil.Uint64(a.core.Progress().CurrentEpoch), nil
}

// getRules returns the network rules in force at a block; null if the block
// does not exist. The tags that select the head answer the rules the next block
// is processed with: an epoch sealed by the head block, or a rule change that
// ends the epoch before Allegro, is seen at once.
func (a *API) getRules(_ context.Context, block dispatch.BlockNumber) (*opera.Rules, error) {
	head := a.chain.Head()
	n, err := a.chain.Number(head, block)
	if err != nil {
		return nil, nil
	}
	if block.IsLatest() {
		n++
	} else if n > head.Number {
		return nil, nil
	}
	rules, err := a.core.Rules(n)
	if errors.Is(err, core.ErrNotFound) {
		return nil, nil
	}
	return rules, err
}
