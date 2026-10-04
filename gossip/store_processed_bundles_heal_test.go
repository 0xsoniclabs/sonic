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
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestBlockScan_MarksBundlesPassingPreExecutionChecks(t *testing.T) {
	const block = 2000
	signer := types.LatestSignerForChainID(big.NewInt(1))
	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	makeBundle := func(first uint64) (*types.Transaction, []*types.Transaction, common.Hash) {
		envelope, txBundle, plan := bundle.NewBuilder().
			WithSigner(signer).
			SetEarliest(first).
			AllOf(bundle.Step(key, &types.AccessListTx{To: &common.Address{1}, Gas: 21_000})).
			BuildEnvelopeBundleAndPlan()
		return envelope, txBundle.GetTransactionsInReferencedOrder(), plan.Hash()
	}

	executed, executedTxs, executedPlan := makeBundle(block - 10)
	reverted, _, revertedPlan := makeBundle(block - 5)
	outOfRange, _, _ := makeBundle(block + 1)
	beforeWindow, beforeWindowTxs, _ := makeBundle(block - 1500)
	processed, _, processedPlan := makeBundle(block - 20)

	scan := blockScan{
		number: block,
		first:  block - 1023,
		last:   block,
		rules: &opera.Rules{Upgrades: opera.Upgrades{
			Allegro: true, Brio: true, TransactionBundles: true,
		}},
		signer:  signer,
		marked:  map[common.Hash]uint64{processedPlan: block - 1},
		blockTx: append(executedTxs, beforeWindowTxs...),
	}

	marks, err := scan.scan([]*types.Transaction{
		executed, reverted, outOfRange, beforeWindow, processed, reverted,
	})
	require.NoError(t, err)
	require.Equal(t, map[common.Hash]bundle.PositionInBlock{
		executedPlan: {Offset: 0, Count: 1},
		revertedPlan: {},
	}, marks)
}

func TestBlockScan_RejectsTransactionsOfUnmarkedPlans(t *testing.T) {
	const block = 2000
	signer := types.LatestSignerForChainID(big.NewInt(1))
	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	_, txBundle, _ := bundle.NewBuilder().
		WithSigner(signer).
		SetEarliest(block - 10).
		AllOf(bundle.Step(key, &types.AccessListTx{To: &common.Address{1}, Gas: 21_000})).
		BuildEnvelopeBundleAndPlan()

	scan := blockScan{
		number: block,
		first:  block - 1023,
		last:   block,
		rules: &opera.Rules{Upgrades: opera.Upgrades{
			Allegro: true, Brio: true, TransactionBundles: true,
		}},
		signer:  signer,
		marked:  map[common.Hash]uint64{},
		blockTx: txBundle.GetTransactionsInReferencedOrder(),
	}

	// The proposal lacks the envelope delivering the block's transactions.
	_, err = scan.scan(nil)
	require.ErrorContains(t, err, "not marked as processed")
}

func TestBlockScan_RefusesUndeterminedNestedPlans(t *testing.T) {
	const block = 2000
	signer := types.LatestSignerForChainID(big.NewInt(1))
	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	inner := bundle.NewBuilder().
		WithSigner(signer).
		SetEarliest(block - 10).
		AllOf(bundle.Step(key, &types.AccessListTx{To: &common.Address{1}, Gas: 21_000})).
		Build()
	outer := bundle.NewBuilder().
		WithSigner(signer).
		SetEarliest(block - 10).
		AllOf(bundle.Step(key, inner)).
		Build()

	scan := blockScan{
		number: block,
		first:  block - 1023,
		last:   block,
		rules: &opera.Rules{Upgrades: opera.Upgrades{
			Allegro: true, Brio: true, TransactionBundles: true,
		}},
		signer: signer,
		marked: map[common.Hash]uint64{},
	}

	// Without transactions in the block, the inner plan may have failed or
	// been rolled back, and it stays in range after the window.
	_, err = scan.scan([]*types.Transaction{outer})
	require.ErrorContains(t, err, "cannot be determined")
}

func TestStore_RestoreProcessedBundles_LeavesStoreUntouchedOnFailure(t *testing.T) {
	store, err := NewMemStore(t)
	require.NoError(t, err)

	store.AddProcessedBundles(10, map[common.Hash]bundle.PositionInBlock{{1}: {}})
	store.AddProcessedBundles(11, map[common.Hash]bundle.PositionInBlock{{2}: {}})
	before := store.EnumerateProcessedBundles()
	_, beforeHash := store.GetLatestProcessedBundleHistoryHash()

	// block 10's retained history hash contradicts the anchor
	err = store.RestoreProcessedBundles(11, common.Hash{0x42})
	require.ErrorContains(t, err, "epoch records")

	require.ElementsMatch(t, before, store.EnumerateProcessedBundles())
	_, afterHash := store.GetLatestProcessedBundleHistoryHash()
	require.Equal(t, beforeHash, afterHash)
}
