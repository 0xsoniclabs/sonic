// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package evmcore

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"math/rand/v2"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

var (
	// testTxPoolConfig is a transaction pool configuration without stateful disk
	// sideeffects used during testing.
	testTxPoolConfig TxPoolConfig

	// eip1559Config is a chain config with EIP-1559 enabled at block 0.
	eip1559Config *params.ChainConfig
)

func init() {
	testTxPoolConfig = DefaultTxPoolConfig
	testTxPoolConfig.Journal = ""

	cpy := *params.TestChainConfig
	eip1559Config = &cpy
	eip1559Config.BerlinBlock = common.Big0
	eip1559Config.LondonBlock = common.Big0
}

type testTxPoolStateDb struct {
	balances map[common.Address]*uint256.Int
	nonces   map[common.Address]uint64
}

func newTestTxPoolStateDb() *testTxPoolStateDb {
	return &testTxPoolStateDb{
		balances: make(map[common.Address]*uint256.Int),
		nonces:   make(map[common.Address]uint64),
	}
}

func (t testTxPoolStateDb) GetNonce(addr common.Address) uint64 {
	return t.nonces[addr]
}

func (t testTxPoolStateDb) GetBalance(addr common.Address) *uint256.Int {
	return t.balances[addr]
}

func (t testTxPoolStateDb) Release() {
	// no-op
}

type testBlockChain struct {
	statedb       *testTxPoolStateDb
	gasLimit      uint64
	chainHeadFeed *event.Feed

	mu sync.RWMutex
}

func NewTestBlockChain(statedb *testTxPoolStateDb) *testBlockChain {
	return &testBlockChain{statedb, 10000000, new(event.Feed), sync.RWMutex{}}
}

func (bc *testBlockChain) changeStateDB(statedb *testTxPoolStateDb) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	bc.statedb = statedb
}

func (bc *testBlockChain) CurrentBlock() *EvmBlock {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	return &EvmBlock{
		EvmHeader: EvmHeader{
			Number:     big.NewInt(1),
			Hash:       common.Hash{},
			ParentHash: common.Hash{},
			Root:       common.Hash{},
			TxHash:     common.Hash{},
			Time:       0,
			Coinbase:   common.Address{},
			GasLimit:   bc.gasLimit,
			GasUsed:    0,
			BaseFee:    nil,
		},
		Transactions: nil,
	}
}

func (bc *testBlockChain) GetCurrentBaseFee() *big.Int {
	return nil
}
func (bc *testBlockChain) MaxGasLimit() uint64 {
	return bc.CurrentBlock().GasLimit
}
func (bc *testBlockChain) Config() *params.ChainConfig {
	return nil
}

func (bc *testBlockChain) GetBlock(hash common.Hash, number uint64) *EvmBlock {
	return bc.CurrentBlock()
}

func (bc *testBlockChain) GetTxPoolStateDB() (TxPoolStateDB, error) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	return bc.statedb, nil
}

func (bc *testBlockChain) SubscribeNewBlock(ch chan<- ChainHeadNotify) event.Subscription {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	return bc.chainHeadFeed.Subscribe(ch)
}

func transaction(nonce uint64, gaslimit uint64, key *ecdsa.PrivateKey) *types.Transaction {
	return pricedTransaction(nonce, gaslimit, big.NewInt(1), key)
}

func pricedTransaction(nonce uint64, gaslimit uint64, gasprice *big.Int, key *ecdsa.PrivateKey) *types.Transaction {
	tx, _ := types.SignTx(types.NewTransaction(nonce, common.Address{}, big.NewInt(100), gaslimit, gasprice, nil), types.HomesteadSigner{}, key)
	return tx
}

func pricedDataTransaction(nonce uint64, gaslimit uint64, gasprice *big.Int, key *ecdsa.PrivateKey, bytes uint64) *types.Transaction {
	data := make([]byte, bytes)
	for i := range data {
		data[i] = byte(rand.IntN(256))
	}

	tx, _ := types.SignTx(types.NewTransaction(nonce, common.Address{}, big.NewInt(0), gaslimit, gasprice, data), types.HomesteadSigner{}, key)
	return tx
}

func dynamicFeeTx(nonce uint64, gaslimit uint64, gasFee *big.Int, tip *big.Int, key *ecdsa.PrivateKey) *types.Transaction {
	tx, _ := types.SignNewTx(key, types.LatestSignerForChainID(params.TestChainConfig.ChainID), &types.DynamicFeeTx{
		ChainID:    params.TestChainConfig.ChainID,
		Nonce:      nonce,
		GasTipCap:  tip,
		GasFeeCap:  gasFee,
		Gas:        gaslimit,
		To:         &common.Address{},
		Value:      big.NewInt(100),
		Data:       nil,
		AccessList: nil,
	})
	return tx
}

func setupTxPool() (*TxPool, *ecdsa.PrivateKey) {
	return setupTxPoolWithConfig(params.TestChainConfig)
}

func setupTxPoolWithConfig(config *params.ChainConfig) (*TxPool, *ecdsa.PrivateKey) {
	blockchain := NewTestBlockChain(newTestTxPoolStateDb())

	key, _ := crypto.GenerateKey()
	pool := NewTxPool(testTxPoolConfig, config, blockchain)

	return pool, key
}

// validateTxPoolInternals checks various consistency invariants within the pool.
func validateTxPoolInternals(pool *TxPool) error {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	// Ensure the total transaction set is consistent with pending + queued
	pending, queued := pool.stats()
	if total := pool.all.Count(); total != pending+queued {
		return fmt.Errorf("total transaction count %d != %d pending + %d queued", total, pending, queued)
	}
	pool.priced.Reheap()
	priced, remote := pool.priced.urgent.Len()+pool.priced.floating.Len(), pool.all.RemoteCount()
	if priced != remote {
		return fmt.Errorf("total priced transaction count %d != %d", priced, remote)
	}
	// Ensure the next nonce to assign is the correct one
	for addr, txs := range pool.pending {
		// Find the last transaction
		var last uint64
		for nonce := range txs.txs.items {
			if last < nonce {
				last = nonce
			}
		}
		if nonce := pool.pendingNonces.get(addr); nonce != last+1 {
			return fmt.Errorf("pending nonce mismatch: have %v, want %v", nonce, last+1)
		}
	}
	return nil
}

// validateEvents checks that the correct number of transaction addition events
// were fired on the pool's event feed.
func validateEvents(events chan NewTxsNotify, count int) error {
	var received []*types.Transaction

	// add a timer to detect non-firing events
	time.Sleep(50 * time.Millisecond)

	for len(received) < count {
		select {
		case ev := <-events:
			received = append(received, ev.Txs...)
		case <-time.After(5 * time.Second):
			return fmt.Errorf("event #%d not fired", len(received))
		}
	}
	if len(received) > count {
		return fmt.Errorf("more than %d events fired: %v", count, received[count:])
	}
	select {
	case ev := <-events:
		return fmt.Errorf("more than %d events fired: %v", count, ev.Txs)

	case <-time.After(50 * time.Millisecond):
		// This branch should be "default", but it's a data race between goroutines,
		// reading the event channel and pushing into it, so better wait a bit ensuring
		// really nothing gets injected.
	}
	return nil
}

func deriveSender(tx *types.Transaction) (common.Address, error) {
	return types.Sender(types.HomesteadSigner{}, tx)
}

type testChain struct {
	*testBlockChain
	address common.Address
	trigger *bool
}

// This test simulates a scenario where a new block is imported during a
// state reset and tests whether the pending state is in sync with the
// block head event that initiated the resetState().
func TestStateChangeDuringTransactionPoolReset(t *testing.T) {
	t.Parallel()

	var (
		key, _  = crypto.GenerateKey()
		address = crypto.PubkeyToAddress(key.PublicKey)
		statedb = newTestTxPoolStateDb()
		trigger = false
	)

	// setup pool with 2 transaction in it
	statedb.balances[address] = new(uint256.Int).SetUint64(params.Ether)
	blockchain := &testChain{NewTestBlockChain(statedb), address, &trigger}

	tx0 := transaction(0, 100000, key)
	tx1 := transaction(1, 100000, key)

	pool := NewTxPool(testTxPoolConfig, params.TestChainConfig, blockchain)
	defer pool.Stop()

	nonce := pool.Nonce(address)
	if nonce != 0 {
		t.Fatalf("Invalid nonce, want 0, got %d", nonce)
	}

	pool.AddRemotesSync([]*types.Transaction{tx0, tx1})

	nonce = pool.Nonce(address)
	if nonce != 2 {
		t.Fatalf("Invalid nonce, want 2, got %d", nonce)
	}

	// trigger state change in the background
	trigger = true
	<-pool.requestReset(nil, nil)

	_, err := pool.Pending(false)
	if err != nil {
		t.Fatalf("Could not fetch pending transactions: %v", err)
	}
	nonce = pool.Nonce(address)
	if nonce != 2 {
		t.Fatalf("Invalid nonce, want 2, got %d", nonce)
	}
}

func testAddBalance(pool *TxPool, addr common.Address, amount *big.Int) {
	pool.mu.Lock()
	original := pool.currentState.(*testTxPoolStateDb).balances[addr]
	if original == nil {
		amountU256 := utils.BigIntToUint256(amount)
		pool.currentState.(*testTxPoolStateDb).balances[addr] = amountU256
	} else {
		if amount.Sign() >= 0 {
			amountU256 := utils.BigIntToUint256(amount)
			pool.currentState.(*testTxPoolStateDb).balances[addr] = original.Add(original, amountU256)
		} else {
			amountU256 := utils.BigIntToUint256(new(big.Int).Mul(amount, big.NewInt(-1)))
			pool.currentState.(*testTxPoolStateDb).balances[addr] = original.Sub(original, amountU256)
		}
	}
	pool.mu.Unlock()
}

func testSetNonce(pool *TxPool, addr common.Address, nonce uint64) {
	pool.mu.Lock()
	pool.currentState.(*testTxPoolStateDb).nonces[addr] = nonce
	pool.mu.Unlock()
}

// TestEIP4844Transactions tests validation of the blob transaction
// when adding it to a transaction pool.
func TestEIP4844Transactions(t *testing.T) {

	configCopy := *params.TestChainConfig
	testConfig := &configCopy

	// initialize the pool
	pool, key := setupTxPoolWithConfig(testConfig)
	defer pool.Stop()

	// get the chain id
	chainId := params.TestChainConfig.ChainID

	// get sender address and put balance on it
	from := crypto.PubkeyToAddress(key.PublicKey)
	balance := new(big.Int)
	balance.SetString("10000000000000000000000000000", 10)
	testAddBalance(pool, from, balance)

	tests := []struct {
		name   string
		txData []byte
		cancun bool
		err    error
	}{
		{"empty blob tx before cancun", nil, false, ErrTxTypeNotSupported},
		{"blob tx before cancun", common.Address{1}.Bytes(), false, ErrTxTypeNotSupported},
		{"empty blob tx", nil, true, nil},
		{"blob tx with data", common.Address{1}.Bytes(), true, ErrTxTypeNotSupported},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			if test.cancun {
				testConfig.CancunTime = new(uint64)
			} else {
				testConfig.CancunTime = nil
			}
			pool.reset(nil, nil)

			tx, err := createTestBlobTransaction(chainId, test.txData)
			if err != nil {
				t.Fatalf("could not create blob tx: %v", err)
			}

			signedTx, err := types.SignTx(tx, types.NewCancunSigner(chainId), key)
			if err != nil {
				t.Fatalf("could not sign tx: %v", err)
			}

			_, err = pool.add(signedTx, false)

			if err != test.err {
				t.Fatalf("expected error %v, got %v", test.err, err)
			}
		})
	}
}

func createTestBlobTransaction(chainId *big.Int, data []byte) (*types.Transaction, error) {

	var (
		sidecar    *types.BlobTxSidecar // The sidecar contains the blob data
		blobHashes []common.Hash
	)

	if data != nil {

		var Blob kzg4844.Blob // Define a blob array to hold the large data payload, blobs are 128kb in length
		copy(Blob[:], data)

		// Compute the commitment for the blob data using KZG4844 cryptographic algorithm
		BlobCommitment, err := kzg4844.BlobToCommitment(&Blob)
		if err != nil {
			return nil, fmt.Errorf("failed to compute blob commitment: %s", err)
		}

		// Compute the proof for the blob data, which will be used to verify the transaction
		BlobProof, err := kzg4844.ComputeBlobProof(&Blob, BlobCommitment)
		if err != nil {
			return nil, fmt.Errorf("failed to compute blob proof: %s", err)
		}

		//Prepare the sidecar data for the transaction, which includes the blob and its cryptographic proof
		sidecar = &types.BlobTxSidecar{
			Blobs:       []kzg4844.Blob{Blob},
			Commitments: []kzg4844.Commitment{BlobCommitment},
			Proofs:      []kzg4844.Proof{BlobProof},
		}

		// Get blob hashes from the sidecar
		blobHashes = sidecar.BlobHashes()
	}

	// Create and return transaction with the blob data and cryptographic proofs
	return types.NewTx(&types.BlobTx{
		ChainID:    uint256.MustFromBig(chainId),
		Nonce:      0,
		GasTipCap:  uint256.NewInt(1e10),  // max priority fee per gas
		GasFeeCap:  uint256.NewInt(50e10), // max fee per gas
		Gas:        250000,                // gas limit for the transaction
		To:         common.Address{},      // recipient's address
		Value:      uint256.NewInt(0),     // value transferred in the transaction
		Data:       nil,                   // No additional data is sent in this transaction
		BlobFeeCap: uint256.NewInt(3e10),  // fee cap for the blob data
		BlobHashes: blobHashes,            // blob hashes in the transaction
		Sidecar:    sidecar,               // sidecar data in the transaction
	}), nil
}

