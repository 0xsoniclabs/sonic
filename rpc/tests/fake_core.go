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

// Package tests holds the support code of the module's tests: FakeCore, an
// in-memory core.Core with a builder for blocks and transactions and failure
// injection per method; NewCarmenCore, a FakeCore over a real in-memory
// Carmen state; and a harness that calls methods through the dispatcher. It is
// the successor of api/rpctest.
package tests

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"sync/atomic"
	"testing"
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

// Mode is how an injected Core method behaves.
type Mode int

const (
	// Normal serves the call.
	Normal Mode = iota
	// Panic panics.
	Panic
	// Fail returns an error, or the zero result for methods without error.
	Fail
	// NotFound returns core.ErrNotFound, or "not found" for methods that have
	// no error.
	NotFound
	// Nil returns nil and zero values without error.
	Nil
)

// ErrInjected is the error of the Fail mode.
var ErrInjected = errors.New("injected failure")

// CoreMethods are the names of the methods of core.Core.
var CoreMethods = []string{
	"ChainInfo", "Header", "Block", "BlockNumberByHash", "Receipts", "Rules", "TxByHash", "BundleInfo",
	"Head", "SubscribeHead", "State",
	"SendTx", "PoolNonce", "PoolTransaction", "PoolTransactions", "PoolContent", "PoolContentFrom",
	"PoolStats", "SubscribeTxs", "FindLogs",
	"Progress", "OracleInputs", "PendingHeads",
}

type txPos struct {
	block, index uint64
}

// FakeCore is an in-memory core.Core. It is safe for concurrent use.
type FakeCore struct {
	t testing.TB

	mu        sync.RWMutex
	info      core.ChainInfo
	blocks    []*evmcore.EvmBlock
	receipts  []types.Receipts
	byHash    map[common.Hash]uint64
	txs       map[common.Hash]txPos
	rules     func(uint64) opera.Rules
	pool      map[common.Hash]*types.Transaction
	poolOrder []common.Hash
	nonces    map[common.Address]uint64
	bundles   map[common.Hash]core.BundleInfo
	progress  core.Progress
	oracle    core.OracleInputs
	stateFn   func(uint64) (state.StateDB, error)
	findLogs  func(from, to uint64, pattern [][]common.Hash, limit uint) ([]*types.Log, error)
	sendTx    func(*types.Transaction) error

	headerHook func(uint64) error
	hidden     bool
	hiddenFrom uint64

	headSubs []*feedSub[core.HeadEvent]
	txSubs   []*feedSub[evmcore.NewTxsNotify]

	inject  map[string]Mode
	delay   time.Duration
	calls   map[string]int
	gates   map[string]chan struct{}
	waiting map[string]int
}

var _ core.Core = (*FakeCore)(nil)

// Option configures a FakeCore.
type Option func(*FakeCore)

// WithChainID sets the chain id (default opera.FakeNetworkID).
func WithChainID(id uint64) Option {
	return func(f *FakeCore) { f.info.ChainID, f.info.NetworkID = id, id }
}

// WithRules sets the rules in force at each block (default: fake network rules
// with the Brio upgrades).
func WithRules(fn func(block uint64) opera.Rules) Option {
	return func(f *FakeCore) { f.rules = fn }
}

// WithUpgradeHeights sets the upgrade heights reported by ChainInfo.
func WithUpgradeHeights(heights ...opera.UpgradeHeight) Option {
	return func(f *FakeCore) { f.info.UpgradeHeights = heights }
}

// WithBlocks appends that many empty blocks after the genesis block.
func WithBlocks(n int) Option {
	return func(f *FakeCore) {
		for range n {
			f.addBlock(nil)
		}
	}
}

