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

package sonicapi

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/api/ethapi"
	rpctest "github.com/0xsoniclabs/sonic/api/rpc_test"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

// txEntry is a test helper that wraps a TransactionArgs as a leaf PrepareBundleEntry.
func txEntry(tx ethapi.TransactionArgs) RPCExecutionStepProposal {
	return txEntryWithFlags(tx, false, false)
}

// txEntryWithFlags wraps a TransactionArgs as a leaf step with execution flags.
func txEntryWithFlags(tx ethapi.TransactionArgs, tolerateFailed, tolerateInvalid bool) RPCExecutionStepProposal {
	return RPCExecutionStepProposal{
		TolerateFailed:  tolerateFailed,
		TolerateInvalid: tolerateInvalid,
		TransactionArgs: tx,
	}
}

func groupEntry(steps ...any) RPCExecutionPlanGroup {
	return groupEntryWithFlags(false, false, steps...)
}

func groupEntryWithFlags(oneOf, tolerateFailures bool, steps ...any) RPCExecutionPlanGroup {
	return RPCExecutionPlanGroup{
		OneOf:            oneOf,
		TolerateFailures: tolerateFailures,
		Steps:            steps,
	}
}

func Test_fillTransactionDefaults(t *testing.T) {
	addr := common.Address{1}
	gasPrice := rpctest.ToHexBigInt(big.NewInt(1e9))

	existingGasPrice := rpctest.ToHexBigInt(big.NewInt(999))
	existingMaxFee := rpctest.ToHexBigInt(big.NewInt(888))
	tip := rpctest.ToHexBigInt(big.NewInt(1))
	explicit := hexutil.Uint64(50000)

	tests := []struct {
		name     string
		txs      ethapi.TransactionArgs
		gasLimit *hexutil.Uint64
		gasPrice *hexutil.Big
		check    func(t *testing.T, txs ethapi.TransactionArgs)
	}{
		{
			name:     "nil gas limits fixed gas price",
			txs:      ethapi.TransactionArgs{From: &addr},
			gasLimit: nil,
			gasPrice: gasPrice,
			check: func(t *testing.T, txs ethapi.TransactionArgs) {
				require.Nil(t, txs.Gas, "gas must remain nil when no limits provided")
				require.Equal(t, gasPrice, txs.GasPrice)
			},
		},
		{
			name:     "gas limits provided fills gas",
			txs:      ethapi.TransactionArgs{From: &addr},
			gasLimit: rpctest.ToHexUint64(21000),
			gasPrice: rpctest.ToHexBigInt(big.NewInt(1)),
			check: func(t *testing.T, txs ethapi.TransactionArgs) {
				require.NotNil(t, txs.Gas)
				require.EqualValues(t, 21000, *txs.Gas)
			},
		},
		{
			name:     "gas limits provided preserves explicit gas",
			txs:      ethapi.TransactionArgs{From: &addr, Gas: &explicit},
			gasLimit: rpctest.ToHexUint64(21000),
			gasPrice: rpctest.ToHexBigInt(big.NewInt(1)),
			check: func(t *testing.T, txs ethapi.TransactionArgs) {
				require.EqualValues(t, 50000, *txs.Gas, "explicit gas must not be overwritten")
			},
		},
		{
			name:     "existing gas price not overwritten",
			txs:      ethapi.TransactionArgs{From: &addr, GasPrice: existingGasPrice},
			gasLimit: nil,
			gasPrice: rpctest.ToHexBigInt(big.NewInt(1e9)),
			check: func(t *testing.T, txs ethapi.TransactionArgs) {
				require.Equal(t, existingGasPrice, txs.GasPrice, "existing GasPrice must not be overwritten")
			},
		},
		{
			name:     "existing MaxFeePerGas not overwritten",
			txs:      ethapi.TransactionArgs{From: &addr, MaxFeePerGas: existingMaxFee},
			gasLimit: nil,
			gasPrice: rpctest.ToHexBigInt(big.NewInt(1e9)),
			check: func(t *testing.T, txs ethapi.TransactionArgs) {
				require.Equal(t, existingMaxFee, txs.MaxFeePerGas, "existing MaxFeePerGas must not be overwritten")
				require.Nil(t, txs.GasPrice)
			},
		},
		{
			name:     "MaxPriorityFeePerGas set causes MaxFeePerGas to be filled",
			txs:      ethapi.TransactionArgs{From: &addr, MaxPriorityFeePerGas: tip},
			gasPrice: gasPrice,
			check: func(t *testing.T, txs ethapi.TransactionArgs) {
				require.Equal(t, gasPrice, txs.MaxFeePerGas)
				require.Nil(t, txs.GasPrice)
			},
		},
		{
			name:     "MaxPriorityFeePerGas with existing MaxFeePerGas skips price fill",
			txs:      ethapi.TransactionArgs{From: &addr, MaxPriorityFeePerGas: tip, MaxFeePerGas: existingMaxFee},
			gasPrice: gasPrice,
			check: func(t *testing.T, txs ethapi.TransactionArgs) {
				require.Equal(t, existingMaxFee, txs.MaxFeePerGas, "existing MaxFeePerGas must not be overwritten")
				require.Nil(t, txs.GasPrice)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			filledArgs := fillTransactionDefaults(tc.txs, tc.gasLimit, tc.gasPrice)
			tc.check(t, filledArgs)
		})
	}
}

