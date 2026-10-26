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
	"github.com/ethereum/go-ethereum/log"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/exec"
)

type callArgs struct {
	Args           exec.CallArgs        `rpc:"args"`
	Block          *chain.BlockRef      `rpc:"block"`
	StateOverrides *exec.StateOverride  `rpc:"stateOverrides"`
	BlockOverrides *exec.BlockOverrides `rpc:"blockOverrides"`
}

// call runs a message call; a revert answers code 3 with the revert data.
func (a *api) call(ctx context.Context, p callArgs) (any, error) {
	cfg := a.env.Config()
	result, err := a.env.Call(ctx, a.env.Latest(ctx), p.Args, chain.OrLatest(p.Block), p.StateOverrides, p.BlockOverrides, cfg.EVMTimeout, cfg.GasCap, nil)
	if err != nil {
		return nil, err
	}
	if len(result.Revert()) > 0 {
		return nil, exec.NewRevertError(result)
	}
	return hexutil.Bytes(result.Return()), result.Err
}

func (a *api) estimateGas(ctx context.Context, p callArgs) (any, error) {
	return a.env.EstimateGas(ctx, a.env.Latest(ctx), p.Args, chain.OrLatest(p.Block), p.StateOverrides, p.BlockOverrides, a.env.Config().GasCap, nil)
}

type accessListArgs struct {
	Args  exec.CallArgs   `rpc:"args"`
	Block *chain.BlockRef `rpc:"block"`
}

// accessListResult is the answer of eth_createAccessList; Error holds the
// failure of the transaction itself.
type accessListResult struct {
	Accesslist *types.AccessList `json:"accessList"`
	Error      string            `json:"error,omitempty"`
	GasUsed    hexutil.Uint64    `json:"gasUsed"`
}

// createAccessList grows an access list until running the transaction with
// it touches nothing outside it. The block defaults to pending.
func (a *api) createAccessList(ctx context.Context, p accessListArgs) (any, error) {
	ref := chain.NumberRef(chain.PendingBlockNumber)
	if p.Block != nil {
		ref = *p.Block
	}
	acl, gasUsed, vmErr, err := a.accessList(ctx, ref, p.Args)
	if err != nil {
		return nil, err
	}
	result := &accessListResult{Accesslist: &acl, GasUsed: hexutil.Uint64(gasUsed)}
	if vmErr != nil {
		result.Error = vmErr.Error()
	}
	return result, nil
}

// accessList keeps the reference's defect: when the caller gives no gas, the
// gas of a round is estimated before the round's access list is set, so a
// round runs with the gas of the round before.
func (a *api) accessList(ctx context.Context, ref chain.BlockRef, args exec.CallArgs) (acl types.AccessList, gasUsed uint64, vmErr error, err error) {
	head := a.env.Latest(ctx)
	db, header, err := a.env.StateAt(head, ref)
	if err != nil {
		return nil, 0, nil, exec.HeaderNotFound(err)
	}
	defer db.Release()

	nogas := args.Gas == nil
	if err := a.env.SetDefaults(ctx, head, &args); err != nil {
		return nil, 0, nil, err
	}
	var to common.Address
	if args.To != nil {
		to = *args.To
	} else {
		to = crypto.CreateAddress(args.Sender(), uint64(*args.Nonce))
	}
	number := header.Number.Uint64()
	chainConfig := a.env.ChainConfig(number)
	precompiles := vm.ActivePrecompiles(chainConfig.Rules(header.Number, false, uint64(header.Time.Unix())))
	exclude := map[common.Address]struct{}{args.Sender(): {}, to: {}}
	for _, addr := range precompiles {
		exclude[addr] = struct{}{}
	}

	prevTracer := logger.NewAccessListTracer(nil, exclude)
	if args.AccessList != nil {
		prevTracer = logger.NewAccessListTracer(*args.AccessList, exclude)
	}
	for {
		accessList := prevTracer.AccessList()
		log.Trace("Creating access list", "input", accessList)
		if nogas {
			args.Gas = nil
			if err := a.env.SetDefaults(ctx, head, &args); err != nil {
				return nil, 0, nil, err
			}
		}
		args.AccessList = &accessList
		res, err := a.accessListRound(ctx, db, header, args, accessList, exclude)
		if err != nil {
			return nil, 0, nil, err
		}
		if res.tracer.Equal(prevTracer) {
			return accessList, res.result.UsedGas, res.result.Err, nil
		}
		prevTracer = res.tracer
	}
}

type accessListRound struct {
	tracer *logger.AccessListTracer
	result *core.ExecutionResult
}

// accessListRound runs the transaction with an access list on a copy of the
// state.
func (a *api) accessListRound(ctx context.Context, db state.StateDB, header *evmcore.EvmHeader, args exec.CallArgs, accessList types.AccessList, exclude map[common.Address]struct{}) (accessListRound, error) {
	st := db.Copy()
	defer st.Release()
	msg, err := args.ToMessage(a.env.Config().GasCap, header.BaseFee)
	if err != nil {
		return accessListRound{}, err
	}
	tracer := logger.NewAccessListTracer(accessList, exclude)
	cfg, err := a.env.VMConfig(ctx, header.Number.Uint64())
	if err != nil {
		return accessListRound{}, err
	}
	cfg.Tracer = tracer.Hooks()
	cfg.NoBaseFee = true
	evm := vm.NewEVM(a.env.BlockContext(header), st, a.env.ChainConfig(header.Number.Uint64()), cfg)
	stop := context.AfterFunc(ctx, evm.Cancel)
	res, err := core.ApplyMessage(evm, msg, core.NewGasPool(msg.GasLimit))
	stop()
	if ctxErr := ctx.Err(); ctxErr != nil {
		return accessListRound{}, ctxErr
	}
	if err != nil {
		return accessListRound{}, fmt.Errorf("failed to apply transaction from sender %v and nonce %d: %w", args.Sender(), uint64(*args.Nonce), err)
	}
	return accessListRound{tracer: tracer, result: res}, nil
}
