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
	"maps"
	"math/big"

	"github.com/0xsoniclabs/sonic/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

// This file offers utilities to build bundles from transaction data. The most
// generic format is the NewBuilder function, enabling the creation of an
// envelope transaction carrying a bundle as follows:
//
//   envelope := NewBuilder().
// 		SetEarliest(12).
// 		SetLatest(15).
// 		AllOf(
// 			Step(key, &types.AccessListTx{
// 				Nonce: 1,
// 			}),
// 			Step(key, &types.AccessListTx{
// 				Nonce: 2,
// 			}),
// 		).
//  	Build()
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
//    ).Build()
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
//    ).Build()
//
// The hope for this library is to provide means for the readable generation of
// bundles in unit tests.

// Step creates a transaction to be included in a bundle, signed by the given
// key. It is a building block to be used as an argument in the builder or in
// utility functions.
func Step(key *ecdsa.PrivateKey, tx any) BuilderStep {
	switch tx := tx.(type) {
	case types.TxData:
		return BuilderStep{txRef: &txReference{key: key, tx: tx}}
	case types.AccessListTx:
		return BuilderStep{txRef: &txReference{key: key, tx: &tx}}
	case types.DynamicFeeTx:
		return BuilderStep{txRef: &txReference{key: key, tx: &tx}}
	case types.BlobTx:
		return BuilderStep{txRef: &txReference{key: key, tx: &tx}}
	case types.SetCodeTx:
		return BuilderStep{txRef: &txReference{key: key, tx: &tx}}
	case *types.Transaction:
		txData := utils.GetTxData(tx)
		// Legacy transactions are promoted to AccessListTx in the builder,
		// to enable marking nested transactions with the bundle-only marker.
		if data, ok := txData.(*types.LegacyTx); ok {
			txData = &types.AccessListTx{
				Nonce:    data.Nonce,
				GasPrice: data.GasPrice,
				Gas:      data.Gas,
				To:       data.To,
				Value:    data.Value,
				Data:     data.Data,
			}
		}
		return Step(key, txData)
	default:
		panic("unsupported TxData type")
	}
}

func AllOf(steps ...BuilderStep) BuilderStep {
	return Group(false, steps...)
}

func OneOf(steps ...BuilderStep) BuilderStep {
	return Group(true, steps...)
}

func Group(oneOf bool, steps ...BuilderStep) BuilderStep {
	return BuilderStep{
		oneOf: oneOf,
		steps: steps,
	}
}

// NewBuilder creates a new bundle builder to create a custom bundle.
func NewBuilder() *builder {
	return &builder{}
}

type builder struct {
	signer           types.Signer
	earliest         *uint64
	latest           *uint64
	root             BuilderStep
	envelopeKey      *ecdsa.PrivateKey
	envelopeNonce    uint64
	envelopeGasPrice *big.Int
}

func (b *builder) SetEarliest(earliest uint64) *builder {
	b.earliest = &earliest
	return b
}

func (b *builder) SetLatest(latest uint64) *builder {
	b.latest = &latest
	return b
}

func (b *builder) WithSigner(signer types.Signer) *builder {
	b.signer = signer
	return b
}

func (b *builder) With(root BuilderStep) *builder {
	b.root = root
	return b
}

func (b *builder) AllOf(steps ...BuilderStep) *builder {
	return b.With(AllOf(steps...))
}

func (b *builder) OneOf(steps ...BuilderStep) *builder {
	return b.With(OneOf(steps...))
}

func (b *builder) SetEnvelopeSenderKey(key *ecdsa.PrivateKey) *builder {
	b.envelopeKey = key
	return b
}

func (b *builder) SetEnvelopeNonce(nonce uint64) *builder {
	b.envelopeNonce = nonce
	return b
}

