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

	"github.com/0xsoniclabs/sonic/eventcheck/gaspowercheck"
	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/evmstore"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/iblockproc"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/utils/concurrent"
)

// RPCCore is the node as the RPC module sees it: the one implementation of
// core.Core, over the store, the pool, the feeds and the peers of the
// service. It starts no goroutine.
type RPCCore struct {
	service *Service
	reader  *EvmStateReader
}

var _ core.Core = (*RPCCore)(nil)

// NewRPCCore creates the RPC module's view of a service.
func NewRPCCore(service *Service) *RPCCore {
	return &RPCCore{
		service: service,
		reader:  &EvmStateReader{ServiceFeed: &service.feed, store: service.store},
	}
}

// The guards the node configuration puts on the transaction index.
var (
	errReceiptsNotIndexed     = errors.New("transactions index is disabled (enable TxIndex and re-process the DAGs)")
	errTransactionsNotIndexed = errors.New("transactions index is disabled (enable TxIndex and re-process the DAG)")
)

// errNotServedYet names a call of core.Core the adapter does not answer yet.
func errNotServedYet(call string) error {
	return fmt.Errorf("the RPC core does not serve %s yet", call)
}

func (c *RPCCore) store() *Store {
	return c.service.store
}

func (c *RPCCore) ChainInfo() core.ChainInfo {
	return core.ChainInfo{
		ChainID:        c.store().GetRules().NetworkID,
		GenesisID:      common.Hash(c.store().GetGenesisID()),
		UpgradeHeights: c.store().GetUpgradeHeights(),
	}
}

// Header reads a block's header as the block would have it, without
// decoding its transactions: their root is stored with the block.
func (c *RPCCore) Header(number uint64) (*evmcore.EvmHeader, error) {
	header := c.reader.Header(common.Hash{}, number)
	if header == nil {
		return nil, core.ErrNotFound
	}
	header.TxHash = c.store().GetBlock(idx.Block(number)).TransactionsHashRoot
	return header, nil
}

func (c *RPCCore) Block(number uint64) (*evmcore.EvmBlock, error) {
	block := c.reader.Block(common.Hash{}, number)
	if block == nil {
		return nil, core.ErrNotFound
	}
	return block, nil
}

func (c *RPCCore) BlockNumberByHash(blockHash common.Hash) (uint64, error) {
	number := c.store().GetBlockIndex(hash.Event(blockHash))
	if number == nil {
		return 0, core.ErrNotFound
	}
	return uint64(*number), nil
}

func (c *RPCCore) Receipts(number uint64) (types.Receipts, error) {
	if !c.service.config.TxIndex {
		return nil, errReceiptsNotIndexed
	}
	block := c.reader.Block(common.Hash{}, number)
	if block == nil {
		return nil, core.ErrNotFound
	}
	return c.store().evm.GetReceipts(
		idx.Block(number),
		c.store().GetEvmChainConfig(idx.Block(number)),
		block.Hash,
		uint64(block.Time.Unix()),
		block.BaseFee,
		new(big.Int), // the network has no blob gas
		block.Transactions,
	), nil
}

// Rules answers the rules of the epoch of a stored block, and above the
// committed head those of the open epoch.
func (c *RPCCore) Rules(number uint64) (*opera.Rules, error) {
	if idx.Block(number) > c.store().GetLatestBlockIndex() {
		rules := c.store().GetRules()
		return &rules, nil
	}
	block := c.store().GetBlock(idx.Block(number))
	if block == nil {
		return nil, core.ErrNotFound
	}
	epochState := c.store().GetHistoryEpochState(block.Epoch)
	if epochState == nil {
		return nil, core.ErrNotFound
	}
	return &epochState.Rules, nil
}

// TransactionByHash finds a transaction of a committed block.
func (c *RPCCore) TransactionByHash(transactionHash common.Hash) (*types.Transaction, uint64, uint64, error) {
	if !c.service.config.TxIndex {
		return nil, 0, 0, errTransactionsNotIndexed
	}
	position := c.store().evm.GetTxPosition(transactionHash)
	if position == nil || position.Block > c.store().GetLatestBlockIndex() {
		return nil, 0, 0, core.ErrNotFound
	}
	transaction := c.store().evm.GetTx(transactionHash)
	if transaction == nil {
		return nil, 0, 0, fmt.Errorf("transactions index is corrupted (position without body), txid=%s, block=%d", transactionHash, position.Block)
	}
	return transaction, uint64(position.Block), uint64(position.BlockOffset), nil
}

func (c *RPCCore) BundleInfo(executionPlanHash common.Hash) (core.BundleInfo, error) {
	info := c.store().GetBundleExecutionInfo(executionPlanHash)
	if info == nil {
		return core.BundleInfo{}, core.ErrNotFound
	}
	return core.BundleInfo{BlockNumber: info.BlockNumber, Offset: info.Position.Offset, Count: info.Position.Count}, nil
}

