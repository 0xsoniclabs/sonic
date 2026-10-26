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

// Package tests holds the test support of the RPC module: FakeCore, the
// in-memory double of core.Core, a real Carmen state behind it, accounts and
// contract code, and the harness that calls a namespace the way a transport
// does.
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
	"testing"
	"time"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/rpc/core"
)

// Mode is the behaviour FakeCore gives to the calls of one method.
type Mode int

const (
	// Normal answers from the fake's data.
	Normal Mode = iota
	// Panic panics inside the call.
	Panic
	// Fail answers ErrInjected.
	Fail
	// NotFound answers core.ErrNotFound.
	NotFound
	// Nil answers the zero values without an error, breaking the rule that
	// a lookup never answers nothing without saying so.
	Nil
)

// Modes lists every mode that injects a fault.
var Modes = []Mode{Panic, Fail, NotFound, Nil}

func (mode Mode) String() string {
	switch mode {
	case Normal:
		return "normal"
	case Panic:
		return "panic"
	case Fail:
		return "fail"
	case NotFound:
		return "not found"
	case Nil:
		return "nil"
	}
	return fmt.Sprintf("mode %d", int(mode))
}

// ErrInjected is the error of a call in the Fail mode.
var ErrInjected = errors.New("injected failure")

// PanicValue is what a call in the Panic mode panics with.
const PanicValue = "injected panic"

// CoreMethods are the names of the methods of core.Core, derived from the
// interface so that a name it does not have cannot be used.
var CoreMethods = methodNames(reflect.TypeFor[core.Core]())

// LoopMethods are the methods of core.Core the head and pool loops call at
// times a test does not control; a count of them says nothing about a
// request.
var LoopMethods = []string{"ChainInfo", "Head", "Progress", "SubscribeHead", "SubscribeTransactions"}

func methodNames(interfaceType reflect.Type) []string {
	names := make([]string, interfaceType.NumMethod())
	for index := range names {
		names[index] = interfaceType.Method(index).Name
	}
	return names
}

// FakeChainID is the chain id of a FakeCore.
const FakeChainID = 4003

// FakeCore is an in-memory core.Core. Its chain grows through AppendBlock,
// its head moves through PublishHead, and each method can be given a fault
// mode, counted and held at a gate. It hands out copies of its data; a test
// changes it only through its setters.
type FakeCore struct {
	t testing.TB

	mutex        sync.Mutex
	chainInfo    core.ChainInfo
	blocks       []*evmcore.EvmBlock
	receipts     []types.Receipts
	epochRules   map[idx.Epoch]opera.Rules
	openRules    opera.Rules
	archived     uint64
	stateRoot    common.Hash
	positions    map[common.Hash]position
	bundles      map[common.Hash]core.BundleInfo
	progress     *core.Progress
	oracleInputs core.OracleInputs
	pool         pool
	openState    func(number uint64) (state.StateDB, error)
	sent         types.Transactions
	sendError    error

	modes map[string]Mode
	calls map[string]int
	gates map[string]chan struct{}
	delay time.Duration

	heads        feed[core.HeadEvent]
	transactions feed[evmcore.NewTxsNotify]
}

type position struct {
	block uint64
	index uint64
}

type pool struct {
	pending map[common.Address]types.Transactions
	queued  map[common.Address]types.Transactions
	nonces  map[common.Address]uint64
}

