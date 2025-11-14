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

package ethapi

import (
	"context"
	"maps"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies/registry"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestForkId_UpgradesProduceDifferentIds(t *testing.T) {
	tests := map[string]struct {
		upgrades opera.Upgrades
		want     forkId
	}{
		"Sonic": {
			upgrades: opera.GetSonicUpgrades(),
			want:     forkId{0x66, 0x42, 0x1a, 0x82},
		},
		"Allegro": {
			upgrades: opera.GetAllegroUpgrades(),
			want:     forkId{0x79, 0x36, 0x28, 0x9c},
		},
		"Brio": {
			upgrades: opera.GetBrioUpgrades(),
			want:     forkId{0xcb, 0x29, 0x12, 0x88},
		},
		// In a real case scenario, SingleProposer and GasSubsidies would be
		// turned on while another upgrade is activated, so we check that the
		// ForkId reflects these changes.
		"Sonic+SingleProposer": {
			upgrades: func() opera.Upgrades {
				upgrades := opera.GetSonicUpgrades()
				upgrades.SingleProposerBlockFormation = true
				return upgrades
			}(),
			want: forkId{0xc4, 0x7a, 0x72, 0x20},
		},
		"Allegro+GasSubsidies": {
			upgrades: func() opera.Upgrades {
				upgrades := opera.GetAllegroUpgrades()
				upgrades.GasSubsidies = true
				return upgrades
			}(),
			want: forkId{0xa1, 0x75, 0xc1, 0xbf},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := makeForkId(test.upgrades)
			require.NoError(t, err, "makeForkHash failed")
			require.Equal(t, test.want, got, "unexpected fork hash")
		})
	}
}

func TestMakeConfigFromUpgrade_Reports_AvailableSystemContracts(t *testing.T) {

	sonicHeight := idx.Block(1)
	allegroHeight := idx.Block(5)
	gasSubsidiesHeight := idx.Block(10)

	tests := map[string]struct {
		upgradeHeight    opera.UpgradeHeight
		wantSysContracts contractRegistry
		wantForkId       hexutil.Bytes
	}{
		"Sonic": {
			upgradeHeight: opera.UpgradeHeight{
				Upgrades: opera.GetSonicUpgrades(),
				Height:   sonicHeight,
			},
			wantSysContracts: contractRegistry{},
			wantForkId:       hexutil.Bytes{0x66, 0x42, 0x1a, 0x82},
		},
		"Allegro": {
			upgradeHeight: opera.UpgradeHeight{
				Upgrades: opera.GetAllegroUpgrades(),
				Height:   allegroHeight,
			},
			wantSysContracts: contractRegistry{"HISTORY_STORAGE": params.HistoryStorageAddress},
			wantForkId:       hexutil.Bytes{0x79, 0x36, 0x28, 0x9c},
		},
		"GasSubsidies": {
			upgradeHeight: opera.UpgradeHeight{
				Upgrades: func() opera.Upgrades {
					return opera.Upgrades{GasSubsidies: true}
				}(),
				Height: gasSubsidiesHeight,
			},
			wantSysContracts: contractRegistry{"GAS_SUBSIDY_REGISTRY": registry.GetAddress()},
			wantForkId:       hexutil.Bytes{0xbd, 0xa1, 0xab, 0x6d},
		},
		"Sonic+GasSubsidies": {
			upgradeHeight: opera.UpgradeHeight{
				Upgrades: func() opera.Upgrades {
					upgrades := opera.GetSonicUpgrades()
					upgrades.GasSubsidies = true
					return upgrades
				}(),
				Height: gasSubsidiesHeight,
			},
			wantSysContracts: contractRegistry{"GAS_SUBSIDY_REGISTRY": registry.GetAddress()},
			wantForkId:       hexutil.Bytes{0xf, 0xc4, 0xb1, 0x6c},
		},
		"Allegro+GasSubsidies": {
			upgradeHeight: opera.UpgradeHeight{
				Upgrades: func() opera.Upgrades {
					upgrades := opera.GetAllegroUpgrades()
					upgrades.GasSubsidies = true
					return upgrades
				}(),
				Height: gasSubsidiesHeight,
			},
			wantSysContracts: contractRegistry{
				"HISTORY_STORAGE":      params.HistoryStorageAddress,
				"GAS_SUBSIDY_REGISTRY": registry.GetAddress(),
			},
			wantForkId: hexutil.Bytes{0xa1, 0x75, 0xc1, 0xbf},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			chainId := big.NewInt(250)

			ctrl := gomock.NewController(t)
			backend := NewMockBackend(ctrl)
			backend.EXPECT().ChainID().Return(chainId)

			result, err := makeConfigFromUpgrade(context.Background(), backend, test.upgradeHeight)
			require.NoError(t, err, "unexpected error from makeConfigFromUpgrade")

			require.Equal(t, test.wantSysContracts, result.SystemContracts,
				"unexpected system contracts")
			require.Equal(t, test.wantForkId, result.ForkId,
				"unexpected fork id")
		})
	}
}

