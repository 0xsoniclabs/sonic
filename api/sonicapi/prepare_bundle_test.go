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
	"fmt"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/api/ethapi"
	"github.com/0xsoniclabs/sonic/api/rpctest"
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

func Test_injectPlanHashIntoAccessLists(t *testing.T) {
	existingAddr := common.Address{0x42}

	tests := map[string]struct {
		txs      []ethapi.TransactionArgs
		planHash common.Hash
		check    func(t *testing.T, txs []ethapi.TransactionArgs)
	}{
		"nil access list creates bundle-only entry": {
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
		"existing entries appended": {
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
		"multiple txs all injected": {
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
		"nil txs no panic": {
			txs:      nil,
			planHash: common.Hash{},
			check:    func(t *testing.T, txs []ethapi.TransactionArgs) {},
		},
		"empty txs no panic": {
			txs:      []ethapi.TransactionArgs{},
			planHash: common.Hash{},
			check:    func(t *testing.T, txs []ethapi.TransactionArgs) {},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
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

func Test_PrepareBundle_MixedGas_ExplicitPreservedMissingEstimated(t *testing.T) {
	addr1 := common.Address{1}
	addr2 := common.Address{2}
	explicitGas := hexutil.Uint64(50000)

	be := rpctest.NewBackendBuilder(t).
		WithAccount(addr1, rpctest.AccountState{Balance: big.NewInt(1e18)}).
		WithAccount(addr2, rpctest.AccountState{Balance: big.NewInt(1e18)}).
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
				txEntry(ethapi.TransactionArgs{
					From:  &addr2,
					To:    &addr1,
					Nonce: rpctest.ToHexUint64(0),
				}),
			},
		},
	}

	result, err := api.PrepareBundle(t.Context(), args)
	require.NoError(t, err)
	require.Len(t, result.Transactions, 2)

	require.EqualValues(t, explicitGas, *result.Transactions[0].Gas, "explicit gas must be preserved")
	require.NotNil(t, result.Transactions[1].Gas, "missing gas must be estimated")
	require.GreaterOrEqual(t, uint64(*result.Transactions[1].Gas), uint64(params.TxGas), "estimated gas must be at least base tx gas")
}

func Test_PrepareBundle_AllGasExplicit_NoEstimationNeeded(t *testing.T) {
	addr1 := common.Address{1}
	addr2 := common.Address{2}
	gas1 := hexutil.Uint64(50000)
	gas2 := hexutil.Uint64(60000)

	be := rpctest.NewBackendBuilder(t).
		WithAccount(addr1, rpctest.AccountState{Balance: big.NewInt(1e18)}).
		WithAccount(addr2, rpctest.AccountState{Balance: big.NewInt(1e18)}).
		Build()

	api := NewPublicBundleAPI(be)

	args := RPCExecutionProposal{
		RPCExecutionPlanGroup: RPCExecutionPlanGroup{
			Steps: []any{
				txEntry(ethapi.TransactionArgs{From: &addr1, To: &addr2, Nonce: rpctest.ToHexUint64(0), Gas: &gas1}),
				txEntry(ethapi.TransactionArgs{From: &addr2, To: &addr1, Nonce: rpctest.ToHexUint64(0), Gas: &gas2}),
			},
		},
	}

	result, err := api.PrepareBundle(t.Context(), args)
	require.NoError(t, err)
	require.EqualValues(t, gas1, *result.Transactions[0].Gas, "explicit gas1 must be preserved exactly")
	require.EqualValues(t, gas2, *result.Transactions[1].Gas, "explicit gas2 must be preserved exactly")
}

func Test_PrepareBundle_GasPrice_ExplicitPreserved(t *testing.T) {
	addr1 := common.Address{1}
	addr2 := common.Address{2}
	existingGasPrice := rpctest.ToHexBigInt(big.NewInt(999))

	be := rpctest.NewBackendBuilder(t).
		WithAccount(addr1, rpctest.AccountState{Balance: big.NewInt(1e18)}).
		WithAccount(addr2, rpctest.AccountState{Balance: big.NewInt(1e18)}).
		Build()

	api := NewPublicBundleAPI(be)

	args := RPCExecutionProposal{
		RPCExecutionPlanGroup: RPCExecutionPlanGroup{
			Steps: []any{
				txEntry(ethapi.TransactionArgs{From: &addr1, To: &addr2, Nonce: rpctest.ToHexUint64(0), GasPrice: existingGasPrice}),
				txEntry(ethapi.TransactionArgs{From: &addr2, To: &addr1, Nonce: rpctest.ToHexUint64(0)}),
			},
		},
	}

	result, err := api.PrepareBundle(t.Context(), args)
	require.NoError(t, err)
	require.Equal(t, existingGasPrice, result.Transactions[0].GasPrice, "explicit GasPrice must be preserved")
	hasPriceField := result.Transactions[1].GasPrice != nil || result.Transactions[1].MaxFeePerGas != nil
	require.True(t, hasPriceField, "missing gas price must be filled")
}

func Test_PrepareBundle_GasPrice_MaxFeeFilledForEIP1559(t *testing.T) {
	addr1 := common.Address{1}
	addr2 := common.Address{2}
	tip := rpctest.ToHexBigInt(big.NewInt(1e8))

	be := rpctest.NewBackendBuilder(t).
		WithAccount(addr1, rpctest.AccountState{Balance: big.NewInt(1e18)}).
		Build()

	api := NewPublicBundleAPI(be)

	args := RPCExecutionProposal{
		RPCExecutionPlanGroup: RPCExecutionPlanGroup{
			Steps: []any{
				txEntry(ethapi.TransactionArgs{
					From:                 &addr1,
					To:                   &addr2,
					Nonce:                rpctest.ToHexUint64(0),
					MaxPriorityFeePerGas: tip,
				}),
			},
		},
	}

	result, err := api.PrepareBundle(t.Context(), args)
	require.NoError(t, err)
	tx := result.Transactions[0]
	require.NotNil(t, tx.MaxFeePerGas, "MaxFeePerGas must be filled for EIP-1559 tx")
	require.Nil(t, tx.GasPrice, "GasPrice must remain nil for EIP-1559 tx")
}

func Test_PrepareBundle_GasPrice_MaxFeeExplicitPreserved(t *testing.T) {
	addr1 := common.Address{1}
	addr2 := common.Address{2}
	existingMaxFee := rpctest.ToHexBigInt(big.NewInt(888))

	be := rpctest.NewBackendBuilder(t).
		WithAccount(addr1, rpctest.AccountState{Balance: big.NewInt(1e18)}).
		Build()

	api := NewPublicBundleAPI(be)

	args := RPCExecutionProposal{
		RPCExecutionPlanGroup: RPCExecutionPlanGroup{
			Steps: []any{
				txEntry(ethapi.TransactionArgs{
					From:         &addr1,
					To:           &addr2,
					Nonce:        rpctest.ToHexUint64(0),
					MaxFeePerGas: existingMaxFee,
				}),
			},
		},
	}

	result, err := api.PrepareBundle(t.Context(), args)
	require.NoError(t, err)
	tx := result.Transactions[0]
	require.Equal(t, existingMaxFee, tx.MaxFeePerGas, "explicit MaxFeePerGas must be preserved")
	require.Nil(t, tx.GasPrice, "GasPrice must remain nil when MaxFeePerGas is set")
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

	require.EqualValues(t, currentBlock+1, result.ExecutionPlan.BlockRange.First)
	require.EqualValues(t, bundle.MaxBlockRangeLength, result.ExecutionPlan.BlockRange.Length)
}

func Test_PrepareBundle_ExplicitBlockRange_IsRespected(t *testing.T) {
	addr1 := common.Address{1}
	addr2 := common.Address{2}
	first := hexutil.Uint64(10)
	length := hexutil.Uint64(20)

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
			First:  first,
			Length: length,
		},
	}

	result, err := api.PrepareBundle(t.Context(), args)
	require.NoError(t, err)

	require.EqualValues(t, first, result.ExecutionPlan.BlockRange.First)
	require.EqualValues(t, length, result.ExecutionPlan.BlockRange.Length)
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
	require.ErrorContains(t, err, "proposed transaction is missing nonce")
}

func Test_PrepareBundle_MissingFrom_ReturnsError(t *testing.T) {
	addr2 := common.Address{2}

	be := rpctest.NewBackendBuilder(t).Build()
	api := NewPublicBundleAPI(be)

	args := RPCExecutionProposal{
		RPCExecutionPlanGroup: RPCExecutionPlanGroup{
			Steps: []any{
				txEntry(ethapi.TransactionArgs{
					To:    &addr2,
					Nonce: rpctest.ToHexUint64(0),
				}),
			},
		},
	}

	_, err := api.PrepareBundle(t.Context(), args)
	require.ErrorContains(t, err, "proposed transaction is missing from field")
}

func Test_PrepareBundle_ConflictingGasPriceFields_ReturnsError(t *testing.T) {
	addr1 := common.Address{1}
	addr2 := common.Address{2}
	gasPrice := rpctest.ToHexBigInt(big.NewInt(1e9))
	maxFee := rpctest.ToHexBigInt(big.NewInt(2e9))

	be := rpctest.NewBackendBuilder(t).
		WithAccount(addr1, rpctest.AccountState{Balance: big.NewInt(1e18)}).
		Build()
	api := NewPublicBundleAPI(be)

	args := RPCExecutionProposal{
		RPCExecutionPlanGroup: RPCExecutionPlanGroup{
			Steps: []any{
				txEntry(ethapi.TransactionArgs{
					From:         &addr1,
					To:           &addr2,
					Nonce:        rpctest.ToHexUint64(0),
					GasPrice:     gasPrice,
					MaxFeePerGas: maxFee,
				}),
			},
		},
	}

	_, err := api.PrepareBundle(t.Context(), args)
	require.ErrorContains(t, err, "proposed transaction cannot have both gasPrice and maxFeePerGas set")
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
	gas := hexutil.Uint64(21000)

	be := rpctest.NewBackendBuilder(t).
		WithAccount(addr1, rpctest.AccountState{Balance: big.NewInt(1e18)}).
		WithAccount(addr2, rpctest.AccountState{Balance: big.NewInt(1e18)}).
		Build()

	api := NewPublicBundleAPI(be)

	args := RPCExecutionProposal{
		RPCExecutionPlanGroup: RPCExecutionPlanGroup{
			OneOf: true,
			Steps: []any{
				txEntry(ethapi.TransactionArgs{From: &addr1, To: &addr2, Nonce: rpctest.ToHexUint64(0), Value: rpctest.ToHexBigInt(big.NewInt(1e15)), Gas: &gas}),
				txEntry(ethapi.TransactionArgs{From: &addr2, To: &addr1, Nonce: rpctest.ToHexUint64(0), Value: rpctest.ToHexBigInt(big.NewInt(1e15)), Gas: &gas}),
			},
		},
	}

	result, err := api.PrepareBundle(t.Context(), args)
	require.NoError(t, err)
	require.Len(t, result.Transactions, 2)

	// Single root step is the OneOf group (unwrapped since root has 1 child with no modifiers).
	require.Len(t, result.ExecutionPlan.Steps, 1)
	oneOfGroup, ok := result.ExecutionPlan.Steps[0].(RPCExecutionPlanGroup)
	require.True(t, ok)
	require.True(t, oneOfGroup.OneOf, "expected OneOf group")
	require.Len(t, oneOfGroup.Steps, 2)
}

func Test_PrepareBundle_TolerateFailed_Flag(t *testing.T) {
	addr1 := common.Address{1}
	addr2 := common.Address{2}
	gas := hexutil.Uint64(21000)

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
					Gas:   &gas,
				}, true, false),
				txEntryWithFlags(ethapi.TransactionArgs{
					From:  &addr1,
					To:    &addr2,
					Nonce: rpctest.ToHexUint64(1),
					Gas:   &gas,
				}, false, true),
			},
		},
	}

	result, err := api.PrepareBundle(t.Context(), args)
	require.NoError(t, err)
	require.Len(t, result.ExecutionPlan.Steps, 1)

	stepGroup, ok := result.ExecutionPlan.Steps[0].(RPCExecutionPlanGroup)
	require.True(t, ok, "expected step group")
	require.Len(t, stepGroup.Steps, 2)

	leafFailed, ok := stepGroup.Steps[0].(RPCExecutionStepComposable)
	require.True(t, ok, "expected leaf step")
	require.True(t, leafFailed.TolerateFailed, "TolerateFailed must be set")
	require.False(t, leafFailed.TolerateInvalid)

	leafInvalid, ok := stepGroup.Steps[1].(RPCExecutionStepComposable)
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

	gas := hexutil.Uint64(21000)
	// OneOf(AllOf(tx1, tx2), tx3-alike via addr1 again)
	args := RPCExecutionProposal{
		RPCExecutionPlanGroup: RPCExecutionPlanGroup{
			Steps: []any{
				groupEntryWithFlags(true, false,
					groupEntry(
						txEntry(ethapi.TransactionArgs{From: &addr1, To: &addr2, Nonce: rpctest.ToHexUint64(0), Value: rpctest.ToHexBigInt(big.NewInt(1e15)), Gas: &gas}),
						txEntry(ethapi.TransactionArgs{From: &addr2, To: &addr1, Nonce: rpctest.ToHexUint64(0), Value: rpctest.ToHexBigInt(big.NewInt(1e15)), Gas: &gas}),
					),
					txEntry(ethapi.TransactionArgs{From: &addr1, To: &addr2, Nonce: rpctest.ToHexUint64(1), Value: rpctest.ToHexBigInt(big.NewInt(1e15)), Gas: &gas}),
				),
			},
		},
	}

	result, err := api.PrepareBundle(t.Context(), args)
	require.NoError(t, err)
	require.Len(t, result.Transactions, 3)

	// Root: 1 element (OneOf group)
	require.Len(t, result.ExecutionPlan.Steps, 1)
	oneOf, ok := result.ExecutionPlan.Steps[0].(RPCExecutionPlanGroup)
	require.True(t, ok)
	require.True(t, oneOf.OneOf)
	require.Len(t, oneOf.Steps, 2)

	// First alt is an AllOf group
	allOf, ok := oneOf.Steps[0].(RPCExecutionPlanGroup)
	require.True(t, ok)
	require.False(t, allOf.OneOf)
	require.Len(t, allOf.Steps, 2)

	// Second alt is a leaf
	_, ok = oneOf.Steps[1].(RPCExecutionStepComposable)
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
	_, ok := result.ExecutionPlan.Steps[0].(RPCExecutionStepComposable)
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
	group, ok := result.ExecutionPlan.Steps[0].(RPCExecutionPlanGroup)
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

	gas := hexutil.Uint64(21000)
	args := RPCExecutionProposal{
		RPCExecutionPlanGroup: RPCExecutionPlanGroup{
			Steps: []any{
				groupEntryWithFlags(
					false, true, txEntry(ethapi.TransactionArgs{From: &addr1, To: &addr2, Nonce: rpctest.ToHexUint64(0), Gas: &gas}),
				),
			},
		},
	}

	result, err := api.PrepareBundle(t.Context(), args)
	require.NoError(t, err)
	require.Len(t, result.Transactions, 1)

	// TolerateFailures flag must prevent single-child unwrap.
	require.Len(t, result.ExecutionPlan.Steps, 1)
	group, ok := result.ExecutionPlan.Steps[0].(RPCExecutionPlanGroup)
	require.True(t, ok, "expected group, not leaf")
	require.False(t, group.OneOf)
	require.Len(t, group.Steps, 1)
}

