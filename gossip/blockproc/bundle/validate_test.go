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
	"slices"
	"strings"
	"testing"

	"github.com/0xsoniclabs/sonic/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

var testChainID = big.NewInt(1)

func TestValidateEnvelope_ValidBundles_AreAccepted(t *testing.T) {
	signer := types.LatestSignerForChainID(testChainID)
	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	tests := map[string]*types.Transaction{
		"empty AllOf bundle":     AllOf().Build(),
		"empty OneOf bundle":     OneOf().Build(),
		"non-empty AllOf bundle": AllOf(Step(key, &types.AccessListTx{})).Build(),
		"non-empty OneOf bundle": OneOf(Step(key, &types.AccessListTx{})).Build(),
	}

	for name, tx := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			bundle, plan, err := ValidateEnvelope(signer, tx)
			require.NoError(err)
			require.NotNil(bundle, "expected a bundle transaction")
			require.NotNil(plan, "expected an execution plan")

			wantedBundle, err := OpenEnvelope(signer, tx)
			require.NoError(err)

			// types.Transactions can not be compared reliably using
			// reflect.DeepEqual, therefore we compare the fields of the bundle
			// separately.
			require.Equal(wantedBundle.Plan, bundle.Plan)
			require.Equal(len(wantedBundle.Transactions), len(bundle.Transactions))
			for i, txRef := range wantedBundle.Transactions {
				require.Equal(txRef.Hash(), bundle.Transactions[i].Hash())
			}

			require.Equal(wantedBundle.Plan, *plan)
		})
	}
}

func TestValidateEnvelope_RegularTransaction_RejectedAsNotBeingAnEnvelope(t *testing.T) {
	signer := types.LatestSignerForChainID(testChainID)
	regularTx := types.NewTx(&types.LegacyTx{
		To:   &common.Address{0x42},
		Data: []byte("this is a regular transaction, not an envelope"),
	})

	bundle, plan, err := ValidateEnvelope(signer, regularTx)
	require.ErrorContains(t, err, "not an envelope transaction")
	require.Nil(t, bundle, "expected no bundle to be returned")
	require.Nil(t, plan, "expected no execution plan to be returned")
}

func TestValidateEnvelope_InvalidEncoding_ReturnsError(t *testing.T) {
	signer := types.LatestSignerForChainID(testChainID)
	envelope := types.NewTx(&types.LegacyTx{
		To:   &BundleProcessor,
		Data: []byte("this is not a valid bundle encoding"),
	})

	_, _, err := ValidateEnvelope(signer, envelope)
	require.ErrorContains(t, err, "failed to decode transaction bundle")
}

func TestValidateEnvelope_DetectsErrorInIntrinsicGasCalculation(t *testing.T) {
	signer := types.LatestSignerForChainID(testChainID)

	bundle := NewBuilder().AllOf().BuildBundle()
	encoded, err := bundle.Encode()
	require.NoError(t, err)

	envelope := types.NewTx(&types.LegacyTx{
		To:   &BundleProcessor,
		Data: encoded,
	})

	injectedError := fmt.Errorf("injected error for test")
	_, _, err = validateEnvelopeInternal(
		signer,
		envelope,
		func(data []byte, accessList types.AccessList) (uint64, error) {
			return 0, injectedError
		},
		nil,
	)

	require.ErrorIs(t, err, injectedError)
}

func TestValidateEnvelope_DetectsErrorInFloorDataGasCalculation(t *testing.T) {
	signer := types.LatestSignerForChainID(testChainID)

	bundle := NewBuilder().AllOf().BuildBundle()
	encoded, err := bundle.Encode()
	require.NoError(t, err)

	envelope := types.NewTx(&types.LegacyTx{
		To:   &BundleProcessor,
		Data: encoded,
	})

	injectedError := fmt.Errorf("injected error for test")
	_, _, err = validateEnvelopeInternal(
		signer,
		envelope,
		func(data []byte, accessList types.AccessList) (uint64, error) {
			return 0, nil
		},
		func(data []byte) (uint64, error) {
			return 0, injectedError
		},
	)

	require.ErrorIs(t, err, injectedError)
}

