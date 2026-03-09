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
	big "math/big"
	"slices"
	"testing"

	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func Test_GetBundleState_ReturnsPermanentlyBlockedForOutdatedBundle(t *testing.T) {
	ctrl := gomock.NewController(t)
	chainState := NewMockChainState(ctrl)

	currentBlock := uint64(100)
	chainState.EXPECT().GetCurrentBlockHeight().Return(currentBlock).AnyTimes()

	bundle := &TransactionBundle{
		Earliest: 0,
		Latest:   currentBlock - 1, // Bundle is outdated
	}

	state := GetBundleState(bundle, chainState)
	require.Equal(t, BundleStatePermanentlyBlocked, state)
}

func Test_GetBundleState_ReturnsTemporaryBlockedForFutureBundle(t *testing.T) {
	ctrl := gomock.NewController(t)
	chainState := NewMockChainState(ctrl)

	currentBlock := uint64(100)
	chainState.EXPECT().GetCurrentBlockHeight().Return(currentBlock).AnyTimes()
	bundle := &TransactionBundle{
		Earliest: currentBlock + 1, // Bundle is in the future
		Latest:   currentBlock + 10,
	}

	state := GetBundleState(bundle, chainState)
	require.Equal(t, BundleStateTemporaryBlocked, state)
}

func Test_GetBundleState_ReturnsRunnableForCurrentBundle(t *testing.T) {
	ctrl := gomock.NewController(t)
	chainState := NewMockChainState(ctrl)

	currentBlock := uint64(100)
	chainState.EXPECT().GetCurrentBlockHeight().Return(currentBlock).AnyTimes()
	chainState.EXPECT().GetChainID().Return(big.NewInt(1)).AnyTimes()
	chainState.EXPECT().GetStateDB().AnyTimes()

	bundle := &TransactionBundle{
		Earliest: currentBlock - 5, // Bundle is valid for current block
		Latest:   currentBlock + 5,
	}

	state := GetBundleState(bundle, chainState)
	require.Equal(t, BundleStateRunnable, state)
}

func Test_GetBundleState_ChecksForNonceConflicts(t *testing.T) {
	const initialNonce = 1
	tests := map[string]struct {
		bundle pattern
		result BundleState
	}{
		"bundle with no transactions": {
			bundle: allOf(), // < will always succeed
			result: BundleStateRunnable,
		},
		"bundle with one transaction with correct nonce": {
			bundle: allOf(1), // one tx with nonce 1
			result: BundleStateRunnable,
		},
		"bundle with future nonce": {
			bundle: allOf(2), // one tx with nonce 2, which is in the future
			result: BundleStateTemporaryBlocked,
		},
		"bundle with outdated nonce": {
			bundle: allOf(0), // one tx with nonce 0, which is outdated
			result: BundleStatePermanentlyBlocked,
		},
		"bundle with different senders": {
			bundle: allOf(0xA1, 0xB1), // two txs from different senders with correct nonces
			result: BundleStateRunnable,
		},
		"bundle with nonce gap": {
			bundle: allOf(1, 3), // two txs from the same sender with a nonce gap (nonce 2 is missing)
			result: BundleStatePermanentlyBlocked,
		},
	}

	keys, _ := createKeys(t)
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			state := state.NewMockStateDB(ctrl)
			state.EXPECT().GetNonce(gomock.Any()).Return(uint64(initialNonce)).AnyTimes()

			chainState := NewMockChainState(ctrl)
			chainState.EXPECT().GetCurrentBlockHeight().AnyTimes()
			chainState.EXPECT().GetChainID().Return(big.NewInt(1)).AnyTimes()
			chainState.EXPECT().GetStateDB().Return(state).AnyTimes()

			chainId := big.NewInt(1)
			signer := types.LatestSignerForChainID(chainId)

			envelop := test.bundle.toBundle(signer, keys)
			bundle, _, err := ValidateTransactionBundle(envelop, signer)
			require.NoError(t, err)

			got := GetBundleState(bundle, chainState)
			require.Equal(t, test.result, got)
		})
	}

}