// NewFakeCore creates a core with a genesis block (number 0).
func NewFakeCore(t testing.TB, opts ...Option) *FakeCore {
	f := &FakeCore{
		t:       t,
		byHash:  map[common.Hash]uint64{},
		txs:     map[common.Hash]txPos{},
		pool:    map[common.Hash]*types.Transaction{},
		nonces:  map[common.Address]uint64{},
		bundles: map[common.Hash]core.BundleInfo{},
		inject:  map[string]Mode{},
		calls:   map[string]int{},
		gates:   map[string]chan struct{}{},
		waiting: map[string]int{},
		info: core.ChainInfo{
			ChainID:        opera.FakeNetworkID,
			NetworkID:      opera.FakeNetworkID,
			GenesisID:      common.HexToHash("0x9e"),
			UpgradeHeights: []opera.UpgradeHeight{opera.MakeUpgradeHeight(opera.GetBrioUpgrades(), 0)},
		},
		oracle: core.OracleInputs{
			PendingRules: opera.FakeNetRules(opera.GetBrioUpgrades()),
			Rules:        opera.FakeNetRules(opera.GetBrioUpgrades()),
			MinTip:       new(big.Int),
		},
	}
	f.rules = func(uint64) opera.Rules { return opera.FakeNetRules(opera.GetBrioUpgrades()) }
	// All the gas power of the validators is unused.
	long := f.oracle.Rules.Economy.LongGasPower
	f.oracle.TotalGasPowerLeft = new(big.Int).Div(
		new(big.Int).Mul(new(big.Int).SetUint64(long.AllocPerSec), new(big.Int).SetUint64(uint64(long.MaxAllocPeriod))),
		big.NewInt(int64(time.Second))).Uint64()
	f.addBlock(nil) // genesis
	for _, opt := range opts {
		opt(f)
	}
	return f
}

// Close releases resources of the core; it is registered with the test.
func (f *FakeCore) Close() {}

// -- building the chain --

// AddBlock appends a block with the transactions, with a successful receipt
// for each, and announces it on the head feed.
func (f *FakeCore) AddBlock(txs ...*types.Transaction) *evmcore.EvmBlock {
	f.mu.Lock()
	block := f.addBlock(txs)
	subs := slicesCloneSubs(f.headSubs)
	logs := collectLogs(f.receipts[len(f.receipts)-1])
	f.mu.Unlock()
	ev := core.HeadEvent{Block: block, Logs: logs}
	for _, sub := range subs {
		sub.send(ev)
	}
	return block
}

// Publish announces an event on the head feed.
func (f *FakeCore) Publish(ev core.HeadEvent) {
	f.mu.Lock()
	subs := slicesCloneSubs(f.headSubs)
	f.mu.Unlock()
	for _, sub := range subs {
		sub.send(ev)
	}
}

// EndHeadSubscriptions ends every head subscription, as a stopped feed does.
func (f *FakeCore) EndHeadSubscriptions() {
	f.mu.Lock()
	subs := slicesCloneSubs(f.headSubs)
	f.mu.Unlock()
	for _, sub := range subs {
		sub.Unsubscribe()
	}
}

// AddBlockSilently appends a block without announcing it on the head feed.
func (f *FakeCore) AddBlockSilently(txs ...*types.Transaction) *evmcore.EvmBlock {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.addBlock(txs)
}

func collectLogs(receipts types.Receipts) []*types.Log {
	var logs []*types.Log
	for _, r := range receipts {
		logs = append(logs, r.Logs...)
	}
	return logs
}

func slicesCloneSubs[T any](in []*feedSub[T]) []*feedSub[T] {
	return append([]*feedSub[T](nil), in...)
}

