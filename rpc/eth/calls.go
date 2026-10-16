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

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
)

func registerCalls(d *dispatch.Dispatcher, a *API, m func(string) string) {
	dispatch.Add4(d, m("call"), a.call, "args", "block?", "stateOverrides?", "blockOverrides?")
	dispatch.Add4(d, m("estimateGas"), a.estimateGas, "args", "block?", "stateOverrides?", "blockOverrides?")
	dispatch.Add2(d, m("createAccessList"), a.createAccessList, "args", "block?")
	dispatch.Add2(d, m("simulateV1"), a.simulateV1, "opts", "block?")
}

// call executes the transaction on the state of the block. The caller can
// specify a batch of contracts for fields overriding. It makes no change in
// the state: it is useful to execute and retrieve values.
func (a *API) call(ctx context.Context, args exec.CallArgs, ref *dispatch.BlockRef, stateOverrides *exec.StateOverride, blockOverrides *exec.BlockOverrides) (hexutil.Bytes, error) {
	return exec.Call(ctx, a.env, a.chain.Head(), args, dispatch.OrLatest(ref), stateOverrides, blockOverrides)
}

// estimateGas returns an estimate of the amount of gas needed to execute the
// transaction against the block.
func (a *API) estimateGas(ctx context.Context, args exec.CallArgs, ref *dispatch.BlockRef, overrides *exec.StateOverride, blockOverrides *exec.BlockOverrides) (hexutil.Uint64, error) {
	return exec.DoEstimateGas(ctx, a.env, a.chain.Head(), args, dispatch.OrLatest(ref), overrides, blockOverrides, a.cfg.GasCap, nil)
}

type accessListResult struct {
	Accesslist *types.AccessList `json:"accessList"`
	Error      string            `json:"error,omitempty"`
	GasUsed    hexutil.Uint64    `json:"gasUsed"`
}

// createAccessList creates an EIP-2930 access list for the transaction.
func (a *API) createAccessList(ctx context.Context, args exec.CallArgs, ref *dispatch.BlockRef) (*accessListResult, error) {
	bNrOrHash := dispatch.RefNumber(dispatch.PendingBlockNumber)
	if ref != nil {
		bNrOrHash = *ref
	}
	acl, gasUsed, vmerr, err := exec.AccessList(ctx, a.env, a.chain.Head(), bNrOrHash, args)
	if err != nil {
		return nil, err
	}
	result := &accessListResult{Accesslist: &acl, GasUsed: hexutil.Uint64(gasUsed)}
	if vmerr != nil {
		result.Error = vmerr.Error()
	}
	return result, nil
}

// simulateV1 executes a series of transactions on top of a base state. The
// transactions are packed into blocks. For each block, block header fields can
// be overridden. The state can also be overridden prior to execution of each
// block. It makes no change in the state.
func (a *API) simulateV1(ctx context.Context, opts simOpts, ref *dispatch.BlockRef) ([]*simBlockResult, error) {
	if len(opts.BlockStateCalls) == 0 {
		return nil, simInvalidParamsError()
	}
	if len(opts.BlockStateCalls) > maxSimulateBlocks {
		return nil, simClientLimitExceededError("too many blocks")
	}
	var totalCalls int
	for _, block := range opts.BlockStateCalls {
		if len(block.Calls) > maxSimulateCallsPerBlock {
			return nil, simClientLimitExceededError(fmt.Sprintf("too many calls in block: %d > %d", len(block.Calls), maxSimulateCallsPerBlock))
		}
		totalCalls += len(block.Calls)
		if totalCalls > maxSimulateTotalCalls {
			return nil, simClientLimitExceededError(fmt.Sprintf("too many calls: %d > %d", totalCalls, maxSimulateTotalCalls))
		}
	}

	head := a.chain.Head()
	statedb, base, err := a.env.StateAt(head, dispatch.OrLatest(ref))
	if err != nil {
		return nil, err
	}
	defer statedb.Release()

	gasCap := a.cfg.GasCap
	if gasCap == 0 {
		gasCap = ^uint64(0) // MaxUint64
	}

	chainConfig := a.env.ChainConfig(base.Number.Uint64())

	sim := &simulator{
		env:            a.env,
		head:           head,
		state:          statedb,
		base:           base,
		chainConfig:    chainConfig,
		gp:             core.NewGasPool(gasCap),
		traceTransfers: opts.TraceTransfers,
		validate:       opts.Validation,
		fullTx:         opts.ReturnFullTransactions,
	}
	return sim.execute(ctx, opts.BlockStateCalls)
}
