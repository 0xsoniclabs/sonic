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

package dispatch_test

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/tests"
)

func TestChain_HeadSnapshotAndFollowing(t *testing.T) {
	fc := tests.NewFakeCore(t, tests.WithBlocks(3))
	h := tests.NewHarness(t, fc, nil)
	require.EqualValues(t, 3, h.Chain.Latest())
	require.Equal(t, fc.BlockAt(3).Hash, h.Chain.Head().Hash)

	h.WaitHead(fc.AddBlock().NumberU64())
}

func TestChain_HeadNeverMovesBackwards(t *testing.T) {
	fc := tests.NewFakeCore(t, tests.WithBlocks(5))
	h := tests.NewHarness(t, fc, nil)
	chain := h.Chain

	fc.AddBlock()
	h.WaitHead(fc.AddBlock().NumberU64()) // number 7

	// An old event does not rewind the head.
	publish(fc, fc.BlockAt(4))
	time.Sleep(20 * time.Millisecond)
	require.EqualValues(t, 7, chain.Latest())
	require.Equal(t, fc.BlockAt(7).Hash, chain.Head().Hash)
}

// publish sends a block on the head feed of the fake core again.
func publish(fc *tests.FakeCore, block *evmcore.EvmBlock) {
	fc.Publish(core.HeadEvent{Block: block})
}

func TestChain_Number(t *testing.T) {
	fc := tests.NewFakeCore(t, tests.WithBlocks(9))
	chain := tests.NewHarness(t, fc, nil).Chain
	head := chain.Head()
	for bn, want := range map[dispatch.BlockNumber]uint64{
		dispatch.LatestBlockNumber:    9,
		dispatch.PendingBlockNumber:   9,
		dispatch.SafeBlockNumber:      9,
		dispatch.FinalizedBlockNumber: 9,
		dispatch.EarliestBlockNumber:  0,
		4:                             4,
		100:                           100, // not checked against the head here
	} {
		got, err := chain.Number(head, bn)
		require.NoError(t, err, bn)
		require.Equal(t, want, got, bn)
	}
	_, err := chain.Number(head, dispatch.BlockNumber(-9))
	require.ErrorIs(t, err, dispatch.ErrInvalidTag)
}

func TestChain_Resolve(t *testing.T) {
	fc := tests.NewFakeCore(t, tests.WithBlocks(9))
	chain := tests.NewHarness(t, fc, nil).Chain
	head := chain.Head()

	for name, test := range map[string]struct {
		ref  dispatch.BlockRef
		want uint64
	}{
		"latest":                {dispatch.RefLatest(), 9},
		"earliest":              {dispatch.RefNumber(dispatch.EarliestBlockNumber), 0},
		"number":                {dispatch.RefNumber(4), 4},
		"number above the head": {dispatch.RefNumber(100), 100}, // not checked against the head here
		"hash":                  {dispatch.RefHash(fc.BlockAt(5).Hash), 5},
	} {
		got, err := chain.Resolve(head, test.ref)
		require.NoError(t, err, name)
		require.Equal(t, test.want, got, name)
	}

	_, err := chain.Resolve(head, dispatch.RefNumber(-9))
	require.ErrorIs(t, err, dispatch.ErrInvalidTag)
	_, err = chain.Resolve(head, dispatch.RefHash(common.Hash{9}))
	require.ErrorIs(t, err, dispatch.ErrUnknownHash)
	_, err = chain.Resolve(head, dispatch.BlockRef{})
	require.ErrorIs(t, err, dispatch.ErrNoSelector)
}

func TestChain_StalledConsumerIsDroppedAndTheHeadAdvances(t *testing.T) {
	fc := tests.NewFakeCore(t)
	h := tests.NewHarness(t, fc, nil)
	chain := h.Chain

	stalled := chain.SubscribeHeads(1) // never read
	healthy := chain.SubscribeHeads(64)
	defer healthy.Close()

	for range 10 {
		fc.AddBlock()
	}
	h.WaitHead(10)
	require.Eventually(t, stalled.Dropped, time.Second, time.Millisecond)
	require.False(t, healthy.Dropped())

	// The dropped consumer's queue is closed after what it had.
	var got int
	for range stalled.Events() {
		got++
	}
	require.LessOrEqual(t, got, 1)

	var seen []uint64
	for len(seen) < 10 {
		select {
		case ev := <-healthy.Events():
			seen = append(seen, ev.Block.NumberU64())
		case <-time.After(time.Second):
			t.Fatalf("healthy consumer saw only %v", seen)
		}
	}
	require.Equal(t, []uint64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, seen)
}

func TestChain_ConsumersSeeABlockBeforeTheHeadServesIt(t *testing.T) {
	fc := tests.NewFakeCore(t)
	h := tests.NewHarness(t, fc, nil)
	consumer := h.Chain.SubscribeHeads(8)
	defer consumer.Close()

	// The head moves only after the block was offered to the consumers: when
	// the head shows block n, the consumer's queue already holds it.
	for range 20 {
		h.WaitHead(fc.AddBlock().NumberU64())
		select {
		case ev := <-consumer.Events():
			require.EqualValues(t, fc.Latest(), ev.Block.NumberU64())
		default:
			t.Fatal("head advanced before the consumer was offered the block")
		}
	}
}

func TestChain_LossOfTheSubscriptionKeepsTheLastHead(t *testing.T) {
	fc := tests.NewFakeCore(t, tests.WithBlocks(2))
	chain := tests.NewHarness(t, fc, nil).Chain
	fc.EndHeadSubscriptions()
	time.Sleep(20 * time.Millisecond)
	require.EqualValues(t, 2, chain.Latest())
	require.Equal(t, fc.BlockAt(2).Hash, chain.Head().Hash)
}

func TestChain_RestartAfterAPanicFollowsTheHeadAgain(t *testing.T) {
	fc := tests.NewFakeCore(t, tests.WithBlocks(2))
	h := tests.NewHarness(t, fc, nil)
	// The lag of the head is measured every second; the measurement panics.
	fc.Inject("Progress", tests.Panic)
	restarted := func() bool { return tests.Counter(h.D.Metrics().LoopRestarts.WithLabelValues("head")) >= 1 }
	require.Eventually(t, restarted, 3*time.Second, time.Millisecond)
	fc.Inject("Progress", tests.Normal)

	// The restarted loop has a subscription of its own.
	require.Eventually(t, func() bool { return fc.Calls("SubscribeHead") >= 2 }, time.Second, time.Millisecond)
	h.WaitHead(fc.AddBlock().NumberU64())
	h.WaitHead(fc.AddBlock().NumberU64())
	require.EqualValues(t, 4, h.Chain.Latest())
	require.Equal(t, fc.BlockAt(4).Hash, h.Chain.Head().Hash)
}

func TestChain_StartFailsIfTheHeadIsUnavailable(t *testing.T) {
	fc := tests.NewFakeCore(t)
	fc.Inject("Head", tests.Fail)
	d := dispatch.New(dispatch.DefaultConfig(), dispatch.NewMetrics(nil))
	defer d.Stop(time.Second)
	chain := dispatch.NewChain(fc, dispatch.DefaultConfig(), d.Metrics())
	require.ErrorContains(t, chain.Start(d), "failed to read the head")
}
