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

package evmstore

import (
	"bytes"

	cc "github.com/0xsoniclabs/carmen/go/common"
	"github.com/0xsoniclabs/carmen/go/common/amount"
	"github.com/0xsoniclabs/carmen/go/common/witness"
	carmen "github.com/0xsoniclabs/carmen/go/state"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/ethereum/go-ethereum/common"
	geth_state "github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/stateless"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie/utils"
	"github.com/holiman/uint256"
)

const (
	// Number of address->curve point associations to keep.
	pointCacheSize = 4096
)

func CreateCarmenStateDb(carmenStateDb carmen.VmStateDB) state.StateDB {
	return &CarmenStateDB{
		db: carmenStateDb,
	}
}

type CarmenStateDB struct {
	db carmen.VmStateDB

	// current block - set by BeginBlock
	blockNum uint64

	// current transaction - set by Prepare
	txHash  common.Hash
	txIndex int

	// collecting all events accessing state information
	accessEvents *geth_state.AccessEvents
}

func (c *CarmenStateDB) Error() error {
	return nil
}

func (c *CarmenStateDB) AddLog(log *types.Log) {
	carmenLog := cc.Log{
		Address: cc.Address(log.Address),
		Topics:  nil,
		Data:    log.Data,
	}
	for _, topic := range log.Topics {
		carmenLog.Topics = append(carmenLog.Topics, cc.Hash(topic))
	}
	c.db.AddLog(&carmenLog)
}

func (c *CarmenStateDB) GetLogs(txHash common.Hash, blockHash common.Hash) []*types.Log {
	if txHash != c.txHash {
		panic("obtaining logs of not-current tx not supported")
	}
	carmenLogs := c.db.GetLogs()
	logs := make([]*types.Log, len(carmenLogs))
	for i, clog := range carmenLogs {
		log := &types.Log{
			Address:     common.Address(clog.Address),
			Topics:      nil,
			Data:        clog.Data,
			BlockNumber: c.blockNum,
			TxHash:      c.txHash,
			TxIndex:     uint(c.txIndex),
			BlockHash:   blockHash,
			Index:       clog.Index,
		}
		for _, topic := range clog.Topics {
			log.Topics = append(log.Topics, common.Hash(topic))
		}
		logs[i] = log
	}
	return logs
}

func (c *CarmenStateDB) Logs() []*types.Log {
	carmenLogs := c.db.GetLogs()
	logs := make([]*types.Log, len(carmenLogs))
	for i, clog := range carmenLogs {
		log := &types.Log{
			Address:     common.Address(clog.Address),
			Topics:      nil,
			Data:        clog.Data,
			BlockNumber: c.blockNum,
			TxHash:      c.txHash,
			TxIndex:     uint(c.txIndex),
			Index:       clog.Index,
		}
		for _, topic := range clog.Topics {
			log.Topics = append(log.Topics, common.Hash(topic))
		}
		logs[i] = log
	}
	return logs
}

func (c *CarmenStateDB) AddPreimage(hash common.Hash, preimage []byte) {
	// ignored - preimages of keys hashes are relevant only for geth trie
}

func (c *CarmenStateDB) AddRefund(gas uint64) {
	c.db.AddRefund(gas)
}

func (c *CarmenStateDB) SubRefund(gas uint64) {
	c.db.SubRefund(gas)
}

func (c *CarmenStateDB) Exist(addr common.Address) bool {
	return c.db.Exist(cc.Address(addr))
}

func (c *CarmenStateDB) Empty(addr common.Address) bool {
	return c.db.Empty(cc.Address(addr))
}

func (c *CarmenStateDB) GetBalance(addr common.Address) *uint256.Int {
	res := c.db.GetBalance(cc.Address(addr)).Uint256()
	return &res
}

func (c *CarmenStateDB) GetNonce(addr common.Address) uint64 {
	return c.db.GetNonce(cc.Address(addr))
}

func (c *CarmenStateDB) TxIndex() int {
	return c.txIndex
}

func (c *CarmenStateDB) GetCode(addr common.Address) []byte {
	return c.db.GetCode(cc.Address(addr))
}

func (c *CarmenStateDB) GetCodeSize(addr common.Address) int {
	return c.db.GetCodeSize(cc.Address(addr))
}

func (c *CarmenStateDB) GetCodeHash(addr common.Address) common.Hash {
	return common.Hash(c.db.GetCodeHash(cc.Address(addr)))
}

func (c *CarmenStateDB) GetState(addr common.Address, key common.Hash) common.Hash {
	return common.Hash(c.db.GetState(cc.Address(addr), cc.Key(key)))
}

