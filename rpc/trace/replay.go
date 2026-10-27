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

package trace

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/txtrace"
)

// defaultTraceTimeout bounds a traced run when RPCEVMTimeout sets none, as
// in the reference.
const defaultTraceTimeout = 5 * time.Second

func (handler *handlers) transaction(ctx context.Context, arguments dispatch.Arguments) (any, error) {
	return handler.traceTransactionHash(ctx, dispatch.Argument[common.Hash](arguments, 0), nil)
}

// get answers the traces of a transaction at the positions of the indexes.
func (handler *handlers) get(ctx context.Context, arguments dispatch.Arguments) (any, error) {
	indexes := dispatch.Argument[[]hexutil.Uint](arguments, 1)
	return handler.traceTransactionHash(ctx, dispatch.Argument[common.Hash](arguments, 0), &indexes)
}

func (handler *handlers) traceTransactionHash(ctx context.Context, hash common.Hash, indexes *[]hexutil.Uint) (*[]txtrace.ActionTrace, error) {
	head, err := handler.environment.Latest(ctx)
	if err != nil {
		return nil, err
	}
	located, err := exec.NullIfAbsent(handler.environment.TransactionByHash(head, hash))
	if err != nil {
		return nil, fmt.Errorf("cannot get transaction %s: %v", hash.String(), err)
	}
	if located == nil {
		return nil, fmt.Errorf("transaction %s not found", hash.String())
	}
	block, err := exec.NullIfAbsent(handler.environment.BlockAt(head, located.Block))
	if err != nil {
		return nil, fmt.Errorf("cannot get block from db %v, error:%v", located.Block, err.Error())
	}
	return handler.replayBlock(head, block, &hash, indexes)
}

// block answers the traces of every transaction of a block.
func (handler *handlers) block(ctx context.Context, arguments dispatch.Arguments) (any, error) {
	head, err := handler.environment.Latest(ctx)
	if err != nil {
		return nil, err
	}
	number, err := handler.resolve(head, dispatch.Argument[chain.BlockReference](arguments, 0))
	if err != nil {
		return nil, err
	}
	block, err := exec.NullIfAbsent(handler.environment.BlockAt(head, number))
	if err != nil {
		return nil, fmt.Errorf("cannot get block %v from db got %v", number, err.Error())
	}
	traces, err := handler.replayBlock(head, block, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot trace block %v got %v", number, err.Error())
	}
	return traces, nil
}

// resolve turns a reference into a block number as the reference's
// resolution did: a number is taken without checking it is served, a tag
// names the head, a hash must be of a served block.
func (handler *handlers) resolve(head exec.Head, reference chain.BlockReference) (uint64, error) {
	if reference.Number != nil && *reference.Number >= 0 {
		return uint64(*reference.Number), nil
	}
	number, err := handler.environment.Resolve(head, reference)
	switch {
	case errors.Is(err, chain.ErrNoSelector):
		return 0, errors.New("unknown header selector")
	case exec.Absent(err):
		return 0, errors.New("block not found")
	}
	return number, err
}

// replayBlock replays a block on the state of its parent and answers the
// traces of its transactions: of one transaction only when the hash is
// given, at the positions of the indexes when they are given. The other
// transactions run untraced, each checked against its receipt.
func (handler *handlers) replayBlock(head exec.Head, block *evmcore.EvmBlock, hash *common.Hash, indexes *[]hexutil.Uint) (*[]txtrace.ActionTrace, error) {
	if block == nil {
		return nil, errors.New("invalid block for tracing")
	}
	if block.NumberU64() == 0 {
		return nil, errors.New("genesis block is not traceable")
	}
	statedb, call, err := handler.environment.ReplayState(head, block)
	var parentState *exec.ParentStateError
	switch {
	case errors.As(err, &parentState):
		return nil, fmt.Errorf("cannot get state for block %v, error: %v", block.NumberU64(), err.Error())
	case err != nil:
		return nil, fmt.Errorf("cannot apply pre-block system calls for block %v, error: %v", block.NumberU64(), err.Error())
	}
	defer statedb.Release()
	receipts, err := exec.NullIfAbsent(handler.environment.ReceiptsAt(head, block.NumberU64()))
	if err != nil {
		return nil, fmt.Errorf("cannot get receipts for block %v, error: %v", block.NumberU64(), err.Error())
	}
	replay := &blockReplay{handler: handler, head: head, block: block, statedb: statedb, call: call, receipts: receipts}
	traces := txtrace.CallTrace{Actions: make([]txtrace.ActionTrace, 0)}
	for position, transaction := range block.Transactions {
		if hash != nil && *hash != transaction.Hash() {
			if err := replay.untraced(position, transaction); err != nil {
				return nil, err
			}
			continue
		}
		found, err := replay.traced(position, transaction)
		if err != nil {
			return nil, err
		}
		traces.AddTraces(found, indexes)
		if hash != nil {
			break
		}
	}
	if len(traces.Actions) > 0 {
		return &traces.Actions, nil
	}
	if indexes != nil || hash != nil {
		return nil, nil
	}
	return emptyBlockTrace(block.Hash, *block.Number), nil
}

// blockReplay is a block being replayed on the state of its parent.
type blockReplay struct {
	handler  *handlers
	head     exec.Head
	block    *evmcore.EvmBlock
	statedb  state.StateDB
	call     *exec.Call
	receipts types.Receipts
}

