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
	"errors"
	"math/big"
	"testing"

	rpctest "github.com/0xsoniclabs/sonic/api/rpc_test"
	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func Test_SubmitBundle_ValidBundle_ReturnsExecutionPlanHash(t *testing.T) {
	addr := common.Address{2}

	tests := []struct {
		name     string
		flags    bundle.ExecutionFlags
		numSteps int
	}{
		{"single tx, AllOf", bundle.EF_AllOf, 1},
		{"two txs, AllOf", bundle.EF_AllOf, 2},
		{"three txs, AllOf", bundle.EF_AllOf, 3},
		{"single tx, OneOf", bundle.EF_OneOf, 1},
		{"two txs, OneOf", bundle.EF_OneOf, 2},
		{"single tx, TolerateFailed", bundle.EF_TolerateFailed, 1},
		{"two txs, TolerateFailed", bundle.EF_TolerateFailed, 2},
		{"single tx, TolerateInvalid", bundle.EF_TolerateInvalid, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var submitted *types.Transaction
			ctrl := gomock.NewController(t)
			pool := rpctest.NewMockTxPool(ctrl)
			pool.EXPECT().AddLocal(gomock.Any()).DoAndReturn(func(tx *types.Transaction) error {
				submitted = tx
				return nil
			})

			be := rpctest.NewBackendBuilder(t).WithPool(pool).Build()
			signer := types.LatestSignerForChainID(be.ChainID())

			steps := make([]bundle.BundleStep, tt.numSteps)
			for i := range steps {
				key, err := crypto.GenerateKey()
				require.NoError(t, err)
				steps[i] = bundle.Step(key, &types.DynamicFeeTx{
					To:  &addr,
					Gas: params.TxGas,
				})
			}

			args := buildSubmitBundleArgs(signer, tt.flags, 1, 100, steps...)

			hash, err := NewPublicBundleAPI(be).SubmitBundle(t.Context(), args)
			require.NoError(t, err)

			// Returned hash must match the execution plan hash derived from the args.
			plan := args.ExecutionPlan
			execPlan := bundle.ExecutionPlan{
				Flags: plan.Flags,
				Range: bundle.BlockRange{
					Earliest: uint64(plan.Earliest),
					Latest:   uint64(plan.Latest),
				},
				Steps: make([]bundle.ExecutionStep, len(plan.Steps)),
			}
			for i, s := range plan.Steps {
				execPlan.Steps[i] = bundle.ExecutionStep{From: s.From, Hash: s.Hash}
			}
			require.Equal(t, execPlan.Hash(), hash)

			// Submitted transaction must be a valid bundle envelope corresponding to the same execution plan.
			require.NotNil(t, submitted)
			require.True(t, bundle.IsEnvelope(submitted))
			decoded, err := bundle.OpenEnvelope(submitted)
			require.NoError(t, err)
			require.Equal(t, execPlan.Flags, decoded.Flags)
			require.EqualValues(t, execPlan.Range.Earliest, decoded.Range.Earliest)
			require.EqualValues(t, execPlan.Range.Latest, decoded.Range.Latest)
			poolPlan, err := bundle.ExtractExecutionPlan(signer, submitted)
			require.NoError(t, err)
			require.Equal(t, execPlan.Hash(), poolPlan.Hash())

		})
	}
}

func Test_SubmitBundle_InvalidTxEncoding_ReturnsError(t *testing.T) {
	tests := []struct {
		name   string
		txData hexutil.Bytes
	}{
		{"empty bytes", hexutil.Bytes{}},
		{"garbage bytes", hexutil.Bytes{0xde, 0xad, 0xbe, 0xef}},
		{"truncated rlp", hexutil.Bytes{0x01, 0x80}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			// AddLocal must not be called, decoding fails before submission.
			pool := rpctest.NewMockTxPool(ctrl)

			be := rpctest.NewBackendBuilder(t).WithPool(pool).Build()
			signer := types.LatestSignerForChainID(be.ChainID())

			key, err := crypto.GenerateKey()
			require.NoError(t, err)
			addr := common.Address{2}

			validArgs := buildSubmitBundleArgs(signer, bundle.EF_AllOf, 1, 100,
				bundle.Step(key, &types.DynamicFeeTx{To: &addr, Gas: params.TxGas}),
			)

			args := SubmitBundleArgs{
				SignedTransactions: []hexutil.Bytes{tt.txData},
				ExecutionPlan:      validArgs.ExecutionPlan,
			}

			_, err = NewPublicBundleAPI(be).SubmitBundle(t.Context(), args)
			require.ErrorContains(t, err, "failed to decode bundled transaction 0")
		})
	}
}

