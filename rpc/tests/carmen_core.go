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
	"github.com/0xsoniclabs/sonic/gossip/evmstore"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// AccountState is an account of the state of a Carmen core.
type AccountState struct {
	Balance *big.Int
	Nonce   uint64
	Code    []byte
	Storage map[common.Hash]common.Hash
}

// NewCarmenCore is a FakeCore whose state is one real Carmen state with an
// S5 archive, holding the accounts at every block: each State opens the
// archive's one block, so that calls run on a real EVM state and witness
// proofs work. The root of every block is the state's.
func NewCarmenCore(t testing.TB, accounts map[common.Address]AccountState) *FakeCore {
	t.Helper()
	carmenState, err := carmen.NewState(carmen.Parameters{
		Variant:      "go-file",
		Schema:       carmen.Schema(5),
		Archive:      carmen.S5Archive,
		Directory:    t.TempDir(),
		LiveCache:    1,
		ArchiveCache: 1,
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, carmenState.Close()) })

	carmenStateDB := carmen.CreateStateDBUsing(carmenState)
	writer := evmstore.CreateCarmenStateDb(carmenStateDB, nil)
	writer.BeginBlock(0)
	for address, account := range accounts {
		if account.Balance != nil {
			writer.SetBalance(address, uint256.MustFromBig(account.Balance))
		}
		writer.SetNonce(address, account.Nonce, tracing.NonceChangeUnspecified)
		if len(account.Code) > 0 {
			writer.SetCode(address, account.Code, tracing.CodeChangeUnspecified)
		}
		for key, value := range account.Storage {
			writer.SetState(address, key, value)
		}
	}
	writer.EndTransaction()
	staged, err := writer.EndBlock(0)
	require.NoError(t, err)
	archived, err := staged.Commit()
	require.NoError(t, err)
	require.NoError(t, archived.Wait())

	fake := NewFakeCore()
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	fake.stateRoot = common.Hash(staged.StateHash())
	fake.blocks[0].Root = fake.stateRoot
	fake.openState = func() (state.StateDB, error) {
		archive, err := carmenStateDB.GetArchiveStateDB(0)
		if err != nil {
			return nil, err
		}
		return evmstore.CreateNonCommittableCarmenStateDb(archive, nil), nil
	}
	return fake
}

// OpenStates is the number of states handed out and not released yet.
func (f *FakeCore) OpenStates() int {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	return f.openStates
}

// trackedState counts its release, and that of its copies, in the fake.
type trackedState struct {
	state.StateDB
	fake    *FakeCore
	release sync.Once
}

func (f *FakeCore) track(stateDB state.StateDB) state.StateDB {
	f.openStates++
	return &trackedState{StateDB: stateDB, fake: f}
}

func (s *trackedState) Copy() state.StateDB {
	s.fake.mutex.Lock()
	defer s.fake.mutex.Unlock()
	return s.fake.track(s.StateDB.Copy())
}

func (s *trackedState) Release() {
	s.release.Do(func() {
		s.fake.mutex.Lock()
		s.fake.openStates--
		s.fake.mutex.Unlock()
		s.StateDB.Release()
	})
}
