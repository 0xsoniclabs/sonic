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

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/gossip/evmstore"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/utils"
)

// newRPCCoreEnv returns a node with a block holding a transfer, and the
// node as the RPC module sees it.
func newRPCCoreEnv(t *testing.T) (*testEnv, *rpcCore, *types.Receipt) {
	t.Helper()
	env := newTestEnv(2, 1, t)
	t.Cleanup(func() { require.NoError(t, env.Close()) })
	receipts, err := env.ApplyTxs(time.Second, env.Transfer(1, 1, utils.ToFtm(1)))
	require.NoError(t, err)
	require.Len(t, receipts, 1)
	return env, env.RPCCore().(*rpcCore), receipts[0]
}

func TestRPCCore_ReadsTheChain(t *testing.T) {
	env, adapter, receipt := newRPCCoreEnv(t)
	number := receipt.BlockNumber.Uint64()

	block, err := adapter.Block(number)
	require.NoError(t, err)
	require.Len(t, block.Transactions, 1)
	header, err := adapter.Header(number)
	require.NoError(t, err)
	require.Equal(t, block.Hash, header.Hash)
	require.Equal(t, common.Hash{}, header.TxHash, "a header carries no transactions root")

	byHash, err := adapter.BlockNumberByHash(block.Hash)
	require.NoError(t, err)
	require.Equal(t, number, byHash)

	receipts, err := adapter.Receipts(number)
	require.NoError(t, err)
	require.Equal(t, receipt.TxHash, receipts[0].TxHash)

	transaction, at, index, err := adapter.TransactionByHash(receipt.TxHash)
	require.NoError(t, err)
	require.Equal(t, receipt.TxHash, transaction.Hash())
	require.Equal(t, number, at)
	require.Zero(t, index)

	rules, err := adapter.Rules(number)
	require.NoError(t, err)
	require.Equal(t, env.store.GetRules().NetworkID, rules.NetworkID)

	info, err := adapter.ChainInfo()
	require.NoError(t, err)
	require.Equal(t, env.store.GetRules().NetworkID, info.ChainID)
	require.NotEmpty(t, info.UpgradeHeights)

	state, err := adapter.State(number)
	require.NoError(t, err)
	defer state.Release()
	require.True(t, state.GetBalance(env.Address(1)).Sign() > 0)
}

func TestRPCCore_ABlockAboveTheCommittedHeadIsNotFound(t *testing.T) {
	_, adapter, _ := newRPCCoreEnv(t)
	above := adapter.committed() + 1
	_, err := adapter.Block(above)
	require.ErrorIs(t, err, core.ErrNotFound)
	_, err = adapter.Header(above)
	require.ErrorIs(t, err, core.ErrNotFound)
	_, err = adapter.Receipts(above)
	require.ErrorIs(t, err, core.ErrNotFound)
	_, err = adapter.State(above)
	require.ErrorIs(t, err, core.ErrNotFound)
	_, err = adapter.Rules(above + 1)
	require.ErrorIs(t, err, core.ErrNotFound)
	_, err = adapter.BlockNumberByHash(common.Hash{1})
	require.ErrorIs(t, err, core.ErrNotFound)
}

func TestRPCCore_TheBlockAfterTheHeadHasTheRulesOfTheOpenEpoch(t *testing.T) {
	env, adapter, _ := newRPCCoreEnv(t)
	rules, err := adapter.Rules(adapter.committed() + 1)
	require.NoError(t, err)
	require.Equal(t, env.store.GetRules().Name, rules.Name)
}

func TestRPCCore_AnUnpublishedTransactionIsNotFound(t *testing.T) {
	env, adapter, _ := newRPCCoreEnv(t)
	transaction := types.NewTx(&types.LegacyTx{Nonce: 77})
	env.store.evm.SetTx(transaction.Hash(), transaction)
	env.store.evm.SetTxPosition(transaction.Hash(), evmstore.TxPosition{Block: idx.Block(adapter.committed() + 1)})
	_, _, _, err := adapter.TransactionByHash(transaction.Hash())
	require.ErrorIs(t, err, core.ErrNotFound)
	_, _, _, err = adapter.TransactionByHash(common.Hash{2})
	require.ErrorIs(t, err, core.ErrNotFound)
}

func TestRPCCore_AnIndexWithoutBodyIsCorrupted(t *testing.T) {
	env, adapter, _ := newRPCCoreEnv(t)
	missing := common.Hash{3}
	env.store.evm.SetTxPosition(missing, evmstore.TxPosition{Block: 1})
	_, _, _, err := adapter.TransactionByHash(missing)
	require.ErrorContains(t, err, "transactions index is corrupted")
}

