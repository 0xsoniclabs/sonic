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
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/utils"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestTxPool_TransactionsAreQueuedAccordingToTheirExecutionStatus(t *testing.T) {

	chainId := big.NewInt(123)
	blockNumber := idx.Block(1)

	rules := opera.Rules{
		NetworkID: chainId.Uint64(),
		Economy: opera.EconomyRules{
			Gas: opera.GasRules{
				MaxEventGas: 30_000_000,
			},
		},
		Upgrades: opera.Upgrades{
			Allegro:            true,
			Brio:               true,
			TransactionBundles: true,
		},
	}
	signer := types.LatestSignerForChainID(chainId)

	chainConfig := opera.CreateTransientEvmChainConfig(
		chainId.Uint64(),
		[]opera.UpgradeHeight{{Upgrades: rules.Upgrades, Height: 0}},
		blockNumber,
	)

	poolConfig := DefaultTxPoolConfig
	poolConfig.Journal = ""
	poolConfig.DisableTxPoolValidation = true

	tests := map[string]struct {
		accountCurrentNonce uint64
		nonceOffset         uint64
		expectedPending     int
		expectedQueued      int
	}{
		"permanently blocked are dropped": {
			// all generated transactions are in the past
			accountCurrentNonce: 1,
			expectedPending:     0,
			expectedQueued:      0,
		},
		"temporarily blocked remain queued": {
			// all generated transactions are in the future
			nonceOffset:     1,
			expectedPending: 0,
			expectedQueued:  1,
		},
		"executable are pending": {
			expectedPending: 1,
			expectedQueued:  0,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			require := require.New(t)
			ctrl := gomock.NewController(t)

			bundleeKey, err := crypto.GenerateKey()
			require.NoError(err)
			bundleeAdress := crypto.PubkeyToAddress(bundleeKey.PublicKey)

			any := gomock.Any()

			// Mock the state to accept any transaction.
			stateDb := state.NewMockStateDB(ctrl)
			mockStateDbExecutionUsageForAccountWithNonce(stateDb, bundleeAdress, test.accountCurrentNonce)

			chain := NewMockStateReader(ctrl)
			chain.EXPECT().CurrentBlock().Return(&EvmBlock{
				EvmHeader: EvmHeader{Number: big.NewInt(int64(blockNumber))},
			}).AnyTimes()
			chain.EXPECT().CurrentConfig().Return(chainConfig).AnyTimes()
			chain.EXPECT().CurrentStateDB().Return(stateDb, nil).AnyTimes()
			chain.EXPECT().CurrentMaxGasLimit().Return(rules.Economy.Gas.MaxEventGas).AnyTimes()
			chain.EXPECT().CurrentBaseFee().Return(big.NewInt(1)).AnyTimes()
			chain.EXPECT().CurrentRules().Return(rules).AnyTimes()

			subscriber := NewMocksubscriber(ctrl)
			subscriber.EXPECT().Err().Return(make(chan error)).AnyTimes()
			subscriber.EXPECT().Unsubscribe().AnyTimes()
			chain.EXPECT().SubscribeNewBlock(any).Return(subscriber).AnyTimes()

			subsidiesCheckFactory := func(opera.Rules, StateReader, state.StateDB, types.Signer) utils.TransactionCheckFunc {
				return nil
			}

			bundleEvaluationCache := NewBundleEvaluationCache()

			pool := newTxPool(poolConfig, chainConfig, chain, subsidiesCheckFactory, bundleEvaluationCache)

			pending, queued := pool.Content()
			expectContents(t, pending, queued, 0, 0)

			tx := bundle.NewBuilder().
				WithSigner(signer).
				SetEnvelopeNonce(0).
				With(bundle.Step(bundleeKey,
					&types.AccessListTx{
						// offsetting the nonce creates gapped transactions
						// which cannot be immediately executed
						Nonce: uint64(0 + test.nonceOffset),
						Gas:   60_000,
					})).
				Build()

			// adding transactions triggers and waits for reorg.
			// promotions/drops are synchronous.
			err = pool.AddLocal(tx)
			require.NoError(err, "failed to add bundle transaction to the pool")

			pending, queued = pool.Content()
			expectContents(t, pending, queued, test.expectedPending, test.expectedQueued)
		})
	}
}

