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
}

// Chain follows the node's head and resolves block references.
type Chain struct {
	config  Config
	source  Source
	head    atomic.Pointer[evmcore.EvmBlock]
	metrics *metrics
}

// New creates a chain without a head; Start reads the first one.
func New(config Config, source Source, registerer prometheus.Registerer) *Chain {
	return &Chain{config: config, source: source, metrics: newMetrics(registerer)}
}

// Start reads the node's head and follows it in a loop of the dispatcher.
func (c *Chain) Start(dispatcher *dispatch.Dispatcher) error {
	head, err := c.source.Head()
	if err != nil {
		return fmt.Errorf("failed to read the head: %w", err)
	}
	c.advance(head)
	dispatcher.Loop("head", c.follow)
	return nil
}

// Head is the served head. It is shared and must not be changed.
func (c *Chain) Head() *evmcore.EvmBlock {
	return c.head.Load()
}

// headBuffer is how many heads the node's feed may hand over before the loop
// has taken them.
const headBuffer = 16

// follow takes the heads of the node's subscription until the context ends.
// It subscribes before it reads the head, so that no head falls between. On
// the loss of the subscription the last head stays served.
func (c *Chain) follow(ctx context.Context) {
	events := make(chan core.HeadNotify, headBuffer)
	subscription := c.source.SubscribeHead(events)
	defer subscription.Unsubscribe()
	if head, err := c.source.Head(); err == nil {
		c.advance(head)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case err := <-subscription.Err():
			log.Warn("RPC head subscription lost, the served head stays", "head", c.Head().NumberU64(), "err", err)
			return
		case event := <-events:
			c.advance(event.Block)
		}
	}
}

// advance serves a new head, never going back.
func (c *Chain) advance(head *evmcore.EvmBlock) {
	for {
		current := c.head.Load()
		if current != nil && current.NumberU64() >= head.NumberU64() {
			return
		}
		if c.head.CompareAndSwap(current, head) {
			break
		}
	}
	c.metrics.observe(head.NumberU64(), c.source.Progress().CurrentBlock)
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

// ResolveNumber turns a block number or tag into a block number. There is no
// pending block: pending, safe and finalized are the head, as latest is.
func (c *Chain) ResolveNumber(head uint64, number BlockNumber) (uint64, error) {
	switch {
	case number >= 0:
		return uint64(number), nil
	case number == EarliestBlockNumber:
		return c.config.HistoryCutoff, nil
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
