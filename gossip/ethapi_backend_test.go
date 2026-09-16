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
	"fmt"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/emitter"
	"github.com/0xsoniclabs/sonic/gossip/evmstore"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/iblockproc"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestEthApiBackend_GetNetworkRules_LoadsRulesFromEpoch(t *testing.T) {
	require := require.New(t)

	blockNumber := idx.Block(12)
	epoch := idx.Epoch(3)

	store, err := NewMemStore(t)
	require.NoError(err)

	store.SetBlock(
		blockNumber,
		inter.NewBlockBuilder().
			WithNumber(uint64(blockNumber)).
			WithEpoch(epoch).
			Build(),
	)
	require.True(store.HasBlock(blockNumber))

	rules := opera.FakeNetRules(opera.Upgrades{})
	rules.Name = "test-rules"

	store.SetHistoryBlockEpochState(
		epoch,
		iblockproc.BlockState{},
		iblockproc.EpochState{
			Epoch: epoch,
			Rules: rules,
		},
	)
	setLatestBlock(store, blockNumber)

	backend := &EthAPIBackend{
		svc: &Service{
			store: store,
		},
		state: &EvmStateReader{
			store: store,
		},
	}

	got, err := backend.GetNetworkRules(t.Context(), blockNumber)
	require.NoError(err)

	// Rules contain functions that cannot be compared directly,
	// so we compare their string representations.
	want := fmt.Sprintf("%+v", rules)
	have := fmt.Sprintf("%+v", got)
	require.Equal(want, have, "Network rules do not match")
}

func TestEthApiBackend_GetNetworkRules_MissingBlockReturnsNilRules(t *testing.T) {
	require := require.New(t)

	blockNumber := idx.Block(12)

	store, err := NewMemStore(t)
	require.NoError(err)
	require.False(store.HasBlock(blockNumber))
	setLatestBlock(store, blockNumber)

	backend := &EthAPIBackend{
		svc: &Service{store: store},
		state: &EvmStateReader{
			store: store,
		},
	}

	rules, err := backend.GetNetworkRules(t.Context(), blockNumber)
	require.NoError(err)
	require.Nil(rules)
}

// setLatestBlock sets the latest block index of the given store.
func setLatestBlock(store *Store, number idx.Block) {
	store.SetBlockEpochState(
		iblockproc.BlockState{LastBlock: iblockproc.BlockCtx{Idx: number}},
		iblockproc.EpochState{},
	)
}

func TestEthApiBackend_GetTransaction_ReturnsTransactionAtItsPosition(t *testing.T) {
	require := require.New(t)

	store, err := NewMemStore(t)
	require.NoError(err)

	tx := types.NewTx(&types.LegacyTx{Nonce: 1})
	store.evm.SetTx(tx.Hash(), tx)
	store.evm.SetTxPosition(tx.Hash(), evmstore.TxPosition{Block: 42, BlockOffset: 7})
	setLatestBlock(store, 42)

	backend := &EthAPIBackend{
		svc: &Service{
			config: Config{TxIndex: true},
			store:  store,
		},
	}

	got, block, offset, err := backend.GetTransaction(t.Context(), tx.Hash())
	require.NoError(err)
	require.Equal(tx.Hash(), got.Hash())
	require.Equal(uint64(42), block)
	require.Equal(uint64(7), offset)
}

func TestEthApiBackend_GetTransaction_ReportsCorruptedIndexIfBodyIsMissing(t *testing.T) {
	require := require.New(t)

	store, err := NewMemStore(t)
	require.NoError(err)

	txHash := common.Hash{1}
	store.evm.SetTxPosition(txHash, evmstore.TxPosition{Block: 42, BlockOffset: 7})
	setLatestBlock(store, 42)

	backend := &EthAPIBackend{
		svc: &Service{
			config: Config{TxIndex: true},
			store:  store,
		},
	}

	_, _, _, err = backend.GetTransaction(t.Context(), txHash)
	require.ErrorContains(err, "index is corrupted")
}

