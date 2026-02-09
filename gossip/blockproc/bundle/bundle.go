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
	"crypto/sha3"
	"fmt"
	"io"
	"math/big"
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

var (
	BundleOnly    = common.HexToAddress("0x00000000000000000000000000000000000B0D1E")
	BundleAddress = common.HexToAddress("0x00000000000000000000000000000000B0D1EADD")
)

// MetaTransaction represents the essential information of a transaction
// that is relevant for the execution of a bundle.
type MetaTransaction struct {
	To                    *common.Address
	From                  common.Address
	Nonce                 uint64
	GasLimit              uint64
	Value                 *big.Int
	Data                  []byte
	BlobHashes            []common.Hash
	SetCodeAuthorizations []types.SetCodeAuthorization
}

func (tm *MetaTransaction) Hash(hasher io.Writer) {
	_ = rlp.Encode(hasher, []any{
		tm.To,
		tm.From,
		tm.Nonce,
		tm.GasLimit,
		tm.Value,
		tm.Data,
		tm.BlobHashes,
		tm.SetCodeAuthorizations,
	})
}

// ExecutionPlan represents the plan for executing a bundle of transactions,
// to which every participant in the bundle shall agree on.
// The execution plan includes the list of transactions to be executed, and
// the execution flags that specify the behavior of the bundle execution.
type ExecutionPlan struct {
	Transactions []MetaTransaction
	Flags        ExecutionFlag
}

const BundleCostOverhead = 20_000

func (e ExecutionPlan) GetCost(gasPrice *big.Int) *big.Int {
	cost := new(big.Int).Mul(big.NewInt(BundleCostOverhead), gasPrice)
	for _, tx := range e.Transactions {
		cost.Add(cost, new(big.Int).Mul(gasPrice, new(big.Int).SetUint64(tx.GasLimit)))
	}
	return cost
}

func (e *ExecutionPlan) Hash() common.Hash {
	hasher := sha3.New256()
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
	Bundle  types.Transactions
	Payment *types.Transaction
	Flags   ExecutionFlag
}

// ExtractExecutionPlan extracts the execution plan from the bundle, deriving
// the sender of each transaction using the provided signer.
func (tb *TransactionBundle) ExtractExecutionPlan(signer types.Signer) (ExecutionPlan, error) {
	msgs := make([]MetaTransaction, 0, len(tb.Bundle))

	for _, btx := range tb.Bundle {
		from, err := types.Sender(signer, btx)
		if err != nil {
			return ExecutionPlan{}, fmt.Errorf("failed to derive sender for bundled tx: %v", err)
		}

		msgs = append(msgs, MetaTransaction{
			To:                    btx.To(),
			From:                  from,
			Nonce:                 btx.Nonce(),
			GasLimit:              btx.Gas(),
			Value:                 btx.Value(),
			Data:                  btx.Data(),
			BlobHashes:            btx.BlobHashes(),
			SetCodeAuthorizations: btx.SetCodeAuthorizations(),
		})
	}

	return ExecutionPlan{
		Transactions: msgs,
		Flags:        tb.Flags,
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

// UnpackTransactionBundle attempts to decode the binary bundle data
// attached in the data field of a transaction bundle,
// It Returns an error if the transaction is not a bundle or if the data is malformed.
func UnpackTransactionBundle(tx *types.Transaction) (TransactionBundle, error) {

	if tx.To() == nil || *tx.To() != BundleAddress {
		return TransactionBundle{}, fmt.Errorf("failed to unpack bundle, not a transaction bundle")

	}

	var bundle TransactionBundle
	if err := rlp.DecodeBytes(tx.Data(), &bundle); err != nil {
		return TransactionBundle{}, fmt.Errorf("failed to unpack bundle, %v", err)
	}

	return bundle, nil
}

// ExecutionFlag represents the execution flags that specify the behavior of the bundle execution.
// Zero value means the default behavior, which is to revert the entire bundle
// if any of the transactions is invalid or fails.
type ExecutionFlag uint16

const (
	// FlagIgnoreInvalid indicates that invalid transactions within the bundle
	// should be ignored, and not cause the entire bundle to revert.
	// if set to true, invalid transactions are ignored; a transaction is
	// invalid if there is an invalid nonce, insufficient balance,
	// insufficient intrinsic gas, a too high gas price, or a too long data
	// section; if false, the presence of a skipped transaction results
	// in all transactions in the bundle being processed before to be reverted,
	// effectively causing all transactions to be skipped;
	FlagIgnoreInvalid ExecutionFlag = 1
	// FlagIgnoreFailed indicates that failed transactions within the bundle
	// should be ignored, and not cause the entire bundle to revert.
	// if set to true, failed transactions are ignored; if false, any failed
	// transaction causes all transactions to be rolled back and to be treated
	// like skipped transactions
	FlagIgnoreFailed ExecutionFlag = 1 << 1
	// FlagAtMostOne indicates that the execution of the bundle should stop
	// after the first successful transaction. If set to true, once a
	// transaction in the bundle succeeds, the execution of the bundle will stop
	// and subsequent transactions in the bundle will be ignored.
	// Notice that if no FlagIgnoreFailed or FlagIgnoreInvalid is set, the
	// execution will stop after the first non-successful transaction, even if
	// the FlagAtMostOne is set.
	FlagAtMostOne ExecutionFlag = 1 << 2
)

func (tb *TransactionBundle) RevertOnInvalidTransaction() bool {
	return tb.Flags&FlagIgnoreInvalid == 0
}

func (tb *TransactionBundle) RevertOnFailedTransaction() bool {
	return tb.Flags&FlagIgnoreFailed == 0
}

func (tb *TransactionBundle) StopAfterFirstSuccessfulTransaction() bool {
	return tb.Flags&FlagAtMostOne != 0
}
