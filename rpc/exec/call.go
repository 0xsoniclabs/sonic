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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/tracers/logger"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/utils"
)

// DoCall executes a message on the state of a block and returns the result of
// the execution. The order of the checks is the one of the node's earlier
// handlers: the block, then the state overrides, then the call object.
//
// preArgs are messages applied before the call, on the same state, whose
// failures are logged and skipped.
func DoCall(
	ctx context.Context,
	env *Env,
	head dispatch.Head,
	args CallArgs,
	ref dispatch.BlockRef,
	overrides *StateOverride,
	blockOverrides *BlockOverrides,
	timeout time.Duration,
	globalGasCap uint64,
	preArgs []CallArgs,
) (*core.ExecutionResult, error) {
	defer func(start time.Time) { log.Debug("Executing EVM call finished", "runtime", time.Since(start)) }(time.Now())

	statedb, header, err := env.StateAt(head, ref)
	if err != nil {
		return nil, err
	}
	defer statedb.Release()
	if err := overrides.Apply(statedb); err != nil {
		return nil, err
	}
	// Setup context so it may be cancelled when the call has completed or, in
	// case of unmetered gas, setup a context with a timeout.
	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}
	// Make sure the context is cancelled when the call has completed; this
	// makes sure resources are cleaned up.
	defer cancel()

	return execute(ctx, env, head, statedb, header, args, overrides, blockOverrides, timeout, globalGasCap, preArgs)
}

// execute runs a message on an opened, overridden state.
func execute(
	ctx context.Context,
	env *Env,
	head dispatch.Head,
	statedb state.StateDB,
	header *evmcore.EvmHeader,
	args CallArgs,
	overrides *StateOverride,
	blockOverrides *BlockOverrides,
	timeout time.Duration,
	globalGasCap uint64,
	preArgs []CallArgs,
) (*core.ExecutionResult, error) {
	// Get a new instance of the EVM.
	msg, err := args.ToMessage(globalGasCap, header.BaseFee, log.Root())
	if err != nil {
		return nil, err
	}
	vmConfig, err := env.VMConfig(header.Number.Uint64())
	if err != nil {
		return nil, err
	}
	if overrides.HasCodesExceedingOnChainLimit() {
		// Use geth as VM for computation
		vmConfig.Tracer = &tracing.Hooks{}
	}

	var blockCtx *vm.BlockContext
	if blockOverrides != nil {
		bctx := env.BlockContext(head, header)
		blockOverrides.Apply(&bctx)
		blockCtx = &bctx
	}
	evm, err := env.NewEVM(head, statedb, header, &vmConfig, blockCtx)
	if err != nil {
		return nil, err
	}
	// Skip gas price checks for API runs.
	evm.Config.NoBaseFee = true
	// Cancel the evm when the context is done. Even if the EVM has finished,
	// cancelling may be done (repeatedly).
	stopCancel := Cancel(ctx, evm)
	defer stopCancel()

	// execute EIP-2935 HistoryStorage contract.
	if evm.ChainConfig().IsPrague(header.Number, uint64(header.Time.Unix())) {
		evmcore.ProcessParentBlockHash(header.ParentHash, evm, statedb)
	}

	// Add sufficient gas to the pool.
	gp := core.NewGasPool(math.MaxUint64)

	// Apply previous transactions if required.
	for i, arg := range preArgs {
		preMsg, err := arg.ToMessage(globalGasCap, header.BaseFee, log.Root())
		if err != nil {
			log.Warn("failed to convert previous transaction to message", "index", i, "err", err)
			continue
		}
		if _, err := core.ApplyMessage(evm, preMsg, gp); err != nil {
			log.Warn("failed to apply previous transaction", "index", i, "err", err)
		}
		statedb.EndTransaction()
	}

	// Execute the message.
	result, err := core.ApplyMessage(evm, msg, gp)

	// If the timer caused an abort, return an appropriate error message
	if evm.Cancelled() {
		return nil, fmt.Errorf("execution aborted (timeout = %v)", timeout)
	}
	if err != nil {
		return result, fmt.Errorf("err: %w (supplied gas %d)", err, msg.GasLimit)
	}
	return result, nil
}

