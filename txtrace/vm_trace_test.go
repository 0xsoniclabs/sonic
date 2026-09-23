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
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/vm/runtime"
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
		vm.POP, vm.RETURN, vm.REVERT, vm.SELFDESTRUCT,
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

func TestVmTraceLogger_SizeLimitDropsTrace(t *testing.T) {
	l := NewVmTraceLogger()
	l.sizeLimit = 2*vmTraceOpSize + 16

	l.onEnter(0, 0x00, addr(1), addr(2), nil, 1000, big.NewInt(0))
	l.onOpcode(0, byte(vm.MSTORE), 1000, 6, &mockOpContext{stack: makeStack(0, 0)}, nil, 0, nil)
	require.NoError(t, l.Err())

	// The MSTORE's 32 byte diff plus a second op exceed the limit.
	l.onOpcode(33, byte(vm.STOP), 994, 0, &mockOpContext{memory: make([]byte, 32)}, nil, 0, nil)
	require.ErrorIs(t, l.Err(), errVmTraceTooLarge)

	// Remaining hooks must not resurrect the dropped trace.
	l.onEnter(1, byte(vm.CALL), addr(2), addr(3), nil, 100, big.NewInt(0))
	l.onOpcode(0, byte(vm.STOP), 100, 0, &mockOpContext{}, nil, 1, nil)
	l.onExit(1, nil, 0, nil, false)
	l.onExit(0, nil, 0, nil, false)
	require.Nil(t, l.GetResult())
	require.ErrorIs(t, l.Err(), errVmTraceTooLarge)
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

func TestVmTraceLogger_MemNilForNonMemoryOp(t *testing.T) {
	// PUSH1 does not write memory → Ex.Mem must be nil.
	l := NewVmTraceLogger()
	l.onEnter(0, 0x00, addr(1), addr(2), nil, 1000, big.NewInt(0))

	// Both contexts have empty memory — nothing changes.
	l.onOpcode(0, byte(vm.PUSH1), 1000, 3, &mockOpContext{}, nil, 0, nil)
	l.onOpcode(2, byte(vm.STOP), 997, 0, &mockOpContext{}, nil, 0, nil)
	l.onExit(0, nil, 3, nil, false)

	result := l.GetResult()
	require.Len(t, result.Ops, 2)
	require.Nil(t, result.Ops[0].Ex.Mem, "PUSH1 must have Mem=nil (no memory write)")
}

func TestVmTraceLogger_MemLastOpIsNil(t *testing.T) {
	// The last op in a frame has no subsequent onOpcode to finalize it,
	// so Ex.Mem must remain nil (post-execution state unavailable in onExit).
	l := NewVmTraceLogger()
	l.onEnter(0, 0x00, addr(1), addr(2), nil, 1000, big.NewInt(0))

	l.onOpcode(0, byte(vm.STOP), 1000, 0, &mockOpContext{}, nil, 0, nil)
	l.onExit(0, nil, 0, nil, false)

	result := l.GetResult()
	require.Len(t, result.Ops, 1)
	require.Nil(t, result.Ops[0].Ex.Mem, "last op must have Mem=nil")
}

func TestVmTraceLogger_MemWriteOffset(t *testing.T) {
	// MSTORE's write offset comes from its own pre-execution stack (the top
	// of stack), not from diffing memory buffers, so Off must track the
	// real write location instead of always being 0.
	mem32 := make([]byte, 32)
	mem32[31] = 0x42

	mem64 := make([]byte, 64)
	mem64[63] = 0x42

	tests := []struct {
		name    string
		mOffset uint64
		postMem []byte
	}{
		{"offset 0", 0, mem32},
		{"offset 32", 32, mem64},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewVmTraceLogger()
			l.onEnter(0, 0x00, addr(1), addr(2), nil, 1000, big.NewInt(0))

			// MSTORE pops offset then value, so offset must be the top
			// (last) element of the stack passed to onOpcode.
			mstoreStack := makeStack(0, tt.mOffset)
			l.onOpcode(0, byte(vm.MSTORE), 1000, 6, &mockOpContext{stack: mstoreStack}, nil, 0, nil)
			l.onOpcode(33, byte(vm.STOP), 994, 0, &mockOpContext{memory: tt.postMem}, nil, 0, nil)
			l.onExit(0, nil, 6, nil, false)

			result := l.GetResult()
			require.Len(t, result.Ops, 2)
			mem := result.Ops[0].Ex.Mem
			require.NotNil(t, mem, "MSTORE must have Mem set")
			require.Equal(t, tt.mOffset, mem.Off)
			require.Equal(t, tt.postMem[tt.mOffset:tt.mOffset+32], []byte(mem.Data))
		})
	}
}

