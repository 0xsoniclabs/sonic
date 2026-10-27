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

package trace

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/tests"
)

// TestTracedRun_StopsTheExecutionAtItsBudget holds V12: a vmTrace over its
// budget stops the run before the execution ends, not after it.
func TestTracedRun_StopsTheExecutionAtItsBudget(t *testing.T) {
	looper := common.HexToAddress("0xc3")
	signer := tests.NewSigner(t)
	carmenCore := tests.NewCarmenCore(t, map[common.Address]tests.Account{
		signer.Address: {Balance: new(big.Int).Lsh(big.NewInt(1), 100)},
		looper:         {Code: []byte{byte(vm.JUMPDEST), byte(vm.PUSH1), 0, byte(vm.JUMP)}},
	})
	carmenCore.AppendBlock(tests.BlockSpec{})
	carmenCore.PublishHead(1)
	var environment *exec.Environment
	harness := tests.NewHarness(t, carmenCore.FakeCore, func(_ *dispatch.Dispatcher, registered *exec.Environment) {
		environment = registered
	})
	harness.WaitForHead(1)

	head, err := environment.Latest(t.Context())
	require.NoError(t, err)
	latest := chain.LatestBlockNumber
	block, err := environment.CallBlock(head, chain.BlockReference{Number: &latest})
	require.NoError(t, err)
	_, statedb, call, err := environment.StateAtTransaction(head, block, 0)
	require.NoError(t, err)
	defer statedb.Release()
	gas := hexutil.Uint64(10_000_000)
	arguments := exec.CallArguments{From: &signer.Address, Recipient: &looper, Gas: &gas}
	transaction, message, err := arguments.ToTracedCall(environment.Config().RPCGasCap, block.BaseFee)
	require.NoError(t, err)

	run := newTracedRun(block, statedb, 0, traceTypes{vmTrace: true}, 10_000)
	executed := 0
	run.hooks = mergeHooks(run.hooks, &tracing.Hooks{OnOpcode: func(uint64, byte, uint64, uint64, tracing.OpContext, []byte, int, error) {
		executed++
	}})
	evm, stop := call.NewEVM(t.Context(), run.statedb, run.hooks)
	defer stop()
	run.evm = evm
	_, err = run.apply(transaction, message, 0)
	require.ErrorIs(t, err, errResponseTooLarge)
	require.Less(t, executed, 1_000, "the loop would run %d operations to its end", gas/4)
}

func TestMergeHooks_RunsBothInOrder(t *testing.T) {
	var calls []string
	record := func(name string) *tracing.Hooks {
		return &tracing.Hooks{
			OnTxStart: func(*tracing.VMContext, *types.Transaction, common.Address) { calls = append(calls, name+" start") },
			OnTxEnd:   func(*types.Receipt, error) { calls = append(calls, name+" end") },
			OnEnter: func(int, byte, common.Address, common.Address, []byte, uint64, *big.Int) {
				calls = append(calls, name+" enter")
			},
			OnExit: func(int, []byte, uint64, error, bool) { calls = append(calls, name+" exit") },
			OnOpcode: func(uint64, byte, uint64, uint64, tracing.OpContext, []byte, int, error) {
				calls = append(calls, name+" opcode")
			},
			OnFault: func(uint64, byte, uint64, uint64, tracing.OpContext, int, error) {
				calls = append(calls, name+" fault")
			},
		}
	}
	merged := mergeHooks(record("first"), record("second"))
	merged.OnTxStart(nil, nil, common.Address{})
	merged.OnTxEnd(nil, nil)
	merged.OnEnter(0, 0, common.Address{}, common.Address{}, nil, 0, nil)
	merged.OnExit(0, nil, 0, nil, false)
	merged.OnOpcode(0, 0, 0, 0, nil, nil, 0, nil)
	merged.OnFault(0, 0, 0, 0, nil, 0, nil)
	require.Equal(t, []string{
		"first start", "second start", "first end", "second end", "first enter", "second enter",
		"first exit", "second exit", "first opcode", "second opcode", "first fault", "second fault",
	}, calls)

	only := record("only")
	require.Same(t, only, mergeHooks(nil, only))
	require.Same(t, only, mergeHooks(only, nil))
	partial := mergeHooks(&tracing.Hooks{}, only)
	calls = nil
	partial.OnFault(0, 0, 0, 0, nil, 0, nil)
	require.Equal(t, []string{"only fault"}, calls)
}