func TestEIP7702Transactions_InvalidTransactionsReturnAnError(t *testing.T) {

	configCopy := *params.TestChainConfig
	testConfig := &configCopy

	// initialize the pool
	pool, key := setupTxPoolWithConfig(testConfig)
	defer pool.Stop()

	// get the chain id
	chainId := params.TestChainConfig.ChainID

	// get sender address and put balance on it
	from := crypto.PubkeyToAddress(key.PublicKey)
	balance := new(big.Int)
	balance.SetString("10000000000000000000000000000", 10)
	testAddBalance(pool, from, balance)

	tests := map[string]struct {
		authorizations []types.SetCodeAuthorization
		pragueTime     *uint64
		expectedErr    error
	}{
		"set code tx before prague": {
			expectedErr: ErrTxTypeNotSupported,
		},
		"set code tx with nil authorizations": {
			pragueTime:  new(uint64),
			expectedErr: ErrEmptyAuthorizations,
		},
		"set code tx with empty authorizations": {
			pragueTime:     new(uint64),
			authorizations: []types.SetCodeAuthorization{},
			expectedErr:    ErrEmptyAuthorizations,
		},
		"set code tx": {
			pragueTime: new(uint64),
			authorizations: []types.SetCodeAuthorization{
				{
					ChainID: *uint256.MustFromBig(chainId),
					Address: common.Address{},
					Nonce:   1,
					V:       1,
					R:       *uint256.NewInt(1),
					S:       *uint256.NewInt(1),
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			testConfig.PragueTime = test.pragueTime
			pool.reset(nil, nil)

			tx := createTestSetCodeTransaction(chainId, test.authorizations)
			signedTx, err := types.SignTx(tx, types.NewPragueSigner(chainId), key)
			if err != nil {
				t.Fatalf("could not sign tx: %v", err)
			}

			_, err = pool.add(signedTx, false)
			if err != test.expectedErr {
				t.Fatalf("expected error %v, got %v", test.expectedErr, err)
			}
		})
	}
}

func createTestSetCodeTransaction(chainId *big.Int, authorizations []types.SetCodeAuthorization) *types.Transaction {

	return types.NewTx(&types.SetCodeTx{
		ChainID:   uint256.MustFromBig(chainId),
		Nonce:     0,
		GasTipCap: uint256.NewInt(1e10),  // max priority fee per gas
		GasFeeCap: uint256.NewInt(50e10), // max fee per gas
		Gas:       250000,                // gas limit for the transaction
		To:        common.Address{},      // sponsored address
		Value:     uint256.NewInt(0),     // value transferred in the transaction
		Data:      nil,                   // No additional data is sent in this transaction
		AuthList:  authorizations,        // authorizations in the transaction
	})
}

func TestInvalidTransactions(t *testing.T) {
	t.Parallel()

	pool, key := setupTxPool()
	defer pool.Stop()

	tx := transaction(0, 100, key)
	from, _ := deriveSender(tx)

	testAddBalance(pool, from, big.NewInt(1))
	if err := pool.AddRemote(tx); !errors.Is(err, ErrInsufficientFunds) {
		t.Error("expected", ErrInsufficientFunds)
	}

	balance := new(big.Int).Add(tx.Value(), new(big.Int).Mul(new(big.Int).SetUint64(tx.Gas()), tx.GasPrice()))
	testAddBalance(pool, from, balance)
	if err := pool.AddRemote(tx); !errors.Is(err, ErrIntrinsicGas) {
		t.Error("expected", ErrIntrinsicGas, "got", err)
	}

	testSetNonce(pool, from, 1)
	testAddBalance(pool, from, big.NewInt(0xffffffffffffff))
	tx = transaction(0, 100000, key)
	if err := pool.AddRemote(tx); !errors.Is(err, ErrNonceTooLow) {
		t.Error("expected", ErrNonceTooLow)
	}

	tx = transaction(1, 100000, key)
	pool.gasPrice = big.NewInt(1000)
	if err := pool.AddRemote(tx); err != ErrUnderpriced {
		t.Error("expected", ErrUnderpriced, "got", err)
	}
	if err := pool.AddLocal(tx); err != nil {
		t.Error("expected", nil, "got", err)
	}
}

func TestTransactionQueue(t *testing.T) {
	t.Parallel()

	pool, key := setupTxPool()
	defer pool.Stop()

	tx := transaction(0, 100, key)
	from, _ := deriveSender(tx)
	testAddBalance(pool, from, big.NewInt(1000))
	<-pool.requestReset(nil, nil)

	if _, err := pool.enqueueTx(tx.Hash(), tx, false, true); err != nil {
		t.Error("failed to enqueue tx", err)
	}
	<-pool.requestPromoteExecutables(newAccountSet(pool.signer, from))
	if len(pool.pending) != 1 {
		t.Error("expected valid txs to be 1 is", len(pool.pending))
	}

	tx = transaction(1, 100, key)
	from, _ = deriveSender(tx)
	testSetNonce(pool, from, 2)
	if _, err := pool.enqueueTx(tx.Hash(), tx, false, true); err != nil {
		t.Error("failed to enqueue tx", err)
	}

	<-pool.requestPromoteExecutables(newAccountSet(pool.signer, from))
	if _, ok := pool.pending[from].txs.items[tx.Nonce()]; ok {
		t.Error("expected transaction to be in tx pool")
	}
	if len(pool.queue) > 0 {
		t.Error("expected transaction queue to be empty. is", len(pool.queue))
	}
}

func TestTransactionQueue2(t *testing.T) {
	t.Parallel()

	pool, key := setupTxPool()
	defer pool.Stop()

	tx1 := transaction(0, 100, key)
	tx2 := transaction(10, 100, key)
	tx3 := transaction(11, 100, key)
	from, _ := deriveSender(tx1)
	testAddBalance(pool, from, big.NewInt(1000))
	pool.reset(nil, nil)

	if _, err := pool.enqueueTx(tx1.Hash(), tx1, false, true); err != nil {
		t.Error("failed to enqueue tx1", err)
	}
	if _, err := pool.enqueueTx(tx2.Hash(), tx2, false, true); err != nil {
		t.Error("failed to enqueue tx2", err)
	}
	if _, err := pool.enqueueTx(tx3.Hash(), tx3, false, true); err != nil {
		t.Error("failed to enqueue tx3", err)
	}

	pool.promoteExecutables([]common.Address{from})
	if len(pool.pending) != 1 {
		t.Error("expected pending length to be 1, got", len(pool.pending))
	}
	if pool.queue[from].Len() != 2 {
		t.Error("expected len(queue) == 2, got", pool.queue[from].Len())
	}
}

func TestTransactionNegativeValue(t *testing.T) {
	t.Parallel()

	pool, key := setupTxPool()
	defer pool.Stop()

	tx, _ := types.SignTx(types.NewTransaction(0, common.Address{}, big.NewInt(-1), 100, big.NewInt(1), nil), types.HomesteadSigner{}, key)
	from, _ := deriveSender(tx)
	testAddBalance(pool, from, big.NewInt(1))
	if err := pool.AddRemote(tx); err != ErrNegativeValue {
		t.Error("expected", ErrNegativeValue, "got", err)
	}
}

func TestTransactionTipAboveFeeCap(t *testing.T) {
	t.Parallel()

	pool, key := setupTxPoolWithConfig(eip1559Config)
	defer pool.Stop()

	tx := dynamicFeeTx(0, 100, big.NewInt(1), big.NewInt(2), key)

	if err := pool.AddRemote(tx); err != ErrTipAboveFeeCap {
		t.Error("expected", ErrTipAboveFeeCap, "got", err)
	}
}

func TestTransactionVeryHighValues(t *testing.T) {
	t.Parallel()

	pool, key := setupTxPoolWithConfig(eip1559Config)
	defer pool.Stop()

	veryBigNumber := big.NewInt(1)
	veryBigNumber.Lsh(veryBigNumber, 300)

	tx := dynamicFeeTx(0, 100, big.NewInt(1), veryBigNumber, key)
	if err := pool.AddRemote(tx); err != ErrTipVeryHigh {
		t.Error("expected", ErrTipVeryHigh, "got", err)
	}

	tx2 := dynamicFeeTx(0, 100, veryBigNumber, big.NewInt(1), key)
	if err := pool.AddRemote(tx2); err != ErrFeeCapVeryHigh {
		t.Error("expected", ErrFeeCapVeryHigh, "got", err)
	}
}

func TestTransactionChainFork(t *testing.T) {
	t.Parallel()

	pool, key := setupTxPool()
	defer pool.Stop()

	addr := crypto.PubkeyToAddress(key.PublicKey)
	resetState := func() {
		statedb := newTestTxPoolStateDb()
		statedb.balances[addr] = uint256.NewInt(100000000000000)

		pool.chain.(*testBlockChain).changeStateDB(statedb)
		<-pool.requestReset(nil, nil)
	}
	resetState()

	tx := transaction(0, 100000, key)
	if _, err := pool.add(tx, false); err != nil {
		t.Error("didn't expect error", err)
	}
	pool.removeTx(tx.Hash(), true)

	// reset the pool's internal state
	resetState()
	if _, err := pool.add(tx, false); err != nil {
		t.Error("didn't expect error", err)
	}
}

func TestTransactionDoubleNonce(t *testing.T) {
	t.Parallel()

	pool, key := setupTxPool()
	defer pool.Stop()

	addr := crypto.PubkeyToAddress(key.PublicKey)
	statedb := newTestTxPoolStateDb()
	statedb.balances[addr] = uint256.NewInt(100000000000000)
	pool.chain.(*testBlockChain).changeStateDB(statedb)

	<-pool.requestReset(nil, nil)

	signer := types.HomesteadSigner{}
	tx1, _ := types.SignTx(types.NewTransaction(0, common.Address{}, big.NewInt(100), 100000, big.NewInt(1), nil), signer, key)
	tx2, _ := types.SignTx(types.NewTransaction(0, common.Address{}, big.NewInt(100), 1000000, big.NewInt(2), nil), signer, key)
	tx3, _ := types.SignTx(types.NewTransaction(0, common.Address{}, big.NewInt(100), 1000000, big.NewInt(1), nil), signer, key)

	// Add the first two transaction, ensure higher priced stays only
	if replace, err := pool.add(tx1, false); err != nil || replace {
		t.Errorf("first transaction insert failed (%v) or reported replacement (%v)", err, replace)
	}
	if replace, err := pool.add(tx2, false); err != nil || !replace {
		t.Errorf("second transaction insert failed (%v) or not reported replacement (%v)", err, replace)
	}
	<-pool.requestPromoteExecutables(newAccountSet(signer, addr))
	if pool.pending[addr].Len() != 1 {
		t.Error("expected 1 pending transactions, got", pool.pending[addr].Len())
	}
	if tx := pool.pending[addr].txs.items[0]; tx.Hash() != tx2.Hash() {
		t.Errorf("transaction mismatch: have %x, want %x", tx.Hash(), tx2.Hash())
	}

	// Add the third transaction and ensure it's not saved (smaller price)
	if _, err := pool.add(tx3, false); err == nil {
		t.Error("expected transaction to be rejected, it was not")
	}

	<-pool.requestPromoteExecutables(newAccountSet(signer, addr))
	if pool.pending[addr].Len() != 1 {
		t.Error("expected 1 pending transactions, got", pool.pending[addr].Len())
	}
	if tx := pool.pending[addr].txs.items[0]; tx.Hash() != tx2.Hash() {
		t.Errorf("transaction mismatch: have %x, want %x", tx.Hash(), tx2.Hash())
	}
	// Ensure the total transaction count is correct
	if pool.all.Count() != 1 {
		t.Error("expected 1 total transactions, got", pool.all.Count())
	}
}

func TestTransactionMissingNonce(t *testing.T) {
	t.Parallel()

	pool, key := setupTxPool()
	defer pool.Stop()

	addr := crypto.PubkeyToAddress(key.PublicKey)
	testAddBalance(pool, addr, big.NewInt(100000000000000))
	tx := transaction(1, 100000, key)
	if _, err := pool.add(tx, false); err != nil {
		t.Error("didn't expect error", err)
	}
	if len(pool.pending) != 0 {
		t.Error("expected 0 pending transactions, got", len(pool.pending))
	}
	if pool.queue[addr].Len() != 1 {
		t.Error("expected 1 queued transaction, got", pool.queue[addr].Len())
	}
	if pool.all.Count() != 1 {
		t.Error("expected 1 total transactions, got", pool.all.Count())
	}
}

func TestTransactionNonceRecovery(t *testing.T) {
	t.Parallel()

	const n = 10
	pool, key := setupTxPool()
	defer pool.Stop()

	addr := crypto.PubkeyToAddress(key.PublicKey)
	testSetNonce(pool, addr, n)
	testAddBalance(pool, addr, big.NewInt(100000000000000))
	<-pool.requestReset(nil, nil)

	tx := transaction(n, 100000, key)
	if err := pool.AddRemote(tx); err != nil {
		t.Error(err)
	}
	// simulate some weird re-order of transactions and missing nonce(s)
	testSetNonce(pool, addr, n-1)
	<-pool.requestReset(nil, nil)
	if fn := pool.Nonce(addr); fn != n-1 {
		t.Errorf("expected nonce to be %d, got %d", n-1, fn)
	}
}

// Tests that if an account runs out of funds, any pending and queued transactions
// are dropped.
func TestTransactionDropping(t *testing.T) {
	t.Parallel()

	// Create a test account and fund it
	pool, key := setupTxPool()
	defer pool.Stop()

	account := crypto.PubkeyToAddress(key.PublicKey)
	testAddBalance(pool, account, big.NewInt(1000))

	// Add some pending and some queued transactions
	var (
		tx0  = transaction(0, 100, key)
		tx1  = transaction(1, 200, key)
		tx2  = transaction(2, 300, key)
		tx10 = transaction(10, 100, key)
		tx11 = transaction(11, 200, key)
		tx12 = transaction(12, 300, key)
	)
	pool.all.Add(tx0, false)
	pool.priced.Put(tx0, false)
	pool.promoteTx(account, tx0.Hash(), tx0)

	pool.all.Add(tx1, false)
	pool.priced.Put(tx1, false)
	pool.promoteTx(account, tx1.Hash(), tx1)

	pool.all.Add(tx2, false)
	pool.priced.Put(tx2, false)
	pool.promoteTx(account, tx2.Hash(), tx2)

	if _, err := pool.enqueueTx(tx10.Hash(), tx10, false, true); err != nil {
		t.Fatalf("failed to enqueue tx10: %v", err)
	}
	if _, err := pool.enqueueTx(tx11.Hash(), tx11, false, true); err != nil {
		t.Fatalf("failed to enqueue tx11: %v", err)
	}
	if _, err := pool.enqueueTx(tx12.Hash(), tx12, false, true); err != nil {
		t.Fatalf("failed to enqueue tx12: %v", err)
	}

	// Check that pre and post validations leave the pool as is
	if pool.pending[account].Len() != 3 {
		t.Errorf("pending transaction mismatch: have %d, want %d", pool.pending[account].Len(), 3)
	}
	if pool.queue[account].Len() != 3 {
		t.Errorf("queued transaction mismatch: have %d, want %d", pool.queue[account].Len(), 3)
	}
	if pool.all.Count() != 6 {
		t.Errorf("total transaction mismatch: have %d, want %d", pool.all.Count(), 6)
	}
	<-pool.requestReset(nil, nil)
	if pool.pending[account].Len() != 3 {
		t.Errorf("pending transaction mismatch: have %d, want %d", pool.pending[account].Len(), 3)
	}
	if pool.queue[account].Len() != 3 {
		t.Errorf("queued transaction mismatch: have %d, want %d", pool.queue[account].Len(), 3)
	}
	if pool.all.Count() != 6 {
		t.Errorf("total transaction mismatch: have %d, want %d", pool.all.Count(), 6)
	}
	// Reduce the balance of the account, and check that invalidated transactions are dropped
	testAddBalance(pool, account, big.NewInt(-650))
	<-pool.requestReset(nil, nil)

	if _, ok := pool.pending[account].txs.items[tx0.Nonce()]; !ok {
		t.Errorf("funded pending transaction missing: %v", tx0)
	}
	if _, ok := pool.pending[account].txs.items[tx1.Nonce()]; !ok {
		t.Errorf("funded pending transaction missing: %v", tx0)
	}
	if _, ok := pool.pending[account].txs.items[tx2.Nonce()]; ok {
		t.Errorf("out-of-fund pending transaction present: %v", tx1)
	}
	if _, ok := pool.queue[account].txs.items[tx10.Nonce()]; !ok {
		t.Errorf("funded queued transaction missing: %v", tx10)
	}
	if _, ok := pool.queue[account].txs.items[tx11.Nonce()]; !ok {
		t.Errorf("funded queued transaction missing: %v", tx10)
	}
	if _, ok := pool.queue[account].txs.items[tx12.Nonce()]; ok {
		t.Errorf("out-of-fund queued transaction present: %v", tx11)
	}
	if pool.all.Count() != 4 {
		t.Errorf("total transaction mismatch: have %d, want %d", pool.all.Count(), 4)
	}
	// Reduce the block gas limit, check that invalidated transactions are dropped
	pool.chain.(*testBlockChain).gasLimit = 100
	<-pool.requestReset(nil, nil)

	if _, ok := pool.pending[account].txs.items[tx0.Nonce()]; !ok {
		t.Errorf("funded pending transaction missing: %v", tx0)
	}
	if _, ok := pool.pending[account].txs.items[tx1.Nonce()]; ok {
		t.Errorf("over-gased pending transaction present: %v", tx1)
	}
	if _, ok := pool.queue[account].txs.items[tx10.Nonce()]; !ok {
		t.Errorf("funded queued transaction missing: %v", tx10)
	}
	if _, ok := pool.queue[account].txs.items[tx11.Nonce()]; ok {
		t.Errorf("over-gased queued transaction present: %v", tx11)
	}
	if pool.all.Count() != 2 {
		t.Errorf("total transaction mismatch: have %d, want %d", pool.all.Count(), 2)
	}
}

// Tests that if a transaction is dropped from the current pending pool (e.g. out
// of fund), all consecutive (still valid, but not executable) transactions are
// postponed back into the future queue to prevent broadcasting them.
func TestTransactionPostponing(t *testing.T) {
	t.Parallel()

	// Create the pool to test the postponing with
	blockchain := NewTestBlockChain(newTestTxPoolStateDb())
	pool := NewTxPool(testTxPoolConfig, params.TestChainConfig, blockchain)
	defer pool.Stop()

	// Create two test accounts to produce different gap profiles with
	keys := make([]*ecdsa.PrivateKey, 2)
	accs := make([]common.Address, len(keys))

	for i := 0; i < len(keys); i++ {
		keys[i], _ = crypto.GenerateKey()
		accs[i] = crypto.PubkeyToAddress(keys[i].PublicKey)

		testAddBalance(pool, crypto.PubkeyToAddress(keys[i].PublicKey), big.NewInt(50100))
	}
	// Add a batch consecutive pending transactions for validation
	txs := []*types.Transaction{}
	for i, key := range keys {

		for j := 0; j < 100; j++ {
			var tx *types.Transaction
			if (i+j)%2 == 0 {
				tx = transaction(uint64(j), 25000, key)
			} else {
				tx = transaction(uint64(j), 50000, key)
			}
			txs = append(txs, tx)
		}
	}
	for i, err := range pool.AddRemotesSync(txs) {
		if err != nil {
			t.Fatalf("tx %d: failed to add transactions: %v", i, err)
		}
	}
	// Check that pre and post validations leave the pool as is
	if pending := pool.pending[accs[0]].Len() + pool.pending[accs[1]].Len(); pending != len(txs) {
		t.Errorf("pending transaction mismatch: have %d, want %d", pending, len(txs))
	}
	if len(pool.queue) != 0 {
		t.Errorf("queued accounts mismatch: have %d, want %d", len(pool.queue), 0)
	}
	if pool.all.Count() != len(txs) {
		t.Errorf("total transaction mismatch: have %d, want %d", pool.all.Count(), len(txs))
	}
	<-pool.requestReset(nil, nil)
	if pending := pool.pending[accs[0]].Len() + pool.pending[accs[1]].Len(); pending != len(txs) {
		t.Errorf("pending transaction mismatch: have %d, want %d", pending, len(txs))
	}
	if len(pool.queue) != 0 {
		t.Errorf("queued accounts mismatch: have %d, want %d", len(pool.queue), 0)
	}
	if pool.all.Count() != len(txs) {
		t.Errorf("total transaction mismatch: have %d, want %d", pool.all.Count(), len(txs))
	}
	// Reduce the balance of the account, and check that transactions are reorganised
	for _, addr := range accs {
		testAddBalance(pool, addr, big.NewInt(-1))
	}
	<-pool.requestReset(nil, nil)

	// The first account's first transaction remains valid, check that subsequent
	// ones are either filtered out, or queued up for later.
	if _, ok := pool.pending[accs[0]].txs.items[txs[0].Nonce()]; !ok {
		t.Errorf("tx %d: valid and funded transaction missing from pending pool: %v", 0, txs[0])
	}
	if _, ok := pool.queue[accs[0]].txs.items[txs[0].Nonce()]; ok {
		t.Errorf("tx %d: valid and funded transaction present in future queue: %v", 0, txs[0])
	}
	for i, tx := range txs[1:100] {
		if i%2 == 1 {
			if _, ok := pool.pending[accs[0]].txs.items[tx.Nonce()]; ok {
				t.Errorf("tx %d: valid but future transaction present in pending pool: %v", i+1, tx)
			}
			if _, ok := pool.queue[accs[0]].txs.items[tx.Nonce()]; !ok {
				t.Errorf("tx %d: valid but future transaction missing from future queue: %v", i+1, tx)
			}
		} else {
			if _, ok := pool.pending[accs[0]].txs.items[tx.Nonce()]; ok {
				t.Errorf("tx %d: out-of-fund transaction present in pending pool: %v", i+1, tx)
			}
			if _, ok := pool.queue[accs[0]].txs.items[tx.Nonce()]; ok {
				t.Errorf("tx %d: out-of-fund transaction present in future queue: %v", i+1, tx)
			}
		}
	}
	// The second account's first transaction got invalid, check that all transactions
	// are either filtered out, or queued up for later.
	if pool.pending[accs[1]] != nil {
		t.Errorf("invalidated account still has pending transactions")
	}
	for i, tx := range txs[100:] {
		if i%2 == 1 {
			if _, ok := pool.queue[accs[1]].txs.items[tx.Nonce()]; !ok {
				t.Errorf("tx %d: valid but future transaction missing from future queue: %v", 100+i, tx)
			}
		} else {
			if _, ok := pool.queue[accs[1]].txs.items[tx.Nonce()]; ok {
				t.Errorf("tx %d: out-of-fund transaction present in future queue: %v", 100+i, tx)
			}
		}
	}
	if pool.all.Count() != len(txs)/2 {
		t.Errorf("total transaction mismatch: have %d, want %d", pool.all.Count(), len(txs)/2)
	}
}

// Tests that if the transaction pool has both executable and non-executable
// transactions from an origin account, filling the nonce gap moves all queued
// ones into the pending pool.
func TestTransactionGapFilling(t *testing.T) {
	t.Parallel()

	// Create a test account and fund it
	pool, key := setupTxPool()
	defer pool.Stop()

	account := crypto.PubkeyToAddress(key.PublicKey)
	testAddBalance(pool, account, big.NewInt(1000000))

	// Keep track of transaction events to ensure all executables get announced
	events := make(chan NewTxsNotify, testTxPoolConfig.AccountQueue+5)
	sub := pool.txFeed.Subscribe(events)
	defer sub.Unsubscribe()

	// Create a pending and a queued transaction with a nonce-gap in between
	pool.AddRemotesSync([]*types.Transaction{
		transaction(0, 100000, key),
		transaction(2, 100000, key),
	})
	pending, queued := pool.Stats()
	if pending != 1 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 1)
	}
	if queued != 1 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 1)
	}
	if err := validateEvents(events, 1); err != nil {
		t.Fatalf("original event firing failed: %v", err)
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
	// Fill the nonce gap and ensure all transactions become pending
	if err := pool.addRemoteSync(transaction(1, 100000, key)); err != nil {
		t.Fatalf("failed to add gapped transaction: %v", err)
	}
	pending, queued = pool.Stats()
	if pending != 3 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 3)
	}
	if queued != 0 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 0)
	}
	if err := validateEvents(events, 2); err != nil {
		t.Fatalf("gap-filling event firing failed: %v", err)
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
}

