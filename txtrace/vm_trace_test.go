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

package txtrace

import (
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestComputePushed_EmptyStacks(t *testing.T) {
	pushed := computePushed(vm.STOP, nil)
	require.Empty(t, pushed)
}

func TestComputePushed_PushOntoEmpty(t *testing.T) {
	// PUSH1 0x60 onto empty stack → push = [0x60]
	cur := []uint256.Int{*uint256.NewInt(0x60)}
	pushed := computePushed(vm.PUSH1, cur)
	require.Len(t, pushed, 1)
	require.Equal(t, big.NewInt(0x60), pushed[0].ToInt())
}

// makeStack builds a []uint256.Int from a list of uint64 values.
func makeStack(vals ...uint64) []uint256.Int {
	s := make([]uint256.Int, len(vals))
	for i, v := range vals {
		s[i] = *uint256.NewInt(v)
	}
	return s
}

func TestComputePushed_PushRange(t *testing.T) {
	// Every opcode in PUSH0..PUSH32 returns exactly 1 stack item regardless of its push size.
	tests := []struct {
		name string
		op   vm.OpCode
	}{
		{"PUSH0", vm.PUSH0},
		{"PUSH1", vm.PUSH1},
		{"PUSH16", vm.PUSH16},
		{"PUSH32", vm.PUSH32},
	}
	stack := makeStack(0xAB)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := computePushed(tt.op, stack)
			require.Len(t, result, 1)
			require.Equal(t, big.NewInt(0xAB), result[0].ToInt())
		})
	}
}

func TestComputePushed_SwapRange(t *testing.T) {
	// SWAPn or DUPn reads the top n+1 items: SWAP1 → 2, SWAP16 → 17.
	tests := []struct {
		name      string
		op        vm.OpCode
		wantCount int
	}{
		{"SWAP1", vm.SWAP1, 2},
		{"SWAP4", vm.SWAP4, 5},
		{"SWAP16", vm.SWAP16, 17},
		{"DUP1", vm.DUP1, 2},
		{"DUP4", vm.DUP4, 5},
		{"DUP16", vm.DUP16, 17},
	}
	// Stack large enough for the largest case (17 elements, values 1..17).
	stack := makeStack(1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := computePushed(tt.op, stack)
			require.Len(t, result, tt.wantCount)
			// Result must be the top wantCount elements from the stack in order.
			for i := range tt.wantCount {
				want := big.NewInt(int64(17 - tt.wantCount + 1 + i))
				require.Equal(t, want, result[i].ToInt(), "index %d", i)
			}
		})
	}
}

func TestComputePushed_DupRange(t *testing.T) {
	// DUPn reads the top n+1 items: DUP1 → 2, DUP16 → 17.
	tests := []struct {
		name      string
		op        vm.OpCode
		wantCount int
	}{
		{"DUP1", vm.DUP1, 2},
		{"DUP4", vm.DUP4, 5},
		{"DUP16", vm.DUP16, 17},
	}
	stack := makeStack(1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := computePushed(tt.op, stack)
			require.Len(t, result, tt.wantCount)
			for i := range tt.wantCount {
				want := big.NewInt(int64(17 - tt.wantCount + 1 + i))
				require.Equal(t, want, result[i].ToInt(), "index %d", i)
			}
		})
	}
}

func TestComputePushed_SingleReturnOpcodes(t *testing.T) {
	// A sample of the explicit single-return opcodes in the switch statement.
	ops := []vm.OpCode{
		vm.ADD, vm.MUL, vm.SUB, vm.DIV, vm.AND, vm.OR, vm.XOR, vm.NOT,
		vm.LT, vm.GT, vm.EQ, vm.ISZERO,
		vm.SLOAD, vm.MLOAD, vm.CALLDATALOAD,
		vm.CALLER, vm.CALLVALUE, vm.ADDRESS, vm.ORIGIN,
		vm.GAS, vm.GASLIMIT, vm.GASPRICE, vm.BASEFEE,
		vm.NUMBER, vm.TIMESTAMP, vm.COINBASE,
		vm.KECCAK256, vm.EXTCODESIZE, vm.EXTCODEHASH, vm.BALANCE, vm.SELFBALANCE,
		vm.RETURNDATASIZE, vm.CALLDATASIZE, vm.CODESIZE,
		vm.PC, vm.MSIZE, vm.BLOCKHASH, vm.CHAINID, vm.DIFFICULTY,
	}
	stack := makeStack(99)
	for _, op := range ops {
		t.Run(op.String(), func(t *testing.T) {
			result := computePushed(op, stack)
			require.Len(t, result, 1, "op %s must return exactly 1 item", op)
			require.Equal(t, big.NewInt(99), result[0].ToInt())
		})
	}
}

