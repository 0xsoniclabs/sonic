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
	"crypto/rand"
	"fmt"
	"math"
	"math/big"
	"slices"
	"testing"

	"github.com/0xsoniclabs/sonic/evmcore/core_types"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func Test_GetBundleState_BundlesDisabled_ReturnsNonExecutable(t *testing.T) {
	ctrl := gomock.NewController(t)
	chainState := NewMockChainStateForBundleEval(ctrl)
	chainState.EXPECT().GetCurrentNetworkRules().Return(opera.Rules{
		NetworkID: 1,
		Upgrades:  opera.Upgrades{TransactionBundles: false},
	}).AnyTimes()

	invalidBundle := types.NewTx(&types.LegacyTx{To: &bundle.BundleProcessor})
	_, _, err := bundle.ValidateEnvelope(nil, invalidBundle)
	require.Error(t, err)

	state := GetBundleState(chainState, invalidBundle)
	require.Equal(t, state, makePermanentlyBlockedState("transaction bundles are not enabled on this network"))
}

func Test_GetBundleState_InvalidBundle_ReturnsNonExecutable(t *testing.T) {
	signer := types.LatestSignerForChainID(big.NewInt(123))
	ctrl := gomock.NewController(t)
	chainState := NewMockChainStateForBundleEval(ctrl)
	chainState.EXPECT().GetCurrentNetworkRules().Return(opera.Rules{
		NetworkID: 1,
		Upgrades:  opera.Upgrades{TransactionBundles: true},
	}).AnyTimes()

	invalidBundle := types.NewTx(&types.LegacyTx{To: &bundle.BundleProcessor})
	_, _, err := bundle.ValidateEnvelope(signer, invalidBundle)
	require.Error(t, err)

	state := GetBundleState(chainState, invalidBundle)
	require.Equal(t, state, makePermanentlyBlockedState(fmt.Sprintf("invalid bundle: %v", err)))
}

func Test_GetBundleState_OutdatedBundle_ReturnsNonExecutable(t *testing.T) {
	ctrl := gomock.NewController(t)
	chainState := NewMockChainStateForBundleEval(ctrl)

	currentBlock := uint64(100)
	currentHeader := &EvmHeader{
		Number: big.NewInt(int64(currentBlock)),
	}
	chainState.EXPECT().GetLatestHeader().Return(currentHeader).AnyTimes()
	chainState.EXPECT().GetCurrentNetworkRules().Return(opera.Rules{
		NetworkID: 1,
		Upgrades:  opera.Upgrades{TransactionBundles: true},
	}).AnyTimes()

	// Build an outdated bundle.
	signer := types.LatestSignerForChainID(big.NewInt(1))
	envelope := bundle.NewBuilder().SetLatest(currentBlock - 1).Build()

	_, _, err := bundle.ValidateEnvelope(signer, envelope)
	require.NoError(t, err)

	state := GetBundleState(chainState, envelope)
	require.Equal(t, state, makePermanentlyBlockedState("bundle has expired"))
}

func Test_GetBundleState_FutureBundle_ReturnsTemporaryBlocked(t *testing.T) {
	ctrl := gomock.NewController(t)
	chainState := NewMockChainStateForBundleEval(ctrl)

	currentBlock := uint64(100)
	currentHeader := &EvmHeader{
		Number: big.NewInt(int64(currentBlock)),
	}
	chainState.EXPECT().GetLatestHeader().Return(currentHeader).AnyTimes()
	chainState.EXPECT().GetCurrentNetworkRules().Return(opera.Rules{
		NetworkID: 1,
		Upgrades:  opera.Upgrades{TransactionBundles: true},
	}).AnyTimes()

	// Build a bundle with a block window in the future
	signer := types.LatestSignerForChainID(big.NewInt(1))
	envelop := bundle.NewBuilder().
		SetEarliest(currentBlock + 1).
		SetLatest(currentBlock + 10).
		Build()

	_, _, err := bundle.ValidateEnvelope(signer, envelop)
	require.NoError(t, err)

	state := GetBundleState(chainState, envelop)
	require.Equal(t, state, makeTemporaryBlockedState("bundle targets future blocks"))
}

