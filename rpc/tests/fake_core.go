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

// Package tests is the test support of the RPC module: FakeCore, the one
// test double of the node, accounts that sign on its chain, and a harness
// that serves a namespace as a transport would.
package tests

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	"math/big"
	"reflect"
	"slices"
	"sync"
	"time"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// Mode is how a FakeCore method answers.
type Mode int

const (
	// Normal answers from the fake's data.
	Normal Mode = iota
	// Panic panics.
	Panic
	// Fail answers ErrInjected.
	Fail
	// NotFound answers core.ErrNotFound.
	NotFound
	// Nil answers a zero value without an error, which no lookup of the node
	// does, so that handlers are checked to survive it.
	Nil
	// NoArchive answers core.ErrNoArchive, as a node without an archive does.
	NoArchive
)

// ErrInjected is the error of a method in Fail mode.
var ErrInjected = errors.New("injected failure")

// ChainID is the chain id of a FakeCore.
const ChainID = 4003

// FakeCore is an in-memory core.Core: a chain of blocks with their receipts,
// a pool, and the node's liveness facts, changed only through its setters
// and handed out as copies. Each method can be put into a Mode, is counted,
// and can be held at a gate; every call waits the fake's delay.
type FakeCore struct {
	mutex      sync.Mutex
	blocks     []*evmcore.EvmBlock
	receipts   []types.Receipts
	rules      opera.Rules
	archived   uint64
	upgrades   []opera.UpgradeHeight
	progress   core.Progress
	peers      int
	pool       map[common.Hash]*types.Transaction
	queued     types.Transactions
	poolNonces map[common.Address]uint64
	bundles    map[common.Hash]core.BundleInfo
	// epoch is the epoch of the blocks added next.
	epoch       uint64
	epochStates map[uint64]core.EpochSnapshot
	liveEpoch   core.EpochSnapshot
	epochHeads  map[uint64][]common.Hash
	events      map[common.Hash]*inter.EventPayload
	logQueries  []core.LogQuery
	modes       map[string]Mode
	calls       map[string]int
	gates       map[string]chan struct{}
	delay       time.Duration
	// openState opens the state of a Carmen core; a fake without it has no
	// state.
	openState  func() (state.StateDB, error)
	openStates int
	stateRoot  common.Hash
	// oracleInputs replace the inputs derived from the rules if set.
	oracleInputs *core.OracleInputs
	heads        feed[core.HeadNotify]
	transactions feed[evmcore.NewTxsNotify]
}

var _ core.Core = (*FakeCore)(nil)

// coreMethods are the names of the methods of core.Core.
var coreMethods = func() map[string]bool {
	names := map[string]bool{}
	for method := range reflect.TypeFor[core.Core]().Methods() {
		names[method.Name] = true
	}
	return names
}()

// NewFakeCore creates a fake with a genesis block as its head.
func NewFakeCore() *FakeCore {
	rules := opera.FakeNetRules(opera.GetCantoUpgrades())
	rules.NetworkID = ChainID
	fake := &FakeCore{
		rules:       rules,
		upgrades:    []opera.UpgradeHeight{opera.MakeUpgradeHeight(rules.Upgrades, 0)},
		pool:        map[common.Hash]*types.Transaction{},
		poolNonces:  map[common.Address]uint64{},
		bundles:     map[common.Hash]core.BundleInfo{},
		epoch:       1,
		epochStates: map[uint64]core.EpochSnapshot{},
		epochHeads:  map[uint64][]common.Hash{},
		events:      map[common.Hash]*inter.EventPayload{},
		modes:       map[string]Mode{},
		calls:       map[string]int{},
		gates:       map[string]chan struct{}{},
	}
	fake.AddBlock()
	return fake
}

