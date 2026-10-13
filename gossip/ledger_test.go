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
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/evmcore/core_types"
	"github.com/0xsoniclabs/sonic/gossip/blockproc"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/evmmodule"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
)

func TestNewLedger_DefaultsNilExecutionMetricsToSharedSink(t *testing.T) {
	custom := &evmcore.SonicBlockExecutionMetrics{}
	tests := map[string]struct {
		metrics evmcore.BlockExecutionMetrics
		want    evmcore.BlockExecutionMetrics
	}{
		"nil metrics use the shared sink": {metrics: nil, want: sonicFeaturesMetrics},
		"given metrics are kept":          {metrics: custom, want: custom},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			l := NewLedger(nil, LedgerConfig{ExecutionMetrics: test.metrics})
			require.Same(t, test.want, l.config.ExecutionMetrics)
			require.IsType(t, &evmmodule.EVMModule{}, l.evmModule)
		})
	}
}

func TestLedger_Head_ReturnsHandleToBlockAtHeadPointer(t *testing.T) {
	store := newInMemoryStoreWithGenesisData(t, opera.GetSonicUpgrades(), 1, 2)
	headIdx := store.GetLatestBlockIndex()

	head, err := NewLedger(store, LedgerConfig{}).Head()
	require.NoError(t, err)
	require.Equal(t, store.GetBlock(headIdx).Hash(), head.Block().Hash())
}

func TestLedger_Head_HeadBlockMissingFromStore_ReportsError(t *testing.T) {
	store := newInMemoryStoreWithGenesisData(t, opera.GetSonicUpgrades(), 1, 2)
	bs := store.GetBlockState()
	bs.LastBlock.Idx++
	store.SetBlockEpochState(bs, store.GetEpochState())

	head, err := NewLedger(store, LedgerConfig{}).Head()
	require.ErrorContains(t, err, "not found in store")
	require.Nil(t, head)
}

func TestHead_BeginBlock_OpensLiveStateAtHeadStateRoot(t *testing.T) {
	store := newInMemoryStoreWithGenesisData(t, opera.GetSonicUpgrades(), 1, 2)
	head, err := NewLedger(store, LedgerConfig{}).Head()
	require.NoError(t, err)

	next := BlockParams{Number: head.Block().Number + 1, Time: head.Block().Time + 1}
	executing, err := head.BeginBlock(next, nil)
	require.NoError(t, err)
	require.Equal(t, head.Block().StateRoot, executing.stateDB().GetStateHash())
}

func TestHead_BeginBlock_SecondCallOnSameHead_ReportsError(t *testing.T) {
	store := newInMemoryStoreWithGenesisData(t, opera.GetSonicUpgrades(), 1, 2)
	head, err := NewLedger(store, LedgerConfig{}).Head()
	require.NoError(t, err)
	next := BlockParams{Number: head.Block().Number + 1, Time: head.Block().Time + 1}
	_, err = head.BeginBlock(next, nil)
	require.NoError(t, err)

	executing, err := head.BeginBlock(next, nil)
	require.ErrorContains(t, err, "head already consumed")
	require.Nil(t, executing)
}

func TestHead_BeginBlock_FailedCallDoesNotConsumeHead(t *testing.T) {
	store := newInMemoryStoreWithGenesisData(t, opera.GetSonicUpgrades(), 1, 2)
	head, err := NewLedger(store, LedgerConfig{}).Head()
	require.NoError(t, err)
	_, err = head.BeginBlock(BlockParams{Number: head.Block().Number + 2, Time: head.Block().Time + 1}, nil)
	require.Error(t, err)

	_, err = head.BeginBlock(BlockParams{Number: head.Block().Number + 1, Time: head.Block().Time + 1}, nil)
	require.NoError(t, err)
}

