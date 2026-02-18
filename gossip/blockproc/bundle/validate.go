package bundle

import (
	"fmt"
	big "math/big"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/core/types"
)

func ValidateTransactionBundle(
	tx *types.Transaction,
	txBundle TransactionBundle,
	signer types.Signer,
	baseFee *big.Int,
	upgrades opera.Upgrades) error {
	if !IsTransactionBundle(tx) {
		// not a bundle transaction, nothing to validate
		return nil
	}

	if !upgrades.Brio || !upgrades.TransactionBundles {
		return nil
	}

	// Check Payment:
	// - payment transaction must exist
	if txBundle.Payment == nil {
		return fmt.Errorf("invalid bundle: missing payment transaction")
	}
	// - payment transaction has the same sender and nonce as the original transaction
	paymentSender, err := signer.Sender(txBundle.Payment)
	if err != nil {
		return fmt.Errorf("invalid bundle: failed to derive sender of the payment transaction: %w", err)
	}
	bundlerAccount, err := signer.Sender(tx)
	if err != nil {
		return fmt.Errorf("invalid bundle: failed to derive sender of the bundle transaction: %w", err)
	}
	if paymentSender != bundlerAccount {
		return fmt.Errorf("invalid bundle: payment transaction sender mismatch; got %s, want %s", paymentSender.Hex(), bundlerAccount.Hex())
	}
	if txBundle.Payment.Nonce() != tx.Nonce() {
		return fmt.Errorf("invalid bundle: payment transaction nonce mismatch; got %d, want %d", txBundle.Payment.Nonce(), tx.Nonce())
	}

	// - payment transaction targets the correct recipient
	if txBundle.Payment.To() == nil || *txBundle.Payment.To() != BundleAddress {
		return fmt.Errorf("invalid bundle: payment transaction must be sent to the bundle contract; got %s, want %s", txBundle.Payment.To().Hex(), BundleAddress.Hex())
	}

	// - gas limit is equal or greater than the sum of the gas limits of all
	//   transactions in the bundle including the payment transaction
	totalGas := txBundle.Payment.Gas()
	for _, btx := range txBundle.Bundle {
		totalGas += btx.Gas()
	}
	if tx.Gas() < totalGas {
		return fmt.Errorf("invalid bundle: insufficient gas limit; got %d, want at least %d", tx.Gas(), totalGas)
	}

	// - gas price is adequate based on the current base fee
	if txBundle.Payment.GasPrice().Cmp(baseFee) < 0 {
		return fmt.Errorf("invalid bundle: payment transaction gas price too low; got %s, want at least %s", txBundle.Payment.GasPrice().String(), baseFee.String())
	}
	for _, btx := range txBundle.Bundle {
		if btx.GasPrice().Cmp(baseFee) < 0 {
			return fmt.Errorf("invalid bundle: transaction %s gas price too low; got %s, want at least %s", btx.Hash().Hex(), btx.GasPrice().String(), baseFee.String())
		}
	}
	// - gas price of all included transactions (including the payment transaction) is less or equal to the gas price of the bundle transaction
	if txBundle.Payment.GasPrice().Cmp(tx.GasPrice()) > 0 {
		return fmt.Errorf("invalid bundle: payment transaction gas price too high; got %s, want at most %s", txBundle.Payment.GasPrice().String(), tx.GasPrice().String())
	}
	for _, btx := range txBundle.Bundle {
		if btx.GasPrice().Cmp(tx.GasPrice()) > 0 {
			return fmt.Errorf("invalid bundle: transaction %s gas price too high; got %s, want at most %s", btx.Hash().Hex(), btx.GasPrice().String(), tx.GasPrice().String())
		}
	}

	// - Account has enough balance to cover the payment + the cost of the
	//   payment transaction itself (gas limit for the payment transaction * gas price + value)
	// TODO:

	// Check execution plan:
	plan, err := txBundle.ExtractExecutionPlan(signer)
	if err != nil {
		return err
	}

	planHash := plan.Hash()
	for _, tx := range txBundle.Bundle {
		// - all transactions in the bundle belong to the same execution plan
		if !BelongsToExecutionPlan(tx, planHash) {
			return fmt.Errorf("transaction %s does not belong to the execution plan", tx.Hash().Hex())
		}
	}

	return nil
}
