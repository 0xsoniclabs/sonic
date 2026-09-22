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

package execspec

import (
	"fmt"

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
	statedb := evmstore.CreateCarmenStateDb(carmenstatedb, nil)
	for addr, a := range accounts {
		statedb.SetCode(addr, a.Code, tracing.CodeChangeGenesis)
		statedb.SetNonce(addr, a.Nonce, tracing.NonceChangeGenesis)
		statedb.SetBalance(addr, uint256.MustFromBig(a.Balance))
		for k, v := range a.Storage {
			statedb.SetState(addr, k, v)
		}
	}
	// Commit and re-open to start with a clean state.
	statedb.EndTransaction()
	if err := evmstore.EndBlockAndCommit(statedb, 0); err != nil {
		panic(fmt.Sprintf("failed to commit genesis state: %v", err))
	}
	statedb.GetStateHash()

	statedb = evmstore.CreateCarmenStateDb(carmenstatedb, nil)
	return tests.StateTestState{StateDB: &carmenStateDB{CarmenStateDB: statedb}}
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

// IntermediateRoot is not supported by Carmen, but for ethereum tests it is only
// required for failing transaction. Rather than calculating the intermediate root,
// we can just end the transaction and block, and return the resulting state root.
func (c *carmenStateDB) IntermediateRoot(deleteEmptyObjects bool) common.Hash {
	c.EndTransaction()
	if err := evmstore.EndBlockAndCommit(c.CarmenStateDB, 0); err != nil {
		panic(fmt.Sprintf("failed to commit block: %v", err))
	}
	return c.GetStateHash()
}

// Commit ends transaction, ends block, and returns the state hash.
func (c *carmenStateDB) Commit(block uint64, deleteEmptyObjects bool, noStorageWiping bool) (common.Hash, error) {
	c.logs = c.CarmenStateDB.Logs() // backup logs, they are deleted on committing a tx/block
	c.EndTransaction()
	if err := evmstore.EndBlockAndCommit(c.CarmenStateDB, block); err != nil {
		return common.Hash{}, err
	}
	return c.GetStateHash(), nil
}
