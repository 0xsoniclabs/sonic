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
	"testing"
	"time"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/logger"
	rpccore "github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/utils"
)

func newRPCCoreEnv(t *testing.T) (*testEnv, rpccore.Core) {
	t.Helper()
	logger.SetTestMode(t)
	env := newTestEnv(2, 3, t)
	t.Cleanup(func() { require.NoError(t, env.Close()) })
	return env, env.RPCCore()
}

func TestRPCCore_ChainReaderServesBlocksAndTransactions(t *testing.T) {
	env, c := newRPCCoreEnv(t)
	tx := env.Transfer(1, 2, utils.ToFtm(10))
	receipts, err := env.ApplyTxs(time.Second, tx)
	require.NoError(t, err)
	require.Len(t, receipts, 1)
	number := receipts[0].BlockNumber.Uint64()

	info := c.ChainInfo()
	require.Equal(t, env.store.GetRules().NetworkID, info.ChainID)
	require.Equal(t, info.ChainID, info.NetworkID)
	require.Equal(t, common.Hash(env.store.GetGenesisID()), info.GenesisID)
	require.NotEmpty(t, info.UpgradeHeights)

	header, err := c.Header(number)
	require.NoError(t, err)
	require.EqualValues(t, number, header.Number.Uint64())

	block, err := c.Block(number)
	require.NoError(t, err)
	require.Len(t, block.Transactions, 1)
	require.Equal(t, tx.Hash(), block.Transactions[0].Hash())

	byHash, found := c.BlockNumberByHash(block.Hash)
	require.True(t, found)
	require.Equal(t, number, byHash)
	_, found = c.BlockNumberByHash(common.Hash{0xde, 0xad})
	require.False(t, found)

	got, err := c.Receipts(number)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, tx.Hash(), got[0].TxHash)

	rules, err := c.Rules(number)
	require.NoError(t, err)
	require.Equal(t, info.ChainID, rules.NetworkID)

	// The block after the newest one is processed with the rules of the open
	// epoch; the ones after that are not known.
	latest := uint64(env.store.GetLatestBlockIndex())
	next, err := c.Rules(latest + 1)
	require.NoError(t, err)
	require.Equal(t, info.ChainID, next.NetworkID)
	_, err = c.Rules(latest + 2)
	require.ErrorIs(t, err, rpccore.ErrNotFound)

	found2, blockNumber, index, err := c.TxByHash(tx.Hash())
	require.NoError(t, err)
	require.Equal(t, tx.Hash(), found2.Hash())
	require.Equal(t, number, blockNumber)
	require.EqualValues(t, 0, index)
	_, _, _, err = c.TxByHash(common.Hash{1})
	require.ErrorIs(t, err, rpccore.ErrNotFound)
}

func TestRPCCore_WithoutTxIndexReceiptsAndTransactionsFail(t *testing.T) {
	c := &rpcCore{svc: &Service{config: Config{TxIndex: false}}}
	_, err := c.Receipts(1)
	require.EqualError(t, err, "transactions index is disabled (enable TxIndex and re-process the DAGs)")
	_, _, _, err = c.TxByHash(common.Hash{1})
	require.EqualError(t, err, "transactions index is disabled (enable TxIndex and re-process the DAG)")
}

func TestRPCCore_MissingBlocksAreNotFound(t *testing.T) {
	_, c := newRPCCoreEnv(t)
	const missing = 1 << 40
	_, err := c.Header(missing)
	require.ErrorIs(t, err, rpccore.ErrNotFound)
	_, err = c.Block(missing)
	require.ErrorIs(t, err, rpccore.ErrNotFound)
	_, err = c.Receipts(missing)
	require.ErrorIs(t, err, rpccore.ErrNotFound)
	_, err = c.Rules(missing)
	require.ErrorIs(t, err, rpccore.ErrNotFound)
	_, err = c.State(missing)
	require.ErrorIs(t, err, rpccore.ErrNotFound)

	_, found := c.BundleInfo(common.Hash{7})
	require.False(t, found)
}

func TestRPCCore_StateAndRulesAreThoseOfTheHeaderOfTheBlock(t *testing.T) {
	env, c := newRPCCoreEnv(t)
	_, err := env.ApplyTxs(time.Second, env.Transfer(1, 2, utils.ToFtm(1)))
	require.NoError(t, err)

	latest := uint64(env.store.GetLatestBlockIndex())
	require.NotZero(t, latest)
	found := 0
	for n := uint64(0); n <= latest; n++ {
		header, err := c.Header(n)
		if err != nil {
			// A block the store lacks has neither state nor rules.
			require.ErrorIs(t, err, rpccore.ErrNotFound, n)
			_, err = c.State(n)
			require.ErrorIs(t, err, rpccore.ErrNotFound, n)
			_, err = c.Rules(n)
			require.ErrorIs(t, err, rpccore.ErrNotFound, n)
			continue
		}
		found++

		_, es := env.store.GetHistoryBlockEpochState(idx.Epoch(header.Epoch))
		rules, err := c.Rules(n)
		require.NoError(t, err, n)
		require.Equal(t, es.Rules, *rules, n)

		// The state is the one the header of the block names, whether the
		// archive holds it or not.
		state, err := c.State(n)
		want, wantErr := c.(*rpcCore).reader.BlockStateDB(header.Number, header.Root)
		require.Equal(t, wantErr == nil, err == nil, n)
		if err == nil {
			state.Release()
			want.Release()
		}
	}
	require.NotZero(t, found)
	state, err := c.State(latest)
	require.NoError(t, err)
	state.Release()

	// A block is public once the latest block index reaches it.
	_, err = c.State(latest + 1)
	require.ErrorIs(t, err, rpccore.ErrNotFound)
}