func TestEIP7910_Config_ReportsErrors(t *testing.T) {

	currentBlock := evmcore.EvmBlock{
		EvmHeader: evmcore.EvmHeader{
			Number: big.NewInt(5),
		},
	}

	tests := map[string]struct {
		backendSetup func(*MockBackend)
		expectedErr  string
	}{
		"fails to get current block": {
			backendSetup: func(mockBackend *MockBackend) {
				mockBackend.EXPECT().CurrentBlock().Return(nil)
			},
			expectedErr: "current block header not found",
		},
		"fails to get upgrade heights for current block": {
			backendSetup: func(mockBackend *MockBackend) {
				mockBackend.EXPECT().CurrentBlock().Return(&currentBlock)
				mockBackend.EXPECT().GetUpgradeHeights().Return(nil)
			},
			expectedErr: "no configs found",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			backend := NewMockBackend(ctrl)

			test.backendSetup(backend)

			api := NewPublicBlockChainAPI(backend)
			_, err := api.Config(context.Background())
			require.Error(t, err)
			require.Contains(t, err.Error(), test.expectedErr)
		})
	}
}

func TestEIP7910_Config_ReturnsConfigs(t *testing.T) {

	chainId := big.NewInt(250)
	currentBlock := evmcore.EvmBlock{
		EvmHeader: evmcore.EvmHeader{
			Number: big.NewInt(5),
		},
	}

	sonicPrecompiled := contractRegistry{
		"BLAKE2F":              common.BytesToAddress([]byte{0x09}),
		"BN254_ADD":            common.BytesToAddress([]byte{0x06}),
		"BN254_MUL":            common.BytesToAddress([]byte{0x07}),
		"BN254_PAIRING":        common.BytesToAddress([]byte{0x08}),
		"ECREC":                common.BytesToAddress([]byte{0x01}),
		"ID":                   common.BytesToAddress([]byte{0x04}),
		"KZG_POINT_EVALUATION": common.BytesToAddress([]byte{0x0A}),
		"MODEXP":               common.BytesToAddress([]byte{0x05}),
		"RIPEMD160":            common.BytesToAddress([]byte{0x03}),
		"SHA256":               common.BytesToAddress([]byte{0x02}),
	}

	allegroPrecompiled := maps.Clone(sonicPrecompiled)
	allegroPrecompiled["BLS12_G1ADD"] = common.BytesToAddress([]byte{0x0B})
	allegroPrecompiled["BLS12_G1MSM"] = common.BytesToAddress([]byte{0x0C})
	allegroPrecompiled["BLS12_G2ADD"] = common.BytesToAddress([]byte{0x0D})
	allegroPrecompiled["BLS12_G2MSM"] = common.BytesToAddress([]byte{0x0E})
	allegroPrecompiled["BLS12_PAIRING_CHECK"] = common.BytesToAddress([]byte{0x0F})
	allegroPrecompiled["BLS12_MAP_FP_TO_G1"] = common.BytesToAddress([]byte{0x10})
	allegroPrecompiled["BLS12_MAP_FP2_TO_G2"] = common.BytesToAddress([]byte{0x11})

	tests := map[string]struct {
		backendSetup func(*MockBackend)
		wantConfig   configResponse
	}{
		"only current block config": {
			backendSetup: func(mockBackend *MockBackend) {
				mockBackend.EXPECT().CurrentBlock().Return(&currentBlock)
				mockBackend.EXPECT().GetUpgradeHeights().
					Return([]opera.UpgradeHeight{{
						Upgrades: opera.GetSonicUpgrades(),
						Height:   idx.Block(1)}})
			},
			wantConfig: func() configResponse {
				sonicId, err := makeForkId(opera.GetSonicUpgrades())
				require.NoError(t, err, "makeForkId failed for sonic upgrades")
				return configResponse{Current: &config{
					ChainId:         (*hexutil.Big)(chainId),
					ForkId:          sonicId[:],
					Precompiles:     sonicPrecompiled,
					SystemContracts: activeSystemContracts(opera.GetSonicUpgrades()),
				},
				}
			}(),
		},
		"current and last block configs": {
			backendSetup: func(mockBackend *MockBackend) {
				mockBackend.EXPECT().CurrentBlock().Return(&currentBlock)
				mockBackend.EXPECT().GetUpgradeHeights().
					Return([]opera.UpgradeHeight{
						{
							Upgrades: opera.GetSonicUpgrades(),
							Height:   idx.Block(1),
						},
						{
							Upgrades: opera.GetAllegroUpgrades(),
							Height:   idx.Block(5),
						},
					})
			},
			wantConfig: func() configResponse {
				sonicId, err := makeForkId(opera.GetSonicUpgrades())
				require.NoError(t, err, "makeForkId failed for sonic upgrades")

				allegroId, err := makeForkId(opera.GetAllegroUpgrades())
				require.NoError(t, err, "makeForkId failed for allegro upgrades")

				return configResponse{
					Current: &config{
						ChainId:         (*hexutil.Big)(chainId),
						ForkId:          allegroId[:],
						Precompiles:     allegroPrecompiled,
						SystemContracts: activeSystemContracts(opera.GetAllegroUpgrades()),
					},
					Last: &config{
						ChainId:         (*hexutil.Big)(chainId),
						ForkId:          sonicId[:],
						Precompiles:     sonicPrecompiled,
						SystemContracts: activeSystemContracts(opera.GetSonicUpgrades()),
					},
				}
			}(),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			backend := NewMockBackend(ctrl)
			backend.EXPECT().ChainID().Return(chainId).AnyTimes()

			test.backendSetup(backend)

			api := NewPublicBlockChainAPI(backend)
			gotConfig, err := api.Config(context.Background())

			require.NoError(t, err)
			require.Equal(t, test.wantConfig, *gotConfig)
		})
	}
}
