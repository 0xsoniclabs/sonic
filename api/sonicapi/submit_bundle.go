package sonicapi

import (
	"context"
	"fmt"

	"github.com/0xsoniclabs/sonic/api/ethapi"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

type SubmitBundleArgs struct {
	// SignedTransactions is the list of transactions that have been signed using
	// the transaction arguments returned by the `sonic_prepareBundle` method.
	// These transactions must be included in the bundle exactly as they were prepared;
	// any modification will invalidate the execution plan and result in an ill-formed bundle.
	SignedTransactions []hexutil.Bytes `json:"signedTransactions"`
	// ExecutionPlan contains the execution plan that each bundled transaction references.
	// This value must be provided as returned by the `sonic_prepareBundle` method;
	// any modification will invalidate the execution plan and result in an ill-formed bundle.
	ExecutionPlan RPCExecutionPlan `json:"executionPlan,omitempty"`
}

// SubmitBundle implements the `sonic_submitBundle` RPC method, which submits a prepared bundle for execution.
func (a *PublicBundleAPI) SubmitBundle(
	ctx context.Context,
	args SubmitBundleArgs,
) (common.Hash, error) {

	// validate input
	if len(args.SignedTransactions) == 0 {
		return common.Hash{}, fmt.Errorf("no transactions provided in the bundle")
	}

	txBundle := bundle.TransactionBundle{
		Transactions: make(types.Transactions, len(args.SignedTransactions)),
		Flags:        args.ExecutionPlan.Flags,
		Range: bundle.BlockRange{
			Earliest: uint64(args.ExecutionPlan.Earliest),
			Latest:   uint64(args.ExecutionPlan.Latest),
		},
	}

	// 1) Decode bundled transactions and compute total gas requirement
	var totalGas uint64
	for i, encodedTx := range args.SignedTransactions {

		tx := new(types.Transaction)
		if err := tx.UnmarshalBinary(encodedTx); err != nil {
			return common.Hash{}, fmt.Errorf("failed to decode bundled transaction %d: %w", i, err)
		}

		txBundle.Transactions[i] = tx
		totalGas += tx.Gas()
	}

	// 2)  Encode the bundle and compute if gas limits are sufficient to cover
	// both the payload and the data-related gas costs.
	data := txBundle.Encode()
	minGas, err := core.IntrinsicGas(data, nil, nil, false, true, true, true)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to finalize bundle: could not calculate intrinsic gas: %w", err)
	}
	floorDataGas, err := core.FloorDataGas(data)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to finalize bundle: could not calculate floor data gas: %w", err)
	}
	totalGas = max(totalGas, minGas, floorDataGas)

	// 3) Make a one use key to sign the bundle
	// TODO: key could be generated only once, but using a single key at the moment it would
	// generate a problem with nonces in the pool.
	key, err := crypto.GenerateKey()
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to generate signing key: %w", err)
	}

	// 4) Sign the bundle transaction with the one-use key and send it to the network
	signer := types.LatestSignerForChainID(a.b.ChainID())
	tx, err := types.SignNewTx(key, signer,
		&types.DynamicFeeTx{
			To:    &bundle.BundleProcessor,
			Nonce: 0,
			Data:  data,
			Gas:   totalGas,
		})
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to sign bundle transaction: %w", err)
	}

	// 5) Validate generated transaction
	_, plan, err := bundle.ValidateEnvelope(signer, tx)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to validate bundle transaction: %w", err)
	}
	submittedPlan := args.ExecutionPlan.ToBundleExecutionPlan()
	if plan.Hash() != submittedPlan.Hash() {
		return common.Hash{}, fmt.Errorf("provided execution plan does not match the signed bundle transaction")
	}

	// 6) Submit the transaction to the network
	_, err = ethapi.SubmitTransaction(ctx, a.b, tx)
	return plan.Hash(), err
}
