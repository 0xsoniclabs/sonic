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

package freeze

import (
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// StateDB wraps the state an EVM runs on and tracks whether a frozen
// account's funds moved. Moves in reverted snapshots do not count.
type StateDB struct {
	vm.StateDB
	checker       Checker
	frozenMoved   bool
	frozenMovedAt map[int]bool // frozenMoved when each snapshot was taken
}

// Wrap returns db with the frozen-move tracking used by the freeze, asking
// checker for the freeze status of addresses.
func Wrap(db vm.StateDB, checker Checker) *StateDB {
	return &StateDB{StateDB: db, checker: checker, frozenMovedAt: map[int]bool{}}
}

func (s *StateDB) Prepare(rules params.Rules, sender, coinbase common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList) {
	s.frozenMoved = false
	clear(s.frozenMovedAt)
	s.StateDB.Prepare(rules, sender, coinbase, dest, precompiles, txAccesses)
}

func (s *StateDB) Snapshot() int {
	id := s.StateDB.Snapshot()
	s.frozenMovedAt[id] = s.frozenMoved
	return id
}

func (s *StateDB) RevertToSnapshot(id int) {
	s.StateDB.RevertToSnapshot(id)
	s.frozenMoved = s.frozenMovedAt[id]
}

// --- native token transfers ---
// AddBalance and SubBalance track every non-zero balance change of a frozen
// account.

func (s *StateDB) AddBalance(addr common.Address, amount *uint256.Int, reason tracing.BalanceChangeReason) uint256.Int {
	if !amount.IsZero() && s.checker.IsFrozen(addr) {
		s.frozenMoved = true
	}
	return s.StateDB.AddBalance(addr, amount, reason)
}

func (s *StateDB) SubBalance(addr common.Address, amount *uint256.Int, reason tracing.BalanceChangeReason) uint256.Int {
	if !amount.IsZero() && s.checker.IsFrozen(addr) {
		s.frozenMoved = true
	}
	return s.StateDB.SubBalance(addr, amount, reason)
}

// --- non-native token transfers and approvals ---
// AddLog tracks ERC-20, ERC-721 and ERC-1155 transfer and approval events
// naming a frozen account.

func (s *StateDB) AddLog(log *types.Log) {
	if isFrozenTokenLog(s.checker, log) {
		s.frozenMoved = true
	}
	s.StateDB.AddLog(log)
}

var (
	transferTopic       = crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)"))
	approvalTopic       = crypto.Keccak256Hash([]byte("Approval(address,address,uint256)"))
	approvalForAllTopic = crypto.Keccak256Hash([]byte("ApprovalForAll(address,address,bool)"))
	transferSingleTopic = crypto.Keccak256Hash([]byte("TransferSingle(address,address,address,uint256,uint256)"))
	transferBatchTopic  = crypto.Keccak256Hash([]byte("TransferBatch(address,address,address,uint256[],uint256[])"))
)

// isFrozenTokenLog reports whether log is a standard ERC-20, ERC-721 or
// ERC-1155 event naming a frozen account, emitted by a token not opted out.
func isFrozenTokenLog(checker Checker, log *types.Log) bool {
	var addresses []common.Hash
	switch topics := log.Topics; {
	// ERC-20: indexed topics = [event signature, address1, address2]
	case len(topics) == 3 && slices.Contains([]common.Hash{transferTopic, approvalTopic}, topics[0]):
		addresses = topics[1:]
	// ERC-721: indexed topics = [event signature, address1, address2, token id]
	case len(topics) == 4 && slices.Contains([]common.Hash{transferTopic, approvalTopic}, topics[0]):
		addresses = topics[1:3]
	// ERC-1155: indexed topics = [event signature, address1, address2, address3]
	case len(topics) == 4 && slices.Contains([]common.Hash{transferSingleTopic, transferBatchTopic}, topics[0]):
		addresses = topics[1:]
	// ERC-721 and ERC-1155 ApprovalForAll: indexed topics = [event signature, address1, address2]
	case len(topics) == 3 && topics[0] == approvalForAllTopic:
		addresses = topics[1:]
	default:
		return false
	}

	if checker.IsOptedOut(log.Address) {
		return false
	}

	isFrozen := func(addr common.Hash) bool {
		return checker.IsFrozen(common.BytesToAddress(addr.Bytes()))
	}
	return slices.ContainsFunc(addresses, isFrozen)
}
