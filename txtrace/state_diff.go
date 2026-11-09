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

package txtrace

import (
	"bytes"
	"math/big"

	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/holiman/uint256"
)

// TraceCallResult is the result structure for trace_call RPC.
// It is compatible with the Erigon/OpenEthereum trace_call format.
type TraceCallResult struct {
	Output    hexutil.Bytes `json:"output"`
	StateDiff StateDiff     `json:"stateDiff"`
	Trace     []ActionTrace `json:"trace"`
	VmTrace   *VmTrace      `json:"vmTrace"`
}

// StateDiff maps addresses to their account state changes during transaction execution.
type StateDiff map[common.Address]*StateDiffAccount

// StateDiffAccount holds all field diffs for a single account.
// Each field uses one of the following markers:
//   - "=" string: no change
//   - {"*": StateDiffBalance/Nonce/Code} map: changed (persisting accounts)
//   - {"+": value} map: created/born (new account or new storage slot in born account)
//   - {"-": value} map: deleted/died (account existed before but not after)
type StateDiffAccount struct {
	Balance any                 `json:"balance"`
	Code    any                 `json:"code"`
	Nonce   any                 `json:"nonce"`
	Storage map[common.Hash]any `json:"storage"`
}

// StateDiffBalance holds the from/to values for a balance change.
type StateDiffBalance struct {
	From *hexutil.Big `json:"from"`
	To   *hexutil.Big `json:"to"`
}

// StateDiffCode holds the from/to values for a code change.
type StateDiffCode struct {
	From hexutil.Bytes `json:"from"`
	To   hexutil.Bytes `json:"to"`
}

// StateDiffNonce holds the from/to values for a nonce change.
type StateDiffNonce struct {
	From hexutil.Uint64 `json:"from"`
	To   hexutil.Uint64 `json:"to"`
}

// StateDiffStorage holds the from/to values for a storage slot change.
type StateDiffStorage struct {
	From common.Hash `json:"from"`
	To   common.Hash `json:"to"`
}

// StateDiffLogger wraps a StateDB to capture the state diff of a transaction.
// It records each account and storage slot as it was before its first
// modification, and GetResult compares them with the state after the
// transaction.
type StateDiffLogger struct {
	state.StateDB
	accounts map[common.Address]*accountState
	storage  map[common.Address]map[common.Hash]common.Hash
}

type accountState struct {
	exists  bool
	balance *big.Int
	nonce   uint64
	code    []byte
}

func NewStateDiffLogger(db state.StateDB) *StateDiffLogger {
	return &StateDiffLogger{
		StateDB:  db,
		accounts: make(map[common.Address]*accountState),
		storage:  make(map[common.Address]map[common.Hash]common.Hash),
	}
}

func (s *StateDiffLogger) CreateAccount(addr common.Address) {
	s.touch(addr)
	s.StateDB.CreateAccount(addr)
}

func (s *StateDiffLogger) CreateContract(addr common.Address) {
	s.touch(addr)
	s.StateDB.CreateContract(addr)
}

func (s *StateDiffLogger) AddBalance(addr common.Address, amount *uint256.Int, reason tracing.BalanceChangeReason) uint256.Int {
	s.touch(addr)
	return s.StateDB.AddBalance(addr, amount, reason)
}

func (s *StateDiffLogger) SubBalance(addr common.Address, amount *uint256.Int, reason tracing.BalanceChangeReason) uint256.Int {
	s.touch(addr)
	return s.StateDB.SubBalance(addr, amount, reason)
}

func (s *StateDiffLogger) SetNonce(addr common.Address, nonce uint64, reason tracing.NonceChangeReason) {
	s.touch(addr)
	s.StateDB.SetNonce(addr, nonce, reason)
}

func (s *StateDiffLogger) SetCode(addr common.Address, code []byte, reason tracing.CodeChangeReason) []byte {
	s.touch(addr)
	return s.StateDB.SetCode(addr, code, reason)
}

func (s *StateDiffLogger) SetState(addr common.Address, slot common.Hash, value common.Hash) common.Hash {
	s.touch(addr)
	prev := s.StateDB.SetState(addr, slot, value)
	if s.storage[addr] == nil {
		s.storage[addr] = make(map[common.Hash]common.Hash)
	}
	if _, ok := s.storage[addr][slot]; !ok {
		s.storage[addr][slot] = prev
	}
	return prev
}

func (s *StateDiffLogger) SelfDestruct(addr common.Address) {
	s.touch(addr)
	s.StateDB.SelfDestruct(addr)
}

func (s *StateDiffLogger) touch(addr common.Address) {
	if _, ok := s.accounts[addr]; !ok {
		s.accounts[addr] = s.readAccount(addr)
	}
}

func (s *StateDiffLogger) readAccount(addr common.Address) *accountState {
	return &accountState{
		exists:  s.Exist(addr),
		balance: s.GetBalance(addr).ToBig(),
		nonce:   s.GetNonce(addr),
		code:    bytes.Clone(s.GetCode(addr)),
	}
}

// GetResult compares the modified accounts with their current state and
// returns the diff in Erigon's stateDiff format:
//   - Persisting accounts: "=" for unchanged, {"*": {from, to}} for changed fields;
//     left out if nothing changed
//   - Born accounts: {"+": value} for all fields and storage slots
//   - Died accounts: {"-": value} for all fields
//
// It must be called after the transaction has ended.
func (s *StateDiffLogger) GetResult() StateDiff {
	diff := make(StateDiff)
	for addr, before := range s.accounts {
		after := s.readAccount(addr)
		switch {
		case before.exists && after.exists:
			storage := s.storageDiff(addr, false)
			sameBalance := before.balance.Cmp(after.balance) == 0
			sameCode := bytes.Equal(before.code, after.code)
			sameNonce := before.nonce == after.nonce
			if sameBalance && sameCode && sameNonce && len(storage) == 0 {
				continue
			}
			diff[addr] = &StateDiffAccount{
				Balance: fieldDiff(sameBalance, &StateDiffBalance{From: (*hexutil.Big)(before.balance), To: (*hexutil.Big)(after.balance)}),
				Code:    fieldDiff(sameCode, &StateDiffCode{From: before.code, To: after.code}),
				Nonce:   fieldDiff(sameNonce, &StateDiffNonce{From: hexutil.Uint64(before.nonce), To: hexutil.Uint64(after.nonce)}),
				Storage: storage,
			}
		case before.exists:
			diff[addr] = markedAccount("-", before, s.storageDiff(addr, false))
		case after.exists:
			diff[addr] = markedAccount("+", after, s.storageDiff(addr, true))
		}
	}
	return diff
}

func markedAccount(marker string, account *accountState, storage map[common.Hash]any) *StateDiffAccount {
	return &StateDiffAccount{
		Balance: map[string]any{marker: (*hexutil.Big)(account.balance)},
		Code:    map[string]any{marker: hexutil.Bytes(account.code)},
		Nonce:   map[string]any{marker: hexutil.Uint64(account.nonce)},
		Storage: storage,
	}
}

func (s *StateDiffLogger) storageDiff(addr common.Address, born bool) map[common.Hash]any {
	diff := make(map[common.Hash]any)
	for slot, before := range s.storage[addr] {
		after := s.GetState(addr, slot)
		switch {
		case before == after:
		case born:
			diff[slot] = map[string]any{"+": after}
		default:
			diff[slot] = map[string]any{"*": &StateDiffStorage{From: before, To: after}}
		}
	}
	return diff
}

func fieldDiff(same bool, change any) any {
	if same {
		return "="
	}
	return map[string]any{"*": change}
}