func expectContents(t *testing.T,
	pending, queued map[common.Address]types.Transactions,
	expectedPending, expectedQueued int,
) {
	t.Helper()

	if expectedPending != 0 {
		require.Len(t, pending, expectedPending, "expected pending transactions for each account")
	} else {
		require.Empty(t, pending, "expected no pending transactions")
	}

	if expectedQueued != 0 {
		require.Len(t, queued, expectedQueued, "expected queued transactions for each account")
	} else {
		require.Empty(t, queued, "expected no queued transactions")
	}
}

func mockStateDbExecutionUsageForAccountWithNonce(
	stateDb *state.MockStateDB,
	sender common.Address,
	nonce uint64,
) {
	any := gomock.Any()

	// When asked for the nonce of the specific sender, return the provided nonce.
	// For any other address, return 0.
	stateDb.EXPECT().GetNonce(sender).Return(nonce).AnyTimes()
	stateDb.EXPECT().GetNonce(any).Return(uint64(0)).AnyTimes()

	stateDb.EXPECT().GetBalance(any).Return(uint256.NewInt(1e18)).AnyTimes()
	stateDb.EXPECT().GetCodeHash(any).Return(types.EmptyCodeHash).AnyTimes()
	stateDb.EXPECT().GetCode(any).Return([]byte{}).AnyTimes()
	stateDb.EXPECT().HasBundleRecentlyBeenProcessed(any).Return(false).AnyTimes()
	stateDb.EXPECT().InterTxSnapshot().AnyTimes()
	stateDb.EXPECT().RevertToInterTxSnapshot(any).AnyTimes()
	stateDb.EXPECT().SetTxContext(any, any).AnyTimes()
	stateDb.EXPECT().EndTransaction().AnyTimes()
	stateDb.EXPECT().AddProcessedBundle(any, any).AnyTimes()
	stateDb.EXPECT().AddAddressToAccessList(any).AnyTimes()
	stateDb.EXPECT().Snapshot().AnyTimes()
	stateDb.EXPECT().RevertToSnapshot(any).AnyTimes()
	stateDb.EXPECT().Exist(any).AnyTimes()
	stateDb.EXPECT().Finalise(any).AnyTimes()
	stateDb.EXPECT().SubBalance(any, any, any).AnyTimes()
	stateDb.EXPECT().Prepare(any, any, any, any, any, any).AnyTimes()
	stateDb.EXPECT().SetNonce(any, any, any).AnyTimes()
	stateDb.EXPECT().GetStorageRoot(any).AnyTimes()
	stateDb.EXPECT().CreateAccount(any).AnyTimes()
	stateDb.EXPECT().CreateContract(any).AnyTimes()
	stateDb.EXPECT().AddBalance(any, any, any).AnyTimes()
	stateDb.EXPECT().AddRefund(any).AnyTimes()
	stateDb.EXPECT().GetRefund().AnyTimes()
	stateDb.EXPECT().SubRefund(any).AnyTimes()
	stateDb.EXPECT().GetLogs(any, any).AnyTimes()
	stateDb.EXPECT().TxIndex().AnyTimes()
}

// ========================== Tools ===========================

// bundleTx creates a transaction that is part of a bundle with the given key and nonce.
// the name follows the convention of other transaction creation tools in the tx pool tests.
func bundleTx(nonce uint64, key *ecdsa.PrivateKey) *types.Transaction {
	signer := types.LatestSignerForChainID(params.TestChainConfig.ChainID)
	return bundle.NewBuilder().
		WithSigner(signer).
		SetEnvelopeSenderKey(key).
		SetEnvelopeNonce(nonce).
		With(bundle.Step(key, &types.AccessListTx{Nonce: nonce})).
		Build()
}
