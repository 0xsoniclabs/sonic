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

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
)

// ApplyPreBlockSystemCalls applies to a state of a block's parent the system
// calls the block makes before its first transaction: from Prague on, the
// write of the parent's hash into the history contract of EIP-2935. It
// answers the EVM of the block over the state, without a base fee, which
// replays the block's transactions.
func (e *Environment) ApplyPreBlockSystemCalls(stateDB state.StateDB, block *evmcore.EvmBlock) (*vm.EVM, error) {
	vmConfig, err := e.VMConfig(block.NumberU64())
	if err != nil {
		return nil, fmt.Errorf("failed to get vm config: %w", err)
	}
	vmConfig.NoBaseFee = true
	evm := e.NewEVM(stateDB, block.Header(), nil, vmConfig)
	if evm.ChainConfig().IsPrague(block.Number, uint64(block.Time.Unix())) {
		evmcore.ProcessParentBlockHash(block.ParentHash, evm, stateDB)
	}
	return evm, nil
}

// ReplayState answers the state a transaction of a stored block ran on, and
// its message: the parent's state with the block's system calls and the
// transactions before it applied on one state, as the legacy replay does.
// An index of 0 in a block without transactions answers the state after the
// system calls and no message. The caller releases the state.
func (e *Environment) ReplayState(head, block *evmcore.EvmBlock, index int) (_ *core.Message, _ state.StateDB, err error) {
	if block.NumberU64() == 0 {
		return nil, nil, errors.New("no transaction in genesis")
	}
	if index > len(block.Transactions) {
		return nil, nil, fmt.Errorf("transaction index %d out of range for block %#x", index, block.Hash)
	}
	stateDB, _, err := e.StateAt(head, chain.BlockReference{Hash: &block.ParentHash})
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		if err != nil {
			stateDB.Release()
		}
	}()
	evm, err := e.ApplyPreBlockSystemCalls(stateDB, block)
	if err != nil {
		return nil, nil, err
	}
	if index == 0 && len(block.Transactions) == 0 {
		return nil, stateDB, nil
	}
	signer := types.LatestSignerForChainID(e.ChainConfig(block.NumberU64()).ChainID)
	for position, transaction := range block.Transactions {
		message, err := evmcore.TxAsMessage(transaction, signer, block.BaseFee)
		if err != nil {
			return nil, nil, err
		}
		if position == index {
			return message, stateDB, nil
		}
		// The network runs blob transactions without blobs only.
		if message.BlobHashes != nil {
			if len(message.BlobHashes) > 0 {
				continue
			}
			message.BlobHashes = nil
		}
		stateDB.SetTxContext(transaction.Hash(), position)
		if _, err := core.ApplyMessage(evm, message, core.NewGasPool(transaction.Gas())); err != nil {
			return nil, nil, fmt.Errorf("transaction %#x failed: %v", transaction.Hash(), err)
		}
		stateDB.EndTransaction()
	}
	return nil, nil, fmt.Errorf("transaction index %d out of range for block %#x", index, block.Hash)
}
