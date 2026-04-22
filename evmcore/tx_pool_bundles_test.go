package evmcore

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/utils"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestTxPool_ExecutableBundleEnvelopesAreIncludedInThePendingSet(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)

	chainId := big.NewInt(1)
	blockNumber := idx.Block(1)
	upgrades := opera.Upgrades{
		Brio:               true,
		TransactionBundles: true,
	}

	chainConfig := opera.CreateTransientEvmChainConfig(
		chainId.Uint64(),
		[]opera.UpgradeHeight{{Upgrades: upgrades, Height: 0}},
		blockNumber,
	)

	// Mock the state to accept any transaction.
	stateDb := state.NewMockStateDB(ctrl)
	stateDb.EXPECT().GetNonce(gomock.Any()).Return(uint64(0)).AnyTimes()
	stateDb.EXPECT().GetBalance(gomock.Any()).Return(uint256.NewInt(1e18)).AnyTimes()
	stateDb.EXPECT().GetCodeHash(gomock.Any()).Return(types.EmptyCodeHash).AnyTimes()
	stateDb.EXPECT().GetCode(gomock.Any()).Return([]byte{}).AnyTimes()
	stateDb.EXPECT().HasBundleRecentlyBeenProcessed(gomock.Any()).Return(false).AnyTimes()

	chain := NewMockStateReader(ctrl)
	chain.EXPECT().CurrentBlock().Return(&EvmBlock{
		EvmHeader: EvmHeader{Number: big.NewInt(1)},
	}).AnyTimes()
	chain.EXPECT().CurrentConfig().Return(chainConfig).AnyTimes()
	chain.EXPECT().CurrentStateDB().Return(stateDb, nil).AnyTimes()
	chain.EXPECT().CurrentMaxGasLimit().Return(uint64(30_000_000)).AnyTimes()
	chain.EXPECT().CurrentBaseFee().Return(big.NewInt(1)).AnyTimes()
	chain.EXPECT().CurrentRules().Return(opera.Rules{Upgrades: upgrades}).AnyTimes()

	sub := NewMocksubscriber(ctrl)
	sub.EXPECT().Err().Return(make(chan error)).AnyTimes()
	sub.EXPECT().Unsubscribe().AnyTimes()
	chain.EXPECT().SubscribeNewBlock(gomock.Any()).Return(sub).AnyTimes()

	subsidiesCheckFactory := func(opera.Rules, StateReader, state.StateDB, types.Signer) utils.TransactionCheckFunc {
		return nil
	}

	// The bundle checker factory returns a function that always considers bundles executable.
	bundlesCheckFactory := func(opera.Rules, StateReader, state.StateDB) utils.TransactionCheckFunc {
		return func(tx *types.Transaction) bool {
			// all bundles are accepted
			return true
		}
	}

	poolConfig := DefaultTxPoolConfig
	poolConfig.Journal = ""
	pool := newTxPool(poolConfig, chainConfig, chain, subsidiesCheckFactory, bundlesCheckFactory)

	// Create and add bundle transactions from several senders.
	const numSenders = 3
	const txsPerSender = 2
	keys := make([]*ecdsa.PrivateKey, numSenders)
	for i := range keys {
		key, err := crypto.GenerateKey()
		require.NoError(err)
		keys[i] = key
	}

	for _, key := range keys {
		for nonce := range txsPerSender {
			tx := bundleTx(uint64(nonce), key)
			err := pool.AddLocal(tx)
			require.NoError(err, "failed to add bundle transaction to the pool")
		}
	}

	pending, err := pool.Pending(false)
	require.NoError(err)
	require.Len(pending, numSenders)

	total := 0
	for addr, txs := range pending {
		fmt.Println("yup")
		require.Len(txs, txsPerSender, "unexpected tx count for %s", addr)
		for _, tx := range txs {
			require.True(bundle.IsEnvelope(tx), "expected bundle envelope")
		}
		total += len(txs)
	}
	require.Equal(numSenders*txsPerSender, total)
}

// ========================== Tools ===========================

type dummySubscription struct{}

func (d *dummySubscription) Err() <-chan error {
	return make(chan error)
}

func (d *dummySubscription) Unsubscribe() {
}

var _ event.Subscription = (*dummySubscription)(nil)

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
