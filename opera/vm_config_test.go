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
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
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
				require.Equal(t, SonicPostAllegroMaxCodeSize, *vmConfig.MaxCodeSize)
				require.Equal(t, SonicPostAllegroMaxInitCodeSize, *vmConfig.MaxInitCodeSize)
			} else {
				require.Nil(t, vmConfig.MaxCodeSize)
				require.Nil(t, vmConfig.MaxInitCodeSize)
			}
		})
	}
}
