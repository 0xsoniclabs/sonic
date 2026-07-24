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
	result, err := l.GetResult()
	require.NoError(t, err)
	require.Nil(t, result, "result must be nil before any execution")
}

func TestVmTraceLogger_EmptyFrame(t *testing.T) {
	// OnEnter + OnExit with no opcodes → trace with empty ops
	l := NewVmTraceLogger()

	code := []byte{0x60, 0x01} // PUSH1 1
	l.onEnter(0, 0x00 /* STOP */, addr(1), addr(2), code, 100, big.NewInt(0))
	l.onExit(0, nil, 0, nil, false)

	result, err := l.GetResult()
	require.NoError(t, err)
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

	result, err := l.GetResult()
	require.NoError(t, err)
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

	result, err := l.GetResult()
	require.NoError(t, err)
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

	result, err := l.GetResult()
	require.NoError(t, err)
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

	result, err := l.GetResult()
	require.NoError(t, err)
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

	result, err := l.GetResult()
	require.NoError(t, err)
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
	result, err := l.GetResult()
	require.NoError(t, err)
	require.Nil(t, result)
}

func TestVmTraceLogger_RevertHasNonNilEx(t *testing.T) {
	// REVERT is a valid opcode execution; onFault with ErrExecutionReverted must not nil Ex.
	l := NewVmTraceLogger()
	l.onEnter(0, 0x00, addr(1), addr(2), nil, 1000, big.NewInt(0))
	l.onOpcode(0, 0xfd /* REVERT */, 1000, 0, &mockOpContext{}, nil, 0, nil)
	l.onFault(0, 0xfd, 1000, 0, &mockOpContext{}, 0, vm.ErrExecutionReverted)
	l.onExit(0, nil, 0, vm.ErrExecutionReverted, true)

	result, err := l.GetResult()
	require.NoError(t, err)
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

	result, err := l.GetResult()
	require.NoError(t, err)
	require.Len(t, result.Ops, 2)
	require.Nil(t, result.Ops[0].Ex.Mem, "PUSH1 must have Mem=nil (no memory write)")
}

func TestVmTraceLogger_MemSetAfterMSTORE(t *testing.T) {
	// MSTORE writes 32 bytes; Ex.Mem must carry the written region with the
	// offset taken from the stack (Parity/OpenEthereum semantics).
	l := NewVmTraceLogger()
	l.onEnter(0, 0x00, addr(1), addr(2), nil, 1000, big.NewInt(0))

	// MSTORE stack (top last): value=0x42, offset=0.
	ctxBeforeMstore := &mockOpContext{stack: makeStack(0x42, 0)}
	l.onOpcode(0, byte(vm.MSTORE), 1000, 6, ctxBeforeMstore, nil, 0, nil)

	// After MSTORE: 32 bytes, last byte = 0x42.
	memAfter := make([]byte, 32)
	memAfter[31] = 0x42
	ctxAfterMstore := &mockOpContext{memory: memAfter}
	l.onOpcode(33, byte(vm.STOP), 994, 0, ctxAfterMstore, nil, 0, nil)

	l.onExit(0, nil, 6, nil, false)

	result, err := l.GetResult()
	require.NoError(t, err)
	require.Len(t, result.Ops, 2)

	mstoreOp := result.Ops[0]
	require.NotNil(t, mstoreOp.Ex)
	require.NotNil(t, mstoreOp.Ex.Mem, "MSTORE must have Mem set")
	require.Equal(t, uint64(0), mstoreOp.Ex.Mem.Off, "Mem.Off must be the MSTORE offset")
	require.Equal(t, []byte(memAfter), []byte(mstoreOp.Ex.Mem.Data), "Mem.Data must equal the written 32-byte region")
}

