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
	"slices"

	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/ethereum/go-ethereum/core/types"
)

// checkParameterLimit refuses criteria with more addresses and topics than
// the limit, unless the limit is zero.
func checkParameterLimit(limit uint, c criteria) error {
	if limit == 0 {
		return nil
	}
	parameters := len(c.Addresses)
	for _, position := range c.Topics {
		parameters += len(position)
	}
	if uint(parameters) > limit {
		return fmt.Errorf("too many query parameters, the limit is %d", limit)
	}
	return nil
}

// receivesMined tells whether a filter or subscription of the criteria
// receives the logs of new blocks, as the legacy event system decides it
// from the bounds, a missing one being latest: bounds that name the pending
// block are accepted but receive nothing, as the network has no pending
// block, and other bounds are refused.
func (c criteria) receivesMined() (bool, error) {
	from, to := chain.LatestBlockNumber, chain.LatestBlockNumber
	if c.FromBlock != nil {
		from = *c.FromBlock
	}
	if c.ToBlock != nil {
		to = *c.ToBlock
	}
	switch {
	case from == chain.PendingBlockNumber && to == chain.PendingBlockNumber:
		return false, nil
	case from == chain.LatestBlockNumber && to == chain.LatestBlockNumber:
		return true, nil
	case from >= 0 && to >= 0 && to >= from:
		return true, nil
	case from >= chain.LatestBlockNumber && to == chain.PendingBlockNumber:
		return false, nil
	case from >= 0 && to == chain.LatestBlockNumber:
		return true, nil
	}
	return false, errors.New("invalid from and to block combination: from > to")
}

// matching are the logs the criteria select, as the legacy filter matches
// them: within the bounds that are numbers, from the addresses, and with
// one of the topics of each position, an empty position matching any.
func (c criteria) matching(logs []*types.Log) []*types.Log {
	var matched []*types.Log
	for _, log := range logs {
		if c.matches(log) {
			matched = append(matched, log)
		}
	}
	return matched
}

func (c criteria) matches(log *types.Log) bool {
	if c.FromBlock != nil && *c.FromBlock >= 0 && uint64(*c.FromBlock) > log.BlockNumber {
		return false
	}
	if c.ToBlock != nil && *c.ToBlock >= 0 && uint64(*c.ToBlock) < log.BlockNumber {
		return false
	}
	if len(c.Addresses) > 0 && !slices.Contains(c.Addresses, log.Address) {
		return false
	}
	if len(c.Topics) > len(log.Topics) {
		return false
	}
	for i, position := range c.Topics {
		if len(position) > 0 && !slices.Contains(position, log.Topics[i]) {
			return false
		}
	}
	return true
}
