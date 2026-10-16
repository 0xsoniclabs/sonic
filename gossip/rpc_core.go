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
	"context"
	"errors"
	"math/big"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	notify "github.com/ethereum/go-ethereum/event"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/evmstore"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	rpccore "github.com/0xsoniclabs/sonic/rpc/core"
)

// RPCCore returns the adapter the RPC module uses to reach the node.
func (s *Service) RPCCore() rpccore.Core {
	return &rpcCore{svc: s, reader: s.EthAPI.state}
}

// rpcCore implements rpccore.Core over the store, the EVM state reader, the
// transaction pool and the service feed. It starts no goroutine: the module
// owns every loop that consumes it.
type rpcCore struct {
	svc    *Service
	reader StateReader
}

var _ rpccore.Core = (*rpcCore)(nil)

// endedSubscription is handed out when the feed it would subscribe to has
// stopped. Its error channel is closed, so the consumer sees the end at once.
type endedSubscription struct{}

func (endedSubscription) Err() <-chan error {
	ch := make(chan error)
	close(ch)
	return ch
}

func (endedSubscription) Unsubscribe() {}

var _ rpccore.Subscription = endedSubscription{}

// asSubscription turns the nil subscription of a closed feed scope into an
// ended one.
func asSubscription(sub notify.Subscription) rpccore.Subscription {
	if sub == nil {
		return endedSubscription{}
	}
	return sub
}

// -- ChainReader --

func (c *rpcCore) ChainInfo() rpccore.ChainInfo {
	id := c.svc.store.GetRules().NetworkID
	return rpccore.ChainInfo{
		ChainID:        id,
		NetworkID:      id,
		GenesisID:      common.Hash(c.svc.store.GetGenesisID()),
		UpgradeHeights: c.svc.store.GetUpgradeHeights(),
	}
}

func (c *rpcCore) Header(number uint64) (*evmcore.EvmHeader, error) {
	header := c.reader.Header(common.Hash{}, number)
	if header == nil {
		return nil, rpccore.ErrNotFound
	}
	return header, nil
}

func (c *rpcCore) Block(number uint64) (*evmcore.EvmBlock, error) {
	block := c.reader.Block(common.Hash{}, number)
	if block == nil {
		return nil, rpccore.ErrNotFound
	}
	return block, nil
}

func (c *rpcCore) BlockNumberByHash(h common.Hash) (uint64, bool) {
	index := c.svc.store.GetBlockIndex(hash.Event(h))
	if index == nil {
		return 0, false
	}
	return uint64(*index), true
}

func (c *rpcCore) Receipts(number uint64) (types.Receipts, error) {
	block := c.reader.Block(common.Hash{}, number)
	if block == nil {
		return nil, rpccore.ErrNotFound
	}
	return c.svc.EthAPI.FetchReceiptsForBlock(block), nil
}

func (c *rpcCore) Rules(number uint64) (*opera.Rules, error) {
	header := c.reader.Header(common.Hash{}, number)
	if header == nil {
		return nil, rpccore.ErrNotFound
	}
	_, es := c.svc.store.GetHistoryBlockEpochState(idx.Epoch(header.Epoch))
	if es == nil {
		return nil, rpccore.ErrNotFound
	}
	rules := es.Rules
	return &rules, nil
}

func (c *rpcCore) TxByHash(h common.Hash) (tx *types.Transaction, block, index uint64, ok bool) {
	position := c.svc.EthAPI.GetTxPosition(h)
	if position == nil {
		return nil, 0, 0, false
	}
	tx = c.svc.store.evm.GetTx(h)
	if tx == nil {
		return nil, 0, 0, false
	}
	return tx, uint64(position.Block), uint64(position.BlockOffset), true
}

func (c *rpcCore) BundleInfo(executionPlanHash common.Hash) (rpccore.BundleInfo, bool) {
	info := c.svc.store.GetBundleExecutionInfo(executionPlanHash)
	if info == nil {
		return rpccore.BundleInfo{}, false
	}
	return rpccore.BundleInfo{
		BlockNumber: info.BlockNumber,
		Offset:      info.Position.Offset,
		Count:       info.Position.Count,
	}, true
}