func TestVmTraceLogger_MemOffsetFromStack(t *testing.T) {
	// MSTORE at offset 32 must report Off=32 and only the written 32-byte region.
	l := NewVmTraceLogger()
	l.onEnter(0, 0x00, addr(1), addr(2), nil, 1000, big.NewInt(0))

	// MSTORE stack (top last): value=0x02, offset=32.
	l.onOpcode(0, byte(vm.MSTORE), 1000, 6, &mockOpContext{stack: makeStack(0x02, 32)}, nil, 0, nil)

	// 64-byte post-execution memory (two MSTORE slots).
	bigMem := make([]byte, 64)
	bigMem[31] = 0x01
	bigMem[63] = 0x02
	l.onOpcode(33, byte(vm.STOP), 994, 0, &mockOpContext{memory: bigMem}, nil, 0, nil)
	l.onExit(0, nil, 6, nil, false)

	result, err := l.GetResult()
	require.NoError(t, err)
	require.Len(t, result.Ops, 2)
	mem := result.Ops[0].Ex.Mem
	require.NotNil(t, mem)
	require.Equal(t, uint64(32), mem.Off, "Off must be the MSTORE offset")
	require.Equal(t, []byte(bigMem[32:64]), []byte(mem.Data), "Data must be the written region only")
}

func TestVmTraceLogger_MemRegionForCallReturnData(t *testing.T) {
	// A CALL op must report the return-data region (retOffset, retSize) of the
	// caller's memory once the frame resumes after the sub-call.
	l := NewVmTraceLogger()
	l.onEnter(0, 0x00, addr(1), addr(2), nil, 1000, big.NewInt(0))

	// CALL stack (top last): retSize=32, retOffset=64, argsLen, argsOff, value, to, gas.
	callStack := makeStack(32, 64, 0, 0, 0, 2, 800)
	l.onOpcode(0, byte(vm.CALL), 1000, 100, &mockOpContext{stack: callStack}, nil, 0, nil)

	// Sub-call runs and exits.
	l.onEnter(1, byte(vm.CALL), addr(2), addr(3), nil, 800, big.NewInt(0))
	l.onExit(1, nil, 10, nil, false)

	// Back in the root frame: memory now holds return data at [64, 96).
	mem := make([]byte, 96)
	mem[64] = 0xAA
	mem[95] = 0xBB
	l.onOpcode(10, byte(vm.STOP), 890, 0, &mockOpContext{memory: mem, stack: makeStack(1)}, nil, 0, nil)
	l.onExit(0, nil, 110, nil, false)

	result, err := l.GetResult()
	require.NoError(t, err)
	require.Len(t, result.Ops, 2)

	callOp := result.Ops[0]
	require.NotNil(t, callOp.Ex.Mem, "CALL must report the return-data region")
	require.Equal(t, uint64(64), callOp.Ex.Mem.Off)
	require.Equal(t, []byte(mem[64:96]), []byte(callOp.Ex.Mem.Data))
}

func TestVmTraceLogger_MemZeroPaddedBeyondMemory(t *testing.T) {
	// A region extending past the current memory size is zero-padded.
	l := NewVmTraceLogger()
	l.onEnter(0, 0x00, addr(1), addr(2), nil, 1000, big.NewInt(0))

	// MSTORE at offset 16 while post-exec memory is only 24 bytes long
	// (cannot happen in the real EVM, but the copy must stay safe).
	l.onOpcode(0, byte(vm.MSTORE), 1000, 6, &mockOpContext{stack: makeStack(0x01, 16)}, nil, 0, nil)
	shortMem := make([]byte, 24)
	shortMem[16] = 0x7F
	l.onOpcode(33, byte(vm.STOP), 994, 0, &mockOpContext{memory: shortMem}, nil, 0, nil)
	l.onExit(0, nil, 6, nil, false)

	result, err := l.GetResult()
	require.NoError(t, err)
	mem := result.Ops[0].Ex.Mem
	require.NotNil(t, mem)
	require.Equal(t, uint64(16), mem.Off)
	require.Len(t, []byte(mem.Data), 32)
	require.Equal(t, byte(0x7F), mem.Data[0])
	require.Equal(t, make([]byte, 24), []byte(mem.Data[8:]), "bytes beyond memory must be zero-padded")
}

