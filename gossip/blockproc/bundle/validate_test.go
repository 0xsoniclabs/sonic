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

func TestValidateEnvelope_IdentifiesBundles(t *testing.T) {

	generator := newTestBundleGenerator(t, 2)
	signer := types.LatestSignerForChainID(testChainID)

	tests := map[string]struct {
		tx           *types.Transaction
		expectBundle bool
	}{
		"not a bundle": {tx: generator.makeNonBundleTx(), expectBundle: false},
		"empty bundle": {tx: generator.makeEmptyBundleTx(), expectBundle: true},
		"valid bundle": {tx: generator.makeValidBundleTx(t), expectBundle: true},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			bundle, plan, err := ValidateEnvelope(signer, test.tx)
			require.NoError(t, err)
			if test.expectBundle {
				require.NotNil(t, bundle, "expected a bundle transaction")
				require.NotNil(t, plan, "expected an execution plan")
			} else {
				require.Nil(t, bundle, "expected no bundle transaction")
				require.Nil(t, plan, "expected no execution plan")
			}
		})
	}
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

func TestValidateEnvelope_DetectsOverFlowInIntrinsicGasCalculation(t *testing.T) {
	signer := types.LatestSignerForChainID(testChainID)

	bundle := TransactionBundle{}
	envelope := types.NewTx(&types.LegacyTx{
		To:   &BundleProcessor,
		Data: bundle.Encode(),
	})

	injectedError := fmt.Errorf("injected error for test")
	_, _, err := validateEnvelopeInternal(
		signer,
		envelope,
		func(data []byte, accessList types.AccessList) (uint64, error) {
			return 0, injectedError
		},
		nil,
	)

	require.ErrorIs(t, err, injectedError)
}

func TestValidateEnvelope_DetectsOverFlowInFloorDataGasCalculation(t *testing.T) {
	signer := types.LatestSignerForChainID(testChainID)

	bundle := TransactionBundle{}
	envelope := types.NewTx(&types.LegacyTx{
		To:   &BundleProcessor,
		Data: bundle.Encode(),
	})

	injectedError := fmt.Errorf("injected error for test")
	_, _, err := validateEnvelopeInternal(
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
	signer := types.LatestSignerForChainID(testChainID)
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
			expectedError: "failed to derive sender",
		},
		"bundle without enough gas for intrinsic cost": {
			tx:            generator.makeBundleTxWithoutEnoughIntrinsicGas(t),
			expectedError: "gas should be more than intrinsic gas",
		},
		"bundle without enough gas for floor gas costs": {
			tx:            generator.makeBundleTxWithoutEnoughFloorGas(t),
			expectedError: "should be more than floor gas",
		},
		"bundle with wrong amount of gas for all transactions": {
			tx:            generator.makeBundleTxWithoutEnoughGasForAllTransactions(t),
			expectedError: "gas limit of envelope does not match gas limit of payload",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, _, err := ValidateEnvelope(signer, test.tx)
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
			From: 10, To: 10, Gas: 21240,
		},
		"multi-block range": {
			From: 7, To: 42, Gas: 21240,
		},
		"max-size block range": {
			From: 100, To: 100 + MaxBlockRange - 1, Gas: 21320,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			bundle := TransactionBundle{
				Earliest: test.From,
				Latest:   test.To,
			}
			tx := types.NewTx(&types.LegacyTx{
				To:   &BundleProcessor,
				Data: bundle.Encode(),
				Gas:  test.Gas,
			})
			require.True(t, IsEnvelope(tx))

			_, _, err := ValidateEnvelope(signer, tx)
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
			Issue: "invalid empty block range [10,5] in execution plan",
		},
		"too large block range": {
			From: 7, To: 7 + MaxBlockRange,
			Issue: "invalid block range in execution plan, duration 1025, limit 1024",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			bundle := TransactionBundle{
				Earliest: test.From,
				Latest:   test.To,
			}
			tx := types.NewTx(&types.LegacyTx{
				To:   &BundleProcessor,
				Data: bundle.Encode(),
			})
			require.True(t, IsEnvelope(tx))

			_, _, err := ValidateEnvelope(signer, tx)
			require.ErrorContains(t, err, test.Issue)
		})
	}
}

// testBundleGenerator allows to generate different types of bundle transactions
// for testing purposes.
// These include valid and invalid bundles, as well as non-bundle transactions.
type testBundleGenerator struct {
	keys []*ecdsa.PrivateKey
	n    int
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
		keys: keys,
		n:    n,
	}
}

func (gen testBundleGenerator) makeEmptyBundleTx() *types.Transaction {
	bundle := TransactionBundle{
		Transactions: types.Transactions{},
		Flags:        EF_AllOf,
	}

	return types.NewTx(&types.LegacyTx{
		To:   &BundleProcessor,
		Data: bundle.Encode(),
		Gas:  21240,
	})
}