func TestValidateEnvelope_ReturnsErrorsOnValidationFailure(t *testing.T) {
	generator := newTestBundleGenerator(t, 2)

	tests := map[string]struct {
		tx            *types.Transaction
		expectedError string
	}{
		"unsound bundle": {
			tx:            generator.makeUnsoundBundleTx(t),
			expectedError: "does not belong to the execution plan",
		},
		"wrongly signed bundle": {
			tx:            generator.makeBundleTxWithWronglySignedTx(t),
			expectedError: "invalid chain id for signer: have 0 want 1",
		},
		"bundle without enough gas for intrinsic cost": {
			tx:            generator.makeBundleTxWithoutEnoughIntrinsicGas(),
			expectedError: "gas should be more than intrinsic gas",
		},
		"bundle without enough gas for floor gas costs": {
			tx:            generator.makeBundleTxWithoutEnoughFloorGas(t),
			expectedError: "should be more than floor gas",
		},
		"bundle with wrong amount of gas for all transactions": {
			tx:            generator.makeBundleTxWithoutEnoughGasForAllTransactions(),
			expectedError: "gas limit of envelope does not match gas limit of payload",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, _, err := ValidateEnvelope(generator.signer, test.tx)
			require.ErrorContains(t, err, test.expectedError)
		})
	}
}

func TestValidateEnvelope_AcceptsValidBlockRanges(t *testing.T) {
	signer := types.LatestSignerForChainID(testChainID)

	tests := map[string]struct {
		From uint64
		To   uint64
		Gas  uint64
	}{
		"single-block range": {
			From: 10, To: 10, Gas: 22240,
		},
		"multi-block range": {
			From: 7, To: 42, Gas: 22240,
		},
		"max-size block range": {
			From: 100, To: 100 + MaxBlockRange - 1, Gas: 22320,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			bundle := TransactionBundle{
				Plan: ExecutionPlan{
					Root: NewTxStep(TxReference{}),
					Range: BlockRange{
						Earliest: test.From,
						Latest:   test.To,
					}},
			}
			encoded, err := bundle.Encode()
			require.NoError(t, err)
			tx := types.NewTx(&types.LegacyTx{
				To:   &BundleProcessor,
				Data: encoded,
				Gas:  test.Gas,
			})
			require.True(t, IsEnvelope(tx))

			_, _, err = ValidateEnvelope(signer, tx)
			require.NoError(t, err)
		})
	}
}

func TestValidateEnvelope_IdentifiesInvalidBlockRanges(t *testing.T) {
	signer := types.LatestSignerForChainID(testChainID)

	tests := map[string]struct {
		From  uint64
		To    uint64
		Issue string
	}{
		"empty block range": {
			From: 10, To: 5,
			Issue: "invalid empty block range [10,5]",
		},
		"too large block range": {
			From: 7, To: 7 + MaxBlockRange,
			Issue: "invalid block range, duration 1025, limit 1024",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			bundle := TransactionBundle{
				Plan: ExecutionPlan{
					Root: NewTxStep(TxReference{}),
					Range: BlockRange{
						Earliest: test.From,
						Latest:   test.To,
					}},
			}
			encoded, err := bundle.Encode()
			require.NoError(t, err)
			tx := types.NewTx(&types.LegacyTx{
				To:   &BundleProcessor,
				Data: encoded,
			})
			require.True(t, IsEnvelope(tx))

			_, _, err = ValidateEnvelope(signer, tx)
			require.ErrorContains(t, err, test.Issue)
		})
	}
}

// testBundleGenerator allows to generate different types of bundle transactions
// for testing purposes.
// These include valid and invalid bundles, as well as non-bundle transactions.
type testBundleGenerator struct {
	keys   []*ecdsa.PrivateKey
	n      int
	signer types.Signer
}

// newTestBundleGenerator creates a new testBundleGenerator with n keys and a signer.
// the valid bundles generated by this generator will contain n transactions signed by the generated keys.
func newTestBundleGenerator(t testing.TB, n int) testBundleGenerator {
	t.Helper()
	keys := make([]*ecdsa.PrivateKey, n)
	for i := range keys {
		key, err := crypto.GenerateKey()
		require.NoError(t, err)
		keys[i] = key
	}

	return testBundleGenerator{
		keys:   keys,
		n:      n,
		signer: types.LatestSignerForChainID(testChainID),
	}
}

func (gen testBundleGenerator) makeValidBundleTx() *types.Transaction {
	receiver := common.Address{0x42}
	gasPerTx := uint64(20_000)

	steps := make([]BuilderStep, 0, gen.n)
	for i := range gen.n {
		steps = append(steps, Step(gen.keys[i], &types.AccessListTx{
			Nonce: uint64(1),
			To:    &receiver,
			Value: big.NewInt(1234),
			Gas:   gasPerTx,
		}))
	}

	return AllOf(steps...).Build()
}

