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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"
)

// VmTrace is the top level VM execution trace for a single call frame.
type VmTrace struct {
	Code hexutil.Bytes `json:"code"`
	Ops  []VmOperation `json:"ops"`
}

// VmOperation represents a single VM opcode step.
type VmOperation struct {
	PC   uint64               `json:"pc"`
	Op   string               `json:"op"`
	Cost uint64               `json:"cost"`
	Ex   *VmExecutedOperation `json:"ex"`  // nil when the opcode caused a fault
	Sub  *VmTrace             `json:"sub"` // non nil for CALL/CREATE sub frames
}

// VmExecutedOperation captures the observable effects of a successfully executed opcode.
type VmExecutedOperation struct {
	Used  uint64        `json:"used"`
	Push  []hexutil.Big `json:"push"`
	Mem   *MemoryDiff   `json:"mem"`
	Store *StorageDiff  `json:"store"`
}

// MemoryDiff records the byte range of VM memory that an opcode wrote.
type MemoryDiff struct {
	Off  uint64        `json:"off"`
	Data hexutil.Bytes `json:"data"`
}

// StorageDiff records a write to a storage slot.
type StorageDiff struct {
	Key common.Hash `json:"key"`
	Val common.Hash `json:"val"`
}

// vmTraceFrame is the call frame state maintained by VmTraceLogger.
type vmTraceFrame struct {
	trace     *VmTrace
	lastOpIdx int       // index of the most recently appended op, -1 if none
	lastOp    vm.OpCode // opcode at lastOpIdx
	memWrite  bool      // whether lastOp writes memory
	memOff    uint64    // memory offset lastOp is about to write, if memWrite
	memSize   uint64    // memory size lastOp is about to write, if memWrite
}

// vmTraceSizeLimit caps the approximate number of bytes a single vmTrace
// retains (code, memory diffs and per-op overhead), protecting the node
// from running out of memory on pathological traces.
const vmTraceSizeLimit = 256 << 20

// vmTraceOpSize approximates the fixed footprint of one traced operation.
const vmTraceOpSize = 128

var errVmTraceTooLarge = errors.New("vmTrace exceeds size limit")

// VmTraceLogger implements VM tracing hooks to build vmTrace.
type VmTraceLogger struct {
	traceStack   []*vmTraceFrame
	result       *VmTrace
	stateDB      tracing.StateDB
	pendingStore *StorageDiff // latest storage write, attributed to the current op
	size         uint64       // approximate bytes retained so far
	sizeLimit    uint64
	err          error // set once the trace is dropped for exceeding sizeLimit
}

// NewVmTraceLogger creates a new VmTraceLogger.
func NewVmTraceLogger() *VmTraceLogger {
	return &VmTraceLogger{sizeLimit: vmTraceSizeLimit}
}

// Hooks returns the tracing hooks.
func (l *VmTraceLogger) Hooks() *tracing.Hooks {
	return &tracing.Hooks{
		OnTxStart:       l.onTxStart,
		OnEnter:         l.onEnter,
		OnExit:          l.onExit,
		OnOpcode:        l.onOpcode,
		OnFault:         l.onFault,
		OnStorageChange: l.OnStorageChange,
	}
}

// GetResult returns the completed vmTrace after execution.
func (l *VmTraceLogger) GetResult() *VmTrace {
	return l.result
}

// Err returns a non-nil error when the trace was dropped for exceeding the
// size limit, in which case GetResult returns nil.
func (l *VmTraceLogger) Err() error {
	return l.err
}

// grow accounts n more retained bytes. Once the limit is exceeded, it drops
// everything collected so far and returns false; the remaining hooks then
// find an empty trace stack and do nothing.
func (l *VmTraceLogger) grow(n uint64) bool {
	if l.err != nil {
		return false
	}
	if n > l.sizeLimit-l.size {
		l.err = errVmTraceTooLarge
		l.traceStack = nil
		l.result = nil
		l.pendingStore = nil
		return false
	}
	l.size += n
	return true
}

// OnStorageChange records a storage write so it can be attributed to the
// currently executing opcode.
func (l *VmTraceLogger) OnStorageChange(_ common.Address, slot common.Hash, _ common.Hash, newVal common.Hash) {
	if l.err != nil {
		return
	}
	l.pendingStore = &StorageDiff{Key: slot, Val: newVal}
}

// onTxStart captures the StateDB reference for contract code look-ups in OnEnter.
func (l *VmTraceLogger) onTxStart(vmCtx *tracing.VMContext, _ *types.Transaction, _ common.Address) {
	l.stateDB = vmCtx.StateDB
}