func TestHead_BeginBlock_StartsEvmAndBlockBuilderFromParamsAndHead(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	store := newInMemoryStoreWithGenesisData(t, opera.GetSonicUpgrades(), 1, 2)
	metrics := &evmcore.SonicBlockExecutionMetrics{}
	evm := blockproc.NewMockEVM(ctrl)
	head, err := newLedgerInternal(store, LedgerConfig{ExecutionMetrics: metrics}, evm).Head()
	require.NoError(err)
	parent := head.Block()
	blockParams := BlockParams{
		Number:       parent.Number + 1,
		Time:         parent.Time + 12,
		Epoch:        7,
		PrevRandao:   common.Hash{9},
		GasLimit:     10,
		UserGasLimit: 11,
		Rules:        opera.FakeNetRules(opera.GetSonicUpgrades()),
	}
	evmCfg := opera.CreateTransientEvmChainConfig(
		blockParams.Rules.NetworkID, store.GetUpgradeHeights(), idx.Block(blockParams.Number),
	)
	newLog := &core_types.Log{}
	var gotLog *core_types.Log
	var startedOn state.StateDB

	evmProcessor := blockproc.NewMockEVMProcessor(ctrl)
	evm.EXPECT().Start(
		idx.Block(blockParams.Number), blockParams.Time, idx.Epoch(7), gomock.Any(), &EvmStateReader{store: store},
		gomock.Any(), blockParams.Rules, evmCfg, blockParams.PrevRandao, metrics,
	).DoAndReturn(func(
		_ idx.Block, _ inter.Timestamp, _ idx.Epoch, statedb state.StateDB, _ evmcore.DummyChain,
		onNewLog func(*core_types.Log), _ opera.Rules, _ *params.ChainConfig, _ common.Hash,
		_ evmcore.BlockExecutionMetrics,
	) blockproc.EVMProcessor {
		startedOn = statedb
		onNewLog(newLog)
		return evmProcessor
	})

	eb, err := head.BeginBlock(blockParams, func(log *core_types.Log) { gotLog = log })
	require.NoError(err)
	require.Same(newLog, gotLog)

	require.Same(startedOn, eb.stateDB())
	require.Same(evmProcessor, eb.evmProcessor)
	require.Equal(blockParams, eb.params)

	block := eb.blockBuilder.Build()
	require.Equal(idx.Epoch(7), block.Epoch)
	require.Equal(blockParams.Number, block.Number)
	require.Equal(parent.Hash(), block.ParentHash)
	require.Equal(blockParams.Time, block.Time)
	require.Equal(common.Hash{9}, block.PrevRandao)
	require.Equal(uint64(10), block.GasLimit)
	require.Equal(uint64(12), block.Duration)
}

func TestHead_BeginBlock_RejectsBlockNotOnTheLedgerHead(t *testing.T) {
	tests := map[string]struct {
		setup   func(l *Ledger, h *Head) BlockParams
		wantErr string
	}{
		"number does not follow the head": {
			setup: func(_ *Ledger, h *Head) BlockParams {
				return BlockParams{Number: h.Block().Number + 2, Time: h.Block().Time + 1}
			},
			wantErr: "does not follow head",
		},
		"time not after the head": {
			setup: func(_ *Ledger, h *Head) BlockParams {
				return BlockParams{Number: h.Block().Number + 1, Time: h.Block().Time}
			},
			wantErr: "is not after head time",
		},
		"head pointer moved on": {
			setup: func(l *Ledger, h *Head) BlockParams {
				bs := l.store.GetBlockState()
				bs.LastBlock.Idx++
				l.store.SetBlockEpochState(bs, l.store.GetEpochState())
				return BlockParams{Number: h.Block().Number + 1, Time: h.Block().Time + 1}
			},
			wantErr: "no longer the ledger's head",
		},
		"head block replaced": {
			setup: func(l *Ledger, h *Head) BlockParams {
				number := h.Block().Number
				l.store.SetBlock(idx.Block(number), inter.NewBlockBuilder().
					WithNumber(number).
					Build())
				return BlockParams{Number: h.Block().Number + 1, Time: h.Block().Time + 1}
			},
			wantErr: "no longer the ledger's head",
		},
		"live state does not match the head": {
			setup: func(l *Ledger, h *Head) BlockParams {
				block := inter.NewBlockBuilder().
					WithNumber(h.Block().Number).
					WithStateRoot(common.Hash{1}).
					Build()
				l.store.SetBlock(idx.Block(block.Number), block)
				h.block = block
				return BlockParams{Number: block.Number + 1, Time: h.Block().Time + 1}
			},
			wantErr: "unexpected state root",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			store := newInMemoryStoreWithGenesisData(
				t, opera.GetSonicUpgrades(), 1, 2,
			)
			l := newLedgerInternal(store, LedgerConfig{}, nil)
			head, err := l.Head()
			require.NoError(t, err)

			executing, err := head.BeginBlock(test.setup(l, head), nil)
			require.ErrorContains(t, err, test.wantErr)
			require.Nil(t, executing)
		})
	}
}