// Head is the newest block whose state is archived, and the committed head on
// a node without an archive.
func (c *RPCCore) Head() (*evmcore.EvmBlock, error) {
	head := c.store().GetLatestBlockIndex()
	archived, empty, err := c.store().evm.GetArchiveBlockHeight()
	if err != nil && !errors.Is(err, evmstore.NoArchiveError) {
		return nil, fmt.Errorf("failed to get latest archive block; %v", err)
	}
	if err == nil && !empty && idx.Block(archived) < head {
		head = idx.Block(archived)
	}
	return c.Block(uint64(head))
}

func (c *RPCCore) SubscribeHead(events chan<- core.HeadNotify) core.Subscription {
	return orEnded(c.service.feed.SubscribeNewHead(events))
}

// State opens the archive at a block. It is read by the block's number and
// verified against its state root.
func (c *RPCCore) State(number uint64) (state.StateDB, error) {
	block := c.store().GetBlock(idx.Block(number))
	if block == nil || idx.Block(number) > c.store().GetLatestBlockIndex() {
		return nil, core.ErrNotFound
	}
	stateDB, err := c.store().evm.GetBlockStateDb(new(big.Int).SetUint64(number), block.StateRoot)
	if errors.Is(err, evmstore.NoArchiveError) {
		return nil, noArchiveError{err}
	}
	return stateDB, err
}

// noArchiveError is the archive's refusal on a node without one, which keeps
// its words and is a core.ErrNoArchive.
type noArchiveError struct {
	error
}

func (e noArchiveError) Is(target error) bool {
	return target == core.ErrNoArchive
}

func (c *RPCCore) SendTransaction(ctx context.Context, transaction *types.Transaction) error {
	return c.service.txpool.AddLocal(transaction)
}

func (c *RPCCore) PoolNonce(address common.Address) uint64 {
	return c.service.txpool.Nonce(address)
}

func (c *RPCCore) PoolTransaction(transactionHash common.Hash) (*types.Transaction, error) {
	transaction := c.service.txpool.Get(transactionHash)
	if transaction == nil {
		return nil, core.ErrNotFound
	}
	return transaction, nil
}

// PoolNextTransactions is the first executable transaction of every sender;
// none when the pool cannot tell.
func (c *RPCCore) PoolNextTransactions() types.Transactions {
	pending, err := c.service.txpool.Pending(false)
	if err != nil {
		return nil
	}
	next := make(types.Transactions, 0, len(pending))
	for _, transactions := range pending {
		if len(transactions) > 0 {
			next = append(next, transactions[0])
		}
	}
	return next
}

func (c *RPCCore) PoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
	return c.service.txpool.Content()
}

func (c *RPCCore) PoolContentFrom(address common.Address) (types.Transactions, types.Transactions) {
	return c.service.txpool.ContentFrom(address)
}

func (c *RPCCore) PoolStats() (int, int) {
	return c.service.txpool.Stats()
}

func (c *RPCCore) SubscribeTransactions(events chan<- evmcore.NewTxsNotify) core.Subscription {
	return orEnded(c.service.txpool.SubscribeNewTxsNotify(events))
}

func (c *RPCCore) Logs(ctx context.Context, query core.LogQuery) ([]*types.Log, error) {
	return nil, errNotServedYet("Logs")
}

func (c *RPCCore) Progress() core.Progress {
	mine := c.service.handler.myProgress()
	highest := c.service.handler.highestPeerProgress()
	lastBlock := c.store().GetBlock(mine.LastBlockIdx)
	return core.Progress{
		CurrentEpoch:     uint64(mine.Epoch),
		CurrentBlock:     uint64(mine.LastBlockIdx),
		CurrentBlockHash: common.Hash(mine.LastBlockAtropos),
		CurrentBlockTime: lastBlock.Time,
		HighestBlock:     uint64(highest.LastBlockIdx),
		HighestEpoch:     uint64(highest.Epoch),
	}
}

func (c *RPCCore) OracleInputs() core.OracleInputs {
	blockState, epochState := c.store().GetBlockEpochState()
	nextEpochRules := epochState.Rules
	if blockState.DirtyRules != nil {
		nextEpochRules = *blockState.DirtyRules
	}
	return core.OracleInputs{
		NextEpochRules:    nextEpochRules,
		TotalGasPowerLeft: c.totalGasPowerLeft(blockState, epochState),
		MinTip:            c.service.txpool.MinTip(),
	}
}

