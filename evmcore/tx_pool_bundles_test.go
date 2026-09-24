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

package evmcore

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"sync/atomic"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/utils"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestTxPool_TransactionsAreQueuedAccordingToTheirExecutionStatus(t *testing.T) {

	chainId := big.NewInt(123)
	blockNumber := idx.Block(1)

	rules := opera.Rules{
		NetworkID: chainId.Uint64(),
		Economy: opera.EconomyRules{
			Gas: opera.GasRules{
				MaxEventGas: 30_000_000,
			},
		},
		Upgrades: opera.Upgrades{
			Allegro:            true,
			Brio:               true,
			TransactionBundles: true,
		},
	}
	signer := types.LatestSignerForChainID(chainId)

	chainConfig := opera.CreateTransientEvmChainConfig(
		chainId.Uint64(),
		[]opera.UpgradeHeight{{Upgrades: rules.Upgrades, Height: 0}},
		blockNumber,
	)

	poolConfig := DefaultTxPoolConfig
	poolConfig.Journal = ""
	poolConfig.DisableTxPoolValidation = true

	tests := map[string]struct {
		accountCurrentNonce uint64
		nonceOffset         uint64
		expectedPending     int
		expectedQueued      int
	}{
		"permanently blocked are dropped": {
			// all generated transactions are in the past
			accountCurrentNonce: 1,
			expectedPending:     0,
			expectedQueued:      0,
		},
		"temporarily blocked remain queued": {
			// all generated transactions are in the future
			nonceOffset:     1,
			expectedPending: 0,
			expectedQueued:  1,
		},
		"executable are pending": {
			expectedPending: 1,
			expectedQueued:  0,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			require := require.New(t)
			ctrl := gomock.NewController(t)

			bundleeKey, err := crypto.GenerateKey()
			require.NoError(err)
			bundleeAdress := crypto.PubkeyToAddress(bundleeKey.PublicKey)

			any := gomock.Any()

			// Mock the state to accept any transaction.
			stateDb := state.NewMockStateDB(ctrl)
			mockStateDbExecutionUsageForAccountWithNonce(stateDb, bundleeAdress, test.accountCurrentNonce)

			chain := NewMockStateReader(ctrl)
			chain.EXPECT().CurrentBlock().Return(&EvmBlock{
				EvmHeader: EvmHeader{Number: big.NewInt(int64(blockNumber))},
			}).AnyTimes()
			chain.EXPECT().CurrentConfig().Return(chainConfig).AnyTimes()
			chain.EXPECT().CurrentStateDB().Return(stateDb, nil).AnyTimes()
			chain.EXPECT().CurrentMaxGasLimit().Return(rules.Economy.Gas.MaxEventGas).AnyTimes()
			chain.EXPECT().CurrentBaseFee().Return(big.NewInt(1)).AnyTimes()
			chain.EXPECT().CurrentRules().Return(rules).AnyTimes()

			subscriber := NewMocksubscriber(ctrl)
			subscriber.EXPECT().Err().Return(make(chan error)).AnyTimes()
			subscriber.EXPECT().Unsubscribe().AnyTimes()
			chain.EXPECT().SubscribeNewBlock(any).Return(subscriber).AnyTimes()

			subsidiesCheckFactory := func(opera.Rules, StateReader, state.StateDB, types.Signer) utils.TransactionCheckFunc {
				return nil
			}

			bundleEvaluationCache := NewBundleEvaluationCache()

			pool := newTxPool(poolConfig, chainConfig, chain, subsidiesCheckFactory, bundleEvaluationCache)

			pending, queued := pool.Content()
			expectContents(t, pending, queued, 0, 0)

			tx := bundle.NewBuilder().
				WithSigner(signer).
				SetEnvelopeNonce(0).
				With(bundle.Step(bundleeKey,
					&types.AccessListTx{
						// offsetting the nonce creates gapped transactions
						// which cannot be immediately executed
						Nonce: uint64(0 + test.nonceOffset),
						Gas:   60_000,
					})).
				Build()

			// adding transactions triggers and waits for reorg.
			// promotions/drops are synchronous.
			err = pool.AddLocal(tx)
			require.NoError(err, "failed to add bundle transaction to the pool")

			pending, queued = pool.Content()
			expectContents(t, pending, queued, test.expectedPending, test.expectedQueued)
		})
	}
}

func expectContents(t *testing.T,
	pending, queued map[common.Address]types.Transactions,
	expectedPending, expectedQueued int,
) {
	t.Helper()

	if expectedPending != 0 {
		require.Len(t, pending, expectedPending, "expected pending transactions for each account")
	} else {
		require.Empty(t, pending, "expected no pending transactions")
	}

	if expectedQueued != 0 {
		require.Len(t, queued, expectedQueued, "expected queued transactions for each account")
	} else {
		require.Empty(t, queued, "expected no queued transactions")
	}
}

