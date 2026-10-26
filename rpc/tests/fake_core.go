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

// Package tests holds the test support of the RPC module: an in-memory Core
// with failure modes, a Core over real Carmen state, accounts, contract code
// and the harness that calls namespaces the way a transport does.
package tests

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"slices"
	"sort"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/rpc/core"
)

// ChainID is the chain id of the fake chain.
const ChainID = 4003

// Mode is how a Core method of the fake answers.
type Mode int

// The modes of a Core method.
const (
	Normal   Mode = iota // the fake's content
	Panic                // a panic
	Fail                 // ErrFake, or the zero value for a method without error
	NotFound             // core.ErrNotFound, or the zero value
	Nil                  // a nil value without error, which Core never answers
)

// Modes are every mode but Normal, for matrices over failures.
var Modes = []Mode{Panic, Fail, NotFound, Nil}

func (m Mode) String() string {
	return [...]string{"normal", "panic", "fail", "not-found", "nil"}[m]
}

// ErrFake is the failure of a method in mode Fail.
var ErrFake = errors.New("fake core failure")

// CoreMethods are the names of the methods of core.Core, derived from the
// interface.
func CoreMethods() []string {
	t := reflect.TypeFor[core.Core]()
	names := make([]string, t.NumMethod())
	for i := range names {
		names[i] = t.Method(i).Name
	}
	return names
}

// FakeCore is an in-memory core.Core: a chain of blocks with receipts, a pool,
// and feeds of heads and pool transactions. Per method it has a mode, a call
// count and a gate that holds its calls; one delay applies to every call. It
// hands out copies, and a test changes it only through its methods.
type FakeCore struct {
	mu sync.Mutex

	blocks    []*evmcore.EvmBlock
	receipts  map[uint64]types.Receipts
	txs       map[common.Hash][2]uint64 // block, index
	head      uint64
	rules     opera.Rules
	openRules opera.Rules
	upgrades  []opera.UpgradeHeight
	root      *common.Hash
	missing   map[uint64]bool
	bundles   map[common.Hash]core.BundleInfo
	txIndex   bool

	pending  map[common.Address]types.Transactions
	queued   map[common.Address]types.Transactions
	nonces   map[common.Address]uint64
	sent     []*types.Transaction
	sendErr  error
	progress core.Progress
	inputs   core.OracleInputs

	state func(number uint64) (state.StateDB, error)

	headSubs []*fakeSub[core.HeadEvent]
	txSubs   []*fakeSub[evmcore.NewTxsNotify]

	modes map[string]Mode
	calls map[string]int
	gates map[string]chan struct{}
	delay time.Duration
}

// NewFakeCore creates a chain with a genesis block, the rules of the latest
// upgrades, and an empty pool. It has no state: State fails.
func NewFakeCore() *FakeCore {
	rules := opera.FakeNetRules(opera.GetBrioUpgrades())
	rules.NetworkID = ChainID
	f := &FakeCore{
		receipts:  make(map[uint64]types.Receipts),
		txs:       make(map[common.Hash][2]uint64),
		rules:     rules,
		openRules: rules,
		upgrades:  []opera.UpgradeHeight{{Upgrades: rules.Upgrades, Height: 0}},
		missing:   make(map[uint64]bool),
		bundles:   make(map[common.Hash]core.BundleInfo),
		txIndex:   true,
		pending:   make(map[common.Address]types.Transactions),
		queued:    make(map[common.Address]types.Transactions),
		nonces:    make(map[common.Address]uint64),
		inputs:    core.OracleInputs{PendingRules: rules, MinTip: new(big.Int)},
		modes:     make(map[string]Mode),
		calls:     make(map[string]int),
		gates:     make(map[string]chan struct{}),
	}
	f.state = func(uint64) (state.StateDB, error) { return nil, errors.New("the fake core has no state") }
	f.appendBlock(nil, nil)
	return f
}

// --- configuration by the test ---

func (f *FakeCore) checkName(name string) {
	if !slices.Contains(CoreMethods(), name) {
		panic(fmt.Sprintf("core.Core has no method %q", name))
	}
}

// SetMode sets how a Core method answers; a name Core does not have panics.
func (f *FakeCore) SetMode(name string, mode Mode) {
	f.checkName(name)
	f.mu.Lock()
	defer f.mu.Unlock()
	f.modes[name] = mode
}

