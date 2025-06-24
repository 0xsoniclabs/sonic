// Copyright 2025 Sonic Operations Ltd
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

package opera

import (
	"github.com/0xsoniclabs/sonic/opera/contracts/evmwriter"
	"github.com/0xsoniclabs/tosca/go/geth_adapter"
	"github.com/0xsoniclabs/tosca/go/interpreter/lfvm"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

// sonicVmConfig is the initial Ethereum VM configuration used for processing
// transactions on a Sonic chain using the Sonic hard-fork.
var sonicVmConfig = func() vm.Config {

	// For transaction processing, Tosca's LFVM is used.
	interpreter, err := lfvm.NewInterpreter(lfvm.Config{})
	if err != nil {
		panic(err)
	}
	lfvmFactory := geth_adapter.NewGethInterpreterFactory(interpreter)

	// For tracing, Geth's EVM is used.
	gethFactory := func(evm *vm.EVM) vm.Interpreter {
		return vm.NewEVMInterpreter(evm)
	}

	return vm.Config{
		StatePrecompiles: map[common.Address]vm.PrecompiledStateContract{
			evmwriter.ContractAddress: &evmwriter.PreCompiledContract{},
		},
		Interpreter:           lfvmFactory,
		InterpreterForTracing: gethFactory,

		// Fantom/Sonic modifications
		ChargeExcessGas:                 true,
		IgnoreGasFeeCap:                 true,
		InsufficientBalanceIsNotAnError: true,
		SkipTipPaymentToCoinbase:        true,
	}
}()

// GetVmConfig returns the VM configuration to be used for processing
// transactions under the given network rules.
func GetVmConfig(rules Rules) vm.Config {
	res := sonicVmConfig

	// don't charge excess gas in single proposer mode
	if rules.Upgrades.SingleProposerBlockFormation {
		res.ChargeExcessGas = false
	}

	return res
}
