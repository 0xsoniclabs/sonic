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

// Package eth serves the eth namespace of the RPC module: the chain, its
// blocks, transactions and receipts, as the legacy handlers answer them.
package eth

import (
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Node is what the namespace asks the node beyond the served chain.
type Node interface {
	Progress() core.Progress
	PoolNonce(address common.Address) uint64
	PoolTransaction(hash common.Hash) (*types.Transaction, error)
}

type handlers struct {
	environment *exec.Environment
	node        Node
}

// Register adds the methods of the namespace to the dispatcher.
func Register(dispatcher *dispatch.Dispatcher, environment *exec.Environment, node Node) {
	h := &handlers{environment: environment, node: node}
	dispatch.Add(dispatcher, "eth_chainId", h.chainID)
	dispatch.Add(dispatcher, "eth_blockNumber", h.blockNumber)
	dispatch.Add(dispatcher, "eth_syncing", h.syncing)
	dispatch.Add(dispatcher, "eth_config", h.config)
	dispatch.Add(dispatcher, "eth_etherbase", zeroAddress)
	dispatch.Add(dispatcher, "eth_coinbase", zeroAddress)
	dispatch.Add(dispatcher, "eth_hashrate", zeroHashrate)

	dispatch.Add(dispatcher, "eth_getBlockByNumber", h.getBlockByNumber)
	dispatch.Add(dispatcher, "eth_getBlockByHash", h.getBlockByHash)
	dispatch.Add(dispatcher, "eth_getHeaderByNumber", h.getHeaderByNumber)
	dispatch.Add(dispatcher, "eth_getHeaderByHash", h.getHeaderByHash)
	dispatch.Add(dispatcher, "eth_getBlockTransactionCountByNumber", h.getBlockTransactionCountByNumber)
	dispatch.Add(dispatcher, "eth_getBlockTransactionCountByHash", h.getBlockTransactionCountByHash)
	dispatch.Add(dispatcher, "eth_getUncleByBlockNumberAndIndex", h.getUncleByBlockNumberAndIndex)
	dispatch.Add(dispatcher, "eth_getUncleByBlockHashAndIndex", h.getUncleByBlockHashAndIndex)
	dispatch.Add(dispatcher, "eth_getUncleCountByBlockNumber", h.getUncleCountByBlockNumber)
	dispatch.Add(dispatcher, "eth_getUncleCountByBlockHash", h.getUncleCountByBlockHash)

	dispatch.Add(dispatcher, "eth_getTransactionByHash", h.getTransactionByHash)
	dispatch.Add(dispatcher, "eth_getRawTransactionByHash", h.getRawTransactionByHash)
	dispatch.Add(dispatcher, "eth_getTransactionByBlockNumberAndIndex", h.getTransactionByBlockNumberAndIndex)
	dispatch.Add(dispatcher, "eth_getTransactionByBlockHashAndIndex", h.getTransactionByBlockHashAndIndex)
	dispatch.Add(dispatcher, "eth_getRawTransactionByBlockNumberAndIndex", h.getRawTransactionByBlockNumberAndIndex)
	dispatch.Add(dispatcher, "eth_getRawTransactionByBlockHashAndIndex", h.getRawTransactionByBlockHashAndIndex)

	dispatch.Add(dispatcher, "eth_getTransactionReceipt", h.getTransactionReceipt)
	dispatch.Add(dispatcher, "eth_getBlockReceipts", h.getBlockReceipts)

	dispatch.Add(dispatcher, "eth_getBalance", h.getBalance)
	dispatch.Add(dispatcher, "eth_getCode", h.getCode)
	dispatch.Add(dispatcher, "eth_getStorageAt", h.getStorageAt)
	dispatch.Add(dispatcher, "eth_getTransactionCount", h.getTransactionCount)
	dispatch.Add(dispatcher, "eth_getAccount", h.getAccount)
	dispatch.Add(dispatcher, "eth_getProof", h.getProof)

	dispatch.Add(dispatcher, "eth_gasPrice", h.gasPrice)
	dispatch.Add(dispatcher, "eth_maxPriorityFeePerGas", h.maxPriorityFeePerGas)
	dispatch.Add(dispatcher, "eth_feeHistory", h.feeHistory)
	dispatch.Add(dispatcher, "eth_blobBaseFee", h.blobBaseFee)

	dispatch.Add(dispatcher, "eth_call", h.call)
	dispatch.Add(dispatcher, "eth_estimateGas", h.estimateGas)
	dispatch.Add(dispatcher, "eth_createAccessList", h.createAccessList)
	dispatch.Add(dispatcher, "eth_simulateV1", h.simulateV1)
}