func Test_SubmitBundle_PoolError_ReturnsError(t *testing.T) {
	poolErr := errors.New("pool is full")

	ctrl := gomock.NewController(t)
	pool := rpctest.NewMockTxPool(ctrl)
	pool.EXPECT().AddLocal(gomock.Any()).Return(poolErr)

	be := rpctest.NewBackendBuilder(t).WithPool(pool).Build()
	signer := types.LatestSignerForChainID(be.ChainID())

	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	addr := common.Address{2}

	args := buildSubmitBundleArgs(signer, bundle.EF_AllOf, 1, 100,
		bundle.Step(key, &types.DynamicFeeTx{To: &addr, Gas: params.TxGas}),
	)

	_, err = NewPublicBundleAPI(be).SubmitBundle(t.Context(), args)
	require.ErrorIs(t, err, poolErr)
}

func Test_SubmitBundle_SubmittedEnvelopeMatchesBundleContents(t *testing.T) {
	var submitted *types.Transaction

	ctrl := gomock.NewController(t)
	pool := rpctest.NewMockTxPool(ctrl)
	pool.EXPECT().AddLocal(gomock.Any()).DoAndReturn(func(tx *types.Transaction) error {
		submitted = tx
		return nil
	})

	be := rpctest.NewBackendBuilder(t).WithPool(pool).Build()
	signer := types.LatestSignerForChainID(be.ChainID())

	key1, err := crypto.GenerateKey()
	require.NoError(t, err)
	key2, err := crypto.GenerateKey()
	require.NoError(t, err)

	addr := common.Address{2}
	args := buildSubmitBundleArgs(signer, bundle.EF_AllOf, 10, 20,
		bundle.Step(key1, &types.DynamicFeeTx{To: &addr, Nonce: 0, Gas: params.TxGas}),
		bundle.Step(key2, &types.DynamicFeeTx{To: &addr, Nonce: 0, Gas: params.TxGas}),
	)

	_, err = NewPublicBundleAPI(be).SubmitBundle(t.Context(), args)
	require.NoError(t, err)
	require.NotNil(t, submitted)

	require.True(t, bundle.IsEnvelope(submitted))
	decoded, err := bundle.OpenEnvelope(submitted)
	require.NoError(t, err)
	require.Len(t, decoded.Transactions, 2)
	require.Equal(t, bundle.EF_AllOf, decoded.Flags)
	require.EqualValues(t, 10, decoded.Range.Earliest)
	require.EqualValues(t, 20, decoded.Range.Latest)
}

func Test_SubmitBundle_EnvelopeGasCoversAllBundledTxs(t *testing.T) {
	var submitted *types.Transaction

	ctrl := gomock.NewController(t)
	pool := rpctest.NewMockTxPool(ctrl)
	pool.EXPECT().AddLocal(gomock.Any()).DoAndReturn(func(tx *types.Transaction) error {
		submitted = tx
		return nil
	})

	be := rpctest.NewBackendBuilder(t).WithPool(pool).Build()
	signer := types.LatestSignerForChainID(be.ChainID())

	addr := common.Address{2}
	highGas := uint64(200_000)

	steps := make([]bundle.BundleStep, 3)
	for i := range steps {
		key, err := crypto.GenerateKey()
		require.NoError(t, err)
		steps[i] = bundle.Step(key, &types.DynamicFeeTx{
			To:    &addr,
			Nonce: uint64(i),
			Gas:   highGas,
		})
	}

	args := buildSubmitBundleArgs(signer, bundle.EF_AllOf, 1, 50, steps...)

	expectedMinGas := uint64(0)
	for _, b := range args.SignedTransactions {
		tx := new(types.Transaction)
		require.NoError(t, tx.UnmarshalBinary(b))
		expectedMinGas += tx.Gas()
	}

	_, err := NewPublicBundleAPI(be).SubmitBundle(t.Context(), args)
	require.NoError(t, err)
	require.NotNil(t, submitted)
	require.GreaterOrEqual(t, submitted.Gas(), expectedMinGas)
}

