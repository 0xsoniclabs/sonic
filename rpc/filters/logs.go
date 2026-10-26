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
	"errors"
	"fmt"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/exec"
)

// findLogs answers a log query at the head, as the reference's Filter did:
// the logs of one block for a hash, else the logs of a range, read through
// the log index when the query names an address or a topic and by a walk
// over the blocks' receipts when it names neither (I11). The logs are
// copies, with the position of their transaction and no null topics (P8).
func (handler *handlers) findLogs(head exec.Head, criteria Criteria) ([]*types.Log, error) {
	if err := checkParameterLimit(handler.environment.Config().LogQueryParameterLimit, criteria); err != nil {
		return nil, err
	}
	var logs []*types.Log
	var err error
	if criteria.BlockHash != nil {
		logs, err = handler.blockLogs(head, *criteria.BlockHash, criteria)
	} else {
		logs, err = handler.rangeLogs(head, criteria)
	}
	if err != nil {
		return nil, err
	}
	return handler.answerLogs(head, logs), nil
}

func (handler *handlers) blockLogs(head exec.Head, hash common.Hash, criteria Criteria) ([]*types.Log, error) {
	header, err := handler.environment.FindHeader(head, chain.ReferenceToHash(hash))
	switch {
	case errors.Is(err, chain.ErrUnknownHash):
		return nil, exec.UnknownHashError(hash)
	case exec.Absent(err):
		return nil, errors.New("unknown block")
	case err != nil:
		return nil, err
	}
	receipts, err := exec.NullIfAbsent(handler.environment.ReceiptsAt(head, header.Number.Uint64()))
	if err != nil {
		return nil, err
	}
	return filterLogs(receiptLogs(receipts), criteria.Addresses, criteria.Topics), nil
}

// rangeLogs reads the logs of a range: a missing or tag bound is the head,
// and the range ends at the head (F6).
func (handler *handlers) rangeLogs(head exec.Head, criteria Criteria) ([]*types.Log, error) {
	requestedFirst, requestedLast := bound(criteria.FromBlock), bound(criteria.ToBlock)
	first, last := head.Number, head.Number
	if requestedFirst >= 0 {
		first = uint64(requestedFirst)
	}
	if requestedLast >= 0 && uint64(requestedLast) < head.Number {
		last = uint64(requestedLast)
	}
	if first > last {
		return nil, nil
	}
	if len(criteria.Addresses) == 0 && !hasTopic(criteria.Topics) {
		// The reference lifted its result limit for a query whose requested
		// bounds were equal, tags included.
		single := requestedFirst == requestedLast
		return handler.walkLogs(head, first, last, criteria, single)
	}
	return handler.indexedLogs(head, first, last, criteria)
}

// bound is a bound of a range as the reference read it: a tag, or a missing
// bound, is negative.
func bound(number *chain.BlockNumber) int64 {
	if number == nil {
		return int64(chain.LatestBlockNumber)
	}
	return int64(*number)
}

func hasTopic(topics [][]common.Hash) bool {
	for _, alternatives := range topics {
		if len(alternatives) > 0 {
			return true
		}
	}
	return false
}

// indexedLogs reads the logs of a range through the log index, in the order
// of their blocks and positions, each with the time of its block.
func (handler *handlers) indexedLogs(head exec.Head, first, last uint64, criteria Criteria) ([]*types.Log, error) {
	limits := handler.environment.Config()
	if last-first > limits.IndexedLogsBlockRangeLimit {
		return nil, fmt.Errorf("too wide blocks range, the limit is %d", limits.IndexedLogsBlockRangeLimit)
	}
	addresses := make([]common.Hash, len(criteria.Addresses))
	for index, address := range criteria.Addresses {
		addresses[index] = common.BytesToHash(address[:])
	}
	pattern := append([][]common.Hash{addresses}, criteria.Topics...)
	resultLimit := limits.LogQueryResultLimit
	if first == last {
		resultLimit = 0
	}
	found, err := handler.logIndex.FindLogs(head.Context(), first, last, pattern, resultLimit)
	if err != nil {
		return nil, err
	}
	logs := make([]*types.Log, len(found))
	for index, entry := range found {
		copied := *entry
		logs[index] = &copied
	}
	sort.SliceStable(logs, func(left, right int) bool {
		if logs[left].BlockNumber != logs[right].BlockNumber {
			return logs[left].BlockNumber < logs[right].BlockNumber
		}
		return logs[left].Index < logs[right].Index
	})
	if err := handler.setTimestamps(head, logs); err != nil {
		return nil, err
	}
	return logs, nil
}

// setTimestamps gives each log the time of its block, reading each block's
// header once.
func (handler *handlers) setTimestamps(head exec.Head, logs []*types.Log) error {
	times := map[uint64]uint64{}
	for _, entry := range logs {
		time, known := times[entry.BlockNumber]
		if !known {
			header, err := handler.environment.HeaderAt(head, entry.BlockNumber)
			if exec.Absent(err) {
				return fmt.Errorf("header for block %d containing relevant log entry not found", entry.BlockNumber)
			}
			if err != nil {
				return fmt.Errorf("failed to get header for block %d containing relevant log entry: %w", entry.BlockNumber, err)
			}
			time = uint64(header.Time.Unix())
			times[entry.BlockNumber] = time
		}
		entry.BlockTimestamp = time
	}
	return nil
}

// walkLogs reads the logs of a range block by block. It ends at the first
// block whose receipts it cannot read and answers what it has (F6).
func (handler *handlers) walkLogs(head exec.Head, first, last uint64, criteria Criteria, single bool) ([]*types.Log, error) {
	limits := handler.environment.Config()
	if last-first > limits.UnindexedLogsBlockRangeLimit {
		return nil, fmt.Errorf("too wide blocks range, the limit is %d", limits.UnindexedLogsBlockRangeLimit)
	}
	resultLimit := limits.LogQueryResultLimit
	if single {
		resultLimit = 0
	}
	var logs []*types.Log
	for number := first; number <= last; number++ {
		if err := head.Context().Err(); err != nil {
			return nil, err
		}
		receipts, err := handler.environment.ReceiptsAt(head, number)
		if exec.Absent(err) {
			return logs, nil
		}
		if err != nil {
			return nil, err
		}
		logs = append(logs, filterLogs(receiptLogs(receipts), criteria.Addresses, criteria.Topics)...)
		if resultLimit > 0 && uint(len(logs)) > resultLimit {
			return nil, fmt.Errorf("too many results, consider narrowing your query criteria, the limit is %d", resultLimit)
		}
	}
	return logs, nil
}

func receiptLogs(receipts types.Receipts) []*types.Log {
	var logs []*types.Log
	for _, receipt := range receipts {
		logs = append(logs, receipt.Logs...)
	}
	return logs
}

// answerLogs copies the logs for the answer, with the position of each
// transaction in its block, read once per transaction, and no null topics.
// No answer is null.
func (handler *handlers) answerLogs(head exec.Head, logs []*types.Log) []*types.Log {
	answered := make([]*types.Log, len(logs))
	positions := map[common.Hash]*exec.Located{}
	for index, entry := range logs {
		copied := *entry
		located, known := positions[copied.TxHash]
		if !known {
			located, _ = exec.NullIfAbsent(handler.environment.TransactionByHash(head, copied.TxHash))
			positions[copied.TxHash] = located
		}
		if located != nil {
			copied.TxIndex = uint(located.Index)
		} else {
			log.Warn("tx index empty", "hash", copied.TxHash)
		}
		if copied.Topics == nil {
			copied.Topics = []common.Hash{}
		}
		answered[index] = &copied
	}
	return answered
}