func TestVmTraceLogger_MemWriteRange(t *testing.T) {
	// memWriteRange must resolve the exact (off, size) each memory-writing
	// opcode is about to write from its pre-execution stack operands, per
	// go-ethereum's core/vm/instructions.go operand order.
	tests := []struct {
		name     string
		op       vm.OpCode
		stack    []uint256.Int
		wantOff  uint64
		wantSize uint64
		wantOk   bool
	}{
		{"MSTORE", vm.MSTORE, makeStack(0, 100), 100, 32, true},
		{"MSTORE8", vm.MSTORE8, makeStack(0, 200), 200, 1, true},
		{"MSTORE short stack", vm.MSTORE, makeStack(5), 0, 0, false},
		{"CALLDATACOPY", vm.CALLDATACOPY, makeStack(50, 7, 300), 300, 50, true},
		{"CODECOPY", vm.CODECOPY, makeStack(20, 3, 150), 150, 20, true},
		{"RETURNDATACOPY", vm.RETURNDATACOPY, makeStack(15, 1, 90), 90, 15, true},
		{"MCOPY", vm.MCOPY, makeStack(40, 5, 250), 250, 40, true},
		{"EXTCODECOPY", vm.EXTCODECOPY, makeStack(20, 9, 400, 0xdead), 400, 20, true},
		{"CALL", vm.CALL, makeStack(60, 500, 10, 1, 0, 0xaaaa, 21000), 500, 60, true},
		{"CALLCODE", vm.CALLCODE, makeStack(60, 500, 10, 1, 0, 0xaaaa, 21000), 500, 60, true},
		{"DELEGATECALL", vm.DELEGATECALL, makeStack(40, 600, 5, 2, 0xbbbb, 21000), 600, 40, true},
		{"STATICCALL", vm.STATICCALL, makeStack(40, 600, 5, 2, 0xbbbb, 21000), 600, 40, true},
		{"ADD (non-memory)", vm.ADD, makeStack(1, 2), 0, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			off, size, ok := memWriteRange(tt.op, tt.stack)
			require.Equal(t, tt.wantOk, ok)
			if tt.wantOk {
				require.Equal(t, tt.wantOff, off)
				require.Equal(t, tt.wantSize, size)
			}
		})
	}
}

