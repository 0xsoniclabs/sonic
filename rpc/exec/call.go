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
	"math"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
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

// WithTimeout bounds a context by a timeout, or only makes it cancellable if
// the timeout is zero.
func WithTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout > 0 {
		return context.WithTimeout(ctx, timeout)
	}
	return context.WithCancel(ctx)
}

// Call runs a message call on the state of a block: the block first, then the
// state overrides, then the call object, as the reference checks them. A
// failed message is wrapped as "err: … (supplied gas …)".
func (e *Env) Call(ctx context.Context, head *Head, args CallArgs, ref chain.BlockRef, overrides *StateOverride, blockOverrides *BlockOverrides, timeout time.Duration, gasCap uint64, preArgs []CallArgs) (*core.ExecutionResult, error) {
	st, header, err := e.StateAt(head, ref)
	if err != nil {
		return nil, HeaderNotFound(err)
	}
	defer st.Release()
	if err := overrides.Apply(st); err != nil {
		return nil, err
	}
	return e.callOn(ctx, st, header, args, blockOverrides, timeout, gasCap, preArgs)
}

// callOn runs the calls of preArgs and then args on a state, ending a
// transaction after each.
func (e *Env) callOn(ctx context.Context, st state.StateDB, header *evmcore.EvmHeader, args CallArgs, blockOverrides *BlockOverrides, timeout time.Duration, gasCap uint64, preArgs []CallArgs) (*core.ExecutionResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err // no EVM run after the request ended
	}
	ctx, cancel := WithTimeout(ctx, timeout)
	defer cancel()

	msg, err := args.ToMessage(gasCap, header.BaseFee)
	if err != nil {
		return nil, err
	}
	vmConfig, err := e.VMConfig(ctx, header.Number.Uint64())
	if err != nil {
		return nil, err
	}
	vmConfig.NoBaseFee = true
	blockCtx := e.BlockContext(header)
	blockOverrides.Apply(&blockCtx)
	evm := vm.NewEVM(blockCtx, st, e.ChainConfig(blockCtx.BlockNumber.Uint64()), vmConfig)
	stop := context.AfterFunc(ctx, evm.Cancel)
	defer stop()

	gp := core.NewGasPool(math.MaxUint64)
	for i, arg := range preArgs {
		preMsg, err := arg.ToMessage(gasCap, header.BaseFee)
		if err != nil {
			log.Warn("failed to convert previous transaction to message", "index", i, "err", err)
			continue
		}
		if _, err := core.ApplyMessage(evm, preMsg, gp); err != nil {
			log.Warn("failed to apply previous transaction", "index", i, "err", err)
		}
		st.EndTransaction()
	}

	result, err := core.ApplyMessage(evm, msg, gp)
	if evm.Cancelled() {
		return nil, fmt.Errorf("execution aborted (timeout = %v)", timeout)
	}
	if err != nil {
		return result, fmt.Errorf("err: %w (supplied gas %d)", err, msg.GasLimit)
	}
	return result, nil
}