// Calls counts the calls of a Core method; a name Core does not have panics.
func (f *FakeCore) Calls(name string) int {
	f.checkName(name)
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls[name]
}

// Gate holds the calls of a Core method until the returned function is called.
func (f *FakeCore) Gate(name string) (release func()) {
	f.checkName(name)
	gate := make(chan struct{})
	f.mu.Lock()
	f.gates[name] = gate
	f.mu.Unlock()
	var once sync.Once
	return func() {
		once.Do(func() {
			f.mu.Lock()
			delete(f.gates, name)
			f.mu.Unlock()
			close(gate)
		})
	}
}

// SetDelay delays every call of a Core method.
func (f *FakeCore) SetDelay(d time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.delay = d
}

// SetTxIndex switches the transaction index of the node on or off.
func (f *FakeCore) SetTxIndex(on bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.txIndex = on
}

// SetRules sets the rules of every block and of the open epoch.
func (f *FakeCore) SetRules(rules opera.Rules) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rules = rules
	f.openRules = rules
	f.upgrades = []opera.UpgradeHeight{{Upgrades: rules.Upgrades, Height: 0}}
}

// SetUpgradeHeights sets the heights at which upgrades came into force. The
// module reads them with a new head.
func (f *FakeCore) SetUpgradeHeights(heights []opera.UpgradeHeight) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.upgrades = slices.Clone(heights)
}

// SetOpenRules sets the rules of the open epoch only, the rules of the block
// after the head.
func (f *FakeCore) SetOpenRules(rules opera.Rules) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.openRules = rules
}

// SetRoot sets the state root of every block, as a state the blocks share
// has it.
func (f *FakeCore) SetRoot(root common.Hash) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.root = &root
	for _, b := range f.blocks {
		b.Root = root
	}
}

// SetMissing makes a block of the chain unreadable, as a block the node lost:
// its header, block and receipts are not found.
func (f *FakeCore) SetMissing(number uint64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.missing[number] = true
}

// SetState sets how State opens the archive of a block.
func (f *FakeCore) SetState(open func(number uint64) (state.StateDB, error)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.state = open
}

// SetBundle records the execution of a bundle.
func (f *FakeCore) SetBundle(planHash common.Hash, info core.BundleInfo) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.bundles[planHash] = info
}

// SetProgress sets the synchronization status.
func (f *FakeCore) SetProgress(p core.Progress) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.progress = p
}

// SetOracleInputs sets the gas power and pending rules the oracle weighs.
func (f *FakeCore) SetOracleInputs(inputs core.OracleInputs) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.inputs = inputs
}

// SetPool sets the pending and queued transactions of the pool.
func (f *FakeCore) SetPool(pending, queued map[common.Address]types.Transactions) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.pending = maps(pending)
	f.queued = maps(queued)
}

func maps(m map[common.Address]types.Transactions) map[common.Address]types.Transactions {
	out := make(map[common.Address]types.Transactions, len(m))
	for k, v := range m {
		out[k] = slices.Clone(v)
	}
	return out
}

// SetNonce sets the pool nonce of an account.
func (f *FakeCore) SetNonce(addr common.Address, nonce uint64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nonces[addr] = nonce
}

// SetSendError makes SendTx refuse every transaction with err.
func (f *FakeCore) SetSendError(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sendErr = err
}

// Sent returns the transactions SendTx accepted.
func (f *FakeCore) Sent() []*types.Transaction {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.sent)
}

// AddBlock appends a block with transactions that all succeed and emit the
// given logs, one list per transaction, and returns it; the head does not
// move.
func (f *FakeCore) AddBlock(txs types.Transactions, logs ...[]*types.Log) *evmcore.EvmBlock {
	f.mu.Lock()
	defer f.mu.Unlock()
	return copyBlock(f.appendBlock(txs, logs))
}

func blockHash(number uint64) common.Hash {
	return crypto.Keccak256Hash(binary.BigEndian.AppendUint64([]byte("fake block"), number))
}

