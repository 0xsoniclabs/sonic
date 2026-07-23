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

package priorities

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/big"
	"slices"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/priorities/registry"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestEvmClassifier_Priority_IsolatesEachQueryWithSnapshot(t *testing.T) {
	tx, signer := makeTx(t)
	snap := &countingSnapshotter{}
	vm := &fakeVM{result: make([]byte, 96)}
	cls := NewEvmClassifier(enabledUpgrades(), vm, signer, snap)

	calls := 3
	for i := 0; i < calls; i++ {
		_, err := cls.Priority(tx)
		require.NoError(t, err)
	}
	require.Equal(t, calls, snap.snapshots, "one snapshot per query")
	require.Equal(t, calls, snap.reverts, "one revert per query")
}

func TestEvmClassifier_Priority_RevertsSnapshotOnError(t *testing.T) {
	tx, signer := makeTx(t)
	snap := &countingSnapshotter{}
	vm := &fakeVM{err: fmt.Errorf("boom")}
	cls := NewEvmClassifier(enabledUpgrades(), vm, signer, snap)

	_, err := cls.Priority(tx)
	require.Error(t, err)
	require.Equal(t, 1, snap.snapshots)
	require.Equal(t, 1, snap.reverts)
}

func TestEvmClassifier_Priority_DelegatesToGetPriority(t *testing.T) {
	tx, signer := makeTx(t)
	id := [16]byte{0xde, 0xad}
	result := make([]byte, 96)
	binary.BigEndian.PutUint64(result[24:32], 3)
	binary.BigEndian.PutUint64(result[56:64], 5)
	copy(result[80:96], id[:])
	vm := &fakeVM{result: result}
	cls := NewEvmClassifier(enabledUpgrades(), vm, signer, &countingSnapshotter{})

	p, err := cls.Priority(tx)
	require.NoError(t, err)
	require.Equal(t, uint32(registry.GetPriorityFunctionSelector), binary.BigEndian.Uint32(vm.gotIn[0:4]))
	require.Equal(t, Priority{Level: 3, Weight: 5, ID: id}, p)
}

func TestTransactionWithPriority_Cmp_ComparesByLevelDescWeightDescHashAsc(t *testing.T) {
	lowHash, highHash := makeTxWithNonce(0), makeTxWithNonce(1)
	if bytes.Compare(lowHash.Hash().Bytes(), highHash.Hash().Bytes()) > 0 {
		lowHash, highHash = highHash, lowHash
	}
	tx := makeTxWithNonce(0)

	cases := map[string]struct {
		a         transactionWithPriority
		b         transactionWithPriority
		expectCmp int
	}{
		"higher level wins": {
			transactionWithPriority{tx, Prio(2, 10, 100)},
			transactionWithPriority{tx, Prio(1, 10, 100)},
			1,
		},
		"lower level loses": {
			transactionWithPriority{tx, Prio(1, 10, 100)},
			transactionWithPriority{tx, Prio(2, 10, 100)},
			-1,
		},
		"same level higher weight wins": {
			transactionWithPriority{tx, Prio(1, 20, 100)},
			transactionWithPriority{tx, Prio(1, 10, 100)},
			1,
		},
		"same level lower weight loses": {
			transactionWithPriority{tx, Prio(1, 10, 100)},
			transactionWithPriority{tx, Prio(1, 20, 100)},
			-1,
		},
		"same level and weight lower hash wins": {
			transactionWithPriority{lowHash, Prio(1, 10, 100)},
			transactionWithPriority{highHash, Prio(1, 10, 100)},
			1,
		},
		"same level and weight higher hash loses": {
			transactionWithPriority{highHash, Prio(1, 10, 100)},
			transactionWithPriority{lowHash, Prio(1, 10, 100)},
			-1,
		},
		"same level and weight and hash is equal": {
			transactionWithPriority{tx, Prio(1, 10, 100)},
			transactionWithPriority{tx, Prio(1, 10, 200)}, // entity does not matter
			0,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.expectCmp, tc.a.cmpLevelWeightHash(tc.b))
		})
	}
}

