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

package rpctest

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func Test_NewBackendBuilder_CanSetChainID(t *testing.T) {

	be := NewBackendBuilder(t).Build()
	require.EqualValues(t, opera.FakeNetworkID, be.ChainID().Uint64())

	for _, v := range []uint64{1, 123, 9999} {
		be := NewBackendBuilder(t).WithChainID(v).Build()
		require.EqualValues(t, v, be.ChainID().Uint64())
	}
}

func Test_NewBackendBuilder_CanSetBlockHistory(t *testing.T) {
	be := NewBackendBuilder(t).Build()
	require.EqualValues(t, 1, be.CurrentBlock().NumberU64())
	be = NewBackendBuilder(t).WithBlockHistory(nil).Build()
	require.EqualValues(t, 1, be.CurrentBlock().NumberU64())

	for _, v := range []uint64{1, 2, 3} {

		blocks := make([]Block, v)
		for i := uint64(0); i < v; i++ {
			blocks[i] = Block{Number: i + 1}
		}

		be := NewBackendBuilder(t).WithBlockHistory(blocks).Build()
		require.EqualValues(t, v, be.CurrentBlock().NumberU64())
	}
}

func Test_NewBackendBuilder_CanSetTxPool(t *testing.T) {
	be := NewBackendBuilder(t).Build()
	require.Panics(t, func() {
		_ = be.SendTx(t.Context(), types.NewTx(&types.LegacyTx{}))
	})

	ctrl := gomock.NewController(t)
	mockPool := NewMockTxPool(ctrl)

	be = NewBackendBuilder(t).WithPool(mockPool).Build()
	mockPool.EXPECT().AddLocal(gomock.Any()).Return(nil).Times(1)

	err := be.SendTx(t.Context(), types.NewTx(&types.LegacyTx{}))
	require.NoError(t, err)
}

func Test_NewBackendBuilder_CanSetInitialState(t *testing.T) {

	var (
		addr1 = common.HexToAddress("0x01")
		addr2 = common.HexToAddress("0x02")
	)

	be := NewBackendBuilder(t).Build()
	latest := rpc.BlockNumber(1)
	state, block, err := be.StateAndBlockByNumberOrHash(t.Context(), rpc.BlockNumberOrHash{BlockNumber: &latest})
	require.NoError(t, err)
	require.NotNil(t, state)
	require.NotNil(t, block)

	zero := state.GetBalance(addr1)
	require.Zero(t, zero.Sign(), "expected zero balance")

	be = NewBackendBuilder(t).
		WithAccount(addr1, AccountState{Balance: big.NewInt(42)}).
		WithAccount(addr2, AccountState{Balance: big.NewInt(43)}).
		Build()
	state, block, err = be.StateAndBlockByNumberOrHash(t.Context(), rpc.BlockNumberOrHash{BlockNumber: &latest})
	require.NoError(t, err)
	require.NotNil(t, state)
	require.NotNil(t, block)

	require.EqualValues(t, state.GetBalance(addr1).Uint64(), 42)
	require.EqualValues(t, state.GetBalance(addr2).Uint64(), 43)
}

func Test_FakeBackend_ProducesCompatibleSigners(t *testing.T) {

	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	for _, chainId := range []uint64{1, 123, 9999} {
		be := NewBackendBuilder(t).WithChainID(chainId).Build()

		signer := be.GetSigner()

		tx, err := types.SignTx(
			types.NewTransaction(1, common.Address{0x42}, big.NewInt(0), 21000, big.NewInt(1), nil),
			signer,
			key,
		)
		require.NoError(t, err)

		referenceSigner := types.LatestSignerForChainID(big.NewInt(int64(chainId)))
		recovered, err := referenceSigner.Sender(tx)
		require.NoError(t, err)
		require.Equal(t, crypto.PubkeyToAddress(key.PublicKey), recovered)
	}
}

func Test_FakeBackend_DefaultBlockHistory(t *testing.T) {
	blockHistory := defaultBlockHistory()
	require.EqualValues(t, 1, len(blockHistory))
	require.EqualValues(t, 1, blockHistory[0].Number)
	require.Equal(t, common.HexToHash("0x1"), blockHistory[0].Hash)
}

func Test_FakeBackend_MultipleUpgrades(t *testing.T) {
	be := NewBackendBuilder(t).
		WithUpgrade(0, opera.GetSonicUpgrades()).
		WithUpgrade(10, opera.GetBrioUpgrades()).
		Build()

	// Check that Prague upgrade is active at correct block heights
	require.False(t, be.ChainConfig(0).IsPrague(big.NewInt(0), 0))
	require.False(t, be.ChainConfig(0).IsPrague(big.NewInt(10), 0))
}

