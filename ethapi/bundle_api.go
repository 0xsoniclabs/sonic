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
	"github.com/0xsoniclabs/sonic/gossip/gasprice/gaspricelimits"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
)

//go:generate mockgen -source=bundle_api.go -destination=bundle_api_mock.go -package=ethapi

type BundleApiBackend interface {
	TxPoolSenderBackend
	EthereunAPIBackend
	BlockchainApiBackend
	BundlesBackend
	RPCLimitsBackend
}

type PublicBundleAPI struct {
	b BundleApiBackend
}

func NewPublicBundleAPI(b BundleApiBackend) *PublicBundleAPI {
	return &PublicBundleAPI{
		b: b,
	}
}

// GetBundleInfo implements the `sonic_getBundleInfo` RPC method, which retrieves
// information about the execution of a transaction bundle.
//
// Since bundles are not stored in the blockchain like regular transactions,
// this method provides information about bundles executed in the recent past.
// The sonic client is not capable of tracking bundles indefinitely, and may return
// null for bundles executed too far in the past.
//
// In the same fashion as `eth_getTransactionReceipt`, this method returns a
// non-error response with null payload if the bundle hasn't been executed yet.
//
// If the bundle has been executed, it returns the block number, position of the
// first transaction of the bundle in the block, and the total number of non-reverted
// transactions.
func (a *PublicBundleAPI) GetBundleInfo(
	ctx context.Context,
	executionPlanHash common.Hash,
) (*RPCBundleInfo, error) {

	// Check whether the given execution plan got already executed.
	info := a.b.GetBundleExecutionInfo(executionPlanHash)
	if info != nil {
		return &RPCBundleInfo{
			Block:    toBlockNum(info.BlockNum),
			Position: toHexUint(uint64(info.Position)),
			Count:    toHexUint(uint64(info.Count)),
		}, nil
	}

	// Otherwise, the state is unknown (default).
	return nil, nil
}

// RPCBundleInfo is the JSON RPC message returned by the GetBundleInfo API, which
// provides information about the status of a transaction bundle.
type RPCBundleInfo struct {
	Block    *rpc.BlockNumber `json:"block,omitempty"`
	Position *hexutil.Uint    `json:"position,omitempty"`
	Count    *hexutil.Uint    `json:"count,omitempty"`
}

type PrepareBundleArgs struct {
	// Transactions specifies the ordered list of transactions to be included in the bundle.
	Transactions []TransactionArgs `json:"transactions"`
	// EarliestBlock specifies the earliest block number at which the bundle can be executed. This allows
	// users to set a lower bound on when their bundle should be considered for execution, ensuring it is
	// not included in blocks before a certain point in time.
	//
	// If left unspecified, the bundle will be eligible for execution starting from the next block after submission.
	EarliestBlock *rpc.BlockNumber `json:"earliestBlock"`
	// LatestBlock specifies the latest block number at which the bundle can be executed. This allows users
	// to set an upper bound on when their bundle should be considered for execution, ensuring it is
	// not included in blocks after a certain point in time. If the bundle is not executed by this block,
	// it will be considered expired and will not be executed.
	//
	// If left unspecified, the bundle will be eligible for execution until 1024 blocks after EarliestBlock.
	LatestBlock *rpc.BlockNumber `json:"latestBlock"`
}

// RPCPreparedBundle is the return type of the `sonic_prepareBundle` RPC method
type RPCPreparedBundle struct {
	// Transactions specifies the ordered list of transactions to be included in the bundle.
	// These must be signed exactly as provided by the bundle_prepare RPC method; any modification
	// will invalidate the execution plan and result in an ill-formed bundle.
	Transactions []TransactionArgs `json:"transactions"`
	// Plan contains the execution plan that each bundled transaction references. This is provided
	// for verification purposes; users may independently compute and validate the execution plan hash.
	Plan RPCExecutionPlan `json:"plan,omitempty"`
}

