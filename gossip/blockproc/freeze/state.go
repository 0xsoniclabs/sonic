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
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// StateDB wraps the state an EVM runs on and counts the moves of a frozen
// account's funds still in place: value transfers and SELFDESTRUCT payouts
// from or to a frozen account, and ERC-20 logs naming one. Remembering the
// count at every snapshot leaves out moves undone by a revert.
type StateDB struct {
	vm.StateDB
	frozenMoves   int
	frozenMovesAt map[int]int // frozenMoves when each snapshot was taken
}

// Wrap returns db with the frozen-move tracking used by the freeze.
func Wrap(db vm.StateDB) *StateDB {
	return &StateDB{StateDB: db, frozenMovesAt: map[int]int{}}
}

// Prepare starts a transaction without frozen moves.
func (s *StateDB) Prepare(rules params.Rules, sender, coinbase common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList) {
	s.frozenMoves = 0
	clear(s.frozenMovesAt)
	s.StateDB.Prepare(rules, sender, coinbase, dest, precompiles, txAccesses)
}

func (s *StateDB) Snapshot() int {
	id := s.StateDB.Snapshot()
	s.frozenMovesAt[id] = s.frozenMoves
	return id
}

func (s *StateDB) RevertToSnapshot(id int) {
	s.StateDB.RevertToSnapshot(id)
	s.frozenMoves = s.frozenMovesAt[id]
}

// movedSince reports whether a frozen account's funds moved after snapshot.
func (s *StateDB) movedSince(snapshot int) bool {
	return s.frozenMoves > s.frozenMovesAt[snapshot]
}

// SubBalance and AddBalance count non-zero value transfers and SELFDESTRUCT
// payouts from or to a frozen account, by the reasons the EVM's Transfer and
// both interpreters' SELFDESTRUCT tag them with. Transfer covers transaction
// value, CALL, CREATE and CREATE2; a frozen contract sending value to itself
// counts too, although nothing leaves it.
func (s *StateDB) SubBalance(addr common.Address, amount *uint256.Int, reason tracing.BalanceChangeReason) uint256.Int {
	s.count(addr, amount, reason)
	return s.StateDB.SubBalance(addr, amount, reason)
}

func (s *StateDB) AddBalance(addr common.Address, amount *uint256.Int, reason tracing.BalanceChangeReason) uint256.Int {
	s.count(addr, amount, reason)
	return s.StateDB.AddBalance(addr, amount, reason)
}

func (s *StateDB) count(addr common.Address, amount *uint256.Int, reason tracing.BalanceChangeReason) {
	switch reason {
	case tracing.BalanceChangeTransfer,
		tracing.BalanceIncreaseSelfdestruct, tracing.BalanceDecreaseSelfdestruct, tracing.BalanceDecreaseSelfdestructBurn:
		if !amount.IsZero() && IsFrozen(s.StateDB, addr) {
			s.frozenMoves++
		}
	}
}