func Test_checkForNonceConflicts_DetectsNonceUsage(t *testing.T) {

	const initialNonce = 1
	tests := map[string]struct {
		bundle pattern
		result BundleState
	}{
		"empty all-of bundle is runnable": {
			bundle: allOf(), // < will always succeed
			result: BundleStateRunnable,
		},
		"empty one-of bundle is permanently blocked": {
			bundle: oneOf(), // < can never succeed
			result: BundleStatePermanentlyBlocked,
		},
		"single all-of transaction with correct nonce": {
			bundle: allOf(1), // one tx with nonce 1
			result: BundleStateRunnable,
		},
		"single one-of transaction with correct nonce": {
			bundle: oneOf(1),
			result: BundleStateRunnable,
		},
		"single all-of transaction with old nonce": {
			bundle: allOf(0),
			result: BundleStatePermanentlyBlocked,
		},
		"single one-of transaction with old nonce": {
			bundle: oneOf(0),
			result: BundleStatePermanentlyBlocked,
		},
		"single all-of transaction with future nonce": {
			bundle: allOf(2),
			result: BundleStateTemporaryBlocked,
		},
		"single one-of transaction with future nonce": {
			bundle: oneOf(2),
			result: BundleStateTemporaryBlocked,
		},
		"multiple all-of transactions with correct nonce order": {
			bundle: allOf(1, 2, 3), // three txs with nonces 1, 2, 3
			result: BundleStateRunnable,
		},
		"multiple one-of transactions with correct nonce order": {
			bundle: oneOf(1, 2, 3),
			result: BundleStateRunnable,
		},
		"multiple all-of transactions out of order": {
			bundle: allOf(2, 1, 3),
			result: BundleStatePermanentlyBlocked,
		},
		"multiple one-of transactions out of order": {
			bundle: oneOf(2, 1, 3),
			result: BundleStateRunnable,
		},
		"multiple all-of with old nonce": {
			bundle: allOf(0, 1, 2),
			result: BundleStatePermanentlyBlocked,
		},
		"multiple one-of with old nonce": {
			bundle: oneOf(0, 1, 2),
			result: BundleStateRunnable,
		},
		"all-of with nonce gap": {
			bundle: allOf(1, 3),
			result: BundleStatePermanentlyBlocked,
		},
		"one-of with nonce gap": {
			bundle: oneOf(1, 3),
			result: BundleStateRunnable,
		},
		"all-of with nonce gap in the future": {
			bundle: allOf(2, 4),
			result: BundleStatePermanentlyBlocked,
		},
		"one-of with nonce gap in the future": {
			bundle: oneOf(2, 4),
			result: BundleStateTemporaryBlocked,
		},
		"nested all-of with consecutive nonces": {
			bundle: allOf(1, allOf(2, 3), 4),
			result: BundleStateRunnable,
		},
		"nested all-of with future nonces": {
			bundle: allOf(2, allOf(3, 4), 5),
			result: BundleStateTemporaryBlocked,
		},
		"nested all-of with nonce gap": {
			bundle: allOf(1, allOf(3, 4), 5),
			result: BundleStatePermanentlyBlocked,
		},
		"nested one-of in all-of": {
			bundle: allOf(1, oneOf(2, 3), 3),
			result: BundleStateRunnable,
		},
		"multiple transactions from different senders with correct nonces": {
			// two txs from sender A with nonces 1 and 2, one tx from sender B with nonce 1
			bundle: allOf(0xA1, 0xB1, 0xA2),
			result: BundleStateRunnable,
		},
		"multiple transactions from different senders with nonce gap for one sender": {
			bundle: allOf(0xA1, 0xB1, 0xA3),
			result: BundleStatePermanentlyBlocked,
		},
		"all-of outdated nonce for one sender but not the other": {
			bundle: allOf(0xA0, 0xB1),
			result: BundleStatePermanentlyBlocked,
		},
		"one-of outdated nonce for one sender but not the other": {
			bundle: oneOf(0xA0, 0xB1),
			result: BundleStateRunnable,
		},
	}

	keys, senders := createKeys(t)
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			chainId := big.NewInt(1)
			signer := types.LatestSignerForChainID(chainId)

			source := NewMockNonceSource(ctrl)
			for _, sender := range senders {
				source.EXPECT().GetNonce(sender).Return(uint64(initialNonce)).MaxTimes(2)
			}

			envelop := test.bundle.toBundle(signer, keys)
			bundle, _, err := ValidateTransactionBundle(envelop, signer)
			require.NoError(t, err)

			got := checkForNonceConflicts(bundle, signer, source)
			require.Equal(t, test.result, got)
		})
	}
}

