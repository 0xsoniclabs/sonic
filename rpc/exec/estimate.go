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
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/utils"
)

// Estimation is a request to estimate the gas of a message on top of a
// block, after the previous messages, with the client's overrides.
type Estimation struct {
	Arguments      CallArguments
	Block          chain.BlockReference
	StateOverrides StateOverride
	BlockOverrides *BlockOverrides
	Previous       []CallArguments
}

// EstimateGas searches the least gas a message needs, as the reference's
// DoEstimateGas did: the highest allowance from the message's gas, the
// transaction gas limit of the open epoch from Osaka on, the funds of the
// sender and the gas cap; then a binary search from 21000 up to it, each try
// after the previous messages. The state is opened and overridden once and
// every try runs on a copy (I8); the rules and the block are read once (H2).
func (environment *Environment) EstimateGas(head Head, estimation Estimation) (hexutil.Uint64, error) {
	arguments := estimation.Arguments
	if arguments.From == nil {
		arguments.From = new(common.Address)
	}
	maxTransactionGas := sync.OnceValues(func() (uint64, error) {
		rules, err := environment.RulesAt(head, head.Number+1)
		return rules.MaxTransactionGas(), err
	})
	highest, err := highestAllowance(arguments, maxTransactionGas)
	if err != nil {
		return 0, err
	}
	header, err := environment.estimationHeader(head, estimation.Block)
	if err != nil {
		return 0, err
	}
	if highest, err = environment.capByOsaka(header, estimation.BlockOverrides, highest, maxTransactionGas); err != nil {
		return 0, err
	}
	feeCap, err := arguments.feeCap()
	if err != nil {
		return 0, err
	}
	statedb, err := environment.OpenState(head, header.Number.Uint64())
	if err != nil {
		return 0, StateLookupError(err)
	}
	defer statedb.Release()
	if err := estimation.StateOverrides.Apply(statedb); err != nil {
		return 0, err
	}
	if highest, err = capByFunds(statedb, arguments, feeCap, highest); err != nil {
		return 0, err
	}
	gasCap := environment.config.RPCGasCap
	if gasCap != 0 && highest > gasCap {
		log.Warn("Caller gas above allowance, capping", "requested", highest, "cap", gasCap)
		highest = gasCap
	}
	call, err := environment.PrepareCall(head, header, estimation.BlockOverrides)
	if err != nil {
		return 0, err
	}
	search := &gasSearch{
		ctx:       head.Context(),
		call:      call,
		state:     statedb,
		arguments: arguments,
		previous:  Messages(estimation.Previous, gasCap, header),
		gasCap:    gasCap,
	}
	return search.run(highest)
}

// highestAllowance is the gas of the message if it is at least the gas of a
// transfer, the transaction gas limit otherwise.
func highestAllowance(arguments CallArguments, maxTransactionGas func() (uint64, error)) (uint64, error) {
	if arguments.Gas != nil && uint64(*arguments.Gas) >= params.TxGas {
		return uint64(*arguments.Gas), nil
	}
	return maxTransactionGas()
}

// estimationHeader returns the header of the block of an estimation, with
// the wordings of the reference's lookup of its number and time (F3).
func (environment *Environment) estimationHeader(head Head, reference chain.BlockReference) (*evmcore.EvmHeader, error) {
	header, err := environment.FindHeader(head, reference)
	switch {
	case errors.Is(err, chain.ErrNoSelector):
		return nil, errors.New("invalid block number or hash")
	case errors.Is(err, chain.ErrUnknownHash):
		return nil, UnknownHashError(*reference.Hash)
	case Absent(err):
		return nil, errors.New("block does not exists")
	}
	return header, err
}

// capByOsaka caps the allowance by the transaction gas limit when the block,
// with the number and time the overrides give it, is under Osaka (EIP-7825).
func (environment *Environment) capByOsaka(header *evmcore.EvmHeader, overrides *BlockOverrides, highest uint64, maxTransactionGas func() (uint64, error)) (uint64, error) {
	number, time := header.Number.Uint64(), uint64(header.Time.Unix())
	if overrides != nil && overrides.Number != nil {
		number = overrides.Number.ToInt().Uint64()
	}
	if overrides != nil && overrides.Time != nil {
		time = uint64(*overrides.Time)
	}
	if !environment.ChainConfig(number).IsOsaka(big.NewInt(int64(number)), time) {
		return highest, nil
	}
	limit, err := maxTransactionGas()
	if err != nil {
		return 0, err
	}
	return min(highest, limit), nil
}

