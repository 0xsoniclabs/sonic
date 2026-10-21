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

package dispatch

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/rpc/core"
)

// Head is the block the module serves as "latest". The header is shared by
// all requests: it must not be modified.
type Head struct {
	Number uint64
	Hash   common.Hash
	Header *evmcore.EvmHeader
}

// Chain is the module's view of the head. It defines "latest" as the newest
// archived block it saw on the head subscription, for everything the module
// serves, and resolves block tags once per request. The head only ever moves
// forward, whatever the order of arrival.
type Chain struct {
	src     core.Core
	cfg     Config
	metrics *Metrics

	head  atomic.Pointer[Head]
	heads *Hub[core.HeadEvent]
	txs   *Hub[evmcore.NewTxsNotify]
}

// NewChain creates a chain view; Start must be called before it is used.
func NewChain(src core.Core, cfg Config, metrics *Metrics) *Chain {
	if metrics == nil {
		metrics = NewMetrics(nil)
	}
	return &Chain{src: src, cfg: cfg, metrics: metrics, heads: NewHub[core.HeadEvent](), txs: NewHub[evmcore.NewTxsNotify]()}
}

// Core returns the node interface the chain view reads.
func (c *Chain) Core() core.Core { return c.src }

// Config returns the configuration of the module.
func (c *Chain) Config() Config { return c.cfg }

// Start takes the head snapshot and starts the loop that follows the head
// subscription. The snapshot is the synchronous Head() of the node, not an
// event the adapter pushes.
func (c *Chain) Start(d *Dispatcher) error {
	events := make(chan core.HeadEvent, 16)
	sub := c.src.SubscribeHead(events)
	block, err := c.src.Head()
	if err != nil {
		sub.Unsubscribe()
		return fmt.Errorf("failed to read the head: %w", err)
	}
	c.setHead(block)
	d.Loop("head", func(ctx context.Context) {
		if sub == nil {
			// The loop restarts after a panic: subscribe again and catch up
			// with the blocks missed meanwhile.
			events = make(chan core.HeadEvent, 16)
			sub = c.src.SubscribeHead(events)
			if block, err := c.src.Head(); err == nil {
				c.setHead(block)
			}
		}
		defer func() {
			sub.Unsubscribe()
			sub = nil
		}()
		c.follow(ctx, events, sub)
	})

	// The pool feed is followed the same way: one subscription to the node,
	// fanned out to bounded queues.
	txEvents := make(chan evmcore.NewTxsNotify, 256)
	txSub := c.src.SubscribeTxs(txEvents)
	d.Loop("pending-txs", func(ctx context.Context) {
		if txSub == nil {
			txEvents = make(chan evmcore.NewTxsNotify, 256)
			txSub = c.src.SubscribeTxs(txEvents)
		}
		defer func() {
			txSub.Unsubscribe()
			txSub = nil
		}()
		subErr := txSub.Err()
		for {
			select {
			case <-ctx.Done():
				return
			case ev := <-txEvents:
				c.txs.Publish(ev)
			case <-subErr:
				log.Warn("RPC pending transaction subscription ended")
				subErr = nil
			}
		}
	})
	return nil
}

// SubscribeTxs registers a consumer of the transactions entering the pool with
// a queue of the given size.
func (c *Chain) SubscribeTxs(size int) *Consumer[evmcore.NewTxsNotify] { return c.txs.Subscribe(size) }

func (c *Chain) follow(ctx context.Context, events <-chan core.HeadEvent, sub core.Subscription) {
	lag := time.NewTicker(time.Second)
	defer lag.Stop()
	subErr := sub.Err()
	for {
		select {
		case <-ctx.Done():
			return
		case ev := <-events:
			c.publish(ev)
		case err, ok := <-subErr:
			// The subscription ended. The module keeps serving the last head.
			if ok && err != nil {
				log.Warn("RPC head subscription ended", "err", err)
			} else {
				log.Warn("RPC head subscription ended")
			}
			subErr = nil
		case <-lag.C:
			c.updateLag()
		}
	}
}

// publish hands the event to every consumer without waiting for any of them,
// and then advances the served head: the head is never ahead of what the
// consumers have been offered.
func (c *Chain) publish(ev core.HeadEvent) {
	if ev.Block == nil {
		return
	}
	c.heads.Publish(ev)
	c.setHead(ev.Block)
}

func (c *Chain) setHead(block *evmcore.EvmBlock) {
	if block == nil || block.Number == nil {
		return
	}
	n := block.Number.Uint64()
	for {
		cur := c.head.Load()
		if cur != nil && cur.Number >= n {
			return
		}
		next := &Head{Number: n, Hash: block.Hash, Header: block.Header()}
		if c.head.CompareAndSwap(cur, next) {
			c.metrics.HeadServed.Set(float64(n))
			return
		}
	}
}

func (c *Chain) updateLag() {
	head := c.head.Load()
	if head == nil {
		return
	}
	committed := c.src.Progress().CurrentBlock
	lag := 0.0
	if committed > head.Number {
		lag = float64(committed - head.Number)
	}
	c.metrics.HeadLag.Set(lag)
}

// Head returns the served head. The result is a snapshot: take it once per
// request and use it for every part of the answer.
func (c *Chain) Head() Head {
	if h := c.head.Load(); h != nil {
		return *h
	}
	return Head{}
}

// Latest returns the number of the served head.
func (c *Chain) Latest() uint64 { return c.Head().Number }

// Earliest returns the first block that is not pruned.
func (c *Chain) Earliest() uint64 { return c.cfg.HistoryPruningCutoff }

// The errors of Resolve and Number. Callers that answer with their own message
// compare against them.
var (
	// ErrInvalidTag is returned for a negative block number that is no known tag.
	ErrInvalidTag = errors.New("invalid block number tag")
	// ErrNoSelector is returned for a reference that holds neither number nor hash.
	ErrNoSelector = errors.New("unknown header selector")
	// ErrUnknownHash is returned for a hash the node has no block for.
	ErrUnknownHash = errors.New("block not found")
)

// Number resolves a block number or tag against the given head: the latest,
// pending, safe and finalized tags are the head, earliest is the pruning
// cutoff and a number is itself. The result is not checked against the head.
func (c *Chain) Number(head Head, bn BlockNumber) (uint64, error) {
	switch {
	case bn.IsLatest():
		return head.Number, nil
	case bn == EarliestBlockNumber:
		return c.Earliest(), nil
	case bn < 0:
		return 0, ErrInvalidTag
	}
	return uint64(bn), nil
}

// Resolve resolves the block a reference names to its number: a number or tag
// as Number does, a hash to the number of its block. The number comes first if
// the reference holds both, which its JSON form does not allow. The result is
// not checked against the head. A reference without number and hash gives
// ErrNoSelector, a hash of no known block ErrUnknownHash.
func (c *Chain) Resolve(head Head, ref BlockRef) (uint64, error) {
	switch {
	case ref.Number != nil:
		return c.Number(head, *ref.Number)
	case ref.Hash != nil:
		n, ok := c.src.BlockNumberByHash(*ref.Hash)
		if !ok {
			return 0, ErrUnknownHash
		}
		return n, nil
	}
	return 0, ErrNoSelector
}

// HeadConsumer receives the blocks that become the head, with their logs. The
// queue is bounded: a consumer that falls behind is dropped, never waited for.
type HeadConsumer = Consumer[core.HeadEvent]

// SubscribeHeads registers a consumer of the head with a queue of the given size.
func (c *Chain) SubscribeHeads(size int) *HeadConsumer { return c.heads.Subscribe(size) }
