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
	"crypto/ecdsa"
	"fmt"
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

const (
	BundleV1 byte = 1
)

var (
	BundleOnly    = common.HexToAddress("0x00000000000000000000000000000000000B0D1E")
	BundleAddress = common.HexToAddress("0x00000000000000000000000000000000B0D1EADD")
	MaxBlockRange = uint64(1024)
)

// ExecutionFlag represents the execution flags that specify the behavior of the bundle execution.
// Zero value means the default behavior, which is to revert the entire bundle (except for payment transaction)
// if any of the transactions is invalid or fails.
type ExecutionFlag uint16

const (
	TolerateInvalid ExecutionFlag = 0b001
	TolerateFailed  ExecutionFlag = 0b010
	AllOf           ExecutionFlag = 0b000
	OneOf           ExecutionFlag = 0b100
)

func (e *ExecutionFlag) TolerateInvalid() bool {
	return e.getFlag(TolerateInvalid)
}

func (e *ExecutionFlag) TolerateFailed() bool {
	return e.getFlag(TolerateFailed)
}

func (e *ExecutionFlag) IsOneOf() bool {
	return e.getFlag(OneOf)
}

func (e *ExecutionFlag) SetTolerateInvalid(tolerateInvalid bool) {
	e.setFlag(TolerateInvalid, tolerateInvalid)
}

func (e *ExecutionFlag) SetTolerateFailed(tolerateFailed bool) {
	e.setFlag(TolerateFailed, tolerateFailed)
}

func (e *ExecutionFlag) SetOneOf(oneOf bool) {
	e.setFlag(OneOf, oneOf)
}

func (e *ExecutionFlag) getFlag(flag ExecutionFlag) bool {
	return *e&flag != 0
}

func (e *ExecutionFlag) setFlag(flag ExecutionFlag, value bool) {
	if value {
		*e = *e | flag
	} else {
		*e = *e &^ flag
	}
}

// ExecutionUnit represents either a single execution step or a nested execution layer.
type ExecutionUnit interface {
	asExecutionStep() *ExecutionStep
	asExecutionLayer() *ExecutionLayer
}

// ExecutionStep represents a single step in the execution plan,
// which corresponds to a transaction to be executed as part of the bundle.
type ExecutionStep struct {
	// From is the sender of the transaction, derived from the signature of the transaction
	From common.Address `json:"from"`
	// Hash is the transaction hash to be signed (not the hash of the transaction including its signature)
	// where the access list has been stripped from the bundle-only mark.
	Hash common.Hash `json:"hash"`
}

func (es *ExecutionStep) asExecutionStep() *ExecutionStep {
	return es
}

func (es *ExecutionStep) asExecutionLayer() *ExecutionLayer {
	return nil
}

type ExecutionLayer struct {
	Units []ExecutionUnit `json:"units"` // Steps to be executed in the bundle, in the order of execution
	Flags ExecutionFlag   `json:"flags"` // Execution flags that specify the behavior of the bundle execution
}

func (el *ExecutionLayer) asExecutionStep() *ExecutionStep {
	return nil
}

func (el *ExecutionLayer) asExecutionLayer() *ExecutionLayer {
	return el
}

// ExecutionPlan represents the plan for executing a bundle of transactions,
// to which every participant in the bundle shall agree on.
// The execution plan includes the list of steps to be executed, in the order of execution
type ExecutionPlan struct {
	Layer    ExecutionLayer `json:"layer"`    // Steps to be executed in the bundle, in the order of execution
	Earliest uint64         `json:"earliest"` // Earliest block this bundle can be included in.
	Latest   uint64         `json:"latest"`   // Latest block this bundle can be included in.
}