// Call runs eth_call: a message executed on the state of a block.
func Call(ctx context.Context, env *Env, head dispatch.Head, args CallArgs, ref dispatch.BlockRef,
	overrides *StateOverride, blockOverrides *BlockOverrides) (hexutil.Bytes, error) {
	result, err := DoCall(ctx, env, head, args, ref, overrides, blockOverrides, env.Cfg.EVMTimeout, env.Cfg.GasCap, nil)
	if err != nil {
		return nil, err
	}
	// If the result contains a revert reason, try to unpack and return it.
	if len(result.Revert()) > 0 {
		return nil, NewRevertError(result)
	}
	return result.Return(), result.Err
}

// DoEstimateGas binary-searches the gas requirement of a message, as it may be
// higher than the amount used.
func DoEstimateGas(
	ctx context.Context,
	env *Env,
	head dispatch.Head,
	args CallArgs,
	ref dispatch.BlockRef,
	overrides *StateOverride,
	blockOverrides *BlockOverrides,
	gasCap uint64,
	preArgs []CallArgs,
) (hexutil.Uint64, error) {
	// Binary search the gas requirement, as it may be higher than the amount used
	var (
		lo  uint64 = params.TxGas - 1
		hi  uint64
		cap uint64
	)
	// Use zero address if sender unspecified.
	if args.From == nil {
		args.From = new(common.Address)
	}
	// Determine the highest gas limit can be used during the estimation.
	if args.Gas != nil && uint64(*args.Gas) >= params.TxGas {
		hi = uint64(*args.Gas)
	} else {
		hi = env.MaxGasLimit(head)
	}

	// Cap the maximum gas allowance according to EIP-7825. In Sonic the max gas
	// limit for a transaction is limited by MaxGasLimit starting with Osaka.
	hi, err := capMaxGas(env, head, ref, blockOverrides, hi)
	if err != nil {
		return 0, err
	}

	// Normalize the max fee per gas the call is willing to spend.
	var feeCap *big.Int
	if args.GasPrice != nil && (args.MaxFeePerGas != nil || args.MaxPriorityFeePerGas != nil) {
		return 0, errors.New("both gasPrice and (maxFeePerGas or maxPriorityFeePerGas) specified")
	} else if args.GasPrice != nil {
		feeCap = args.GasPrice.ToInt()
	} else if args.MaxFeePerGas != nil {
		feeCap = args.MaxFeePerGas.ToInt()
	} else {
		feeCap = common.Big0
	}
	// Recap the highest gas limit with account's available balance.
	if feeCap.BitLen() != 0 {
		allowance, err := fundableGas(env, head, ref, overrides, &args, feeCap, hi)
		if err != nil {
			return 0, err
		}
		hi = allowance
	}
	// Recap the highest gas allowance with specified gascap.
	if gasCap != 0 && hi > gasCap {
		log.Warn("Caller gas above allowance, capping", "requested", hi, "cap", gasCap)
		hi = gasCap
	}
	cap = hi

	// Create a helper to check if a gas allowance results in an executable transaction
	executable := func(gas uint64) (bool, *core.ExecutionResult, error) {
		args.Gas = (*hexutil.Uint64)(&gas)

		result, err := DoCall(ctx, env, head, args, ref, overrides, blockOverrides, 0, gasCap, preArgs)
		if err != nil {
			if errors.Is(err, core.ErrIntrinsicGas) ||
				errors.Is(err, core.ErrFloorDataGas) {
				return true, nil, nil // Special case, raise gas limit
			}
			if errors.Is(err, core.ErrGasLimitTooHigh) {
				return true, nil, nil // Special case, lower gas limit
			}
			return true, nil, err // Bail out
		}
		return result.Failed(), result, nil
	}
	// Execute the binary search and hone in on an executable gas limit
	for lo+1 < hi {
		mid := (hi + lo) / 2
		failed, _, err := executable(mid)

		// If the error is not nil(consensus error), it means the provided message
		// call or transaction will never be accepted no matter how much gas it is
		// assigned. Return the error directly, don't struggle any more.
		if err != nil {
			return 0, err
		}
		if failed {
			lo = mid
		} else {
			hi = mid
		}
	}
	// Reject the transaction as invalid if it still fails at the highest allowance
	if hi == cap {
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
			// Otherwise, the specified gas cap is too low
			return 0, fmt.Errorf("gas required exceeds allowance (%d)", cap)
		}
	}
	return hexutil.Uint64(hi), nil
}