func (f *FakeCore) addBlock(txs []*types.Transaction) *evmcore.EvmBlock {
	number := uint64(len(f.blocks))
	var parent common.Hash
	var parentTime inter.Timestamp
	if number > 0 {
		parent = f.blocks[number-1].Hash
		parentTime = f.blocks[number-1].Time
	}
	header := &evmcore.EvmHeader{
		Number:     new(big.Int).SetUint64(number),
		Hash:       crypto.Keccak256Hash([]byte(fmt.Sprintf("fake block %d", number))),
		ParentHash: parent,
		Root:       crypto.Keccak256Hash([]byte(fmt.Sprintf("fake root %d", number))),
		Time:       parentTime + inter.Timestamp(time.Second),
		Duration:   time.Second,
		GasLimit:   1_000_000_000,
		BaseFee:    big.NewInt(1000),
		Epoch:      1,
	}
	if number == 0 {
		header.Time = inter.Timestamp(time.Now().Add(-time.Hour).UnixNano())
	}
	block := evmcore.NewEvmBlock(header, types.Transactions(txs))
	receipts := make(types.Receipts, len(txs))
	var cumulative uint64
	for i, tx := range txs {
		cumulative += 21000
		receipts[i] = &types.Receipt{
			Status:            types.ReceiptStatusSuccessful,
			CumulativeGasUsed: cumulative,
			GasUsed:           21000,
			TxHash:            tx.Hash(),
			BlockHash:         header.Hash,
			BlockNumber:       big.NewInt(int64(number)),
			TransactionIndex:  uint(i),
		}
		f.txs[tx.Hash()] = txPos{number, uint64(i)}
		delete(f.pool, tx.Hash())
	}
	block.GasUsed = cumulative
	f.blocks = append(f.blocks, block)
	f.receipts = append(f.receipts, receipts)
	f.byHash[header.Hash] = number
	f.progress.CurrentBlock = number
	f.progress.CurrentBlockHash = header.Hash
	f.progress.CurrentBlockTime = uint64(header.Time)
	f.progress.HighestBlock = number
	f.progress.CurrentEpoch = 1
	f.progress.HighestEpoch = 1
	return block
}

// SetReceipts replaces the receipts of a block.
func (f *FakeCore) SetReceipts(number uint64, receipts types.Receipts) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.receipts[number] = receipts
}

// ReceiptsOf returns the receipts of a block for the test to modify in place.
func (f *FakeCore) ReceiptsOf(number uint64) types.Receipts {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.receipts[number]
}

// AnnounceUpTo announces every block up to the number on the head feed, in order.
func (f *FakeCore) AnnounceUpTo(number uint64) {
	f.mu.RLock()
	blocks := append([]*evmcore.EvmBlock(nil), f.blocks[1:number+1]...)
	receipts := append([]types.Receipts(nil), f.receipts[1:number+1]...)
	subs := slicesCloneSubs(f.headSubs)
	f.mu.RUnlock()
	for i, block := range blocks {
		ev := core.HeadEvent{Block: block, Logs: collectLogs(receipts[i])}
		for _, sub := range subs {
			sub.send(ev)
		}
	}
}

// SetHeaderHook makes Header fail for the blocks the hook rejects.
func (f *FakeCore) SetHeaderHook(hook func(number uint64) error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.headerHook = hook
}

// HideBlocksFrom makes the blocks from the number on unknown to Header and
// Receipts, as if the node had not written them yet.
func (f *FakeCore) HideBlocksFrom(number uint64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.hiddenFrom, f.hidden = number, true
}

// SetState sets the function that opens the state at a block.
func (f *FakeCore) SetState(fn func(number uint64) (state.StateDB, error)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stateFn = fn
}

// SetFindLogs sets the log index.
func (f *FakeCore) SetFindLogs(fn func(from, to uint64, pattern [][]common.Hash, limit uint) ([]*types.Log, error)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.findLogs = fn
}

// SetSendTx sets the verdict of the pool on a transaction; by default the
// transaction is added to the pool.
func (f *FakeCore) SetSendTx(fn func(*types.Transaction) error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sendTx = fn
}

// SetPoolNonce sets the nonce the pool reports for an account.
func (f *FakeCore) SetPoolNonce(addr common.Address, nonce uint64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nonces[addr] = nonce
}

// SetBundleInfo records where a bundle was executed.
func (f *FakeCore) SetBundleInfo(hash common.Hash, info core.BundleInfo) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.bundles[hash] = info
}

// SetProgress sets the sync progress.
func (f *FakeCore) SetProgress(p core.Progress) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.progress = p
}

// SetOracleInputs sets the inputs of the gas price oracle.
func (f *FakeCore) SetOracleInputs(in core.OracleInputs) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.oracle = in
}

// AddPoolTransaction puts a transaction into the pool and announces it.
func (f *FakeCore) AddPoolTransaction(tx *types.Transaction) {
	f.mu.Lock()
	f.addToPool(tx)
	subs := slicesCloneSubs(f.txSubs)
	f.mu.Unlock()
	for _, sub := range subs {
		sub.send(evmcore.NewTxsNotify{Txs: []*types.Transaction{tx}})
	}
}

