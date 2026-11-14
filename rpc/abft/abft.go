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

// Package abft serves the abft namespace of the RPC module: the validators of
// an epoch and what the node has recorded of them in the open epoch.
package abft

import (
	"context"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/iblockproc"
	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// Node is what the namespace asks the node beyond the served chain.
type Node interface {
	LiveEpochState() core.EpochSnapshot
}

type handlers struct {
	environment *exec.Environment
	node        Node
}

// Register adds the methods of the namespace to the dispatcher.
func Register(dispatcher *dispatch.Dispatcher, environment *exec.Environment, node Node) {
	h := &handlers{environment: environment, node: node}
	dispatch.Add(dispatcher, "abft_getValidators", h.getValidators)
	dispatch.Add(dispatcher, "abft_getDowntime", h.getDowntime)
	dispatch.Add(dispatcher, "abft_getEpochUptime", h.getEpochUptime)
	dispatch.Add(dispatcher, "abft_getOriginatedEpochFee", h.getOriginatedEpochFee)
}

type epochParams struct {
	Epoch chain.BlockNumber
}

// getValidators answers the weight and public key of each validator of an
// epoch, null for one the node keeps no states of. For pending, the profiles
// are those of the next epoch.
func (h *handlers) getValidators(_ context.Context, params epochParams) (map[hexutil.Uint64]any, error) {
	states, err := h.environment.EpochStateOf(h.environment.Head(), params.Epoch)
	if exec.IsMissing(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	profiles := states.EpochState.ValidatorProfiles
	if params.Epoch == chain.PendingBlockNumber {
		profiles = states.BlockState.NextValidatorProfiles
	}
	validators := map[hexutil.Uint64]any{}
	for _, id := range states.EpochState.Validators.IDs() {
		validators[hexutil.Uint64(id)] = map[string]any{
			"weight": (*hexutil.Big)(profiles[id].Weight),
			"pubkey": profiles[id].PubKey.String(),
		}
	}
	return validators, nil
}

type validatorParams struct {
	ValidatorID hexutil.Uint
}

// getDowntime answers how many blocks and how long a validator of the open
// epoch has been offline, zero within the slack the rules allow and for one
// that is no validator.
func (h *handlers) getDowntime(_ context.Context, params validatorParams) (map[string]any, error) {
	var blocks idx.Block
	var period inter.Timestamp
	states := h.node.LiveEpochState()
	if state := validatorState(states, params.ValidatorID); state != nil {
		last := states.BlockState.LastBlock
		if last.Idx > state.LastBlock {
			blocks = last.Idx - state.LastBlock
		}
		if last.Time > state.LastOnlineTime {
			period = last.Time - state.LastOnlineTime
		}
		if blocks < states.EpochState.Rules.Economy.BlockMissedSlack {
			blocks, period = 0, 0
		}
	}
	return map[string]any{"offlineBlocks": hexutil.Uint64(blocks), "offlineTime": hexutil.Uint64(period)}, nil
}

// getEpochUptime answers how long, in nanoseconds, a validator has been
// online in the open epoch, zero for one that is no validator.
func (h *handlers) getEpochUptime(_ context.Context, params validatorParams) (hexutil.Uint64, error) {
	state := validatorState(h.node.LiveEpochState(), params.ValidatorID)
	if state == nil {
		return 0, nil
	}
	return hexutil.Uint64(state.Uptime), nil
}

// getOriginatedEpochFee answers the fees of the transactions a validator
// originated in the open epoch, null for one that is no validator.
func (h *handlers) getOriginatedEpochFee(_ context.Context, params validatorParams) (*hexutil.Big, error) {
	state := validatorState(h.node.LiveEpochState(), params.ValidatorID)
	if state == nil {
		return nil, nil
	}
	return (*hexutil.Big)(state.Originated), nil
}

// validatorState is the state of a validator in the states the node read
// together, nil for one that is no validator of their epoch.
func validatorState(states core.EpochSnapshot, id hexutil.Uint) *iblockproc.ValidatorBlockState {
	validators := states.EpochState.Validators
	if !validators.Exists(idx.ValidatorID(id)) {
		return nil
	}
	return states.BlockState.GetValidatorState(idx.ValidatorID(id), validators)
}
