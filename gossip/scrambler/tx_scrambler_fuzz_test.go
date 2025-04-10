package scrambler_test

import (
	"bytes"
	"crypto/ecdsa"
	"math"
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
	maxMetaTransactionEncodedSize := sizeOfEncodedMetaTransaction(f)

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

		stream := rlp.NewStream(bytes.NewReader(encoded), 0)

		metaTxs := make([]metaTransaction, 0)
		if err := stream.Decode(&metaTxs); err != nil {
			t.Skip("invalid input", err)
		}
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

		scrambledOrderIsReproducible(t, ordered, signer)
	})
}

func sizeOfEncodedMetaTransaction(t testing.TB) int {
	bytes, err := rlp.EncodeToBytes(metaTransaction{
		SenderAccount: 0xff,
		Nonce:         math.MaxUint64,
		GasPrice:      math.MaxUint64,
	})
	require.NoError(t, err)
	return len(bytes)
}

// scrambledOrderIsReproducible is a naive implementation that checks if the scrambler order is
// reproducible.
// It is meant to be used by tests only.
func scrambledOrderIsReproducible(t testing.TB, ordered types.Transactions, signer types.Signer) {
	t.Helper()

	testList := slices.Clone(ordered)

	// shuffle the list, but in a deterministic way
	slices.SortFunc(testList, func(a, b *types.Transaction) int {
		return bytes.Compare(a.Hash().Bytes(), b.Hash().Bytes())
	})

	reOrdered := scrambler.GetExecutionOrder(testList, signer, true)
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

func TestContainsDuplicates_DetectsCollisionsOfSenderAndNonce(t *testing.T) {
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
// fields relevant for hte scrambler.
// It can be encoded and decoded using RLP, and this is used for the fuzzing
// to easily generate lists of transactions.
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