func Test_resolveBlockRange(t *testing.T) {
	hexN := func(n uint64) hexutil.Uint64 { b := hexutil.Uint64(n); return b }

	tests := []struct {
		name          string
		currentBlock  uint64
		blockRange    *RPCRange
		wantEarliest  uint64
		wantLatest    uint64
		errorContains string
	}{
		{
			name:         "nil both defaults from current block",
			currentBlock: 10,
			wantEarliest: 11,
			wantLatest:   10 + bundle.MaxBlockRange,
		},
		{
			name:          "only earliest",
			currentBlock:  10,
			blockRange:    &RPCRange{Earliest: hexN(50)},
			errorContains: "invalid block range",
		},
		{
			name:         "explicit latest",
			currentBlock: 10,
			blockRange:   &RPCRange{Latest: hexN(200)},
			wantEarliest: 0,
			wantLatest:   200,
		},
		{
			name:          "range exceeds MaxBlockRange when only latest set",
			currentBlock:  10,
			blockRange:    &RPCRange{Latest: hexN(10 + bundle.MaxBlockRange + 100)},
			errorContains: "invalid block range",
		},
		{
			name:         "both explicit",
			currentBlock: 10,
			blockRange:   &RPCRange{Earliest: hexN(5), Latest: hexN(20)},
			wantEarliest: 5,
			wantLatest:   20,
		},
		{
			name:         "current block zero earliest is one",
			currentBlock: 0,
			wantEarliest: 1,
			wantLatest:   bundle.MaxBlockRange,
		},
		{
			name:          "latest is less than earliest",
			currentBlock:  100,
			blockRange:    &RPCRange{Earliest: hexN(50), Latest: hexN(40)},
			errorContains: "invalid block range",
		},
		{
			name:          "latest before implicit earliest from current block",
			currentBlock:  10,
			blockRange:    &RPCRange{Latest: hexN(5)},
			errorContains: "invalid block range",
		},
		{
			name:          "greater than Max block range",
			currentBlock:  100,
			blockRange:    &RPCRange{Earliest: hexN(50), Latest: hexN(50 + bundle.MaxBlockRange + 1)},
			errorContains: "invalid block range",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r, err := validateBlockRange(tc.currentBlock, tc.blockRange)
			if tc.errorContains != "" {
				require.ErrorContains(t, err, tc.errorContains)
			} else {
				require.NoError(t, err)
				require.EqualValues(t, tc.wantEarliest, r.Earliest)
				require.EqualValues(t, tc.wantLatest, r.Latest)
			}
		})
	}
}

