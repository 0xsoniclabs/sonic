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

package eth

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
)

var (
	// transferTopic is keccak256("Transfer(address,address,uint256)").
	transferTopic = common.HexToHash("ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")
	// transferAddress is the address of ERC-7528 for the native token.
	transferAddress = common.HexToAddress("0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE")
)

// frameLogs are the logs of one call frame.
type frameLogs struct {
	logs []*types.Log
}

// simulationTracer collects the logs of a simulated call, and the transfers
// of the native token as logs when asked to. A reverted frame drops its logs.
type simulationTracer struct {
	frames           []*frameLogs
	count            int
	traceTransfers   bool
	blockNumber      uint64
	blockTimestamp   uint64
	transactionHash  common.Hash
	transactionIndex uint
}

func newSimulationTracer(traceTransfers bool, blockNumber, blockTimestamp uint64) *simulationTracer {
	return &simulationTracer{traceTransfers: traceTransfers, blockNumber: blockNumber, blockTimestamp: blockTimestamp}
}

// Hooks returns the hooks the EVM reports frames and logs to.
func (tracer *simulationTracer) Hooks() *tracing.Hooks {
	return &tracing.Hooks{OnEnter: tracer.onEnter, OnExit: tracer.onExit, OnLog: tracer.onLog}
}

func (tracer *simulationTracer) onEnter(_ int, kind byte, from, recipient common.Address, _ []byte, _ uint64, value *big.Int) {
	tracer.frames = append(tracer.frames, &frameLogs{})
	if vm.OpCode(kind) != vm.DELEGATECALL && value != nil && value.Cmp(common.Big0) > 0 {
		tracer.captureTransfer(from, recipient, value)
	}
}

func (tracer *simulationTracer) onExit(_ int, _ []byte, _ uint64, _ error, reverted bool) {
	if reverted {
		tracer.frames = tracer.frames[:len(tracer.frames)-1]
	}
}

func (tracer *simulationTracer) onLog(log *types.Log) {
	tracer.captureLog(log.Address, log.Topics, log.Data)
}

func (tracer *simulationTracer) captureLog(address common.Address, topics []common.Hash, data []byte) {
	frame := tracer.frames[len(tracer.frames)-1]
	frame.logs = append(frame.logs, &types.Log{
		Address:        address,
		Topics:         topics,
		Data:           data,
		BlockNumber:    tracer.blockNumber,
		BlockTimestamp: tracer.blockTimestamp,
		TxHash:         tracer.transactionHash,
		TxIndex:        tracer.transactionIndex,
		Index:          uint(tracer.count),
	})
	tracer.count++
}

func (tracer *simulationTracer) captureTransfer(from, recipient common.Address, value *big.Int) {
	if !tracer.traceTransfers {
		return
	}
	topics := []common.Hash{transferTopic, common.BytesToHash(from.Bytes()), common.BytesToHash(recipient.Bytes())}
	tracer.captureLog(transferAddress, topics, common.BigToHash(value).Bytes())
}

// reset starts the logs of the next transaction.
func (tracer *simulationTracer) reset(transactionHash common.Hash, transactionIndex uint) {
	tracer.frames = nil
	tracer.transactionHash = transactionHash
	tracer.transactionIndex = transactionIndex
}

// Logs returns the logs of the frames that did not revert.
func (tracer *simulationTracer) Logs() []*types.Log {
	logs := []*types.Log{}
	for _, frame := range tracer.frames {
		logs = append(logs, frame.logs...)
	}
	return logs
}