func Test_GetBundleState_FailedTrialRun_ReturnsNonExecutable(t *testing.T) {

	ctrl := gomock.NewController(t)
	chainState := NewMockChainStateForBundleEval(ctrl)
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
		Upgrades:  opera.Upgrades{TransactionBundles: true},
	}).AnyTimes()
	chainState.EXPECT().StateDB().Return(stateDb)
	stateDb.EXPECT().Release()

	envelope := bundle.NewBuilder().
		SetEarliest(currentBlock - 5).
		SetLatest(currentBlock + 5).
		Build()

	rejectEverything := func(*types.Transaction, ChainStateForBundleEval, state.StateDB) bool {
		return false
	}

	state := getBundleState(chainState, envelope, rejectEverything)
	require.Equal(t, state, makePermanentlyBlockedState("bundle trial-run failed"))
}

func Test_GetBundleState_ValidBundle_ReturnsRunnable(t *testing.T) {
	ctrl := gomock.NewController(t)
	chainState := NewMockChainStateForBundleEval(ctrl)
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
		Upgrades:  opera.Upgrades{TransactionBundles: true},
	}).AnyTimes()
	chainState.EXPECT().StateDB().Return(stateDb)
	stateDb.EXPECT().Release()

	// Build a bundle with a valid block window.
	envelope := bundle.NewBuilder().
		SetEarliest(currentBlock - 5).
		SetLatest(currentBlock + 5).
		Build()

	acceptEverything := func(*types.Transaction, ChainStateForBundleEval, state.StateDB) bool {
		return true
	}

	state := getBundleState(chainState, envelope, acceptEverything)
	require.Equal(t, state, makeRunnableState())
}

