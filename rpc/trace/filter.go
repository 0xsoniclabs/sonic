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

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/ethereum/go-ethereum/common"
)

// filterParams select the traces of a range of blocks by the addresses
// value leaves and reaches, and a window of them.
type filterParams struct {
	Filter filterArguments
}

type filterArguments struct {
	FromAddress *[]common.Address     `json:"fromAddress"`
	ToAddress   *[]common.Address     `json:"toAddress"`
	FromBlock   *chain.BlockReference `json:"fromBlock"`
	ToBlock     *chain.BlockReference `json:"toBlock"`
	After       uint                  `json:"after"`
	Count       uint                  `json:"count"`
}

// filter answers the traces of the blocks of a range that the addresses
// select, within MaxResponseSize. Without a window the blocks are replayed in
// parallel and their traces answered in the order they come, as the legacy
// handler does.
func (h *handlers) filter(ctx context.Context, params filterParams) (dispatch.Encoded, error) {
	arguments := params.Filter
	answer, err := newResultBuffer(h.environment.Config().MaxResponseSize)
	if err != nil {
		return nil, err
	}
	head := h.environment.Head()
	selection, err := h.parseFilter(head, arguments)
	if err != nil {
		return nil, err
	}
	if arguments.Count == 0 && arguments.After == 0 {
		err = h.filterInParallel(ctx, head, selection, answer)
	} else {
		err = h.filterInOrder(ctx, head, selection, arguments.After, arguments.Count, answer)
	}
	if err != nil {
		return nil, err
	}
	return answer.result()
}

// selection is a range of blocks and the addresses that select their traces.
type selection struct {
	from, to uint64
	senders  map[common.Address]bool
	targets  map[common.Address]bool
}

// parseFilter resolves the range, which ends at the served head unless
// named, and refuses one wider than MaxTraceFilterRange before any block is
// read.
func (h *handlers) parseFilter(head *evmcore.EvmBlock, arguments filterArguments) (selection, error) {
	selected := selection{to: head.NumberU64()}
	var err error
	if arguments.FromBlock != nil {
		if selected.from, err = h.environment.Resolve(head, *arguments.FromBlock); err != nil {
			return selection{}, err
		}
	}
	if arguments.ToBlock != nil {
		if selected.to, err = h.environment.Resolve(head, *arguments.ToBlock); err != nil {
			return selection{}, err
		}
	}
	if limit := h.environment.Config().MaxTraceFilterRange; limit > 0 && selected.to > selected.from && selected.to-selected.from > limit {
		return selection{}, fmt.Errorf("too wide blocks range, the limit is %d", limit)
	}
	selected.senders = addressSet(arguments.FromAddress)
	selected.targets = addressSet(arguments.ToAddress)
	return selected, nil
}

func addressSet(addresses *[]common.Address) map[common.Address]bool {
	if addresses == nil {
		return nil
	}
	set := map[common.Address]bool{}
	for _, address := range *addresses {
		set[address] = true
	}
	return set
}

// filterInOrder replays the blocks one after the other and keeps the
// traces after the first after ones, at most count of them unless zero.
func (h *handlers) filterInOrder(ctx context.Context, head *evmcore.EvmBlock, selected selection, after, count uint, answer *resultBuffer) error {
	var seen, added uint
	for number := selected.from; number <= selected.to; number++ {
		traces, err := h.selectedTraces(ctx, head, number, selected)
		if err != nil {
			return err
		}
		for _, trace := range traces {
			if seen >= after {
				if err := answer.add(&trace); err != nil {
					return err
				}
				added++
			}
			if count > 0 && added >= count {
				return nil
			}
			seen++
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
	return nil
}

// filterInParallel replays the blocks in a group of the dispatcher, a worker
// per processor, and adds the traces of each block as it comes.
func (h *handlers) filterInParallel(ctx context.Context, head *evmcore.EvmBlock, selected selection, answer *resultBuffer) error {
	group, workers := dispatch.NewGroup(ctx)
	var mutex sync.Mutex
	next := selected.from
	take := func() (uint64, bool) {
		mutex.Lock()
		defer mutex.Unlock()
		if next > selected.to {
			return 0, false
		}
		next++
		return next - 1, true
	}
	for range runtime.NumCPU() {
		group.Go(func() error {
			for number, more := take(); more && workers.Err() == nil; number, more = take() {
				traces, err := h.selectedTraces(workers, head, number, selected)
				if err != nil {
					return err
				}
				for _, trace := range traces {
					if err := answer.add(&trace); err != nil {
						return err
					}
				}
			}
			return nil
		})
	}
	err := group.Wait()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return fmt.Errorf("timeout when replaying tx")
	}
	return err
}

// selectedTraces replays a block and answers its traces that the addresses
// select.
func (h *handlers) selectedTraces(ctx context.Context, head *evmcore.EvmBlock, number uint64, selected selection) ([]ActionTrace, error) {
	block, err := h.environment.BlockAt(head, number)
	if exec.IsMissing(err) {
		return nil, fmt.Errorf("cannot get block from db %v", number)
	}
	if err != nil {
		return nil, fmt.Errorf("cannot get block from db %v, error:%v", number, err.Error())
	}
	if len(block.Transactions) == 0 {
		return []ActionTrace{}, nil
	}
	traces, err := h.replayBlock(ctx, head, block, nil, nil)
	if err != nil {
		return nil, err
	}
	var kept []ActionTrace
	for _, trace := range *traces {
		if trace.Action == nil {
			continue
		}
		from, to := filterAddresses(&trace)
		if selects(selected.senders, from) && selects(selected.targets, to) {
			kept = append(kept, trace)
		}
	}
	return kept, nil
}

// filterAddresses are the addresses value leaves and reaches in a trace, as
// OpenEthereum matches them; a failed creation reaches none.
func filterAddresses(trace *ActionTrace) (from, to *common.Address) {
	switch trace.TraceType {
	case CREATE:
		if trace.Result != nil {
			to = trace.Result.Address
		}
		return trace.Action.From, to
	case SELFDESTRUCT:
		return trace.Action.Address, trace.Action.RefundAddress
	}
	return trace.Action.From, trace.Action.To
}

// selects tells whether an address is in a set; any is without one.
func selects(set map[common.Address]bool, address *common.Address) bool {
	return len(set) == 0 || address != nil && set[*address]
}

// resultBuffer collects encoded traces into a JSON array within a size, as
// the legacy result buffer does: the brackets and commas count, and a size of
// zero holds nothing.
type resultBuffer struct {
	mutex  sync.Mutex
	buffer bytes.Buffer
	limit  int
}

func newResultBuffer(limit int) (*resultBuffer, error) {
	answer := &resultBuffer{limit: limit}
	return answer, answer.write([]byte("["))
}

func (b *resultBuffer) add(trace *ActionTrace) error {
	encoded, err := json.Marshal(trace)
	if err != nil {
		return err
	}
	b.mutex.Lock()
	defer b.mutex.Unlock()
	if b.buffer.Len() > 1 {
		if err := b.write([]byte(",")); err != nil {
			return err
		}
	}
	return b.write(encoded)
}

func (b *resultBuffer) result() (dispatch.Encoded, error) {
	if err := b.write([]byte("]")); err != nil {
		return nil, err
	}
	return b.buffer.Bytes(), nil
}

// write appends bytes within the size; the caller holds the lock or owns the
// buffer.
func (b *resultBuffer) write(data []byte) error {
	if b.buffer.Len()+len(data) > b.limit {
		return errResponseTooLarge
	}
	b.buffer.Write(data)
	return nil
}
