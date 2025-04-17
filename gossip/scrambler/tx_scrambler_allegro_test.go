package scrambler

import (
	"cmp"
	"crypto/ecdsa"
	"math/big"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestScrambler_EndToEndTest(t *testing.T) {
	numTransactions := 256
	signer := types.NewPragueSigner(big.NewInt(1))

	// generate numTransactions account keys
	accountKeys := make([]*ecdsa.PrivateKey, numTransactions)
	for i := range numTransactions {
		key, err := crypto.GenerateKey()
		require.NoError(t, err)
		accountKeys[i] = key
	}

	transactions := make([]*types.Transaction, 0, numTransactions)
	for i := range numTransactions {
		key := accountKeys[i]
		tx, err := types.SignTx(types.NewTx(&types.LegacyTx{
			Nonce:    uint64(i),
			GasPrice: big.NewInt(int64(42 + i)),
		}), signer, key)
		require.NoError(t, err)
		transactions = append(transactions, tx)
	}

	Scramble(transactions, 42, signer)
	if len(transactions) != numTransactions {
		t.Fatalf("expected %d transactions, got %d", numTransactions, len(transactions))
	}
}

func TestScrambler_ScramblingIsDeterministic(t *testing.T) {
	entries := []ScramblerEntry{
		&dummyScramblerEntry{
			hash:     common.Hash{1},
			sender:   common.Address{1},
			nonce:    1,
			gasPrice: big.NewInt(1),
		},
		&dummyScramblerEntry{
			hash:     common.Hash{2},
			sender:   common.Address{2},
			nonce:    2,
			gasPrice: big.NewInt(2),
		},
		&dummyScramblerEntry{
			hash:     common.Hash{3},
			sender:   common.Address{1},
			nonce:    1,
			gasPrice: big.NewInt(1),
		},
		&dummyScramblerEntry{
			hash:     common.Hash{4},
			sender:   common.Address{2},
			nonce:    2,
			gasPrice: big.NewInt(2),
		},
	}

	scrambledEntries := slices.Clone(entries)
	permutation1 := scramblePermutation(scrambledEntries, 42)
	applyPermutation(scrambledEntries, permutation1)
	for range 10 {
		shuffleEntries(entries)
		permutation2 := scramblePermutation(entries, 42)
		applyPermutation(entries, permutation2)
		require.Equal(t, scrambledEntries, entries, "scrambling should be deterministic")
	}
}

func TestScrambler_ScramblingIsDeterministicRandomInput(t *testing.T) {
	entries := generateScramblerInput(1000)

	scrambledEntries := slices.Clone(entries)
	permutation1 := scramblePermutation(scrambledEntries, 42)
	applyPermutation(scrambledEntries, permutation1)
	for range 10 {
		shuffleEntries(entries)
		permutation2 := scramblePermutation(entries, 42)
		applyPermutation(entries, permutation2)
		require.Equal(t, scrambledEntries, entries, "scrambling should be deterministic")
	}
}

func TestScrambler_OrderIsBasedOnNonceGasPriceAndHash(t *testing.T) {
	entries := generateScramblerInput(10000)

	scrambledEntries := slices.Clone(entries)
	permutation1 := scramblePermutation(scrambledEntries, 42)
	applyPermutation(scrambledEntries, permutation1)

	previous := entries[0]
	previousSender := previous.Sender()
	for i, entry := range scrambledEntries {
		if i == 0 {
			continue
		}
		if previousSender.Cmp(entry.Sender()) != 0 {
			previous = entry
			previousSender = entry.Sender()
			continue
		}
		if cmp.Compare(previous.Nonce(), entry.Nonce()) < 0 {
			previous = entry
			continue
		}
		if previous.GasPrice().Cmp(entry.GasPrice()) > 0 {
			previous = entry
			continue
		}
		if previous.Hash().Cmp(entry.Hash()) < 0 {
			previous = entry
			continue
		}
		t.Fatal("order is not based on sender nonce gas price and hash")
	}
}

func TestScrambler_ScrambleEntriesIsDeterministic(t *testing.T) {
	entries := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	reference := slices.Clone(entries)
	scrambleEntries(reference, 42)

	for range 100 {
		scrambled := slices.Clone(entries)
		scrambleEntries(scrambled, 42)

		require.Equal(t, reference, scrambled, "scrambling should be deterministic")
	}
}

func TestScrambler_ScrambleEntriesReturnsDifferentOrders(t *testing.T) {
	entries := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	for seed := range 10 {
		scrambled := slices.Clone(entries)
		scrambleEntries(scrambled, uint64(seed+1))

		require.NotEqual(t, entries, scrambled, "scrambling should return different orders")
	}
}

func TestScrambler_ApplyPermutation(t *testing.T) {
	items := make([]int, 1000)
	for i := range items {
		items[i] = i
	}

	indices := rand.Perm(len(items))
	indicesCopy := slices.Clone(indices)
	applyPermutation(items, indicesCopy)

	require.Equal(t, indices, items, "permutation should be the same as the original items")
}

func generateScramblerInput(size int) []ScramblerEntry {
	entries := make([]ScramblerEntry, size)
	for i := range size {
		// ~1/10th of the entries will have the same sender, nonce or gas price
		sender := rand.IntN(size / 10)
		nonce := rand.IntN(size / 10)
		gasPrice := rand.IntN(size / 10)
		entries[i] = &dummyScramblerEntry{
			hash:     common.Hash(uint256.NewInt(uint64(i)).Bytes32()),
			sender:   common.Address(uint256.NewInt(uint64(sender)).Bytes20()),
			nonce:    uint64(nonce),
			gasPrice: big.NewInt(int64(gasPrice)),
		}
	}

	return entries
}