func Test_GetBundleState_ChecksForNonceConflicts(t *testing.T) {

	temporaryBlocked := makeTemporaryBlockedState("gapped nonce")
	permanentlyBlocked := makePermanentlyBlockedState("bundle nonce check execution failed")

	const initialNonce = 1
	tests := map[string]struct {
		bundle pattern
		result BundleState
	}{
		"bundle with no transactions": {
			bundle: allOf(), // < will always succeed
			result: makeRunnableState(),
		},
		"bundle with one transaction with correct nonce": {
			bundle: allOf(1), // one tx with nonce 1
			result: makeRunnableState(),
		},
		"bundle with future nonce": {
			bundle: allOf(2), // one tx with nonce 2, which is in the future
			result: temporaryBlocked,
		},
		"bundle with outdated nonce": {
			bundle: allOf(0), // one tx with nonce 0, which is outdated
			result: permanentlyBlocked,
		},
		"bundle with different senders": {
			bundle: allOf(0xA1, 0xB1), // two txs from different senders with correct nonces
			result: makeRunnableState(),
		},
		"bundle with nonce gap": {
			bundle: allOf(1, 3), // two txs from the same sender with a nonce gap (nonce 2 is missing)
			result: permanentlyBlocked,
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
			chainState := NewMockChainStateForBundleEval(ctrl)
			chainState.EXPECT().GetLatestHeader().Return(currentHeader).AnyTimes()
			chainState.EXPECT().GetCurrentNetworkRules().Return(opera.Rules{
				NetworkID: 1,
				Upgrades:  opera.Upgrades{TransactionBundles: true},
			}).AnyTimes()
			chainState.EXPECT().StateDB().Return(db)
			db.EXPECT().Release()

			chainId := big.NewInt(1)
			signer := types.LatestSignerForChainID(chainId)

			envelope := test.bundle.toBundle(keys)
			_, _, err := bundle.ValidateEnvelope(signer, envelope)
			require.NoError(t, err)

			acceptEverything := func(*types.Transaction, ChainStateForBundleEval, state.StateDB) bool {
				return true
			}

			got := getBundleState(chainState, envelope, acceptEverything)
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
			result: makeRunnableState(),
		},
		"empty one-of bundle is non-executable": {
			bundle: oneOf(), // < can never succeed
			result: makePermanentlyBlockedState("bundle nonce check execution failed"),
		},
		"single all-of transaction with correct nonce": {
			bundle: allOf(1), // one tx with nonce 1
			result: makeRunnableState(),
		},
		"single one-of transaction with correct nonce": {
			bundle: oneOf(1),
			result: makeRunnableState(),
		},
		"single all-of transaction with old nonce": {
			bundle: allOf(0),
			result: makePermanentlyBlockedState("bundle nonce check execution failed"),
		},
		"single one-of transaction with old nonce": {
			bundle: oneOf(0),
			result: makePermanentlyBlockedState("bundle nonce check execution failed"),
		},
		"single all-of transaction with future nonce": {
			bundle: allOf(2),
			result: makeTemporaryBlockedState("gapped nonce"),
		},
		"single one-of transaction with future nonce": {
			bundle: oneOf(2),
			result: makeTemporaryBlockedState("gapped nonce"),
		},
		"multiple all-of transactions with correct nonce order": {
			bundle: allOf(1, 2, 3), // three txs with nonces 1, 2, 3
			result: makeRunnableState(),
		},
		"multiple one-of transactions with correct nonce order": {
			bundle: oneOf(1, 2, 3),
			result: makeRunnableState(),
		},
		"multiple all-of transactions out of order": {
			bundle: allOf(2, 1, 3),
			result: makePermanentlyBlockedState("bundle nonce check execution failed"),
		},
		"multiple one-of transactions out of order": {
			bundle: oneOf(2, 1, 3),
			result: makeRunnableState(),
		},
		"multiple all-of with old nonce": {
			bundle: allOf(0, 1, 2),
			result: makePermanentlyBlockedState("bundle nonce check execution failed"),
		},
		"multiple one-of with old nonce": {
			bundle: oneOf(0, 1, 2),
			result: makeRunnableState(),
		},
		"all-of with nonce gap": {
			bundle: allOf(1, 3),
			result: makePermanentlyBlockedState("bundle nonce check execution failed"),
		},
		"one-of with nonce gap": {
			bundle: oneOf(1, 3),
			result: makeRunnableState(),
		},
		"all-of with nonce gap in the future": {
			bundle: allOf(2, 4),
			result: makePermanentlyBlockedState("bundle nonce check execution failed"),
		},
		"one-of with nonce gap in the future": {
			bundle: oneOf(2, 4),
			result: makeTemporaryBlockedState("gapped nonce"),
		},
		"nested all-of with consecutive nonces": {
			bundle: allOf(1, allOf(2, 3), 4),
			result: makeRunnableState(),
		},
		"nested all-of with future nonces": {
			bundle: allOf(2, allOf(3, 4), 5),
			result: makeTemporaryBlockedState("gapped nonce"),
		},
		"nested all-of with nonce gap": {
			bundle: allOf(1, allOf(3, 4), 5),
			result: makePermanentlyBlockedState("bundle nonce check execution failed"),
		},
		"nested one-of in all-of": {
			bundle: allOf(1, oneOf(2, 3), 3),
			result: makeRunnableState(),
		},
		"multiple transactions from different senders with correct nonces": {
			// two txs from sender A with nonces 1 and 2, one tx from sender B with nonce 1
			bundle: allOf(0xA1, 0xB1, 0xA2),
			result: makeRunnableState(),
		},
		"multiple transactions from different senders with nonce gap for one sender": {
			bundle: allOf(0xA1, 0xB1, 0xA3),
			result: makePermanentlyBlockedState("bundle nonce check execution failed"),
		},
		"all-of outdated nonce for one sender but not the other": {
			bundle: allOf(0xA0, 0xB1),
			result: makePermanentlyBlockedState("bundle nonce check execution failed"),
		},
		"one-of outdated nonce for one sender but not the other": {
			bundle: oneOf(0xA0, 0xB1),
			result: makeRunnableState(),
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

			envelope := test.bundle.toBundle(keys)
			bundle, _, err := bundle.ValidateEnvelope(signer, envelope)
			require.NoError(t, err)

			got := checkForNonceConflicts(bundle, signer, source)
			require.Equal(t, test.result, got)
		})
	}
}

