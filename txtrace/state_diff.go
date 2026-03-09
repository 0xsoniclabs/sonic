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
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/vm"
)

// TraceCallResult is the result structure for trace_call RPC.
// It is compatible with the Erigon/OpenEthereum trace_call format.
type TraceCallResult struct {
	Output    hexutil.Bytes `json:"output"`
	StateDiff StateDiff     `json:"stateDiff"`
	Trace     []ActionTrace `json:"trace"`
	VmTrace   any           `json:"vmTrace"`
}

// StateDiff maps addresses to their account state changes during transaction execution.
type StateDiff map[common.Address]*StateDiffAccount

// StateDiffAccount holds all field diffs for a single account.
// Each field uses one of the following markers:
//   - "=" string: no change
//   - {"*": StateDiffBalance/Nonce/Code} map: changed (persisting accounts)
//   - {"+": value} map: created/born (new account or new storage slot in born account)
//   - {"-": value} map: deleted/died (self-destructed account)
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

// StateDiffLogger captures account state changes during EVM execution using tracing hooks.
// Use Hooks() with evmstore.WrapStateDbWithLogger to capture state changes.
type StateDiffLogger struct {
	balances       map[common.Address]*balanceDiff
	nonces         map[common.Address]*nonceDiff
	codes          map[common.Address]*codeDiff
	storage        map[common.Address]map[common.Hash]*storageDiff
	selfDestructed map[common.Address]struct{} // tracks self-destructed (died) accounts
}

type balanceDiff struct {
	before *big.Int
	after  *big.Int
}

type nonceDiff struct {
	before uint64
	after  uint64
}

type codeDiff struct {
	before []byte
	after  []byte
}

type storageDiff struct {
	before common.Hash
	after  common.Hash
}

// NewStateDiffLogger creates a new StateDiffLogger.
func NewStateDiffLogger() *StateDiffLogger {
	return &StateDiffLogger{
		balances:       make(map[common.Address]*balanceDiff),
		nonces:         make(map[common.Address]*nonceDiff),
		codes:          make(map[common.Address]*codeDiff),
		storage:        make(map[common.Address]map[common.Hash]*storageDiff),
		selfDestructed: make(map[common.Address]struct{}),
	}
}

// Hooks returns the tracing hooks for the StateDiffLogger.
// The hooks track balance, nonce, code, and storage changes.
func (s *StateDiffLogger) Hooks() *tracing.Hooks {
	return &tracing.Hooks{
		OnBalanceChange: s.onBalanceChange,
		OnNonceChange:   s.onNonceChange,
		OnCodeChange:    s.onCodeChange,
		OnStorageChange: s.onStorageChange,
	}
}

func (s *StateDiffLogger) onBalanceChange(addr common.Address, prev, newBal *big.Int, reason tracing.BalanceChangeReason) {
	prevCopy := new(big.Int)
	if prev != nil {
		prevCopy.Set(prev)
	}
	newCopy := new(big.Int)
	if newBal != nil {
		newCopy.Set(newBal)
	}

	// Track self-destructed accounts to show them as "died" in the diff.
	if reason == tracing.BalanceDecreaseSelfdestruct || reason == tracing.BalanceDecreaseSelfdestructBurn {
		s.selfDestructed[addr] = struct{}{}
	}

	if existing, ok := s.balances[addr]; ok {
		existing.after = newCopy
	} else {
		s.balances[addr] = &balanceDiff{before: prevCopy, after: newCopy}
	}
}

func (s *StateDiffLogger) onNonceChange(addr common.Address, prev, newNonce uint64) {
	if existing, ok := s.nonces[addr]; ok {
		existing.after = newNonce
	} else {
		s.nonces[addr] = &nonceDiff{before: prev, after: newNonce}
	}
}

func (s *StateDiffLogger) onCodeChange(addr common.Address, _ common.Hash, prevCode []byte, _ common.Hash, newCode []byte) {
	prevCopy := make([]byte, len(prevCode))
	copy(prevCopy, prevCode)
	newCopy := make([]byte, len(newCode))
	copy(newCopy, newCode)
	if existing, ok := s.codes[addr]; ok {
		existing.after = newCopy
	} else {
		s.codes[addr] = &codeDiff{before: prevCopy, after: newCopy}
	}
}

