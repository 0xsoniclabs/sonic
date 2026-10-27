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
	"math"
	"math/big"
	"testing"
	"time"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/evmcore/core_types"
	"github.com/0xsoniclabs/sonic/gossip/blockproc"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/evmmodule"
	"github.com/0xsoniclabs/sonic/gossip/evmstore"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
)

func TestLedger_Head_ReturnsHandleToBlockAtHeadPointer(t *testing.T) {
	store := newInMemoryStoreWithGenesisData(t, opera.GetSonicUpgrades(), 1, 2)
	headIdx := store.GetLatestBlockIndex()

	head, err := NewLedger(store, nil, nil).Head()
	require.NoError(t, err)
	require.Equal(t, store.GetBlock(headIdx).Hash(), head.Block().Hash())
}

func TestLedger_Head_HeadBlockMissingFromStore_ReportsError(t *testing.T) {
	store := newInMemoryStoreWithGenesisData(t, opera.GetSonicUpgrades(), 1, 2)
	bs := store.GetBlockState()
	bs.LastBlock.Idx++
	store.SetBlockEpochState(bs, store.GetEpochState())

	head, err := NewLedger(store, nil, nil).Head()
	require.ErrorContains(t, err, "not found in store")
	require.Nil(t, head)
}

func TestHead_BeginBlock_OpensLiveStateAtHeadStateRoot(t *testing.T) {
	store := newInMemoryStoreWithGenesisData(t, opera.GetSonicUpgrades(), 1, 2)
	head, err := NewLedger(store, evmmodule.New(), nil).Head()
	require.NoError(t, err)

	blockParams := BlockParams{Time: head.Block().Time + 1}
	executing, err := head.BeginBlock(blockParams, nil)
	require.NoError(t, err)
	require.Equal(t, head.Block().StateRoot, executing.stateDB().GetStateHash())
}

func TestHead_BeginBlock_SecondCallOnSameHead_Panics(t *testing.T) {
	store := newInMemoryStoreWithGenesisData(t, opera.GetSonicUpgrades(), 1, 2)
	head, err := NewLedger(store, evmmodule.New(), nil).Head()
	require.NoError(t, err)
	blockParams := BlockParams{Time: head.Block().Time + 1}
	_, err = head.BeginBlock(blockParams, nil)
	require.NoError(t, err)

	require.PanicsWithValue(t, "head already consumed", func() { _, _ = head.BeginBlock(blockParams, nil) })
}

func TestHead_BeginBlock_FailedCallDoesNotConsumeHead(t *testing.T) {
	tests := map[string]struct {
		setup   func(l *ledger, blockParams *BlockParams) (undo func())
		wantErr string
	}{
		"block time not after head": {
			setup: func(_ *ledger, blockParams *BlockParams) func() {
				blockParams.Time--
				return func() { blockParams.Time++ }
			},
			wantErr: "is not after head time",
		},
		"head pointer moved on": {
			setup: func(l *ledger, _ *BlockParams) func() {
				bs := l.store.GetBlockState()
				bs.LastBlock.Idx++
				l.store.SetBlockEpochState(bs, l.store.GetEpochState())
				return func() {
					bs.LastBlock.Idx--
					l.store.SetBlockEpochState(bs, l.store.GetEpochState())
				}
			},
			wantErr: "is no longer the ledger's head",
		},
		"live state cannot be opened": {
			setup: func(l *ledger, _ *BlockParams) func() {
				evm := l.store.evm
				l.store.evm = &evmstore.Store{}
				return func() { l.store.evm = evm }
			},
			wantErr: "EvmStore is not open",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			store := newInMemoryStoreWithGenesisData(t, opera.GetSonicUpgrades(), 1, 2)
			l := NewLedger(store, evmmodule.New(), nil)
			head, err := l.Head()
			require.NoError(t, err)
			blockParams := BlockParams{Time: head.Block().Time + 1}

			undo := test.setup(l, &blockParams)
			_, err = head.BeginBlock(blockParams, nil)
			require.ErrorContains(t, err, test.wantErr)

			undo()
			_, err = head.BeginBlock(blockParams, nil)
			require.NoError(t, err)
		})
	}
}