func Test_checkForNonceConflicts_LowestReferencedNoncesCannotBeDerived_ReturnsNonExecutable(t *testing.T) {
	invalidTx := types.NewTx(&types.LegacyTx{})
	bundle := &bundle.TransactionBundle{
		Transactions: map[bundle.TxReference]*types.Transaction{
			{}: invalidTx,
		},
	}
	signer := types.LatestSignerForChainID(big.NewInt(1))
	_, err := getLowestReferencedNonces(bundle, signer)
	require.Error(t, err)

	got := checkForNonceConflicts(bundle, signer, nil)
	require.Equal(t, got, makePermanentlyBlockedState("could not get lowest nonce for all accounts: failed to derive sender: invalid transaction v, r, s values"))
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

			envelope := test.bundle.toBundle(keys)
			bundle, _, err := bundle.ValidateEnvelope(signer, envelope)
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
		Transactions: map[bundle.TxReference]*types.Transaction{
			{}: types.NewTx(&types.LegacyTx{}),
		},
	}
	_, err := getLowestReferencedNonces(&bundle, signer)
	require.ErrorContains(t, err, "failed to derive sender")
}

func Test_getLowestReferencedNonces_DetectsInvalidNestedBundle(t *testing.T) {
	require := require.New(t)
	invalidBundle := types.NewTx(&types.LegacyTx{To: &bundle.BundleProcessor})
	require.True(bundle.IsEnvelope(invalidBundle))

	signer := types.LatestSignerForChainID(big.NewInt(1))
	_, _, err := bundle.ValidateEnvelope(signer, invalidBundle)
	require.Error(err)

	bundle := bundle.TransactionBundle{
		Transactions: map[bundle.TxReference]*types.Transaction{
			{}: invalidBundle,
		},
	}
	_, err = getLowestReferencedNonces(&bundle, signer)
	require.ErrorContains(err, "invalid nested bundle")
}