func mockStateDbExecutionUsageForAccountWithNonce(
	stateDb *state.MockStateDB,
	sender common.Address,
	nonce uint64,
) {
	any := gomock.Any()

	// When asked for the nonce of the specific sender, return the provided nonce.
	// For any other address, return 0.
	stateDb.EXPECT().GetNonce(sender).Return(nonce).AnyTimes()
	stateDb.EXPECT().GetNonce(any).Return(uint64(0)).AnyTimes()

	stateDb.EXPECT().GetBalance(any).Return(uint256.NewInt(1e18)).AnyTimes()
	stateDb.EXPECT().GetCodeHash(any).Return(types.EmptyCodeHash).AnyTimes()
	stateDb.EXPECT().GetCode(any).Return([]byte{}).AnyTimes()
	stateDb.EXPECT().HasBundleRecentlyBeenProcessed(any).Return(false).AnyTimes()
	stateDb.EXPECT().InterTxSnapshot().AnyTimes()
	stateDb.EXPECT().RevertToInterTxSnapshot(any).AnyTimes()
	stateDb.EXPECT().SetTxContext(any, any).AnyTimes()
	stateDb.EXPECT().EndTransaction().AnyTimes()
	stateDb.EXPECT().AddProcessedBundle(any, any).AnyTimes()
	stateDb.EXPECT().AddAddressToAccessList(any).AnyTimes()
	stateDb.EXPECT().Snapshot().AnyTimes()
	stateDb.EXPECT().RevertToSnapshot(any).AnyTimes()
	stateDb.EXPECT().Exist(any).AnyTimes()
	stateDb.EXPECT().Finalise(any).AnyTimes()
	stateDb.EXPECT().SubBalance(any, any, any).AnyTimes()
	stateDb.EXPECT().Prepare(any, any, any, any, any, any).AnyTimes()
	stateDb.EXPECT().SetNonce(any, any, any).AnyTimes()
	stateDb.EXPECT().GetStorageRoot(any).AnyTimes()
	stateDb.EXPECT().CreateAccount(any).AnyTimes()
	stateDb.EXPECT().CreateContract(any).AnyTimes()
	stateDb.EXPECT().AddBalance(any, any, any).AnyTimes()
	stateDb.EXPECT().AddRefund(any).AnyTimes()
	stateDb.EXPECT().GetRefund().AnyTimes()
	stateDb.EXPECT().SubRefund(any).AnyTimes()
	stateDb.EXPECT().GetLogs(any, any).AnyTimes()
	stateDb.EXPECT().TxIndex().AnyTimes()
}

// ========================== Tools ===========================

// bundleTx creates a transaction that is part of a bundle with the given key and nonce.
// the name follows the convention of other transaction creation tools in the tx pool tests.
func bundleTx(nonce uint64, key *ecdsa.PrivateKey) *types.Transaction {
	signer := types.LatestSignerForChainID(params.TestChainConfig.ChainID)
	return bundle.NewBuilder().
		WithSigner(signer).
		SetEnvelopeSenderKey(key).
		SetEnvelopeNonce(nonce).
		With(bundle.Step(key, &types.AccessListTx{Nonce: nonce})).
		Build()
}

