package scrambler

import (
	"cmp"
	"crypto/ecdsa"
	"fmt"
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

// dummyScramblerEntry represents scramblery entry data used for testing
type dummyScramblerEntry struct {
	hash     common.Hash    // transaction hash
	sender   common.Address // sender of the transaction
	nonce    uint64         // transaction nonce
	gasPrice *big.Int       // transaction gasPrice
}

func (s *dummyScramblerEntry) Hash() common.Hash {
	return s.hash
}

func (s *dummyScramblerEntry) Sender() common.Address {
	return s.sender
}

func (s *dummyScramblerEntry) Nonce() uint64 {
	return s.nonce
}

func (s *dummyScramblerEntry) GasPrice() *big.Int {
	return s.gasPrice
}

func compareFunc(a ScramblerEntry, b ScramblerEntry) int {
	addrCmp := a.Sender().Cmp(b.Sender())
	if addrCmp != 0 {
		return addrCmp
	}
	res := cmp.Compare(a.Nonce(), b.Nonce())
	if res != 0 {
		return res
	}
	res = a.GasPrice().Cmp(b.GasPrice())
	if res != 0 {
		return res
	}
	return a.Hash().Cmp(b.Hash())
}

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

	transactions = Scramble(signer, 42, transactions)
	if len(transactions) != numTransactions {
		t.Fatalf("expected %d transactions, got %d", numTransactions, len(transactions))
	}
}

func TestTxScrambler_DuplicatesAreRemoved(t *testing.T) {
	entries := []ScramblerEntry{
		&dummyScramblerEntry{
			hash:   common.Hash{1},
			sender: common.Address{1},
			nonce:  1,
		},
		&dummyScramblerEntry{
			hash:   common.Hash{1},
			sender: common.Address{1},
			nonce:  1,
		},
		&dummyScramblerEntry{
			hash:   common.Hash{1},
			sender: common.Address{1},
			nonce:  1,
		},
		&dummyScramblerEntry{
			hash:   common.Hash{1},
			sender: common.Address{1},
			nonce:  1,
		},
		&dummyScramblerEntry{
			hash:   common.Hash{1},
			sender: common.Address{1},
			nonce:  1,
		},
	}

	shuffleEntries(entries)
	entries = scrambleTransactions(42, entries)
	seen := map[common.Hash]struct{}{}
	for _, entry := range entries {
		if _, ok := seen[entry.Hash()]; ok {
			t.Fatalf("duplicate entry found %s", entry.Hash().Hex())
		}
		seen[entry.Hash()] = struct{}{}
	}
}

func TestTxScrambler_SortTransactionsWithSameSender_SortsByNonce(t *testing.T) {
	entries := []ScramblerEntry{
		&dummyScramblerEntry{
			hash:   common.Hash{1},
			sender: common.Address{1},
			nonce:  1,
		},
		&dummyScramblerEntry{
			hash:   common.Hash{2},
			sender: common.Address{1},
			nonce:  2,
		},
		&dummyScramblerEntry{
			hash:   common.Hash{3},
			sender: common.Address{1},
			nonce:  3,
		},
		&dummyScramblerEntry{
			hash:   common.Hash{4},
			sender: common.Address{1},
			nonce:  4,
		},
		&dummyScramblerEntry{
			hash:   common.Hash{5},
			sender: common.Address{1},
			nonce:  5,
		},
	}

	shuffleEntries(entries)
	entries = scrambleTransactions(42, entries)
	for i := range entries {
		for j := i + 1; j < len(entries); j++ {
			if entries[i].Nonce() > entries[j].Nonce() {
				t.Errorf("incorrect nonce order %d must be before %d", entries[j].Nonce(), entries[i].Nonce())
			}
		}
	}
}

func TestTxScrambler_SortTransactionsWithSameSender_SortsByGasIfNonceIsSame(t *testing.T) {
	entries := []ScramblerEntry{
		&dummyScramblerEntry{
			hash:     common.Hash{1},
			sender:   common.Address{1},
			nonce:    1,
			gasPrice: big.NewInt(1),
		},
		&dummyScramblerEntry{
			hash:     common.Hash{2},
			sender:   common.Address{1},
			nonce:    1,
			gasPrice: big.NewInt(2),
		},
		&dummyScramblerEntry{
			hash:     common.Hash{3},
			sender:   common.Address{1},
			nonce:    1,
			gasPrice: big.NewInt(3),
		},
		&dummyScramblerEntry{
			hash:     common.Hash{4},
			sender:   common.Address{1},
			nonce:    1,
			gasPrice: big.NewInt(4),
		},
	}

	shuffleEntries(entries)
	entries = scrambleTransactions(42, entries)
	for i := range entries {
		for j := i + 1; j < len(entries); j++ {
			if entries[i].GasPrice().Uint64() < entries[j].GasPrice().Uint64() {
				t.Errorf("incorrect gas price order %d must be before %d", entries[i].GasPrice(), entries[j].GasPrice())
			}
		}
	}
}

