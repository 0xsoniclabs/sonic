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
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/priorities"
	"github.com/0xsoniclabs/sonic/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestTxWithMetadata_StageFollowsPriorityAndTurn(t *testing.T) {
	tests := map[string]struct {
		priority priorities.Priority
		myTurn   bool
		want     stage
	}{
		"prioritized, my turn":     {withPrio, true, stagePrioritizedMyTurn},
		"prioritized, not my turn": {withPrio, false, stagePrioritizedNotMyTurn},
		"ordinary, my turn":        {withoutPrio, true, stageNotPrioritized},
		"ordinary, not my turn":    {withoutPrio, false, stageNotPrioritized},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			entry := txWithMetadata{priority: test.priority, myTurn: test.myTurn}
			require.Equal(t, test.want, entry.stage())
		})
	}
}

func TestComputeEffectiveTip_TipCanBeComputedForAllTransactionKinds(t *testing.T) {

	baseFee := uint256.NewInt(50)

	// This test ensures that transactions with zero gas tip do not overflow
	// when calculating the miner fee for sorting purposes.

	tests := map[string]struct {
		tx            *types.Transaction
		expectedError error
		expectedTip   uint64
	}{
		"sponsored transaction": {
			tx: types.NewTx(&types.DynamicFeeTx{
				To:        &common.Address{}, // not a contract creation
				Value:     big.NewInt(100),
				GasFeeCap: big.NewInt(0),
				GasTipCap: big.NewInt(0),
				V:         big.NewInt(27), // non-internal, since internal transaction cannot be sponsored
			}),
			expectedTip: 0,
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
			expectedTip: 0,
		},
		"non sponsored transaction with enough fee cap and tip": {
			tx: types.NewTx(&types.DynamicFeeTx{
				To:        &common.Address{}, // not a contract creation
				Gas:       100,
				GasFeeCap: big.NewInt(100),
				GasTipCap: big.NewInt(10),
			}),
			expectedTip: 10,
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
			expectedTip: 50, // gas price - base fee
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
			tip, err := computeEffectiveTip(lazy, baseFee)
			require.ErrorIs(t, err, test.expectedError)
			if test.expectedError == nil {
				require.EqualValues(t, test.expectedTip, tip.Uint64())
			}
		})
	}
}

func TestCompareTxByStagePriorityPriceTime_OrdersByStageThenPriorityThenTipThenTime(t *testing.T) {
	later := baseTime.Add(time.Second)
	entry := func(priority priorities.Priority, myTurn bool, tip uint64, at time.Time) *txWithMetadata {
		return &txWithMetadata{
			tx:       &txpool.LazyTransaction{Time: at},
			tip:      uint256.NewInt(tip),
			priority: priority,
			myTurn:   myTurn,
		}
	}

	tests := map[string]struct {
		a, b     *txWithMetadata
		expected int
	}{
		"my turn with priority precedes not my turn, whatever its priority and tip and time": {
			a:        entry(withPrio, true, 1, later),
			b:        entry(withPrio, false, 100, baseTime),
			expected: 1,
		},
		"my turn with priority precedes without priority, whatever its turn and tip and time": {
			a:        entry(withPrio, true, 1, later),
			b:        entry(withoutPrio, true, 100, baseTime),
			expected: 1,
		},
		"not my turn with priority precedes without priority, whatever its turn and tip and time": {
			a:        entry(withPrio, false, 1, later),
			b:        entry(withoutPrio, true, 100, baseTime),
			expected: 1,
		},
		"without priority, the turn is ignored": {
			a:        entry(withoutPrio, true, 1, baseTime),
			b:        entry(withoutPrio, false, 1, baseTime),
			expected: 0,
		},
		"higher level precedes, whatever its weight and tip and time": {
			a:        entry(prio(2, 1, 0), true, 1, later),
			b:        entry(prio(1, 9, 0), true, 100, baseTime),
			expected: 1,
		},
		"higher weight precedes, whatever its tip and time": {
			a:        entry(prio(1, 2, 0), true, 1, later),
			b:        entry(prio(1, 1, 0), true, 100, baseTime),
			expected: 1,
		},
		"without priority, the weight is ignored": {
			a:        entry(prio(0, 1, 0), true, 1, baseTime),
			b:        entry(prio(0, 9, 0), true, 1, baseTime),
			expected: 0,
		},
		"higher tip precedes, whatever its time": {
			a:        entry(withoutPrio, true, 2, later),
			b:        entry(withoutPrio, true, 1, baseTime),
			expected: 1,
		},
		"earlier time precedes": {
			a:        entry(withoutPrio, true, 1, baseTime),
			b:        entry(withoutPrio, true, 1, later),
			expected: 1,
		},
		"entries differing only in priority ID are equivalent": {
			a:        entry(prio(1, 1, 1), true, 1, baseTime),
			b:        entry(prio(1, 1, 2), true, 1, baseTime),
			expected: 0,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := compareTxByStagePriorityPriceTime(test.a, test.b)
			require.Equal(t, test.expected, got)
			reversed := compareTxByStagePriorityPriceTime(test.b, test.a)
			require.Equal(t, -test.expected, reversed)
		})
	}
}

