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

// Package chain holds the head the RPC module serves: the one "latest" of
// every request, the feeds of new blocks and pool transactions, and the
// resolution of the block references clients send.
package chain

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
)

// Queue sizes of the hubs' consumers: blocks, and batches of pool
// transactions.
const (
	headQueue = 1024
	txsQueue  = 4096
)

// lagInterval is how often the head loop updates rpc_head_lag while no block
// arrives.
const lagInterval = time.Second

// Config holds the limits of resolving a block reference.
type Config struct {
	// HistoryPruningCutoff is the first block the node keeps; earliest
	// resolves to it.
	HistoryPruningCutoff uint64
}

// DefaultConfig is the one definition of the pruning cutoff's default.
func DefaultConfig() Config {
	return Config{HistoryPruningCutoff: 0}
}

// Source is what the chain reads of the node.
type Source interface {
	core.HeadSource
	ChainInfo() core.ChainInfo
	BlockNumberByHash(hash common.Hash) (uint64, error)
	SubscribeTxs(ch chan<- evmcore.NewTxsNotify) core.Subscription
	Progress() core.Progress
}

// Head is the newest archived block the module received. Every tag of one
// request resolves to it, for state, blocks, receipts, logs and the block
// number alike.
type Head struct {
	Block *evmcore.EvmBlock
}

// Number is the number of the head block.
func (h *Head) Number() uint64 {
	return h.Block.NumberU64()
}

// Chain serves the head and its feeds.
type Chain struct {
	src Source
	cfg Config
	d   *dispatch.Dispatcher

	head  atomic.Pointer[Head]
	info  atomic.Pointer[core.ChainInfo]
	heads *Hub[core.HeadEvent]
	txs   *Hub[[]*types.Transaction]

	headRuns, poolRuns atomic.Int64

	served prometheus.Gauge
	lag    prometheus.Gauge
}

// New creates the chain. Start must run before the first request.
func New(src Source, cfg Config, d *dispatch.Dispatcher) *Chain {
	c := &Chain{
		src:   src,
		cfg:   cfg,
		d:     d,
		heads: NewHub[core.HeadEvent](headQueue),
		txs:   NewHub[[]*types.Transaction](txsQueue),
	}
	reg := d.Metrics().Registerer()
	c.served = dispatch.Register(reg, prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "rpc_head_served", Help: "Number of the head block the RPC module serves.",
	}))
	c.lag = dispatch.Register(reg, prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "rpc_head_lag", Help: "Committed head minus the head the RPC module serves, in blocks.",
	}))
	return c
}

// Start reads the head and the chain's facts, and starts the head and pool
// loops.
func (c *Chain) Start() error {
	head, err := c.src.Head()
	if err == nil && head == nil {
		err = core.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("reading the head: %w", err)
	}
	if !c.refreshInfo() {
		return errors.New("reading the chain info: no chain id")
	}
	c.advance(head)
	c.d.Loop("head", c.headLoop)
	c.d.Loop("pool", c.poolLoop)
	return nil
}

// Latest is the head every tag of a request resolves to. A request reads it
// once and holds it.
func (c *Chain) Latest() *Head {
	return c.head.Load()
}

// Info is the chain's facts as of the served head. The head loop reads them
// with each new block, so that no request asks the node for them; an answer
// without a chain id keeps the last one.
func (c *Chain) Info() core.ChainInfo {
	return *c.info.Load()
}

func (c *Chain) refreshInfo() bool {
	info := c.src.ChainInfo()
	if info.ChainID == nil {
		return false
	}
	c.info.Store(&info)
	return true
}

// Heads is the feed of new archived blocks with their logs. A consumer sees a
// block before the head serves it.
func (c *Chain) Heads() *Hub[core.HeadEvent] {
	return c.heads
}

// Txs is the feed of transactions entering the pool.
func (c *Chain) Txs() *Hub[[]*types.Transaction] {
	return c.txs
}

