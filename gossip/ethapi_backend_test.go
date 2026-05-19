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

package gossip

import (
	"fmt"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/emitter"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/iblockproc"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestEthApiBackend_GetNetworkRules_LoadsRulesFromEpoch(t *testing.T) {
	require := require.New(t)

	blockNumber := idx.Block(12)
	epoch := idx.Epoch(3)

	store, err := NewMemStore(t)
	require.NoError(err)

	store.SetBlock(
		blockNumber,
		inter.NewBlockBuilder().
			WithNumber(uint64(blockNumber)).
			WithEpoch(epoch).
			Build(),
	)
	require.True(store.HasBlock(blockNumber))

	rules := opera.FakeNetRules(opera.Upgrades{})
	rules.Name = "test-rules"

	store.SetHistoryBlockEpochState(
		epoch,
		iblockproc.BlockState{},
		iblockproc.EpochState{
			Epoch: epoch,
			Rules: rules,
		},
	)

	backend := &EthAPIBackend{
		svc: &Service{
			store: store,
		},
		state: &EvmStateReader{
			store: store,
		},
	}

	got, err := backend.GetNetworkRules(t.Context(), blockNumber)
	require.NoError(err)

	// Rules contain functions that cannot be compared directly,
	// so we compare their string representations.
	want := fmt.Sprintf("%+v", rules)
	have := fmt.Sprintf("%+v", got)
	require.Equal(want, have, "Network rules do not match")
}

func TestEthApiBackend_GetNetworkRules_MissingBlockReturnsNilRules(t *testing.T) {
	require := require.New(t)

	blockNumber := idx.Block(12)

	store, err := NewMemStore(t)
	require.NoError(err)
	require.False(store.HasBlock(blockNumber))

	backend := &EthAPIBackend{
		state: &EvmStateReader{
			store: store,
		},
	}

	rules, err := backend.GetNetworkRules(t.Context(), blockNumber)
	require.NoError(err)
	require.Nil(rules)
}

func TestEthApiBackend_IsTestOnlyApiEnabled_ReturnsConfigFlagValue(t *testing.T) {
	for _, enabled := range []bool{true, false} {
		t.Run(fmt.Sprintf("enabled=%v", enabled), func(t *testing.T) {
			require := require.New(t)

			backend := &EthAPIBackend{
				svc: &Service{
					config: Config{
						EnableTestOnlyApi: enabled,
					},
				},
			}

			result := backend.IsTestOnlyApiEnabled()
			require.Equal(enabled, result)
		})
	}
}

func TestEthApiBackend_ProposeTransactions_ReturnsErrorWhenTestOnlyApiDisabled(t *testing.T) {
	require := require.New(t)

	backend := &EthAPIBackend{
		svc: &Service{
			config: Config{
				EnableTestOnlyApi: false,
			},
		},
	}

	err := backend.ProposeTransactions(nil)
	require.Error(err)
	require.Contains(err.Error(), "disabled")
}

func TestEthApiBackend_ProposeTransactions_ReturnsErrorWhenNoEmitters(t *testing.T) {
	require := require.New(t)

	backend := &EthAPIBackend{
		svc: &Service{
			config:   Config{EnableTestOnlyApi: true},
			emitters: []*emitter.Emitter{}, // No emitters available
		},
	}

	err := backend.ProposeTransactions(nil)
	require.Error(err)
	require.Contains(err.Error(), "no emitters")
}

func TestEthApiBackend_proposeTransactionsInternal_ForwardsRequestToEmitter(t *testing.T) {
	ctrl := gomock.NewController(t)
	emitter := NewMockforceableEmitter(ctrl)

	backend := &EthAPIBackend{
		svc: &Service{
			config: Config{EnableTestOnlyApi: true},
		},
	}

	txs := []*types.Transaction{
		types.NewTx(&types.LegacyTx{Nonce: 1}),
		types.NewTx(&types.LegacyTx{Nonce: 2}),
	}

	emitter.EXPECT().ForceEventEmissionForTesting(txs).Return(nil)
	require.NoError(t, backend.proposeTransactionsInternal(txs, emitter))
}

func TestEthApiBackend_proposeTransactionsInternal_ReturnsEmissionIssue(t *testing.T) {
	ctrl := gomock.NewController(t)
	emitter := NewMockforceableEmitter(ctrl)

	backend := &EthAPIBackend{
		svc: &Service{
			config: Config{EnableTestOnlyApi: true},
		},
	}

	issue := fmt.Errorf("injected test issue")

	emitter.EXPECT().ForceEventEmissionForTesting(nil).Return(issue)
	require.ErrorIs(t, backend.proposeTransactionsInternal(nil, emitter), issue)
}

func TestEthApiBackend_epochWithDefault_RejectsEpochsAboveUint32Max(t *testing.T) {
	const currentEpoch = idx.Epoch(1000)

	store, err := NewMemStore(t)
	require.NoError(t, err)
	store.SetBlockEpochState(iblockproc.BlockState{}, iblockproc.EpochState{Epoch: currentEpoch})
	backend := &EthAPIBackend{
		svc: &Service{
			store: store,
		},
	}

	// 2^32 + currentEpoch truncates to currentEpoch when cast to uint32,
	// so a naive idx.Epoch(epoch) <= current check would incorrectly accept it.
	outOfRange := rpc.BlockNumber(1<<32 + int64(currentEpoch))
	_, err = backend.epochWithDefault(t.Context(), outOfRange)
	require.Error(t, err, "epoch value above uint32 max must be rejected")
}
