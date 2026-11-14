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
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/stretchr/testify/require"
)

// A failed frame drops its logs and frees their indices; a frame that
// succeeds hands its logs to the frame around it, and the transaction keeps
// the logs of its outermost frame unless that failed.
func TestLogCollector_KeepsTheLogsOfFramesThatSucceed(t *testing.T) {
	first, second, third := common.Address{0x01}, common.Address{0x02}, common.Address{0x03}
	collector := newLogCollector(false, 7, 8)
	hooks := collector.hooks()
	emit := func(address common.Address) { hooks.OnLog(&types.Log{Address: address}) }

	collector.startTransaction(common.Hash{0x0a}, 1)
	hooks.OnEnter(0, byte(vm.CALL), common.Address{}, first, nil, 0, nil)
	emit(first)
	hooks.OnEnter(1, byte(vm.CALL), first, second, nil, 0, nil)
	emit(second)
	hooks.OnExit(1, nil, 0, vm.ErrExecutionReverted, true)
	hooks.OnEnter(1, byte(vm.CALL), first, third, nil, 0, nil)
	emit(third)
	hooks.OnExit(1, nil, 0, nil, false)
	hooks.OnExit(0, nil, 0, nil, false)
	logs := collector.logs()
	require.Len(t, logs, 2)
	for i, address := range []common.Address{first, third} {
		require.Equal(t, &types.Log{Address: address, BlockNumber: 7, BlockTimestamp: 8, TxHash: common.Hash{0x0a}, TxIndex: 1, Index: uint(i)}, logs[i])
	}

	collector.startTransaction(common.Hash{0x0b}, 2)
	hooks.OnEnter(0, byte(vm.CALL), common.Address{}, first, nil, 0, nil)
	emit(first)
	hooks.OnExit(0, nil, 0, vm.ErrExecutionReverted, true)
	require.Empty(t, collector.logs())

	collector.startTransaction(common.Hash{0x0c}, 3)
	hooks.OnEnter(0, byte(vm.CALL), common.Address{}, first, nil, 0, nil)
	emit(first)
	hooks.OnExit(0, nil, 0, nil, false)
	require.Equal(t, uint(2), collector.logs()[0].Index, "the indices run through the block")
}

// Transfers are logged if asked, except those of a delegate call or call
// code, which move no value of their own.
func TestLogCollector_LogsTransfersIfAsked(t *testing.T) {
	value := big.NewInt(5)
	cases := map[string]struct {
		trace bool
		op    vm.OpCode
		value *big.Int
		logs  int
	}{
		"not asked":     {false, vm.CALL, value, 0},
		"call":          {true, vm.CALL, value, 1},
		"no value":      {true, vm.CALL, new(big.Int), 0},
		"delegate call": {true, vm.DELEGATECALL, value, 0},
		"call code":     {true, vm.CALLCODE, value, 0},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			collector := newLogCollector(test.trace, 0, 0)
			hooks := collector.hooks()
			collector.startTransaction(common.Hash{}, 0)
			hooks.OnEnter(0, byte(test.op), common.Address{0x01}, common.Address{0x02}, nil, 0, test.value)
			hooks.OnExit(0, nil, 0, nil, false)
			require.Len(t, collector.logs(), test.logs)
		})
	}
}
