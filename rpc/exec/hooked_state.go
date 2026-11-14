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

package exec

import (
	"math/big"

	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
)

// WithHooks wraps a state so that it reports its changes and logs to the
// tracing hooks, as the node's logging state does: the EVM reports only its
// own steps, and leaves these to the state.
func WithHooks(stateDB state.StateDB, hooks *tracing.Hooks) state.StateDB {
	return &hookedState{StateDB: stateDB, hooks: hooks, selfDestructed: map[common.Address]struct{}{}}
}

type hookedState struct {
	state.StateDB
	hooks          *tracing.Hooks
	selfDestructed map[common.Address]struct{}
}

func (s *hookedState) AddBalance(address common.Address, amount *uint256.Int, reason tracing.BalanceChangeReason) uint256.Int {
	previous := s.StateDB.AddBalance(address, amount, reason)
	if s.hooks.OnBalanceChange != nil && !amount.IsZero() {
		s.hooks.OnBalanceChange(address, previous.ToBig(), s.GetBalance(address).ToBig(), reason)
	}
	return previous
}

func (s *hookedState) SubBalance(address common.Address, amount *uint256.Int, reason tracing.BalanceChangeReason) uint256.Int {
	previous := s.StateDB.SubBalance(address, amount, reason)
	if s.hooks.OnBalanceChange != nil && !amount.IsZero() {
		s.hooks.OnBalanceChange(address, previous.ToBig(), s.GetBalance(address).ToBig(), reason)
	}
	return previous
}

func (s *hookedState) SetCode(address common.Address, code []byte, reason tracing.CodeChangeReason) []byte {
	previousHash := s.GetCodeHash(address)
	previous := s.StateDB.SetCode(address, code, reason)
	if s.hooks.OnCodeChange != nil {
		s.hooks.OnCodeChange(address, previousHash, previous, s.GetCodeHash(address), code)
	}
	return previous
}

func (s *hookedState) SetNonce(address common.Address, nonce uint64, reason tracing.NonceChangeReason) {
	if s.hooks.OnNonceChange != nil {
		s.hooks.OnNonceChange(address, s.GetNonce(address), nonce)
	}
	s.StateDB.SetNonce(address, nonce, reason)
}

func (s *hookedState) SetState(address common.Address, slot common.Hash, value common.Hash) common.Hash {
	previous := s.StateDB.SetState(address, slot, value)
	if s.hooks.OnStorageChange != nil {
		s.hooks.OnStorageChange(address, slot, previous, value)
	}
	return previous
}

func (s *hookedState) AddLog(log *types.Log) {
	s.StateDB.AddLog(log)
	if s.hooks.OnLog != nil {
		s.hooks.OnLog(log)
	}
}

func (s *hookedState) SelfDestruct(address common.Address) {
	if s.hooks.OnBalanceChange != nil {
		if previous := s.GetBalance(address); previous.Sign() > 0 {
			s.hooks.OnBalanceChange(address, previous.ToBig(), new(big.Int), tracing.BalanceDecreaseSelfdestruct)
		}
		s.selfDestructed[address] = struct{}{}
	}
	s.StateDB.SelfDestruct(address)
}

// EndTransaction reports the burning of what reached an account after it
// self-destructed.
func (s *hookedState) EndTransaction() {
	if s.hooks.OnBalanceChange != nil {
		for address := range s.selfDestructed {
			if s.HasSelfDestructed(address) {
				s.hooks.OnBalanceChange(address, s.GetBalance(address).ToBig(), new(big.Int), tracing.BalanceDecreaseSelfdestructBurn)
			}
		}
		s.selfDestructed = map[common.Address]struct{}{}
	}
	s.StateDB.EndTransaction()
}
