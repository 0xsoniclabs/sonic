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
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/0xsoniclabs/sonic/rpc/chain"
)

// FilterCriteria selects logs: by block hash, or by a range of blocks, and by
// address and topics.
type FilterCriteria struct {
	BlockHash *common.Hash
	FromBlock *big.Int
	ToBlock   *big.Int
	Addresses []common.Address
	Topics    [][]common.Hash
}

// UnmarshalJSON reads the criteria as the reference did: a single address or a
// list, and topics as strings, lists of strings or null, nested at most twice.
func (c *FilterCriteria) UnmarshalJSON(data []byte) error {
	type input struct {
		BlockHash *common.Hash       `json:"blockHash"`
		FromBlock *chain.BlockNumber `json:"fromBlock"`
		ToBlock   *chain.BlockNumber `json:"toBlock"`
		Addresses any                `json:"address"`
		Topics    []any              `json:"topics"`
	}
	var raw input
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if raw.BlockHash != nil {
		if raw.FromBlock != nil || raw.ToBlock != nil {
			return fmt.Errorf("cannot specify both BlockHash and FromBlock/ToBlock, choose one or the other")
		}
		c.BlockHash = raw.BlockHash
	} else {
		if raw.FromBlock != nil {
			c.FromBlock = big.NewInt(int64(*raw.FromBlock))
		}
		if raw.ToBlock != nil {
			c.ToBlock = big.NewInt(int64(*raw.ToBlock))
		}
	}

	c.Addresses = []common.Address{}
	if raw.Addresses != nil {
		switch rawAddr := raw.Addresses.(type) {
		case []any:
			for i, addr := range rawAddr {
				s, ok := addr.(string)
				if !ok {
					return fmt.Errorf("non-string address at index %d", i)
				}
				decoded, err := decodeAddress(s)
				if err != nil {
					return fmt.Errorf("invalid address at index %d: %v", i, err)
				}
				c.Addresses = append(c.Addresses, decoded)
			}
		case string:
			decoded, err := decodeAddress(rawAddr)
			if err != nil {
				return fmt.Errorf("invalid address: %v", err)
			}
			c.Addresses = []common.Address{decoded}
		default:
			return errors.New("invalid addresses in query")
		}
	}

	if len(raw.Topics) > 0 {
		c.Topics = make([][]common.Hash, len(raw.Topics))
		for i, t := range raw.Topics {
			switch topic := t.(type) {
			case nil:
				// a wildcard
			case string:
				decoded, err := decodeTopic(topic)
				if err != nil {
					return err
				}
				c.Topics[i] = []common.Hash{decoded}
			case []any:
				for _, rawTopic := range topic {
					if rawTopic == nil {
						// a null alternative matches all
						c.Topics[i] = nil
						break
					}
					s, ok := rawTopic.(string)
					if !ok {
						return fmt.Errorf("invalid topic(s)")
					}
					decoded, err := decodeTopic(s)
					if err != nil {
						return err
					}
					c.Topics[i] = append(c.Topics[i], decoded)
				}
			default:
				return fmt.Errorf("invalid topic(s)")
			}
		}
	}
	return nil
}

func decodeAddress(s string) (common.Address, error) {
	b, err := hexutil.Decode(s)
	if err == nil && len(b) != common.AddressLength {
		err = fmt.Errorf("hex has invalid length %d after decoding; expected %d for address", len(b), common.AddressLength)
	}
	return common.BytesToAddress(b), err
}

func decodeTopic(s string) (common.Hash, error) {
	b, err := hexutil.Decode(s)
	if err == nil && len(b) != common.HashLength {
		err = fmt.Errorf("hex has invalid length %d after decoding; expected %d for topic", len(b), common.HashLength)
	}
	return common.BytesToHash(b), err
}

// checkParameterLimit bounds the addresses and topics of a query; every
// filter path checks it before anything else.
func checkParameterLimit(limit uint, addresses []common.Address, topics [][]common.Hash) error {
	if limit == 0 {
		return nil
	}
	n := len(addresses)
	for _, sub := range topics {
		n += len(sub)
	}
	if uint(n) > limit {
		return fmt.Errorf("too many query parameters, the limit is %d", limit)
	}
	return nil
}

// filterLogs returns the logs that match a range, the addresses and the
// topics. Negative bounds are tags and do not restrict.
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
		if len(addresses) > 0 && !slices.Contains(addresses, log.Address) {
			continue
		}
		if len(topics) > len(log.Topics) {
			continue
		}
		for i, sub := range topics {
			match := len(sub) == 0 // an empty set is a wildcard
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

// logRange is how a log filter or subscription follows new blocks.
type logRange int

const (
	// liveLogs receive the logs of new blocks that match.
	liveLogs logRange = iota
	// pendingLogs ask for pending logs, which Sonic does not have: they
	// receive nothing.
	pendingLogs
)

// classifyRange checks the range of a log filter or subscription as the
// reference's event system did.
func classifyRange(crit FilterCriteria) (logRange, error) {
	from, to := chain.LatestBlockNumber, chain.LatestBlockNumber
	if crit.FromBlock != nil {
		from = chain.BlockNumber(crit.FromBlock.Int64())
	}
	if crit.ToBlock != nil {
		to = chain.BlockNumber(crit.ToBlock.Int64())
	}
	switch {
	case from == chain.PendingBlockNumber && to == chain.PendingBlockNumber:
		return pendingLogs, nil
	case from == chain.LatestBlockNumber && to == chain.LatestBlockNumber:
		return liveLogs, nil
	case from >= 0 && to >= 0 && to >= from:
		return liveLogs, nil
	case from >= chain.LatestBlockNumber && to == chain.PendingBlockNumber:
		return pendingLogs, nil
	case from >= 0 && to == chain.LatestBlockNumber:
		return liveLogs, nil
	}
	return 0, fmt.Errorf("invalid from and to block combination: from > to")
}
