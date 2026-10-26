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
	"fmt"
	"math/big"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	notify "github.com/ethereum/go-ethereum/event"

	"github.com/0xsoniclabs/sonic/eventcheck/gaspowercheck"
	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/evmstore"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	rpccore "github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/utils/concurrent"
)

// The guards the node's configuration sets on the RPC module's reads, worded
// as the legacy backend worded them.
var (
	errReceiptsWithoutTxIndex = errors.New("transactions index is disabled (enable TxIndex and re-process the DAGs)")
	errTxWithoutTxIndex       = errors.New("transactions index is disabled (enable TxIndex and re-process the DAG)")
)

// rpcCore is the RPC module's view of the node. It starts no goroutine; every
// lookup reports a missing item as rpccore.ErrNotFound.
type rpcCore struct {
	svc   *Service
	store *Store
	state *EvmStateReader
}

var _ rpccore.Core = (*rpcCore)(nil)

// RPCCore returns the node as the RPC module sees it.
func (s *Service) RPCCore() rpccore.Core {
	return &rpcCore{svc: s, store: s.store, state: s.stateReader}
}

func (c *rpcCore) ChainInfo() rpccore.ChainInfo {
	networkID := c.store.GetRules().NetworkID
	return rpccore.ChainInfo{
		ChainID:        new(big.Int).SetUint64(networkID),
		NetworkID:      networkID,
		GenesisID:      common.Hash(c.store.GetGenesisID()),
		UpgradeHeights: c.store.GetUpgradeHeights(),
	}
}

func (c *rpcCore) Header(number uint64) (*evmcore.EvmHeader, error) {
	header := c.state.Header(common.Hash{}, number)
	if header == nil {
		return nil, rpccore.ErrNotFound
	}
	return header, nil
}

func (c *rpcCore) Block(number uint64) (*evmcore.EvmBlock, error) {
	block := c.state.Block(common.Hash{}, number)
	if block == nil {
		return nil, rpccore.ErrNotFound
	}
	return block, nil
}

func (c *rpcCore) BlockNumberByHash(h common.Hash) (uint64, error) {
	index := c.store.GetBlockIndex(hash.Event(h))
	if index == nil {
		return 0, rpccore.ErrNotFound
	}
	return uint64(*index), nil
}

func (c *rpcCore) Receipts(number uint64) (types.Receipts, error) {
	if !c.svc.config.TxIndex {
		return nil, errReceiptsWithoutTxIndex
	}
	block := c.state.Block(common.Hash{}, number)
	if block == nil {
		return nil, rpccore.ErrNotFound
	}
	return c.svc.EthAPI.FetchReceiptsForBlock(block), nil
}

// Rules are the rules of a block's epoch, read from the stored block rather
// than the EVM header; for the block after the committed head, the rules of
// the open epoch.
func (c *rpcCore) Rules(number uint64) (*opera.Rules, error) {
	latest := uint64(c.store.GetLatestBlockIndex())
	switch {
	case number == latest+1:
		rules := c.store.GetRules()
		return &rules, nil
	case number > latest:
		return nil, rpccore.ErrNotFound
	}
	block := c.store.GetBlock(idx.Block(number))
	if block == nil {
		return nil, rpccore.ErrNotFound
	}
	es := c.store.GetHistoryEpochState(block.Epoch)
	if es == nil {
		return nil, rpccore.ErrNotFound
	}
	rules := es.Rules
	return &rules, nil
}

func (c *rpcCore) TxByHash(h common.Hash) (*types.Transaction, uint64, uint64, error) {
	if !c.svc.config.TxIndex {
		return nil, 0, 0, errTxWithoutTxIndex
	}
	position := c.svc.EthAPI.GetTxPosition(h)
	if position == nil {
		return nil, 0, 0, rpccore.ErrNotFound
	}
	tx := c.store.evm.GetTx(h)
	if tx == nil {
		return nil, 0, 0, fmt.Errorf("transactions index is corrupted (position without body), txid=%s, block=%d", h, position.Block)
	}
	return tx, uint64(position.Block), uint64(position.BlockOffset), nil
}

func (c *rpcCore) BundleInfo(executionPlanHash common.Hash) (rpccore.BundleInfo, error) {
	info := c.store.GetBundleExecutionInfo(executionPlanHash)
	if info == nil {
		return rpccore.BundleInfo{}, rpccore.ErrNotFound
	}
	return rpccore.BundleInfo{BlockNumber: info.BlockNumber, Offset: info.Position.Offset, Count: info.Position.Count}, nil
}

// Head is the newest block whose state is archived; without an archive, as on
// a validator, the committed head.
func (c *rpcCore) Head() (*evmcore.EvmBlock, error) {
	latest := c.store.GetLatestBlockIndex()
	height, empty, err := c.store.evm.GetArchiveBlockHeight()
	switch {
	case errors.Is(err, evmstore.NoArchiveError):
	case err != nil:
		return nil, fmt.Errorf("failed to get latest archive block; %v", err)
	case !empty && idx.Block(height) < latest:
		latest = idx.Block(height)
	}
	block := c.state.Block(common.Hash{}, uint64(latest))
	if block == nil {
		return nil, rpccore.ErrNotFound
	}
	return block, nil
}

func (c *rpcCore) SubscribeHead(ch chan<- rpccore.HeadEvent) rpccore.Subscription {
	return orEnded(c.svc.feed.SubscribeHead(ch))
}

func (c *rpcCore) State(number uint64) (state.StateDB, error) {
	if number > uint64(c.store.GetLatestBlockIndex()) {
		return nil, rpccore.ErrNotFound
	}
	block := c.store.GetBlock(idx.Block(number))
	if block == nil {
		return nil, rpccore.ErrNotFound
	}
	return c.state.BlockStateDB(new(big.Int).SetUint64(number), common.Hash(block.StateRoot))
}