func TestExecutingBlock_Run_ForwardsBlockGasLimitToEVMProcessor(t *testing.T) {
	ctrl := gomock.NewController(t)
	evmProcessor := blockproc.NewMockEVMProcessor(ctrl)
	blockBuilder := inter.NewBlockBuilder()

	// Create some dummy transactions with gas usage
	tx1 := types.NewTx(&types.LegacyTx{Nonce: 1, Gas: 1000})
	tx2 := types.NewTx(&types.LegacyTx{Nonce: 2, Gas: 1000})
	tx3 := types.NewTx(&types.LegacyTx{Nonce: 3, Gas: 1000})

	// Set up receipts for each transaction
	receipt1 := &types.Receipt{GasUsed: 1000}
	receipt2 := &types.Receipt{GasUsed: 1000}
	receipt3 := &types.Receipt{GasUsed: 1000}

	userTransactionGasLimit := uint64(3000)

	orderedTxs := []*types.Transaction{tx1, tx2, tx3}

	// Mock EVMProcessor.Execute to be called once with all txs and the full gas limit
	evmProcessor.EXPECT().
		Execute(orderedTxs, userTransactionGasLimit, gomock.Any()).
		Return(evmcore.ProcessSummary{ProcessedTransactions: []evmcore.ProcessedTransaction{
			{Transaction: tx1, Receipt: receipt1},
			{Transaction: tx2, Receipt: receipt2},
			{Transaction: tx3, Receipt: receipt3},
		}})

	upgrades := opera.Upgrades{Brio: true}
	processor := &ExecutingBlock{
		evmProcessor: evmProcessor,
		blockBuilder: blockBuilder,
		params:       BlockParams{UserGasLimit: userTransactionGasLimit, Rules: opera.Rules{Upgrades: upgrades}},
	}
	processor.Run(orderedTxs)

	// All transactions should be included
	gotTxs := blockBuilder.GetTransactions()
	require.Equal(t, types.Transactions{tx1, tx2, tx3}, gotTxs)
}

func TestExecutingBlock_Run_TransactionsWithNoReceiptAreNotIncluded(t *testing.T) {
	ctrl := gomock.NewController(t)
	evmProcessor := blockproc.NewMockEVMProcessor(ctrl)
	blockBuilder := inter.NewBlockBuilder()

	tx := types.NewTx(&types.LegacyTx{Nonce: 1})

	// Simulate skipped transaction (no receipt)
	evmProcessor.EXPECT().
		Execute([]*types.Transaction{tx}, gomock.Any(), gomock.Any()).
		Return(evmcore.ProcessSummary{ProcessedTransactions: []evmcore.ProcessedTransaction{{Transaction: tx, Receipt: nil}}})

	upgrades := opera.Upgrades{Brio: true}
	processor := &ExecutingBlock{
		evmProcessor: evmProcessor,
		blockBuilder: blockBuilder,
		params:       BlockParams{UserGasLimit: 10000, Rules: opera.Rules{Upgrades: upgrades}},
	}
	processor.Run([]*types.Transaction{tx})

	// Should not be added to blockBuilder
	gotTxs := blockBuilder.GetTransactions()
	require.Empty(t, gotTxs)
}

func TestExecutingBlock_Run_DeductsInternalTxsSize(t *testing.T) {
	ctrl := gomock.NewController(t)
	evmProcessor := blockproc.NewMockEVMProcessor(ctrl)
	blockBuilder := inter.NewBlockBuilder()

	// Add an internal tx to blockBuilder
	internalTx := types.NewTx(&types.LegacyTx{Data: make([]byte, 1000)})
	blockBuilder.AddTransaction(internalTx, &types.Receipt{})

	// Create a user tx
	userTx := types.NewTx(&types.LegacyTx{Data: make([]byte, 1000)})

	// Ensure the evmProcessor is called with the correct remaining size after accounting for the internal transaction.
	orderedTxs := []*types.Transaction{userTx}
	evmProcessor.EXPECT().
		Execute(orderedTxs, uint64(10000), params.MaxBlockSize-rlpEncodedMaxHeaderSizeInBytes-internalTx.Size()).
		Return(evmcore.ProcessSummary{ProcessedTransactions: []evmcore.ProcessedTransaction{
			{Transaction: userTx, Receipt: &types.Receipt{}},
		}})

	upgrades := opera.Upgrades{Brio: true}
	processor := &ExecutingBlock{
		evmProcessor: evmProcessor,
		blockBuilder: blockBuilder,
		params:       BlockParams{UserGasLimit: 10000, Rules: opera.Rules{Upgrades: upgrades}},
	}
	processor.Run(orderedTxs)

	// Ensure both internal and user transactions are included.
	gotTxs := blockBuilder.GetTransactions()
	require.Equal(t, types.Transactions{internalTx, userTx}, gotTxs)
}

