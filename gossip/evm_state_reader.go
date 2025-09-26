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

type EvmStateReader struct {
	*ServiceFeed

	store *Store
	gpo   *gasprice.Oracle
}

// GetCurrentBaseFee returns the base fee charged in the most recent block.
func (r *EvmStateReader) GetCurrentBaseFee() *big.Int {
	res := r.store.GetBlock(r.store.GetLatestBlockIndex()).BaseFee
	return new(big.Int).Set(res)
}

func (r *EvmStateReader) MaxGasLimit() uint64 {
	rules := r.store.GetRules()
	maxEmptyEventGas := rules.Economy.Gas.EventGas +
		uint64(rules.Dag.MaxParents-rules.Dag.MaxFreeParents)*rules.Economy.Gas.ParentGas +
		uint64(rules.Dag.MaxExtraData)*rules.Economy.Gas.ExtraDataGas
	if rules.Economy.Gas.MaxEventGas < maxEmptyEventGas {
		return 0
	}
	return rules.Economy.Gas.MaxEventGas - maxEmptyEventGas
}

func (r *EvmStateReader) Config() *params.ChainConfig {
	blockNumber := r.CurrentBlock().Number
	return r.store.GetEvmChainConfig(idx.Block(blockNumber.Uint64()))
}

func (r *EvmStateReader) CurrentBlock() *evmcore.EvmBlock {
	n := r.store.GetLatestBlockIndex()

	return r.getBlock(common.Hash{}, n, true)
}

func (r *EvmStateReader) CurrentHeader() *evmcore.EvmHeader {
	n := r.store.GetLatestBlockIndex()

	return r.getBlock(common.Hash{}, n, false).Header()
}

func (r *EvmStateReader) LastHeaderWithArchiveState() (*evmcore.EvmHeader, error) {
	latestBlock := r.store.GetLatestBlockIndex()

	// make sure the block is present in the archive
	latestArchiveBlock, empty, err := r.store.evm.GetArchiveBlockHeight()
	if err != nil {
		return nil, fmt.Errorf("failed to get latest archive block; %v", err)
	}
	if !empty && idx.Block(latestArchiveBlock) < latestBlock {
		latestBlock = idx.Block(latestArchiveBlock)
	}

	return r.getBlock(common.Hash{}, latestBlock, false).Header(), nil
}

func (r *EvmStateReader) GetHeaderByNumber(n uint64) *evmcore.EvmHeader {
	return r.GetHeader(common.Hash{}, n)
}

func (r *EvmStateReader) GetHeader(h common.Hash, n uint64) *evmcore.EvmHeader {
	return r.getBlock(h, idx.Block(n), false).Header()
}

func (r *EvmStateReader) GetBlock(h common.Hash, n uint64) *evmcore.EvmBlock {
	return r.getBlock(h, idx.Block(n), true)
}

func (r *EvmStateReader) getBlock(h common.Hash, n idx.Block, readTxs bool) *evmcore.EvmBlock {
	block := r.store.GetBlock(n)
	if block == nil {
		return nil
	}
	if (h != common.Hash{}) && (h != block.Hash()) {
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

// GetTxPoolStateDB obtains StateDB for TxPool
func (r *EvmStateReader) GetTxPoolStateDB() (state.StateDB, error) {
	return r.store.evm.GetTxPoolStateDB()
}

// GetRpcStateDB obtains archive StateDB for RPC requests evaluation
func (r *EvmStateReader) GetRpcStateDB(blockNum *big.Int, stateRoot common.Hash) (state.StateDB, error) {
	return r.store.evm.GetRpcStateDb(blockNum, stateRoot)
}
