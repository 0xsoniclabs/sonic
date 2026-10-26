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

package sonic

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/tests"
)

// transactionEntry is a test helper that wraps a CallArguments as a leaf PrepareBundleEntry.
func transactionEntry(transaction exec.CallArguments) RPCExecutionStepProposal {
	return transactionEntryWithFlags(transaction, false, false)
}

// transactionEntryWithFlags wraps a CallArguments as a leaf step with execution flags.
func transactionEntryWithFlags(transaction exec.CallArguments, tolerateFailed, tolerateInvalid bool) RPCExecutionStepProposal {
	return RPCExecutionStepProposal{
		TolerateFailed:  tolerateFailed,
		TolerateInvalid: tolerateInvalid,
		CallArguments:   transaction,
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
	existingAddress := common.Address{0x42}

	cases := map[string]struct {
		transactions []exec.CallArguments
		planHash     common.Hash
		check        func(t *testing.T, transactions []exec.CallArguments)
	}{
		"nil access list creates bundle-only entry": {
			transactions: []exec.CallArguments{{}},
			planHash:     common.Hash{0xab},
			check: func(t *testing.T, transactions []exec.CallArguments) {
				require.NotNil(t, transactions[0].AccessList)
				entries := *transactions[0].AccessList
				require.Len(t, entries, 1)
				require.Equal(t, bundle.BundleOnly, entries[0].Address)
				require.Equal(t, []common.Hash{{0xab}}, entries[0].StorageKeys)
			},
		},
		"existing entries appended": {
			transactions: []exec.CallArguments{{AccessList: &types.AccessList{{Address: existingAddress}}}},
			planHash:     common.Hash{0xcd},
			check: func(t *testing.T, transactions []exec.CallArguments) {
				entries := *transactions[0].AccessList
				require.Len(t, entries, 2)
				require.Equal(t, existingAddress, entries[0].Address)
				require.Equal(t, bundle.BundleOnly, entries[1].Address)
				require.Equal(t, []common.Hash{{0xcd}}, entries[1].StorageKeys)
			},
		},
		"multiple txs all injected": {
			transactions: []exec.CallArguments{{}, {}, {}},
			planHash:     common.Hash{0x01},
			check: func(t *testing.T, transactions []exec.CallArguments) {
				for index, transaction := range transactions {
					require.NotNil(t, transaction.AccessList, "tx %d missing access list", index)
					entries := *transaction.AccessList
					require.Len(t, entries, 1, "tx %d should have exactly one entry", index)
					require.Equal(t, bundle.BundleOnly, entries[0].Address)
				}
			},
		},
		"nil txs no panic": {
			transactions: nil,
			planHash:     common.Hash{},
			check:        func(t *testing.T, transactions []exec.CallArguments) {},
		},
		"empty txs no panic": {
			transactions: []exec.CallArguments{},
			planHash:     common.Hash{},
			check:        func(t *testing.T, transactions []exec.CallArguments) {},
		},
	}

	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			injectPlanHashIntoAccessLists(test.transactions, test.planHash)
			test.check(t, test.transactions)
		})
	}
}

func Test_PrepareBundle_SingleTransaction_GasAndPriceEstimated(t *testing.T) {
	account1 := common.Address{1}
	account2 := common.Address{2}

	backend := newBackendBuilder(t).
		WithAccount(account1, tests.Account{Balance: big.NewInt(1e18)}).
		Build()

	api := newBundleAPI(backend)

	arguments := RPCExecutionProposal{
		RPCExecutionPlanGroup: RPCExecutionPlanGroup{
			Steps: []any{
				transactionEntry(exec.CallArguments{
					From:      &account1,
					Recipient: &account2,
					Nonce:     toHexUint64(0),
				}),
			},
		},
	}

	result, err := api.PrepareBundle(t.Context(), arguments)
	require.NoError(t, err)
	require.Len(t, result.Transactions, 1)

	transaction := result.Transactions[0]
	require.NotNil(t, transaction.Gas, "gas must be estimated when nil")
	require.GreaterOrEqual(t, uint64(*transaction.Gas), uint64(params.TxGas))

	hasPriceField := transaction.GasPrice != nil || transaction.MaxFeePerGas != nil
	require.True(t, hasPriceField, "gas price must be set")
}

func Test_PrepareBundle_AccessListContainsConsistentPlanHash(t *testing.T) {
	account1 := common.Address{1}
	account2 := common.Address{2}

	backend := newBackendBuilder(t).
		WithAccount(account1, tests.Account{Balance: big.NewInt(1e18)}).
		Build()

	api := newBundleAPI(backend)

	arguments := RPCExecutionProposal{
		RPCExecutionPlanGroup: RPCExecutionPlanGroup{
			Steps: []any{
				transactionEntry(exec.CallArguments{From: &account1, Recipient: &account2, Nonce: toHexUint64(0), Value: toHexBigInt(big.NewInt(1e15))}),
				transactionEntry(exec.CallArguments{From: &account2, Recipient: &account1, Nonce: toHexUint64(0), Value: toHexBigInt(big.NewInt(1e15))}),
			},
		},
	}

	result, err := api.PrepareBundle(t.Context(), arguments)
	require.NoError(t, err)
	require.Len(t, result.Transactions, 2)

	var planHash common.Hash
	for index, transaction := range result.Transactions {
		require.NotNil(t, transaction.AccessList)
		for _, entry := range *transaction.AccessList {
			if entry.Address == bundle.BundleOnly {
				require.Len(t, entry.StorageKeys, 1)
				if index == 0 {
					planHash = entry.StorageKeys[0]
				} else {
					require.Equal(t, planHash, entry.StorageKeys[0], "tx %d has different plan hash", index)
				}
			}
		}
	}
}

