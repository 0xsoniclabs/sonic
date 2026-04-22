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
	"encoding/json"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/api/ethapi"
	rpctest "github.com/0xsoniclabs/sonic/api/rpc_test"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

// txStep is a test helper that wraps a TransactionArgs as a leaf PrepareBundleStep.
func txStep(tx ethapi.TransactionArgs) PrepareBundleStep {
	return PrepareBundleStep{Tx: &PrepareBundleTxStep{Transaction: tx}}
}

// txStepWithFlags wraps a TransactionArgs as a leaf step with execution flags.
func txStepWithFlags(tx ethapi.TransactionArgs, tolerateFailed, tolerateInvalid bool) PrepareBundleStep {
	return PrepareBundleStep{Tx: &PrepareBundleTxStep{
		Transaction:     tx,
		TolerateFailed:  tolerateFailed,
		TolerateInvalid: tolerateInvalid,
	}}
}

func Test_fillTransactionDefaults(t *testing.T) {
	addr := common.Address{1}
	gasPrice := rpctest.ToHexBigInt(big.NewInt(1e9))

	existingGasPrice := rpctest.ToHexBigInt(big.NewInt(999))
	existingMaxFee := rpctest.ToHexBigInt(big.NewInt(888))
	tip := rpctest.ToHexBigInt(big.NewInt(1))
	explicit := hexutil.Uint64(50000)

	tests := []struct {
		name      string
		txs       []ethapi.TransactionArgs
		gasLimits []hexutil.Uint64
		gasPrice  *hexutil.Big
		check     func(t *testing.T, txs []ethapi.TransactionArgs)
	}{
		{
			name:      "nil gas limits fixed gas price",
			txs:       []ethapi.TransactionArgs{{From: &addr}},
			gasLimits: nil,
			gasPrice:  gasPrice,
			check: func(t *testing.T, txs []ethapi.TransactionArgs) {
				require.Nil(t, txs[0].Gas, "gas must remain nil when no limits provided")
				require.Equal(t, gasPrice, txs[0].GasPrice)
			},
		},
		{
			name:      "gas limits provided fills gas",
			txs:       []ethapi.TransactionArgs{{From: &addr}},
			gasLimits: []hexutil.Uint64{21000},
			gasPrice:  rpctest.ToHexBigInt(big.NewInt(1)),
			check: func(t *testing.T, txs []ethapi.TransactionArgs) {
				require.NotNil(t, txs[0].Gas)
				require.EqualValues(t, 21000, *txs[0].Gas)
			},
		},
		{
			name:      "gas limits provided preserves explicit gas",
			txs:       []ethapi.TransactionArgs{{From: &addr, Gas: &explicit}},
			gasLimits: []hexutil.Uint64{21000},
			gasPrice:  rpctest.ToHexBigInt(big.NewInt(1)),
			check: func(t *testing.T, txs []ethapi.TransactionArgs) {
				require.EqualValues(t, 50000, *txs[0].Gas, "explicit gas must not be overwritten")
			},
		},
		{
			name:      "existing gas price not overwritten",
			txs:       []ethapi.TransactionArgs{{From: &addr, GasPrice: existingGasPrice}},
			gasLimits: nil,
			gasPrice:  rpctest.ToHexBigInt(big.NewInt(1e9)),
			check: func(t *testing.T, txs []ethapi.TransactionArgs) {
				require.Equal(t, existingGasPrice, txs[0].GasPrice, "existing GasPrice must not be overwritten")
			},
		},
		{
			name:     "existing MaxFeePerGas not overwritten",
			txs:      []ethapi.TransactionArgs{{From: &addr, MaxFeePerGas: existingMaxFee}},
			gasPrice: rpctest.ToHexBigInt(big.NewInt(1e9)),
			check: func(t *testing.T, txs []ethapi.TransactionArgs) {
				require.Equal(t, existingMaxFee, txs[0].MaxFeePerGas, "existing MaxFeePerGas must not be overwritten")
				require.Nil(t, txs[0].GasPrice)
			},
		},
		{
			name:     "MaxPriorityFeePerGas set causes MaxFeePerGas to be filled",
			txs:      []ethapi.TransactionArgs{{From: &addr, MaxPriorityFeePerGas: tip}},
			gasPrice: gasPrice,
			check: func(t *testing.T, txs []ethapi.TransactionArgs) {
				require.Equal(t, gasPrice, txs[0].MaxFeePerGas)
				require.Nil(t, txs[0].GasPrice)
			},
		},
		{
			name:      "multiple txs each filled with correct limit and price",
			txs:       []ethapi.TransactionArgs{{From: &addr}, {From: &addr}, {From: &addr}},
			gasLimits: []hexutil.Uint64{21000, 42000, 63000},
			gasPrice:  gasPrice,
			check: func(t *testing.T, txs []ethapi.TransactionArgs) {
				for i, expected := range []uint64{21000, 42000, 63000} {
					require.NotNil(t, txs[i].Gas)
					require.EqualValues(t, expected, *txs[i].Gas)
					require.Equal(t, gasPrice, txs[i].GasPrice)
				}
			},
		},
		{
			name:      "fewer limits than txs only fills available",
			txs:       []ethapi.TransactionArgs{{From: &addr}, {From: &addr}},
			gasLimits: []hexutil.Uint64{21000},
			gasPrice:  rpctest.ToHexBigInt(big.NewInt(1)),
			check: func(t *testing.T, txs []ethapi.TransactionArgs) {
				require.NotNil(t, txs[0].Gas)
				require.EqualValues(t, 21000, *txs[0].Gas)
				require.Nil(t, txs[1].Gas, "tx without a corresponding limit must remain nil")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fillTransactionDefaults(tc.txs, tc.gasLimits, tc.gasPrice)
			tc.check(t, tc.txs)
		})
	}
}