func Test_SubmitBundle_BlockRangeIsPreservedInPool(t *testing.T) {
	tests := []struct {
		name     string
		earliest uint64
		latest   uint64
	}{
		{"range [1,1]", 1, 1},
		{"range [1,100]", 1, 100},
		{"range [50,50]", 50, 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var submitted *types.Transaction

			ctrl := gomock.NewController(t)
			pool := rpctest.NewMockTxPool(ctrl)
			pool.EXPECT().AddLocal(gomock.Any()).DoAndReturn(func(tx *types.Transaction) error {
				submitted = tx
				return nil
			})

			be := rpctest.NewBackendBuilder(t).WithPool(pool).Build()
			signer := types.LatestSignerForChainID(be.ChainID())

			key, err := crypto.GenerateKey()
			require.NoError(t, err)
			addr := common.Address{2}

			args := buildSubmitBundleArgs(signer, bundle.EF_AllOf, tt.earliest, tt.latest,
				bundle.Step(key, &types.DynamicFeeTx{To: &addr, Gas: params.TxGas}),
			)

			_, err = NewPublicBundleAPI(be).SubmitBundle(t.Context(), args)
			require.NoError(t, err)
			require.NotNil(t, submitted)

			decoded, err := bundle.OpenEnvelope(submitted)
			require.NoError(t, err)
			require.Equal(t, tt.earliest, decoded.Range.Earliest)
			require.Equal(t, tt.latest, decoded.Range.Latest)
		})
	}
}

func Test_SubmitBundle_InvalidBlockRange_ReturnsError(t *testing.T) {
	tests := []struct {
		name     string
		earliest uint64
		latest   uint64
		errMsg   string
	}{
		{
			name:     "earliest > latest",
			earliest: 10,
			latest:   5,
			errMsg:   "latest block number cannot be smaller than earliest block number",
		},
		{
			name:     "range too large",
			earliest: 0,
			latest:   bundle.MaxBlockRange + 1,
			errMsg:   "invalid block range",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			// AddLocal must not be called, validation fails before submission.
			pool := rpctest.NewMockTxPool(ctrl)

			be := rpctest.NewBackendBuilder(t).WithPool(pool).Build()
			signer := types.LatestSignerForChainID(be.ChainID())

			key, err := crypto.GenerateKey()
			require.NoError(t, err)
			addr := common.Address{2}

			// Build args with a valid range, then override to invalid.
			tb, plan := bundle.NewBuilder(signer).
				SetEarliest(1).
				SetLatest(10).
				With(bundle.Step(key, &types.DynamicFeeTx{To: &addr, Gas: params.TxGas})).
				BuildBundleAndPlan()

			signedTxs := make([]hexutil.Bytes, len(tb.Transactions))
			for i, tx := range tb.Transactions {
				data, err := tx.MarshalBinary()
				require.NoError(t, err)
				signedTxs[i] = data
			}

			rpcPlan := NewRPCExecutionPlan(plan)
			rpcPlan.Earliest = rpc.BlockNumber(tt.earliest)
			rpcPlan.Latest = rpc.BlockNumber(tt.latest)

			args := SubmitBundleArgs{
				SignedTransactions: signedTxs,
				ExecutionPlan:      rpcPlan,
			}

			_, err = NewPublicBundleAPI(be).SubmitBundle(t.Context(), args)
			require.ErrorContains(t, err, tt.errMsg)
		})
	}
}

func Test_SubmitBundle_InvalidExecutionPlanBlockNumbers_ReturnsError(t *testing.T) {
	tests := []struct {
		name     string
		earliest rpc.BlockNumber
	}{
		{
			name:     "latest as earliest",
			earliest: rpc.LatestBlockNumber,
		},
		{
			name:     "pending as earliest",
			earliest: rpc.PendingBlockNumber,
		},
		{
			name:     "finalized as earliest",
			earliest: rpc.FinalizedBlockNumber,
		},
		{
			name:     "safe as earliest",
			earliest: rpc.SafeBlockNumber,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			// AddLocal must not be called, request validation fails before submission.
			pool := rpctest.NewMockTxPool(ctrl)
			be := rpctest.NewBackendBuilder(t).
				WithPool(pool).
				WithBlockHistory(
					[]rpctest.Block{
						{Number: 1},
						{Number: 5},
						{Number: 10},
					},
				).
				Build()
			signer := types.LatestSignerForChainID(be.ChainID())
			key, err := crypto.GenerateKey()
			require.NoError(t, err)
			addr := common.Address{2}
			args := buildSubmitBundleArgs(signer, bundle.EF_AllOf, 1, 100,
				bundle.Step(key, &types.DynamicFeeTx{To: &addr, Gas: params.TxGas}),
			)
			args.ExecutionPlan.Earliest = tt.earliest
			args.ExecutionPlan.Latest = rpc.BlockNumber(1)
			_, err = NewPublicBundleAPI(be).SubmitBundle(t.Context(), args)
			require.ErrorContains(t, err, "latest block number cannot be smaller than earliest block number")
		})
	}
}