func (gen testBundleGenerator) makeUnsoundBundleTx(t testing.TB) *types.Transaction {
	t.Helper()
	receiver := common.Address{0x42}

	// Generate n metaTransactions from n different senders
	// execution plan hash is not correct, therefore the bundle is unsound
	invalidExecutionPlanHash := common.Hash{0x99}
	signedTransactions := make(map[TxReference]*types.Transaction, gen.n)
	for i := range gen.n {
		tx := types.AccessListTx{
			Nonce: uint64(1),
			To:    &receiver,
			Value: big.NewInt(1234),
			AccessList: []types.AccessTuple{
				{
					Address: BundleOnly,
					StorageKeys: []common.Hash{
						invalidExecutionPlanHash,
					},
				},
			},
		}

		signedTx, err := types.SignTx(types.NewTx(&tx), gen.signer, gen.keys[i])
		require.NoError(t, err)

		txRef := TxReference{From: crypto.PubkeyToAddress(gen.keys[i].PublicKey)}
		signedTransactions[txRef] = signedTx
	}

	// prepare the bundle
	bundle := TransactionBundle{
		Transactions: signedTransactions,
		Plan: ExecutionPlan{
			Root: NewTxStep(TxReference{}),
		},
	}

	data, err := bundle.Encode()
	require.NoError(t, err)
	floorGas, err := core.FloorDataGas(data)
	require.NoError(t, err)

	return types.NewTx(&types.LegacyTx{
		To:   &BundleProcessor,
		Data: data,
		Gas:  floorGas,
	})

}

func (gen testBundleGenerator) makeBundleTxWithWronglySignedTx(t testing.TB) *types.Transaction {
	t.Helper()
	receiver := common.Address{0x42}

	ExecutionPlanHash := common.Hash{0x99}
	tx := types.AccessListTx{
		Nonce: uint64(1),
		To:    &receiver,
		Value: big.NewInt(1234),
		AccessList: []types.AccessTuple{
			{
				Address: BundleOnly,
				StorageKeys: []common.Hash{
					ExecutionPlanHash,
				},
			},
		},
	}

	// Transaction is not really signed, therefore the sender cannot be derived
	unsignedTransaction := types.NewTx(&tx)

	// prepare the bundle
	bundle := TransactionBundle{
		Transactions: map[TxReference]*types.Transaction{
			{}: unsignedTransaction,
		},
		Plan: ExecutionPlan{
			Root: NewTxStep(TxReference{}),
		},
	}

	encoded, err := bundle.Encode()
	require.NoError(t, err)

	return types.NewTx(&types.LegacyTx{
		To:   &BundleProcessor,
		Data: encoded,
		Gas:  21096,
	})
}

func (gen testBundleGenerator) makeBundleTxWithoutEnoughIntrinsicGas() *types.Transaction {
	tx := gen.makeValidBundleTx()
	// reduce the gas in tx
	tx = types.NewTx(&types.LegacyTx{
		To:   &BundleProcessor,
		Data: tx.Data(),
		Gas:  10_000, // not enough gas for the bundle
	})
	return tx
}

func (gen testBundleGenerator) makeBundleTxWithoutEnoughFloorGas(t testing.TB) *types.Transaction {
	t.Helper()
	bundle := TransactionBundle{
		Transactions: map[TxReference]*types.Transaction{
			{}: types.MustSignNewTx(gen.keys[0], gen.signer, &types.AccessListTx{
				Data: make([]byte, 1<<10), // < high data usage
			}),
		},
		Plan: ExecutionPlan{
			Root: NewTxStep(TxReference{}),
		},
	}

	data, err := bundle.Encode()
	require.NoError(t, err)
	floorGas, err := core.FloorDataGas(data)
	require.NoError(t, err)

	// reduce the gas in tx
	return types.NewTx(&types.LegacyTx{
		To:   &BundleProcessor,
		Data: data,
		Gas:  floorGas - 1,
	})
}

func (gen testBundleGenerator) makeBundleTxWithoutEnoughGasForAllTransactions() *types.Transaction {
	tx := gen.makeValidBundleTx()
	// reduce the gas in tx
	return types.NewTx(&types.LegacyTx{
		To:   &BundleProcessor,
		Data: tx.Data(),
		Gas:  38_000, // not enough gas for all transactions in the bundle
	})
}