// Tests that if the transaction count belonging to a single account goes above
// some threshold, the higher transactions are dropped to prevent DOS attacks.
func TestTransactionQueueAccountLimiting(t *testing.T) {
	t.Parallel()

	// Create a test account and fund it
	pool, key := setupTxPool()
	defer pool.Stop()

	account := crypto.PubkeyToAddress(key.PublicKey)
	testAddBalance(pool, account, big.NewInt(1000000))

	// Keep queuing up transactions and make sure all above a limit are dropped
	for i := uint64(1); i <= testTxPoolConfig.AccountQueue+5; i++ {
		if err := pool.addRemoteSync(transaction(i, 100000, key)); err != nil {
			t.Fatalf("tx %d: failed to add transaction: %v", i, err)
		}
		if len(pool.pending) != 0 {
			t.Errorf("tx %d: pending pool size mismatch: have %d, want %d", i, len(pool.pending), 0)
		}
		if i <= testTxPoolConfig.AccountQueue {
			if pool.queue[account].Len() != int(i) {
				t.Errorf("tx %d: queue size mismatch: have %d, want %d", i, pool.queue[account].Len(), i)
			}
		} else {
			if pool.queue[account].Len() != int(testTxPoolConfig.AccountQueue) {
				t.Errorf("tx %d: queue limit mismatch: have %d, want %d", i, pool.queue[account].Len(), testTxPoolConfig.AccountQueue)
			}
		}
	}
	if pool.all.Count() != int(testTxPoolConfig.AccountQueue) {
		t.Errorf("total transaction mismatch: have %d, want %d", pool.all.Count(), testTxPoolConfig.AccountQueue)
	}
}

// Tests that if the transaction count belonging to multiple accounts go above
// some threshold, the higher transactions are dropped to prevent DOS attacks.
//
// This logic should not hold for local transactions, unless the local tracking
// mechanism is disabled.
func TestTransactionQueueGlobalLimiting(t *testing.T) {
	testTransactionQueueGlobalLimiting(t, false)
}
func TestTransactionQueueGlobalLimitingNoLocals(t *testing.T) {
	testTransactionQueueGlobalLimiting(t, true)
}

func testTransactionQueueGlobalLimiting(t *testing.T, nolocals bool) {
	t.Parallel()

	// Create the pool to test the limit enforcement with
	blockchain := NewTestBlockChain(newTestTxPoolStateDb())

	config := testTxPoolConfig
	config.NoLocals = nolocals
	config.GlobalQueue = config.AccountQueue*3 - 1 // reduce the queue limits to shorten test time (-1 to make it non divisible)

	pool := NewTxPool(config, params.TestChainConfig, blockchain)
	defer pool.Stop()

	// Create a number of test accounts and fund them (last one will be the local)
	keys := make([]*ecdsa.PrivateKey, 5)
	for i := 0; i < len(keys); i++ {
		keys[i], _ = crypto.GenerateKey()
		testAddBalance(pool, crypto.PubkeyToAddress(keys[i].PublicKey), big.NewInt(1000000))
	}
	local := keys[len(keys)-1]

	// Generate and queue a batch of transactions
	nonces := make(map[common.Address]uint64)

	txs := make(types.Transactions, 0, 3*config.GlobalQueue)
	for len(txs) < cap(txs) {
		key := keys[rand.IntN(len(keys)-1)] // skip adding transactions with the local account
		addr := crypto.PubkeyToAddress(key.PublicKey)

		txs = append(txs, transaction(nonces[addr]+1, 100000, key))
		nonces[addr]++
	}
	// Import the batch and verify that limits have been enforced
	pool.AddRemotesSync(txs)

	queued := 0
	for addr, list := range pool.queue {
		if list.Len() > int(config.AccountQueue) {
			t.Errorf("addr %x: queued accounts overflown allowance: %d > %d", addr, list.Len(), config.AccountQueue)
		}
		queued += list.Len()
	}
	if queued > int(config.GlobalQueue) {
		t.Fatalf("total transactions overflow allowance: %d > %d", queued, config.GlobalQueue)
	}
	// Generate a batch of transactions from the local account and import them
	txs = txs[:0]
	for i := uint64(0); i < 3*config.GlobalQueue; i++ {
		txs = append(txs, transaction(i+1, 100000, local))
	}
	pool.AddLocals(txs)

	// If locals are disabled, the previous eviction algorithm should apply here too
	if nolocals {
		queued := 0
		for addr, list := range pool.queue {
			if list.Len() > int(config.AccountQueue) {
				t.Errorf("addr %x: queued accounts overflown allowance: %d > %d", addr, list.Len(), config.AccountQueue)
			}
			queued += list.Len()
		}
		if queued > int(config.GlobalQueue) {
			t.Fatalf("total transactions overflow allowance: %d > %d", queued, config.GlobalQueue)
		}
	} else {
		// Local exemptions are enabled, make sure the local account owned the queue
		if len(pool.queue) != 1 {
			t.Errorf("multiple accounts in queue: have %v, want %v", len(pool.queue), 1)
		}
		// Also ensure no local transactions are ever dropped, even if above global limits
		if queued := pool.queue[crypto.PubkeyToAddress(local.PublicKey)].Len(); uint64(queued) != 3*config.GlobalQueue {
			t.Fatalf("local account queued transaction count mismatch: have %v, want %v", queued, 3*config.GlobalQueue)
		}
	}
}