func TestExecutingBlock_Run_SkipsTxsExceedingSizeLimit(t *testing.T) {
	ctrl := gomock.NewController(t)
	evmProcessor := blockproc.NewMockEVMProcessor(ctrl)
	blockBuilder := inter.NewBlockBuilder()

	// Create a tx that exceeds the size limit
	largeTx := types.NewTx(&types.LegacyTx{Data: make([]byte, params.MaxBlockSize)})

	// Following txs are not skipped
	tx0 := types.NewTx(&types.LegacyTx{Data: []byte{0x01}})
	tx1 := types.NewTx(&types.LegacyTx{Data: []byte{0x01}})

	orderedTxs := []*types.Transaction{largeTx, tx0, tx1}

	// All txs are passed to Execute; the EVMProcessor handles size-based skipping
	evmProcessor.EXPECT().
		Execute(orderedTxs, uint64(10000), gomock.Any()).
		Return(evmcore.ProcessSummary{ProcessedTransactions: []evmcore.ProcessedTransaction{
			{Transaction: largeTx, Receipt: nil}, // skipped due to size
			{Transaction: tx0, Receipt: &types.Receipt{}},
			{Transaction: tx1, Receipt: &types.Receipt{}},
		}})

	upgrades := opera.Upgrades{Brio: true}
	processor := &ExecutingBlock{
		evmProcessor: evmProcessor,
		blockBuilder: blockBuilder,
		params:       BlockParams{UserGasLimit: 10000, Rules: opera.Rules{Upgrades: upgrades}},
	}
	processor.Run(orderedTxs)

	// Huge transactions should not be added
	gotTxs := blockBuilder.GetTransactions()
	require.Equal(t, types.Transactions{tx0, tx1}, gotTxs)
}

