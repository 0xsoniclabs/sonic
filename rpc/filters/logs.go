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
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
)

// query is one run of a log query. It is the node's earlier log filter:
// indexed queries go through Core.FindLogs, queries without address and topic,
// and queries by block hash, scan blocks through Core.Header and Core.Receipts.
type query struct {
	core  core.Core
	cfg   dispatch.Config
	chain *dispatch.Chain

	addresses []common.Address
	topics    [][]common.Hash

	block      common.Hash // block hash if filtering a single block
	byHash     bool
	begin, end int64 // range interval if filtering multiple blocks

	resultLimit uint // maximum number of log results, 0 means no limit
}

// checkParameterLimit rejects criteria whose combined number of addresses and
// topics exceeds the limit. The limit holds on every filter path.
func checkParameterLimit(cfg dispatch.Config, addresses []common.Address, topics [][]common.Hash) error {
	limit := cfg.LogQueryParameterLimit
	if limit == 0 {
		return nil
	}
	numParameters := len(addresses)
	for _, sub := range topics {
		numParameters += len(sub)
	}
	if uint(numParameters) > limit {
		return fmt.Errorf("too many query parameters, the limit is %d", limit)
	}
	return nil
}

// newQuery builds the query of the criteria, checking the parameter limit first.
func (a *API) newQuery(crit Criteria) (*query, error) {
	if err := checkParameterLimit(a.cfg, crit.Addresses, crit.Topics); err != nil {
		return nil, err
	}
	q := &query{core: a.core, cfg: a.cfg, chain: a.chain, addresses: crit.Addresses, topics: crit.Topics}
	if crit.BlockHash != nil {
		// Block filter requested, construct a single-shot filter.
		q.block, q.byHash = *crit.BlockHash, true
		return q, nil
	}
	// Convert the RPC block numbers into internal representations.
	q.begin = dispatch.LatestBlockNumber.Int64()
	if crit.FromBlock != nil {
		q.begin = crit.FromBlock.Int64()
	}
	q.end = dispatch.LatestBlockNumber.Int64()
	if crit.ToBlock != nil {
		q.end = crit.ToBlock.Int64()
	}
	q.resultLimit = a.cfg.LogQueryResultLimit
	// The result limit is ignored if only a single block is queried.
	if q.begin == q.end {
		q.resultLimit = 0
	}
	return q, nil
}

// Logs searches the blockchain for matching log entries.
func (q *query) Logs(ctx context.Context) ([]*types.Log, error) {
	head := q.chain.Head()
	var logs []*types.Log
	var err error
	if q.byHash {
		logs, err = q.fromBlockByHash(head)
	} else {
		logs, err = q.fromBlockRange(ctx, head)
	}
	if err != nil {
		return nil, err
	}

	// Update the transaction index of each log.
	for i := range logs {
		// Logs may be shared with other requests through the receipts cache;
		// copy them so that neither this update nor callers modify the shared ones.
		l := *logs[i]
		logs[i] = &l
		if _, _, index, err := q.core.TxByHash(l.TxHash); err == nil {
			l.TxIndex = uint(index)
		} else {
			log.Warn("tx index empty", "hash", l.TxHash)
		}
	}
	return logs, nil
}

func (q *query) fromBlockByHash(head dispatch.Head) ([]*types.Log, error) {
	n, ok := q.core.BlockNumberByHash(q.block)
	if !ok {
		return nil, fmt.Errorf("block with hash %s not found", q.block.String())
	}
	if n > head.Number {
		return nil, errors.New("unknown block")
	}
	if _, err := q.core.Header(n); err != nil {
		if errors.Is(err, core.ErrNotFound) {
			return nil, errors.New("unknown block")
		}
		return nil, err
	}
	return q.blockLogs(n)
}

func (q *query) fromBlockRange(ctx context.Context, head dispatch.Head) ([]*types.Log, error) {
	if head.Header == nil {
		return nil, nil
	}
	latest := head.Number

	begin := uint64(q.begin)
	if q.begin < 0 {
		begin = latest
	}
	// The limits apply to the range as it was requested, like the reference
	// release does. Blocks beyond the served head are not published yet, so the
	// walk stops at the head.
	requestedEnd := uint64(q.end)
	if q.end < 0 {
		requestedEnd = latest
	}
	if begin > requestedEnd {
		return nil, nil
	}

	if isEmpty(q.topics) && len(q.addresses) == 0 {
		return q.unindexedLogs(ctx, begin, requestedEnd, latest)
	}
	return q.indexedLogs(ctx, begin, requestedEnd, latest)
}

