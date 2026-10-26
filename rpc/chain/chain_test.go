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

package chain_test

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/tests"
)

func noMethods(*dispatch.Dispatcher, *exec.Environment) {}

// newChain starts a harness with no methods and returns it and its core,
// once the loops subscribed to the feeds.
func newChain(t *testing.T, mutations ...tests.Mutate) (*tests.Harness, *tests.FakeCore) {
	t.Helper()
	fake := tests.NewFakeCore(t)
	harness := tests.NewHarness(t, fake, noMethods, mutations...)
	tests.Eventually(t, func() bool { return fake.HeadSubscribers() == 1 && fake.PoolSubscribers() == 1 }, "the loops to subscribe")
	return harness, fake
}

func newRegistry() *prometheus.Registry {
	return prometheus.NewRegistry()
}

func newDispatcher(t *testing.T) *dispatch.Dispatcher {
	dispatcher := dispatch.New(dispatch.DefaultConfig(), dispatch.NewMetrics(newRegistry()))
	t.Cleanup(dispatcher.Stop)
	return dispatcher
}

func TestResolve_TurnsEveryReferenceIntoANumber(t *testing.T) {
	fake := tests.NewFakeCore(t)
	for range 4 {
		fake.AppendBlock(tests.BlockSpec{})
	}
	fake.PublishHead(3)
	unarchived := fake.AppendBlock(tests.BlockSpec{})
	harness := tests.NewHarness(t, fake, noMethods, func(configs *tests.Configs) {
		configs.Chain.HistoryPruningCutoff = 1
	})
	head, err := harness.Chain.Latest()
	require.NoError(t, err)
	require.Equal(t, uint64(3), head.Number)
	known, err := fake.Block(2)
	require.NoError(t, err)

	number := func(value chain.BlockNumber) chain.BlockReference { return chain.ReferenceToNumber(value) }
	tests := map[string]struct {
		reference chain.BlockReference
		number    uint64
		err       error
	}{
		"latest":                     {reference: number(chain.LatestBlockNumber), number: 3},
		"pending":                    {reference: number(chain.PendingBlockNumber), number: 3},
		"safe":                       {reference: number(chain.SafeBlockNumber), number: 3},
		"finalized":                  {reference: number(chain.FinalizedBlockNumber), number: 3},
		"earliest":                   {reference: number(chain.EarliestBlockNumber), number: 1},
		"number":                     {reference: number(2), number: 2},
		"the head":                   {reference: number(3), number: 3},
		"above the head":             {reference: number(4), err: core.ErrNotFound},
		"far above the head":         {reference: number(1 << 62), err: core.ErrNotFound},
		"hash":                       {reference: chain.ReferenceToHash(known.Hash), number: 2},
		"unknown hash":               {reference: chain.ReferenceToHash(common.HexToHash("0x01")), err: chain.ErrUnknownHash},
		"hash above the head":        {reference: chain.ReferenceToHash(unarchived.Hash), err: chain.ErrUnknownHash},
		"invalid tag":                {reference: number(-6), err: chain.ErrInvalidTag},
		"reference without selector": {err: chain.ErrNoSelector},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			resolved, err := harness.Chain.Resolve(head, test.reference)
			if test.err != nil {
				require.ErrorIs(t, err, test.err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.number, resolved)
		})
	}
}

func TestResolve_HandsOnAFailureOfTheNode(t *testing.T) {
	harness, fake := newChain(t)
	head, err := harness.Chain.Latest()
	require.NoError(t, err)
	fake.SetMode("BlockNumberByHash", tests.Fail)
	_, err = harness.Chain.Resolve(head, chain.ReferenceToHash(common.HexToHash("0x01")))
	require.ErrorIs(t, err, tests.ErrInjected)
}

func TestChain_ServesTheHeadTheNodePublishes(t *testing.T) {
	harness, fake := newChain(t)
	block := fake.AppendAndPublish(tests.BlockSpec{})
	harness.WaitForHead(block.NumberU64())
	head, err := harness.Chain.Latest()
	require.NoError(t, err)
	require.Equal(t, block.Hash, head.Block.Hash)
	require.Equal(t, float64(block.NumberU64()), tests.Sum(t, harness.Registry, "rpc_head_served"))
}

// TestChain_AHeadNewerThanTheServedOneDropsTheConsumers holds I18 for the
// blocks a run of the loop did not see published.
func TestChain_AHeadNewerThanTheServedOneDropsTheConsumers(t *testing.T) {
	harness, fake := newChain(t)
	consumer := harness.Chain.Heads().Subscribe()
	fake.SetMode("SubscribeHead", tests.Fail)
	fake.EndFeeds()
	tests.Eventually(t, func() bool { return consumer.Err() != nil }, "the consumer to be dropped")
	require.ErrorIs(t, consumer.Err(), chain.ErrFeedRestarted)
}

