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

package evmcore

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// This file contains tests related to gas subsidies in the transaction pool.
// This file is intentionally separated from other pool tests to avoid polluting
// them with extra test tools.

func TestTxPool_SponsoredTransactionsAreIncludedInThePendingSet(t *testing.T) {
	ctrl := gomock.NewController(t)

	chainId := big.NewInt(1)
	blockNumber := idx.Block(1)
	poolConfig := TxPoolConfig{MinimumTip: 15}
	upgrades := opera.Upgrades{GasSubsidies: true}

	// Create a ChainConfig instance with the expected features enabled
	// at the block height.
	chainConfig := opera.CreateTransientEvmChainConfig(
		chainId.Uint64(),
		[]opera.UpgradeHeight{{Upgrades: upgrades, Height: 0}},
		blockNumber,
	)

	// mock the external chain dependencies
	chain := mockChain(ctrl, chainConfig, upgrades)

	// Instantiate the pool
	pool := NewTxPool(poolConfig, chainConfig, chain)

	// transactions per sender
	const transactionsPerSender = 5

	// Queue some sponsored transactions
	const sponsoredBatches = 5
	for range sponsoredBatches {
		txs := createTransactions(t, &types.LegacyTx{
			GasPrice: big.NewInt(0),
			Gas:      21_000,
			To:       &common.Address{1}, // not a contract creation
		}, chainId, transactionsPerSender)

		for _, tx := range txs {
			err := pool.addRemoteSync(tx)
			require.NoError(t, err)
		}
	}

	// Add some valid normal transactions with tips above the minimum
	const tippedBatches = 5
	for range tippedBatches {
		txs := createTransactions(t, &types.DynamicFeeTx{
			GasTipCap: big.NewInt(int64(poolConfig.MinimumTip)), // valid tip
			GasFeeCap: big.NewInt(100),
			Gas:       21_000,
			To:        &common.Address{1}, // not a contract creation
		}, chainId, transactionsPerSender)

		for _, tx := range txs {
			err := pool.addRemoteSync(tx)
			require.NoError(t, err)
		}
	}

	// Add some valid local transactions with tips bellow the minimum
	const localBatches = 5
	for range localBatches {
		txs := createTransactions(t, &types.DynamicFeeTx{
			GasTipCap: big.NewInt(int64(poolConfig.MinimumTip - 1)), // below minimum tip, but valid as local
			GasFeeCap: big.NewInt(100),
			Gas:       21_000,
			To:        &common.Address{1}, // not a contract creation
		}, chainId, transactionsPerSender)

		for _, tx := range txs {
			err := pool.AddLocal(tx)
			require.NoError(t, err)
		}
	}

	pending, err := pool.Pending(true) // with tips enforcement
	require.NoError(t, err)
	require.Len(t, pending,
		sponsoredBatches+tippedBatches+localBatches,
		"expected all valid txs to be included")

	pendingSponsored := make([]*types.Transaction, 0, len(pending))
	pendingNormal := make([]*types.Transaction, 0, len(pending))
	for _, txs := range pending {
		// in this test, one tx per sender
		for _, tx := range txs {
			if tx.GasPrice().Sign() == 0 {
				pendingSponsored = append(pendingSponsored, tx)
			} else {
				pendingNormal = append(pendingNormal, tx)
			}
		}
	}
	require.Len(t, pendingSponsored,
		sponsoredBatches*transactionsPerSender,
		"expected all sponsored txs to be found")
	require.Len(t, pendingNormal,
		tippedBatches*transactionsPerSender+
			localBatches*transactionsPerSender,
		"expected all tipped txs to be found")
}

////////////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////////////

// mockChain creates a mock chain with basic expectations which allow to accept
// any transaction in the pool.
func mockChain(ctrl *gomock.Controller, chainConfig *params.ChainConfig, upgrades opera.Upgrades) *MockStateReader {
	state := state.NewMockStateDB(ctrl)
	state.EXPECT().GetNonce(gomock.Any()).Return(uint64(0)).AnyTimes()
	state.EXPECT().GetBalance(gomock.Any()).Return(uint256.NewInt(1e18)).AnyTimes()
	state.EXPECT().GetCodeHash(gomock.Any()).Return(types.EmptyCodeHash).AnyTimes()

	chain := NewMockStateReader(ctrl)
	chain.EXPECT().CurrentBlock().Return(&EvmBlock{
		EvmHeader: EvmHeader{
			Number: big.NewInt(1),
		},
	}).AnyTimes()
	chain.EXPECT().Config().Return(chainConfig).AnyTimes()
	chain.EXPECT().GetTxPoolStateDB().Return(state, nil).AnyTimes()
	chain.EXPECT().MaxGasLimit().Return(uint64(30_000_000)).AnyTimes()
	chain.EXPECT().GetCurrentBaseFee().Return(big.NewInt(1)).AnyTimes()

	sub := NewMocksubscriber(ctrl)
	sub.EXPECT().Err().Return(make(chan error)).AnyTimes()
	sub.EXPECT().Unsubscribe().AnyTimes()

	chain.EXPECT().SubscribeNewBlock(gomock.Any()).Return(sub).AnyTimes()
	chain.EXPECT().GetCurrentRules().
		Return(opera.Rules{Upgrades: upgrades}).AnyTimes()
	return chain
}

// singTx creates and signs a transaction with a new key for each call.
func createTransactions(t *testing.T, txData types.TxData, chainId *big.Int, n int) []*types.Transaction {
	t.Helper()
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	res := make([]*types.Transaction, n)

	for i := range n {
		SetNonce(txData, uint64(i))

		signer := types.LatestSignerForChainID(chainId)
		tx := types.MustSignNewTx(key, signer, txData)
		res[i] = tx
	}
	return res
}

//go:generate mockgen -source=tx_pool_subsidies_test.go -destination=tx_pool_subsidies_test_mock.go -package=evmcore

func SetNonce(tx types.TxData, nonce uint64) {
	switch d := (tx).(type) {
	case *types.LegacyTx:
		d.Nonce = nonce
	case *types.AccessListTx:
		d.Nonce = nonce
	case *types.DynamicFeeTx:
		d.Nonce = nonce
	case *types.BlobTx:
		d.Nonce = nonce
	case *types.SetCodeTx:
		d.Nonce = nonce
	default:
		panic("unknown tx type")
	}
}

// subscriber is a wrapper around event.Subscription to allow mocking it.
type subscriber interface {
	event.Subscription
}

// suppress unused warning
var _ subscriber