// AddBlock appends a block with the given transactions, each successful and
// using 21000 gas, and answers its number. The block is stored, not yet
// archived: SetHead serves it.
func (f *FakeCore) AddBlock(transactions ...*types.Transaction) uint64 {
	receipts := make(types.Receipts, len(transactions))
	for i := range transactions {
		receipts[i] = &types.Receipt{
			Status:            types.ReceiptStatusSuccessful,
			CumulativeGasUsed: uint64(i+1) * 21000,
			Logs:              []*types.Log{},
		}
	}
	return f.AddBlockWithReceipts(transactions, receipts)
}

// AddBlockWithReceipts appends a block with the given transactions and the
// stored part of their receipts (status, cumulative gas and logs), and
// answers its number. The other fields of the receipts are derived as the
// node derives them.
func (f *FakeCore) AddBlockWithReceipts(transactions types.Transactions, receipts types.Receipts) uint64 {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	number := uint64(len(f.blocks))
	header := &evmcore.EvmHeader{
		Number:   new(big.Int).SetUint64(number),
		Hash:     crypto.Keccak256Hash([]byte(fmt.Sprintf("block %d", number))),
		Root:     f.stateRoot,
		Time:     inter.Timestamp(number+1) * inter.Timestamp(time.Second),
		Duration: time.Second,
		GasLimit: 1 << 40,
		BaseFee:  big.NewInt(1e9),
		Epoch:    idx.Epoch(f.epoch),
	}
	if number > 0 {
		header.ParentHash = f.blocks[number-1].Hash
	}
	var gasUsed uint64
	if len(receipts) > 0 {
		gasUsed = receipts[len(receipts)-1].CumulativeGasUsed
	}
	header.GasUsed = gasUsed
	block := evmcore.NewEvmBlock(header, slices.Clone(transactions))
	config := opera.CreateTransientEvmChainConfig(ChainID, f.upgrades, 0)
	derived := copyReceipts(receipts)
	if err := derived.DeriveFields(config, block.Hash, number, uint64(block.Time.Unix()), block.BaseFee, new(big.Int), block.Transactions); err != nil {
		panic(err)
	}
	f.blocks = append(f.blocks, block)
	f.receipts = append(f.receipts, derived)
	return number
}

// SetHead archives the blocks up to number and announces the new head with
// its logs on the head feed.
func (f *FakeCore) SetHead(number uint64) {
	f.mutex.Lock()
	f.archived = number
	block := copyBlock(f.blocks[number])
	var logs []*types.Log
	for _, receipt := range f.receipts[number] {
		logs = append(logs, receipt.Logs...)
	}
	f.mutex.Unlock()
	f.heads.send(core.HeadNotify{Block: block, Logs: logs})
}

// SetProgress sets the synchronisation state the node reports.
func (f *FakeCore) SetProgress(progress core.Progress) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.progress = progress
}

// SetPeerCount sets the number of peers the node reports.
func (f *FakeCore) SetPeerCount(peers int) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.peers = peers
}

// SetUpgradeHeights sets the upgrade heights the chain reports.
func (f *FakeCore) SetUpgradeHeights(heights []opera.UpgradeHeight) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.upgrades = slices.Clone(heights)
}

// AddPoolTransaction puts a transaction into the pool and announces it on
// the pool's feed.
func (f *FakeCore) AddPoolTransaction(transaction *types.Transaction) {
	f.mutex.Lock()
	f.pool[transaction.Hash()] = transaction
	f.mutex.Unlock()
	f.transactions.send(evmcore.NewTxsNotify{Txs: []*types.Transaction{transaction}})
}

// AddQueuedTransaction puts a transaction into the pool's queue, which only
// the pool's content and statistics tell.
func (f *FakeCore) AddQueuedTransaction(transaction *types.Transaction) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.queued = append(f.queued, transaction)
}

// SetBundleInfo records where the bundle of an execution plan was executed.
func (f *FakeCore) SetBundleInfo(executionPlanHash common.Hash, info core.BundleInfo) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.bundles[executionPlanHash] = info
}

// SetEpoch sets the epoch of the blocks added next.
func (f *FakeCore) SetEpoch(epoch uint64) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.epoch = epoch
}

// SetEpochState sets the states stored when an epoch opened.
func (f *FakeCore) SetEpochState(epoch uint64, states core.EpochSnapshot) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.epochStates[epoch] = states
}

