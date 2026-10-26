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
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
)

// Call is the block a request runs messages on top of: its header, the EVM
// configuration of its epoch and the block context with the client's
// overrides, read once for every run of the request (H2).
type Call struct {
	Header      *evmcore.EvmHeader
	config      vm.Config
	context     vm.BlockContext
	chainConfig *params.ChainConfig
}

// PrepareCall reads what the runs of a call on top of a block need. The EVM
// configuration comes from the rules of the block's epoch; the chain
// configuration from the block number of the context, which an override may
// change, as in the reference (I12).
func (environment *Environment) PrepareCall(head Head, header *evmcore.EvmHeader, overrides *BlockOverrides) (*Call, error) {
	rules, err := environment.RulesAt(head, header.Number.Uint64())
	if err != nil {
		return nil, err
	}
	context := environment.BlockContext(head, header)
	overrides.Apply(&context)
	return &Call{
		Header:      header,
		config:      opera.GetVmConfig(rules),
		context:     context,
		chainConfig: environment.ChainConfig(context.BlockNumber.Uint64()),
	}, nil
}

// BlockContext returns the EVM's context of a block of the chain. BLOCKHASH
// is answered by one lookup by number, with the zero hash at and above the
// block, whatever block number an override sets (V11).
func (environment *Environment) BlockContext(head Head, header *evmcore.EvmHeader) vm.BlockContext {
	context := evmcore.NewEVMBlockContext(header, nil, nil)
	context.GetHash = environment.blockHashes(head, header.Number.Uint64())
	return context
}

func (environment *Environment) blockHashes(head Head, below uint64) vm.GetHashFunc {
	return func(number uint64) common.Hash {
		if number >= below {
			return common.Hash{}
		}
		header, err := environment.HeaderAt(head, number)
		if err != nil {
			return common.Hash{}
		}
		return header.Hash
	}
}

// newEVM returns an EVM for runs on the state, with the hooks of a tracer if
// any and without the base fee check of calls. It is cancelled when ctx ends;
// stop ends the watch.
func (call *Call) newEVM(ctx context.Context, statedb state.StateDB, hooks *tracing.Hooks) (evm *vm.EVM, stop func() bool) {
	config := call.config
	config.Tracer = hooks
	config.NoBaseFee = true
	evm = vm.NewEVM(call.context, statedb, call.chainConfig, config)
	return evm, context.AfterFunc(ctx, evm.Cancel)
}

// Run applies a message after the previous ones, with one EVM for all of
// them, as the reference's DoCall did: a transaction ends after each previous
// message, whose failures are logged and skipped (I8); the run stops at the
// timeout, if any; a message that cannot be applied is reported with its gas
// (F5).
func (call *Call) Run(ctx context.Context, statedb state.StateDB, previous []*core.Message, message *core.Message, timeout time.Duration) (*core.ExecutionResult, error) {
	ctx, cancel := withOptionalTimeout(ctx, timeout)
	defer cancel()
	evm, stop := call.newEVM(ctx, statedb, nil)
	defer stop()
	gasPool := core.NewGasPool(math.MaxUint64)
	for index, previousMessage := range previous {
		if _, err := core.ApplyMessage(evm, previousMessage, gasPool); err != nil {
			log.Warn("failed to apply previous transaction", "index", index, "err", err)
		}
		statedb.EndTransaction()
	}
	result, err := core.ApplyMessage(evm, message, gasPool)
	if evm.Cancelled() {
		return nil, fmt.Errorf("execution aborted (timeout = %v)", timeout)
	}
	if err != nil {
		return result, fmt.Errorf("err: %w (supplied gas %d)", err, message.GasLimit)
	}
	return result, nil
}

// Trace applies a message with the hooks of a tracer, in a gas pool of the
// message's gas. A message that cannot be applied is reported with the bare
// error.
func (call *Call) Trace(ctx context.Context, statedb state.StateDB, message *core.Message, hooks *tracing.Hooks) (*core.ExecutionResult, error) {
	evm, stop := call.newEVM(ctx, statedb, hooks)
	defer stop()
	return core.ApplyMessage(evm, message, core.NewGasPool(message.GasLimit))
}

// Messages converts the arguments of the messages a call runs after into
// messages; one that cannot be converted is logged and skipped, as in the
// reference.
func Messages(previous []CallArguments, gasCap uint64, header *evmcore.EvmHeader) []*core.Message {
	messages := make([]*core.Message, 0, len(previous))
	for index, arguments := range previous {
		message, err := arguments.ToMessage(gasCap, header.BaseFee)
		if err != nil {
			log.Warn("failed to convert previous transaction to message", "index", index, "err", err)
			continue
		}
		messages = append(messages, message)
	}
	return messages
}

// withOptionalTimeout bounds a context by a timeout when one is given.
func withOptionalTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout > 0 {
		return context.WithTimeout(ctx, timeout)
	}
	return context.WithCancel(ctx)
}

// RevertError reports a call that reverted, as the reference did: code 3,
// the revert data hex encoded, and the reason if the data holds one.
func RevertError(result *core.ExecutionResult) error {
	err := errors.New("execution reverted")
	if reason, unpackErr := abi.UnpackRevert(result.Revert()); unpackErr == nil {
		err = fmt.Errorf("execution reverted: %v", reason)
	}
	return &revertError{error: err, reason: hexutil.Encode(result.Revert())}
}

type revertError struct {
	error
	reason string
}

func (err *revertError) ErrorCode() int {
	return 3
}

func (err *revertError) ErrorData() any {
	return err.reason
}
