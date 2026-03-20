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

package evmcore

import (
	"crypto/ecdsa"
	"math/big"
	"slices"
	"testing"

	"github.com/0xsoniclabs/sonic/evmcore/core_types"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func Test_GetBundleState_ReturnsNonExecutableForInvalidBundle(t *testing.T) {
	ctrl := gomock.NewController(t)
	chainState := NewMockChainState(ctrl)
	chainState.EXPECT().GetCurrentNetworkRules().Return(opera.Rules{
		NetworkID: 1,
	}).AnyTimes()

	invalidBundle := types.NewTx(&types.LegacyTx{To: &bundle.BundleProcessor})
	_, _, err := bundle.ValidateTransactionBundle(invalidBundle)
	require.Error(t, err)

	state := GetBundleState(chainState, invalidBundle)
	require.Equal(t, false, state.Executable)
	require.Contains(t, state.Reasons[0], "invalid bundle")
}

func Test_GetBundleState_ReturnsNonExecutableForOutdatedBundle(t *testing.T) {
	ctrl := gomock.NewController(t)
	chainState := NewMockChainState(ctrl)

	currentBlock := uint64(100)
	currentHeader := &EvmHeader{
		Number: big.NewInt(int64(currentBlock)),
	}
	chainState.EXPECT().GetLatestHeader().Return(currentHeader).AnyTimes()
	chainState.EXPECT().GetCurrentNetworkRules().Return(opera.Rules{
		NetworkID: 1,
	}).AnyTimes()

	// Build an outdated bundle.
	envelop := bundle.NewBuilder().Latest(currentBlock - 1).Build()

	_, _, err := bundle.ValidateTransactionBundle(envelop)
	require.NoError(t, err)

	state := GetBundleState(chainState, envelop)
	require.Equal(t, false, state.Executable)
	require.Contains(t, state.Reasons[0], ErrBundleLatestPassed.Error())
}

func Test_GetBundleState_ReturnsTemporaryBlockedForFutureBundle(t *testing.T) {
	ctrl := gomock.NewController(t)
	chainState := NewMockChainState(ctrl)

	currentBlock := uint64(100)
	currentHeader := &EvmHeader{
		Number: big.NewInt(int64(currentBlock)),
	}
	chainState.EXPECT().GetLatestHeader().Return(currentHeader).AnyTimes()
	chainState.EXPECT().GetCurrentNetworkRules().Return(opera.Rules{
		NetworkID: 1,
	}).AnyTimes()

	// Build an bundle with a block window in the future
	envelop := bundle.NewBuilder().
		Earliest(currentBlock + 1).
		Latest(currentBlock + 10).
		Build()

	_, _, err := bundle.ValidateTransactionBundle(envelop)
	require.NoError(t, err)

	state := GetBundleState(chainState, envelop)
	require.True(t, state.Executable)
	require.True(t, state.TemporarilyBlocked)
}

func Test_GetBundleState_ReturnsNonExecutable_ForFailedTrialRun(t *testing.T) {

	ctrl := gomock.NewController(t)
	chainState := NewMockChainState(ctrl)
	stateDb := state.NewMockStateDB(ctrl)
	stateDb.EXPECT().InterTxSnapshot().Return(12)
	stateDb.EXPECT().RevertToInterTxSnapshot(12)

	currentBlock := uint64(100)
	currentHeader := &EvmHeader{
		Number: big.NewInt(int64(currentBlock)),
	}
	chainState.EXPECT().GetLatestHeader().Return(currentHeader).AnyTimes()
	chainState.EXPECT().GetCurrentNetworkRules().Return(opera.Rules{
		NetworkID: 1,
	}).AnyTimes()
	chainState.EXPECT().StateDB().Return(stateDb).AnyTimes()

	envelop := bundle.NewBuilder().
		Earliest(currentBlock - 5).
		Latest(currentBlock + 5).
		Build()

	rejectEverything := func(*types.Transaction, ChainState, state.StateDB) bool {
		return false
	}

	state := getBundleState(chainState, envelop, rejectEverything)
	require.False(t, state.Executable)
	require.Contains(t, state.Reasons[0], "trial-run failed")
}

func Test_GetBundleState_ReturnsRunnableForCurrentBundle(t *testing.T) {
	ctrl := gomock.NewController(t)
	chainState := NewMockChainState(ctrl)
	stateDb := state.NewMockStateDB(ctrl)
	stateDb.EXPECT().InterTxSnapshot().Return(12)
	stateDb.EXPECT().RevertToInterTxSnapshot(12)

	currentBlock := uint64(100)
	currentHeader := &EvmHeader{
		Number: big.NewInt(int64(currentBlock)),
	}
	chainState.EXPECT().GetLatestHeader().Return(currentHeader).AnyTimes()
	chainState.EXPECT().GetCurrentNetworkRules().Return(opera.Rules{
		NetworkID: 1,
	}).AnyTimes()
	chainState.EXPECT().StateDB().Return(stateDb).AnyTimes()

	// Build a bundle with a valid block window.
	envelop := bundle.NewBuilder().
		Earliest(currentBlock - 5).
		Latest(currentBlock + 5).
		Build()

	acceptEverything := func(*types.Transaction, ChainState, state.StateDB) bool {
		return true
	}

	state := getBundleState(chainState, envelop, acceptEverything)
	require.True(t, state.Executable)
	require.False(t, state.TemporarilyBlocked)
}

func Test_GetBundleState_ChecksForNonceConflicts(t *testing.T) {

	executableBundleState := BundleState{Executable: true}
	temporarilyBlockedBundleState := BundleState{Executable: true, TemporarilyBlocked: true}
	nonExecutableBundleState := BundleState{
		Executable: false,
		Reasons:    []string{"nonce conflict check failed", "bundle nonce check execution failed"}}

	const initialNonce = 1
	tests := map[string]struct {
		bundle pattern
		result BundleState
	}{
		"bundle with no transactions": {
			bundle: allOf(), // < will always succeed
			result: executableBundleState,
		},
		"bundle with one transaction with correct nonce": {
			bundle: allOf(1), // one tx with nonce 1
			result: executableBundleState,
		},
		"bundle with future nonce": {
			bundle: allOf(2), // one tx with nonce 2, which is in the future
			result: temporarilyBlockedBundleState,
		},
		"bundle with outdated nonce": {
			bundle: allOf(0), // one tx with nonce 0, which is outdated
			result: nonExecutableBundleState,
		},
		"bundle with different senders": {
			bundle: allOf(0xA1, 0xB1), // two txs from different senders with correct nonces
			result: executableBundleState,
		},
		"bundle with nonce gap": {
			bundle: allOf(1, 3), // two txs from the same sender with a nonce gap (nonce 2 is missing)
			result: nonExecutableBundleState,
		},
	}

	keys, _ := createKeys(t)
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			db := state.NewMockStateDB(ctrl)
			db.EXPECT().GetNonce(gomock.Any()).Return(uint64(initialNonce)).AnyTimes()
			db.EXPECT().InterTxSnapshot().AnyTimes()
			db.EXPECT().RevertToInterTxSnapshot(gomock.Any()).AnyTimes()

			currentHeader := &EvmHeader{
				Number: big.NewInt(0),
			}
			chainState := NewMockChainState(ctrl)
			chainState.EXPECT().GetLatestHeader().Return(currentHeader).AnyTimes()
			chainState.EXPECT().GetCurrentNetworkRules().Return(opera.Rules{
				NetworkID: 1,
			}).AnyTimes()
			chainState.EXPECT().StateDB().Return(db).AnyTimes()

			chainId := big.NewInt(1)
			signer := types.LatestSignerForChainID(chainId)

			envelop := test.bundle.toBundle(signer, keys)
			_, _, err := bundle.ValidateTransactionBundle(envelop)
			require.NoError(t, err)

			acceptEverything := func(*types.Transaction, ChainState, state.StateDB) bool {
				return true
			}

			got := getBundleState(chainState, envelop, acceptEverything)
			require.Equal(t, test.result, got)
		})
	}
}