func TestChain_StartReadsTheHeadBeforeAnyEvent(t *testing.T) {
	fake := tests.NewFakeCore(t)
	fake.AppendBlock(tests.BlockSpec{})
	fake.PublishHead(1)
	harness := tests.NewHarness(t, fake, noMethods)
	head, err := harness.Chain.Latest()
	require.NoError(t, err)
	require.Equal(t, uint64(1), head.Number)
}

func TestChain_StartFailsWithoutAHead(t *testing.T) {
	fake := tests.NewFakeCore(t)
	fake.SetMode("Head", tests.Nil)
	served := chain.New(chain.DefaultConfig(), fake, dispatch.NewMetrics(newRegistry()))
	require.ErrorIs(t, served.Start(newDispatcher(t)), chain.ErrNoHead)
	_, err := served.Latest()
	require.ErrorIs(t, err, chain.ErrNoHead)

	fake.SetMode("Head", tests.Fail)
	require.ErrorIs(t, served.Start(newDispatcher(t)), tests.ErrInjected)
}

func TestChain_StartFailsWithoutTheChainInfo(t *testing.T) {
	fake := tests.NewFakeCore(t)
	fake.SetMode("ChainInfo", tests.Fail)
	served := chain.New(chain.DefaultConfig(), fake, dispatch.NewMetrics(newRegistry()))
	require.ErrorIs(t, served.Start(newDispatcher(t)), tests.ErrInjected)
	require.Equal(t, core.ChainInfo{}, served.ChainInfo())
}

func TestChain_TheHeadOnlyMovesForward(t *testing.T) {
	harness, fake := newChain(t)
	first := fake.AppendAndPublish(tests.BlockSpec{})
	second := fake.AppendAndPublish(tests.BlockSpec{})
	harness.WaitForHead(second.NumberU64())
	consumer := harness.Chain.Heads().Subscribe()
	defer consumer.Close()
	fake.SendHeadEvent(core.HeadEvent{Block: first})
	fake.SendHeadEvent(core.HeadEvent{})
	third := fake.AppendAndPublish(tests.BlockSpec{})
	harness.WaitForHead(third.NumberU64())
	head, err := harness.Chain.Latest()
	require.NoError(t, err)
	require.Equal(t, third.NumberU64(), head.Number)
}

// TestChain_ConsumersReceiveABlockBeforeTheHeadServesIt holds I14.
func TestChain_ConsumersReceiveABlockBeforeTheHeadServesIt(t *testing.T) {
	harness, fake := newChain(t)
	consumer := harness.Chain.Heads().Subscribe()
	defer consumer.Close()
	for range 20 {
		block := fake.AppendAndPublish(tests.BlockSpec{})
		harness.WaitForHead(block.NumberU64())
		select {
		case event := <-consumer.Events():
			require.Equal(t, block.NumberU64(), event.Block.NumberU64())
		default:
			t.Fatalf("block %d is served but its event was not queued", block.NumberU64())
		}
	}
}

// TestChain_AStalledConsumerIsDroppedWhileTheHeadAdvances holds I15.
func TestChain_AStalledConsumerIsDroppedWhileTheHeadAdvances(t *testing.T) {
	harness, fake := newChain(t)
	stalled := harness.Chain.Heads().Subscribe()
	reading := harness.Chain.Heads().Subscribe()
	defer reading.Close()
	received := make(chan uint64, 4096)
	readerDone := make(chan struct{})
	go func() {
		defer close(readerDone)
		for event := range reading.Events() {
			received <- event.Block.NumberU64()
		}
	}()
	// The blocks come in batches far below the queue of a consumer, each
	// read before the next comes, so that the reading consumer keeps up even
	// when its goroutine gets little time; the stalled one reads none.
	var last *evmcore.EvmBlock
	for batch := range 11 {
		for range 100 {
			last = fake.AppendAndPublish(tests.BlockSpec{})
		}
		tests.Eventually(t, func() bool { return len(received) == (batch+1)*100 }, "the reading consumer to get batch %d", batch)
	}
	harness.WaitForHead(last.NumberU64())
	tests.Eventually(t, func() bool { return stalled.Err() != nil }, "the stalled consumer to be dropped")
	require.ErrorIs(t, stalled.Err(), chain.ErrFellBehind)
	drained := 0
	for range stalled.Events() {
		drained++
	}
	require.Positive(t, drained, "the queue of the dropped consumer keeps what it held")
	tests.Eventually(t, func() bool { return len(received) == 1100 }, "the reading consumer to get every block")
	require.Nil(t, reading.Err())
}

