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
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/0xsoniclabs/sonic/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
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
func OpenEnvelope(
	signer types.Signer,
	tx *types.Transaction,
) (TransactionBundle, error) {
	if !IsEnvelope(tx) {
		return TransactionBundle{}, fmt.Errorf("not an envelope")
	}
	return decode(signer, tx.Data())
}

var (
	// BundleOnly is an address used in the access list of transactions to mark
	// them as bundle-only, meaning they are intended to be executed as part of
	// a bundle and not included in the block on their own.
	BundleOnly = common.HexToAddress("0x00000000000000000000000000000000000B0D1E")

	// BundleProcessor is the address to which envelope transactions are sending
	// their payload containing the bundle of transactions to be executed.
	BundleProcessor = common.HexToAddress("0x00000000000000000000000000000000B0D1EADD")
)

// TransactionBundle represents a bundle of transactions, which are to be executed
// sequentially within the same block. The bundle comprises an indexed list of
// transactions and an execution plan that specifies the order of execution of
// the transactions and the block range in which the bundle can be included.
type TransactionBundle struct {
	Transactions map[TxReference]*types.Transaction
	Plan         ExecutionPlan
}

// GetTransactionsInReferencedOrder returns the transactions stored in the bundle
// in the order they are referenced by the execution plan. The main intention
// for this function is to provide a readable mechanism for tests having build
// bundles using the builder to access the contained singed transactions.
// Note that the map retaining the transactions in the bundle on its own would
// make it difficult to retrieve individual transactions.
func (tb *TransactionBundle) GetTransactionsInReferencedOrder() []*types.Transaction {
	refs := tb.Plan.Root.GetTransactionReferencesInReferencedOrder()
	var txs []*types.Transaction
	for _, ref := range refs {
		txs = append(txs, tb.Transactions[ref])
	}
	return txs
}

func (tb *TransactionBundle) encode() ([]byte, error) {
	return encodeInternal(bundleEncodingVersion, tb)
}

// --- internal utilities ---

// removeBundleOnlyMark is an utility function that removes the bundle-only mark
// from the access list of a transaction.
// This function is used to derive the hash of the transactions used in the
// execution plan, which is based on the transaction data without the bundle-only mark.
//
// By doing so, the signature of the transaction is erased. Therefore, the sender
// or the ChainId can no longer be derived from the resulting transaction.
func removeBundleOnlyMark(tx *types.Transaction) (*types.Transaction, error) {
	curAccessList := tx.AccessList()
	newAccessList := make([]types.AccessTuple, 0, len(curAccessList))
	for _, cur := range curAccessList {
		if cur.Address != BundleOnly {
			newAccessList = append(newAccessList, cur)
		}
	}

	// Create a copy of the transaction data with the modified access list.
	txData := utils.GetTxData(tx)
	switch data := txData.(type) {
	case *types.AccessListTx:
		data.AccessList = newAccessList
	case *types.DynamicFeeTx:
		data.AccessList = newAccessList
	case *types.BlobTx:
		data.AccessList = newAccessList
	case *types.SetCodeTx:
		data.AccessList = newAccessList
	default:
		return nil, fmt.Errorf("unsupported transaction type: %d", tx.Type())
	}

	return types.NewTx(txData), nil
}

const (
	bundleEncodingVersion byte = 1
)

type bundleEncodingV1 struct {
	Transactions types.Transactions
	Plan         []byte
}

func encodeInternal(
	version byte,
	bundle *TransactionBundle,
) ([]byte, error) {

	// Create canonical form of list of included transactions.
	transactions := slices.Collect(maps.Values(bundle.Transactions))
	slices.SortFunc(transactions, func(a, b *types.Transaction) int {
		hashA := a.Hash()
		hashB := b.Hash()
		return bytes.Compare(hashA[:], hashB[:])
	})

	// serialize the execution plan and the full bundle data
	encodedPlan := bytes.NewBuffer(nil)
	buffer := bytes.Buffer{}
	err := errors.Join(
		bundle.Plan.encode(encodedPlan),
		rlp.Encode(&buffer, version),
		rlp.Encode(&buffer, bundleEncodingV1{
			transactions,
			encodedPlan.Bytes(),
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to encode transaction bundle: %w", err)
	}
	return buffer.Bytes(), nil
}

func decode(signer types.Signer, data []byte) (TransactionBundle, error) {
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

	bundle.Transactions = make(map[TxReference]*types.Transaction)
	for _, tx := range payload.Transactions {
		sender, err := types.Sender(signer, tx)
		if err != nil {
			return bundle, err
		}
		withoutMarker, err := removeBundleOnlyMark(tx)
		if err != nil {
			return bundle, fmt.Errorf("failed to remove bundle-only mark: %v", err)
		}
		txRef := TxReference{
			From: sender,
			Hash: signer.Hash(withoutMarker),
		}
		bundle.Transactions[txRef] = tx
	}

	if err := bundle.Plan.decode(bytes.NewReader(payload.Plan)); err != nil {
		return bundle, fmt.Errorf("failed to decode execution plan: %v", err)
	}
	return bundle, nil
}
