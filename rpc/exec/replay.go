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
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter/state"
)

// ParentStateError is the failure to open the state a block was processed
// on, worded as the state lookups word it; a method that replays blocks
// tells it from the failures after it, as the reference did.
type ParentStateError struct {
	err error
}

func (failure *ParentStateError) Error() string {
	return failure.err.Error()
}

func (failure *ParentStateError) Unwrap() error {
	return failure.err
}

// ReplayState opens the state a block was processed on and runs what block
// processing runs before the first transaction, as the reference's replays
// did: the history contract of EIP-2935 from Prague on. It returns the state,
// which the caller releases, and the call its transactions run in.
func (environment *Environment) ReplayState(head Head, block *evmcore.EvmBlock) (state.StateDB, *Call, error) {
	statedb, err := environment.OpenState(head, block.NumberU64()-1)
	if err != nil {
		return nil, nil, &ParentStateError{err: StateLookupError(err)}
	}
	// The state is released by the caller once it has it; until then, on
	// any failure or panic, here (I7).
	handedOver := false
	defer func() {
		if !handedOver {
			statedb.Release()
		}
	}()
	call, err := environment.PrepareCall(head, block.Header(), nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get vm config: %w", err)
	}
	evm, stop := call.NewEVM(head.Context(), statedb, nil)
	defer stop()
	if evm.ChainConfig().IsPrague(block.Number, uint64(block.Time.Unix())) {
		evmcore.ProcessParentBlockHash(block.ParentHash, evm, statedb)
	}
	handedOver = true
	return statedb, call, nil
}

// StateAtTransaction replays the transactions of a block before the one at
// the index, as the reference's stateAtTransaction did, and returns the
// message of that transaction, nil for the index after the last, with the
// state, which the caller releases, and the call to run it in.
func (environment *Environment) StateAtTransaction(head Head, block *evmcore.EvmBlock, index int) (*core.Message, state.StateDB, *Call, error) {
	if block.NumberU64() == 0 {
		return nil, nil, nil, errors.New("no transaction in genesis")
	}
	if index > len(block.Transactions) {
		return nil, nil, nil, fmt.Errorf("transaction index %d out of range for block %#x", index, block.Hash)
	}
	statedb, call, err := environment.ReplayState(head, block)
	if err != nil {
		return nil, nil, nil, err
	}
	handedOver := false
	defer func() {
		if !handedOver {
			statedb.Release()
		}
	}()
	message, err := environment.replayUntil(head, block, index, statedb, call)
	if err != nil {
		return nil, nil, nil, err
	}
	handedOver = true
	return message, statedb, call, nil
}

func (environment *Environment) replayUntil(head Head, block *evmcore.EvmBlock, index int, statedb state.StateDB, call *Call) (*core.Message, error) {
	if index == 0 && len(block.Transactions) == 0 {
		return nil, nil
	}
	evm, stop := call.NewEVM(head.Context(), statedb, nil)
	defer stop()
	signer := environment.Signer()
	for position, transaction := range block.Transactions {
		message, err := evmcore.TxAsMessage(transaction, signer, block.BaseFee)
		if err != nil {
			return nil, err
		}
		if position == index {
			return message, nil
		}
		// Sonic supports blob transactions only without blobs; one with
		// blobs is skipped, as block processing skips it.
		if message.BlobHashes != nil {
			if len(message.BlobHashes) > 0 {
				continue
			}
			message.BlobHashes = nil
		}
		statedb.SetTxContext(transaction.Hash(), position)
		if _, err := core.ApplyMessage(evm, message, core.NewGasPool(transaction.Gas())); err != nil {
			return nil, fmt.Errorf("transaction %#x failed: %v", transaction.Hash(), err)
		}
		statedb.EndTransaction()
	}
	return nil, fmt.Errorf("transaction index %d out of range for block %#x", index, block.Hash)
}

// TransactionMessage converts a transaction of a block into its message.
func (environment *Environment) TransactionMessage(block *evmcore.EvmBlock, transaction *types.Transaction) (*core.Message, error) {
	return evmcore.TxAsMessage(transaction, environment.Signer(), block.BaseFee)
}