// TestChain_ARestartedHeadLoopDropsItsConsumersAndWorks holds P3 and I18 for
// the head loop.
func TestChain_ARestartedHeadLoopDropsItsConsumersAndWorks(t *testing.T) {
	harness, fake := newChain(t)
	consumer := harness.Chain.Heads().Subscribe()
	before := fake.Calls("SubscribeHead")

	fake.SetMode("Head", tests.Panic)
	fake.EndFeeds()
	tests.Eventually(t, func() bool { return consumer.Err() != nil }, "the consumers to be dropped")
	require.ErrorIs(t, consumer.Err(), chain.ErrFeedRestarted)
	require.Positive(t, tests.Sum(t, harness.Registry, "rpc_loop_restarts_total"))

	fake.SetMode("Head", tests.Normal)
	tests.Eventually(t, func() bool { return fake.HeadSubscribers() == 1 }, "the head loop to subscribe again")
	require.Greater(t, fake.Calls("SubscribeHead"), before)
	after := harness.Chain.Heads().Subscribe()
	defer after.Close()
	block := fake.AppendAndPublish(tests.BlockSpec{})
	harness.WaitForHead(block.NumberU64())
	event := <-after.Events()
	require.Equal(t, block.NumberU64(), event.Block.NumberU64(), "a consumer that came after the restart gets every event")
}

// TestChain_KeepsServingTheLastHeadWhenTheFeedEnds holds I9.
func TestChain_KeepsServingTheLastHeadWhenTheFeedEnds(t *testing.T) {
	harness, fake := newChain(t)
	block := fake.AppendAndPublish(tests.BlockSpec{})
	harness.WaitForHead(block.NumberU64())
	fake.SetMode("SubscribeHead", tests.Fail)
	fake.EndFeeds()
	tests.Eventually(t, func() bool { return tests.Sum(t, harness.Registry, "rpc_loop_restarts_total") >= 1 }, "the head loop to restart")
	head, err := harness.Chain.Latest()
	require.NoError(t, err)
	require.Equal(t, block.NumberU64(), head.Number)
}

func TestChain_ReportsTheLagBehindTheCommittedHead(t *testing.T) {
	harness, fake := newChain(t)
	fake.AppendBlock(tests.BlockSpec{})
	fake.AppendBlock(tests.BlockSpec{})
	block := fake.AppendBlock(tests.BlockSpec{})
	fake.PublishHead(block.NumberU64() - 2)
	harness.WaitForHead(block.NumberU64() - 2)
	tests.Eventually(t, func() bool { return tests.Sum(t, harness.Registry, "rpc_head_lag") == 2 }, "the lag to be reported")
	served, committed := harness.Chain.Health()
	require.Equal(t, block.NumberU64()-2, served)
	require.Equal(t, block.NumberU64(), committed)
}

func TestChain_KeepsTheLastChainInfoWhenReadingItFails(t *testing.T) {
	harness, fake := newChain(t)
	info := harness.Chain.ChainInfo()
	require.Equal(t, uint64(tests.FakeChainID), info.ChainID)
	fake.SetMode("ChainInfo", tests.Fail)
	block := fake.AppendAndPublish(tests.BlockSpec{})
	harness.WaitForHead(block.NumberU64())
	require.Equal(t, info, harness.Chain.ChainInfo())
}

func TestChain_HandsPoolTransactionsToItsConsumers(t *testing.T) {
	harness, fake := newChain(t)
	consumer := harness.Chain.Transactions().Subscribe()
	defer consumer.Close()
	transaction := types.NewTx(&types.LegacyTx{Nonce: 7})
	fake.SendPoolTransactions(transaction)
	event := <-consumer.Events()
	require.Equal(t, transaction.Hash(), event.Txs[0].Hash())
}

// TestChain_ARestartedPoolLoopDropsItsConsumersAndWorks holds P3 and I18 for
// the pool loop.
func TestChain_ARestartedPoolLoopDropsItsConsumersAndWorks(t *testing.T) {
	harness, fake := newChain(t)
	consumer := harness.Chain.Transactions().Subscribe()
	fake.EndFeeds()
	tests.Eventually(t, func() bool { return consumer.Err() != nil }, "the consumers to be dropped")
	require.ErrorIs(t, consumer.Err(), chain.ErrFeedRestarted)
	tests.Eventually(t, func() bool { return fake.PoolSubscribers() == 1 }, "the pool loop to subscribe again")
	after := harness.Chain.Transactions().Subscribe()
	defer after.Close()
	transaction := types.NewTx(&types.LegacyTx{Nonce: 8})
	fake.SendPoolTransactions(transaction)
	event := <-after.Events()
	require.Equal(t, transaction.Hash(), event.Txs[0].Hash())
}