func Test_PrepareBundle_ExplicitGasLimit_NotOverwritten(t *testing.T) {
	account1 := common.Address{1}
	account2 := common.Address{2}
	explicitGas := hexutil.Uint64(50000)

	backend := newBackendBuilder(t).
		WithAccount(account1, tests.Account{Balance: big.NewInt(1e18)}).
		Build()

	api := newBundleAPI(backend)

	arguments := RPCExecutionProposal{
		RPCExecutionPlanGroup: RPCExecutionPlanGroup{
			Steps: []any{
				transactionEntry(exec.CallArguments{
					From:      &account1,
					Recipient: &account2,
					Nonce:     toHexUint64(0),
					Gas:       &explicitGas,
				}),
			},
		},
	}

	result, err := api.PrepareBundle(t.Context(), arguments)
	require.NoError(t, err)
	require.EqualValues(t, explicitGas, *result.Transactions[0].Gas)
}

func Test_PrepareBundle_MixedGas_ExplicitPreservedMissingEstimated(t *testing.T) {
	account1 := common.Address{1}
	account2 := common.Address{2}
	explicitGas := hexutil.Uint64(50000)

	backend := newBackendBuilder(t).
		WithAccount(account1, tests.Account{Balance: big.NewInt(1e18)}).
		WithAccount(account2, tests.Account{Balance: big.NewInt(1e18)}).
		Build()

	api := newBundleAPI(backend)

	arguments := RPCExecutionProposal{
		RPCExecutionPlanGroup: RPCExecutionPlanGroup{
			Steps: []any{
				transactionEntry(exec.CallArguments{
					From:      &account1,
					Recipient: &account2,
					Nonce:     toHexUint64(0),
					Gas:       &explicitGas,
				}),
				transactionEntry(exec.CallArguments{
					From:      &account2,
					Recipient: &account1,
					Nonce:     toHexUint64(0),
				}),
			},
		},
	}

	result, err := api.PrepareBundle(t.Context(), arguments)
	require.NoError(t, err)
	require.Len(t, result.Transactions, 2)

	require.EqualValues(t, explicitGas, *result.Transactions[0].Gas, "explicit gas must be preserved")
	require.NotNil(t, result.Transactions[1].Gas, "missing gas must be estimated")
	require.GreaterOrEqual(t, uint64(*result.Transactions[1].Gas), uint64(params.TxGas), "estimated gas must be at least base tx gas")
}

func Test_PrepareBundle_AllGasExplicit_NoEstimationNeeded(t *testing.T) {
	account1 := common.Address{1}
	account2 := common.Address{2}
	gas1 := hexutil.Uint64(50000)
	gas2 := hexutil.Uint64(60000)

	backend := newBackendBuilder(t).
		WithAccount(account1, tests.Account{Balance: big.NewInt(1e18)}).
		WithAccount(account2, tests.Account{Balance: big.NewInt(1e18)}).
		Build()

	api := newBundleAPI(backend)

	arguments := RPCExecutionProposal{
		RPCExecutionPlanGroup: RPCExecutionPlanGroup{
			Steps: []any{
				transactionEntry(exec.CallArguments{From: &account1, Recipient: &account2, Nonce: toHexUint64(0), Gas: &gas1}),
				transactionEntry(exec.CallArguments{From: &account2, Recipient: &account1, Nonce: toHexUint64(0), Gas: &gas2}),
			},
		},
	}

	result, err := api.PrepareBundle(t.Context(), arguments)
	require.NoError(t, err)
	require.EqualValues(t, gas1, *result.Transactions[0].Gas, "explicit gas1 must be preserved exactly")
	require.EqualValues(t, gas2, *result.Transactions[1].Gas, "explicit gas2 must be preserved exactly")
}

func Test_PrepareBundle_GasPrice_ExplicitPreserved(t *testing.T) {
	account1 := common.Address{1}
	account2 := common.Address{2}
	existingGasPrice := toHexBigInt(big.NewInt(999))

	backend := newBackendBuilder(t).
		WithAccount(account1, tests.Account{Balance: big.NewInt(1e18)}).
		WithAccount(account2, tests.Account{Balance: big.NewInt(1e18)}).
		Build()

	api := newBundleAPI(backend)

	arguments := RPCExecutionProposal{
		RPCExecutionPlanGroup: RPCExecutionPlanGroup{
			Steps: []any{
				transactionEntry(exec.CallArguments{From: &account1, Recipient: &account2, Nonce: toHexUint64(0), GasPrice: existingGasPrice}),
				transactionEntry(exec.CallArguments{From: &account2, Recipient: &account1, Nonce: toHexUint64(0)}),
			},
		},
	}

	result, err := api.PrepareBundle(t.Context(), arguments)
	require.NoError(t, err)
	require.Equal(t, existingGasPrice, result.Transactions[0].GasPrice, "explicit GasPrice must be preserved")
	hasPriceField := result.Transactions[1].GasPrice != nil || result.Transactions[1].MaxFeePerGas != nil
	require.True(t, hasPriceField, "missing gas price must be filled")
}

