// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package filters

import (
	"context"
	"errors"
	"math/big"
	"sort"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	notify "github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/evmstore"
	"github.com/0xsoniclabs/sonic/topicsdb"
)

//go:generate mockgen -source=filter.go -package=filters -destination=filter_mock.go

type Backend interface {
	HeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*evmcore.EvmHeader, error)
	HeaderByHash(ctx context.Context, blockHash common.Hash) (*evmcore.EvmHeader, error)
	GetReceipts(ctx context.Context, blockHash common.Hash) (types.Receipts, error)
	GetReceiptsByNumber(ctx context.Context, number rpc.BlockNumber) (types.Receipts, error)
	GetLogs(ctx context.Context, blockHash common.Hash) ([][]*types.Log, error)
	GetTxPosition(txid common.Hash) *evmstore.TxPosition

	SubscribeNewBlockNotify(ch chan<- evmcore.ChainHeadNotify) notify.Subscription
	SubscribeNewTxsNotify(chan<- evmcore.NewTxsNotify) notify.Subscription
	SubscribeLogsNotify(ch chan<- []*types.Log) notify.Subscription

	EvmLogIndex() topicsdb.Index

	CalcBlockExtApi() bool
}

// Filter can be used to retrieve and filter logs.
type Filter struct {
	backend Backend
	config  Config

	addresses []common.Address
	topics    [][]common.Hash

	block      common.Hash // Block hash if filtering a single block
	begin, end int64       // Range interval if filtering multiple blocks
}

// NewRangeFilter creates a new filter which inspects the blocks to
// figure out whether a particular block is interesting or not.
func NewRangeFilter(backend Backend, cfg Config, begin, end int64, addresses []common.Address, topics [][]common.Hash) *Filter {
	// Create a generic filter and convert it into a range filter
	filter := newFilter(backend, cfg, addresses, topics)

	filter.begin = begin
	filter.end = end

	return filter
}

// NewBlockFilter creates a new filter which directly inspects the contents of
// a block to figure out whether it is interesting or not.
func NewBlockFilter(backend Backend, cfg Config, block common.Hash, addresses []common.Address, topics [][]common.Hash) *Filter {
	// Create a generic filter and convert it into a block filter
	filter := newFilter(backend, cfg, addresses, topics)

	filter.block = block

	return filter
}

// newFilter creates a generic filter that can either filter based on a block hash,
// or based on range queries. The search criteria needs to be explicitly set.
func newFilter(backend Backend, cfg Config, addresses []common.Address, topics [][]common.Hash) *Filter {
	return &Filter{
		backend:   backend,
		config:    cfg,
		addresses: addresses,
		topics:    topics,
	}
}

// Logs searches the blockchain for matching log entries, returning all from the
// first block that contains matches, updating the start of the filter accordingly.
func (f *Filter) Logs(ctx context.Context) ([]*types.Log, error) {

	headers := make([]*evmcore.EvmHeader, 0)

	if f.block != common.Hash(hash.Zero) {

		block, err := f.backend.HeaderByHash(ctx, f.block)
		if err != nil {
			return nil, err
		}
		if block == nil {
			return nil, errors.New("unknown block")
		}
		headers = append(headers, block)
	} else {
		for i := f.begin; i <= f.end; i++ {
			block, err := f.backend.HeaderByNumber(ctx, rpc.BlockNumber(i))
			if err != nil {
				return nil, err
			}
			if block == nil {
				return nil, errors.New("unknown block")
			}
			headers = append(headers, block)
		}
	}

	// get logs from the blocks
	resultLogs := make([]*types.Log, 0)

	for _, block := range headers {

		receipts, err := f.backend.GetReceiptsByNumber(ctx, rpc.BlockNumber(block.Number.Uint64()))
		if err != nil {
			return nil, err
		}

		for i, receipt := range receipts {
			logs := filterLogs(receipt.Logs, nil, nil, f.addresses, f.topics)
			if len(logs) > 0 {
				for _, log := range logs {
					// set BlockTimestamp
					log.BlockTimestamp = uint64(block.Time.Unix())
					// set transaction index
					log.TxIndex = uint(i)
					resultLogs = append(resultLogs, log)
				}
			}
		}
	}

	sortLogsByBlockNumberAndLogIndex(resultLogs)

	return resultLogs, nil
}

func sortLogsByBlockNumberAndLogIndex(logs []*types.Log) {
	sort.Slice(logs, func(i, j int) bool {
		if logs[i].BlockNumber != logs[j].BlockNumber {
			return logs[i].BlockNumber < logs[j].BlockNumber
		}
		return logs[i].Index < logs[j].Index
	})
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

	// if nothing to filter, return all input logs
	if fromBlock == nil && toBlock == nil && len(addresses) == 0 && len(topics) == 0 {
		return logs
	}

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
			continue
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
