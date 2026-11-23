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

	"github.com/0xsoniclabs/sonic/gossip/blockproc/freeze"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestTxPool_AccountFreeze_RejectsTransactionsOfFrozenSenders(t *testing.T) {
	checker := freeze.NewMockChecker(gomock.NewController(t))
	pool, key := setupAccountFreezeTxPool(checker)
	defer pool.Stop()
	other, _ := crypto.GenerateKey()
	frozen := crypto.PubkeyToAddress(key.PublicKey)
	checker.EXPECT().IsFrozen(gomock.Any()).DoAndReturn(func(addr common.Address) bool {
		return addr == frozen
	}).AnyTimes()

	for _, k := range []*ecdsa.PrivateKey{key, other} {
		testAddBalance(pool, crypto.PubkeyToAddress(k.PublicKey), big.NewInt(1_000_000))
	}

	require.ErrorIs(t, pool.AddLocal(transaction(0, 100_000, key)), ErrFrozenAccount)
	require.NoError(t, pool.AddLocal(transaction(0, 100_000, other)))
}

func TestTxPool_AccountFreeze_DropsTransactionsOfSendersFrozenWhileInThePool(t *testing.T) {
	checker := freeze.NewMockChecker(gomock.NewController(t))
	pool, key := setupAccountFreezeTxPool(checker)
	defer pool.Stop()
	other, _ := crypto.GenerateKey()
	frozen, isFrozen := crypto.PubkeyToAddress(key.PublicKey), atomic.Bool{}
	checker.EXPECT().IsFrozen(gomock.Any()).DoAndReturn(func(addr common.Address) bool {
		return addr == frozen && isFrozen.Load()
	}).AnyTimes()

	for _, k := range []*ecdsa.PrivateKey{key, other} {
		testAddBalance(pool, crypto.PubkeyToAddress(k.PublicKey), big.NewInt(1_000_000))
		require.NoError(t, pool.AddLocal(transaction(0, 100_000, k)))
		require.NoError(t, pool.AddLocal(transaction(2, 100_000, k)))
	}
	<-pool.requestReset(nil, nil)
	pending, queued := pool.Stats()
	require.Equal(t, 2, pending)
	require.Equal(t, 2, queued)

	isFrozen.Store(true)
	<-pool.requestReset(nil, nil)

	pending, queued = pool.Stats()
	require.Equal(t, 1, pending)
	require.Equal(t, 1, queued)
	require.Zero(t, pool.Nonce(frozen))
	require.Equal(t, uint64(1), pool.Nonce(crypto.PubkeyToAddress(other.PublicKey)))
	require.NoError(t, validateTxPoolInternals(pool))
}

func TestTxPool_AccountFreeze_FreezingBlockAlsoMiningATransaction_KeepsStateNonce(t *testing.T) {
	checker := freeze.NewMockChecker(gomock.NewController(t))
	pool, key := setupAccountFreezeTxPool(checker)
	defer pool.Stop()
	frozen, isFrozen := crypto.PubkeyToAddress(key.PublicKey), atomic.Bool{}
	checker.EXPECT().IsFrozen(gomock.Any()).DoAndReturn(func(addr common.Address) bool {
		return addr == frozen && isFrozen.Load()
	}).AnyTimes()

	testAddBalance(pool, frozen, big.NewInt(1_000_000))
	require.NoError(t, pool.AddLocal(transaction(0, 100_000, key)))
	require.NoError(t, pool.AddLocal(transaction(1, 100_000, key)))
	<-pool.requestReset(nil, nil)

	testSetNonce(pool, frozen, 1)
	isFrozen.Store(true)
	<-pool.requestReset(nil, nil)

	pending, queued := pool.Stats()
	require.Zero(t, pending)
	require.Zero(t, queued)
	require.Equal(t, uint64(1), pool.Nonce(frozen))
	require.NoError(t, validateTxPoolInternals(pool))
}