// SetEnvelopeGasPrice sets the gas price for the envelope transaction.
// An envelope with gas price is still a valid envelope. This function is
// added to be able to generate test cases.
func (b *builder) SetEnvelopeGasPrice(gasPrice *big.Int) *builder {
	b.envelopeGasPrice = gasPrice
	return b
}
func (b *builder) BuildBundleAndPlan() (*TransactionBundle, ExecutionPlan) {

	// Set up defaults for meta flags.
	earliest := uint64(0)
	latest := uint64(MaxBlockRange - 1)
	if b.earliest != nil {
		earliest = *b.earliest
		latest = earliest + MaxBlockRange - 1
	}
	if b.latest != nil {
		latest = *b.latest
	}

	if b.signer == nil {
		b.signer = types.LatestSignerForChainID(big.NewInt(1))
	}

	// Collect all transactions from the steps, to be included in the bundle.
	transactions := b.root.collectTransactions(b.signer)

	// Add the costs for the additional marker to the gas limit.
	markerCosts := params.TxAccessListAddressGas + params.TxAccessListStorageKeyGas
	for _, step := range transactions {
		// Fix the gas limit for nested envelops to be accurate.
		tx := types.NewTx(step.tx)
		newGasLimit := tx.Gas() + markerCosts

		// For nested envelopes, the gas limit needs to be accurately adjusted
		// to pass the bundle validation test.
		if IsEnvelope(tx) {
			innerBundle, _, err := ValidateEnvelope(b.signer, tx)
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
		case *types.BlobTx:
			data.Gas = newGasLimit
		case *types.SetCodeTx:
			data.Gas = newGasLimit
		default:
			panic("unsupported TxData type for gas adjustment")
		}
	}

	// Update transaction references in hierarchy to match updated transactions.
	root := b.root.updateTxReferences(b.signer, transactions)
	transactions = root.collectTransactions(b.signer)

	// Create an Execution Plan for the bundle.
	plan := ExecutionPlan{
		Root: root.toStep(b.signer),
		Range: BlockRange{
			Earliest: earliest,
			Latest:   latest,
		},
	}

	// Record the transaction hashes before adding the marker.
	txReferences := make(map[common.Hash]TxReference)
	for hash, step := range transactions {
		txRef := TxReference{
			From: crypto.PubkeyToAddress(step.key.PublicKey),
			Hash: b.signer.Hash(types.NewTx(step.tx)),
		}
		txReferences[hash] = txRef
	}

	// Get hash of execution plan and annotate transactions with it.
	execPlanHash := plan.Hash()
	marker := types.AccessTuple{
		Address:     BundleOnly,
		StorageKeys: []common.Hash{execPlanHash},
	}
	for _, step := range transactions {
		switch data := step.tx.(type) {
		case *types.DynamicFeeTx:
			data.AccessList = append(data.AccessList, marker)
		case *types.AccessListTx:
			data.AccessList = append(data.AccessList, marker)
		case *types.BlobTx:
			data.AccessList = append(data.AccessList, marker)
		case *types.SetCodeTx:
			data.AccessList = append(data.AccessList, marker)
		default:
			panic("unsupported TxData type for marker annotation")
		}
	}

	// Sign the modified TxData instances.
	txs := make(map[TxReference]*types.Transaction)
	for hash, step := range transactions {
		txRef := txReferences[hash]
		txs[txRef] = types.MustSignNewTx(step.key, b.signer, step.tx)
	}

	return &TransactionBundle{
		Transactions: txs,
		Plan:         plan,
	}, plan
}

// BuildEnvelopeBundleAndPlan returns an envelope transaction along its
// bundle and execution plan
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
	return newEnvelope(b.signer, key, b.envelopeNonce, b.envelopeGasPrice, bundle), bundle, plan
}

// BuildEnvelope returns an envelope transaction and its execution plan
func (b *builder) BuildEnvelopeAndPlan() (*types.Transaction, ExecutionPlan) {
	envelope, _, plan := b.BuildEnvelopeBundleAndPlan()
	return envelope, plan
}

// BuildBundle returns a transaction bundle without wrapping it in an envelope.
func (b *builder) BuildBundle() TransactionBundle {
	bundle, _ := b.BuildBundleAndPlan()
	return *bundle
}

// Build returns an envelope transaction
func (b *builder) Build() *types.Transaction {
	envelope, _ := b.BuildEnvelopeAndPlan()
	return envelope
}

// --- implementation details ---

// BuilderStep is a single transaction or a nested group in a bundle to build.
type BuilderStep struct {
	flags ExecutionFlags

	// -- single transaction field --
	txRef *txReference // < if nil, it is a group

	// -- fields for a group step --
	oneOf bool
	steps []BuilderStep
}