// Tests that if an account remains idle for a prolonged amount of time, any
// non-executable transactions queued up are dropped to prevent wasting resources
// on shuffling them around.
//
// This logic should not hold for local transactions, unless the local tracking
// mechanism is disabled.
func TestTransactionQueueTimeLimiting(t *testing.T) {
	testTransactionQueueTimeLimiting(t, false)
}
func TestTransactionQueueTimeLimitingNoLocals(t *testing.T) {
	testTransactionQueueTimeLimiting(t, true)
}

func testTransactionQueueTimeLimiting(t *testing.T, nolocals bool) {
	// Reduce the eviction interval to a testable amount
	defer func(old time.Duration) { evictionInterval = old }(evictionInterval)
	evictionInterval = time.Millisecond * 100

	// Create the pool to test the non-expiration enforcement
	statedb := newTestTxPoolStateDb()
	blockchain := NewTestBlockChain(statedb)

	config := testTxPoolConfig
	config.Lifetime = time.Second
	config.NoLocals = nolocals

	pool := NewTxPool(config, params.TestChainConfig, blockchain)
	defer pool.Stop()

	// Create two test accounts to ensure remotes expire but locals do not
	local, _ := crypto.GenerateKey()
	remote, _ := crypto.GenerateKey()

	testAddBalance(pool, crypto.PubkeyToAddress(local.PublicKey), big.NewInt(1000000000))
	testAddBalance(pool, crypto.PubkeyToAddress(remote.PublicKey), big.NewInt(1000000000))

	// Add the two transactions and ensure they both are queued up
	if err := pool.AddLocal(pricedTransaction(1, 100000, big.NewInt(1), local)); err != nil {
		t.Fatalf("failed to add local transaction: %v", err)
	}
	if err := pool.AddRemote(pricedTransaction(1, 100000, big.NewInt(1), remote)); err != nil {
		t.Fatalf("failed to add remote transaction: %v", err)
	}
	pending, queued := pool.Stats()
	if pending != 0 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 0)
	}
	if queued != 2 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 2)
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}

	// Allow the eviction interval to run
	time.Sleep(2 * evictionInterval)

	// Transactions should not be evicted from the queue yet since lifetime duration has not passed
	pending, queued = pool.Stats()
	if pending != 0 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 0)
	}
	if queued != 2 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 2)
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}

	// Wait a bit for eviction to run and clean up any leftovers, and ensure only the local remains
	time.Sleep(2 * config.Lifetime)

	pending, queued = pool.Stats()
	if pending != 0 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 0)
	}
	if nolocals {
		if queued != 0 {
			t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 0)
		}
	} else {
		if queued != 1 {
			t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 1)
		}
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}

	// remove current transactions and increase nonce to prepare for a reset and cleanup
	statedb.nonces[crypto.PubkeyToAddress(remote.PublicKey)] = 2
	statedb.nonces[crypto.PubkeyToAddress(local.PublicKey)] = 2
	<-pool.requestReset(nil, nil)

	// make sure queue, pending are cleared
	pending, queued = pool.Stats()
	if pending != 0 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 0)
	}
	if queued != 0 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 0)
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}

	// Queue gapped transactions
	if err := pool.AddLocal(pricedTransaction(4, 100000, big.NewInt(1), local)); err != nil {
		t.Fatalf("failed to add remote transaction: %v", err)
	}
	if err := pool.addRemoteSync(pricedTransaction(4, 100000, big.NewInt(1), remote)); err != nil {
		t.Fatalf("failed to add remote transaction: %v", err)
	}
	time.Sleep(5 * evictionInterval) // A half lifetime pass

	// Queue executable transactions, the life cycle should be restarted.
	if err := pool.AddLocal(pricedTransaction(2, 100000, big.NewInt(1), local)); err != nil {
		t.Fatalf("failed to add remote transaction: %v", err)
	}
	if err := pool.addRemoteSync(pricedTransaction(2, 100000, big.NewInt(1), remote)); err != nil {
		t.Fatalf("failed to add remote transaction: %v", err)
	}
	time.Sleep(6 * evictionInterval)

	// All gapped transactions shouldn't be kicked out
	pending, queued = pool.Stats()
	if pending != 2 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 2)
	}
	if queued != 2 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 3)
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}

	// The whole life time pass after last promotion, kick out stale transactions
	time.Sleep(2 * config.Lifetime)
	pending, queued = pool.Stats()
	if pending != 2 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 2)
	}
	if nolocals {
		if queued != 0 {
			t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 0)
		}
	} else {
		if queued != 1 {
			t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 1)
		}
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
}

func TestTransactionQueueTruncating(t *testing.T) {
	// Create the pool to test the queue truncation when GlobalQueue is exceeded
	statedb := newTestTxPoolStateDb()
	blockchain := NewTestBlockChain(statedb)

	config := testTxPoolConfig
	config.GlobalQueue = 2

	pool := NewTxPool(config, params.TestChainConfig, blockchain)
	defer pool.Stop()

	active, _ := crypto.GenerateKey()
	inactive, _ := crypto.GenerateKey()
	activeAddr := crypto.PubkeyToAddress(active.PublicKey)
	inactiveAddr := crypto.PubkeyToAddress(inactive.PublicKey)

	testAddBalance(pool, crypto.PubkeyToAddress(active.PublicKey), big.NewInt(1000000000))
	testAddBalance(pool, crypto.PubkeyToAddress(inactive.PublicKey), big.NewInt(1000000000))

	if err := pool.addRemoteSync(pricedTransaction(5, 100000, big.NewInt(1), active)); err != nil {
		t.Fatalf("failed to add transaction: %v", err)
	}
	if err := pool.addRemoteSync(pricedTransaction(5, 100000, big.NewInt(1), inactive)); err != nil {
		t.Fatalf("failed to add transaction: %v", err)
	}

	// add pending tx - should update last activity timestamp of the sender, should not affect the content of the queue
	if err := pool.addRemoteSync(pricedTransaction(0, 100000, big.NewInt(1), active)); err != nil {
		t.Fatalf("failed to add transaction: %v", err)
	}

	if queued := pool.queue[activeAddr].Len(); queued != 1 {
		t.Fatalf("queued transactions for active sender mismatched: have %d, want %d", queued, 1)
	}
	if queued := pool.queue[inactiveAddr].Len(); queued != 1 {
		t.Fatalf("queued transactions for inactive sender mismatched: have %d, want %d", queued, 1)
	}

	// add another queued tx - should be truncated immediately
	if err := pool.addRemoteSync(pricedTransaction(6, 100000, big.NewInt(1), inactive)); err != nil {
		t.Fatalf("failed to add transaction: %v", err)
	}

	// add tx of active sender - should replace tx of the inactive sender
	if err := pool.addRemoteSync(pricedTransaction(6, 100000, big.NewInt(1), active)); err != nil {
		t.Fatalf("failed to add transaction: %v", err)
	}

	if queued := pool.queue[activeAddr].Len(); queued != 2 {
		t.Fatalf("queued transactions for active sender mismatched: have %d, want %d", queued, 2)
	}
	if queue := pool.queue[inactiveAddr]; queue != nil {
		t.Fatalf("queued transactions for inactive sender mismatched: have %d, want nil", queue.Len())
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
}

// Tests that even if the transaction count belonging to a single account goes
// above some threshold, as long as the transactions are executable, they are
// accepted.
func TestTransactionPendingLimiting(t *testing.T) {
	t.Parallel()

	// Create a test account and fund it
	pool, key := setupTxPool()
	defer pool.Stop()

	account := crypto.PubkeyToAddress(key.PublicKey)
	testAddBalance(pool, account, big.NewInt(1000000))

	// Keep track of transaction events to ensure all executables get announced
	events := make(chan NewTxsNotify, testTxPoolConfig.AccountQueue+5)
	sub := pool.txFeed.Subscribe(events)
	defer sub.Unsubscribe()

	// Keep queuing up transactions and make sure all above a limit are dropped
	for i := uint64(0); i < testTxPoolConfig.AccountQueue+5; i++ {
		if err := pool.addRemoteSync(transaction(i, 100000, key)); err != nil {
			t.Fatalf("tx %d: failed to add transaction: %v", i, err)
		}
		if pool.pending[account].Len() != int(i)+1 {
			t.Errorf("tx %d: pending pool size mismatch: have %d, want %d", i, pool.pending[account].Len(), i+1)
		}
		if len(pool.queue) != 0 {
			t.Errorf("tx %d: queue size mismatch: have %d, want %d", i, pool.queue[account].Len(), 0)
		}
	}
	if pool.all.Count() != int(testTxPoolConfig.AccountQueue+5) {
		t.Errorf("total transaction mismatch: have %d, want %d", pool.all.Count(), testTxPoolConfig.AccountQueue+5)
	}
	if err := validateEvents(events, int(testTxPoolConfig.AccountQueue+5)); err != nil {
		t.Fatalf("event firing failed: %v", err)
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
}

// Tests that if the transaction count belonging to multiple accounts go above
// some hard threshold, the higher transactions are dropped to prevent DOS
// attacks.
func TestTransactionPendingGlobalLimiting(t *testing.T) {
	t.Parallel()

	// Create the pool to test the limit enforcement with
	statedb := newTestTxPoolStateDb()
	blockchain := NewTestBlockChain(statedb)

	config := testTxPoolConfig
	config.GlobalSlots = config.AccountSlots * 10

	pool := NewTxPool(config, params.TestChainConfig, blockchain)
	defer pool.Stop()

	// Create a number of test accounts and fund them
	keys := make([]*ecdsa.PrivateKey, 5)
	for i := 0; i < len(keys); i++ {
		keys[i], _ = crypto.GenerateKey()
		testAddBalance(pool, crypto.PubkeyToAddress(keys[i].PublicKey), big.NewInt(1000000))
	}
	// Generate and queue a batch of transactions
	nonces := make(map[common.Address]uint64)

	txs := types.Transactions{}
	for _, key := range keys {
		addr := crypto.PubkeyToAddress(key.PublicKey)
		for j := 0; j < int(config.GlobalSlots)/len(keys)*2; j++ {
			txs = append(txs, transaction(nonces[addr], 100000, key))
			nonces[addr]++
		}
	}
	// Import the batch and verify that limits have been enforced
	pool.AddRemotesSync(txs)

	pending := 0
	for _, list := range pool.pending {
		pending += list.Len()
	}
	if pending > int(config.GlobalSlots) {
		t.Fatalf("total pending transactions overflow allowance: %d > %d", pending, config.GlobalSlots)
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
}

// Test the limit on transaction size is enforced correctly.
// This test verifies every transaction having allowed size
// is added to the pool, and longer transactions are rejected.
func TestTransactionAllowedTxSize(t *testing.T) {
	t.Parallel()

	// Create a test account and fund it
	pool, key := setupTxPool()
	defer pool.Stop()

	account := crypto.PubkeyToAddress(key.PublicKey)
	testAddBalance(pool, account, big.NewInt(1000000000))

	// Compute maximal data size for transactions (lower bound).
	//
	// It is assumed the fields in the transaction (except of the data) are:
	//   - nonce     <= 32 bytes
	//   - gasPrice  <= 32 bytes
	//   - gasLimit  <= 32 bytes
	//   - recipient == 20 bytes
	//   - value     <= 32 bytes
	//   - signature == 65 bytes
	// All those fields are summed up to at most 213 bytes.
	baseSize := uint64(213)
	dataSize := txMaxSize - baseSize

	// Try adding a transaction with maximal allowed size
	tx := pricedDataTransaction(0, pool.currentMaxGas, big.NewInt(1), key, dataSize)
	if err := pool.addRemoteSync(tx); err != nil {
		t.Fatalf("failed to add transaction of size %d, close to maximal: %v", int(tx.Size()), err)
	}
	// Try adding a transaction with random allowed size
	if err := pool.addRemoteSync(pricedDataTransaction(1, pool.currentMaxGas, big.NewInt(1), key, uint64(rand.IntN(int(dataSize))))); err != nil {
		t.Fatalf("failed to add transaction of random allowed size: %v", err)
	}
	// Try adding a transaction of minimal not allowed size
	if err := pool.addRemoteSync(pricedDataTransaction(2, pool.currentMaxGas, big.NewInt(1), key, txMaxSize)); err == nil {
		t.Fatalf("expected rejection on slightly oversize transaction")
	}
	// Try adding a transaction of random not allowed size
	if err := pool.addRemoteSync(pricedDataTransaction(2, pool.currentMaxGas, big.NewInt(1), key, dataSize+1+uint64(rand.IntN(10*txMaxSize)))); err == nil {
		t.Fatalf("expected rejection on oversize transaction")
	}
	// Run some sanity checks on the pool internals
	pending, queued := pool.Stats()
	if pending != 2 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 2)
	}
	if queued != 0 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 0)
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
}

