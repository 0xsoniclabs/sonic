// Copyright 2024 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package ethapi

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
)

var (
	// keccak256("Transfer(address,address,uint256)")
	simTransferTopic = common.HexToHash("ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")
	// ERC-7528 canonical address for native ETH/Sonic events
	simTransferAddress = common.HexToAddress("0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE")
)

type callLogs struct {
	logs []*types.Log
}

// simTracer collects logs for simulated transactions.
type simTracer struct {
	calls          []*callLogs
	count          int
	traceTransfers bool
	blockNumber    uint64
	blockTimestamp uint64
	blockHash      common.Hash
	txHash         common.Hash
	txIdx          uint
}

func newSimTracer(traceTransfers bool, blockNumber, blockTimestamp uint64, blockHash, txHash common.Hash, txIndex uint) *simTracer {
	return &simTracer{
		traceTransfers: traceTransfers,
		blockNumber:    blockNumber,
		blockTimestamp: blockTimestamp,
		blockHash:      blockHash,
		txHash:         txHash,
		txIdx:          txIndex,
	}
}

// Hooks returns the tracing hooks for the simTracer.
// They are triggered during EVM execution to capture logs.
func (t *simTracer) Hooks() *tracing.Hooks {
	return &tracing.Hooks{
		OnEnter: t.onEnter,
		OnExit:  t.onExit,
		OnLog:   t.onLog,
	}
}

// onEnter is called when the EVM enters a new call frame. It captures transfer
func (t *simTracer) onEnter(depth int, typ byte, from, to common.Address, input []byte, gas uint64, value *big.Int) {
	t.calls = append(t.calls, &callLogs{})
	if vm.OpCode(typ) != vm.DELEGATECALL && value != nil && value.Cmp(common.Big0) > 0 {

		t.captureTransfer(from, to, value)
	}
}

// onExit is called when the EVM exits a call frame.
func (t *simTracer) onExit(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
	if reverted {
		t.calls = t.calls[:len(t.calls)-1]
	}
}

func (t *simTracer) onLog(log *types.Log) {
	t.captureLog(log.Address, log.Topics, log.Data)
}

func (t *simTracer) captureLog(address common.Address, topics []common.Hash, data []byte) {
	t.calls[len(t.calls)-1].logs = append(t.calls[len(t.calls)-1].logs, &types.Log{
		Address:        address,
		Topics:         topics,
		Data:           data,
		BlockNumber:    t.blockNumber,
		BlockTimestamp: t.blockTimestamp,
		BlockHash:      t.blockHash,
		TxHash:         t.txHash,
		TxIndex:        t.txIdx,
		Index:          uint(t.count),
	})
	t.count++
}

func (t *simTracer) captureTransfer(from, to common.Address, value *big.Int) {
	if !t.traceTransfers {
		return
	}
	topics := []common.Hash{
		simTransferTopic,
		common.BytesToHash(from.Bytes()),
		common.BytesToHash(to.Bytes()),
	}
	t.captureLog(simTransferAddress, topics, common.BigToHash(value).Bytes())
}

func (t *simTracer) reset(txHash common.Hash, txIdx uint) {
	t.calls = nil
	t.txHash = txHash
	t.txIdx = txIdx
}

// Logs returns the collected logs from the tracer.
func (t *simTracer) Logs() []*types.Log {
	logs := []*types.Log{}
	for _, call := range t.calls {
		logs = append(logs, call.logs...)
	}
	return logs
}
