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

package exec

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

// Estimation is a gas estimation of a call on top of a block.
type Estimation struct {
	Arguments      CallArguments
	Block          chain.BlockReference
	StateOverrides *StateOverride
	BlockOverrides *BlockOverrides
	// Before are calls run first in every run of the estimation.
	Before []CallArguments
}

// EstimateGas searches the least gas with which the call does not fail,
// between the intrinsic gas and the most the call may use: the gas given or
// the transaction limit, capped from Osaka on, by the funds of the sender and
// by the gas cap. Each run starts from the overridden state of the block.
func (e *Environment) EstimateGas(ctx context.Context, head *evmcore.EvmBlock, estimation Estimation) (hexutil.Uint64, error) {
	args := estimation.Arguments
	if args.From == nil {
		args.From = new(common.Address)
	}
	header, err := e.estimationHeader(head, estimation.Block)
	if err != nil {
		return 0, err
	}
	highest, err := e.highestGas(head, header, args, estimation.BlockOverrides)
	if err != nil {
		return 0, err
	}
	feeCap, err := args.feeCap()
	if err != nil {
		return 0, err
	}
	runs := &estimationRuns{environment: e, header: header, estimation: estimation}
	defer runs.release()
	if feeCap.BitLen() != 0 {
		if highest, err = runs.capByFunds(args, feeCap, highest); err != nil {
			return 0, err
		}
	}
	if gasCap := e.config.RPCGasCap; gasCap != 0 && highest > gasCap {
		log.Warn("Caller gas above allowance, capping", "requested", highest, "cap", gasCap)
		highest = gasCap
	}
	return runs.search(ctx, args, highest)
}

// estimationHeader reads the block of an estimation, wording a missing one
// as the legacy estimation does.
func (e *Environment) estimationHeader(head *evmcore.EvmBlock, reference chain.BlockReference) (*evmcore.EvmHeader, error) {
	header, err := e.FindHeader(head, reference)
	switch {
	case errors.Is(err, chain.ErrMissingSelector):
		return nil, errors.New("invalid block number or hash")
	case errors.Is(err, chain.ErrUnknownHash):
		return nil, fmt.Errorf("block with hash %s not found", reference.Hash.String())
	case IsMissing(err) || errors.Is(err, chain.ErrInvalidTag):
		return nil, errors.New("block does not exists")
	}
	return header, err
}

// highestGas is the gas given, or the transaction limit of the block after the
// head, capped by the limit from Osaka on; the block overrides decide the
// upgrade.
func (e *Environment) highestGas(head *evmcore.EvmBlock, header *evmcore.EvmHeader, args CallArguments, overrides *BlockOverrides) (uint64, error) {
	maxGasLimit, err := e.MaxGasLimit(head)
	if err != nil {
		return 0, err
	}
	highest := maxGasLimit
	if args.Gas != nil && uint64(*args.Gas) >= params.TxGas {
		highest = uint64(*args.Gas)
	}
	number, time := header.Number.Uint64(), uint64(header.Time.Unix())
	if overrides != nil && overrides.Number != nil {
		number = overrides.Number.ToInt().Uint64()
	}
	if overrides != nil && overrides.Time != nil {
		time = uint64(*overrides.Time)
	}
	if e.ChainConfig(number).IsOsaka(new(big.Int).SetUint64(number), time) {
		highest = min(highest, maxGasLimit)
	}
	return highest, nil
}

// feeCap is the most the call pays per gas.
func (a *CallArguments) feeCap() (*big.Int, error) {
	switch {
	case a.GasPrice != nil && (a.MaxFeePerGas != nil || a.MaxPriorityFeePerGas != nil):
		return nil, errBothPricings
	case a.GasPrice != nil:
		return a.GasPrice.ToInt(), nil
	case a.MaxFeePerGas != nil:
		return a.MaxFeePerGas.ToInt(), nil
	default:
		return common.Big0, nil
	}
}

