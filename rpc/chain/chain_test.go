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
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/tests"
)

func newChain(t *testing.T, blocks int, mutate ...tests.Mutate) (*tests.FakeCore, *tests.Harness) {
	t.Helper()
	fc := tests.NewFakeCore()
	for range blocks {
		fc.AddBlock(nil)
	}
	fc.Publish(uint64(blocks))
	h := tests.NewHarness(t, fc, nil, mutate...)
	// Consumers that subscribe from here on see every block.
	tests.WaitFor(t, func() bool { return fc.HeadSubscribers() == 1 })
	return fc, h
}

func next[T any](t *testing.T, c *chain.Consumer[T]) T {
	t.Helper()
	select {
	case ev := <-c.Events():
		return ev
	case <-c.Dropped():
		t.Fatal("the consumer was dropped")
	case <-time.After(5 * time.Second):
		t.Fatal("no event")
	}
	var zero T
	return zero
}

func TestResolve_TurnsEveryReferenceIntoANumberOrASentinel(t *testing.T) {
	fc, h := newChain(t, 5, func(c *tests.Configs) { c.Chain.HistoryPruningCutoff = 2 })
	head := h.Chain.Latest()
	block3, err := fc.Block(3)
	require.NoError(t, err)

	tests := map[string]struct {
		ref  chain.BlockRef
		want uint64
		err  error
	}{
		"latest":          {ref: chain.NumberRef(chain.LatestBlockNumber), want: 5},
		"pending":         {ref: chain.NumberRef(chain.PendingBlockNumber), want: 5},
		"safe":            {ref: chain.NumberRef(chain.SafeBlockNumber), want: 5},
		"finalized":       {ref: chain.NumberRef(chain.FinalizedBlockNumber), want: 5},
		"earliest":        {ref: chain.NumberRef(chain.EarliestBlockNumber), want: 2},
		"number":          {ref: chain.NumberRef(1), want: 1},
		"above the head":  {ref: chain.NumberRef(100), want: 100},
		"hash":            {ref: chain.HashRef(block3.Hash), want: 3},
		"unknown hash":    {ref: chain.HashRef(common.Hash{1}), err: chain.ErrUnknownHash},
		"no selector":     {ref: chain.BlockRef{}, err: chain.ErrNoSelector},
		"invalid tag":     {ref: chain.NumberRef(-6), err: chain.ErrInvalidTag},
		"max uint64 cast": {ref: chain.NumberRef(castToBlockNumber(math.MaxUint64 - 1)), want: 5},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := h.Chain.Resolve(head, tc.ref)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestBlockNumber_ANumberAboveMaxInt64IsNotATag(t *testing.T) {
	var bn chain.BlockNumber
	require.Error(t, json.Unmarshal([]byte(`"0x8000000000000000"`), &bn))
	require.NoError(t, json.Unmarshal([]byte(`"0x7fffffffffffffff"`), &bn))
	require.Equal(t, chain.BlockNumber(math.MaxInt64), bn)
	// A uint64 cast to a block number above MaxInt64 becomes a tag: callers
	// keep block numbers as uint64.
	require.Equal(t, chain.LatestBlockNumber, castToBlockNumber(math.MaxUint64-1))
}

func TestBlockNumber_DecodesAsGoEthereum(t *testing.T) {
	tests := map[string]struct {
		in   string
		want chain.BlockNumber
		err  bool
	}{
		"latest":   {in: `"latest"`, want: chain.LatestBlockNumber},
		"earliest": {in: `"earliest"`, want: chain.EarliestBlockNumber},
		"pending":  {in: `"pending"`, want: chain.PendingBlockNumber},
		"hex":      {in: `"0x10"`, want: 16},
		"decimal":  {in: `16`, err: true},
		"null":     {in: `null`, err: true},
		"garbage":  {in: `"x"`, err: true},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var bn chain.BlockNumber
			err := json.Unmarshal([]byte(tc.in), &bn)
			if tc.err {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, bn)
		})
	}
	text, err := chain.BlockNumber(16).MarshalText()
	require.NoError(t, err)
	require.Equal(t, "0x10", string(text))
	require.Equal(t, "latest", chain.LatestBlockNumber.String())
	require.Equal(t, "<invalid -7>", chain.BlockNumber(-7).String())
}