func TestHead_BeginBlock_StartsEvmAndBlockBuilderFromParamsAndHead(t *testing.T) {
	ctrl := gomock.NewController(t)

	store := newInMemoryStoreWithGenesisData(t, opera.GetSonicUpgrades(), 1, 2)
	feed := &ServiceFeed{}
	evm := blockproc.NewMockEVM(ctrl)
	head, err := NewLedger(store, evm, feed).Head()
	require.NoError(t, err)

	parent := head.Block()
	blockParams := BlockParams{
		Time:                     parent.Time + 3,
		Epoch:                    4,
		PrevRandao:               common.Hash{5},
		GasLimit:                 6,
		UserTransactionsGasLimit: 7,
		Rules:                    opera.FakeNetRules(opera.GetSonicUpgrades()),
	}
	evmCfg := opera.CreateTransientEvmChainConfig(
		blockParams.Rules.NetworkID, store.GetUpgradeHeights(), idx.Block(parent.Number+1),
	)
	newLog := &core_types.Log{}
	var gotLog *core_types.Log
	var startedOn state.StateDB

	evmProcessor := blockproc.NewMockEVMProcessor(ctrl)
	evm.EXPECT().Start(
		idx.Block(parent.Number+1), blockParams.Time, blockParams.Epoch, gomock.Any(), &EvmStateReader{ServiceFeed: feed, store: store},
		gomock.Any(), blockParams.Rules, evmCfg, blockParams.PrevRandao, sonicFeaturesMetrics,
	).DoAndReturn(func(
		_ idx.Block, _ inter.Timestamp, _ idx.Epoch, statedb state.StateDB, _ evmcore.DummyChain,
		onNewLog func(*core_types.Log), _ opera.Rules, _ *params.ChainConfig, _ common.Hash,
		_ evmcore.BlockExecutionMetrics,
	) blockproc.EVMProcessor {
		startedOn = statedb
		onNewLog(newLog)
		return evmProcessor
	})

	executing, err := head.BeginBlock(blockParams, func(log *core_types.Log) { gotLog = log })
	require.NoError(t, err)
	require.Same(t, newLog, gotLog)

	eb := executing.(*executingBlock)
	require.Same(t, startedOn, eb.stateDB())
	require.Same(t, evmProcessor, eb.evmProcessor)
	require.Equal(t, blockParams, eb.params)

	block := eb.blockBuilder.Build()
	require.Equal(t, parent.Number+1, block.Number)
	require.Equal(t, parent.Hash(), block.ParentHash)
	require.Equal(t, blockParams.Time, block.Time)
	require.Equal(t, uint64(3), block.Duration)
	require.Equal(t, blockParams.Epoch, block.Epoch)
	require.Equal(t, blockParams.PrevRandao, block.PrevRandao)
	require.Equal(t, blockParams.GasLimit, block.GasLimit)
}