func Test_checkForNonceConflicts_ReturnsPermanentlyBlockedIfLowestReferencedNoncesCannotBeDerived(t *testing.T) {
	invalidTx := types.NewTx(&types.LegacyTx{})
	bundle := &TransactionBundle{
		Bundle: []*types.Transaction{invalidTx},
	}
	signer := types.LatestSignerForChainID(big.NewInt(1))
	_, err := getLowestReferencedNonces(bundle, signer)
	require.Error(t, err)

	got := checkForNonceConflicts(bundle, signer, nil)
	require.Equal(t, BundleStatePermanentlyBlocked, got)
}

func Test_getLowestReferencedNonces_ReturnsLowestNoncesInBundle(t *testing.T) {

	tests := map[string]struct {
		bundle   pattern
		expected map[int]uint64
	}{
		"empty bundle": {
			bundle:   allOf(),
			expected: map[int]uint64{},
		},
		"single transaction": {
			bundle:   allOf(0xA1),
			expected: map[int]uint64{0xA: 1},
		},
		"multiple transactions from same sender": {
			bundle:   allOf(0xA2, 0xA1, 0xA3),
			expected: map[int]uint64{0xA: 1},
		},
		"multiple transactions from different senders": {
			bundle:   allOf(0xA2, 0xB3, 0xA1, 0xB4),
			expected: map[int]uint64{0xA: 1, 0xB: 3},
		},
		"nested bundles": {
			bundle:   allOf(0xA2, oneOf(0xB3, 0xB4), allOf(0xA1, 0xA3)),
			expected: map[int]uint64{0xA: 1, 0xB: 3},
		},
	}

	keys, senders := createKeys(t)
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			chainId := big.NewInt(1)
			signer := types.LatestSignerForChainID(chainId)

			envelop := test.bundle.toBundle(signer, keys)
			bundle, _, err := ValidateTransactionBundle(envelop, signer)
			require.NoError(t, err)

			lowest, err := getLowestReferencedNonces(bundle, signer)
			require.NoError(t, err)

			got := make(map[int]uint64)
			for addr, nonce := range lowest {
				got[slices.Index(senders, addr)] = nonce
			}
			require.Equal(t, test.expected, got)
		})
	}
}

func Test_getLowestReferencedNonces_ReturnsIfSenderCannotBeDerived(t *testing.T) {
	signer := types.LatestSignerForChainID(big.NewInt(1))
	bundle := TransactionBundle{
		// Add a transaction with a missing signature.
		Bundle: []*types.Transaction{types.NewTx(&types.LegacyTx{})},
	}
	_, err := getLowestReferencedNonces(&bundle, signer)
	require.ErrorContains(t, err, "failed to derive sender")
}

func Test_getLowestReferencedNonces_DetectsInvalidNestedBundle(t *testing.T) {
	require := require.New(t)
	invalidBundle := types.NewTx(&types.LegacyTx{To: &BundleAddress})
	require.True(IsTransactionBundle(invalidBundle))

	signer := types.LatestSignerForChainID(big.NewInt(1))
	_, _, err := ValidateTransactionBundle(invalidBundle, signer)
	require.Error(err)

	bundle := TransactionBundle{
		Bundle: []*types.Transaction{invalidBundle},
	}
	_, err = getLowestReferencedNonces(&bundle, signer)
	require.ErrorContains(err, "invalid nested bundle")
}

func Test_runner_Run_ReturnsErrorForInvalidNestedBundle(t *testing.T) {
	require := require.New(t)
	invalidBundle := types.NewTx(&types.LegacyTx{To: &BundleAddress})
	require.True(IsTransactionBundle(invalidBundle))

	runner := &_runner{
		signer:         types.LatestSignerForChainID(big.NewInt(1)),
		acceptedSender: make(map[common.Address]struct{}),
	}

	result := runner.Run(invalidBundle)
	require.Equal(TransactionResultInvalid, result)
}