func (c *CarmenStateDB) GetTransientState(addr common.Address, key common.Hash) common.Hash {
	return common.Hash(c.db.GetTransientState(cc.Address(addr), cc.Key(key)))
}

func (c *CarmenStateDB) GetProof(addr common.Address, keys []common.Hash) (witness.Proof, error) {
	if db, ok := c.db.(carmen.NonCommittableStateDB); ok {
		cKeys := make([]cc.Key, len(keys))
		for i, key := range keys {
			cKeys[i] = cc.Key(key)
		}
		return db.CreateWitnessProof(cc.Address(addr), cKeys...)
	} else {
		panic("unable get proof from not a NonCommittableStateDB")
	}
}

func (c *CarmenStateDB) GetStorageRoot(addr common.Address) common.Hash {
	empty := c.db.HasEmptyStorage(cc.Address(addr))
	var h common.Hash
	if !empty {
		// Carmen does not provide a method to get the storage root for performance reasons
		// as getting a storage root needs computation of hashes in the trie.
		// In practice, the method GetStorageRoot here is used in the EVM only to assess
		// if the storage is empty. For this reason, this method returns a dummy hash here just
		// not to equal to the empty hash when the storage is not empty.
		h[0] = 1
	}
	return h
}

func (c *CarmenStateDB) GetCommittedState(addr common.Address, hash common.Hash) common.Hash {
	return common.Hash(c.db.GetCommittedState(cc.Address(addr), cc.Key(hash)))
}

func (c *CarmenStateDB) HasSelfDestructed(addr common.Address) bool {
	return c.db.HasSuicided(cc.Address(addr))
}

func (c *CarmenStateDB) AddBalance(addr common.Address, value *uint256.Int, reason tracing.BalanceChangeReason) uint256.Int {
	before := c.db.GetBalance(cc.Address(addr)).Uint256()
	c.db.AddBalance(cc.Address(addr), amount.NewFromUint256(value))
	return before
}

func (c *CarmenStateDB) SubBalance(addr common.Address, value *uint256.Int, reason tracing.BalanceChangeReason) uint256.Int {
	before := c.db.GetBalance(cc.Address(addr)).Uint256()
	c.db.SubBalance(cc.Address(addr), amount.NewFromUint256(value))
	return before
}

func (c *CarmenStateDB) SetBalance(addr common.Address, balance *uint256.Int) {
	origBalance := c.db.GetBalance(cc.Address(addr)).Uint256()
	if origBalance.Cmp(balance) < 0 {
		c.db.AddBalance(cc.Address(addr), amount.NewFromUint256(new(uint256.Int).Sub(balance, &origBalance)))
	} else {
		c.db.SubBalance(cc.Address(addr), amount.NewFromUint256(new(uint256.Int).Sub(&origBalance, balance)))
	}
}

func (c *CarmenStateDB) SetNonce(addr common.Address, nonce uint64, _ tracing.NonceChangeReason) {
	c.db.SetNonce(cc.Address(addr), nonce)
}

func (c *CarmenStateDB) SetCode(addr common.Address, code []byte) []byte {
	old := bytes.Clone(c.db.GetCode(cc.Address(addr)))
	c.db.SetCode(cc.Address(addr), code)
	return old
}

func (c *CarmenStateDB) SetState(addr common.Address, key, value common.Hash) common.Hash {
	before := c.db.GetState(cc.Address(addr), cc.Key(key))
	c.db.SetState(cc.Address(addr), cc.Key(key), cc.Value(value))
	return common.Hash(before)
}

func (c *CarmenStateDB) SetTransientState(addr common.Address, key, value common.Hash) {
	c.db.SetTransientState(cc.Address(addr), cc.Key(key), cc.Value(value))
}

func (c *CarmenStateDB) SetStorage(addr common.Address, storage map[common.Hash]common.Hash) {
	origCode := c.db.GetCode(cc.Address(addr))
	origNonce := c.db.GetNonce(cc.Address(addr))
	origBalance := c.db.GetBalance(cc.Address(addr))

	// Suicide the account to clear the storage
	c.db.Suicide(cc.Address(addr))
	c.db.CreateAccount(cc.Address(addr))

	// insert new storage
	for key, value := range storage {
		c.db.SetState(cc.Address(addr), cc.Key(key), cc.Value(value))
	}

	// recover properties of the original account
	c.db.SetCode(cc.Address(addr), origCode)
	c.db.SetNonce(cc.Address(addr), origNonce)
	c.db.AddBalance(cc.Address(addr), origBalance)
}