// totalGasPowerLeft is the long-term gas power the validators have obtained,
// read from the last event of each validator in the epoch, or from the last
// one of the epoch before for a validator without an event yet.
func (c *RPCCore) totalGasPowerLeft(blockState iblockproc.BlockState, epochState iblockproc.EpochState) uint64 {
	lastEvents := c.store().GetLastEvents(epochState.Epoch)
	if lastEvents == nil {
		lastEvents = concurrent.WrapValidatorEventsSet(map[idx.ValidatorID]hash.Event{})
	}
	lastEvents.RLock()
	defer lastEvents.RUnlock()
	gasPowerConfig := gaspowercheck.Config{
		Idx:                inter.LongTermGas,
		AllocPerSec:        epochState.Rules.Economy.LongGasPower.AllocPerSec,
		MaxAllocPeriod:     epochState.Rules.Economy.LongGasPower.MaxAllocPeriod,
		MinEnsuredAlloc:    epochState.Rules.Economy.Gas.MaxEventGas,
		StartupAllocPeriod: epochState.Rules.Economy.LongGasPower.StartupAllocPeriod,
		MinStartupGas:      epochState.Rules.Economy.LongGasPower.MinStartupGas,
	}
	seen := map[idx.ValidatorID]bool{}
	total := uint64(0)
	for _, last := range lastEvents.Val {
		event := c.store().GetEvent(last)
		left := event.GasPowerLeft().Gas[inter.LongTermGas]
		left += blockState.GetValidatorState(event.Creator(), epochState.Validators).DirtyGasRefund
		_, maximum, _ := gaspowercheck.CalcValidatorGasPowerPerSec(event.Creator(), epochState.Validators, gasPowerConfig)
		total += min(left, maximum)
		seen[event.Creator()] = true
	}
	for i := idx.Validator(0); i < epochState.Validators.Len(); i++ {
		validator := epochState.Validators.GetID(i)
		if seen[validator] {
			continue
		}
		left := epochState.ValidatorStates[i].PrevEpochEvent.GasPowerLeft.Gas[inter.LongTermGas]
		left += epochState.ValidatorStates[i].GasRefund
		_, maximum, startup := gaspowercheck.CalcValidatorGasPowerPerSec(validator, epochState.Validators, gasPowerConfig)
		total += max(min(left, maximum), startup)
	}
	return total
}

func (c *RPCCore) PeerCount() int {
	return c.service.p2pServer.PeerCount()
}

// EpochOf answers the epoch of a stored block, and above the committed head
// the open epoch.
func (c *RPCCore) EpochOf(number uint64) (uint64, error) {
	if idx.Block(number) > c.store().GetLatestBlockIndex() {
		return uint64(c.store().GetEpoch()), nil
	}
	block := c.store().GetBlock(idx.Block(number))
	if block == nil {
		return 0, core.ErrNotFound
	}
	return uint64(block.Epoch), nil
}

func (c *RPCCore) EpochState(epoch uint64) (core.EpochSnapshot, error) {
	blockState, epochState := c.store().GetHistoryBlockEpochState(idx.Epoch(epoch))
	if blockState == nil || epochState == nil {
		return core.EpochSnapshot{}, core.ErrNotFound
	}
	return core.EpochSnapshot{BlockState: *blockState, EpochState: *epochState}, nil
}

func (c *RPCCore) LiveEpochState() core.EpochSnapshot {
	blockState, epochState := c.store().GetBlockEpochState()
	return core.EpochSnapshot{BlockState: blockState, EpochState: epochState}
}

func (c *RPCCore) LiveEpochHeads(epoch uint64) ([]common.Hash, error) {
	heads := c.store().GetHeads(idx.Epoch(epoch))
	if heads == nil {
		return nil, core.ErrNotFound
	}
	heads.RLock()
	defer heads.RUnlock()
	return eventHashes(heads.Val.Slice()), nil
}

func (c *RPCCore) FindEvents(epoch uint64, lamport uint32, prefix []byte) []common.Hash {
	return eventHashes(c.store().FindEventHashes(idx.Epoch(epoch), idx.Lamport(lamport), prefix))
}

func (c *RPCCore) Event(id common.Hash) (*inter.Event, error) {
	event := c.store().GetEvent(hash.Event(id))
	if event == nil {
		return nil, core.ErrNotFound
	}
	return event, nil
}

func (c *RPCCore) EventPayload(id common.Hash) (*inter.EventPayload, error) {
	payload := c.store().GetEventPayload(hash.Event(id))
	if payload == nil {
		return nil, core.ErrNotFound
	}
	return payload, nil
}

func (c *RPCCore) SendTransactions(transactions types.Transactions) error {
	if !c.service.config.EnableTestOnlyApi {
		return errors.New("this feature is disabled")
	}
	return errors.Join(c.service.txpool.AddLocals(transactions)...)
}

func (c *RPCCore) EmitTransactions(transactions types.Transactions) error {
	if !c.service.config.EnableTestOnlyApi {
		return errors.New("this feature is disabled")
	}
	if len(c.service.emitters) == 0 {
		return errors.New("no emitters available to propose transactions")
	}
	return c.service.emitters[0].ForceEventEmissionForTesting(transactions)
}

func eventHashes(events hash.Events) []common.Hash {
	hashes := make([]common.Hash, len(events))
	for i, event := range events {
		hashes[i] = common.Hash(event)
	}
	return hashes
}

// orEnded answers a feed's subscription, or an ended one where a stopped feed
// answers none.
func orEnded(subscription core.Subscription) core.Subscription {
	if subscription == nil {
		return endedSubscription{}
	}
	return subscription
}

// endedSubscription is a subscription that delivers nothing more.
type endedSubscription struct{}

var closedErrors = func() chan error {
	errors := make(chan error)
	close(errors)
	return errors
}()

func (endedSubscription) Err() <-chan error { return closedErrors }

func (endedSubscription) Unsubscribe() {}
