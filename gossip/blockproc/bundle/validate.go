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

package bundle

import (
	"errors"
	"fmt"
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
)

var ErrBundleGasLimitTooLow = errors.New("gas limit of bundle transaction does not match the sum of the gas limits of the transactions in the bundle")

// ValidateTransactionBundle validates a bundle transaction.
// It checks that the transaction is a valid bundle transaction and that all transactions in the bundle belong to the same execution plan.
// If the transaction is a valid transaction bundle, it returns the decoded transaction bundle and nil (no error).
// If the transaction is not a bundle transaction, or if bundle transactions are not enabled, it returns nil,nil (no bundle, no error).
func ValidateTransactionBundle(
	envelopeTx *types.Transaction,
) (*TransactionBundle, *ExecutionPlan, error) {

	if !IsEnvelope(envelopeTx) {
		// not a bundle transaction, nothing to validate
		return nil, nil, nil
	}

	txBundle, err := decode(envelopeTx.Data())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode transaction bundle: %v", err)
	}

	// TODO: this function shall validate bundle correctness,
	// the current implementation is preliminary to enable prototyping.
	// This code needs to be developed

	chainId := envelopeTx.ChainId()
	if envelopeTx.Type() == types.LegacyTxType {
		for _, tx := range txBundle.Transactions {
			if tx.Type() != types.LegacyTxType {
				cur := tx.ChainId()
				if cur != nil && cur.Sign() != 0 {
					chainId = cur
					break
				}
			}
		}
	}

	var signer types.Signer = types.HomesteadSigner{}
	if chainId != nil && chainId.Sign() != 0 {
		signer = types.LatestSignerForChainID(chainId)
	}

	plan, err := txBundle.extractExecutionPlan(signer)
	if err != nil {
		return nil, nil, err
	}

	if plan.Latest < plan.Earliest {
		return nil, nil, fmt.Errorf("invalid empty block range [%d,%d] in execution plan", plan.Earliest, plan.Latest)
	}
	rangeSize := plan.Latest - plan.Earliest + 1

	if rangeSize > MaxBlockRange {
		return nil, nil, fmt.Errorf("invalid block range in execution plan, duration %d, limit %d", rangeSize, MaxBlockRange)
	}

	planHash := plan.Hash()
	for _, tx := range txBundle.Transactions {
		// check that all transactions in the bundle belong to the same execution plan
		if !belongsToExecutionPlan(tx, planHash) {
			return nil, nil, fmt.Errorf("transaction %s does not belong to the execution plan", tx.Hash().Hex())
		}
	}

	bundleGas := envelopeTx.Gas()
	// Ensure the transaction has more gas than the basic tx fee.
	intrGas, err := core.IntrinsicGas(
		envelopeTx.Data(),
		envelopeTx.AccessList(),
		nil,   // code auth is not used in the bundle transaction
		false, // bundle transaction is not a contract creation
		true,  // is homestead
		true,  // is istanbul
		true,  // is shanghai
	)
	if err != nil {
		return nil, nil, err
	}
	if envelopeTx.Gas() < intrGas {
		return nil, nil, fmt.Errorf("%w, gas should be more than %v", core.ErrIntrinsicGas, intrGas)
	}
	// gas limit of the bundle has to be exactly the aggregated gas of all the
	// transactions in the bundle or the intrinsic gas of the bundle
	// transaction, whichever is higher.
	gasLimit := uint64(0)
	for _, innerTx := range txBundle.Transactions {
		gasLimit += innerTx.Gas()
	}

	// EIP-7623 part of Prague revision: Floor data gas
	// see: https://eips.ethereum.org/EIPS/eip-7623
	floorDataGas, err := core.FloorDataGas(envelopeTx.Data())
	if err != nil {
		return nil, nil, err
	}
	if envelopeTx.Gas() < floorDataGas {
		return nil, nil, fmt.Errorf("%w: gas should be more than %d", core.ErrFloorDataGas, floorDataGas)
	}

	gasNeeded := max(gasLimit, intrGas, floorDataGas)
	if bundleGas != gasNeeded {
		return nil, nil, fmt.Errorf("%w: bundle gas limit %d but needs %d", ErrBundleGasLimitTooLow, envelopeTx.Gas(), gasNeeded)
	}

	return &txBundle, &plan, nil
}

