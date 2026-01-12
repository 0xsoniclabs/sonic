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

package gossip

import (
	"fmt"
	"testing"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/iblockproc"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
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

func TestEthApiBackend_BlockByNumber_ReturnsBlockWhenRequesting(t *testing.T) {
	require := require.New(t)

	cases := map[string]struct {
		requestedBlock rpc.BlockNumber
	}{

		"latest block in store greater than archive": {
			requestedBlock: rpc.LatestBlockNumber,
		},
		"safe block": {
			requestedBlock: rpc.SafeBlockNumber,
		},
		"finalized block": {
			requestedBlock: rpc.FinalizedBlockNumber,
		},
		"pending block": {
			requestedBlock: rpc.PendingBlockNumber,
		},
	}

	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			// Genesis generates archive with block 2
			store := newInMemoryStoreWithGenesisData(t, opera.GetAllegroUpgrades(), 1, idx.Epoch(1))
			// Add an extra block in store which is not in archive
			newStoreBlock := idx.Block(3)
			store.SetBlock(
				newStoreBlock,
				inter.NewBlockBuilder().
					WithNumber(uint64(newStoreBlock)).
					Build(),
			)
			require.True(store.HasBlock(newStoreBlock))

			backend := &EthAPIBackend{
				state: &EvmStateReader{
					store: store,
				},
			}

			block, err := backend.BlockByNumber(t.Context(), test.requestedBlock)
			require.NoError(err)
			require.NotNil(block, "Expected non-nil block for requested number %v", test.requestedBlock)
			require.Equal(uint64(2), block.NumberU64(), "Returned block number mismatch")
		})
	}
}

func TestEthApiBackend_BlockByNumber_ReturnsBlockZero_WhenRequestingEarliest(t *testing.T) {
	require := require.New(t)

	backend := &EthAPIBackend{
		state: &EvmStateReader{
			store: newInMemoryStoreWithGenesisData(t,
				opera.GetAllegroUpgrades(),
				1,
				idx.Epoch(1)),
		},
	}

	block, err := backend.BlockByNumber(t.Context(), rpc.EarliestBlockNumber)
	require.NoError(err)
	require.NotNil(block, "Expected non-nil block for earliest block request")
	require.Equal(uint64(0), block.NumberU64(), "Returned block number mismatch for earliest block request")
}

func TestEthApiBackend_BlockByNumber_ReturnsNil_WhenRequestedBlockDiffersInArchiveAndStore(t *testing.T) {
	require := require.New(t)

	store := newInMemoryStoreWithGenesisData(t, opera.GetAllegroUpgrades(), 1, idx.Epoch(1))
	// Add an extra block in store which is not in archive
	newStoreBlock := idx.Block(2)
	store.SetBlock(
		newStoreBlock,
		inter.NewBlockBuilder().
			WithNumber(uint64(newStoreBlock)).
			Build(),
	)
	require.True(store.HasBlock(newStoreBlock))

	backend := &EthAPIBackend{
		state: &EvmStateReader{
			store: store,
		},
	}
	block, err := backend.BlockByNumber(t.Context(), rpc.BlockNumber(2))
	// since the same block is different in archive than in store, we expect nil
	require.NoError(err)
	require.Nil(block, "Expected nil block for requested number 2 as it differs in archive")
}

func TestEthApiBackend_BlockByNumber_ReturnsNil_WhenRequestedBlockInStoreButNotInArchive(t *testing.T) {
	require := require.New(t)

	store := newInMemoryStoreWithGenesisData(t, opera.GetAllegroUpgrades(), 1, idx.Epoch(1))
	// Add an extra block in store which is not in archive
	newStoreBlock := idx.Block(3)
	store.SetBlock(
		newStoreBlock,
		inter.NewBlockBuilder().
			WithNumber(uint64(newStoreBlock)).
			Build(),
	)
	require.True(store.HasBlock(newStoreBlock))

	backend := &EthAPIBackend{
		state: &EvmStateReader{
			store: store,
		},
	}
	block, err := backend.BlockByNumber(t.Context(), rpc.BlockNumber(3))
	// since the requested block is not in archive, we expect nil
	require.NoError(err)
	require.Nil(block, "Expected nil block for requested number 3 as it is not in archive")
}

func TestEthApiBackend_BlockByNumber_ReturnsBlock_WhenRequestedBlockInArchiveAndStoreAreSame(t *testing.T) {
	require := require.New(t)

	store := newInMemoryStoreWithGenesisData(t, opera.GetAllegroUpgrades(), 1, idx.Epoch(1))

	backend := &EthAPIBackend{
		state: &EvmStateReader{
			store: store,
		},
	}
	block, err := backend.BlockByNumber(t.Context(), rpc.BlockNumber(2))
	// since the requested block is the same in archive and store, we expect the block
	require.NoError(err)
	require.NotNil(block, "Expected non-nil block for requested number 2 as it is same in archive and store")
	require.Equal(uint64(2), block.NumberU64(), "Returned block number mismatch for requested number 2")
}
