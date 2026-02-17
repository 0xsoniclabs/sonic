// Copyright 2025 Sonic Operations Ltd
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
	"bytes"
	"fmt"
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

const (
	BundleV1 uint16 = 1
)

var (
	BundleOnly    = common.HexToAddress("0x00000000000000000000000000000000000B0D1E")
	BundleAddress = common.HexToAddress("0x00000000000000000000000000000000B0D1EADD")
)

// ExecutionFlag represents the execution flags that specify the behavior of the bundle execution.
// Zero value means the default behavior, which is to revert the entire bundle (except for payment transaction)
// if any of the transactions is invalid or fails.
type ExecutionFlag uint16

// ExecutionStep represents a single step in the execution plan,
// which corresponds to a transaction to be executed as part of the bundle.
type ExecutionStep struct {
	// From is the sender of the transaction, derived from the signature of the transaction
	From common.Address
	// Hash is the transaction hash to be signed (not the hash of the transaction including its signature)
	// where the access list has been stripped from the bundle-only mark.
	Hash common.Hash
}

// ExecutionPlan represents the plan for executing a bundle of transactions,
// to which every participant in the bundle shall agree on.
// The execution plan includes the list of steps to be executed, in the order of execution
type ExecutionPlan struct {
	Steps []ExecutionStep
	Flags ExecutionFlag
}

// Hash computes the execution plan hash
// The hash is computed with Keccak256, and is based on the RLP encoding of the type
// rlp([Steps, Flags]), where Steps is of type [[{20 bytes}, {32 bytes}]...] where
// ... means “zero or more of the thing to the left”
func (e *ExecutionPlan) Hash() common.Hash {
	hasher := crypto.NewKeccakState()
	_ = rlp.Encode(hasher, e)
	return common.BytesToHash(hasher.Sum(nil))
}

// TransactionBundle represents a bundle of transactions, which are to be executed
// sequentially within the same block. A payment transaction is included to
// pay ahead of time for the execution of the bundle.
// The execution flags can be used to specify the behavior regarding skipped,
// failed or successful transactions within the bundle, or whenever stop the
// execution after the first successful transaction.
// The default behavior (if no flags are set) is to revert the entire bundle
// if any of the transactions is invalid or fails.
// A reverted bundle will still include the payment transaction into the block,
// consuming the payment and nonce, and preventing this transaction from being
// included in future blocks.
type TransactionBundle struct {
	Version uint16
	Payment *types.Transaction
	Bundle  types.Transactions
	Flags   ExecutionFlag
}

// ExtractExecutionPlan extracts the execution plan from the bundle, deriving
// the sender of each transaction using the provided signer.
func (tb *TransactionBundle) ExtractExecutionPlan(signer types.Signer) (ExecutionPlan, error) {

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

	txs := make([]ExecutionStep, 0, len(tb.Bundle))
	for _, tx := range tb.Bundle {

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
			return ExecutionPlan{}, fmt.Errorf("invalid bundle: unsupported transaction type %d", tx.Type())
		}

		sender, err := signer.Sender(tx)
		if err != nil {
			return ExecutionPlan{}, fmt.Errorf("failed to derive sender: %v", err)
		}
		hash := signer.Hash(types.NewTx(txData))
		txs = append(txs, ExecutionStep{
			From: sender,
			Hash: hash,
		})
	}

	return ExecutionPlan{
		Steps: txs,
		Flags: tb.Flags,
	}, nil
}

// IsBundleOnly checks if the transaction is bundle-only, meaning it is intended
// to be executed as part of a bundle and not included in the block on its own.
func IsBundleOnly(tx *types.Transaction) bool {
	for _, entry := range tx.AccessList() {
		if entry.Address == BundleOnly {
			return true
		}
	}
	return false
}

// BelongsToExecutionPlan  correspond to one step in the execution plan,
func BelongsToExecutionPlan(tx *types.Transaction, executionPlanHash common.Hash) bool {
	for _, entry := range tx.AccessList() {
		if entry.Address == BundleOnly &&
			slices.Contains(entry.StorageKeys, executionPlanHash) {
			return true
		}
	}
	return false
}

// IsTransactionBundle checks if the transaction is a transaction bundle, meaning
// it is intended to be executed as a bundle containing multiple transactions
// and not included in the block on its own.
func IsTransactionBundle(tx *types.Transaction) bool {
	return tx.To() != nil && *tx.To() == BundleAddress
}

func Encode(bundle TransactionBundle) []byte {

	buffer := bytes.Buffer{}
	// encode into a buffer can only fail due to OOM
	// since we are encoding a struct with fixed fields, we can ignore the error
	_ = rlp.Encode(&buffer, bundle.Version)
	_ = rlp.Encode(&buffer, []any{
		bundle.Payment,
		bundle.Bundle,
		bundle.Flags,
	})
	return buffer.Bytes()
}

func Decode(data []byte) (TransactionBundle, error) {
	var bundle TransactionBundle

	_, version, rest, err := rlp.Split(data)
	if err != nil {
		return bundle, fmt.Errorf("failed to decode transaction bundle: %v", err)
	}
	if err := rlp.DecodeBytes(version, &bundle.Version); err != nil {
		return bundle, fmt.Errorf("failed to decode version: %v", err)
	}
	if bundle.Version != BundleV1 {
		return bundle, fmt.Errorf("unsupported bundle version: %d", bundle.Version)
	}

	payload := struct {
		Payment *types.Transaction
		Bundle  types.Transactions
		Flags   ExecutionFlag
	}{}
	if err := rlp.DecodeBytes(rest, &payload); err != nil {
		return bundle, fmt.Errorf("failed to decode transaction bundle: %v", err)
	}
	bundle.Payment = payload.Payment
	bundle.Bundle = payload.Bundle
	bundle.Flags = payload.Flags
	return bundle, nil
}