func TestValidateBundle_ValidBundles_AreAccepted(t *testing.T) {
	key1, err := crypto.GenerateKey()
	require.NoError(t, err)

	key2, err := crypto.GenerateKey()
	require.NoError(t, err)

	validBundles := []*builder{
		NewBuilder().AllOf(),
		NewBuilder().OneOf(),
		NewBuilder().AllOf(
			Step(key1, &types.AccessListTx{}),
			Step(key2, &types.AccessListTx{}),
		),
		NewBuilder().OneOf(
			AllOf(
				Step(key1, &types.AccessListTx{}),
				Step(key2, &types.AccessListTx{}),
			),
			AllOf(
				Step(key2, &types.AccessListTx{}),
				Step(key1, &types.AccessListTx{}),
			),
		),
	}

	signer := types.LatestSignerForChainID(big.NewInt(1))
	for _, builder := range validBundles {
		bundle := builder.WithSigner(signer).BuildBundle()
		require.NoError(t, validateBundle(signer, bundle))
	}
}

func TestValidateBundle_InvalidPlan_Rejected(t *testing.T) {
	bundle := TransactionBundle{}

	issue := validatePlan(bundle.Plan)
	require.Error(t, issue)

	got := validateBundle(nil, bundle)
	require.ErrorContains(t, got, "invalid execution plan")
	require.ErrorContains(t, got, issue.Error())
}

func TestValidateBundle_NilTransaction_Rejected(t *testing.T) {
	tests := map[string][]*types.Transaction{
		"single nil transaction": {nil},
		"nil and non-nil transactions": {
			types.NewTx(&types.AccessListTx{}),
			nil,
			types.NewTx(&types.AccessListTx{}),
		},
	}

	for name, transactions := range tests {
		t.Run(name, func(t *testing.T) {
			validPlan := ExecutionPlan{
				Range: BlockRange{Earliest: 10, Latest: 20},
				Root:  NewTxStep(TxReference{}),
			}
			require.NoError(t, validatePlan(validPlan))

			index := map[TxReference]*types.Transaction{}
			for i, tx := range transactions {
				index[TxReference{From: common.Address{byte(i + 1)}}] = tx
			}

			bundle := TransactionBundle{
				Plan:         validPlan,
				Transactions: index,
			}

			require.ErrorContains(t, validateBundle(nil, bundle),
				"invalid nil transaction in bundle",
			)
		})
	}
}

func TestValidateBundle_InconsistentChainIds_Rejected(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	signer12 := types.LatestSignerForChainID(big.NewInt(12))
	signer14 := types.LatestSignerForChainID(big.NewInt(14))
	signer16 := types.LatestSignerForChainID(big.NewInt(16))

	tests := map[string][]*types.Transaction{
		"two different chain IDs": {
			types.MustSignNewTx(key, signer12, &types.AccessListTx{Nonce: 1}),
			types.MustSignNewTx(key, signer14, &types.AccessListTx{Nonce: 2}),
		},
		"multiple different chain IDs": {
			types.MustSignNewTx(key, signer12, &types.AccessListTx{Nonce: 1}),
			types.MustSignNewTx(key, signer14, &types.AccessListTx{Nonce: 2}),
			types.MustSignNewTx(key, signer16, &types.AccessListTx{Nonce: 3}),
		},
		"first different": {
			types.MustSignNewTx(key, signer14, &types.AccessListTx{Nonce: 1}),
			types.MustSignNewTx(key, signer12, &types.AccessListTx{Nonce: 2}),
			types.MustSignNewTx(key, signer12, &types.AccessListTx{Nonce: 3}),
		},
		"middle different": {
			types.MustSignNewTx(key, signer12, &types.AccessListTx{Nonce: 1}),
			types.MustSignNewTx(key, signer14, &types.AccessListTx{Nonce: 2}),
			types.MustSignNewTx(key, signer12, &types.AccessListTx{Nonce: 3}),
		},
		"last different": {
			types.MustSignNewTx(key, signer12, &types.AccessListTx{Nonce: 1}),
			types.MustSignNewTx(key, signer12, &types.AccessListTx{Nonce: 2}),
			types.MustSignNewTx(key, signer14, &types.AccessListTx{Nonce: 3}),
		},
	}

	for name, transactions := range tests {
		t.Run(name, func(t *testing.T) {
			validPlan := ExecutionPlan{
				Range: BlockRange{Earliest: 10, Latest: 20},
				Root:  NewTxStep(TxReference{}),
			}
			require.NoError(t, validatePlan(validPlan))

			signer := signer12
			sender := crypto.PubkeyToAddress(key.PublicKey)

			index := map[TxReference]*types.Transaction{}
			for _, tx := range transactions {
				stripped, err := removeBundleOnlyMark(tx)
				require.NoError(t, err)
				hash := signer.Hash(stripped)

				ref := TxReference{
					From: sender,
					Hash: hash,
				}
				index[ref] = tx
			}

			bundle := TransactionBundle{
				Plan:         validPlan,
				Transactions: index,
			}

			require.ErrorContains(t, validateBundle(signer12, bundle),
				"invalid transaction in bundle: invalid chain id",
			)
		})
	}
}

