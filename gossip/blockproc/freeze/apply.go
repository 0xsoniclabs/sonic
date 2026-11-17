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
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// frozenRevert is the revert data of transactions failed by the freeze, the
// encoding of Error("account is frozen").
var frozenRevert = func() []byte {
	stringType, _ := abi.NewType("string", "", nil)
	reason, err := abi.Arguments{{Type: stringType}}.Pack("account is frozen")
	if err != nil {
		panic(err)
	}
	return append(common.FromHex("0x08c379a0"), reason...)
}()

// NewEVM creates an EVM that tracks moves of frozen accounts' funds, for
// ApplyMessage to fail the transactions that made them. Tracers see such a
// transaction's outermost call revert with "account is frozen".
func NewEVM(context vm.BlockContext, db vm.StateDB, chainConfig *params.ChainConfig, config vm.Config) *vm.EVM {
	state := Wrap(db)
	var evm *vm.EVM
	if hooks := config.Tracer; hooks != nil && hooks.OnExit != nil {
		wrapped := *hooks
		wrapped.OnExit = func(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
			if depth == 0 && err == nil && state.mustFail(evm.Origin) {
				output, err, reverted = frozenRevert, vm.ErrExecutionReverted, true
			}
			hooks.OnExit(depth, output, gasUsed, err, reverted)
		}
		config.Tracer = &wrapped
	}
	evm = vm.NewEVM(context, state, chainConfig, config)
	return evm
}

// ApplyMessage runs msg like core.ApplyMessage. On an EVM created by NewEVM,
// a message whose execution moved a frozen account's funds is rolled back as
// a whole, EIP-7702 authorizations included, and settled as a failed
// transaction: the nonce is used, the gas is charged and the result reverts
// with "account is frozen". Sonic pays no tip to the coinbase, so the sender's
// payment is the whole fee.
func ApplyMessage(evm *vm.EVM, msg *core.Message, gp *core.GasPool) (*core.ExecutionResult, error) {
	state, ok := evm.StateDB.(*StateDB)
	if !ok {
		return core.ApplyMessage(evm, msg, gp)
	}
	snapshot := state.Snapshot()
	nonce := state.GetNonce(msg.From)
	result, err := core.ApplyMessage(evm, msg, gp)
	if err != nil || !state.mustFail(msg.From) {
		return result, err
	}
	state.RevertToSnapshot(snapshot)
	state.SetNonce(msg.From, nonce+1, tracing.NonceChangeEoACall)
	fee := new(uint256.Int).Mul(uint256.NewInt(result.UsedGas), uint256.MustFromBig(msg.GasPrice))
	state.SubBalance(msg.From, fee, tracing.BalanceDecreaseGasBuy)
	return &core.ExecutionResult{
		UsedGas:    result.UsedGas,
		MaxUsedGas: result.MaxUsedGas,
		Err:        vm.ErrExecutionReverted,
		ReturnData: frozenRevert,
	}, nil
}

// mustFail reports whether the running transaction of sender moved a frozen
// account's funds. Internal transactions (zero sender), go-ethereum's system
// calls and opted-out operators are exempt.
func (s *StateDB) mustFail(sender common.Address) bool {
	return s.frozenMoves > 0 &&
		sender != (common.Address{}) &&
		sender != params.SystemAddress &&
		!IsOptedOut(s, sender)
}
