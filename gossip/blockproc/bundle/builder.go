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
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

// This file offers utilities to build bundles from transaction data. The most
// generic format is the NewBundle function, enabling the creation of an
// envelope transaction carrying a bundle as follows:
//
//   envelope := NewBuilder().
// 		WithFlags(EF_AllOf|EF_TolerateFailed).
// 		Earliest(12).
// 		Latest(15).
// 		With(
// 			Step(key, &types.AccessListTx{
// 				Nonce: 1,
// 			}),
// 			Step(key, &types.AccessListTx{
// 				Nonce: 2,
// 			}),
// 		).Build()
//
// The resulting envelope carries a valid bundle of signed transactions.
// For convenience, further abbreviations are supported. For example:
//
//    envelopeA := AllOf(
// 			Step(key, &types.AccessListTx{
// 				Nonce: 1,
// 			}),
// 			Step(key, &types.AccessListTx{
// 				Nonce: 2,
// 			}),
//    )
//
// Also nested bundles are supported by using
//
//    envelopeB := OneOf(
// 			Step(key, envelopeA),
// 			Step(key, AllOf(
// 				Step(key, &types.AccessListTx{
// 					Nonce: 1,
// 				}),
// 				Step(key, &types.AccessListTx{
// 					Nonce: 2,
// 				}),
// 			)),
//    )
//
// The hope for this library is to provide means for the readable generation of
// bundles in unit tests.

// Step creates a transaction to be included in a bundle, signed by the given
// key. It is a building block to be used as an argument in the builder or in
// utility functions.
func Step(key *ecdsa.PrivateKey, tx any) BundleStep {
	switch tx := tx.(type) {
	case types.TxData:
		return BundleStep{key: key, tx: tx}
	case types.AccessListTx:
		return BundleStep{key: key, tx: &tx}
	case types.DynamicFeeTx:
		return BundleStep{key: key, tx: &tx}
	case *types.Transaction:
		return Step(key, &types.AccessListTx{
			ChainID:    tx.ChainId(),
			Nonce:      tx.Nonce(),
			GasPrice:   tx.GasPrice(),
			Gas:        tx.Gas(),
			To:         tx.To(),
			Value:      tx.Value(),
			Data:       tx.Data(),
			AccessList: tx.AccessList(),
		})
	default:
		panic("unsupported TxData type")
	}
}

// BundleStep is a single transaction in a bundle to build.
type BundleStep struct {
	key *ecdsa.PrivateKey
	tx  types.TxData
}

// NewBuilder creates a new bundle builder to create a custom bundle.
func NewBuilder() *builder {
	return &builder{}
}

type builder struct {
	flags       *ExecutionFlag
	earliest    *uint64
	latest      *uint64
	steps       []BundleStep
	envelopeKey *ecdsa.PrivateKey
}

func (b *builder) WithFlags(flags ExecutionFlag) *builder {
	b.flags = &flags
	return b
}

func (b *builder) Earliest(earliest uint64) *builder {
	b.earliest = &earliest
	return b
}

func (b *builder) Latest(latest uint64) *builder {
	b.latest = &latest
	return b
}

func (b *builder) With(steps ...BundleStep) *builder {
	b.steps = append(b.steps, steps...)
	return b
}

func (b *builder) WithEnvelopKey(key *ecdsa.PrivateKey) *builder {
	b.envelopeKey = key
	return b
}