func TestValidateBundle_MissingSigner_ProducesAnError(t *testing.T) {
	validPlan := ExecutionPlan{
		Range: BlockRange{Earliest: 10, Latest: 20},
		Root:  NewTxStep(TxReference{}),
	}

	bundle := TransactionBundle{
		Plan: validPlan,
	}

	require.ErrorContains(t, validateBundle(nil, bundle), "signer is nil")
}

func TestValidateBundle_InvalidIndex_Rejected(t *testing.T) {
	signer := types.LatestSignerForChainID(big.NewInt(123))
	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	bundle := NewBuilder().AllOf(
		Step(key, &types.AccessListTx{Nonce: 1}),
	).WithSigner(signer).BuildBundle()

	require.NoError(t, validateBundle(signer, bundle))

	ref := slices.Collect(maps.Keys(bundle.Transactions))[0]
	validTxData := utils.GetTxData(bundle.Transactions[ref]).(*types.AccessListTx)

	// using an unsigned transaction in the index is detected
	validTxData.R = nil
	unsignedTx := types.NewTx(validTxData)
	bundle.Transactions[ref] = unsignedTx
	require.ErrorContains(t, validateBundle(signer, bundle),
		"invalid transaction in bundle",
	)

	// Changing the signer of the transaction is detected
	otherKey, err := crypto.GenerateKey()
	require.NoError(t, err)
	otherTx := types.MustSignNewTx(otherKey, signer, validTxData)
	bundle.Transactions[ref] = otherTx

	require.ErrorContains(t, validateBundle(signer, bundle),
		"sender in transaction reference does not match actual sender",
	)

	// Change in the transaction data is detected
	validTxData.Value = big.NewInt(1234)
	changedDataTx := types.MustSignNewTx(key, signer, validTxData)
	bundle.Transactions[ref] = changedDataTx

	require.ErrorContains(t, validateBundle(signer, bundle),
		"content of transaction does not match transaction hash",
	)
}

func TestValidateBundle_UsageOfLegacyTransaction_Rejected(t *testing.T) {
	validPlan := ExecutionPlan{
		Range: BlockRange{Earliest: 10, Latest: 20},
		Root:  NewTxStep(TxReference{}),
	}
	require.NoError(t, validatePlan(validPlan))

	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	signer := types.LatestSignerForChainID(big.NewInt(123))
	tx := types.MustSignNewTx(key, signer, &types.LegacyTx{})

	ref := TxReference{
		From: crypto.PubkeyToAddress(key.PublicKey),
		Hash: signer.Hash(tx),
	}

	index := map[TxReference]*types.Transaction{
		ref: tx,
	}

	bundle := TransactionBundle{
		Plan:         validPlan,
		Transactions: index,
	}

	require.ErrorContains(t, validateBundle(signer, bundle),
		"invalid transaction in bundle: unsupported transaction type: 0",
	)
}

func TestValidateBundle_TransactionNotAgreeingToPlan_Rejected(t *testing.T) {
	signer := types.LatestSignerForChainID(big.NewInt(123))
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	bundle := NewBuilder().AllOf(
		Step(key, &types.AccessListTx{}),
	).WithSigner(signer).BuildBundle()

	require.NoError(t, validateBundle(signer, bundle))

	originalTx := bundle.GetTransactionsInReferencedOrder()[0]

	// removing the agreement to the execution plan is detected
	noAgreementData := utils.GetTxData(originalTx).(*types.AccessListTx)
	noAgreementData.AccessList = []types.AccessTuple{{Address: BundleOnly}}
	noAgreement := types.MustSignNewTx(key, signer, noAgreementData)
	require.True(t, IsBundleOnly(noAgreement))

	ref := slices.Collect(maps.Keys(bundle.Transactions))[0]
	bundle.Transactions[ref] = noAgreement

	require.ErrorContains(t, validateBundle(signer, bundle),
		"contains transaction not approving the execution plan",
	)

	// restore the valid bundle
	bundle.Transactions[ref] = originalTx
	require.NoError(t, validateBundle(signer, bundle))

	// replacing the agreement with another execution hash is also detected
	otherAgreementData := utils.GetTxData(originalTx).(*types.AccessListTx)
	otherAgreementData.AccessList = []types.AccessTuple{{
		Address:     BundleOnly,
		StorageKeys: []common.Hash{{0x99}},
	}}
	otherAgreement := types.MustSignNewTx(key, signer, otherAgreementData)
	require.True(t, IsBundleOnly(otherAgreement))

	bundle.Transactions[ref] = otherAgreement

	require.ErrorContains(t, validateBundle(signer, bundle),
		"contains transaction not approving the execution plan",
	)
}