func (f *FakeCore) addToPool(tx *types.Transaction) {
	if _, ok := f.pool[tx.Hash()]; !ok {
		f.poolOrder = append(f.poolOrder, tx.Hash())
	}
	f.pool[tx.Hash()] = tx
}

// Block returns a block of the chain, for expectations in tests.
func (f *FakeCore) BlockAt(number uint64) *evmcore.EvmBlock {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.blocks[number]
}

// Latest is the number of the newest block.
func (f *FakeCore) Latest() uint64 {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return uint64(len(f.blocks) - 1)
}

// -- injection --

// Inject sets the mode of a method; "*" sets all methods.
func (f *FakeCore) Inject(method string, mode Mode) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if method == "*" {
		for _, m := range CoreMethods {
			f.inject[m] = mode
		}
		return
	}
	f.inject[method] = mode
}

// SetDelay makes every call take that long, for cancellation tests.
func (f *FakeCore) SetDelay(d time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.delay = d
}

// Calls returns how often a method was called.
func (f *FakeCore) Calls(method string) int {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.calls[method]
}

// BlockCalls makes calls of the method wait until the returned function is called.
func (f *FakeCore) BlockCalls(method string) (release func()) {
	gate := make(chan struct{})
	f.mu.Lock()
	f.gates[method] = gate
	f.mu.Unlock()
	var once sync.Once
	return func() { once.Do(func() { close(gate) }) }
}

// WaitBlocked waits until a call of the method waits at its gate.
func (f *FakeCore) WaitBlocked(method string) {
	for {
		f.mu.RLock()
		n := f.waiting[method]
		f.mu.RUnlock()
		if n > 0 {
			return
		}
		time.Sleep(time.Millisecond)
	}
}

// enter counts the call, applies the delay and returns the mode of the method.
func (f *FakeCore) enter(method string) Mode {
	f.mu.Lock()
	f.calls[method]++
	mode := f.inject[method]
	delay := f.delay
	gate := f.gates[method]
	if gate != nil {
		f.waiting[method]++
	}
	f.mu.Unlock()
	if gate != nil {
		<-gate
		f.mu.Lock()
		f.waiting[method]--
		f.mu.Unlock()
	}
	if delay > 0 {
		time.Sleep(delay)
	}
	if mode == Panic {
		panic(fmt.Sprintf("injected panic in Core.%s", method))
	}
	return mode
}

func (m Mode) err() error {
	switch m {
	case Fail:
		return ErrInjected
	case NotFound:
		return core.ErrNotFound
	}
	return nil
}

// -- ChainReader --

