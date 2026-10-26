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

// Package eth serves the eth namespace, which the node also serves as ftm.
package eth

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"

	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
)

// Node is what the namespace reads of the node beyond the environment: the
// pool and the node's progress.
type Node interface {
	core.Pool
	core.NodeInfo
}

type handlers struct {
	environment *exec.Environment
	node        Node
}

// Register adds the methods of the namespace, under the given namespace name
// (eth or ftm).
func Register(dispatcher *dispatch.Dispatcher, environment *exec.Environment, node Node, namespace string) {
	handler := &handlers{environment: environment, node: node}
	add := func(name string, function dispatch.Handler, parameters ...dispatch.Parameter) {
		dispatcher.Add(namespace+"_"+name, function, parameters...)
	}
	block := dispatch.Required[chain.BlockNumber]("block")
	hash := dispatch.Required[common.Hash]("hash")
	fullTransactions := dispatch.Required[bool]("fullTransactions")
	index := dispatch.Required[hexutil.Uint]("index")

	add("chainId", handler.chainID)
	add("blockNumber", handler.blockNumber)
	add("syncing", handler.syncing)
	add("currentEpoch", handler.currentEpoch)
	add("getRules", handler.getRules, block)
	add("config", handler.config)
	add("etherbase", handler.zeroAddress)
	add("coinbase", handler.zeroAddress)
	add("hashrate", handler.hashrate)

	add("gasPrice", handler.gasPrice)
	add("maxPriorityFeePerGas", handler.maxPriorityFeePerGas)
	add("feeHistory", handler.feeHistory,
		dispatch.Required[math.HexOrDecimal64]("blockCount"),
		dispatch.Required[chain.BlockNumber]("newestBlock"),
		dispatch.Required[[]float64]("rewardPercentiles"))
	add("blobBaseFee", handler.blobBaseFee)

	address := dispatch.Required[common.Address]("address")
	optionalBlock := dispatch.Optional[chain.BlockReference]("block")
	add("getBalance", handler.getBalance, address, optionalBlock)
	add("getCode", handler.getCode, address, optionalBlock)
	add("getStorageAt", handler.getStorageAt, address, dispatch.Required[string]("key"), optionalBlock)
	add("getTransactionCount", handler.getTransactionCount, address, optionalBlock)
	add("getAccount", handler.getAccount, address, optionalBlock)
	add("getProof", handler.getProof, address, dispatch.Required[[]string]("storageKeys"), optionalBlock)

	transaction := dispatch.Required[exec.CallArguments]("transaction")
	stateOverrides := dispatch.Optional[exec.StateOverride]("stateOverrides")
	blockOverrides := dispatch.Optional[exec.BlockOverrides]("blockOverrides")
	add("call", handler.call, transaction, optionalBlock, stateOverrides, blockOverrides)
	add("estimateGas", handler.estimateGas, transaction, optionalBlock, stateOverrides, blockOverrides)
	add("createAccessList", handler.createAccessList, transaction, optionalBlock)
	add("simulateV1", handler.simulateV1, dispatch.Required[simulateOptions]("options"), optionalBlock)

	encoded := dispatch.Required[hexutil.Bytes]("transaction")
	add("sendRawTransaction", handler.sendRawTransaction, encoded)
	add("sendRawTransactionSync", handler.sendRawTransactionSync, encoded, dispatch.Optional[hexutil.Uint64]("timeout"))

	add("getBlockByNumber", handler.getBlockByNumber, block, fullTransactions)
	add("getBlockByHash", handler.getBlockByHash, hash, fullTransactions)
	add("getHeaderByNumber", handler.getHeaderByNumber, block)
	add("getHeaderByHash", handler.getHeaderByHash, hash)
	add("getBlockReceipts", handler.getBlockReceipts, dispatch.Required[chain.BlockReference]("block"))
	add("getBlockTransactionCountByNumber", handler.getBlockTransactionCountByNumber, block)
	add("getBlockTransactionCountByHash", handler.getBlockTransactionCountByHash, hash)
	add("getUncleByBlockNumberAndIndex", handler.getUncleByBlockNumberAndIndex, block, index)
	add("getUncleByBlockHashAndIndex", handler.getUncleByBlockHashAndIndex, hash, index)
	add("getUncleCountByBlockNumber", handler.getUncleCountByBlockNumber, block)
	add("getUncleCountByBlockHash", handler.getUncleCountByBlockHash, hash)

	add("getTransactionByBlockNumberAndIndex", handler.getTransactionByBlockNumberAndIndex, block, index)
	add("getTransactionByBlockHashAndIndex", handler.getTransactionByBlockHashAndIndex, hash, index)
	add("getRawTransactionByBlockNumberAndIndex", handler.getRawTransactionByBlockNumberAndIndex, block, index)
	add("getRawTransactionByBlockHashAndIndex", handler.getRawTransactionByBlockHashAndIndex, hash, index)
	add("getTransactionByHash", handler.getTransactionByHash, hash)
	add("getRawTransactionByHash", handler.getRawTransactionByHash, hash)
	add("getTransactionReceipt", handler.getTransactionReceipt, hash)
}