// fundableGas caps the gas limit hi with what the sender can pay for.
func fundableGas(env *Env, head dispatch.Head, ref dispatch.BlockRef, overrides *StateOverride, args *CallArgs, feeCap *big.Int, hi uint64) (uint64, error) {
	statedb, _, err := env.StateAt(head, ref)
	if err != nil {
		return 0, err
	}
	defer statedb.Release()
	if err := overrides.Apply(statedb); err != nil {
		return 0, err
	}
	balance := statedb.GetBalance(*args.From) // from can't be nil
	available := utils.Uint256ToBigInt(balance)
	if args.Value != nil {
		if args.Value.ToInt().Cmp(available) >= 0 {
			return 0, errors.New("insufficient funds for transfer")
		}
		available.Sub(available, args.Value.ToInt())
	}
	allowance := new(big.Int).Div(available, feeCap)

	// If the allowance is larger than maximum uint64, skip checking
	if allowance.IsUint64() && hi > allowance.Uint64() {
		transfer := args.Value
		if transfer == nil {
			transfer = new(hexutil.Big)
		}
		log.Warn("Gas estimation capped by limited funds", "original", hi, "balance", balance,
			"sent", transfer.ToInt(), "maxFeePerGas", feeCap, "fundable", allowance)
		hi = allowance.Uint64()
	}
	return hi, nil
}

// capMaxGas determines the maximum gas limit that can be used for a
// transaction, applying the Osaka gas limit rule if applicable. It checks if the
// given block is subject to the Osaka rule, and if so, caps the gas limit to
// the maximum allowed gas limit.
func capMaxGas(env *Env, head dispatch.Head, ref dispatch.BlockRef, blockOverrides *BlockOverrides, hi uint64) (uint64, error) {
	if osaka, err := isOsaka(env, head, ref, blockOverrides); err != nil {
		return 0, err
	} else if osaka {
		hi = min(hi, env.MaxGasLimit(head))
	}
	return hi, nil
}

// isOsaka checks if the given block corresponds to a block that uses the Osaka
// gas limit rule. The block number or its timestamp may be overridden by the
// given block overrides.
func isOsaka(env *Env, head dispatch.Head, ref dispatch.BlockRef, blockOverrides *BlockOverrides) (bool, error) {
	blockNumber, blockTime, err := numberAndTime(env, head, ref)
	if err != nil {
		return false, err
	}

	// check overrides
	if blockOverrides != nil {
		if blockOverrides.Number != nil {
			blockNumber = blockOverrides.Number.ToInt().Uint64()
		}
		if blockOverrides.Time != nil {
			blockTime = uint64((*blockOverrides.Time))
		}
	}

	// use Osaka gas limit rule
	return env.ChainConfig(blockNumber).IsOsaka(big.NewInt(int64(blockNumber)), blockTime), nil
}

// numberAndTime returns the block number and time for the given block
// reference. If the number or hash is invalid or does not exist, an error is
// returned.
func numberAndTime(env *Env, head dispatch.Head, ref dispatch.BlockRef) (uint64, uint64, error) {
	n, err := env.Chain.Resolve(head, ref)
	switch {
	case errors.Is(err, dispatch.ErrNoSelector):
		return 0, 0, errors.New("invalid block number or hash")
	case errors.Is(err, dispatch.ErrUnknownHash):
		return 0, 0, UnknownHashError(*ref.Hash)
	case err != nil:
		return 0, 0, errors.New("block does not exists")
	}
	header, err := env.HeaderAt(head, n)
	if err != nil {
		return 0, 0, err
	}
	if header == nil {
		return 0, 0, errors.New("block does not exists")
	}
	return header.Number.Uint64(), uint64(header.Time.Unix()), nil
}

