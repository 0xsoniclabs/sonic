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

package scrambler

import (
	"bytes"
	"cmp"
	"math/big"
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// ScramblerEntry stores meta information about transaction for sorting and filtering them.
type ScramblerEntry interface {
	// Hash returns the transaction hash
	Hash() common.Hash
	// Sender returns the sender of the transaction
	Sender() common.Address
	// Nonce returns the transaction nonce
	Nonce() uint64
	// GasPrice returns the transaction gas price
	GasPrice() *big.Int
}

// newScramblerTransaction creates a wrapper around *types.Transaction which implements ScramblerEntry.
func newScramblerTransaction(signer types.Signer, tx *types.Transaction) (ScramblerEntry, error) {
	// if address cannot be derived, it is excluded from address sorting
	sender, err := types.Sender(signer, tx)
	if err != nil {
		return nil, err
	}
	return &scramblerTransaction{
		Transaction: tx,
		sender:      sender,
	}, nil
}

type scramblerTransaction struct {
	*types.Transaction
	sender common.Address
}

func (tx *scramblerTransaction) Sender() common.Address {
	return tx.sender
}

// GetExecutionOrder returns correct order of the transactions.
// If Sonic is enabled, the tx scrambler is used, otherwise the
// order stays unchanged. If signer is unable to derive sender for
// a transaction, this transaction is not excluded from the final list.
func GetExecutionOrder(unorderedTxs types.Transactions, signer types.Signer, isSonic bool) types.Transactions {
	// Don't use scrambler if Sonic is not enabled
	if !isSonic {
		return unorderedTxs
	}

	unorderedEntries := make([]ScramblerEntry, 0, len(unorderedTxs))
	for _, tx := range unorderedTxs {
		entry, err := newScramblerTransaction(signer, tx)
		if err != nil {
			// unable to create entry - skip
			continue
		}
		unorderedEntries = append(unorderedEntries, entry)
	}

	orderedEntries := filterAndOrderTransactions(unorderedEntries)
	orderedTxs := make(types.Transactions, len(orderedEntries))
	for i, tx := range orderedEntries {
		// Cast back the transactions to pass it to the processor
		orderedTxs[i] = tx.(*scramblerTransaction).Transaction
	}

	return orderedTxs
}

// filterAndOrderTransactions first removes any entries with duplicate hashes, then sorts the list by XORed hashes.
// Furthermore, if there are entries with same sender, these entries are sorted by their nonce (lower comes first).
// If nonce from same sender is equal, entries are sorted by gas prices (higher comes first).
// If gas prices from same sender is equal, entries are sorted by their hashes.
func filterAndOrderTransactions(entries []ScramblerEntry) []ScramblerEntry {
	uniqueList, salt, hasDuplicatedAddresses := analyseEntryList(entries)
	scrambleTransactions(uniqueList, salt)
	// do we need to sort more?
	if hasDuplicatedAddresses {
		sortTransactionsWithSameSender(uniqueList)
	}
	return uniqueList
}

// sortTransactionsWithSameSender finds any duplicate senders and sorts their transactions by nonce ascending.
func sortTransactionsWithSameSender(entries []ScramblerEntry) {
	senderNonceOrder := slices.Clone(entries)
	// sort copied slice so that it has all txs from same address together + sorted by nonce ascending
	slices.SortFunc(senderNonceOrder, func(a, b ScramblerEntry) int {
		res := a.Sender().Cmp(b.Sender())
		if res != 0 {
			return res
		}
		// if addresses are same, sort by nonce
		res = cmp.Compare(a.Nonce(), b.Nonce())
		if res != 0 {
			return res
		}
		// if nonce is same, sort by gas price
		res = b.GasPrice().Cmp(a.GasPrice())
		if res != 0 {
			return res
		}
		// if both nonce and gas prices are equal, sort by hash
		// note: at this point, hashes can never be same - duplicates are removed
		return a.Hash().Cmp(b.Hash())
	})
	// find the first entry for each sender in the senderNonceOrder
	senderIndex := make(map[common.Address]int)
	for idx, entry := range senderNonceOrder {
		sender := entry.Sender()
		if _, found := senderIndex[sender]; !found {
			senderIndex[sender] = idx
		}
	}
	// replace already scrambled entries so that they are sorted by nonce
	for idx := range entries {
		sender := entries[idx].Sender()
		entries[idx] = senderNonceOrder[senderIndex[sender]]
		senderIndex[sender]++
	}
}

// scrambleTransactions scrambles transactions by comparing its XORed hashes with salt
func scrambleTransactions(list []ScramblerEntry, salt [32]byte) {
	var aX, bX [32]byte
	slices.SortFunc(list, func(a, b ScramblerEntry) int {
		aX = xorBytes32(a.Hash(), salt)
		bX = xorBytes32(b.Hash(), salt)
		return bytes.Compare(aX[:], bX[:])
	})
}

// analyseEntryList removes any transactions with duplicate hashes and creates the XOR salt from the unique tx list.
// Furthermore, it returns whether given list of entries contains duplicate addresses.
func analyseEntryList(entries []ScramblerEntry) ([]ScramblerEntry, [32]byte, bool) {
	var (
		salt                  [32]byte
		hasDuplicateAddresses bool
	)
	seenHashes := make(map[common.Hash]struct{})
	seenAddresses := make(map[common.Address]struct{})
	uniqueList := make([]ScramblerEntry, 0, len(entries))
	for _, entry := range entries {
		// skip any duplicate hashes
		if _, ok := seenHashes[entry.Hash()]; ok {
			continue
		}
		// mark whether we have duplicate addresses
		sender := entry.Sender()
		if _, ok := seenAddresses[sender]; ok {
			hasDuplicateAddresses = true
		}
		seenAddresses[sender] = struct{}{}
		salt = xorBytes32(salt, entry.Hash())
		uniqueList = append(uniqueList, entry)
		seenHashes[entry.Hash()] = struct{}{}

	}

	return uniqueList, salt, hasDuplicateAddresses
}

func xorBytes32(a, b [32]byte) (dst [32]byte) {
	for i := 0; i < 32; i++ {
		dst[i] = a[i] ^ b[i]
	}
	return
}
