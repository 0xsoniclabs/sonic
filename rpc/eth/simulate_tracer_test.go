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

// The tests of the reference's log_tracer.go, ported.

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/stretchr/testify/require"
)

func TestSimulationTracer_CapturesATransferAsALog(t *testing.T) {
	sender, recipient := common.Address{1}, common.Address{2}
	tracer := newSimulationTracer(true, 5, 1000)
	tracer.reset(common.Hash{4}, 3)
	tracer.onEnter(0, byte(vm.CALL), sender, recipient, nil, 0, big.NewInt(1_000_000))

	logs := tracer.Logs()
	require.Len(t, logs, 1)
	require.Equal(t, transferAddress, logs[0].Address)
	require.Equal(t, []common.Hash{transferTopic, common.BytesToHash(sender.Bytes()), common.BytesToHash(recipient.Bytes())}, logs[0].Topics)
	require.Equal(t, common.BigToHash(big.NewInt(1_000_000)).Bytes(), logs[0].Data)
	require.Equal(t, uint64(5), logs[0].BlockNumber)
	require.Equal(t, uint64(1000), logs[0].BlockTimestamp)
	require.Equal(t, common.Hash{4}, logs[0].TxHash)
	require.Equal(t, uint(3), logs[0].TxIndex)
}

func TestSimulationTracer_CapturesNoTransferUnlessAsked(t *testing.T) {
	tracer := newSimulationTracer(false, 1, 0)
	tracer.onEnter(0, byte(vm.CALL), common.Address{1}, common.Address{2}, nil, 0, big.NewInt(1))
	require.Empty(t, tracer.Logs())

	traced := newSimulationTracer(true, 1, 0)
	traced.onEnter(0, byte(vm.DELEGATECALL), common.Address{1}, common.Address{2}, nil, 0, big.NewInt(1))
	traced.onEnter(1, byte(vm.CALL), common.Address{1}, common.Address{2}, nil, 0, big.NewInt(0))
	require.Empty(t, traced.Logs(), "a delegate call and a call without value transfer nothing")
}

func TestSimulationTracer_DropsTheLogsOfARevertedFrame(t *testing.T) {
	tracer := newSimulationTracer(true, 1, 0)
	tracer.onEnter(0, byte(vm.CALL), common.Address{1}, common.Address{2}, nil, 0, nil)
	tracer.onLog(&types.Log{Address: common.Address{7}})
	tracer.onEnter(1, byte(vm.CALL), common.Address{2}, common.Address{3}, nil, 0, nil)
	tracer.onLog(&types.Log{Address: common.Address{8}})
	tracer.onExit(1, nil, 0, nil, true)
	tracer.onExit(0, nil, 0, nil, false)

	logs := tracer.Logs()
	require.Len(t, logs, 1, "the reverted inner frame's log is dropped")
	require.Equal(t, common.Address{7}, logs[0].Address)
	require.Equal(t, uint(0), logs[0].Index)

	tracer.onEnter(0, byte(vm.CALL), common.Address{1}, common.Address{2}, nil, 0, nil)
	tracer.onLog(&types.Log{Address: common.Address{9}})
	tracer.onExit(0, nil, 0, nil, true)
	require.Len(t, tracer.Logs(), 1, "a reverted top frame drops its own logs")
}

func TestSimulationTracer_ResetStartsTheNextTransaction(t *testing.T) {
	tracer := newSimulationTracer(true, 1, 0)
	tracer.onEnter(0, byte(vm.CALL), common.Address{1}, common.Address{2}, nil, 0, nil)
	tracer.onLog(&types.Log{})
	tracer.reset(common.Hash{42}, 3)
	require.Empty(t, tracer.Logs())
	tracer.onEnter(0, byte(vm.CALL), common.Address{1}, common.Address{2}, nil, 0, nil)
	tracer.onLog(&types.Log{})
	logs := tracer.Logs()
	require.Equal(t, common.Hash{42}, logs[0].TxHash)
	require.Equal(t, uint(1), logs[0].Index, "the index counts the logs of the block")
}
