// Copyright 2025 Sonic Operations Ltd
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

package transactional_state

import (
	"errors"
	"maps"
	"slices"

	"github.com/0xsoniclabs/carmen/go/common/witness"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/ethereum/go-ethereum/common"
	eth_state "github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/stateless"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie/utils"
	"github.com/holiman/uint256"
	"golang.org/x/crypto/sha3"
)

type createdType int

const (
	notCreated createdType = iota
	createdAccount
	createdContract
)

type selfDestructType int

const (
	noSelfDestruct selfDestructType = iota
	selfDestructTypeRegular
	selfDestructType6780
)

type account struct {
	created        createdType
	balance        *uint256.Int
	nonce          uint64
	codeHash       common.Hash
	code           []byte
	storageRoot    common.Hash
	selfDestructed selfDestructType
}

func (a *account) IsEmpty() bool {
	return a.balance.IsZero() && a.nonce == 0 && a.codeHash == (common.Hash{}) // storageRoot is not considered
}

func (a *account) Clear() {
	a.balance = uint256.NewInt(0)
	a.nonce = 0
	a.codeHash = common.Hash{}
	a.code = nil
	a.storageRoot = common.Hash{}
}

type refundOp int

const (
	refundOpAdd refundOp = iota
	refundOpSub
)

type refundOperation struct {
	amount uint64
	op     refundOp
}

type dbTransaction struct {
	txHash  common.Hash
	txIndex int

	storages          map[common.Address]map[common.Hash]common.Hash
	accounts          map[common.Address]account
	refund            []refundOperation
	transientStorages map[common.Address]map[common.Hash]common.Hash
	accessList        map[common.Address]map[common.Hash]struct{}

	logs []*types.Log
}

func newDbTransaction(hash common.Hash, index int, lastTx *dbTransaction) *dbTransaction {
	if lastTx == nil {
		lastTx = &dbTransaction{
			storages: make(map[common.Address]map[common.Hash]common.Hash),
			accounts: make(map[common.Address]account),
		}
	}

	return &dbTransaction{
		txHash:  hash,
		txIndex: index,

		storages: maps.Clone(lastTx.storages),
		accounts: maps.Clone(lastTx.accounts),

		refund:            make([]refundOperation, 0),
		transientStorages: make(map[common.Address]map[common.Hash]common.Hash),
		accessList:        make(map[common.Address]map[common.Hash]struct{}),
		logs:              make([]*types.Log, 0),
	}
}

type TransactionalState struct {
	inner state.StateDB

	errors []error

	transactions []dbTransaction
	currentTx    *dbTransaction
}

func NewTransactionalState(inner state.StateDB) *TransactionalState {
	return &TransactionalState{
		inner:  inner,
		errors: make([]error, 0),
	}
}

func (t *TransactionalState) Commit() {

	for _, errs := range t.errors {
		log.Error("Error in db transaction", "error", errs)
	}

	// iterate through all transactions and commit their changes to the inner StateDB
	for _, tx := range t.transactions {

		t.inner.SetTxContext(tx.txHash, tx.txIndex)

		// commit accounts
		for addr, acct := range tx.accounts {

			switch acct.created {
			case createdAccount:
				t.inner.CreateAccount(addr)
			case createdContract:
				t.inner.CreateContract(addr)
			}

			t.inner.SetBalance(addr, acct.balance)
			t.inner.SetNonce(addr, acct.nonce, tracing.NonceChangeUnspecified)
			t.inner.SetCode(addr, acct.code, tracing.CodeChangeUnspecified)

			switch acct.selfDestructed {
			case selfDestructType6780:
				t.inner.SelfDestruct6780(addr)
			case selfDestructTypeRegular:
				t.inner.SelfDestruct(addr)
			}
		}

		// commit storage
		for addr, storage := range tx.storages {
			for key, value := range storage {
				t.inner.SetState(addr, key, value)
			}
		}

		// commit refund
		for _, refundOp := range tx.refund {
			switch refundOp.op {
			case refundOpAdd:
				t.inner.AddRefund(refundOp.amount)
			case refundOpSub:
				t.inner.SubRefund(refundOp.amount)
			}
		}

		// commit logs
		for _, log := range tx.logs {
			t.inner.AddLog(log)
		}

		t.inner.EndTransaction()
	}
}