// receipt is the receipt of the transaction at a position. A list too short
// for it, which the reference indexed unguarded in its untraced branch, is
// answered with an error (section 6, F8).
func (replay *blockReplay) receipt(position int, transaction *types.Transaction) (*types.Receipt, error) {
	if len(replay.receipts) <= position || replay.receipts[position] == nil {
		return nil, fmt.Errorf("no receipt found for transaction %s", transaction.Hash().String())
	}
	return replay.receipts[position], nil
}

func (replay *blockReplay) message(transaction *types.Transaction) (*core.Message, error) {
	message, err := replay.handler.environment.TransactionMessage(replay.block, transaction)
	if err != nil {
		return nil, fmt.Errorf("cannot get message from transaction %s, error %s", transaction.Hash().String(), err)
	}
	return message, nil
}

// traced replays a transaction with the action tracer.
func (replay *blockReplay) traced(position int, transaction *types.Transaction) (*[]txtrace.ActionTrace, error) {
	message, err := replay.message(transaction)
	if err != nil {
		return nil, err
	}
	receipt, err := replay.receipt(position, transaction)
	if err != nil {
		return nil, err
	}
	traces, err := replay.trace(message, transaction, uint64(receipt.TransactionIndex), receipt.Status)
	if err != nil {
		return nil, fmt.Errorf("cannot get transaction trace for transaction %s, error %s", transaction.Hash().String(), err)
	}
	return traces, nil
}

// trace runs a transaction with the action tracer and checks the status of
// the run against the receipt's.
func (replay *blockReplay) trace(message *core.Message, transaction *types.Transaction, index uint64, status uint64) (*[]txtrace.ActionTrace, error) {
	tracer := txtrace.NewTraceStructLogger(replay.block, uint(index))
	ctx, cancel := context.WithTimeout(replay.head.Context(), replay.handler.timeout())
	defer cancel()
	evm, stop := replay.call.NewEVM(ctx, replay.statedb, tracer.Hooks())
	defer stop()
	replay.statedb.SetTxContext(transaction.Hash(), int(index))
	block := replay.block
	// The reference passed the index as the counter of the gas used.
	result, err := evmcore.ApplyTransactionWithEVM(message, replay.call.ChainConfig(), core.NewGasPool(message.GasLimit), replay.statedb, block.Number, block.Hash, transaction, &index, evm)
	traces := tracer.GetResult()
	if err != nil {
		if status == types.ReceiptStatusSuccessful {
			return nil, fmt.Errorf("invalid transaction replay state at %s, error: %s", transaction.Hash().String(), err.Error())
		}
		errorTrace := txtrace.GetErrorTraceFromMsg(message, block.Hash, *block.Number, transaction.Hash(), index, err)
		return &[]txtrace.ActionTrace{*errorTrace}, nil
	}
	if evm.Cancelled() {
		return nil, errors.New("EVM was cancelled when replaying tx")
	}
	if status != result.Status {
		return nil, fmt.Errorf("invalid transaction replay state at %s, want %v but got %v", transaction.Hash().String(), status, result.Status)
	}
	return traces, nil
}

// untraced replays a transaction without tracing, to give the following
// ones the state it left, and checks its outcome against its receipt.
func (replay *blockReplay) untraced(position int, transaction *types.Transaction) error {
	log.Debug("Replaying transaction without trace", "txHash", transaction.Hash().String())
	message, err := replay.message(transaction)
	if err != nil {
		return err
	}
	replay.statedb.SetTxContext(transaction.Hash(), position)
	evm, stop := replay.call.NewEVM(replay.head.Context(), replay.statedb, nil)
	defer stop()
	result, err := core.ApplyMessage(evm, message, core.NewGasPool(message.GasLimit))
	failed := false
	if err != nil {
		failed = true
		log.Error("Cannot replay transaction", "txHash", transaction.Hash().String(), "err", err.Error())
	}
	if result != nil && result.Err != nil {
		failed = true
		log.Debug("Error replaying transaction", "txHash", transaction.Hash().String(), "err", result.Err.Error())
	}
	replay.statedb.EndTransaction()
	receipt, err := replay.receipt(position, transaction)
	if err != nil {
		return err
	}
	if failed == (receipt.Status == types.ReceiptStatusSuccessful) {
		return fmt.Errorf("invalid transaction replay state at %s", transaction.Hash().String())
	}
	return nil
}

// timeout bounds a traced run: RPCEVMTimeout, or the default without one.
func (handler *handlers) timeout() time.Duration {
	if timeout := handler.environment.Config().RPCEVMTimeout; timeout > 0 {
		return timeout
	}
	return defaultTraceTimeout
}

// emptyBlockTrace is the trace of a block without transactions.
func emptyBlockTrace(blockHash common.Hash, blockNumber big.Int) *[]txtrace.ActionTrace {
	traces := txtrace.CallTrace{Actions: make([]txtrace.ActionTrace, 0)}
	blockTrace := txtrace.CreateActionTrace(blockHash, blockNumber, common.Hash{}, 0, "empty")
	blockTrace.Action = txtrace.NewAddressAction(common.Address{}, 0, []byte{}, nil, hexutil.Big{}, nil)
	blockTrace.Error = "Empty block"
	traces.AddTrace(blockTrace)
	return &traces.Actions
}