// SetLiveEpochState sets the block and epoch states the node has now.
func (f *FakeCore) SetLiveEpochState(states core.EpochSnapshot) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.liveEpoch = states
}

// SetLiveEpochHeads sets the events without descendants of the open epoch,
// the only one whose heads the node keeps.
func (f *FakeCore) SetLiveEpochHeads(epoch uint64, heads []common.Hash) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.epochHeads = map[uint64][]common.Hash{epoch: heads}
}

// AddEvent puts an event into the DAG.
func (f *FakeCore) AddEvent(event *inter.EventPayload) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.events[common.Hash(event.ID())] = event
}

// SetPoolNonce sets the pending nonce of a sender.
func (f *FakeCore) SetPoolNonce(address common.Address, nonce uint64) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.poolNonces[address] = nonce
}

// SetRules sets the rules of every block.
func (f *FakeCore) SetRules(rules opera.Rules) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.rules = rules
}

// SetOracleInputs sets what the gas price oracle reads.
func (f *FakeCore) SetOracleInputs(inputs core.OracleInputs) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.oracleInputs = &inputs
}

// SetMode sets how a method of core.Core answers.
func (f *FakeCore) SetMode(method string, mode Mode) {
	mustBeCoreMethod(method)
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.modes[method] = mode
}

// Calls answers how often a method of core.Core was called.
func (f *FakeCore) Calls(method string) int {
	mustBeCoreMethod(method)
	f.mutex.Lock()
	defer f.mutex.Unlock()
	return f.calls[method]
}

// Gate holds the calls of a method until the returned function opens the
// gate.
func (f *FakeCore) Gate(method string) (open func()) {
	mustBeCoreMethod(method)
	gate := make(chan struct{})
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.gates[method] = gate
	return sync.OnceFunc(func() { close(gate) })
}

// SetDelay makes every call wait before it answers.
func (f *FakeCore) SetDelay(delay time.Duration) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.delay = delay
}

// StopFeeds ends every subscription; later ones are ended at once, as the
// node's stopped feeds answer them.
func (f *FakeCore) StopFeeds() {
	f.heads.stop()
	f.transactions.stop()
}

func mustBeCoreMethod(method string) {
	if !coreMethods[method] {
		panic(fmt.Sprintf("%s is not a method of core.Core", method))
	}
}

// enter counts a call, holds it at its gate and for the delay, and answers
// its mode; in Panic mode it panics.
func (f *FakeCore) enter(method string) Mode {
	f.mutex.Lock()
	f.calls[method]++
	mode, gate, delay := f.modes[method], f.gates[method], f.delay
	f.mutex.Unlock()
	if gate != nil {
		<-gate
	}
	time.Sleep(delay)
	if mode == Panic {
		panic(fmt.Sprintf("injected panic in %s", method))
	}
	return mode
}

// lookup answers a method that can fail: in its mode, or from the fake's
// data under its lock.
func lookup[Value any](f *FakeCore, method string, answer func() (Value, error)) (Value, error) {
	var zero Value
	switch f.enter(method) {
	case Fail:
		return zero, ErrInjected
	case NotFound:
		return zero, core.ErrNotFound
	case NoArchive:
		return zero, core.ErrNoArchive
	case Nil:
		return zero, nil
	}
	f.mutex.Lock()
	defer f.mutex.Unlock()
	return answer()
}

// read answers a method that cannot fail: zero in a failing mode, or from
// the fake's data under its lock.
func read[Value any](f *FakeCore, method string, answer func() Value) Value {
	if f.enter(method) != Normal {
		var zero Value
		return zero
	}
	f.mutex.Lock()
	defer f.mutex.Unlock()
	return answer()
}

func (f *FakeCore) ChainInfo() core.ChainInfo {
	return read(f, "ChainInfo", func() core.ChainInfo {
		return core.ChainInfo{
			ChainID:        ChainID,
			GenesisID:      f.blocks[0].Hash,
			UpgradeHeights: slices.Clone(f.upgrades),
		}
	})
}