func TestVmTraceLogger_SizeLimitAbortsTrace(t *testing.T) {
	// Exceeding vmTraceSizeLimit must abort the trace, release the data, and
	// surface an error from GetResult.
	l := NewVmTraceLogger()
	l.onEnter(0, 0x00, addr(1), addr(2), nil, 1000, big.NewInt(0))
	l.traceSize = vmTraceSizeLimit // next accounted byte tips over the limit

	l.onOpcode(0, byte(vm.MSTORE), 1000, 6, &mockOpContext{stack: makeStack(0x01, 0)}, nil, 0, nil)
	l.onOpcode(33, byte(vm.STOP), 994, 0, &mockOpContext{memory: make([]byte, 32)}, nil, 0, nil)
	l.onExit(0, nil, 6, nil, false)

	result, err := l.GetResult()
	require.ErrorIs(t, err, errVmTraceTooLarge)
	require.Nil(t, result)
	require.Nil(t, l.traceStack, "accumulated data must be released on abort")
}

func TestMemTouchedRegion(t *testing.T) {
	// Stacks are built with makeStack (top of stack = last element).
	tests := []struct {
		name     string
		op       vm.OpCode
		stack    []uint256.Int
		wantShow bool
		wantOff  uint64
		wantLen  uint64
	}{
		// Region-reporting opcodes, offsets/lengths from the pre-execution stack.
		{"MSTORE", vm.MSTORE, makeStack(0x42, 5), true, 5, 32},
		{"MLOAD", vm.MLOAD, makeStack(7), true, 7, 32},
		{"MSTORE8", vm.MSTORE8, makeStack(0x42, 9), true, 9, 1},
		{"CALLDATACOPY", vm.CALLDATACOPY, makeStack(10, 3, 64), true, 64, 10},
		{"CODECOPY", vm.CODECOPY, makeStack(10, 3, 64), true, 64, 10},
		{"RETURNDATACOPY", vm.RETURNDATACOPY, makeStack(10, 3, 64), true, 64, 10},
		{"MCOPY", vm.MCOPY, makeStack(10, 3, 64), true, 64, 10},
		{"EXTCODECOPY", vm.EXTCODECOPY, makeStack(10, 3, 64, 0xAA), true, 64, 10},
		{"CALL", vm.CALL, makeStack(32, 96, 0, 0, 0, 2, 800), true, 96, 32},
		{"CALLCODE", vm.CALLCODE, makeStack(32, 96, 0, 0, 0, 2, 800), true, 96, 32},
		{"DELEGATECALL", vm.DELEGATECALL, makeStack(32, 96, 0, 0, 2, 800), true, 96, 32},
		{"STATICCALL", vm.STATICCALL, makeStack(32, 96, 0, 0, 2, 800), true, 96, 32},

		// Zero-length regions are still reported as touched; the logger skips
		// emitting a Mem entry for them (memLen > 0 guard).
		{"CALLDATACOPY zero length", vm.CALLDATACOPY, makeStack(0, 3, 64), true, 64, 0},
		{"CALL zero retSize", vm.CALL, makeStack(0, 96, 0, 0, 0, 2, 800), true, 96, 0},

		// Stack too shallow for the operands → nothing reported (the op will fault).
		{"MSTORE empty stack", vm.MSTORE, nil, false, 0, 0},
		{"MLOAD empty stack", vm.MLOAD, nil, false, 0, 0},
		{"CALLDATACOPY short stack", vm.CALLDATACOPY, makeStack(1, 2), false, 0, 0},
		{"EXTCODECOPY short stack", vm.EXTCODECOPY, makeStack(1, 2, 3), false, 0, 0},
		{"CALL short stack", vm.CALL, makeStack(1, 2, 3, 4, 5, 6), false, 0, 0},
		{"STATICCALL short stack", vm.STATICCALL, makeStack(1, 2, 3, 4, 5), false, 0, 0},

		// Opcodes that never report a region, matching OpenEthereum. CREATE,
		// CREATE2 and SELFDESTRUCT are explicitly excluded there.
		{"ADD", vm.ADD, makeStack(1, 2), false, 0, 0},
		{"SSTORE", vm.SSTORE, makeStack(1, 2), false, 0, 0},
		{"KECCAK256 reads only", vm.KECCAK256, makeStack(32, 0), false, 0, 0},
		{"RETURN", vm.RETURN, makeStack(32, 0), false, 0, 0},
		{"REVERT", vm.REVERT, makeStack(32, 0), false, 0, 0},
		{"LOG0 reads only", vm.LOG0, makeStack(32, 0), false, 0, 0},
		{"CREATE", vm.CREATE, makeStack(32, 0, 0), false, 0, 0},
		{"CREATE2", vm.CREATE2, makeStack(1, 32, 0, 0), false, 0, 0},
		{"SELFDESTRUCT", vm.SELFDESTRUCT, makeStack(2), false, 0, 0},
		{"STOP", vm.STOP, nil, false, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			show, off, size := memTouchedRegion(tt.op, tt.stack)
			require.Equal(t, tt.wantShow, show)
			require.Equal(t, tt.wantOff, off)
			require.Equal(t, tt.wantLen, size)
		})
	}
}