func (t *TransactionalState) fetchAccount(addr common.Address) account {
	if t.currentTx == nil {
		return account{
			balance:     t.inner.GetBalance(addr),
			nonce:       t.inner.GetNonce(addr),
			codeHash:    t.inner.GetCodeHash(addr),
			code:        t.inner.GetCode(addr),
			storageRoot: t.inner.GetStorageRoot(addr),
		}
	}
	if _, ok := t.currentTx.accounts[addr]; !ok {
		t.currentTx.accounts[addr] = account{
			balance:     t.inner.GetBalance(addr),
			nonce:       t.inner.GetNonce(addr),
			codeHash:    t.inner.GetCodeHash(addr),
			code:        t.inner.GetCode(addr),
			storageRoot: t.inner.GetStorageRoot(addr),
		}
	}

	return t.currentTx.accounts[addr]
}

func (t *TransactionalState) updateAccount(addr common.Address, acct account) {
	if t.currentTx == nil {
		t.logError(errors.New("no current transaction to update account"))
		return
	}

	t.currentTx.accounts[addr] = acct
}

func (t *TransactionalState) logError(err error) {
	t.errors = append(t.errors, err)
}

// ////////////////////////
// =======================
// ////////////////////////

func (t *TransactionalState) Error() error {
	return errors.Join(t.errors...)
}

func (t *TransactionalState) GetLogs(txHash common.Hash, blockHash common.Hash) []*types.Log {

	idx := slices.IndexFunc(t.transactions, func(tx dbTransaction) bool {
		return tx.txHash == txHash
	})
	if idx == -1 {
		return nil
	}
	return t.transactions[idx].logs
}

func (t *TransactionalState) SetTxContext(txHash common.Hash, index int) {
	t.transactions = append(t.transactions, *newDbTransaction(txHash, index, t.currentTx))
	t.currentTx = &t.transactions[len(t.transactions)-1]
}

func (t *TransactionalState) TxIndex() int {
	if t.currentTx == nil {
		return -1
	}
	return t.currentTx.txIndex
}

func (t *TransactionalState) GetProof(addr common.Address, keys []common.Hash) (witness.Proof, error) {
	return t.inner.GetProof(addr, keys)
}

func (t *TransactionalState) SetBalance(addr common.Address, amount *uint256.Int) {
	acc := t.fetchAccount(addr)
	acc.balance = amount
	t.updateAccount(addr, acc)
}

func (t *TransactionalState) SetStorage(addr common.Address, storage map[common.Hash]common.Hash) {
	if t.currentTx == nil {
		t.logError(errors.New("no current transaction to set storage"))
		return
	}
	t.currentTx.storages[addr] = storage
}

func (t *TransactionalState) Copy() state.StateDB       { panic("not implemented") }
func (t *TransactionalState) GetStateHash() common.Hash { panic("not implemented") }
func (t *TransactionalState) BeginBlock(number uint64)  { panic("not implemented") }
func (t *TransactionalState) EndBlock(number uint64)    { panic("not implemented") }
func (t *TransactionalState) EndTransaction()           {}
func (t *TransactionalState) Release()                  {}

// ////////////////////////
// =======================
// ////////////////////////

func (t *TransactionalState) CreateAccount(addr common.Address) {
	t.updateAccount(addr, account{created: createdAccount})
}

func (t *TransactionalState) CreateContract(addr common.Address) {
	t.updateAccount(addr, account{created: createdContract})
}

func (t *TransactionalState) SubBalance(addr common.Address, sub *uint256.Int, _ tracing.BalanceChangeReason) uint256.Int {
	prev := t.GetBalance(addr)
	newBalance := uint256.NewInt(0).Sub(prev, sub)
	t.SetBalance(addr, newBalance)
	return *prev
}

func (t *TransactionalState) AddBalance(addr common.Address, add *uint256.Int, _ tracing.BalanceChangeReason) uint256.Int {
	prev := t.GetBalance(addr)
	newBalance := uint256.NewInt(0).Add(prev, add)
	t.SetBalance(addr, newBalance)
	return *prev
}

func (t *TransactionalState) GetBalance(addr common.Address) *uint256.Int {
	acc := t.fetchAccount(addr)
	return acc.balance
}

func (t *TransactionalState) GetNonce(addr common.Address) uint64 {
	acc := t.fetchAccount(addr)
	return acc.nonce
}