func Test_getLowestReferencedNonces_ReportsErrorWhileObtainingNoncesOfNestedBundles(t *testing.T) {
	require := require.New(t)
	invalidInner := types.NewTx(&types.LegacyTx{To: &bundle.BundleProcessor})
	require.True(bundle.IsEnvelope(invalidInner))

	signer := types.LatestSignerForChainID(big.NewInt(1))
	_, _, err := bundle.ValidateEnvelope(signer, invalidInner)
	require.Error(err)

	key, err := crypto.GenerateKey()
	require.NoError(err)

	middle := bundle.NewBuilder().
		With(bundle.Step(key, invalidInner)).
		Build()

	outer, _ := bundle.NewBuilder().
		With(bundle.Step(key, middle)).
		BuildBundleAndPlan()

	_, err = getLowestReferencedNonces(outer, signer)
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

func Test_makeRunnableState_ReturnsRunnableState(t *testing.T) {
	state := makeRunnableState()
	require.Equal(t, BundleState{
		Executable:         true,
		TemporarilyBlocked: false,
		Reasons:            nil,
	}, state)
}

func Test_makeTemporaryBlockedState_ReturnsTemporaryBlockedState(t *testing.T) {
	state := makeTemporaryBlockedState("some reason")
	require.Equal(t, BundleState{
		Executable:         false,
		TemporarilyBlocked: true,
		Reasons:            []string{"some reason"},
	}, state)
}

func Test_makePermanentlyBlockedState_ReturnsPermanentlyBlockedState(t *testing.T) {
	state := makePermanentlyBlockedState("some reason")
	require.Equal(t, BundleState{
		Executable:         false,
		TemporarilyBlocked: false,
		Reasons:            []string{"some reason"},
	}, state)
}

func Test_trialRunBundle_DoesRunTransactionsThroughEVMAndReturnsIfTransactionsGotAccepted(t *testing.T) {
	// This is an integration test for the trialRunBundle function that
	// performs an actual run on the EVM.

	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	tests := map[string]struct {
		envelope       *types.Transaction
		expectedResult bool
	}{
		"empty all-of is rejected": {
			envelope:       bundle.AllOf().Build(),
			expectedResult: false,
		},
		"empty one-of is rejected": {
			envelope:       bundle.OneOf().Build(),
			expectedResult: false,
		},
		"single transaction that gets accepted": {
			envelope: bundle.AllOf(
				bundle.Step(key, &types.AccessListTx{
					To:  &common.Address{},
					Gas: 21_000,
				}),
			).Build(),
			expectedResult: true,
		},
		"single transaction that is skipped": {
			envelope: bundle.AllOf(
				bundle.Step(key, &types.AccessListTx{
					To:  &common.Address{},
					Gas: 0, // < not enough gas
				}),
			).Build(),
			expectedResult: false,
		},
		"multiple accepted transactions": {
			envelope: bundle.AllOf(
				bundle.Step(key, &types.AccessListTx{
					To:  &common.Address{},
					Gas: 21_000,
				}),
				bundle.Step(key, &types.AccessListTx{
					To:  &common.Address{1},
					Gas: 21_000,
				}),
			).Build(),
			expectedResult: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			chainState := NewMockChainStateForBundleEval(ctrl)
			chainState.EXPECT().GetLatestHeader().Return(&EvmHeader{
				Number:  big.NewInt(0),
				BaseFee: new(big.Int),
			}).AnyTimes()
			chainState.EXPECT().GetEvmChainConfig(gomock.Any()).Return(&params.ChainConfig{
				ChainID: big.NewInt(1),
			}).AnyTimes()
			chainState.EXPECT().GetCurrentNetworkRules().Return(opera.Rules{
				Upgrades: opera.Upgrades{TransactionBundles: true},
			}).AnyTimes()

			// setup of the state DB to support the EVM execution, the actual
			// values are not relevant for this test;
			any := gomock.Any()
			stateDb := state.NewMockStateDB(ctrl)
			stateDb.EXPECT().HasBundleRecentlyBeenProcessed(any)
			stateDb.EXPECT().InterTxSnapshot().AnyTimes()
			stateDb.EXPECT().RevertToInterTxSnapshot(any).AnyTimes()
			stateDb.EXPECT().AddProcessedBundle(any, any)
			stateDb.EXPECT().Prepare(any, any, any, any, any, any).AnyTimes()
			stateDb.EXPECT().SetTxContext(any, any).AnyTimes()
			stateDb.EXPECT().GetNonce(any).AnyTimes()
			stateDb.EXPECT().SetNonce(any, any, any).AnyTimes()
			stateDb.EXPECT().GetBalance(any).Return(uint256.NewInt(math.MaxInt64)).AnyTimes()
			stateDb.EXPECT().SubBalance(any, any, any).AnyTimes()
			stateDb.EXPECT().AddBalance(any, any, any).AnyTimes()
			stateDb.EXPECT().Snapshot().AnyTimes()
			stateDb.EXPECT().RevertToSnapshot(any).AnyTimes()
			stateDb.EXPECT().GetCodeHash(any).Return(types.EmptyCodeHash).AnyTimes()
			stateDb.EXPECT().GetCode(any).AnyTimes()
			stateDb.EXPECT().GetCodeSize(any).AnyTimes()
			stateDb.EXPECT().Exist(any).Return(true).AnyTimes()
			stateDb.EXPECT().GetRefund().AnyTimes()
			stateDb.EXPECT().AddRefund(any).AnyTimes()
			stateDb.EXPECT().SubRefund(any).AnyTimes()
			stateDb.EXPECT().GetLogs(any, any).AnyTimes()
			stateDb.EXPECT().TxIndex().AnyTimes()
			stateDb.EXPECT().EndTransaction().AnyTimes()

			// run the bundle through the EVM and check the result
			got := trialRunBundle(tc.envelope, chainState, stateDb)
			require.Equal(t, tc.expectedResult, got)
		})
	}
}