// estimationRuns runs an estimation's call, each run on a copy of one state of
// the block with the state overrides applied, opened at the first run.
type estimationRuns struct {
	environment *Environment
	header      *evmcore.EvmHeader
	estimation  Estimation
	base        state.StateDB
}

func (r *estimationRuns) state() (state.StateDB, error) {
	if r.base != nil {
		return r.base, nil
	}
	base, err := r.environment.openState(r.header.Number.Uint64(), r.estimation.Block)
	if err != nil {
		return nil, err
	}
	if err := r.estimation.StateOverrides.Apply(base); err != nil {
		base.Release()
		return nil, err
	}
	r.base = base
	return base, nil
}

func (r *estimationRuns) release() {
	if r.base != nil {
		r.base.Release()
	}
}

// capByFunds caps the gas by what the sender's balance, less the value, pays
// at the fee cap.
func (r *estimationRuns) capByFunds(args CallArguments, feeCap *big.Int, highest uint64) (uint64, error) {
	base, err := r.state()
	if err != nil {
		return 0, err
	}
	balance := base.GetBalance(*args.From)
	available := utils.Uint256ToBigInt(balance)
	if args.Value != nil {
		if args.Value.ToInt().Cmp(available) >= 0 {
			return 0, errors.New("insufficient funds for transfer")
		}
		available.Sub(available, args.Value.ToInt())
	}
	allowance := new(big.Int).Div(available, feeCap)
	if allowance.IsUint64() && highest > allowance.Uint64() {
		transfer := args.Value
		if transfer == nil {
			transfer = new(hexutil.Big)
		}
		log.Warn("Gas estimation capped by limited funds", "original", highest, "balance", balance,
			"sent", transfer.ToInt(), "maxFeePerGas", feeCap, "fundable", allowance)
		return allowance.Uint64(), nil
	}
	return highest, nil
}

// run runs the call with the gas, and tells whether it failed. A refusal for
// too little or too much gas is a failure to search on; another refusal ends
// the search.
func (r *estimationRuns) run(ctx context.Context, args CallArguments, gas uint64) (bool, *core.ExecutionResult, error) {
	base, err := r.state()
	if err != nil {
		return true, nil, err
	}
	branch := base.Copy()
	defer branch.Release()
	args.Gas = (*hexutil.Uint64)(&gas)
	result, err := r.environment.Execute(ctx, branch, r.header, Call{
		Arguments:      args,
		BlockOverrides: r.estimation.BlockOverrides,
		Before:         r.estimation.Before,
	})
	switch {
	case errors.Is(err, core.ErrIntrinsicGas), errors.Is(err, core.ErrFloorDataGas), errors.Is(err, core.ErrGasLimitTooHigh):
		return true, nil, nil
	case err != nil:
		return true, nil, err
	}
	return result.Failed(), result, nil
}

// search halves the range of gas between the intrinsic gas and the highest
// until it holds one value, and answers why the call fails at the highest
// when it does.
func (r *estimationRuns) search(ctx context.Context, args CallArguments, highest uint64) (hexutil.Uint64, error) {
	lowest, gasCap := params.TxGas-1, highest
	for lowest+1 < highest {
		middle := (highest + lowest) / 2
		failed, _, err := r.run(ctx, args, middle)
		if err != nil {
			return 0, err
		}
		if failed {
			lowest = middle
		} else {
			highest = middle
		}
	}
	if highest != gasCap {
		return hexutil.Uint64(highest), nil
	}
	failed, result, err := r.run(ctx, args, highest)
	if err != nil {
		return 0, err
	}
	if !failed {
		return hexutil.Uint64(highest), nil
	}
	if result != nil && result.Err != vm.ErrOutOfGas {
		if len(result.Revert()) > 0 {
			return 0, NewRevertError(result)
		}
		return 0, result.Err
	}
	return 0, fmt.Errorf("gas required exceeds allowance (%d)", gasCap)
}