func TestHead_BeginBlock_FailedPrecondition_ReportsError(t *testing.T) {
	tests := map[string]struct {
		beforeHead func(l *ledger)
		afterHead  func(l *ledger, h Head, blockParams *BlockParams)
		wantErr    string
	}{
		"block time not after head": {
			afterHead: func(_ *ledger, _ Head, blockParams *BlockParams) {
				blockParams.Time--
			},
			wantErr: "is not after head time",
		},
		"head pointer moved on": {
			afterHead: func(l *ledger, _ Head, _ *BlockParams) {
				bs := l.store.GetBlockState()
				bs.LastBlock.Idx++
				l.store.SetBlockEpochState(bs, l.store.GetEpochState())
			},
			wantErr: "is no longer the ledger's head",
		},
		"head block replaced": {
			afterHead: func(l *ledger, h Head, _ *BlockParams) {
				number := h.Block().Number
				l.store.SetBlock(idx.Block(number), inter.NewBlockBuilder().
					WithNumber(number).
					Build())
			},
			wantErr: "is no longer the ledger's head",
		},
		"live state does not match the head": {
			beforeHead: func(l *ledger) {
				number := l.store.GetLatestBlockIndex()
				l.store.SetBlock(number, inter.NewBlockBuilder().
					WithNumber(uint64(number)).
					WithStateRoot(common.Hash{1}).
					Build())
			},
			wantErr: "unexpected state root",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			store := newInMemoryStoreWithGenesisData(t, opera.GetSonicUpgrades(), 1, 2)
			l := NewLedger(store, nil, nil)
			if test.beforeHead != nil {
				test.beforeHead(l)
			}
			head, err := l.Head()
			require.NoError(t, err)

			blockParams := BlockParams{Time: head.Block().Time + 1}
			if test.afterHead != nil {
				test.afterHead(l, head, &blockParams)
			}
			executing, err := head.BeginBlock(blockParams, nil)
			require.ErrorContains(t, err, test.wantErr)
			require.Nil(t, executing)
		})
	}
}

func TestExecutingBlock_Run_ExecutesWithinUserTransactionsGasLimit(t *testing.T) {
	txs := types.Transactions{types.NewTx(&types.LegacyTx{Nonce: 1})}
	blockParams := BlockParams{GasLimit: 4, UserTransactionsGasLimit: 3}

	evmProcessor := blockproc.NewMockEVMProcessor(gomock.NewController(t))
	evmProcessor.EXPECT().Execute(txs, blockParams.UserTransactionsGasLimit, uint64(math.MaxUint64)).Return(evmcore.ProcessSummary{})

	executing := &executingBlock{
		evmProcessor: evmProcessor,
		blockBuilder: inter.NewBlockBuilder(),
		params:       blockParams,
	}
	executing.Run(txs)
}

func TestExecutingBlock_runInternal_ExecutesWithinBlockGasLimitAndUnlimitedSize(t *testing.T) {
	accepted := types.NewTx(&types.LegacyTx{Nonce: 1})
	skipped := types.NewTx(&types.LegacyTx{Nonce: 2})
	txs := types.Transactions{accepted, skipped}
	summary := evmcore.ProcessSummary{ProcessedTransactions: []evmcore.ProcessedTransaction{
		{Transaction: accepted, Receipt: &types.Receipt{}},
		{Transaction: skipped},
	}}

	blockParams := BlockParams{
		GasLimit:                 4,
		UserTransactionsGasLimit: 3,
		// Brio enables the size limit for user batches, so this checks that internal batches ignore it.
		Rules: opera.Rules{Upgrades: opera.Upgrades{Brio: true}},
	}

	evmProcessor := blockproc.NewMockEVMProcessor(gomock.NewController(t))
	evmProcessor.EXPECT().Execute(txs, blockParams.GasLimit, uint64(math.MaxUint64)).Return(summary)

	executing := &executingBlock{
		evmProcessor: evmProcessor,
		blockBuilder: inter.NewBlockBuilder(),
		params:       blockParams,
	}

	require.Equal(t, summary, executing.runInternal(txs))
	require.Equal(t, types.Transactions{accepted}, executing.blockBuilder.GetTransactions())
}