func (t *TransactionalState) SetNonce(addr common.Address, nonce uint64, _ tracing.NonceChangeReason) {
	acc := t.fetchAccount(addr)
	acc.nonce = nonce
	t.updateAccount(addr, acc)
}

func (t *TransactionalState) GetCodeHash(addr common.Address) common.Hash {
	acc := t.fetchAccount(addr)
	return acc.codeHash
}

func (t *TransactionalState) GetCode(addr common.Address) []byte {
	acc := t.fetchAccount(addr)
	return acc.code
}

func (t *TransactionalState) SetCode(addr common.Address, code []byte, _ tracing.CodeChangeReason) []byte {
	acc := t.fetchAccount(addr)
	prevCode := acc.code
	acc.code = code
	hasher := sha3.NewLegacyKeccak256()
	acc.codeHash = common.BytesToHash(hasher.Sum(code))
	t.updateAccount(addr, acc)
	return prevCode
}

func (t *TransactionalState) GetCodeSize(addr common.Address) int {
	code := t.GetCode(addr)
	return len(code)
}

func (t *TransactionalState) AddRefund(add uint64) {
	if t.currentTx == nil {
		t.logError(errors.New("no current transaction to add refund"))
		return
	}
	t.currentTx.refund = append(t.currentTx.refund, refundOperation{
		amount: add,
		op:     refundOpAdd,
	})
}

func (t *TransactionalState) SubRefund(sub uint64) {
	if t.currentTx == nil {
		t.logError(errors.New("no current transaction to add refund"))
		return
	}
	t.currentTx.refund = append(t.currentTx.refund, refundOperation{
		amount: sub,
		op:     refundOpSub,
	})
}

func (t *TransactionalState) GetRefund() uint64 {
	totalRefund := uint64(0)
	if t.currentTx != nil {
		for _, rlog := range t.currentTx.refund {
			switch rlog.op {
			case refundOpAdd:
				totalRefund += rlog.amount
			case refundOpSub:
				totalRefund -= rlog.amount
			}
		}
	}
	return totalRefund
}

func (t *TransactionalState) GetStateAndCommittedState(addr common.Address, hash common.Hash) (common.Hash, common.Hash) {
	_, committed := t.inner.GetStateAndCommittedState(addr, hash)
	current := t.GetState(addr, hash)
	return current, committed
}

func (t *TransactionalState) GetState(addr common.Address, hash common.Hash) common.Hash {
	if t.currentTx == nil {
		t.logError(errors.New("no current transaction to get state"))
		return common.Hash{}
	}
	if _, ok := t.currentTx.storages[addr]; !ok {
		t.currentTx.storages[addr] = make(map[common.Hash]common.Hash)
	}
	if _, ok := t.currentTx.storages[addr][hash]; !ok {
		t.currentTx.storages[addr][hash] = t.inner.GetState(addr, hash)
	}
	return t.currentTx.storages[addr][hash]
}

func (t *TransactionalState) SetState(addr common.Address, key common.Hash, value common.Hash) common.Hash {
	before := t.GetState(addr, key)
	if t.currentTx == nil {
		t.logError(errors.New("no current transaction to set state"))
		return common.Hash{}
	}
	if _, ok := t.currentTx.storages[addr]; !ok {
		t.currentTx.storages[addr] = make(map[common.Hash]common.Hash)
	}
	t.currentTx.storages[addr][key] = value
	return before
}

func (t *TransactionalState) GetStorageRoot(addr common.Address) common.Hash {
	acc := t.fetchAccount(addr)
	return acc.storageRoot
}

func (t *TransactionalState) GetTransientState(addr common.Address, key common.Hash) common.Hash {
	if t.currentTx == nil {
		t.logError(errors.New("no current transaction to get transient state"))
		return common.Hash{}
	}
	if _, ok := t.currentTx.transientStorages[addr]; !ok {
		t.currentTx.transientStorages[addr] = make(map[common.Hash]common.Hash)
	}
	return t.currentTx.transientStorages[addr][key]
}

func (t *TransactionalState) SetTransientState(addr common.Address, key, value common.Hash) {
	if t.currentTx == nil {
		t.logError(errors.New("no current transaction to set transient state"))
		return
	}
	if _, ok := t.currentTx.transientStorages[addr]; !ok {
		t.currentTx.transientStorages[addr] = make(map[common.Hash]common.Hash)
	}
	t.currentTx.transientStorages[addr][key] = value
}

