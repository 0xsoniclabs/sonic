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
	"testing"

	carmen "github.com/0xsoniclabs/carmen/go/state"
	"github.com/0xsoniclabs/carmen/go/state/gostate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/holiman/uint256"

	"github.com/0xsoniclabs/sonic/gossip/evmstore"
	"github.com/0xsoniclabs/sonic/inter/state"
)

// Account is the state of an account a CarmenCore starts with.
type Account struct {
	Balance *big.Int
	Nonce   uint64
	Code    []byte
	Storage map[common.Hash]common.Hash
}

// CarmenCore is a FakeCore whose State opens a view of one real in-memory
// Carmen state, the same for every block. The blocks carry the root of that
// state, so witness proofs verify against them.
type CarmenCore struct {
	*FakeCore

	live carmen.State

	mutex    sync.Mutex
	opened   int
	released int
}

// NewCarmenCore returns a CarmenCore whose state holds the given accounts.
func NewCarmenCore(t testing.TB, accounts map[common.Address]Account) *CarmenCore {
	live, err := carmen.NewState(carmen.Parameters{
		Variant:      gostate.VariantGoMemory,
		Schema:       carmen.Schema(5),
		Archive:      carmen.NoArchive,
		Directory:    t.TempDir(),
		LiveCache:    1,
		ArchiveCache: 1,
	})
	if err != nil {
		t.Fatalf("failed to create the carmen state: %v", err)
	}
	t.Cleanup(func() {
		if err := live.Close(); err != nil {
			t.Errorf("failed to close the carmen state: %v", err)
		}
	})

	root := writeAccounts(t, live, accounts)
	fake := NewFakeCore(t)
	fake.setStateRoot(root)
	carmenCore := &CarmenCore{FakeCore: fake, live: live}
	fake.SetStates(carmenCore.openState)
	return carmenCore
}

func writeAccounts(t testing.TB, live carmen.State, accounts map[common.Address]Account) common.Hash {
	writer := evmstore.CreateCarmenStateDb(carmen.CreateStateDBUsing(live), nil)
	writer.BeginBlock(0)
	for address, account := range accounts {
		if account.Balance != nil {
			writer.SetBalance(address, uint256.MustFromBig(account.Balance))
		}
		if account.Nonce != 0 {
			writer.SetNonce(address, account.Nonce, tracing.NonceChangeUnspecified)
		}
		if len(account.Code) > 0 {
			writer.SetCode(address, account.Code, tracing.CodeChangeUnspecified)
		}
		for key, value := range account.Storage {
			writer.SetState(address, key, value)
		}
	}
	writer.EndTransaction()
	staged, err := writer.EndBlock(0)
	if err != nil {
		t.Fatalf("failed to end the block of the accounts: %v", err)
	}
	// Without an archive the commit has nothing to wait for; its handle is nil.
	if _, err := staged.Commit(); err != nil {
		t.Fatalf("failed to commit the block of the accounts: %v", err)
	}
	return common.Hash(staged.StateHash())
}

func (carmenCore *CarmenCore) openState(uint64) (state.StateDB, error) {
	carmenCore.mutex.Lock()
	carmenCore.opened++
	carmenCore.mutex.Unlock()
	view := evmstore.CreateNonCommittableCarmenStateDb(carmen.CreateNonCommittableStateDBUsing(carmenCore.live), nil)
	return &trackedState{StateDB: view, release: carmenCore.countRelease}, nil
}

func (carmenCore *CarmenCore) countRelease() {
	carmenCore.mutex.Lock()
	defer carmenCore.mutex.Unlock()
	carmenCore.released++
}

// StateCount tells how many states were opened and released.
type StateCount struct {
	Opened   int
	Released int
}

// States reports the states opened and released so far.
func (carmenCore *CarmenCore) States() StateCount {
	carmenCore.mutex.Lock()
	defer carmenCore.mutex.Unlock()
	return StateCount{Opened: carmenCore.opened, Released: carmenCore.released}
}

// trackedState counts each of its releases, so that a state released twice
// shows as well as one never released. A copy is a state of its own whose
// release is not counted, as it is not one the request opened from Core.
type trackedState struct {
	state.StateDB
	release func()
}

func (tracked *trackedState) Release() {
	tracked.release()
	tracked.StateDB.Release()
}

// setStateRoot makes every block, present and future, carry the given state
// root.
func (fake *FakeCore) setStateRoot(root common.Hash) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	fake.stateRoot = root
	for _, block := range fake.blocks {
		block.Root = root
	}
}
