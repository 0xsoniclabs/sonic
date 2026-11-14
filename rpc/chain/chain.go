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

// Package chain holds the head the RPC module serves and turns the block
// references of clients into block numbers.
//
// The served head is the newest archived block seen on the node's head
// subscription. A request reads it once and resolves every tag against it,
// so all parts of an answer, eth_blockNumber included, agree on one latest.
package chain

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/prometheus/client_golang/prometheus"
)

// Config holds the settings of the served chain.
type Config struct {
	// HistoryCutoff is the first block the node keeps, which earliest names.
	HistoryCutoff uint64
}

// The ways a block reference fails to name a block.
var (
	ErrMissingSelector = errors.New("unknown header selector")
	ErrUnknownHash     = errors.New("block not found")
	ErrInvalidTag      = errors.New("invalid block tag")
)

// Source is what the chain reads from the node.
type Source interface {
	core.HeadSource
	BlockNumberByHash(hash common.Hash) (uint64, error)
	Progress() core.Progress
	SubscribeTransactions(events chan<- evmcore.NewTxsNotify) core.Subscription
}

// Chain follows the node's head and resolves block references. It hands the
// heads and the transactions that enter the pool to consumers through hubs.
type Chain struct {
	config       Config
	source       Source
	head         atomic.Pointer[servedHead]
	heads        *Hub[core.HeadNotify]
	transactions *Hub[evmcore.NewTxsNotify]
	metrics      *metrics
}

// servedHead is a served head with the channel closed once a newer one is
// served.
type servedHead struct {
	block *evmcore.EvmBlock
	next  chan struct{}
}

// New creates a chain without a head; Start reads the first one.
func New(config Config, source Source, registerer prometheus.Registerer) *Chain {
	return &Chain{
		config:       config,
		source:       source,
		heads:        NewHub[core.HeadNotify](),
		transactions: NewHub[evmcore.NewTxsNotify](),
		metrics:      newMetrics(registerer),
	}
}

// Start reads the node's head and follows it and the pool's feed in loops of
// the dispatcher.
func (c *Chain) Start(dispatcher *dispatch.Dispatcher) error {
	head, err := c.source.Head()
	if err != nil {
		return fmt.Errorf("failed to read the head: %w", err)
	}
	c.advance(head)
	dispatcher.Loop("head", c.follow)
	dispatcher.Loop("pool feed", c.followPool)
	return nil
}

// SubscribeHeads adds a consumer of the heads served after the current one,
// with their logs, each handed over before it is served.
func (c *Chain) SubscribeHeads(buffer int) *Consumer[core.HeadNotify] {
	return c.heads.Subscribe(buffer)
}

// SubscribeTransactions adds a consumer of the transactions that enter the
// pool.
func (c *Chain) SubscribeTransactions(buffer int) *Consumer[evmcore.NewTxsNotify] {
	return c.transactions.Subscribe(buffer)
}

// Head is the served head. It is shared and must not be changed.
func (c *Chain) Head() *evmcore.EvmBlock {
	return c.head.Load().block
}

// NextHead is closed once a head newer than the served one is served.
func (c *Chain) NextHead() <-chan struct{} {
	return c.head.Load().next
}

// headBuffer is how many heads the node's feed may hand over before the loop
// has taken them.
const headBuffer = 16

// follow takes the heads of the node's subscription until the context ends.
// It subscribes before it reads the head, so that no head falls between. A
// head is handed to the consumers before it is served, so that what they
// tell agrees with the served head. Whenever consumers would see a gap they
// are dropped: when the head read at the start is newer than the served
// one, and when the loop ends, also by a panic. On the loss of the
// subscription the last head stays served.
func (c *Chain) follow(ctx context.Context) {
	defer c.heads.DropAll(ErrFeedLost)
	events := make(chan core.HeadNotify, headBuffer)
	subscription := c.source.SubscribeHead(events)
	defer subscription.Unsubscribe()
	if head, err := c.source.Head(); err == nil && head.NumberU64() > c.Head().NumberU64() {
		// The heads up to this one were not handed over.
		c.heads.dropAll(ErrFeedLost, func() { c.serve(head) })
		c.observe()
	}
	for {
		select {
		case <-ctx.Done():
			return
		case err := <-subscription.Err():
			log.Warn("RPC head subscription lost, the served head stays", "head", c.Head().NumberU64(), "err", err)
			return
		case event := <-events:
			if event.Block.NumberU64() > c.Head().NumberU64() {
				c.heads.publish(event, func() { c.serve(event.Block) })
				c.observe()
			}
		}
	}
}