func (t *TransactionalState) SelfDestruct(addr common.Address) uint256.Int {
	acc := t.fetchAccount(addr)
	prevBalance := acc.balance
	acc.selfDestructed = selfDestructTypeRegular
	t.updateAccount(addr, acc)
	return *prevBalance
}

func (t *TransactionalState) HasSelfDestructed(addr common.Address) bool {
	if t.currentTx == nil {
		t.logError(errors.New("no current transaction to check selfdestruct"))
		return false
	}
	acc, ok := t.currentTx.accounts[addr]
	if !ok {
		return false
	}
	return acc.selfDestructed != noSelfDestruct
}

func (t *TransactionalState) SelfDestruct6780(addr common.Address) (uint256.Int, bool) {
	acc := t.fetchAccount(addr)
	prevBalance := acc.balance
	acc.selfDestructed = selfDestructType6780
	t.updateAccount(addr, acc)
	// has this contract been created in this tx?
	return *prevBalance, acc.created == createdContract
}

func (t *TransactionalState) Exist(addr common.Address) bool {
	acc := t.fetchAccount(addr)
	return !acc.IsEmpty() || acc.selfDestructed != noSelfDestruct
}

func (t *TransactionalState) Empty(addr common.Address) bool {
	acc := t.fetchAccount(addr)
	return acc.IsEmpty()
}

func (t *TransactionalState) AddressInAccessList(addr common.Address) bool {
	if t.currentTx == nil {
		t.logError(errors.New("no current transaction to check access list"))
		return false
	}
	_, ok := t.currentTx.accessList[addr]
	return ok
}

func (t *TransactionalState) SlotInAccessList(addr common.Address, slot common.Hash) (addressOk bool, slotOk bool) {
	if t.currentTx == nil {
		t.logError(errors.New("no current transaction to check access list"))
		return false, false
	}
	slots, ok := t.currentTx.accessList[addr]
	if !ok {
		return false, false
	}
	_, slotOk = slots[slot]
	return true, slotOk
}

func (t *TransactionalState) AddAddressToAccessList(addr common.Address) {
	if t.currentTx == nil {
		t.logError(errors.New("no current transaction to add access list"))
		return
	}
	if _, ok := t.currentTx.accessList[addr]; !ok {
		t.currentTx.accessList[addr] = make(map[common.Hash]struct{})
	}
}

func (t *TransactionalState) AddSlotToAccessList(addr common.Address, slot common.Hash) {
	if t.currentTx == nil {
		t.logError(errors.New("no current transaction to add access list"))
		return
	}
	t.AddAddressToAccessList(addr)
	t.currentTx.accessList[addr][slot] = struct{}{}
}

func (t *TransactionalState) Prepare(
	rules params.Rules,
	sender, coinbase common.Address,
	dest *common.Address,
	precompiles []common.Address,
	txAccesses types.AccessList) {

	t.AddAddressToAccessList(sender)

	if dest != nil {
		t.AddAddressToAccessList(*dest)
	}
	for _, addr := range precompiles {
		t.AddAddressToAccessList(addr)
	}
	for _, el := range txAccesses {
		t.AddAddressToAccessList(el.Address)
		for _, key := range el.StorageKeys {
			t.AddSlotToAccessList(el.Address, key)
		}
	}
	if rules.IsShanghai {
		t.AddAddressToAccessList(coinbase)
	}
}

func (t *TransactionalState) RevertToSnapshot(int) {}

func (t *TransactionalState) Snapshot() int { return -1 }

func (t *TransactionalState) AddLog(log *types.Log) {
	if t.currentTx == nil {
		t.logError(errors.New("no current transaction to add log"))
		return
	}
	t.currentTx.logs = append(t.currentTx.logs, log)
}

func (t *TransactionalState) PointCache() *utils.PointCache {
	return t.inner.PointCache()
}

func (t *TransactionalState) AddPreimage(common.Hash, []byte) {
	t.logError(errors.New("add preimage is not implemented"))
}

func (t *TransactionalState) Witness() *stateless.Witness {
	return t.inner.Witness()
}

func (t *TransactionalState) AccessEvents() *eth_state.AccessEvents {
	return t.inner.AccessEvents()
}

func (t *TransactionalState) Finalise(bool) {
	// TransactionalEngine cannot finalise a block. The parent StateDB must do that.
	t.logError(errors.New("db transaction cannot be finalized"))
}