// EstimateGas searches the lowest gas a call succeeds with, between the
// intrinsic gas of a transfer and the lowest of the call's gas, the largest
// transaction of the open epoch, the funds of the sender and the gas cap.
// Every step of the search runs on a copy of one state of the block.
func (e *Env) EstimateGas(ctx context.Context, head *Head, args CallArgs, ref chain.BlockRef, overrides *StateOverride, blockOverrides *BlockOverrides, gasCap uint64, preArgs []CallArgs) (hexutil.Uint64, error) {
	var (
		lo uint64 = params.TxGas - 1
		hi uint64
	)
	if args.From == nil {
		args.From = new(common.Address)
	}

	header, err := e.FindHeader(head, ref)
	switch {
	case errors.Is(err, chain.ErrNoSelector):
		return 0, errors.New("invalid block number or hash")
	case errors.Is(err, chain.ErrUnknownHash):
		return 0, UnknownHashError(*ref.Hash)
	case IsMissing(err):
		return 0, errors.New("block does not exists")
	case err != nil:
		return 0, err
	}

	maxGasLimit, err := e.MaxGasLimit(head)
	if err != nil {
		return 0, err
	}
	if args.Gas != nil && uint64(*args.Gas) >= params.TxGas {
		hi = uint64(*args.Gas)
	} else {
		hi = maxGasLimit
	}
	// From Osaka on, EIP-7825 caps the gas of a transaction, by the largest
	// transaction of the epoch in Sonic.
	if e.isOsaka(header, blockOverrides) {
		hi = min(hi, maxGasLimit)
	}

	var feeCap *big.Int
	switch {
	case args.GasPrice != nil && (args.MaxFeePerGas != nil || args.MaxPriorityFeePerGas != nil):
		return 0, errBothFeeKinds
	case args.GasPrice != nil:
		feeCap = args.GasPrice.ToInt()
	case args.MaxFeePerGas != nil:
		feeCap = args.MaxFeePerGas.ToInt()
	default:
		feeCap = common.Big0
	}

	var base state.StateDB
	defer func() {
		if base != nil {
			base.Release()
		}
	}()
	openBase := func() error {
		if base != nil {
			return nil
		}
		st, err := e.StateAtNumber(head, header.Number.Uint64())
		if err != nil {
			return HeaderNotFound(err)
		}
		base = st
		return overrides.Apply(base)
	}

	// Cap the gas by the funds of the sender.
	if feeCap.BitLen() != 0 {
		if err := openBase(); err != nil {
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
		if allowance.IsUint64() && hi > allowance.Uint64() {
			transfer := args.Value
			if transfer == nil {
				transfer = new(hexutil.Big)
			}
			log.Warn("Gas estimation capped by limited funds", "original", hi, "balance", balance,
				"sent", transfer.ToInt(), "maxFeePerGas", feeCap, "fundable", allowance)
			hi = allowance.Uint64()
		}
	}
	if gasCap != 0 && hi > gasCap {
		log.Warn("Caller gas above allowance, capping", "requested", hi, "cap", gasCap)
		hi = gasCap
	}
	limit := hi

	// executable reports whether the call fails with the given gas; an error
	// means it fails with any gas.
	executable := func(gas uint64) (bool, *core.ExecutionResult, error) {
		if err := openBase(); err != nil {
			return true, nil, err
		}
		args.Gas = (*hexutil.Uint64)(&gas)
		st := base.Copy()
		defer st.Release()
		result, err := e.callOn(ctx, st, header, args, blockOverrides, 0, gasCap, preArgs)
		if err != nil {
			if errors.Is(err, core.ErrIntrinsicGas) || errors.Is(err, core.ErrFloorDataGas) {
				return true, nil, nil // raise the gas
			}
			if errors.Is(err, core.ErrGasLimitTooHigh) {
				return true, nil, nil // lower the gas
			}
			return true, nil, err
		}
		return result.Failed(), result, nil
	}
	for lo+1 < hi {
		mid := (hi + lo) / 2
		failed, _, err := executable(mid)
		if err != nil {
			return 0, err
		}
		if failed {
			lo = mid
		} else {
			hi = mid
		}
	}
	if hi == limit {
		failed, result, err := executable(hi)
		if err != nil {
			return 0, err
		}
		if failed {
			if result != nil && result.Err != vm.ErrOutOfGas {
				if len(result.Revert()) > 0 {
					return 0, NewRevertError(result)
				}
				return 0, result.Err
			}
			return 0, fmt.Errorf("gas required exceeds allowance (%d)", limit)
		}
	}
	return hexutil.Uint64(hi), nil
}

// isOsaka reports whether the call's block, or the block number and time its
// overrides set, follows Osaka.
func (e *Env) isOsaka(header *evmcore.EvmHeader, blockOverrides *BlockOverrides) bool {
	number, timestamp := header.Number.Uint64(), uint64(header.Time.Unix())
	if blockOverrides != nil {
		if blockOverrides.Number != nil {
			number = blockOverrides.Number.ToInt().Uint64()
		}
		if blockOverrides.Time != nil {
			timestamp = uint64(*blockOverrides.Time)
		}
	}
	return e.ChainConfig(number).IsOsaka(new(big.Int).SetUint64(number), timestamp)
}

// RevertError is a reverted call: code 3, with the revert data.
type RevertError struct {
	error
	reason string // hex encoded revert data
}

// NewRevertError describes the revert of a call, with its reason if the data
// decodes to one.
func NewRevertError(result *core.ExecutionResult) *RevertError {
	reason, errUnpack := abi.UnpackRevert(result.Revert())
	err := errors.New("execution reverted")
	if errUnpack == nil {
		err = fmt.Errorf("execution reverted: %v", reason)
	}
	return &RevertError{error: err, reason: hexutil.Encode(result.Revert())}
}

// ErrorCode is the code of a revert.
func (e *RevertError) ErrorCode() int {
	return 3
}

// ErrorData is the hex encoded revert data.
func (e *RevertError) ErrorData() any {
	return e.reason
}