func TestMemTouchedRegion_OffsetBeyondUint64DoesNotPanic(t *testing.T) {
	// An offset ≥ 2^64 cannot be executed (the op faults on gas), but the
	// region decoding must not panic on it. Uint64() truncates; the value is
	// irrelevant because onFault nils the op's Ex before any Mem is emitted.
	huge := uint256.MustFromBig(new(big.Int).Lsh(big.NewInt(1), 64))
	stack := []uint256.Int{*uint256.NewInt(0x42), *huge}
	require.NotPanics(t, func() {
		show, _, size := memTouchedRegion(vm.MSTORE, stack)
		require.True(t, show)
		require.Equal(t, uint64(32), size)
	})
}

func TestMemoryRegion(t *testing.T) {
	mem := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	tests := []struct {
		name string
		off  uint64
		size uint64
		want []byte
	}{
		{"fully inside", 2, 4, []byte{2, 3, 4, 5}},
		{"exact fit", 0, 10, mem},
		{"partially beyond end", 8, 4, []byte{8, 9, 0, 0}},
		{"offset at memory size", 10, 3, []byte{0, 0, 0}},
		{"offset beyond memory size", 20, 3, []byte{0, 0, 0}},
		{"zero size", 2, 0, []byte{}},
		{"empty memory", 0, 3, []byte{0, 0, 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := mem
			if tt.name == "empty memory" {
				src = nil
			}
			got := memoryRegion(src, tt.off, tt.size)
			require.Equal(t, tt.want, []byte(got))
		})
	}
}

func TestVmTraceLogger_ZeroLengthCopyHasNoMem(t *testing.T) {
	// CALLDATACOPY with length 0 touches nothing → Mem must stay nil.
	l := NewVmTraceLogger()
	l.onEnter(0, 0x00, addr(1), addr(2), nil, 1000, big.NewInt(0))

	l.onOpcode(0, byte(vm.CALLDATACOPY), 1000, 3, &mockOpContext{stack: makeStack(0, 0, 64)}, nil, 0, nil)
	l.onOpcode(1, byte(vm.STOP), 997, 0, &mockOpContext{memory: make([]byte, 96)}, nil, 0, nil)
	l.onExit(0, nil, 3, nil, false)

	result, err := l.GetResult()
	require.NoError(t, err)
	require.Len(t, result.Ops, 2)
	require.Nil(t, result.Ops[0].Ex.Mem, "zero-length copy must not produce a Mem entry")
}

func TestVmTraceLogger_MLOADReportsReadRegion(t *testing.T) {
	// Parity quirk: MLOAD reports the 32-byte region it reads.
	l := NewVmTraceLogger()
	l.onEnter(0, 0x00, addr(1), addr(2), nil, 1000, big.NewInt(0))

	mem := make([]byte, 64)
	mem[32] = 0x11
	mem[63] = 0x22
	l.onOpcode(0, byte(vm.MLOAD), 1000, 3, &mockOpContext{stack: makeStack(32), memory: mem}, nil, 0, nil)
	l.onOpcode(1, byte(vm.STOP), 997, 0, &mockOpContext{memory: mem, stack: makeStack(0x11)}, nil, 0, nil)
	l.onExit(0, nil, 3, nil, false)

	result, err := l.GetResult()
	require.NoError(t, err)
	require.Len(t, result.Ops, 2)
	mloadOp := result.Ops[0]
	require.NotNil(t, mloadOp.Ex.Mem, "MLOAD must report the region it reads")
	require.Equal(t, uint64(32), mloadOp.Ex.Mem.Off)
	require.Equal(t, []byte(mem[32:64]), []byte(mloadOp.Ex.Mem.Data))
}

