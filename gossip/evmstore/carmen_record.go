package evmstore

import (
	"maps"

	cc "github.com/0xsoniclabs/carmen/go/common"
	"github.com/0xsoniclabs/carmen/go/common/witness"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/substate/substate"
	stypes "github.com/0xsoniclabs/substate/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/holiman/uint256"
)

// record-replay
// RecordCarmenStateDB is a wrapper around CarmenStateDB that records information for creation of the SubstateDb
type RecordCarmenStateDB struct {
	SubstatePreAlloc    substate.WorldState
	SubstatePostAlloc   substate.WorldState
	SubstateBlockHashes map[uint64]common.Hash
	AccessedStorage     map[common.Address]map[common.Hash]common.Hash

	*CarmenStateDB
}

func (c *RecordCarmenStateDB) Exist(addr common.Address) bool {
	c.substateRecordAccess(addr)
	return c.CarmenStateDB.Exist(addr)
}

func (c *RecordCarmenStateDB) Empty(addr common.Address) bool {
	c.substateRecordAccess(addr)
	return c.CarmenStateDB.Empty(addr)
}

func (c *RecordCarmenStateDB) GetBalance(addr common.Address) *uint256.Int {

	c.substateRecordAccess(addr)
	return c.CarmenStateDB.GetBalance(addr)
}

func (c *RecordCarmenStateDB) GetNonce(addr common.Address) uint64 {

	c.substateRecordAccess(addr)
	return c.CarmenStateDB.GetNonce(addr)
}

func (c *RecordCarmenStateDB) GetCode(addr common.Address) []byte {

	c.substateRecordAccess(addr)
	return c.CarmenStateDB.GetCode(addr)
}

func (c *RecordCarmenStateDB) GetCodeSize(addr common.Address) int {
	c.substateRecordAccess(addr)
	return c.CarmenStateDB.GetCodeSize(addr)
}

func (c *RecordCarmenStateDB) GetCodeHash(addr common.Address) common.Hash {
	c.substateRecordAccess(addr)
	return c.CarmenStateDB.GetCodeHash(addr)
}

func (c *RecordCarmenStateDB) GetState(addr common.Address, key common.Hash) common.Hash {
	value := c.CarmenStateDB.GetState(addr, key)
	c.substateRecordAccess(addr)
	c.substateStorageAccess(addr, key, value)
	return value
}

func (c *RecordCarmenStateDB) GetTransientState(addr common.Address, key common.Hash) common.Hash {
	c.substateRecordAccess(addr)
	return c.CarmenStateDB.GetTransientState(addr, key)
}

func (c *RecordCarmenStateDB) GetProof(addr common.Address, keys []common.Hash) (witness.Proof, error) {
	c.substateRecordAccess(addr)
	return c.CarmenStateDB.GetProof(addr, keys)
}

func (c *RecordCarmenStateDB) GetStorageRoot(addr common.Address) common.Hash {
	c.substateRecordAccess(addr)
	return c.CarmenStateDB.GetStorageRoot(addr)
}

func (c *RecordCarmenStateDB) GetCommittedState(addr common.Address, hash common.Hash) common.Hash {
	c.substateRecordAccess(addr)
	return c.CarmenStateDB.GetCommittedState(addr, hash)
}

func (c *RecordCarmenStateDB) HasSelfDestructed(addr common.Address) bool {
	c.substateRecordAccess(addr)
	return c.CarmenStateDB.HasSelfDestructed(addr)
}

func (c *RecordCarmenStateDB) AddBalance(addr common.Address, value *uint256.Int, reason tracing.BalanceChangeReason) {
	c.substateRecordAccess(addr)
	c.CarmenStateDB.AddBalance(addr, value, reason)
}

func (c *RecordCarmenStateDB) SubBalance(addr common.Address, value *uint256.Int, reason tracing.BalanceChangeReason) {
	c.substateRecordAccess(addr)
	c.CarmenStateDB.SubBalance(addr, value, reason)
}

func (c *RecordCarmenStateDB) SetBalance(addr common.Address, balance *uint256.Int) {
	c.substateRecordAccess(addr)
	c.CarmenStateDB.SetBalance(addr, balance)
}

func (c *RecordCarmenStateDB) SetNonce(addr common.Address, nonce uint64) {
	c.substateRecordAccess(addr)
	c.CarmenStateDB.SetNonce(addr, nonce)
}

func (c *RecordCarmenStateDB) SetCode(addr common.Address, code []byte) {
	c.substateRecordAccess(addr)
	c.CarmenStateDB.SetCode(addr, code)
}

func (c *RecordCarmenStateDB) SetState(addr common.Address, key, value common.Hash) {
	c.substateRecordAccess(addr)
	c.substateStorageAccess(addr, key, value)
	c.CarmenStateDB.SetState(addr, key, value)
}

func (c *RecordCarmenStateDB) SetTransientState(addr common.Address, key, value common.Hash) {
	c.substateRecordAccess(addr)
	c.CarmenStateDB.SetTransientState(addr, key, value)
}

func (c *RecordCarmenStateDB) SetStorage(addr common.Address, storage map[common.Hash]common.Hash) {
	c.substateRecordAccess(addr)
	for key, value := range storage {
		c.substateStorageAccess(addr, key, value)
	}
	c.CarmenStateDB.SetStorage(addr, storage)
}

func (c *RecordCarmenStateDB) SelfDestruct(addr common.Address) {
	c.substateRecordAccess(addr)
	c.CarmenStateDB.SelfDestruct(addr)
}

func (c *RecordCarmenStateDB) Selfdestruct6780(addr common.Address) {
	c.substateRecordAccess(addr)
	c.CarmenStateDB.Selfdestruct6780(addr)
}

