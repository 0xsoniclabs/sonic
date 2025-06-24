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

package scrambler_test

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/binary"
	"math/big"
	"slices"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/scrambler"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/require"
)

func FuzzScrambler(f *testing.F) {

	signer := types.NewPragueSigner(big.NewInt(1))

	// generate 256 account keys
	accountKeys := make([]*ecdsa.PrivateKey, 256)
	for i := range 256 {
		key, err := crypto.GenerateKey()
		require.NoError(f, err)
		accountKeys[i] = key
	}
	maxMetaTransactionEncodedSize := metaTransactionSizeSerializeSize

	f.Add(encodeTxList(f, []metaTransaction{}))
	f.Add(encodeTxList(f, []metaTransaction{
		{SenderAccount: 0, Nonce: 0, GasPrice: 0},
		{SenderAccount: 0, Nonce: 1, GasPrice: 0},
		{SenderAccount: 0, Nonce: 2, GasPrice: 0},
	}))
	f.Add(encodeTxList(f, []metaTransaction{
		{SenderAccount: 0, Nonce: 0, GasPrice: 1},
		{SenderAccount: 1, Nonce: 0, GasPrice: 10_000},
		{SenderAccount: 255, Nonce: 3, GasPrice: 10_000_000_000},
	}))

	f.Fuzz(func(t *testing.T, encoded []byte) {

		// Bind the input to some reasonable size.
		// metaTransactions serialization size is variable, use worst case scenario
		if len(encoded) > 10_000*maxMetaTransactionEncodedSize {
			t.Skip("input too large")
		}

		metaTxs := parseFuzzedInput(encoded)

		if containsDuplicates(metaTxs) {
			// the scrambler takes as a precondition that transactions cannot be duplicated
			t.Skip("contains duplicates")
		}

		txs := make([]*types.Transaction, 0, len(metaTxs))
		for _, metaTx := range metaTxs {

			key := accountKeys[metaTx.SenderAccount]
			tx, err := types.SignTx(types.NewTx(&types.LegacyTx{
				Nonce:    metaTx.Nonce,
				GasPrice: big.NewInt(int64(metaTx.GasPrice)),
			}), signer, key)
			require.NoError(t, err)

			txs = append(txs, tx)
		}

		ordered := scrambler.GetExecutionOrder(txs, signer, true)

		// clone result to re-shuffle, reorder, and compare the results
		testList := slices.Clone(ordered)

		// shuffle the list, but in a deterministic way
		slices.SortFunc(testList, func(a, b *types.Transaction) int {
			return bytes.Compare(a.Hash().Bytes(), b.Hash().Bytes())
		})

		// re-order the list
		reOrdered := scrambler.GetExecutionOrder(testList, signer, true)

		// compare the results
		if expected, got := len(reOrdered), len(ordered); expected != got {
			t.Fatalf("scrambler did not produce same number of transactions; expected %d, got %d", expected, got)
		}
		for i := range reOrdered {
			if reOrdered[i].Hash() != ordered[i].Hash() {
				t.Errorf("transactions are not sorted")
				for i, tx := range ordered {
					sender, _ := types.Sender(signer, tx)
					t.Logf("tx[%d]: hash %s sender %s nonce: %d gasprice, %d", i, tx.Hash().Hex(), sender.Hex(), tx.Nonce(), tx.GasPrice())
				}
			}
		}
	})
}

func containsDuplicates(txs []metaTransaction) bool {
	seen := make(map[uint8]map[uint64]struct{})
	for _, tx := range txs {
		if _, ok := seen[tx.SenderAccount]; !ok {
			seen[tx.SenderAccount] = make(map[uint64]struct{})
		}
		if _, ok := seen[tx.SenderAccount][tx.Nonce]; ok {
			return true
		}
		seen[tx.SenderAccount][tx.Nonce] = struct{}{}
	}
	return false
}

