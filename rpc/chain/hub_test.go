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
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/tests"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func requireEnded[Event any](t *testing.T, consumer *chain.Consumer[Event], err error) {
	t.Helper()
	select {
	case <-consumer.Ended():
	case <-time.After(time.Second):
		t.Fatal("the consumer did not end")
	}
	require.Equal(t, err, consumer.Err())
}

func TestHub_DropsAConsumerThatFallsBehind(t *testing.T) {
	hub := chain.NewHub[int]()
	slow, fast := hub.Subscribe(1), hub.Subscribe(3)
	require.Equal(t, 2, hub.Consumers())
	for event := range 3 {
		hub.Publish(event)
	}
	requireEnded(t, slow, chain.ErrFellBehind)
	for want := range 3 {
		require.Equal(t, want, <-fast.Events(), "the others get every event in order")
	}
	require.Nil(t, fast.Err())
	require.Equal(t, 1, hub.Consumers())

	fast.Unsubscribe()
	requireEnded(t, fast, nil)
	require.Zero(t, hub.Consumers())
	hub.Publish(3)
	fast.Unsubscribe()
}

func TestHub_DropsEveryConsumerWithTheReason(t *testing.T) {
	hub := chain.NewHub[int]()
	consumers := []*chain.Consumer[int]{hub.Subscribe(1), hub.Subscribe(1)}
	reason := errors.New("reason")
	hub.DropAll(reason)
	for _, consumer := range consumers {
		requireEnded(t, consumer, reason)
	}
}

// A head is handed to the consumers before it is served, so that a consumer
// has every head the chain serves after it subscribed.
func TestChain_HandsAHeadOverBeforeServingIt(t *testing.T) {
	fake := tests.NewFakeCore()
	fake.AddBlock()
	fake.AddBlock()
	served, _ := startChain(t, fake, chain.Config{})
	require.Eventually(t, func() bool { return fake.Calls("SubscribeHead") == 1 }, time.Second, time.Millisecond)
	consumer := served.SubscribeHeads(2)
	fake.SetHead(1)
	waitForHead(t, served, 1)
	select {
	case event := <-consumer.Events():
		require.Equal(t, uint64(1), event.Block.NumberU64())
	default:
		t.Fatal("the head was served before it was handed over")
	}
	fake.SetHead(1)
	fake.SetHead(2)
	require.Equal(t, uint64(2), (<-consumer.Events()).Block.NumberU64(), "an old head is not handed over again")
}

// A head the loop reads at its start that is newer than the served one ends
// the consumers, as the heads between were not handed over.
func TestChain_DropsTheConsumersWhenItsStartSkipsHeads(t *testing.T) {
	fake := tests.NewFakeCore()
	fake.AddBlock()
	open := fake.Gate("SubscribeHead")
	served, _ := startChain(t, fake, chain.Config{})
	consumer := served.SubscribeHeads(1)
	fake.SetHead(1) // before the loop subscribes
	open()
	waitForHead(t, served, 1)
	requireEnded(t, consumer, chain.ErrFeedLost)
}

func TestChain_DropsTheConsumersWhenItsFeedsEnd(t *testing.T) {
	fake := tests.NewFakeCore()
	served, _ := startChain(t, fake, chain.Config{})
	heads, transactions := served.SubscribeHeads(1), served.SubscribeTransactions(1)
	fake.StopFeeds()
	requireEnded(t, heads, chain.ErrFeedLost)
	requireEnded(t, transactions, chain.ErrFeedLost)
}

func TestChain_DropsTheConsumersOfALoopThatPanics(t *testing.T) {
	fake := tests.NewFakeCore()
	fake.AddBlock()
	served, _ := startChain(t, fake, chain.Config{})
	consumer := served.SubscribeHeads(1)
	fake.SetMode("Progress", tests.Panic)
	fake.SetHead(1)
	requireEnded(t, consumer, chain.ErrFeedLost)
	fake.SetMode("Progress", tests.Normal)
}

func TestChain_HandsOverTheTransactionsThatEnterThePool(t *testing.T) {
	fake := tests.NewFakeCore()
	served, _ := startChain(t, fake, chain.Config{})
	consumer := served.SubscribeTransactions(1)
	recipient := common.Address{0x42}
	transaction := tests.NewAccount().Sign(&types.LegacyTx{To: &recipient, Gas: 21000, GasPrice: big.NewInt(1)})
	fake.AddPoolTransaction(transaction)
	event := <-consumer.Events()
	require.Equal(t, transaction.Hash(), event.Txs[0].Hash())
}