// Tests that if transactions start being capped, transactions are also removed from 'all'
func TestTransactionCapClearsFromAll(t *testing.T) {
	t.Parallel()

	// Create the pool to test the limit enforcement with
	statedb := newTestTxPoolStateDb()
	blockchain := NewTestBlockChain(statedb)

	config := testTxPoolConfig
	config.AccountSlots = 2
	config.AccountQueue = 2
	config.GlobalSlots = 8

	pool := NewTxPool(config, params.TestChainConfig, blockchain)
	defer pool.Stop()

	// Create a number of test accounts and fund them
	key, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(key.PublicKey)
	testAddBalance(pool, addr, big.NewInt(1000000))

	txs := types.Transactions{}
	for j := 0; j < int(config.GlobalSlots)*2; j++ {
		txs = append(txs, transaction(uint64(j), 100000, key))
	}
	// Import the batch and verify that limits have been enforced
	pool.AddRemotesSync(txs)
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
}

// Tests that if the transaction count belonging to multiple accounts go above
// some hard threshold, if they are under the minimum guaranteed slot count then
// the transactions are still kept.
func TestTransactionPendingMinimumAllowance(t *testing.T) {
	t.Parallel()

	// Create the pool to test the limit enforcement with
	statedb := newTestTxPoolStateDb()
	blockchain := NewTestBlockChain(statedb)

	config := testTxPoolConfig
	config.GlobalSlots = 1

	pool := NewTxPool(config, params.TestChainConfig, blockchain)
	defer pool.Stop()

	// Create a number of test accounts and fund them
	keys := make([]*ecdsa.PrivateKey, 5)
	for i := 0; i < len(keys); i++ {
		keys[i], _ = crypto.GenerateKey()
		testAddBalance(pool, crypto.PubkeyToAddress(keys[i].PublicKey), big.NewInt(1000000))
	}
	// Generate and queue a batch of transactions
	nonces := make(map[common.Address]uint64)

	txs := types.Transactions{}
	for _, key := range keys {
		addr := crypto.PubkeyToAddress(key.PublicKey)
		for j := 0; j < int(config.AccountSlots)*2; j++ {
			txs = append(txs, transaction(nonces[addr], 100000, key))
			nonces[addr]++
		}
	}
	// Import the batch and verify that limits have been enforced
	pool.AddRemotesSync(txs)

	for addr, list := range pool.pending {
		if list.Len() != int(config.AccountSlots) {
			t.Errorf("addr %x: total pending transactions mismatch: have %d, want %d", addr, list.Len(), config.AccountSlots)
		}
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
}

// Tests that setting the transaction pool gas price to a higher value correctly
// discards everything cheaper than that and moves any gapped transactions back
// from the pending pool to the queue.
//
// Note, local transactions are never allowed to be dropped.
func TestTransactionPoolRepricing(t *testing.T) {
	t.Parallel()

	// Create the pool to test the pricing enforcement with
	statedb := newTestTxPoolStateDb()
	blockchain := NewTestBlockChain(statedb)

	pool := NewTxPool(testTxPoolConfig, params.TestChainConfig, blockchain)
	defer pool.Stop()

	// Keep track of transaction events to ensure all executables get announced
	events := make(chan NewTxsNotify, 32)
	sub := pool.txFeed.Subscribe(events)
	defer sub.Unsubscribe()

	// Create a number of test accounts and fund them
	keys := make([]*ecdsa.PrivateKey, 4)
	for i := 0; i < len(keys); i++ {
		keys[i], _ = crypto.GenerateKey()
		testAddBalance(pool, crypto.PubkeyToAddress(keys[i].PublicKey), big.NewInt(1000000))
	}
	// Generate and queue a batch of transactions, both pending and queued
	txs := types.Transactions{}

	txs = append(txs, pricedTransaction(0, 100000, big.NewInt(2), keys[0]))
	txs = append(txs, pricedTransaction(1, 100000, big.NewInt(1), keys[0]))
	txs = append(txs, pricedTransaction(2, 100000, big.NewInt(2), keys[0]))

	txs = append(txs, pricedTransaction(0, 100000, big.NewInt(1), keys[1]))
	txs = append(txs, pricedTransaction(1, 100000, big.NewInt(2), keys[1]))
	txs = append(txs, pricedTransaction(2, 100000, big.NewInt(2), keys[1]))

	txs = append(txs, pricedTransaction(1, 100000, big.NewInt(2), keys[2]))
	txs = append(txs, pricedTransaction(2, 100000, big.NewInt(1), keys[2]))
	txs = append(txs, pricedTransaction(3, 100000, big.NewInt(2), keys[2]))

	ltx := pricedTransaction(0, 100000, big.NewInt(1), keys[3])

	// Import the batch and that both pending and queued transactions match up
	if errs := pool.AddRemotesSync(txs); errors.Join(errs...) != nil {
		t.Fatalf("failed to add remote transactions: %v", errors.Join(errs...))
	}
	if err := pool.AddLocal(ltx); err != nil {
		t.Fatalf("failed to add local transaction: %v", err)
	}

	pending, queued := pool.Stats()
	if pending != 7 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 7)
	}
	if queued != 3 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 3)
	}
	if err := validateEvents(events, 7); err != nil {
		t.Fatalf("original event firing failed: %v", err)
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
	// Reprice the pool and check that underpriced transactions get dropped
	pool.SetGasPrice(big.NewInt(2))

	pending, queued = pool.Stats()
	if pending != 2 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 2)
	}
	if queued != 5 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 5)
	}
	if err := validateEvents(events, 0); err != nil {
		t.Fatalf("reprice event firing failed: %v", err)
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
	// Check that we can't add the old transactions back
	if err := pool.AddRemote(pricedTransaction(1, 100000, big.NewInt(1), keys[0])); err != ErrUnderpriced {
		t.Fatalf("adding underpriced pending transaction error mismatch: have %v, want %v", err, ErrUnderpriced)
	}
	if err := pool.AddRemote(pricedTransaction(0, 100000, big.NewInt(1), keys[1])); err != ErrUnderpriced {
		t.Fatalf("adding underpriced pending transaction error mismatch: have %v, want %v", err, ErrUnderpriced)
	}
	if err := pool.AddRemote(pricedTransaction(2, 100000, big.NewInt(1), keys[2])); err != ErrUnderpriced {
		t.Fatalf("adding underpriced queued transaction error mismatch: have %v, want %v", err, ErrUnderpriced)
	}
	if err := validateEvents(events, 0); err != nil {
		t.Fatalf("post-reprice event firing failed: %v", err)
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
	// However we can add local underpriced transactions
	tx := pricedTransaction(1, 100000, big.NewInt(1), keys[3])
	if err := pool.AddLocal(tx); err != nil {
		t.Fatalf("failed to add underpriced local transaction: %v", err)
	}
	if pending, _ = pool.Stats(); pending != 3 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 3)
	}
	if err := validateEvents(events, 1); err != nil {
		t.Fatalf("post-reprice local event firing failed: %v", err)
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
	// And we can fill gaps with properly priced transactions
	if err := pool.AddRemote(pricedTransaction(1, 100000, big.NewInt(2), keys[0])); err != nil {
		t.Fatalf("failed to add pending transaction: %v", err)
	}
	if err := pool.AddRemote(pricedTransaction(0, 100000, big.NewInt(2), keys[1])); err != nil {
		t.Fatalf("failed to add pending transaction: %v", err)
	}
	if err := pool.AddRemote(pricedTransaction(2, 100000, big.NewInt(2), keys[2])); err != nil {
		t.Fatalf("failed to add queued transaction: %v", err)
	}
	if err := validateEvents(events, 5); err != nil {
		t.Fatalf("post-reprice event firing failed: %v", err)
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
}

// Tests that setting the transaction pool gas price to a higher value correctly
// discards everything cheaper (legacy & dynamic fee) than that and moves any
// gapped transactions back from the pending pool to the queue.
//
// Note, local transactions are never allowed to be dropped.
func TestTransactionPoolRepricingDynamicFee(t *testing.T) {
	t.Parallel()

	// Create the pool to test the pricing enforcement with
	pool, _ := setupTxPoolWithConfig(eip1559Config)
	defer pool.Stop()

	// Keep track of transaction events to ensure all executables get announced
	events := make(chan NewTxsNotify, 32)
	sub := pool.txFeed.Subscribe(events)
	defer sub.Unsubscribe()

	// Create a number of test accounts and fund them
	keys := make([]*ecdsa.PrivateKey, 4)
	for i := 0; i < len(keys); i++ {
		keys[i], _ = crypto.GenerateKey()
		testAddBalance(pool, crypto.PubkeyToAddress(keys[i].PublicKey), big.NewInt(1000000))
	}
	// Generate and queue a batch of transactions, both pending and queued
	txs := types.Transactions{}

	txs = append(txs, pricedTransaction(0, 100000, big.NewInt(2), keys[0]))
	txs = append(txs, pricedTransaction(1, 100000, big.NewInt(1), keys[0]))
	txs = append(txs, pricedTransaction(2, 100000, big.NewInt(2), keys[0]))

	txs = append(txs, dynamicFeeTx(0, 100000, big.NewInt(2), big.NewInt(1), keys[1]))
	txs = append(txs, dynamicFeeTx(1, 100000, big.NewInt(3), big.NewInt(2), keys[1]))
	txs = append(txs, dynamicFeeTx(2, 100000, big.NewInt(3), big.NewInt(2), keys[1]))

	txs = append(txs, dynamicFeeTx(1, 100000, big.NewInt(2), big.NewInt(2), keys[2]))
	txs = append(txs, dynamicFeeTx(2, 100000, big.NewInt(1), big.NewInt(1), keys[2]))
	txs = append(txs, dynamicFeeTx(3, 100000, big.NewInt(2), big.NewInt(2), keys[2]))

	ltx := dynamicFeeTx(0, 100000, big.NewInt(2), big.NewInt(1), keys[3])

	// Import the batch and that both pending and queued transactions match up
	if errs := pool.AddRemotesSync(txs); errors.Join(errs...) != nil {
		t.Fatalf("failed to add transactions: %v", errs)
	}
	if err := pool.AddLocal(ltx); err != nil {
		t.Fatalf("failed to add local transaction: %v", err)
	}

	pending, queued := pool.Stats()
	if pending != 7 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 7)
	}
	if queued != 3 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 3)
	}
	if err := validateEvents(events, 7); err != nil {
		t.Fatalf("original event firing failed: %v", err)
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
	// Reprice the pool and check that underpriced transactions get dropped
	pool.SetGasPrice(big.NewInt(2))

	pending, queued = pool.Stats()
	if pending != 2 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 2)
	}
	if queued != 5 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 5)
	}
	if err := validateEvents(events, 0); err != nil {
		t.Fatalf("reprice event firing failed: %v", err)
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
	// Check that we can't add the old transactions back
	tx := pricedTransaction(1, 100000, big.NewInt(1), keys[0])
	if err := pool.AddRemote(tx); err != ErrUnderpriced {
		t.Fatalf("adding underpriced pending transaction error mismatch: have %v, want %v", err, ErrUnderpriced)
	}
	tx = dynamicFeeTx(0, 100000, big.NewInt(2), big.NewInt(1), keys[1])
	if err := pool.AddRemote(tx); err != ErrUnderpriced {
		t.Fatalf("adding underpriced pending transaction error mismatch: have %v, want %v", err, ErrUnderpriced)
	}
	tx = dynamicFeeTx(2, 100000, big.NewInt(1), big.NewInt(1), keys[2])
	if err := pool.AddRemote(tx); err != ErrUnderpriced {
		t.Fatalf("adding underpriced queued transaction error mismatch: have %v, want %v", err, ErrUnderpriced)
	}
	if err := validateEvents(events, 0); err != nil {
		t.Fatalf("post-reprice event firing failed: %v", err)
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
	// However we can add local underpriced transactions
	tx = dynamicFeeTx(1, 100000, big.NewInt(1), big.NewInt(1), keys[3])
	if err := pool.AddLocal(tx); err != nil {
		t.Fatalf("failed to add underpriced local transaction: %v", err)
	}
	if pending, _ = pool.Stats(); pending != 3 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 3)
	}
	if err := validateEvents(events, 1); err != nil {
		t.Fatalf("post-reprice local event firing failed: %v", err)
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
	// And we can fill gaps with properly priced transactions
	tx = pricedTransaction(1, 100000, big.NewInt(2), keys[0])
	if err := pool.AddRemote(tx); err != nil {
		t.Fatalf("failed to add pending transaction: %v", err)
	}
	tx = dynamicFeeTx(0, 100000, big.NewInt(3), big.NewInt(2), keys[1])
	if err := pool.AddRemote(tx); err != nil {
		t.Fatalf("failed to add pending transaction: %v", err)
	}
	tx = dynamicFeeTx(2, 100000, big.NewInt(2), big.NewInt(2), keys[2])
	if err := pool.AddRemote(tx); err != nil {
		t.Fatalf("failed to add queued transaction: %v", err)
	}
	if err := validateEvents(events, 5); err != nil {
		t.Fatalf("post-reprice event firing failed: %v", err)
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
}

// Tests that setting the transaction pool gas price to a higher value does not
// remove local transactions (legacy & dynamic fee).
func TestTransactionPoolRepricingKeepsLocals(t *testing.T) {
	t.Parallel()

	// Create the pool to test the pricing enforcement with
	statedb := newTestTxPoolStateDb()
	blockchain := NewTestBlockChain(statedb)

	pool := NewTxPool(testTxPoolConfig, eip1559Config, blockchain)
	defer pool.Stop()

	// Create a number of test accounts and fund them
	keys := make([]*ecdsa.PrivateKey, 3)
	for i := 0; i < len(keys); i++ {
		keys[i], _ = crypto.GenerateKey()
		testAddBalance(pool, crypto.PubkeyToAddress(keys[i].PublicKey), big.NewInt(1000*1000000))
	}
	// Create transaction (both pending and queued) with a linearly growing gasprice
	for i := uint64(0); i < 500; i++ {
		// Add pending transaction.
		pendingTx := pricedTransaction(i, 100000, big.NewInt(int64(i)), keys[2])
		if err := pool.AddLocal(pendingTx); err != nil {
			t.Fatal(err)
		}
		// Add queued transaction.
		queuedTx := pricedTransaction(i+501, 100000, big.NewInt(int64(i)), keys[2])
		if err := pool.AddLocal(queuedTx); err != nil {
			t.Fatal(err)
		}

		// Add pending dynamic fee transaction.
		pendingTx = dynamicFeeTx(i, 100000, big.NewInt(int64(i)+1), big.NewInt(int64(i)), keys[1])
		if err := pool.AddLocal(pendingTx); err != nil {
			t.Fatal(err)
		}
		// Add queued dynamic fee transaction.
		queuedTx = dynamicFeeTx(i+501, 100000, big.NewInt(int64(i)+1), big.NewInt(int64(i)), keys[1])
		if err := pool.AddLocal(queuedTx); err != nil {
			t.Fatal(err)
		}
	}
	pending, queued := pool.Stats()
	expPending, expQueued := 1000, 1000
	validate := func() {
		pending, queued = pool.Stats()
		if pending != expPending {
			t.Fatalf("pending transactions mismatched: have %d, want %d", pending, expPending)
		}
		if queued != expQueued {
			t.Fatalf("queued transactions mismatched: have %d, want %d", queued, expQueued)
		}

		if err := validateTxPoolInternals(pool); err != nil {
			t.Fatalf("pool internal state corrupted: %v", err)
		}
	}
	validate()

	// Reprice the pool and check that nothing is dropped
	pool.SetGasPrice(big.NewInt(2))
	validate()

	pool.SetGasPrice(big.NewInt(2))
	pool.SetGasPrice(big.NewInt(4))
	pool.SetGasPrice(big.NewInt(8))
	pool.SetGasPrice(big.NewInt(100))
	validate()
}

// Tests that when the pool reaches its global transaction limit, underpriced
// transactions are gradually shifted out for more expensive ones and any gapped
// pending transactions are moved into the queue.
//
// Note, local transactions are never allowed to be dropped.
func TestTransactionPoolUnderpricing(t *testing.T) {
	t.Parallel()

	// Create the pool to test the pricing enforcement with
	statedb := newTestTxPoolStateDb()
	blockchain := NewTestBlockChain(statedb)

	config := testTxPoolConfig
	config.GlobalSlots = 2
	config.GlobalQueue = 2

	pool := NewTxPool(config, params.TestChainConfig, blockchain)
	defer pool.Stop()

	// Keep track of transaction events to ensure all executables get announced
	events := make(chan NewTxsNotify, 32)
	sub := pool.txFeed.Subscribe(events)
	defer sub.Unsubscribe()

	// Create a number of test accounts and fund them
	keys := make([]*ecdsa.PrivateKey, 4)
	for i := 0; i < len(keys); i++ {
		keys[i], _ = crypto.GenerateKey()
		testAddBalance(pool, crypto.PubkeyToAddress(keys[i].PublicKey), big.NewInt(1000000))
	}
	// Generate and queue a batch of transactions, both pending and queued
	txs := types.Transactions{}

	txs = append(txs, pricedTransaction(0, 100000, big.NewInt(1), keys[0]))
	txs = append(txs, pricedTransaction(1, 100000, big.NewInt(2), keys[0]))

	txs = append(txs, pricedTransaction(1, 100000, big.NewInt(1), keys[1]))

	ltx := pricedTransaction(0, 100000, big.NewInt(1), keys[2])

	// Import the batch and that both pending and queued transactions match up
	if errs := pool.AddRemotes(txs); errors.Join(errs...) != nil {
		t.Fatalf("failed to add remote transactions: %v", errs)
	}
	if err := pool.AddLocal(ltx); err != nil {
		t.Fatalf("failed to add local transaction: %v", err)
	}

	pending, queued := pool.Stats()
	if pending != 3 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 3)
	}
	if queued != 1 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 1)
	}
	if err := validateEvents(events, 3); err != nil {
		t.Fatalf("original event firing failed: %v", err)
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
	// Ensure that adding an underpriced transaction on block limit fails
	if err := pool.AddRemote(pricedTransaction(0, 100000, big.NewInt(1), keys[1])); err != ErrUnderpriced {
		t.Fatalf("adding underpriced pending transaction error mismatch: have %v, want %v", err, ErrUnderpriced)
	}
	// Ensure that adding high priced transactions drops cheap ones, but not own
	if err := pool.AddRemote(pricedTransaction(0, 100000, big.NewInt(3), keys[1])); err != nil { // +K1:0 => -K1:1 => Pend K0:0, K0:1, K1:0, K2:0; Que -
		t.Fatalf("failed to add well priced transaction: %v", err)
	}
	if err := pool.AddRemote(pricedTransaction(2, 100000, big.NewInt(4), keys[1])); err != nil { // +K1:2 => -K0:0 => Pend K1:0, K2:0; Que K0:1 K1:2
		t.Fatalf("failed to add well priced transaction: %v", err)
	}
	if err := pool.AddRemote(pricedTransaction(3, 100000, big.NewInt(5), keys[1])); err != nil { // +K1:3 => -K0:1 => Pend K1:0, K2:0; Que K1:2 K1:3
		t.Fatalf("failed to add well priced transaction: %v", err)
	}
	pending, queued = pool.Stats()
	if pending != 2 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 2)
	}
	if queued != 2 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 2)
	}
	if err := validateEvents(events, 1); err != nil {
		t.Fatalf("additional event firing failed: %v", err)
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
	// Ensure that adding local transactions can push out even higher priced ones
	ltx = pricedTransaction(1, 100000, big.NewInt(0), keys[2])
	if err := pool.AddLocal(ltx); err != nil {
		t.Fatalf("failed to append underpriced local transaction: %v", err)
	}
	ltx = pricedTransaction(0, 100000, big.NewInt(0), keys[3])
	if err := pool.AddLocal(ltx); err != nil {
		t.Fatalf("failed to add new underpriced local transaction: %v", err)
	}
	pending, queued = pool.Stats()
	if pending != 3 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 3)
	}
	if queued != 1 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 1)
	}
	if err := validateEvents(events, 2); err != nil {
		t.Fatalf("local event firing failed: %v", err)
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
}