func Test_FakeBackend_GetNetworkRules(t *testing.T) {
	be := NewBackendBuilder(t).
		WithUpgrade(0, opera.GetSonicUpgrades()).
		WithUpgrade(10, opera.GetAllegroUpgrades()).
		WithUpgrade(20, opera.GetBrioUpgrades()).
		Build()

	type expectedUpgrade struct {
		Sonic   bool
		Allegro bool
		Brio    bool
	}

	tests := []struct {
		name        string
		blockHeight idx.Block
		expected    expectedUpgrade
	}{
		{
			name:        "Before any upgrades",
			blockHeight: 5,
			expected: expectedUpgrade{
				Sonic:   true,
				Allegro: false,
				Brio:    false,
			},
		},
		{
			name:        "After Allegro upgrade",
			blockHeight: 15,
			expected: expectedUpgrade{

				Sonic:   true,
				Allegro: true,
				Brio:    false,
			},
		},
		{
			name:        "After Brio upgrade",
			blockHeight: 25,
			expected: expectedUpgrade{
				Sonic:   true,
				Allegro: true,
				Brio:    true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rules, err := be.GetNetworkRules(t.Context(), tt.blockHeight)
			require.NoError(t, err)
			require.NotNil(t, rules)
			require.Equal(t, tt.expected.Sonic, rules.Upgrades.Sonic, "unexpected Sonic upgrade status")
			require.Equal(t, tt.expected.Allegro, rules.Upgrades.Allegro, "unexpected Allegro upgrade status")
			require.Equal(t, tt.expected.Brio, rules.Upgrades.Brio, "unexpected Brio upgrade status")
		})
	}
}

func Test_FakeBackend_BlockByNumber(t *testing.T) {
	be := NewBackendBuilder(t).
		WithBlockHistory([]Block{
			{Number: 1, Hash: common.HexToHash("0x1")},
			{Number: 2, Hash: common.HexToHash("0x2")},
			{Number: 3, Hash: common.HexToHash("0x3")},
		}).
		Build()

	tests := []struct {
		name          string
		blockNumber   rpc.BlockNumber
		expected      uint64
		errorContains string
	}{
		{name: "Latest block", blockNumber: rpc.LatestBlockNumber, expected: 3},
		{name: "Pending block", blockNumber: rpc.PendingBlockNumber, expected: 3},
		{name: "Safe block", blockNumber: rpc.SafeBlockNumber, expected: 3},
		{name: "Finalized block", blockNumber: rpc.FinalizedBlockNumber, expected: 3},
		{name: "Earliest block", blockNumber: rpc.EarliestBlockNumber, expected: 1},
		{name: "Specific block number", blockNumber: 1, expected: 1},
		{name: "Specific block number", blockNumber: 2, expected: 2},
		{name: "Specific block number", blockNumber: 3, expected: 3},
		{name: "Non-existent block number", blockNumber: 4, errorContains: "block number not found"},
		{name: "Negative block number", blockNumber: -10, errorContains: "block number not found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			block, err := be.BlockByNumber(t.Context(), tt.blockNumber)
			if tt.errorContains != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorContains)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, block)
			require.EqualValues(t, tt.expected, block.NumberU64())
		})
	}
}

func Test_FakeBackend_GetReceiptsByNumber(t *testing.T) {

	be := NewBackendBuilder(t).
		WithBlockHistory([]Block{
			{
				Number: 1,
				Hash:   common.HexToHash("0x1"),
				Transactions: map[common.Hash]*Transaction{
					common.HexToHash("0xabc"): {
						tx: types.NewTx(
							&types.LegacyTx{},
						),
						blockNumber: 2,
						txIndex:     0,
						receipt:     &types.Receipt{},
					},
					common.HexToHash("0xdef"): {
						tx: types.NewTx(
							&types.LegacyTx{},
						),
						blockNumber: 2,
						txIndex:     0,
						receipt:     &types.Receipt{},
					},
				},
			},
			{
				Number: 2,
				Hash:   common.HexToHash("0x2"),
				Transactions: map[common.Hash]*Transaction{
					common.HexToHash("0xijk"): {
						tx: types.NewTx(
							&types.LegacyTx{},
						),
						blockNumber: 2,
						txIndex:     0,
						receipt:     &types.Receipt{},
					},
				},
			},
		}).
		Build()

	tests := []struct {
		name          string
		blockNumber   rpc.BlockNumber
		expectedCount int
		errorContains string
	}{
		{name: "Receipts for block 1", blockNumber: 1, expectedCount: 2},
		{name: "Receipts for block 2", blockNumber: 2, expectedCount: 1},
		{name: "Non-existent block number", blockNumber: 3, errorContains: "block number not found"},
		{name: "Negative block number", blockNumber: -10, errorContains: "block number not found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			receipts, err := be.GetReceiptsByNumber(t.Context(), tt.blockNumber)
			if tt.errorContains != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorContains)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, receipts)
			require.Len(t, receipts, tt.expectedCount)
		})
	}
}

