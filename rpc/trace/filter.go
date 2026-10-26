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

package trace

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"sync"

	"github.com/ethereum/go-ethereum/common"

	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/txtrace"
)

// FilterArgs selects the traces of trace_filter: a block range, senders and
// recipients, and a window of the matching traces.
type FilterArgs struct {
	FromAddress *[]common.Address `json:"fromAddress"`
	ToAddress   *[]common.Address `json:"toAddress"`
	FromBlock   *chain.BlockRef   `json:"fromBlock"`
	ToBlock     *chain.BlockRef   `json:"toBlock"`
	After       uint              `json:"after"`
	Count       uint              `json:"count"`
}

type filterArg struct {
	Args FilterArgs `rpc:"args"`
}

// filter replays the blocks of a range and answers their traces that match.
// Without a window the blocks are replayed in parallel, and the traces come
// in the order the replays finish.
func (a *api) filter(ctx context.Context, p filterArg) (any, error) {
	buf, err := newResultBuffer(a.env.Config().MaxResponseSize)
	if err != nil {
		return nil, err
	}
	head := a.env.Latest(ctx)
	from, to, fromAddrs, toAddrs, err := a.parseFilter(head, p.Args)
	if err != nil {
		return nil, err
	}
	if p.Args.Count == 0 && p.Args.After == 0 {
		err = a.filterParallel(ctx, head, from, to, fromAddrs, toAddrs, buf)
	} else {
		err = a.filterSerial(ctx, head, from, to, fromAddrs, toAddrs, p.Args, buf)
	}
	if err != nil {
		return nil, err
	}
	result, err := buf.result()
	if err != nil {
		return nil, err
	}
	return dispatch.Encoded(result), nil
}

// parseFilter resolves the range and refuses one wider than the limit before
// any block is read.
func (a *api) parseFilter(head *exec.Head, args FilterArgs) (from, to uint64, fromAddrs, toAddrs map[common.Address]struct{}, err error) {
	if args.FromBlock != nil {
		if from, err = a.env.Resolve(head, *args.FromBlock); err != nil {
			return 0, 0, nil, nil, err
		}
	}
	to = head.Number()
	if args.ToBlock != nil {
		if to, err = a.env.Resolve(head, *args.ToBlock); err != nil {
			return 0, 0, nil, nil, err
		}
	}
	if limit := a.env.Config().MaxTraceFilterRange; limit > 0 && to > from && to-from > limit {
		return 0, 0, nil, nil, fmt.Errorf("too wide blocks range, the limit is %d", limit)
	}
	set := func(addrs *[]common.Address) map[common.Address]struct{} {
		if addrs == nil {
			return nil
		}
		m := make(map[common.Address]struct{}, len(*addrs))
		for _, addr := range *addrs {
			m[addr] = struct{}{}
		}
		return m
	}
	return from, to, set(args.FromAddress), set(args.ToAddress), nil
}