// appendBlock appends a block; the caller holds the lock.
func (f *FakeCore) appendBlock(txs types.Transactions, logs [][]*types.Log) *evmcore.EvmBlock {
	number := uint64(len(f.blocks))
	var parent common.Hash
	if number > 0 {
		parent = f.blocks[number-1].Hash
	}
	header := &evmcore.EvmHeader{
		Number:          new(big.Int).SetUint64(number),
		Hash:            blockHash(number),
		ParentHash:      parent,
		Root:            f.blockRoot(number),
		Time:            inter.FromUnix(int64(1_700_000_000 + number)),
		GasLimit:        f.rules.Blocks.MaxBlockGas,
		BaseFee:         new(big.Int).Set(f.rules.Economy.MinGasPrice),
		WithdrawalsHash: &types.EmptyWithdrawalsHash,
		PrevRandao:      common.BigToHash(new(big.Int).SetUint64(number + 1)),
		Epoch:           1,
	}
	block := evmcore.NewEvmBlock(header, slices.Clone(txs))
	receipts := make(types.Receipts, len(txs))
	var cumulative uint64
	logIndex := uint(0)
	for i, tx := range txs {
		cumulative += tx.Gas()
		receipt := &types.Receipt{
			Type:              tx.Type(),
			Status:            types.ReceiptStatusSuccessful,
			CumulativeGasUsed: cumulative,
			TxHash:            tx.Hash(),
			GasUsed:           tx.Gas(),
			BlockHash:         block.Hash,
			BlockNumber:       new(big.Int).SetUint64(number),
			TransactionIndex:  uint(i),
			Logs:              []*types.Log{},
		}
		if i < len(logs) {
			for _, l := range logs[i] {
				c := *l
				c.BlockNumber = number
				c.BlockHash = block.Hash
				c.BlockTimestamp = uint64(header.Time.Unix())
				c.TxHash = tx.Hash()
				c.TxIndex = uint(i)
				c.Index = logIndex
				logIndex++
				receipt.Logs = append(receipt.Logs, &c)
			}
		}
		receipt.Bloom = types.CreateBloom(receipt)
		receipts[i] = receipt
		f.txs[tx.Hash()] = [2]uint64{number, uint64(i)}
	}
	f.blocks = append(f.blocks, block)
	f.receipts[number] = receipts
	return block
}

func (f *FakeCore) blockRoot(number uint64) common.Hash {
	if f.root != nil {
		return *f.root
	}
	return crypto.Keccak256Hash([]byte("fake root"), binary.BigEndian.AppendUint64(nil, number))
}

// Publish moves the head to a block and sends the head event of every block
// up to it to the subscribers, as the node's feed does.
func (f *FakeCore) Publish(number uint64) {
	f.mu.Lock()
	var events []core.HeadEvent
	for n := f.head + 1; n <= number && n < uint64(len(f.blocks)); n++ {
		var logs []*types.Log
		for _, r := range f.receipts[n] {
			logs = append(logs, r.Logs...)
		}
		events = append(events, core.HeadEvent{Block: copyBlock(f.blocks[n]), Logs: logs})
	}
	if number > f.head {
		f.head = min(number, uint64(len(f.blocks)-1))
	}
	subs := slices.Clone(f.headSubs)
	f.mu.Unlock()
	for _, ev := range events {
		for _, sub := range subs {
			sub.send(ev)
		}
	}
}

// AddAndPublish appends a block and publishes it.
func (f *FakeCore) AddAndPublish(txs types.Transactions, logs ...[]*types.Log) *evmcore.EvmBlock {
	block := f.AddBlock(txs, logs...)
	f.Publish(block.NumberU64())
	return block
}

// EmitHead sends a head event to the subscribers as it is, without changing
// the head: a repeated or stale event, as a feed may deliver.
func (f *FakeCore) EmitHead(ev core.HeadEvent) {
	f.mu.Lock()
	subs := slices.Clone(f.headSubs)
	f.mu.Unlock()
	for _, sub := range subs {
		sub.send(ev)
	}
}

// EmitTxs sends transactions entering the pool to the subscribers.
func (f *FakeCore) EmitTxs(txs types.Transactions) {
	f.mu.Lock()
	subs := slices.Clone(f.txSubs)
	f.mu.Unlock()
	for _, sub := range subs {
		sub.send(evmcore.NewTxsNotify{Txs: txs})
	}
}

