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
	"errors"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/carmen/go/common/witness"
	carmen "github.com/0xsoniclabs/carmen/go/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/holiman/uint256"

	"github.com/0xsoniclabs/sonic/gossip/evmstore"
	"github.com/0xsoniclabs/sonic/inter/state"
)

// CarmenState is a real in-memory Carmen state the tests of the execution
// methods run on. Its `Open` hands out independent copies, as the node hands
// out the archive state of a block.
//
// Note: the EndBlock of a Carmen state without archive returns a nil channel;
// nothing here waits on it.
type CarmenState struct {
	base state.StateDB
}

// NewCarmenState creates an empty state.
func NewCarmenState(t testing.TB) *CarmenState {
	t.Helper()
	carmenState, err := carmen.NewState(carmen.Parameters{
		Variant:      "go-file",
		Schema:       carmen.Schema(5),
		Archive:      carmen.S5Archive,
		Directory:    t.TempDir(),
		LiveCache:    1, // use minimum cache (not default)
		ArchiveCache: 1,
	})
	if err != nil {
		t.Fatalf("failed to create carmen state: %s", err)
	}
	t.Cleanup(func() { _ = carmenState.Close() })
	// The execution plan store does not track historic states.
	var processedBundleStore evmstore.ProcessedBundleStore
	stateDb := carmen.CreateNonCommittableStateDBUsing(carmenState)
	return &CarmenState{base: evmstore.CreateNonCommittableCarmenStateDb(stateDb, processedBundleStore)}
}

// SetBalance sets the balance of an account.
func (s *CarmenState) SetBalance(addr common.Address, balance *big.Int) *CarmenState {
	s.base.SetBalance(addr, uint256.MustFromBig(balance))
	return s
}

// SetNonce sets the nonce of an account.
func (s *CarmenState) SetNonce(addr common.Address, nonce uint64) *CarmenState {
	s.base.SetNonce(addr, nonce, tracing.NonceChangeUnspecified)
	return s
}

// SetCode sets the code of an account.
func (s *CarmenState) SetCode(addr common.Address, code []byte) *CarmenState {
	s.base.SetCode(addr, code, tracing.CodeChangeUnspecified)
	return s
}

// SetStorage sets a storage slot of an account.
func (s *CarmenState) SetStorage(addr common.Address, key, value common.Hash) *CarmenState {
	s.base.SetState(addr, key, value)
	return s
}

// Open returns an independent copy of the state, which the caller releases.
func (s *CarmenState) Open() state.StateDB { return &noProofs{s.base.Copy()} }

// noProofs reports that witness proofs are not available: the in-memory Carmen
// variant cannot create them.
type noProofs struct{ state.StateDB }

func (n *noProofs) GetProof(common.Address, []common.Hash) (witness.Proof, error) {
	return nil, errors.New("proofs are not available in the in-memory state")
}

func (n *noProofs) Copy() state.StateDB { return &noProofs{n.StateDB.Copy()} }

// NewCarmenCore creates a FakeCore whose State(n) is a copy of the Carmen state
// for every block.
func NewCarmenCore(t testing.TB, opts ...Option) (*FakeCore, *CarmenState) {
	t.Helper()
	fc := NewFakeCore(t, opts...)
	cs := NewCarmenState(t)
	fc.setState(func(uint64) (state.StateDB, error) { return cs.Open(), nil })
	return fc, cs
}
