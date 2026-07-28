// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package emitter

import (
	"crypto/ecdsa"
	"math/big"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/priorities"
	"github.com/0xsoniclabs/sonic/gossip/gasprice"
	"github.com/0xsoniclabs/sonic/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestTransactionPriceNonceSortLegacy(t *testing.T) {
	t.Parallel()
	testTransactionPriceNonceSort(t, nil)
}

func TestTransactionPriceNonceSort1559(t *testing.T) {
	t.Parallel()
	testTransactionPriceNonceSort(t, big.NewInt(0))
	testTransactionPriceNonceSort(t, big.NewInt(5))
	testTransactionPriceNonceSort(t, big.NewInt(50))
}

// Tests that transactions can be correctly sorted according to their price in
// decreasing order, but at the same time with increasing nonces when issued by
// the same account.
func testTransactionPriceNonceSort(t *testing.T, baseFee *big.Int) {
	// Generate a batch of accounts to start with
	keys := make([]*ecdsa.PrivateKey, 25)
	for i := 0; i < len(keys); i++ {
		keys[i], _ = crypto.GenerateKey()
	}
	signer := types.LatestSignerForChainID(common.Big1)

	// Generate a batch of transactions with overlapping values, but shifted nonces
	groups := map[common.Address][]*txpool.LazyTransaction{}
	expectedCount := 0
	for start, key := range keys {
		addr := crypto.PubkeyToAddress(key.PublicKey)
		count := 25
		for i := 0; i < 25; i++ {
			var tx *types.Transaction
			gasFeeCap := rand.IntN(50)
			if baseFee == nil {
				tx = types.NewTx(&types.LegacyTx{
					Nonce: uint64(start + i),
					// no to, cannot be a sponsored tx
					Value:    big.NewInt(100),
					Gas:      100,
					GasPrice: big.NewInt(int64(gasFeeCap)),
					Data:     nil,
				})
			} else {
				tx = types.NewTx(&types.DynamicFeeTx{
					Nonce: uint64(start + i),
					// no to, cannot be a sponsored tx
					Value:     big.NewInt(100),
					Gas:       100,
					GasFeeCap: big.NewInt(int64(gasFeeCap)),
					GasTipCap: big.NewInt(int64(rand.IntN(gasFeeCap + 1))),
					Data:      nil,
				})
				if count == 25 && int64(gasFeeCap) < baseFee.Int64() {
					count = i
				}
			}
			tx, err := types.SignTx(tx, signer, key)
			if err != nil {
				t.Fatalf("failed to sign tx: %s", err)
			}
			groups[addr] = append(groups[addr], &txpool.LazyTransaction{
				Hash:      tx.Hash(),
				Tx:        tx,
				Time:      tx.Time(),
				GasFeeCap: uint256.MustFromBig(tx.GasFeeCap()),
				GasTipCap: uint256.MustFromBig(tx.GasTipCap()),
				Gas:       tx.Gas(),
				BlobGas:   tx.BlobGas(),
			})
		}
		expectedCount += count
	}
	// Sort the transactions and cross check the nonce ordering
	txset := newTransactionsByPriorityAndPriceAndNonce(groups, baseFee, nil, alwaysMyTurn)

	txs := types.Transactions{}
	for entry, ok := txset.Peek(); ok; entry, ok = txset.Peek() {
		txs = append(txs, entry.tx.Tx)
		txset.Shift()
	}
	if len(txs) != expectedCount {
		t.Errorf("expected %d transactions, found %d", expectedCount, len(txs))
	}
	for i, txi := range txs {
		fromi, _ := types.Sender(signer, txi)

		// Make sure the nonce order is valid
		for j, txj := range txs[i+1:] {
			fromj, _ := types.Sender(signer, txj)
			if fromi == fromj && txi.Nonce() > txj.Nonce() {
				t.Errorf("invalid nonce ordering: tx #%d (A=%x N=%v) < tx #%d (A=%x N=%v)", i, fromi[:4], txi.Nonce(), i+j, fromj[:4], txj.Nonce())
			}
		}
		// If the next tx has different from account, the price must be lower than the current one
		if i+1 < len(txs) {
			next := txs[i+1]
			fromNext, _ := types.Sender(signer, next)
			tip, err := gasprice.EffectiveGasTip(txi, baseFee)
			nextTip, nextErr := gasprice.EffectiveGasTip(next, baseFee)
			if err != nil || nextErr != nil {
				t.Errorf("error calculating effective tip: %v, %v", err, nextErr)
			}
			if fromi != fromNext && tip.Cmp(nextTip) < 0 {
				t.Errorf("invalid gasprice ordering: tx #%d (A=%x P=%v) < tx #%d (A=%x P=%v)", i, fromi[:4], txi.GasPrice(), i+1, fromNext[:4], next.GasPrice())
			}
		}
	}
}