// Test_PrepareBundle_GasPrice_MaxFeeFilledForEIP1559 gives the gas: the
// reference's test chain had no base fee, under which a tip alone could be
// estimated; on a chain with a base fee the tip above a fee cap of zero
// fails the estimate, in the module as in the reference.
func Test_PrepareBundle_GasPrice_MaxFeeFilledForEIP1559(t *testing.T) {
	account1 := common.Address{1}
	account2 := common.Address{2}
	tip := toHexBigInt(big.NewInt(1e8))
	gas := hexutil.Uint64(21000)

	backend := newBackendBuilder(t).
		WithAccount(account1, tests.Account{Balance: big.NewInt(1e18)}).
		Build()

	api := newBundleAPI(backend)

	arguments := RPCExecutionProposal{
		RPCExecutionPlanGroup: RPCExecutionPlanGroup{
			Steps: []any{
				transactionEntry(exec.CallArguments{
					From:                 &account1,
					Recipient:            &account2,
					Nonce:                toHexUint64(0),
					MaxPriorityFeePerGas: tip,
					Gas:                  &gas,
				}),
			},
		},
	}

	result, err := api.PrepareBundle(t.Context(), arguments)
	require.NoError(t, err)
	transaction := result.Transactions[0]
	require.NotNil(t, transaction.MaxFeePerGas, "MaxFeePerGas must be filled for EIP-1559 tx")
	require.Nil(t, transaction.GasPrice, "GasPrice must remain nil for EIP-1559 tx")
}

func Test_PrepareBundle_GasPrice_MaxFeeExplicitPreserved(t *testing.T) {
	account1 := common.Address{1}
	account2 := common.Address{2}
	existingMaxFee := toHexBigInt(big.NewInt(888))

	backend := newBackendBuilder(t).
		WithAccount(account1, tests.Account{Balance: big.NewInt(1e18)}).
		Build()

	api := newBundleAPI(backend)

	arguments := RPCExecutionProposal{
		RPCExecutionPlanGroup: RPCExecutionPlanGroup{
			Steps: []any{
				transactionEntry(exec.CallArguments{
					From:         &account1,
					Recipient:    &account2,
					Nonce:        toHexUint64(0),
					MaxFeePerGas: existingMaxFee,
				}),
			},
		},
	}

	result, err := api.PrepareBundle(t.Context(), arguments)
	require.NoError(t, err)
	transaction := result.Transactions[0]
	require.Equal(t, existingMaxFee, transaction.MaxFeePerGas, "explicit MaxFeePerGas must be preserved")
	require.Nil(t, transaction.GasPrice, "GasPrice must remain nil when MaxFeePerGas is set")
}

func Test_PrepareBundle_DefaultBlockRange_IsCurrentBlockPlusOne(t *testing.T) {
	account1 := common.Address{1}
	account2 := common.Address{2}

	backend := newBackendBuilder(t).
		WithAccount(account1, tests.Account{Balance: big.NewInt(1e18)}).
		Build()

	api := newBundleAPI(backend)

	currentBlock := backend.CurrentBlock().NumberU64()

	arguments := RPCExecutionProposal{
		RPCExecutionPlanGroup: RPCExecutionPlanGroup{
			Steps: []any{
				transactionEntry(exec.CallArguments{
					From:      &account1,
					Recipient: &account2,
					Nonce:     toHexUint64(0),
				}),
			},
		},
	}

	result, err := api.PrepareBundle(t.Context(), arguments)
	require.NoError(t, err)

	require.EqualValues(t, currentBlock+1, result.ExecutionPlan.BlockRange.First)
	require.EqualValues(t, bundle.MaxBlockRangeLength, result.ExecutionPlan.BlockRange.Length)
}

func Test_PrepareBundle_ExplicitBlockRange_IsRespected(t *testing.T) {
	account1 := common.Address{1}
	account2 := common.Address{2}
	first := hexutil.Uint64(10)
	length := hexutil.Uint64(20)

	backend := newBackendBuilder(t).
		WithAccount(account1, tests.Account{Balance: big.NewInt(1e18)}).
		Build()

	api := newBundleAPI(backend)

	arguments := RPCExecutionProposal{
		RPCExecutionPlanGroup: RPCExecutionPlanGroup{
			Steps: []any{
				transactionEntry(exec.CallArguments{
					From:      &account1,
					Recipient: &account2,
					Nonce:     toHexUint64(0),
				}),
			},
		},
		BlockRange: &RPCRange{
			First:  first,
			Length: length,
		},
	}

	result, err := api.PrepareBundle(t.Context(), arguments)
	require.NoError(t, err)

	require.EqualValues(t, first, result.ExecutionPlan.BlockRange.First)
	require.EqualValues(t, length, result.ExecutionPlan.BlockRange.Length)
}

func Test_PrepareBundle_MissingNonce_ReturnsError(t *testing.T) {
	account1 := common.Address{1}
	account2 := common.Address{2}

	backend := newBackendBuilder(t).
		WithAccount(account1, tests.Account{Balance: big.NewInt(1e18)}).
		Build()

	api := newBundleAPI(backend)

	arguments := RPCExecutionProposal{
		RPCExecutionPlanGroup: RPCExecutionPlanGroup{
			Steps: []any{
				transactionEntry(exec.CallArguments{
					From:      &account1,
					Recipient: &account2,
				}),
			},
		},
	}

	_, err := api.PrepareBundle(t.Context(), arguments)
	require.ErrorContains(t, err, "proposed transaction is missing nonce")
}

func Test_PrepareBundle_MissingFrom_ReturnsError(t *testing.T) {
	account2 := common.Address{2}

	backend := newBackendBuilder(t).Build()
	api := newBundleAPI(backend)

	arguments := RPCExecutionProposal{
		RPCExecutionPlanGroup: RPCExecutionPlanGroup{
			Steps: []any{
				transactionEntry(exec.CallArguments{
					Recipient: &account2,
					Nonce:     toHexUint64(0),
				}),
			},
		},
	}

	_, err := api.PrepareBundle(t.Context(), arguments)
	require.ErrorContains(t, err, "proposed transaction is missing from field")
}

