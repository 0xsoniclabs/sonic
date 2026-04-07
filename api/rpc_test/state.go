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

package rpctest

import (
	"testing"

	carmen "github.com/0xsoniclabs/carmen/go/state"
	"github.com/0xsoniclabs/sonic/gossip/evmstore"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/holiman/uint256"
)

// testState is a full state.StateDB implementation for testing purposes.
// It is created from a fakeBackend via StateAndBlockByNumberOrHash
type testState struct {
	state.StateDB
}

// NewTestState creates a new testState with a fresh carmen state backend.
func NewTestState(t *testing.T) *testState {
	carmenDir := t.TempDir()
	carmenState, err := carmen.NewState(carmen.Parameters{
		Variant:      "go-file",
		Schema:       carmen.Schema(5),
		Archive:      carmen.S5Archive,
		Directory:    carmenDir,
		LiveCache:    1, // use minimum cache (not default)
		ArchiveCache: 1, // use minimum cache (not default)
	})
	if err != nil {
		t.Fatalf("failed to create carmen state; %s", err)
	}

	carmenStateDb := carmen.CreateNonCommittableStateDBUsing(carmenState)
	return &testState{evmstore.CreateCarmenStateDb(carmenStateDb)}
}

func (t *testState) Copy() state.StateDB {
	return &testState{t.StateDB.Copy()}
}

func (t *testState) setAccount(addr common.Address, acc AccountState) {
	if acc.Balance != nil {
		t.SetBalance(addr, uint256.MustFromBig(acc.Balance))
	}
	if acc.Nonce != 0 {
		t.SetNonce(addr, acc.Nonce, tracing.NonceChangeUnspecified)
	}
	if len(acc.Code) > 0 {
		t.SetCode(addr, acc.Code, tracing.CodeChangeUnspecified)
	}
	for k, v := range acc.Store {
		t.SetState(addr, k, v)
	}
}
