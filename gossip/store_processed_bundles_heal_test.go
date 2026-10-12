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
	"errors"
	"math/big"
	"slices"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/iblockproc"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/utils"
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
	misfueled, _, _ := makeBundle(block - 30)
	misfueled = types.MustSignNewTx(key, signer, &types.AccessListTx{
		To: misfueled.To(), Gas: misfueled.Gas() + 1, Data: misfueled.Data(),
	})

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
		executed, reverted, outOfRange, beforeWindow, processed, reverted, misfueled,
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
	tests := map[string]struct {
		corrupt, retained bool
		err               string
	}{
		"reproducible":           {},
		"diverged":               {corrupt: true, err: "diverged"},
		"retained hash diverges": {retained: true, err: "store holds"},
	}
	for name, test := range tests {
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
			if test.corrupt {
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
			if test.retained {
				store.AddProcessedBundles(2010, map[common.Hash]bundle.PositionInBlock{{4}: {}})
			}
			store.AddProcessedBundles(2025, map[common.Hash]bundle.PositionInBlock{{3}: {}})
			_, staleHash := store.GetLatestProcessedBundleHistoryHash()

			require.ErrorContains(t, store.RestoreProcessedBundles(4), "not available")
			// the window of epoch 1's seal reaches back to the genesis
			require.ErrorContains(t, store.RestoreProcessedBundles(2), "re-sync")

			err = store.RestoreProcessedBundles(3)
			_, latest := store.GetLatestProcessedBundleHistoryHash()
			if test.err != "" {
				require.ErrorContains(t, err, test.err)
				require.Equal(t, staleHash, latest)
				return
			}
			require.NoError(t, err)
			require.Empty(t, store.EnumerateProcessedBundles())
			require.Equal(t, nextBundleHistoryHash(anchor, nil, 2010), latest)
		})
	}
}

func TestStore_RestoreProcessedBundles_ClearsTableWithoutBundleHistory(t *testing.T) {
	store, err := NewMemStore(t)
	require.NoError(t, err)
	store.SetHistoryBlockEpochState(1,
		iblockproc.BlockState{LastBlock: iblockproc.BlockCtx{Idx: 2000}},
		iblockproc.EpochState{Epoch: 1, Validators: pos.NewBuilder().Build()})
	store.AddProcessedBundles(2025, map[common.Hash]bundle.PositionInBlock{{3}: {}})

	require.NoError(t, store.RestoreProcessedBundles(1))
	require.Empty(t, store.EnumerateProcessedBundles())
	_, found := store.GetProcessedBundleHistoryHash(2025)
	require.False(t, found)
}

func TestStore_ReplayEpochBlocks_ReproducesStoredBlocks(t *testing.T) {
	store, first, last := newBundleReplayStore(t)

	// The replay yields the inputs of each block, which repeat transactions
	// already executed; the user transactions of the block are among them.
	for _, window := range [][2]uint64{{first, last}, {first + 2, last - 1}} {
		var visited []uint64
		executed := 0
		err := store.replayEpochBlocks(2, window[0], window[1], func(number uint64, time inter.Timestamp, _ *opera.Rules, _ types.Signer, txs types.Transactions) error {
			visited = append(visited, number)
			block := store.GetBlock(idx.Block(number))
			require.Equal(t, block.Time, time)
			for _, tx := range store.GetBlockTxs(idx.Block(number), block) {
				if r, _, _ := tx.RawSignatureValues(); r.Sign() == 0 {
					continue // < internal transaction
				}
				require.True(t, slices.ContainsFunc(txs, func(input *types.Transaction) bool {
					return input.Hash() == tx.Hash()
				}), "block %d misses %x", number, tx.Hash())
				executed++
			}
			return nil
		})
		require.NoError(t, err)
		var expected []uint64
		for n := window[0]; n <= window[1]; n++ {
			expected = append(expected, n)
		}
		require.Equal(t, expected, visited)
		require.NotZero(t, executed)
	}

	marks, err := store.collectProcessedBundles(first, last)
	require.NoError(t, err)
	require.Empty(t, marks)
}

func TestStore_ReplayEpochBlocks_RefusesStoresItCannotReproduce(t *testing.T) {
	ignore := func(uint64, inter.Timestamp, *opera.Rules, types.Signer, types.Transactions) error { return nil }
	tests := map[string]struct {
		tamper func(*Store, uint64)
		err    string
	}{
		"block time differs": {
			tamper: func(store *Store, last uint64) {
				block := *store.GetBlock(idx.Block(last))
				block.Time++
				store.SetBlock(idx.Block(last), &block)
			},
			err: "consensus replay yields block",
		},
		"single proposer": {
			tamper: func(store *Store, last uint64) {
				bs, es := store.GetHistoryBlockEpochState(2)
				es.Rules.Upgrades.SingleProposerBlockFormation = true
				store.SetHistoryBlockEpochState(2, *bs, *es)
			},
			err: "single-proposer",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			store, first, last := newBundleReplayStore(t)
			test.tamper(store, last)
			_, err := store.collectProcessedBundles(first, last)
			require.ErrorContains(t, err, test.err)
		})
	}

	t.Run("epoch seals later", func(t *testing.T) {
		store, first, last := newBundleReplayStore(t)
		bs, es := store.GetHistoryBlockEpochState(3)
		bs.LastBlock.Idx += 5
		store.SetHistoryBlockEpochState(3, *bs, *es)
		err := store.replayEpochBlocks(2, first, last+5, ignore)
		require.ErrorContains(t, err, "consensus replay of epoch 2 ends at block")
	})

	t.Run("visit fails", func(t *testing.T) {
		store, first, last := newBundleReplayStore(t)
		failure := errors.New("visit failed")
		err := store.replayEpochBlocks(2, first, last, func(uint64, inter.Timestamp, *opera.Rules, types.Signer, types.Transactions) error {
			return failure
		})
		require.ErrorIs(t, err, failure)
	})

	t.Run("next epoch missing", func(t *testing.T) {
		store, first, last := newBundleReplayStore(t)
		require.ErrorContains(t, store.replayEpochBlocks(3, first, last, ignore), "epoch 3 is not available")
		_, err := store.collectProcessedBundles(first, last+100)
		require.ErrorContains(t, err, "not available")
	})

	t.Run("event missing", func(t *testing.T) {
		store, _, _ := newBundleReplayStore(t)
		require.Nil(t, eventSource{store}.GetEvent(hash.Event{}))
	})
}

// newBundleReplayStore returns the store of a chain with transaction bundles
// whose epoch 2 holds the blocks [first, last] and is sealed.
func newBundleReplayStore(t *testing.T) (store *Store, first, last uint64) {
	upgrades := opera.GetBrioUpgrades()
	upgrades.TransactionBundles = true
	upgrades.SingleProposerBlockFormation = false
	env := newTestEnvWithUpgrades(2, 3, upgrades, t)
	t.Cleanup(func() { require.NoError(t, env.Close()) })

	for _, spent := range []time.Duration{sameEpoch, sameEpoch, nextEpoch, sameEpoch} {
		_, err := env.ApplyTxs(spent, env.Transfer(1, 2, utils.ToFtm(1)))
		require.NoError(t, err)
	}
	start, _ := env.store.GetHistoryBlockEpochState(2)
	end, _ := env.store.GetHistoryBlockEpochState(3)
	require.NotNil(t, end)
	return env.store, uint64(start.LastBlock.Idx) + 1, uint64(end.LastBlock.Idx)
}