func TestFuzzScrambler_ContainsDuplicates_DetectsCollisionsOfSenderAndNonce(t *testing.T) {
	tests := map[string]struct {
		txs                []metaTransaction
		expectedDuplicates bool
	}{
		"empty": {
			txs:                []metaTransaction{},
			expectedDuplicates: false,
		},
		"no duplicates, different sender": {
			txs: []metaTransaction{
				{SenderAccount: 0, Nonce: 0},
				{SenderAccount: 1, Nonce: 0},
			},
		},
		"no duplicates, same sender": {
			txs: []metaTransaction{
				{SenderAccount: 0, Nonce: 0},
				{SenderAccount: 0, Nonce: 1},
			},
		},
		"contains duplicates": {
			txs: []metaTransaction{
				{SenderAccount: 0, Nonce: 0},
				{SenderAccount: 0, Nonce: 0},
			},
			expectedDuplicates: true,
		},
		"contains duplicates, interleaved sender": {
			txs: []metaTransaction{
				{SenderAccount: 0, Nonce: 0},
				{SenderAccount: 1, Nonce: 0},
				{SenderAccount: 0, Nonce: 0},
			},
			expectedDuplicates: true,
		},
		"contains duplicates, interleaved nonce": {
			txs: []metaTransaction{
				{SenderAccount: 0, Nonce: 0},
				{SenderAccount: 0, Nonce: 1},
				{SenderAccount: 0, Nonce: 0},
			},
			expectedDuplicates: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, test.expectedDuplicates, containsDuplicates(test.txs))
		})
	}

}

// metaTransaction is a simplified representation of a transaction with the
// fields relevant for the scrambler.
type metaTransaction struct {
	SenderAccount uint8
	Nonce         uint64
	GasPrice      uint64
}

func encodeTxList(t testing.TB, txs []metaTransaction) []byte {
	buf := new(bytes.Buffer)
	require.NoError(t, rlp.Encode(buf, txs))
	return buf.Bytes()
}

const metaTransactionSizeSerializeSize = 17

func parseFuzzedInput(encoded []byte) []metaTransaction {
	txs := make([]metaTransaction, 0, len(encoded)/metaTransactionSizeSerializeSize)

	for i := 0; i < len(encoded); i = i + metaTransactionSizeSerializeSize {
		if i+metaTransactionSizeSerializeSize > len(encoded) {
			// incomplete transaction, ignore
			break
		}

		next := encoded[i : i+metaTransactionSizeSerializeSize]

		senderAccount := uint8(next[0])
		nonce := binary.BigEndian.Uint64(next[1:9])
		gasPrice := binary.BigEndian.Uint64(next[9:17])

		tx := metaTransaction{
			SenderAccount: senderAccount,
			Nonce:         nonce,
			GasPrice:      gasPrice,
		}
		txs = append(txs, tx)
	}

	return txs
}

func TestFuzzScrambler_MetaTransactions_CanBeParsedFromBinaryBuffer(t *testing.T) {

	tests := map[string]struct {
		binary   []byte
		expected []metaTransaction
	}{

		"empty": {
			binary:   []byte{},
			expected: []metaTransaction{},
		},
		"single": {
			binary: []byte{
				7,
				1, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 1,
			},
			expected: []metaTransaction{
				{
					SenderAccount: 7,
					Nonce:         1 << (7 * 8),
					GasPrice:      1,
				},
			},
		},
		"multiple": {
			binary: []byte{
				7,
				1, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 1,
				8,
				2, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 2,
			},
			expected: []metaTransaction{
				{
					SenderAccount: 7,
					Nonce:         1 << (7 * 8),
					GasPrice:      1,
				},
				{
					SenderAccount: 8,
					Nonce:         2 << (7 * 8),
					GasPrice:      2,
				},
			},
		},
		"incomplete is ignored": {
			binary: []byte{
				7,
				1, 0, 0, 0, 0, 0, 0, 0,
			},
			expected: []metaTransaction{},
		},
		"last incomplete is ignored": {
			binary: []byte{
				7,
				1, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 1,
				8,
			},
			expected: []metaTransaction{
				{
					SenderAccount: 7,
					Nonce:         1 << (7 * 8),
					GasPrice:      1,
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			parsed := parseFuzzedInput(test.binary)
			require.Equal(t, test.expected, parsed)
		})
	}
}
