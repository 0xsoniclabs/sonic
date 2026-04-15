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
	"testing"

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