// NewFakeCore returns a FakeCore holding a genesis block, archived and
// served as the head.
func NewFakeCore(t testing.TB) *FakeCore {
	rules := opera.FakeNetRules(opera.GetAllegroUpgrades())
	rules.NetworkID = FakeChainID
	fake := &FakeCore{
		t: t,
		chainInfo: core.ChainInfo{
			ChainID:        FakeChainID,
			GenesisID:      common.HexToHash("0x6e6573"),
			UpgradeHeights: []opera.UpgradeHeight{{Upgrades: rules.Upgrades, Height: 0}},
		},
		epochRules: map[idx.Epoch]opera.Rules{0: rules, 1: rules},
		openRules:  rules,
		positions:  map[common.Hash]position{},
		bundles:    map[common.Hash]core.BundleInfo{},
		pool: pool{
			pending: map[common.Address]types.Transactions{},
			queued:  map[common.Address]types.Transactions{},
			nonces:  map[common.Address]uint64{},
		},
		oracleInputs: core.OracleInputs{PendingRules: rules, MinTip: new(big.Int)},
		modes:        map[string]Mode{},
		calls:        map[string]int{},
		gates:        map[string]chan struct{}{},
	}
	fake.AppendBlock(BlockSpec{})
	return fake
}

// BlockSpec describes a block to append. Zero fields take defaults: epoch 1
// (0 for genesis), a base fee of 1 gwei, a gas limit of 30 million and a time
// one second after the parent. Receipts are made for transactions that have
// none: successful, with the gas of the transaction and no logs.
type BlockSpec struct {
	Transactions types.Transactions
	Receipts     types.Receipts
	Epoch        idx.Epoch
	BaseFee      *big.Int
	GasLimit     uint64
	Root         common.Hash
}

// AppendBlock adds a committed block on top of the chain and returns a copy
// of it. The archived head does not move; PublishHead moves it.
func (fake *FakeCore) AppendBlock(spec BlockSpec) *evmcore.EvmBlock {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()

	number := uint64(len(fake.blocks))
	block := evmcore.NewEvmBlock(fake.newHeader(number, spec), spec.Transactions)
	fake.blocks = append(fake.blocks, block)
	fake.receipts = append(fake.receipts, newReceipts(block, spec.Receipts))
	for index, transaction := range spec.Transactions {
		fake.positions[transaction.Hash()] = position{block: number, index: uint64(index)}
	}
	return copyBlock(block)
}

func (fake *FakeCore) newHeader(number uint64, spec BlockSpec) *evmcore.EvmHeader {
	epoch := spec.Epoch
	if epoch == 0 && number > 0 {
		epoch = 1
	}
	baseFee := spec.BaseFee
	if baseFee == nil {
		baseFee = big.NewInt(1_000_000_000)
	}
	gasLimit := spec.GasLimit
	if gasLimit == 0 {
		gasLimit = 30_000_000
	}
	var parent common.Hash
	timestamp := inter.FromUnix(1_700_000_000)
	if number > 0 {
		previous := fake.blocks[number-1]
		parent = previous.Hash
		timestamp = previous.Time + inter.Timestamp(time.Second)
	}
	var gasUsed uint64
	for _, transaction := range spec.Transactions {
		gasUsed += transaction.Gas()
	}
	root := spec.Root
	if root == (common.Hash{}) {
		root = fake.stateRoot
	}
	return &evmcore.EvmHeader{
		Number:          new(big.Int).SetUint64(number),
		Hash:            blockHash(number, parent),
		ParentHash:      parent,
		Root:            root,
		Time:            timestamp,
		Duration:        time.Second,
		GasLimit:        gasLimit,
		GasUsed:         gasUsed,
		BaseFee:         baseFee,
		PrevRandao:      blockHash(number, common.Hash{1}),
		WithdrawalsHash: &types.EmptyWithdrawalsHash,
		Epoch:           epoch,
	}
}

func blockHash(number uint64, parent common.Hash) common.Hash {
	return crypto.Keccak256Hash(binary.BigEndian.AppendUint64(parent.Bytes(), number))
}

