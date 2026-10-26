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
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/tracers/logger"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
)

// call runs a message on top of a block, in the legacy order: the block, the
// state overrides, the message, the rules of the block (F2). A revert is
// answered with its data (code 3); a call that fails otherwise with its
// error.
func (handler *handlers) call(ctx context.Context, arguments dispatch.Arguments) (any, error) {
	callArguments := dispatch.Argument[exec.CallArguments](arguments, 0)
	head, statedb, header, err := handler.stateAt(ctx, dispatch.ArgumentOr(arguments, 1, chain.Latest()))
	if err != nil {
		return nil, err
	}
	defer statedb.Release()
	if err := dispatch.ArgumentOr[exec.StateOverride](arguments, 2, nil).Apply(statedb); err != nil {
		return nil, err
	}
	limits := handler.environment.Config()
	message, err := callArguments.ToMessage(limits.RPCGasCap, header.BaseFee)
	if err != nil {
		return nil, err
	}
	call, err := handler.environment.PrepareCall(head, header, dispatch.Argument[*exec.BlockOverrides](arguments, 3))
	if err != nil {
		return nil, err
	}
	result, err := call.Run(ctx, statedb, nil, message, limits.RPCEVMTimeout)
	if err != nil {
		return nil, err
	}
	if len(result.Revert()) > 0 {
		return nil, exec.RevertError(result)
	}
	return hexutil.Bytes(result.Return()), result.Err
}

func (handler *handlers) estimateGas(ctx context.Context, arguments dispatch.Arguments) (any, error) {
	head, err := handler.environment.Latest(ctx)
	if err != nil {
		return nil, err
	}
	return handler.environment.EstimateGas(head, exec.Estimation{
		Arguments:      dispatch.Argument[exec.CallArguments](arguments, 0),
		Block:          dispatch.ArgumentOr(arguments, 1, chain.Latest()),
		StateOverrides: dispatch.ArgumentOr[exec.StateOverride](arguments, 2, nil),
		BlockOverrides: dispatch.Argument[*exec.BlockOverrides](arguments, 3),
	})
}

// accessListResult is the answer of eth_createAccessList: the list, the gas
// the transaction used with it, and the error of a transaction that failed.
type accessListResult struct {
	AccessList *types.AccessList `json:"accessList"`
	Error      string            `json:"error,omitempty"`
	GasUsed    hexutil.Uint64    `json:"gasUsed"`
}

// accessListSearch is the search of eth_createAccessList: it runs the
// transaction with the access list the last run collected until a run
// collects no more.
type accessListSearch struct {
	handler   *handlers
	head      exec.Head
	call      *exec.Call
	state     state.StateDB
	arguments exec.CallArguments
	exclude   map[common.Address]struct{}
}

// createAccessList reproduces the reference's AccessList with its two bugs
// (section 6, F8): the gas of a round is estimated with the access list of
// the round before, and the estimate ignores input.
func (handler *handlers) createAccessList(ctx context.Context, arguments dispatch.Arguments) (any, error) {
	head, statedb, header, err := handler.stateAt(ctx, dispatch.ArgumentOr(arguments, 1, chain.ReferenceToNumber(chain.PendingBlockNumber)))
	if err != nil {
		return nil, err
	}
	defer statedb.Release()
	search := &accessListSearch{handler: handler, head: head, state: statedb, arguments: dispatch.Argument[exec.CallArguments](arguments, 0)}
	estimateEveryRound := search.arguments.Gas == nil
	if err := handler.environment.SetDefaults(head, &search.arguments, handler.node.PoolNonce); err != nil {
		return nil, err
	}
	search.exclude = handler.excludedFromAccessList(header, &search.arguments)
	if search.call, err = handler.environment.PrepareCall(head, header, nil); err != nil {
		return nil, err
	}
	return search.run(estimateEveryRound)
}

// excludedFromAccessList are the sender, the recipient or created contract,
// and the precompiles, which an access list need not name.
func (handler *handlers) excludedFromAccessList(header *evmcore.EvmHeader, arguments *exec.CallArguments) map[common.Address]struct{} {
	recipient := crypto.CreateAddress(arguments.Sender(), uint64(*arguments.Nonce))
	if arguments.Recipient != nil {
		recipient = *arguments.Recipient
	}
	chainConfig := handler.environment.ChainConfig(header.Number.Uint64())
	exclude := map[common.Address]struct{}{arguments.Sender(): {}, recipient: {}}
	for _, address := range vm.ActivePrecompiles(chainConfig.Rules(header.Number, false, uint64(header.Time.Unix()))) {
		exclude[address] = struct{}{}
	}
	return exclude
}

func (search *accessListSearch) run(estimateEveryRound bool) (*accessListResult, error) {
	initial := types.AccessList(nil)
	if search.arguments.AccessList != nil {
		initial = *search.arguments.AccessList
	}
	previous := logger.NewAccessListTracer(initial, search.exclude)
	for {
		accessList := previous.AccessList()
		if estimateEveryRound {
			search.arguments.Gas = nil
			if err := search.handler.environment.SetDefaults(search.head, &search.arguments, search.handler.node.PoolNonce); err != nil {
				return nil, err
			}
		}
		search.arguments.AccessList = &accessList
		tracer := logger.NewAccessListTracer(accessList, search.exclude)
		result, err := search.round(tracer)
		if err != nil {
			return nil, err
		}
		if tracer.Equal(previous) {
			answer := &accessListResult{AccessList: &accessList, GasUsed: hexutil.Uint64(result.UsedGas)}
			if result.Err != nil {
				answer.Error = result.Err.Error()
			}
			return answer, nil
		}
		previous = tracer
	}
}

// round runs the transaction with the tracer on a copy of the state. A
// transaction that cannot be applied is reported with its sender and nonce,
// with the bare error (F5).
func (search *accessListSearch) round(tracer *logger.AccessListTracer) (*core.ExecutionResult, error) {
	limits := search.handler.environment.Config()
	message, err := search.arguments.ToMessage(limits.RPCGasCap, search.call.Header.BaseFee)
	if err != nil {
		return nil, err
	}
	statedb := search.state.Copy()
	defer statedb.Release()
	ctx := search.head.Context()
	result, err := search.call.Trace(ctx, statedb, message, tracer.Hooks())
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, ctxErr
	}
	if err != nil {
		return nil, fmt.Errorf("failed to apply transaction from sender %v and nonce %d: %w",
			search.arguments.Sender(), uint64(*search.arguments.Nonce), err)
	}
	return result, nil
}