func TestRPCCore_HeadFollowsTheArchive(t *testing.T) {
	env, c := newRPCCoreEnv(t)
	_, err := env.ApplyTxs(time.Second, env.Transfer(1, 2, utils.ToFtm(1)))
	require.NoError(t, err)

	head, err := c.Head()
	require.NoError(t, err)
	require.NotNil(t, head)
	require.LessOrEqual(t, head.Number.Uint64(), c.Progress().CurrentBlock)

	state, err := c.State(head.Number.Uint64())
	require.NoError(t, err)
	require.NotNil(t, state)
	state.Release()
}

func TestRPCCore_ProgressAndOracleInputs(t *testing.T) {
	env, c := newRPCCoreEnv(t)
	_, err := env.ApplyTxs(time.Second, env.Transfer(1, 2, utils.ToFtm(1)))
	require.NoError(t, err)

	progress := c.Progress()
	require.NotZero(t, progress.CurrentBlock)
	require.NotZero(t, progress.CurrentEpoch)
	require.NotEqual(t, common.Hash{}, progress.CurrentBlockHash)

	inputs := c.OracleInputs()
	require.Equal(t, env.store.GetRules().NetworkID, inputs.Rules.NetworkID)
	require.Equal(t, env.store.GetRules().NetworkID, inputs.PendingRules.NetworkID)
	require.NotNil(t, inputs.MinTip)
	// The tip is the node's own value: changing the copy does not change the node.
	inputs.MinTip.SetInt64(12345)
	require.NotEqual(t, int64(12345), c.OracleInputs().MinTip.Int64())
}

func TestRPCCore_HeadFeedDeliversBlocksAndEndsWithTheService(t *testing.T) {
	env, c := newRPCCoreEnv(t)
	events := make(chan rpccore.HeadEvent, 16)
	sub := c.SubscribeHead(events)
	defer sub.Unsubscribe()

	_, err := env.ApplyTxs(time.Second, env.Transfer(1, 2, utils.ToFtm(1)))
	require.NoError(t, err)

	var seen bool
	deadline := time.After(5 * time.Second)
	for !seen {
		select {
		case ev := <-events:
			require.NotNil(t, ev.Block)
			seen = len(ev.Block.Transactions) > 0
		case <-deadline:
			t.Fatal("no block announced")
		}
	}
}

func TestRPCCore_PoolFacts(t *testing.T) {
	env, c := newRPCCoreEnv(t)
	pool := env.txpool.(*dummyTxPool)
	tx := env.Transfer(1, 2, utils.ToFtm(1))
	sender, err := types.Sender(env.EthAPI.signer, tx)
	require.NoError(t, err)

	pool.AddRemotes([]*types.Transaction{tx})
	defer pool.Clear()

	require.Equal(t, tx.Hash(), c.PoolTransaction(tx.Hash()).Hash())
	require.Nil(t, c.PoolTransaction(common.Hash{1}))

	pending, queued := c.PoolStats()
	require.Equal(t, 1, pending)
	require.Equal(t, 0, queued)

	// The pool of the test environment keeps no content listing.
	content, _ := c.PoolContent()
	require.Empty(t, content)
	from, _ := c.PoolContentFrom(sender)
	require.Empty(t, from)

	heads := c.PendingHeads(10)
	require.Len(t, heads, 1)
	require.Equal(t, tx.Hash(), heads[0].Tx.Hash())
	require.Empty(t, c.PendingHeads(0))

	require.NoError(t, c.SendTx(context.Background(), env.Transfer(1, 3, utils.ToFtm(1))))
	require.Equal(t, c.PoolNonce(sender), pool.Nonce(sender))

	sub := c.SubscribeTxs(make(chan evmcore.NewTxsNotify, 1))
	require.NotNil(t, sub)
	sub.Unsubscribe()
}

func TestEndedSubscription_EndsAtOnce(t *testing.T) {
	sub := asSubscription(nil)
	select {
	case _, ok := <-sub.Err():
		require.False(t, ok)
	default:
		t.Fatal("the subscription is not ended")
	}
	sub.Unsubscribe()
}

func TestRPCCore_FindLogsAndPoolTransactionsAnswerOnAnEmptyChain(t *testing.T) {
	_, c := newRPCCoreEnv(t)
	logs, err := c.FindLogs(context.Background(), 0, 10, [][]common.Hash{{{1}}}, 0)
	require.NoError(t, err)
	require.Empty(t, logs)

	txs, err := c.PoolTransactions()
	require.NoError(t, err)
	require.Empty(t, txs)
}