func TestVmTraceLogger_MemSequentialWrites(t *testing.T) {
	// Two MSTOREs to different offsets must each report only their own
	// 32-byte range, not a growing or duplicated full-buffer snapshot.
	l := NewVmTraceLogger()
	l.onEnter(0, 0x00, addr(1), addr(2), nil, 1000, big.NewInt(0))

	mem32 := make([]byte, 32)
	mem32[31] = 0x11

	mem64 := make([]byte, 64)
	copy(mem64, mem32)
	mem64[63] = 0x22

	// MSTORE(0, 0x11)
	l.onOpcode(0, byte(vm.MSTORE), 1000, 6, &mockOpContext{stack: makeStack(0, 0)}, nil, 0, nil)
	// MSTORE(32, 0x22); scope reflects the post-execution memory of the previous op.
	l.onOpcode(33, byte(vm.MSTORE), 994, 6, &mockOpContext{stack: makeStack(0, 32), memory: mem32}, nil, 0, nil)
	// STOP; scope reflects the post-execution memory of the second MSTORE.
	l.onOpcode(66, byte(vm.STOP), 988, 0, &mockOpContext{memory: mem64}, nil, 0, nil)
	l.onExit(0, nil, 12, nil, false)

	result := l.GetResult()
	require.Len(t, result.Ops, 3)

	mem1 := result.Ops[0].Ex.Mem
	require.NotNil(t, mem1)
	require.Equal(t, uint64(0), mem1.Off)
	require.Equal(t, mem32, []byte(mem1.Data), "first MSTORE's diff must cover only its own 32 bytes")

	mem2 := result.Ops[1].Ex.Mem
	require.NotNil(t, mem2)
	require.Equal(t, uint64(32), mem2.Off)
	require.Equal(t, mem64[32:64], []byte(mem2.Data), "second MSTORE's diff must cover only its own 32 bytes, not the whole buffer")
}

func TestVmTraceLogger_CallMemWriteClamped(t *testing.T) {
	// CALL's reserved return-data region [retOffset, retOffset+retSize) is only
	// partially written when the callee returns fewer bytes than retSize;
	// go-ethereum's Memory.Set never zero-fills the remainder, so the trace
	// must report only the actually-written prefix, not the full reservation.
	l := NewVmTraceLogger()
	l.onEnter(0, 0x00, addr(1), addr(2), nil, 1000, big.NewInt(0))

	// CALL stack (bottom to top): retSize=32, retOffset=0, inSize=0, inOffset=0, value=0, addr, gas.
	callStack := makeStack(32, 0, 0, 0, 0, 0xaaaa, 21000)
	l.onOpcode(0, byte(vm.CALL), 21000, 100, &mockOpContext{stack: callStack}, nil, 0, nil)

	// Callee returned only 10 bytes; [10,32) still holds a stale sentinel from
	// an earlier, unrelated write and must not be reported as written by CALL.
	postMem := make([]byte, 32)
	for i := range postMem {
		postMem[i] = 0xFF
	}
	returned := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	copy(postMem, returned)

	// rData is evm.returnData left by the CALL, i.e. the callee's actual return value.
	l.onOpcode(1, byte(vm.STOP), 20900, 0, &mockOpContext{memory: postMem}, returned, 0, nil)
	l.onExit(0, nil, 100, nil, false)

	result := l.GetResult()
	require.Len(t, result.Ops, 2)
	mem := result.Ops[0].Ex.Mem
	require.NotNil(t, mem, "CALL must report the bytes it actually wrote")
	require.Equal(t, uint64(0), mem.Off)
	require.Equal(t, returned, []byte(mem.Data), "must not include the reserved-but-unwritten tail")
}

func TestVmTraceLogger_CallMemWriteNilOnNonRevertError(t *testing.T) {
	// When a CALL fails with anything other than a revert, go-ethereum skips
	// Memory.Set entirely, so nothing was written — Ex.Mem must stay nil even
	// though the reserved [retOffset, retOffset+retSize) range is non-empty.
	l := NewVmTraceLogger()
	l.onEnter(0, 0x00, addr(1), addr(2), nil, 1000, big.NewInt(0))

	callStack := makeStack(32, 0, 0, 0, 0, 0xaaaa, 21000)
	l.onOpcode(0, byte(vm.CALL), 21000, 100, &mockOpContext{stack: callStack}, nil, 0, nil)

	// Memory untouched by CALL still holds whatever was there before it ran.
	postMem := make([]byte, 32)
	for i := range postMem {
		postMem[i] = 0xFF
	}
	// rData is empty: a non-revert failure (e.g. out of gas) leaves ret == nil.
	l.onOpcode(1, byte(vm.STOP), 20900, 0, &mockOpContext{memory: postMem}, nil, 0, nil)
	l.onExit(0, nil, 100, nil, false)

	result := l.GetResult()
	require.Len(t, result.Ops, 2)
	require.Nil(t, result.Ops[0].Ex.Mem, "CALL that wrote nothing must have Mem=nil")
}

