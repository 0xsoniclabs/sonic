package sonicapi

import (
	"testing"

	rpctest "github.com/0xsoniclabs/sonic/api/rpc_test"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func Test_SonicAPI_SubmitBundle(t *testing.T) {

	ctrl := gomock.NewController(t)
	pool := rpctest.NewMockTxPool(ctrl)

	be := rpctest.NewBackendBuilder(t).
		WithPool(pool).
		Build()
	api := NewPublicBundleAPI(be)

	wallet := rpctest.NewWallet(t)
	bundle, plan := bundle.NewBuilder(be.GetSigner()).
		With(bundle.Step(wallet.PrivateKey, &types.AccessListTx{Nonce: 1})).
		With(bundle.Step(wallet.PrivateKey, &types.AccessListTx{Nonce: 2})).
		BuildBundleAndPlan()

	pool.EXPECT().AddLocal(envelopeMatcher{
		signer: be.GetSigner(), planHash: plan.Hash(),
	}).Return(nil)

	args := encodeBundleRequest(t, plan, bundle.Transactions)
	_, err := api.SubmitBundle(t.Context(), args)
	require.NoError(t, err)
}

func encodeBundleRequest(t *testing.T, plan bundle.ExecutionPlan, txs []*types.Transaction) SubmitBundleArgs {
	encodedTransactions := make([]hexutil.Bytes, len(txs))
	for i, tx := range txs {
		data, err := tx.MarshalBinary()
		require.NoError(t, err)
		encodedTransactions[i] = hexutil.Bytes(data)
	}
	return SubmitBundleArgs{
		SignedTransactions: encodedTransactions,
		ExecutionPlan:      NewRPCExecutionPlan(plan),
	}
}

type envelopeMatcher struct {
	signer   types.Signer
	planHash common.Hash
}

func (m envelopeMatcher) Matches(x interface{}) bool {
	tx, ok := x.(*types.Transaction)
	if !ok {
		return false
	}
	if !bundle.IsEnvelope(tx) {
		return false
	}

	plan, err := bundle.ExtractExecutionPlan(m.signer, tx)
	if err != nil {
		return false
	}

	return plan.Hash() == m.planHash
}

func (m envelopeMatcher) String() string {
	return "envelopeMatcher for plan hash " + m.planHash.Hex()
}