func (gen testBundleGenerator) makeValidBundleTx(t testing.TB) *types.Transaction {
	t.Helper()
	receiver := common.Address{0x42}
	gasPerTx := uint64(20_000)

	signer := types.LatestSignerForChainID(testChainID)

	//  Generate n metaTransactions from n different senders
	metaTransactions := make([]types.AccessListTx, gen.n)
	txHash := make([]common.Hash, gen.n)
	sender := make([]common.Address, gen.n)
	for i := range gen.n {
		tx := types.AccessListTx{
			Nonce: uint64(1),
			To:    &receiver,
			Value: big.NewInt(1234),
			Gas:   gasPerTx,
		}

		txHash[i] = signer.Hash(types.NewTx(&tx))
		metaTransactions[i] = tx
		sender[i] = crypto.PubkeyToAddress(gen.keys[i].PublicKey)
	}

	// prepare execution  plan
	plan := ExecutionPlan{
		Steps: make([]ExecutionStep, gen.n),
		Flags: 0,
	}
	for i := range gen.n {
		plan.Steps[i] = ExecutionStep{
			From: sender[i],
			Hash: txHash[i],
		}
	}

	// amend transactions with the execution plan hash
	// and sign them
	planHash := plan.Hash()
	signedTransactions := make(types.Transactions, gen.n)
	for i := range gen.n {
		tx := metaTransactions[i]
		tx.AccessList = append(tx.AccessList, types.AccessTuple{
			Address: BundleOnly,
			StorageKeys: []common.Hash{
				planHash,
			},
		})

		signedTx, err := types.SignTx(types.NewTx(&tx), signer, gen.keys[i])
		require.NoError(t, err)
		signedTransactions[i] = signedTx
	}

	// prepare the bundle
	bundle := TransactionBundle{
		Transactions: signedTransactions,
		Flags:        EF_AllOf,
	}

	return signTransactionWithUseOnceKey(&types.LegacyTx{
		To:   &BundleProcessor,
		Data: bundle.Encode(),
		Gas:  gasPerTx * uint64(gen.n),
	})
}

func (gen testBundleGenerator) makeUnsoundBundleTx(t testing.TB) *types.Transaction {
	t.Helper()
	receiver := common.Address{0x42}

	signer := types.LatestSignerForChainID(testChainID)

	// Generate n metaTransactions from n different senders
	// execution plan hash is not correct, therefore the bundle is unsound
	invalidExecutionPlanHash := common.Hash{0x99}
	signedTransactions := make(types.Transactions, gen.n)
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

		signedTx, err := types.SignTx(types.NewTx(&tx), signer, gen.keys[i])
		require.NoError(t, err)
		signedTransactions[i] = signedTx
	}

	// prepare the bundle
	bundle := TransactionBundle{
		Transactions: signedTransactions,
		Flags:        0,
	}

	data := bundle.Encode()
	floorGas, err := core.FloorDataGas(data)
	require.NoError(t, err)

	return signTransactionWithUseOnceKey(&types.LegacyTx{
		To:   &BundleProcessor,
		Data: bundle.Encode(),
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
		Transactions: []*types.Transaction{unsignedTransaction},
		Flags:        0,
	}

	return types.NewTx(&types.LegacyTx{
		To:   &BundleProcessor,
		Data: bundle.Encode(),
		Gas:  21096,
	})
}

func (gen testBundleGenerator) makeNonBundleTx() *types.Transaction {
	someAddress := common.Address{0x42}
	return signTransactionWithUseOnceKey(&types.LegacyTx{
		To:   &someAddress,
		Data: []byte("this is not a bundle"),
		Gas:  21096,
	})
}

func (gen testBundleGenerator) makeBundleTxWithoutEnoughIntrinsicGas(t testing.TB) *types.Transaction {
	tx := gen.makeValidBundleTx(t)
	// reduce the gas in tx
	tx = signTransactionWithUseOnceKey(&types.LegacyTx{
		To:   &BundleProcessor,
		Data: tx.Data(),
		Gas:  10_000, // not enough gas for the bundle
	})
	return tx
}

func (gen testBundleGenerator) makeBundleTxWithoutEnoughFloorGas(t testing.TB) *types.Transaction {
	signer := types.LatestSignerForChainID(testChainID)

	bundle := TransactionBundle{
		Transactions: types.Transactions{
			types.MustSignNewTx(gen.keys[0], signer, &types.AccessListTx{
				Data: make([]byte, 1<<10), // < high data usage
			}),
		},
	}

	data := bundle.Encode()
	floorGas, err := core.FloorDataGas(data)
	require.NoError(t, err)

	// reduce the gas in tx
	return signTransactionWithUseOnceKey(&types.LegacyTx{
		To:   &BundleProcessor,
		Data: data,
		Gas:  floorGas - 1,
	})
}

func (gen testBundleGenerator) makeBundleTxWithoutEnoughGasForAllTransactions(t testing.TB) *types.Transaction {
	tx := gen.makeValidBundleTx(t)
	// reduce the gas in tx
	tx = signTransactionWithUseOnceKey(&types.LegacyTx{
		To:   &BundleProcessor,
		Data: tx.Data(),
		Gas:  35_000, // not enough gas for all transactions in the bundle
	})
	return tx
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
			tx := signTransactionWithUseOnceKey(test.tx)
			require.Equal(t, test.expected,
				belongsToExecutionPlan(tx, test.executionPlanHash))
		})
	}

}

func signTransactionWithUseOnceKey(tx types.TxData) *types.Transaction {
	key, _ := crypto.GenerateKey()
	signer := types.LatestSignerForChainID(testChainID)
	return types.MustSignNewTx(key, signer, tx)
}