func newReceipts(block *evmcore.EvmBlock, given types.Receipts) types.Receipts {
	receipts := slices.Clone(given)
	var cumulative uint64
	var logIndex uint
	for index, transaction := range block.Transactions {
		if index >= len(receipts) {
			receipts = append(receipts, &types.Receipt{
				Type:    transaction.Type(),
				Status:  types.ReceiptStatusSuccessful,
				GasUsed: transaction.Gas(),
			})
		}
		receipt := receipts[index]
		cumulative += receipt.GasUsed
		receipt.CumulativeGasUsed = cumulative
		receipt.TxHash = transaction.Hash()
		receipt.BlockHash = block.Hash
		receipt.BlockNumber = new(big.Int).Set(block.Number)
		receipt.TransactionIndex = uint(index)
		for _, log := range receipt.Logs {
			log.TxHash = transaction.Hash()
			log.TxIndex = uint(index)
			log.BlockHash = block.Hash
			log.BlockNumber = block.NumberU64()
			log.BlockTimestamp = uint64(block.Time.Unix())
			log.Index = logIndex
			logIndex++
		}
		receipt.Bloom = types.CreateBloom(receipt)
	}
	return receipts
}

// PublishHead archives the blocks up to the given number and sends each one
// that was not archived yet on the head feed, with its logs. Like the node's
// feed, it waits for every subscriber to take each event.
func (fake *FakeCore) PublishHead(number uint64) {
	fake.mutex.Lock()
	if number >= uint64(len(fake.blocks)) {
		fake.mutex.Unlock()
		fake.t.Fatalf("cannot publish block %d, the chain ends at %d", number, len(fake.blocks)-1)
	}
	var events []core.HeadEvent
	for next := fake.archived + 1; next <= number; next++ {
		events = append(events, core.HeadEvent{Block: copyBlock(fake.blocks[next]), Logs: blockLogs(fake.receipts[next])})
	}
	if number > fake.archived {
		fake.archived = number
	}
	fake.mutex.Unlock()

	for _, event := range events {
		fake.heads.send(event)
	}
}

// AppendAndPublish appends a block and publishes it as the new head.
func (fake *FakeCore) AppendAndPublish(spec BlockSpec) *evmcore.EvmBlock {
	block := fake.AppendBlock(spec)
	fake.PublishHead(block.NumberU64())
	return block
}

// SendHeadEvent sends an event on the head feed without archiving anything,
// for tests that need the feed to deliver what the chain does not say.
func (fake *FakeCore) SendHeadEvent(event core.HeadEvent) {
	fake.heads.send(event)
}

// SendPoolTransactions sends transactions on the pool feed.
func (fake *FakeCore) SendPoolTransactions(transactions ...*types.Transaction) {
	fake.transactions.send(evmcore.NewTxsNotify{Txs: transactions})
}

// HeadSubscribers is the number of live subscriptions of the head feed.
func (fake *FakeCore) HeadSubscribers() int {
	return fake.heads.count()
}

// PoolSubscribers is the number of live subscriptions of the pool feed.
func (fake *FakeCore) PoolSubscribers() int {
	return fake.transactions.count()
}

// EndFeeds ends every subscription of both feeds, as a node that stops.
func (fake *FakeCore) EndFeeds() {
	fake.heads.end()
	fake.transactions.end()
}

func blockLogs(receipts types.Receipts) []*types.Log {
	var logs []*types.Log
	for _, receipt := range receipts {
		logs = append(logs, receipt.Logs...)
	}
	return logs
}

// SetMode gives the calls of a method of core.Core a fault mode.
func (fake *FakeCore) SetMode(method string, mode Mode) {
	fake.checkMethod(method)
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	fake.modes[method] = mode
}

// SetAllModes gives every method of core.Core the same mode.
func (fake *FakeCore) SetAllModes(mode Mode) {
	for _, method := range CoreMethods {
		fake.SetMode(method, mode)
	}
}

// Calls is the number of calls the method of core.Core received.
func (fake *FakeCore) Calls(method string) int {
	fake.checkMethod(method)
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	return fake.calls[method]
}

