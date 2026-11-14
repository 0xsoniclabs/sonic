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

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/tracers/logger"
)

type callParams struct {
	Arguments      exec.CallArguments
	Block          *chain.BlockReference
	StateOverrides *exec.StateOverride
	BlockOverrides *exec.BlockOverrides
}

// call runs a call on the state of a block, the head if it names none, and
// answers its return data; a revert is an error with the revert data.
func (h *handlers) call(ctx context.Context, params callParams) (hexutil.Bytes, error) {
	stateDB, header, err := h.environment.StateAt(h.environment.Head(), orLatest(params.Block))
	if err != nil {
		return nil, err
	}
	defer stateDB.Release()
	if err := params.StateOverrides.Apply(stateDB); err != nil {
		return nil, err
	}
	result, err := h.environment.Execute(ctx, stateDB, header, exec.Call{
		Arguments:      params.Arguments,
		BlockOverrides: params.BlockOverrides,
		Timeout:        h.environment.Config().RPCEVMTimeout,
	})
	if err != nil {
		return nil, err
	}
	if len(result.Revert()) > 0 {
		return nil, exec.NewRevertError(result)
	}
	return result.Return(), result.Err
}

func (h *handlers) estimateGas(ctx context.Context, params callParams) (hexutil.Uint64, error) {
	return h.environment.EstimateGas(ctx, h.environment.Head(), exec.Estimation{
		Block:          orLatest(params.Block),
		StateOverrides: params.StateOverrides,
		BlockOverrides: params.BlockOverrides,
	}, params.Arguments)
}

type accessListParams struct {
	Arguments exec.CallArguments
	Block     *chain.BlockReference
}

type accessListResult struct {
	AccessList *types.AccessList `json:"accessList"`
	Error      string            `json:"error,omitempty"`
	GasUsed    hexutil.Uint64    `json:"gasUsed"`
}

// createAccessList answers the access list of a call, the pending block's
// if it names no block, with the gas the call uses with it and the reason it
// fails, if it does.
func (h *handlers) createAccessList(ctx context.Context, params accessListParams) (*accessListResult, error) {
	reference := chain.ReferenceTo(chain.PendingBlockNumber)
	if params.Block != nil {
		reference = *params.Block
	}
	accessList, gasUsed, vmErr, err := h.accessList(ctx, reference, params.Arguments)
	if err != nil {
		return nil, err
	}
	result := &accessListResult{AccessList: &accessList, GasUsed: hexutil.Uint64(gasUsed)}
	if vmErr != nil {
		result.Error = vmErr.Error()
	}
	return result, nil
}

// accessList runs the call with the access list it found so far until the
// list does not grow any more. The sender, the recipient and the precompiles
// are left out of it. Without gas given, every list gets its own estimate,
// as the legacy handler does; it estimates before it sets the list, so a
// round runs with the gas of the list before.
func (h *handlers) accessList(ctx context.Context, reference chain.BlockReference, args exec.CallArguments) (types.AccessList, uint64, error, error) {
	head := h.environment.Head()
	stateDB, header, err := h.environment.StateAt(head, reference)
	if err != nil {
		return nil, 0, nil, err
	}
	defer stateDB.Release()
	noGas := args.Gas == nil
	if err := h.environment.SetDefaults(ctx, head, &args); err != nil {
		return nil, 0, nil, err
	}
	to := crypto.CreateAddress(args.Sender(), uint64(*args.Nonce))
	if args.To != nil {
		to = *args.To
	}
	rules := h.environment.ChainConfig(header.Number.Uint64()).Rules(header.Number, false, uint64(header.Time.Unix()))
	excluded := map[common.Address]struct{}{args.Sender(): {}, to: {}}
	for _, address := range vm.ActivePrecompiles(rules) {
		excluded[address] = struct{}{}
	}
	previous := logger.NewAccessListTracer(nil, excluded)
	if args.AccessList != nil {
		previous = logger.NewAccessListTracer(*args.AccessList, excluded)
	}
	vmConfig, err := h.environment.VMConfig(header.Number.Uint64())
	if err != nil {
		return nil, 0, nil, err
	}
	vmConfig.NoBaseFee = true
	for {
		accessList := previous.AccessList()
		if noGas {
			args.Gas = nil
			if err := h.environment.SetDefaults(ctx, head, &args); err != nil {
				return nil, 0, nil, err
			}
		}
		args.AccessList = &accessList
		message, err := args.ToMessage(h.environment.Config().RPCGasCap, header.BaseFee)
		if err != nil {
			return nil, 0, nil, err
		}
		tracer := logger.NewAccessListTracer(accessList, excluded)
		config := vmConfig
		config.Tracer = tracer.Hooks()
		result, err := h.runTraced(ctx, stateDB, header, config, message)
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, 0, nil, ctxErr
		}
		if err != nil {
			return nil, 0, nil, fmt.Errorf("failed to apply transaction from sender %v and nonce %d: %w", args.Sender(), uint64(*args.Nonce), err)
		}
		if tracer.Equal(previous) {
			return accessList, result.UsedGas, result.Err, nil
		}
		previous = tracer
	}
}

// runTraced runs a message on a copy of the state.
func (h *handlers) runTraced(ctx context.Context, base state.StateDB, header *evmcore.EvmHeader, config vm.Config, message *core.Message) (*core.ExecutionResult, error) {
	branch := base.Copy()
	defer branch.Release()
	evm := h.environment.NewEVM(branch, header, nil, config)
	stop := context.AfterFunc(ctx, evm.Cancel)
	defer stop()
	return core.ApplyMessage(evm, message, core.NewGasPool(message.GasLimit))
}
