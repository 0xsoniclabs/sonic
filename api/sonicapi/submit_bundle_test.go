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
	"crypto/ecdsa"
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

func Test_SubmitBundle_EmptySignedTransactions_ReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	pool := rpctest.NewMockTxPool(ctrl)
	be := rpctest.NewBackendBuilder(t).WithPool(pool).Build()

	args := SubmitBundleArgs{
		SignedTransactions: []hexutil.Bytes{},
		ExecutionPlan:      RPCExecutionPlanComposable{},
	}

	_, err := NewPublicBundleAPI(be).SubmitBundle(t.Context(), args)
	require.ErrorContains(t, err, "signedTransactions must not be empty")
}

func Test_SubmitBundle_InvalidExecutionPlan_ReturnsError(t *testing.T) {
	addr := common.Address{1}
	hash := common.Hash{2}

	singleStep := &RPCExecutionStepComposable{From: addr, Hash: hash}
	groupStep := &RPCExecutionPlanGroup[RPCExecutionStepComposable]{
		Steps: []RPCExecutionPlanLevel[RPCExecutionStepComposable]{
			{Single: singleStep},
		},
	}

	tests := []struct {
		name   string
		root   RPCExecutionPlanLevel[RPCExecutionStepComposable]
		errMsg string
	}{
		{
			name:   "root has neither single nor group",
			root:   RPCExecutionPlanLevel[RPCExecutionStepComposable]{},
			errMsg: "invalid execution plan root",
		},
		{
			name: "root has both single and group",
			root: RPCExecutionPlanLevel[RPCExecutionStepComposable]{
				Single: singleStep,
				Group:  groupStep,
			},
			errMsg: "invalid execution plan root",
		},
		{
			name: "nested step has neither single nor group",
			root: RPCExecutionPlanLevel[RPCExecutionStepComposable]{
				Group: &RPCExecutionPlanGroup[RPCExecutionStepComposable]{
					Steps: []RPCExecutionPlanLevel[RPCExecutionStepComposable]{
						{Single: singleStep},
						{}, // invalid nested level
					},
				},
			},
			errMsg: "invalid execution plan root",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			pool := rpctest.NewMockTxPool(ctrl)
			be := rpctest.NewBackendBuilder(t).WithPool(pool).Build()

			// Need at least one signed tx so the empty-slice guard doesn't trigger first.
			key, err := crypto.GenerateKey()
			require.NoError(t, err)
			signer := types.LatestSignerForChainID(be.ChainID())
			tx, err := types.SignNewTx(key, signer, &types.DynamicFeeTx{
				To:  &addr,
				Gas: params.TxGas,
			})
			require.NoError(t, err)
			txData, err := tx.MarshalBinary()
			require.NoError(t, err)

			args := SubmitBundleArgs{
				SignedTransactions: []hexutil.Bytes{txData},
				ExecutionPlan: RPCExecutionPlanComposable{
					BlockRange: RPCRange{Earliest: 1, Latest: 100},
					Root:       tt.root,
				},
			}

			_, err = NewPublicBundleAPI(be).SubmitBundle(t.Context(), args)
			require.ErrorContains(t, err, tt.errMsg)
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

			// Build valid args, then override block range to invalid values.
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

func Test_SubmitBundle_HierarchicalPlan_ReturnsExecutionPlanHash(t *testing.T) {
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
	key1, err := crypto.GenerateKey()
	require.NoError(t, err)
	key2, err := crypto.GenerateKey()
	require.NoError(t, err)
	key3, err := crypto.GenerateKey()
	require.NoError(t, err)

	// Build a hierarchical plan: AllOf(tx1, OneOf(tx2, tx3))
	// where tx2 and tx3 are in a OneOf group with TolerateFailed.
	inner := bundle.OneOf(
		bundle.Step(key2, &types.DynamicFeeTx{To: &addr, Gas: params.TxGas}).WithFlags(bundle.EF_TolerateFailed),
		bundle.Step(key3, &types.DynamicFeeTx{To: &addr, Gas: params.TxGas}).WithFlags(bundle.EF_TolerateFailed),
	)
	root := bundle.AllOf(
		bundle.Step(key1, &types.DynamicFeeTx{To: &addr, Gas: params.TxGas}),
		inner,
	)

	tb, plan := bundle.NewBuilder().
		WithSigner(signer).
		SetEarliest(5).
		SetLatest(50).
		With(root).
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

	args := SubmitBundleArgs{
		SignedTransactions: signedTxs,
		ExecutionPlan:      rpcPlan,
	}

	hash, err := NewPublicBundleAPI(be).SubmitBundle(t.Context(), args)
	require.NoError(t, err)
	require.NotNil(t, submitted)
	require.True(t, bundle.IsEnvelope(submitted))

	_, submittedPlan, err := bundle.ValidateEnvelope(signer, submitted)
	require.NoError(t, err)
	require.Equal(t, submittedPlan.Hash(), hash)
	require.EqualValues(t, 5, submittedPlan.Range.Earliest)
	require.EqualValues(t, 50, submittedPlan.Range.Latest)
}

func Test_SubmitBundle_PlanTxCountMismatch_ReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	pool := rpctest.NewMockTxPool(ctrl)

	be := rpctest.NewBackendBuilder(t).WithPool(pool).Build()
	signer := types.LatestSignerForChainID(be.ChainID())

	addr := common.Address{2}
	key1, err := crypto.GenerateKey()
	require.NoError(t, err)
	key2, err := crypto.GenerateKey()
	require.NoError(t, err)

	// Build plan with 2 txs but only provide 1 signed transaction.
	tb, plan := bundle.NewBuilder().
		WithSigner(signer).
		SetEarliest(1).
		SetLatest(10).
		With(bundle.AllOf(
			bundle.Step(key1, &types.DynamicFeeTx{To: &addr, Gas: params.TxGas}),
			bundle.Step(key2, &types.DynamicFeeTx{To: &addr, Gas: params.TxGas}),
		)).
		BuildBundleAndPlan()

	txsInOrder := tb.GetTransactionsInReferencedOrder()
	data, err := txsInOrder[0].MarshalBinary()
	require.NoError(t, err)

	rpcPlan, err := NewRPCExecutionPlanComposable(plan)
	require.NoError(t, err)

	args := SubmitBundleArgs{
		SignedTransactions: []hexutil.Bytes{data}, // only 1, plan has 2
		ExecutionPlan:      rpcPlan,
	}

	_, err = NewPublicBundleAPI(be).SubmitBundle(t.Context(), args)
	require.ErrorContains(t, err, "must match signedTransactions count")
}

func Test_SubmitBundle_TxNotBelongingToExecutionPlan_ReturnsError(t *testing.T) {
	tests := []struct {
		name   string
		makeTx func(t *testing.T, key *ecdsa.PrivateKey, signer types.Signer, addr common.Address) *types.Transaction
		errMsg string
	}{
		{
			name: "plain tx without bundle-only mark",
			makeTx: func(t *testing.T, key *ecdsa.PrivateKey, signer types.Signer, addr common.Address) *types.Transaction {
				tx, err := types.SignNewTx(key, signer, &types.DynamicFeeTx{
					To:  &addr,
					Gas: params.TxGas,
				})
				require.NoError(t, err)
				return tx
			},
			errMsg: "failed to validate bundle transaction",
		},
		{
			name: "tx with bundle-only mark but wrong plan hash",
			makeTx: func(t *testing.T, key *ecdsa.PrivateKey, signer types.Signer, addr common.Address) *types.Transaction {
				wrongPlanHash := common.Hash{0xff}
				tx, err := types.SignNewTx(key, signer, &types.DynamicFeeTx{
					To:  &addr,
					Gas: params.TxGas,
					AccessList: types.AccessList{
						{
							Address:     bundle.BundleOnly,
							StorageKeys: []common.Hash{wrongPlanHash},
						},
					},
				})
				require.NoError(t, err)
				return tx
			},
			errMsg: "failed to validate bundle transaction",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			pool := rpctest.NewMockTxPool(ctrl)
			be := rpctest.NewBackendBuilder(t).WithPool(pool).Build()
			signer := types.LatestSignerForChainID(be.ChainID())

			key, err := crypto.GenerateKey()
			require.NoError(t, err)
			addr := common.Address{2}

			tx := tt.makeTx(t, key, signer, addr)
			from, err := types.Sender(signer, tx)
			require.NoError(t, err)

			// Build plan referencing this tx. For a tx without the bundle-only mark,
			// the plan hash equals signer.Hash(tx) since removeBundleOnlyMark is a no-op.
			rpcPlan := RPCExecutionPlanComposable{
				BlockRange: RPCRange{Earliest: 1, Latest: 100},
				Root: RPCExecutionPlanLevel[RPCExecutionStepComposable]{
					Single: &RPCExecutionStepComposable{
						From: from,
						Hash: signer.Hash(tx),
					},
				},
			}

			txData, err := tx.MarshalBinary()
			require.NoError(t, err)

			args := SubmitBundleArgs{
				SignedTransactions: []hexutil.Bytes{txData},
				ExecutionPlan:      rpcPlan,
			}

			_, err = NewPublicBundleAPI(be).SubmitBundle(t.Context(), args)
			require.ErrorContains(t, err, tt.errMsg)
		})
	}
}

func makeEvmBlock(number uint64) *evmcore.EvmBlock {
	n := big.NewInt(int64(number))
	return &evmcore.EvmBlock{EvmHeader: evmcore.EvmHeader{Number: n}}
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