func Test_injectPlanHashIntoAccessLists(t *testing.T) {
	existingAddr := common.Address{0x42}

	tests := []struct {
		name     string
		txs      []ethapi.TransactionArgs
		planHash common.Hash
		check    func(t *testing.T, txs []ethapi.TransactionArgs)
	}{
		{
			name:     "nil access list creates bundle-only entry",
			txs:      []ethapi.TransactionArgs{{}},
			planHash: common.Hash{0xab},
			check: func(t *testing.T, txs []ethapi.TransactionArgs) {
				require.NotNil(t, txs[0].AccessList)
				al := *txs[0].AccessList
				require.Len(t, al, 1)
				require.Equal(t, bundle.BundleOnly, al[0].Address)
				require.Equal(t, []common.Hash{{0xab}}, al[0].StorageKeys)
			},
		},
		{
			name:     "existing entries appended",
			txs:      []ethapi.TransactionArgs{{AccessList: &types.AccessList{{Address: existingAddr}}}},
			planHash: common.Hash{0xcd},
			check: func(t *testing.T, txs []ethapi.TransactionArgs) {
				al := *txs[0].AccessList
				require.Len(t, al, 2)
				require.Equal(t, existingAddr, al[0].Address)
				require.Equal(t, bundle.BundleOnly, al[1].Address)
				require.Equal(t, []common.Hash{{0xcd}}, al[1].StorageKeys)
			},
		},
		{
			name:     "multiple txs all injected",
			txs:      []ethapi.TransactionArgs{{}, {}, {}},
			planHash: common.Hash{0x01},
			check: func(t *testing.T, txs []ethapi.TransactionArgs) {
				for i, tx := range txs {
					require.NotNil(t, tx.AccessList, "tx %d missing access list", i)
					al := *tx.AccessList
					require.Len(t, al, 1, "tx %d should have exactly one entry", i)
					require.Equal(t, bundle.BundleOnly, al[0].Address)
				}
			},
		},
		{
			name:     "nil txs no panic",
			txs:      nil,
			planHash: common.Hash{},
			check:    func(t *testing.T, txs []ethapi.TransactionArgs) {},
		},
		{
			name:     "empty txs no panic",
			txs:      []ethapi.TransactionArgs{},
			planHash: common.Hash{},
			check:    func(t *testing.T, txs []ethapi.TransactionArgs) {},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			injectPlanHashIntoAccessLists(tc.txs, tc.planHash)
			tc.check(t, tc.txs)
		})
	}
}

func Test_PrepareBundle_SingleTx_GasAndPriceEstimated(t *testing.T) {
	addr1 := common.Address{1}
	addr2 := common.Address{2}

	be := rpctest.NewBackendBuilder(t).
		WithAccount(addr1, rpctest.AccountState{Balance: big.NewInt(1e18)}).
		Build()

	api := NewPublicBundleAPI(be)

	args := RPCExecutionProposal{
		RPCExecutionPlanGroup: RPCExecutionPlanGroup{
			Steps: []any{
				txEntry(ethapi.TransactionArgs{
					From:  &addr1,
					To:    &addr2,
					Nonce: rpctest.ToHexUint64(0),
				}),
			},
		},
	}

	result, err := api.PrepareBundle(t.Context(), args)
	require.NoError(t, err)
	require.Len(t, result.Transactions, 1)

	tx := result.Transactions[0]
	require.NotNil(t, tx.Gas, "gas must be estimated when nil")
	require.GreaterOrEqual(t, uint64(*tx.Gas), uint64(params.TxGas))

	hasPriceField := tx.GasPrice != nil || tx.MaxFeePerGas != nil
	require.True(t, hasPriceField, "gas price must be set")
}

func Test_PrepareBundle_AccessListContainsConsistentPlanHash(t *testing.T) {
	addr1 := common.Address{1}
	addr2 := common.Address{2}

	be := rpctest.NewBackendBuilder(t).
		WithAccount(addr1, rpctest.AccountState{Balance: big.NewInt(1e18)}).
		Build()

	api := NewPublicBundleAPI(be)

	args := RPCExecutionProposal{
		RPCExecutionPlanGroup: RPCExecutionPlanGroup{
			Steps: []any{
				txEntry(ethapi.TransactionArgs{From: &addr1, To: &addr2, Nonce: rpctest.ToHexUint64(0), Value: rpctest.ToHexBigInt(big.NewInt(1e15))}),
				txEntry(ethapi.TransactionArgs{From: &addr2, To: &addr1, Nonce: rpctest.ToHexUint64(0), Value: rpctest.ToHexBigInt(big.NewInt(1e15))}),
			},
		},
	}

	result, err := api.PrepareBundle(t.Context(), args)
	require.NoError(t, err)
	require.Len(t, result.Transactions, 2)

	var planHash common.Hash
	for i, tx := range result.Transactions {
		require.NotNil(t, tx.AccessList)
		for _, entry := range *tx.AccessList {
			if entry.Address == bundle.BundleOnly {
				require.Len(t, entry.StorageKeys, 1)
				if i == 0 {
					planHash = entry.StorageKeys[0]
				} else {
					require.Equal(t, planHash, entry.StorageKeys[0], "tx %d has different plan hash", i)
				}
			}
		}
	}
}

