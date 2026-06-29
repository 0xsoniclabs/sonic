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

package gossip

import (
	"math/big"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/priorities"
	"github.com/0xsoniclabs/sonic/gossip/gasprice"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

// applyTransactionPriorities reorders the (already base-ordered and filtered)
// transactions of a block according to the on-chain priority registry. It is the
// single authoritative ordering step and is applied identically in both legacy
// and single-proposer modes (so a proposer's order is overridden).
//
// It is a no-op unless the TransactionPriorities upgrade is active. Every
// registry query is run against the block-start state through a snapshot that is
// immediately reverted, so the queries leave no residue in the state used for
// block execution. The query EVM context is derived exclusively from consensus
// inputs, so every validator reproduces the same ordering. Any query error is
// treated as "not prioritized" and never aborts block formation.
func applyTransactionPriorities(
	txs types.Transactions,
	rules opera.Rules,
	chainCfg *params.ChainConfig,
	statedb state.StateDB,
	reader evmcore.DummyChain,
	signer types.Signer,
	blockIdx idx.Block,
	blockTime inter.Timestamp,
	randao common.Hash,
	parent *evmcore.EvmHeader,
) types.Transactions {
	if !rules.Upgrades.TransactionPriorities || len(txs) == 0 {
		return txs
	}

	header := priorityQueryHeader(rules, blockIdx, blockTime, randao, parent)
	blockContext := evmcore.NewEVMBlockContext(header, reader, nil)
	evm := vm.NewEVM(blockContext, statedb, chainCfg, opera.GetVmConfig(rules))

	cfg, err := priorities.GetConfig(rules.Upgrades, evm)
	if err != nil {
		// Deterministic fallback: every validator that fails to read the config
		// reaches the same (un-prioritized) ordering.
		log.Warn("failed to read priority config, using fallback", "err", err)
		cfg = priorities.FallbackConfig
	}

	classifier := priorities.NewEvmClassifier(rules.Upgrades, evm, signer, statedb)
	return priorities.Prioritize(txs, classifier, cfg)
}

// priorityQueryHeader builds the EVM header used for priority registry queries.
// It mirrors the header construction in the EVM module (evmmodule.evmBlockWith)
// using the same consensus-derived inputs, so the query context matches the block
// being formed and is identical across validators.
func priorityQueryHeader(
	rules opera.Rules,
	blockIdx idx.Block,
	blockTime inter.Timestamp,
	randao common.Hash,
	parent *evmcore.EvmHeader,
) *evmcore.EvmHeader {
	baseFee := gasprice.GetBaseFeeForNextBlock(gasprice.ParentBlockInfo{
		BaseFee:  parent.BaseFee,
		Duration: parent.Duration,
		GasUsed:  parent.GasUsed,
	}, rules.Economy)

	return &evmcore.EvmHeader{
		Number:      new(big.Int).SetUint64(uint64(blockIdx)),
		ParentHash:  parent.Hash,
		Time:        blockTime,
		Coinbase:    evmcore.GetCoinbase(),
		GasLimit:    rules.Blocks.MaxBlockGas,
		BaseFee:     baseFee,
		BlobBaseFee: evmcore.GetBlobBaseFee().ToBig(),
		PrevRandao:  randao,
	}
}
