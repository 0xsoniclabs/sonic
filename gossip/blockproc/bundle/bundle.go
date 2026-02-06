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

	"github.com/Fantom-foundation/lachesis-base/common/bigendian"
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
	// TODO: study which members of a real transaction are really needed
	// for the execution to provide consent on the intended execution.
	// Note: ERC-4337 userOperations have a similar intent as this struct,
	// those include gas and gas price fields. At this moment I consider them
	// irrelevant for the semantics of the execution intention, but there may
	// still be some reasons why to commit to those ahead of time.
	// Adding these fields this early makes transaction construction more complex
	// as the gas estimation is done via simulation of the transaction.
	// Gas costs are volatile as well.
	To                    *common.Address
	From                  common.Address
	Nonce                 uint64
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

func (e ExecutionPlan) GetCost() *big.Int {
	// TODO: calculate the execution cost of the bundle.
	// This function may need an extra argument for the current basefee
	// cost = basefee * gasUsed [ + constant overhead]
	return big.NewInt(20000)
}

func (e *ExecutionPlan) Hash() common.Hash {
	hasher := sha3.New256()
	for _, msg := range e.Transactions {
		msg.Hash(hasher)
	}
	_, _ = hasher.Write(bigendian.Uint16ToBytes(uint16(e.Flags)))
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

// Validate performs basic validation checks on the bundle, invalid bundles
// shall not be included in the chain.
func (tb *TransactionBundle) Validate(signer types.Signer) error {

	executionPlan, err := tb.ExtractExecutionPlan(signer)
	if err != nil {
		return fmt.Errorf("failed to get execution plan: %v", err)
	}

	// validate payment tx
	if !BelongsToExecutionPlan(tb.Payment, executionPlan.Hash()) {
		return fmt.Errorf("payment transaction is not bundle-only")
	}

	// Validate transactions in the bundle
	for _, btx := range tb.Bundle {
		if !BelongsToExecutionPlan(btx, executionPlan.Hash()) {
			return fmt.Errorf("bundled transaction is not bundle-only")
		}
	}

	// TODO: some missing checks: (may require changes to the interface to access other data)
	// - is gas limit sufficient?
	// - is payment sufficient to cover the cost of the bundle?
	// - are nonces of payment and bundled transactions equal?
	// - are nonces of bundled transactions correct (not already used)?
	// - is the payment sender the same as the sender of the bundled transaction (no transaction in this implementation)

	return nil
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
	// should be ignored, and not cause the entire bundle to revert. If the
	// flag is present, the ignored transactions will no be part of the block
	// and no receipt will be generated for them. The execution of the bundle
	// will continue with the next transaction in the bundle.
	FlagIgnoreInvalid ExecutionFlag = 1 << 0
	// FlagIgnoreReverts indicates that failed transactions within the bundle
	// should be ignored, and not cause the entire bundle to revert. If the
	// flag is present, the failed transactions will be included in the block
	// and a receipt with a failed status will be generated for them. The
	// execution of the bundle will continue with the next transaction in the
	// bundle.
	FlagIgnoreReverts ExecutionFlag = 1 << 1
	// FlagAtMostOne indicates that the execution of the bundle should stop
	// after the first successful transaction. If the flag is present, once a
	// transaction in the bundle succeeds, the execution of the bundle will stop
	// and no more transactions in the bundle will be executed.
	// Notice that if no FlagIgnoreReverts or FlagIgnoreInvalid is set, the
	// execution will stop after the first non-successful transaction, even if
	// the FlagAtMostOne is set.
	FlagAtMostOne ExecutionFlag = 1 << 2
)

func (tb *TransactionBundle) RevertOnInvalidTransaction() bool {
	return tb.Flags&FlagIgnoreInvalid == 0
}

func (tb *TransactionBundle) RevertOnFailedTransaction() bool {
	return tb.Flags&FlagIgnoreReverts == 0
}

func (tb *TransactionBundle) StopAfterFirstSuccessfulTransaction() bool {
	return tb.Flags&FlagAtMostOne != 0
}
