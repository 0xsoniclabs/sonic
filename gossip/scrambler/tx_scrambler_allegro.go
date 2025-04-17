package scrambler

import (
	"cmp"
	"maps"
	"slices"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum/go-ethereum/core/types"
)

// Scramble takes a list of transactions and a seed and scrambles the input
// list of transactions. The scrambling is done in a way that is deterministic
// and can be reproduced by using the same seed. The transactions are grouped
// by sender and sorted by nonce, gas price and hash. The senders are then
// shuffled using the seed and the transactions are reordered according to
// the shuffled senders.
func Scramble(transactions []*types.Transaction, seed uint64, signer types.Signer) {
	// Convert transactions to scrambler entries
	entries := convertToScramblerEntry(transactions, signer)

	// Get scrambled order
	permutation := scramblePermutation(entries, seed)

	// Apply permutation to transactions
	// This is done in place to avoid copying the transactions
	applyPermutation(transactions, permutation)
}

// convertToScramblerEntry converts a list of transactions to a list of
// scrambler entries. The scrambler entry contains the fields required for
// scrambling the transactions.
func convertToScramblerEntry(transactions []*types.Transaction, signer types.Signer) []ScramblerEntry {
	entries := make([]ScramblerEntry, 0, len(transactions))
	for _, tx := range transactions {
		sender, err := types.Sender(signer, tx)
		if err != nil {
			continue
		}

		entry := &scramblerTransaction{
			Transaction: tx,
			sender:      sender,
		}
		entries = append(entries, entry)
	}
	return entries
}

// scramblePermutation takes a seed and a list of transactions and returns
// a permutation of the transactions. The permutation is done in a way that
// is deterministic and can be reproduced by using the same seed.
func scramblePermutation(transactions []ScramblerEntry, seed uint64) []int {
	// Group transactions by sender
	sendersTransactions := map[common.Address][]int{}
	for idx, tx := range transactions {
		sendersTransactions[tx.Sender()] = append(sendersTransactions[tx.Sender()], idx)
	}

	// Sort transactions by nonce, gas price and hash
	for _, txs := range sendersTransactions {
		// Sort by nonce
		slices.SortFunc(txs, func(idxA, idxB int) int {
			a := transactions[idxA]
			b := transactions[idxB]
			res := cmp.Compare(a.Nonce(), b.Nonce())
			if res != 0 {
				return res
			}
			// if nonce is same, sort by gas price
			res = b.GasPrice().Cmp(a.GasPrice())
			if res != 0 {
				return res
			}
			return a.Hash().Cmp(b.Hash())
		})
	}

	senders := slices.Collect(maps.Keys(sendersTransactions))
	// Sort senders by address, so that the shuffle is deterministic
	slices.SortFunc(senders, func(a, b common.Address) int {
		return a.Cmp(b)
	})
	// Shuffle senders
	senders = scrambleEntries(senders, seed)

	// Save permutation of transactions
	permutation := make([]int, 0, len(transactions))
	for _, sender := range senders {
		permutation = append(permutation, sendersTransactions[sender]...)
	}

	return permutation
}

// scrambleEntries shuffles the entries in place using the given seed.
// It uses the Fisher-Yates shuffle algorithm to ensure that the shuffle is
// uniform and unbiased.
func scrambleEntries[T any](entries []T, seed uint64) []T {
	randomGenerator := randomGenerator{}
	randomGenerator.seed(seed)
	for i := len(entries) - 1; i > 0; i-- {
		j := randomGenerator.randN(uint64(i + 1))
		entries[i], entries[j] = entries[j], entries[i]
	}

	return entries
}

// applyPermutation applies a permutation in place.
// Note that also the indices are modified
// based on https://devblogs.microsoft.com/oldnewthing/20170102-00/?p=95095
func applyPermutation[T any](items []T, indices []int) {
	for idx := range items {
		current := idx
		for {
			if idx == indices[current] {
				break
			}
			next := indices[current]
			items[current], items[next] = items[next], items[current]
			indices[current] = current
			current = next
		}
		indices[current] = current
	}
}
