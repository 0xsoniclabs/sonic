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

	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

// criteria select logs: a block, or a range of blocks whose missing bounds
// are latest, and addresses and topics, as the legacy filter criteria decode
// them.
type criteria struct {
	BlockHash *common.Hash
	FromBlock *chain.BlockNumber
	ToBlock   *chain.BlockNumber
	Addresses []common.Address
	// Topics holds a set of topics per position; an empty set matches any.
	Topics [][]common.Hash
}

// UnmarshalJSON decodes criteria as the legacy criteria decode, whose
// wording names the type of the raw criteria filters.input.
func (c *criteria) UnmarshalJSON(data []byte) error {
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
			return errors.New("cannot specify both BlockHash and FromBlock/ToBlock, choose one or the other")
		}
		c.BlockHash = raw.BlockHash
	} else {
		c.FromBlock, c.ToBlock = raw.FromBlock, raw.ToBlock
	}
	addresses, err := decodeAddresses(raw.Addresses)
	if err != nil {
		return err
	}
	c.Addresses = addresses
	c.Topics, err = decodeTopics(raw.Topics)
	return err
}

// decodeAddresses decodes one address or a list of them.
func decodeAddresses(raw any) ([]common.Address, error) {
	addresses := []common.Address{}
	switch raw := raw.(type) {
	case nil:
	case []any:
		for i, element := range raw {
			text, ok := element.(string)
			if !ok {
				return nil, fmt.Errorf("non-string address at index %d", i)
			}
			address, err := decodeAddress(text)
			if err != nil {
				return nil, fmt.Errorf("invalid address at index %d: %v", i, err)
			}
			addresses = append(addresses, address)
		}
	case string:
		address, err := decodeAddress(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid address: %v", err)
		}
		addresses = append(addresses, address)
	default:
		return nil, errors.New("invalid addresses in query")
	}
	return addresses, nil
}

// decodeTopics decodes a topic, a list of topics or null per position. A
// null, also in a list, matches any topic.
func decodeTopics(raw []any) ([][]common.Hash, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	topics := make([][]common.Hash, len(raw))
	for i, position := range raw {
		switch position := position.(type) {
		case nil:
		case string:
			topic, err := decodeTopic(position)
			if err != nil {
				return nil, err
			}
			topics[i] = []common.Hash{topic}
		case []any:
			for _, element := range position {
				if element == nil {
					topics[i] = nil
					break
				}
				text, ok := element.(string)
				if !ok {
					return nil, errors.New("invalid topic(s)")
				}
				topic, err := decodeTopic(text)
				if err != nil {
					return nil, err
				}
				topics[i] = append(topics[i], topic)
			}
		default:
			return nil, errors.New("invalid topic(s)")
		}
	}
	return topics, nil
}

func decodeAddress(text string) (common.Address, error) {
	decoded, err := hexutil.Decode(text)
	if err == nil && len(decoded) != common.AddressLength {
		err = fmt.Errorf("hex has invalid length %d after decoding; expected %d for address", len(decoded), common.AddressLength)
	}
	return common.BytesToAddress(decoded), err
}

func decodeTopic(text string) (common.Hash, error) {
	decoded, err := hexutil.Decode(text)
	if err == nil && len(decoded) != common.HashLength {
		err = fmt.Errorf("hex has invalid length %d after decoding; expected %d for topic", len(decoded), common.HashLength)
	}
	return common.BytesToHash(decoded), err
}

// query is the log query of the criteria over the blocks up to the head, a
// missing bound being latest.
func (c criteria) query(head uint64) core.LogQuery {
	query := core.LogQuery{
		BlockHash: c.BlockHash,
		Begin:     int64(chain.LatestBlockNumber),
		End:       int64(chain.LatestBlockNumber),
		Addresses: c.Addresses,
		Topics:    c.Topics,
		Head:      head,
	}
	if c.FromBlock != nil {
		query.Begin = int64(*c.FromBlock)
	}
	if c.ToBlock != nil {
		query.End = int64(*c.ToBlock)
	}
	return query
}

// answered is logs as an answer has them: a list, also when empty, and each
// log with a list of topics, also when it has none. It copies a log it
// changes, as logs are shared.
func answered(logs []*types.Log) []*types.Log {
	answer := make([]*types.Log, len(logs))
	for i, log := range logs {
		if log.Topics == nil {
			copied := *log
			copied.Topics = []common.Hash{}
			log = &copied
		}
		answer[i] = log
	}
	return answer
}
