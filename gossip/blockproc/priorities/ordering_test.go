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
	"fmt"
	"math/big"
	"slices"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// fakeClassifier maps transactions (by hash) to priorities, with optional errors.
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
	return zeroPriority(), nil
}

func makeTxN(nonce uint64) *types.Transaction {
	return makeTxG(nonce, 21000)
}

func makeTxG(nonce, gas uint64) *types.Transaction {
	to := common.Address{0xbb}
	return types.NewTransaction(nonce, to, big.NewInt(0), gas, big.NewInt(1), nil)
}

func prio(level, weight int64, id byte) Priority {
	p := Priority{}
	p.Level.SetUint64(uint64(level))
	p.Weight.SetUint64(uint64(weight))
	p.Id[31] = id
	return p
}

func hashes(txs types.Transactions) []common.Hash {
	out := make([]common.Hash, len(txs))
	for i, tx := range txs {
		out[i] = tx.Hash()
	}
	return out
}

func requirePermutation(t *testing.T, got, base types.Transactions) {
	t.Helper()
	require.Len(t, got, len(base))
	gh := hashes(got)
	bh := hashes(base)
	slices.SortFunc(gh, func(a, b common.Hash) int { return bytes.Compare(a[:], b[:]) })
	slices.SortFunc(bh, func(a, b common.Hash) int { return bytes.Compare(a[:], b[:]) })
	require.Equal(t, bh, gh, "result must be a permutation of the input")
}

func TestPrioritize_NoPriorities_IsIdentity(t *testing.T) {
	base := types.Transactions{makeTxN(0), makeTxN(1), makeTxN(2)}
	got := Prioritize(base, fakeClassifier{}, Config{MaxGasPerEntityPerBlock: 1_000_000})
	require.Equal(t, hashes(base), hashes(got))
}

func TestPrioritize_PartitionsByLevelThenWeight(t *testing.T) {
	a, b, c, d := makeTxN(0), makeTxN(1), makeTxN(2), makeTxN(3)
	base := types.Transactions{a, b, c, d}
	cls := fakeClassifier{prio: map[common.Hash]Priority{
		a.Hash(): prio(1, 10, 1), // level 1
		// b is non-prioritized
		c.Hash(): prio(2, 5, 2),  // level 2 (highest -> first)
		d.Hash(): prio(1, 20, 3), // level 1, higher weight than a
	}}
	got := Prioritize(base, cls, Config{MaxGasPerEntityPerBlock: 1_000_000})
	// level 2 first, then level 1 by weight desc (d before a), then non-prio b
	require.Equal(t, hashes(types.Transactions{c, d, a, b}), hashes(got))
	requirePermutation(t, got, base)
}

func TestPrioritize_TieBrokenByHash(t *testing.T) {
	a, b := makeTxN(0), makeTxN(3)
	base := types.Transactions{a, b}
	require.True(t, bytes.Compare(a.Hash().Bytes(), b.Hash().Bytes()) > 0) // initially they are in the wrong order
	cls := fakeClassifier{prio: map[common.Hash]Priority{
		a.Hash(): prio(1, 10, 1),
		b.Hash(): prio(1, 10, 2), // same level+weight, different id
	}}
	got := Prioritize(base, cls, Config{MaxGasPerEntityPerBlock: 1_000_000})

	// Expected order is the two txs sorted by ascending hash.
	want := types.Transactions{a, b}
	if bytes.Compare(a.Hash().Bytes(), b.Hash().Bytes()) > 0 {
		want = types.Transactions{b, a}
	}
	require.Equal(t, hashes(want), hashes(got))
}

func TestPrioritize_RateLimitDemotesExcessToBaseOrder(t *testing.T) {
	a1, x, a2, a3, a4 := makeTxN(0), makeTxN(1), makeTxN(2), makeTxN(3), makeTxN(4)
	base := types.Transactions{a1, x, a2, a3, a4}
	// a1, a2, a3, a4 are all from the same entity (id=1)
	cls := fakeClassifier{prio: map[common.Hash]Priority{
		a1.Hash(): prio(1, 10, 1),
		// x is non-prioritized
		a2.Hash(): prio(1, 30, 1),
		a3.Hash(): prio(2, 20, 1),
		a4.Hash(): prio(2, 0, 1),
	}}
	// Budget for exactly 3 txs of the entity; the 4th would exceed and is demoted.
	got := Prioritize(base, cls, Config{MaxGasPerEntityPerBlock: 3 * 21000})
	// Keep top 3 by level and weight at the front; a1 (demoted) and x keep
	// their base-order positions.
	require.Equal(t, hashes(types.Transactions{a3, a4, a2, a1, x}), hashes(got))
	requirePermutation(t, got, base)
}

