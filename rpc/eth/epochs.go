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
	"errors"
	"math/big"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
)

// currentEpoch answers the epoch the block after the head runs in.
func (h *handlers) currentEpoch(context.Context, struct{}) (hexutil.Uint64, error) {
	current, err := h.environment.CurrentEpoch(h.environment.Head())
	return hexutil.Uint64(current), err
}

type epochParams struct {
	Epoch chain.BlockNumber
}

// getRules answers the network rules of an epoch, null for one the node
// keeps no states of.
func (h *handlers) getRules(_ context.Context, params epochParams) (*opera.Rules, error) {
	states, err := h.environment.EpochStateOf(h.environment.Head(), params.Epoch)
	if exec.IsMissing(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &states.EpochState.Rules, nil
}

// getEpochBlock answers the last block before an epoch, 0 for one the node
// keeps no states of.
func (h *handlers) getEpochBlock(_ context.Context, params epochParams) (hexutil.Uint64, error) {
	states, err := exec.NoneIfMissing(h.environment.EpochStateOf(h.environment.Head(), params.Epoch))
	return hexutil.Uint64(states.BlockState.LastBlock.Idx), err
}

// getEpochStats answers the start and end of the last sealed epoch, named by
// latest or by its number, with zero fees and weights, as the legacy handler
// does.
func (h *handlers) getEpochStats(_ context.Context, params epochParams) (map[string]any, error) {
	log.Warn("GetEpochStats API call is deprecated. Consider retrieving data from SFC v3 contract.")
	head := h.environment.Head()
	current, err := h.environment.CurrentEpoch(head)
	if err != nil {
		return nil, err
	}
	if params.Epoch != chain.LatestBlockNumber && params.Epoch != chain.BlockNumber(current)-1 {
		return nil, errors.New("getEpochStats API call doesn't support retrieving previous sealed epochs")
	}
	states, err := h.environment.EpochStateOf(head, chain.BlockNumber(current))
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"epoch":                 hexutil.Uint64(current - 1),
		"start":                 hexutil.Uint64(states.EpochState.PrevEpochStart),
		"end":                   hexutil.Uint64(states.EpochState.EpochStart),
		"totalFee":              (*hexutil.Big)(new(big.Int)),
		"totalBaseRewardWeight": (*hexutil.Big)(new(big.Int)),
		"totalTxRewardWeight":   (*hexutil.Big)(new(big.Int)),
	}, nil
}
