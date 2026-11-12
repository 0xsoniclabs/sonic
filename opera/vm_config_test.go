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

package opera

import (
	"fmt"
	"math"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	geth_state "github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestGetVmConfig_SingleProposerModeDisablesExcessGasCharging(t *testing.T) {
	for _, singleProposerMode := range []bool{true, false} {
		t.Run(fmt.Sprintf("SingleProposerModeEnabled=%t", singleProposerMode), func(t *testing.T) {
			require := require.New(t)
			rules := Rules{
				Upgrades: Upgrades{
					SingleProposerBlockFormation: singleProposerMode,
				},
			}

			vmConfig := GetVmConfig(rules)

			require.NotEqual(singleProposerMode, vmConfig.ChargeExcessGas)
		})
	}
}

func TestGetVmConfig_NonBrioUpgrade_DoesNotSetMaxTxGas(t *testing.T) {

	rules := Rules{
		Upgrades: Upgrades{
			Brio: false,
		},
	}

	vmConfig := GetVmConfig(rules)

	require.Nil(t, vmConfig.MaxTxGas)
}

func TestGetVmConfig_BrioUpgrade_CopiesMaxEventGasValue(t *testing.T) {
	want := uint64(123456)
	rules := Rules{
		Upgrades: Upgrades{
			Brio: true,
		},
		Economy: EconomyRules{
			Gas: GasRules{
				MaxEventGas: want,
			},
		},
	}

	vmConfig := GetVmConfig(rules)

	require.NotNil(t, vmConfig.MaxTxGas)
	require.Equal(t, want, *vmConfig.MaxTxGas)
	require.False(t, &rules.Economy.Gas.MaxEventGas == vmConfig.MaxTxGas)
}

func TestGetVmConfig_BrioUpgradeFromLfvmToSfvmInterpreter(t *testing.T) {
	tests := map[string]struct {
		brioEnabled bool
	}{
		"Allegro": {brioEnabled: false},
		"Brio":    {brioEnabled: true},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			rules := Rules{
				Upgrades: Upgrades{
					Sonic:   true,
					Allegro: true,
					Brio:    test.brioEnabled,
				},
			}

			vmConfig := GetVmConfig(rules)
			require.NotNil(t, vmConfig.Interpreter)

			// create a contract with code size exceeding the LFVM code size limit
			contract := vm.Contract{
				Code: make([]byte, math.MaxUint16+1),
			}

			// tosca VMs only support Istanbul and later
			zero := big.NewInt(0)
			chainConfig := &params.ChainConfig{
				HomesteadBlock:      zero,
				DAOForkBlock:        zero,
				EIP150Block:         zero,
				EIP155Block:         zero,
				EIP158Block:         zero,
				ByzantiumBlock:      zero,
				ConstantinopleBlock: zero,
				PetersburgBlock:     zero,
				IstanbulBlock:       zero,
			}

			ctrl := gomock.NewController(t)
			stateDb := state.NewMockStateDB(ctrl)
			stateDb.EXPECT().AddRefund(gomock.Any())
			stateDb.EXPECT().GetRefund().Return(uint64(0))
			stateDb.EXPECT().SubRefund(gomock.Any())
			if test.brioEnabled {
				stateDb.EXPECT().AddRefund(gomock.Any())
			}

			evm := vm.NewEVM(vm.BlockContext{BlockNumber: big.NewInt(1)}, stateDb, chainConfig, vmConfig)
			_, err := vmConfig.Interpreter(evm).Interpret(&contract, []byte{}, false)

			// verify that no code size limit is enforced when Brio (SFVM) is enabled
			if test.brioEnabled {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, "max code size exceeded")
			}
		})
	}
}

