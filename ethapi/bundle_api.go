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

package ethapi

import (
	"context"
	"fmt"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

type PublicBundleAPI struct {
	b Backend
}

func NewPublicBundleAPI(b Backend) *PublicBundleAPI {
	return &PublicBundleAPI{
		b: b,
	}
}

//go:generate stringer -type=BundleStatus -output bundle_status_string.go -trimprefix BundleStatus

type BundleStatus int

const (
	BundleStatusUnknown  BundleStatus = 0
	BundleStatusPending  BundleStatus = 1
	BundleStatusExecuted BundleStatus = 2
)

func (a *PublicBundleAPI) GetBundleInfo(
	ctx context.Context,
	executionPlanHash common.Hash,
) (BundleInfo, error) {

	// Since there is no global lock on the state, and a bundle can be executed
	// and removed from the pool in-between checking for the execution info and
	// the pool state, we check this twice. A valid bundle will only be removed
	// form the pool after it has been executed.
	for range 2 {

		// Check whether the given execution plan got already executed.
		info, err := a.b.GetBundleExecutionInfo(executionPlanHash)
		if err != nil {
			return BundleInfo{}, err
		}
		if info != nil {
			return BundleInfo{
				Status:   BundleStatusExecuted,
				Block:    &info.BlockNum,
				Position: &info.Position,
				Count:    &info.Count,
			}, nil
		}

		// Check whether the given execution plan is pending in the Tx Pool.
		if isInPool := a.b.IsBundleInPool(executionPlanHash); isInPool {
			return BundleInfo{
				Status: BundleStatusPending,
			}, nil
		}

	}

	// Otherwise, the state is unknown (default).
	return BundleInfo{}, nil
}

// BundleInfo is the JSON RPC message returned by the GetBundleInfo API, which
// provides information about the status of a transaction bundle.
type BundleInfo struct {
	Status   BundleStatus `json:"status"`
	Block    *uint64      `json:"block,omitempty"`
	Position *uint32      `json:"position,omitempty"`
	Count    *uint32      `json:"count,omitempty"`
}

type PrepareBundleArgs struct {
	// Transactions specifies the ordered list of transactions to be included in the bundle.
	Transactions []TransactionArgs `json:"transactions"`
	// ExecutionFlags defines the execution behavior of the bundle, such as whether it should be executed
	// exclusively or if it can be executed alongside other bundles. This is represented as a bitmask,
	// where specific bits correspond to different execution options.
	ExecutionFlags hexutil.Uint `json:"executionFlags"`
	// EarliestBlock specifies the earliest block number at which the bundle can be executed. This allows
	// users to set a lower bound on when their bundle should be considered for execution, ensuring it is
	// not included in blocks before a certain point in time.
	EarliestBlock rpc.BlockNumber `json:"earliestBlock"`
	// LatestBlock specifies the latest block number at which the bundle can be executed. This allows users
	// to set an upper bound on when their bundle should be considered for execution, ensuring it is not included in blocks after a certain point in time. If the bundle is not executed by this block, it will be considered expired and will not be executed.
	LatestBlock rpc.BlockNumber `json:"latestBlock"`
}

// PreparedBundle is the return type of the `sonic_prepareBundle` RPC method
type PreparedBundle struct {
	// Transactions specifies the ordered list of transactions to be included in the bundle.
	// These must be signed exactly as provided by the bundle_prepare RPC method; any modification
	// will invalidate the execution plan and result in an ill-formed bundle.
	Transactions []TransactionArgs `json:"transactions"`
	// Plan contains the execution plan that each bundled transaction references. This is provided
	// for verification purposes; users may independently compute and validate the execution plan hash.
	Plan bundle.ExecutionPlan `json:"plan,omitempty"`
}

// PrepareBundle implements the `sonic_prepareBundle` RPC method.
// This function streamlines the creation of transaction bundles by preparing an execution plan
// based on the provided transaction order and execution flags.
//
// It accepts a list of unsigned transactions, constructs the corresponding execution plan,
// and updates each transaction to include the bundler-only marker, ensuring they are executed
// exclusively as part of the specified plan.
//
// The returned transactions must be signed without altering any fields; any modification may
// invalidate the execution plan.
func (a *PublicBundleAPI) PrepareBundle(
	ctx context.Context,
	args PrepareBundleArgs,
) (*PreparedBundle, error) {

	gasCap := a.b.RPCGasCap()
	basefee := a.b.MinGasPrice()

	// 1) Read transactions from arguments and prepare fields
	from := make([]common.Address, len(args.Transactions))
	transactions := make([]*types.Transaction, len(args.Transactions))
	for i, txArgs := range args.Transactions {
		msg, err := txArgs.ToMessage(gasCap, basefee, log.Root())
		if err != nil {
			return nil, fmt.Errorf("failed to prepare bundle: transaction %d conversion error: %w", i, err)
		}

		tx := asTransaction(msg)

		switch tx.Type() {
		case types.LegacyTxType, types.BlobTxType, types.SetCodeTxType:
			return nil, fmt.Errorf("transaction %d has unsupported type %d: only AccessList and DynamicFee transactions are supported in bundles", i, tx.Type())
		}

		from[i] = msg.From
		transactions[i] = tx
	}

	// 2) Prepare execution plan
	chainID := a.b.ChainID()
	signer := types.LatestSignerForChainID(chainID)
	plan := bundle.ExecutionPlan{
		Flags:    bundle.ExecutionFlag(args.ExecutionFlags),
		Steps:    make([]bundle.ExecutionStep, len(transactions)),
		Earliest: uint64(args.EarliestBlock),
		Latest:   uint64(args.LatestBlock),
	}
	for i, tx := range transactions {
		plan.Steps[i] = bundle.ExecutionStep{
			From: from[i],
			Hash: signer.Hash(tx),
		}
	}

	// 3) Update bundle transactions with execution plan hash
	planHash := plan.Hash()
	for i := range transactions {
		tx := args.Transactions[i]
		var accessList types.AccessList
		if tx.AccessList != nil {
			accessList = *tx.AccessList
		}
		accessList = append(accessList, types.AccessTuple{
			Address: bundle.BundleOnly,
			StorageKeys: []common.Hash{
				planHash,
			}})
		tx.AccessList = &accessList
		args.Transactions[i] = tx
	}

	bundle := PreparedBundle{
		Transactions: args.Transactions,
		Plan:         plan,
	}

	return &bundle, nil
}

type SubmitBundleArgs struct {
	// SignedTransactions is the list of transactions that have been signed using the transaction arguments returned by the `sonic_prepareBundle` method.
	// These transactions must be included in the bundle exactly as they were prepared; any modification will invalidate the execution plan and result in an ill-formed bundle.
	SignedTransactions []hexutil.Bytes `json:"signedTransactions"`
	// ExecutionPlan contains the execution plan that each bundled transaction references.
	// This value must be provided as returned by the `sonic_prepareBundle` method;
	// any modification will invalidate the execution plan and result in an ill-formed bundle.
	ExecutionPlan bundle.ExecutionPlan `json:"plan,omitempty"`
}

// SubmitBundle implements the `sonic_submitBundle` RPC method, which allows users to submit a prepared transaction bundle for execution.
// This function accepts a list of transactions that have been signed using the transaction arguments returned by the `sonic_prepareBundle` method,
// along with execution parameters such as flags and block range.
// It validates the transactions against the execution plan and submits the bundle to the network for execution.
func (a *PublicBundleAPI) SubmitBundle(
	ctx context.Context,
	args SubmitBundleArgs,
) (common.Hash, error) {

	txBundle := bundle.TransactionBundle{
		Version:  bundle.BundleV1,
		Bundle:   make(types.Transactions, len(args.SignedTransactions)),
		Flags:    args.ExecutionPlan.Flags,
		Earliest: args.ExecutionPlan.Earliest,
		Latest:   args.ExecutionPlan.Latest,
	}

	// 1) Decode bundled transactions and compute total gas requirement
	var totalGas uint64
	for i, encodedTx := range args.SignedTransactions {

		tx := new(types.Transaction)
		if err := tx.UnmarshalBinary(encodedTx); err != nil {
			return common.Hash{}, fmt.Errorf("failed to decode bundled transaction %d: %w", i, err)
		}

		txBundle.Bundle[i] = tx
		totalGas += tx.Gas()
	}

	// 2)  Encode the bundle and compute if gas limits are sufficient to cover
	// both the payload and the data-related gas costs.
	data := bundle.Encode(txBundle)
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
			To:    &bundle.BundleAddress,
			Nonce: 0,
			Data:  data,
			Gas:   totalGas,
		})
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to sign bundle transaction: %w", err)
	}

	// 5) Validate generated transaction
	_, plan, err := bundle.ValidateTransactionBundle(tx, signer)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to validate bundle transaction: %w", err)
	}

	// 6) Submit the transaction to the network
	_, err = SubmitTransaction(ctx, a.b, tx)
	return plan.Hash(), err
}

func asTransaction(msg *core.Message) *types.Transaction {
	if msg.GasPrice == nil || msg.GasPrice.Sign() == 0 {
		// use dynamic fee transaction
		return types.NewTx(&types.DynamicFeeTx{
			To:         msg.To,
			Nonce:      msg.Nonce,
			Gas:        msg.GasLimit,
			GasFeeCap:  msg.GasFeeCap,
			GasTipCap:  msg.GasTipCap,
			Value:      msg.Value,
			Data:       msg.Data,
			AccessList: msg.AccessList,
		})
	} else {
		// use access list transaction
		return types.NewTx(&types.AccessListTx{
			To:         msg.To,
			Nonce:      msg.Nonce,
			Gas:        msg.GasLimit,
			GasPrice:   msg.GasPrice,
			Value:      msg.Value,
			Data:       msg.Data,
			AccessList: msg.AccessList,
		})
	}
}