func Test_PrepareBundle_ConflictingGasPriceFields_ReturnsError(t *testing.T) {
	account1 := common.Address{1}
	account2 := common.Address{2}
	gasPrice := toHexBigInt(big.NewInt(1e9))
	maxFee := toHexBigInt(big.NewInt(2e9))

	backend := newBackendBuilder(t).
		WithAccount(account1, tests.Account{Balance: big.NewInt(1e18)}).
		Build()
	api := newBundleAPI(backend)

	arguments := RPCExecutionProposal{
		RPCExecutionPlanGroup: RPCExecutionPlanGroup{
			Steps: []any{
				transactionEntry(exec.CallArguments{
					From:         &account1,
					Recipient:    &account2,
					Nonce:        toHexUint64(0),
					GasPrice:     gasPrice,
					MaxFeePerGas: maxFee,
				}),
			},
		},
	}

	_, err := api.PrepareBundle(t.Context(), arguments)
	require.ErrorContains(t, err, "proposed transaction cannot have both gasPrice and maxFeePerGas set")
}

func Test_PrepareBundle_EmptyTransactions_ReturnsEmptyBundle(t *testing.T) {
	backend := newBackendBuilder(t).Build()
	api := newBundleAPI(backend)

	arguments := RPCExecutionProposal{
		RPCExecutionPlanGroup: RPCExecutionPlanGroup{
			Steps: []any{},
		},
	}
	result, err := api.PrepareBundle(t.Context(), arguments)
	require.ErrorContains(t, err, "proposed group must include at least one step")
	require.Nil(t, result)
}

func Test_PrepareBundle_OneOfGroup_BuildsOneOfPlan(t *testing.T) {
	account1 := common.Address{1}
	account2 := common.Address{2}
	gas := hexutil.Uint64(21000)

	backend := newBackendBuilder(t).
		WithAccount(account1, tests.Account{Balance: big.NewInt(1e18)}).
		WithAccount(account2, tests.Account{Balance: big.NewInt(1e18)}).
		Build()

	api := newBundleAPI(backend)

	arguments := RPCExecutionProposal{
		RPCExecutionPlanGroup: RPCExecutionPlanGroup{
			OneOf: true,
			Steps: []any{
				transactionEntry(exec.CallArguments{From: &account1, Recipient: &account2, Nonce: toHexUint64(0), Value: toHexBigInt(big.NewInt(1e15)), Gas: &gas}),
				transactionEntry(exec.CallArguments{From: &account2, Recipient: &account1, Nonce: toHexUint64(0), Value: toHexBigInt(big.NewInt(1e15)), Gas: &gas}),
			},
		},
	}

	result, err := api.PrepareBundle(t.Context(), arguments)
	require.NoError(t, err)
	require.Len(t, result.Transactions, 2)

	require.True(t, result.ExecutionPlan.OneOf)
	require.Len(t, result.ExecutionPlan.Steps, 2)
}