func TestCodeSizeLimits_BrioSetsCustomCodeSizeLimits(t *testing.T) {
	tests := map[string]bool{
		"Allegro": false,
		"Brio":    true,
	}

	for name, brioEnabled := range tests {
		t.Run(name, func(t *testing.T) {
			rules := Rules{
				Upgrades: Upgrades{
					Sonic:   true,
					Allegro: true,
					Brio:    brioEnabled,
				},
			}

			vmConfig := GetVmConfig(rules)

			if brioEnabled {
				require.NotNil(t, vmConfig.MaxCodeSize)
				require.NotNil(t, vmConfig.MaxInitCodeSize)
				require.NotNil(t, vmConfig.MaxInitCodeSizeVMCreate)
				require.Equal(t, SonicPostAllegroMaxCodeSize, *vmConfig.MaxCodeSize)
				require.Equal(t, SonicPostAllegroMaxInitCodeSize, *vmConfig.MaxInitCodeSize)
				require.Equal(t, SonicPostAllegroVMCreateMaxInitCodeSize, *vmConfig.MaxInitCodeSizeVMCreate)
			} else {
				require.Nil(t, vmConfig.MaxCodeSize)
				require.Nil(t, vmConfig.MaxInitCodeSize)
				require.Nil(t, vmConfig.MaxInitCodeSizeVMCreate)
			}
		})
	}
}

func TestCodeSizeLimits_CreateOpcodesHaveTheSameInitCodeSizeLimitInAllInterpreters(t *testing.T) {
	for name, upgrades := range GetAllHardForksInOrder() {
		limit := uint64(params.MaxInitCodeSize)
		if upgrades.Brio {
			limit = SonicPostAllegroVMCreateMaxInitCodeSize
		}
		for _, op := range []vm.OpCode{vm.CREATE, vm.CREATE2} {
			for size, wantSuccess := range map[uint64]bool{limit: true, limit + 1: false} {
				t.Run(fmt.Sprintf("%s/%v/size=%d", name, op, size), func(t *testing.T) {
					vmConfig := GetVmConfig(Rules{Upgrades: upgrades})
					interpreters := map[string]vm.InterpreterFactory{
						"processing": vmConfig.Interpreter,
						"tracing":    vmConfig.InterpreterForTracing,
					}
					gasLeft := map[string]uint64{}
					for interpreterName, interpreter := range interpreters {
						vmConfig.Interpreter = interpreter
						created, left, err := createWithInitCodeSize(t, upgrades, vmConfig, op, size)
						gasLeft[interpreterName] = left
						if wantSuccess {
							require.NoError(t, err, interpreterName)
							require.NotZero(t, created, interpreterName)
						} else if interpreterName == "tracing" {
							require.ErrorContains(t, err, vm.ErrMaxInitCodeSizeExceeded.Error())
						} else {
							require.Error(t, err, interpreterName)
						}
					}
					require.Equal(t, gasLeft["processing"], gasLeft["tracing"])
				})
			}
		}
	}
}

// createWithInitCodeSize runs a contract that creates a contract from
// initCodeSize zero bytes with the given opcode and returns its address
// and the gas left.
func createWithInitCodeSize(
	t *testing.T,
	upgrades Upgrades,
	vmConfig vm.Config,
	op vm.OpCode,
	initCodeSize uint64,
) (common.Address, uint64, error) {
	t.Helper()
	stateDB, err := geth_state.New(types.EmptyRootHash, geth_state.NewDatabaseForTesting())
	require.NoError(t, err)

	var code []byte
	if op == vm.CREATE2 {
		code = append(code, byte(vm.PUSH1), 0) // salt
	}
	code = append(code,
		byte(vm.PUSH3), byte(initCodeSize>>16), byte(initCodeSize>>8), byte(initCodeSize),
		byte(vm.PUSH1), 0, // offset
		byte(vm.PUSH1), 0, // value
		byte(op),
		byte(vm.PUSH1), 0,
		byte(vm.MSTORE),
		byte(vm.PUSH1), 32,
		byte(vm.PUSH1), 0,
		byte(vm.RETURN),
	)
	factory := common.Address{1}
	stateDB.SetCode(factory, code, tracing.CodeChangeUnspecified)

	chainConfig := CreateTransientEvmChainConfig(
		FakeNetworkID,
		[]UpgradeHeight{MakeUpgradeHeight(upgrades, 0)},
		0,
	)
	blockContext := vm.BlockContext{
		CanTransfer: core.CanTransfer,
		Transfer:    core.Transfer,
		BlockNumber: big.NewInt(0),
		Random:      &common.Hash{},
	}
	evm := vm.NewEVM(blockContext, stateDB, chainConfig, vmConfig)
	ret, gasLeft, err := evm.Call(common.Address{2}, factory, nil, 10_000_000, uint256.NewInt(0))
	return common.BytesToAddress(ret), gasLeft, err
}