func TestBlockRef_DecodesAsGoEthereum(t *testing.T) {
	hash := common.HexToHash("0x1234")
	tests := map[string]struct {
		in   string
		want chain.BlockRef
		err  bool
	}{
		"tag":         {in: `"latest"`, want: chain.LatestRef()},
		"number":      {in: `"0x2"`, want: chain.NumberRef(2)},
		"hash":        {in: `"` + hash.Hex() + `"`, want: chain.HashRef(hash)},
		"object":      {in: `{"blockNumber":"0x2"}`, want: chain.NumberRef(2)},
		"object hash": {in: `{"blockHash":"` + hash.Hex() + `","requireCanonical":true}`, want: chain.BlockRef{Hash: &hash, RequireCanonical: true}},
		"empty":       {in: `{}`, want: chain.BlockRef{}},
		"both":        {in: `{"blockNumber":"0x2","blockHash":"` + hash.Hex() + `"}`, err: true},
		"too high":    {in: `"0x8000000000000000"`, err: true},
		"bad hash":    {in: `"0x` + "zz" + hash.Hex()[4:] + `"`, err: true},
		"not a ref":   {in: `5`, err: true},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var ref chain.BlockRef
			err := json.Unmarshal([]byte(tc.in), &ref)
			if tc.err {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, ref)
		})
	}
	require.Equal(t, chain.LatestRef(), chain.OrLatest(nil))
	require.True(t, chain.NumberRef(chain.PendingBlockNumber).IsPending())
	require.Equal(t, "nil", chain.BlockRef{}.String())
}

func TestHead_ConsumersSeeABlockBeforeTheHeadServesIt(t *testing.T) {
	fc, h := newChain(t, 1)
	consumer := h.Chain.Heads().Subscribe()
	defer consumer.Close()
	block := fc.AddAndPublish(nil)
	h.WaitHead(block.NumberU64())
	select {
	case ev := <-consumer.Events():
		require.Equal(t, block.NumberU64(), ev.Block.NumberU64())
	default:
		t.Fatal("the head serves a block its consumers have not been handed")
	}
}

func TestHead_NeverGoesBack(t *testing.T) {
	fc, h := newChain(t, 3)
	h.WaitHead(3)
	consumer := h.Chain.Heads().Subscribe()
	defer consumer.Close()
	// Events at or below the head are neither served nor handed out again.
	for _, n := range []uint64{1, 3} {
		old, err := fc.Block(n)
		require.NoError(t, err)
		fc.EmitHead(core.HeadEvent{Block: old})
	}
	fc.EmitHead(core.HeadEvent{})
	fc.AddAndPublish(nil)
	ev := next(t, consumer)
	require.Equal(t, uint64(4), ev.Block.NumberU64())
	require.Equal(t, uint64(4), h.Chain.Latest().Number())
}

func TestHead_AStalledConsumerIsDroppedWhileTheHeadAdvances(t *testing.T) {
	fc, h := newChain(t, 0)
	stalled := h.Chain.Heads().Subscribe()
	reading := h.Chain.Heads().Subscribe()
	defer reading.Close()
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case <-reading.Events():
			case <-reading.Dropped():
				return
			case <-time.After(5 * time.Second):
				return
			}
		}
	}()
	const blocks = 1100 // above the queue of a consumer
	for range blocks {
		// One at a time, so that the reading consumer keeps up.
		h.WaitHead(fc.AddAndPublish(nil).NumberU64())
	}
	select {
	case <-stalled.Dropped():
	default:
		t.Fatal("a consumer that does not read is not dropped")
	}
	select {
	case <-reading.Dropped():
		t.Fatal("a consumer that reads is dropped")
	default:
	}
}

func TestHead_AConsumerThatMissedBlocksBeforeTheSubscriptionIsDropped(t *testing.T) {
	fc := tests.NewFakeCore()
	fc.AddBlock(nil)
	fc.Publish(1)
	release := fc.Gate("SubscribeHead")
	var consumer *chain.Consumer[core.HeadEvent]
	h := tests.NewHarness(t, fc, func(h *tests.Harness) {
		consumer = h.Chain.Heads().Subscribe()
	})
	// A block arrives while the head loop has not subscribed yet.
	fc.AddBlock(nil)
	fc.Publish(2)
	release()
	h.WaitHead(2)
	select {
	case <-consumer.Dropped():
	default:
		t.Fatal("a consumer that missed a block is not dropped")
	}
}