// -- HeadSource --

func (c *rpcCore) Head() (*evmcore.EvmBlock, error) {
	block, err := c.reader.LastBlockWithArchiveState(true)
	if err != nil {
		// A node without archive serves its newest block.
		if errors.Is(err, evmstore.NoArchiveError) {
			return c.reader.CurrentBlock(), nil
		}
		return nil, err
	}
	if block == nil {
		return nil, rpccore.ErrNotFound
	}
	return block, nil
}

func (c *rpcCore) SubscribeHead(ch chan<- rpccore.HeadEvent) rpccore.Subscription {
	return asSubscription(c.svc.feed.SubscribeHead(ch))
}

// -- StateProvider --

func (c *rpcCore) State(number uint64) (state.StateDB, error) {
	header := c.reader.Header(common.Hash{}, number)
	if header == nil {
		return nil, rpccore.ErrNotFound
	}
	return c.reader.BlockStateDB(header.Number, header.Root)
}

// -- Pool --

func (c *rpcCore) SendTx(_ context.Context, tx *types.Transaction) error {
	return c.svc.txpool.AddLocal(tx)
}

func (c *rpcCore) PoolNonce(addr common.Address) uint64 {
	return c.svc.txpool.Nonce(addr)
}

func (c *rpcCore) PoolTransaction(h common.Hash) *types.Transaction {
	return c.svc.txpool.Get(h)
}

func (c *rpcCore) PoolTransactions() (types.Transactions, error) {
	return c.svc.EthAPI.GetPoolTransactions()
}

func (c *rpcCore) PoolContent() (pending, queued map[common.Address]types.Transactions) {
	return c.svc.txpool.Content()
}

func (c *rpcCore) PoolContentFrom(addr common.Address) (pending, queued types.Transactions) {
	return c.svc.txpool.ContentFrom(addr)
}

func (c *rpcCore) PoolStats() (pending, queued int) {
	return c.svc.txpool.Stats()
}

func (c *rpcCore) SubscribeTxs(ch chan<- evmcore.NewTxsNotify) rpccore.Subscription {
	return asSubscription(c.svc.txpool.SubscribeNewTxsNotify(ch))
}

// -- LogIndex --

func (c *rpcCore) FindLogs(ctx context.Context, from, to uint64, pattern [][]common.Hash, limit uint) ([]*types.Log, error) {
	return c.svc.store.evm.EvmLogs.FindInBlocks(ctx, idx.Block(from), idx.Block(to), pattern, limit)
}

// -- NodeInfo --

func (c *rpcCore) Progress() rpccore.Progress {
	p := c.svc.EthAPI.Progress()
	return rpccore.Progress{
		CurrentEpoch:     uint64(p.CurrentEpoch),
		CurrentBlock:     uint64(p.CurrentBlock),
		CurrentBlockHash: common.Hash(p.CurrentBlockHash),
		CurrentBlockTime: uint64(p.CurrentBlockTime),
		HighestBlock:     uint64(p.HighestBlock),
		HighestEpoch:     uint64(p.HighestEpoch),
	}
}

func (c *rpcCore) OracleInputs() rpccore.OracleInputs {
	backend := &GPOBackend{c.svc.store, c.svc.txpool}
	return rpccore.OracleInputs{
		PendingRules:      backend.GetPendingRules(),
		Rules:             backend.GetRules(),
		TotalGasPowerLeft: backend.TotalGasPowerLeft(),
		MinTip:            new(big.Int).Set(backend.MinGasTip()),
	}
}

func (c *rpcCore) PendingHeads(max int) []rpccore.PendingHead {
	pending, err := c.svc.txpool.Pending(false)
	if err != nil {
		return nil
	}
	heads := make([]rpccore.PendingHead, 0, min(len(pending), max))
	for _, txs := range pending {
		if len(heads) >= max {
			break
		}
		if len(txs) == 0 {
			continue
		}
		heads = append(heads, rpccore.PendingHead{Tx: txs[0]})
	}
	return heads
}