func TestExecutingBlock_Finalize_AppliesSonicMainnetGasLimitException(t *testing.T) {
	const announcedGasLimit = uint64(123)

	tests := map[string]struct {
		networkID uint64
		number    uint64
		want      uint64
	}{
		"Sonic mainnet block 8054923 uses the hard-coded exception": {
			networkID: 146,
			number:    8054923,
			want:      0x12a05f200, // header gas limit of that block on Sonic mainnet
		},
		"different block on Sonic mainnet keeps the announced limit": {
			networkID: 146,
			number:    8054924,
			want:      announcedGasLimit,
		},
		"same block number on a different network keeps the announced limit": {
			networkID: 250,
			number:    8054923,
			want:      announcedGasLimit,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			evmProcessor := blockproc.NewMockEVMProcessor(ctrl)
			evmProcessor.EXPECT().Finalize().Return(&evmcore.EvmBlock{
				EvmHeader: evmcore.EvmHeader{BaseFee: big.NewInt(0)},
			}, 0, types.Receipts{})

			executing := &executingBlock{
				number:       test.number,
				evmProcessor: evmProcessor,
				blockBuilder: inter.NewBlockBuilder().
					WithNumber(test.number).
					WithGasLimit(announcedGasLimit),
				params: BlockParams{
					Rules: opera.Rules{
						NetworkID: test.networkID,
						// Differs from the hard-coded limit, so the exception can't come from the rules.
						Blocks: opera.BlocksRules{MaxBlockGas: 456},
					},
				},
			}

			candidate := executing.Finalize().Candidate()
			require.Equal(t, test.want, candidate.Block.GasLimit)
		})
	}
}

func TestExecutingBlock_Finalize_AssemblesBlockFromExecutionResults(t *testing.T) {
	receipts := types.Receipts{{Logs: []*types.Log{{}, {}}}}

	evmProcessor := blockproc.NewMockEVMProcessor(gomock.NewController(t))
	evmProcessor.EXPECT().Finalize().Return(&evmcore.EvmBlock{EvmHeader: evmcore.EvmHeader{
		Root:    common.Hash{1},
		GasUsed: 2,
		BaseFee: big.NewInt(3),
	}}, 4, receipts)

	executing := &executingBlock{
		evmProcessor: evmProcessor,
		blockBuilder: inter.NewBlockBuilder().WithTime(inter.Timestamp(5 * time.Second)).WithDuration(6),
	}

	candidate := executing.Finalize().Candidate()
	block := candidate.Block
	require.Equal(t, common.Hash{1}, block.StateRoot)
	require.Equal(t, uint64(2), block.GasUsed)
	require.Equal(t, big.NewInt(3), block.BaseFee)
	require.Equal(t, block.Hash(), candidate.EvmBlock.Hash)
	require.Equal(t, time.Duration(6), candidate.EvmBlock.Duration)
	require.Equal(t, 4, candidate.NumSkipped)
	require.Equal(t, receipts, candidate.Receipts)
	for _, receipt := range candidate.Receipts {
		require.Equal(t, block.Hash(), receipt.BlockHash)
		for _, log := range receipt.Logs {
			require.Equal(t, block.Hash(), log.BlockHash)
			require.Equal(t, uint64(5), log.BlockTimestamp)
		}
	}
}

func TestExecutingBlock_AfterFinalize_Panics(t *testing.T) {
	tests := map[string]func(ExecutingBlock){
		"stateDB":     func(executing ExecutingBlock) { executing.stateDB() },
		"Run":         func(executing ExecutingBlock) { executing.Run(nil) },
		"runInternal": func(executing ExecutingBlock) { executing.runInternal(nil) },
		"Finalize":    func(executing ExecutingBlock) { executing.Finalize() },
	}

	for name, call := range tests {
		t.Run(name, func(t *testing.T) {
			evmProcessor := blockproc.NewMockEVMProcessor(gomock.NewController(t))
			evmProcessor.EXPECT().Finalize().Return(&evmcore.EvmBlock{EvmHeader: evmcore.EvmHeader{BaseFee: big.NewInt(0)}}, 0, nil)
			executing := &executingBlock{evmProcessor: evmProcessor, blockBuilder: inter.NewBlockBuilder()}
			executing.Finalize()

			require.PanicsWithValue(t, "executing block already consumed", func() { call(executing) })
		})
	}
}
