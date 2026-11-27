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

package freeze

import (
	"bytes"
	"errors"
	"slices"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// ErrFrozenAccount is returned if a message is sent by an account listed as
// frozen in the freeze registry.
var ErrFrozenAccount = errors.New("account is frozen")

// frozenRevertData is the revert data of transactions failed by the freeze, the
// encoding of Error("account is frozen").
var frozenRevertData = func() []byte {
	stringType, err := abi.NewType("string", "", nil)
	if err != nil {
		panic(err)
	}
	reason, err := abi.Arguments{{Type: stringType}}.Pack(ErrFrozenAccount.Error())
	if err != nil {
		panic(err)
	}
	return append(crypto.Keccak256([]byte("Error(string)"))[:4], reason...)
}()

// errFrozenRevert is the error tracers see when the outermost call of a message
// moving frozen funds exits. Its text is that of vm.ErrExecutionReverted, so
// tracers keep go-ethereum's output, and errors.Is matches it with both
// vm.ErrExecutionReverted and ErrFrozenAccount. Unlike the revert data, a
// contract reverting with the reason "account is frozen" cannot produce it.
var errFrozenRevert error = frozenRevertError{}

type frozenRevertError struct{}

func (frozenRevertError) Error() string {
	return vm.ErrExecutionReverted.Error()
}

func (frozenRevertError) Is(target error) bool {
	return target == vm.ErrExecutionReverted || target == ErrFrozenAccount
}

// IsFrozenRevert reports whether result is the revert ApplyMessage reports for a
// message moving frozen funds. A message whose contract reverts with the reason
// "account is frozen" matches too, so a match does not prove that the freeze
// applied. Tracers get a proof instead: errors.Is matches the error of the
// outermost exit with ErrFrozenAccount.
func IsFrozenRevert(result *core.ExecutionResult) bool {
	return result != nil &&
		result.Err == vm.ErrExecutionReverted &&
		bytes.Equal(result.ReturnData, frozenRevertData)
}

// ApplyMessage runs msg like core.ApplyMessage. If checker is enabled, a frozen
// sender fails with ErrFrozenAccount, and a message moving frozen funds is fully
// rolled back, EIP-7702 authorizations included, and reverts, keeping only its
// nonce use and gas fee. That fee keeps the gas refund and pays no tip, so evm
// must set SkipTipPaymentToCoinbase. The result of such a message has the error
// vm.ErrExecutionReverted, as ExecutionResult.Revert requires, and the revert
// data of Error("account is frozen"); IsFrozenRevert recognizes it. Tracers see
// its outermost call revert, see withFreezeRevert.
func ApplyMessage(evm *vm.EVM, msg *core.Message, gp *core.GasPool, checker Checker) (*core.ExecutionResult, error) {
	// before the upgrade, run msg exactly as go-ethereum does
	if !checker.IsEnabled() {
		return core.ApplyMessage(evm, msg, gp)
	}

	// the sender is frozen, so fail immediately
	if checker.IsFrozen(msg.From) {
		return nil, ErrFrozenAccount
	}

	// while msg runs, the EVM's state tracks frozen moves and its tracer sees
	// the freeze's revert
	db, tracer := evm.StateDB, evm.Config.Tracer
	defer func() { evm.StateDB, evm.Config.Tracer = db, tracer }()
	state := Wrap(evm.StateDB, checker)
	evm.StateDB = state
	evm.Config.Tracer = withFreezeRevert(tracer, state, msg.From)

	snapshot := state.Snapshot()
	nonce := state.GetNonce(msg.From)
	result, err := core.ApplyMessage(evm, msg, gp)

	// the tx failed or did not touch frozen funds, so nothing to do
	if err != nil || !violatesFreeze(state, msg.From) {
		return result, err
	}

	// the tx succeeded but must fail because it moves frozen funds
	// revert all its effects, bump the nonce and charge the gas
	state.RevertToSnapshot(snapshot)
	reason := tracing.NonceChangeEoACall
	if msg.To == nil {
		reason = tracing.NonceChangeContractCreator
	}
	state.SetNonce(msg.From, nonce+1, reason)
	// charge the gas used, keeping the refund a real revert would drop
	fee := new(uint256.Int).Mul(uint256.NewInt(result.UsedGas), msg.GasPrice)
	state.SubBalance(msg.From, fee, tracing.BalanceDecreaseGasBuy)
	return &core.ExecutionResult{
		UsedGas:    result.UsedGas,
		MaxUsedGas: result.MaxUsedGas,
		Err:        vm.ErrExecutionReverted,
		ReturnData: slices.Clone(frozenRevertData),
	}, nil
}

// withFreezeRevert returns tracer with exit hooks that report the outermost call
// of sender's message as reverted with Error("account is frozen") and
// errFrozenRevert if state has recorded a frozen move by then. Inner calls keep
// their results. It returns tracer itself if tracer has no exit hook.
//
// Only the exit hooks change. The other hooks, such as those of balances,
// nonces, storage and logs, still see the effects that ApplyMessage then rolls
// back, followed by its writes of the nonce and the fee, so tracers must discard
// the effects of a call that exits reverted, as they do for any revert.
//
// The outermost call exits before the gas refund. If the refund goes to a
// sender that msg itself froze, the refund is a frozen move that reverts msg,
// while its tracer saw the outermost call succeed.
func withFreezeRevert(tracer *tracing.Hooks, state *StateDB, sender common.Address) *tracing.Hooks {
	if tracer == nil || (tracer.OnExit == nil && tracer.OnExitV2 == nil) {
		return tracer
	}
	exit := func(depth int, output []byte, reverted bool, err error) ([]byte, bool, error) {
		if depth == 0 && violatesFreeze(state, sender) {
			return slices.Clone(frozenRevertData), true, vm.VMErrorFromErr(errFrozenRevert)
		}
		return output, reverted, err
	}
	hooks := *tracer
	if tracer.OnExit != nil {
		hooks.OnExit = func(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
			output, reverted, err = exit(depth, output, reverted, err)
			tracer.OnExit(depth, output, gasUsed, err, reverted)
		}
	}
	if tracer.OnExitV2 != nil {
		hooks.OnExitV2 = func(depth int, output []byte, gasLeft tracing.Gas, err error, reverted bool) {
			output, reverted, err = exit(depth, output, reverted, err)
			tracer.OnExitV2(depth, output, gasLeft, err, reverted)
		}
	}
	return &hooks
}

// violatesFreeze reports whether sender's transaction moved frozen funds,
// exempting internal transactions and system calls.
func violatesFreeze(state *StateDB, sender common.Address) bool {
	return state.frozenMoved &&
		sender != (common.Address{}) &&
		sender != params.SystemAddress
}