// AccessList creates an access list for the given transaction. If the access
// list creation fails an error is returned. If the transaction itself fails, a
// vmErr is returned.
func AccessList(ctx context.Context, env *Env, head dispatch.Head, ref dispatch.BlockRef, args CallArgs) (acl types.AccessList, gasUsed uint64, vmErr error, err error) {
	// Retrieve the execution context
	db, header, err := env.StateAt(head, ref)
	if err != nil {
		return nil, 0, nil, err
	}
	defer db.Release()
	// If the gas amount is not set, extract this as it will depend on access
	// lists and we'll need to reestimate every time
	nogas := args.Gas == nil

	// Ensure any missing fields are filled, extract the recipient and input data
	if err := args.SetDefaults(ctx, env, head); err != nil {
		return nil, 0, nil, err
	}
	var to common.Address
	if args.To != nil {
		to = *args.To
	} else {
		to = crypto.CreateAddress(args.from(), uint64(*args.Nonce))
	}
	// Retrieve the precompiles since they don't need to be added to the access list
	chainConfig := env.ChainConfig(header.Number.Uint64())
	precompiles := vm.ActivePrecompiles(chainConfig.Rules(header.Number, false, uint64(header.Time.Unix())))

	// addressesToExclude contains sender, receiver and precompiles
	addressesToExclude := map[common.Address]struct{}{args.from(): {}, to: {}}
	for _, addr := range precompiles {
		addressesToExclude[addr] = struct{}{}
	}

	// Create an initial tracer
	prevTracer := logger.NewAccessListTracer(nil, addressesToExclude)
	if args.AccessList != nil {
		prevTracer = logger.NewAccessListTracer(*args.AccessList, addressesToExclude)
	}
	for {
		// Retrieve the current access list to expand
		accessList := prevTracer.AccessList()
		log.Trace("Creating access list", "input", accessList)

		// If no gas amount was specified, each unique access list needs it's own
		// gas calculation. This is quite expensive, but we need to be accurate
		// and it's convered by the sender only anyway.
		if nogas {
			args.Gas = nil
			if err := args.SetDefaults(ctx, env, head); err != nil {
				return nil, 0, nil, err // shouldn't happen, just in case
			}
		}
		// Copy the original db so we don't modify it
		statedb := db.Copy()
		// Set the accesslist to the last al
		args.AccessList = &accessList
		msg, err := args.ToMessage(env.Cfg.GasCap, header.BaseFee, log.Root())
		if err != nil {
			statedb.Release()
			return nil, 0, nil, err
		}

		// Apply the transaction with the access list tracer
		tracer := logger.NewAccessListTracer(accessList, addressesToExclude)
		config, err := env.VMConfig(header.Number.Uint64())
		if err != nil {
			statedb.Release()
			return nil, 0, nil, err
		}
		config.Tracer = tracer.Hooks()
		config.NoBaseFee = true
		vmenv, err := env.NewEVM(head, statedb, header, &config, nil)
		if err != nil {
			statedb.Release()
			return nil, 0, nil, err
		}
		stopCancel := Cancel(ctx, vmenv)
		res, err := core.ApplyMessage(vmenv, msg, core.NewGasPool(msg.GasLimit))
		stopCancel()
		statedb.Release()
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, 0, nil, ctxErr
		}
		if err != nil {
			return nil, 0, nil, fmt.Errorf("failed to apply transaction from sender %v and nonce %d: %w",
				args.from(), uint64(*args.Nonce), err)
		}
		if tracer.Equal(prevTracer) {
			return accessList, res.UsedGas, res.Err, nil
		}
		prevTracer = tracer
	}
}
