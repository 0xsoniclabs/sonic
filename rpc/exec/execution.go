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
	"fmt"
	"math"
	"time"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
)

// BlockContext is the block context of a run on top of a block. BLOCKHASH
// reads one header by number, and answers the zero hash at and above the
// block, so that no value of a client makes it walk the chain.
func (e *Environment) BlockContext(header *evmcore.EvmHeader) vm.BlockContext {
	blockContext := evmcore.NewEVMBlockContext(header, nil, nil)
	blockContext.GetHash = e.blockHash(header.Number.Uint64())
	return blockContext
}

func (e *Environment) blockHash(top uint64) vm.GetHashFunc {
	return func(number uint64) common.Hash {
		if number >= top {
			return common.Hash{}
		}
		header, err := e.reader.Header(number)
		if err != nil {
			return common.Hash{}
		}
		return header.Hash
	}
}

// NewEVM is an EVM on the state of a block, under the block's rules and with
// the chain configuration of the block context's number, block overrides
// applied.
func (e *Environment) NewEVM(stateDB state.StateDB, header *evmcore.EvmHeader, overrides *BlockOverrides, vmConfig vm.Config) *vm.EVM {
	blockContext := e.BlockContext(header)
	overrides.apply(&blockContext)
	return vm.NewEVM(blockContext, stateDB, e.ChainConfig(blockContext.BlockNumber.Uint64()), vmConfig)
}

// Call is a call to run on top of a block.
type Call struct {
	Arguments      CallArguments
	BlockOverrides *BlockOverrides
	// Timeout bounds the run; zero leaves it to the request.
	Timeout time.Duration
	// Before are calls run first, whose failures are only logged.
	Before []CallArguments
}

// Execute runs a call on a state of the block, which it changes. A run
// stopped by the timeout or the request fails; a message the EVM refuses is
// reported with the gas it was given.
func (e *Environment) Execute(ctx context.Context, stateDB state.StateDB, header *evmcore.EvmHeader, call Call) (*core.ExecutionResult, error) {
	var cancel context.CancelFunc
	if call.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, call.Timeout)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}
	defer cancel()

	message, err := call.Arguments.ToMessage(e.config.RPCGasCap, header.BaseFee)
	if err != nil {
		return nil, err
	}
	vmConfig, err := e.VMConfig(header.Number.Uint64())
	if err != nil {
		return nil, err
	}
	evm := e.NewEVM(stateDB, header, call.BlockOverrides, vmConfig)
	evm.Config.NoBaseFee = true
	stop := context.AfterFunc(ctx, evm.Cancel)
	defer stop()

	gasPool := core.NewGasPool(math.MaxUint64)
	for i, before := range call.Before {
		beforeMessage, err := before.ToMessage(e.config.RPCGasCap, header.BaseFee)
		if err != nil {
			log.Warn("failed to convert previous transaction to message", "index", i, "err", err)
			continue
		}
		if _, err := core.ApplyMessage(evm, beforeMessage, gasPool); err != nil {
			log.Warn("failed to apply previous transaction", "index", i, "err", err)
		}
		stateDB.EndTransaction()
	}
	result, err := core.ApplyMessage(evm, message, gasPool)
	if evm.Cancelled() {
		return nil, fmt.Errorf("execution aborted (timeout = %v)", call.Timeout)
	}
	if err != nil {
		return result, fmt.Errorf("err: %w (supplied gas %d)", err, message.GasLimit)
	}
	return result, nil
}

// RevertError is a reverted call: code 3, with the revert data, and the
// reason in its message when the data carries one.
type RevertError struct {
	message string
	data    string
}

// NewRevertError words the revert of a result.
func NewRevertError(result *core.ExecutionResult) *RevertError {
	message := "execution reverted"
	if reason, err := abi.UnpackRevert(result.Revert()); err == nil {
		message = fmt.Sprintf("execution reverted: %v", reason)
	}
	return &RevertError{message: message, data: hexutil.Encode(result.Revert())}
}

func (e *RevertError) Error() string { return e.message }

// ErrorCode is the JSON-RPC code of a revert.
func (e *RevertError) ErrorCode() int { return 3 }

// ErrorData is the revert data in hex.
func (e *RevertError) ErrorData() any { return e.data }
