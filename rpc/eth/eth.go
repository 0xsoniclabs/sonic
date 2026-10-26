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

// Package eth serves the eth namespace, mirrored as ftm.
package eth

import (
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
)

// api holds the dependencies of the eth handlers, each once.
type api struct {
	env  *exec.Env
	pool core.Pool
	node core.NodeInfo
}

// Register adds the eth methods under the namespace ns, "eth" or its mirror
// "ftm". The pool serves pending nonces, pool lookups and sends; the node
// info serves the synchronization status.
func Register(d *dispatch.Dispatcher, env *exec.Env, pool core.Pool, node core.NodeInfo, ns string) {
	a := &api{env: env, pool: pool, node: node}
	add := func(name string, m dispatch.Method) { d.Add(ns+"_"+name, m) }

	// chain facts
	add("chainId", dispatch.Func(a.chainID))
	add("blockNumber", dispatch.Func(a.blockNumber))
	add("syncing", dispatch.Func(a.syncing))
	add("currentEpoch", dispatch.Func(a.currentEpoch))
	add("getRules", dispatch.Func(a.getRules))
	add("config", dispatch.Func(a.config))
	add("etherbase", dispatch.Func(zeroAddress))
	add("coinbase", dispatch.Func(zeroAddress))
	add("hashrate", dispatch.Func(zeroHashrate))

	// fees
	add("gasPrice", dispatch.Func(a.gasPrice))
	add("maxPriorityFeePerGas", dispatch.Func(a.maxPriorityFeePerGas))
	add("feeHistory", dispatch.Func(a.feeHistory))
	add("blobBaseFee", dispatch.Func(blobBaseFee))

	// blocks and headers
	add("getBlockByNumber", dispatch.Func(a.getBlockByNumber))
	add("getBlockByHash", dispatch.Func(a.getBlockByHash))
	add("getHeaderByNumber", dispatch.Func(a.getHeaderByNumber))
	add("getHeaderByHash", dispatch.Func(a.getHeaderByHash))
	add("getBlockReceipts", dispatch.Func(a.getBlockReceipts))
	add("getBlockTransactionCountByNumber", dispatch.Func(a.getBlockTransactionCountByNumber))
	add("getBlockTransactionCountByHash", dispatch.Func(a.getBlockTransactionCountByHash))
	add("getUncleByBlockNumberAndIndex", dispatch.Func(noUncleByNumber))
	add("getUncleByBlockHashAndIndex", dispatch.Func(noUncleByHash))
	add("getUncleCountByBlockNumber", dispatch.Func(a.getUncleCountByBlockNumber))
	add("getUncleCountByBlockHash", dispatch.Func(a.getUncleCountByBlockHash))

	// transactions
	add("getTransactionByBlockNumberAndIndex", dispatch.Func(a.getTransactionByBlockNumberAndIndex))
	add("getTransactionByBlockHashAndIndex", dispatch.Func(a.getTransactionByBlockHashAndIndex))
	add("getRawTransactionByBlockNumberAndIndex", dispatch.Func(a.getRawTransactionByBlockNumberAndIndex))
	add("getRawTransactionByBlockHashAndIndex", dispatch.Func(a.getRawTransactionByBlockHashAndIndex))
	add("getTransactionByHash", dispatch.Func(a.getTransactionByHash))
	add("getRawTransactionByHash", dispatch.Func(a.getRawTransactionByHash))
	add("getTransactionReceipt", dispatch.Func(a.getTransactionReceipt))

	// state
	add("getBalance", dispatch.Func(a.getBalance))
	add("getCode", dispatch.Func(a.getCode))
	add("getStorageAt", dispatch.Func(a.getStorageAt))
	add("getTransactionCount", dispatch.Func(a.getTransactionCount))
	add("getAccount", dispatch.Func(a.getAccount))
	add("getProof", dispatch.Func(a.getProof))

	// execution
	add("call", dispatch.Func(a.call))
	add("estimateGas", dispatch.Func(a.estimateGas))
	add("createAccessList", dispatch.Func(a.createAccessList))
	add("simulateV1", dispatch.Func(a.simulateV1))

	// sending
	add("sendRawTransaction", dispatch.Func(a.sendRawTransaction))
	add("sendRawTransactionSync", dispatch.Func(a.sendRawTransactionSync))
	add("pendingTransactions", dispatch.Func(a.pendingTransactions))
}