func TestTransactionWithPriority_CmpNonceHash_ComparesByNonceAscHashAsc(t *testing.T) {
	// Same nonce, different hash
	lowHash, highHash := makeTxWithNonceAndGas(0, 21000), makeTxWithNonceAndGas(0, 22000)
	if bytes.Compare(lowHash.Hash().Bytes(), highHash.Hash().Bytes()) > 0 {
		lowHash, highHash = highHash, lowHash
	}
	prio := Prio(1, 10, 100)

	cases := map[string]struct {
		a         transactionWithPriority
		b         transactionWithPriority
		expectCmp int
	}{
		"lower nonce wins": {
			transactionWithPriority{makeTxWithNonce(0), prio},
			transactionWithPriority{makeTxWithNonce(1), prio},
			-1,
		},
		"higher nonce loses": {
			transactionWithPriority{makeTxWithNonce(1), prio},
			transactionWithPriority{makeTxWithNonce(0), prio},
			1,
		},
		"same nonce lower hash wins": {
			transactionWithPriority{lowHash, prio},
			transactionWithPriority{highHash, prio},
			-1,
		},
		"same nonce higher hash loses": {
			transactionWithPriority{highHash, prio},
			transactionWithPriority{lowHash, prio},
			1,
		},
		"same nonce and hash is equal": {
			transactionWithPriority{lowHash, prio},
			transactionWithPriority{lowHash, Prio(2, 20, 200)}, // prio does not matter
			0,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.expectCmp, tc.a.cmpNonceHash(tc.b))
		})
	}
}

func TestPrioritize_NoPriorities_IsIdentity(t *testing.T) {
	base := types.Transactions{makeTxWithNonce(0), makeTxWithNonce(1), makeTxWithNonce(2)}
	// No transaction is prioritized, so the signer and nonce reader are unused.
	got := Prioritize(base, fakeClassifier{}, fakeSigner{}, fakeNonceReader{}, Config{MaxGasPerEntityPerBlock: 1_000_000})
	require.Equal(t, hashes(base), hashes(got))
}

// TestPrioritize_EndToEnd exercises the four helpers together: classification,
// per-sender nonce sequences, the budgeted prefix, and the remainder
// recombination.
// Each entity has its own budget of two 21k-gas txs (42000):
//   - A (entity 100): a0, a1 prioritized and contiguous -> both in prefix.
//   - B (entity 101): b0 is level 2 -> first in prefix.
//   - C (entity 102): c is prioritized but nonce-gapped -> demoted.
//   - D (entity 103): d0, d1, d2 prioritized; the budget demotes d2.
//   - x: not prioritized.
func TestPrioritize_EndToEnd(t *testing.T) {
	a0, a1 := makeTxWithNonce(0), makeTxWithNonce(1)
	b0 := makeTxWithNonce(5)
	c := makeTxWithNonce(7)
	d0, d1, d2 := makeTxWithNonce(2), makeTxWithNonce(3), makeTxWithNonce(4)
	x := makeTxWithNonce(8)
	base := types.Transactions{a0, a1, b0, c, d0, d1, d2, x}
	cls := fakeClassifier{prio: map[common.Hash]Priority{
		a0.Hash(): Prio(1, 10, 100), a1.Hash(): Prio(1, 50, 100),
		b0.Hash(): Prio(2, 20, 101),
		c.Hash():  Prio(1, 90, 102),
		d0.Hash(): Prio(1, 40, 103), d1.Hash(): Prio(1, 30, 103), d2.Hash(): Prio(1, 20, 103),
	}}
	A, B, C, D, X := common.Address{0xa}, common.Address{0xb}, common.Address{0xc}, common.Address{0xd}, common.Address{0x9}
	signer := fakeSigner{sender: map[common.Hash]common.Address{
		a0.Hash(): A, a1.Hash(): A,
		b0.Hash(): B,
		c.Hash():  C,
		d0.Hash(): D, d1.Hash(): D, d2.Hash(): D,
		x.Hash(): X,
	}}
	nonceReader := fakeNonceReader{A: 0, B: 5, C: 6, D: 2, X: 8}
	got := Prioritize(base, cls, signer, nonceReader, Config{MaxGasPerEntityPerBlock: 2 * 21000})
	// prefix: b0 (level 2), then d0, d1 (entity 4 budget), then a0, a1;
	// remainder in base order: c (nonce-gapped), d2 (over budget), x (not
	// prioritized).
	require.Equal(t, hashes(types.Transactions{b0, d0, d1, a0, a1, c, d2, x}), hashes(got))
}