// onEnter creates a new VmTrace frame for each call or create.
func (l *VmTraceLogger) onEnter(depth int, typ byte, _ common.Address, to common.Address, input []byte, gas uint64, _ *big.Int) {
	if l.err != nil {
		return
	}
	var (
		code    []byte
		noTrace = false
	)
	switch vm.OpCode(typ) {
	case vm.CREATE, vm.CREATE2:
		// input is the init bytecode for CREATE/CREATE2
		code = make([]byte, len(input))
		copy(code, input)
	case vm.SELFDESTRUCT:
		noTrace = true
	default:
		// For CALL variants, look up the callee's deployed code
		if l.stateDB != nil {
			code = l.stateDB.GetCode(to)
		}
	}

	if !l.grow(uint64(len(code))) {
		return
	}

	newTrace := &VmTrace{
		Code: hexutil.Bytes(code),
		Ops:  make([]VmOperation, 0),
	}

	frame := &vmTraceFrame{
		trace:     newTrace,
		lastOpIdx: -1,
	}

	// Link this trace to the parent frame's last operation as a sub-trace.
	if depth > 0 && len(l.traceStack) > 0 {
		parent := l.traceStack[len(l.traceStack)-1]
		if parent.lastOpIdx >= 0 && !noTrace {
			parent.trace.Ops[parent.lastOpIdx].Sub = newTrace
		}
	}

	l.traceStack = append(l.traceStack, frame)
}

// onExit finalizes the exiting frame's last operation and pops the frame.
func (l *VmTraceLogger) onExit(depth int, _ []byte, _ uint64, _ error, _ bool) {
	if len(l.traceStack) == 0 {
		return
	}
	frame := l.traceStack[len(l.traceStack)-1]
	l.traceStack = l.traceStack[:len(l.traceStack)-1]

	// Finalize the last operation's Store. Mem is left nil: terminal ops (STOP,
	// RETURN, REVERT) do not write memory, and post-execution memory is unavailable
	// in onExit since scope is gone.
	if frame.lastOpIdx >= 0 {
		op := &frame.trace.Ops[frame.lastOpIdx]
		if op.Ex != nil {
			if l.pendingStore != nil {
				op.Ex.Store = l.pendingStore
				l.pendingStore = nil
			}
		}
	}

	// Save the completed root trace so GetResult can return it.
	if depth == 0 {
		l.result = frame.trace
	}
}

// onOpcode is called before each opcode executes.
// It finalizes the previous op's Ex using the current post execution
// state, then records a new operation entry for the current opcode.
func (l *VmTraceLogger) onOpcode(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, _ []byte, _ int, opErr error) {
	if len(l.traceStack) == 0 {
		return
	}
	frame := l.traceStack[len(l.traceStack)-1]

	// Finalize the previous operation's Push, Mem, and Store now that we have
	// the post execution state.
	if frame.lastOpIdx >= 0 {
		prevOp := &frame.trace.Ops[frame.lastOpIdx]
		if prevOp.Ex != nil {
			prevOp.Ex.Push = computePushed(frame.lastOp, scope.StackData())
			if frame.memWrite && frame.memSize > 0 {
				// Report the full reserved [off, off+size) region, matching Parity/OpenEthereum's vmTrace semantics.
				if !l.grow(frame.memSize) {
					return
				}
				data := memoryRegion(scope.MemoryData(), frame.memOff, frame.memSize)
				prevOp.Ex.Mem = &MemoryDiff{Off: frame.memOff, Data: data}
			}
			if l.pendingStore != nil {
				prevOp.Ex.Store = l.pendingStore
				l.pendingStore = nil
			}
		}
	}

	if !l.grow(vmTraceOpSize) {
		return
	}

	// Create a new operation entry. Ex.Used = gas before this opcode (gasCopy from
	// interpreter, captured before any gas deduction for this op).
	//
	// go-ethereum's interpreter reports pre-execution failures (stack
	// underflow/overflow, out-of-gas on constant or dynamic gas, memory-size
	// overflow) through this OnOpcode call's err, never through OnFault — the
	// opcode never actually executes, so it gets no execution result, matching
	// the onFault treatment. ErrExecutionReverted is the one error that
	// represents a valid REVERT execution.
	var ex *VmExecutedOperation
	if opErr == nil || errors.Is(opErr, vm.ErrExecutionReverted) {
		ex = &VmExecutedOperation{Used: gas, Push: []hexutil.Big{}}
	}
	newOp := VmOperation{
		Op:   vm.OpCode(op).String(),
		PC:   pc,
		Cost: cost,
		Ex:   ex,
	}
	frame.trace.Ops = append(frame.trace.Ops, newOp)
	frame.lastOpIdx = len(frame.trace.Ops) - 1
	frame.lastOp = vm.OpCode(op)
	// Determine the memory range this opcode is about to write
	frame.memOff, frame.memSize, frame.memWrite = memWriteRange(vm.OpCode(op), scope.StackData())
}