func Test_PrepareBundle_ExplicitGasLimit_NotOverwritten(t *testing.T) {
	addr1 := common.Address{1}
	addr2 := common.Address{2}
	explicitGas := hexutil.Uint64(50000)

	be := rpctest.NewBackendBuilder(t).
		WithAccount(addr1, rpctest.AccountState{Balance: big.NewInt(1e18)}).
		Build()

	api := NewPublicBundleAPI(be)

	args := RPCExecutionProposal{
		RPCExecutionPlanGroup: RPCExecutionPlanGroup{
			Steps: []any{
				txEntry(ethapi.TransactionArgs{
					From:  &addr1,
					To:    &addr2,
					Nonce: rpctest.ToHexUint64(0),
					Gas:   &explicitGas,
				}),
			},
		},
	}

	result, err := api.PrepareBundle(t.Context(), args)
	require.NoError(t, err)
	require.EqualValues(t, explicitGas, *result.Transactions[0].Gas)
}

func Test_PrepareBundle_DefaultBlockRange_IsCurrentBlockPlusOne(t *testing.T) {
	addr1 := common.Address{1}
	addr2 := common.Address{2}

	be := rpctest.NewBackendBuilder(t).
		WithAccount(addr1, rpctest.AccountState{Balance: big.NewInt(1e18)}).
		Build()

	api := NewPublicBundleAPI(be)

	currentBlock := be.CurrentBlock().NumberU64()

	args := RPCExecutionProposal{
		RPCExecutionPlanGroup: RPCExecutionPlanGroup{
			Steps: []any{
				txEntry(ethapi.TransactionArgs{
					From:  &addr1,
					To:    &addr2,
					Nonce: rpctest.ToHexUint64(0),
				}),
			},
		},
	}

	result, err := api.PrepareBundle(t.Context(), args)
	require.NoError(t, err)

	require.EqualValues(t, currentBlock+1, result.ExecutionPlan.BlockRange.Earliest)
	require.EqualValues(t, currentBlock+bundle.MaxBlockRange, result.ExecutionPlan.BlockRange.Latest)
}

func Test_PrepareBundle_ExplicitBlockRange_IsRespected(t *testing.T) {
	addr1 := common.Address{1}
	addr2 := common.Address{2}
	earliest := hexutil.Uint64(10)
	latest := hexutil.Uint64(20)

	be := rpctest.NewBackendBuilder(t).
		WithAccount(addr1, rpctest.AccountState{Balance: big.NewInt(1e18)}).
		Build()

	api := NewPublicBundleAPI(be)

	args := RPCExecutionProposal{
		RPCExecutionPlanGroup: RPCExecutionPlanGroup{
			Steps: []any{
				txEntry(ethapi.TransactionArgs{
					From:  &addr1,
					To:    &addr2,
					Nonce: rpctest.ToHexUint64(0),
				}),
			},
		},
		BlockRange: &RPCRange{
			Earliest: earliest,
			Latest:   latest,
		},
	}

	result, err := api.PrepareBundle(t.Context(), args)
	require.NoError(t, err)

	require.EqualValues(t, earliest, result.ExecutionPlan.BlockRange.Earliest)
	require.EqualValues(t, latest, result.ExecutionPlan.BlockRange.Latest)
}