func TestClassify_PairsTransactionsWithPriorityTreatingErrorsAsNotPrioritized(t *testing.T) {
	a, b, c := makeTxWithNonce(0), makeTxWithNonce(1), makeTxWithNonce(2)
	cls := fakeClassifier{
		prio: map[common.Hash]Priority{
			a.Hash(): Prio(1, 10, 100),
			b.Hash(): Prio(2, 20, 101),
		},
		errOn: map[common.Hash]bool{
			b.Hash(): true, // error overrides the priority
		},
	}
	got := classify(types.Transactions{a, b, c}, cls)
	require.Equal(t, []transactionWithPriority{
		{a, Prio(1, 10, 100)},
		{b, Priority{}}, // error => not prioritized
		{c, Priority{}}, // unclassified => not prioritized
	}, got)
}

func TestPrioritizedSenderSequences_ReducesToContiguousNonceSequenceOfPrioTxsFromAccountNonce(t *testing.T) {
	type tx struct {
		nonce  uint64
		level  uint64 // 0 = not prioritized
		sender byte
	}
	tests := map[string]struct {
		txs         []tx
		startNonces map[byte]uint64 // per-sender block-start nonce
		// per-sender surviving entry indices, in nonce order
		expectedSequences map[byte][]int
	}{
		"non-prioritized skipped": {
			txs:               []tx{{nonce: 0, level: 0, sender: 0xa}},
			startNonces:       map[byte]uint64{0xa: 0},
			expectedSequences: map[byte][]int{},
		},
		"stale nonce demoted": {
			txs:               []tx{{nonce: 3, level: 1, sender: 0xa}},
			startNonces:       map[byte]uint64{0xa: 5},
			expectedSequences: map[byte][]int{},
		},
		"nonce gap demoted": {
			txs:               []tx{{nonce: 2, level: 1, sender: 0xa}},
			startNonces:       map[byte]uint64{0xa: 0},
			expectedSequences: map[byte][]int{},
		},
		"stale lower nonce keeps the next": {
			txs:               []tx{{nonce: 0, level: 1, sender: 0xa}, {nonce: 1, level: 1, sender: 0xa}},
			startNonces:       map[byte]uint64{0xa: 1},
			expectedSequences: map[byte][]int{0xa: {1}},
		},
		"gap after sequence demotes the tail": {
			// nonce 1 not prioritized breaks the sequence before nonce 2.
			txs:               []tx{{nonce: 0, level: 1, sender: 0xa}, {nonce: 1, level: 0, sender: 0xa}, {nonce: 2, level: 1, sender: 0xa}},
			startNonces:       map[byte]uint64{0xa: 0},
			expectedSequences: map[byte][]int{0xa: {0}},
		},
		"contiguous sequence kept": {
			txs:               []tx{{nonce: 0, level: 1, sender: 0xa}, {nonce: 1, level: 1, sender: 0xa}, {nonce: 2, level: 1, sender: 0xa}},
			startNonces:       map[byte]uint64{0xa: 0},
			expectedSequences: map[byte][]int{0xa: {0, 1, 2}},
		},
		"sequence ordered by nonce not input order": {
			txs:               []tx{{nonce: 1, level: 1, sender: 0xa}, {nonce: 0, level: 1, sender: 0xa}},
			startNonces:       map[byte]uint64{0xa: 0},
			expectedSequences: map[byte][]int{0xa: {1, 0}},
		},
		"per sender independent": {
			txs:               []tx{{nonce: 1, level: 1, sender: 0xa}, {nonce: 2, level: 1, sender: 0xb}},
			startNonces:       map[byte]uint64{0xa: 0, 0xb: 2},
			expectedSequences: map[byte][]int{0xb: {1}},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			entries := make([]transactionWithPriority, len(tc.txs))
			signer := fakeSigner{sender: map[common.Hash]common.Address{}}
			nonceReader := fakeNonceReader{}
			for i, spec := range tc.txs {
				txn := makeTxWithNonce(spec.nonce)
				entries[i] = transactionWithPriority{txn, Prio(spec.level, 10, 100)}
				addr := common.Address{spec.sender}
				signer.sender[txn.Hash()] = addr
				nonceReader[addr] = tc.startNonces[spec.sender]
			}
			want := map[common.Address][]int{}
			for s, idxs := range tc.expectedSequences {
				want[common.Address{s}] = idxs
			}
			require.Equal(t, want, prioritizedSenderSequences(entries, signer, nonceReader))
		})
	}
}