func TestTxScrambler_SortTransactionsWithSameSender_SortsByHashIfNonceAndGasIsSame(t *testing.T) {
	entries := []ScramblerEntry{
		&dummyScramblerEntry{
			hash:     common.Hash{1},
			sender:   common.Address{1},
			nonce:    1,
			gasPrice: big.NewInt(1),
		},
		&dummyScramblerEntry{
			hash:     common.Hash{2},
			sender:   common.Address{1},
			nonce:    1,
			gasPrice: big.NewInt(1),
		},
		&dummyScramblerEntry{
			hash:     common.Hash{3},
			sender:   common.Address{1},
			nonce:    1,
			gasPrice: big.NewInt(1),
		},
		&dummyScramblerEntry{
			hash:     common.Hash{4},
			sender:   common.Address{1},
			nonce:    1,
			gasPrice: big.NewInt(1),
		},
	}

	shuffleEntries(entries)
	entries = scrambleTransactions(42, entries)
	for i := range entries {
		for j := i + 1; j < len(entries); j++ {
			if entries[i].Hash().Cmp(entries[j].Hash()) > 0 {
				t.Errorf("incorrect hash order %d must be before %d", entries[i].Hash(), entries[j].Hash())
			}
		}
	}
}

func TestTxScrambler_ScrambleTransactions_ScrambleIsDeterministic(t *testing.T) {
	entries := []ScramblerEntry{
		&dummyScramblerEntry{sender: common.Address{1}, hash: common.Hash{1}},
		&dummyScramblerEntry{sender: common.Address{2}, hash: common.Hash{2}},
		&dummyScramblerEntry{sender: common.Address{3}, hash: common.Hash{3}},
		&dummyScramblerEntry{sender: common.Address{4}, hash: common.Hash{4}},
	}

	entriesCopy := slices.Clone(entries)

	for range 10 {
		entries = scrambleTransactions(42, entries)
		for range 10 {
			shuffleEntries(entriesCopy)
			entriesCopy = scrambleTransactions(42, entriesCopy)
			if slices.CompareFunc(entries, entriesCopy, compareFunc) != 0 {
				t.Fatalf("scramble is not deterministic %v vs %v", entries, entriesCopy)
			}
		}
	}
}

func TestTxScrambler_FilterAndOrderTransactions_SortIsDeterministic_RepeatedData(t *testing.T) {
	tests := []struct {
		name    string
		entries []ScramblerEntry
	}{
		{
			name: "repeated hashes",
			entries: []ScramblerEntry{
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
					sender:   common.Address{3},
					nonce:    3,
					gasPrice: big.NewInt(3),
				},
				&dummyScramblerEntry{
					hash:     common.Hash{2},
					sender:   common.Address{2},
					nonce:    2,
					gasPrice: big.NewInt(2),
				},
				&dummyScramblerEntry{
					hash:     common.Hash{1},
					sender:   common.Address{1},
					nonce:    1,
					gasPrice: big.NewInt(1),
				},
			},
		},
		{
			name: "repeated addresses",
			entries: []ScramblerEntry{
				&dummyScramblerEntry{
					hash:   common.Hash{1},
					sender: common.Address{1},
					nonce:  1,
				},
				&dummyScramblerEntry{
					hash:   common.Hash{2},
					sender: common.Address{2},
					nonce:  2,
				},
				&dummyScramblerEntry{
					hash:   common.Hash{3},
					sender: common.Address{3},
					nonce:  3,
				},
				&dummyScramblerEntry{
					hash:   common.Hash{4},
					sender: common.Address{2},
					nonce:  4,
				},
				&dummyScramblerEntry{
					hash:   common.Hash{5},
					sender: common.Address{1},
					nonce:  5,
				},
			},
		},
		{
			name: "repeated addresses and nonces",
			entries: []ScramblerEntry{
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
					sender:   common.Address{3},
					nonce:    3,
					gasPrice: big.NewInt(3),
				},
				&dummyScramblerEntry{
					hash:     common.Hash{4},
					sender:   common.Address{2},
					nonce:    2,
					gasPrice: big.NewInt(4),
				},
				&dummyScramblerEntry{
					hash:     common.Hash{5},
					sender:   common.Address{1},
					nonce:    1,
					gasPrice: big.NewInt(5),
				},
			},
		},
		{
			name: "repeated addresses, nonces and gas prices",
			entries: []ScramblerEntry{
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
					sender:   common.Address{3},
					nonce:    3,
					gasPrice: big.NewInt(3),
				},
				&dummyScramblerEntry{
					hash:     common.Hash{4},
					sender:   common.Address{2},
					nonce:    2,
					gasPrice: big.NewInt(2),
				},
				&dummyScramblerEntry{
					hash:     common.Hash{5},
					sender:   common.Address{1},
					nonce:    1,
					gasPrice: big.NewInt(1),
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res1 := test.entries
			res2 := slices.Clone(res1)
			// shuffle one array
			shuffleEntries(res2)

			res1 = scrambleTransactions(42, res1)
			res2 = scrambleTransactions(42, res2)
			if slices.CompareFunc(res1, res2, compareFunc) != 0 {
				t.Error("slices have different order - algorithm is not deterministic")
			}
		})
	}
}