func Test_parseRPCBlockNumber(t *testing.T) {
	tests := []struct {
		name          string
		num           rpc.BlockNumber
		currentBlock  *evmcore.EvmBlock
		wantNum       uint64
		wantErrSubstr string
	}{
		{
			name:    "explicit block number",
			num:     rpc.BlockNumber(42),
			wantNum: 42,
		},
		{
			name:    "block number zero",
			num:     rpc.BlockNumber(0),
			wantNum: 0,
		},
		{
			name:    "large block number",
			num:     rpc.BlockNumber(1_000_000),
			wantNum: 1_000_000,
		},
		{
			name:    "earliest block number returns zero",
			num:     rpc.EarliestBlockNumber,
			wantNum: 0,
		},
		{
			name:         "latest with current block",
			num:          rpc.LatestBlockNumber,
			currentBlock: makeEvmBlock(99),
			wantNum:      99,
		},
		{
			name:         "pending with current block",
			num:          rpc.PendingBlockNumber,
			currentBlock: makeEvmBlock(5),
			wantNum:      5,
		},
		{
			name:         "finalized with current block",
			num:          rpc.FinalizedBlockNumber,
			currentBlock: makeEvmBlock(77),
			wantNum:      77,
		},
		{
			name:         "safe with current block",
			num:          rpc.SafeBlockNumber,
			currentBlock: makeEvmBlock(33),
			wantNum:      33,
		},
		{
			name:          "latest without current block returns error",
			num:           rpc.LatestBlockNumber,
			currentBlock:  nil,
			wantErrSubstr: "no current block",
		},
		{
			name:          "pending without current block returns error",
			num:           rpc.PendingBlockNumber,
			currentBlock:  nil,
			wantErrSubstr: "no current block",
		},
		{
			name:          "finalized without current block returns error",
			num:           rpc.FinalizedBlockNumber,
			currentBlock:  nil,
			wantErrSubstr: "no current block",
		},
		{
			name:          "safe without current block returns error",
			num:           rpc.SafeBlockNumber,
			currentBlock:  nil,
			wantErrSubstr: "no current block",
		},
		{
			name:          "arbitrary negative number returns error",
			num:           rpc.BlockNumber(-100),
			wantErrSubstr: "block number cannot be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mock := NewMockBundleApiBackend(ctrl)

			needsCurrentBlock := tt.num == rpc.PendingBlockNumber ||
				tt.num == rpc.LatestBlockNumber ||
				tt.num == rpc.FinalizedBlockNumber ||
				tt.num == rpc.SafeBlockNumber
			if needsCurrentBlock {
				// parseRPCBlockNumber calls CurrentBlock() twice: once for the nil
				// guard and once to read the block number (only when block != nil).
				if tt.currentBlock != nil {
					mock.EXPECT().CurrentBlock().Return(tt.currentBlock).Times(1)
				} else {
					mock.EXPECT().CurrentBlock().Return(nil).Times(1)
				}
			}

			got, err := parseRPCBlockNumber(tt.num, mock)
			if tt.wantErrSubstr != "" {
				require.ErrorContains(t, err, tt.wantErrSubstr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.wantNum, got)
			}
		})
	}
}

func makeEvmBlock(number uint64) *evmcore.EvmBlock {
	n := big.NewInt(int64(number))
	return &evmcore.EvmBlock{EvmHeader: evmcore.EvmHeader{Number: n}}
}

// buildSubmitBundleArgs creates a valid SubmitBundleArgs using the bundle builder.
func buildSubmitBundleArgs(
	signer types.Signer,
	flags bundle.ExecutionFlags,
	earliest, latest uint64,
	steps ...bundle.BundleStep,
) SubmitBundleArgs {
	tb, plan := bundle.NewBuilder(signer).
		SetFlags(flags).
		SetEarliest(earliest).
		SetLatest(latest).
		With(steps...).
		BuildBundleAndPlan()

	signedTxs := make([]hexutil.Bytes, len(tb.Transactions))
	for i, tx := range tb.Transactions {
		data, err := tx.MarshalBinary()
		if err != nil {
			panic(err)
		}
		signedTxs[i] = data
	}

	return SubmitBundleArgs{
		SignedTransactions: signedTxs,
		ExecutionPlan:      NewRPCExecutionPlan(plan),
	}
}