func (b *builder) BuildBundleAndPlan() (*TransactionBundle, ExecutionPlan) {

	// Set up defaults for meta flags.
	flags := EF_AllOf
	if b.flags != nil {
		flags = *b.flags
	}
	earliest := uint64(0)
	latest := uint64(MaxBlockRange - 1)
	if b.earliest != nil {
		earliest = *b.earliest
		latest = earliest + MaxBlockRange - 1
	}
	if b.latest != nil {
		latest = *b.latest
	}

	// Add the costs for the additional marker to the gas limit.
	markerCosts := params.TxAccessListAddressGas + params.TxAccessListStorageKeyGas
	for _, step := range b.steps {
		// Fix the gas limit for nested envelops to be accurate.
		tx := types.NewTx(step.tx)
		newGasLimit := tx.Gas() + markerCosts

		// For nested envelopes, the gas price needs to be accurately adjusted
		// to pass the bundle validation test.
		if IsEnvelope(tx) {
			innerBundle, _, err := ValidateTransactionBundle(tx)
			if err == nil {
				marker := types.AccessTuple{
					Address:     BundleOnly,
					StorageKeys: []common.Hash{{1, 2, 3}}, // < value not relevant
				}
				newGasLimit = getGasLimitForEnvelope(
					innerBundle, tx.Data(), []types.AccessTuple{marker},
				)
			}
		}

		switch data := step.tx.(type) {
		case *types.DynamicFeeTx:
			data.Gas = newGasLimit
		case *types.AccessListTx:
			data.Gas = newGasLimit
		}

	}

	// Get chain ID from transactions, if any.
	var chainId *big.Int
	for _, step := range b.steps {
		tx := types.NewTx(step.tx)
		if curId := tx.ChainId(); curId != nil && curId.Sign() > 0 {
			chainId = curId
			break
		}
	}

	if chainId == nil {
		chainId = big.NewInt(1)
	}

	// Create an Execution Plan for the bundle.
	signer := types.LatestSignerForChainID(chainId)

	plan := ExecutionPlan{
		Steps:    make([]ExecutionStep, len(b.steps)),
		Flags:    flags,
		Earliest: earliest,
		Latest:   latest,
	}
	for i, step := range b.steps {
		plan.Steps[i] = ExecutionStep{
			From: crypto.PubkeyToAddress(step.key.PublicKey),
			Hash: signer.Hash(types.NewTx(step.tx)),
		}
	}

	// Get hash of execution plan and annotate transactions with it.
	execPlanHash := plan.Hash()
	marker := types.AccessTuple{
		Address:     BundleOnly,
		StorageKeys: []common.Hash{execPlanHash},
	}
	for _, step := range b.steps {
		switch data := step.tx.(type) {
		case *types.DynamicFeeTx:
			data.AccessList = append(data.AccessList, marker)
		case *types.AccessListTx:
			data.AccessList = append(data.AccessList, marker)
		}
	}

	// Sign the modified TxData instances.
	txs := make([]*types.Transaction, len(b.steps))
	for i, step := range b.steps {
		txs[i] = types.MustSignNewTx(step.key, signer, step.tx)
	}

	return &TransactionBundle{
		Transactions: txs,
		Flags:        flags,
		Earliest:     earliest,
		Latest:       latest,
	}, plan
}

func (b *builder) BuildEnvelopeBundleAndPlan() (
	*types.Transaction,
	*TransactionBundle,
	ExecutionPlan,
) {
	// Build the bundle and wrap it in an envelope.
	key := b.envelopeKey
	if key == nil {
		newKey, err := crypto.GenerateKey()
		if err != nil {
			panic(fmt.Sprintf("failed to generate new key: %v", err))
		}
		key = newKey
	}
	bundle, plan := b.BuildBundleAndPlan()
	return newEnvelope(key, bundle), bundle, plan
}

func (b *builder) BuildEnvelopeAndPlan() (*types.Transaction, ExecutionPlan) {
	envelop, _, plan := b.BuildEnvelopeBundleAndPlan()
	return envelop, plan
}

func (b *builder) BuildBundle() *TransactionBundle {
	bundle, _ := b.BuildBundleAndPlan()
	return bundle
}

func (b *builder) Build() *types.Transaction {
	envelope, _ := b.BuildEnvelopeAndPlan()
	return envelope
}

// --- Utility Wrappers ---

func AllOf(steps ...BundleStep) *types.Transaction {
	return NewBuilder().WithFlags(EF_AllOf).With(steps...).Build()
}

func OneOf(steps ...BundleStep) *types.Transaction {
	return NewBuilder().WithFlags(EF_OneOf).With(steps...).Build()
}

func MustWrapIntoEnvelope(bundle *TransactionBundle) *types.Transaction {
	key, err := crypto.GenerateKey()
	if err != nil {
		panic(fmt.Sprintf("failed to generate new key: %v", err))
	}
	return newEnvelope(key, bundle)
}

// --- implementation details ---

// Wraps the given bundle into an envelope transaction.
func newEnvelope(
	key *ecdsa.PrivateKey,
	bundle *TransactionBundle,
) *types.Transaction {

	payload := bundle.Encode()
	gasLimit := getGasLimitForEnvelope(bundle, payload, nil)

	chainId := big.NewInt(1)
	if len(bundle.Transactions) > 0 {
		chainId = bundle.Transactions[0].ChainId()
	}

	signer := types.LatestSignerForChainID(chainId)
	return types.MustSignNewTx(key, signer, &types.AccessListTx{
		ChainID: chainId,
		To:      &BundleProcessor,
		Data:    payload,
		Gas:     gasLimit,
	})
}

func getGasLimitForEnvelope(
	bundle *TransactionBundle,
	payload []byte,
	accessList []types.AccessTuple,
) uint64 {

	intrinsic, err := core.IntrinsicGas(
		payload,
		accessList,
		nil,   // code auth is not used in the bundle transaction
		false, // bundle transaction is not a contract creation
		true,  // is homestead
		true,  // is istanbul
		true,  // is shanghai
	)
	if err != nil {
		panic(err)
	}

	floorDataGas, err := core.FloorDataGas(payload)
	if err != nil {
		panic(err)
	}

	txGasSum := uint64(0)
	for _, tx := range bundle.Transactions {
		txGasSum += tx.Gas()
	}

	return max(intrinsic, floorDataGas, txGasSum)
}
