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
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/iblockproc"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

func TestEthApiBackend_GetNetworkRules_LoadsRulesFromEpoch(t *testing.T) {
	require := require.New(t)

	epoch := idx.Epoch(1)
	store := newInMemoryStoreWithGenesisData(t, opera.GetAllegroUpgrades(), 1, epoch)

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

	got, err := backend.GetNetworkRules(t.Context(), idx.Block(2))
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

	store := newInMemoryStoreWithGenesisData(t, opera.GetAllegroUpgrades(), 1, idx.Epoch(1))
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

func TestEthApiBackend_BlockByNumber_ReturnsLatestBlockWhenRequesting(t *testing.T) {
	require := require.New(t)

	lastArchiveBlockNumber := big.NewInt(5)

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
		"specific block number": {
			requestedBlock: rpc.BlockNumber(lastArchiveBlockNumber.Int64()),
		},
	}

	block := evmcore.EvmBlock{
		EvmHeader: evmcore.EvmHeader{Number: lastArchiveBlockNumber}}

	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			stateReader := NewMockStateReader(ctrl)
			stateReader.EXPECT().LastBlockWithArchiveState(true).Return(&block, nil).AnyTimes()
			stateReader.EXPECT().Block(common.Hash{}, uint64(5)).Return(&block).AnyTimes()
			stateReader.EXPECT().BlockStateDB(lastArchiveBlockNumber, gomock.Any()).
				Return(state.NewMockStateDB(ctrl), nil)

			backend := &EthAPIBackend{state: stateReader}

			block, err := backend.BlockByNumber(t.Context(), test.requestedBlock)
			require.NoError(err)
			require.NotNil(block, "Expected non-nil block for requested number %v", test.requestedBlock)
			require.Equal(uint64(5), block.NumberU64(), "Returned block number mismatch")
		})
	}
}

func TestEthApiBackend_BlockByNumber_ReturnsBlockZero_WhenRequestingEarliest(t *testing.T) {
	require := require.New(t)

	firstBlockNumber := big.NewInt(0)
	block := evmcore.EvmBlock{EvmHeader: evmcore.EvmHeader{Number: firstBlockNumber}}

	ctrl := gomock.NewController(t)
	stateReader := NewMockStateReader(ctrl)
	stateReader.EXPECT().Block(common.Hash{}, uint64(0)).Return(&block)
	stateReader.EXPECT().BlockStateDB(firstBlockNumber, gomock.Any()).Return(state.NewMockStateDB(ctrl), nil)

	backend := &EthAPIBackend{
		state: stateReader,
	}

	gotBlock, err := backend.BlockByNumber(t.Context(), rpc.EarliestBlockNumber)
	require.NoError(err)
	require.NotNil(gotBlock, "Expected non-nil block for earliest block request")
	require.Equal(uint64(0), gotBlock.NumberU64(), "Returned block number mismatch for earliest block request")
}

func TestEthApiBackend_BlockByNumber_ReturnsNil_WhenRequestedBlockIsNotInArchive(t *testing.T) {
	require := require.New(t)

	lastArchiveBlockNumber := big.NewInt(5)
	lastStoreBlock := evmcore.EvmBlock{
		EvmHeader: evmcore.EvmHeader{Number: lastArchiveBlockNumber}}

	ctrl := gomock.NewController(t)
	stateReader := NewMockStateReader(ctrl)
	// store returns block 5
	stateReader.EXPECT().Block(common.Hash{}, uint64(5)).Return(&lastStoreBlock)
	// archive returns error for block 5
	stateReader.EXPECT().BlockStateDB(lastArchiveBlockNumber, gomock.Any()).
		Return(nil, fmt.Errorf("block does not exists in archive"))

	backend := &EthAPIBackend{state: stateReader}

	block, err := backend.BlockByNumber(t.Context(), rpc.BlockNumber(5))
	// since the requested block is not in archive, we expect nil
	require.NoError(err)
	require.Nil(block, "Expected nil block for requested number 5 as it is not in archive")
}

func TestEthApiBackend_BlockByNumber_ReturnsNil_WhenRequestedBlockDiffersInArchiveAndStore(t *testing.T) {
	require := require.New(t)

	store := newInMemoryStoreWithGenesisData(t, opera.GetAllegroUpgrades(), 1, idx.Epoch(1))
	// overwrite an existing block in store to differ from archive
	newStoreBlock := idx.Block(2)

	// overwrite an existing block in store to differ from archive
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

func BenchmarkBlockByNumber_MissingBlock(b *testing.B) {
	store := newInMemoryStoreWithGenesisData(b, opera.GetAllegroUpgrades(), 1, idx.Epoch(2))
	backend := &EthAPIBackend{
		state: &EvmStateReader{
			store: store,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := backend.BlockByNumber(b.Context(), rpc.BlockNumber(1000))
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

func BenchmarkBlockByNumber_ExistingBlock(b *testing.B) {
	store := newInMemoryStoreWithGenesisData(b, opera.GetAllegroUpgrades(), 1, idx.Epoch(10))
	backend := &EthAPIBackend{
		state: &EvmStateReader{
			store: store,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := backend.BlockByNumber(b.Context(), rpc.BlockNumber(5))
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}