func (c *rpcCore) SendTx(_ context.Context, tx *types.Transaction) error {
	return c.svc.txpool.AddLocal(tx)
}

func (c *rpcCore) PoolNonce(addr common.Address) uint64 {
	return c.svc.txpool.Nonce(addr)
}

func (c *rpcCore) PoolTransaction(h common.Hash) (*types.Transaction, error) {
	tx := c.svc.txpool.Get(h)
	if tx == nil {
		return nil, rpccore.ErrNotFound
	}
	return tx, nil
}

func (c *rpcCore) PoolTransactions() (types.Transactions, error) {
	return c.svc.EthAPI.GetPoolTransactions()
}

func (c *rpcCore) PoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
	return c.svc.txpool.Content()
}

func (c *rpcCore) PoolContentFrom(addr common.Address) (types.Transactions, types.Transactions) {
	return c.svc.txpool.ContentFrom(addr)
}

func (c *rpcCore) PoolStats() (int, int) {
	return c.svc.txpool.Stats()
}

func (c *rpcCore) SubscribeTxs(ch chan<- evmcore.NewTxsNotify) rpccore.Subscription {
	return orEnded(c.svc.txpool.SubscribeNewTxsNotify(ch))
}

func (c *rpcCore) FindLogs(ctx context.Context, from, to uint64, pattern [][]common.Hash, limit uint) ([]*types.Log, error) {
	return c.store.evm.EvmLogs.FindInBlocks(ctx, idx.Block(from), idx.Block(to), pattern, limit)
}

func (c *rpcCore) Progress() rpccore.Progress {
	p := c.svc.EthAPI.Progress()
	return rpccore.Progress{
		CurrentEpoch:     uint64(p.CurrentEpoch),
		CurrentBlock:     uint64(p.CurrentBlock),
		CurrentBlockHash: common.Hash(p.CurrentBlockHash),
		CurrentBlockTime: p.CurrentBlockTime,
		HighestBlock:     uint64(p.HighestBlock),
		HighestEpoch:     uint64(p.HighestEpoch),
	}
}

func (c *rpcCore) OracleInputs() rpccore.OracleInputs {
	return rpccore.OracleInputs{
		PendingRules:      c.pendingRules(),
		TotalGasPowerLeft: c.totalGasPowerLeft(),
		MinTip:            c.svc.txpool.MinTip(),
	}
}

// pendingRules are the rules of the open epoch, those of the next block.
func (c *rpcCore) pendingRules() opera.Rules {
	bs, es := c.store.GetBlockEpochState()
	if bs.DirtyRules != nil {
		return *bs.DirtyRules
	}
	return es.Rules
}

// totalGasPowerLeft returns a total amount of obtained gas power by the validators, according to the latest events from each validator
func (c *rpcCore) totalGasPowerLeft() uint64 {
	bs, es := c.store.GetBlockEpochState()
	set := c.store.GetLastEvents(es.Epoch)
	if set == nil {
		set = concurrent.WrapValidatorEventsSet(map[idx.ValidatorID]hash.Event{})
	}
	set.RLock()
	defer set.RUnlock()
	metValidators := map[idx.ValidatorID]bool{}
	total := uint64(0)
	gasPowerCheckCfg := gaspowercheck.Config{
		Idx:                inter.LongTermGas,
		AllocPerSec:        es.Rules.Economy.LongGasPower.AllocPerSec,
		MaxAllocPeriod:     es.Rules.Economy.LongGasPower.MaxAllocPeriod,
		MinEnsuredAlloc:    es.Rules.Economy.Gas.MaxEventGas,
		StartupAllocPeriod: es.Rules.Economy.LongGasPower.StartupAllocPeriod,
		MinStartupGas:      es.Rules.Economy.LongGasPower.MinStartupGas,
	}
	// count GasPowerLeft from latest events of this epoch
	for _, tip := range set.Val {
		e := c.store.GetEvent(tip)
		left := e.GasPowerLeft().Gas[inter.LongTermGas]
		left += bs.GetValidatorState(e.Creator(), es.Validators).DirtyGasRefund

		_, max, _ := gaspowercheck.CalcValidatorGasPowerPerSec(e.Creator(), es.Validators, gasPowerCheckCfg)
		if left > max {
			left = max
		}
		total += left

		metValidators[e.Creator()] = true
	}
	// count GasPowerLeft from last events of prev epoch if no event in current epoch is present
	for i := idx.Validator(0); i < es.Validators.Len(); i++ {
		vid := es.Validators.GetID(i)
		if !metValidators[vid] {
			left := es.ValidatorStates[i].PrevEpochEvent.GasPowerLeft.Gas[inter.LongTermGas]
			left += es.ValidatorStates[i].GasRefund

			_, max, startup := gaspowercheck.CalcValidatorGasPowerPerSec(vid, es.Validators, gasPowerCheckCfg)
			if left > max {
				left = max
			}
			if left < startup {
				left = startup
			}
			total += left
		}
	}

	return total
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
		tx := txs[0]
		heads = append(heads, rpccore.PendingHead{Gas: tx.Gas(), GasTipCap: tx.GasTipCap(), GasFeeCap: tx.GasFeeCap()})
	}
	return heads
}

// endedSubscription stands for the subscription of a stopped feed.
type endedSubscription struct{ err chan error }

func (s endedSubscription) Err() <-chan error { return s.err }
func (s endedSubscription) Unsubscribe()      {}

// orEnded turns the nil subscription a stopped feed hands out into one that
// has already ended.
func orEnded(sub notify.Subscription) rpccore.Subscription {
	if sub == nil {
		ch := make(chan error)
		close(ch)
		return endedSubscription{err: ch}
	}
	return sub
}
