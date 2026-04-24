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

package rpcs

import (
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// --- response decode types for sonic_prepareBundle ---

type prepareBundleResult struct {
	Transactions  []prepareBundleTxResult `json:"transactions"`
	ExecutionPlan executionPlanResult     `json:"executionPlan"`
}

type executionPlanResult struct {
	BlockRange       blockRangeResult `json:"blockRange"`
	Steps            []rpcStep        `json:"steps"`
	OneOf            bool             `json:"oneOf,omitempty"`
	TolerateFailures bool             `json:"tolerateFailures,omitempty"`
}

type blockRangeResult struct {
	Earliest hexutil.Uint64 `json:"earliest"`
	Latest   hexutil.Uint64 `json:"latest"`
}

type prepareBundleTxResult struct {
	Gas        *hexutil.Uint64   `json:"gas,omitempty"`
	AccessList *types.AccessList `json:"accessList,omitempty"`
}

// rpcStep represents either a leaf tx step (From+Hash set) or a group step (Steps set).
type rpcStep struct {
	From             common.Address `json:"from,omitempty"`
	Hash             common.Hash    `json:"hash,omitempty"`
	TolerateFailed   bool           `json:"tolerateFailed,omitempty"`
	TolerateInvalid  bool           `json:"tolerateInvalid,omitempty"`
	OneOf            bool           `json:"oneOf,omitempty"`
	TolerateFailures bool           `json:"tolerateFailures,omitempty"`
	Steps            []rpcStep      `json:"steps,omitempty"`
}

// TestPrepareBundle tests the sonic_prepareBundle RPC method.
func TestPrepareBundle(t *testing.T) {
	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{})
	sponsor := net.GetSessionSponsor()
	from := sponsor.Address()
	to := common.HexToAddress("0x2222222222222222222222222222222222222222")

	testCases := []struct {
		name        string
		args        map[string]any
		wantErr     bool
		wantTxCount int
		checkResult func(*testing.T, prepareBundleResult)
	}{
		// --- flat transactions shorthand ---
		{
			name: "flat_transactions_single",
			args: map[string]any{
				"transactions": []any{txArgs(from, to, 1)},
			},
			wantTxCount: 1,
			checkResult: func(t *testing.T, r prepareBundleResult) {
				// single tx: root group is elided, leaf step surfaces directly
				require.Len(t, r.ExecutionPlan.Steps, 1, "single tx elides to direct leaf step")
				require.NotEqual(t, common.Address{}, r.ExecutionPlan.Steps[0].From, "leaf step must have From set")
			},
		},
		{
			name: "flat_transactions_multiple",
			args: map[string]any{
				"transactions": []any{txArgs(from, to, 1), txArgs(from, to, 2)},
			},
			wantTxCount: 2,
			checkResult: func(t *testing.T, r prepareBundleResult) {
				// multiple txs: wrapped in a root all-of group
				require.Len(t, r.ExecutionPlan.Steps, 1, "multiple txs wrap in a root all-of group")
				require.Len(t, r.ExecutionPlan.Steps[0].Steps, 2)
				require.False(t, r.ExecutionPlan.Steps[0].OneOf)
			},
		},
		// --- entries (nested structure) ---
		{
			name: "entries_single_tx",
			args: map[string]any{
				"entries": []any{txEntry(from, to, 1)},
			},
			wantTxCount: 1,
			checkResult: func(t *testing.T, r prepareBundleResult) {
				require.Len(t, r.ExecutionPlan.Steps, 1)
				require.NotEqual(t, common.Address{}, r.ExecutionPlan.Steps[0].From)
			},
		},
		{
			name: "entries_two_txs_all_of",
			args: map[string]any{
				"entries": []any{txEntry(from, to, 1), txEntry(from, to, 2)},
			},
			wantTxCount: 2,
			checkResult: func(t *testing.T, r prepareBundleResult) {
				require.Len(t, r.ExecutionPlan.Steps, 1, "root all-of group wraps 2 txs")
				require.Len(t, r.ExecutionPlan.Steps[0].Steps, 2)
				require.False(t, r.ExecutionPlan.Steps[0].OneOf)
			},
		},
		{
			name: "nested_one_of_group",
			args: map[string]any{
				"entries": []any{
					txEntry(from, to, 1),
					groupEntry(true, txEntry(from, to, 2), txEntry(from, to, 3)),
				},
			},
			wantTxCount: 3,
			checkResult: func(t *testing.T, r prepareBundleResult) {
				require.Len(t, r.ExecutionPlan.Steps, 1, "root all-of group")
				root := r.ExecutionPlan.Steps[0]
				require.Len(t, root.Steps, 2, "root has leaf + oneOf group")
				require.NotEqual(t, common.Address{}, root.Steps[0].From, "first child is a leaf")
				require.True(t, root.Steps[1].OneOf, "second child is a oneOf group")
				require.Len(t, root.Steps[1].Steps, 2, "oneOf group has 2 leaves")
			},
		},
		{
			name: "nested_all_of_group",
			args: map[string]any{
				"entries": []any{
					txEntry(from, to, 1),
					groupEntry(false, txEntry(from, to, 2), txEntry(from, to, 3)),
				},
			},
			wantTxCount: 3,
			checkResult: func(t *testing.T, r prepareBundleResult) {
				require.Len(t, r.ExecutionPlan.Steps, 1)
				root := r.ExecutionPlan.Steps[0]
				require.Len(t, root.Steps, 2)
				require.False(t, root.Steps[1].OneOf, "nested group is all-of")
				require.Len(t, root.Steps[1].Steps, 2)
			},
		},
		// --- execution flags ---
		{
			name: "tolerate_failed_flag_preserved",
			args: map[string]any{
				"entries": []any{txEntryWithFlags(from, to, 1, true, false)},
			},
			wantTxCount: 1,
			checkResult: func(t *testing.T, r prepareBundleResult) {
				require.Len(t, r.ExecutionPlan.Steps, 1)
				require.True(t, r.ExecutionPlan.Steps[0].TolerateFailed)
				require.False(t, r.ExecutionPlan.Steps[0].TolerateInvalid)
			},
		},
		{
			name: "tolerate_invalid_flag_preserved",
			args: map[string]any{
				"entries": []any{txEntryWithFlags(from, to, 1, false, true)},
			},
			wantTxCount: 1,
			checkResult: func(t *testing.T, r prepareBundleResult) {
				require.Len(t, r.ExecutionPlan.Steps, 1)
				require.False(t, r.ExecutionPlan.Steps[0].TolerateFailed)
				require.True(t, r.ExecutionPlan.Steps[0].TolerateInvalid)
			},
		},
		{
			name: "one_of_group_flag",
			args: map[string]any{
				"entries": []any{
					groupEntry(true, txEntry(from, to, 1), txEntry(from, to, 2)),
				},
			},
			wantTxCount: 2,
			checkResult: func(t *testing.T, r prepareBundleResult) {
				// single-child no-modifier root is elided → root IS the oneOf group
				require.Len(t, r.ExecutionPlan.Steps, 1)
				require.True(t, r.ExecutionPlan.Steps[0].OneOf)
				require.Len(t, r.ExecutionPlan.Steps[0].Steps, 2)
			},
		},
		{
			name: "tolerate_failures_on_group",
			args: map[string]any{
				"entries": []any{
					groupEntryWithFlags(false, true, txEntry(from, to, 1), txEntry(from, to, 2)),
				},
			},
			wantTxCount: 2,
			checkResult: func(t *testing.T, r prepareBundleResult) {
				// single-child no-modifier root is elided → root IS the tolerateFailures group
				require.Len(t, r.ExecutionPlan.Steps, 1)
				require.True(t, r.ExecutionPlan.Steps[0].TolerateFailures)
			},
		},
		// --- bundle-only marker and plan hash ---
		{
			name: "bundle_only_marker_injected",
			args: map[string]any{
				"entries": []any{txEntry(from, to, 1)},
			},
			wantTxCount: 1,
		},
		{
			name: "plan_hash_consistent_across_txs",
			args: map[string]any{
				"entries": []any{txEntry(from, to, 1), txEntry(from, to, 2)},
			},
			wantTxCount: 2,
		},
		// --- gas auto-fill ---
		{
			name: "gas_filled_when_omitted",
			args: map[string]any{
				"entries": []any{txEntryNoGas(from, to, 1)},
			},
			wantTxCount: 1,
			checkResult: func(t *testing.T, r prepareBundleResult) {
				require.NotNil(t, r.Transactions[0].Gas, "gas must be filled in when omitted")
				require.Greater(t, uint64(*r.Transactions[0].Gas), uint64(0))
			},
		},
		// --- block range ---
		{
			name: "custom_block_range",
			args: map[string]any{
				"entries":       []any{txEntry(from, to, 1)},
				"earliestBlock": hexutil.Uint64(100),
				"latestBlock":   hexutil.Uint64(200),
			},
			wantTxCount: 1,
			checkResult: func(t *testing.T, r prepareBundleResult) {
				require.Equal(t, hexutil.Uint64(100), r.ExecutionPlan.BlockRange.Earliest)
				require.Equal(t, hexutil.Uint64(200), r.ExecutionPlan.BlockRange.Latest)
			},
		},
		{
			name: "default_block_range_bounded_by_max",
			args: map[string]any{
				"entries": []any{txEntry(from, to, 1)},
			},
			wantTxCount: 1,
			checkResult: func(t *testing.T, r prepareBundleResult) {
				require.Greater(t, uint64(r.ExecutionPlan.BlockRange.Earliest), uint64(0), "earliest must be above genesis")
				rangeSize := uint64(r.ExecutionPlan.BlockRange.Latest) - uint64(r.ExecutionPlan.BlockRange.Earliest) + 1
				require.EqualValues(t, bundle.MaxBlockRange, rangeSize, "default range must be exactly MaxBlockRange")
			},
		},
		// --- error cases ---
		{
			name: "error_both_transactions_and_entries",
			args: map[string]any{
				"transactions": []any{txArgs(from, to, 1)},
				"entries":      []any{txEntry(from, to, 1)},
			},
			wantErr: true,
		},
		{
			name: "error_invalid_block_range_latest_before_earliest",
			args: map[string]any{
				"entries":       []any{txEntry(from, to, 1)},
				"earliestBlock": hexutil.Uint64(200),
				"latestBlock":   hexutil.Uint64(100),
			},
			wantErr: true,
		},
		{
			name: "error_range_too_large",
			args: map[string]any{
				"entries":       []any{txEntry(from, to, 1)},
				"earliestBlock": hexutil.Uint64(1),
				"latestBlock":   hexutil.Uint64(1 + bundle.MaxBlockRange), // range = MaxBlockRange+1 blocks
			},
			wantErr: true,
		},
		{
			name: "empty_transactions_returns_empty",
			args: map[string]any{
				"transactions": []any{},
			},
			wantTxCount: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client, err := net.GetClient()
			require.NoError(t, err)
			defer client.Close()

			var result prepareBundleResult
			err = client.Client().Call(&result, "sonic_prepareBundle", tc.args)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, result.Transactions, tc.wantTxCount)
			if tc.wantTxCount > 0 {
				checkBundleOnlyMarker(t, result)
			}
			if tc.checkResult != nil {
				tc.checkResult(t, result)
			}
		})
	}
}