func Test_PrepareBundle_TolerateFailed_Flag(t *testing.T) {
	account1 := common.Address{1}
	account2 := common.Address{2}
	gas := hexutil.Uint64(21000)

	backend := newBackendBuilder(t).
		WithAccount(account1, tests.Account{Balance: big.NewInt(1e18)}).
		Build()

	api := newBundleAPI(backend)

	arguments := RPCExecutionProposal{
		RPCExecutionPlanGroup: RPCExecutionPlanGroup{
			Steps: []any{
				transactionEntryWithFlags(exec.CallArguments{
					From:      &account1,
					Recipient: &account2,
					Nonce:     toHexUint64(0),
					Gas:       &gas,
				}, true, false),
				transactionEntryWithFlags(exec.CallArguments{
					From:      &account1,
					Recipient: &account2,
					Nonce:     toHexUint64(1),
					Gas:       &gas,
				}, false, true),
			},
		},
	}

	result, err := api.PrepareBundle(t.Context(), arguments)
	require.NoError(t, err)
	stepGroup := result.ExecutionPlan.RPCExecutionPlanGroup
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
	account1 := common.Address{1}
	account2 := common.Address{2}

	backend := newBackendBuilder(t).
		WithAccount(account1, tests.Account{Balance: big.NewInt(1e18)}).
		WithAccount(account2, tests.Account{Balance: big.NewInt(1e18)}).
		Build()

	api := newBundleAPI(backend)

	gas := hexutil.Uint64(21000)
	// OneOf(AllOf(tx1, tx2), tx3-alike via addr1 again)
	arguments := RPCExecutionProposal{
		RPCExecutionPlanGroup: RPCExecutionPlanGroup{
			Steps: []any{
				groupEntryWithFlags(true, false,
					groupEntry(
						transactionEntry(exec.CallArguments{From: &account1, Recipient: &account2, Nonce: toHexUint64(0), Value: toHexBigInt(big.NewInt(1e15)), Gas: &gas}),
						transactionEntry(exec.CallArguments{From: &account2, Recipient: &account1, Nonce: toHexUint64(0), Value: toHexBigInt(big.NewInt(1e15)), Gas: &gas}),
					),
					transactionEntry(exec.CallArguments{From: &account1, Recipient: &account2, Nonce: toHexUint64(1), Value: toHexBigInt(big.NewInt(1e15)), Gas: &gas}),
				),
			},
		},
	}

	result, err := api.PrepareBundle(t.Context(), arguments)
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

func Test_PrepareBundle_FlatTransactions_SingleTransaction(t *testing.T) {
	account1 := common.Address{1}
	account2 := common.Address{2}

	backend := newBackendBuilder(t).
		WithAccount(account1, tests.Account{Balance: big.NewInt(1e18)}).
		WithAccount(account2, tests.Account{Balance: big.NewInt(1e18)}).
		Build()

	api := newBundleAPI(backend)

	transaction := exec.CallArguments{From: &account1, Recipient: &account2, Nonce: toHexUint64(0), Value: toHexBigInt(big.NewInt(1e15))}
	arguments := RPCExecutionProposal{
		RPCExecutionPlanGroup: RPCExecutionPlanGroup{
			Steps: []any{
				transactionEntry(transaction),
			},
		},
	}

	result, err := api.PrepareBundle(t.Context(), arguments)
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

func Test_PrepareBundle_FlatTransactions_MultipleTransactions(t *testing.T) {
	account1 := common.Address{1}
	account2 := common.Address{2}

	backend := newBackendBuilder(t).
		WithAccount(account1, tests.Account{Balance: big.NewInt(1e18)}).
		WithAccount(account2, tests.Account{Balance: big.NewInt(1e18)}).
		Build()

	api := newBundleAPI(backend)

	arguments := RPCExecutionProposal{
		RPCExecutionPlanGroup: RPCExecutionPlanGroup{
			Steps: []any{
				transactionEntry(exec.CallArguments{From: &account1, Recipient: &account2, Nonce: toHexUint64(0), Value: toHexBigInt(big.NewInt(1e15))}),
				transactionEntry(exec.CallArguments{From: &account2, Recipient: &account1, Nonce: toHexUint64(0), Value: toHexBigInt(big.NewInt(1e15))}),
			},
		},
	}

	result, err := api.PrepareBundle(t.Context(), arguments)
	require.NoError(t, err)
	require.Len(t, result.Transactions, 2)

	// BundleOnly injected into every tx
	for index, transaction := range result.Transactions {
		require.NotNil(t, transaction.AccessList, "tx %d missing access list", index)
		found := false
		for _, entry := range *transaction.AccessList {
			if entry.Address == bundle.BundleOnly {
				found = true
				break
			}
		}
		require.True(t, found, "tx %d missing BundleOnly marker", index)
	}

	require.False(t, result.ExecutionPlan.OneOf)
	require.Len(t, result.ExecutionPlan.Steps, 2)
}

func Test_PrepareBundle_FlatTransactions_OrderPreserved(t *testing.T) {
	account1 := common.Address{1}
	account2 := common.Address{2}

	backend := newBackendBuilder(t).
		WithAccount(account1, tests.Account{Balance: big.NewInt(1e18)}).
		WithAccount(account2, tests.Account{Balance: big.NewInt(1e18)}).
		Build()

	api := newBundleAPI(backend)
	arguments := RPCExecutionProposal{
		RPCExecutionPlanGroup: RPCExecutionPlanGroup{
			Steps: []any{
				transactionEntry(exec.CallArguments{From: &account1, Recipient: &account2, Nonce: toHexUint64(0), Value: toHexBigInt(big.NewInt(1e15))}),
				transactionEntry(exec.CallArguments{From: &account2, Recipient: &account1, Nonce: toHexUint64(0), Value: toHexBigInt(big.NewInt(1e15))}),
			},
		},
	}

	result, err := api.PrepareBundle(t.Context(), arguments)
	require.NoError(t, err)
	require.Len(t, result.Transactions, 2)

	require.Equal(t, &account1, result.Transactions[0].From)
	require.Equal(t, &account2, result.Transactions[0].Recipient)
	require.Equal(t, &account2, result.Transactions[1].From)
	require.Equal(t, &account1, result.Transactions[1].Recipient)
}

func Test_PrepareBundle_SingleChildGroup_NotUnwrapped(t *testing.T) {
	account1 := common.Address{1}
	account2 := common.Address{2}
	gas := hexutil.Uint64(21000)
	transaction := transactionEntry(exec.CallArguments{From: &account1, Recipient: &account2, Nonce: toHexUint64(0), Gas: &gas})

	cases := map[string]RPCExecutionPlanGroup{
		"plain":               groupEntry(transaction),
		"one-of":              groupEntryWithFlags(true, false, transaction),
		"tolerating failures": groupEntryWithFlags(false, true, transaction),
	}

	for name, group := range cases {
		t.Run(name, func(t *testing.T) {
			backend := newBackendBuilder(t).
				WithAccount(account1, tests.Account{Balance: big.NewInt(1e18)}).
				Build()
			api := newBundleAPI(backend)

			arguments := RPCExecutionProposal{RPCExecutionPlanGroup: groupEntry(group)}
			result, err := api.PrepareBundle(t.Context(), arguments)
			require.NoError(t, err)
			require.Len(t, result.Transactions, 1)

			require.Len(t, result.ExecutionPlan.Steps, 1)
			got, ok := result.ExecutionPlan.Steps[0].(RPCExecutionPlanGroup)
			require.True(t, ok)
			require.Equal(t, group.OneOf, got.OneOf)
			require.Equal(t, group.TolerateFailures, got.TolerateFailures)
			require.Len(t, got.Steps, 1)
		})
	}
}

// stripBundleMarker removes the BundleOnly access-list entry from txArgs so that
// ToTransaction() + signer.Hash() reproduce the pre-injection plan hash.
// The plan hash is computed before injectPlanHashIntoAccessLists runs, so the
// marker must be stripped to match what was hashed.
func stripBundleMarker(arguments exec.CallArguments) exec.CallArguments {
	if arguments.AccessList == nil {
		return arguments
	}
	var clean types.AccessList
	for _, entry := range *arguments.AccessList {
		if entry.Address != bundle.BundleOnly {
			clean = append(clean, entry)
		}
	}
	if len(clean) == 0 {
		arguments.AccessList = nil
	} else {
		arguments.AccessList = &clean
	}
	return arguments
}

// collectLeafHashes returns all RPCExecutionStepComposable.Hash values from the
// plan in depth-first left-to-right order.
func collectLeafHashes(steps []any) []common.Hash {
	var hashes []common.Hash
	for _, step := range steps {
		switch typed := step.(type) {
		case RPCExecutionStepComposable:
			hashes = append(hashes, typed.Hash)
		case RPCExecutionPlanGroup:
			hashes = append(hashes, collectLeafHashes(typed.Steps)...)
		default:
			panic(fmt.Sprintf("unexpected step type %T", step))
		}

	}
	return hashes
}

func Test_PrepareBundle_PlanHashesMatchTransactions(t *testing.T) {
	account1 := common.Address{1}
	account2 := common.Address{2}
	account3 := common.Address{3}

	backend := newBackendBuilder(t).
		WithAccount(account1, tests.Account{Balance: big.NewInt(1e18)}).
		WithAccount(account2, tests.Account{Balance: big.NewInt(1e18)}).
		WithAccount(account3, tests.Account{Balance: big.NewInt(1e18)}).
		Build()
	api := newBundleAPI(backend)
	signer := types.LatestSignerForChainID(backend.ChainID())

	explicitGas := hexutil.Uint64(50000)
	tip := toHexBigInt(big.NewInt(1e8))

	cases := map[string]struct {
		proposal             RPCExecutionProposal
		wantTransactionCount int
		extraCheck           func(t *testing.T, result *RPCPreparedBundle)
	}{
		"single tx AllOf": {
			proposal: RPCExecutionProposal{
				RPCExecutionPlanGroup: RPCExecutionPlanGroup{
					Steps: []any{
						transactionEntry(exec.CallArguments{From: &account1, Recipient: &account2, Nonce: toHexUint64(0)}),
					},
				},
			},
			wantTransactionCount: 1,
		},
		"two txs AllOf different senders": {
			proposal: RPCExecutionProposal{
				RPCExecutionPlanGroup: RPCExecutionPlanGroup{
					Steps: []any{
						transactionEntry(exec.CallArguments{From: &account1, Recipient: &account2, Nonce: toHexUint64(0)}),
						transactionEntry(exec.CallArguments{From: &account2, Recipient: &account3, Nonce: toHexUint64(0)}),
					},
				},
			},
			wantTransactionCount: 2,
		},
		"three txs AllOf all senders": {
			proposal: RPCExecutionProposal{
				RPCExecutionPlanGroup: RPCExecutionPlanGroup{
					Steps: []any{
						transactionEntry(exec.CallArguments{From: &account1, Recipient: &account2, Nonce: toHexUint64(0)}),
						transactionEntry(exec.CallArguments{From: &account2, Recipient: &account3, Nonce: toHexUint64(0)}),
						transactionEntry(exec.CallArguments{From: &account3, Recipient: &account1, Nonce: toHexUint64(0)}),
					},
				},
			},
			wantTransactionCount: 3,
		},
		"OneOf group different senders": {
			proposal: RPCExecutionProposal{
				RPCExecutionPlanGroup: RPCExecutionPlanGroup{
					OneOf: true,
					Steps: []any{
						transactionEntry(exec.CallArguments{From: &account1, Recipient: &account2, Nonce: toHexUint64(0), Gas: &explicitGas}),
						transactionEntry(exec.CallArguments{From: &account2, Recipient: &account1, Nonce: toHexUint64(0), Gas: &explicitGas}),
					},
				},
			},
			wantTransactionCount: 2,
		},
		"nested groups OneOf(AllOf(tx1,tx2),tx3) depth-first": {
			proposal: RPCExecutionProposal{
				RPCExecutionPlanGroup: RPCExecutionPlanGroup{
					Steps: []any{
						groupEntryWithFlags(true, false,
							groupEntry(
								transactionEntry(exec.CallArguments{From: &account1, Recipient: &account2, Nonce: toHexUint64(0), Gas: &explicitGas}),
								transactionEntry(exec.CallArguments{From: &account2, Recipient: &account3, Nonce: toHexUint64(0), Gas: &explicitGas}),
							),
							transactionEntry(exec.CallArguments{From: &account3, Recipient: &account1, Nonce: toHexUint64(0), Gas: &explicitGas}),
						),
					},
				},
			},
			wantTransactionCount: 3,
		},
		"explicit gas reflected in hash": {
			proposal: RPCExecutionProposal{
				RPCExecutionPlanGroup: RPCExecutionPlanGroup{
					Steps: []any{
						transactionEntry(exec.CallArguments{From: &account1, Recipient: &account3, Nonce: toHexUint64(0), Gas: &explicitGas}),
					},
				},
			},
			wantTransactionCount: 1,
		},
		"EIP-1559 tx bundle marker strip required": {
			proposal: RPCExecutionProposal{
				RPCExecutionPlanGroup: RPCExecutionPlanGroup{
					Steps: []any{
						// The gas is given for the reason of
						// Test_PrepareBundle_GasPrice_MaxFeeFilledForEIP1559.
						transactionEntry(exec.CallArguments{From: &account2, Recipient: &account3, Nonce: toHexUint64(0), MaxPriorityFeePerGas: tip, Gas: &explicitGas}),
					},
				},
			},
			wantTransactionCount: 1,
			extraCheck: func(t *testing.T, result *RPCPreparedBundle) {
				planHashes := collectLeafHashes(result.ExecutionPlan.Steps)
				// For EIP-1559 (type 2), the access list is part of the signing hash.
				// Without stripping the marker the hash must differ from the plan hash.
				markerTransaction, err := result.Transactions[0].ToTransaction()
				require.NoError(t, err)
				withMarker := signer.Hash(markerTransaction)
				require.NotEqual(t, planHashes[0], withMarker, "stripping bundle marker must be necessary for EIP-1559 txs")
			},
		},
		"tolerate flags preserved alongside hash": {
			proposal: RPCExecutionProposal{
				RPCExecutionPlanGroup: RPCExecutionPlanGroup{
					Steps: []any{
						transactionEntryWithFlags(exec.CallArguments{From: &account1, Recipient: &account2, Nonce: toHexUint64(0), Gas: &explicitGas}, true, false),
						transactionEntryWithFlags(exec.CallArguments{From: &account2, Recipient: &account3, Nonce: toHexUint64(1), Gas: &explicitGas}, false, true),
					},
				},
			},
			wantTransactionCount: 2,
			extraCheck: func(t *testing.T, result *RPCPreparedBundle) {
				group := result.ExecutionPlan.RPCExecutionPlanGroup
				leaf0 := group.Steps[0].(RPCExecutionStepComposable)
				leaf1 := group.Steps[1].(RPCExecutionStepComposable)
				require.True(t, leaf0.TolerateFailed)
				require.False(t, leaf0.TolerateInvalid)
				require.False(t, leaf1.TolerateFailed)
				require.True(t, leaf1.TolerateInvalid)
			},
		},
	}

	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			result, err := api.PrepareBundle(t.Context(), test.proposal)
			require.NoError(t, err)
			require.Len(t, result.Transactions, test.wantTransactionCount)

			// The hash used in the execution plan shall be equal to the hash
			// of the transaction after removing the bundle marker. All referenced
			// transactions in the execution plan shall be present in the transaction list.
			expectedHashes := make([]common.Hash, len(result.Transactions))
			for index, arguments := range result.Transactions {
				clean := stripBundleMarker(arguments)
				transaction, err := toTransactionForBundles(clean)
				require.NoError(t, err)
				expectedHashes[index] = signer.Hash(transaction)
			}
			planHashes := collectLeafHashes(result.ExecutionPlan.Steps)
			require.Equal(t, expectedHashes, planHashes)

			if test.extraCheck != nil {
				test.extraCheck(t, result)
			}
		})
	}
}

