package ethapi

import (
	"context"
	"fmt"
	"math/big"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

type PublicBundleAPI struct {
	b Backend
}

func NewPublicBundleAPI(b Backend) *PublicBundleAPI {
	return &PublicBundleAPI{b: b}
}

// BundleArgs encapsulates the parameters required for constructing a transaction bundle.
// It contains the prepared transactions, payment details, gas requirements, and the execution plan
// necessary for the bundle's integrity and proper execution.
type BundleArgs struct {
	// Transactions specifies the ordered list of transactions to be included in the bundle.
	// These must be signed exactly as provided by the bundle_prepare RPC method; any modification
	// may invalidate the execution plan and result in an ill-formed bundle.
	Transactions []TransactionArgs `json:"transactions"`
	// Payment defines the collateral transaction included in the bundle, which is executed prior
	// to the bundled transactions to mitigate potential abuse.
	Payment TransactionArgs `json:"payment"`
	// Gas represents the aggregate gas limit required for the entire bundle. The final transaction
	// bundle must utilize this gas limit to ensure sufficient resources are allocated for execution.
	Gas *hexutil.Uint64 `json:"gas,omitempty"`
	// Plan contains the execution plan that each bundled transaction references. This is provided
	// for verification purposes; users may independently compute and validate the execution plan hash.
	Plan bundle.ExecutionPlan `json:"plan,omitempty"`
}

// Prepare implements the `bundle_prepare` RPC method.
// This function streamlines the creation of transaction bundles by preparing an execution plan
// based on the provided transaction order and execution flags.
//
// It accepts a list of unsigned transactions, constructs the corresponding execution plan,
// and updates each transaction to include the bundler-only marker, ensuring they are executed
// exclusively as part of the specified plan.
//
// The returned transactions must be signed without altering any fields; any modification may
// invalidate the execution plan.
func (bApi *PublicBundleAPI) Prepare(
	ctx context.Context,
	transactionArgs []TransactionArgs,
	bundler common.Address,
	executionFlags uint8,
) (*BundleArgs, error) {

	gasCap := bApi.b.RPCGasCap()
	basefee := bApi.b.MinGasPrice()

	var maxGasPrice *big.Int
	cost := big.NewInt(0)
	totalGas := uint64(0)

	// 1) Read transactions from arguments and prepare fields
	from := make([]common.Address, len(transactionArgs))
	transactions := make([]*types.Transaction, len(transactionArgs))
	for i, txArgs := range transactionArgs {
		msg, err := txArgs.ToMessage(gasCap, basefee, log.Root())
		if err != nil {
			return nil, fmt.Errorf("failed to prepare bundle: transaction %d conversion error: %w", i, err)
		}

		fmt.Println("tx_gasPrice", msg.GasPrice, msg.GasFeeCap)
		// TODO: validate transactions?
		// - allowed types
		// - chain ID

		sanitizeMessage(msg, basefee)

		from[i] = msg.From
		transactions[i] = asTransaction(msg)
		price := greaterOf(msg.GasPrice, msg.GasFeeCap)
		maxGasPrice = greaterOf(maxGasPrice, price)
		cost = new(big.Int).Add(cost,
			new(big.Int).Mul(price, big.NewInt(int64(msg.GasLimit))))
		totalGas += msg.GasLimit
	}

	// 2) Prepare execution plan
	chainID := bApi.b.ChainID()
	signer := types.LatestSignerForChainID(chainID)
	plan := bundle.ExecutionPlan{
		Flags: bundle.ExecutionFlag(executionFlags),
		Steps: make([]bundle.ExecutionStep, len(transactions)),
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
		tx := transactionArgs[i]
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
		transactionArgs[i] = tx
	}

	// 4) Make payment transaction
	cost = new(big.Int).Add(cost, getCostOverhead(len(transactions)))
	gasPrice, err := bApi.suggestGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare bundle: could not suggest gas price: %w", err)
	}

	payment := TransactionArgs{
		From:     &bundler, // Note: without sender, transaction cannot be estimated. This requires the bundler argument
		To:       &bundle.BundleAddress,
		GasPrice: gasPrice,
		Value:    (*hexutil.Big)(cost),
		AccessList: &types.AccessList{
			{
				Address:     bundle.BundleOnly,
				StorageKeys: []common.Hash{}, // Note: required value for serialization
			},
		},
	}
	paymentGas, err := bApi.estimateGasForPayment(ctx, payment)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare bundle: payment transaction gas estimation failed: %w", err)
	}
	payment.Gas = &paymentGas
	totalGas += uint64(paymentGas)

	bundle := BundleArgs{
		Transactions: transactionArgs,
		Payment:      payment,
		Gas:          (*hexutil.Uint64)(&totalGas),
		Plan:         plan,
	}

	return &bundle, nil
}