// feeCap is the most a message pays per gas, zero if it names no price.
func (arguments *CallArguments) feeCap() (*big.Int, error) {
	switch {
	case arguments.GasPrice != nil && (arguments.MaxFeePerGas != nil || arguments.MaxPriorityFeePerGas != nil):
		return nil, errBothPriceStyles
	case arguments.GasPrice != nil:
		return arguments.GasPrice.ToInt(), nil
	case arguments.MaxFeePerGas != nil:
		return arguments.MaxFeePerGas.ToInt(), nil
	}
	return common.Big0, nil
}

// capByFunds caps the allowance by the gas the sender can pay for at the fee
// cap after the value it transfers.
func capByFunds(statedb state.StateDB, arguments CallArguments, feeCap *big.Int, highest uint64) (uint64, error) {
	if feeCap.BitLen() == 0 {
		return highest, nil
	}
	balance := statedb.GetBalance(*arguments.From)
	available := utils.Uint256ToBigInt(balance)
	if arguments.Value != nil {
		if arguments.Value.ToInt().Cmp(available) >= 0 {
			return 0, errors.New("insufficient funds for transfer")
		}
		available.Sub(available, arguments.Value.ToInt())
	}
	allowance := new(big.Int).Div(available, feeCap)
	if !allowance.IsUint64() || highest <= allowance.Uint64() {
		return highest, nil
	}
	transfer := arguments.Value
	if transfer == nil {
		transfer = new(hexutil.Big)
	}
	log.Warn("Gas estimation capped by limited funds", "original", highest, "balance", balance,
		"sent", transfer.ToInt(), "maxFeePerGas", feeCap, "fundable", allowance)
	return allowance.Uint64(), nil
}

// gasSearch is the binary search of an estimation.
type gasSearch struct {
	ctx       context.Context
	call      *Call
	state     state.StateDB
	arguments CallArguments
	previous  []*core.Message
	gasCap    uint64
}

func (search *gasSearch) run(highest uint64) (hexutil.Uint64, error) {
	lowest, allowance := params.TxGas-1, highest
	for lowest+1 < highest {
		middle := (highest + lowest) / 2
		failed, _, err := search.executable(middle)
		if err != nil {
			return 0, err
		}
		if failed {
			lowest = middle
		} else {
			highest = middle
		}
	}
	if highest == allowance {
		if err := search.checkAllowance(allowance); err != nil {
			return 0, err
		}
	}
	return hexutil.Uint64(highest), nil
}

// checkAllowance rejects a message that fails with all the gas it may have:
// with its revert, its error, or, if it ran out of gas, the allowance.
func (search *gasSearch) checkAllowance(allowance uint64) error {
	failed, result, err := search.executable(allowance)
	if err != nil || !failed {
		return err
	}
	if result != nil && result.Err != vm.ErrOutOfGas {
		if len(result.Revert()) > 0 {
			return RevertError(result)
		}
		return result.Err
	}
	return fmt.Errorf("gas required exceeds allowance (%d)", allowance)
}

// executable runs the message with the given gas on a copy of the state. Too
// little gas for the message's intrinsic or floor data cost, or more than a
// transaction may have, is a failure to search past; any other error ends
// the search.
func (search *gasSearch) executable(gas uint64) (failed bool, result *core.ExecutionResult, err error) {
	search.arguments.Gas = (*hexutil.Uint64)(&gas)
	message, err := search.arguments.ToMessage(search.gasCap, search.call.Header.BaseFee)
	if err != nil {
		return true, nil, err
	}
	statedb := search.state.Copy()
	defer statedb.Release()
	result, err = search.call.Run(search.ctx, statedb, search.previous, message, 0)
	if err != nil {
		if errors.Is(err, core.ErrIntrinsicGas) || errors.Is(err, core.ErrFloorDataGas) || errors.Is(err, core.ErrGasLimitTooHigh) {
			return true, nil, nil
		}
		return true, nil, err
	}
	return result.Failed(), result, nil
}