func TestHead_TheLastHeadIsKeptWhenTheFeedEnds(t *testing.T) {
	fc, h := newChain(t, 2)
	h.WaitHead(2)
	fc.EndFeeds()
	fc.AddBlock(nil)
	time.Sleep(20 * time.Millisecond)
	require.Equal(t, uint64(2), h.Chain.Latest().Number())
}

func TestHead_ARestartedLoopDropsItsConsumersAndServesAgain(t *testing.T) {
	fc := tests.NewFakeCore()
	fc.AddBlock(nil)
	fc.Publish(1)
	fc.SetMode("SubscribeHead", tests.Panic)
	var consumer *chain.Consumer[core.HeadEvent]
	h := tests.NewHarness(t, fc, func(h *tests.Harness) {
		consumer = h.Chain.Heads().Subscribe()
	})
	tests.WaitFor(t, func() bool { return fc.Calls("SubscribeHead") >= 1 })
	fc.SetMode("SubscribeHead", tests.Normal)
	tests.WaitFor(t, func() bool { return fc.HeadSubscribers() == 1 })
	select {
	case <-consumer.Dropped():
	default:
		t.Fatal("a consumer that missed the events of the restart is not dropped")
	}
	fc.AddAndPublish(nil)
	h.WaitHead(2)
	require.Equal(t, 1.0, tests.Scrape(t, h.Registry, "rpc_loop_restarts_total", map[string]string{"loop": "head"}))
}

func TestPool_RelaysTheTransactionsOfThePool(t *testing.T) {
	fc, h := newChain(t, 0)
	consumer := h.Chain.Txs().Subscribe()
	defer consumer.Close()
	tx := types.NewTx(&types.LegacyTx{Nonce: 1})
	tests.WaitFor(t, func() bool { return fc.Calls("SubscribeTxs") >= 1 })
	fc.EmitTxs(types.Transactions{tx})
	select {
	case txs := <-consumer.Events():
		require.Equal(t, tx.Hash(), txs[0].Hash())
	case <-time.After(5 * time.Second):
		t.Fatal("no transactions relayed")
	}
}

func TestPool_ARestartedLoopDropsItsConsumers(t *testing.T) {
	fc := tests.NewFakeCore()
	fc.SetMode("SubscribeTxs", tests.Panic)
	var consumer *chain.Consumer[[]*types.Transaction]
	tests.NewHarness(t, fc, func(h *tests.Harness) {
		consumer = h.Chain.Txs().Subscribe()
	})
	tests.WaitFor(t, func() bool { return fc.Calls("SubscribeTxs") >= 1 })
	fc.SetMode("SubscribeTxs", tests.Normal)
	tests.WaitFor(t, func() bool { return fc.Calls("SubscribeTxs") >= 2 })
	tests.WaitFor(t, func() bool {
		select {
		case <-consumer.Dropped():
			return true
		default:
			return false
		}
	})
}

func TestMetrics_HeadAndLag(t *testing.T) {
	fc, h := newChain(t, 3)
	h.WaitHead(3)
	fc.SetProgress(core.Progress{CurrentBlock: 7})
	require.Equal(t, uint64(4), h.Chain.Lag())
	require.Equal(t, 3.0, tests.Scrape(t, h.Registry, "rpc_head_served", nil))
	require.Equal(t, 4.0, tests.Scrape(t, h.Registry, "rpc_head_lag", nil))
}

func TestHub_DropAllTellsEveryConsumer(t *testing.T) {
	hub := chain.NewHub[int](1)
	a, b := hub.Subscribe(), hub.Subscribe()
	require.Equal(t, 2, hub.Len())
	hub.DropAll()
	for _, c := range []*chain.Consumer[int]{a, b} {
		select {
		case <-c.Dropped():
		default:
			t.Fatal("a consumer was not told")
		}
	}
	require.Zero(t, hub.Len())
	c := hub.Subscribe()
	c.Close()
	require.Zero(t, hub.Len())
}

func TestStart_FailsWithoutAHead(t *testing.T) {
	fc := tests.NewFakeCore()
	fc.SetMode("Head", tests.Fail)
	d := dispatch.New(dispatch.DefaultConfig(), dispatch.NewMetrics(prometheus.NewRegistry()))
	defer d.Stop(time.Second)
	require.ErrorIs(t, chain.New(fc, chain.DefaultConfig(), d).Start(), tests.ErrFake)
}

// castToBlockNumber is the cast of a uint64 to a block number the reference
// made, which turns a number above MaxInt64 into a tag.
func castToBlockNumber(n uint64) chain.BlockNumber {
	return chain.BlockNumber(n)
}