func (f *FakeCore) ChainInfo() core.ChainInfo {
	if f.enter("ChainInfo") != Normal {
		return core.ChainInfo{ChainID: 1, NetworkID: 1}
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.info
}

func (f *FakeCore) Header(number uint64) (*evmcore.EvmHeader, error) {
	if mode := f.enter("Header"); mode != Normal {
		return nil, mode.err()
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	if number >= uint64(len(f.blocks)) || f.hidden && number >= f.hiddenFrom {
		return nil, core.ErrNotFound
	}
	if f.headerHook != nil {
		if err := f.headerHook(number); err != nil {
			return nil, err
		}
	}
	return f.blocks[number].Header(), nil
}

func (f *FakeCore) Block(number uint64) (*evmcore.EvmBlock, error) {
	if mode := f.enter("Block"); mode != Normal {
		return nil, mode.err()
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	if number >= uint64(len(f.blocks)) {
		return nil, core.ErrNotFound
	}
	return f.blocks[number], nil
}

func (f *FakeCore) BlockNumberByHash(hash common.Hash) (uint64, bool) {
	if f.enter("BlockNumberByHash") != Normal {
		return 0, false
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	n, ok := f.byHash[hash]
	return n, ok
}

func (f *FakeCore) Receipts(number uint64) (types.Receipts, error) {
	if mode := f.enter("Receipts"); mode != Normal {
		return nil, mode.err()
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	if number >= uint64(len(f.receipts)) || f.hidden && number >= f.hiddenFrom {
		return nil, core.ErrNotFound
	}
	return f.receipts[number], nil
}

func (f *FakeCore) Rules(number uint64) (*opera.Rules, error) {
	if mode := f.enter("Rules"); mode != Normal {
		return nil, mode.err()
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	rules := f.rules(number)
	return &rules, nil
}

func (f *FakeCore) TxByHash(hash common.Hash) (*types.Transaction, uint64, uint64, bool) {
	if f.enter("TxByHash") != Normal {
		return nil, 0, 0, false
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	pos, ok := f.txs[hash]
	if !ok {
		return nil, 0, 0, false
	}
	return f.blocks[pos.block].Transactions[pos.index], pos.block, pos.index, true
}

func (f *FakeCore) BundleInfo(hash common.Hash) (core.BundleInfo, bool) {
	if f.enter("BundleInfo") != Normal {
		return core.BundleInfo{}, false
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	info, ok := f.bundles[hash]
	return info, ok
}

// -- HeadSource --

func (f *FakeCore) Head() (*evmcore.EvmBlock, error) {
	if mode := f.enter("Head"); mode != Normal {
		return nil, mode.err()
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.blocks[len(f.blocks)-1], nil
}

func (f *FakeCore) SubscribeHead(ch chan<- core.HeadEvent) core.Subscription {
	f.enter("SubscribeHead")
	sub := newFeedSub(ch)
	f.mu.Lock()
	f.headSubs = append(f.headSubs, sub)
	f.mu.Unlock()
	return sub
}

// -- StateProvider --

func (f *FakeCore) State(number uint64) (state.StateDB, error) {
	if mode := f.enter("State"); mode != Normal {
		return nil, mode.err()
	}
	f.mu.RLock()
	fn := f.stateFn
	n := uint64(len(f.blocks))
	f.mu.RUnlock()
	if number >= n {
		return nil, core.ErrNotFound
	}
	if fn == nil {
		return nil, errors.New("fake core has no state")
	}
	return fn(number)
}

// -- Pool --

func (f *FakeCore) SendTx(_ context.Context, tx *types.Transaction) error {
	if mode := f.enter("SendTx"); mode != Normal {
		return mode.err()
	}
	f.mu.RLock()
	fn := f.sendTx
	f.mu.RUnlock()
	if fn != nil {
		if err := fn(tx); err != nil {
			return err
		}
	}
	f.AddPoolTransaction(tx)
	return nil
}

func (f *FakeCore) PoolNonce(addr common.Address) uint64 {
	if f.enter("PoolNonce") != Normal {
		return 0
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.nonces[addr]
}

func (f *FakeCore) PoolTransaction(hash common.Hash) *types.Transaction {
	if f.enter("PoolTransaction") != Normal {
		return nil
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.pool[hash]
}

func (f *FakeCore) PoolTransactions() (types.Transactions, error) {
	if mode := f.enter("PoolTransactions"); mode != Normal {
		return nil, mode.err()
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	var txs types.Transactions
	for _, h := range f.poolOrder {
		if tx, ok := f.pool[h]; ok {
			txs = append(txs, tx)
		}
	}
	return txs, nil
}

// poolBySender groups the pool by sender in arrival order.
func (f *FakeCore) poolBySender() map[common.Address]types.Transactions {
	res := map[common.Address]types.Transactions{}
	signer := types.LatestSignerForChainID(new(big.Int).SetUint64(f.info.ChainID))
	for _, h := range f.poolOrder {
		tx, ok := f.pool[h]
		if !ok {
			continue
		}
		from, err := types.Sender(signer, tx)
		if err != nil {
			continue
		}
		res[from] = append(res[from], tx)
	}
	return res
}

func (f *FakeCore) PoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
	if f.enter("PoolContent") != Normal {
		return nil, nil
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.poolBySender(), map[common.Address]types.Transactions{}
}

func (f *FakeCore) PoolContentFrom(addr common.Address) (types.Transactions, types.Transactions) {
	if f.enter("PoolContentFrom") != Normal {
		return nil, nil
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.poolBySender()[addr], nil
}

func (f *FakeCore) PoolStats() (int, int) {
	if f.enter("PoolStats") != Normal {
		return 0, 0
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	return len(f.pool), 0
}

func (f *FakeCore) SubscribeTxs(ch chan<- evmcore.NewTxsNotify) core.Subscription {
	f.enter("SubscribeTxs")
	sub := newFeedSub(ch)
	f.mu.Lock()
	f.txSubs = append(f.txSubs, sub)
	f.mu.Unlock()
	return sub
}

// -- LogIndex --

func (f *FakeCore) FindLogs(_ context.Context, from, to uint64, pattern [][]common.Hash, limit uint) ([]*types.Log, error) {
	if mode := f.enter("FindLogs"); mode != Normal {
		return nil, mode.err()
	}
	f.mu.RLock()
	fn := f.findLogs
	f.mu.RUnlock()
	if fn == nil {
		return nil, errors.New("fake core has no log index")
	}
	return fn(from, to, pattern, limit)
}

// -- NodeInfo --

func (f *FakeCore) Progress() core.Progress {
	if f.enter("Progress") != Normal {
		return core.Progress{}
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.progress
}

func (f *FakeCore) OracleInputs() core.OracleInputs {
	if f.enter("OracleInputs") != Normal {
		return core.OracleInputs{MinTip: new(big.Int)}
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.oracle
}

func (f *FakeCore) PendingHeads(max int) []core.PendingHead {
	if f.enter("PendingHeads") != Normal {
		return nil
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	var heads []core.PendingHead
	for _, txs := range f.poolBySender() {
		if len(heads) >= max {
			break
		}
		heads = append(heads, core.PendingHead{Tx: txs[0]})
	}
	return heads
}

// feedSub delivers values to a channel until it is unsubscribed.
type feedSub[T any] struct {
	ch   chan<- T
	errs chan error
	quit chan struct{}
	once sync.Once
}

func newFeedSub[T any](ch chan<- T) *feedSub[T] {
	return &feedSub[T]{ch: ch, errs: make(chan error), quit: make(chan struct{})}
}

func (s *feedSub[T]) send(v T) {
	select {
	case s.ch <- v:
	case <-s.quit:
	}
}

func (s *feedSub[T]) Err() <-chan error { return s.errs }

func (s *feedSub[T]) Unsubscribe() {
	s.once.Do(func() {
		close(s.quit)
		close(s.errs)
	})
}

// StateTracker counts the states a FakeCore hands out and takes back, to check
// that every state is released exactly once (I7).
type StateTracker struct {
	opened   atomic.Int64
	released atomic.Int64
	twice    atomic.Int64
}

// Opened is the number of states handed out, copies included.
func (s *StateTracker) Opened() int64 { return s.opened.Load() }

// Released is the number of releases.
func (s *StateTracker) Released() int64 { return s.released.Load() }

// Require fails the test unless every state was released exactly once.
func (s *StateTracker) Require(t testing.TB) {
	t.Helper()
	if s.twice.Load() != 0 {
		t.Errorf("%d states were released more than once", s.twice.Load())
	}
	if s.opened.Load() != s.released.Load() {
		t.Errorf("%d states were opened and %d released", s.opened.Load(), s.released.Load())
	}
}

type trackedState struct {
	state.StateDB
	tracker  *StateTracker
	released atomic.Bool
}

func (s *trackedState) Release() {
	if s.released.Swap(true) {
		s.tracker.twice.Add(1)
		return
	}
	s.tracker.released.Add(1)
	s.StateDB.Release()
}

func (s *trackedState) Copy() state.StateDB {
	s.tracker.opened.Add(1)
	return &trackedState{StateDB: s.StateDB.Copy(), tracker: s.tracker}
}

// TrackStates makes the core count the states it hands out. Call it after the
// state function is set.
func (f *FakeCore) TrackStates() *StateTracker {
	tracker := &StateTracker{}
	f.mu.Lock()
	defer f.mu.Unlock()
	inner := f.stateFn
	f.stateFn = func(n uint64) (state.StateDB, error) {
		st, err := inner(n)
		if err != nil {
			return nil, err
		}
		tracker.opened.Add(1)
		return &trackedState{StateDB: st, tracker: tracker}, nil
	}
	return tracker
}