func TestTxPool_EvictTransactionsOfEvaluatedBundles_DropsBundleOnlyTransactionsOfEvaluatedBundles(t *testing.T) {
	for name, withEnvelope := range map[string]bool{
		"with envelope":    true,
		"without envelope": false,
	} {
		t.Run(name, func(t *testing.T) {
			chainId := big.NewInt(123)
			blockNumber := idx.Block(1)

			rules := opera.Rules{
				NetworkID: chainId.Uint64(),
				Economy: opera.EconomyRules{
					Gas: opera.GasRules{
						MaxEventGas: 30_000_000,
					},
				},
				Upgrades: opera.Upgrades{
					Allegro:            true,
					Brio:               true,
					TransactionBundles: true,
				},
			}
			signer := types.LatestSignerForChainID(chainId)

			chainConfig := opera.CreateTransientEvmChainConfig(
				chainId.Uint64(),
				[]opera.UpgradeHeight{{Upgrades: rules.Upgrades, Height: 0}},
				blockNumber,
			)

			poolConfig := DefaultTxPoolConfig
			poolConfig.Journal = ""
			poolConfig.DisableTxPoolValidation = true

			require := require.New(t)
			ctrl := gomock.NewController(t)

			bundleeKey, err := crypto.GenerateKey()
			require.NoError(err)
			bundleeAddress := crypto.PubkeyToAddress(bundleeKey.PublicKey)

			envelope, txBundle, plan := bundle.NewBuilder().
				WithSigner(signer).
				SetEnvelopeNonce(0).
				AllOf(
					bundle.Step(bundleeKey, &types.AccessListTx{Nonce: 0, Gas: 60_000}),
				).
				BuildEnvelopeBundleAndPlan()

			// Registered before the catch-all expectations installed below.
			evaluated := atomic.Bool{}
			stateDb := state.NewMockStateDB(ctrl)
			stateDb.EXPECT().HasBundleRecentlyBeenProcessed(gomock.Any()).
				DoAndReturn(func(execPlanHash common.Hash) bool {
					return evaluated.Load() && execPlanHash == plan.Hash()
				}).AnyTimes()
			mockStateDbExecutionUsageForAccountWithNonce(stateDb, bundleeAddress, 0)
			stateDb.EXPECT().Release().AnyTimes() // < released on every new head

			chain := NewMockStateReader(ctrl)
			chain.EXPECT().CurrentBlock().DoAndReturn(func() *EvmBlock {
				// The evaluating block advances the head, making the pool
				// re-evaluate the bundles it knows.
				head := blockNumber
				if evaluated.Load() {
					head++
				}
				return &EvmBlock{EvmHeader: EvmHeader{Number: big.NewInt(int64(head))}}
			}).AnyTimes()
			chain.EXPECT().CurrentConfig().Return(chainConfig).AnyTimes()
			chain.EXPECT().CurrentStateDB().Return(stateDb, nil).AnyTimes()
			chain.EXPECT().CurrentMaxGasLimit().Return(rules.Economy.Gas.MaxEventGas).AnyTimes()
			chain.EXPECT().CurrentBaseFee().Return(big.NewInt(1)).AnyTimes()
			chain.EXPECT().CurrentRules().Return(rules).AnyTimes()

			subscriber := NewMocksubscriber(ctrl)
			subscriber.EXPECT().Err().Return(make(chan error)).AnyTimes()
			subscriber.EXPECT().Unsubscribe().AnyTimes()
			chain.EXPECT().SubscribeNewBlock(gomock.Any()).Return(subscriber).AnyTimes()

			subsidiesCheckFactory := func(opera.Rules, StateReader, state.StateDB, types.Signer) utils.TransactionCheckFunc {
				return nil
			}

			pool := newTxPool(poolConfig, chainConfig, chain, subsidiesCheckFactory,
				NewBundleEvaluationCache())

			// The pool knows the transactions of the bundle, which occupy the nonces
			// of their sender, but not necessarily its envelope, e.g. when a wallet
			// signed them before they got packed into one.
			envelopes := 0
			if withEnvelope {
				envelopes = 1
			}
			bundledTxs := txBundle.GetTransactionsInReferencedOrder()
			require.Len(bundledTxs, 1)
			if withEnvelope {
				require.NoError(pool.AddLocal(envelope))
			}
			for _, tx := range bundledTxs {
				require.True(bundle.IsBundleOnly(tx))
				require.NoError(pool.AddLocal(tx))
			}
			require.Equal(len(bundledTxs)+envelopes, pool.Count())
			require.Equal(uint64(1), pool.Nonce(bundleeAddress))

			// A new head not evaluating the bundle retains all of them.
			<-pool.requestReset(nil, nil)
			require.Equal(len(bundledTxs)+envelopes, pool.Count())
			require.Equal(uint64(1), pool.Nonce(bundleeAddress))

			// Once the bundle got evaluated, its transactions are of no use anymore.
			evaluated.Store(true)
			<-pool.requestReset(nil, nil)

			for _, tx := range bundledTxs {
				require.Nil(pool.Get(tx.Hash()),
					"bundle-only transaction of an evaluated bundle must be evicted")
			}
			require.Nil(pool.Get(envelope.Hash()),
				"the envelope of an evaluated bundle must be dropped as well")
			require.Zero(pool.Count())
			require.Equal(uint64(0), pool.Nonce(bundleeAddress),
				"the evicted transaction must not block the nonce of its sender")
			require.NoError(validateTxPoolInternals(pool))
		})
	}
}