func TestValidateBundle_MissingTransactionInIndex_Rejected(t *testing.T) {
	signer := types.LatestSignerForChainID(big.NewInt(123))
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	bundle := NewBuilder().AllOf(
		Step(key, &types.AccessListTx{Nonce: 1}),
		Step(key, &types.AccessListTx{Nonce: 2}),
	).WithSigner(signer).BuildBundle()

	require.NoError(t, validateBundle(signer, bundle))

	ref1 := slices.Collect(maps.Keys(bundle.Transactions))[0]
	delete(bundle.Transactions, ref1)
	require.ErrorContains(t, validateBundle(signer, bundle),
		"missing transaction referenced by the execution plan",
	)
}

func TestValidateBundle_AdditionalTransactionInIndex_Rejected(t *testing.T) {
	signer := types.LatestSignerForChainID(big.NewInt(123))
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	bundle := NewBuilder().AllOf(
		Step(key, &types.AccessListTx{Nonce: 1}),
		Step(key, &types.AccessListTx{Nonce: 2}),
	).WithSigner(signer).BuildBundle()

	require.NoError(t, validateBundle(signer, bundle))

	key2, err := crypto.GenerateKey()
	require.NoError(t, err)

	ref1 := slices.Collect(maps.Keys(bundle.Transactions))[0]
	validTx := bundle.Transactions[ref1]

	extraTx := types.MustSignNewTx(key2, signer, utils.GetTxData(validTx))
	refExtra := TxReference{
		From: crypto.PubkeyToAddress(key2.PublicKey),
		Hash: ref1.Hash,
	}
	bundle.Transactions[refExtra] = extraTx

	require.ErrorContains(t, validateBundle(signer, bundle),
		"contains transaction not referenced by the execution plan",
	)
}

func TestValidatePlan_AcceptsValidPlans(t *testing.T) {
	validPlans := []ExecutionPlan{
		{
			Root:  NewTxStep(TxReference{}),
			Range: BlockRange{Earliest: 10, Latest: 10},
		},
		{
			Root:  NewAllOfStep(NewTxStep(TxReference{}), NewTxStep(TxReference{})),
			Range: BlockRange{Earliest: 10, Latest: 20},
		},
		{
			Root:  NewOneOfStep(NewTxStep(TxReference{}), NewTxStep(TxReference{})),
			Range: BlockRange{Earliest: 0, Latest: MaxBlockRange - 1},
		},
	}

	for _, plan := range validPlans {
		require.NoError(t, validatePlan(plan))
	}
}

func TestValidatePlan_DetectsInvalidPlans(t *testing.T) {
	tests := map[string]struct {
		plan  ExecutionPlan
		issue string
	}{
		"empty plan": {
			plan:  ExecutionPlan{},
			issue: "invalid execution plan",
		},
		"invalid root step": {
			plan: ExecutionPlan{
				Root:  ExecutionStep{}, // invalid step
				Range: BlockRange{Earliest: 10, Latest: 20},
			},
			issue: "invalid execution plan",
		},
		"invalid block range": {
			plan: ExecutionPlan{
				Root:  NewTxStep(TxReference{}),
				Range: BlockRange{Earliest: 20, Latest: 10}, // invalid range
			},
			issue: "invalid block range",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require.ErrorContains(t, validatePlan(test.plan), test.issue)
		})
	}

	invalidPlanAndRange := ExecutionPlan{
		Root:  ExecutionStep{},                      // invalid step
		Range: BlockRange{Earliest: 20, Latest: 10}, // invalid range
	}

	require.Error(t, validateStep(invalidPlanAndRange.Root))
	require.Error(t, validateRange(invalidPlanAndRange.Range))
	require.Error(t, validatePlan(invalidPlanAndRange))
}