func Test_PrepareBundle_GasEstimationCompatibility(t *testing.T) {
	account1 := common.Address{1}
	account2 := common.Address{2}
	gas := hexutil.Uint64(21000)

	backend := newBackendBuilder(t).
		WithAccount(account1, tests.Account{Balance: big.NewInt(1e18)}).
		WithAccount(account2, tests.Account{Balance: big.NewInt(1e18)}).
		Build()
	api := newBundleAPI(backend)

	cases := map[string]struct {
		arguments     RPCExecutionProposal
		errorContains string // empty means success expected
	}{
		// --- error cases: gas estimation needed + incompatible flags ---
		"group tolerateFailures needs gas": {
			arguments: RPCExecutionProposal{RPCExecutionPlanGroup: RPCExecutionPlanGroup{
				Steps: []any{
					groupEntryWithFlags(false, true,
						transactionEntry(exec.CallArguments{From: &account1, Recipient: &account2, Nonce: toHexUint64(0)}),
					),
				},
			}},
			errorContains: "tolerateFailures",
		},
		"group oneOf needs gas": {
			arguments: RPCExecutionProposal{RPCExecutionPlanGroup: RPCExecutionPlanGroup{
				Steps: []any{
					groupEntryWithFlags(true, false,
						transactionEntry(exec.CallArguments{From: &account1, Recipient: &account2, Nonce: toHexUint64(0), Gas: &gas}),
						transactionEntry(exec.CallArguments{From: &account2, Recipient: &account1, Nonce: toHexUint64(0)}),
					),
				},
			}},
			errorContains: "oneOf",
		},
		"group oneOf+tolerateFailures needs gas": {
			arguments: RPCExecutionProposal{RPCExecutionPlanGroup: RPCExecutionPlanGroup{
				Steps: []any{
					groupEntryWithFlags(true, true,
						transactionEntry(exec.CallArguments{From: &account1, Recipient: &account2, Nonce: toHexUint64(0)}),
					),
				},
			}},
			errorContains: "tolerateFailures",
		},
		"step tolerateFailed needs gas": {
			arguments: RPCExecutionProposal{RPCExecutionPlanGroup: RPCExecutionPlanGroup{
				Steps: []any{
					transactionEntryWithFlags(exec.CallArguments{From: &account1, Recipient: &account2, Nonce: toHexUint64(0)}, true, false),
				},
			}},
			errorContains: "tolerateFailed",
		},
		"step tolerateInvalid needs gas": {
			arguments: RPCExecutionProposal{RPCExecutionPlanGroup: RPCExecutionPlanGroup{
				Steps: []any{
					transactionEntryWithFlags(exec.CallArguments{From: &account1, Recipient: &account2, Nonce: toHexUint64(0)}, false, true),
				},
			}},
			errorContains: "tolerateInvalid",
		},
		"step tolerateFailed+tolerateInvalid needs gas": {
			arguments: RPCExecutionProposal{RPCExecutionPlanGroup: RPCExecutionPlanGroup{
				Steps: []any{
					transactionEntryWithFlags(exec.CallArguments{From: &account1, Recipient: &account2, Nonce: toHexUint64(0)}, true, true),
				},
			}},
			errorContains: "tolerateFailed",
		},
		"nested group tolerateFailures needs gas": {
			arguments: RPCExecutionProposal{RPCExecutionPlanGroup: RPCExecutionPlanGroup{
				Steps: []any{
					groupEntry(
						groupEntryWithFlags(false, true,
							transactionEntry(exec.CallArguments{From: &account1, Recipient: &account2, Nonce: toHexUint64(0)}),
						),
					),
				},
			}},
			errorContains: "tolerateFailures",
		},
		"root tolerateFailures needs gas": {
			arguments: RPCExecutionProposal{RPCExecutionPlanGroup: RPCExecutionPlanGroup{
				TolerateFailures: true,
				Steps: []any{
					transactionEntry(exec.CallArguments{From: &account1, Recipient: &account2, Nonce: toHexUint64(0)}),
				},
			}},
			errorContains: "tolerateFailures",
		},
		// --- success cases: no incompatible flags, gas estimated ---
		"flat proposal no flags gas estimated": {
			arguments: RPCExecutionProposal{RPCExecutionPlanGroup: RPCExecutionPlanGroup{
				Steps: []any{
					transactionEntry(exec.CallArguments{From: &account1, Recipient: &account2, Nonce: toHexUint64(0)}),
					transactionEntry(exec.CallArguments{From: &account2, Recipient: &account1, Nonce: toHexUint64(0)}),
				},
			}},
		},
		// --- success cases: all gas explicit, no estimation needed ---
		"group tolerateFailures all gas explicit": {
			arguments: RPCExecutionProposal{RPCExecutionPlanGroup: RPCExecutionPlanGroup{
				Steps: []any{
					groupEntryWithFlags(false, true,
						transactionEntry(exec.CallArguments{From: &account1, Recipient: &account2, Nonce: toHexUint64(0), Gas: &gas}),
					),
				},
			}},
		},
		"group oneOf all gas explicit": {
			arguments: RPCExecutionProposal{RPCExecutionPlanGroup: RPCExecutionPlanGroup{
				Steps: []any{
					groupEntryWithFlags(true, false,
						transactionEntry(exec.CallArguments{From: &account1, Recipient: &account2, Nonce: toHexUint64(0), Gas: &gas}),
						transactionEntry(exec.CallArguments{From: &account2, Recipient: &account1, Nonce: toHexUint64(0), Gas: &gas}),
					),
				},
			}},
		},
		"step tolerateFailed all gas explicit": {
			arguments: RPCExecutionProposal{RPCExecutionPlanGroup: RPCExecutionPlanGroup{
				Steps: []any{
					transactionEntryWithFlags(exec.CallArguments{From: &account1, Recipient: &account2, Nonce: toHexUint64(0), Gas: &gas}, true, false),
				},
			}},
		},
		"step tolerateInvalid all gas explicit": {
			arguments: RPCExecutionProposal{RPCExecutionPlanGroup: RPCExecutionPlanGroup{
				Steps: []any{
					transactionEntryWithFlags(exec.CallArguments{From: &account1, Recipient: &account2, Nonce: toHexUint64(0), Gas: &gas}, false, true),
				},
			}},
		},
		"step tolerateFailed+tolerateInvalid all gas explicit": {
			arguments: RPCExecutionProposal{RPCExecutionPlanGroup: RPCExecutionPlanGroup{
				Steps: []any{
					transactionEntryWithFlags(exec.CallArguments{From: &account1, Recipient: &account2, Nonce: toHexUint64(0), Gas: &gas}, true, true),
				},
			}},
		},
	}

	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			result, err := api.PrepareBundle(t.Context(), test.arguments)
			if test.errorContains != "" {
				require.ErrorContains(t, err, test.errorContains)
				require.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
			}
		})
	}
}