// memWriteRange returns the memory byte range the opcode is about to
// touch, derived from its pre-execution stack operands (mirrors the
// operand order in go-ethereum's core/vm/instructions.go and eips.go —
// MSTORE, MSTORE8, the *COPY family (including MCOPY), and the CALL
// family are the only memory-writing opcodes). ok is false when this
// opcode does not touch memory, or the stack is too shallow for it to
// execute.
func memWriteRange(op vm.OpCode, stack []uint256.Int) (off, size uint64, ok bool) {
	n := len(stack)
	top := func(k int) uint64 { return stack[n-1-k].Uint64() } // k=0 is the top of stack

	switch op {
	case vm.MLOAD:
		// stack: offset (this opcode doesn't write, but Parity/OpenEthereum reports it anyway - compatibility)
		if n >= 1 {
			return top(0), 32, true
		}
	case vm.MSTORE:
		// stack: offset, value
		if n >= 2 {
			return top(0), 32, true
		}
	case vm.MSTORE8:
		// stack: offset, value
		if n >= 2 {
			return top(0), 1, true
		}
	case vm.CALLDATACOPY, vm.CODECOPY, vm.RETURNDATACOPY, vm.MCOPY:
		// stack: memOffset, dataOffset/codeOffset/src, length
		if n >= 3 {
			return top(0), top(2), true
		}
	case vm.EXTCODECOPY:
		// stack: address, memOffset, codeOffset, length
		if n >= 4 {
			return top(1), top(3), true
		}
	case vm.CALL, vm.CALLCODE:
		// stack: gas, addr, value, inOffset, inSize, retOffset, retSize
		if n >= 7 {
			return top(5), top(6), true
		}
	case vm.DELEGATECALL, vm.STATICCALL:
		// stack: gas, addr, inOffset, inSize, retOffset, retSize
		if n >= 6 {
			return top(4), top(5), true
		}
	}
	return 0, 0, false
}

// memoryRegion copies the size bytes of mem starting at off, zero-padding
// any part that lies beyond mem's current length.
func memoryRegion(mem []byte, off, size uint64) []byte {
	data := make([]byte, size)
	if off < uint64(len(mem)) {
		copy(data, mem[off:])
	}
	return data
}

// onFault is called when an opcode causes a fault.
// The faulting operation's Ex is set to nil (no successful execution result),
// except for ErrExecutionReverted which represents a valid REVERT opcode execution.
func (l *VmTraceLogger) onFault(_ uint64, _ byte, _, _ uint64, _ tracing.OpContext, _ int, err error) {
	if errors.Is(err, vm.ErrExecutionReverted) {
		// REVERT is a valid opcode execution; its Ex remains intact.
		return
	}
	if len(l.traceStack) == 0 {
		return
	}
	frame := l.traceStack[len(l.traceStack)-1]
	if frame.lastOpIdx >= 0 {
		frame.trace.Ops[frame.lastOpIdx].Ex = nil
	}
}

// computePushed returns the values pushed/affected on the VM stack by the previous opcode.
func computePushed(op vm.OpCode, stack []uint256.Int) []hexutil.Big {

	var count int
	switch {
	case op >= vm.PUSH0 && op <= vm.PUSH32:
		count = 1
	case op >= vm.SWAP1 && op <= vm.SWAP16:
		count = int(op-vm.SWAP1) + 2
	case op >= vm.DUP1 && op <= vm.DUP16:
		count = int(op-vm.DUP1) + 2
	}
	switch op {
	case vm.CALLDATALOAD, vm.SLOAD, vm.MLOAD, vm.CALLDATASIZE, vm.LT, vm.GT, vm.DIV, vm.SDIV, vm.SAR, vm.AND, vm.EQ, vm.CALLVALUE, vm.ISZERO,
		vm.ADD, vm.EXP, vm.CALLER, vm.KECCAK256, vm.SUB, vm.ADDRESS, vm.GAS, vm.MUL, vm.RETURNDATASIZE, vm.NOT, vm.SHR, vm.SHL,
		vm.EXTCODESIZE, vm.SLT, vm.OR, vm.NUMBER, vm.PC, vm.TIMESTAMP, vm.BALANCE, vm.SELFBALANCE, vm.MULMOD, vm.ADDMOD, vm.BASEFEE,
		vm.BLOCKHASH, vm.BYTE, vm.XOR, vm.ORIGIN, vm.CODESIZE, vm.MOD, vm.SIGNEXTEND, vm.GASLIMIT, vm.DIFFICULTY, vm.SGT, vm.GASPRICE,
		vm.MSIZE, vm.EXTCODEHASH, vm.SMOD, vm.CHAINID, vm.COINBASE, vm.TLOAD, vm.CALL, vm.DELEGATECALL, vm.STATICCALL, vm.CREATE, vm.CREATE2:
		count = 1
	}

	if count > 0 {
		if count > len(stack) {
			count = len(stack)
		}
		result := make([]hexutil.Big, count)
		for i := range count {
			result[i] = hexutil.Big(*stack[len(stack)-count+i].ToBig())
		}
		return result
	}

	return []hexutil.Big{}
}