// Tests that if multiple transactions have the same price, the ones seen earlier
// are prioritized to avoid network spam attacks aiming for a specific ordering.
func TestTransactionTimeSort(t *testing.T) {
	t.Parallel()
	// Generate a batch of accounts to start with
	keys := make([]*ecdsa.PrivateKey, 5)
	for i := 0; i < len(keys); i++ {
		keys[i], _ = crypto.GenerateKey()
	}
	signer := types.HomesteadSigner{}

	// Generate a batch of transactions with overlapping prices, but different creation times
	groups := map[common.Address][]*txpool.LazyTransaction{}
	for start, key := range keys {
		addr := crypto.PubkeyToAddress(key.PublicKey)

		tx, _ := types.SignTx(types.NewTransaction(0, common.Address{}, big.NewInt(100), 100, big.NewInt(1), nil), signer, key)
		tx.SetTime(time.Unix(0, int64(len(keys)-start)))

		groups[addr] = append(groups[addr], &txpool.LazyTransaction{
			Hash:      tx.Hash(),
			Tx:        tx,
			Time:      tx.Time(),
			GasFeeCap: uint256.MustFromBig(tx.GasFeeCap()),
			GasTipCap: uint256.MustFromBig(tx.GasTipCap()),
			Gas:       tx.Gas(),
			BlobGas:   tx.BlobGas(),
		})
	}
	// Sort the transactions and cross check the nonce ordering
	txset := newTransactionsByPriorityAndPriceAndNonce(groups, nil, nil, alwaysMyTurn)

	txs := types.Transactions{}
	for entry, ok := txset.Peek(); ok; entry, ok = txset.Peek() {
		txs = append(txs, entry.tx.Tx)
		txset.Shift()
	}
	if len(txs) != len(keys) {
		t.Errorf("expected %d transactions, found %d", len(keys), len(txs))
	}
	for i, txi := range txs {
		fromi, _ := types.Sender(signer, txi)
		if i+1 < len(txs) {
			next := txs[i+1]
			fromNext, _ := types.Sender(signer, next)

			if txi.GasPrice().Cmp(next.GasPrice()) < 0 {
				t.Errorf("invalid gasprice ordering: tx #%d (A=%x P=%v) < tx #%d (A=%x P=%v)", i, fromi[:4], txi.GasPrice(), i+1, fromNext[:4], next.GasPrice())
			}
			// Make sure time order is ascending if the txs have the same gas price
			if txi.GasPrice().Cmp(next.GasPrice()) == 0 && txi.Time().After(next.Time()) {
				t.Errorf("invalid received time ordering: tx #%d (A=%x T=%v) > tx #%d (A=%x T=%v)", i, fromi[:4], txi.Time(), i+1, fromNext[:4], next.Time())
			}
		}
	}
}