func TestVmTraceLogger_MemDiffsFromRealEvm(t *testing.T) {
	returner := common.HexToAddress("0xcc") // returns 0xbeef
	reverter := common.HexToAddress("0xdd") // reverts with 0xdead
	failer := common.HexToAddress("0xee")   // hits INVALID
	calleeCode := func(v uint16, end byte) []byte {
		return []byte{0x61, byte(v >> 8), byte(v), 0x60, 0, 0x52, 0x60, 2, 0x60, 30, end}
	}
	callFrom := func(op vm.OpCode, retOff byte, to common.Address) []byte {
		code := []byte{0x60, 32, 0x60, retOff, 0x60, 0, 0x60, 0}
		if op == vm.CALL {
			code = append(code, 0x60, 0) // value
		}
		return append(code, 0x60, to[19], byte(vm.GAS), byte(op), byte(vm.POP))
	}

	code := []byte{
		0x60, 0x42, 0x60, 0, byte(vm.MSTORE),
		0x60, 0, byte(vm.MLOAD), byte(vm.POP),
		0x60, 0x7f, 0x60, 0x40, byte(vm.MSTORE8),
		0x60, 32, 0x60, 0, 0x60, 0x60, byte(vm.MCOPY),
	}
	code = append(code, callFrom(vm.CALL, 0x80, returner)...)
	code = append(code, callFrom(vm.STATICCALL, 0xa0, reverter)...)
	code = append(code, callFrom(vm.CALL, 0xc0, failer)...)
	code = append(code, byte(vm.STOP))

	run := func(l *VmTraceLogger) error {
		statedb, err := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
		require.NoError(t, err)
		statedb.SetCode(returner, calleeCode(0xbeef, byte(vm.RETURN)), tracing.CodeChangeUnspecified)
		statedb.SetCode(reverter, calleeCode(0xdead, byte(vm.REVERT)), tracing.CodeChangeUnspecified)
		statedb.SetCode(failer, []byte{byte(vm.INVALID)}, tracing.CodeChangeUnspecified)
		_, _, err = runtime.Execute(code, nil, &runtime.Config{
			State:     statedb,
			GasLimit:  1_000_000,
			EVMConfig: vm.Config{Tracer: l.Hooks()},
		})
		return err
	}

	t.Run("diffs", func(t *testing.T) {
		l := NewVmTraceLogger()
		require.NoError(t, run(l))
		require.NoError(t, l.Err())

		word := common.LeftPadBytes([]byte{0x42}, 32)
		type diff struct {
			op  string
			mem MemoryDiff
		}
		want := []diff{
			{"MSTORE", MemoryDiff{Off: 0, Data: word}},
			// MLOAD only reads memory, so it has no entry.
			{"MSTORE8", MemoryDiff{Off: 0x40, Data: []byte{0x7f}}},
			{"MCOPY", MemoryDiff{Off: 0x60, Data: word}},
			// CALL family: only the bytes actually returned, and nothing for the failed CALL.
			{"CALL", MemoryDiff{Off: 0x80, Data: []byte{0xbe, 0xef}}},
			{"STATICCALL", MemoryDiff{Off: 0xa0, Data: []byte{0xde, 0xad}}},
		}
		var got []diff
		for _, op := range l.GetResult().Ops {
			if op.Ex != nil && op.Ex.Mem != nil {
				got = append(got, diff{op.Op, *op.Ex.Mem})
			}
		}
		require.Equal(t, want, got)
	})

	t.Run("size limit", func(t *testing.T) {
		l := NewVmTraceLogger()
		l.sizeLimit = 10 * vmTraceOpSize
		require.NoError(t, run(l))
		require.ErrorIs(t, l.Err(), errVmTraceTooLarge)
		require.Nil(t, l.GetResult())
	})
}