func (s *StateDiffLogger) onStorageChange(addr common.Address, slot common.Hash, prev, newVal common.Hash) {
	if s.storage[addr] == nil {
		s.storage[addr] = make(map[common.Hash]*storageDiff)
	}
	if existing, ok := s.storage[addr][slot]; ok {
		existing.after = newVal
	} else {
		s.storage[addr][slot] = &storageDiff{before: prev, after: newVal}
	}
}

// OnEnter is called by the EVM for each call frame entry.
// It tracks SELFDESTRUCT opcodes to correctly identify accounts that die,
// even when the state is in read-only mode (where LoggingStateDB.SelfDestruct is never called).
func (s *StateDiffLogger) OnEnter(_ int, typ byte, from common.Address, _ common.Address, _ []byte, _ uint64, _ *big.Int) {
	if vm.OpCode(typ) == vm.SELFDESTRUCT {
		s.selfDestructed[from] = struct{}{}
	}
}

// GetResult returns the StateDiff capturing all state changes recorded so far.
// Produces output compatible with Erigon's stateDiff format:
//   - Persisting accounts: "=" for unchanged, {"*": {from, to}} for changed fields
//   - Born (new) accounts: {"+" : value} for all fields, {"+" : slot_value} for storage
//   - Died (self-destructed) accounts: {"-": value} for all fields
func (s *StateDiffLogger) GetResult() StateDiff {
	// Collect all changed addresses.
	addrs := make(map[common.Address]struct{})
	for addr := range s.balances {
		addrs[addr] = struct{}{}
	}
	for addr := range s.nonces {
		addrs[addr] = struct{}{}
	}
	for addr := range s.codes {
		addrs[addr] = struct{}{}
	}
	for addr := range s.storage {
		addrs[addr] = struct{}{}
	}

	diff := make(StateDiff)
	for addr := range addrs {
		account := &StateDiffAccount{
			Storage: make(map[common.Hash]any),
		}

		_, died := s.selfDestructed[addr]
		existedBefore := s.accountExistedBefore(addr)

		// Account created and self-destructed in same tx: exclude entirely from diff.
		if died && !existedBefore {
			continue
		}

		born := !died && !existedBefore

		switch {
		case died:
			// Self-destructed account: show pre-death values with "-" operator.
			account.Balance = s.diedBalanceDiff(addr)
			account.Code = s.diedCodeDiff(addr)
			account.Nonce = s.diedNonceDiff(addr)
			// Storage changes that happened before death keep "*" format.
			if storageChanges, ok := s.storage[addr]; ok {
				for slot, sc := range storageChanges {
					account.Storage[slot] = map[string]any{
						"*": &StateDiffStorage{From: sc.before, To: sc.after},
					}
				}
			}

		case born:
			// Newly created account: show post-creation values with "+" operator.
			account.Balance = s.bornBalanceDiff(addr)
			account.Code = s.bornCodeDiff(addr)
			account.Nonce = s.bornNonceDiff(addr)
			// For born accounts, storage shows only the final value with "+".
			if storageChanges, ok := s.storage[addr]; ok {
				for slot, sc := range storageChanges {
					account.Storage[slot] = map[string]any{"+": sc.after}
				}
			}

		default:
			// Persisting account: use "=" for unchanged, {"*": {from, to}} for changed.
			if bc, ok := s.balances[addr]; ok {
				if bc.before.Cmp(bc.after) == 0 {
					account.Balance = "="
				} else {
					account.Balance = map[string]any{
						"*": &StateDiffBalance{
							From: (*hexutil.Big)(bc.before),
							To:   (*hexutil.Big)(bc.after),
						},
					}
				}
			} else {
				account.Balance = "="
			}

			if nc, ok := s.nonces[addr]; ok {
				if nc.before == nc.after {
					account.Nonce = "="
				} else {
					account.Nonce = map[string]any{
						"*": &StateDiffNonce{
							From: hexutil.Uint64(nc.before),
							To:   hexutil.Uint64(nc.after),
						},
					}
				}
			} else {
				account.Nonce = "="
			}

			if cc, ok := s.codes[addr]; ok {
				if string(cc.before) == string(cc.after) {
					account.Code = "="
				} else {
					account.Code = map[string]any{
						"*": &StateDiffCode{
							From: hexutil.Bytes(cc.before),
							To:   hexutil.Bytes(cc.after),
						},
					}
				}
			} else {
				account.Code = "="
			}

			// Persisting account storage: always use "*" with from/to pair
			// (even for zero→nonzero slots, matching Erigon's format).
			if storageChanges, ok := s.storage[addr]; ok {
				for slot, sc := range storageChanges {
					account.Storage[slot] = map[string]any{
						"*": &StateDiffStorage{From: sc.before, To: sc.after},
					}
				}
			}
		}

		diff[addr] = account
	}
	return diff
}