func Test_PrepareBundle_MissingNonce_ReturnsError(t *testing.T) {
	addr1 := common.Address{1}
	addr2 := common.Address{2}

	be := rpctest.NewBackendBuilder(t).
		WithAccount(addr1, rpctest.AccountState{Balance: big.NewInt(1e18)}).
		Build()

	api := NewPublicBundleAPI(be)

	args := RPCExecutionProposal{
		RPCExecutionPlanGroup: RPCExecutionPlanGroup{
			Steps: []any{
				txEntry(ethapi.TransactionArgs{
					From: &addr1,
					To:   &addr2,
				}),
			},
		},
	}

	_, err := api.PrepareBundle(t.Context(), args)
	require.ErrorContains(t, err, "transaction 0 is missing nonce")
}

func Test_PrepareBundle_MultipleTxs_AllOfPlan(t *testing.T) {
	addr1 := common.Address{1}
	addr2 := common.Address{2}

	be := rpctest.NewBackendBuilder(t).
		WithAccount(addr1, rpctest.AccountState{Balance: big.NewInt(1e18)}).
		WithAccount(addr2, rpctest.AccountState{Balance: big.NewInt(1e18)}).
		Build()

	api := NewPublicBundleAPI(be)

	args := RPCExecutionProposal{
		RPCExecutionPlanGroup: RPCExecutionPlanGroup{
			Steps: []any{
				txEntry(ethapi.TransactionArgs{From: &addr1, To: &addr2, Nonce: rpctest.ToHexUint64(0), Value: rpctest.ToHexBigInt(big.NewInt(1e15))}),
				txEntry(ethapi.TransactionArgs{From: &addr2, To: &addr1, Nonce: rpctest.ToHexUint64(0), Value: rpctest.ToHexBigInt(big.NewInt(1e15))}),
			},
		},
	}

	result, err := api.PrepareBundle(t.Context(), args)
	require.NoError(t, err)
	require.Len(t, result.Transactions, 2)

	// Two txs at root produce an AllOf group: outer Steps has 1 group, inner has 2 steps.
	require.Len(t, result.ExecutionPlan.Steps, 1)
	innerGroup, ok := result.ExecutionPlan.Steps[0].(*RPCExecutionPlanGroup)
	require.True(t, ok, "expected inner AllOf group")
	require.False(t, innerGroup.OneOf)
	require.Len(t, innerGroup.Steps, 2)

	// Each tx must have the BundleOnly marker.
	for i, tx := range result.Transactions {
		require.NotNil(t, tx.AccessList, "tx %d missing access list", i)
		var found bool
		for _, entry := range *tx.AccessList {
			if entry.Address == bundle.BundleOnly {
				found = true
			}
		}
		require.True(t, found, "tx %d missing BundleOnly marker", i)
	}
}

func Test_PrepareBundle_EmptyTransactions_ReturnsEmptyBundle(t *testing.T) {
	be := rpctest.NewBackendBuilder(t).Build()
	api := NewPublicBundleAPI(be)

	args := RPCExecutionProposal{
		RPCExecutionPlanGroup: RPCExecutionPlanGroup{
			Steps: []any{},
		},
	}
	result, err := api.PrepareBundle(t.Context(), args)
	require.ErrorContains(t, err, "proposed group must include at least one step")
	require.Nil(t, result)
}

func Test_PrepareBundle_OneOfGroup_BuildsOneOfPlan(t *testing.T) {
	addr1 := common.Address{1}
	addr2 := common.Address{2}

	be := rpctest.NewBackendBuilder(t).
		WithAccount(addr1, rpctest.AccountState{Balance: big.NewInt(1e18)}).
		WithAccount(addr2, rpctest.AccountState{Balance: big.NewInt(1e18)}).
		Build()

	api := NewPublicBundleAPI(be)

	args := RPCExecutionProposal{
		RPCExecutionPlanGroup: RPCExecutionPlanGroup{
			OneOf: true,
			Steps: []any{
				txEntry(ethapi.TransactionArgs{From: &addr1, To: &addr2, Nonce: rpctest.ToHexUint64(0), Value: rpctest.ToHexBigInt(big.NewInt(1e15))}),
				txEntry(ethapi.TransactionArgs{From: &addr2, To: &addr1, Nonce: rpctest.ToHexUint64(0), Value: rpctest.ToHexBigInt(big.NewInt(1e15))}),
			},
		},
	}

	result, err := api.PrepareBundle(t.Context(), args)
	require.NoError(t, err)
	require.Len(t, result.Transactions, 2)

	// Single root step is the OneOf group (unwrapped since root has 1 child with no modifiers).
	require.Len(t, result.ExecutionPlan.Steps, 1)
	oneOfGroup, ok := result.ExecutionPlan.Steps[0].(*RPCExecutionPlanGroup)
	require.True(t, ok)
	require.True(t, oneOfGroup.OneOf, "expected OneOf group")
	require.Len(t, oneOfGroup.Steps, 2)
}

