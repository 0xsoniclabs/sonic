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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"

	"github.com/0xsoniclabs/sonic/inter/state"
)

// WithStateHooks wraps a state so that the hooks of a tracer see the changes
// made through it: balances, code, nonces, storage, logs and self-destructs.
func WithStateHooks(st state.StateDB, hooks *tracing.Hooks) state.StateDB {
	return &hookedState{StateDB: st, hooks: hooks, selfDestructed: make(map[common.Address]struct{})}
}

type hookedState struct {
	state.StateDB
	hooks          *tracing.Hooks
	selfDestructed map[common.Address]struct{}
}

func (s *hookedState) AddBalance(addr common.Address, amount *uint256.Int, reason tracing.BalanceChangeReason) uint256.Int {
	prev := s.StateDB.AddBalance(addr, amount, reason)
	if s.hooks.OnBalanceChange != nil && !amount.IsZero() {
		s.hooks.OnBalanceChange(addr, prev.ToBig(), s.GetBalance(addr).ToBig(), reason)
	}
	return prev
}

func (s *hookedState) SubBalance(addr common.Address, amount *uint256.Int, reason tracing.BalanceChangeReason) uint256.Int {
	prev := s.StateDB.SubBalance(addr, amount, reason)
	if s.hooks.OnBalanceChange != nil && !amount.IsZero() {
		s.hooks.OnBalanceChange(addr, prev.ToBig(), s.GetBalance(addr).ToBig(), reason)
	}
	return prev
}

func (s *hookedState) SetCode(addr common.Address, code []byte, reason tracing.CodeChangeReason) []byte {
	prevCodeHash := s.GetCodeHash(addr)
	prevCode := s.StateDB.SetCode(addr, code, reason)
	if s.hooks.OnCodeChange != nil {
		s.hooks.OnCodeChange(addr, prevCodeHash, prevCode, s.GetCodeHash(addr), code)
	}
	return prevCode
}

func (s *hookedState) SetNonce(addr common.Address, nonce uint64, reason tracing.NonceChangeReason) {
	if s.hooks.OnNonceChange != nil {
		s.hooks.OnNonceChange(addr, s.GetNonce(addr), nonce)
	}
	s.StateDB.SetNonce(addr, nonce, reason)
}

func (s *hookedState) SetState(addr common.Address, slot common.Hash, value common.Hash) common.Hash {
	prev := s.StateDB.SetState(addr, slot, value)
	if s.hooks.OnStorageChange != nil {
		s.hooks.OnStorageChange(addr, slot, prev, value)
	}
	return prev
}

func (s *hookedState) AddLog(log *types.Log) {
	if s.hooks.OnLog != nil {
		s.hooks.OnLog(log)
	}
	s.StateDB.AddLog(log)
}

func (s *hookedState) SelfDestruct(addr common.Address) {
	if s.hooks.OnBalanceChange != nil {
		if prev := s.GetBalance(addr); prev.Sign() > 0 {
			s.hooks.OnBalanceChange(addr, prev.ToBig(), new(big.Int), tracing.BalanceDecreaseSelfdestruct)
		}
		s.selfDestructed[addr] = struct{}{}
	}
	s.StateDB.SelfDestruct(addr)
}

// EndTransaction reports the balance an account received after it destructed
// itself as burnt.
func (s *hookedState) EndTransaction() {
	if s.hooks.OnBalanceChange != nil {
		for addr := range s.selfDestructed {
			if s.HasSelfDestructed(addr) {
				prev := s.GetBalance(addr)
				s.hooks.OnBalanceChange(addr, prev.ToBig(), new(big.Int), tracing.BalanceDecreaseSelfdestructBurn)
			}
		}
		s.selfDestructed = make(map[common.Address]struct{})
	}
	s.StateDB.EndTransaction()
}
