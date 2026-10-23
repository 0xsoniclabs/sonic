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
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// ReplayChain is a chain over a real state whose newest block holds two
// transactions of Alice: a transfer to Bob and a call of Contract, which
// writes storage. The replay methods (trace_*, debug_trace*) execute them
// again.
type ReplayChain struct {
	Core     *FakeCore
	State    *CarmenState
	Alice    Account
	Bob      common.Address
	Contract common.Address     // runs StoreCode
	Transfer *types.Transaction // Alice sends 5 to Bob, nonce 0
	Store    *types.Transaction // Alice calls Contract, nonce 1
	Block    uint64             // the block of both transactions
}

// NewReplayChain creates the chain: genesis, one empty block and the block of
// the two transactions.
func NewReplayChain(t testing.TB) *ReplayChain {
	t.Helper()
	fc, cs := NewCarmenCore(t, WithBlocks(1))
	r := &ReplayChain{
		Core:     fc,
		State:    cs,
		Alice:    NewAccount(t, cs, big.NewInt(1e18)),
		Bob:      common.HexToAddress("0xb0b"),
		Contract: common.HexToAddress("0xc0de"),
	}
	cs.SetCode(r.Contract, StoreCode())
	r.Transfer = r.Alice.Transfer(t, 0, r.Bob, 5)
	r.Store = r.Alice.Call(t, 1, r.Contract, 0, nil, 100_000)
	r.Block = fc.AddBlock(r.Transfer, r.Store).NumberU64()
	return r
}
