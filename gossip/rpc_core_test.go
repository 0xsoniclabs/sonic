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

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/gossip/evmstore"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/iblockproc"
	"github.com/0xsoniclabs/sonic/opera"
	rpccore "github.com/0xsoniclabs/sonic/rpc/core"
)

func newTestRPCCore(t *testing.T, cfg Config) (*rpcCore, *Store) {
	t.Helper()
	store, err := NewMemStore(t)
	require.NoError(t, err)
	svc := &Service{config: cfg, store: store}
	svc.EthAPI = &EthAPIBackend{svc: svc}
	return &rpcCore{svc: svc, store: store, state: &EvmStateReader{store: store}}, store
}

func TestRPCCore_RulesAreThoseOfTheBlocksEpoch(t *testing.T) {
	c, store := newTestRPCCore(t, Config{})
	const blockNumber, epoch = idx.Block(12), idx.Epoch(3)
	store.SetBlock(blockNumber, inter.NewBlockBuilder().WithNumber(uint64(blockNumber)).WithEpoch(epoch).Build())
	setLatestBlockIndex(store, blockNumber)
	rules := opera.FakeNetRules(opera.Upgrades{})
	rules.Name = "epoch-rules"
	store.SetHistoryBlockEpochState(epoch, iblockproc.BlockState{}, iblockproc.EpochState{Epoch: epoch, Rules: rules})

	got, err := c.Rules(uint64(blockNumber))
	require.NoError(t, err)
	// Rules hold functions; their printed forms are compared.
	require.Equal(t, fmt.Sprintf("%+v", rules), fmt.Sprintf("%+v", *got))

	_, err = c.Rules(uint64(blockNumber) - 1)
	require.ErrorIs(t, err, rpccore.ErrNotFound, "a block the node does not have")
	_, err = c.Rules(uint64(blockNumber) + 2)
	require.ErrorIs(t, err, rpccore.ErrNotFound, "above the open epoch")
	open, err := c.Rules(uint64(blockNumber) + 1)
	require.NoError(t, err)
	require.Equal(t, store.GetRules().Name, open.Name, "the block after the head has the open epoch's rules")
}

func TestRPCCore_TxByHashAnswersThePosition(t *testing.T) {
	c, store := newTestRPCCore(t, Config{TxIndex: true})
	tx := types.NewTx(&types.LegacyTx{Nonce: 1})
	store.evm.SetTx(tx.Hash(), tx)
	store.evm.SetTxPosition(tx.Hash(), evmstore.TxPosition{Block: 42, BlockOffset: 7})
	setLatestBlockIndex(store, 42)

	got, block, offset, err := c.TxByHash(tx.Hash())
	require.NoError(t, err)
	require.Equal(t, tx.Hash(), got.Hash())
	require.Equal(t, uint64(42), block)
	require.Equal(t, uint64(7), offset)
	_, _, _, err = c.TxByHash(common.Hash{9})
	require.ErrorIs(t, err, rpccore.ErrNotFound)
}

// TestRPCCore_TxByHashIsNotReportedBeforeItsBlockIsPublished: the transaction
// index is written before the latest block index advances; a transaction is
// unknown until then.
func TestRPCCore_TxByHashIsNotReportedBeforeItsBlockIsPublished(t *testing.T) {
	c, store := newTestRPCCore(t, Config{TxIndex: true})
	const blockNumber = idx.Block(42)
	tx := types.NewTx(&types.LegacyTx{Nonce: 1})
	store.evm.SetTx(tx.Hash(), tx)
	store.evm.SetTxPosition(tx.Hash(), evmstore.TxPosition{Block: blockNumber, BlockOffset: 7})

	setLatestBlockIndex(store, blockNumber-1)
	_, _, _, err := c.TxByHash(tx.Hash())
	require.ErrorIs(t, err, rpccore.ErrNotFound)
	setLatestBlockIndex(store, blockNumber)
	_, block, _, err := c.TxByHash(tx.Hash())
	require.NoError(t, err)
	require.Equal(t, uint64(blockNumber), block)
}

func TestRPCCore_TxByHashReportsACorruptedIndex(t *testing.T) {
	c, store := newTestRPCCore(t, Config{TxIndex: true})
	store.evm.SetTxPosition(common.Hash{1}, evmstore.TxPosition{Block: 42, BlockOffset: 7})
	setLatestBlockIndex(store, 42)
	_, _, _, err := c.TxByHash(common.Hash{1})
	require.ErrorContains(t, err, "index is corrupted")
}

// TestRPCCore_TxIndexOffIsAnError is the guard of the TxIndex configuration
// (section 2 of rpc/PLAN.md): the reads that need the index answer its error
// in the legacy wording, not an empty result.
func TestRPCCore_TxIndexOffIsAnError(t *testing.T) {
	c, store := newTestRPCCore(t, Config{TxIndex: false})
	tx := types.NewTx(&types.LegacyTx{Nonce: 1})
	store.evm.SetTx(tx.Hash(), tx)
	store.evm.SetTxPosition(tx.Hash(), evmstore.TxPosition{Block: 1})
	setLatestBlockIndex(store, 1)
	_, _, _, err := c.TxByHash(tx.Hash())
	require.EqualError(t, err, "transactions index is disabled (enable TxIndex and re-process the DAG)")
	_, err = c.Receipts(1)
	require.EqualError(t, err, "transactions index is disabled (enable TxIndex and re-process the DAGs)")
}

func TestRPCCore_UnknownBundlesAndHashesAreNotFound(t *testing.T) {
	c, _ := newTestRPCCore(t, Config{})
	_, err := c.BundleInfo(common.Hash{1})
	require.ErrorIs(t, err, rpccore.ErrNotFound)
	_, err = c.BlockNumberByHash(common.Hash{1})
	require.ErrorIs(t, err, rpccore.ErrNotFound)
}