// Hold makes the calls of a method wait until the returned function is called.
func (fake *FakeCore) Hold(method string) (release func()) {
	fake.checkMethod(method)
	gate := make(chan struct{})
	fake.mutex.Lock()
	fake.gates[method] = gate
	fake.mutex.Unlock()
	var once sync.Once
	release = func() {
		once.Do(func() {
			fake.mutex.Lock()
			delete(fake.gates, method)
			fake.mutex.Unlock()
			close(gate)
		})
	}
	fake.t.Cleanup(release)
	return release
}

// SetDelay makes every call of every method wait for the given duration.
func (fake *FakeCore) SetDelay(delay time.Duration) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	fake.delay = delay
}

func (fake *FakeCore) checkMethod(method string) {
	if !slices.Contains(CoreMethods, method) {
		fake.t.Fatalf("core.Core has no method %q", method)
	}
}

// enter records a call of a method and applies its gate, the delay and its
// mode. It returns injected=true with the error to answer when the call must
// not answer from the fake's data; the error is nil in the Nil mode.
func (fake *FakeCore) enter(method string) (injected bool, err error) {
	fake.mutex.Lock()
	fake.calls[method]++
	mode := fake.modes[method]
	gate := fake.gates[method]
	delay := fake.delay
	fake.mutex.Unlock()

	if gate != nil {
		<-gate
	}
	if delay > 0 {
		time.Sleep(delay)
	}
	switch mode {
	case Panic:
		panic(PanicValue)
	case Fail:
		return true, ErrInjected
	case NotFound:
		return true, core.ErrNotFound
	case Nil:
		return true, nil
	}
	return false, nil
}

// SetChainInfo replaces the facts ChainInfo answers.
func (fake *FakeCore) SetChainInfo(info core.ChainInfo) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	fake.chainInfo = info
}

// SetEpochRules sets the rules of the blocks of an epoch.
func (fake *FakeCore) SetEpochRules(epoch idx.Epoch, rules opera.Rules) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	fake.epochRules[epoch] = rules
}

// SetOpenRules sets the rules of the open epoch, those of the block after
// the committed head.
func (fake *FakeCore) SetOpenRules(rules opera.Rules) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	fake.openRules = rules
}

// SetBundle records where a bundle was executed.
func (fake *FakeCore) SetBundle(executionPlanHash common.Hash, info core.BundleInfo) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	fake.bundles[executionPlanHash] = info
}

// SetProgress replaces what Progress answers. Without it, Progress reports
// the committed head as current and highest block.
func (fake *FakeCore) SetProgress(progress core.Progress) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	fake.progress = &progress
}

// SetOracleInputs replaces what OracleInputs answers.
func (fake *FakeCore) SetOracleInputs(inputs core.OracleInputs) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	fake.oracleInputs = inputs
}

// SetPool replaces the content of the pool.
func (fake *FakeCore) SetPool(pending, queued map[common.Address]types.Transactions) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	fake.pool.pending = copyContent(pending)
	fake.pool.queued = copyContent(queued)
}

// SetPoolNonce sets the nonce the pool reports for an account.
func (fake *FakeCore) SetPoolNonce(address common.Address, nonce uint64) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	fake.pool.nonces[address] = nonce
}

// SetSendError makes SendTransaction answer the given error, as a pool that
// refuses a transaction.
func (fake *FakeCore) SetSendError(err error) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	fake.sendError = err
}

// Sent returns the transactions SendTransaction accepted.
func (fake *FakeCore) Sent() types.Transactions {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	return slices.Clone(fake.sent)
}

// SetStates makes State open states with the given function.
func (fake *FakeCore) SetStates(open func(number uint64) (state.StateDB, error)) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	fake.openState = open
}

// ChainHead is the number of the committed head.
func (fake *FakeCore) ChainHead() uint64 {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	return uint64(len(fake.blocks) - 1)
}

// ChainInfo implements core.ChainReader.
func (fake *FakeCore) ChainInfo() (core.ChainInfo, error) {
	if injected, err := fake.enter("ChainInfo"); injected {
		return core.ChainInfo{}, err
	}
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	info := fake.chainInfo
	info.UpgradeHeights = slices.Clone(info.UpgradeHeights)
	return info, nil
}