func Test_trialRunBundle_UsesRandomPrevRandaoValue(t *testing.T) {
	// This test verifies that the trialRunBundle function indeed uses a random
	// source for determining PrevRandao values. It does so by running code
	// that reads the PrevRandao and stores it in a storage slot at position 0.
	require := require.New(t)
	ctrl := gomock.NewController(t)

	key, err := crypto.GenerateKey()
	require.NoError(err)

	targetAddress := common.Address{1}
	envelope := bundle.NewBuilder().AllOf(
		bundle.Step(key, types.AccessListTx{
			To:  &targetAddress,
			Gas: 50_000,
		}),
	).Build()

	code := []byte{
		byte(vm.PREVRANDAO),
		byte(vm.PUSH0),
		byte(vm.SSTORE),
	}

	// setup of the state DB to support the EVM execution, the actual
	// values are not relevant for this test;
	any := gomock.Any()
	db := state.NewMockStateDB(ctrl)
	db.EXPECT().InterTxSnapshot().AnyTimes()
	db.EXPECT().RevertToInterTxSnapshot(any).AnyTimes()
	db.EXPECT().Prepare(any, any, any, any, any, any).AnyTimes()
	db.EXPECT().SetTxContext(any, any).AnyTimes()
	db.EXPECT().GetNonce(any).AnyTimes()
	db.EXPECT().SetNonce(any, any, any).AnyTimes()
	db.EXPECT().GetBalance(any).Return(uint256.NewInt(math.MaxInt64)).AnyTimes()
	db.EXPECT().SubBalance(any, any, any).AnyTimes()
	db.EXPECT().AddBalance(any, any, any).AnyTimes()
	db.EXPECT().Snapshot().AnyTimes()
	db.EXPECT().RevertToSnapshot(any).AnyTimes()
	db.EXPECT().GetCodeHash(any).Return(types.EmptyCodeHash).AnyTimes()
	db.EXPECT().Exist(any).Return(true).AnyTimes()
	db.EXPECT().GetRefund().AnyTimes()
	db.EXPECT().AddRefund(any).AnyTimes()
	db.EXPECT().SubRefund(any).AnyTimes()
	db.EXPECT().GetLogs(any, any).AnyTimes()
	db.EXPECT().TxIndex().AnyTimes()
	db.EXPECT().EndTransaction().AnyTimes()
	db.EXPECT().HasBundleRecentlyBeenProcessed(any).AnyTimes()
	db.EXPECT().AddProcessedBundle(any, any).AnyTimes()
	db.EXPECT().SlotInAccessList(any, any).AnyTimes()
	db.EXPECT().AddSlotToAccessList(any, any).AnyTimes()
	db.EXPECT().GetStateAndCommittedState(any, any).AnyTimes()

	// The critical parts causing the code execution:
	db.EXPECT().GetCode(targetAddress).Return(code).AnyTimes()
	db.EXPECT().GetCode(any).AnyTimes()

	// Track values being stored into values
	seenHashes := map[common.Hash]struct{}{}
	db.EXPECT().SetState(any, any, any).DoAndReturn(
		func(_ common.Address, key common.Hash, value common.Hash) common.Hash {
			require.Zero(key)
			_, seen := seenHashes[value]
			require.False(seen, "seen hash %v multiple times", value)
			seenHashes[value] = struct{}{}
			return common.Hash{}
		},
	).AnyTimes()

	chainState := NewMockChainStateForBundleEval(ctrl)
	chainState.EXPECT().GetLatestHeader().Return(&EvmHeader{
		Number:  big.NewInt(0),
		BaseFee: new(big.Int),
	}).AnyTimes()
	chainState.EXPECT().GetEvmChainConfig(any).Return(&params.ChainConfig{
		ChainID:            big.NewInt(1),
		LondonBlock:        new(big.Int).SetUint64(0),
		MergeNetsplitBlock: new(big.Int).SetUint64(0),
		ShanghaiTime:       new(uint64),
		CancunTime:         new(uint64),
	}).AnyTimes()
	rules := opera.GetBrioUpgrades()
	rules.TransactionBundles = true
	chainState.EXPECT().GetCurrentNetworkRules().Return(opera.Rules{
		Upgrades: rules,
	}).AnyTimes()

	const N = 10
	for range N {
		trialRunBundle(envelope, chainState, db)
	}
	require.Len(seenHashes, N)
}