// --- helpers for building RPC request JSON ---

// txArgs builds flat TransactionArgs JSON for use in the "transactions" shorthand list.
func txArgs(from, to common.Address, nonce uint64) map[string]any {
	return map[string]any{
		"from":     from,
		"to":       to,
		"nonce":    hexutil.Uint64(nonce),
		"value":    (*hexutil.Big)(hexutil.MustDecodeBig("0x1")),
		"gas":      hexutil.Uint64(21000),
		"gasPrice": (*hexutil.Big)(hexutil.MustDecodeBig("0x1")),
	}
}

// txArgsNoGas builds flat TransactionArgs JSON without gas fields to trigger auto-estimation.
func txArgsNoGas(from, to common.Address, nonce uint64) map[string]any {
	return map[string]any{
		"from":  from,
		"to":    to,
		"nonce": hexutil.Uint64(nonce),
		"value": (*hexutil.Big)(hexutil.MustDecodeBig("0x1")),
	}
}

// txEntry wraps txArgs in a "transaction" discriminator for use in an "entries" list.
func txEntry(from, to common.Address, nonce uint64) map[string]any {
	return map[string]any{"transaction": txArgs(from, to, nonce)}
}

// txEntryNoGas wraps txArgsNoGas in a "transaction" discriminator.
func txEntryNoGas(from, to common.Address, nonce uint64) map[string]any {
	return map[string]any{"transaction": txArgsNoGas(from, to, nonce)}
}

