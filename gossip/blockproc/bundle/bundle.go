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
	"bytes"
	"fmt"
	"math"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

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

// IsEnvelope checks if the transaction is an envelope of a bundle, meaning
// it is carrying the encoding of a list of transactions to be executed as a
// bundle.
// Note: this function does not check the validity of the bundle data.
func IsEnvelope(tx *types.Transaction) bool {
	return tx.To() != nil && *tx.To() == BundleProcessor
}

// OpenEnvelope extracts the bundle enclosed in the given envelope.
func OpenEnvelope(tx *types.Transaction) (TransactionBundle, error) {
	if !IsEnvelope(tx) {
		return TransactionBundle{}, fmt.Errorf("not an envelope")
	}
	return decode(tx.Data())
}

// ExtractExecutionPlan extracts the execution plan from the given envelope.
func ExtractExecutionPlan(
	signer types.Signer,
	tx *types.Transaction,
) (ExecutionPlan, error) {
	bundle, err := OpenEnvelope(tx)
	if err != nil {
		return ExecutionPlan{}, err
	}
	plan, err := bundle.extractExecutionPlan(signer)
	if err != nil {
		return ExecutionPlan{}, err
	}
	return plan, nil
}

var (
	// BundleOnly is an address used in the access list of transactions to mark
	// them as bundle-only, meaning they are intended to be executed as part of
	// a bundle and not included in the block on their own.
	BundleOnly = common.HexToAddress("0x00000000000000000000000000000000000B0D1E")

	// BundleProcessor is the address to which envelope transactions are sending
	// their payload containing the bundle of transactions to be executed.
	BundleProcessor = common.HexToAddress("0x00000000000000000000000000000000B0D1EADD")

	// MaxBlockRange is the maximum allowed block range (Latest - Earliest) for
	// allowed for the validity period of a bundle.
	MaxBlockRange = uint64(1024)
)

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
	Steps []ExecutionStep // Steps to be executed in the bundle, in the order of execution
	Flags ExecutionFlags  // Execution flags that specify the behavior of the bundle execution
	Range BlockRange      // Block range [Earliest, Latest] in which the bundle can be included
}

// BlockRange represents a range of blocks, defined by an earliest and latest
// block number. The covered block range is a closed interval [Earliest, Latest],
// meaning that the earliest and latest blocks are included in the range.
// For instance, [0,0] is a valid block range that only includes the block
// number 0, while [0,1] includes both blocks 0 and 1. An interval where Latest
// is smaller than Earliest, such as [1,0], is a valid empty range.
type BlockRange struct {
	Earliest uint64
	Latest   uint64
}

// MakeMaxRangeStartingAt creates a block range of maximum allowed size, starting
// at the given block number.
func MakeMaxRangeStartingAt(blockNum uint64) BlockRange {
	latest := blockNum + MaxBlockRange - 1
	if blockNum > math.MaxUint64-MaxBlockRange {
		// if the starting block number is too close to maxUint64,
		// we cannot create a full range of MaxBlockRange blocks without overflowing.
		// In this case, we create the largest possible range starting at blockNum,
		// which ends at the maximum uint64 value.
		latest = math.MaxUint64
	}
	return BlockRange{
		Earliest: blockNum,
		Latest:   latest,
	}
}

// Size returns the size of the block range, which is the number of blocks
// included in the range.
func (r BlockRange) Size() uint64 {
	if r.Latest < r.Earliest {
		return 0
	}
	// check for overflow, which can happen if Latest is close to the maximum uint64 value and Earliest is close to 0
	if r.Latest-r.Earliest > math.MaxUint64-1 {
		return math.MaxUint64
	}
	return r.Latest - r.Earliest + 1
}

// IsInRange checks if the given block number is within this block range.
// The range is a closed interval [Earliest, Latest], meaning that blocks with
// numbers from Earliest through Latest (inclusive) are considered in range.
func (r BlockRange) IsInRange(blockNum uint64) bool {
	return blockNum >= r.Earliest && blockNum <= r.Latest
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
	Transactions types.Transactions
	Flags        ExecutionFlags
	Range        BlockRange // Block range [Earliest, Latest] in which the bundle can be included
}

func (tb *TransactionBundle) Encode() []byte {
	return encodeInternal(bundleEncodingVersion, tb)
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
		Steps: txs,
		Flags: tb.Flags,
		Range: tb.Range,
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

const (
	bundleEncodingVersion byte = 1
)

type bundleEncodingV1 struct {
	Bundle   types.Transactions
	Flags    ExecutionFlags
	Earliest uint64
	Latest   uint64
}

func encodeInternal(
	version byte,
	bundle *TransactionBundle,
) []byte {

	buffer := bytes.Buffer{}
	// encode into a buffer can only fail due to OOM
	// since we are encoding a struct with fixed fields, we can ignore the error
	_ = rlp.Encode(&buffer, version)
	_ = rlp.Encode(&buffer, bundleEncodingV1{
		bundle.Transactions,
		bundle.Flags,
		bundle.Range.Earliest,
		bundle.Range.Latest,
	})
	return buffer.Bytes()
}

func decode(data []byte) (TransactionBundle, error) {
	var bundle TransactionBundle

	_, encodedVersion, rest, err := rlp.Split(data)
	if err != nil {
		return bundle, fmt.Errorf("failed to decode transaction bundle: %v", err)
	}
	var version byte
	if err := rlp.DecodeBytes(encodedVersion, &version); err != nil {
		return bundle, fmt.Errorf("failed to decode version: %v", err)
	}
	if version != bundleEncodingVersion {
		return bundle, fmt.Errorf("unsupported bundle version: %d", version)
	}

	var payload bundleEncodingV1
	if err := rlp.DecodeBytes(rest, &payload); err != nil {
		return bundle, fmt.Errorf("failed to decode transaction bundle: %v", err)
	}
	bundle.Transactions = payload.Bundle
	bundle.Flags = payload.Flags
	bundle.Range = BlockRange{
		Earliest: payload.Earliest,
		Latest:   payload.Latest,
	}
	return bundle, nil
}
