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
	txset := newTransactionsByPriceAndNonce(signer, groups, baseFee)

	txs := types.Transactions{}
	for tx, _ := txset.Peek(); tx != nil; tx, _ = txset.Peek() {
		txs = append(txs, tx.Tx)
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
			tip, err := txi.EffectiveGasTip(baseFee)
			nextTip, nextErr := next.EffectiveGasTip(baseFee)
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
	txset := newTransactionsByPriceAndNonce(signer, groups, nil)

	txs := types.Transactions{}
	for tx, _ := txset.Peek(); tx != nil; tx, _ = txset.Peek() {
		txs = append(txs, tx.Tx)
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
				GasFeeCap: utils.BigIntToUint256(test.tx.GasFeeCap()),
				GasTipCap: utils.BigIntToUint256(test.tx.GasTipCap()),
				Gas:       test.tx.Gas(),
				BlobGas:   test.tx.BlobGas(),
			}
			from := common.Address{1}

			withFee, err := newTxWithMinerFee(lazy, from, baseFee)
			require.ErrorIs(t, err, test.expectedError)
			if test.expectedError == nil {
				require.EqualValues(t, withFee.fees.Uint64(), test.expectedMinerFee)
			}
		})
	}
}