// filterSerial replays the blocks in order and keeps the window of traces
// after After, up to Count, as the reference counted them.
func (a *api) filterSerial(ctx context.Context, head *exec.Head, from, to uint64, fromAddrs, toAddrs map[common.Address]struct{}, args FilterArgs, buf *resultBuffer) error {
	var added, counted uint
	for n := from; n <= to; n++ {
		traces, err := a.blockTraces(ctx, head, n, fromAddrs, toAddrs)
		if err != nil {
			return err
		}
		for _, trace := range traces {
			if counted >= args.After {
				if err := buf.add(&trace); err != nil {
					return err
				}
				added++
			}
			if added >= args.Count {
				return nil
			}
			counted++
		}
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	return nil
}

// filterParallel replays the blocks on one worker per CPU, under the
// request's group: a failing worker cancels the others.
func (a *api) filterParallel(ctx context.Context, head *exec.Head, from, to uint64, fromAddrs, toAddrs map[common.Address]struct{}, buf *resultBuffer) error {
	g, gctx := a.env.Dispatcher().Group(ctx)
	blocks := make(chan uint64, 1)
	results := make(chan []txtrace.ActionTrace, 1)
	var producers sync.WaitGroup
	producers.Add(1 + runtime.NumCPU())
	g.Go(func() error {
		defer producers.Done()
		defer close(blocks)
		for n := from; n <= to; n++ {
			select {
			case blocks <- n:
			case <-gctx.Done():
				return nil
			}
		}
		return nil
	})
	for range runtime.NumCPU() {
		g.Go(func() error {
			defer producers.Done()
			for n := range blocks {
				if gctx.Err() != nil {
					return nil
				}
				traces, err := a.blockTraces(gctx, head, n, fromAddrs, toAddrs)
				if err != nil {
					return err
				}
				if len(traces) == 0 {
					continue
				}
				select {
				case results <- traces:
				case <-gctx.Done():
					return nil
				}
			}
			return nil
		})
	}
	g.Go(func() error {
		producers.Wait()
		close(results)
		return nil
	})
	g.Go(func() error {
		for traces := range results {
			for _, trace := range traces {
				if err := buf.add(&trace); err != nil {
					return err
				}
			}
		}
		return nil
	})
	err := g.Wait()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return fmt.Errorf("timeout when replaying tx")
	}
	if err != nil {
		return err
	}
	return ctx.Err()
}

// blockTraces replays a block and returns its traces whose sender and
// recipient match.
func (a *api) blockTraces(ctx context.Context, head *exec.Head, n uint64, fromAddrs, toAddrs map[common.Address]struct{}) ([]txtrace.ActionTrace, error) {
	block, err := a.env.BlockAt(head, n)
	if err != nil {
		return nil, fmt.Errorf("cannot get block from db %v", n)
	}
	if block.Transactions.Len() == 0 {
		return []txtrace.ActionTrace{}, nil
	}
	traces, err := a.replayBlock(ctx, head, block, nil, nil)
	if err != nil {
		return nil, err
	}
	matching := make([]txtrace.ActionTrace, 0)
	for _, trace := range *traces {
		if trace.Action != nil && containsAddress(trace.Action.From, trace.Action.To, fromAddrs, toAddrs) {
			matching = append(matching, trace)
		}
	}
	return matching, nil
}

func containsAddress(from, to *common.Address, fromAddrs, toAddrs map[common.Address]struct{}) bool {
	if len(fromAddrs) > 0 {
		if from == nil {
			return false
		}
		if _, ok := fromAddrs[*from]; !ok {
			return false
		}
	}
	if len(toAddrs) > 0 {
		if to == nil {
			return false
		}
		if _, ok := toAddrs[*to]; !ok {
			return false
		}
	}
	return true
}

// resultBuffer builds the JSON list of traces within MaxResponseSize. As in
// the reference, a size of zero leaves no room for anything.
type resultBuffer struct {
	mu    sync.Mutex
	buf   bytes.Buffer
	limit int
}

func newResultBuffer(limit int) (*resultBuffer, error) {
	b := &resultBuffer{limit: limit}
	b.buf.Grow(max(0, min(limit, 10*1024)))
	if err := b.write([]byte("[")); err != nil {
		return nil, err
	}
	return b, nil
}

func (b *resultBuffer) add(v any) error {
	enc, err := json.Marshal(v)
	if err != nil {
		return err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.buf.Len() > 1 {
		if err := b.write([]byte(",")); err != nil {
			return err
		}
	}
	return b.write(enc)
}

func (b *resultBuffer) result() ([]byte, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.write([]byte("]")); err != nil {
		return nil, err
	}
	return b.buf.Bytes(), nil
}

func (b *resultBuffer) write(p []byte) error {
	if b.buf.Len()+len(p) > b.limit {
		return ErrResponseTooLarge
	}
	b.buf.Write(p)
	return nil
}
