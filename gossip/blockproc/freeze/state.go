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
// AddBalance and SubBalance track non-zero value transfers (transaction value,
// CALL, CREATE, CREATE2) and SELFDESTRUCT payouts from or to a frozen account.

func (s *StateDB) AddBalance(addr common.Address, amount *uint256.Int, reason tracing.BalanceChangeReason) uint256.Int {
	if isFrozenMove(s.checker, addr, amount, reason) {
		s.frozenMoved = true
	}
	return s.StateDB.AddBalance(addr, amount, reason)
}

func (s *StateDB) SubBalance(addr common.Address, amount *uint256.Int, reason tracing.BalanceChangeReason) uint256.Int {
	if isFrozenMove(s.checker, addr, amount, reason) {
		s.frozenMoved = true
	}
	return s.StateDB.SubBalance(addr, amount, reason)
}

func isFrozenMove(checker Checker, addr common.Address, amount *uint256.Int, reason tracing.BalanceChangeReason) bool {
	switch reason {
	case tracing.BalanceChangeTransfer,
		tracing.BalanceIncreaseSelfdestruct,
		tracing.BalanceDecreaseSelfdestruct,
		tracing.BalanceDecreaseSelfdestructBurn:
		return !amount.IsZero() && checker.IsFrozen(addr)
	}
	return false
}
