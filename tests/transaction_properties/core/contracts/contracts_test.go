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

package contracts

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

func TestRegistry_EveryContractHasCodeAndSomethingToCall(t *testing.T) {
	for _, contract := range Registry {
		require.NotEmpty(t, contract.Code, contract.Name)
		require.NotEmpty(t, contract.Methods, contract.Name)
	}
}

// TestDrawCall_AnUnmutatedCallIsOneItsContractCouldHaveBeenSent checks the generator against the
// bindings it draws from: whatever it packs must decode back through the same method's inputs, or a
// call believed valid never reaches the code it names.
func TestDrawCall_AnUnmutatedCallIsOneItsContractCouldHaveBeenSent(t *testing.T) {
	deployed := make([]Deployed, len(Registry))
	for i := range Registry {
		deployed[i] = Deployed{Contract: i, Address: common.Address{byte(i)}}
	}

	rapid.Check(t, func(rt *rapid.T) {
		call := DrawCall(rt, "call", deployed)
		if call.Mutation.Kind != MutateNone {
			return
		}

		method := Registry[call.Contract].Methods[call.Method]
		data := call.Data()
		require.Equal(rt, method.ID, data[:4], "%v", call)

		_, err := method.Inputs.Unpack(data[4:])
		require.NoError(rt, err, "%v", call)
	})
}

func TestMutation_LeavesCallDataNoContractWouldAccept(t *testing.T) {
	call := &Call{Args: make([]byte, 32)}
	valid := call.Data()

	for kind := MutateSelector; kind <= MutateDropArgs; kind++ {
		call.Mutation = Mutation{Kind: kind, At: 5, Byte: 0x7f}
		require.NotEqual(t, valid, call.Data(), "%v", kind)
	}
}