// txEntryWithFlags adds tolerateFailed/tolerateInvalid flags to a txEntry.
func txEntryWithFlags(from, to common.Address, nonce uint64, tolerateFailed, tolerateInvalid bool) map[string]any {
	entry := txEntry(from, to, nonce)
	if tolerateFailed {
		entry["tolerateFailed"] = true
	}
	if tolerateInvalid {
		entry["tolerateInvalid"] = true
	}
	return entry
}

// groupEntry builds a group step JSON using the "entries" discriminator key.
func groupEntry(oneOf bool, entries ...any) map[string]any {
	return map[string]any{
		"oneOf":   oneOf,
		"entries": entries,
	}
}

// groupEntryWithFlags builds a group step JSON with tolerateFailures flag.
func groupEntryWithFlags(oneOf, tolerateFailures bool, entries ...any) map[string]any {
	return map[string]any{
		"oneOf":            oneOf,
		"tolerateFailures": tolerateFailures,
		"entries":          entries,
	}
}

// checkBundleOnlyMarker asserts every transaction has the BundleOnly address in its
// access list, and that all transactions share the same plan hash storage key.
func checkBundleOnlyMarker(t *testing.T, result prepareBundleResult) {
	t.Helper()
	var planHash common.Hash
	for i, tx := range result.Transactions {
		require.NotNil(t, tx.AccessList, "tx %d must have accessList", i)
		found := false
		for _, entry := range *tx.AccessList {
			if entry.Address == bundle.BundleOnly {
				found = true
				require.Len(t, entry.StorageKeys, 1, "tx %d BundleOnly entry must have exactly 1 storage key", i)
				if i == 0 {
					planHash = entry.StorageKeys[0]
				} else {
					require.Equal(t, planHash, entry.StorageKeys[0], "tx %d must share plan hash with tx 0", i)
				}
				break
			}
		}
		require.True(t, found, "tx %d must have BundleOnly address in accessList", i)
	}
}