// --- internal utilities ---

// extractExecutionPlan extracts the execution plan from the bundle, deriving
// the sender of each transaction using the provided signer.
func (tb *TransactionBundle) extractExecutionPlan(signer types.Signer) (ExecutionPlan, error) {

	txs := make([]ExecutionStep, 0, len(tb.Transactions))
	for _, tx := range tb.Transactions {

		// derive the sender before stripping the bundle-only mark from the access list
		// as this operation erases the original signature
		sender, err := signer.Sender(tx)
		if err != nil {
			return ExecutionPlan{}, fmt.Errorf("failed to derive sender: %v", err)
		}

		// hash the transaction after removing the bundle-only mark from the access list
		tx, err := removeBundleOnlyMark(tx)
		if err != nil {
			return ExecutionPlan{}, err
		}
		hash := signer.Hash(tx)

		txs = append(txs, ExecutionStep{
			From: sender,
			Hash: hash,
		})
	}

	return ExecutionPlan{
		Steps:    txs,
		Flags:    tb.Flags,
		Earliest: tb.Earliest,
		Latest:   tb.Latest,
	}, nil
}

// removeBundleOnlyMark is an utility function that removes the bundle-only mark
// from the access list of a transaction.
// This function is used to derive the hash of the transactions used in the
// execution plan, which is based on the transaction data without the bundle-only mark.
//
// By doing so, the signature of the transaction is erased. Therefore, the sender
// or the ChainId can no longer be derived from the resulting transaction.
func removeBundleOnlyMark(tx *types.Transaction) (*types.Transaction, error) {
	removeBundleOnlyMark := func(tx *types.Transaction) types.AccessList {
		var accessList types.AccessList
		for _, entry := range tx.AccessList() {
			if entry.Address == BundleOnly {
				continue
			}
			accessList = append(accessList, entry)
		}
		return accessList
	}

	var txData types.TxData
	switch tx.Type() {
	case types.AccessListTxType:
		txData = &types.AccessListTx{
			Nonce:      tx.Nonce(),
			GasPrice:   tx.GasPrice(),
			Gas:        tx.Gas(),
			To:         tx.To(),
			Value:      tx.Value(),
			Data:       tx.Data(),
			AccessList: removeBundleOnlyMark(tx),
		}
	case types.DynamicFeeTxType:
		txData = &types.DynamicFeeTx{
			Nonce:      tx.Nonce(),
			GasTipCap:  tx.GasTipCap(),
			GasFeeCap:  tx.GasFeeCap(),
			Gas:        tx.Gas(),
			To:         tx.To(),
			Value:      tx.Value(),
			Data:       tx.Data(),
			AccessList: removeBundleOnlyMark(tx),
		}
	default:
		// Note:
		// - Legacy transactions cannot be bundled, because they lack of access list
		// - Blob transactions have dubious usefulness in bundles and are not fully supported in Sonic
		// - SetCodeTransactions have special interactions with other transactions, and they are not supported in bundles
		return nil, fmt.Errorf("invalid bundle: unsupported transaction type %d", tx.Type())
	}
	return types.NewTx(txData), nil
}

// belongsToExecutionPlan checks if the given transaction correspond to one step in the execution plan.
func belongsToExecutionPlan(tx *types.Transaction, executionPlanHash common.Hash) bool {
	for _, entry := range tx.AccessList() {
		if entry.Address == BundleOnly &&
			slices.Contains(entry.StorageKeys, executionPlanHash) {
			return true
		}
	}
	return false
}