func Test_checkForNonceConflicts_DetectsNonceUsage(t *testing.T) {

	executableBundleState := BundleState{Executable: true}
	temporarilyBlockedBundleState := BundleState{Executable: true, TemporarilyBlocked: true}
	nonExecutableBundleState := BundleState{Executable: false}

	const initialNonce = 1
	tests := map[string]struct {
		bundle pattern
		result BundleState
	}{
		"empty all-of bundle is runnable": {
			bundle: allOf(), // < will always succeed
			result: executableBundleState,
		},
		"empty one-of bundle is non-executable": {
			bundle: oneOf(), // < can never succeed
			result: nonExecutableBundleState,
		},
		"single all-of transaction with correct nonce": {
			bundle: allOf(1), // one tx with nonce 1
			result: executableBundleState,
		},
		"single one-of transaction with correct nonce": {
			bundle: oneOf(1),
			result: executableBundleState,
		},
		"single all-of transaction with old nonce": {
			bundle: allOf(0),
			result: nonExecutableBundleState,
		},
		"single one-of transaction with old nonce": {
			bundle: oneOf(0),
			result: nonExecutableBundleState,
		},
		"single all-of transaction with future nonce": {
			bundle: allOf(2),
			result: temporarilyBlockedBundleState,
		},
		"single one-of transaction with future nonce": {
			bundle: oneOf(2),
			result: temporarilyBlockedBundleState,
		},
		"multiple all-of transactions with correct nonce order": {
			bundle: allOf(1, 2, 3), // three txs with nonces 1, 2, 3
			result: executableBundleState,
		},
		"multiple one-of transactions with correct nonce order": {
			bundle: oneOf(1, 2, 3),
			result: executableBundleState,
		},
		"multiple all-of transactions out of order": {
			bundle: allOf(2, 1, 3),
			result: nonExecutableBundleState,
		},
		"multiple one-of transactions out of order": {
			bundle: oneOf(2, 1, 3),
			result: executableBundleState,
		},
		"multiple all-of with old nonce": {
			bundle: allOf(0, 1, 2),
			result: nonExecutableBundleState,
		},
		"multiple one-of with old nonce": {
			bundle: oneOf(0, 1, 2),
			result: executableBundleState,
		},
		"all-of with nonce gap": {
			bundle: allOf(1, 3),
			result: nonExecutableBundleState,
		},
		"one-of with nonce gap": {
			bundle: oneOf(1, 3),
			result: executableBundleState,
		},
		"all-of with nonce gap in the future": {
			bundle: allOf(2, 4),
			result: nonExecutableBundleState,
		},
		"one-of with nonce gap in the future": {
			bundle: oneOf(2, 4),
			result: temporarilyBlockedBundleState,
		},
		"nested all-of with consecutive nonces": {
			bundle: allOf(1, allOf(2, 3), 4),
			result: executableBundleState,
		},
		"nested all-of with future nonces": {
			bundle: allOf(2, allOf(3, 4), 5),
			result: temporarilyBlockedBundleState,
		},
		"nested all-of with nonce gap": {
			bundle: allOf(1, allOf(3, 4), 5),
			result: nonExecutableBundleState,
		},
		"nested one-of in all-of": {
			bundle: allOf(1, oneOf(2, 3), 3),
			result: executableBundleState,
		},
		"multiple transactions from different senders with correct nonces": {
			// two txs from sender A with nonces 1 and 2, one tx from sender B with nonce 1
			bundle: allOf(0xA1, 0xB1, 0xA2),
			result: executableBundleState,
		},
		"multiple transactions from different senders with nonce gap for one sender": {
			bundle: allOf(0xA1, 0xB1, 0xA3),
			result: nonExecutableBundleState,
		},
		"all-of outdated nonce for one sender but not the other": {
			bundle: allOf(0xA0, 0xB1),
			result: nonExecutableBundleState,
		},
		"one-of outdated nonce for one sender but not the other": {
			bundle: oneOf(0xA0, 0xB1),
			result: executableBundleState,
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
			bundle, _, err := bundle.ValidateTransactionBundle(envelop)
			require.NoError(t, err)

			got := checkForNonceConflicts(bundle, signer, source)
			require.Equal(t, test.result, got)
		})
	}
}

