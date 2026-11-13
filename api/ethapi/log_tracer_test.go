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

package ethapi

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	gethstate "github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/vm/program"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestSimTracer_CaptureTransfer_CreatesExpectedLog(t *testing.T) {
	from := common.Address{1}
	to := common.Address{2}
	value := big.NewInt(1_000_000)

	blockNum := uint64(5)
	blockTs := uint64(1000)
	blockHash := common.Hash{3}
	txHash := common.Hash{4}

	tracer := newSimTracer(true, blockNum, blockTs, blockHash, txHash, 0)

	tracer.logs = [][]*types.Log{nil}
	tracer.captureTransfer(from, to, value)

	require.Len(t, tracer.logs, 1)
	require.Len(t, tracer.logs[0], 1)
	log := tracer.logs[0][0]
	require.Equal(t, simTransferAddress, log.Address)
	require.Equal(t, simTransferTopic, log.Topics[0])
	require.Equal(t, common.BytesToHash(from.Bytes()), log.Topics[1])
	require.Equal(t, common.BytesToHash(to.Bytes()), log.Topics[2])
	require.Equal(t, blockNum, log.BlockNumber)
	require.Equal(t, blockTs, log.BlockTimestamp)
	require.Equal(t, blockHash, log.BlockHash)
	require.Equal(t, txHash, log.TxHash)
}

func TestSimTracer_CaptureTransfer_NoOpWhenTracingDisabled(t *testing.T) {
	tracer := newSimTracer(false, 1, 0, common.Hash{}, common.Hash{}, 0)

	tracer.captureTransfer(common.Address{1}, common.Address{2}, big.NewInt(1))

	require.Empty(t, tracer.logs)
}

func TestSimTracer_Reset_ClearsLogsAndUpdatesContext(t *testing.T) {
	tracer := newSimTracer(true, 1, 0, common.Hash{1}, common.Hash{2}, 0)
	tracer.logs = [][]*types.Log{{{Address: common.Address{1}}}}

	newHash := common.Hash{42}
	newIdx := uint(3)
	tracer.reset(newHash, newIdx)

	require.Empty(t, tracer.Logs())
	require.Equal(t, newHash, tracer.txHash)
	require.Equal(t, newIdx, tracer.txIdx)
}

func TestSimTracer_Hooks_ReturnsNonNilWhenTracingEnabled(t *testing.T) {
	tracer := newSimTracer(true, 1, 0, common.Hash{}, common.Hash{}, 0)
	require.NotNil(t, tracer.Hooks())
}

var (
	simTracerSender   = common.Address{0x01}
	simTracerEntry    = common.Address{0x0a}
	simTracerReverter = common.Address{0x0b}
	simTracerLogger   = common.Address{0x0c}
)

func TestSimTracer_ReportsOnlyLogsAndTransfersThatHappened(t *testing.T) {
	sender, entry, reverter, logger := simTracerSender, simTracerEntry, simTracerReverter, simTracerLogger

	tests := map[string]struct {
		entryCode []byte
		wantErr   error
		want      []simLog
	}{
		"reverted call with a successful inner call": {
			entryCode: emitLog(program.New().Call(nil, reverter, 1, 0, 0, 0, 0).Op(vm.POP), 0xaa).Op(vm.STOP).Bytes(),
			want: []simLog{
				transferLog(sender, entry, 5, 0),
				emittedLog(entry, 0xaa, 4),
			},
		},
		"failed transaction with a successful inner call": {
			entryCode: program.New().Call(nil, logger, 1, 0, 0, 0, 0).Op(vm.POP).Push(0).Push(0).Op(vm.REVERT).Bytes(),
			wantErr:   vm.ErrExecutionReverted,
		},
		"successful inner call": {
			entryCode: emitLog(program.New().Call(nil, logger, 1, 0, 0, 0, 0).Op(vm.POP), 0xaa).Op(vm.STOP).Bytes(),
			want: []simLog{
				transferLog(sender, entry, 5, 0),
				transferLog(entry, logger, 1, 1),
				emittedLog(logger, 0xcc, 2),
				emittedLog(entry, 0xaa, 3),
			},
		},
		"delegatecall moves no value": {
			entryCode: program.New().DelegateCall(nil, logger, 0, 0, 0, 0).Op(vm.POP, vm.STOP).Bytes(),
			want: []simLog{
				transferLog(sender, entry, 5, 0),
				emittedLog(entry, 0xcc, 1),
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			tracer := newSimTracer(true, 1, 0, common.Hash{}, common.Hash{}, 0)
			err := runWithSimTracer(t, tracer, test.entryCode, 5)
			require.ErrorIs(t, err, test.wantErr)
			require.Equal(t, test.want, toSimLogs(tracer.Logs()))
		})
	}
}

// runWithSimTracer runs a call from simTracerSender to simTracerEntry, which
// runs entryCode, on go-ethereum's interpreter with the tracer attached.
func runWithSimTracer(t *testing.T, tracer *simTracer, entryCode []byte, value uint64) error {
	t.Helper()
	statedb, err := gethstate.New(types.EmptyRootHash, gethstate.NewDatabaseForTesting())
	require.NoError(t, err)

	reverterCode := emitLog(program.New().Call(nil, simTracerLogger, 0, 0, 0, 0, 0).Op(vm.POP), 0xbb).Push(0).Push(0).Op(vm.REVERT)
	loggerCode := emitLog(program.New(), 0xcc).Op(vm.STOP)
	statedb.SetCode(simTracerEntry, entryCode, tracing.CodeChangeUnspecified)
	statedb.SetCode(simTracerReverter, reverterCode.Bytes(), tracing.CodeChangeUnspecified)
	statedb.SetCode(simTracerLogger, loggerCode.Bytes(), tracing.CodeChangeUnspecified)
	statedb.AddBalance(simTracerSender, uint256.NewInt(1_000_000), tracing.BalanceChangeUnspecified)

	hooks := tracer.Hooks()
	blockCtx := vm.BlockContext{
		CanTransfer: core.CanTransfer,
		Transfer:    core.Transfer,
		BlockNumber: big.NewInt(1),
		Difficulty:  big.NewInt(0),
		Random:      &common.Hash{},
		BaseFee:     big.NewInt(0),
		GasLimit:    10_000_000,
	}
	chainConfig := makeChainConfig(opera.GetBrioUpgrades())
	evm := vm.NewEVM(blockCtx, gethstate.NewHookedState(statedb, hooks), chainConfig, vm.Config{Tracer: hooks})
	_, _, err = evm.Call(simTracerSender, simTracerEntry, nil, 1_000_000, uint256.NewInt(value))
	return err
}

func emitLog(p *program.Program, topic byte) *program.Program {
	return p.Push(common.Hash{topic}).Push(0).Push(0).Op(vm.LOG1)
}

type simLog struct {
	Address common.Address
	Topics  []common.Hash
	Data    string
	Index   uint
}

func toSimLogs(logs []*types.Log) []simLog {
	var res []simLog
	for _, log := range logs {
		res = append(res, simLog{log.Address, log.Topics, string(log.Data), log.Index})
	}
	return res
}

func transferLog(from, to common.Address, value int64, index uint) simLog {
	return simLog{
		Address: simTransferAddress,
		Topics:  []common.Hash{simTransferTopic, common.BytesToHash(from.Bytes()), common.BytesToHash(to.Bytes())},
		Data:    string(common.BigToHash(big.NewInt(value)).Bytes()),
		Index:   index,
	}
}

func emittedLog(address common.Address, topic byte, index uint) simLog {
	return simLog{Address: address, Topics: []common.Hash{{topic}}, Index: index}
}