// PrepareBundle implements the `sonic_prepareBundle` RPC method.
// This function streamlines the creation of transaction bundles by preparing an execution plan
// based on the provided transaction order, to be executed within a specified block range.
//
// It accepts a list of unsigned transactions, constructs the corresponding execution plan,
// and updates each transaction to include the bundler-only marker, ensuring they are executed
// exclusively as part of the specified plan.
//
// Bundled transactions with uninitialized gas limits will have their gas estimated by this method, which will take into account
// potential state changes from previous transactions in the bundle. However, users can also choose to set gas limits on their own;
// in this case, the provided gas limits will be used without modification.
//
// Bundled transactions with uninitialized gas price fields (GasPrice for access list transactions,
// or both MaxFeePerGas and MaxPriorityFeePerGas for EIP-1559 capable transactions) will have their gas price
// set to the current suggested gas price by this method.
// However, users can also choose to set gas price fields on their own; in this case,
// the provided gas price fields will be used without modification, even if this is zero.
//
// The returned transactions must be signed without altering any fields; any modification may
// invalidate the execution plan and prevent the bundle from being executed.
func (a *PublicBundleAPI) PrepareBundle(
	ctx context.Context,
	args PrepareBundleArgs,
) (*RPCPreparedBundle, error) {

	gasCap := a.b.RPCGasCap()
	basefee := a.b.MinGasPrice()

	// Fill in transaction fields left empty by users
	var needsGasEstimation bool
	for _, tx := range args.Transactions {
		if tx.Gas == nil || *tx.Gas == 0 {
			needsGasEstimation = true
			break
		}
	}
	var gasLimits BundleGasLimits
	if needsGasEstimation {
		var err error
		// If any of the transactions has gas limit set to 0, we need to estimate gas for all of them,
		// since they might be mutually dependent and affect each other's gas estimation.
		gasLimits, err = a.EstimateGasForTransactions(ctx, args.Transactions, nil, nil, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to prepare bundle: gas estimation failed: %w", err)
		}
	}
	gasPrice := a.suggestGasPrice()
	for i, tx := range args.Transactions {
		if len(gasLimits.GasLimits) > i && (tx.Gas == nil) {
			tx.Gas = &gasLimits.GasLimits[i]
		}
		if tx.GasPrice == nil && tx.MaxFeePerGas == nil {
			if tx.MaxPriorityFeePerGas == nil {
				tx.GasPrice = gasPrice
			} else {
				tx.MaxFeePerGas = gasPrice
			}
		}
		args.Transactions[i] = tx
	}

	// Convert transactions *types.Transaction, which can be later serialized
	from := make([]common.Address, len(args.Transactions))
	transactions := make([]*types.Transaction, len(args.Transactions))
	for i, txArgs := range args.Transactions {
		msg, err := txArgs.ToMessage(gasCap, basefee, log.Root())
		if err != nil {
			return nil, fmt.Errorf("failed to prepare bundle: transaction %d conversion error: %w", i, err)
		}

		tx, err := asTransaction(msg)
		if err != nil {
			return nil, fmt.Errorf("failed to prepare bundle: transaction %d conversion error: %w", i, err)
		}

		from[i] = msg.From
		transactions[i] = tx
	}

	earliest := a.b.CurrentBlock().NumberU64() + 1
	if args.EarliestBlock != nil {
		earliest = uint64(*args.EarliestBlock)
	}
	latest := earliest + bundle.MaxBlockRange - 1
	if args.LatestBlock != nil {
		latest = uint64(*args.LatestBlock)
	}

	// Prepare execution plan
	chainID := a.b.ChainID()
	signer := types.LatestSignerForChainID(chainID)
	plan := bundle.ExecutionPlan{
		// Current api do not expose flags to users, this can be introduced in the future if needed.
		Flags:    bundle.ExecutionFlag(0),
		Steps:    make([]bundle.ExecutionStep, len(transactions)),
		Earliest: earliest,
		Latest:   latest,
	}
	for i, tx := range transactions {
		plan.Steps[i] = bundle.ExecutionStep{
			From: from[i],
			Hash: signer.Hash(tx),
		}
	}

	// Update bundle transactions with execution plan hash
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

	bundle := RPCPreparedBundle{
		Transactions: args.Transactions,
		Plan:         NewRPCExecutionPlan(plan),
	}

	return &bundle, nil
}

func (a *PublicBundleAPI) suggestGasPrice() *hexutil.Big {
	price := a.b.CurrentBlock().Header().BaseFee
	price = gaspricelimits.GetSuggestedGasPriceForNewTransactions(price)
	return (*hexutil.Big)(price)
}

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

	txBundle := bundle.TransactionBundle{
		Transactions: make(types.Transactions, len(args.SignedTransactions)),
		Flags:        args.ExecutionPlan.Flags,
		Earliest:     uint64(args.ExecutionPlan.Earliest),
		Latest:       uint64(args.ExecutionPlan.Latest),
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
	_, plan, err := bundle.ValidateTransactionBundle(tx)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to validate bundle transaction: %w", err)
	}

	// 6) Submit the transaction to the network
	_, err = SubmitTransaction(ctx, a.b.(Backend), tx)
	return plan.Hash(), err
}