func TestComputePushed_ZeroReturnOpcodes(t *testing.T) {
	// Opcodes not in any push/swap/dup/explicit list return an empty (non-nil) slice.
	ops := []vm.OpCode{
		vm.STOP, vm.MSTORE, vm.MSTORE8, vm.SSTORE,
		vm.JUMP, vm.JUMPI, vm.JUMPDEST,
		vm.POP, vm.RETURN, vm.REVERT,
		vm.CALL, vm.DELEGATECALL, vm.STATICCALL,
		vm.CREATE, vm.CREATE2, vm.SELFDESTRUCT,
	}
	stack := makeStack(1, 2, 3)
	for _, op := range ops {
		t.Run(op.String(), func(t *testing.T) {
			result := computePushed(op, stack)
			require.NotNil(t, result)
			require.Empty(t, result, "op %s must return no items", op)
		})
	}
}

func TestComputePushed_StackClamp(t *testing.T) {
	// When the requested count exceeds the actual stack depth, it is clamped to len(stack).
	tests := []struct {
		name      string
		op        vm.OpCode
		stack     []uint256.Int
		wantCount int
	}{
		// SWAP2 wants 3, but stack has only 2 → clamp to 2.
		{"SWAP2 stack too small", vm.SWAP2, makeStack(10, 20), 2},
		// PUSH1 wants 1, but stack is empty → 0.
		{"PUSH1 empty stack", vm.PUSH1, nil, 0},
		// DUP4 wants 5, stack has 3 → clamp to 3.
		{"DUP4 stack too small", vm.DUP4, makeStack(1, 2, 3), 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := computePushed(tt.op, tt.stack)
			require.Len(t, result, tt.wantCount)
		})
	}
}

func TestVmTraceLogger_GetResultBeforeExecution(t *testing.T) {
	l := NewVmTraceLogger()
	require.Nil(t, l.GetResult(), "result must be nil before any execution")
}

func TestVmTraceLogger_EmptyFrame(t *testing.T) {
	// OnEnter + OnExit with no opcodes → trace with empty ops
	l := NewVmTraceLogger()

	code := []byte{0x60, 0x01} // PUSH1 1
	l.onEnter(0, 0x00 /* STOP */, addr(1), addr(2), code, 100, big.NewInt(0))
	l.onExit(0, nil, 0, nil, false)

	result := l.GetResult()
	require.NotNil(t, result)
	require.Empty(t, result.Ops)
}

func TestVmTraceLogger_OnFaultSetsExToNil(t *testing.T) {
	l := NewVmTraceLogger()

	l.onEnter(0, 0x00, addr(1), addr(2), []byte{0x60}, 1000, big.NewInt(0))

	// Simulate one opcode that then faults
	l.onOpcode(0, 0x60 /* PUSH1 */, 1000, 3, &mockOpContext{}, nil, 0, nil)
	l.onFault(0, 0x60, 1000, 3, &mockOpContext{}, 0, errFoo)

	l.onExit(0, nil, 3, errFoo, true)

	result := l.GetResult()
	require.NotNil(t, result)
	require.Len(t, result.Ops, 1)
	require.Nil(t, result.Ops[0].Ex, "Ex must be nil after a fault")
}

func TestVmTraceLogger_StorageChangeAttributed(t *testing.T) {
	l := NewVmTraceLogger()
	l.onEnter(0, 0x00, addr(1), addr(2), nil, 1000, big.NewInt(0))

	ctx1 := &mockOpContext{}
	l.onOpcode(0, 0x55 /* SSTORE */, 1000, 20000, ctx1, nil, 0, nil)

	slot := common.Hash{0x01}
	val := common.Hash{0xFF}
	l.OnStorageChange(addr(2), slot, common.Hash{}, val)

	// Next op finalizes the SSTORE op's Ex (including store)
	ctx2 := &mockOpContext{}
	l.onOpcode(4, 0x00 /* STOP */, 980, 0, ctx2, nil, 0, nil)

	l.onExit(0, nil, 20000, nil, false)

	result := l.GetResult()
	require.NotNil(t, result)
	require.Len(t, result.Ops, 2)

	sstoreOp := result.Ops[0]
	require.NotNil(t, sstoreOp.Ex)
	require.NotNil(t, sstoreOp.Ex.Store)
	require.Equal(t, slot, sstoreOp.Ex.Store.Key)
	require.Equal(t, val, sstoreOp.Ex.Store.Val)
}

func TestVmTraceLogger_GasAccountingUsed(t *testing.T) {
	l := NewVmTraceLogger()
	l.onEnter(0, 0x00, addr(1), addr(2), nil, 1000, big.NewInt(0))

	// PUSH1 costs 3 gas; gas parameter = gas before this op = 1000
	l.onOpcode(0, 0x60, 1000, 3, &mockOpContext{}, nil, 0, nil)
	// Next op: gas=997 (gas before STOP = gas after PUSH1)
	l.onOpcode(2, 0x00, 997, 0, &mockOpContext{}, nil, 0, nil)

	l.onExit(0, nil, 3, nil, false)

	result := l.GetResult()
	require.NotNil(t, result)
	require.Len(t, result.Ops, 2)

	// PUSH1 op Ex.Used = gas before PUSH1 = 1000
	require.Equal(t, uint64(1000), result.Ops[0].Ex.Used)

	// STOP op Ex.Used = gas before STOP = 997
	require.Equal(t, uint64(997), result.Ops[1].Ex.Used)
}