func Test_trialRunBundleInternal_CreatesSnapshotAndRevertsAfterExecution(t *testing.T) {
	ctrl := gomock.NewController(t)

	any := gomock.Any()
	processor := NewMocktransactionProcessor(ctrl)
	processor.EXPECT().Run(any, any)

	factory := NewMocktransactionProcessorFactory(ctrl)
	factory.EXPECT().newTransactionProcessor(any, any, any).DoAndReturn(
		func(_ ChainState, db state.StateDB, _ *EvmBlock) transactionProcessor {
			db.GetNonce(common.Address{12})
			return processor
		},
	)

	db := state.NewMockStateDB(ctrl)
	gomock.InOrder(
		db.EXPECT().InterTxSnapshot().Return(42), // created before use
		db.EXPECT().GetNonce(common.Address{12}), // simulated use
		db.EXPECT().RevertToInterTxSnapshot(42),  // reverted after use
	)

	chainState := NewMockChainStateForBundleEval(ctrl)
	chainState.EXPECT().GetLatestHeader().Return(&EvmHeader{
		Number: big.NewInt(0),
	})

	trialRunBundleInternal(nil, chainState, db, factory, rand.Read)
}

func Test_trialRunBundleInternal_UsesRandomSourceToFillPrevRandao(t *testing.T) {
	require := require.New(t)
	var randomHash common.Hash
	_, err := rand.Read(randomHash[:])
	require.NoError(err)

	values := map[string]common.Hash{
		"zero":   {},
		"one":    {1},
		"random": randomHash,
	}

	for name, prevRandao := range values {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			called := false
			read := func(trg []byte) (int, error) {
				called = true
				return copy(trg, prevRandao[:]), nil
			}

			any := gomock.Any()
			processor := NewMocktransactionProcessor(ctrl)
			processor.EXPECT().Run(any, any)

			factory := NewMocktransactionProcessorFactory(ctrl)
			factory.EXPECT().newTransactionProcessor(any, any, any).DoAndReturn(
				func(_ ChainState, _ state.StateDB, block *EvmBlock) transactionProcessor {
					require.Equal(prevRandao, block.PrevRandao)
					return processor
				},
			)

			db := state.NewMockStateDB(ctrl)
			gomock.InOrder(
				db.EXPECT().InterTxSnapshot().Return(42),
				db.EXPECT().RevertToInterTxSnapshot(42),
			)

			chainState := NewMockChainStateForBundleEval(ctrl)
			chainState.EXPECT().GetLatestHeader().Return(&EvmHeader{
				Number: big.NewInt(0),
			})

			trialRunBundleInternal(nil, chainState, db, factory, read)
			require.True(called)
		})
	}
}

func Test_trialRunBundleInternal_FailsIfRandomSourceFails(t *testing.T) {
	tests := map[string]struct {
		n   int
		err error
	}{
		"wrong length": {n: 10}, // should be length of hash = 32
		"with error":   {err: fmt.Errorf("injected error")},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			chain := NewMockChainStateForBundleEval(ctrl)
			chain.EXPECT().GetLatestHeader().Return(&EvmHeader{})

			readRandom := func([]byte) (int, error) {
				return tc.n, tc.err
			}

			require.False(t, trialRunBundleInternal(nil, chain, nil, nil, readRandom))
		})
	}
}

func Test_trialRunBundleInternal_DerivesHeaderFieldsFromChainState(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)

	latestHeader := &EvmHeader{
		Number:  big.NewInt(123),
		Time:    456,
		BaseFee: big.NewInt(789),
	}

	chainState := NewMockChainStateForBundleEval(ctrl)
	chainState.EXPECT().GetLatestHeader().Return(latestHeader)

	any := gomock.Any()
	processor := NewMocktransactionProcessor(ctrl)
	processor.EXPECT().Run(any, any)

	factory := NewMocktransactionProcessorFactory(ctrl)
	factory.EXPECT().newTransactionProcessor(any, any, any).DoAndReturn(
		func(_ ChainState, _ state.StateDB, block *EvmBlock) transactionProcessor {

			// check all the header fields forwarded to the EVM
			require.Equal(new(big.Int).Add(latestHeader.Number, big.NewInt(1)), block.Number) // latest header number + 1
			require.Equal(latestHeader.Time+1, block.Time)
			require.Equal(latestHeader.GasLimit, block.GasLimit)
			require.Equal(GetCoinbase(), block.Coinbase)
			require.NotZero(block.PrevRandao)
			require.Equal(latestHeader.BaseFee, block.BaseFee)

			blobBaseFee := GetBlobBaseFee()
			require.Equal(blobBaseFee.ToBig(), block.BlobBaseFee)

			return processor
		},
	)

	db := state.NewMockStateDB(ctrl)
	db.EXPECT().InterTxSnapshot()
	db.EXPECT().RevertToInterTxSnapshot(any)

	trialRunBundleInternal(nil, chainState, db, factory, rand.Read)
}