func Test_checkForNonceConflicts_ReturnsNonExecutable_WhenLowestReferencedNoncesCannotBeDerived(t *testing.T) {
	invalidTx := types.NewTx(&types.LegacyTx{})
	bundle := &bundle.TransactionBundle{
		Transactions: []*types.Transaction{invalidTx},
	}
	signer := types.LatestSignerForChainID(big.NewInt(1))
	_, err := getLowestReferencedNonces(bundle, signer)
	require.Error(t, err)

	got := checkForNonceConflicts(bundle, signer, nil)
	require.False(t, got.Executable)
	require.Contains(t, got.Reasons[0], "failed to derive sender")
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
			bundle, _, err := bundle.ValidateTransactionBundle(envelop)
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
	bundle := bundle.TransactionBundle{
		// Add a transaction with a missing signature.
		Transactions: []*types.Transaction{types.NewTx(&types.LegacyTx{})},
	}
	_, err := getLowestReferencedNonces(&bundle, signer)
	require.ErrorContains(t, err, "failed to derive sender")
}

func Test_getLowestReferencedNonces_DetectsInvalidNestedBundle(t *testing.T) {
	require := require.New(t)
	invalidBundle := types.NewTx(&types.LegacyTx{To: &bundle.BundleProcessor})
	require.True(bundle.IsEnvelope(invalidBundle))

	signer := types.LatestSignerForChainID(big.NewInt(1))
	_, _, err := bundle.ValidateTransactionBundle(invalidBundle)
	require.Error(err)

	bundle := bundle.TransactionBundle{
		Transactions: []*types.Transaction{invalidBundle},
	}
	_, err = getLowestReferencedNonces(&bundle, signer)
	require.ErrorContains(err, "invalid nested bundle")
}