// TestRPCCore_WithoutTxIndexReceiptsAndLookupsFail holds the guard the
// legacy backend takes from svc.config.TxIndex: the module answers its error,
// not an empty result.
func TestRPCCore_WithoutTxIndexReceiptsAndLookupsFail(t *testing.T) {
	_, adapter, receipt := newRPCCoreEnv(t)
	adapter.config.TxIndex = false
	_, err := adapter.Receipts(receipt.BlockNumber.Uint64())
	require.EqualError(t, err, "transactions index is disabled (enable TxIndex and re-process the DAGs)")
	_, _, _, err = adapter.TransactionByHash(receipt.TxHash)
	require.EqualError(t, err, "transactions index is disabled (enable TxIndex and re-process the DAG)")
}

func TestRPCCore_HeadIsTheNewestArchivedBlock(t *testing.T) {
	_, adapter, receipt := newRPCCoreEnv(t)
	head, err := adapter.Head()
	require.NoError(t, err)
	require.GreaterOrEqual(t, head.NumberU64(), receipt.BlockNumber.Uint64())
	require.LessOrEqual(t, head.NumberU64(), adapter.committed())
}

func TestRPCCore_BundleInfo(t *testing.T) {
	env, adapter, _ := newRPCCoreEnv(t)
	_, err := adapter.BundleInfo(common.Hash{4})
	require.ErrorIs(t, err, core.ErrNotFound)
	plan := common.Hash{5}
	env.store.AddProcessedBundles(adapter.committed(), map[common.Hash]bundle.PositionInBlock{plan: {Offset: 2, Count: 3}})
	info, err := adapter.BundleInfo(plan)
	require.NoError(t, err)
	require.Equal(t, core.BundleInfo{BlockNumber: adapter.committed(), Offset: 2, Count: 3}, info)
}

func TestRPCCore_TheHeadFeedDeliversArchivedBlocksWithTheirLogs(t *testing.T) {
	env, adapter, _ := newRPCCoreEnv(t)
	events := make(chan core.HeadEvent, 16)
	subscription := adapter.SubscribeHead(events)
	defer subscription.Unsubscribe()
	receipts, err := env.ApplyTxs(time.Second, env.Transfer(1, 1, utils.ToFtm(1)))
	require.NoError(t, err)
	deadline := time.After(10 * time.Second)
	for {
		select {
		case event := <-events:
			if event.Block.NumberU64() == receipts[0].BlockNumber.Uint64() {
				return
			}
		case <-deadline:
			t.Fatal("the head feed did not deliver the block")
		}
	}
}

func TestRPCCore_AStoppedFeedAnswersAnEndedSubscription(t *testing.T) {
	env, adapter, _ := newRPCCoreEnv(t)
	env.feed.scope.Close()
	subscription := adapter.SubscribeHead(make(chan core.HeadEvent))
	select {
	case <-subscription.Err():
	default:
		t.Fatal("the subscription of a stopped feed has not ended")
	}
	subscription.Unsubscribe()
	require.Equal(t, endedSubscription{}, liveOrEnded(nil))
}

func TestRPCCore_ThePool(t *testing.T) {
	env, adapter, _ := newRPCCoreEnv(t)
	transfer := env.Transfer(1, 1, big.NewInt(1))
	require.NoError(t, adapter.SendTransaction(context.Background(), transfer))
	pending, err := adapter.PoolTransactions()
	require.NoError(t, err)
	require.Len(t, pending, 1)
	found, err := adapter.PoolTransaction(transfer.Hash())
	require.NoError(t, err)
	require.Equal(t, transfer.Hash(), found.Hash())
	_, err = adapter.PoolTransaction(common.Hash{6})
	require.ErrorIs(t, err, core.ErrNotFound)
	pendingCount, _ := adapter.PoolStats()
	require.Equal(t, 1, pendingCount)
	adapter.PoolContent()
	adapter.PoolContentFrom(env.Address(1))
	require.Len(t, adapter.PendingHeads(), 1)
	adapter.PoolNonce(env.Address(1))
	subscription := adapter.SubscribeTransactions(make(chan evmcore.NewTxsNotify, 1))
	subscription.Unsubscribe()
}

func TestRPCCore_NodeInfo(t *testing.T) {
	env, adapter, _ := newRPCCoreEnv(t)
	progress := adapter.Progress()
	require.Equal(t, uint64(env.store.GetEpoch()), progress.CurrentEpoch)
	require.Equal(t, adapter.committed(), progress.CurrentBlock)
	require.NotZero(t, progress.CurrentBlockTime)
	inputs := adapter.OracleInputs()
	require.NotNil(t, inputs.MinTip)
	require.Positive(t, inputs.TotalGasPowerLeft)
}

func TestRPCCore_FindLogsReadsTheIndex(t *testing.T) {
	_, adapter, receipt := newRPCCoreEnv(t)
	number := receipt.BlockNumber.Uint64()
	logs, err := adapter.FindLogs(context.Background(), number, number, [][]common.Hash{{common.BytesToHash(receipt.ContractAddress.Bytes())}}, 0)
	require.NoError(t, err)
	require.Empty(t, logs, "a plain transfer emits no log")
}