func Test_PrepareBundle_TolerateFailed_Flag(t *testing.T) {
	addr1 := common.Address{1}
	addr2 := common.Address{2}

	be := rpctest.NewBackendBuilder(t).
		WithAccount(addr1, rpctest.AccountState{Balance: big.NewInt(1e18)}).
		Build()

	api := NewPublicBundleAPI(be)

	args := RPCExecutionProposal{
		RPCExecutionPlanGroup: RPCExecutionPlanGroup{
			Steps: []any{
				txEntryWithFlags(ethapi.TransactionArgs{
					From:  &addr1,
					To:    &addr2,
					Nonce: rpctest.ToHexUint64(0),
				}, true, false),
				txEntryWithFlags(ethapi.TransactionArgs{
					From:  &addr1,
					To:    &addr2,
					Nonce: rpctest.ToHexUint64(0),
				}, false, true),
			},
		},
	}

	result, err := api.PrepareBundle(t.Context(), args)
	require.NoError(t, err)
	require.Len(t, result.ExecutionPlan.Steps, 1)

	stepGroup, ok := result.ExecutionPlan.Steps[0].(*RPCExecutionPlanGroup)
	require.True(t, ok, "expected step group")
	require.Len(t, stepGroup.Steps, 2)

	leafFailed, ok := stepGroup.Steps[0].(*RPCExecutionStepComposable)
	require.True(t, ok, "expected leaf step")
	require.True(t, leafFailed.TolerateFailed, "TolerateFailed must be set")
	require.False(t, leafFailed.TolerateInvalid)

	leafInvalid, ok := stepGroup.Steps[1].(*RPCExecutionStepComposable)
	require.True(t, ok, "expected leaf step")
	require.True(t, leafInvalid.TolerateInvalid, "TolerateInvalid must be set")
	require.False(t, leafInvalid.TolerateFailed)
}

func Test_PrepareBundle_NestedGroups(t *testing.T) {
	addr1 := common.Address{1}
	addr2 := common.Address{2}

	be := rpctest.NewBackendBuilder(t).
		WithAccount(addr1, rpctest.AccountState{Balance: big.NewInt(1e18)}).
		WithAccount(addr2, rpctest.AccountState{Balance: big.NewInt(1e18)}).
		Build()

	api := NewPublicBundleAPI(be)

	// OneOf(AllOf(tx1, tx2), tx3-alike via addr1 again)
	args := RPCExecutionProposal{
		RPCExecutionPlanGroup: RPCExecutionPlanGroup{
			Steps: []any{
				groupEntryWithFlags(true, false,
					groupEntry(
						txEntry(ethapi.TransactionArgs{From: &addr1, To: &addr2, Nonce: rpctest.ToHexUint64(0), Value: rpctest.ToHexBigInt(big.NewInt(1e15))}),
						txEntry(ethapi.TransactionArgs{From: &addr2, To: &addr1, Nonce: rpctest.ToHexUint64(0), Value: rpctest.ToHexBigInt(big.NewInt(1e15))}),
					),
					txEntry(ethapi.TransactionArgs{From: &addr1, To: &addr2, Nonce: rpctest.ToHexUint64(1), Value: rpctest.ToHexBigInt(big.NewInt(1e15))}),
				),
			},
		},
	}

	result, err := api.PrepareBundle(t.Context(), args)
	require.NoError(t, err)
	require.Len(t, result.Transactions, 3)

	// Root: 1 element (OneOf group)
	require.Len(t, result.ExecutionPlan.Steps, 1)
	oneOf, ok := result.ExecutionPlan.Steps[0].(*RPCExecutionPlanGroup)
	require.True(t, ok)
	require.True(t, oneOf.OneOf)
	require.Len(t, oneOf.Steps, 2)

	// First alt is an AllOf group
	allOf, ok := oneOf.Steps[0].(*RPCExecutionPlanGroup)
	require.True(t, ok)
	require.False(t, allOf.OneOf)
	require.Len(t, allOf.Steps, 2)

	// Second alt is a leaf
	_, ok = oneOf.Steps[1].(*RPCExecutionStepComposable)
	require.True(t, ok)
}