func TestValidateStep_AcceptsValidSteps(t *testing.T) {
	validSteps := []ExecutionStep{
		// -- atomic steps --
		NewTxStep(TxReference{}),
		NewTxStep(TxReference{}).WithFlags(EF_Default),
		NewTxStep(TxReference{}).WithFlags(EF_TolerateFailed),
		NewTxStep(TxReference{}).WithFlags(EF_TolerateInvalid),
		NewTxStep(TxReference{}).WithFlags(EF_TolerateFailed | EF_TolerateInvalid),

		// -- all-of steps --
		NewAllOfStep(),
		NewAllOfStep(
			NewTxStep(TxReference{}),
			NewTxStep(TxReference{}),
		),
		NewAllOfStep(
			NewTxStep(TxReference{}),
			NewTxStep(TxReference{}),
		).WithFlags(EF_TolerateFailed),

		// -- one-of steps --
		NewOneOfStep(),
		NewOneOfStep(
			NewTxStep(TxReference{}),
			NewTxStep(TxReference{}),
		),
		NewOneOfStep(
			NewTxStep(TxReference{}),
			NewTxStep(TxReference{}),
		).WithFlags(EF_TolerateFailed),

		// -- combined steps --
		NewOneOfStep(
			NewTxStep(TxReference{}),
			NewAllOfStep(
				NewTxStep(TxReference{}),
			),
		),
	}

	for _, step := range validSteps {
		require.NoError(t, validateStep(step))
	}
}

func TestValidateStep_DetectsInvalidSteps(t *testing.T) {
	tests := map[string]struct {
		step  ExecutionStep
		issue string
	}{
		"empty step": {
			step:  ExecutionStep{},
			issue: "malformed execution step",
		},
		"step with both single and group set": {
			step: ExecutionStep{
				single: &single{},
				group:  &group{},
			},
			issue: "malformed execution step",
		},
		"invalid execution flags": {
			step:  NewTxStep(TxReference{}).WithFlags(0xFF),
			issue: "invalid execution flags in step",
		},
		"malformed nested all-of step": {
			step: NewAllOfStep(
				ExecutionStep{}, // invalid step
			),
			issue: "malformed execution step",
		},
		"malformed nested one-of step": {
			step: NewOneOfStep(
				ExecutionStep{}, // invalid step
			),
			issue: "malformed execution step",
		},
		"invalid nested execution flags": {
			step: NewAllOfStep(
				NewTxStep(TxReference{}),
				NewOneOfStep(
					NewTxStep(TxReference{}),
					NewTxStep(TxReference{}).WithFlags(0xFF),
					NewTxStep(TxReference{}),
				),
				NewTxStep(TxReference{}),
			),
			issue: "invalid execution flags in step",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require.ErrorContains(t, validateStep(test.step), test.issue)
		})
	}
}

func TestValidateStep_DetectsExcessiveNesting(t *testing.T) {
	require.NoError(t, validateStep(NewAllOfStep(
		wrapInNested(NewTxStep(TxReference{}), MaxNestingDepth-1),
		wrapInNested(NewTxStep(TxReference{}), MaxNestingDepth-1),
		wrapInNested(NewTxStep(TxReference{}), MaxNestingDepth-1),
	)))

	require.ErrorContains(t, validateStep(NewAllOfStep(
		wrapInNested(NewTxStep(TxReference{}), MaxNestingDepth-1),
		wrapInNested(NewTxStep(TxReference{}), MaxNestingDepth),
		wrapInNested(NewTxStep(TxReference{}), MaxNestingDepth-1),
	)), "exceeds maximum nesting depth")

	for depth := range MaxNestingDepth + 2 {
		step := wrapInNested(NewTxStep(TxReference{}), depth)
		if depth <= MaxNestingDepth {
			require.NoError(t, validateStep(step))
		} else {
			require.ErrorContains(t, validateStep(step), "exceeds maximum nesting depth")
		}
	}
}

func TestValidateStep_MaximumNestingDepthMatchesConstant(t *testing.T) {
	inner := NewTxStep(TxReference{})
	allowed := wrapInNested(inner, MaxNestingDepth)
	invalid := wrapInNested(inner, MaxNestingDepth+1)

	// make sure wrapInNested produces the correct number of nested groups
	count := strings.Count(allowed.String(), "OneOf")
	require.Equal(t, MaxNestingDepth, count)

	count = strings.Count(invalid.String(), "OneOf")
	require.Equal(t, MaxNestingDepth+1, count)

	require.NoError(t, validateStep(allowed))
	require.ErrorContains(t, validateStep(invalid), "exceeds maximum nesting depth")

}

