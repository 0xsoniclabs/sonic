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

package gossip

import (
	"context"
	"math/big"
	"testing"
	"time"

	carmen "github.com/0xsoniclabs/carmen/go/state"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/kvdb/flushable"
	"github.com/Fantom-foundation/lachesis-base/kvdb/memorydb"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/gossip/evmstore"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/iblockproc"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/rpc/core"
)

// newRPCCoreChain runs a node of one validator that has produced a block with
// a transfer, and answers the adapter over it with that transfer.
func newRPCCoreChain(t *testing.T) (*RPCCore, *testEnv, *types.Transaction) {
	t.Helper()
	env := newTestEnv(2, 1, t)
	t.Cleanup(func() { require.NoError(t, env.Close()) })
	transfer := env.Transfer(1, 1, big.NewInt(1))
	_, err := env.ApplyTxs(sameEpoch, transfer)
	require.NoError(t, err)
	return NewRPCCore(env.Service), env, transfer
}

// newRPCCoreStore answers the adapter over a bare in-memory store whose
// blocks and states a test sets itself.
func newRPCCoreStore(t *testing.T, config Config) (*RPCCore, *Store) {
	t.Helper()
	store, err := NewMemStore(t)
	require.NoError(t, err)
	return NewRPCCore(&Service{config: config, store: store, txpool: &dummyTxPool{}}), store
}

func TestRPCCore_ReadsTheChain(t *testing.T) {
	rpcCore, env, transfer := newRPCCoreChain(t)
	latest := uint64(env.store.GetLatestBlockIndex())
	_, number, index, err := rpcCore.TransactionByHash(transfer.Hash())
	require.NoError(t, err)
	require.Zero(t, index)

	block, err := rpcCore.Block(number)
	require.NoError(t, err)
	require.Equal(t, transfer.Hash(), block.Transactions[0].Hash())

	header, err := rpcCore.Header(number)
	require.NoError(t, err)
	require.Equal(t, block.Header().ToJson(nil), header.ToJson(nil), "the header is the block's, its transactions root included")

	found, err := rpcCore.BlockNumberByHash(block.Hash)
	require.NoError(t, err)
	require.Equal(t, number, found)

	receipts, err := rpcCore.Receipts(number)
	require.NoError(t, err)
	require.Equal(t, transfer.Hash(), receipts[0].TxHash)

	head, err := rpcCore.Head()
	require.NoError(t, err)
	require.Equal(t, latest, head.NumberU64())

	stateDB, err := rpcCore.State(number)
	require.NoError(t, err)
	stateDB.Release()

	info := rpcCore.ChainInfo()
	require.Equal(t, env.store.GetRules().NetworkID, info.ChainID)
	require.Equal(t, common.Hash(env.store.GetGenesisID()), info.GenesisID)
	require.Equal(t, env.store.GetUpgradeHeights(), info.UpgradeHeights)
}

func TestRPCCore_AnswersNotFoundAboveTheCommittedHead(t *testing.T) {
	rpcCore, env, _ := newRPCCoreChain(t)
	above := uint64(env.store.GetLatestBlockIndex()) + 1
	cases := map[string]func() error{
		"Header": func() error {
			_, err := rpcCore.Header(above)
			return err
		},
		"Block": func() error {
			_, err := rpcCore.Block(above)
			return err
		},
		"Receipts": func() error {
			_, err := rpcCore.Receipts(above)
			return err
		},
		"State": func() error {
			_, err := rpcCore.State(above)
			return err
		},
		"BlockNumberByHash": func() error {
			_, err := rpcCore.BlockNumberByHash(common.Hash{0x01})
			return err
		},
		"TransactionByHash": func() error {
			_, _, _, err := rpcCore.TransactionByHash(common.Hash{0x01})
			return err
		},
		"BundleInfo": func() error {
			_, err := rpcCore.BundleInfo(common.Hash{0x01})
			return err
		},
		"PoolTransaction": func() error {
			_, err := rpcCore.PoolTransaction(common.Hash{0x01})
			return err
		},
		"Event": func() error {
			_, err := rpcCore.Event(common.Hash{0x01})
			return err
		},
		"EventPayload": func() error {
			_, err := rpcCore.EventPayload(common.Hash{0x01})
			return err
		},
		"EpochState": func() error {
			_, err := rpcCore.EpochState(999)
			return err
		},
		"LiveEpochHeads": func() error {
			_, err := rpcCore.LiveEpochHeads(1)
			return err
		},
	}
	for name, lookup := range cases {
		t.Run(name, func(t *testing.T) {
			require.ErrorIs(t, lookup(), core.ErrNotFound)
		})
	}
}