// EndFeeds ends every subscription, as a stopping node does.
func (f *FakeCore) EndFeeds() {
	f.mu.Lock()
	heads, txs := f.headSubs, f.txSubs
	f.headSubs, f.txSubs = nil, nil
	f.mu.Unlock()
	for _, s := range heads {
		s.Unsubscribe()
	}
	for _, s := range txs {
		s.Unsubscribe()
	}
}

// HeadSubscribers counts the live subscriptions of the head feed.
func (f *FakeCore) HeadSubscribers() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, s := range f.headSubs {
		if !s.ended() {
			n++
		}
	}
	return n
}

func copyBlock(b *evmcore.EvmBlock) *evmcore.EvmBlock {
	c := *b
	c.Number = new(big.Int).Set(b.Number)
	if b.BaseFee != nil {
		c.BaseFee = new(big.Int).Set(b.BaseFee)
	}
	c.Transactions = slices.Clone(b.Transactions)
	return &c
}

// --- core.Core ---

// enter applies the delay, the gate and the mode of a method; it reports a
// mode other than Normal.
func (f *FakeCore) enter(name string) Mode {
	f.mu.Lock()
	f.calls[name]++
	mode, delay, gate := f.modes[name], f.delay, f.gates[name]
	f.mu.Unlock()
	if delay > 0 {
		time.Sleep(delay)
	}
	if gate != nil {
		<-gate
	}
	if mode == Panic {
		panic(fmt.Sprintf("fake core: %s panics", name))
	}
	return mode
}