func TestEthApiBackend_GetTransaction_IsNotFoundUntilItsBlockIsCoveredByLatest(t *testing.T) {
	const txBlock = 42
	testCases := map[string]struct {
		latest idx.Block
		found  bool
	}{
		"latest block behind the transaction block": {latest: txBlock - 1, found: false},
		"latest block is the transaction block":     {latest: txBlock, found: true},
		"latest block after the transaction block":  {latest: txBlock + 1, found: true},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)

			store, err := NewMemStore(t)
			require.NoError(err)

			tx := types.NewTx(&types.LegacyTx{Nonce: 1})
			store.evm.SetTx(tx.Hash(), tx)
			store.evm.SetTxPosition(tx.Hash(), evmstore.TxPosition{Block: txBlock, BlockOffset: 0})
			setLatestBlock(store, test.latest)

			backend := &EthAPIBackend{
				svc: &Service{
					config: Config{TxIndex: true},
					store:  store,
				},
			}

			got, block, _, err := backend.GetTransaction(t.Context(), tx.Hash())
			require.NoError(err)
			if test.found {
				require.NotNil(got)
				require.Equal(uint64(txBlock), block)
			} else {
				require.Nil(got, "transaction must not be reported before its block is the latest block")
			}
		})
	}
}

func TestEthApiBackend_BlockByNumber_DoesNotServeBlocksBeyondLatest(t *testing.T) {
	const stored = 42
	testCases := map[string]struct {
		latest    idx.Block
		requested rpc.BlockNumber
		found     bool
	}{
		"requested block behind latest":  {latest: stored + 1, requested: stored, found: true},
		"requested block is latest":      {latest: stored, requested: stored, found: true},
		"requested block beyond latest":  {latest: stored - 1, requested: stored, found: false},
		"latest tag resolves to latest":  {latest: stored, requested: rpc.LatestBlockNumber, found: true},
		"pending tag resolves to latest": {latest: stored, requested: rpc.PendingBlockNumber, found: true},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)

			store, err := NewMemStore(t)
			require.NoError(err)

			store.SetBlock(stored, inter.NewBlockBuilder().WithNumber(stored).Build())
			setLatestBlock(store, test.latest)

			backend := &EthAPIBackend{
				svc:   &Service{store: store},
				state: &EvmStateReader{store: store},
			}

			block, err := backend.BlockByNumber(t.Context(), test.requested)
			require.NoError(err)
			if test.found {
				require.NotNil(block)
				require.Equal(uint64(stored), block.NumberU64())
			} else {
				require.Nil(block, "block must not be served before it is covered by the latest block")
			}
		})
	}
}

func TestEthApiBackend_StateAndBlockByNumberOrHash_DoesNotServeBlocksBeyondLatest(t *testing.T) {
	const stored = 42
	block := inter.NewBlockBuilder().WithNumber(stored).Build()

	testCases := map[string]struct {
		selector rpc.BlockNumberOrHash
	}{
		"by number": {selector: rpc.BlockNumberOrHashWithNumber(stored)},
		"by hash":   {selector: rpc.BlockNumberOrHashWithHash(common.Hash(block.Hash()), false)},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)

			store, err := NewMemStore(t)
			require.NoError(err)

			store.SetBlock(stored, block)
			store.SetBlockIndex(block.Hash(), stored)
			setLatestBlock(store, stored-1)

			backend := &EthAPIBackend{
				svc:   &Service{store: store},
				state: &EvmStateReader{store: store},
			}

			_, _, err = backend.StateAndBlockByNumberOrHash(t.Context(), test.selector)
			require.ErrorContains(err, "header not found")
		})
	}
}

func TestEthApiBackend_IsTestOnlyApiEnabled_ReturnsConfigFlagValue(t *testing.T) {
	for _, enabled := range []bool{true, false} {
		t.Run(fmt.Sprintf("enabled=%v", enabled), func(t *testing.T) {
			require := require.New(t)

			backend := &EthAPIBackend{
				svc: &Service{
					config: Config{
						EnableTestOnlyApi: enabled,
					},
				},
			}

			result := backend.IsTestOnlyApiEnabled()
			require.Equal(enabled, result)
		})
	}
}

func TestEthApiBackend_ProposeTransactions_ReturnsErrorWhenTestOnlyApiDisabled(t *testing.T) {
	require := require.New(t)

	backend := &EthAPIBackend{
		svc: &Service{
			config: Config{
				EnableTestOnlyApi: false,
			},
		},
	}

	err := backend.ProposeTransactions(nil)
	require.Error(err)
	require.Contains(err.Error(), "disabled")
}

