package evmcore

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

//go:generate mockgen -source=tx_pool_subsidies_test.go -destination=tx_pool_subsidies_test_mock.go -package=evmcore

type subscriber interface {
	event.Subscription
}

// This file contains tests related to gas subsidies in the transaction pool.
// This file is intentionally separated from other pool tests to avoid polluting
// them with extra test tools.

func TestTxPool_SponsoredTransactionsAreIncludedInThePendingSet(t *testing.T) {
	ctrl := gomock.NewController(t)

	chainId := big.NewInt(1)
	blockNumber := idx.Block(1)
	poolConfig := TxPoolConfig{MinimumTip: 15}
	upgrades := opera.Upgrades{GasSubsidies: true}

	// Create a ChainConfig instance with the expected features enabled
	// at the block height.
	chainConfig := opera.CreateTransientEvmChainConfig(
		chainId.Uint64(),
		[]opera.UpgradeHeight{{Upgrades: upgrades, Height: 0}},
		blockNumber,
	)

	// mock the external chain dependencies
	chain := mockChain(ctrl, chainConfig, upgrades)

	// Instantiate the pool
	pool := NewTxPool(poolConfig, chainConfig, chain)

	// Queue some sponsored transactions
	const sponsoredTxs = 5
	for range sponsoredTxs {
		tx := signTx(t, &types.LegacyTx{
			GasPrice: big.NewInt(0),
			Gas:      21_000,
			To:       &common.Address{1}, // not a contract creation
		}, chainId)
		err := pool.addRemoteSync(tx)
		require.NoError(t, err)
	}

	// Add some valid normal transactions with tips above the minimum
	const tippedTransactions = 5
	for range tippedTransactions {
		tx := signTx(t, &types.DynamicFeeTx{
			GasTipCap: big.NewInt(int64(poolConfig.MinimumTip)), // valid tip
			GasFeeCap: big.NewInt(100),
			Gas:       21_000,
			To:        &common.Address{1}, // not a contract creation
		}, chainId)
		err := pool.addRemoteSync(tx)
		require.NoError(t, err)
	}

	// Add some valid local transactions with tips bellow the minimum
	const localTransactions = 5
	for range localTransactions {
		tx := signTx(t, &types.DynamicFeeTx{
			GasTipCap: big.NewInt(int64(poolConfig.MinimumTip - 1)), // below minimum tip, but valid as local
			GasFeeCap: big.NewInt(100),
			Gas:       21_000,
			To:        &common.Address{1}, // not a contract creation
		}, chainId)
		err := pool.AddLocal(tx)
		require.NoError(t, err)
	}

	pending, err := pool.Pending(true) // with tips enforcement
	require.NoError(t, err)
	require.Len(t,
		pending, sponsoredTxs+tippedTransactions+localTransactions,
		"expected all valid txs to be included")

	pendingSponsored := make([]*types.Transaction, 0, len(pending))
	pendingNormal := make([]*types.Transaction, 0, len(pending))
	for _, txs := range pending {
		// in this test, one tx per sender
		for _, tx := range txs {
			if tx.GasPrice().Sign() == 0 {
				pendingSponsored = append(pendingSponsored, tx)
			} else {
				pendingNormal = append(pendingNormal, tx)
			}
		}
	}
	require.Len(t, pendingSponsored, sponsoredTxs, "expected all sponsored txs to be included")
	require.Len(t, pendingNormal, tippedTransactions+localTransactions, "expected all tipped txs to be included")
}

////////////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////////////

// mockChain creates a mock chain with basic expectations which allow to accept
// any transaction in the pool.
func mockChain(ctrl *gomock.Controller, chainConfig *params.ChainConfig, upgrades opera.Upgrades) *MockStateReader {
	state := state.NewMockStateDB(ctrl)
	state.EXPECT().GetNonce(gomock.Any()).Return(uint64(0)).AnyTimes()
	state.EXPECT().GetBalance(gomock.Any()).Return(uint256.NewInt(1e18)).AnyTimes()
	state.EXPECT().GetCodeHash(gomock.Any()).Return(types.EmptyCodeHash).AnyTimes()

	chain := NewMockStateReader(ctrl)
	chain.EXPECT().CurrentBlock().Return(&EvmBlock{
		EvmHeader: EvmHeader{
			Number: big.NewInt(1),
		},
	}).AnyTimes()
	chain.EXPECT().Config().Return(chainConfig).AnyTimes()
	chain.EXPECT().GetTxPoolStateDB().Return(state, nil).AnyTimes()
	chain.EXPECT().MaxGasLimit().Return(uint64(30_000_000)).AnyTimes()
	chain.EXPECT().GetCurrentBaseFee().Return(big.NewInt(1)).AnyTimes()

	sub := NewMocksubscriber(ctrl)
	sub.EXPECT().Err().Return(make(chan error)).AnyTimes()
	sub.EXPECT().Unsubscribe().AnyTimes()

	chain.EXPECT().SubscribeNewBlock(gomock.Any()).Return(sub).AnyTimes()
	chain.EXPECT().GetCurrentRules().
		Return(opera.Rules{Upgrades: upgrades}).AnyTimes()
	return chain
}

// singTx creates and signs a transaction with a new key for each call.
func signTx(t *testing.T, txData types.TxData, chainId *big.Int) *types.Transaction {
	t.Helper()
	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	signer := types.LatestSignerForChainID(chainId)
	return types.MustSignNewTx(key, signer, txData)
}

// suppress unused warning
var _ subscriber
