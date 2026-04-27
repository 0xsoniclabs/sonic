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
	Steps            []step           `json:"steps"`
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

// step represents either a leaf tx step (From+Hash set) or a group step (Steps set).
type step struct {
	From             common.Address `json:"from,omitempty"`
	Hash             common.Hash    `json:"hash,omitempty"`
	TolerateFailed   bool           `json:"tolerateFailed,omitempty"`
	TolerateInvalid  bool           `json:"tolerateInvalid,omitempty"`
	OneOf            bool           `json:"oneOf,omitempty"`
	TolerateFailures bool           `json:"tolerateFailures,omitempty"`
	Steps            []step         `json:"steps,omitempty"`
}

// TestPrepareBundle tests the sonic_prepareBundle RPC method.
func TestPrepareBundle(t *testing.T) {
	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{})
	sponsor := net.GetSessionSponsor()
	from := sponsor.Address()
	to := common.HexToAddress("0x2222222222222222222222222222222222222222")

	testCases := map[string]struct {
		args        map[string]any
		wantErr     bool
		wantTxCount int
		checkResult func(*testing.T, prepareBundleResult)
	}{
		"single_tx": {
			args: map[string]any{
				"steps": []any{txStep(from, to, 1)},
			},
			wantTxCount: 1,
			checkResult: func(t *testing.T, r prepareBundleResult) {
				// single leaf at root: wrapped in an AllOf group
				require.Len(t, r.ExecutionPlan.Steps, 1)
				group := r.ExecutionPlan.Steps[0]
				require.Len(t, group.Steps, 1, "single tx wrapped in AllOf group")
				require.NotEqual(t, common.Address{}, group.Steps[0].From, "leaf must have From set")
			},
		},
		"two_txs_all_of": {
			args: map[string]any{
				"steps": []any{txStep(from, to, 1), txStep(from, to, 2)},
			},
			wantTxCount: 2,
			checkResult: func(t *testing.T, r prepareBundleResult) {
				require.Len(t, r.ExecutionPlan.Steps, 1, "multiple txs wrapped in root AllOf group")
				require.Len(t, r.ExecutionPlan.Steps[0].Steps, 2)
				require.False(t, r.ExecutionPlan.Steps[0].OneOf)
			},
		},
		"nested_one_of_group": {
			args: map[string]any{
				"steps": []any{
					txStep(from, to, 1),
					groupStep(true, txStep(from, to, 2), txStep(from, to, 3)),
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
		"nested_all_of_group": {
			args: map[string]any{
				"steps": []any{
					txStep(from, to, 1),
					groupStep(false, txStep(from, to, 2), txStep(from, to, 3)),
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
		"tolerate_failed_flag_preserved": {
			args: map[string]any{
				"steps": []any{txStepWithFlags(from, to, 1, true, false)},
			},
			wantTxCount: 1,
			checkResult: func(t *testing.T, r prepareBundleResult) {
				require.Len(t, r.ExecutionPlan.Steps, 1)
				group := r.ExecutionPlan.Steps[0]
				require.Len(t, group.Steps, 1)
				require.True(t, group.Steps[0].TolerateFailed)
				require.False(t, group.Steps[0].TolerateInvalid)
			},
		},
		"tolerate_invalid_flag_preserved": {
			args: map[string]any{
				"steps": []any{txStepWithFlags(from, to, 1, false, true)},
			},
			wantTxCount: 1,
			checkResult: func(t *testing.T, r prepareBundleResult) {
				require.Len(t, r.ExecutionPlan.Steps, 1)
				group := r.ExecutionPlan.Steps[0]
				require.Len(t, group.Steps, 1)
				require.False(t, group.Steps[0].TolerateFailed)
				require.True(t, group.Steps[0].TolerateInvalid)
			},
		},
		"one_of_group_flag": {
			args: map[string]any{
				"steps": []any{
					groupStep(true, txStep(from, to, 1), txStep(from, to, 2)),
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
		"tolerate_failures_on_group": {
			args: map[string]any{
				"steps": []any{
					groupStepWithFlags(false, true, txStep(from, to, 1), txStep(from, to, 2)),
				},
			},
			wantTxCount: 2,
			checkResult: func(t *testing.T, r prepareBundleResult) {
				// TolerateFailures prevents the single-child root from eliding its child.
				require.Len(t, r.ExecutionPlan.Steps, 1)
				require.Len(t, r.ExecutionPlan.Steps[0].Steps, 2)
			},
		},
		"bundle_only_marker_injected": {
			args: map[string]any{
				"steps": []any{txStep(from, to, 1)},
			},
			wantTxCount: 1,
		},
		"plan_hash_consistent_across_txs": {
			args: map[string]any{
				"steps": []any{txStep(from, to, 1), txStep(from, to, 2)},
			},
			wantTxCount: 2,
		},
		"gas_filled_when_omitted": {
			args: map[string]any{
				"steps": []any{txStepNoGas(from, to, 1)},
			},
			wantTxCount: 1,
			checkResult: func(t *testing.T, r prepareBundleResult) {
				require.NotNil(t, r.Transactions[0].Gas, "gas must be filled in when omitted")
				require.Greater(t, uint64(*r.Transactions[0].Gas), uint64(0))
			},
		},
		"custom_block_range": {
			args: map[string]any{
				"steps": []any{txStep(from, to, 1)},
				"blockRange": map[string]any{
					"earliest": hexutil.Uint64(100),
					"latest":   hexutil.Uint64(200),
				},
			},
			wantTxCount: 1,
			checkResult: func(t *testing.T, r prepareBundleResult) {
				require.Equal(t, hexutil.Uint64(100), r.ExecutionPlan.BlockRange.Earliest)
				require.Equal(t, hexutil.Uint64(200), r.ExecutionPlan.BlockRange.Latest)
			},
		},
		"default_block_range_bounded_by_max": {
			args: map[string]any{
				"steps": []any{txStep(from, to, 1)},
			},
			wantTxCount: 1,
			checkResult: func(t *testing.T, r prepareBundleResult) {
				require.Greater(t, uint64(r.ExecutionPlan.BlockRange.Earliest), uint64(0), "earliest must be above genesis")
				rangeSize := uint64(r.ExecutionPlan.BlockRange.Latest) - uint64(r.ExecutionPlan.BlockRange.Earliest) + 1
				require.EqualValues(t, bundle.MaxBlockRange, rangeSize, "default range must be exactly MaxBlockRange")
			},
		},
		"error_invalid_block_range_latest_before_earliest": {
			args: map[string]any{
				"steps": []any{txStep(from, to, 1)},
				"blockRange": map[string]any{
					"earliest": hexutil.Uint64(200),
					"latest":   hexutil.Uint64(100),
				},
			},
			wantErr: true,
		},
		"error_range_too_large": {
			args: map[string]any{
				"steps": []any{txStep(from, to, 1)},
				"blockRange": map[string]any{
					"earliest": hexutil.Uint64(1),
					"latest":   hexutil.Uint64(1 + bundle.MaxBlockRange), // range = MaxBlockRange+1 blocks
				},
			},
			wantErr: true,
		},
		"empty_steps_returns_error": {
			args: map[string]any{
				"steps": []any{},
			},
			wantErr: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
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

// txStep builds a leaf step JSON with gas fields set.
func txStep(from, to common.Address, nonce uint64) map[string]any {
	return map[string]any{
		"from":     from,
		"to":       to,
		"nonce":    hexutil.Uint64(nonce),
		"value":    (*hexutil.Big)(hexutil.MustDecodeBig("0x1")),
		"gas":      hexutil.Uint64(21000),
		"gasPrice": (*hexutil.Big)(hexutil.MustDecodeBig("0x1")),
	}
}

// txStepNoGas builds a leaf step without gas fields to trigger auto-estimation.
func txStepNoGas(from, to common.Address, nonce uint64) map[string]any {
	return map[string]any{
		"from":  from,
		"to":    to,
		"nonce": hexutil.Uint64(nonce),
		"value": (*hexutil.Big)(hexutil.MustDecodeBig("0x1")),
	}
}

// txStepWithFlags adds tolerateFailed/tolerateInvalid flags to a txStep.
func txStepWithFlags(from, to common.Address, nonce uint64, tolerateFailed, tolerateInvalid bool) map[string]any {
	step := txStep(from, to, nonce)
	if tolerateFailed {
		step["tolerateFailed"] = true
	}
	if tolerateInvalid {
		step["tolerateInvalid"] = true
	}
	return step
}

// groupStep builds a group step JSON.
func groupStep(oneOf bool, steps ...any) map[string]any {
	return map[string]any{
		"oneOf": oneOf,
		"steps": steps,
	}
}

// groupStepWithFlags builds a group step JSON with tolerateFailures flag.
func groupStepWithFlags(oneOf, tolerateFailures bool, steps ...any) map[string]any {
	return map[string]any{
		"oneOf":            oneOf,
		"tolerateFailures": tolerateFailures,
		"steps":            steps,
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
