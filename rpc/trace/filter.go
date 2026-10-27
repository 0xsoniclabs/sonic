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
	"context"
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

// filterArguments are the argument of trace_filter.
type filterArguments struct {
	FromAddress *[]common.Address     `json:"fromAddress"`
	ToAddress   *[]common.Address     `json:"toAddress"`
	FromBlock   *chain.BlockReference `json:"fromBlock"`
	ToBlock     *chain.BlockReference `json:"toBlock"`
	After       uint                  `json:"after"`
	Count       uint                  `json:"count"`
}

// filterRange is a parsed trace_filter: its blocks and its addresses.
type filterRange struct {
	first, last         uint64
	senders, recipients map[common.Address]struct{}
}

// filter answers the traces of a range of blocks whose sender or recipient
// the arguments name, the range within MaxTraceFilterRange (V2) and the
// answer within MaxResponseSize. Without a window (after, count) the blocks
// are replayed in parallel and the traces come in any order, as in the
// reference.
func (handler *handlers) filter(ctx context.Context, arguments dispatch.Arguments) (any, error) {
	filter := dispatch.Argument[filterArguments](arguments, 0)
	buffer, err := newResultBuffer(handler.environment.Config().MaxResponseSize)
	if err != nil {
		return nil, err
	}
	head, err := handler.environment.Latest(ctx)
	if err != nil {
		return nil, err
	}
	parsed, err := handler.parseFilter(head, filter)
	if err != nil {
		return nil, err
	}
	if filter.Count == 0 && filter.After == 0 {
		err = handler.filterInParallel(head, parsed, buffer)
	} else {
		err = handler.filterInSeries(head, parsed, filter, buffer)
	}
	if err != nil {
		return nil, err
	}
	result, err := buffer.result()
	if err != nil {
		return nil, err
	}
	return dispatch.Encoded(result), nil
}

func (handler *handlers) parseFilter(head exec.Head, filter filterArguments) (filterRange, error) {
	parsed := filterRange{last: head.Number}
	if filter.FromBlock != nil {
		first, err := handler.resolve(head, *filter.FromBlock)
		if err != nil {
			return filterRange{}, err
		}
		parsed.first = first
	}
	if filter.ToBlock != nil {
		last, err := handler.resolve(head, *filter.ToBlock)
		if err != nil {
			return filterRange{}, err
		}
		parsed.last = last
	}
	if limit := handler.environment.Config().MaxTraceFilterRange; limit > 0 && parsed.last > parsed.first && parsed.last-parsed.first > limit {
		return filterRange{}, fmt.Errorf("too wide blocks range, the limit is %d", limit)
	}
	parsed.senders = addressSet(filter.FromAddress)
	parsed.recipients = addressSet(filter.ToAddress)
	return parsed, nil
}

func addressSet(addresses *[]common.Address) map[common.Address]struct{} {
	if addresses == nil {
		return nil
	}
	set := make(map[common.Address]struct{}, len(*addresses))
	for _, address := range *addresses {
		set[address] = struct{}{}
	}
	return set
}

// filterInSeries replays the blocks in order and keeps the window of
// traces the arguments ask for.
func (handler *handlers) filterInSeries(head exec.Head, parsed filterRange, filter filterArguments, buffer *resultBuffer) error {
	var added, counted uint
	for number := parsed.first; number <= parsed.last; number++ {
		traces, err := handler.blockTraces(head, number, parsed)
		if err != nil {
			return err
		}
		for index := range traces {
			if counted >= filter.After {
				if err := buffer.add(&traces[index]); err != nil {
					return err
				}
				added++
			}
			if added >= filter.Count {
				return nil
			}
			counted++
		}
		if err := head.Context().Err(); err != nil {
			return err
		}
	}
	return nil
}

// filterInParallel replays the blocks with one child per CPU of the
// request's group (P2): a child that fails cancels the others.
func (handler *handlers) filterInParallel(head exec.Head, parsed filterRange, buffer *resultBuffer) error {
	group, ctx := handler.dispatcher.Group(head.Context())
	view := handler.environment.View(ctx, head.Head)
	blocks := make(chan uint64)
	var collecting sync.Mutex
	for range runtime.NumCPU() {
		group.Go(ctx, func(ctx context.Context) error {
			for number := range blocks {
				traces, err := handler.blockTraces(view, number, parsed)
				if err != nil {
					return err
				}
				if err := collect(&collecting, buffer, traces); err != nil {
					return err
				}
			}
			return nil
		})
	}
	feed(ctx, parsed, blocks)
	err := group.Wait()
	if errors.Is(head.Context().Err(), context.DeadlineExceeded) {
		return errors.New("timeout when replaying tx")
	}
	return err
}

// feed hands the blocks of the range to the children until the range or
// the request ends.
func feed(ctx context.Context, parsed filterRange, blocks chan<- uint64) {
	defer close(blocks)
	for number := parsed.first; number <= parsed.last; number++ {
		select {
		case blocks <- number:
		case <-ctx.Done():
			return
		}
	}
}

func collect(collecting *sync.Mutex, buffer *resultBuffer, traces []txtrace.ActionTrace) error {
	collecting.Lock()
	defer collecting.Unlock()
	for index := range traces {
		if err := buffer.add(&traces[index]); err != nil {
			return err
		}
	}
	return nil
}

// blockTraces replays a block and keeps the traces whose action's sender
// and recipient the filter allows.
func (handler *handlers) blockTraces(head exec.Head, number uint64, parsed filterRange) ([]txtrace.ActionTrace, error) {
	block, err := exec.NullIfAbsent(handler.environment.BlockAt(head, number))
	if err != nil {
		return nil, fmt.Errorf("cannot get block from db %v, error:%v", number, err.Error())
	}
	if block == nil {
		return nil, fmt.Errorf("cannot get block from db %v", number)
	}
	kept := make([]txtrace.ActionTrace, 0)
	if len(block.Transactions) == 0 {
		return kept, nil
	}
	traces, err := handler.replayBlock(head, block, nil, nil)
	if err != nil {
		return nil, err
	}
	for _, trace := range *traces {
		if trace.Action != nil && allowed(trace.Action.From, trace.Action.To, parsed) {
			kept = append(kept, trace)
		}
	}
	return kept, nil
}

func allowed(sender, recipient *common.Address, parsed filterRange) bool {
	return inSet(sender, parsed.senders) && inSet(recipient, parsed.recipients)
}

// inSet tells whether an address is in a set; an empty set allows any.
func inSet(address *common.Address, set map[common.Address]struct{}) bool {
	if len(set) == 0 {
		return true
	}
	if address == nil {
		return false
	}
	_, found := set[*address]
	return found
}