// headLoop takes its subscription and the head itself on every run: what an
// earlier run took was released by its panic. A restarted run lost the
// blocks that arrived in between, so it drops every consumer first.
func (c *Chain) headLoop(ctx context.Context) {
	if c.headRuns.Add(1) > 1 {
		c.heads.DropAll()
	}
	events := make(chan core.HeadEvent, headQueue)
	sub := c.src.SubscribeHead(events)
	defer sub.Unsubscribe()
	if head, err := c.src.Head(); err == nil && head != nil {
		// Blocks that arrived before the subscription reach no consumer; a
		// consumer that would miss them is dropped.
		if cur := c.head.Load(); cur != nil && head.NumberU64() > cur.Number() {
			c.heads.DropAll()
		}
		c.refreshInfo()
		c.advance(head)
	} else {
		log.Warn("RPC head not readable, serving the last one", "err", err)
	}
	c.updateLag()

	ticker := time.NewTicker(lagInterval)
	defer ticker.Stop()
	feed := sub.Err()
	for {
		select {
		case ev := <-events:
			c.publish(ev)
		case err, ok := <-feed:
			// The node's feed ended; the last head is still served.
			log.Warn("RPC head subscription ended, serving the last head", "head", c.Latest().Number(), "err", err, "closed", !ok)
			feed = nil
		case <-ticker.C:
			if lag := c.updateLag(); feed == nil && lag > 0 {
				log.Warn("RPC head lags behind the committed head", "head", c.Latest().Number(), "lag", lag)
			}
		case <-ctx.Done():
			return
		}
	}
}

// publish hands a new block to the consumers, then serves it as the head. The
// head never goes back; a block at or below it is not published again.
func (c *Chain) publish(ev core.HeadEvent) {
	if ev.Block == nil {
		return
	}
	if cur := c.head.Load(); cur != nil && ev.Block.NumberU64() <= cur.Number() {
		return
	}
	c.heads.Publish(ev)
	c.refreshInfo()
	c.advance(ev.Block)
	c.updateLag()
}

func (c *Chain) advance(block *evmcore.EvmBlock) {
	for {
		cur := c.head.Load()
		if cur != nil && block.NumberU64() <= cur.Number() {
			return
		}
		if c.head.CompareAndSwap(cur, &Head{Block: block}) {
			c.served.Set(float64(block.NumberU64()))
			return
		}
	}
}

// Lag is how many blocks the served head is behind the committed one.
func (c *Chain) Lag() uint64 {
	return c.updateLag()
}

func (c *Chain) updateLag() uint64 {
	committed := c.src.Progress().CurrentBlock
	served := c.Latest().Number()
	var lag uint64
	if committed > served {
		lag = committed - served
	}
	c.lag.Set(float64(lag))
	return lag
}

// poolLoop relays the pool's feed; a restarted run drops every consumer, as
// the head loop does.
func (c *Chain) poolLoop(ctx context.Context) {
	if c.poolRuns.Add(1) > 1 {
		c.txs.DropAll()
	}
	events := make(chan evmcore.NewTxsNotify, txsQueue)
	sub := c.src.SubscribeTxs(events)
	defer sub.Unsubscribe()
	feed := sub.Err()
	for {
		select {
		case ev := <-events:
			c.txs.Publish(ev.Txs)
		case err, ok := <-feed:
			log.Warn("RPC pool subscription ended", "err", err, "closed", !ok)
			feed = nil
		case <-ctx.Done():
			return
		}
	}
}

// Sentinel errors of Resolve; a method maps them to the wording of its legacy
// counterpart.
var (
	// ErrNoSelector is a reference with neither a number nor a hash.
	ErrNoSelector = errors.New("unknown header selector")
	// ErrUnknownHash is a reference to a hash the node does not know.
	ErrUnknownHash = errors.New("block not found")
	// ErrInvalidTag is a negative number that is not a tag.
	ErrInvalidTag = errors.New("invalid block tag")
)

// Resolve turns a reference into a block number against the head the request
// holds. Tags resolve to the head, earliest to the pruning cutoff; a number
// is returned as it is, above the head too.
func (c *Chain) Resolve(head *Head, ref BlockRef) (uint64, error) {
	switch {
	case ref.Number != nil:
		return c.ResolveNumber(head, *ref.Number)
	case ref.Hash != nil:
		n, err := c.src.BlockNumberByHash(*ref.Hash)
		if errors.Is(err, core.ErrNotFound) {
			return 0, ErrUnknownHash
		}
		return n, err
	}
	return 0, ErrNoSelector
}

// ResolveNumber turns a block number or tag into a block number.
func (c *Chain) ResolveNumber(head *Head, bn BlockNumber) (uint64, error) {
	switch bn {
	case LatestBlockNumber, PendingBlockNumber, SafeBlockNumber, FinalizedBlockNumber:
		return head.Number(), nil
	case EarliestBlockNumber:
		return c.cfg.HistoryPruningCutoff, nil
	}
	if bn < 0 {
		return 0, ErrInvalidTag
	}
	return uint64(bn), nil
}