// Tests that more expensive transactions push out cheap ones from the pool, but
// without producing instability by creating gaps that start jumping transactions
// back and forth between queued/pending.
func TestTransactionPoolStableUnderpricing(t *testing.T) {
	t.Parallel()

	// Create the pool to test the pricing enforcement with
	statedb := newTestTxPoolStateDb()
	blockchain := NewTestBlockChain(statedb)

	config := testTxPoolConfig
	config.GlobalSlots = 128
	config.GlobalQueue = 0

	pool := NewTxPool(config, params.TestChainConfig, blockchain)
	defer pool.Stop()

	// Keep track of transaction events to ensure all executables get announced
	events := make(chan NewTxsNotify, 32)
	sub := pool.txFeed.Subscribe(events)
	defer sub.Unsubscribe()

	// Create a number of test accounts and fund them
	keys := make([]*ecdsa.PrivateKey, 2)
	for i := 0; i < len(keys); i++ {
		keys[i], _ = crypto.GenerateKey()
		testAddBalance(pool, crypto.PubkeyToAddress(keys[i].PublicKey), big.NewInt(1000000))
	}
	// Fill up the entire queue with the same transaction price points
	txs := types.Transactions{}
	for i := uint64(0); i < config.GlobalSlots; i++ {
		txs = append(txs, pricedTransaction(i, 100000, big.NewInt(1), keys[0]))
	}
	pool.AddRemotesSync(txs)

	pending, queued := pool.Stats()
	if pending != int(config.GlobalSlots) {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, config.GlobalSlots)
	}
	if queued != 0 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 0)
	}
	if err := validateEvents(events, int(config.GlobalSlots)); err != nil {
		t.Fatalf("original event firing failed: %v", err)
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
	// Ensure that adding high priced transactions drops a cheap, but doesn't produce a gap
	if err := pool.addRemoteSync(pricedTransaction(0, 100000, big.NewInt(3), keys[1])); err != nil {
		t.Fatalf("failed to add well priced transaction: %v", err)
	}
	pending, queued = pool.Stats()
	if pending != int(config.GlobalSlots) {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, config.GlobalSlots)
	}
	if queued != 0 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 0)
	}
	if err := validateEvents(events, 1); err != nil {
		t.Fatalf("additional event firing failed: %v", err)
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
}

// Tests that when the pool reaches its global transaction limit, underpriced
// transactions (legacy & dynamic fee) are gradually shifted out for more
// expensive ones and any gapped pending transactions are moved into the queue.
//
// Note, local transactions are never allowed to be dropped.
func TestTransactionPoolUnderpricingDynamicFee(t *testing.T) {
	t.Parallel()

	pool, _ := setupTxPoolWithConfig(eip1559Config)
	defer pool.Stop()

	pool.config.GlobalSlots = 2
	pool.config.GlobalQueue = 2

	// Keep track of transaction events to ensure all executables get announced
	events := make(chan NewTxsNotify, 32)
	sub := pool.txFeed.Subscribe(events)
	defer sub.Unsubscribe()

	// Create a number of test accounts and fund them
	keys := make([]*ecdsa.PrivateKey, 4)
	for i := 0; i < len(keys); i++ {
		keys[i], _ = crypto.GenerateKey()
		testAddBalance(pool, crypto.PubkeyToAddress(keys[i].PublicKey), big.NewInt(1000000))
	}

	// Generate and queue a batch of transactions, both pending and queued
	txs := types.Transactions{}

	txs = append(txs, dynamicFeeTx(0, 100000, big.NewInt(3), big.NewInt(2), keys[0]))
	txs = append(txs, pricedTransaction(1, 100000, big.NewInt(2), keys[0]))
	txs = append(txs, dynamicFeeTx(1, 100000, big.NewInt(2), big.NewInt(1), keys[1]))

	ltx := dynamicFeeTx(0, 100000, big.NewInt(2), big.NewInt(1), keys[2])

	// Import the batch and that both pending and queued transactions match up
	// Pend K0:0, K0:1; Que K1:1
	if errs := pool.AddRemotes(txs); errors.Join(errs...) != nil {
		t.Fatalf("failed to add remote transactions: %v", errs)
	}
	// +K2:0 => Pend K0:0, K0:1, K2:0; Que K1:1
	if err := pool.AddLocal(ltx); err != nil {
		t.Fatalf("failed to add local transaction: %v", err)
	}

	pending, queued := pool.Stats()
	if pending != 3 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 3)
	}
	if queued != 1 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 1)
	}
	if err := validateEvents(events, 3); err != nil {
		t.Fatalf("original event firing failed: %v", err)
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}

	// Ensure that adding an underpriced transaction fails
	tx := dynamicFeeTx(0, 100000, big.NewInt(2), big.NewInt(1), keys[1])
	if err := pool.AddRemote(tx); err != ErrUnderpriced { // Pend K0:0, K0:1, K2:0; Que K1:1
		t.Fatalf("adding underpriced pending transaction error mismatch: have %v, want %v", err, ErrUnderpriced)
	}

	// Ensure that adding high priced transactions drops cheap ones, but not own
	tx = pricedTransaction(0, 100000, big.NewInt(2), keys[1])
	if err := pool.AddRemote(tx); err != nil { // +K1:0, -K1:1 => Pend K0:0, K0:1, K1:0, K2:0; Que -
		t.Fatalf("failed to add well priced transaction: %v", err)
	}

	tx = pricedTransaction(2, 100000, big.NewInt(3), keys[1])
	if err := pool.AddRemote(tx); err != nil { // +K1:2, -K0:1 => Pend K0:0 K1:0, K2:0; Que K1:2
		t.Fatalf("failed to add well priced transaction: %v", err)
	}
	tx = dynamicFeeTx(3, 100000, big.NewInt(4), big.NewInt(1), keys[1])
	if err := pool.AddRemote(tx); err != nil { // +K1:3, -K1:0 => Pend K0:0 K2:0; Que K1:2 K1:3
		t.Fatalf("failed to add well priced transaction: %v", err)
	}
	pending, queued = pool.Stats()
	if pending != 2 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 2)
	}
	if queued != 2 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 2)
	}
	if err := validateEvents(events, 1); err != nil {
		t.Fatalf("additional event firing failed: %v", err)
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
	// Ensure that adding local transactions can push out even higher priced ones
	ltx = dynamicFeeTx(1, 100000, big.NewInt(0), big.NewInt(0), keys[2])
	if err := pool.AddLocal(ltx); err != nil {
		t.Fatalf("failed to append underpriced local transaction: %v", err)
	}
	ltx = dynamicFeeTx(0, 100000, big.NewInt(0), big.NewInt(0), keys[3])
	if err := pool.AddLocal(ltx); err != nil {
		t.Fatalf("failed to add new underpriced local transaction: %v", err)
	}
	pending, queued = pool.Stats()
	if pending != 3 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 3)
	}
	if queued != 1 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 1)
	}
	if err := validateEvents(events, 2); err != nil {
		t.Fatalf("local event firing failed: %v", err)
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
}

