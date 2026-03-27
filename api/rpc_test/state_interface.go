package rpctest

import (
	"maps"
	"math/big"
	"slices"

	"github.com/0xsoniclabs/carmen/go/common/witness"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/ethereum/go-ethereum/common"
	geth_state "github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/stateless"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie/utils"
	"github.com/holiman/uint256"
)

func (t testState) getAccount(addr common.Address) TestAccount {
	if acc, ok := t.state[addr]; ok {
		return acc
	}
	return TestAccount{}
}

func (t testState) setAccount(addr common.Address, acc TestAccount) {
	t.state[addr] = acc
}

// AccessEvents implements [state.StateDB].
func (t testState) AccessEvents() *geth_state.AccessEvents {
	return nil
}

// AddAddressToAccessList implements [state.StateDB].
func (t testState) AddAddressToAccessList(addr common.Address) {}

// AddBalance implements [state.StateDB].
func (t testState) AddBalance(addr common.Address, amount *uint256.Int, _ tracing.BalanceChangeReason) uint256.Int {
	acc := t.getAccount(addr)
	prev := new(uint256.Int)
	if acc.Balance != nil {
		prev.SetFromBig(acc.Balance)
	}
	if acc.Balance == nil {
		acc.Balance = new(big.Int)
	}
	acc.Balance.Add(acc.Balance, amount.ToBig())
	t.setAccount(addr, acc)
	return *prev
}

// AddLog implements [state.StateDB].
func (t testState) AddLog(_ *types.Log) {}

// AddPreimage implements [state.StateDB].
func (t testState) AddPreimage(_ common.Hash, _ []byte) {}

// AddRefund implements [state.StateDB].
func (t testState) AddRefund(_ uint64) {}

// AddSlotToAccessList implements [state.StateDB].
func (t testState) AddSlotToAccessList(addr common.Address, slot common.Hash) {}

// AddressInAccessList implements [state.StateDB].
func (t testState) AddressInAccessList(addr common.Address) bool {
	return false
}

// BeginBlock implements [state.StateDB].
func (t testState) BeginBlock(number uint64) {}

// Copy implements [state.StateDB].
func (t testState) Copy() state.StateDB {
	newState := make(map[common.Address]TestAccount, len(t.state))
	for addr, acc := range t.state {
		newAcc := TestAccount{
			Nonce: acc.Nonce,
			Code:  slices.Clone(acc.Code),
			Store: maps.Clone(acc.Store),
		}
		if acc.Balance != nil {
			newAcc.Balance = new(big.Int).Set(acc.Balance)
		}
		newState[addr] = newAcc
	}
	return testState{state: newState}
}

// CreateAccount implements [state.StateDB].
func (t testState) CreateAccount(addr common.Address) {
	t.state[addr] = TestAccount{}
}

// CreateContract implements [state.StateDB].
func (t testState) CreateContract(addr common.Address) {
	t.state[addr] = TestAccount{}
}

// Empty implements [state.StateDB].
func (t testState) Empty(addr common.Address) bool {
	acc, ok := t.state[addr]
	if !ok {
		return true
	}
	return acc.Nonce == 0 &&
		(acc.Balance == nil || acc.Balance.Sign() == 0) &&
		len(acc.Code) == 0
}

// EndBlock implements [state.StateDB].
func (t testState) EndBlock(number uint64) <-chan error {
	ch := make(chan error)
	close(ch)
	return ch
}

// EndTransaction implements [state.StateDB].
func (t testState) EndTransaction() {}

// Error implements [state.StateDB].
func (t testState) Error() error {
	return nil
}

// Exist implements [state.StateDB].
func (t testState) Exist(addr common.Address) bool {
	_, ok := t.state[addr]
	return ok
}

// Finalise implements [state.StateDB].
func (t testState) Finalise(_ bool) {}

// GetBalance implements [state.StateDB].
func (t testState) GetBalance(addr common.Address) *uint256.Int {
	acc := t.getAccount(addr)
	if acc.Balance == nil {
		return new(uint256.Int)
	}
	result, _ := uint256.FromBig(acc.Balance)
	return result
}

// GetCode implements [state.StateDB].
func (t testState) GetCode(addr common.Address) []byte {
	return t.getAccount(addr).Code
}

// GetCodeHash implements [state.StateDB].
func (t testState) GetCodeHash(addr common.Address) common.Hash {
	if _, ok := t.state[addr]; !ok {
		return common.Hash{}
	}
	return crypto.Keccak256Hash(t.getAccount(addr).Code)
}

// GetCodeSize implements [state.StateDB].
func (t testState) GetCodeSize(addr common.Address) int {
	return len(t.getAccount(addr).Code)
}

// GetLogs implements [state.StateDB].
func (t testState) GetLogs(hash common.Hash, blockHash common.Hash) []*types.Log {
	return nil
}

// GetNonce implements [state.StateDB].
func (t testState) GetNonce(addr common.Address) uint64 {
	return t.getAccount(addr).Nonce
}

// GetProof implements [state.StateDB].
func (t testState) GetProof(addr common.Address, keys []common.Hash) (witness.Proof, error) {
	return nil, nil
}

// GetRefund implements [state.StateDB].
func (t testState) GetRefund() uint64 {
	return 0
}