func (c *RecordCarmenStateDB) CreateAccount(addr common.Address) {
	c.substateRecordAccess(addr)
	c.CarmenStateDB.CreateAccount(addr)
}

func (c *RecordCarmenStateDB) CreateContract(addr common.Address) {
	c.substateRecordAccess(addr)
	c.CarmenStateDB.CreateContract(addr)
}

func (c *RecordCarmenStateDB) Copy() state.StateDB {
	return &RecordCarmenStateDB{
		CarmenStateDB:       c.CarmenStateDB.Copy().(*CarmenStateDB),
		SubstatePreAlloc:    maps.Clone(c.SubstatePreAlloc),
		SubstatePostAlloc:   maps.Clone(c.SubstatePostAlloc),
		SubstateBlockHashes: maps.Clone(c.SubstateBlockHashes),
		AccessedStorage:     maps.Clone(c.AccessedStorage),
	}
}

func (c *RecordCarmenStateDB) Finalise() {
	dirtyAddresses := c.RecordPreFinalise()

	c.CarmenStateDB.Finalise()

	c.RecordPostFinalise(dirtyAddresses)
}

func (c *RecordCarmenStateDB) SetTxContext(txHash common.Hash, txIndex int) {
	c.SubstatePreAlloc = make(substate.WorldState)
	c.SubstatePostAlloc = make(substate.WorldState)
	c.SubstateBlockHashes = make(map[uint64]common.Hash)
	c.AccessedStorage = make(map[common.Address]map[common.Hash]common.Hash)

	c.CarmenStateDB.SetTxContext(txHash, txIndex)
}

func (c *RecordCarmenStateDB) GetSubstatePreAlloc() substate.WorldState {
	return c.SubstatePreAlloc
}

func (c *RecordCarmenStateDB) GetSubstatePostAlloc() substate.WorldState {
	return c.SubstatePostAlloc
}

func (c *RecordCarmenStateDB) GetSubstateBlockHashes() map[uint64]common.Hash {
	return c.SubstateBlockHashes
}

func (c *RecordCarmenStateDB) substateRecordAccess(addr common.Address) {
	if c.db.Exist(cc.Address(addr)) && !c.db.HasSuicided(cc.Address(addr)) {
		// insert the account in StateDB.SubstatePreAlloc
		if _, exist := c.SubstatePreAlloc[stypes.Address(addr)]; !exist {
			c.SubstatePreAlloc[stypes.Address(addr)] = substate.NewAccount(c.db.GetNonce(cc.Address(addr)), c.db.GetBalance(cc.Address(addr)).ToBig(), c.db.GetCode(cc.Address(addr)))
		}
	}

	// insert empty account in StateDB.SubstatePreAlloc
	// This will prevent insertion of new account created in txs
	if _, exist := c.SubstatePreAlloc[stypes.Address(addr)]; !exist {
		c.SubstatePreAlloc[stypes.Address(addr)] = nil
	}
}

func (c *RecordCarmenStateDB) substateStorageAccess(addr common.Address, key common.Hash, value common.Hash) {
	if l, found := c.AccessedStorage[addr]; found {
		if _, f2 := l[key]; !f2 {
			c.AccessedStorage[addr][key] = value
		}
	} else {
		c.AccessedStorage[addr] = make(map[common.Hash]common.Hash)
		c.AccessedStorage[addr][key] = value
	}
}

func (c *RecordCarmenStateDB) RecordPreFinalise() map[cc.Address]struct{} {
	dirtyAddresses := make(map[cc.Address]struct{})

	// copy original storage values to Prestate and Poststate
	for addr, sa := range c.SubstatePreAlloc {
		if sa == nil {
			dirtyAddresses[cc.Address(addr)] = struct{}{}
			delete(c.SubstatePreAlloc, addr)
			continue
		}

		comAddr := common.BytesToAddress(addr.Bytes())
		ac, found := c.AccessedStorage[comAddr]
		if found {
			for key := range ac {
				valueM, f := c.AccessedStorage[comAddr][key]
				if !f {
					panic("key not found in AccessedStorage")
				}
				value := c.GetCommittedState(comAddr, key)

				if value != valueM {
					panic("value mismatch")
				}

				sa.Storage[stypes.Hash(key)] = stypes.Hash(value)

			}
		}
		c.SubstatePostAlloc[addr] = sa.Copy()
	}
	return dirtyAddresses
}

func (c *RecordCarmenStateDB) RecordPostFinalise(dirtyAddresses map[cc.Address]struct{}) {
	for address := range dirtyAddresses {
		if c.db.Exist(address) {
			s := make(map[stypes.Hash]stypes.Hash)
			for key := range c.AccessedStorage[common.Address(address)] {
				s[stypes.Hash(key)] = stypes.Hash{}
			}
			c.SubstatePostAlloc[stypes.Address(address)] = &substate.Account{Storage: s}
		}
	}

	c.AccessedStorage = nil

	toDelete := make([]stypes.Address, 0)
	for address, acc := range c.SubstatePostAlloc {
		if c.db.HasSuicided(cc.Address(address)) {
			toDelete = append(toDelete, address)
			continue
		}

		// update the account in StateDB.SubstatePostAlloc
		acc.Balance = c.db.GetBalance(cc.Address(address)).ToBig()
		acc.Nonce = c.db.GetNonce(cc.Address(address))
		acc.Code = c.db.GetCode(cc.Address(address))
		storageToUpdate := make(map[stypes.Hash]stypes.Hash)
		for key := range acc.Storage {
			storageToUpdate[key] = stypes.Hash(c.db.GetState(cc.Address(address), cc.Key(key)))
		}
		acc.Storage = storageToUpdate
	}

	for _, address := range toDelete {
		delete(c.SubstatePostAlloc, address)
	}
}
