// Copyright 2025 Sonic Operations Ltd
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

package gossip

import (
	"fmt"
	"math/big"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/gasprice"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
)

//go:generate mockgen -source=evm_state_reader.go -destination=evm_state_reader_mock.go -package=gossip

// StateReader defines methods to access EVM state.
type StateReader interface {
	// CurrentBaseFee returns the base fee charged in the most recent block.
	CurrentBaseFee() *big.Int
	// CurrentMaxGasLimit returns the maximum gas limit of the most recent epoch.
	CurrentMaxGasLimit() uint64
	// CurrentConfig returns the chain config applicable to the most recent block.
	CurrentConfig() *params.ChainConfig
	// CurrentRules returns the rules applicable to the most recent epoch.
	CurrentRules() opera.Rules
	// CurrentBlock returns the most recent block.
	// This method is the recommended option for fast access to the latest block
	CurrentBlock() *evmcore.EvmBlock
	// LastBlockWithArchiveState returns the most recent block with archive state.
	// This method shall be the preferable way to get the latest block for
	// operations that require access to the full block (e.g., RPC calls),
	LastBlockWithArchiveState(withTxs bool) (*evmcore.EvmBlock, error)
	// Header returns the header of the block with the given number.
	// If the block is not found, nil is returned.
	// If the hash provided is not zero and does not match the hash of the
	// block found, nil is returned.
	Header(verificationHash common.Hash, number uint64) *evmcore.EvmHeader
	// Block returns the block with the given number.
	// If the block is not found, nil is returned.
	// If the hash provided is not zero and does not match the hash of the
	// block found, nil is returned.
	Block(verificationHash common.Hash, number uint64) *evmcore.EvmBlock
	// CurrentStateDB returns a read-only access to stateDB.
	CurrentStateDB() (state.StateDB, error)
	// BlockStateDB returns stateDB for the given block number.
	// An error is returned if the live state is not initialized, it failed to
	// find the block in the archive, or the state root of the block found
	// does not match the given state root.
	BlockStateDB(blockNum *big.Int, stateRoot common.Hash) (state.StateDB, error)
}

// EvmStateReader implements StateReader interface.
type EvmStateReader struct {
	*ServiceFeed

	store *Store
	gpo   *gasprice.Oracle
}

// CurrentBaseFee returns the base fee of the most recent block.
func (r *EvmStateReader) CurrentBaseFee() *big.Int {
	res := r.store.GetBlock(r.store.GetLatestBlockIndex()).BaseFee
	return new(big.Int).Set(res)
}

// CurrentMaxGasLimit returns the maximum gas limit of the most recent epoch.
func (r *EvmStateReader) CurrentMaxGasLimit() uint64 {
	rules := r.store.GetRules()
	maxEmptyEventGas := rules.Economy.Gas.EventGas +
		uint64(rules.Dag.MaxParents-rules.Dag.MaxFreeParents)*rules.Economy.Gas.ParentGas +
		uint64(rules.Dag.MaxExtraData)*rules.Economy.Gas.ExtraDataGas
	if rules.Economy.Gas.MaxEventGas < maxEmptyEventGas {
		return 0
	}
	return rules.Economy.Gas.MaxEventGas - maxEmptyEventGas
}

// CurrentConfig returns the chain config applicable to the most recent block.
func (r *EvmStateReader) CurrentConfig() *params.ChainConfig {
	return r.store.GetEvmChainConfig(r.store.GetLatestBlockIndex())
}

// CurrentRules returns the rules applicable to the most recent epoch.
func (r *EvmStateReader) CurrentRules() opera.Rules {
	return r.store.GetRules()
}

// CurrentBlock returns the most recent block.
// This method is the recommended option for fast access to the latest block.
func (r *EvmStateReader) CurrentBlock() *evmcore.EvmBlock {
	n := r.store.GetLatestBlockIndex()

	return r.getBlock(common.Hash{}, n, true)
}

