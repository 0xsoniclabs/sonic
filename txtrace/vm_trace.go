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

// MemoryDiff records the memory region written (or, for MLOAD, read) by an
// opcode, matching the Parity/OpenEthereum vmTrace "mem" semantics.
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
	showMem   bool      // lastOp touches a memory region to be reported
	memOff    uint64    // offset of the memory region touched by lastOp
	memLen    uint64    // length of the memory region touched by lastOp
}

// vmTraceSizeLimit caps the cumulative size of data retained in a single
// vmTrace (memory regions, code, per-op overhead). Tracing aborts with an
// error once the limit is exceeded, protecting the node from OOM on
// pathological traces.
const vmTraceSizeLimit = 256 << 20 // 256 MB

// vmTraceOpOverhead approximates the fixed per-operation memory footprint
// counted against vmTraceSizeLimit.
const vmTraceOpOverhead = 64

// errVmTraceTooLarge is returned by GetResult when the trace exceeded vmTraceSizeLimit.
var errVmTraceTooLarge = errors.New("vmTrace result exceeds size limit")

// VmTraceLogger implements VM tracing hooks to build vmTrace.
type VmTraceLogger struct {
	traceStack   []*vmTraceFrame
	result       *VmTrace
	stateDB      tracing.StateDB
	pendingStore *StorageDiff // latest storage write, attributed to the current op
	traceSize    uint64       // approximate retained bytes, checked against vmTraceSizeLimit
	err          error        // set when the trace was aborted (size limit exceeded)
}

// NewVmTraceLogger creates a new VmTraceLogger.
func NewVmTraceLogger() *VmTraceLogger {
	return &VmTraceLogger{}
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

// GetResult returns the completed vmTrace after execution, or an error when
// the trace was aborted because it exceeded the size limit.
func (l *VmTraceLogger) GetResult() (*VmTrace, error) {
	if l.err != nil {
		return nil, l.err
	}
	return l.result, nil
}

// addSize accounts n bytes against the trace size limit. When the limit is
// exceeded, the trace is aborted: the error is recorded and all accumulated
// data is released so it can be garbage collected.
func (l *VmTraceLogger) addSize(n uint64) {
	l.traceSize += n
	if l.traceSize > vmTraceSizeLimit {
		l.err = errVmTraceTooLarge
		l.traceStack = nil
		l.result = nil
		l.pendingStore = nil
	}
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

	newTrace := &VmTrace{
		Code: hexutil.Bytes(code),
		Ops:  make([]VmOperation, 0),
	}

	frame := &vmTraceFrame{
		trace:     newTrace,
		lastOpIdx: -1,
	}

	l.addSize(uint64(len(code)))
	if l.err != nil {
		return
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
	if l.err != nil || len(l.traceStack) == 0 {
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
func (l *VmTraceLogger) onOpcode(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, _ []byte, _ int, _ error) {
	if l.err != nil || len(l.traceStack) == 0 {
		return
	}
	frame := l.traceStack[len(l.traceStack)-1]

	// Finalize the previous operation's Push, Mem, and Store now that we have
	// the post execution state.
	if frame.lastOpIdx >= 0 {
		prevOp := &frame.trace.Ops[frame.lastOpIdx]
		if prevOp.Ex != nil {
			prevOp.Ex.Push = computePushed(frame.lastOp, scope.StackData())
			// scope.MemoryData() here is post-execution of the previous op.
			// Record only the memory region the previous op touched, matching
			// the Parity/OpenEthereum "mem" semantics. For CALL-family ops this
			// fires when the frame resumes after the sub-call, so the region
			// holds the return data written to the caller's memory.
			if frame.showMem && frame.memLen > 0 {
				data := memoryRegion(scope.MemoryData(), frame.memOff, frame.memLen)
				prevOp.Ex.Mem = &MemoryDiff{Off: frame.memOff, Data: data}
				l.addSize(uint64(len(data)))
				if l.err != nil {
					return
				}
			}
			if l.pendingStore != nil {
				prevOp.Ex.Store = l.pendingStore
				l.pendingStore = nil
			}
		}
	}

	// Create a new operation entry. Ex.Used = gas before this opcode (gasCopy from
	// interpreter, captured before any gas deduction for this op).
	newOp := VmOperation{
		Op:   vm.OpCode(op).String(),
		PC:   pc,
		Cost: cost,
		Ex:   &VmExecutedOperation{Used: gas, Push: []hexutil.Big{}},
	}
	frame.trace.Ops = append(frame.trace.Ops, newOp)
	frame.lastOpIdx = len(frame.trace.Ops) - 1
	frame.lastOp = vm.OpCode(op)

	// Determine the memory region this opcode will touch. The stack is
	// pre-execution here, so operand offsets/lengths can be read from it.
	frame.showMem, frame.memOff, frame.memLen = memTouchedRegion(vm.OpCode(op), scope.StackData())
	l.addSize(vmTraceOpOverhead)
}

// stackBack returns the n-th element from the top of the stack (0 = top).
func stackBack(st []uint256.Int, n int) *uint256.Int {
	return &st[len(st)-1-n]
}

// memTouchedRegion returns the memory region reported for an opcode in the
// Parity/OpenEthereum vmTrace format: the region written by the op (MLOAD is
// the historical exception — Parity reports the region it reads). Offsets and
// lengths come from the pre-execution stack. CREATE/CREATE2/SELFDESTRUCT
// report no region, matching OpenEthereum.
func memTouchedRegion(op vm.OpCode, st []uint256.Int) (show bool, off, size uint64) {
	switch op {
	case vm.MSTORE, vm.MLOAD:
		if len(st) > 0 {
			return true, stackBack(st, 0).Uint64(), 32
		}
	case vm.MSTORE8:
		if len(st) > 0 {
			return true, stackBack(st, 0).Uint64(), 1
		}
	case vm.RETURNDATACOPY, vm.CALLDATACOPY, vm.CODECOPY, vm.MCOPY:
		if len(st) > 2 {
			return true, stackBack(st, 0).Uint64(), stackBack(st, 2).Uint64()
		}
	case vm.EXTCODECOPY:
		if len(st) > 3 {
			return true, stackBack(st, 1).Uint64(), stackBack(st, 3).Uint64()
		}
	case vm.STATICCALL, vm.DELEGATECALL:
		if len(st) > 5 {
			return true, stackBack(st, 4).Uint64(), stackBack(st, 5).Uint64()
		}
	case vm.CALL, vm.CALLCODE:
		if len(st) > 6 {
			return true, stackBack(st, 5).Uint64(), stackBack(st, 6).Uint64()
		}
	}
	return false, 0, 0
}

// memoryRegion copies [off, off+size) from mem, zero-padding any part that
// lies beyond the current memory size.
func memoryRegion(mem []byte, off, size uint64) hexutil.Bytes {
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
	if l.err != nil || len(l.traceStack) == 0 {
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
