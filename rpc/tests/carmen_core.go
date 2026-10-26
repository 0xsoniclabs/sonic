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
package tests

import (
	"math/big"
	"sync"
	"sync/atomic"
	"testing"

	carmen "github.com/0xsoniclabs/carmen/go/state"
	_ "github.com/0xsoniclabs/carmen/go/state/gostate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/holiman/uint256"

	"github.com/0xsoniclabs/sonic/gossip/evmstore"
	"github.com/0xsoniclabs/sonic/inter/state"
)

// Account is the state of an account to set up.
type Account struct {
	Balance *big.Int
	Nonce   uint64
	Code    []byte
	Storage map[common.Hash]common.Hash
}

// CarmenCore is a FakeCore whose State is a copy of one real in-memory Carmen
// state, the same for every block. It counts the states it opened and those
// released.
type CarmenCore struct {
	*FakeCore
	mu       sync.Mutex
	base     state.StateDB
	opened   atomic.Int64
	released atomic.Int64
}

// NewCarmenCore creates a fake chain over a Carmen state holding the accounts.
// The accounts are block 0 of an in-memory archive, so that the state answers
// witness proofs as the node's archive does; every block reads that state.
func NewCarmenCore(t testing.TB, accounts map[common.Address]Account) *CarmenCore {
	t.Helper()
	cs, err := carmen.NewState(carmen.Parameters{
		Variant:   "go-memory",
		Schema:    carmen.Schema(5),
		Archive:   carmen.S5Archive,
		Directory: t.TempDir(),
		// Small caches: a test holds a few accounts, and several cores may
		// live at once.
		LiveCache:    16 << 20,
		ArchiveCache: 16 << 20,
	})
	if err != nil {
		t.Fatalf("creating carmen state: %v", err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	live := carmen.CreateStateDBUsing(cs)
	genesis := evmstore.CreateCarmenStateDb(live, nil)
	genesis.BeginBlock(0)
	for addr, acc := range accounts {
		writeAccount(genesis, addr, acc)
	}
	staged, err := genesis.EndBlock(0)
	if err != nil {
		t.Fatalf("sealing the accounts: %v", err)
	}
	root := staged.StateHash()
	wait, err := staged.Commit()
	if err != nil {
		t.Fatalf("archiving the accounts: %v", err)
	}
	if err := wait.Wait(); err != nil {
		t.Fatalf("archiving the accounts: %v", err)
	}
	archive, err := live.GetArchiveStateDB(0)
	if err != nil {
		t.Fatalf("opening the archive: %v", err)
	}
	c := &CarmenCore{
		FakeCore: NewFakeCore(),
		base:     evmstore.CreateNonCommittableCarmenStateDb(archive, nil),
	}
	t.Cleanup(c.base.Release)
	c.SetRoot(common.Hash(root))
	c.SetState(c.open)
	return c
}

func writeAccount(db state.StateDB, addr common.Address, acc Account) {
	if acc.Balance != nil {
		db.SetBalance(addr, uint256.MustFromBig(acc.Balance))
	}
	if acc.Nonce != 0 {
		db.SetNonce(addr, acc.Nonce, tracing.NonceChangeUnspecified)
	}
	if len(acc.Code) > 0 {
		db.SetCode(addr, acc.Code, tracing.CodeChangeUnspecified)
	}
	for k, v := range acc.Storage {
		db.SetState(addr, k, v)
	}
	db.EndTransaction()
}

// SetAccount writes an account into the state every block shares. It is not
// in the archive: witness proofs do not see it.
func (c *CarmenCore) SetAccount(addr common.Address, acc Account) {
	c.mu.Lock()
	defer c.mu.Unlock()
	writeAccount(c.base, addr, acc)
}

func (c *CarmenCore) open(uint64) (state.StateDB, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.opened.Add(1)
	return &trackedState{StateDB: c.base.Copy(), core: c}, nil
}

// TrackStates reports how many states were opened and how many released.
func (c *CarmenCore) TrackStates() (opened, released int) {
	return int(c.opened.Load()), int(c.released.Load())
}

// trackedState counts every release of it, and the copies made of it; a state
// released twice shows as more releases than states opened.
type trackedState struct {
	state.StateDB
	core *CarmenCore
	once sync.Once
}

func (s *trackedState) Copy() state.StateDB {
	s.core.opened.Add(1)
	return &trackedState{StateDB: s.StateDB.Copy(), core: s.core}
}

func (s *trackedState) Release() {
	s.core.released.Add(1)
	s.once.Do(s.StateDB.Release)
}
