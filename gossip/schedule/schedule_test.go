package schedule

import (
	"reflect"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPartition_Examples_ProduceExpectedPartition(t *testing.T) {
	tests := map[string]struct {
		transactions []transaction
		partitions   [][]int
	}{
		"empty": {
			[]transaction{},
			[][]int{},
		},
		"single": {
			[]transaction{
				tx(a(0, 0)),
			},
			[][]int{{0}},
		},
		"independent": {
			[]transaction{
				tx(a(0, 0)),
				tx(a(1, 0)),
			},
			[][]int{{0}, {1}},
		},
		"dependent": {
			[]transaction{
				tx(a(0, 0)),
				tx(a(0, 1)),
			},
			[][]int{{0, 1}},
		},
		"chain": {
			[]transaction{
				tx(a(0, 0), a(1, 0)),
				tx(a(1, 1), a(2, 0)),
				tx(a(2, 1), a(3, 0)),
				tx(a(3, 1), a(4, 0)),
			},
			[][]int{{0, 1, 2, 3}},
		},
		"two chains": {
			[]transaction{
				tx(a(0, 0), a(1, 0)),
				tx(a(7, 1), a(8, 0)),
				tx(a(1, 1), a(2, 0)),
				tx(a(8, 1), a(9, 0)),
			},
			[][]int{{0, 2}, {1, 3}},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			got := partition(tt.transactions)
			require.Equal(len(tt.partitions), len(got))

			// convert the partition into indices to make comparison easier
			res := [][]int{}
			for _, transactions := range got {
				res = append(res, toIndices(t, tt.transactions, transactions))
			}

			// the result can be in an arbitrary order, so we need to sort the
			// partitions before comparing
			for i := range res {
				slices.Sort(res[i])
			}
			require.ElementsMatch(tt.partitions, res)
		})
	}
}

// --- Sorting ---

func TestSort_Examples(t *testing.T) {

	tests := map[string]struct {
		transactions []transaction
		result       []int
	}{
		"empty": {},
		"only one transaction": {
			[]transaction{
				tx(a(0, 0)),
			},
			[]int{0},
		},
		"identifies authorizations as an enabler": {
			[]transaction{
				tx(a(10, 2)),
				tx(a(12, 1), a(10, 1)),
			},
			[]int{1, 0},
		},
		"identifies authorizations as a disabler": {
			[]transaction{
				tx(a(10, 1)),
				tx(a(12, 1), a(10, 1)),
			},
			[]int{0, 1},
		},
		"delays authentications if another transaction could enable it": {
			[]transaction{
				tx(a(10, 1), a(12, 2)),
				tx(a(12, 1)),
			},
			[]int{1, 0},
		},
		"favor longer authentication lists in conflict cases": {
			[]transaction{
				// Only one of these transactions can succeed, while the other
				// one is turned obsolete. In such cases, the one with the
				// longer list of authorizations should be favored.
				tx(a(10, 1), a(12, 1)),
				tx(a(12, 1), a(10, 1), a(14, 2)),
			},
			[]int{1},
		},
		"authentication should not be processed if it blocks a transaction": {
			[]transaction{
				// both are ready, neither has all authorizations ready to be
				// processed, but the execution of the second would prevent the
				// first from being processed. Thus, the first should be
				// processed first.
				tx(a(10, 1), a(10, 3)),
				tx(a(12, 1), a(10, 1)),
			},
			[]int{0, 1},
		},
		"self-authorization is not a transaction blocker": {
			[]transaction{
				tx(a(10, 1), a(10, 2)),
			},
			[]int{0},
		},
		"unreachable pending is not blocking executable transactions": {
			[]transaction{
				tx(a(10, 3)), // < unreachable
				tx(a(12, 1), a(10, 3)),
				tx(a(10, 1), a(12, 1)), // should be after the previous one
			},
			[]int{1, 2},
		},
		"inactive authorizations should be ignored": {
			// The second transaction should not be prevented from being
			// processed first due to its first authorization which, if
			// processed correctly, would invalidate the third transaction.
			// Since initial the account 10 has a nonce of 1, the authorizations
			// of the second transaction have no effect. Thus, it should be
			// processed first.
			[]transaction{
				tx(a(10, 1), a(12, 1)),
				tx(a(12, 1), a(10, 2), a(10, 3)),
				tx(a(10, 2)),
			},
			[]int{1, 0, 2},
		},
		/*
			// Something not supported:
			"deliberately fail authorization to enable additional transaction": {
				[]transaction{
					tx(a(10, 1)),
					tx(a(12, 1), a(10, 2)),
					tx(a(14, 1), a(10, 3)), // < should be processed before the 2nd to avoid blocking the 4th
					tx(a(10, 3), a(14, 1)),
				},
				[]int{0, 2, 1, 3},
			},
		*/
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)

			// make sure the test examples contain connected transactions
			parts := partition(tt.transactions)
			require.LessOrEqual(len(parts), 1)

			initial := makeFakeNonceProvider(tt.transactions)

			got := sort(tt.transactions, state(initial))
			res := toIndices(t, tt.transactions, got)
			require.Equal(tt.result, res)
		})
	}
}

// --- Interleave ---

func TestInterleave_PreservesRelativeOrder(t *testing.T) {

	partitions := [][][]int{
		{{0, 1, 2, 3}},
		{{0, 1}, {2, 3}},
		{{0, 1}, {2}, {3}},
		{{0}, {1}, {2}, {3}},
	}

	for _, partition := range partitions {
		for seed := range uint64(10) {
			merged := interleave(partition, seed)
			for _, part := range partition {
				for i := range len(part) - 1 {
					posA := slices.Index(merged, part[i])
					posB := slices.Index(merged, part[i+1])
					require.Positive(t, posA)
					require.Positive(t, posB)
					require.Less(t, posA, posB)
				}
			}
		}
	}
}

// --- Test Utilities ---

func toIndices(
	t *testing.T,
	full []transaction,
	sub []transaction,
) []int {
	t.Helper()
	var indices []int
	for _, cur := range sub {
		pos := slices.IndexFunc(full, func(tx transaction) bool {
			return reflect.DeepEqual(tx, cur)
		})
		require.NotEqual(t, -1, pos)
		indices = append(indices, pos)
	}
	return indices
}

// fakeNonceProvider is a fake nonce provider for testing purposes. Nonces can
// be defined explicitly for each sender or derived from a list of transactions
// using makeFakeNonceProvider.
type fakeNonceProvider map[sender]nonce

func (f fakeNonceProvider) GetNonce(sender sender) nonce {
	return f[sender]
}

// makeFakeNonceProvider creates a fake nonce provider that returns the smallest
// nonce for each sender that is referenced by any action in the list of
// transactions. This is the general assumption for the initial state in tests.
func makeFakeNonceProvider(transactions []transaction) fakeNonceProvider {
	res := fakeNonceProvider{}
	for _, tx := range transactions {
		for _, a := range tx.actions() {
			if nonce, found := res[a.sender]; !found || a.nonce < nonce {
				res[a.sender] = a.nonce
			}
		}
	}
	return res
}

func a(s sender, n nonce) action {
	return action{s, n}
}

func tx(m action, a ...action) transaction {
	return transaction{m, a}
}