// accountExistedBefore returns true if the account existed before this transaction,
// inferred from the "before" values captured at the time of the first state change.
// An account is considered non-existent (dead) if its nonce, balance, and code
// were all zero/empty — matching Ethereum's EIP-161 empty account definition.
// A storage-only change (no balance/nonce/code hooks) implies a pre-existing contract.
func (s *StateDiffLogger) accountExistedBefore(addr common.Address) bool {
	if bc, ok := s.balances[addr]; ok && bc.before.Sign() > 0 {
		return true
	}
	if nc, ok := s.nonces[addr]; ok && nc.before > 0 {
		return true
	}
	if cc, ok := s.codes[addr]; ok && len(cc.before) > 0 {
		return true
	}
	// Storage-only changes (no balance/nonce/code hooks) means the account
	// already existed as a contract with code but no other field changed.
	_, hasStorage := s.storage[addr]
	_, hasBalance := s.balances[addr]
	_, hasNonce := s.nonces[addr]
	_, hasCode := s.codes[addr]
	if hasStorage && !hasBalance && !hasNonce && !hasCode {
		return true
	}
	return false
}

// diedBalanceDiff returns the "{-: before_balance}" marker for a self-destructed account.
func (s *StateDiffLogger) diedBalanceDiff(addr common.Address) any {
	before := new(big.Int)
	if bc, ok := s.balances[addr]; ok {
		before = bc.before
	}
	return map[string]any{"-": (*hexutil.Big)(before)}
}

// diedCodeDiff returns the "{-: before_code}" marker for a self-destructed account.
func (s *StateDiffLogger) diedCodeDiff(addr common.Address) any {
	var before []byte
	if cc, ok := s.codes[addr]; ok {
		before = cc.before
	}
	return map[string]any{"-": hexutil.Bytes(before)}
}

// diedNonceDiff returns the "{-: before_nonce}" marker for a self-destructed account.
func (s *StateDiffLogger) diedNonceDiff(addr common.Address) any {
	var before uint64
	if nc, ok := s.nonces[addr]; ok {
		before = nc.before
	}
	return map[string]any{"-": hexutil.Uint64(before)}
}

// bornBalanceDiff returns the "{+: after_balance}" marker for a newly created account.
func (s *StateDiffLogger) bornBalanceDiff(addr common.Address) any {
	after := new(big.Int)
	if bc, ok := s.balances[addr]; ok {
		after = bc.after
	}
	return map[string]any{"+": (*hexutil.Big)(after)}
}

// bornCodeDiff returns the "{+: after_code}" marker for a newly created account.
func (s *StateDiffLogger) bornCodeDiff(addr common.Address) any {
	var after []byte
	if cc, ok := s.codes[addr]; ok {
		after = cc.after
	}
	return map[string]any{"+": hexutil.Bytes(after)}
}

// bornNonceDiff returns the "{+: after_nonce}" marker for a newly created account.
func (s *StateDiffLogger) bornNonceDiff(addr common.Address) any {
	var after uint64
	if nc, ok := s.nonces[addr]; ok {
		after = nc.after
	}
	return map[string]any{"+": hexutil.Uint64(after)}
}
