package bundle_validate

import (
	"errors"
	"fmt"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

var (
	ErrMissingBundleAddress              = errors.New("missing bundle address in access list")
	ErrMissingBundleOnlyMarker           = errors.New("missing bundle-only marker")
	ErrEmptyExecutionPlan                = errors.New("empty execution plan")
	ErrInvalidExecutionPlan              = errors.New("invalid execution plan")
	ErrFailedToExtractExecutionPlan      = errors.New("failed to extract execution plan from bundle")
	ErrInvalidPaymentTransaction         = errors.New("invalid payment transaction")
	ErrFailedToValidateTransaction       = errors.New("failed to validate transaction in bundle")
	ErrPaymentDoesNotTargetBundleAddress = errors.New("payment transaction does not target bundle address")
	ErrBundleGasLimitTooLow              = errors.New("bundle gas limit is too low to cover the gas of the bundle transactions and the payment transaction")
	ErrBundleOverpriced                  = errors.New("bundle gas price is too high compared to the transactions in the bundle")

	ErrPaymentSenderMismatch = errors.New("payment transaction sender does not match bundle transaction sender")
	ErrPaymentGasTooLow      = errors.New("payment transaction gas limit is too low to cover the payment transaction itself")
)

// ValidateTransactionBundle validates the transaction bundle by checking:
// - the presence of the bundle-only marker
// - the validity of the execution plan
//
// It returns an error if any of the checks fail.
func ValidateTransactionBundle(tx *types.Transaction, signer types.Signer) (bundle.TransactionBundle, error) {
	var res bundle.TransactionBundle
	if !bundle.IsTransactionBundle(tx) {
		return res, ErrMissingBundleAddress
	}
	// plan is encoded in data, no data means no plan.
	if len(tx.Data()) == 0 {
		return res, ErrEmptyExecutionPlan
	}

	bundle, err := bundle.Decode(tx.Data())
	if err != nil {
		return res, fmt.Errorf("%w: %v", ErrInvalidExecutionPlan, err)
	}

	bundleSender, err := signer.Sender(tx)
	if err != nil {
		return res, fmt.Errorf("failed to derive sender of the bundle transaction: %v", err)
	}

	plan, err := bundle.ExtractExecutionPlan(signer)
	if err != nil {
		return res, fmt.Errorf("%w: %v", ErrFailedToExtractExecutionPlan, err)
	}

	// TODO: some verification on the flag?
	if len(plan.Steps) == 0 {
		return res, ErrEmptyExecutionPlan
	}

	// validate all transaction in the bundle.
	for i, tx := range bundle.Bundle {
		err := validateBundleOnlyTx(tx, signer)
		if err != nil {
			return res, fmt.Errorf("%w %d: %v", ErrFailedToValidateTransaction, i, err)
		}
	}

	// validate payment transaction
	if err := validatePaymentTx(bundle.Payment, bundle.Bundle, signer, bundleSender); err != nil {
		return res, fmt.Errorf("%w: %w", ErrInvalidPaymentTransaction, err)
	}

	// gas limit of the bundle has to be at least the aggregated gas of
	// all the transactions in the bundle plus the payment transaction.
	gasLimit := bundle.Payment.Gas()
	for _, tx := range bundle.Bundle {
		gasLimit += tx.Gas()
	}
	if tx.Gas() < gasLimit {
		return res, fmt.Errorf("%w: bundle gas limit %d but needs %d", ErrBundleGasLimitTooLow, tx.Gas(), gasLimit)
	}

	// gas price of the bundle can be at most the lowest of prices between
	// the transactions in the bundle and the payment transaction.
	gasPrice := bundle.Payment.GasPrice()
	for _, tx := range bundle.Bundle {
		if tx.GasPrice().Cmp(gasPrice) < 0 {
			gasPrice = tx.GasPrice()
		}
	}
	if tx.GasPrice().Cmp(gasPrice) > 0 {
		return res, fmt.Errorf("%w: bundle gas price %d but lowest gas price in the bundle is %d", ErrBundleOverpriced, tx.GasPrice().Uint64(), gasPrice.Uint64())
	}

	return res, nil
}

func validateBundleOnlyTx(tx *types.Transaction, signer types.Signer) error {
	if !bundle.IsBundleOnly(tx) {
		return ErrMissingBundleOnlyMarker
	}
	return nil
}

func validatePaymentTx(
	paymentTx *types.Transaction,
	txs types.Transactions,
	signer types.Signer,
	bundleSender common.Address) error {

	if paymentTx.To() == nil || *paymentTx.To() != bundle.BundleAddress {
		return ErrPaymentDoesNotTargetBundleAddress
	}

	sender, err := signer.Sender(paymentTx)
	if err != nil {
		return fmt.Errorf("failed to derive sender of the payment transaction: %v", err)
	}
	if sender != bundleSender {
		return fmt.Errorf("payment transaction sender %s does not match bundle sender %s", sender.Hex(), bundleSender.Hex())
	}

	var txsGas uint64
	for _, tx := range txs {
		txsGas += tx.Gas()
	}
	if paymentTx.Value().Uint64() < txsGas {
		return ErrPaymentGasTooLow
	}
	return nil
}
