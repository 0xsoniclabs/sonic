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

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/evmstore"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/rpc/core"
)

// The errors of a node that keeps no transaction index, worded as the legacy
// backend words them.
var (
	errReceiptsWithoutIndex     = errors.New("transactions index is disabled (enable TxIndex and re-process the DAGs)")
	errTransactionsWithoutIndex = errors.New("transactions index is disabled (enable TxIndex and re-process the DAG)")
)

// rpcCore is the node as the RPC module sees it, over the store, the state
// reader, the pool and the feeds. It starts no goroutine on the module's
// behalf, and it reports a block above the committed head as not found.
type rpcCore struct {
	store   *Store
	reader  *EvmStateReader
	txpool  TxPool
	feed    *ServiceFeed
	handler *handler
	config  Config
	oracle  *GPOBackend
}

// RPCCore returns the node as the RPC module sees it.
func (s *Service) RPCCore() core.Core {
	return &rpcCore{
		store:   s.store,
		reader:  s.EthAPI.state.(*EvmStateReader),
		txpool:  s.txpool,
		feed:    &s.feed,
		handler: s.handler,
		config:  s.config,
		oracle:  &GPOBackend{store: s.store, txpool: s.txpool},
	}
}

func (adapter *rpcCore) committed() uint64 {
	return uint64(adapter.store.GetLatestBlockIndex())
}

func (adapter *rpcCore) ChainInfo() (core.ChainInfo, error) {
	return core.ChainInfo{
		ChainID:        adapter.store.GetRules().NetworkID,
		GenesisID:      common.Hash(adapter.store.GetGenesisID()),
		UpgradeHeights: adapter.store.GetUpgradeHeights(),
	}, nil
}

func (adapter *rpcCore) Header(number uint64) (*evmcore.EvmHeader, error) {
	header := adapter.reader.Header(common.Hash{}, number)
	if header == nil {
		return nil, core.ErrNotFound
	}
	return header, nil
}

func (adapter *rpcCore) Block(number uint64) (*evmcore.EvmBlock, error) {
	block := adapter.reader.Block(common.Hash{}, number)
	if block == nil {
		return nil, core.ErrNotFound
	}
	return block, nil
}

func (adapter *rpcCore) BlockNumberByHash(blockHash common.Hash) (uint64, error) {
	number := adapter.store.GetBlockIndex(hash.Event(blockHash))
	if number == nil || uint64(*number) > adapter.committed() {
		return 0, core.ErrNotFound
	}
	return uint64(*number), nil
}

func (adapter *rpcCore) Receipts(number uint64) (types.Receipts, error) {
	if !adapter.config.TxIndex {
		return nil, errReceiptsWithoutIndex
	}
	block := adapter.reader.Block(common.Hash{}, number)
	if block == nil {
		return nil, core.ErrNotFound
	}
	return adapter.store.evm.GetReceipts(
		idx.Block(number),
		adapter.store.GetEvmChainConfig(idx.Block(number)),
		block.Hash,
		uint64(block.Time.Unix()),
		block.BaseFee,
		new(big.Int), // Sonic has no blob gas
		block.Transactions,
	), nil
}

// Rules reads the epoch from the stored block, not from the EVM header, whose
// build would also look up the epoch state and the parent hash.
func (adapter *rpcCore) Rules(number uint64) (opera.Rules, error) {
	committed := adapter.committed()
	if number == committed+1 {
		return adapter.store.GetRules(), nil
	}
	if number > committed {
		return opera.Rules{}, core.ErrNotFound
	}
	block := adapter.store.GetBlock(idx.Block(number))
	if block == nil {
		return opera.Rules{}, core.ErrNotFound
	}
	epochState := adapter.store.GetHistoryEpochState(block.Epoch)
	if epochState == nil {
		return opera.Rules{}, fmt.Errorf("no network rules found for block height %d", number)
	}
	return epochState.Rules, nil
}

func (adapter *rpcCore) TransactionByHash(transactionHash common.Hash) (*types.Transaction, uint64, uint64, error) {
	if !adapter.config.TxIndex {
		return nil, 0, 0, errTransactionsWithoutIndex
	}
	position := adapter.store.evm.GetTxPosition(transactionHash)
	if position == nil || uint64(position.Block) > adapter.committed() {
		return nil, 0, 0, core.ErrNotFound
	}
	transaction := adapter.store.evm.GetTx(transactionHash)
	if transaction == nil {
		return nil, 0, 0, fmt.Errorf("transactions index is corrupted (position without body), txid=%s, block=%d", transactionHash, position.Block)
	}
	return transaction, uint64(position.Block), uint64(position.BlockOffset), nil
}

func (adapter *rpcCore) BundleInfo(executionPlanHash common.Hash) (core.BundleInfo, error) {
	info := adapter.store.GetBundleExecutionInfo(executionPlanHash)
	if info == nil {
		return core.BundleInfo{}, core.ErrNotFound
	}
	return core.BundleInfo{
		BlockNumber: info.BlockNumber,
		Offset:      info.Position.Offset,
		Count:       info.Position.Count,
	}, nil
}