func TestPrioritize_GasBudget_PacksManyCheapOrFewExpensive(t *testing.T) {
	// Two entities, each with the same 200_000 gas budget:
	//   - cheap entity (id 1): 3 cheap txs of 50_000 -> all three fit (150_000).
	//   - expensive entity (id 2): 3 txs of 80_000 -> only 2 fit (160_000);
	//     the third would push to 240_000 > 200_000 and is demoted.
	c1, c2, c3 := makeTxG(0, 50_000), makeTxG(1, 50_000), makeTxG(2, 50_000)
	e1, e2, e3 := makeTxG(3, 80_000), makeTxG(4, 80_000), makeTxG(5, 80_000)
	base := types.Transactions{c1, e1, c2, e2, c3, e3}
	cls := fakeClassifier{prio: map[common.Hash]Priority{
		c1.Hash(): prio(1, 30, 1),
		c2.Hash(): prio(1, 20, 1),
		c3.Hash(): prio(1, 10, 1),
		e1.Hash(): prio(1, 30, 2),
		e2.Hash(): prio(1, 20, 2),
		e3.Hash(): prio(1, 10, 2), // this one gets demoted
	}}
	got := Prioritize(base, cls, Config{MaxGasPerEntityPerBlock: 200_000})

	// Prioritized set contains all cheap txs and the top two expensive txs,
	// globally sorted by (level desc, weight desc, hash asc). Within each
	// weight the two entities' txs (cheap vs expensive) tie-break by hash.
	prioritizedIn := func(txs ...*types.Transaction) map[common.Hash]bool {
		m := make(map[common.Hash]bool, len(txs))
		for _, tx := range txs {
			m[tx.Hash()] = true
		}
		return m
	}
	prioritized := prioritizedIn(c1, c2, c3, e1, e2)
	require.Len(t, got, len(base))
	// The first 5 txs must be exactly the prioritized set.
	for i := 0; i < 5; i++ {
		require.True(t, prioritized[got[i].Hash()], "tx at %d must be prioritized", i)
	}
	// e3 is demoted and keeps its base-order position (last).
	require.Equal(t, e3.Hash(), got[5].Hash())
	requirePermutation(t, got, base)
}

func TestPrioritize_ZeroLimit_PrioritizesNothing(t *testing.T) {
	a, b := makeTxN(0), makeTxN(1)
	base := types.Transactions{a, b}
	cls := fakeClassifier{prio: map[common.Hash]Priority{
		a.Hash(): prio(5, 99, 1),
	}}
	got := Prioritize(base, cls, Config{MaxGasPerEntityPerBlock: 0})
	require.Equal(t, hashes(base), hashes(got), "with budget 0 nothing is prioritized")
}

func TestPrioritize_ClassifierError_TreatedAsNotPrioritized(t *testing.T) {
	a, b := makeTxN(0), makeTxN(1)
	base := types.Transactions{a, b}
	// without the error, b would be prioritized and come first
	cls := fakeClassifier{
		prio:  map[common.Hash]Priority{b.Hash(): prio(1, 10, 1)},
		errOn: map[common.Hash]bool{b.Hash(): true}, // error overrides priority
	}
	got := Prioritize(base, cls, Config{MaxGasPerEntityPerBlock: 1_000_000})
	require.Equal(t, hashes(base), hashes(got), "error => not prioritized => identity")
}

// countingSnapshotter records snapshot/revert calls and asserts balance.
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

func TestEvmClassifier_IsolatesEachQueryWithSnapshot(t *testing.T) {
	tx, signer := makeTx(t)
	snap := &countingSnapshotter{}
	vm := &fakeVM{result: make([]byte, 96)}
	cls := NewEvmClassifier(enabledUpgrades(), vm, signer, snap)

	for i := 0; i < 3; i++ {
		_, err := cls.Priority(tx)
		require.NoError(t, err)
	}
	require.Equal(t, 3, snap.snapshots, "one snapshot per query")
	require.Equal(t, 3, snap.reverts, "one revert per query")
}

func TestPrioritize_IsDeterministic(t *testing.T) {
	txs := make(types.Transactions, 0, 20)
	cls := fakeClassifier{prio: map[common.Hash]Priority{}}
	for i := uint64(0); i < 20; i++ {
		tx := makeTxN(i)
		txs = append(txs, tx)
		if i%2 == 0 {
			cls.prio[tx.Hash()] = prio(int64(1+i%3), int64(i), byte(i%4))
		}
	}
	cfg := Config{MaxGasPerEntityPerBlock: 3 * 21000}
	first := Prioritize(slices.Clone(txs), cls, cfg)
	for i := 0; i < 5; i++ {
		again := Prioritize(slices.Clone(txs), cls, cfg)
		require.Equal(t, hashes(first), hashes(again))
	}
	requirePermutation(t, first, txs)
}