func Test_FakeBackend_GetTransaction(t *testing.T) {

	tx := types.NewTx(&types.LegacyTx{})

	be := NewBackendBuilder(t).
		WithBlockHistory([]Block{
			{
				Number: 2,
				Hash:   common.HexToHash("0x1"),
				Transactions: map[common.Hash]*Transaction{
					tx.Hash(): {
						tx:          tx,
						blockNumber: 2,
						txIndex:     0,
						receipt:     &types.Receipt{},
					},
				},
			},
		}).
		Build()

	tests := []struct {
		name                string
		txHash              common.Hash
		expectedFound       bool
		expectedBlockNumber uint64
		expectedTxIndex     uint64
	}{
		{name: "Existing transaction", txHash: tx.Hash(), expectedFound: true, expectedBlockNumber: 2, expectedTxIndex: 0},
		{name: "Non-existent transaction", txHash: common.HexToHash("0xdef"), expectedFound: false, expectedBlockNumber: 0, expectedTxIndex: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx, blockNr, txIndex, err := be.GetTransaction(t.Context(), tt.txHash)

			if tt.expectedFound {
				require.NoError(t, err)
				require.NotNil(t, tx)
				require.Equal(t, tt.expectedBlockNumber, blockNr)
				require.Equal(t, tt.expectedTxIndex, txIndex)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), "transaction not found")
				require.Nil(t, tx)
				require.Equal(t, tt.expectedBlockNumber, blockNr)
				require.Equal(t, tt.expectedTxIndex, txIndex)
			}
		})
	}
}

func Test_FakeBackend_HeaderByNumber(t *testing.T) {

	be := NewBackendBuilder(t).
		WithBlockHistory([]Block{
			{Number: 1, Hash: common.HexToHash("0x1")},
			{Number: 2, Hash: common.HexToHash("0x2")},
			{Number: 3, Hash: common.HexToHash("0x3")},
		}).
		Build()

	tests := []struct {
		name          string
		blockNumber   rpc.BlockNumber
		expected      uint64
		errorContains string
	}{
		{name: "Latest block", blockNumber: rpc.LatestBlockNumber, expected: 3},
		{name: "Pending block", blockNumber: rpc.PendingBlockNumber, expected: 3},
		{name: "Safe block", blockNumber: rpc.SafeBlockNumber, expected: 3},
		{name: "Finalized block", blockNumber: rpc.FinalizedBlockNumber, expected: 3},
		{name: "Earliest block", blockNumber: rpc.EarliestBlockNumber, expected: 1},
		{name: "Specific block number", blockNumber: 1, expected: 1},
		{name: "Specific block number", blockNumber: 2, expected: 2},
		{name: "Specific block number", blockNumber: 3, expected: 3},
		{name: "Non-existent block number", blockNumber: 4, errorContains: "block number not found"},
		{name: "Negative block number", blockNumber: -10, errorContains: "block number not found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header, err := be.HeaderByNumber(t.Context(), tt.blockNumber)
			if tt.errorContains != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorContains)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, header)
			require.EqualValues(t, tt.expected, header.Number.Uint64())
		})
	}
}

func Test_ToEvmHeader_BaseFee_included(t *testing.T) {
	tests := []struct {
		name            string
		block           Block
		expectedBaseFee *big.Int
	}{
		{
			name: "Base fee included",
			block: Block{
				Number:  1,
				Hash:    common.HexToHash("0x1"),
				BaseFee: big.NewInt(100),
			},
			expectedBaseFee: big.NewInt(100),
		},
		{
			name: "Base fee not included (zero)",
			block: Block{
				Number:  2,
				Hash:    common.HexToHash("0x2"),
				BaseFee: big.NewInt(0),
			},
			expectedBaseFee: big.NewInt(0),
		},
		{
			name: "Base fee not specified (nil)",
			block: Block{
				Number: 2,
				Hash:   common.HexToHash("0x2"),
			},
			expectedBaseFee: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evmHeader := ToEvmHeader(tt.block)
			require.NotNil(t, evmHeader)
			require.Equal(t, tt.expectedBaseFee, evmHeader.BaseFee)
		})
	}
}