// Tests whether highest fee cap transaction is retained after a batch of high effective
// tip transactions are added and vice versa
func TestDualHeapEviction(t *testing.T) {
	t.Parallel()

	pool, _ := setupTxPoolWithConfig(eip1559Config)
	defer pool.Stop()

	pool.config.GlobalSlots = 10
	pool.config.GlobalQueue = 10

	var (
		highTip, highCap *types.Transaction
		baseFee          int
	)

	check := func(tx *types.Transaction, name string) {
		if pool.all.GetRemote(tx.Hash()) == nil {
			t.Fatalf("highest %s transaction evicted from the pool", name)
		}
	}

	add := func(urgent bool) {
		txs := make([]*types.Transaction, 20)
		for i := range txs {
			// Create a test accounts and fund it
			key, _ := crypto.GenerateKey()
			testAddBalance(pool, crypto.PubkeyToAddress(key.PublicKey), big.NewInt(1000000000000))
			if urgent {
				txs[i] = dynamicFeeTx(0, 100000, big.NewInt(int64(baseFee+1+i)), big.NewInt(int64(1+i)), key)
				highTip = txs[i]
			} else {
				txs[i] = dynamicFeeTx(0, 100000, big.NewInt(int64(baseFee+200+i)), big.NewInt(1), key)
				highCap = txs[i]
			}
		}
		pool.AddRemotesSync(txs)
		pending, queued := pool.Stats()
		if pending+queued != 20 {
			t.Fatalf("transaction count mismatch: have %d, want %d", pending+queued, 10)
		}
	}

	add(false)
	for baseFee = 0; baseFee <= 1000; baseFee += 100 {
		pool.priced.SetBaseFee(big.NewInt(int64(baseFee)))
		add(true)
		check(highCap, "fee cap")
		add(false)
		check(highTip, "effective tip")
	}

	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
}

// Tests that the pool rejects duplicate transactions.
func TestTransactionDeduplication(t *testing.T) {
	t.Parallel()

	// Create the pool to test the pricing enforcement with
	statedb := newTestTxPoolStateDb()
	blockchain := NewTestBlockChain(statedb)

	pool := NewTxPool(testTxPoolConfig, params.TestChainConfig, blockchain)
	defer pool.Stop()

	// Create a test account to add transactions with
	key, _ := crypto.GenerateKey()
	testAddBalance(pool, crypto.PubkeyToAddress(key.PublicKey), big.NewInt(1000000000))

	// Create a batch of transactions and add a few of them
	txs := make([]*types.Transaction, 16)
	for i := 0; i < len(txs); i++ {
		txs[i] = pricedTransaction(uint64(i), 100000, big.NewInt(1), key)
	}
	var firsts []*types.Transaction
	for i := 0; i < len(txs); i += 2 {
		firsts = append(firsts, txs[i])
	}
	errs := pool.AddRemotesSync(firsts)
	if len(errs) != len(firsts) {
		t.Fatalf("first add mismatching result count: have %d, want %d", len(errs), len(firsts))
	}
	for i, err := range errs {
		if err != nil {
			t.Errorf("add %d failed: %v", i, err)
		}
	}
	pending, queued := pool.Stats()
	if pending != 1 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 1)
	}
	if queued != len(txs)/2-1 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, len(txs)/2-1)
	}
	// Try to add all of them now and ensure previous ones error out as knowns
	errs = pool.AddRemotesSync(txs)
	if len(errs) != len(txs) {
		t.Fatalf("all add mismatching result count: have %d, want %d", len(errs), len(txs))
	}
	for i, err := range errs {
		if i%2 == 0 && err == nil {
			t.Errorf("add %d succeeded, should have failed as known", i)
		}
		if i%2 == 1 && err != nil {
			t.Errorf("add %d failed: %v", i, err)
		}
	}
	pending, queued = pool.Stats()
	if pending != len(txs) {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, len(txs))
	}
	if queued != 0 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 0)
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
}

// Tests that the pool rejects replacement transactions that don't meet the minimum
// price bump required.
func TestTransactionReplacement(t *testing.T) {
	t.Parallel()

	// Create the pool to test the pricing enforcement with
	statedb := newTestTxPoolStateDb()
	blockchain := NewTestBlockChain(statedb)

	pool := NewTxPool(testTxPoolConfig, params.TestChainConfig, blockchain)
	defer pool.Stop()

	// Keep track of transaction events to ensure all executables get announced
	events := make(chan NewTxsNotify, 32)
	sub := pool.txFeed.Subscribe(events)
	defer sub.Unsubscribe()

	// Create a test account to add transactions with
	key, _ := crypto.GenerateKey()
	testAddBalance(pool, crypto.PubkeyToAddress(key.PublicKey), big.NewInt(1000000000))

	// Add pending transactions, ensuring the minimum price bump is enforced for replacement (for ultra low prices too)
	price := int64(100)
	threshold := (price * (100 + int64(testTxPoolConfig.PriceBump))) / 100

	if err := pool.addRemoteSync(pricedTransaction(0, 100000, big.NewInt(1), key)); err != nil {
		t.Fatalf("failed to add original cheap pending transaction: %v", err)
	}
	if err := pool.AddRemote(pricedTransaction(0, 100001, big.NewInt(1), key)); err != ErrReplaceUnderpriced {
		t.Fatalf("original cheap pending transaction replacement error mismatch: have %v, want %v", err, ErrReplaceUnderpriced)
	}
	if err := pool.AddRemote(pricedTransaction(0, 100000, big.NewInt(2), key)); err != nil {
		t.Fatalf("failed to replace original cheap pending transaction: %v", err)
	}
	if err := validateEvents(events, 2); err != nil {
		t.Fatalf("cheap replacement event firing failed: %v", err)
	}

	if err := pool.addRemoteSync(pricedTransaction(0, 100000, big.NewInt(price), key)); err != nil {
		t.Fatalf("failed to add original proper pending transaction: %v", err)
	}
	if err := pool.AddRemote(pricedTransaction(0, 100001, big.NewInt(threshold-1), key)); err != ErrReplaceUnderpriced {
		t.Fatalf("original proper pending transaction replacement error mismatch: have %v, want %v", err, ErrReplaceUnderpriced)
	}
	if err := pool.AddRemote(pricedTransaction(0, 100000, big.NewInt(threshold), key)); err != nil {
		t.Fatalf("failed to replace original proper pending transaction: %v", err)
	}
	if err := validateEvents(events, 2); err != nil {
		t.Fatalf("proper replacement event firing failed: %v", err)
	}

	// Add queued transactions, ensuring the minimum price bump is enforced for replacement (for ultra low prices too)
	if err := pool.AddRemote(pricedTransaction(2, 100000, big.NewInt(1), key)); err != nil {
		t.Fatalf("failed to add original cheap queued transaction: %v", err)
	}
	if err := pool.AddRemote(pricedTransaction(2, 100001, big.NewInt(1), key)); err != ErrReplaceUnderpriced {
		t.Fatalf("original cheap queued transaction replacement error mismatch: have %v, want %v", err, ErrReplaceUnderpriced)
	}
	if err := pool.AddRemote(pricedTransaction(2, 100000, big.NewInt(2), key)); err != nil {
		t.Fatalf("failed to replace original cheap queued transaction: %v", err)
	}

	if err := pool.AddRemote(pricedTransaction(2, 100000, big.NewInt(price), key)); err != nil {
		t.Fatalf("failed to add original proper queued transaction: %v", err)
	}
	if err := pool.AddRemote(pricedTransaction(2, 100001, big.NewInt(threshold-1), key)); err != ErrReplaceUnderpriced {
		t.Fatalf("original proper queued transaction replacement error mismatch: have %v, want %v", err, ErrReplaceUnderpriced)
	}
	if err := pool.AddRemote(pricedTransaction(2, 100000, big.NewInt(threshold), key)); err != nil {
		t.Fatalf("failed to replace original proper queued transaction: %v", err)
	}

	if err := validateEvents(events, 0); err != nil {
		t.Fatalf("queued replacement event firing failed: %v", err)
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
}

// Tests that the pool rejects replacement dynamic fee transactions that don't
// meet the replacement policy
func TestTransactionReplacementDynamicFee(t *testing.T) {
	t.Parallel()

	pool, key := setupTxPoolWithConfig(eip1559Config)
	defer pool.Stop()
	testAddBalance(pool, crypto.PubkeyToAddress(key.PublicKey), big.NewInt(1000000000))

	gasFeeCap := int64(100)
	gasTipCap := int64(60)

	tests := map[string]struct {
		originalTx    *types.Transaction
		replacementTx *types.Transaction
		expectedErr   error
	}{
		"Reject not bumping tip and fee cap": {
			originalTx:    dynamicFeeTx(0, 100000, big.NewInt(gasFeeCap), big.NewInt(gasTipCap), key),
			replacementTx: dynamicFeeTx(0, 100001, big.NewInt(gasFeeCap), big.NewInt(gasTipCap), key),
			expectedErr:   ErrReplaceUnderpriced,
		},
		"Reject bumping fee cap only": {
			originalTx:    dynamicFeeTx(1, 100000, big.NewInt(gasFeeCap), big.NewInt(gasTipCap), key),
			replacementTx: dynamicFeeTx(1, 100000, big.NewInt(gasFeeCap+1), big.NewInt(gasTipCap), key),
			expectedErr:   ErrReplaceUnderpriced,
		},
		"Accept bumping tip only": {
			originalTx:    dynamicFeeTx(2, 100000, big.NewInt(gasFeeCap), big.NewInt(gasTipCap), key),
			replacementTx: dynamicFeeTx(2, 100000, big.NewInt(gasFeeCap), big.NewInt(gasTipCap+6), key),
			expectedErr:   nil,
		},
		"Accept bumping both": {
			originalTx:    dynamicFeeTx(3, 100000, big.NewInt(gasFeeCap), big.NewInt(gasTipCap), key),
			replacementTx: dynamicFeeTx(3, 100000, big.NewInt(gasFeeCap+10), big.NewInt(gasTipCap+6), key),
			expectedErr:   nil,
		},
		"Reject Tip larger than Fee Cap": {
			originalTx:    dynamicFeeTx(4, 100000, big.NewInt(gasFeeCap), big.NewInt(gasFeeCap), key),
			replacementTx: dynamicFeeTx(4, 100000, big.NewInt(gasFeeCap), big.NewInt(gasFeeCap+10), key),
			expectedErr:   ErrTipAboveFeeCap,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := pool.AddRemote(test.originalTx)
			require.NoError(t, err)

			err = pool.AddRemote(test.replacementTx)
			require.Equal(t, test.expectedErr, err)
		})
	}
}

func TestTransactionPool_FeedAnnouncesChanges(t *testing.T) {
	t.Parallel()

	// Create the pool to test the pricing enforcement with
	pool, key := setupTxPoolWithConfig(eip1559Config)
	defer pool.Stop()
	testAddBalance(pool, crypto.PubkeyToAddress(key.PublicKey), big.NewInt(1000000000))

	// Keep track of transaction feed to ensure all executables get announced
	feed := make(chan NewTxsNotify, 32)
	sub := pool.txFeed.Subscribe(feed)
	defer sub.Unsubscribe()

	// Add a transaction and ensure it's announced
	err := pool.AddLocal(dynamicFeeTx(0, 100000, big.NewInt(2), big.NewInt(1), key))
	require.NoError(t, err)
	require.NoError(t, validateEvents(feed, 1))

	// Add a replacement transaction and ensure it's announced
	err = pool.AddLocal(dynamicFeeTx(0, 100000, big.NewInt(3), big.NewInt(3), key))
	require.NoError(t, err)
	require.NoError(t, validateEvents(feed, 1))

	// Add a future transaction and ensure it's not announced
	err = pool.AddLocal(dynamicFeeTx(2, 100000, big.NewInt(3), big.NewInt(3), key))
	require.NoError(t, err)
	require.NoError(t, validateEvents(feed, 0))

	// Add the missing nonce and ensure both are announced
	err = pool.AddLocal(dynamicFeeTx(1, 100000, big.NewInt(3), big.NewInt(3), key))
	require.NoError(t, err)
	require.NoError(t, validateEvents(feed, 2))
}

// Tests that local transactions are journaled to disk, but remote transactions
// get discarded between restarts.
func TestTransactionJournaling(t *testing.T)         { testTransactionJournaling(t, false) }
func TestTransactionJournalingNoLocals(t *testing.T) { testTransactionJournaling(t, true) }

func testTransactionJournaling(t *testing.T, nolocals bool) {
	t.Parallel()

	// Create a temporary file for the journal
	file, err := os.CreateTemp("", "")
	if err != nil {
		t.Fatalf("failed to create temporary journal: %v", err)
	}
	journal := file.Name()

	// Clean up the temporary file, we only need the path for now
	if err := file.Close(); err != nil {
		t.Fatalf("failed to close temporary journal: %v", err)
	}
	if err := os.Remove(journal); err != nil {
		t.Fatalf("failed to remove temporary journal: %v", err)
	}

	// Create the original pool to inject transaction into the journal
	statedb := newTestTxPoolStateDb()
	blockchain := NewTestBlockChain(statedb)

	config := testTxPoolConfig
	config.NoLocals = nolocals
	config.Journal = journal
	config.Rejournal = time.Second

	pool := NewTxPool(config, params.TestChainConfig, blockchain)

	// Create two test accounts to ensure remotes expire but locals do not
	local, _ := crypto.GenerateKey()
	remote, _ := crypto.GenerateKey()

	testAddBalance(pool, crypto.PubkeyToAddress(local.PublicKey), big.NewInt(1000000000))
	testAddBalance(pool, crypto.PubkeyToAddress(remote.PublicKey), big.NewInt(1000000000))

	// Add three local and a remote transactions and ensure they are queued up
	if err := pool.AddLocal(pricedTransaction(0, 100000, big.NewInt(1), local)); err != nil {
		t.Fatalf("failed to add local transaction: %v", err)
	}
	if err := pool.AddLocal(pricedTransaction(1, 100000, big.NewInt(1), local)); err != nil {
		t.Fatalf("failed to add local transaction: %v", err)
	}
	if err := pool.AddLocal(pricedTransaction(2, 100000, big.NewInt(1), local)); err != nil {
		t.Fatalf("failed to add local transaction: %v", err)
	}
	if err := pool.addRemoteSync(pricedTransaction(0, 100000, big.NewInt(1), remote)); err != nil {
		t.Fatalf("failed to add remote transaction: %v", err)
	}
	pending, queued := pool.Stats()
	if pending != 4 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 4)
	}
	if queued != 0 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 0)
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
	// Terminate the old pool, bump the local nonce, create a new pool and ensure relevant transaction survive
	pool.Stop()
	statedb.nonces[crypto.PubkeyToAddress(local.PublicKey)] = 1
	blockchain = NewTestBlockChain(statedb)

	pool = NewTxPool(config, params.TestChainConfig, blockchain)

	pending, queued = pool.Stats()
	if queued != 0 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 0)
	}
	if nolocals {
		if pending != 0 {
			t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 0)
		}
	} else {
		if pending != 2 {
			t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 2)
		}
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
	// Bump the nonce temporarily and ensure the newly invalidated transaction is removed
	statedb.nonces[crypto.PubkeyToAddress(local.PublicKey)] = 2
	<-pool.requestReset(nil, nil)
	time.Sleep(2 * config.Rejournal)
	pool.Stop()

	statedb.nonces[crypto.PubkeyToAddress(local.PublicKey)] = 1
	blockchain = NewTestBlockChain(statedb)
	pool = NewTxPool(config, params.TestChainConfig, blockchain)

	pending, queued = pool.Stats()
	if pending != 0 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 0)
	}
	if nolocals {
		if queued != 0 {
			t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 0)
		}
	} else {
		if queued != 1 {
			t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 1)
		}
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
	pool.Stop()
}