func TestExecutingBlock_Run_InternalTransactionsHaveNoImpactOnTheUserTransactionGas(t *testing.T) {
	ctrl := gomock.NewController(t)
	statedb := state.NewMockStateDB(ctrl)

	statedb.EXPECT().BeginBlock(gomock.Any())
	statedb.EXPECT().SetTxContext(gomock.Any(), gomock.Any()).AnyTimes()
	statedb.EXPECT().GetBalance(gomock.Any()).Return(uint256.NewInt(0)).AnyTimes()
	statedb.EXPECT().SubBalance(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	statedb.EXPECT().Prepare(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	statedb.EXPECT().GetNonce(gomock.Any()).Return(uint64(0)).AnyTimes()
	statedb.EXPECT().SetNonce(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	statedb.EXPECT().GetCode(gomock.Any()).AnyTimes()
	statedb.EXPECT().Snapshot().AnyTimes()
	statedb.EXPECT().Exist(gomock.Any()).Return(true).AnyTimes()
	statedb.EXPECT().AddBalance(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	statedb.EXPECT().GetRefund().AnyTimes()
	statedb.EXPECT().GetLogs(gomock.Any(), gomock.Any()).AnyTimes()
	statedb.EXPECT().EndTransaction().AnyTimes()
	statedb.EXPECT().TxIndex().AnyTimes()

	evmModule := evmmodule.New()
	evmProcessor := evmModule.Start(
		0,
		0,
		0,
		statedb,
		&EvmStateReader{},
		func(l *core_types.Log) {},
		opera.Rules{},
		&params.ChainConfig{},
		common.Hash{},
		nil,
	)
	blockBuilder := inter.NewBlockBuilder()

	internalTx := types.NewTx(&types.LegacyTx{To: &common.Address{0x41}, Gas: 21_000})
	internalTxGasLimit := uint64(30_000)
	internalProcessedTxs := evmProcessor.Execute([]*types.Transaction{internalTx}, internalTxGasLimit, math.MaxUint64).ProcessedTransactions
	require.Len(t, internalProcessedTxs, 1)

	userTx0 := types.NewTx(&types.LegacyTx{To: &common.Address{0x42}, Gas: 21_000})
	skippedTx := types.NewTx(&types.LegacyTx{To: &common.Address{0x44}, Gas: 21_000})

	// the user transaction only fits into the user transaction gas limit if
	// the internal transaction gas is not counted towards it
	userTransactionGasLimit := uint64(30_000)
	upgrades := opera.Upgrades{Brio: true}
	processor := &ExecutingBlock{
		evmProcessor: evmProcessor,
		blockBuilder: blockBuilder,
		params:       BlockParams{UserGasLimit: userTransactionGasLimit, Rules: opera.Rules{Upgrades: upgrades}},
	}
	processor.Run([]*types.Transaction{userTx0, skippedTx})

	gotTxs := blockBuilder.GetTransactions()
	require.Equal(t, types.Transactions{userTx0}, gotTxs)
}

func TestExecutingBlock_Run_MetricsAreForwardedToStateProcessor(t *testing.T) {
	ctrl := gomock.NewController(t)
	statedb := state.NewMockStateDB(ctrl)
	mockMetrics := evmcore.NewMockBlockExecutionMetrics(ctrl)

	statedb.EXPECT().BeginBlock(gomock.Any())
	statedb.EXPECT().SetTxContext(gomock.Any(), gomock.Any()).AnyTimes()
	statedb.EXPECT().GetBalance(gomock.Any()).Return(uint256.NewInt(0)).AnyTimes()
	statedb.EXPECT().SubBalance(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	statedb.EXPECT().Prepare(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	statedb.EXPECT().GetNonce(gomock.Any()).Return(uint64(0)).AnyTimes()
	statedb.EXPECT().SetNonce(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	statedb.EXPECT().GetCode(gomock.Any()).AnyTimes()
	statedb.EXPECT().Snapshot().AnyTimes()
	statedb.EXPECT().RevertToSnapshot(gomock.Any()).AnyTimes()
	statedb.EXPECT().Exist(gomock.Any()).Return(true).AnyTimes()
	statedb.EXPECT().AddBalance(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	statedb.EXPECT().GetRefund().AnyTimes()
	statedb.EXPECT().GetLogs(gomock.Any(), gomock.Any()).AnyTimes()
	statedb.EXPECT().EndTransaction().AnyTimes()
	statedb.EXPECT().TxIndex().AnyTimes()

	evmModule := evmmodule.New()
	evmProcessor := evmModule.Start(
		0,
		0,
		0,
		statedb,
		&EvmStateReader{},
		func(l *core_types.Log) {},
		opera.Rules{Upgrades: opera.Upgrades{Brio: true, GasSubsidies: true}},
		&params.ChainConfig{},
		common.Hash{},
		mockMetrics,
	)
	blockBuilder := inter.NewBlockBuilder()

	// A sponsored transaction (gas price 0) will be skipped because there is
	// no subsidies registry contract deployed. This should trigger the
	// SkippedSponsoredTxs counter via the metrics forwarded to the state processor.
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	signer := types.LatestSignerForChainID(nil)
	sponsoredTx := types.MustSignNewTx(key, signer, &types.LegacyTx{To: &common.Address{0x42}, Gas: 21_000})
	require.True(t, subsidies.IsSponsorshipRequest(sponsoredTx))

	mockMetrics.EXPECT().IncSkippedSponsoredTx()

	upgrades := opera.Upgrades{Brio: true, GasSubsidies: true}
	processor := &ExecutingBlock{
		evmProcessor: evmProcessor,
		blockBuilder: blockBuilder,
		params:       BlockParams{UserGasLimit: 30_000, Rules: opera.Rules{Upgrades: upgrades}},
	}
	processor.Run([]*types.Transaction{sponsoredTx})

	// The sponsored tx is skipped (no receipt) so it should not appear in the block.
	require.Empty(t, blockBuilder.GetTransactions())
}

func TestExecutingBlock_Run_SponsoredTxSizeIsAccountedCorrectly(t *testing.T) {
	tests := map[string]struct {
		gasPrice      int64
		isSponsoredTx bool
	}{
		"normal tx": {
			gasPrice:      10,
			isSponsoredTx: false,
		},
		"sponsored tx": {
			gasPrice:      0,
			isSponsoredTx: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			evmProcessor := blockproc.NewMockEVMProcessor(ctrl)
			blockBuilder := inter.NewBlockBuilder()

			tx0 := types.NewTx(&types.LegacyTx{
				Data: make([]byte, params.MaxBlockSize-5000),
			})
			tx1 := types.NewTx(&types.LegacyTx{
				To:       &common.Address{0x42},
				GasPrice: big.NewInt(test.gasPrice),
				V:        big.NewInt(1),
			})
			remainingSize := params.MaxBlockSize - tx0.Size() - tx1.Size() - rlpEncodedMaxHeaderSizeInBytes
			tx2 := types.NewTx(&types.LegacyTx{
				Data: make([]byte, remainingSize-100), // leave some room for other fields in tx
			})

			var processedTxs []evmcore.ProcessedTransaction
			var feeChargingTransaction *types.Transaction
			if test.gasPrice == 0 {
				feeChargingTransaction = subsidies.NewPostTxBuilder().BuildForTesting()
				processedTxs = []evmcore.ProcessedTransaction{
					{Transaction: tx0, Receipt: &types.Receipt{}},
					{Transaction: tx1, Receipt: &types.Receipt{}},
					{Transaction: feeChargingTransaction, Receipt: &types.Receipt{}},
					{Transaction: tx2, Receipt: nil}, // skipped due to size
				}
			} else {
				processedTxs = []evmcore.ProcessedTransaction{
					{Transaction: tx0, Receipt: &types.Receipt{}},
					{Transaction: tx1, Receipt: &types.Receipt{}},
					{Transaction: tx2, Receipt: &types.Receipt{}},
				}
			}

			orderedTxs := []*types.Transaction{tx0, tx1, tx2}
			evmProcessor.EXPECT().
				Execute(orderedTxs, uint64(10000), gomock.Any()).
				Return(evmcore.ProcessSummary{
					ProcessedTransactions: processedTxs,
				})

			upgrades := opera.Upgrades{Brio: true}
			processor := &ExecutingBlock{
				evmProcessor: evmProcessor,
				blockBuilder: blockBuilder,
				params:       BlockParams{UserGasLimit: 10000, Rules: opera.Rules{Upgrades: upgrades}},
			}
			processor.Run(orderedTxs)

			gotTxs := blockBuilder.GetTransactions()
			require.Contains(t, gotTxs, tx0)
			require.Contains(t, gotTxs, tx1)

			if test.isSponsoredTx {
				// ensure the sponsored transaction is followed by the gas paying transaction
				require.Equal(t, tx1, gotTxs[1])
				require.Equal(t, feeChargingTransaction, gotTxs[2])

				require.NotContains(t, gotTxs, tx2)
			} else {
				require.Contains(t, gotTxs, tx2)
			}
		})
	}
}

func TestExecutingBlock_Run_SkipUserTransactionIfInternalTransactionsExceedBlockSizeLimit(t *testing.T) {
	tests := map[string][]*types.Transaction{
		"single huge internal tx": {
			types.NewTx(&types.LegacyTx{Data: make([]byte, params.MaxBlockSize)}),
		},
		"multiple internal txs exceeding block size": {
			types.NewTx(&types.LegacyTx{Data: make([]byte, params.MaxBlockSize/4)}),
			types.NewTx(&types.LegacyTx{Data: make([]byte, params.MaxBlockSize/4)}),
			types.NewTx(&types.LegacyTx{Data: make([]byte, params.MaxBlockSize/4)}),
			types.NewTx(&types.LegacyTx{Data: make([]byte, params.MaxBlockSize/4)}),
		},
	}

	for name, internalTxs := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			evmProcessor := blockproc.NewMockEVMProcessor(ctrl)
			blockBuilder := inter.NewBlockBuilder()

			for _, internalTx := range internalTxs {
				// Add internal txs to blockBuilder
				blockBuilder.AddTransaction(internalTx, &types.Receipt{})
			}

			// Create a user tx that would only fit without the internal tx
			userTx := types.NewTx(&types.LegacyTx{})
			upgrades := opera.Upgrades{Brio: true}

			evmProcessor.EXPECT().Execute(gomock.Any(), gomock.Any(), gomock.Any()).Return(evmcore.ProcessSummary{})
			processor := &ExecutingBlock{
				evmProcessor: evmProcessor,
				blockBuilder: blockBuilder,
				params:       BlockParams{UserGasLimit: 10000, Rules: opera.Rules{Upgrades: upgrades}},
			}
			processor.Run([]*types.Transaction{userTx})

			// Only internal tx should be present
			gotTxs := blockBuilder.GetTransactions()
			require.Equal(t, types.Transactions(internalTxs), gotTxs)
		})
	}
}

func TestExecutingBlock_Run_InternalTxsExceedingBlockSizePassesZeroRemainingSize(t *testing.T) {
	ctrl := gomock.NewController(t)
	evmProcessor := blockproc.NewMockEVMProcessor(ctrl)
	blockBuilder := inter.NewBlockBuilder()

	// Add two internal transactions: one that nearly fills the block, and one that exceeds the remaining size.
	smallTx := types.NewTx(&types.LegacyTx{Data: make([]byte, params.MaxBlockSize-rlpEncodedMaxHeaderSizeInBytes-100)})
	overflowTx := types.NewTx(&types.LegacyTx{Data: make([]byte, 200)})
	blockBuilder.AddTransaction(smallTx, &types.Receipt{})
	blockBuilder.AddTransaction(overflowTx, &types.Receipt{})

	userTx := types.NewTx(&types.LegacyTx{})
	orderedTxs := []*types.Transaction{userTx}
	upgrades := opera.Upgrades{Brio: true}

	// After the overflow is detected, remainingSize is set to 0
	evmProcessor.EXPECT().
		Execute(orderedTxs, uint64(10000), gomock.Any()).
		DoAndReturn(func(txs []*types.Transaction, gas uint64, size uint64) evmcore.ProcessSummary {
			require.Equal(t, uint64(0), size) // Ensure the user transactions are processed with zero remaining size
			return evmcore.ProcessSummary{}
		})

	processor := &ExecutingBlock{
		evmProcessor: evmProcessor,
		blockBuilder: blockBuilder,
		params:       BlockParams{UserGasLimit: 10000, Rules: opera.Rules{Upgrades: upgrades}},
	}
	processor.Run(orderedTxs)
}

func TestExecutingBlock_Run_RecordsOriginTransactionForAcceptedProcessedTransactions(t *testing.T) {
	ctrl := gomock.NewController(t)
	evmProcessor := blockproc.NewMockEVMProcessor(ctrl)

	transactions := []*types.Transaction{
		types.NewTx(&types.LegacyTx{Nonce: 0}),
		types.NewTx(&types.LegacyTx{Nonce: 1}),
		types.NewTx(&types.LegacyTx{Nonce: 2}),
	}

	extras := []*types.Transaction{
		types.NewTx(&types.LegacyTx{Nonce: 11}),
		types.NewTx(&types.LegacyTx{Nonce: 21}),
		types.NewTx(&types.LegacyTx{Nonce: 22}),
		types.NewTx(&types.LegacyTx{Nonce: 23}),
	}

	wantCausedBy := map[common.Hash]common.Hash{
		transactions[0].Hash(): transactions[0].Hash(),
		transactions[1].Hash(): transactions[1].Hash(),
		extras[0].Hash():       transactions[1].Hash(),
		extras[1].Hash():       transactions[2].Hash(),
		extras[2].Hash():       transactions[2].Hash(),
		// no entry for extras[3] since it has no receipt
	}

	evmProcessor.EXPECT().
		Execute(transactions, gomock.Any(), gomock.Any()).
		Return(evmcore.ProcessSummary{
			ProcessedTransactions: []evmcore.ProcessedTransaction{
				{Transaction: transactions[0], Receipt: &types.Receipt{}},
				{Transaction: transactions[1], Receipt: &types.Receipt{}},
				{Transaction: extras[0], Receipt: &types.Receipt{}},
				{Transaction: extras[1], Receipt: &types.Receipt{}},
				{Transaction: extras[2], Receipt: &types.Receipt{}},
				{Transaction: extras[3], Receipt: nil},
			},
			CausedBy: wantCausedBy,
		})

	processor := &ExecutingBlock{
		evmProcessor: evmProcessor,
		blockBuilder: inter.NewBlockBuilder(),
		params:       BlockParams{UserGasLimit: math.MaxUint64, Rules: opera.Rules{Upgrades: opera.Upgrades{Brio: true}}},
	}
	gotCausedBy := processor.Run(transactions).CausedBy
	require.Equal(t, wantCausedBy, gotCausedBy)
}

func TestExecutingBlock_Run_BlockSizeLimitIsEnforcedStartingFromBrio(t *testing.T) {
	tests := map[string]struct {
		upgrades          opera.Upgrades
		expectedSizeLimit uint64
	}{
		"before Brio, no size limit is enforced": {
			upgrades:          opera.Upgrades{Brio: false},
			expectedSizeLimit: math.MaxUint64,
		},
		"with Brio, block size limit is enforced": {
			upgrades:          opera.Upgrades{Brio: true},
			expectedSizeLimit: uint64(params.MaxBlockSize - rlpEncodedMaxHeaderSizeInBytes),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			evmProcessor := blockproc.NewMockEVMProcessor(ctrl)
			blockBuilder := inter.NewBlockBuilder()

			// Ensure the evmProcessor is called with the expected size limit
			evmProcessor.EXPECT().Execute(gomock.Any(), gomock.Any(), test.expectedSizeLimit)

			processor := &ExecutingBlock{
				evmProcessor: evmProcessor,
				blockBuilder: blockBuilder,
				params:       BlockParams{UserGasLimit: math.MaxUint64, Rules: opera.Rules{Upgrades: test.upgrades}},
			}
			processor.Run([]*types.Transaction{})
		})
	}
}

func TestExecutingBlock_runInternal_ExecutesWithinBlockGasLimitAndUnlimitedSize(t *testing.T) {
	accepted := types.NewTx(&types.LegacyTx{Nonce: 1})
	skipped := types.NewTx(&types.LegacyTx{Nonce: 2})
	txs := types.Transactions{accepted, skipped}
	summary := evmcore.ProcessSummary{ProcessedTransactions: []evmcore.ProcessedTransaction{
		{Transaction: accepted, Receipt: &types.Receipt{}},
		{Transaction: skipped},
	}}

	evmProcessor := blockproc.NewMockEVMProcessor(gomock.NewController(t))
	evmProcessor.EXPECT().Execute(txs, uint64(42), uint64(math.MaxUint64)).Return(summary)

	processor := &ExecutingBlock{
		evmProcessor: evmProcessor,
		blockBuilder: inter.NewBlockBuilder(),
		params: BlockParams{
			GasLimit:     42,
			UserGasLimit: 21,
			// Brio would limit the size of user batches; internal batches stay unlimited.
			Rules: opera.Rules{Upgrades: opera.Upgrades{Brio: true}},
		},
	}

	require.Equal(t, summary, processor.runInternal(txs))
	require.Equal(t, types.Transactions{accepted}, processor.blockBuilder.GetTransactions())
}

// TestExecutingBlock_Finalize_AppliesSonicMainnetGasLimitException verifies that
// Finalize stamps the hard-coded gas limit for Sonic mainnet block 8054923
// regardless of the rules' MaxBlockGas, and leaves the announced gas limit
// untouched for any other block.
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
			require := require.New(t)
			ctrl := gomock.NewController(t)

			evmProcessor := blockproc.NewMockEVMProcessor(ctrl)
			evmProcessor.EXPECT().Finalize().Return(&evmcore.EvmBlock{
				EvmHeader: evmcore.EvmHeader{BaseFee: big.NewInt(0)},
			}, 0, types.Receipts{})

			processor := &ExecutingBlock{
				evmProcessor: evmProcessor,
				blockBuilder: inter.NewBlockBuilder().
					WithNumber(test.number).
					WithGasLimit(announcedGasLimit),
				params: BlockParams{
					Number: test.number,
					// A MaxBlockGas distinct from the hard-coded constant proves
					// the stamped value comes from the constant, not the rules.
					Rules: opera.Rules{
						NetworkID: test.networkID,
						Blocks:    opera.BlocksRules{MaxBlockGas: sonicMainnetBlock8054923GasLimit + 1},
					},
				},
			}

			candidate := processor.Finalize().Candidate()
			require.Equal(test.want, candidate.Block.GasLimit)
		})
	}
}

