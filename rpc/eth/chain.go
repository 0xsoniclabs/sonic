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

package eth

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
)

func (handler *handlers) chainID(context.Context, dispatch.Arguments) (any, error) {
	return hexutil.Uint64(handler.environment.ChainInfo().ChainID), nil
}

func (handler *handlers) blockNumber(ctx context.Context, _ dispatch.Arguments) (any, error) {
	head, err := handler.environment.Latest(ctx)
	if err != nil {
		return nil, err
	}
	return hexutil.Uint64(head.Number), nil
}

// syncingAge is how old the newest block may be while the node counts as
// synchronized; it is well above the longest emission interval.
const syncingAge = 90 * time.Minute

func (handler *handlers) syncing(context.Context, dispatch.Arguments) (any, error) {
	progress := handler.node.Progress()
	if time.Since(progress.CurrentBlockTime.Time()) <= syncingAge {
		return false, nil
	}
	return map[string]any{
		"startingBlock":    hexutil.Uint64(0),
		"currentEpoch":     hexutil.Uint64(progress.CurrentEpoch),
		"currentBlock":     hexutil.Uint64(progress.CurrentBlock),
		"currentBlockHash": progress.CurrentBlockHash.Hex(),
		"currentBlockTime": hexutil.Uint64(progress.CurrentBlockTime),
		"highestBlock":     hexutil.Uint64(progress.HighestBlock),
		"highestEpoch":     hexutil.Uint64(progress.HighestEpoch),
		"pulledStates":     hexutil.Uint64(0),
		"knownStates":      hexutil.Uint64(0),
	}, nil
}

func (handler *handlers) currentEpoch(context.Context, dispatch.Arguments) (any, error) {
	return hexutil.Uint64(handler.node.Progress().CurrentEpoch), nil
}

// getRules answers the rules in force at a block. The tags that name the
// head answer the rules the next block is processed with, those of the open
// epoch, as the reference's open epoch did; a block the module does not have
// answers null.
func (handler *handlers) getRules(ctx context.Context, arguments dispatch.Arguments) (any, error) {
	number := dispatch.Argument[chain.BlockNumber](arguments, 0)
	head, err := handler.environment.Latest(ctx)
	if err != nil {
		return nil, err
	}
	block, err := handler.environment.Resolve(head, chain.ReferenceToNumber(number))
	if err != nil {
		return exec.NullIfAbsent[any](nil, err)
	}
	if number.IsTag() && number != chain.EarliestBlockNumber {
		block = head.Number + 1
	}
	rules, err := handler.environment.RulesAt(head, block)
	if err != nil {
		return exec.NullIfAbsent[any](nil, err)
	}
	return &rules, nil
}

func (handler *handlers) zeroAddress(context.Context, dispatch.Arguments) (any, error) {
	return common.Address{}, nil
}

func (handler *handlers) hashrate(context.Context, dispatch.Arguments) (any, error) {
	return hexutil.Uint64(0), nil
}