func wrapInNested(inner ExecutionStep, depth int) ExecutionStep {
	if depth == 0 {
		return inner
	}
	return NewOneOfStep(
		wrapInNested(inner, depth-1),
	)
}

func TestValidateRange_AcceptsValidRanges(t *testing.T) {
	tests := []BlockRange{
		{0, 0},
		{0, 100},
		{0, MaxBlockRange - 1},
		{10, 10},
		{10, 100},
		{10, MaxBlockRange + 9},
	}

	for _, tc := range tests {
		require.NoError(t, validateRange(tc))
	}
}

func TestValidateRange_DetectsInvalidRanges(t *testing.T) {
	tests := []struct {
		blockRange BlockRange
		issue      string
	}{
		{
			blockRange: BlockRange{1, 0},
			issue:      "empty block range",
		},
		{
			blockRange: BlockRange{10, 9},
			issue:      "empty block range",
		},
		{
			blockRange: BlockRange{MaxBlockRange, 0},
			issue:      "empty block range",
		},
		{
			blockRange: BlockRange{0, MaxBlockRange},
			issue:      "invalid block range",
		},
		{
			blockRange: BlockRange{10, MaxBlockRange + 10},
			issue:      "invalid block range",
		},
		{
			blockRange: BlockRange{10, MaxBlockRange + 100},
			issue:      "invalid block range",
		},
	}

	for _, tc := range tests {
		require.ErrorContains(t, validateRange(tc.blockRange), tc.issue)
	}
}

func TestValidateRange_ComprehensiveRangeChecks(t *testing.T) {
	for earliest := range 2 * MaxBlockRange {
		for latest := range 2 * MaxBlockRange {
			r := BlockRange{earliest, latest}
			if size := r.Size(); size > 0 && size <= MaxBlockRange {
				require.NoError(t, validateRange(r),
					"earliest=%d, latest=%d", earliest, latest,
				)
			} else {
				require.Error(t, validateRange(r),
					"earliest=%d, latest=%d", earliest, latest,
				)
			}
		}
	}
}

func TestBelongsToExecutionPlan_IdentifiesTransactionsWhichSignTheExecutionPlan(t *testing.T) {

	executionPlanHash := common.Hash{0x01, 0x02, 0x03}

	tests := map[string]struct {
		tx                types.TxData
		executionPlanHash common.Hash
		expected          bool
	}{
		"transaction without access list": {
			tx:       &types.LegacyTx{},
			expected: false,
		},
		"transaction with bundle-only but no plan hash": {
			tx: &types.AccessListTx{
				AccessList: types.AccessList{
					{
						Address: BundleOnly,
					},
				},
			},
			executionPlanHash: executionPlanHash,
			expected:          false,
		},
		"fragmented access list": {
			tx: &types.AccessListTx{
				AccessList: types.AccessList{
					{
						Address: BundleOnly,
					},
					{
						Address:     common.HexToAddress("0x0000000000000000000000000000000000000001"),
						StorageKeys: []common.Hash{executionPlanHash},
					},
				},
			},
			executionPlanHash: executionPlanHash,
			expected:          false,
		},
		"transaction with bundle-only and matching plan hash": {
			tx: &types.AccessListTx{
				AccessList: types.AccessList{
					{
						Address:     BundleOnly,
						StorageKeys: []common.Hash{executionPlanHash},
					},
				},
			},
			executionPlanHash: executionPlanHash,
			expected:          true,
		},
		"transaction with multiple accepted plans": {
			tx: &types.AccessListTx{
				AccessList: types.AccessList{
					{
						Address:     BundleOnly,
						StorageKeys: []common.Hash{{0x0A}},
					},
					{
						Address:     BundleOnly,
						StorageKeys: []common.Hash{executionPlanHash},
					},
					{
						Address:     BundleOnly,
						StorageKeys: []common.Hash{{0x0B}},
					},
				},
			},
			executionPlanHash: executionPlanHash,
			expected:          true,
		},
		"transaction with multiple accepted plans compact": {
			tx: &types.AccessListTx{
				AccessList: types.AccessList{
					{
						Address:     BundleOnly,
						StorageKeys: []common.Hash{{0x0A}, executionPlanHash, {0x0B}},
					},
				},
			},
			executionPlanHash: executionPlanHash,
			expected:          true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, test.expected,
				belongsToExecutionPlan(types.NewTx(test.tx), test.executionPlanHash))
		})
	}
}