func Test_PrepareBundle_SingleChildGroup_Plain_IsUnwrapped(t *testing.T) {
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

	// Plain group must collapse it's single child.
	require.Len(t, result.ExecutionPlan.Steps, 1)
	_, ok := result.ExecutionPlan.Steps[0].(RPCExecutionStepComposable)
	require.True(t, ok, "expected group, not leaf")
}

// stripBundleMarker removes the BundleOnly access-list entry from txArgs so that
// ToTransaction() + signer.Hash() reproduce the pre-injection plan hash.
// The plan hash is computed before injectPlanHashIntoAccessLists runs, so the
// marker must be stripped to match what was hashed.
func stripBundleMarker(args ethapi.TransactionArgs) ethapi.TransactionArgs {
	if args.AccessList == nil {
		return args
	}
	var clean types.AccessList
	for _, e := range *args.AccessList {
		if e.Address != bundle.BundleOnly {
			clean = append(clean, e)
		}
	}
	if len(clean) == 0 {
		args.AccessList = nil
	} else {
		args.AccessList = &clean
	}
	return args
}

// collectLeafHashes returns all RPCExecutionStepComposable.Hash values from the
// plan in depth-first left-to-right order.
func collectLeafHashes(steps []any) []common.Hash {
	var hashes []common.Hash
	for _, s := range steps {
		switch v := s.(type) {
		case RPCExecutionStepComposable:
			hashes = append(hashes, v.Hash)
		case RPCExecutionPlanGroup:
			hashes = append(hashes, collectLeafHashes(v.Steps)...)
		default:
			panic(fmt.Sprintf("unexpected step type %T", s))
		}

	}
	return hashes
}