// GetState implements [state.StateDB].
func (t testState) GetState(addr common.Address, key common.Hash) common.Hash {
	return t.getAccount(addr).Store[key]
}

// GetStateAndCommittedState implements [state.StateDB].
func (t testState) GetStateAndCommittedState(addr common.Address, key common.Hash) (common.Hash, common.Hash) {
	val := t.getAccount(addr).Store[key]
	return val, val
}

// GetStateHash implements [state.StateDB].
func (t testState) GetStateHash() common.Hash {
	return common.Hash{}
}

// GetStorageRoot implements [state.StateDB].
func (t testState) GetStorageRoot(addr common.Address) common.Hash {
	return common.Hash{}
}

// GetTransientState implements [state.StateDB].
func (t testState) GetTransientState(addr common.Address, key common.Hash) common.Hash {
	return common.Hash{}
}

// HasSelfDestructed implements [state.StateDB].
func (t testState) HasSelfDestructed(_ common.Address) bool {
	return false
}

// InterTxSnapshot implements [state.StateDB].
func (t testState) InterTxSnapshot() int {
	return 0
}

// PointCache implements [state.StateDB].
func (t testState) PointCache() *utils.PointCache {
	return nil
}

// Prepare implements [state.StateDB].
func (t testState) Prepare(rules params.Rules, sender common.Address, coinbase common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList) {
}

// Release implements [state.StateDB].
func (t testState) Release() {}

// RevertToInterTxSnapshot implements [state.StateDB].
func (t testState) RevertToInterTxSnapshot(id int) {}

// RevertToSnapshot implements [state.StateDB].
func (t testState) RevertToSnapshot(_ int) {}

// SelfDestruct implements [state.StateDB].
func (t testState) SelfDestruct(addr common.Address) uint256.Int {
	acc := t.getAccount(addr)
	prev := new(uint256.Int)
	if acc.Balance != nil {
		prev.SetFromBig(acc.Balance)
	}
	delete(t.state, addr)
	return *prev
}

// SelfDestruct6780 implements [state.StateDB].
func (t testState) SelfDestruct6780(addr common.Address) (uint256.Int, bool) {
	acc := t.getAccount(addr)
	prev := new(uint256.Int)
	if acc.Balance != nil {
		prev.SetFromBig(acc.Balance)
	}
	delete(t.state, addr)
	return *prev, true
}

// SetBalance implements [state.StateDB].
func (t testState) SetBalance(addr common.Address, amount *uint256.Int) {
	acc := t.getAccount(addr)
	acc.Balance = amount.ToBig()
	t.setAccount(addr, acc)
}

// SetCode implements [state.StateDB].
func (t testState) SetCode(addr common.Address, code []byte, _ tracing.CodeChangeReason) []byte {
	acc := t.getAccount(addr)
	prev := acc.Code
	acc.Code = code
	t.setAccount(addr, acc)
	return prev
}

// SetNonce implements [state.StateDB].
func (t testState) SetNonce(addr common.Address, nonce uint64, _ tracing.NonceChangeReason) {
	acc := t.getAccount(addr)
	acc.Nonce = nonce
	t.setAccount(addr, acc)
}

// SetState implements [state.StateDB].
func (t testState) SetState(addr common.Address, key common.Hash, value common.Hash) common.Hash {
	acc := t.getAccount(addr)
	prev := acc.Store[key]
	if acc.Store == nil {
		acc.Store = make(map[common.Hash]common.Hash)
	}
	acc.Store[key] = value
	t.setAccount(addr, acc)
	return prev
}

// SetStorage implements [state.StateDB].
func (t testState) SetStorage(addr common.Address, storage map[common.Hash]common.Hash) {
	acc := t.getAccount(addr)
	acc.Store = storage
	t.setAccount(addr, acc)
}

// SetTransientState implements [state.StateDB].
func (t testState) SetTransientState(addr common.Address, key common.Hash, value common.Hash) {}

// SetTxContext implements [state.StateDB].
func (t testState) SetTxContext(thash common.Hash, ti int) {}

// SlotInAccessList implements [state.StateDB].
func (t testState) SlotInAccessList(addr common.Address, slot common.Hash) (addressOk bool, slotOk bool) {
	return false, false
}

// Snapshot implements [state.StateDB].
func (t testState) Snapshot() int {
	return 0
}

// SubBalance implements [state.StateDB].
func (t testState) SubBalance(addr common.Address, amount *uint256.Int, _ tracing.BalanceChangeReason) uint256.Int {
	acc := t.getAccount(addr)
	prev := new(uint256.Int)
	if acc.Balance != nil {
		prev.SetFromBig(acc.Balance)
	}
	if acc.Balance == nil {
		acc.Balance = new(big.Int)
	}
	acc.Balance.Sub(acc.Balance, amount.ToBig())
	t.setAccount(addr, acc)
	return *prev
}

// SubRefund implements [state.StateDB].
func (t testState) SubRefund(_ uint64) {}

// TxIndex implements [state.StateDB].
func (t testState) TxIndex() int {
	return 0
}

// Witness implements [state.StateDB].
func (t testState) Witness() *stateless.Witness {
	return nil
}