func TestExecutingBlock_Finalize_AssemblesBlockFromExecutionResults(t *testing.T) {
	require := require.New(t)
	receipts := types.Receipts{{Logs: []*types.Log{{}, {}}}}

	evmProcessor := blockproc.NewMockEVMProcessor(gomock.NewController(t))
	evmProcessor.EXPECT().Finalize().Return(&evmcore.EvmBlock{EvmHeader: evmcore.EvmHeader{
		Root:    common.Hash{1},
		GasUsed: 2,
		BaseFee: big.NewInt(3),
	}}, 4, receipts)

	processor := &ExecutingBlock{
		evmProcessor: evmProcessor,
		blockBuilder: inter.NewBlockBuilder().WithTime(inter.Timestamp(5 * time.Second)).WithDuration(6),
	}

	candidate := processor.Finalize().Candidate()
	block := candidate.Block
	require.Equal(common.Hash{1}, block.StateRoot)
	require.Equal(uint64(2), block.GasUsed)
	require.Equal(big.NewInt(3), block.BaseFee)
	require.Equal(block.Hash(), candidate.EvmBlock.Hash)
	require.Equal(time.Duration(6), candidate.EvmBlock.Duration)
	require.Equal(4, candidate.NumSkipped)
	require.Equal(receipts, candidate.Receipts)
	for _, receipt := range candidate.Receipts {
		require.Equal(block.Hash(), receipt.BlockHash)
		for _, log := range receipt.Logs {
			require.Equal(block.Hash(), log.BlockHash)
			require.Equal(uint64(5), log.BlockTimestamp)
		}
	}
}

func TestExecutingBlock_AfterFinalize_Panics(t *testing.T) {
	tests := map[string]func(*ExecutingBlock){
		"stateDB":     func(eb *ExecutingBlock) { eb.stateDB() },
		"Run":         func(eb *ExecutingBlock) { eb.Run(nil) },
		"runInternal": func(eb *ExecutingBlock) { eb.runInternal(nil) },
		"Finalize":    func(eb *ExecutingBlock) { eb.Finalize() },
	}

	for name, call := range tests {
		t.Run(name, func(t *testing.T) {
			evmProcessor := blockproc.NewMockEVMProcessor(gomock.NewController(t))
			evmProcessor.EXPECT().Finalize().Return(&evmcore.EvmBlock{EvmHeader: evmcore.EvmHeader{BaseFee: big.NewInt(0)}}, 0, nil)
			executing := &ExecutingBlock{evmProcessor: evmProcessor, blockBuilder: inter.NewBlockBuilder()}
			executing.Finalize()

			require.Panics(t, func() { call(executing) })
		})
	}
}