func Test_PrepareBundle_FlatTransactions_SingleTx(t *testing.T) {
	addr1 := common.Address{1}
	addr2 := common.Address{2}

	be := rpctest.NewBackendBuilder(t).
		WithAccount(addr1, rpctest.AccountState{Balance: big.NewInt(1e18)}).
		WithAccount(addr2, rpctest.AccountState{Balance: big.NewInt(1e18)}).
		Build()

	api := NewPublicBundleAPI(be)

	tx := ethapi.TransactionArgs{From: &addr1, To: &addr2, Nonce: rpctest.ToHexUint64(0), Value: rpctest.ToHexBigInt(big.NewInt(1e15))}
	args := RPCExecutionProposal{
		RPCExecutionPlanGroup: RPCExecutionPlanGroup{
			Steps: []any{
				txEntry(tx),
			},
		},
	}

	result, err := api.PrepareBundle(t.Context(), args)
	require.NoError(t, err)
	require.Len(t, result.Transactions, 1)

	// BundleOnly marker injected
	require.NotNil(t, result.Transactions[0].AccessList)
	found := false
	for _, entry := range *result.Transactions[0].AccessList {
		if entry.Address == bundle.BundleOnly {
			found = true
			break
		}
	}
	require.True(t, found, "expected BundleOnly marker in access list")

	require.Len(t, result.ExecutionPlan.Steps, 1)
	group, ok := result.ExecutionPlan.Steps[0].(*RPCExecutionPlanGroup)
	require.True(t, ok)
	require.False(t, group.OneOf)
	require.Len(t, group.Steps, 1)
	_, ok = group.Steps[0].(*RPCExecutionStepComposable)
	require.True(t, ok)
}

func Test_PrepareBundle_FlatTransactions_MultipleTxs(t *testing.T) {
	addr1 := common.Address{1}
	addr2 := common.Address{2}

	be := rpctest.NewBackendBuilder(t).
		WithAccount(addr1, rpctest.AccountState{Balance: big.NewInt(1e18)}).
		WithAccount(addr2, rpctest.AccountState{Balance: big.NewInt(1e18)}).
		Build()

	api := NewPublicBundleAPI(be)

	args := RPCExecutionProposal{
		RPCExecutionPlanGroup: RPCExecutionPlanGroup{
			Steps: []any{
				txEntry(ethapi.TransactionArgs{From: &addr1, To: &addr2, Nonce: rpctest.ToHexUint64(0), Value: rpctest.ToHexBigInt(big.NewInt(1e15))}),
				txEntry(ethapi.TransactionArgs{From: &addr2, To: &addr1, Nonce: rpctest.ToHexUint64(0), Value: rpctest.ToHexBigInt(big.NewInt(1e15))}),
			},
		},
	}

	result, err := api.PrepareBundle(t.Context(), args)
	require.NoError(t, err)
	require.Len(t, result.Transactions, 2)

	// BundleOnly injected into every tx
	for i, tx := range result.Transactions {
		require.NotNil(t, tx.AccessList, "tx %d missing access list", i)
		found := false
		for _, entry := range *tx.AccessList {
			if entry.Address == bundle.BundleOnly {
				found = true
				break
			}
		}
		require.True(t, found, "tx %d missing BundleOnly marker", i)
	}

	// Two-leaf AllOf: outer steps holds one AllOf group with two leaves
	require.Len(t, result.ExecutionPlan.Steps, 1)
	group, ok := result.ExecutionPlan.Steps[0].(*RPCExecutionPlanGroup)
	require.True(t, ok)
	require.False(t, group.OneOf)
	require.Len(t, group.Steps, 2)
}

