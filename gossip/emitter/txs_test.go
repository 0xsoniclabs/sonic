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

package emitter

import (
	"fmt"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/gossip/emitter/config"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func Test_DefaultMaxTxsPerAddress_Equals_txTurnNonces(t *testing.T) {

	// Although MaxTxsPerAddress can be configured, having a value less than txTurnNonces
	// could yield performance issues when dispatching batches of transactions.
	// MaxTxsPerAddress should be greater or equal to txTurnNonces to ensure timely
	// emission of transactions. Default value for this parameter should be exactly txTurnNonces.

	defaultConfig := config.DefaultConfig()
	require.EqualValues(t, txTurnNonces, defaultConfig.MaxTxsPerAddress, "Default MaxTxsPerAddress should equal txTurnNonces")
}

func Test_Emitter_isValidBundleTx_AcceptsValidBundleIfBundlesAreEnabled(t *testing.T) {
	for _, bundlesEnabled := range []bool{true, false} {
		t.Run(fmt.Sprintf("enabled=%t", bundlesEnabled), func(t *testing.T) {
			require := require.New(t)
			ctrl := gomock.NewController(t)

			rules := opera.Rules{
				Upgrades: opera.Upgrades{
					TransactionBundles: bundlesEnabled,
				},
			}

			external := NewMockExternal(ctrl)
			external.EXPECT().GetRules().Return(rules)
			external.EXPECT().GetLatestBlockIndex().Return(idx.Block(100)).AnyTimes()
			external.EXPECT().HasBundleRecentlyBeenProcessed(gomock.Any()).AnyTimes()

			emitter := &Emitter{
				world: World{External: external},
			}

			tx := types.NewTx(&types.LegacyTx{
				To: &bundle.BundleAddress,
				Data: bundle.Encode(bundle.TransactionBundle{
					Version:  1,
					Earliest: 50,
					Latest:   150,
				}),
			})
			_, _, err := bundle.ValidateTransactionBundle(tx, nil)
			require.NoError(err)

			require.Equal(bundlesEnabled, emitter.isValidBundleTx(tx))
		})
	}
}

func Test_Emitter_isValidBundleTx_RejectsInvalidBundle(t *testing.T) {
	tests := map[string]*types.Transaction{
		"not a bundle": types.NewTx(&types.LegacyTx{}),
		"invalid bundle data": types.NewTx(&types.LegacyTx{
			To:   &bundle.BundleAddress,
			Data: []byte{0x01, 0x02, 0x03},
		}),
		"bundle with out-of-range block numbers": types.NewTx(&types.LegacyTx{
			To: &bundle.BundleAddress,
			Data: bundle.Encode(bundle.TransactionBundle{
				Version:  1,
				Earliest: 150,
				Latest:   250,
			}),
		}),
	}

	for name, tx := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			ctrl := gomock.NewController(t)

			rules := opera.Rules{
				Upgrades: opera.Upgrades{
					TransactionBundles: true,
				},
			}

			external := NewMockExternal(ctrl)
			external.EXPECT().GetRules().Return(rules).AnyTimes()
			external.EXPECT().GetLatestBlockIndex().Return(idx.Block(100)).AnyTimes()
			external.EXPECT().HasBundleRecentlyBeenProcessed(gomock.Any()).AnyTimes()

			emitter := &Emitter{
				world: World{External: external},
			}

			require.False(emitter.isValidBundleTx(tx))
		})
	}
}

func Test_Emitter_isValidBundleTx_RejectsAlreadyProcessedBundle(t *testing.T) {
	for _, processed := range []bool{true, false} {
		t.Run(fmt.Sprintf("processed=%t", processed), func(t *testing.T) {
			ctrl := gomock.NewController(t)

			rules := opera.Rules{
				Upgrades: opera.Upgrades{
					TransactionBundles: true,
				},
			}

			external := NewMockExternal(ctrl)
			external.EXPECT().GetRules().Return(rules).AnyTimes()
			external.EXPECT().GetLatestBlockIndex().Return(idx.Block(100)).AnyTimes()
			external.EXPECT().HasBundleRecentlyBeenProcessed(gomock.Any()).Return(processed).AnyTimes()

			emitter := &Emitter{
				world: World{External: external},
			}

			tx := types.NewTx(&types.LegacyTx{
				To: &bundle.BundleAddress,
				Data: bundle.Encode(bundle.TransactionBundle{
					Version:  1,
					Earliest: 50,
					Latest:   150,
				}),
			})
			_, _, err := bundle.ValidateTransactionBundle(tx, nil)
			require.NoError(t, err)

			require.Equal(t, !processed, emitter.isValidBundleTx(tx))
		})
	}
}