func TestRPCCore_HidesATransactionOfAnUnpublishedBlock(t *testing.T) {
	rpcCore, store := newRPCCoreStore(t, Config{TxIndex: true})
	transaction := types.NewTx(&types.LegacyTx{Nonce: 1})
	store.evm.SetTx(transaction.Hash(), transaction)
	store.evm.SetTxPosition(transaction.Hash(), evmstore.TxPosition{Block: 42, BlockOffset: 7})
	setLatestBlockIndex(store, 41)
	_, _, _, err := rpcCore.TransactionByHash(transaction.Hash())
	require.ErrorIs(t, err, core.ErrNotFound)

	setLatestBlockIndex(store, 42)
	found, block, index, err := rpcCore.TransactionByHash(transaction.Hash())
	require.NoError(t, err)
	require.Equal(t, transaction.Hash(), found.Hash())
	require.Equal(t, uint64(42), block)
	require.Equal(t, uint64(7), index)
}

func TestRPCCore_ReportsACorruptedTransactionIndex(t *testing.T) {
	rpcCore, store := newRPCCoreStore(t, Config{TxIndex: true})
	store.evm.SetTxPosition(common.Hash{1}, evmstore.TxPosition{Block: 42, BlockOffset: 7})
	setLatestBlockIndex(store, 42)
	_, _, _, err := rpcCore.TransactionByHash(common.Hash{1})
	require.ErrorContains(t, err, "transactions index is corrupted (position without body)")
}

// One test per guard the node configuration puts on the calls.
func TestRPCCore_GuardsOfTheNodeConfiguration(t *testing.T) {
	rpcCore, _ := newRPCCoreStore(t, Config{TxIndex: false, EnableTestOnlyApi: false})
	_, err := rpcCore.Receipts(0)
	require.EqualError(t, err, "transactions index is disabled (enable TxIndex and re-process the DAGs)")
	_, _, _, err = rpcCore.TransactionByHash(common.Hash{})
	require.EqualError(t, err, "transactions index is disabled (enable TxIndex and re-process the DAG)")
	require.EqualError(t, rpcCore.SendTransactions(nil), "this feature is disabled")
	require.EqualError(t, rpcCore.EmitTransactions(nil), "this feature is disabled")
}

func TestRPCCore_TestOnlyCallsWhenEnabled(t *testing.T) {
	rpcCore, _ := newRPCCoreStore(t, Config{EnableTestOnlyApi: true})
	transaction := types.NewTx(&types.LegacyTx{Nonce: 1})
	require.NoError(t, rpcCore.SendTransactions(types.Transactions{transaction}))
	added, err := rpcCore.PoolTransaction(transaction.Hash())
	require.NoError(t, err)
	require.Equal(t, transaction.Hash(), added.Hash())
	require.EqualError(t, rpcCore.EmitTransactions(nil), "no emitters available to propose transactions")
}

// The rules and the epoch of a block are those of its epoch; above the
// committed head they are the open epoch's, even when the head seals an
// epoch and the next block runs under the rules of the epoch it opens.
func TestRPCCore_RulesAndEpochAboveTheHeadAreTheOpenEpochs(t *testing.T) {
	rpcCore, store := newRPCCoreStore(t, Config{})
	sealed, open := opera.FakeNetRules(opera.GetSonicUpgrades()), opera.FakeNetRules(opera.GetAllegroUpgrades())
	sealed.Name, open.Name = "sealed", "open"
	store.SetBlock(5, inter.NewBlockBuilder().WithNumber(5).WithEpoch(3).Build())
	setLatestBlockIndex(store, 5)
	store.SetHistoryBlockEpochState(3, iblockproc.BlockState{}, iblockproc.EpochState{Epoch: 3, Rules: sealed})
	store.SetBlockEpochState(iblockproc.BlockState{LastBlock: iblockproc.BlockCtx{Idx: 5}}, iblockproc.EpochState{Epoch: 4, Rules: open})

	rules, err := rpcCore.Rules(5)
	require.NoError(t, err)
	require.Equal(t, "sealed", rules.Name)
	rules, err = rpcCore.Rules(6)
	require.NoError(t, err)
	require.Equal(t, "open", rules.Name)

	epoch, err := rpcCore.EpochOf(5)
	require.NoError(t, err)
	require.Equal(t, uint64(3), epoch)
	epoch, err = rpcCore.EpochOf(6)
	require.NoError(t, err)
	require.Equal(t, uint64(4), epoch)

	setLatestBlockIndex(store, 7) // a committed block that is not stored
	_, err = rpcCore.Rules(6)
	require.ErrorIs(t, err, core.ErrNotFound)
	_, err = rpcCore.EpochOf(6)
	require.ErrorIs(t, err, core.ErrNotFound)

	store.SetBlock(6, inter.NewBlockBuilder().WithNumber(6).WithEpoch(9).Build())
	_, err = rpcCore.Rules(6)
	require.ErrorIs(t, err, core.ErrNotFound, "the epoch of the block has no stored state")
}