// TestTransactionsByPriorityAndPriceAndNonce_OrdersByStagePriorityTipAndTime
// cross-checks the composite order on a whole set: that the metadata the set
// attaches to each transaction feeds the comparison as intended, that only a
// sender's lowest nonce takes part in that comparison, and that a promoted
// nonce is staged on its own metadata rather than its predecessor's - which
// makes the resulting order non-monotonic.
func TestTransactionsByPriorityAndPriceAndNonce_OrdersByStagePriorityTipAndTime(t *testing.T) {
	specs := []struct {
		name     string
		sender   string
		priority priorities.Priority
		myTurn   bool
		tip      int64
		delay    time.Duration
	}{
		{name: "level2", sender: "a", priority: prio(2, 1, 1), myTurn: true, tip: 1},
		{name: "level1/weight2", sender: "b", priority: prio(1, 2, 2), myTurn: true, tip: 1},
		{name: "level1/weight1", sender: "c", priority: prio(1, 1, 3), myTurn: true, tip: 1},
		{name: "prioNotMyTurn", sender: "d", priority: prio(9, 9, 4), tip: 1},
		{name: "tip50", sender: "e", tip: 50},
		{name: "tip5/early", sender: "f", tip: 5},
		{name: "tip5/late", sender: "g", tip: 5, delay: time.Second},
		{name: "blocking", sender: "h", tip: 10},
		{name: "blocked", sender: "h", priority: prio(3, 1, 5), myTurn: true, tip: 100},
	}

	expected := []string{
		// prioritized and my turn, by level, then weight
		"level2", "level1/weight2", "level1/weight1",
		// prioritized but not my turn, whatever its level and weight
		"prioNotMyTurn",
		// not prioritized, by tip - the blocked nonce takes no part in the
		// comparison until its predecessor has been consumed
		"tip50", "blocking",
		// promoted and staged on its own priority, ahead of what is left
		"blocked",
		// not prioritized, equal tips broken by the first-seen time
		"tip5/early", "tip5/late",
	}

	type sender struct {
		key  *ecdsa.PrivateKey
		addr common.Address
	}
	senders := map[string]sender{}
	txBySender := map[common.Address][]*txpool.LazyTransaction{}
	prioByHash := map[common.Hash]priorities.Priority{}
	nameByHash := map[common.Hash]string{}
	turnByHash := map[common.Hash]bool{}
	for _, spec := range specs {
		if _, ok := senders[spec.sender]; !ok {
			key, addr := newSenderKey(t)
			senders[spec.sender] = sender{key, addr}
		}
		s := senders[spec.sender]
		tx := makeLazyTx(t, s.key, uint64(len(txBySender[s.addr])), spec.tip, baseTime.Add(spec.delay))
		txBySender[s.addr] = append(txBySender[s.addr], tx)
		nameByHash[tx.Hash] = spec.name
		prioByHash[tx.Hash] = spec.priority
		turnByHash[tx.Hash] = spec.myTurn
	}

	classifier := priorities.NewMockClassifier(gomock.NewController(t))
	classifier.EXPECT().Priority(gomock.Any()).DoAndReturn(
		func(tx *types.Transaction) (priorities.Priority, error) {
			return prioByHash[tx.Hash()], nil
		},
	).AnyTimes()

	context := &priorityContext{
		cache:      evmcore.NewPriorityCache(evmcore.DefaultTxPoolConfig),
		classifier: classifier,
	}
	txset := newTransactionsByPriorityAndPriceAndNonce(txBySender, nil, context,
		func(tx *txpool.LazyTransaction) bool { return turnByHash[tx.Hash] })

	got := make([]string, 0, len(expected))
	for _, e := range drain(txset) {
		got = append(got, nameByHash[e.tx.Hash])
	}
	require.Equal(t, expected, got)
}

func TestNewTransactionsByPriorityAndPriceAndNonce_DropsSenderTailWithoutComputableTip(t *testing.T) {
	baseFee := big.NewInt(50)

	tests := map[string]struct {
		tips []int64
		want int
	}{
		"every nonce covers the base fee": {[]int64{100, 100, 100}, 3},
		"a later nonce falls short":       {[]int64{100, 10, 100}, 1},
		"the first nonce falls short":     {[]int64{10, 100}, 0},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			key, addr := newSenderKey(t)
			txs := make([]*txpool.LazyTransaction, len(test.tips))
			for i, tip := range test.tips {
				txs[i] = makeLazyTx(t, key, uint64(i), tip, baseTime)
			}
			txset := newTransactionsByPriorityAndPriceAndNonce(
				map[common.Address][]*txpool.LazyTransaction{addr: txs}, baseFee, nil, alwaysMyTurn)

			require.Len(t, drain(txset), test.want)
		})
	}
}

var (
	orderingSigner = types.LatestSignerForChainID(common.Big1)
	baseTime       = time.Unix(1_000, 0)

	withPrio    = prio(1, 1, 1)
	withoutPrio = prio(0, 0, 0)
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

func alwaysMyTurn(*txpool.LazyTransaction) bool { return true }

// myTurnFor returns a turn policy putting only the given transactions at this
// validator's turn.
func myTurnFor(txs ...*types.Transaction) func(tx *txpool.LazyTransaction) bool {
	mine := make(map[common.Hash]bool, len(txs))
	for _, tx := range txs {
		mine[tx.Hash()] = true
	}
	return func(tx *txpool.LazyTransaction) bool { return mine[tx.Hash] }
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