func Test_runner_Run_ReturnsErrorForInvalidNestedBundle(t *testing.T) {
	require := require.New(t)
	invalidBundle := types.NewTx(&types.LegacyTx{To: &bundle.BundleProcessor})
	require.True(bundle.IsEnvelope(invalidBundle))

	runner := &dryRunner{
		signer:         types.LatestSignerForChainID(big.NewInt(1)),
		acceptedSender: make(map[common.Address]struct{}),
	}

	result := runner.Run(invalidBundle)
	require.Equal(core_types.TransactionResultInvalid, result)
}

func Test_runner_Run_ReturnsInvalidForTransactionsWithoutSignature(t *testing.T) {
	tx := types.NewTx(&types.LegacyTx{})
	runner := &dryRunner{
		signer:         types.LatestSignerForChainID(big.NewInt(1)),
		acceptedSender: make(map[common.Address]struct{}),
	}

	result := runner.Run(tx)
	require.Equal(t, core_types.TransactionResultInvalid, result)
}

// --- Utility functions to build test bundles ---

func allOf(nested ...any) pattern {
	return pattern{
		flags:  bundle.EF_AllOf,
		nested: nested,
	}
}

func oneOf(nested ...any) pattern {
	return pattern{
		flags:  bundle.EF_OneOf,
		nested: nested,
	}
}

type pattern struct {
	flags  bundle.ExecutionFlag
	nested []any
}

func (p pattern) toBundle(
	signer types.Signer,
	keys []*ecdsa.PrivateKey,
) *types.Transaction {
	// convert elements into steps
	steps := make([]bundle.BundleStep, 0, len(p.nested))
	for _, element := range p.nested {
		switch v := element.(type) {
		case int:
			steps = append(steps, bundle.Step(
				keys[0xF&(v>>4)],
				&types.AccessListTx{
					Nonce: uint64(0xF & v),
					Gas:   21_240,
				},
			))
		case pattern:
			steps = append(steps, bundle.Step(
				keys[0], // fore envelope transaction, any key is fine
				v.toBundle(signer, keys),
			))
		default:
			panic("unsupported element type")
		}
	}

	// Build the resulting bundle.
	return bundle.NewBuilder().WithFlags(p.flags).With(steps...).Build()
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