func TestPrioritizedSenderSequences_SameNonceDemotesAllButLowestHash(t *testing.T) {
	txs := []*types.Transaction{makeTxWithNonceAndGas(0, 21000), makeTxWithNonceAndGas(0, 22000), makeTxWithNonceAndGas(0, 23000)}
	slices.SortFunc(txs, func(a, b *types.Transaction) int {
		return bytes.Compare(a.Hash().Bytes(), b.Hash().Bytes())
	})
	// the lowest hash (txs[0]) is at index 1
	entries := []transactionWithPriority{{txs[2], Prio(1, 10, 100)}, {txs[0], Prio(1, 10, 100)}, {txs[1], Prio(1, 10, 100)}}
	signer := fakeSigner{sender: map[common.Hash]common.Address{
		txs[0].Hash(): {0xa}, txs[1].Hash(): {0xa}, txs[2].Hash(): {0xa},
	}}
	nonceReader := fakeNonceReader{{0xa}: 0}
	require.Equal(t, map[common.Address][]int{{0xa}: {1}}, prioritizedSenderSequences(entries, signer, nonceReader))
}

func TestPrioritizedSenderSequences_SenderExtractionFailureDemotesTransaction(t *testing.T) {
	bad, good := makeTxWithNonce(0), makeTxWithNonce(1)
	entries := []transactionWithPriority{{bad, Prio(1, 10, 100)}, {good, Prio(1, 10, 100)}}
	signer := fakeSigner{
		sender: map[common.Hash]common.Address{good.Hash(): {0xa}},
		errOn:  map[common.Hash]bool{bad.Hash(): true},
	}
	nonceReader := fakeNonceReader{{0xa}: 1}
	require.Equal(t, map[common.Address][]int{{0xa}: {1}}, prioritizedSenderSequences(entries, signer, nonceReader))
}