// Finalize implements the `bundle_finalize` RPC method.
// This function finalizes the transaction bundle by decoding the provided transactions, computing
// the total gas requirement, and constructing the final transaction that the bundler must sign.
//
// It accepts the raw transactions and payment details, decodes them, and calculates the necessary
// gas to ensure successful execution. The returned transaction must be signed by the bundler and
// sent to the network without modification; any alteration may invalidate the bundle.
func (bApi *PublicBundleAPI) Finalize(
	ctx context.Context,
	rawTransactions []hexutil.Bytes,
	rawPayment hexutil.Bytes,
	bundler common.Address,
	flags uint8) (*TransactionArgs, error) {

	txBundle := bundle.TransactionBundle{
		Version: bundle.BundleV1,
		Bundle:  make(types.Transactions, len(rawTransactions)),
	}

	// 1) Decode bundled transactions and compute total gas requirement, and
	// the minimum price which enables all transactions
	var totalGas uint64
	maxGasPrice := big.NewInt(0)
	for i, encodedTx := range rawTransactions {
		tx := new(types.Transaction)
		if err := tx.UnmarshalBinary(encodedTx); err != nil {
			return nil, fmt.Errorf("failed to decode bundled transaction %d: %w", i, err)
		}
		txBundle.Bundle[i] = tx
		totalGas += tx.Gas()
		maxGasPrice = greaterOf(maxGasPrice, tx.GasPrice())
	}
	// 2) Decode payment transaction and add its gas requirement to total
	txBundle.Payment = new(types.Transaction)
	if err := txBundle.Payment.UnmarshalBinary(rawPayment); err != nil {
		return nil, fmt.Errorf("failed to decode payment transaction: %w", err)
	}
	totalGas += txBundle.Payment.Gas()
	maxGasPrice = greaterOf(maxGasPrice, txBundle.Payment.GasPrice())

	// 3)  Encode the bundle and compute if gas limits are sufficient to cover
	// both the payload and the data-related gas costs.
	data := bundle.Encode(txBundle)
	minGas, err := core.IntrinsicGas(data, nil, nil, false, true, true, true)
	if err != nil {
		return nil, fmt.Errorf("failed to finalize bundle: could not calculate intrinsic gas: %w", err)
	}
	totalGas = max(totalGas, minGas)

	// 4) Construct the final transaction, which has to be signed by the bundler
	hexData := hexutil.Bytes(data)
	nonce := hexutil.Uint64(txBundle.Payment.Nonce())
	return &TransactionArgs{
		From:         &bundler,
		To:           &bundle.BundleAddress,
		Nonce:        &nonce,
		Gas:          (*hexutil.Uint64)(&totalGas),
		MaxFeePerGas: (*hexutil.Big)(maxGasPrice),
		// Max priority fee guarantees that the transaction replaces the payment
		// transaction, if both meet on the tx_pool.
		// TODO: decide whenever this affects priorities, and we want to fine tune it.
		MaxPriorityFeePerGas: (*hexutil.Big)(maxGasPrice),
		Data:                 &hexData,
	}, nil
}

func (bApi *PublicBundleAPI) estimateGasForPayment(ctx context.Context, payment TransactionArgs) (hexutil.Uint64, error) {
	api := NewPublicBlockChainAPI(bApi.b)
	return api.EstimateGas(ctx, payment, nil, nil, nil)
}

func (bApi *PublicBundleAPI) suggestGasPrice(ctx context.Context) (*hexutil.Big, error) {
	api := NewPublicEthereumAPI(bApi.b)
	return api.GasPrice(ctx)
}

func getCostOverhead(numTxs int) *big.Int {
	v := new(big.Int).Mul(big.NewInt(int64(numTxs)), big.NewInt(10000)) // TODO: proper cost estimation
	v = new(big.Int).Add(v, big.NewInt(50000))                          // base cost
	return v
}

func greaterOf(a, b *big.Int) *big.Int {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	if a.Cmp(b) >= 0 {
		return a
	}
	return b
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

type SignedBundle struct {
	Transactions hexutil.Bytes `json:"transactions"`
	Payment      hexutil.Bytes `json:"payment"`
}

func sanitizeMessage(msg *core.Message, defaultGasPrice *big.Int) {
	if msg.GasPrice == nil && msg.GasFeeCap == nil && msg.GasTipCap == nil {
		msg.GasPrice = defaultGasPrice
	}
}