func (f *FakeCore) Header(number uint64) (*evmcore.EvmHeader, error) {
	return lookup(f, "Header", func() (*evmcore.EvmHeader, error) {
		block, err := f.storedBlock(number)
		if err != nil {
			return nil, err
		}
		return block.Header(), nil
	})
}

func (f *FakeCore) Block(number uint64) (*evmcore.EvmBlock, error) {
	return lookup(f, "Block", func() (*evmcore.EvmBlock, error) {
		block, err := f.storedBlock(number)
		if err != nil {
			return nil, err
		}
		return copyBlock(block), nil
	})
}

func (f *FakeCore) storedBlock(number uint64) (*evmcore.EvmBlock, error) {
	if number >= uint64(len(f.blocks)) {
		return nil, core.ErrNotFound
	}
	return f.blocks[number], nil
}

func (f *FakeCore) BlockNumberByHash(hash common.Hash) (uint64, error) {
	return lookup(f, "BlockNumberByHash", func() (uint64, error) { return f.blockNumberByHash(hash) })
}

// blockNumberByHash finds a stored block; the caller holds the lock.
func (f *FakeCore) blockNumberByHash(hash common.Hash) (uint64, error) {
	for _, block := range f.blocks {
		if block.Hash == hash {
			return block.NumberU64(), nil
		}
	}
	return 0, core.ErrNotFound
}

func (f *FakeCore) Receipts(number uint64) (types.Receipts, error) {
	return lookup(f, "Receipts", func() (types.Receipts, error) {
		if number >= uint64(len(f.receipts)) {
			return nil, core.ErrNotFound
		}
		return copyReceipts(f.receipts[number]), nil
	})
}

func (f *FakeCore) Rules(number uint64) (*opera.Rules, error) {
	return lookup(f, "Rules", func() (*opera.Rules, error) {
		rules := f.rules.Copy()
		return &rules, nil
	})
}

func (f *FakeCore) TransactionByHash(hash common.Hash) (*types.Transaction, uint64, uint64, error) {
	type position struct {
		transaction  *types.Transaction
		block, index uint64
	}
	found, err := lookup(f, "TransactionByHash", func() (position, error) {
		for number, block := range f.blocks[:f.archived+1] {
			for index, transaction := range block.Transactions {
				if transaction.Hash() == hash {
					return position{transaction, uint64(number), uint64(index)}, nil
				}
			}
		}
		return position{}, core.ErrNotFound
	})
	return found.transaction, found.block, found.index, err
}

func (f *FakeCore) BundleInfo(executionPlanHash common.Hash) (core.BundleInfo, error) {
	return lookup(f, "BundleInfo", func() (core.BundleInfo, error) {
		info, found := f.bundles[executionPlanHash]
		if !found {
			return core.BundleInfo{}, core.ErrNotFound
		}
		return info, nil
	})
}

func (f *FakeCore) Head() (*evmcore.EvmBlock, error) {
	return lookup(f, "Head", func() (*evmcore.EvmBlock, error) {
		return copyBlock(f.blocks[f.archived]), nil
	})
}

func (f *FakeCore) SubscribeHead(events chan<- core.HeadNotify) core.Subscription {
	return read(f, "SubscribeHead", func() core.Subscription { return f.heads.subscribe(events) })
}

func (f *FakeCore) State(number uint64) (state.StateDB, error) {
	return lookup(f, "State", func() (state.StateDB, error) {
		if f.openState == nil {
			return nil, errors.New("the fake core holds no state")
		}
		if number >= uint64(len(f.blocks)) {
			return nil, core.ErrNotFound
		}
		opened, err := f.openState()
		if err != nil {
			return nil, err
		}
		return f.track(opened), nil
	})
}