func TestTxPool_EvictStaleBundleOnlyTransactions_DropsBundleOnlyTransactionsWaitingTooLong(t *testing.T) {
	tests := map[string]struct {
		brio     bool
		lifetime time.Duration
		evicted  bool
	}{
		"stale":       {brio: true, lifetime: time.Nanosecond, evicted: true},
		"fresh":       {brio: true, lifetime: time.Hour},
		"before brio": {lifetime: time.Nanosecond},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			pool, remoteKey := setupTxPool()
			defer pool.Stop()
			pool.waitForIdleReorgLoop_forTesting()

			pool.mu.Lock()
			defer pool.mu.Unlock()
			if test.brio {
				pool.chain = brioTestBlockChain{pool.chain}
			}
			pool.config.BundleOnlyLifetime = test.lifetime

			// Local transactions are not exempt, they can not run on their own either.
			localKey, err := crypto.GenerateKey()
			require.NoError(err)
			pool.locals.add(crypto.PubkeyToAddress(localKey.PublicKey))

			add := func(key *ecdsa.PrivateKey, nonce uint64, accessList types.AccessList) *types.Transaction {
				tx := types.MustSignNewTx(key, pool.signer, &types.AccessListTx{
					Nonce:      nonce,
					Gas:        100_000,
					GasPrice:   big.NewInt(1),
					AccessList: accessList,
				})
				pool.all.Add(tx, false)
				pool.promoteTx(crypto.PubkeyToAddress(key.PublicKey), tx.Hash(), tx)
				return tx
			}
			mark := types.AccessList{{Address: bundle.BundleOnly, StorageKeys: []common.Hash{{0x1}}}}
			bundleOnly := []*types.Transaction{add(remoteKey, 0, mark), add(localKey, 0, mark)}
			regular := add(remoteKey, 1, nil)

			time.Sleep(time.Millisecond) // < let the stale case expire
			pool.evictStaleBundleOnlyTransactions()

			for _, tx := range bundleOnly {
				if test.evicted {
					require.Nil(pool.all.Get(tx.Hash()), "stale bundle-only transaction must be evicted")
				} else {
					require.NotNil(pool.all.Get(tx.Hash()), "bundle-only transaction must be retained")
				}
			}
			require.NotNil(pool.all.Get(regular.Hash()), "regular transactions must be retained")
		})
	}
}

// BenchmarkTxPool_EvictTransactionsOfEvaluatedBundles measures the scan run on
// every new head, excluding the cost of looking up processed bundles, which is
// reported as lookups/op. No bundle has been processed, so nothing is evicted.
func BenchmarkTxPool_EvictTransactionsOfEvaluatedBundles(b *testing.B) {
	for _, size := range []int{1_000, 10_000} {
		for _, percentBundleOnly := range []int{0, 10, 100} {
			name := fmt.Sprintf("txs=%d/bundleOnly=%d%%", size, percentBundleOnly)
			b.Run(name, func(b *testing.B) {
				benchmarkEvictTransactionsOfEvaluatedBundles(b, size, percentBundleOnly)
			})
		}
	}
}

func benchmarkEvictTransactionsOfEvaluatedBundles(b *testing.B, size, percentBundleOnly int) {
	pool, key := setupTxPool()
	defer pool.Stop()
	pool.waitForIdleReorgLoop_forTesting()

	pool.mu.Lock()
	defer pool.mu.Unlock()

	stateDb := &countingBundleStateDb{testTxPoolStateDb: newTestTxPoolStateDb()}
	pool.chain = brioTestBlockChain{pool.chain}
	pool.currentState = stateDb

	account := crypto.PubkeyToAddress(key.PublicKey)
	for i := range size {
		var accessList types.AccessList
		if i*100 < size*percentBundleOnly {
			accessList = types.AccessList{{
				Address:     bundle.BundleOnly,
				StorageKeys: []common.Hash{{byte(i), byte(i >> 8)}},
			}}
		}
		tx := types.MustSignNewTx(key, pool.signer, &types.AccessListTx{
			Nonce:      uint64(i),
			Gas:        100_000,
			GasPrice:   big.NewInt(1),
			AccessList: accessList,
		})
		pool.all.Add(tx, false)
		pool.promoteTx(account, tx.Hash(), tx)
	}

	b.ResetTimer()
	for range b.N {
		pool.evictTransactionsOfEvaluatedBundles()
	}
	b.ReportMetric(float64(stateDb.lookups)/float64(b.N), "lookups/op")
	require.Equal(b, size, pool.all.Count())
}

type brioTestBlockChain struct {
	StateReader
}

func (brioTestBlockChain) CurrentRules() opera.Rules {
	return opera.Rules{Upgrades: opera.Upgrades{Brio: true}}
}

type countingBundleStateDb struct {
	*testTxPoolStateDb
	lookups int
}

func (s *countingBundleStateDb) HasBundleRecentlyBeenProcessed(common.Hash) bool {
	s.lookups++
	return false
}