func TestPrepareBundle_ReturnsError_WhenProposalDepthExceedsMaximum(t *testing.T) {
	for _, depth := range []int{
		1,
		2,
		bundle.MaxGroupNestingDepth - 1,
		bundle.MaxGroupNestingDepth,
		bundle.MaxGroupNestingDepth + 1,
		1024,
		4999, // maximum json encodable nesting, with 10k nested json groups
	} {
		t.Run(fmt.Sprintf("depth=%d", depth), func(t *testing.T) {

			account1 := common.Address{1}
			account2 := common.Address{2}
			gas := hexutil.Uint64(21000)

			backend := newBackendBuilder(t).
				WithAccount(account1, tests.Account{Balance: big.NewInt(1e18)}).
				Build()
			api := newBundleAPI(backend)

			proposal := makeNestedBundleProposal(
				depth,
				exec.CallArguments{
					From:      &account1,
					Recipient: &account2,
					Nonce:     toHexUint64(0),
					Gas:       &gas,
				})

			rawJSON, err := json.MarshalIndent(proposal, "", "  ")
			require.NoError(t, err)
			err = os.WriteFile(fmt.Sprintf("/tmp/prepare_bundle_depth_%d.json", depth), rawJSON, 0o644)
			require.NoError(t, err)

			_, err = api.PrepareBundle(t.Context(), proposal)
			if depth <= bundle.MaxGroupNestingDepth {
				require.NoError(t, err, "expected no error for depth %d", depth)
			} else {
				require.ErrorContains(t, err, fmt.Sprintf("execution plan exceeds maximum nesting depth of %d", bundle.MaxGroupNestingDepth))
			}
		})
	}
}

func makeNestedBundleProposal(
	depth int,
	transaction exec.CallArguments,
) RPCExecutionProposal {
	step := any(transactionEntry(transaction))
	for range depth - 1 {
		step = groupEntry(step)
	}

	return RPCExecutionProposal{
		RPCExecutionPlanGroup: RPCExecutionPlanGroup{
			Steps: []any{step},
		},
	}
}