func (c *CarmenStateDB) SelfDestruct(addr common.Address) uint256.Int {
	prevBalance := *c.GetBalance(addr)
	c.db.Suicide(cc.Address(addr))
	return prevBalance
}

func (c *CarmenStateDB) SelfDestruct6780(addr common.Address) (uint256.Int, bool) {
	prevBalance := *c.GetBalance(addr)
	return prevBalance, c.db.SuicideNewContract(cc.Address(addr))
}

func (c *CarmenStateDB) CreateAccount(addr common.Address) {
	c.db.CreateAccount(cc.Address(addr))
}

func (c *CarmenStateDB) CreateContract(addr common.Address) {
	c.db.CreateContract(cc.Address(addr))
}

func (c *CarmenStateDB) Copy() state.StateDB {
	if db, ok := c.db.(carmen.NonCommittableStateDB); ok {
		return CreateCarmenStateDb(db.Copy())
	} else {
		panic("unable to copy committable (live) StateDB")
	}
}

func (c *CarmenStateDB) Snapshot() int {
	return c.db.Snapshot()
}

func (c *CarmenStateDB) RevertToSnapshot(revid int) {
	c.db.RevertToSnapshot(revid)
}

func (c *CarmenStateDB) GetRefund() uint64 {
	return c.db.GetRefund()
}

func (c *CarmenStateDB) EndTransaction() {
	c.db.EndTransaction()
}

func (c *CarmenStateDB) Finalise(bool) {
	// ignored
}

// SetTxContext sets the current transaction hash and index which are
// used when the EVM emits new state logs.
func (c *CarmenStateDB) SetTxContext(txHash common.Hash, txIndex int) {
	c.txHash = txHash
	c.txIndex = txIndex
	c.db.ClearAccessList()
}

func (c *CarmenStateDB) BeginBlock(number uint64) {
	utils.NewPointCache(pointCacheSize)
	c.accessEvents = geth_state.NewAccessEvents(nil)
	c.blockNum = number
	if db, ok := c.db.(carmen.StateDB); ok {
		db.BeginBlock()
	}
}

func (c *CarmenStateDB) EndBlock(number uint64) {
	if db, ok := c.db.(carmen.StateDB); ok {
		db.EndBlock(number)
	}
}

func (c *CarmenStateDB) GetStateHash() common.Hash {
	return common.Hash(c.db.GetHash())
}

func (c *CarmenStateDB) Prepare(rules params.Rules, sender, coinbase common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList) {
	// TODO: consider rules of Paris and Cancun revisions
	c.db.ClearAccessList()
	c.db.AddAddressToAccessList(cc.Address(sender))
	if dest != nil {
		c.db.AddAddressToAccessList(cc.Address(*dest))
	}
	for _, addr := range precompiles {
		c.db.AddAddressToAccessList(cc.Address(addr))
	}
	for _, el := range txAccesses {
		c.db.AddAddressToAccessList(cc.Address(el.Address))
		for _, key := range el.StorageKeys {
			c.db.AddSlotToAccessList(cc.Address(el.Address), cc.Key(key))
		}
	}
	if rules.IsShanghai {
		c.db.AddAddressToAccessList(cc.Address(coinbase))
	}
}

func (c *CarmenStateDB) AddAddressToAccessList(addr common.Address) {
	c.db.AddAddressToAccessList(cc.Address(addr))
}

func (c *CarmenStateDB) AddSlotToAccessList(addr common.Address, slot common.Hash) {
	c.db.AddSlotToAccessList(cc.Address(addr), cc.Key(slot))
}

func (c *CarmenStateDB) AddressInAccessList(addr common.Address) bool {
	return c.db.IsAddressInAccessList(cc.Address(addr))
}

func (c *CarmenStateDB) SlotInAccessList(addr common.Address, slot common.Hash) (addressPresent bool, slotPresent bool) {
	return c.db.IsSlotInAccessList(cc.Address(addr), cc.Key(slot))
}

// PointCache returns the point cache used in computations of verkle trees
func (c *CarmenStateDB) PointCache() *utils.PointCache {
	return nil // used only when IsEIP4762 (verkle trees) enabled
}

// Witness retrieves the current state witness being collected
func (c *CarmenStateDB) Witness() *stateless.Witness {
	return nil // set to not-nil only when vmConfig.EnableWitnessCollection
}

func (c *CarmenStateDB) Release() {
	if db, ok := c.db.(carmen.NonCommittableStateDB); ok {
		db.Release()
	}
}

// AccessEvents returns an empty list of accessed states. In ethereum, this is used to
// collect the accessed states for the stateless client.
func (c *CarmenStateDB) AccessEvents() *geth_state.AccessEvents {
	return c.accessEvents
}