// TestTransactionStatusCheck tests that the pool can correctly retrieve the
// pending status of individual transactions.
func TestTransactionStatusCheck(t *testing.T) {
	t.Parallel()

	// Create the pool to test the status retrievals with
	statedb := newTestTxPoolStateDb()
	blockchain := NewTestBlockChain(statedb)

	pool := NewTxPool(testTxPoolConfig, params.TestChainConfig, blockchain)
	defer pool.Stop()

	// Create the test accounts to check various transaction statuses with
	keys := make([]*ecdsa.PrivateKey, 3)
	for i := 0; i < len(keys); i++ {
		keys[i], _ = crypto.GenerateKey()
		testAddBalance(pool, crypto.PubkeyToAddress(keys[i].PublicKey), big.NewInt(1000000))
	}
	// Generate and queue a batch of transactions, both pending and queued
	txs := types.Transactions{}

	txs = append(txs, pricedTransaction(0, 100000, big.NewInt(1), keys[0])) // Pending only
	txs = append(txs, pricedTransaction(0, 100000, big.NewInt(1), keys[1])) // Pending and queued
	txs = append(txs, pricedTransaction(2, 100000, big.NewInt(1), keys[1]))
	txs = append(txs, pricedTransaction(2, 100000, big.NewInt(1), keys[2])) // Queued only

	// Import the transaction and ensure they are correctly added
	pool.AddRemotesSync(txs)

	pending, queued := pool.Stats()
	if pending != 2 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 2)
	}
	if queued != 2 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 2)
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
	// Retrieve the status of each transaction and validate them
	hashes := make([]common.Hash, len(txs))
	for i, tx := range txs {
		hashes[i] = tx.Hash()
	}
	hashes = append(hashes, common.Hash{})

	statuses := pool.Status(hashes)
	expect := []TxStatus{TxStatusPending, TxStatusPending, TxStatusQueued, TxStatusQueued, TxStatusUnknown}

	for i := 0; i < len(statuses); i++ {
		if statuses[i] != expect[i] {
			t.Errorf("transaction %d: status mismatch: have %v, want %v", i, statuses[i], expect[i])
		}
	}
}

// Test the transaction slots consumption is computed correctly
func TestTransactionSlotCount(t *testing.T) {
	t.Parallel()

	key, _ := crypto.GenerateKey()

	// Check that an empty transaction consumes a single slot
	smallTx := pricedDataTransaction(0, 0, big.NewInt(0), key, 0)
	if slots := numSlots(smallTx); slots != 1 {
		t.Fatalf("small transactions slot count mismatch: have %d want %d", slots, 1)
	}
	// Check that a large transaction consumes the correct number of slots
	bigTx := pricedDataTransaction(0, 0, big.NewInt(0), key, uint64(10*txSlotSize))
	if slots := numSlots(bigTx); slots != 11 {
		t.Fatalf("big transactions slot count mismatch: have %d want %d", slots, 11)
	}
}

func TestSampleHashes_AllExpectedTransactionsAreReturned(t *testing.T) {
	statedb := newTestTxPoolStateDb()
	blockchain := NewTestBlockChain(statedb)

	pool := NewTxPool(testTxPoolConfig, params.TestChainConfig, blockchain)
	defer pool.Stop()

	expectedPendingTxs := make(map[common.Hash]int)
	expectedQueuedTxs := make(map[common.Hash]int)
	for acc := 0; acc < 5; acc++ {
		account, _ := crypto.GenerateKey()
		testAddBalance(pool, crypto.PubkeyToAddress(account.PublicKey), big.NewInt(1000000))
		// add pending txs
		for i := uint64(0); i < 20; i++ {
			tx := pricedTransaction(i, 100000, big.NewInt(1), account)
			if err := pool.addRemoteSync(tx); err != nil {
				t.Fatalf("failed to add transaction: %v", err)
			}
			if i < 8 { // first eight pending expected
				expectedPendingTxs[tx.Hash()] = 0
			}
		}

		// add queued txs
		for i := uint64(0); i < 15; i++ {
			tx := pricedTransaction(i+1000, 100000, big.NewInt(1), account)
			if err := pool.addRemoteSync(tx); err != nil {
				t.Fatalf("failed to add transaction: %v", err)
			}
			if i < 2 { // first two queued expected
				expectedQueuedTxs[tx.Hash()] = 0
			}
		}
	}

	pending, queued := pool.stats()
	if pending != 100 || queued != 75 {
		t.Fatalf("failed to fill the pool, incorrect amount of pending/queued: %d/%d", pending, queued)
	}

	samplingTimes := 100
	for i := 0; i < samplingTimes; i++ {
		samplePending, sampleQueued := 0, 0
		for _, txHash := range pool.SampleHashes(100) {
			tx := pool.Get(txHash)
			if tx.Nonce() < 1000 {
				samplePending++
			} else {
				sampleQueued++
			}
			if _, contains := expectedPendingTxs[txHash]; contains {
				expectedPendingTxs[txHash]++
			}
			if _, contains := expectedQueuedTxs[txHash]; contains {
				expectedQueuedTxs[txHash]++
			}
		}

		if samplePending != 90 || sampleQueued != 10 {
			t.Errorf("incorrect amount of pending/queued in the sample: %d/%d", samplePending, sampleQueued)
		}
	}

	for txHash, occurrences := range expectedPendingTxs {
		tx := pool.Get(txHash)
		if occurrences != samplingTimes {
			t.Errorf("expected pending tx %x (nonce %d) present %d times in samples, expected 10", txHash, tx.Nonce(), occurrences)
		}
	}
	for txHash, occurrences := range expectedQueuedTxs {
		tx := pool.Get(txHash)
		if occurrences == 0 {
			t.Errorf("expected tx %x (nonce %d) missing in samples", txHash, tx.Nonce())
		}
		if occurrences > samplingTimes {
			t.Errorf("expected tx %x (nonce %d) present in samples in more occurences than expected", txHash, tx.Nonce())
		}
	}
}

func TestSampleHashesManySenders(t *testing.T) {
	statedb := newTestTxPoolStateDb()
	blockchain := NewTestBlockChain(statedb)

	pool := NewTxPool(testTxPoolConfig, params.TestChainConfig, blockchain)
	defer pool.Stop()

	expectedTxs := make(map[common.Hash]int)
	for acc := 0; acc < 10; acc++ {
		account, _ := crypto.GenerateKey()
		address := crypto.PubkeyToAddress(account.PublicKey)
		testAddBalance(pool, address, big.NewInt(1000000))
		// add pending txs
		for i := uint64(0); i < 4; i++ {
			tx := pricedTransaction(i, 100000, big.NewInt(1), account)
			if err := pool.addRemoteSync(tx); err != nil {
				t.Fatalf("failed to add transaction: %v", err)
			}
			if i == 0 { // first pending expected
				expectedTxs[tx.Hash()] = 0
			}
		}
	}

	pending, queued := pool.stats()
	if pending != 40 || queued != 0 {
		t.Fatalf("failed to fill the pool, incorrect amount of pending/queued: %d/%d", pending, queued)
	}

	samplingTimes := 100
	for i := 0; i < samplingTimes; i++ {
		samples := pool.SampleHashes(5)
		if len(samples) != 4 { // should get 4 pending + 1 queued (but we have no queued)
			t.Errorf("unexpected amount of returned txs - returned %d, expected 4", len(samples))
		}
		for _, txHash := range samples {
			if _, contains := expectedTxs[txHash]; contains {
				expectedTxs[txHash]++
			} else {
				t.Errorf("unexpected tx %x in the sample", txHash)
			}
		}
	}

	for txHash, occurrences := range expectedTxs {
		tx := pool.Get(txHash)
		if occurrences == 0 {
			t.Errorf("expected tx %x (nonce %d) missing in samples", txHash, tx.Nonce())
		}
		if occurrences > samplingTimes {
			t.Errorf("expected tx %x (nonce %d) present in samples in more occurrences than expected", txHash, tx.Nonce())
		}
	}
}

// Benchmarks the speed of validating the contents of the pending queue of the
// transaction pool.
func BenchmarkPendingDemotion100(b *testing.B)   { benchmarkPendingDemotion(b, 100) }
func BenchmarkPendingDemotion1000(b *testing.B)  { benchmarkPendingDemotion(b, 1000) }
func BenchmarkPendingDemotion10000(b *testing.B) { benchmarkPendingDemotion(b, 10000) }

func benchmarkPendingDemotion(b *testing.B, size int) {
	// Add a batch of transactions to a pool one by one
	pool, key := setupTxPool()
	defer pool.Stop()

	account := crypto.PubkeyToAddress(key.PublicKey)
	testAddBalance(pool, account, big.NewInt(1000000))

	for i := 0; i < size; i++ {
		tx := transaction(uint64(i), 100000, key)
		pool.promoteTx(account, tx.Hash(), tx)
	}
	// Benchmark the speed of pool validation
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pool.demoteUnexecutables()
	}
}

// Benchmarks the speed of scheduling the contents of the future queue of the
// transaction pool.
func BenchmarkFuturePromotion100(b *testing.B)   { benchmarkFuturePromotion(b, 100) }
func BenchmarkFuturePromotion1000(b *testing.B)  { benchmarkFuturePromotion(b, 1000) }
func BenchmarkFuturePromotion10000(b *testing.B) { benchmarkFuturePromotion(b, 10000) }

func benchmarkFuturePromotion(b *testing.B, size int) {
	// Add a batch of transactions to a pool one by one
	pool, key := setupTxPool()
	defer pool.Stop()

	account := crypto.PubkeyToAddress(key.PublicKey)
	testAddBalance(pool, account, big.NewInt(1000000))

	for i := 0; i < size; i++ {
		tx := transaction(uint64(1+i), 100000, key)
		_, err := pool.enqueueTx(tx.Hash(), tx, false, true)
		if err != nil {
			b.Fatalf("failed to enqueue transaction: %v", err)
		}
	}
	// Benchmark the speed of pool validation
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pool.promoteExecutables(nil)
	}
}

// Benchmarks the speed of batched transaction insertion.
func BenchmarkPoolBatchInsert100(b *testing.B)   { benchmarkPoolBatchInsert(b, 100, false) }
func BenchmarkPoolBatchInsert1000(b *testing.B)  { benchmarkPoolBatchInsert(b, 1000, false) }
func BenchmarkPoolBatchInsert10000(b *testing.B) { benchmarkPoolBatchInsert(b, 10000, false) }

func BenchmarkPoolBatchLocalInsert100(b *testing.B)   { benchmarkPoolBatchInsert(b, 100, true) }
func BenchmarkPoolBatchLocalInsert1000(b *testing.B)  { benchmarkPoolBatchInsert(b, 1000, true) }
func BenchmarkPoolBatchLocalInsert10000(b *testing.B) { benchmarkPoolBatchInsert(b, 10000, true) }

func benchmarkPoolBatchInsert(b *testing.B, size int, local bool) {
	// Generate a batch of transactions to enqueue into the pool
	pool, key := setupTxPool()
	defer pool.Stop()

	account := crypto.PubkeyToAddress(key.PublicKey)
	testAddBalance(pool, account, big.NewInt(1000000))

	batches := make([]types.Transactions, b.N)
	for i := 0; i < b.N; i++ {
		batches[i] = make(types.Transactions, size)
		for j := 0; j < size; j++ {
			batches[i][j] = transaction(uint64(size*i+j), 100000, key)
		}
	}
	// Benchmark importing the transactions into the queue
	b.ResetTimer()
	for _, batch := range batches {
		if local {
			pool.AddLocals(batch)
		} else {
			pool.AddRemotes(batch)
		}
	}
}

func BenchmarkInsertRemoteWithAllLocals(b *testing.B) {
	// Allocate keys for testing
	key, _ := crypto.GenerateKey()
	account := crypto.PubkeyToAddress(key.PublicKey)

	remoteKey, _ := crypto.GenerateKey()
	remoteAddr := crypto.PubkeyToAddress(remoteKey.PublicKey)

	locals := make([]*types.Transaction, 4096+1024) // Occupy all slots
	for i := 0; i < len(locals); i++ {
		locals[i] = transaction(uint64(i), 100000, key)
	}
	remotes := make([]*types.Transaction, 1000)
	for i := 0; i < len(remotes); i++ {
		remotes[i] = pricedTransaction(uint64(i), 100000, big.NewInt(2), remoteKey) // Higher gasprice
	}
	// Benchmark importing the transactions into the queue
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		pool, _ := setupTxPool()
		testAddBalance(pool, account, big.NewInt(100000000))
		for _, local := range locals {
			err := pool.AddLocal(local)
			if err != nil {
				b.Fatalf("failed to add local transaction: %v", err)
			}
		}

		b.StartTimer()
		// Assign a high enough balance for testing
		testAddBalance(pool, remoteAddr, big.NewInt(100000000))
		for i := 0; i < len(remotes); i++ {
			pool.AddRemotes([]*types.Transaction{remotes[i]})
		}
		pool.Stop()
	}
}