func TestComputePrioritizedTxsPrefix_SelectsByPriorityRespectingNonceOrderUnderPerEntityBudget(t *testing.T) {
	tests := map[string]struct {
		entries  []transactionWithPriority
		bySender map[common.Address][]int
		budget   uint64
		want     []int
	}{
		"single sender kept in nonce order despite ascending priorities": {
			// 3 transactions from same sender with ascending nonces and priorities.
			// nonce order is respected and higher-priority transactions do not jump ahead.
			entries: []transactionWithPriority{
				{makeTxWithNonce(0), Prio(1, 10, 100)},
				{makeTxWithNonce(1), Prio(2, 20, 100)},
				{makeTxWithNonce(2), Prio(3, 30, 100)},
			},
			bySender: map[common.Address][]int{{0xa}: {0, 1, 2}},
			budget:   1_000_000,
			want:     []int{0, 1, 2},
		},
		"orders by level then weight": {
			// three transactions from different entities (100, 101, 102) with
			// different priorities
			entries: []transactionWithPriority{
				{makeTxWithNonce(0), Prio(1, 20, 100)},
				{makeTxWithNonce(1), Prio(2, 10, 101)},
				{makeTxWithNonce(2), Prio(1, 30, 102)},
			},
			bySender: map[common.Address][]int{{0xa}: {0}, {0xb}: {1}, {0xc}: {2}},
			budget:   1_000_000,
			want:     []int{1, 2, 0},
		},
		"per-entity budget demotes excess": {
			// three transactions from the same entity (100) but different
			// senders; the budget fits two
			entries: []transactionWithPriority{
				{makeTxWithNonce(0), Prio(1, 10, 100)},
				{makeTxWithNonce(1), Prio(1, 30, 100)},
				{makeTxWithNonce(2), Prio(1, 20, 100)},
			},
			bySender: map[common.Address][]int{{0xa}: {0}, {0xb}: {1}, {0xc}: {2}},
			budget:   2 * 21000,
			want:     []int{1, 2},
		},
		"over-budget frontier blocks sender chain not the entity": {
			// one entity, two senders: A's 100k frontier busts the 50k budget,
			// demoting its whole chain (even the cheap index 1); B still fits.
			entries: []transactionWithPriority{
				{makeTxWithNonceAndGas(0, 100_000), Prio(1, 20, 100)},
				{makeTxWithNonceAndGas(1, 10_000), Prio(1, 10, 100)},
				{makeTxWithNonceAndGas(0, 10_000), Prio(1, 5, 100)},
			},
			bySender: map[common.Address][]int{{0xa}: {0, 1}, {0xb}: {2}},
			budget:   50_000,
			want:     []int{2},
		},
		"frontier follows nonce order within a sender": {
			// 4 transactions from 2 entities and senders
			// the next tx is only picked from the lowest nonce frontier
			// pick 1: choose between index 0 and 2 -> pick 2
			// pick 2: choose between index 0 and 3 -> pick 0
			// pick 3: choose between index 1 and 3 -> pick 1
			// pick 4: only index 3 remaining       -> pick 3
			entries: []transactionWithPriority{
				{makeTxWithNonce(0), Prio(1, 20, 100)},
				{makeTxWithNonce(1), Prio(1, 40, 100)},
				{makeTxWithNonce(0), Prio(1, 30, 101)},
				{makeTxWithNonce(1), Prio(1, 10, 101)},
			},
			bySender: map[common.Address][]int{{0xa}: {0, 1}, {0xb}: {2, 3}},
			budget:   1_000_000,
			want:     []int{2, 0, 1, 3},
		},
		"zero budget selects nothing": {
			entries:  []transactionWithPriority{{makeTxWithNonce(0), Prio(5, 99, 100)}},
			bySender: map[common.Address][]int{{0xa}: {0}},
			budget:   0,
			want:     []int{},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.want, computePrioritizedTxsPrefix(tc.entries, tc.bySender, tc.budget))
		})
	}
}

func TestComputePrioritizedTxsPrefix_BreaksLevelWeightTiesByHashAscending(t *testing.T) {
	lowHash, highHash := makeTxWithNonce(0), makeTxWithNonce(1)
	if bytes.Compare(lowHash.Hash().Bytes(), highHash.Hash().Bytes()) > 0 {
		lowHash, highHash = highHash, lowHash
	}

	entries := []transactionWithPriority{{highHash, Prio(1, 10, 100)}, {lowHash, Prio(1, 10, 101)}}
	bySender := map[common.Address][]int{{0xa}: {0}, {0xb}: {1}}
	require.Equal(t, []int{1, 0}, computePrioritizedTxsPrefix(entries, bySender, 1_000_000))
}

func TestComputePrioritizedTxsPrefix_PacksManyCheapOrFewExpensivePerEntity(t *testing.T) {
	// entity 1: three 50k-gas txs -> all fit (150k). entity 2: three 80k-gas txs
	// -> only two fit (160k); the third exceeds 200k and is demoted.
	entries := []transactionWithPriority{
		{makeTxWithNonceAndGas(0, 50_000), Prio(1, 30, 100)},
		{makeTxWithNonceAndGas(1, 50_000), Prio(1, 20, 100)},
		{makeTxWithNonceAndGas(2, 50_000), Prio(1, 10, 100)},
		{makeTxWithNonceAndGas(3, 80_000), Prio(1, 30, 101)},
		{makeTxWithNonceAndGas(4, 80_000), Prio(1, 20, 101)},
		{makeTxWithNonceAndGas(5, 80_000), Prio(1, 10, 101)},
	}
	bySender := map[common.Address][]int{{0x1}: {0, 1, 2}, {0x2}: {3, 4, 5}}
	got := computePrioritizedTxsPrefix(entries, bySender, 200_000)
	// index 5 (third expensive) demoted
	require.ElementsMatch(t, []int{0, 1, 2, 3, 4}, got)
}