func TestRPCCore_ReadsTheConsensusState(t *testing.T) {
	rpcCore, store := newRPCCoreStore(t, Config{})
	live := iblockproc.EpochState{Epoch: 4}
	store.SetBlockEpochState(iblockproc.BlockState{LastBlock: iblockproc.BlockCtx{Idx: 9}}, live)
	store.SetHistoryBlockEpochState(3, iblockproc.BlockState{LastBlock: iblockproc.BlockCtx{Idx: 5}}, iblockproc.EpochState{Epoch: 3})

	snapshot := rpcCore.LiveEpochState()
	require.Equal(t, idx.Block(9), snapshot.BlockState.LastBlock.Idx)
	require.Equal(t, idx.Epoch(4), snapshot.EpochState.Epoch)

	history, err := rpcCore.EpochState(3)
	require.NoError(t, err)
	require.Equal(t, idx.Block(5), history.BlockState.LastBlock.Idx)
}

func TestRPCCore_ReadsTheHeadsOfTheOpenEpoch(t *testing.T) {
	rpcCore, env, _ := newRPCCoreChain(t)
	open := env.store.GetEpoch()
	heads, err := rpcCore.LiveEpochHeads(uint64(open))
	require.NoError(t, err)
	require.Equal(t, eventHashes(env.store.GetHeadsSlice(open)), heads)
	_, err = rpcCore.LiveEpochHeads(uint64(open) - 1)
	require.ErrorIs(t, err, core.ErrNotFound, "heads are kept for the open epoch only")
}

func TestRPCCore_FindsEvents(t *testing.T) {
	rpcCore, env, _ := newRPCCoreChain(t)
	var last *inter.EventPayload
	env.store.ForEachEpochEvent(env.store.GetEpoch(), func(event *inter.EventPayload) bool {
		last = event
		return true
	})
	require.NotNil(t, last)
	id := common.Hash(last.ID())
	require.Equal(t, []common.Hash{id}, rpcCore.FindEvents(uint64(last.Epoch()), uint32(last.Lamport()), id[8:12]))
	event, err := rpcCore.Event(id)
	require.NoError(t, err)
	require.Equal(t, last.ID(), event.ID())
	payload, err := rpcCore.EventPayload(id)
	require.NoError(t, err)
	require.Equal(t, last.ID(), payload.ID())
}

func TestRPCCore_AnswersBundleInfo(t *testing.T) {
	rpcCore, store := newRPCCoreStore(t, Config{})
	plan := common.Hash{0x0b}
	store.AddProcessedBundles(12, map[common.Hash]bundle.PositionInBlock{plan: {Offset: 2, Count: 3}})
	info, err := rpcCore.BundleInfo(plan)
	require.NoError(t, err)
	require.Equal(t, core.BundleInfo{BlockNumber: 12, Offset: 2, Count: 3}, info)
}

func TestRPCCore_DelegatesToThePool(t *testing.T) {
	rpcCore, env, _ := newRPCCoreChain(t)
	pool := env.txpool.(*dummyTxPool)
	first := env.Transfer(1, 1, big.NewInt(1))
	second := env.Transfer(1, 1, big.NewInt(1))
	events := make(chan evmcore.NewTxsNotify, 1)
	subscription := rpcCore.SubscribeTransactions(events)
	defer subscription.Unsubscribe()

	require.NoError(t, rpcCore.SendTransaction(context.Background(), second))
	pool.AddRemotes([]*types.Transaction{first})
	defer pool.Clear()

	found, err := rpcCore.PoolTransaction(first.Hash())
	require.NoError(t, err)
	require.Equal(t, first.Hash(), found.Hash())
	require.Equal(t, types.Transactions{first}, rpcCore.PoolNextTransactions(), "only the next of a sender")
	pending, queued := rpcCore.PoolStats()
	require.Equal(t, 2, pending)
	require.Zero(t, queued)
	require.Zero(t, rpcCore.PoolNonce(env.Address(1)))
	content, _ := rpcCore.PoolContent()
	require.Nil(t, content)
	contentFrom, _ := rpcCore.PoolContentFrom(env.Address(1))
	require.Nil(t, contentFrom)
	require.Equal(t, big.NewInt(0), rpcCore.OracleInputs().MinTip)
}