// Header implements core.ChainReader. Like the node's, the header carries no
// transactions root.
func (fake *FakeCore) Header(number uint64) (*evmcore.EvmHeader, error) {
	if injected, err := fake.enter("Header"); injected {
		return nil, err
	}
	block, err := fake.block(number)
	if err != nil {
		return nil, err
	}
	header := block.Header()
	header.TxHash = common.Hash{}
	return header, nil
}

// Block implements core.ChainReader.
func (fake *FakeCore) Block(number uint64) (*evmcore.EvmBlock, error) {
	if injected, err := fake.enter("Block"); injected {
		return nil, err
	}
	return fake.block(number)
}

func (fake *FakeCore) block(number uint64) (*evmcore.EvmBlock, error) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	if number >= uint64(len(fake.blocks)) {
		return nil, core.ErrNotFound
	}
	return copyBlock(fake.blocks[number]), nil
}

// BlockNumberByHash implements core.ChainReader.
func (fake *FakeCore) BlockNumberByHash(hash common.Hash) (uint64, error) {
	if injected, err := fake.enter("BlockNumberByHash"); injected {
		return 0, err
	}
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	for number, block := range fake.blocks {
		if block.Hash == hash {
			return uint64(number), nil
		}
	}
	return 0, core.ErrNotFound
}

// Receipts implements core.ChainReader.
func (fake *FakeCore) Receipts(number uint64) (types.Receipts, error) {
	if injected, err := fake.enter("Receipts"); injected {
		return nil, err
	}
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	if number >= uint64(len(fake.receipts)) {
		return nil, core.ErrNotFound
	}
	return copyReceipts(fake.receipts[number]), nil
}

// Rules implements core.ChainReader.
func (fake *FakeCore) Rules(number uint64) (opera.Rules, error) {
	if injected, err := fake.enter("Rules"); injected {
		return opera.Rules{}, err
	}
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	switch {
	case number < uint64(len(fake.blocks)):
		rules, found := fake.epochRules[fake.blocks[number].Epoch]
		if !found {
			return opera.Rules{}, core.ErrNotFound
		}
		return rules, nil
	case number == uint64(len(fake.blocks)):
		return fake.openRules, nil
	}
	return opera.Rules{}, core.ErrNotFound
}

// TransactionByHash implements core.ChainReader.
func (fake *FakeCore) TransactionByHash(hash common.Hash) (*types.Transaction, uint64, uint64, error) {
	if injected, err := fake.enter("TransactionByHash"); injected {
		return nil, 0, 0, err
	}
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	found, ok := fake.positions[hash]
	if !ok {
		return nil, 0, 0, core.ErrNotFound
	}
	return fake.blocks[found.block].Transactions[found.index], found.block, found.index, nil
}

// BundleInfo implements core.ChainReader.
func (fake *FakeCore) BundleInfo(executionPlanHash common.Hash) (core.BundleInfo, error) {
	if injected, err := fake.enter("BundleInfo"); injected {
		return core.BundleInfo{}, err
	}
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	info, found := fake.bundles[executionPlanHash]
	if !found {
		return core.BundleInfo{}, core.ErrNotFound
	}
	return info, nil
}

// Head implements core.HeadSource.
func (fake *FakeCore) Head() (*evmcore.EvmBlock, error) {
	if injected, err := fake.enter("Head"); injected {
		return nil, err
	}
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	return copyBlock(fake.blocks[fake.archived]), nil
}

// SubscribeHead implements core.HeadSource.
func (fake *FakeCore) SubscribeHead(events chan<- core.HeadEvent) core.Subscription {
	if injected, _ := fake.enter("SubscribeHead"); injected {
		return endedSubscription()
	}
	return fake.heads.subscribe(events)
}