func Test_PrepareBundle_FlatTransactions_OrderPreserved(t *testing.T) {
	addr1 := common.Address{1}
	addr2 := common.Address{2}

	be := rpctest.NewBackendBuilder(t).
		WithAccount(addr1, rpctest.AccountState{Balance: big.NewInt(1e18)}).
		WithAccount(addr2, rpctest.AccountState{Balance: big.NewInt(1e18)}).
		Build()

	api := NewPublicBundleAPI(be)
	args := RPCExecutionProposal{
		RPCExecutionPlanGroup: RPCExecutionPlanGroup{
			Steps: []any{
				txEntry(ethapi.TransactionArgs{From: &addr1, To: &addr2, Nonce: rpctest.ToHexUint64(0), Value: rpctest.ToHexBigInt(big.NewInt(1e15))}),
				txEntry(ethapi.TransactionArgs{From: &addr2, To: &addr1, Nonce: rpctest.ToHexUint64(0), Value: rpctest.ToHexBigInt(big.NewInt(1e15))}),
			},
		},
	}

	result, err := api.PrepareBundle(t.Context(), args)
	require.NoError(t, err)
	require.Len(t, result.Transactions, 2)

	require.Equal(t, &addr1, result.Transactions[0].From)
	require.Equal(t, &addr2, result.Transactions[0].To)
	require.Equal(t, &addr2, result.Transactions[1].From)
	require.Equal(t, &addr1, result.Transactions[1].To)
}

func Test_PrepareBundle_SingleChildGroup_TolerateFailures_NotUnwrapped(t *testing.T) {
	addr1 := common.Address{1}
	addr2 := common.Address{2}

	be := rpctest.NewBackendBuilder(t).
		WithAccount(addr1, rpctest.AccountState{Balance: big.NewInt(1e18)}).
		Build()

	api := NewPublicBundleAPI(be)

	args := RPCExecutionProposal{
		RPCExecutionPlanGroup: RPCExecutionPlanGroup{
			Steps: []any{
				groupEntryWithFlags(
					false, true, txEntry(ethapi.TransactionArgs{From: &addr1, To: &addr2, Nonce: rpctest.ToHexUint64(0)}),
				),
			},
		},
	}

	result, err := api.PrepareBundle(t.Context(), args)
	require.NoError(t, err)
	require.Len(t, result.Transactions, 1)

	// TolerateFailures flag must prevent single-child unwrap.
	require.Len(t, result.ExecutionPlan.Steps, 1)
	group, ok := result.ExecutionPlan.Steps[0].(*RPCExecutionPlanGroup)
	require.True(t, ok, "expected group, not leaf")
	require.False(t, group.OneOf)
	require.Len(t, group.Steps, 1)
}

func Test_PrepareBundle_SingleChildGroup_TolerateFailures_NotUnwrapped2(t *testing.T) {
	addr1 := common.Address{1}
	addr2 := common.Address{2}

	be := rpctest.NewBackendBuilder(t).
		WithAccount(addr1, rpctest.AccountState{Balance: big.NewInt(1e18)}).
		Build()

	api := NewPublicBundleAPI(be)

	rpcGroup := RPCExecutionPlanGroup{
		Steps: []any{
			txEntry(ethapi.TransactionArgs{From: &addr1, To: &addr2, Nonce: rpctest.ToHexUint64(0)}),
		},
	}

	args := RPCExecutionProposal{
		RPCExecutionPlanGroup: RPCExecutionPlanGroup{
			Steps: []any{
				rpcGroup,
			},
		},
	}

	result, err := api.PrepareBundle(t.Context(), args)
	require.NoError(t, err)
	require.Len(t, result.Transactions, 1)

	// TolerateFailures flag must prevent single-child unwrap.
	require.Len(t, result.ExecutionPlan.Steps, 1)
	group, ok := result.ExecutionPlan.Steps[0].(*RPCExecutionPlanGroup)
	require.True(t, ok, "expected group, not leaf")
	require.False(t, group.OneOf)
	require.Len(t, group.Steps, 1)
}