func Test_runner_Run_ReturnsInvalidForTransactionsWithoutSignature(t *testing.T) {
	tx := types.NewTx(&types.LegacyTx{})
	runner := &_runner{
		signer:         types.LatestSignerForChainID(big.NewInt(1)),
		acceptedSender: make(map[common.Address]struct{}),
	}

	result := runner.Run(tx)
	require.Equal(t, TransactionResultInvalid, result)
}

// --- Utility functions to build test bundles ---

func allOf(nested ...any) pattern {
	return pattern{
		flags:  AllOf,
		nested: nested,
	}
}

func oneOf(nested ...any) pattern {
	return pattern{
		flags:  OneOf,
		nested: nested,
	}
}

type pattern struct {
	flags  ExecutionFlag
	nested []any
}

func (p pattern) toBundle(
	signer types.Signer,
	keys []*ecdsa.PrivateKey,
) *types.Transaction {
	return types.MustSignNewTx(keys[0], signer, p._toTxData(signer, keys))
}

func (p pattern) _toTxData(
	signer types.Signer,
	keys []*ecdsa.PrivateKey,
) *types.AccessListTx {
	// convert nested into transactions
	txs := []*types.AccessListTx{}
	senders := []common.Address{}
	keysToSign := []*ecdsa.PrivateKey{}
	for _, element := range p.nested {
		switch v := element.(type) {
		case int:
			txs = append(txs, &types.AccessListTx{
				Nonce: uint64(0xF & v),
				Gas:   21_096,
			})
			key := keys[0xF&(v>>4)]
			keysToSign = append(keysToSign, key)
			senders = append(senders, crypto.PubkeyToAddress(key.PublicKey))
		case pattern:
			txs = append(txs, v._toTxData(signer, keys))
			key := keys[0] // fore envelope transaction, any key is fine
			keysToSign = append(keysToSign, key)
			senders = append(senders, crypto.PubkeyToAddress(key.PublicKey))
		default:
			panic("unsupported element type")
		}
	}

	// create the execution plan for this bundle
	plan := ExecutionPlan{
		Flags: p.flags,
	}
	for i, tx := range txs {
		plan.Steps = append(plan.Steps, ExecutionStep{
			From: senders[i],
			Hash: signer.Hash(types.NewTx(tx)),
		})
	}

	// Attach the execution plan hash to all transactions.
	execPlanHash := plan.Hash()
	for _, tx := range txs {
		tx.AccessList = append(tx.AccessList, types.AccessTuple{
			Address: BundleOnly,
			StorageKeys: []common.Hash{
				execPlanHash,
			},
		})
	}

	// Turn transaction data into signed transactions.
	signedTxs := []*types.Transaction{}
	for i, tx := range txs {
		signedTxs = append(signedTxs, types.MustSignNewTx(keysToSign[i], signer, tx))
	}

	data := Encode(TransactionBundle{
		Version: BundleV1,
		Bundle:  signedTxs,
		Flags:   p.flags,
	})

	// Compute the gas limit for the envelop transaction.
	gasLimit := uint64(0)
	for _, tx := range signedTxs {
		gasLimit += tx.Gas()
	}

	intrGas, err := core.IntrinsicGas(
		data,
		nil,   // no access list
		nil,   // no set-code authorization
		false, // is contract creation
		true,  // is homestead
		true,  // is istanbul
		true,  // is shanghai
	)
	if err != nil {
		panic(err)
	}

	// Wrap up bundle into an envelope transaction.
	return &types.AccessListTx{
		To:   &BundleAddress,
		Data: data,
		Gas:  max(gasLimit, intrGas),
	}
}

func createKeys(t *testing.T) ([]*ecdsa.PrivateKey, []common.Address) {
	t.Helper()
	keys := make([]*ecdsa.PrivateKey, 16)
	for i := range keys {
		key, err := crypto.GenerateKey()
		require.NoError(t, err)
		keys[i] = key
	}
	senders := make([]common.Address, len(keys))
	for i, key := range keys {
		senders[i] = crypto.PubkeyToAddress(key.PublicKey)
	}
	return keys, senders
}
