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

package bundle

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

// ExecutionStep represents a single step in the execution plan,
// which corresponds to a transaction to be executed as part of the bundle.
type ExecutionStep struct {
	// From is the sender of the transaction, derived from the signature of the transaction
	From common.Address
	// Hash is the transaction hash to be signed (not the hash of the transaction including its signature)
	// where the access list has been stripped from the bundle-only mark.
	Hash common.Hash
}

// ExecutionPlan represents the plan for executing a bundle of transactions,
// to which every participant in the bundle shall agree on.
// The execution plan includes the list of steps to be executed, in the order of execution
type ExecutionPlan struct {
	Steps []ExecutionStep // Steps to be executed in the bundle, in the order of execution
	Flags ExecutionFlags  // Execution flags that specify the behavior of the bundle execution
	Range BlockRange      // Block range [Earliest, Latest] in which the bundle can be included
}

// Hash computes the execution plan hash
// The hash is computed with Keccak256, and is based on the RLP encoding of the type
// rlp([Steps, Flags]), where Steps is of type [[{20 bytes}, {32 bytes}]...] where
// ... means “zero or more of the thing to the left”
func (e *ExecutionPlan) Hash() common.Hash {
	hasher := crypto.NewKeccakState()
	_ = rlp.Encode(hasher, e)
	return common.BytesToHash(hasher.Sum(nil))
}
