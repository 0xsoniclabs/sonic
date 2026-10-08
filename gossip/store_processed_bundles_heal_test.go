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
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/iblockproc"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
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

func TestBlockScan_RefusesNestedBundles(t *testing.T) {
	const block = 2000
	signer := types.LatestSignerForChainID(big.NewInt(1))
	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	// also when the enclosing plan starts before the window
	for _, first := range []uint64{block - 10, block - 1500} {
		inner := bundle.NewBuilder().
			WithSigner(signer).
			SetEarliest(block - 10).
			AllOf(bundle.Step(key, &types.AccessListTx{To: &common.Address{1}, Gas: 21_000})).
			Build()
		outer := bundle.NewBuilder().
			WithSigner(signer).
			SetEarliest(first).
			AllOf(bundle.Step(key, inner)).
			Build()

		scan := blockScan{
			epoch:  7,
			number: block,
			first:  block - 1023,
			rules: &opera.Rules{Upgrades: opera.Upgrades{
				Allegro: true, Brio: true, TransactionBundles: true,
			}},
			signer: signer,
			marked: map[common.Hash]uint64{},
		}
		_, err = scan.scan([]*types.Transaction{outer})
		require.ErrorContains(t, err, "epoch 7 contains nested bundles")
		require.ErrorContains(t, err, "heal to epoch 7")
	}
}

func TestStore_RestoreProcessedBundles_VerifiesEpochHistoryHash(t *testing.T) {
	for name, corrupt := range map[string]bool{"reproducible": false, "diverged": true} {
		t.Run(name, func(t *testing.T) {
			store, err := NewMemStore(t)
			require.NoError(t, err)

			// epochs 1 and 2 hold blocks up to 2000 and 2001-2010, without
			// bundles
			for n := uint64(1); n <= 2010; n++ {
				epoch := idx.Epoch(1)
				if n > 2000 {
					epoch = 2
				}
				store.SetBlock(idx.Block(n), inter.NewBlockBuilder().
					WithNumber(n).WithEpoch(epoch).Build())
			}
			previous := common.Hash{1}
			anchor := previous
			for block := uint64(2000); block < 2010; block++ {
				anchor = nextBundleHistoryHash(anchor, nil, block)
			}
			if corrupt {
				anchor = common.Hash{2}
			}
			for epoch, state := range map[idx.Epoch]struct {
				last idx.Block
				hash common.Hash
			}{1: {0, common.Hash{}}, 2: {2000, previous}, 3: {2010, anchor}} {
				store.SetHistoryBlockEpochState(epoch,
					iblockproc.BlockState{LastBlock: iblockproc.BlockCtx{Idx: state.last}},
					iblockproc.EpochState{
						Epoch:                          epoch,
						Validators:                     pos.NewBuilder().Build(),
						EpochEndExecutionPlanChainHash: hash.Hash(state.hash),
					})
			}

			// a stale table left by the head
			store.AddProcessedBundles(2025, map[common.Hash]bundle.PositionInBlock{{3}: {}})
			_, staleHash := store.GetLatestProcessedBundleHistoryHash()

			// the window of epoch 1's seal reaches back to the genesis
			require.ErrorContains(t, store.RestoreProcessedBundles(2), "re-sync")

			err = store.RestoreProcessedBundles(3)
			_, latest := store.GetLatestProcessedBundleHistoryHash()
			if corrupt {
				require.ErrorContains(t, err, "diverged")
				require.Equal(t, staleHash, latest)
				return
			}
			require.NoError(t, err)
			require.Empty(t, store.EnumerateProcessedBundles())
			require.Equal(t, nextBundleHistoryHash(anchor, nil, 2010), latest)
		})
	}
}