func TestTransactionsOrdering_MinerFeesCanBeComputedWithAllTransactions(t *testing.T) {

	baseFee := uint256.NewInt(50)

	// This test ensures that transactions with zero gas tip do not overflow
	// when calculating the miner fee for sorting purposes.

	tests := map[string]struct {
		tx               *types.Transaction
		expectedError    error
		expectedMinerFee uint64
	}{
		"sponsored transaction": {
			tx: types.NewTx(&types.DynamicFeeTx{
				To:        &common.Address{}, // not a contract creation
				Value:     big.NewInt(100),
				GasFeeCap: big.NewInt(0),
				GasTipCap: big.NewInt(0),
				V:         big.NewInt(27), // non-internal, since internal transaction cannot be sponsored
			}),
			expectedMinerFee: 0,
		},
		"non sponsored transaction": {
			tx: types.NewTx(&types.DynamicFeeTx{
				To:        &common.Address{}, // not a contract creation
				Value:     big.NewInt(100),
				Gas:       100,
				GasFeeCap: big.NewInt(0),
				GasTipCap: big.NewInt(0),
			}),
			expectedError: types.ErrGasFeeCapTooLow,
		},
		"non sponsored transaction enough fee cap and zero tip": {
			tx: types.NewTx(&types.DynamicFeeTx{
				To:        &common.Address{}, // not a contract creation
				Gas:       100,
				GasFeeCap: big.NewInt(100),
				GasTipCap: big.NewInt(0),
			}),
			expectedMinerFee: 0,
		},
		"non sponsored transaction with enough fee cap and tip": {
			tx: types.NewTx(&types.DynamicFeeTx{
				To:        &common.Address{}, // not a contract creation
				Gas:       100,
				GasFeeCap: big.NewInt(100),
				GasTipCap: big.NewInt(10),
			}),
			expectedMinerFee: 10,
		},
		"non sponsored legacy transaction": {
			// legacy transactions have a default tip equal to the gas price
			tx: types.NewTx(&types.LegacyTx{
				Nonce:    uint64(0),
				To:       &common.Address{},
				Value:    big.NewInt(100),
				Gas:      100,
				GasPrice: big.NewInt(100),
			}),
			expectedMinerFee: 50, // gas price - base fee
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			lazy := &txpool.LazyTransaction{
				Hash:      test.tx.Hash(),
				Tx:        test.tx,
				Time:      test.tx.Time(),
				GasFeeCap: utils.BigIntToUint256Clamped(test.tx.GasFeeCap()),
				GasTipCap: utils.BigIntToUint256Clamped(test.tx.GasTipCap()),
				Gas:       test.tx.Gas(),
				BlobGas:   test.tx.BlobGas(),
			}
			from := common.Address{1}

			withFee, err := newTxWithMetadata(lazy, from, baseFee, priorities.Priority{}, false)
			require.ErrorIs(t, err, test.expectedError)
			if test.expectedError == nil {
				require.EqualValues(t, withFee.tip.Uint64(), test.expectedMinerFee)
			}
		})
	}
}

func TestTxWithMinerFee_StageFollowsPriorityAndTurn(t *testing.T) {
	tests := map[string]struct {
		priority priorities.Priority
		myTurn   bool
		want     stage
	}{
		"prioritized, my turn":     {prioritized(1), true, stagePrioritizedMyTurn},
		"prioritized, not my turn": {prioritized(1), false, stagePrioritizedNotMyTurn},
		"ordinary, my turn":        {priorities.Priority{}, true, stageOrdinary},
		"ordinary, not my turn":    {priorities.Priority{}, false, stageOrdinary},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			entry := txWithMetadata{priority: test.priority, myTurn: test.myTurn}
			require.Equal(t, test.want, entry.stage())
		})
	}
}

func TestTxOrdering_Peek_OnEmptySetReturnsFalse(t *testing.T) {
	txset := newTxSet(map[common.Address][]*txpool.LazyTransaction{}, nil, alwaysMyTurn)
	_, ok := txset.Peek()
	require.False(t, ok)
}

func TestTxOrdering_NonceOrderPreserved(t *testing.T) {
	keyA, addrA := newSenderKey(t)
	keyB, addrB := newSenderKey(t)
	keyC, addrC := newSenderKey(t)

	build := func(k *ecdsa.PrivateKey) []*txpool.LazyTransaction {
		return []*txpool.LazyTransaction{
			makeLazyTx(t, k, 0, 5, baseTime),
			makeLazyTx(t, k, 1, 9, baseTime),
			makeLazyTx(t, k, 2, 1, baseTime),
			makeLazyTx(t, k, 3, 7, baseTime),
		}
	}
	a, b, c := build(keyA), build(keyB), build(keyC)

	// Scatter priorities across senders and nonces.
	classifier := fakePriorityClassifier{byHash: map[common.Hash]priorities.Priority{
		a[0].Hash: prioritized(1),
		a[2].Hash: prioritized(1),
		b[1].Hash: priorityWith(2, 2, 1),
		c[0].Hash: prioritized(3),
		c[3].Hash: prioritized(3),
	}}

	txset := newTxSet(map[common.Address][]*txpool.LazyTransaction{
		addrA: a, addrB: b, addrC: c,
	}, classifier, alwaysMyTurn)

	out := drain(txset)
	require.Len(t, out, 12)
	assertNonceOrder(t, out)
}