// State implements core.StateProvider. Without SetStates it opens no state.
func (fake *FakeCore) State(number uint64) (state.StateDB, error) {
	if injected, err := fake.enter("State"); injected {
		return nil, err
	}
	fake.mutex.Lock()
	open := fake.openState
	known := number < uint64(len(fake.blocks))
	fake.mutex.Unlock()
	if !known || open == nil {
		return nil, core.ErrNotFound
	}
	return open(number)
}

// SendTransaction implements core.Pool.
func (fake *FakeCore) SendTransaction(_ context.Context, transaction *types.Transaction) error {
	if injected, err := fake.enter("SendTransaction"); injected {
		return err
	}
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	if fake.sendError != nil {
		return fake.sendError
	}
	fake.sent = append(fake.sent, transaction)
	return nil
}

// PoolNonce implements core.Pool.
func (fake *FakeCore) PoolNonce(address common.Address) uint64 {
	if injected, _ := fake.enter("PoolNonce"); injected {
		return 0
	}
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	return fake.pool.nonces[address]
}

// PoolTransaction implements core.Pool.
func (fake *FakeCore) PoolTransaction(hash common.Hash) (*types.Transaction, error) {
	if injected, err := fake.enter("PoolTransaction"); injected {
		return nil, err
	}
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	for _, content := range []map[common.Address]types.Transactions{fake.pool.pending, fake.pool.queued} {
		for _, transactions := range content {
			for _, transaction := range transactions {
				if transaction.Hash() == hash {
					return transaction, nil
				}
			}
		}
	}
	return nil, core.ErrNotFound
}

// PoolTransactions implements core.Pool.
func (fake *FakeCore) PoolTransactions() (types.Transactions, error) {
	if injected, err := fake.enter("PoolTransactions"); injected {
		return nil, err
	}
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	var all types.Transactions
	for _, address := range sortedAddresses(fake.pool.pending) {
		all = append(all, fake.pool.pending[address]...)
	}
	return all, nil
}

// PoolContent implements core.Pool.
func (fake *FakeCore) PoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
	if injected, _ := fake.enter("PoolContent"); injected {
		return nil, nil
	}
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	return copyContent(fake.pool.pending), copyContent(fake.pool.queued)
}

// PoolContentFrom implements core.Pool.
func (fake *FakeCore) PoolContentFrom(address common.Address) (types.Transactions, types.Transactions) {
	if injected, _ := fake.enter("PoolContentFrom"); injected {
		return nil, nil
	}
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	return slices.Clone(fake.pool.pending[address]), slices.Clone(fake.pool.queued[address])
}

// PoolStats implements core.Pool.
func (fake *FakeCore) PoolStats() (int, int) {
	if injected, _ := fake.enter("PoolStats"); injected {
		return 0, 0
	}
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	return countTransactions(fake.pool.pending), countTransactions(fake.pool.queued)
}

// SubscribeTransactions implements core.Pool.
func (fake *FakeCore) SubscribeTransactions(events chan<- evmcore.NewTxsNotify) core.Subscription {
	if injected, _ := fake.enter("SubscribeTransactions"); injected {
		return endedSubscription()
	}
	return fake.transactions.subscribe(events)
}

// FindLogs implements core.LogIndex like the node's index: pattern[0] holds
// the addresses as hashes, pattern[1:] the topics, an empty position matches
// anything. Like the index, it leaves the transaction index and the timestamp
// of the logs unset.
func (fake *FakeCore) FindLogs(ctx context.Context, first, last uint64, pattern [][]common.Hash, limit uint) ([]*types.Log, error) {
	if injected, err := fake.enter("FindLogs"); injected {
		return nil, err
	}
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	var found []*types.Log
	for number := first; number <= last && number < uint64(len(fake.receipts)); number++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		for _, log := range blockLogs(fake.receipts[number]) {
			if !matchesPattern(log, pattern) {
				continue
			}
			copied := *log
			copied.TxIndex = 0
			copied.BlockTimestamp = 0
			found = append(found, &copied)
			if limit > 0 && uint(len(found)) > limit {
				return nil, fmt.Errorf("too many results, consider narrowing your query criteria, the limit is %d", limit)
			}
		}
	}
	return found, nil
}