func Test_PrepareBundle_PlanHashesMatchTransactions(t *testing.T) {
	addr1 := common.Address{1}
	addr2 := common.Address{2}
	addr3 := common.Address{3}

	be := rpctest.NewBackendBuilder(t).
		WithAccount(addr1, rpctest.AccountState{Balance: big.NewInt(1e18)}).
		WithAccount(addr2, rpctest.AccountState{Balance: big.NewInt(1e18)}).
		WithAccount(addr3, rpctest.AccountState{Balance: big.NewInt(1e18)}).
		Build()
	api := NewPublicBundleAPI(be)
	signer := types.LatestSignerForChainID(be.ChainID())

	explicitGas := hexutil.Uint64(50000)
	tip := rpctest.ToHexBigInt(big.NewInt(1e8))

	tests := map[string]struct {
		proposal    RPCExecutionProposal
		wantTxCount int
		extraCheck  func(t *testing.T, result *RPCPreparedBundle)
	}{
		"single tx AllOf": {
			proposal: RPCExecutionProposal{
				RPCExecutionPlanGroup: RPCExecutionPlanGroup{
					Steps: []any{
						txEntry(ethapi.TransactionArgs{From: &addr1, To: &addr2, Nonce: rpctest.ToHexUint64(0)}),
					},
				},
			},
			wantTxCount: 1,
		},
		"two txs AllOf different senders": {
			proposal: RPCExecutionProposal{
				RPCExecutionPlanGroup: RPCExecutionPlanGroup{
					Steps: []any{
						txEntry(ethapi.TransactionArgs{From: &addr1, To: &addr2, Nonce: rpctest.ToHexUint64(0)}),
						txEntry(ethapi.TransactionArgs{From: &addr2, To: &addr3, Nonce: rpctest.ToHexUint64(0)}),
					},
				},
			},
			wantTxCount: 2,
		},
		"three txs AllOf all senders": {
			proposal: RPCExecutionProposal{
				RPCExecutionPlanGroup: RPCExecutionPlanGroup{
					Steps: []any{
						txEntry(ethapi.TransactionArgs{From: &addr1, To: &addr2, Nonce: rpctest.ToHexUint64(0)}),
						txEntry(ethapi.TransactionArgs{From: &addr2, To: &addr3, Nonce: rpctest.ToHexUint64(0)}),
						txEntry(ethapi.TransactionArgs{From: &addr3, To: &addr1, Nonce: rpctest.ToHexUint64(0)}),
					},
				},
			},
			wantTxCount: 3,
		},
		"OneOf group different senders": {
			proposal: RPCExecutionProposal{
				RPCExecutionPlanGroup: RPCExecutionPlanGroup{
					OneOf: true,
					Steps: []any{
						txEntry(ethapi.TransactionArgs{From: &addr1, To: &addr2, Nonce: rpctest.ToHexUint64(0), Gas: &explicitGas}),
						txEntry(ethapi.TransactionArgs{From: &addr2, To: &addr1, Nonce: rpctest.ToHexUint64(0), Gas: &explicitGas}),
					},
				},
			},
			wantTxCount: 2,
		},
		"nested groups OneOf(AllOf(tx1,tx2),tx3) depth-first": {
			proposal: RPCExecutionProposal{
				RPCExecutionPlanGroup: RPCExecutionPlanGroup{
					Steps: []any{
						groupEntryWithFlags(true, false,
							groupEntry(
								txEntry(ethapi.TransactionArgs{From: &addr1, To: &addr2, Nonce: rpctest.ToHexUint64(0), Gas: &explicitGas}),
								txEntry(ethapi.TransactionArgs{From: &addr2, To: &addr3, Nonce: rpctest.ToHexUint64(0), Gas: &explicitGas}),
							),
							txEntry(ethapi.TransactionArgs{From: &addr3, To: &addr1, Nonce: rpctest.ToHexUint64(0), Gas: &explicitGas}),
						),
					},
				},
			},
			wantTxCount: 3,
		},
		"explicit gas reflected in hash": {
			proposal: RPCExecutionProposal{
				RPCExecutionPlanGroup: RPCExecutionPlanGroup{
					Steps: []any{
						txEntry(ethapi.TransactionArgs{From: &addr1, To: &addr3, Nonce: rpctest.ToHexUint64(0), Gas: &explicitGas}),
					},
				},
			},
			wantTxCount: 1,
		},
		"EIP-1559 tx bundle marker strip required": {
			proposal: RPCExecutionProposal{
				RPCExecutionPlanGroup: RPCExecutionPlanGroup{
					Steps: []any{
						txEntry(ethapi.TransactionArgs{From: &addr2, To: &addr3, Nonce: rpctest.ToHexUint64(0), MaxPriorityFeePerGas: tip}),
					},
				},
			},
			wantTxCount: 1,
			extraCheck: func(t *testing.T, result *RPCPreparedBundle) {
				planHashes := collectLeafHashes(result.ExecutionPlan.Steps)
				// For EIP-1559 (type 2), the access list is part of the signing hash.
				// Without stripping the marker the hash must differ from the plan hash.
				markerTx, err := result.Transactions[0].ToTransaction()
				require.NoError(t, err)
				withMarker := signer.Hash(markerTx)
				require.NotEqual(t, planHashes[0], withMarker, "stripping bundle marker must be necessary for EIP-1559 txs")
			},
		},
		"tolerate flags preserved alongside hash": {
			proposal: RPCExecutionProposal{
				RPCExecutionPlanGroup: RPCExecutionPlanGroup{
					Steps: []any{
						txEntryWithFlags(ethapi.TransactionArgs{From: &addr1, To: &addr2, Nonce: rpctest.ToHexUint64(0), Gas: &explicitGas}, true, false),
						txEntryWithFlags(ethapi.TransactionArgs{From: &addr2, To: &addr3, Nonce: rpctest.ToHexUint64(1), Gas: &explicitGas}, false, true),
					},
				},
			},
			wantTxCount: 2,
			extraCheck: func(t *testing.T, result *RPCPreparedBundle) {
				group := result.ExecutionPlan.Steps[0].(RPCExecutionPlanGroup)
				leaf0 := group.Steps[0].(RPCExecutionStepComposable)
				leaf1 := group.Steps[1].(RPCExecutionStepComposable)
				require.True(t, leaf0.TolerateFailed)
				require.False(t, leaf0.TolerateInvalid)
				require.False(t, leaf1.TolerateFailed)
				require.True(t, leaf1.TolerateInvalid)
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := api.PrepareBundle(t.Context(), tc.proposal)
			require.NoError(t, err)
			require.Len(t, result.Transactions, tc.wantTxCount)

			// The hash used in the execution plan shall be equal to the hash
			// of the transaction after removing the bundle marker. All referenced
			// transactions in the execution plan shall be present in the transaction list.
			expectedHashes := make([]common.Hash, len(result.Transactions))
			for i, txArgs := range result.Transactions {
				clean := stripBundleMarker(txArgs)
				tx, err := toTransactionForBundles(clean)
				require.NoError(t, err)
				expectedHashes[i] = signer.Hash(tx)
			}
			planHashes := collectLeafHashes(result.ExecutionPlan.Steps)
			require.Equal(t, expectedHashes, planHashes)

			if tc.extraCheck != nil {
				tc.extraCheck(t, result)
			}
		})
	}
}

