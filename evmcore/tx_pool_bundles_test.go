package evmcore

import (
	"crypto/ecdsa"
	"fmt"
	big "math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/utils"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestTxPool_BundleEnvelopesAreIncludedInThePendingSet(t *testing.T) {
	ctrl := gomock.NewController(t)

	poolConfig := TxPoolConfig{MinimumTip: 15}
	upgrades := opera.Upgrades{Brio: true, TransactionBundles: true}

	// Create a ChainConfig instance with the expected features enabled
	// at the block height.
	chainConfig := opera.CreateTransientEvmChainConfig(
		params.TestChainConfig.ChainID.Uint64(),
		[]opera.UpgradeHeight{{Upgrades: upgrades, Height: 0}},
		idx.Block(1),
	)

	// mock the external stateReader dependencies
	stateReader := NewMockStateReader(ctrl)
	stateReader.EXPECT().CurrentBlock().Return(
		&EvmBlock{
			EvmHeader: EvmHeader{
				Number:  big.NewInt(1),
				BaseFee: big.NewInt(1e9),
			},
		},
	).AnyTimes()

	subsidiesCheckFactory := func(opera.Rules, StateReader, state.StateDB, types.Signer) utils.TransactionCheckFunc {
		// This test accepts all sponsorship requests
		return func(tx *types.Transaction) bool {
			return true
		}
	}
	bundlesCheckFactory := func(opera.Rules, StateReader, state.StateDB, types.Signer) utils.TransactionCheckFunc {
		return nil
	}

	// Instantiate the pool
	pool := newTxPool(poolConfig, chainConfig, stateReader, subsidiesCheckFactory, bundlesCheckFactory)

	// stateDb := state.NewMockStateDB(ctrl)
	// pool.currentState = stateDb

	const accountCount = 5
	const transactionsCount = 5

	for range accountCount {
		key, err := crypto.GenerateKey()
		require.NoError(t, err)

		for i := range transactionsCount {

			tx := bundleTx(key, uint64(i))
			err := pool.addRemoteSync(tx)
			require.NoError(t, err)
		}
	}
}

func Test_makeTransientBundleCheck_RejectsMalformedBundles(t *testing.T) {
	pool, key := setupTxPool()
	canBundleBeExecuted := makeTransientBundleCheck(pool)
	res := canBundleBeExecuted(types.MustSignNewTx(key, pool.signer, &types.AccessListTx{}))
	require.False(t, res)
}

func Test_makeTransientBundleCheck_RejectsRecentlyExecutedBundles(t *testing.T) {
	pool, key := setupTxPool()

	canBundleBeExecuted := makeTransientBundleCheck(pool)
	ctrl := gomock.NewController(t)
	stateDbMock := state.NewMockStateDB(ctrl)
	pool.currentState = stateDbMock

	stateDbMock.EXPECT().HasBundleRecentlyBeenProcessed(gomock.Any()).Return(true)

	res := canBundleBeExecuted(bundleTx(key, 0))
	require.False(t, res)
}

func Test_makeTransientBundleCheck_RejectsBundlesWhichCannotBeExecuted(t *testing.T) {
	for _, accepted := range []bool{true, false} {
		t.Run(fmt.Sprintf("accepted=%v", accepted), func(t *testing.T) {

			pool, key := setupTxPool()

			pool.bundleCheckerCache = utils.NewCheckerCache(-1)
			pool.bundleCheckerFactory = func(
				rules opera.Rules,
				chain StateReader,
				state state.StateDB,
				signer types.Signer,
			) utils.TransactionCheckFunc {
				return func(tx *types.Transaction) bool {
					return accepted
				}
			}

			canBundleBeExecuted := makeTransientBundleCheck(pool)
			ctrl := gomock.NewController(t)
			stateDbMock := state.NewMockStateDB(ctrl)
			pool.currentState = stateDbMock
			stateDbMock.EXPECT().HasBundleRecentlyBeenProcessed(gomock.Any()).Return(false)

			res := canBundleBeExecuted(bundleTx(key, 0))
			require.Equal(t, accepted, res)
		})
	}
}

// ========================== Tools ===========================

// bundleTx creates a transaction that is part of a bundle with the given key and nonce.
// the name follows the convention of other transaction creation tools in the tx pool tests.
func bundleTx(key *ecdsa.PrivateKey, nonce uint64) *types.Transaction {
	signer := types.LatestSignerForChainID(params.TestChainConfig.ChainID)
	return bundle.NewBuilder(signer).
		SetEnvelopeSenderKey(key).
		SetEnvelopeNonce(nonce).
		With(bundle.Step(key, &types.AccessListTx{Nonce: nonce})).
		Build()
}