// indexedLogs returns the logs matching the criteria based on the topics index.
func (q *query) indexedLogs(ctx context.Context, begin, requestedEnd, latest uint64) ([]*types.Log, error) {
	if requestedEnd-begin > q.cfg.IndexedLogsBlockRangeLimit {
		return nil, fmt.Errorf("too wide blocks range, the limit is %d", q.cfg.IndexedLogsBlockRangeLimit)
	}
	end := min(requestedEnd, latest)
	if begin > end {
		return nil, nil
	}
	addresses := make([]common.Hash, len(q.addresses))
	for i, addr := range q.addresses {
		addresses[i] = common.BytesToHash(addr[:])
	}

	pattern := make([][]common.Hash, 1, len(q.topics)+1)
	pattern[0] = addresses
	pattern = append(pattern, q.topics...)

	resultLimit := q.cfg.LogQueryResultLimit
	if begin == requestedEnd {
		resultLimit = 0 // no limit for a single block
	}
	logs, err := q.core.FindLogs(ctx, begin, end, pattern, resultLimit)
	if err != nil {
		return nil, err
	}
	sortLogsByBlockNumberAndLogIndex(logs)

	for _, l := range logs {
		// Fetch the timestamp of the log from the header.
		header, err := q.core.Header(l.BlockNumber)
		if err != nil && !errors.Is(err, core.ErrNotFound) {
			return nil, fmt.Errorf("failed to get header for block %d containing relevant log entry: %w", l.BlockNumber, err)
		}
		if header == nil || err != nil {
			return nil, fmt.Errorf("header for block %d containing relevant log entry not found", l.BlockNumber)
		}
		l.BlockTimestamp = uint64(header.Time.Unix())
	}
	return logs, nil
}

func sortLogsByBlockNumberAndLogIndex(logs []*types.Log) {
	sort.Slice(logs, func(i, j int) bool {
		if logs[i].BlockNumber != logs[j].BlockNumber {
			return logs[i].BlockNumber < logs[j].BlockNumber
		}
		return logs[i].Index < logs[j].Index
	})
}

// unindexedLogs returns the logs matching the criteria based on raw block
// iteration. The walk ends at the first block it cannot read and returns what
// it has found so far.
func (q *query) unindexedLogs(ctx context.Context, begin, requestedEnd, latest uint64) (logs []*types.Log, err error) {
	if requestedEnd-begin > q.cfg.UnindexedLogsBlockRangeLimit {
		return nil, fmt.Errorf("too wide blocks range, the limit is %d", q.cfg.UnindexedLogsBlockRangeLimit)
	}
	end := min(requestedEnd, latest)

	for n := begin; n <= end; n++ {
		if err = ctx.Err(); err != nil {
			return
		}
		header, herr := q.core.Header(n)
		if errors.Is(herr, core.ErrNotFound) {
			return // the walk ends at the first block that is not there
		}
		if herr != nil {
			return logs, herr
		}
		if header == nil {
			return
		}
		var found []*types.Log
		found, err = q.blockLogs(n)
		if err != nil {
			return
		}
		logs = append(logs, found...)
		if q.resultLimit > 0 && uint(len(logs)) > q.resultLimit {
			return nil, fmt.Errorf("too many results, consider narrowing your query criteria, the limit is %d", q.resultLimit)
		}
		if n == ^uint64(0) {
			break
		}
	}
	return
}

// blockLogs returns the logs matching the criteria within a single block.
func (q *query) blockLogs(n uint64) ([]*types.Log, error) {
	receipts, err := q.core.Receipts(n)
	if errors.Is(err, core.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var unfiltered []*types.Log
	for _, r := range receipts {
		unfiltered = append(unfiltered, r.Logs...)
	}
	logs := filterLogs(unfiltered, nil, nil, q.addresses, q.topics)
	if len(logs) > 0 {
		return logs, nil
	}
	return nil, nil
}

func includes(addresses []common.Address, a common.Address) bool {
	for _, addr := range addresses {
		if addr == a {
			return true
		}
	}
	return false
}

// filterLogs creates a slice of logs matching the given criteria.
func filterLogs(logs []*types.Log, fromBlock, toBlock *big.Int, addresses []common.Address, topics [][]common.Hash) []*types.Log {
	var ret []*types.Log
Logs:
	for _, log := range logs {
		if fromBlock != nil && fromBlock.Int64() >= 0 && fromBlock.Uint64() > log.BlockNumber {
			continue
		}
		if toBlock != nil && toBlock.Int64() >= 0 && toBlock.Uint64() < log.BlockNumber {
			continue
		}

		if len(addresses) > 0 && !includes(addresses, log.Address) {
			continue
		}
		// If the to filtered topics is greater than the amount of topics in logs, skip.
		if len(topics) > len(log.Topics) {
			continue Logs
		}
		for i, sub := range topics {
			match := len(sub) == 0 // empty rule set == wildcard
			for _, topic := range sub {
				if log.Topics[i] == topic {
					match = true
					break
				}
			}
			if !match {
				continue Logs
			}
		}
		ret = append(ret, log)
	}
	return ret
}

func isEmpty(topics [][]common.Hash) bool {
	for _, tt := range topics {
		if len(tt) > 0 {
			return false
		}
	}
	return true
}

// returnLogs returns an empty log array in case the given logs array is nil,
// and makes sure no log has nil topics. A log without topics is replaced by a
// copy: the log may be shared with other requests.
func returnLogs(logs []*types.Log) []*types.Log {
	if logs == nil {
		return []*types.Log{}
	}
	for i, l := range logs {
		if l.Topics == nil {
			fixed := *l
			fixed.Topics = []common.Hash{}
			logs[i] = &fixed
		}
	}
	return logs
}

// returnHashes returns an empty hash array in case the given one is nil.
func returnHashes(hashes []common.Hash) []common.Hash {
	if hashes == nil {
		return []common.Hash{}
	}
	return hashes
}