func TestTxPool_AccountFreeze_DoesNotAnnounceTransactionsOfFrozenSenders(t *testing.T) {
	checker := freeze.NewMockChecker(gomock.NewController(t))
	pool, key := setupAccountFreezeTxPool(checker)
	defer pool.Stop()
	frozen, isFrozen := crypto.PubkeyToAddress(key.PublicKey), atomic.Bool{}
	checker.EXPECT().IsFrozen(gomock.Any()).DoAndReturn(func(addr common.Address) bool {
		return addr == frozen && isFrozen.Load()
	}).AnyTimes()

	testAddBalance(pool, frozen, big.NewInt(1_000_000))
	require.NoError(t, pool.AddLocal(transaction(1, 100_000, key)))
	<-pool.requestReset(nil, nil)

	events := make(chan NewTxsNotify, 8)
	sub := pool.SubscribeNewTxsNotify(events)
	defer sub.Unsubscribe()

	testSetNonce(pool, frozen, 1) // makes the queued transaction executable
	isFrozen.Store(true)
	<-pool.requestReset(nil, nil)

	pending, queued := pool.Stats()
	require.Zero(t, pending)
	require.Zero(t, queued)
	require.NoError(t, validateEvents(events, 0))
}

func TestTxPool_AccountFreezeDisabled_IgnoresFrozenSenders(t *testing.T) {
	pool, key := setupTxPool()
	defer pool.Stop()
	checker := freeze.NewMockChecker(gomock.NewController(t)) // isFrozen never called
	setFreezeChecker(pool, checker)

	testAddBalance(pool, crypto.PubkeyToAddress(key.PublicKey), big.NewInt(1_000_000))
	require.NoError(t, pool.AddLocal(transaction(0, 100_000, key)))
	<-pool.requestReset(nil, nil)
	pending, _ := pool.Stats()
	require.Equal(t, 1, pending)
}

// setupAccountFreezeTxPool is setupTxPool on a chain with AccountFreeze on,
// with freeze status reported by checker.
func setupAccountFreezeTxPool(checker freeze.Checker) (*TxPool, *ecdsa.PrivateKey) {
	pool, key := setupTxPoolWithChain(params.TestChainConfig, accountFreezeTestBlockChain{NewTestBlockChain(newTestTxPoolStateDb())})
	setFreezeChecker(pool, checker)
	return pool, key
}

func setFreezeChecker(pool *TxPool, checker freeze.Checker) {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	pool.freezeCheckerFactory = func(state.StateDB) freeze.Checker { return checker }
}

type accountFreezeTestBlockChain struct {
	StateReader
}

func (accountFreezeTestBlockChain) CurrentRules() opera.Rules {
	return opera.Rules{Upgrades: opera.Upgrades{AccountFreeze: true}}
}

// BenchmarkTxPool_DropFrozenSenders measures evicting 100 frozen senders, each
// holding the given number of pending transactions.
func BenchmarkTxPool_DropFrozenSenders(b *testing.B) {
	log.SetDefault(log.NewLogger(log.DiscardHandler()))
	for _, perSender := range []int{1, 4, 16, 64} {
		b.Run(fmt.Sprintf("senders=100/txs=%d", perSender), func(b *testing.B) {
			pool, _ := setupAccountFreezeTxPool(allFrozenChecker{})
			defer pool.Stop()
			pool.waitForIdleReorgLoop_forTesting()
			pool.mu.Lock()
			defer pool.mu.Unlock()

			var txs []*types.Transaction
			for range 100 {
				key, _ := crypto.GenerateKey()
				for nonce := range perSender {
					txs = append(txs, transaction(uint64(nonce), 100_000, key))
				}
			}
			b.ResetTimer()
			for range b.N {
				b.StopTimer()
				for _, tx := range txs {
					from, _ := types.Sender(pool.signer, tx)
					pool.all.Add(tx, false)
					pool.promoteTx(from, tx.Hash(), tx)
				}
				b.StartTimer()
				pool.dropFrozenSenders()
			}
			b.StopTimer()
			require.Zero(b, pool.all.Count())
		})
	}
}

type allFrozenChecker struct{}

func (allFrozenChecker) IsFrozen(common.Address) bool   { return true }
func (allFrozenChecker) IsOptedOut(common.Address) bool { return false }