func TestTxScrambler_FilterAndOrderTransactions_RandomInput(t *testing.T) {
	// this tests these input sizes:
	// 1, 4, 16, 64, 256, 1024
	for i := int64(1); i <= 1024; i = i * 4 {
		entries := createRandomScramblerEntries(i)
		copy := slices.Clone(entries)
		shuffleEntries(copy)
		entries = scrambleTransactions(42, entries)
		copy = scrambleTransactions(42, copy)
		if slices.CompareFunc(entries, copy, compareFunc) != 0 {
			t.Error("slices have different order - algorithm is not deterministic")
		}
	}
}

// Permutations and pseudo random number generation

func TestPseudoRandomNumberGenerator_OutputIsEquidistributed(t *testing.T) {
	rnd := xorShiftStar{}
	rnd.seed(42)
	n := 1000000

	seenXor := map[uint64]struct{}{}
	seenRand := map[uint64]struct{}{}

	for range n {
		seenXor[rnd.randN(uint64(n))] = struct{}{}
		seenRand[rnd.randN(uint64(n))] = struct{}{}
	}

	if len(seenXor) < len(seenRand)*99/100 {
		t.Fatalf("Homemade rand is not within 1 percent of rand library: %d vs. %d", len(seenXor), len(seenRand))
	}
}
func TestScrambler_shuffleSendersReturnsDifferentPermutations(t *testing.T) {
	n := 8
	numPermutations := fac(n)
	seeds := make([]int, numPermutations)
	for i := range seeds {
		seeds[i] = i
	}

	permutations := map[string]struct{}{}
	for _, seed := range seeds {
		input := []rune{}
		for i := range n {
			input = append(input, rune('A'+i))
		}
		out := shuffleSenders(uint64(seed), input)
		permutations[string(out)] = struct{}{}
	}

	// We don't get all permutations, but we should get at least half of them
	if len(permutations) < numPermutations/2 {
		t.Fatalf("Missing permutations, only got %d of %d: %v", len(permutations), numPermutations, permutations)
	}
}

// Apply permutation
func TestApplyPermutation(t *testing.T) {
	items := make([]int, 1000000)
	for i := range items {
		items[i] = i
	}

	indices := rand.Perm(len(items))
	permutation := applyPermutation(indices, items)

	for i := range items {
		if items[i] != permutation[i] {
			t.Errorf("Incorrect permutation at index %d: %d != %d", i, items[i], permutation[i])
		}
	}
}

// Benchmarks

func BenchmarkTxScrambler(b *testing.B) {
	for size := int64(10); size < 100_000; size *= 10 {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			for i := 1; i <= b.N; i++ {
				entries := createRandomScramblerEntries(size)
				b.StartTimer()
				scrambleTransactions(42, entries)
				b.StopTimer()
			}
		})
	}
}

// helper functions

func createRandomScramblerEntries(size int64) []ScramblerEntry {
	var entries []ScramblerEntry
	for j := int64(0); j < size; j++ {
		// same hashes must have same data
		r := rand.IntN(100 - 1)
		entries = append(entries, &dummyScramblerEntry{
			hash:     common.Hash(uint256.NewInt(uint64(j)).Bytes32()),
			sender:   common.Address{byte(r)},
			nonce:    uint64(r),
			gasPrice: big.NewInt(int64(r)),
		})
	}
	return entries
}

func shuffleEntries(entries []ScramblerEntry) {
	rand.Shuffle(len(entries), func(i, j int) {
		entries[i], entries[j] = entries[j], entries[i]
	})
}

func fac(n int) int {
	if n == 0 {
		return 1
	}
	return n * fac(n-1)
}