func TestVmTraceLogger_FaultedMemoryOpHasNoMem(t *testing.T) {
	// A memory op that faults gets Ex=nil; the pending region must not be
	// emitted and nothing may panic.
	l := NewVmTraceLogger()
	l.onEnter(0, 0x00, addr(1), addr(2), nil, 1000, big.NewInt(0))

	l.onOpcode(0, byte(vm.MSTORE), 1000, 6, &mockOpContext{stack: makeStack(0x42, 0)}, nil, 0, nil)
	l.onFault(0, byte(vm.MSTORE), 1000, 6, &mockOpContext{}, 0, errFoo)
	l.onExit(0, nil, 1000, errFoo, true)

	result, err := l.GetResult()
	require.NoError(t, err)
	require.Len(t, result.Ops, 1)
	require.Nil(t, result.Ops[0].Ex, "faulted op must have Ex=nil")
}

func TestVmTraceLogger_SizeLimitBoundaryIsNotExceeded(t *testing.T) {
	// Reaching the limit exactly is fine; only exceeding it aborts.
	l := NewVmTraceLogger()
	l.onEnter(0, 0x00, addr(1), addr(2), nil, 1000, big.NewInt(0))
	l.traceSize = vmTraceSizeLimit - vmTraceOpOverhead

	l.onOpcode(0, byte(vm.STOP), 1000, 0, &mockOpContext{}, nil, 0, nil)
	l.onExit(0, nil, 0, nil, false)

	result, err := l.GetResult()
	require.NoError(t, err)
	require.Len(t, result.Ops, 1)
}

func TestVmTraceLogger_OversizedCodeAbortsAndHooksStayInert(t *testing.T) {
	// Contract code accounting can trip the limit in onEnter; every later hook
	// must be a no-op without panicking.
	l := NewVmTraceLogger()
	l.traceSize = vmTraceSizeLimit - 10

	l.onEnter(0, byte(vm.CREATE), addr(1), addr(2), make([]byte, 100), 1000, big.NewInt(0))

	require.NotPanics(t, func() {
		l.onEnter(1, byte(vm.CALL), addr(2), addr(3), nil, 500, big.NewInt(0))
		l.onOpcode(0, byte(vm.MSTORE), 1000, 6, &mockOpContext{stack: makeStack(0x42, 0)}, nil, 0, nil)
		l.onFault(0, byte(vm.MSTORE), 1000, 6, &mockOpContext{}, 0, errFoo)
		l.OnStorageChange(addr(2), common.Hash{0x01}, common.Hash{}, common.Hash{0x02})
		l.onExit(1, nil, 0, nil, false)
		l.onExit(0, nil, 0, nil, false)
	})

	result, err := l.GetResult()
	require.ErrorIs(t, err, errVmTraceTooLarge)
	require.Nil(t, result)
	require.Nil(t, l.traceStack, "accumulated data must be released on abort")
}

func TestVmTraceLogger_MemLastOpIsNil(t *testing.T) {
	// The last op in a frame has no subsequent onOpcode to finalize it,
	// so Ex.Mem must remain nil (post-execution state unavailable in onExit).
	l := NewVmTraceLogger()
	l.onEnter(0, 0x00, addr(1), addr(2), nil, 1000, big.NewInt(0))

	l.onOpcode(0, byte(vm.STOP), 1000, 0, &mockOpContext{}, nil, 0, nil)
	l.onExit(0, nil, 0, nil, false)

	result, err := l.GetResult()
	require.NoError(t, err)
	require.Len(t, result.Ops, 1)
	require.Nil(t, result.Ops[0].Ex.Mem, "last op must have Mem=nil")
}