// Head returns the newest block whose state is archived; a node without an
// archive serves its committed head.
func (adapter *rpcCore) Head() (*evmcore.EvmBlock, error) {
	latest := adapter.committed()
	height, empty, err := adapter.store.evm.GetArchiveBlockHeight()
	if err != nil && !errors.Is(err, evmstore.NoArchiveError) {
		return nil, fmt.Errorf("failed to get latest archive block; %v", err)
	}
	if err == nil && !empty && height < latest {
		latest = height
	}
	return adapter.Block(latest)
}

func (adapter *rpcCore) SubscribeHead(events chan<- core.HeadEvent) core.Subscription {
	return liveOrEnded(adapter.feed.SubscribeNewHead(events))
}

func (adapter *rpcCore) State(number uint64) (state.StateDB, error) {
	if number > adapter.committed() {
		return nil, core.ErrNotFound
	}
	block := adapter.store.GetBlock(idx.Block(number))
	if block == nil {
		return nil, core.ErrNotFound
	}
	return adapter.store.evm.GetBlockStateDb(new(big.Int).SetUint64(number), block.StateRoot)
}

func (adapter *rpcCore) SendTransaction(_ context.Context, transaction *types.Transaction) error {
	return adapter.txpool.AddLocal(transaction)
}

func (adapter *rpcCore) PoolNonce(address common.Address) uint64 {
	return adapter.txpool.Nonce(address)
}

func (adapter *rpcCore) PoolTransaction(transactionHash common.Hash) (*types.Transaction, error) {
	transaction := adapter.txpool.Get(transactionHash)
	if transaction == nil {
		return nil, core.ErrNotFound
	}
	return transaction, nil
}

func (adapter *rpcCore) PoolTransactions() (types.Transactions, error) {
	pending, err := adapter.txpool.Pending(false)
	if err != nil {
		return nil, err
	}
	var transactions types.Transactions
	for _, batch := range pending {
		transactions = append(transactions, batch...)
	}
	return transactions, nil
}

func (adapter *rpcCore) PoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
	return adapter.txpool.Content()
}

func (adapter *rpcCore) PoolContentFrom(address common.Address) (types.Transactions, types.Transactions) {
	return adapter.txpool.ContentFrom(address)
}

func (adapter *rpcCore) PoolStats() (int, int) {
	return adapter.txpool.Stats()
}

func (adapter *rpcCore) SubscribeTransactions(events chan<- evmcore.NewTxsNotify) core.Subscription {
	return liveOrEnded(adapter.txpool.SubscribeNewTxsNotify(events))
}

func (adapter *rpcCore) FindLogs(ctx context.Context, from, to uint64, pattern [][]common.Hash, limit uint) ([]*types.Log, error) {
	return adapter.store.evm.EvmLogs.FindInBlocks(ctx, idx.Block(from), idx.Block(to), pattern, limit)
}

func (adapter *rpcCore) Progress() core.Progress {
	own := adapter.handler.myProgress()
	highest := adapter.handler.highestPeerProgress()
	var blockTime uint64
	if block := adapter.store.GetBlock(own.LastBlockIdx); block != nil {
		blockTime = uint64(block.Time)
	}
	return core.Progress{
		CurrentEpoch:     uint64(own.Epoch),
		CurrentBlock:     uint64(own.LastBlockIdx),
		CurrentBlockHash: common.Hash(own.LastBlockAtropos),
		CurrentBlockTime: inter.Timestamp(blockTime),
		HighestBlock:     uint64(highest.LastBlockIdx),
		HighestEpoch:     uint64(highest.Epoch),
	}
}

func (adapter *rpcCore) OracleInputs() core.OracleInputs {
	return core.OracleInputs{
		PendingRules:      adapter.oracle.GetPendingRules(),
		TotalGasPowerLeft: adapter.oracle.TotalGasPowerLeft(),
		MinTip:            adapter.txpool.MinTip(),
	}
}

func (adapter *rpcCore) PendingHeads() types.Transactions {
	pending, err := adapter.txpool.Pending(false)
	if err != nil {
		return nil
	}
	heads := make(types.Transactions, 0, len(pending))
	for _, transactions := range pending {
		if len(transactions) > 0 {
			heads = append(heads, transactions[0])
		}
	}
	return heads
}

// liveOrEnded answers the subscription of a stopped feed, which the feed's
// scope hands out as nil, with one that has already ended.
func liveOrEnded(subscription notify.Subscription) core.Subscription {
	if subscription == nil {
		return endedSubscription{}
	}
	return subscription
}

// endedSubscription is a subscription that has already ended.
type endedSubscription struct{}

var closedErrors = func() chan error {
	ended := make(chan error)
	close(ended)
	return ended
}()

func (endedSubscription) Err() <-chan error { return closedErrors }

func (endedSubscription) Unsubscribe() {}