func asTransaction(msg *core.Message) (*types.Transaction, error) {

	if len(msg.BlobHashes) != 0 || msg.BlobGasFeeCap != nil {
		return nil, fmt.Errorf("blob transactions are not supported in bundles")
	}
	if len(msg.SetCodeAuthorizations) != 0 {
		return nil, fmt.Errorf("transactions with set code authorization are not supported in bundles")
	}

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
		}), nil
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
		}), nil
	}
}

type BundleGasLimits struct {
	// GasLimits contains the estimated gas limit for each transaction in the
	// bundle, in the same order as the input transactions.
	GasLimits []hexutil.Uint64 `json:"gasLimits"`
}

// EstimateGasForTransactions implements the `sonic_estimateGasForTransactions` RPC method.
// It estimates the gas required for each provided transaction,
// applying state changes from previous transactions when estimating subsequent ones.
// Transactions that become invalid or fail during execution for later estimations are ignored.
// This method can help getting gas estimates for mutually depending transactions in bundles.
func (a *PublicBundleAPI) EstimateGasForTransactions(
	ctx context.Context,
	args []TransactionArgs,
	blockNrOrHash *rpc.BlockNumberOrHash,
	overrides *StateOverride,
	blockOverrides *BlockOverrides,
) (BundleGasLimits, error) {

	if len(args) > 16 {
		return BundleGasLimits{}, fmt.Errorf("too many transactions to estimate gas for: got %d, max is 16", len(args))
	}

	bNrOrHash := rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber)
	if blockNrOrHash != nil {
		bNrOrHash = *blockNrOrHash
	}

	gasCap := a.b.RPCGasCap()
	eval := &estimator{
		ctx:            ctx,
		b:              a.b.(Backend),
		blockNrOrHash:  bNrOrHash,
		overrides:      overrides,
		blockOverrides: blockOverrides,
		gasCap:         gasCap,
	}

	gasLimits, err := doEstimateGasForTransactions(args, eval)
	if err != nil {
		return BundleGasLimits{}, err
	}
	return BundleGasLimits{GasLimits: gasLimits}, nil
}

type GasEstimator interface {
	EstimateGas(args TransactionArgs, preArgs []TransactionArgs) (hexutil.Uint64, error)
}

type estimator struct {
	ctx            context.Context
	b              Backend
	blockNrOrHash  rpc.BlockNumberOrHash
	overrides      *StateOverride
	blockOverrides *BlockOverrides
	gasCap         uint64
}

func (e *estimator) EstimateGas(args TransactionArgs, preArgs []TransactionArgs) (hexutil.Uint64, error) {
	gas, err := DoEstimateGas(e.ctx, e.b, args, e.blockNrOrHash, e.overrides, e.blockOverrides, e.gasCap, preArgs)
	if err != nil {
		return 0, err
	}

	return gas, nil
}

func doEstimateGasForTransactions(
	args []TransactionArgs,
	eval GasEstimator,
) ([]hexutil.Uint64, error) {
	gasLimits := make([]hexutil.Uint64, len(args))
	preArgs := make([]TransactionArgs, 0, len(args))
	for i, arg := range args {
		gas, err := eval.EstimateGas(arg, preArgs)
		if err != nil {
			return nil, err
		}

		preArgs = append(preArgs, arg)
		preArgs[len(preArgs)-1].Gas = (*hexutil.Uint64)(&gas)

		gasLimits[i] = gas +
			hexutil.Uint64(params.TxAccessListAddressGas) + // add gas for bundle only address
			hexutil.Uint64(params.TxAccessListStorageKeyGas) // add gas for execution plan hash
	}
	return gasLimits, nil
}

type RPCExecutionStep struct {
	From common.Address `json:"from"`
	Hash common.Hash    `json:"hash"`
}

type RPCExecutionPlan struct {
	Flags    bundle.ExecutionFlag `json:"flags"`
	Steps    []RPCExecutionStep   `json:"steps"`
	Earliest rpc.BlockNumber      `json:"earliest"`
	Latest   rpc.BlockNumber      `json:"latest"`
}

func NewRPCExecutionPlan(plan bundle.ExecutionPlan) RPCExecutionPlan {
	steps := make([]RPCExecutionStep, len(plan.Steps))
	for i, step := range plan.Steps {
		steps[i] = RPCExecutionStep{
			From: step.From,
			Hash: step.Hash,
		}
	}

	return RPCExecutionPlan{
		Flags:    plan.Flags,
		Steps:    steps,
		Earliest: rpc.BlockNumber(plan.Earliest),
		Latest:   rpc.BlockNumber(plan.Latest),
	}
}

func toBlockNum(num uint64) *rpc.BlockNumber {
	bNr := rpc.BlockNumber(num)
	return &bNr
}

func toHexUint(num uint64) *hexutil.Uint {
	hNum := hexutil.Uint(num)
	return &hNum
}
