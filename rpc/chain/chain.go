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

// Package chain holds the head the module serves: the newest archived block
// it received, the loops that follow the node's head and pool feeds and fan
// them out to bounded queues, the block references clients send, and the one
// way a reference becomes a block number.
package chain

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
)

// Config holds the settings of the chain.
type Config struct {
	// HistoryPruningCutoff is the first block the node keeps, the block
	// "earliest" names.
	HistoryPruningCutoff uint64
}

// DefaultConfig returns the default settings: no block is pruned.
func DefaultConfig() Config {
	return Config{}
}

// The capacities of the queues the loops read from the node and hand to each
// consumer. A consumer that falls further behind is dropped.
const (
	feedCapacity     = 1024
	consumerCapacity = 1024
)

// Source is what the chain reads from the node.
type Source interface {
	core.HeadSource
	ChainInfo() (core.ChainInfo, error)
	BlockNumberByHash(hash common.Hash) (uint64, error)
	SubscribeTransactions(events chan<- evmcore.NewTxsNotify) core.Subscription
	Progress() core.Progress
}

// Head is the block every tag of a request resolves to. Its block is shared
// and must not be modified.
type Head struct {
	Number uint64
	Block  *evmcore.EvmBlock
}

// Chain follows the node's head and pool. Its head only moves forward.
type Chain struct {
	config  Config
	source  Source
	metrics *dispatch.Metrics

	head      atomic.Pointer[Head]
	info      atomic.Pointer[core.ChainInfo]
	committed atomic.Uint64

	heads        *Hub[core.HeadEvent]
	transactions *Hub[evmcore.NewTxsNotify]
}

// New returns a chain over the source. Start reads its head.
func New(config Config, source Source, metrics *dispatch.Metrics) *Chain {
	return &Chain{
		config:       config,
		source:       source,
		metrics:      metrics,
		heads:        NewHub[core.HeadEvent](consumerCapacity),
		transactions: NewHub[evmcore.NewTxsNotify](consumerCapacity),
	}
}

// ErrNoHead is answered while the chain has not read its first head.
var ErrNoHead = errors.New("no head block available yet")

// Start reads the head and the facts of the chain, then starts the head and
// the pool loops on the dispatcher.
func (chain *Chain) Start(dispatcher *dispatch.Dispatcher) error {
	block, err := chain.source.Head()
	if err != nil {
		return fmt.Errorf("failed to read the head: %w", err)
	}
	if block == nil {
		return ErrNoHead
	}
	chain.advance(block)
	if err := chain.refreshInfo(); err != nil {
		return fmt.Errorf("failed to read the chain info: %w", err)
	}
	dispatcher.Loop("head", chain.followHeads())
	dispatcher.Loop("pool", chain.followPool())
	return nil
}

// Latest returns the head every tag resolves to.
func (chain *Chain) Latest() (Head, error) {
	head := chain.head.Load()
	if head == nil {
		return Head{}, ErrNoHead
	}
	return *head, nil
}

// ChainInfo returns the facts of the chain as last read by the head loop.
func (chain *Chain) ChainInfo() core.ChainInfo {
	info := chain.info.Load()
	if info == nil {
		return core.ChainInfo{}
	}
	return *info
}

// Health reports the served head and the committed head as last seen.
func (chain *Chain) Health() (served, committed uint64) {
	head := chain.head.Load()
	if head != nil {
		served = head.Number
	}
	return served, max(chain.committed.Load(), served)
}

// Heads is the hub of new archived blocks with their logs.
func (chain *Chain) Heads() *Hub[core.HeadEvent] {
	return chain.heads
}

// Transactions is the hub of transactions entering the pool.
func (chain *Chain) Transactions() *Hub[evmcore.NewTxsNotify] {
	return chain.transactions
}

// followHeads returns the run of the head loop. Each run subscribes and then
// reads the head itself.
func (chain *Chain) followHeads() func(ctx context.Context) error {
	runs := 0
	return func(ctx context.Context) error {
		runs++
		events := make(chan core.HeadEvent, feedCapacity)
		subscription := chain.source.SubscribeHead(events)
		defer subscription.Unsubscribe()
		chain.catchUp(runs > 1)
		for {
			select {
			case <-ctx.Done():
				return nil
			case event := <-events:
				chain.receive(event)
			case err := <-subscription.Err():
				served, _ := chain.Health()
				log.Warn("RPC head subscription ended, serving the last head", "head", served, "err", err)
				return fmt.Errorf("head subscription ended: %v", err)
			}
		}
	}
}

// catchUp reads the head once the loop subscribed. The blocks between the
// served head and a newer one, and those of the gap a restart leaves, were
// published to no consumer: every consumer is dropped, as one that fell
// behind.
func (chain *Chain) catchUp(restarted bool) {
	if restarted {
		chain.heads.DropAll(ErrFeedRestarted)
	}
	block, err := chain.source.Head()
	if err != nil || block == nil {
		return
	}
	if served, _ := chain.Health(); block.NumberU64() > served {
		chain.heads.DropAll(ErrFeedRestarted)
	}
	chain.advance(block)
	chain.observeCommitted()
}

// receive hands a new block to the consumers and only then serves it as the
// head, so that a consumer never misses what a request already sees.
func (chain *Chain) receive(event core.HeadEvent) {
	if event.Block == nil {
		return
	}
	chain.heads.Publish(event)
	chain.advance(event.Block)
	if err := chain.refreshInfo(); err != nil {
		log.Warn("RPC keeps the last chain info", "err", err)
	}
	chain.observeCommitted()
}

// observeCommitted reads the committed head and reports the lag of the
// served one behind it.
func (chain *Chain) observeCommitted() {
	chain.committed.Store(chain.source.Progress().CurrentBlock)
	served, committed := chain.Health()
	chain.metrics.HeadLag.Set(float64(committed - served))
}

// advance moves the head forward to the block; an older block is ignored.
func (chain *Chain) advance(block *evmcore.EvmBlock) {
	next := &Head{Number: block.NumberU64(), Block: block}
	for {
		current := chain.head.Load()
		if current != nil && current.Number >= next.Number {
			return
		}
		if chain.head.CompareAndSwap(current, next) {
			chain.metrics.HeadServed.Set(float64(next.Number))
			return
		}
	}
}

// refreshInfo reads the facts of the chain, keeping the last ones when the
// read fails.
func (chain *Chain) refreshInfo() error {
	info, err := chain.source.ChainInfo()
	if err != nil {
		return err
	}
	chain.info.Store(&info)
	return nil
}

// followPool returns the run of the pool loop, which hands the transactions
// entering the pool to the consumers.
func (chain *Chain) followPool() func(ctx context.Context) error {
	runs := 0
	return func(ctx context.Context) error {
		runs++
		events := make(chan evmcore.NewTxsNotify, feedCapacity)
		subscription := chain.source.SubscribeTransactions(events)
		defer subscription.Unsubscribe()
		if runs > 1 {
			chain.transactions.DropAll(ErrFeedRestarted)
		}
		for {
			select {
			case <-ctx.Done():
				return nil
			case event := <-events:
				chain.transactions.Publish(event)
			case err := <-subscription.Err():
				return fmt.Errorf("pool subscription ended: %v", err)
			}
		}
	}
}