// IsInRange checks if the given block number is within the range of the
// execution plan.
func (e *ExecutionPlan) IsInRange(blockNum uint64) bool {
	return blockNum >= e.Earliest && blockNum <= e.Latest
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

type BundleUnit interface {
	AsTransaction() *BundleTransaction
	AsBundleLayer() *BundleLayer
	SignedToExecutionUnit(signer types.Signer) (ExecutionUnit, error)
	UnsignedToExecutionUnit(signer types.Signer) (ExecutionUnit, error)
}

type BundleTransaction struct {
	Tx     *types.Transaction
	Sender *Account `rlp:"-"` // If Tx is not signed yet, Sender can be used to derive the sender address for the execution plan. The Sender is not included in the RLP encoding because at that point Tx should be signed, and the sender can be derived from the signature of the transaction.
}

type Account struct {
	PrivateKey *ecdsa.PrivateKey
}

func NewAccount() *Account {
	key, _ := crypto.GenerateKey()
	return &Account{PrivateKey: key}
}

func (a *Account) Address() common.Address {
	return crypto.PubkeyToAddress(a.PrivateKey.PublicKey)
}

func (bt *BundleTransaction) AsTransaction() *BundleTransaction {
	return bt
}

func (bt *BundleTransaction) AsBundleLayer() *BundleLayer {
	return nil
}

func (bt *BundleTransaction) SignedToExecutionUnit(signer types.Signer) (ExecutionUnit, error) {
	sender, err := signer.Sender(bt.Tx)
	if err != nil {
		return nil, fmt.Errorf("failed to derive sender: %v", err)
	}

	// hash the transaction after removing the bundle-only mark from the access list
	tx, err := removeBundleOnlyMark(bt.Tx)
	if err != nil {
		return nil, err
	}
	hash := signer.Hash(tx)

	return &ExecutionStep{From: sender, Hash: hash}, nil
}

func (bt *BundleTransaction) UnsignedToExecutionUnit(signer types.Signer) (ExecutionUnit, error) {
	hash := signer.Hash(bt.Tx)

	return &ExecutionStep{From: bt.Sender.Address(), Hash: hash}, nil
}

type BundleLayer struct {
	Units []BundleUnit
	Flags ExecutionFlag
}

func (bl *BundleLayer) AsTransaction() *BundleTransaction {
	return nil
}

func (bl *BundleLayer) AsBundleLayer() *BundleLayer {
	return bl
}

func (bl *BundleLayer) SignedToExecutionUnit(signer types.Signer) (ExecutionUnit, error) {
	units := make([]ExecutionUnit, len(bl.Units))
	for i, u := range bl.Units {
		unit, err := u.SignedToExecutionUnit(signer)
		if err != nil {
			return nil, err
		}
		units[i] = unit
	}
	return &ExecutionLayer{Units: units, Flags: bl.Flags}, nil
}

func (bl *BundleLayer) UnsignedToExecutionUnit(signer types.Signer) (ExecutionUnit, error) {
	units := make([]ExecutionUnit, len(bl.Units))
	for i, u := range bl.Units {
		unit, err := u.UnsignedToExecutionUnit(signer)
		if err != nil {
			return nil, err
		}
		units[i] = unit
	}
	return &ExecutionLayer{Units: units, Flags: bl.Flags}, nil
}

func TotalGas(layer *BundleLayer) uint64 {
	gas := uint64(0)
	for _, unit := range layer.Units {
		if t := unit.AsTransaction(); t != nil {
			gas += t.Tx.Gas()
		} else {
			gas += TotalGas(unit.AsBundleLayer())
		}
	}
	return gas
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
	Version  byte
	Layer    BundleLayer
	Earliest uint64 // Earliest block this bundle can be included in.
	Latest   uint64 // Latest block this bundle can be included in.
}

func (bl *BundleLayer) toExecutionLayer(signer types.Signer) (ExecutionLayer, error) {
	units := make([]ExecutionUnit, len(bl.Units))
	for i, u := range bl.Units {
		unit, err := u.SignedToExecutionUnit(signer)
		if err != nil {
			return ExecutionLayer{}, err
		}
		units[i] = unit
	}
	return ExecutionLayer{Units: units, Flags: bl.Flags}, nil
}

// ExtractExecutionPlan extracts the execution plan from the bundle, deriving
// the sender of each transaction using the provided signer.
func (tb *TransactionBundle) ExtractExecutionPlan(signer types.Signer) (ExecutionPlan, error) {
	layer, err := tb.Layer.toExecutionLayer(signer)
	if err != nil {
		return ExecutionPlan{}, err
	}

	return ExecutionPlan{Layer: layer, Earliest: tb.Earliest, Latest: tb.Latest}, nil
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

// BelongsToExecutionPlan checks if the given transaction correspond to one step in the execution plan.
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
	_ = rlp.Encode(&buffer, TransactionBundleRlp{
		Layer:    BundleLayerRlp{Units: wrapAll(bundle.Layer.Units), Flags: bundle.Layer.Flags},
		Earliest: bundle.Earliest,
		Latest:   bundle.Latest,
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

	bundleRlp := TransactionBundleRlp{}
	if err := rlp.DecodeBytes(rest, &bundleRlp); err != nil {
		return bundle, fmt.Errorf("failed to decode transaction bundle: %v", err)
	}
	bundle.Layer = BundleLayer{Units: unwrapAll(bundleRlp.Layer.Units), Flags: bundleRlp.Layer.Flags}
	bundle.Earliest = bundleRlp.Earliest
	bundle.Latest = bundleRlp.Latest
	return bundle, nil
}
