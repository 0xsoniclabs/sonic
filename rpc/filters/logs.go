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

package filters

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/format"
)

// queryLogs answers a log query: a single block by hash, or a range of blocks.
// Indexed queries go through the node's log index; queries with no address
// and no topic, and block hash queries, read the receipts of each block.
func (s *System) queryLogs(ctx context.Context, crit FilterCriteria) ([]*types.Log, error) {
	cfg := s.env.Config()
	if err := checkParameterLimit(cfg.LogQueryParameterLimit, crit.Addresses, crit.Topics); err != nil {
		return nil, err
	}
	head := s.env.Latest(ctx)
	var (
		logs []*types.Log
		err  error
	)
	if crit.BlockHash != nil {
		logs, err = s.blockHashLogs(head, *crit.BlockHash, crit)
	} else {
		logs, err = s.rangeLogs(ctx, head, crit)
	}
	if err != nil {
		return nil, err
	}
	// The logs come from shared caches; each one is changed on a copy.
	for i, l := range logs {
		c := *l
		if _, _, index, err := s.env.TxByHash(head, c.TxHash); err == nil {
			c.TxIndex = uint(index)
		} else {
			log.Warn("tx index empty", "hash", c.TxHash)
		}
		logs[i] = &c
	}
	return format.Logs(logs), nil
}

func (s *System) blockHashLogs(head *exec.Head, hash common.Hash, crit FilterCriteria) ([]*types.Log, error) {
	header, err := s.env.FindHeader(head, chain.HashRef(hash))
	switch {
	case errors.Is(err, chain.ErrUnknownHash):
		return nil, exec.UnknownHashError(hash)
	case exec.IsMissing(err):
		return nil, errors.New("unknown block")
	case err != nil:
		return nil, err
	}
	return s.blockLogs(head, header.Number.Uint64(), crit)
}

// rangeLogs resolves the range against the head: a tag, earliest included,
// is the head, and the end is clamped to it. The range limits are checked on
// these bounds before any block is read.
func (s *System) rangeLogs(ctx context.Context, head *exec.Head, crit FilterCriteria) ([]*types.Log, error) {
	cfg := s.env.Config()
	begin, end := int64(chain.LatestBlockNumber), int64(chain.LatestBlockNumber)
	if crit.FromBlock != nil {
		begin = crit.FromBlock.Int64()
	}
	if crit.ToBlock != nil {
		end = crit.ToBlock.Int64()
	}
	resultLimit := cfg.LogQueryResultLimit
	if begin == end {
		resultLimit = 0 // a single block has no result limit
	}

	headNumber := head.Number()
	from := uint64(begin)
	if begin < 0 {
		from = headNumber
	}
	to := uint64(end)
	if end < 0 || to > headNumber {
		to = headNumber
	}
	if from > to {
		return nil, nil
	}
	if isEmpty(crit.Topics) && len(crit.Addresses) == 0 {
		return s.scanLogs(ctx, head, from, to, crit, resultLimit)
	}
	return s.indexedLogs(ctx, head, from, to, crit)
}

func (s *System) indexedLogs(ctx context.Context, head *exec.Head, from, to uint64, crit FilterCriteria) ([]*types.Log, error) {
	cfg := s.env.Config()
	if to-from > cfg.IndexedLogsBlockRangeLimit {
		return nil, fmt.Errorf("too wide blocks range, the limit is %d", cfg.IndexedLogsBlockRangeLimit)
	}
	addresses := make([]common.Hash, len(crit.Addresses))
	for i, addr := range crit.Addresses {
		addresses[i] = common.BytesToHash(addr[:])
	}
	pattern := make([][]common.Hash, 1, len(crit.Topics)+1)
	pattern[0] = addresses
	pattern = append(pattern, crit.Topics...)

	resultLimit := cfg.LogQueryResultLimit
	if from == to {
		resultLimit = 0
	}
	found, err := s.logs.FindLogs(ctx, from, to, pattern, resultLimit)
	if err != nil {
		return nil, err
	}
	sort.Slice(found, func(i, j int) bool {
		if found[i].BlockNumber != found[j].BlockNumber {
			return found[i].BlockNumber < found[j].BlockNumber
		}
		return found[i].Index < found[j].Index
	})
	logs := make([]*types.Log, len(found))
	for i, l := range found {
		header, err := s.env.HeaderAt(head, l.BlockNumber)
		if exec.IsMissing(err) {
			return nil, fmt.Errorf("header for block %d containing relevant log entry not found", l.BlockNumber)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to get header for block %d containing relevant log entry: %w", l.BlockNumber, err)
		}
		c := *l
		c.BlockTimestamp = uint64(header.Time.Unix())
		logs[i] = &c
	}
	return logs, nil
}

// scanLogs reads the receipts of every block of the range. It ends at the
// first block it cannot read and answers what it found up to there.
func (s *System) scanLogs(ctx context.Context, head *exec.Head, from, to uint64, crit FilterCriteria, resultLimit uint) ([]*types.Log, error) {
	cfg := s.env.Config()
	if to-from > cfg.UnindexedLogsBlockRangeLimit {
		return nil, fmt.Errorf("too wide blocks range, the limit is %d", cfg.UnindexedLogsBlockRangeLimit)
	}
	var logs []*types.Log
	for n := from; n <= to; n++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if _, err := s.env.HeaderAt(head, n); err != nil {
			if exec.IsMissing(err) {
				return logs, nil
			}
			return nil, err
		}
		found, err := s.blockLogs(head, n, crit)
		if err != nil {
			return nil, err
		}
		logs = append(logs, found...)
		if resultLimit > 0 && uint(len(logs)) > resultLimit {
			return nil, fmt.Errorf("too many results, consider narrowing your query criteria, the limit is %d", resultLimit)
		}
	}
	return logs, nil
}

// blockLogs returns the logs of a block's receipts that match.
func (s *System) blockLogs(head *exec.Head, n uint64, crit FilterCriteria) ([]*types.Log, error) {
	receipts, err := s.env.ReceiptsAt(head, n)
	if err != nil && !exec.IsMissing(err) {
		return nil, err
	}
	var all []*types.Log
	for _, receipt := range receipts {
		all = append(all, receipt.Logs...)
	}
	return filterLogs(all, nil, nil, crit.Addresses, crit.Topics), nil
}