func Test_PrepareBundle_GasEstimationCompatibility(t *testing.T) {
	addr1 := common.Address{1}
	addr2 := common.Address{2}
	gas := hexutil.Uint64(21000)

	be := rpctest.NewBackendBuilder(t).
		WithAccount(addr1, rpctest.AccountState{Balance: big.NewInt(1e18)}).
		WithAccount(addr2, rpctest.AccountState{Balance: big.NewInt(1e18)}).
		Build()
	api := NewPublicBundleAPI(be)

	tests := map[string]struct {
		args          RPCExecutionProposal
		errorContains string // empty means success expected
	}{
		// --- error cases: gas estimation needed + incompatible flags ---
		"group tolerateFailures needs gas": {
			args: RPCExecutionProposal{RPCExecutionPlanGroup: RPCExecutionPlanGroup{
				Steps: []any{
					groupEntryWithFlags(false, true,
						txEntry(ethapi.TransactionArgs{From: &addr1, To: &addr2, Nonce: rpctest.ToHexUint64(0)}),
					),
				},
			}},
			errorContains: "tolerateFailures",
		},
		"group oneOf needs gas": {
			args: RPCExecutionProposal{RPCExecutionPlanGroup: RPCExecutionPlanGroup{
				Steps: []any{
					groupEntryWithFlags(true, false,
						txEntry(ethapi.TransactionArgs{From: &addr1, To: &addr2, Nonce: rpctest.ToHexUint64(0), Gas: &gas}),
						txEntry(ethapi.TransactionArgs{From: &addr2, To: &addr1, Nonce: rpctest.ToHexUint64(0)}),
					),
				},
			}},
			errorContains: "oneOf",
		},
		"group oneOf+tolerateFailures needs gas": {
			args: RPCExecutionProposal{RPCExecutionPlanGroup: RPCExecutionPlanGroup{
				Steps: []any{
					groupEntryWithFlags(true, true,
						txEntry(ethapi.TransactionArgs{From: &addr1, To: &addr2, Nonce: rpctest.ToHexUint64(0)}),
					),
				},
			}},
			errorContains: "tolerateFailures",
		},
		"step tolerateFailed needs gas": {
			args: RPCExecutionProposal{RPCExecutionPlanGroup: RPCExecutionPlanGroup{
				Steps: []any{
					txEntryWithFlags(ethapi.TransactionArgs{From: &addr1, To: &addr2, Nonce: rpctest.ToHexUint64(0)}, true, false),
				},
			}},
			errorContains: "tolerateFailed",
		},
		"step tolerateInvalid needs gas": {
			args: RPCExecutionProposal{RPCExecutionPlanGroup: RPCExecutionPlanGroup{
				Steps: []any{
					txEntryWithFlags(ethapi.TransactionArgs{From: &addr1, To: &addr2, Nonce: rpctest.ToHexUint64(0)}, false, true),
				},
			}},
			errorContains: "tolerateInvalid",
		},
		"step tolerateFailed+tolerateInvalid needs gas": {
			args: RPCExecutionProposal{RPCExecutionPlanGroup: RPCExecutionPlanGroup{
				Steps: []any{
					txEntryWithFlags(ethapi.TransactionArgs{From: &addr1, To: &addr2, Nonce: rpctest.ToHexUint64(0)}, true, true),
				},
			}},
			errorContains: "tolerateFailed",
		},
		"nested group tolerateFailures needs gas": {
			args: RPCExecutionProposal{RPCExecutionPlanGroup: RPCExecutionPlanGroup{
				Steps: []any{
					groupEntry(
						groupEntryWithFlags(false, true,
							txEntry(ethapi.TransactionArgs{From: &addr1, To: &addr2, Nonce: rpctest.ToHexUint64(0)}),
						),
					),
				},
			}},
			errorContains: "tolerateFailures",
		},
		"root tolerateFailures needs gas": {
			args: RPCExecutionProposal{RPCExecutionPlanGroup: RPCExecutionPlanGroup{
				TolerateFailures: true,
				Steps: []any{
					txEntry(ethapi.TransactionArgs{From: &addr1, To: &addr2, Nonce: rpctest.ToHexUint64(0)}),
				},
			}},
			errorContains: "tolerateFailures",
		},
		// --- success cases: no incompatible flags, gas estimated ---
		"flat proposal no flags gas estimated": {
			args: RPCExecutionProposal{RPCExecutionPlanGroup: RPCExecutionPlanGroup{
				Steps: []any{
					txEntry(ethapi.TransactionArgs{From: &addr1, To: &addr2, Nonce: rpctest.ToHexUint64(0)}),
					txEntry(ethapi.TransactionArgs{From: &addr2, To: &addr1, Nonce: rpctest.ToHexUint64(0)}),
				},
			}},
		},
		// --- success cases: all gas explicit, no estimation needed ---
		"group tolerateFailures all gas explicit": {
			args: RPCExecutionProposal{RPCExecutionPlanGroup: RPCExecutionPlanGroup{
				Steps: []any{
					groupEntryWithFlags(false, true,
						txEntry(ethapi.TransactionArgs{From: &addr1, To: &addr2, Nonce: rpctest.ToHexUint64(0), Gas: &gas}),
					),
				},
			}},
		},
		"group oneOf all gas explicit": {
			args: RPCExecutionProposal{RPCExecutionPlanGroup: RPCExecutionPlanGroup{
				Steps: []any{
					groupEntryWithFlags(true, false,
						txEntry(ethapi.TransactionArgs{From: &addr1, To: &addr2, Nonce: rpctest.ToHexUint64(0), Gas: &gas}),
						txEntry(ethapi.TransactionArgs{From: &addr2, To: &addr1, Nonce: rpctest.ToHexUint64(0), Gas: &gas}),
					),
				},
			}},
		},
		"step tolerateFailed all gas explicit": {
			args: RPCExecutionProposal{RPCExecutionPlanGroup: RPCExecutionPlanGroup{
				Steps: []any{
					txEntryWithFlags(ethapi.TransactionArgs{From: &addr1, To: &addr2, Nonce: rpctest.ToHexUint64(0), Gas: &gas}, true, false),
				},
			}},
		},
		"step tolerateInvalid all gas explicit": {
			args: RPCExecutionProposal{RPCExecutionPlanGroup: RPCExecutionPlanGroup{
				Steps: []any{
					txEntryWithFlags(ethapi.TransactionArgs{From: &addr1, To: &addr2, Nonce: rpctest.ToHexUint64(0), Gas: &gas}, false, true),
				},
			}},
		},
		"step tolerateFailed+tolerateInvalid all gas explicit": {
			args: RPCExecutionProposal{RPCExecutionPlanGroup: RPCExecutionPlanGroup{
				Steps: []any{
					txEntryWithFlags(ethapi.TransactionArgs{From: &addr1, To: &addr2, Nonce: rpctest.ToHexUint64(0), Gas: &gas}, true, true),
				},
			}},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := api.PrepareBundle(t.Context(), tc.args)
			if tc.errorContains != "" {
				require.ErrorContains(t, err, tc.errorContains)
				require.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
			}
		})
	}
}
