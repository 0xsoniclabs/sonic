package sonicapi

import (
	"testing"

	"github.com/0xsoniclabs/sonic/api/ethapi"
	rpctest "github.com/0xsoniclabs/sonic/api/rpc_test"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/stretchr/testify/require"
)

func TestBundlesAPI_Prepare_ComputesSameStepsAsBuilder(t *testing.T) {
	acc1 := rpctest.NewAccount()
	acc2 := rpctest.NewAccount()

	be := rpctest.NewBackendBuilder().Build()
	api := NewPublicBundleAPI(be)

	txRequest1 := ethapi.TransactionArgs{
		ChainID:  rpctest.ToHexBigInt(1),
		From:     acc1.Address(),
		To:       acc2.Address(),
		Nonce:    rpctest.ToHexUint64(0),
		Gas:      rpctest.ToHexUint64(21000),
		GasPrice: rpctest.ToHexBigInt(10_000),
	}
	prepared, err := api.PrepareBundle(
		t.Context(),
		PrepareBundleArgs{
			Transactions: []ethapi.TransactionArgs{
				txRequest1,
			},
		})
	require.NoError(t, err)
	require.Len(t, prepared.Transactions, 1)
	require.Equal(t, prepared.Plan.Steps[0].From, *acc1.Address())

	_, plan := bundle.NewBuilder(be.GetSigner()).
		With(bundle.Step(acc1.PrivateKey, txRequest1.ToTransaction())).
		SetEarliest(uint64(prepared.Plan.Earliest)).
		SetLatest(uint64(prepared.Plan.Latest)).
		BuildEnvelopeAndPlan()
	require.Equal(t, plan.Steps[0].Hash, prepared.Plan.Steps[0].Hash)

	// returned := ToBundlePlan(signer, *prepared.Plan)
}

// func ToBundlePlan(signer types.Signer, rpcPlan RPCExecutionPlan) bundle.ExecutionPlan {
// 	plan := bundle.ExecutionPlan{
// 		Steps: make([]bundle.ExecutionStep, len(args.Transactions)),
// 		Flags: args.Plan.Flags,
// 		Range: bundle.BlockRange{
// 			Earliest: uint64(args.Plan.Earliest),
// 			Latest:   uint64(args.Plan.Latest),
// 		},
// 	}
// 	for i, txArgs := range args.Transactions {
// 		tx := txArgs.ToTransaction()
// 		plan.Steps[i] = bundle.ExecutionStep{
// 			From: *txArgs.From,
// 			Hash: signer.Hash(tx),
// 		}
// 	}
// 	return plan
// }