func TestTxOrdering_Peek_OrdersByPriorityLevelThenWeightThenPriceThenTime(t *testing.T) {
	keyA, addrA := newSenderKey(t)
	keyB, addrB := newSenderKey(t)
	keyC, addrC := newSenderKey(t)
	keyD, addrD := newSenderKey(t)
	keyE, addrE := newSenderKey(t)
	keyF, addrF := newSenderKey(t)
	keyG, addrG := newSenderKey(t)

	a := makeLazyTx(t, keyA, 0, 1, baseTime)
	b := makeLazyTx(t, keyB, 0, 1, baseTime)
	c := makeLazyTx(t, keyC, 0, 1, baseTime)
	d := makeLazyTx(t, keyD, 0, 2, baseTime)
	e := makeLazyTx(t, keyE, 0, 4, baseTime)
	f := makeLazyTx(t, keyF, 0, 5, baseTime.Add(time.Second))
	g := makeLazyTx(t, keyG, 0, 5, baseTime)
	classifier := fakePriorityClassifier{byHash: map[common.Hash]priorities.Priority{
		a.Hash: priorityWith(1, 1, 2),
		b.Hash: priorityWith(2, 2, 1),
		c.Hash: priorityWith(3, 1, 1),
		d.Hash: priorityWith(4, 1, 1),
	}}
	txset := newTxSet(map[common.Address][]*txpool.LazyTransaction{
		addrA: {a}, addrB: {b}, addrC: {c}, addrD: {d}, addrE: {e}, addrF: {f}, addrG: {g},
	}, classifier, alwaysMyTurn)

	require.Equal(t,
		[]common.Hash{
			b.Hash, // level 2, weight 1, price 1
			a.Hash, // level 1, weight 2, price 1
			d.Hash, // level 1, weight 1, price 2
			c.Hash, // level 1, weight 1, price 1
			g.Hash, // non-prioritized, price 5
			f.Hash, // non-prioritized, price 5, later time
			e.Hash, // non-prioritized, price 4
		},
		hashesOf(drain(txset)),
	)
}

// TestTxOrdering_Peek_OrdersByStageBeforePriority verifies that the stage
// dominates the composite order: prioritized heads of another validator's turn
// follow every prioritized head of this one's and precede every ordinary head,
// whatever their priority and tip.
func TestTxOrdering_Peek_OrdersByStageBeforePriority(t *testing.T) {
	prio := make([]*txpool.LazyTransaction, 4)
	byHash := map[common.Hash]priorities.Priority{}
	txs := map[common.Address][]*txpool.LazyTransaction{}
	for i := range prio {
		key, addr := newSenderKey(t)
		prio[i] = makeLazyTx(t, key, 0, 1, baseTime)
		byHash[prio[i].Hash] = priorityWith(byte(i), uint64(len(prio)-i), 1)
		txs[addr] = []*txpool.LazyTransaction{prio[i]}
	}
	keyO, addrO := newSenderKey(t)
	ordinary := makeLazyTx(t, keyO, 0, 100, baseTime) // the highest tip of all
	txs[addrO] = []*txpool.LazyTransaction{ordinary}

	classifier := fakePriorityClassifier{byHash: byHash}
	// The two highest-priority transactions are another validator's turn.
	txset := newTxSet(txs, classifier, notMyTurnFor(prio[0], prio[1]))

	require.Equal(t,
		[]common.Hash{
			prio[2].Hash, prio[3].Hash, // my turn, levels 2 and 1
			prio[0].Hash, prio[1].Hash, // not my turn, despite levels 4 and 3
			ordinary.Hash, // ordinary, despite the highest tip
		},
		hashesOf(drain(txset)),
	)
}