// transactionsBuffer is how many batches of transactions the pool's feed may
// hand over before the loop has taken them.
const transactionsBuffer = 256

// followPool hands the transactions that enter the pool to the consumers
// until the context ends. When the loop ends, also by a panic, the consumers
// are dropped, as they would see a gap.
func (c *Chain) followPool(ctx context.Context) {
	defer c.transactions.DropAll(ErrFeedLost)
	events := make(chan evmcore.NewTxsNotify, transactionsBuffer)
	subscription := c.source.SubscribeTransactions(events)
	defer subscription.Unsubscribe()
	for {
		select {
		case <-ctx.Done():
			return
		case err := <-subscription.Err():
			log.Warn("RPC pool subscription lost", "err", err)
			return
		case event := <-events:
			c.transactions.Publish(event)
		}
	}
}

// advance serves a head newer than the served one and reports the lag.
func (c *Chain) advance(head *evmcore.EvmBlock) {
	c.serve(head)
	c.observe()
}

// serve makes a head newer than the served one the served one, and tells
// whoever waits for it. Start and then the head loop call it, one at a time.
func (c *Chain) serve(head *evmcore.EvmBlock) {
	current := c.head.Load()
	if current != nil && current.block.NumberU64() >= head.NumberU64() {
		return
	}
	c.head.Store(&servedHead{block: head, next: make(chan struct{})})
	if current != nil {
		close(current.next)
	}
}

// observe reports the served head and how far it trails the node's.
func (c *Chain) observe() {
	c.metrics.observe(c.Head().NumberU64(), c.source.Progress().CurrentBlock)
}

// Resolve turns a block reference into a block number: a tag against the
// head, a hash by its block. A number is answered as it is, also above the
// head.
func (c *Chain) Resolve(head uint64, reference BlockReference) (uint64, error) {
	switch {
	case reference.Number != nil:
		return c.ResolveNumber(head, *reference.Number)
	case reference.Hash != nil:
		number, err := c.source.BlockNumberByHash(*reference.Hash)
		if errors.Is(err, core.ErrNotFound) {
			return 0, ErrUnknownHash
		}
		return number, err
	default:
		return 0, ErrMissingSelector
	}
}

// HistoryCutoff is the first block the node keeps, which earliest names.
func (c *Chain) HistoryCutoff() uint64 {
	return c.config.HistoryCutoff
}

// ResolveNumber turns a block number or tag into a block number. There is no
// pending block: pending, safe and finalized are the head, as latest is.
func (c *Chain) ResolveNumber(head uint64, number BlockNumber) (uint64, error) {
	switch {
	case number >= 0:
		return uint64(number), nil
	case number == EarliestBlockNumber:
		return c.HistoryCutoff(), nil
	case number == LatestBlockNumber, number == PendingBlockNumber,
		number == SafeBlockNumber, number == FinalizedBlockNumber:
		return head, nil
	default:
		return 0, ErrInvalidTag
	}
}

// metrics tell how far the served head trails the node's.
type metrics struct {
	served prometheus.Gauge
	lag    prometheus.Gauge
}

func newMetrics(registerer prometheus.Registerer) *metrics {
	return &metrics{
		served: dispatch.Register(registerer, prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "rpc_head_served",
			Help: "Number of the head the RPC module serves.",
		})),
		lag: dispatch.Register(registerer, prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "rpc_head_lag",
			Help: "Blocks the served head trails the committed head by.",
		})),
	}
}

func (m *metrics) observe(served, committed uint64) {
	m.served.Set(float64(served))
	m.lag.Set(float64(committed) - float64(served))
}
