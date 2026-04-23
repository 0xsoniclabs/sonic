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
	"errors"
	"testing"

	rpctest "github.com/0xsoniclabs/sonic/api/rpc_test"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
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
		{"single tx, default flags", bundle.EF_Default, 1},
		{"two txs, default flags", bundle.EF_Default, 2},
		{"three txs, default flags", bundle.EF_Default, 3},
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

			steps := make([]bundle.BuilderStep, tt.numSteps)
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

			// Submitted transaction must be a valid bundle envelope.
			require.NotNil(t, submitted)
			require.True(t, bundle.IsEnvelope(submitted))

			// Returned hash must match the execution plan extracted from the submitted envelope.
			_, submittedPlan, err := bundle.ValidateEnvelope(signer, submitted)
			require.NoError(t, err)
			require.Equal(t, submittedPlan.Hash(), hash)

			// Block range must be preserved.
			require.EqualValues(t, 1, submittedPlan.Range.Earliest)
			require.EqualValues(t, 100, submittedPlan.Range.Latest)
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

			validArgs := buildSubmitBundleArgs(signer, bundle.EF_Default, 1, 100,
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

	args := buildSubmitBundleArgs(signer, bundle.EF_Default, 1, 100,
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
	args := buildSubmitBundleArgs(signer, bundle.EF_Default, 10, 20,
		bundle.Step(key1, &types.DynamicFeeTx{To: &addr, Nonce: 0, Gas: params.TxGas}),
		bundle.Step(key2, &types.DynamicFeeTx{To: &addr, Nonce: 0, Gas: params.TxGas}),
	)

	_, err = NewPublicBundleAPI(be).SubmitBundle(t.Context(), args)
	require.NoError(t, err)
	require.NotNil(t, submitted)

	require.True(t, bundle.IsEnvelope(submitted))
	decoded, err := bundle.OpenEnvelope(signer, submitted)
	require.NoError(t, err)
	require.Len(t, decoded.Transactions, 2)
	require.EqualValues(t, 10, decoded.Plan.Range.Earliest)
	require.EqualValues(t, 20, decoded.Plan.Range.Latest)
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

	steps := make([]bundle.BuilderStep, 3)
	for i := range steps {
		key, err := crypto.GenerateKey()
		require.NoError(t, err)
		steps[i] = bundle.Step(key, &types.DynamicFeeTx{
			To:    &addr,
			Nonce: uint64(i),
			Gas:   highGas,
		})
	}

	args := buildSubmitBundleArgs(signer, bundle.EF_Default, 1, 50, steps...)

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

			args := buildSubmitBundleArgs(signer, bundle.EF_Default, tt.earliest, tt.latest,
				bundle.Step(key, &types.DynamicFeeTx{To: &addr, Gas: params.TxGas}),
			)

			_, err = NewPublicBundleAPI(be).SubmitBundle(t.Context(), args)
			require.NoError(t, err)
			require.NotNil(t, submitted)

			decoded, err := bundle.OpenEnvelope(signer, submitted)
			require.NoError(t, err)
			require.Equal(t, tt.earliest, decoded.Plan.Range.Earliest)
			require.Equal(t, tt.latest, decoded.Plan.Range.Latest)
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
			tb, plan := bundle.NewBuilder().
				WithSigner(signer).
				SetEarliest(1).
				SetLatest(10).
				With(bundle.Step(key, &types.DynamicFeeTx{To: &addr, Gas: params.TxGas})).
				BuildBundleAndPlan()

			txsInOrder := tb.GetTransactionsInReferencedOrder()
			signedTxs := make([]hexutil.Bytes, len(txsInOrder))
			for i, tx := range txsInOrder {
				data, err := tx.MarshalBinary()
				require.NoError(t, err)
				signedTxs[i] = data
			}

			rpcPlan, err := NewRPCExecutionPlanComposable(plan)
			require.NoError(t, err)
			rpcPlan.BlockRange.Earliest = hexutil.Uint64(tt.earliest)
			rpcPlan.BlockRange.Latest = hexutil.Uint64(tt.latest)

			args := SubmitBundleArgs{
				SignedTransactions: signedTxs,
				ExecutionPlan:      rpcPlan,
			}

			_, err = NewPublicBundleAPI(be).SubmitBundle(t.Context(), args)
			require.ErrorContains(t, err, tt.errMsg)
		})
	}
}

func Test_SubmitBundle_JSONRoundTrip_Works(t *testing.T) {
	tests := []struct {
		name     string
		flags    bundle.ExecutionFlags
		numSteps int
	}{
		{"single tx, default flags", bundle.EF_Default, 1},
		{"two txs, default flags", bundle.EF_Default, 2},
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

			addr := common.Address{2}
			steps := make([]bundle.BuilderStep, tt.numSteps)
			for i := range steps {
				key, err := crypto.GenerateKey()
				require.NoError(t, err)
				steps[i] = bundle.Step(key, &types.DynamicFeeTx{To: &addr, Gas: params.TxGas})
			}
			args := buildSubmitBundleArgs(signer, tt.flags, 1, 100, steps...)

			// Simulate JSON-RPC wire transport
			data, err := json.Marshal(args)
			require.NoError(t, err)
			var deserialized SubmitBundleArgs
			require.NoError(t, json.Unmarshal(data, &deserialized))

			_, err = NewPublicBundleAPI(be).SubmitBundle(t.Context(), deserialized)
			require.NoError(t, err)
			require.NotNil(t, submitted)
			require.True(t, bundle.IsEnvelope(submitted))
		})
	}
}

// buildSubmitBundleArgs creates a valid SubmitBundleArgs using the bundle builder.
// The flags are applied to each transaction step individually.
func buildSubmitBundleArgs(
	signer types.Signer,
	flags bundle.ExecutionFlags,
	earliest, latest uint64,
	steps ...bundle.BuilderStep,
) SubmitBundleArgs {
	stepsWithFlags := make([]bundle.BuilderStep, len(steps))
	for i, s := range steps {
		stepsWithFlags[i] = s.WithFlags(flags)
	}

	var root bundle.BuilderStep
	if len(stepsWithFlags) == 1 {
		root = stepsWithFlags[0]
	} else {
		root = bundle.AllOf(stepsWithFlags...)
	}

	tb, plan := bundle.NewBuilder().
		WithSigner(signer).
		SetEarliest(earliest).
		SetLatest(latest).
		With(root).
		BuildBundleAndPlan()

	txsInOrder := tb.GetTransactionsInReferencedOrder()
	signedTxs := make([]hexutil.Bytes, len(txsInOrder))
	for i, tx := range txsInOrder {
		data, err := tx.MarshalBinary()
		if err != nil {
			panic(err)
		}
		signedTxs[i] = data
	}

	rpcPlan, err := NewRPCExecutionPlanComposable(plan)
	if err != nil {
		panic(err)
	}
	return SubmitBundleArgs{
		SignedTransactions: signedTxs,
		ExecutionPlan:      rpcPlan,
	}
}