func TestRPCCore_OracleInputsReadTheStagedRules(t *testing.T) {
	rpcCore, env, _ := newRPCCoreChain(t)
	blockState, epochState := env.store.GetBlockEpochState()
	require.Equal(t, epochState.Rules.Name, rpcCore.OracleInputs().NextEpochRules.Name)

	staged := epochState.Rules.Copy()
	staged.Name = "staged"
	blockState.DirtyRules = &staged
	env.store.SetBlockEpochState(blockState, epochState)
	require.Equal(t, "staged", rpcCore.OracleInputs().NextEpochRules.Name)
}

func TestRPCCore_OracleInputsCountTheGasPowerOfTheValidators(t *testing.T) {
	rpcCore, _, _ := newRPCCoreChain(t)
	require.NotZero(t, rpcCore.OracleInputs().TotalGasPowerLeft)
}

func TestRPCCore_DeliversTheHeads(t *testing.T) {
	rpcCore, env, _ := newRPCCoreChain(t)
	heads := make(chan core.HeadNotify, 16)
	subscription := rpcCore.SubscribeHead(heads)
	defer subscription.Unsubscribe()

	_, err := env.ApplyTxs(sameEpoch, env.Transfer(1, 1, big.NewInt(1)))
	require.NoError(t, err)
	// Applying a transaction may produce several blocks; their heads arrive
	// in order, the last one being the committed head.
	latest := uint64(env.store.GetLatestBlockIndex())
	timeout := time.After(5 * time.Second)
	for {
		select {
		case head := <-heads:
			if head.Block.NumberU64() == latest {
				return
			}
			require.Less(t, head.Block.NumberU64(), latest)
		case <-timeout:
			t.Fatal("the committed head was not delivered")
		}
	}
}

func TestRPCCore_AStoppedFeedAnswersAnEndedSubscription(t *testing.T) {
	rpcCore, env, _ := newRPCCoreChain(t)
	env.feed.Stop()
	subscription := rpcCore.SubscribeHead(make(chan core.HeadNotify))
	_, open := <-subscription.Err()
	require.False(t, open)
	subscription.Unsubscribe()
}

func TestRPCCore_ReportsProgressAndPeers(t *testing.T) {
	rpcCore, env, _ := newRPCCoreChain(t)
	progress := rpcCore.Progress()
	require.Equal(t, uint64(env.store.GetLatestBlockIndex()), progress.CurrentBlock)
	require.Equal(t, uint64(env.store.GetEpoch()), progress.CurrentEpoch)
	require.Equal(t, progress.CurrentBlock, progress.HighestBlock)

	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	env.p2pServer = &p2p.Server{Config: p2p.Config{PrivateKey: key, MaxPeers: 1, NoDiscovery: true}}
	require.NoError(t, env.p2pServer.Start())
	defer env.p2pServer.Stop()
	require.Zero(t, rpcCore.PeerCount())
}

// The calls of core.Core the adapter does not answer yet.
func TestRPCCore_CallsNotServedYet(t *testing.T) {
	rpcCore, _ := newRPCCoreStore(t, Config{})
	_, err := rpcCore.Logs(context.Background(), core.LogQuery{})
	require.EqualError(t, err, "the RPC core does not serve Logs yet")
}

// On a node without an archive, as in validator mode, the head is the
// committed head and no state can be read.
func TestRPCCore_WithoutAnArchiveServesTheCommittedHeadAndNoState(t *testing.T) {
	config := MemTestStoreConfig(t.TempDir())
	config.EVM.StateDb.Archive = carmen.NoArchive
	store, err := NewStore(flushable.NewSyncedPool(memorydb.NewProducer(""), []byte{0}), config)
	require.NoError(t, err)
	require.NoError(t, store.evm.Open())
	t.Cleanup(func() { require.NoError(t, store.Close()) })
	store.SetBlock(0, inter.NewBlockBuilder().WithNumber(0).Build())
	store.SetBlock(1, inter.NewBlockBuilder().WithNumber(1).Build())
	setLatestBlockIndex(store, 1)
	rpcCore := NewRPCCore(&Service{store: store})

	head, err := rpcCore.Head()
	require.NoError(t, err)
	require.Equal(t, uint64(1), head.NumberU64())
	_, err = rpcCore.State(1)
	require.ErrorIs(t, err, core.ErrNoArchive)
	require.EqualError(t, err, evmstore.NoArchiveError.Error(), "the archive's words are kept")
}
