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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/0xsoniclabs/sonic/rpc/chain"
)

// Criteria are what a log query, a log filter or a log subscription asks
// for: go-ethereum's FilterQuery, decoded as the reference decoded it. A tag
// in a bound is kept as its negative value, which names the head (V4).
type Criteria struct {
	BlockHash *common.Hash
	FromBlock *chain.BlockNumber
	ToBlock   *chain.BlockNumber
	Addresses []common.Address
	Topics    [][]common.Hash
}

// UnmarshalJSON decodes the criteria: an address or a list of them, and
// topics that are each null, one topic or a list of topics, a null in a list
// matching any topic.
func (criteria *Criteria) UnmarshalJSON(data []byte) error {
	var raw struct {
		BlockHash *common.Hash       `json:"blockHash"`
		FromBlock *chain.BlockNumber `json:"fromBlock"`
		ToBlock   *chain.BlockNumber `json:"toBlock"`
		Addresses any                `json:"address"`
		Topics    []any              `json:"topics"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if raw.BlockHash != nil {
		if raw.FromBlock != nil || raw.ToBlock != nil {
			return errors.New("cannot specify both BlockHash and FromBlock/ToBlock, choose one or the other")
		}
		criteria.BlockHash = raw.BlockHash
	} else {
		criteria.FromBlock = raw.FromBlock
		criteria.ToBlock = raw.ToBlock
	}
	addresses, err := decodeAddresses(raw.Addresses)
	if err != nil {
		return err
	}
	criteria.Addresses = addresses
	if criteria.Topics, err = decodeTopics(raw.Topics); err != nil {
		return err
	}
	return nil
}

func decodeAddresses(raw any) ([]common.Address, error) {
	addresses := []common.Address{}
	switch value := raw.(type) {
	case nil:
	case []any:
		for index, element := range value {
			text, ok := element.(string)
			if !ok {
				return nil, fmt.Errorf("non-string address at index %d", index)
			}
			address, err := decodeAddress(text)
			if err != nil {
				return nil, fmt.Errorf("invalid address at index %d: %v", index, err)
			}
			addresses = append(addresses, address)
		}
	case string:
		address, err := decodeAddress(value)
		if err != nil {
			return nil, fmt.Errorf("invalid address: %v", err)
		}
		addresses = []common.Address{address}
	default:
		return nil, errors.New("invalid addresses in query")
	}
	return addresses, nil
}

// decodeTopics decodes the topic positions; a null position matches any
// topic, and so does a list with a null in it.
func decodeTopics(raw []any) ([][]common.Hash, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	topics := make([][]common.Hash, len(raw))
	for position, value := range raw {
		switch element := value.(type) {
		case nil:
		case string:
			topic, err := decodeTopic(element)
			if err != nil {
				return nil, err
			}
			topics[position] = []common.Hash{topic}
		case []any:
			alternatives, err := decodeAlternatives(element)
			if err != nil {
				return nil, err
			}
			topics[position] = alternatives
		default:
			return nil, errors.New("invalid topic(s)")
		}
	}
	return topics, nil
}

func decodeAlternatives(raw []any) ([]common.Hash, error) {
	var alternatives []common.Hash
	for _, element := range raw {
		if element == nil {
			return nil, nil
		}
		text, ok := element.(string)
		if !ok {
			return nil, errors.New("invalid topic(s)")
		}
		topic, err := decodeTopic(text)
		if err != nil {
			return nil, err
		}
		alternatives = append(alternatives, topic)
	}
	return alternatives, nil
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

// checkParameterLimit refuses criteria with more addresses and topics than
// the limit, zero for none, on every path a criteria takes (V4).
func checkParameterLimit(limit uint, criteria Criteria) error {
	if limit == 0 {
		return nil
	}
	count := len(criteria.Addresses)
	for _, alternatives := range criteria.Topics {
		count += len(alternatives)
	}
	if uint(count) > limit {
		return fmt.Errorf("too many query parameters, the limit is %d", limit)
	}
	return nil
}

func matchesAddressesAndTopics(log *types.Log, addresses []common.Address, topics [][]common.Hash) bool {
	if len(addresses) > 0 && !containsAddress(addresses, log.Address) {
		return false
	}
	if len(topics) > len(log.Topics) {
		return false
	}
	for position, alternatives := range topics {
		if len(alternatives) > 0 && !containsHash(alternatives, log.Topics[position]) {
			return false
		}
	}
	return true
}

func containsAddress(addresses []common.Address, address common.Address) bool {
	for _, candidate := range addresses {
		if candidate == address {
			return true
		}
	}
	return false
}

func containsHash(hashes []common.Hash, hash common.Hash) bool {
	for _, candidate := range hashes {
		if candidate == hash {
			return true
		}
	}
	return false
}

// filterLogs keeps the logs that match the addresses and topics.
func filterLogs(logs []*types.Log, addresses []common.Address, topics [][]common.Hash) []*types.Log {
	var kept []*types.Log
	for _, log := range logs {
		if matchesAddressesAndTopics(log, addresses, topics) {
			kept = append(kept, log)
		}
	}
	return kept
}
