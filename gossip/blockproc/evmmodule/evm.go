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

package evmmodule

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/blockproc"
	"github.com/0xsoniclabs/sonic/gossip/gasprice"
	"github.com/0xsoniclabs/sonic/inter/iblockproc"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
)

type EVMModule struct{}

func New() *EVMModule {
	return &EVMModule{}
}

func (p *EVMModule) Start(
	block iblockproc.BlockCtx,
	statedb state.StateDB,
	reader evmcore.DummyChain,
	onNewLog func(*evmcore.Log),
	net opera.Rules,
	evmCfg *params.ChainConfig,
	prevrandao common.Hash,
) blockproc.EVMProcessor {
	var prevBlockHash common.Hash
	var baseFee *big.Int
	if block.Idx == 0 {
		baseFee = gasprice.GetInitialBaseFee(net.Economy)
	} else {
		header := reader.GetHeader(common.Hash{}, uint64(block.Idx-1))
		prevBlockHash = header.Hash
		baseFee = gasprice.GetBaseFeeForNextBlock(gasprice.ParentBlockInfo{
			BaseFee:  header.BaseFee,
			Duration: header.Duration,
			GasUsed:  header.GasUsed,
		}, net.Economy)
	}

	// Start block
	statedb.BeginBlock(uint64(block.Idx))

	return &OperaEVMProcessor{
		block:         block,
		reader:        reader,
		statedb:       statedb,
		onNewLog:      onNewLog,
		net:           net,
		evmCfg:        evmCfg,
		blockIdx:      uint64(block.Idx),
		prevBlockHash: prevBlockHash,
		prevRandao:    prevrandao,
		gasBaseFee:    baseFee,
		rules:         net,
	}
}

type OperaEVMProcessor struct {
	block    iblockproc.BlockCtx
	reader   evmcore.DummyChain
	statedb  state.StateDB
	onNewLog func(*evmcore.Log)
	net      opera.Rules
	evmCfg   *params.ChainConfig

	blockIdx      uint64
	prevBlockHash common.Hash
	gasBaseFee    *big.Int

	gasUsed uint64

	incomingTxs types.Transactions
	skippedTxs  []uint32
	receipts    types.Receipts
	prevRandao  common.Hash

	rules opera.Rules
}

func (p *OperaEVMProcessor) evmBlockWith(txs types.Transactions) *evmcore.EvmBlock {
	baseFee := p.net.Economy.MinGasPrice
	if !p.net.Upgrades.London {
		baseFee = nil
	} else if p.net.Upgrades.Sonic {
		baseFee = p.gasBaseFee
	}

	prevRandao := common.Hash{}
	// This condition must be kept, otherwise Sonic will not be able to synchronize
	if p.net.Upgrades.Sonic {
		prevRandao = p.prevRandao
	}

	var withdrawalsHash *common.Hash = nil
	if p.net.Upgrades.Sonic {
		withdrawalsHash = &types.EmptyWithdrawalsHash
	}

	blobBaseFee := evmcore.GetBlobBaseFee()
	h := &evmcore.EvmHeader{
		Number:          new(big.Int).SetUint64(p.blockIdx),
		ParentHash:      p.prevBlockHash,
		Root:            common.Hash{}, // state root is added later
		Time:            p.block.Time,
		Coinbase:        evmcore.GetCoinbase(),
		GasLimit:        p.net.Blocks.MaxBlockGas,
		GasUsed:         p.gasUsed,
		BaseFee:         baseFee,
		BlobBaseFee:     blobBaseFee.ToBig(),
		PrevRandao:      prevRandao,
		WithdrawalsHash: withdrawalsHash,
		Epoch:           p.block.Atropos.Epoch(),
	}

	return evmcore.NewEvmBlock(h, txs)
}

func (p *OperaEVMProcessor) Execute(txs types.Transactions, gasLimit uint64) types.Receipts {
	evmProcessor := evmcore.NewStateProcessor(p.evmCfg, p.reader)
	txsOffset := uint(len(p.incomingTxs))

	vmConfig := opera.GetVmConfig(p.rules)

	// Process txs
	evmBlock := p.evmBlockWith(txs)
	receipts, skipped := evmProcessor.Process(evmBlock, p.statedb, vmConfig, gasLimit, &p.gasUsed, p.onNewLog)

	if txsOffset > 0 {
		for i, n := range skipped {
			skipped[i] = n + uint32(txsOffset)
		}
		for _, r := range receipts {
			if r != nil {
				r.TransactionIndex += txsOffset
				for _, l := range r.Logs {
					l.TxIndex += txsOffset
				}
			}
		}
	}

	p.incomingTxs = append(p.incomingTxs, txs...)
	p.skippedTxs = append(p.skippedTxs, skipped...)
	for _, receipt := range receipts {
		if receipt != nil {
			p.receipts = append(p.receipts, receipt)
		}
	}

	return receipts
}

func (p *OperaEVMProcessor) Finalize() (evmBlock *evmcore.EvmBlock, numSkipped int, receipts types.Receipts) {
	evmBlock = p.evmBlockWith(
		// Filter skipped transactions. Receipts are filtered already
		filterSkippedTxs(p.incomingTxs, p.skippedTxs),
	)
	numSkipped = len(p.skippedTxs)
	receipts = p.receipts

	// Commit block
	p.statedb.EndBlock(evmBlock.Number.Uint64())

	// Get state root
	evmBlock.Root = p.statedb.GetStateHash()

	return
}

func filterSkippedTxs(txs types.Transactions, skippedTxs []uint32) types.Transactions {
	if len(skippedTxs) == 0 {
		// short circuit if nothing to skip
		return txs
	}
	skipCount := 0
	filteredTxs := make(types.Transactions, 0, len(txs))
	for i, tx := range txs {
		if skipCount < len(skippedTxs) && skippedTxs[skipCount] == uint32(i) {
			skipCount++
		} else {
			filteredTxs = append(filteredTxs, tx)
		}
	}

	return filteredTxs
}