func Test_trialRunBundleInternal_ForwardsEnvelopeToProcessor(t *testing.T) {
	ctrl := gomock.NewController(t)

	myEnvelope := &types.Transaction{}

	latestHeader := &EvmHeader{
		Number: big.NewInt(123),
	}

	chainState := NewMockChainStateForBundleEval(ctrl)
	chainState.EXPECT().GetLatestHeader().Return(latestHeader)

	any := gomock.Any()
	processor := NewMocktransactionProcessor(ctrl)
	processor.EXPECT().Run(0, myEnvelope) // < test target

	factory := NewMocktransactionProcessorFactory(ctrl)
	factory.EXPECT().newTransactionProcessor(any, any, any).Return(processor)

	db := state.NewMockStateDB(ctrl)
	db.EXPECT().InterTxSnapshot()
	db.EXPECT().RevertToInterTxSnapshot(any)

	trialRunBundleInternal(myEnvelope, chainState, db, factory, rand.Read)
}

func Test_trialRunBundleInternal_UsesPresentsOfReceiptToDecideResult(t *testing.T) {

	tests := map[string]struct {
		processedTxs   []ProcessedTransaction
		expectedResult bool
	}{
		"no result": {
			processedTxs:   nil,
			expectedResult: false,
		},
		"single result without receipt": {
			processedTxs:   []ProcessedTransaction{{}},
			expectedResult: false,
		},
		"single result with receipt": {
			processedTxs:   []ProcessedTransaction{{Receipt: &types.Receipt{}}},
			expectedResult: true,
		},
		"multiple results without receipt": {
			processedTxs:   []ProcessedTransaction{{}, {}, {}},
			expectedResult: false,
		},
		"multiple results with some receipt": {
			processedTxs:   []ProcessedTransaction{{}, {Receipt: &types.Receipt{}}, {}},
			expectedResult: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			myEnvelope := &types.Transaction{}

			latestHeader := &EvmHeader{
				Number: big.NewInt(123),
			}

			chainState := NewMockChainStateForBundleEval(ctrl)
			chainState.EXPECT().GetLatestHeader().Return(latestHeader)

			any := gomock.Any()
			processor := NewMocktransactionProcessor(ctrl)
			processor.EXPECT().Run(any, any).Return(ProcessSummary{
				ProcessedTransactions: tc.processedTxs,
			})

			factory := NewMocktransactionProcessorFactory(ctrl)
			factory.EXPECT().newTransactionProcessor(any, any, any).Return(processor)

			db := state.NewMockStateDB(ctrl)
			db.EXPECT().InterTxSnapshot()
			db.EXPECT().RevertToInterTxSnapshot(any)

			got := trialRunBundleInternal(myEnvelope, chainState, db, factory, rand.Read)
			require.Equal(t, tc.expectedResult, got)
		})
	}
}

// --- Utility functions to build test bundles ---

func allOf(nested ...any) pattern {
	return pattern{
		oneOf:  false,
		nested: nested,
	}
}

func oneOf(nested ...any) pattern {
	return pattern{
		oneOf:  true,
		nested: nested,
	}
}

type pattern struct {
	oneOf  bool
	nested []any
}

func (p pattern) toBundle(
	keys []*ecdsa.PrivateKey,
) *types.Transaction {
	// convert elements into steps
	steps := make([]bundle.BuilderStep, 0, len(p.nested))
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
				keys[0], // for envelope transaction, any key is fine
				v.toBundle(keys),
			))
		default:
			panic("unsupported element type")
		}
	}

	// Build the resulting bundle.
	return bundle.NewBuilder().With(bundle.Group(p.oneOf, steps...)).Build()
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
