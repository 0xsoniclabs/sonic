// Copyright 2025 Sonic Operations Ltd
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
	carmen "github.com/0xsoniclabs/carmen/go/state"
	"github.com/0xsoniclabs/sonic/gossip/evmstore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/tests"
	"github.com/holiman/uint256"
)

// carmenFactory is a factory for creating Carmen database.
type carmenFactory struct {
	st carmen.State
}

// NewTestStateDB creates a new tests.StateTestState wrapping Carmen as a state database.
func (f carmenFactory) NewTestStateDB(accounts types.GenesisAlloc) tests.StateTestState {
	carmenstatedb := carmen.CreateCustomStateDBUsing(f.st, 1024)
	statedb := evmstore.CreateCarmenStateDb(carmenstatedb)
	for addr, a := range accounts {
		statedb.SetCode(addr, a.Code)
		statedb.SetNonce(addr, a.Nonce, tracing.NonceChangeUnspecified)
		statedb.SetBalance(addr, uint256.MustFromBig(a.Balance))
		for k, v := range a.Storage {
			statedb.SetState(addr, k, v)
		}
	}
	// Commit and re-open to start with a clean state.
	statedb.EndTransaction()
	statedb.EndBlock(0)
	statedb.GetStateHash()

	statedb = evmstore.CreateCarmenStateDb(carmenstatedb)
	return tests.StateTestState{StateDB: &carmenStateDB{CarmenStateDB: statedb.(*evmstore.CarmenStateDB)}}
}

// carmenStateDB is a wrapper for tests.TestStateDB adapting to Carmen.
type carmenStateDB struct {
	*evmstore.CarmenStateDB
	logs []*types.Log
}

// Database method not supported by Carmen.
func (c *carmenStateDB) Database() state.Database {
	return nil
}

// Logs returns the logs, available only after Commit is called.
func (c *carmenStateDB) Logs() []*types.Log {
	return c.logs
}

// SetLogger method not supported by Carmen.
func (c *carmenStateDB) SetLogger(l *tracing.Hooks) {
	// no-op
}

func (c *carmenStateDB) SetBalance(addr common.Address, amount *uint256.Int, reason tracing.BalanceChangeReason) {
	c.CarmenStateDB.SetBalance(addr, amount)
}

// IntermediateRoot method not supported by Carmen.
func (c *carmenStateDB) IntermediateRoot(deleteEmptyObjects bool) common.Hash {
	return common.Hash{}
}

// Commit ends transaction, ends block, and returns the state hash.
func (c *carmenStateDB) Commit(block uint64, deleteEmptyObjects bool, noStorageWiping bool) (common.Hash, error) {
	c.logs = c.CarmenStateDB.Logs() // backup logs, they are deleted on committing a tx/block
	c.EndTransaction()
	c.EndBlock(block)
	return c.GetStateHash(), nil
}