// SendTransaction puts a transaction into the pool and announces it on the
// pool's feed.
func (f *FakeCore) SendTransaction(ctx context.Context, transaction *types.Transaction) error {
	added := false
	_, err := lookup(f, "SendTransaction", func() (struct{}, error) {
		f.pool[transaction.Hash()] = transaction
		added = true
		return struct{}{}, nil
	})
	if added {
		f.transactions.send(evmcore.NewTxsNotify{Txs: []*types.Transaction{transaction}})
	}
	return err
}

func (f *FakeCore) PoolNonce(address common.Address) uint64 {
	return read(f, "PoolNonce", func() uint64 { return f.poolNonces[address] })
}

func (f *FakeCore) PoolTransaction(hash common.Hash) (*types.Transaction, error) {
	return lookup(f, "PoolTransaction", func() (*types.Transaction, error) {
		transaction, found := f.pool[hash]
		if !found {
			return nil, core.ErrNotFound
		}
		return transaction, nil
	})
}

func (f *FakeCore) PoolNextTransactions() types.Transactions {
	return read(f, "PoolNextTransactions", func() types.Transactions { return slices.Collect(maps.Values(f.pool)) })
}

func (f *FakeCore) PoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
	content := read(f, "PoolContent", func() [2]map[common.Address]types.Transactions {
		return [2]map[common.Address]types.Transactions{bySender(slices.Collect(maps.Values(f.pool))), bySender(f.queued)}
	})
	return content[0], content[1]
}

func (f *FakeCore) PoolContentFrom(address common.Address) (types.Transactions, types.Transactions) {
	content := read(f, "PoolContentFrom", func() [2]types.Transactions {
		return [2]types.Transactions{bySender(slices.Collect(maps.Values(f.pool)))[address], bySender(f.queued)[address]}
	})
	return content[0], content[1]
}

// bySender groups transactions by their senders, in the order of their
// nonces.
func bySender(transactions types.Transactions) map[common.Address]types.Transactions {
	signer := types.LatestSignerForChainID(big.NewInt(ChainID))
	grouped := map[common.Address]types.Transactions{}
	for _, transaction := range transactions {
		sender, err := types.Sender(signer, transaction)
		if err != nil {
			panic(err)
		}
		grouped[sender] = append(grouped[sender], transaction)
	}
	for _, group := range grouped {
		slices.SortFunc(group, func(a, b *types.Transaction) int { return cmp.Compare(a.Nonce(), b.Nonce()) })
	}
	return grouped
}

func (f *FakeCore) PoolStats() (int, int) {
	stats := read(f, "PoolStats", func() [2]int { return [2]int{len(f.pool), len(f.queued)} })
	return stats[0], stats[1]
}

func (f *FakeCore) SubscribeTransactions(events chan<- evmcore.NewTxsNotify) core.Subscription {
	return read(f, "SubscribeTransactions", func() core.Subscription { return f.transactions.subscribe(events) })
}

// Logs answers every log of the blocks a query names, the block of its hash
// or its range, a tag bound naming the query's head; it leaves the criteria
// to the node's query code, which it does not copy. LogQueries tells the
// queries it was asked.
func (f *FakeCore) Logs(ctx context.Context, query core.LogQuery) ([]*types.Log, error) {
	return lookup(f, "Logs", func() ([]*types.Log, error) {
		f.logQueries = append(f.logQueries, query)
		bound := func(number int64) int64 {
			if number < 0 {
				return int64(query.Head)
			}
			return number
		}
		begin, end := bound(query.Begin), min(bound(query.End), int64(len(f.blocks)-1))
		if query.BlockHash != nil {
			number, err := f.blockNumberByHash(*query.BlockHash)
			if err != nil {
				return nil, err
			}
			begin, end = int64(number), int64(number)
		}
		var logs []*types.Log
		for number := begin; number <= end; number++ {
			for _, receipt := range f.receipts[number] {
				logs = append(logs, receipt.Logs...)
			}
		}
		return logs, nil
	})
}

// LogQueries are the queries Logs was asked, in order.
func (f *FakeCore) LogQueries() []core.LogQuery {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	return slices.Clone(f.logQueries)
}

func (f *FakeCore) Progress() core.Progress {
	return read(f, "Progress", func() core.Progress { return f.progress })
}