func matchesPattern(log *types.Log, pattern [][]common.Hash) bool {
	for position, alternatives := range pattern {
		if len(alternatives) == 0 {
			continue
		}
		var value common.Hash
		switch {
		case position == 0:
			value = common.BytesToHash(log.Address.Bytes())
		case position-1 < len(log.Topics):
			value = log.Topics[position-1]
		default:
			return false
		}
		if !slices.Contains(alternatives, value) {
			return false
		}
	}
	return true
}

// Progress implements core.NodeInfo.
func (fake *FakeCore) Progress() core.Progress {
	if injected, _ := fake.enter("Progress"); injected {
		return core.Progress{}
	}
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	if fake.progress != nil {
		return *fake.progress
	}
	head := fake.blocks[len(fake.blocks)-1]
	return core.Progress{
		CurrentEpoch:     uint64(head.Epoch),
		CurrentBlock:     head.NumberU64(),
		CurrentBlockHash: head.Hash,
		CurrentBlockTime: head.Time,
		HighestBlock:     head.NumberU64(),
		HighestEpoch:     uint64(head.Epoch),
	}
}

// OracleInputs implements core.NodeInfo.
func (fake *FakeCore) OracleInputs() core.OracleInputs {
	if injected, _ := fake.enter("OracleInputs"); injected {
		return core.OracleInputs{}
	}
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	inputs := fake.oracleInputs
	if inputs.MinTip != nil {
		inputs.MinTip = new(big.Int).Set(inputs.MinTip)
	}
	return inputs
}

// PendingHeads implements core.NodeInfo.
func (fake *FakeCore) PendingHeads() types.Transactions {
	if injected, _ := fake.enter("PendingHeads"); injected {
		return nil
	}
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	var heads types.Transactions
	for _, address := range sortedAddresses(fake.pool.pending) {
		if transactions := fake.pool.pending[address]; len(transactions) > 0 {
			heads = append(heads, transactions[0])
		}
	}
	return heads
}

func copyBlock(block *evmcore.EvmBlock) *evmcore.EvmBlock {
	copied := *block
	copied.Number = new(big.Int).Set(block.Number)
	if block.BaseFee != nil {
		copied.BaseFee = new(big.Int).Set(block.BaseFee)
	}
	copied.Transactions = slices.Clone(block.Transactions)
	return &copied
}

func copyReceipts(receipts types.Receipts) types.Receipts {
	copied := make(types.Receipts, len(receipts))
	for index, receipt := range receipts {
		receiptCopy := *receipt
		receiptCopy.Logs = make([]*types.Log, len(receipt.Logs))
		for position, log := range receipt.Logs {
			logCopy := *log
			logCopy.Topics = slices.Clone(log.Topics)
			receiptCopy.Logs[position] = &logCopy
		}
		if receipt.Logs == nil {
			receiptCopy.Logs = nil
		}
		copied[index] = &receiptCopy
	}
	return copied
}

func copyContent(content map[common.Address]types.Transactions) map[common.Address]types.Transactions {
	copied := make(map[common.Address]types.Transactions, len(content))
	for address, transactions := range content {
		copied[address] = slices.Clone(transactions)
	}
	return copied
}

func countTransactions(content map[common.Address]types.Transactions) int {
	count := 0
	for _, transactions := range content {
		count += len(transactions)
	}
	return count
}

func sortedAddresses(content map[common.Address]types.Transactions) []common.Address {
	addresses := make([]common.Address, 0, len(content))
	for address := range content {
		addresses = append(addresses, address)
	}
	sort.Slice(addresses, func(first, second int) bool {
		return addresses[first].Cmp(addresses[second]) < 0
	})
	return addresses
}
