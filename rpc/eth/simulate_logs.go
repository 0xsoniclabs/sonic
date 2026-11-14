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

	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
)

var (
	// transferTopic is keccak256("Transfer(address,address,uint256)").
	transferTopic = common.HexToHash("ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")
	// transferAddress is the address ERC-7528 gives the native currency.
	transferAddress = common.HexToAddress("0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE")
)

// logCollector collects the logs of a simulated transaction, dropping those
// of a failed call frame, and the transfers of value as logs if asked.
type logCollector struct {
	// frames holds the logs of every open call frame, innermost last; once
	// the transaction ends, its logs.
	frames         [][]*types.Log
	count          int
	traceTransfers bool
	blockNumber    uint64
	blockTimestamp uint64
	txHash         common.Hash
	txIndex        uint
}

func newLogCollector(traceTransfers bool, blockNumber, blockTimestamp uint64) *logCollector {
	return &logCollector{traceTransfers: traceTransfers, blockNumber: blockNumber, blockTimestamp: blockTimestamp}
}

func (c *logCollector) hooks() *tracing.Hooks {
	return &tracing.Hooks{OnEnter: c.onEnter, OnExit: c.onExit, OnLog: c.onLog}
}

func (c *logCollector) onEnter(depth int, kind byte, from, to common.Address, input []byte, gas uint64, value *big.Int) {
	c.frames = append(c.frames, nil)
	op := vm.OpCode(kind)
	if op != vm.DELEGATECALL && op != vm.CALLCODE && value != nil && value.Sign() > 0 && c.traceTransfers {
		c.collect(transferAddress, []common.Hash{transferTopic, common.BytesToHash(from.Bytes()), common.BytesToHash(to.Bytes())},
			common.BigToHash(value).Bytes())
	}
}

// onExit closes a call frame: the outermost one keeps its logs unless it
// failed, an inner one hands them to the frame around it.
func (c *logCollector) onExit(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
	last := len(c.frames) - 1
	frame := c.frames[last]
	if reverted {
		// The logs of a failed frame are the latest ones, so their indices
		// are free again.
		c.count -= len(frame)
		frame = nil
	}
	if depth == 0 {
		c.frames[0] = frame
		return
	}
	c.frames = c.frames[:last]
	c.frames[last-1] = append(c.frames[last-1], frame...)
}

func (c *logCollector) onLog(log *types.Log) {
	c.collect(log.Address, log.Topics, log.Data)
}

func (c *logCollector) collect(address common.Address, topics []common.Hash, data []byte) {
	c.frames[len(c.frames)-1] = append(c.frames[len(c.frames)-1], &types.Log{
		Address:        address,
		Topics:         topics,
		Data:           data,
		BlockNumber:    c.blockNumber,
		BlockTimestamp: c.blockTimestamp,
		TxHash:         c.txHash,
		TxIndex:        c.txIndex,
		Index:          uint(c.count),
	})
	c.count++
}

// startTransaction begins the logs of the next transaction.
func (c *logCollector) startTransaction(txHash common.Hash, txIndex uint) {
	c.frames = nil
	c.txHash = txHash
	c.txIndex = txIndex
}

// logs are the logs of the last transaction that were not reverted. Every
// transaction enters the EVM, which opens its outermost frame.
func (c *logCollector) logs() []*types.Log {
	return c.frames[0]
}

// loggingState reports the logs a state receives to the collector, as the
// EVM leaves that to the state.
type loggingState struct {
	state.StateDB
	collector *logCollector
}

func (s *loggingState) AddLog(log *types.Log) {
	s.StateDB.AddLog(log)
	s.collector.onLog(log)
}