func TestEthApiBackend_ProposeTransactions_ReturnsErrorWhenNoEmitters(t *testing.T) {
	require := require.New(t)

	backend := &EthAPIBackend{
		svc: &Service{
			config:   Config{EnableTestOnlyApi: true},
			emitters: []*emitter.Emitter{}, // No emitters available
		},
	}

	err := backend.ProposeTransactions(nil)
	require.Error(err)
	require.Contains(err.Error(), "no emitters")
}

func TestEthApiBackend_proposeTransactionsInternal_ForwardsRequestToEmitter(t *testing.T) {
	ctrl := gomock.NewController(t)
	emitter := NewMockforceableEmitter(ctrl)

	backend := &EthAPIBackend{
		svc: &Service{
			config: Config{EnableTestOnlyApi: true},
		},
	}

	txs := []*types.Transaction{
		types.NewTx(&types.LegacyTx{Nonce: 1}),
		types.NewTx(&types.LegacyTx{Nonce: 2}),
	}

	emitter.EXPECT().ForceEventEmissionForTesting(txs).Return(nil)
	require.NoError(t, backend.proposeTransactionsInternal(txs, emitter))
}

func TestEthApiBackend_proposeTransactionsInternal_ReturnsEmissionIssue(t *testing.T) {
	ctrl := gomock.NewController(t)
	emitter := NewMockforceableEmitter(ctrl)

	backend := &EthAPIBackend{
		svc: &Service{
			config: Config{EnableTestOnlyApi: true},
		},
	}

	issue := fmt.Errorf("injected test issue")

	emitter.EXPECT().ForceEventEmissionForTesting(nil).Return(issue)
	require.ErrorIs(t, backend.proposeTransactionsInternal(nil, emitter), issue)
}

func TestEthApiBackend_AddTransactions_ReturnsErrorWhenTestOnlyApiDisabled(t *testing.T) {
	backend := &EthAPIBackend{
		svc: &Service{
			config: Config{EnableTestOnlyApi: false},
		},
	}

	err := backend.AddTransactions(nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "disabled")
}

func TestEthApiBackend_AddTransactions_ReturnsErrorWhenTxPoolMissing(t *testing.T) {
	backend := &EthAPIBackend{
		svc: &Service{
			config: Config{EnableTestOnlyApi: true},
		},
	}

	err := backend.AddTransactions(nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no transaction pool")
}

func TestEthApiBackend_AddTransactions_ForwardsTransactionsToPool(t *testing.T) {
	ctrl := gomock.NewController(t)
	pool := NewMockTxPool(ctrl)

	backend := &EthAPIBackend{
		svc: &Service{
			config: Config{EnableTestOnlyApi: true},
			txpool: pool,
		},
	}

	txs := []*types.Transaction{
		types.NewTx(&types.LegacyTx{Nonce: 1}),
		types.NewTx(&types.LegacyTx{Nonce: 2}),
	}

	pool.EXPECT().AddLocals(types.Transactions(txs)).Return([]error{nil, nil})
	require.NoError(t, backend.AddTransactions(txs))
}

func TestEthApiBackend_AddTransactions_ReturnsPoolRejections(t *testing.T) {
	ctrl := gomock.NewController(t)
	pool := NewMockTxPool(ctrl)

	backend := &EthAPIBackend{
		svc: &Service{
			config: Config{EnableTestOnlyApi: true},
			txpool: pool,
		},
	}

	issue := fmt.Errorf("injected test issue")

	pool.EXPECT().AddLocals(gomock.Any()).Return([]error{nil, issue})
	require.ErrorIs(t, backend.AddTransactions(nil), issue)
}

func TestEthApiBackend_epochWithDefault_RejectsEpochsAboveUint32Max(t *testing.T) {
	const currentEpoch = idx.Epoch(1000)

	store, err := NewMemStore(t)
	require.NoError(t, err)
	store.SetBlockEpochState(iblockproc.BlockState{}, iblockproc.EpochState{Epoch: currentEpoch})
	backend := &EthAPIBackend{
		svc: &Service{
			store: store,
		},
	}

	// 2^32 + currentEpoch truncates to currentEpoch when cast to uint32,
	// so a naive idx.Epoch(epoch) <= current check would incorrectly accept it.
	outOfRange := rpc.BlockNumber(1<<32 + int64(currentEpoch))
	_, err = backend.epochWithDefault(t.Context(), outOfRange)
	require.Error(t, err, "epoch value above uint32 max must be rejected")
}