func TestCombinePrioPrefixWithRemainder_PrefixThenRemainderInBaseOrder(t *testing.T) {
	t0, t1, t2, t3 := makeTxWithNonce(0), makeTxWithNonce(1), makeTxWithNonce(2), makeTxWithNonce(3)
	entries := []transactionWithPriority{{t0, Priority{}}, {t1, Priority{}}, {t2, Priority{}}, {t3, Priority{}}}
	tests := map[string]struct {
		prefix []int
		want   types.Transactions
	}{
		"empty prefix is identity": {
			nil,
			types.Transactions{t0, t1, t2, t3},
		},
		"prefix in order then remainder": {
			[]int{2, 0},
			types.Transactions{t2, t0, t1, t3},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, hashes(tc.want), hashes(combinePrioPrefixWithRemainder(entries, tc.prefix)))
		})
	}
}

// fakeClassifier maps transactions (by hash) to priorities, with optional
// errors.
type fakeClassifier struct {
	prio  map[common.Hash]Priority
	errOn map[common.Hash]bool
}

func (c fakeClassifier) Priority(tx *types.Transaction) (Priority, error) {
	if c.errOn[tx.Hash()] {
		return Priority{}, fmt.Errorf("boom")
	}
	if p, ok := c.prio[tx.Hash()]; ok {
		return p, nil
	}
	return Priority{}, nil
}

// fakeSigner recovers a per-transaction sender from a hash map, standing in for
// a real signer over unsigned test transactions. Hashes in errOn, and hashes
// absent from the map, fail recovery.
type fakeSigner struct {
	types.Signer
	sender map[common.Hash]common.Address
	errOn  map[common.Hash]bool
}

func (s fakeSigner) Sender(tx *types.Transaction) (common.Address, error) {
	if s.errOn[tx.Hash()] {
		return common.Address{}, fmt.Errorf("cannot recover sender")
	}
	addr, ok := s.sender[tx.Hash()]
	if !ok {
		return common.Address{}, fmt.Errorf("unknown sender for tx %s", tx.Hash())
	}
	return addr, nil
}

func (fakeSigner) Equal(types.Signer) bool { return false }

// fakeNonceReader is a NonceReader backed by per-address block-start account
// nonces.
type fakeNonceReader map[common.Address]uint64

func (f fakeNonceReader) GetNonce(sender common.Address) uint64 { return f[sender] }

// countingSnapshotter records snapshot/revert calls.
type countingSnapshotter struct {
	snapshots int
	reverts   int
}

func (c *countingSnapshotter) Snapshot() int {
	c.snapshots++
	return c.snapshots
}

func (c *countingSnapshotter) RevertToSnapshot(int) {
	c.reverts++
}

// makeTxWithNonce creates a transaction with the given nonce and 21k gas.
func makeTxWithNonce(nonce uint64) *types.Transaction {
	return makeTxWithNonceAndGas(nonce, 21000)
}

// makeTxWithNonceAndGas creates a transaction with the given nonce and gas.
func makeTxWithNonceAndGas(nonce, gas uint64) *types.Transaction {
	to := common.Address{0xbb}
	return types.NewTransaction(nonce, to, big.NewInt(0), gas, big.NewInt(1), nil)
}

// hashes collects the hashes of the transactions.
func hashes(txs types.Transactions) []common.Hash {
	out := make([]common.Hash, len(txs))
	for i, tx := range txs {
		out[i] = tx.Hash()
	}
	return out
}