// answer runs a lookup in its mode.
func answer[T any](f *FakeCore, name string, lookup func() (T, error)) (T, error) {
	var zero T
	switch f.enter(name) {
	case Fail:
		return zero, ErrFake
	case NotFound:
		return zero, core.ErrNotFound
	case Nil:
		return zero, nil
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return lookup()
}

// value runs a method without error in its mode: any failure is the zero
// value.
func value[T any](f *FakeCore, name string, get func() T) T {
	if f.enter(name) != Normal {
		var zero T
		return zero
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return get()
}

func (f *FakeCore) ChainInfo() core.ChainInfo {
	return value(f, "ChainInfo", func() core.ChainInfo {
		return core.ChainInfo{
			ChainID:        big.NewInt(ChainID),
			NetworkID:      ChainID,
			GenesisID:      common.HexToHash("0x6e57"),
			UpgradeHeights: slices.Clone(f.upgrades),
		}
	})
}

func (f *FakeCore) Header(number uint64) (*evmcore.EvmHeader, error) {
	return answer(f, "Header", func() (*evmcore.EvmHeader, error) {
		if number >= uint64(len(f.blocks)) || f.missing[number] {
			return nil, core.ErrNotFound
		}
		return f.blocks[number].Header(), nil
	})
}

func (f *FakeCore) Block(number uint64) (*evmcore.EvmBlock, error) {
	return answer(f, "Block", func() (*evmcore.EvmBlock, error) {
		if number >= uint64(len(f.blocks)) || f.missing[number] {
			return nil, core.ErrNotFound
		}
		return copyBlock(f.blocks[number]), nil
	})
}

func (f *FakeCore) BlockNumberByHash(hash common.Hash) (uint64, error) {
	return answer(f, "BlockNumberByHash", func() (uint64, error) {
		for _, b := range f.blocks {
			if b.Hash == hash {
				return b.NumberU64(), nil
			}
		}
		return 0, core.ErrNotFound
	})
}

func (f *FakeCore) Receipts(number uint64) (types.Receipts, error) {
	return answer(f, "Receipts", func() (types.Receipts, error) {
		if !f.txIndex {
			return nil, errors.New("transactions index is disabled (enable TxIndex and re-process the DAGs)")
		}
		receipts, ok := f.receipts[number]
		if !ok || f.missing[number] {
			return nil, core.ErrNotFound
		}
		return slices.Clone(receipts), nil
	})
}

func (f *FakeCore) Rules(number uint64) (*opera.Rules, error) {
	return answer(f, "Rules", func() (*opera.Rules, error) {
		latest := uint64(len(f.blocks) - 1)
		switch {
		case number == latest+1:
			rules := f.openRules.Copy()
			return &rules, nil
		case number > latest:
			return nil, core.ErrNotFound
		}
		rules := f.rules.Copy()
		return &rules, nil
	})
}

func (f *FakeCore) TxByHash(hash common.Hash) (*types.Transaction, uint64, uint64, error) {
	type position struct {
		tx           *types.Transaction
		block, index uint64
	}
	p, err := answer(f, "TxByHash", func() (*position, error) {
		if !f.txIndex {
			return nil, errors.New("transactions index is disabled (enable TxIndex and re-process the DAG)")
		}
		pos, ok := f.txs[hash]
		if !ok {
			return nil, core.ErrNotFound
		}
		return &position{tx: f.blocks[pos[0]].Transactions[pos[1]], block: pos[0], index: pos[1]}, nil
	})
	if err != nil || p == nil {
		return nil, 0, 0, err
	}
	return p.tx, p.block, p.index, nil
}

func (f *FakeCore) BundleInfo(planHash common.Hash) (core.BundleInfo, error) {
	return answer(f, "BundleInfo", func() (core.BundleInfo, error) {
		info, ok := f.bundles[planHash]
		if !ok {
			return core.BundleInfo{}, core.ErrNotFound
		}
		return info, nil
	})
}

func (f *FakeCore) Head() (*evmcore.EvmBlock, error) {
	return answer(f, "Head", func() (*evmcore.EvmBlock, error) {
		return copyBlock(f.blocks[f.head]), nil
	})
}

func (f *FakeCore) SubscribeHead(ch chan<- core.HeadEvent) core.Subscription {
	if f.enter("SubscribeHead") != Normal {
		return endedSub()
	}
	sub := newFakeSub(ch)
	f.mu.Lock()
	defer f.mu.Unlock()
	f.headSubs = append(f.headSubs, sub)
	return sub
}

func (f *FakeCore) State(number uint64) (state.StateDB, error) {
	return answer(f, "State", func() (state.StateDB, error) {
		if number >= uint64(len(f.blocks)) {
			return nil, core.ErrNotFound
		}
		return f.state(number)
	})
}

func (f *FakeCore) SendTx(_ context.Context, tx *types.Transaction) error {
	_, err := answer(f, "SendTx", func() (struct{}, error) {
		if f.sendErr != nil {
			return struct{}{}, f.sendErr
		}
		f.sent = append(f.sent, tx)
		return struct{}{}, nil
	})
	return err
}

func (f *FakeCore) PoolNonce(addr common.Address) uint64 {
	return value(f, "PoolNonce", func() uint64 { return f.nonces[addr] })
}

func (f *FakeCore) PoolTransaction(hash common.Hash) (*types.Transaction, error) {
	return answer(f, "PoolTransaction", func() (*types.Transaction, error) {
		for _, list := range []map[common.Address]types.Transactions{f.pending, f.queued} {
			for _, txs := range list {
				for _, tx := range txs {
					if tx.Hash() == hash {
						return tx, nil
					}
				}
			}
		}
		return nil, core.ErrNotFound
	})
}

func (f *FakeCore) PoolTransactions() (types.Transactions, error) {
	return answer(f, "PoolTransactions", func() (types.Transactions, error) {
		var all types.Transactions
		for _, addr := range sortedAccounts(f.pending) {
			all = append(all, f.pending[addr]...)
		}
		return all, nil
	})
}

func sortedAccounts(m map[common.Address]types.Transactions) []common.Address {
	addrs := make([]common.Address, 0, len(m))
	for a := range m {
		addrs = append(addrs, a)
	}
	sort.Slice(addrs, func(i, j int) bool { return addrs[i].Cmp(addrs[j]) < 0 })
	return addrs
}

func (f *FakeCore) PoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
	type content struct {
		pending, queued map[common.Address]types.Transactions
	}
	c := value(f, "PoolContent", func() content { return content{maps(f.pending), maps(f.queued)} })
	return c.pending, c.queued
}

func (f *FakeCore) PoolContentFrom(addr common.Address) (types.Transactions, types.Transactions) {
	type content struct{ pending, queued types.Transactions }
	c := value(f, "PoolContentFrom", func() content {
		return content{slices.Clone(f.pending[addr]), slices.Clone(f.queued[addr])}
	})
	return c.pending, c.queued
}

func (f *FakeCore) PoolStats() (int, int) {
	type stats struct{ pending, queued int }
	count := func(m map[common.Address]types.Transactions) int {
		n := 0
		for _, txs := range m {
			n += len(txs)
		}
		return n
	}
	s := value(f, "PoolStats", func() stats { return stats{count(f.pending), count(f.queued)} })
	return s.pending, s.queued
}

func (f *FakeCore) SubscribeTxs(ch chan<- evmcore.NewTxsNotify) core.Subscription {
	if f.enter("SubscribeTxs") != Normal {
		return endedSub()
	}
	sub := newFakeSub(ch)
	f.mu.Lock()
	defer f.mu.Unlock()
	f.txSubs = append(f.txSubs, sub)
	return sub
}

// FindLogs scans the receipts of the range, as the node's index answers: the
// first element of the pattern holds addresses as hashes, the others topics.
func (f *FakeCore) FindLogs(ctx context.Context, from, to uint64, pattern [][]common.Hash, limit uint) ([]*types.Log, error) {
	return answer(f, "FindLogs", func() ([]*types.Log, error) {
		var found []*types.Log
		for n := from; n <= to && n < uint64(len(f.blocks)); n++ {
			for _, r := range f.receipts[n] {
				for _, l := range r.Logs {
					if matches(l, pattern) {
						c := *l
						found = append(found, &c)
					}
				}
			}
			if limit > 0 && uint(len(found)) > limit {
				return nil, fmt.Errorf("too many results, consider narrowing your query criteria, the limit is %d", limit)
			}
		}
		return found, nil
	})
}

func matches(l *types.Log, pattern [][]common.Hash) bool {
	for i, alternatives := range pattern {
		if len(alternatives) == 0 {
			continue
		}
		var value common.Hash
		if i == 0 {
			value = common.BytesToHash(l.Address[:])
		} else {
			if i-1 >= len(l.Topics) {
				return false
			}
			value = l.Topics[i-1]
		}
		if !slices.Contains(alternatives, value) {
			return false
		}
	}
	return true
}

func (f *FakeCore) Progress() core.Progress {
	return value(f, "Progress", func() core.Progress {
		p := f.progress
		if p.CurrentBlock == 0 {
			p.CurrentBlock = uint64(len(f.blocks) - 1)
			p.CurrentBlockTime = f.blocks[len(f.blocks)-1].Time
		}
		return p
	})
}

func (f *FakeCore) OracleInputs() core.OracleInputs {
	return value(f, "OracleInputs", func() core.OracleInputs {
		in := f.inputs
		if in.MinTip != nil {
			in.MinTip = new(big.Int).Set(in.MinTip)
		}
		return in
	})
}

func (f *FakeCore) PendingHeads(max int) []core.PendingHead {
	return value(f, "PendingHeads", func() []core.PendingHead {
		var heads []core.PendingHead
		for _, addr := range sortedAccounts(f.pending) {
			txs := f.pending[addr]
			if len(heads) >= max || len(txs) == 0 {
				continue
			}
			heads = append(heads, core.PendingHead{Gas: txs[0].Gas(), GasTipCap: txs[0].GasTipCap(), GasFeeCap: txs[0].GasFeeCap()})
		}
		return heads
	})
}

// fakeSub is a subscription of a fake feed; sending waits for the
// subscriber, as go-ethereum's feed does.
type fakeSub[T any] struct {
	ch   chan<- T
	err  chan error
	done chan struct{}
	once sync.Once
}

func newFakeSub[T any](ch chan<- T) *fakeSub[T] {
	return &fakeSub[T]{ch: ch, err: make(chan error, 1), done: make(chan struct{})}
}

func (s *fakeSub[T]) send(v T) {
	select {
	case s.ch <- v:
	case <-s.done:
	}
}

func (s *fakeSub[T]) Err() <-chan error { return s.err }

func (s *fakeSub[T]) Unsubscribe() {
	s.once.Do(func() {
		close(s.done)
		close(s.err)
	})
}

func (s *fakeSub[T]) ended() bool {
	select {
	case <-s.done:
		return true
	default:
		return false
	}
}

type ended struct{ err chan error }

func (e ended) Err() <-chan error { return e.err }
func (e ended) Unsubscribe()      {}

func endedSub() core.Subscription {
	ch := make(chan error)
	close(ch)
	return ended{err: ch}
}