func TestTxOrdering_Peek_PrioritizedCannotJumpOwnLowerNonce(t *testing.T) {
	keyA, addrA := newSenderKey(t)
	keyB, addrB := newSenderKey(t)

	a0 := makeLazyTx(t, keyA, 0, 1, baseTime)
	a1 := makeLazyTx(t, keyA, 1, 100, baseTime)
	b0 := makeLazyTx(t, keyB, 0, 10, baseTime)

	classifier := fakePriorityClassifier{byHash: map[common.Hash]priorities.Priority{
		a1.Hash: prioritized(1),
	}}
	txset := newTxSet(map[common.Address][]*txpool.LazyTransaction{
		addrA: {a0, a1}, addrB: {b0},
	}, classifier, alwaysMyTurn)

	// No prioritized head exists at construction: both heads are non-prioritized.
	head, ok := txset.Peek()
	require.True(t, ok)
	require.Equal(t, stageOrdinary, head.stage())

	order := hashesOf(drain(txset))
	// b0 (higher tip) then a0, and only afterwards the prioritized a1.
	require.Equal(t, []common.Hash{b0.Hash, a0.Hash, a1.Hash}, order)
}

// TestTxOrdering_Shift_StagesPromotedHead verifies that the promoted head is
// staged on its own priority and turn.
func TestTxOrdering_Shift_StagesPromotedHead(t *testing.T) {
	keyA, addrA := newSenderKey(t)
	head := makeLazyTx(t, keyA, 0, 10, baseTime)
	next := makeLazyTx(t, keyA, 1, 10, baseTime)

	tests := map[string]struct {
		nextPrioritized bool
		nextMyTurn      bool
		want            stage
	}{
		"prioritized, my turn":     {true, true, stagePrioritizedMyTurn},
		"prioritized, not my turn": {true, false, stagePrioritizedNotMyTurn},
		"ordinary, my turn":        {false, true, stageOrdinary},
		"ordinary, not my turn":    {false, false, stageOrdinary},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			byHash := map[common.Hash]priorities.Priority{head.Hash: prioritized(1)}
			if test.nextPrioritized {
				byHash[next.Hash] = prioritized(1)
			}
			policy := alwaysMyTurn
			if !test.nextMyTurn {
				policy = notMyTurnFor(next)
			}
			txset := newTxSet(
				map[common.Address][]*txpool.LazyTransaction{addrA: {head, next}},
				fakePriorityClassifier{byHash: byHash}, policy,
			)

			txset.Shift()

			got, ok := txset.Peek()
			require.True(t, ok)
			require.Equal(t, next.Hash, got.tx.Hash)
			require.Equal(t, test.want, got.stage())
		})
	}
}

// TestTxOrdering_Shift_PromotedHeadCanReenterAnEarlierStage verifies that the
// stage of a sender is not monotonic: the transaction promoted after a
// not-my-turn head is staged on its own turn and thereby overtakes the
// higher-priority heads still waiting in the later stage.
func TestTxOrdering_Shift_PromotedHeadCanReenterAnEarlierStage(t *testing.T) {
	keyA, addrA := newSenderKey(t)
	a0 := makeLazyTx(t, keyA, 0, 1, baseTime)
	a1 := makeLazyTx(t, keyA, 1, 1, baseTime)
	keyB, addrB := newSenderKey(t)
	b0 := makeLazyTx(t, keyB, 0, 1, baseTime)

	classifier := fakePriorityClassifier{byHash: map[common.Hash]priorities.Priority{
		a0.Hash: priorityWith(1, 5, 1),
		a1.Hash: priorityWith(2, 1, 1),
		b0.Hash: priorityWith(3, 3, 1),
	}}
	txset := newTxSet(map[common.Address][]*txpool.LazyTransaction{
		addrA: {a0, a1}, addrB: {b0},
	}, classifier, notMyTurnFor(a0, b0))

	// a1 is this validator's turn, so it precedes b0 despite its lower level.
	require.Equal(t, []common.Hash{a0.Hash, a1.Hash, b0.Hash}, hashesOf(drain(txset)))
}

func TestTxOrdering_Shift_ExhaustedSenderLeavesTheSet(t *testing.T) {
	keyA, addrA := newSenderKey(t)
	head := makeLazyTx(t, keyA, 0, 10, baseTime)
	classifier := fakePriorityClassifier{byHash: map[common.Hash]priorities.Priority{
		head.Hash: prioritized(1),
	}}
	txset := newTxSet(map[common.Address][]*txpool.LazyTransaction{addrA: {head}}, classifier, alwaysMyTurn)

	txset.Shift()

	_, ok := txset.Peek()
	require.False(t, ok)
}