// LastBlockWithArchiveState returns the most recent block with archive.
// This method shall be the preferable way to get the latest block for
// operations that require access to the full state (e.g., RPC calls).
func (r *EvmStateReader) LastBlockWithArchiveState(withTxs bool) (*evmcore.EvmBlock, error) {
	latestBlock := r.store.GetLatestBlockIndex()

	// make sure the block is present in the archive
	latestArchiveBlock, empty, err := r.store.evm.GetArchiveBlockHeight()
	if err != nil {
		return nil, fmt.Errorf("failed to get latest archive block; %v", err)
	}
	if !empty && idx.Block(latestArchiveBlock) < latestBlock {
		latestBlock = idx.Block(latestArchiveBlock)
	}

	return r.getBlock(common.Hash{}, latestBlock, withTxs), nil
}

// Header returns the header of the block with the given number.
// If the block is not found, nil is returned.
// If the hash provided is not zero and does not match the hash of the
// block found, nil is returned.
func (r *EvmStateReader) Header(verificationHash common.Hash, number uint64) *evmcore.EvmHeader {
	return r.getBlock(verificationHash, idx.Block(number), false).Header()
}

// Block returns the block with the given number.
// If the block is not found, nil is returned.
// If the hash provided is not zero and does not match the hash of the block
// found, nil is returned.
func (r *EvmStateReader) Block(verificationHash common.Hash, number uint64) *evmcore.EvmBlock {
	return r.getBlock(verificationHash, idx.Block(number), true)
}

// getBlock is an internal method to get a block by number.
// If the hash provided is not zero and does not match the hash of the block
// found, nil is returned.
func (r *EvmStateReader) getBlock(verificationHash common.Hash, n idx.Block, readTxs bool) *evmcore.EvmBlock {
	block := r.store.GetBlock(n)
	if block == nil {
		return nil
	}
	if (verificationHash != common.Hash{}) && (verificationHash != block.Hash()) {
		return nil
	}
	if readTxs {
		if cached := r.store.EvmStore().GetCachedEvmBlock(n); cached != nil {
			return cached
		}
	}

	var transactions types.Transactions
	if readTxs {
		transactions = r.store.GetBlockTxs(n, block)
	} else {
		transactions = make(types.Transactions, 0)
	}

	// find block rules
	epoch := block.Epoch
	es := r.store.GetHistoryEpochState(epoch)
	var rules opera.Rules
	if es != nil {
		rules = es.Rules
	}

	// There is no epoch state for epoch 0 comprising block 0.
	// For this epoch, London and Sonic upgrades are enabled.
	// TODO: instead of hard-coding these values here, a corresponding
	// epoch state should be included in the genesis procedure to be
	// consistent. See issue #72.
	if epoch == 0 {
		rules.Upgrades.London = true
		rules.Upgrades.Sonic = true
	}

	var prev common.Hash
	if n != 0 {
		block := r.store.GetBlock(n - 1)
		if block != nil {
			prev = block.Hash()
		}
	}
	evmHeader := evmcore.ToEvmHeader(block, prev, rules)

	var evmBlock *evmcore.EvmBlock
	if readTxs {
		evmBlock = evmcore.NewEvmBlock(evmHeader, transactions)
		r.store.EvmStore().SetCachedEvmBlock(n, evmBlock)
	} else {
		// not completed block here
		evmBlock = &evmcore.EvmBlock{
			EvmHeader: *evmHeader,
		}
	}

	return evmBlock
}

// CurrentStateDB returns a read-only StateDB for the current state.
func (r *EvmStateReader) CurrentStateDB() (state.StateDB, error) {
	return r.store.evm.GetCurrentStateDb()
}

// BlockStateDB returns stateDB for the given block number and state root.
// An error is returned if the live state is not initialized, it failed to
// find the block in the archive, or the state root of the block found
// does not match the given state root.
func (r *EvmStateReader) BlockStateDB(blockNum *big.Int, stateRoot common.Hash) (state.StateDB, error) {
	return r.store.evm.GetBlockStateDb(blockNum, stateRoot)
}