func (f *FakeCore) OracleInputs() core.OracleInputs {
	return read(f, "OracleInputs", func() core.OracleInputs {
		if f.oracleInputs != nil {
			return *f.oracleInputs
		}
		return core.OracleInputs{NextEpochRules: f.rules.Copy(), MinTip: new(big.Int)}
	})
}

func (f *FakeCore) PeerCount() int {
	return read(f, "PeerCount", func() int { return f.peers })
}

func (f *FakeCore) EpochOf(number uint64) (uint64, error) {
	return lookup(f, "EpochOf", func() (uint64, error) {
		block, err := f.storedBlock(min(number, uint64(len(f.blocks)-1)))
		if err != nil {
			return 0, err
		}
		return uint64(block.Epoch), nil
	})
}

func (f *FakeCore) EpochState(epoch uint64) (core.EpochSnapshot, error) {
	return lookup(f, "EpochState", func() (core.EpochSnapshot, error) {
		states, found := f.epochStates[epoch]
		if !found {
			return core.EpochSnapshot{}, core.ErrNotFound
		}
		return states, nil
	})
}

func (f *FakeCore) LiveEpochState() core.EpochSnapshot {
	return read(f, "LiveEpochState", func() core.EpochSnapshot { return f.liveEpoch })
}

func (f *FakeCore) LiveEpochHeads(epoch uint64) ([]common.Hash, error) {
	return lookup(f, "LiveEpochHeads", func() ([]common.Hash, error) {
		heads, found := f.epochHeads[epoch]
		if !found {
			return nil, core.ErrNotFound
		}
		return slices.Clone(heads), nil
	})
}

// FindEvents answers the events whose ids start with the epoch, the lamport
// and the prefix, as the node's store finds them.
func (f *FakeCore) FindEvents(epoch uint64, lamport uint32, prefix []byte) []common.Hash {
	return read(f, "FindEvents", func() []common.Hash {
		start := append(append(idx.Epoch(epoch).Bytes(), idx.Lamport(lamport).Bytes()...), prefix...)
		var found []common.Hash
		for id := range f.events {
			if bytes.HasPrefix(id.Bytes(), start) {
				found = append(found, id)
			}
		}
		return found
	})
}

func (f *FakeCore) Event(id common.Hash) (*inter.Event, error) {
	return lookup(f, "Event", func() (*inter.Event, error) {
		event, found := f.events[id]
		if !found {
			return nil, core.ErrNotFound
		}
		return &event.Event, nil
	})
}

func (f *FakeCore) EventPayload(id common.Hash) (*inter.EventPayload, error) {
	return lookup(f, "EventPayload", func() (*inter.EventPayload, error) {
		event, found := f.events[id]
		if !found {
			return nil, core.ErrNotFound
		}
		return event, nil
	})
}

func (f *FakeCore) SendTransactions(transactions types.Transactions) error {
	_, err := lookup(f, "SendTransactions", func() (struct{}, error) { return struct{}{}, nil })
	return err
}

func (f *FakeCore) EmitTransactions(transactions types.Transactions) error {
	_, err := lookup(f, "EmitTransactions", func() (struct{}, error) { return struct{}{}, nil })
	return err
}

func copyBlock(block *evmcore.EvmBlock) *evmcore.EvmBlock {
	copied := *block
	copied.Number = new(big.Int).Set(block.Number)
	copied.BaseFee = new(big.Int).Set(block.BaseFee)
	copied.Transactions = slices.Clone(block.Transactions)
	return &copied
}

func copyReceipts(receipts types.Receipts) types.Receipts {
	copied := make(types.Receipts, len(receipts))
	for i, receipt := range receipts {
		receiptCopy := *receipt
		receiptCopy.Logs = make([]*types.Log, len(receipt.Logs))
		for j, log := range receipt.Logs {
			logCopy := *log
			receiptCopy.Logs[j] = &logCopy
		}
		copied[i] = &receiptCopy
	}
	return copied
}