func TestTxOrdering_Discard_DropsSenderRemainder(t *testing.T) {
	keyA, addrA := newSenderKey(t)
	head := makeLazyTx(t, keyA, 0, 10, baseTime)
	next := makeLazyTx(t, keyA, 1, 10, baseTime)
	classifier := fakePriorityClassifier{byHash: map[common.Hash]priorities.Priority{
		head.Hash: prioritized(1), next.Hash: prioritized(1),
	}}
	txset := newTxSet(map[common.Address][]*txpool.LazyTransaction{addrA: {head, next}}, classifier, alwaysMyTurn)

	txset.Discard()

	_, ok := txset.Peek()
	require.False(t, ok)
}

var (
	orderingSigner = types.LatestSignerForChainID(common.Big1)
	baseTime       = time.Unix(1_000, 0)
)

// newSenderKey returns a fresh private key and its derived sender address.
func newSenderKey(t *testing.T) (*ecdsa.PrivateKey, common.Address) {
	t.Helper()
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	return key, crypto.PubkeyToAddress(key.PublicKey)
}

// makeLazyTx builds a signed dynamic-fee LazyTransaction. With a nil base fee
// (as used by these tests) the effective miner tip equals tip, so tip is the
// "price" used for ordering. at is the transaction's first-seen time, used as
// the final ordering tie-break.
func makeLazyTx(t *testing.T, key *ecdsa.PrivateKey, nonce uint64, tip int64, at time.Time) *txpool.LazyTransaction {
	t.Helper()
	raw, err := types.SignTx(types.NewTx(&types.DynamicFeeTx{
		ChainID:   common.Big1,
		Nonce:     nonce,
		To:        &common.Address{0x2},
		Value:     big.NewInt(0),
		Gas:       21000,
		GasFeeCap: big.NewInt(tip),
		GasTipCap: big.NewInt(tip),
	}), orderingSigner, key)
	require.NoError(t, err)
	raw.SetTime(at)
	return &txpool.LazyTransaction{
		Hash:      raw.Hash(),
		Tx:        raw,
		Time:      raw.Time(),
		GasFeeCap: uint256.MustFromBig(raw.GasFeeCap()),
		GasTipCap: uint256.MustFromBig(raw.GasTipCap()),
		Gas:       raw.Gas(),
	}
}

// newTxSet builds a transaction set with a nil base fee.
func newTxSet(txs map[common.Address][]*txpool.LazyTransaction, classifier priorities.Classifier, policy func(tx *txpool.LazyTransaction) bool) *transactionsByPriorityAndPriceAndNonce {
	return newTransactionsByPriorityAndPriceAndNonce(txs, nil, classifier, policy)
}

// notMyTurnFor returns a turn policy putting every transaction but the given
// ones at this validator's turn.
func notMyTurnFor(txs ...*txpool.LazyTransaction) func(tx *txpool.LazyTransaction) bool {
	theirs := make(map[common.Hash]bool, len(txs))
	for _, tx := range txs {
		theirs[tx.Hash] = true
	}
	return func(tx *txpool.LazyTransaction) bool { return !theirs[tx.Hash] }
}

// drain consumes the whole set in its ordering.
func drain(txset *transactionsByPriorityAndPriceAndNonce) []*txWithMetadata {
	var out []*txWithMetadata
	for e, ok := txset.Peek(); ok; e, ok = txset.Peek() {
		out = append(out, e)
		txset.Shift()
	}
	return out
}

func hashesOf(entries []*txWithMetadata) []common.Hash {
	out := make([]common.Hash, len(entries))
	for i, e := range entries {
		out[i] = e.tx.Hash
	}
	return out
}

// assertNonceOrder verifies that, within each sender, nonces are emitted in
// strictly increasing order.
func assertNonceOrder(t *testing.T, entries []*txWithMetadata) {
	t.Helper()
	last := map[common.Address]uint64{}
	for _, e := range entries {
		nonce := e.tx.Tx.Nonce()
		if prev, ok := last[e.from]; ok {
			require.Greater(t, nonce, prev)
		}
		last[e.from] = nonce
	}
}