func Test_resolveBlockRange(t *testing.T) {
	hexN := func(n uint64) *hexutil.Uint64 { b := hexutil.Uint64(n); return &b }

	tests := []struct {
		name          string
		currentBlock  uint64
		earliest      *hexutil.Uint64
		latest        *hexutil.Uint64
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
			name:         "explicit earliest",
			currentBlock: 10,
			earliest:     hexN(50),
			wantEarliest: 50,
			wantLatest:   50 + bundle.MaxBlockRange - 1,
		},
		{
			name:         "explicit latest",
			currentBlock: 10,
			latest:       hexN(200),
			wantEarliest: 11,
			wantLatest:   200,
		},
		{
			name:          "explicit latest greater than MaxBlockRange",
			currentBlock:  10,
			latest:        hexN(10 + bundle.MaxBlockRange + 100),
			errorContains: "invalid block range",
		},
		{
			name:         "both explicit",
			currentBlock: 100,
			earliest:     hexN(5),
			latest:       hexN(20),
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
			earliest:      hexN(50),
			latest:        hexN(40),
			errorContains: "invalid block range",
		},
		{
			name:          "greater than Max block range",
			currentBlock:  100,
			earliest:      hexN(50),
			latest:        hexN(50 + bundle.MaxBlockRange + 1),
			errorContains: "invalid block range",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r, err := resolveBlockRange(tc.currentBlock, tc.earliest, tc.latest)
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

func Test_asTransaction_UnsupportedTypes_ReturnsError(t *testing.T) {
	tests := []struct {
		name    string
		msg     *core.Message
		wantErr string
	}{
		{
			name:    "blob hashes",
			msg:     &core.Message{BlobHashes: []common.Hash{{0x01}}},
			wantErr: "blob transactions are not supported",
		},
		{
			name:    "blob gas fee cap",
			msg:     &core.Message{BlobGasFeeCap: big.NewInt(1)},
			wantErr: "blob transactions are not supported",
		},
		{
			name:    "set code authorizations",
			msg:     &core.Message{SetCodeAuthorizations: []types.SetCodeAuthorization{{}}},
			wantErr: "transactions with set code authorization are not supported",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := asTransaction(tc.msg)
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}

func Test_asTransaction_TxType(t *testing.T) {
	to := common.Address{2}

	tests := []struct {
		name          string
		msg           *core.Message
		wantType      int
		errorContains string
	}{
		{
			name: "nil gas price returns dynamic fee tx",
			msg: &core.Message{
				To:        &to,
				GasLimit:  params.TxGas,
				GasFeeCap: big.NewInt(1e9),
				GasTipCap: big.NewInt(1e6),
				Value:     big.NewInt(0),
			},
			wantType: types.DynamicFeeTxType,
		},
		{
			name: "zero gas price with gas fee cap returns dynamic fee tx",
			msg: &core.Message{
				To:        &to,
				GasPrice:  big.NewInt(0),
				GasFeeCap: big.NewInt(1),
				GasLimit:  params.TxGas,
				Value:     big.NewInt(0),
			},
			wantType: types.DynamicFeeTxType,
		},
		{
			name: "zero gas price with gas tip cap returns dynamic fee tx",
			msg: &core.Message{
				To:        &to,
				GasPrice:  big.NewInt(0),
				GasTipCap: big.NewInt(1),
				GasLimit:  params.TxGas,
				Value:     big.NewInt(0),
			},
			wantType: types.DynamicFeeTxType,
		},
		{
			name: "positive gas price returns access list tx",
			msg: &core.Message{
				To:       &to,
				GasPrice: big.NewInt(1e9),
				GasLimit: params.TxGas,
				Value:    big.NewInt(0),
			},
			wantType: types.AccessListTxType,
		},
		{
			name: "positive gas price and gas fee cap returns dynamic fee tx",
			msg: &core.Message{
				To:        &to,
				GasPrice:  big.NewInt(1e9),
				GasFeeCap: big.NewInt(1e9),
				Value:     big.NewInt(0),
			},
			errorContains: "cannot set both gas price",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tx, err := asTransaction(tc.msg)
			if tc.errorContains != "" {
				require.ErrorContains(t, err, tc.errorContains)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.wantType, int(tx.Type()))
			}
		})
	}
}

func Test_asTransaction_PreservesFields(t *testing.T) {
	to := common.Address{0xde}
	accessList := types.AccessList{{Address: common.Address{0xaa}}}
	msg := &core.Message{
		To:         &to,
		Nonce:      7,
		GasLimit:   100_000,
		GasFeeCap:  big.NewInt(2e9),
		GasTipCap:  big.NewInt(1e6),
		Value:      big.NewInt(42),
		Data:       []byte{0x01, 0x02},
		AccessList: accessList,
	}
	tx, err := asTransaction(msg)
	require.NoError(t, err)
	require.Equal(t, to, *tx.To())
	require.EqualValues(t, 7, tx.Nonce())
	require.EqualValues(t, 100_000, tx.Gas())
	require.Equal(t, big.NewInt(42), tx.Value())
	require.Equal(t, []byte{0x01, 0x02}, tx.Data())
}

func Test_PrepareBundleStep_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name          string
		json          string
		wantTx        bool
		wantGroup     bool
		errorContains string
	}{
		{
			name:   "transaction field to leaf",
			json:   `{"transaction":{"from":"0x0100000000000000000000000000000000000000"}}`,
			wantTx: true,
		},
		{
			name:      "steps field to group",
			json:      `{"steps":[]}`,
			wantGroup: true,
		},
		{
			name:          "both fields to error",
			json:          `{"transaction":{},"steps":[]}`,
			errorContains: "not both",
		},
		{
			name:          "neither field to error",
			json:          `{"tolerateFailed":true}`,
			errorContains: "must have either 'transaction' or 'steps'",
		},
		{
			name:          "invalid JSON to error",
			json:          `not-json`,
			errorContains: "invalid character",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var step PrepareBundleStep
			err := json.Unmarshal([]byte(tc.json), &step)
			if tc.errorContains != "" {
				require.ErrorContains(t, err, tc.errorContains)
				return
			}
			require.NoError(t, err)
			if tc.wantTx {
				require.NotNil(t, step.Tx)
				require.Nil(t, step.Group)
			}
			if tc.wantGroup {
				require.NotNil(t, step.Group)
				require.Nil(t, step.Tx)
			}
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

	args := PrepareBundleArgs{
		PrepareBundleGroupStep: PrepareBundleGroupStep{
			Steps: []PrepareBundleStep{
				txStep(ethapi.TransactionArgs{
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

func Test_PrepareBundle_PlanHashConsistentAcrossTxs(t *testing.T) {
	addr1 := common.Address{1}
	addr2 := common.Address{2}

	be := rpctest.NewBackendBuilder(t).
		WithAccount(addr1, rpctest.AccountState{Balance: big.NewInt(1e18)}).
		WithAccount(addr2, rpctest.AccountState{Balance: big.NewInt(1e18)}).
		Build()

	api := NewPublicBundleAPI(be)

	args := PrepareBundleArgs{
		PrepareBundleGroupStep: PrepareBundleGroupStep{
			Steps: []PrepareBundleStep{
				txStep(ethapi.TransactionArgs{From: &addr1, To: &addr2, Nonce: rpctest.ToHexUint64(0), Value: rpctest.ToHexBigInt(big.NewInt(1e15))}),
				txStep(ethapi.TransactionArgs{From: &addr2, To: &addr1, Nonce: rpctest.ToHexUint64(0), Value: rpctest.ToHexBigInt(big.NewInt(1e15))}),
			},
		},
	}

	result, err := api.PrepareBundle(t.Context(), args)
	require.NoError(t, err)

	// All txs must carry the same plan hash.
	var planHash common.Hash
	for i, tx := range result.Transactions {
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

func Test_PrepareBundle_AccessListContainsConsistentPlanHash(t *testing.T) {
	addr1 := common.Address{1}
	addr2 := common.Address{2}

	be := rpctest.NewBackendBuilder(t).
		WithAccount(addr1, rpctest.AccountState{Balance: big.NewInt(1e18)}).
		Build()

	api := NewPublicBundleAPI(be)

	args := PrepareBundleArgs{
		PrepareBundleGroupStep: PrepareBundleGroupStep{
			Steps: []PrepareBundleStep{
				txStep(ethapi.TransactionArgs{From: &addr1, To: &addr2, Nonce: rpctest.ToHexUint64(0), Value: rpctest.ToHexBigInt(big.NewInt(1e15))}),
				txStep(ethapi.TransactionArgs{From: &addr2, To: &addr1, Nonce: rpctest.ToHexUint64(0), Value: rpctest.ToHexBigInt(big.NewInt(1e15))}),
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
			require.NotNil(t, entry.Address)
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

	args := PrepareBundleArgs{
		PrepareBundleGroupStep: PrepareBundleGroupStep{
			Steps: []PrepareBundleStep{
				txStep(ethapi.TransactionArgs{
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

	args := PrepareBundleArgs{
		PrepareBundleGroupStep: PrepareBundleGroupStep{
			Steps: []PrepareBundleStep{
				txStep(ethapi.TransactionArgs{
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

	args := PrepareBundleArgs{
		PrepareBundleGroupStep: PrepareBundleGroupStep{
			Steps: []PrepareBundleStep{
				txStep(ethapi.TransactionArgs{
					From:  &addr1,
					To:    &addr2,
					Nonce: rpctest.ToHexUint64(0),
				}),
			},
		},
		EarliestBlock: &earliest,
		LatestBlock:   &latest,
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

	args := PrepareBundleArgs{
		PrepareBundleGroupStep: PrepareBundleGroupStep{
			Steps: []PrepareBundleStep{
				txStep(ethapi.TransactionArgs{
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

	args := PrepareBundleArgs{
		PrepareBundleGroupStep: PrepareBundleGroupStep{
			Steps: []PrepareBundleStep{
				txStep(ethapi.TransactionArgs{From: &addr1, To: &addr2, Nonce: rpctest.ToHexUint64(0), Value: rpctest.ToHexBigInt(big.NewInt(1e15))}),
				txStep(ethapi.TransactionArgs{From: &addr2, To: &addr1, Nonce: rpctest.ToHexUint64(0), Value: rpctest.ToHexBigInt(big.NewInt(1e15))}),
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

	args := PrepareBundleArgs{PrepareBundleGroupStep: PrepareBundleGroupStep{Steps: []PrepareBundleStep{}}}
	result, err := api.PrepareBundle(t.Context(), args)
	require.NoError(t, err)
	require.Empty(t, result.Transactions)
}

func Test_PrepareBundle_OneOfGroup_BuildsOneOfPlan(t *testing.T) {
	addr1 := common.Address{1}
	addr2 := common.Address{2}

	be := rpctest.NewBackendBuilder(t).
		WithAccount(addr1, rpctest.AccountState{Balance: big.NewInt(1e18)}).
		WithAccount(addr2, rpctest.AccountState{Balance: big.NewInt(1e18)}).
		Build()

	api := NewPublicBundleAPI(be)

	args := PrepareBundleArgs{
		PrepareBundleGroupStep: PrepareBundleGroupStep{
			Steps: []PrepareBundleStep{
				{Group: &PrepareBundleGroupStep{
					OneOf: true,
					Steps: []PrepareBundleStep{
						txStep(ethapi.TransactionArgs{From: &addr1, To: &addr2, Nonce: rpctest.ToHexUint64(0), Value: rpctest.ToHexBigInt(big.NewInt(1e15))}),
						txStep(ethapi.TransactionArgs{From: &addr2, To: &addr1, Nonce: rpctest.ToHexUint64(0), Value: rpctest.ToHexBigInt(big.NewInt(1e15))}),
					},
				}},
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

	args := PrepareBundleArgs{
		PrepareBundleGroupStep: PrepareBundleGroupStep{
			Steps: []PrepareBundleStep{
				txStepWithFlags(ethapi.TransactionArgs{
					From:  &addr1,
					To:    &addr2,
					Nonce: rpctest.ToHexUint64(0),
				}, true, false),
			},
		},
	}

	result, err := api.PrepareBundle(t.Context(), args)
	require.NoError(t, err)
	require.Len(t, result.ExecutionPlan.Steps, 1)

	leaf, ok := result.ExecutionPlan.Steps[0].(*RPCExecutionStepComposable)
	require.True(t, ok, "expected leaf step")
	require.True(t, leaf.TolerateFailed, "TolerateFailed must be set")
	require.False(t, leaf.TolerateInvalid)
}

func Test_PrepareBundle_TolerateInvalid_Flag(t *testing.T) {
	addr1 := common.Address{1}
	addr2 := common.Address{2}

	be := rpctest.NewBackendBuilder(t).
		WithAccount(addr1, rpctest.AccountState{Balance: big.NewInt(1e18)}).
		Build()

	api := NewPublicBundleAPI(be)

	args := PrepareBundleArgs{
		PrepareBundleGroupStep: PrepareBundleGroupStep{
			Steps: []PrepareBundleStep{
				txStepWithFlags(ethapi.TransactionArgs{
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

	leaf, ok := result.ExecutionPlan.Steps[0].(*RPCExecutionStepComposable)
	require.True(t, ok, "expected leaf step")
	require.False(t, leaf.TolerateFailed)
	require.True(t, leaf.TolerateInvalid, "TolerateInvalid must be set")
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
	args := PrepareBundleArgs{
		PrepareBundleGroupStep: PrepareBundleGroupStep{
			Steps: []PrepareBundleStep{
				{Group: &PrepareBundleGroupStep{
					OneOf: true,
					Steps: []PrepareBundleStep{
						{Group: &PrepareBundleGroupStep{
							Steps: []PrepareBundleStep{
								txStep(ethapi.TransactionArgs{From: &addr1, To: &addr2, Nonce: rpctest.ToHexUint64(0), Value: rpctest.ToHexBigInt(big.NewInt(1e15))}),
								txStep(ethapi.TransactionArgs{From: &addr2, To: &addr1, Nonce: rpctest.ToHexUint64(0), Value: rpctest.ToHexBigInt(big.NewInt(1e15))}),
							},
						}},
						txStep(ethapi.TransactionArgs{From: &addr1, To: &addr2, Nonce: rpctest.ToHexUint64(1), Value: rpctest.ToHexBigInt(big.NewInt(1e15))}),
					},
				}},
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