func TestVmTraceLogger_SubTraceLinked(t *testing.T) {
	l := NewVmTraceLogger()

	// Root call
	l.onEnter(0, 0x00, addr(1), addr(2), []byte{}, 1000, big.NewInt(0))

	// CALL opcode in root
	l.onOpcode(0, 0xF1 /* CALL */, 1000, 100, &mockOpContext{}, nil, 0, nil)

	// Sub-call enters (depth=1)
	subCode := []byte{0x60, 0x01}
	l.onEnter(1, 0xF1, addr(2), addr(3), subCode, 800, big.NewInt(0))
	l.onExit(1, nil, 10, nil, false)

	// Back in root: next op finalizes the CALL op (with Sub linked)
	l.onOpcode(10, 0x00 /* STOP */, 890, 0, &mockOpContext{}, nil, 0, nil)
	l.onExit(0, nil, 110, nil, false)

	result := l.GetResult()
	require.NotNil(t, result)
	require.Len(t, result.Ops, 2)

	callOp := result.Ops[0]
	require.NotNil(t, callOp.Sub, "CALL op must have a sub-trace")
	require.Empty(t, callOp.Sub.Ops, "sub-trace had no opcodes")
}

func TestVmTraceLogger_StorageChangeOnLastOp(t *testing.T) {
	// Verifies that a storage change from an SSTORE that is the very last opcode
	// (no subsequent onOpcode call before onExit) is correctly attributed to Ex.Store.
	l := NewVmTraceLogger()
	l.onEnter(0, 0x00, addr(1), addr(2), nil, 1000, big.NewInt(0))

	// SSTORE is the only (and last) opcode — no subsequent onOpcode drains pendingStore.
	l.onOpcode(0, 0x55 /* SSTORE */, 1000, 20000, &mockOpContext{}, nil, 0, nil)

	slot := common.Hash{0x02}
	val := common.Hash{0xAB}
	l.OnStorageChange(addr(2), slot, common.Hash{}, val)

	// Frame exits immediately; onExit must consume pendingStore.
	l.onExit(0, nil, 20000, nil, false)

	result := l.GetResult()
	require.NotNil(t, result)
	require.Len(t, result.Ops, 1)

	sstoreOp := result.Ops[0]
	require.NotNil(t, sstoreOp.Ex, "Ex must not be nil for a non-faulting op")
	require.NotNil(t, sstoreOp.Ex.Store, "Ex.Store must be populated even when SSTORE is the last op")
	require.Equal(t, slot, sstoreOp.Ex.Store.Key)
	require.Equal(t, val, sstoreOp.Ex.Store.Val)
}

func TestVmTraceLogger_NoOpsOnEmptyExec(t *testing.T) {
	l := NewVmTraceLogger()
	// Guard: multiple OnExit calls without matching OnEnter must not panic
	l.onExit(0, nil, 0, nil, false)
	require.Nil(t, l.GetResult())
}

func TestVmTraceLogger_RevertHasNonNilEx(t *testing.T) {
	// REVERT is a valid opcode execution; onFault with ErrExecutionReverted must not nil Ex.
	l := NewVmTraceLogger()
	l.onEnter(0, 0x00, addr(1), addr(2), nil, 1000, big.NewInt(0))
	l.onOpcode(0, 0xfd /* REVERT */, 1000, 0, &mockOpContext{}, nil, 0, nil)
	l.onFault(0, 0xfd, 1000, 0, &mockOpContext{}, 0, vm.ErrExecutionReverted)
	l.onExit(0, nil, 0, vm.ErrExecutionReverted, true)

	result := l.GetResult()
	require.NotNil(t, result)
	require.Len(t, result.Ops, 1)
	require.NotNil(t, result.Ops[0].Ex, "REVERT must have non-nil Ex")
	require.Equal(t, uint64(1000), result.Ops[0].Ex.Used)
}

// --- helpers ---

var errFoo = errors.New("test error")

func addr(n byte) common.Address {
	return common.Address{n}
}

// mockOpContext implements tracing.OpContext with configurable stack and memory.
type mockOpContext struct {
	stack  []uint256.Int
	memory []byte
}

func (m *mockOpContext) MemoryData() []byte       { return m.memory }
func (m *mockOpContext) StackData() []uint256.Int { return m.stack }
func (m *mockOpContext) Caller() common.Address   { return common.Address{} }
func (m *mockOpContext) Address() common.Address  { return common.Address{} }
func (m *mockOpContext) CallValue() *uint256.Int  { return uint256.NewInt(0) }
func (m *mockOpContext) CallInput() []byte        { return nil }
func (m *mockOpContext) ContractCode() []byte     { return nil }