// WithFlags sets execution flags for this step. It can be used to mark steps as
// tolerating invalid or failed transaction results.
func (s BuilderStep) WithFlags(flags ExecutionFlags) BuilderStep {
	s.flags = flags
	return s
}

// Build is a utility function to directly build an envelope transaction from
// this step. It is a shortcut for
//
//	NewBuilder().With(step).Build()
//
// which can be convenient for simple bundles with a single step.
func (s BuilderStep) Build() *types.Transaction {
	return NewBuilder().With(s).Build()
}

// collectTransactions recursively collects all transactions reachable from this
// step, including nested steps.
func (s *BuilderStep) collectTransactions(signer types.Signer) map[common.Hash]txReference {
	txs := make(map[common.Hash]txReference)
	if s.txRef != nil {
		txs[s.txRef.hash(signer)] = *s.txRef
	} else {
		for _, step := range s.steps {
			maps.Copy(txs, step.collectTransactions(signer))
		}
	}
	return txs
}

// updateTxReferences recursively updates the transaction references in this step
// according to the given map of updated transactions. This is used to propagate
// changes in the transactions (e.g. gas adjustments) through the hierarchy of
// steps while building bundles, so that the final execution plan correctly
// references the updated transactions.
func (s *BuilderStep) updateTxReferences(
	signer types.Signer,
	updatedTxs map[common.Hash]txReference,
) BuilderStep {
	if s.txRef != nil {
		if updatedRef, ok := updatedTxs[s.txRef.hash(signer)]; ok {
			return BuilderStep{txRef: &updatedRef}
		}
		return *s
	}
	updatedSteps := make([]BuilderStep, len(s.steps))
	for i, step := range s.steps {
		updatedSteps[i] = step.updateTxReferences(signer, updatedTxs)
	}
	return BuilderStep{
		oneOf: s.oneOf,
		flags: s.flags,
		steps: updatedSteps,
	}
}

// toStep converts this BuilderStep into an ExecutionStep, which is used in the
// execution plan. This recursive function is used by the builder to convert the
// hierarchy of BuilderSteps into the corresponding ExecutionStep hierarchy.
func (s *BuilderStep) toStep(
	signer types.Signer,
) ExecutionStep {
	var res ExecutionStep
	if s.txRef != nil {
		res = NewTxStep(TxReference{
			From: crypto.PubkeyToAddress(s.txRef.key.PublicKey),
			Hash: signer.Hash(types.NewTx(s.txRef.tx)),
		})
	} else {
		var subSteps []ExecutionStep
		for _, step := range s.steps {
			subSteps = append(subSteps, step.toStep(signer))
		}
		if s.oneOf {
			res = NewOneOfStep(subSteps...)
		} else {
			res = NewAllOfStep(subSteps...)
		}
	}
	return res.WithFlags(s.flags)
}

// txReference is a helper struct to keep track of a transaction and its signing
// key during the building process, before the final transactions are signed.
type txReference struct {
	key *ecdsa.PrivateKey
	tx  types.TxData
}

// hash computes a unique hash for this transaction reference, to be used in
// maps during the building process.
func (r *txReference) hash(signer types.Signer) common.Hash {
	sender := crypto.PubkeyToAddress(r.key.PublicKey)
	txHash := signer.Hash(types.NewTx(r.tx))
	return crypto.Keccak256Hash(sender.Bytes(), txHash.Bytes())
}

// Wraps the given bundle into an envelope transaction.
func newEnvelope(
	signer types.Signer,
	key *ecdsa.PrivateKey,
	nonce uint64,
	gasPrice *big.Int,
	bundle *TransactionBundle,
) *types.Transaction {

	payload, err := bundle.encode()
	if err != nil {
		panic(fmt.Sprintf("failed to encode bundle: %v", err))
	}
	gasLimit := getGasLimitForEnvelope(bundle, payload, nil)

	return types.MustSignNewTx(key, signer, &types.AccessListTx{
		To:       &BundleProcessor,
		Nonce:    nonce,
		Data:     payload,
		Gas:      gasLimit,
		GasPrice: gasPrice,
	})
}

// getGasLimitForEnvelope calculates the gas limit for an envelope transaction
// based on the given payload and access list.
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
