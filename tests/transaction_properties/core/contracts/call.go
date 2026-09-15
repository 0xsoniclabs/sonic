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
	"bytes"
	"fmt"
	"slices"

	"github.com/ethereum/go-ethereum/common"
)

// Deployed is one registry contract as it exists on the network under test.
type Deployed struct {
	Contract int // index into Registry
	Address  common.Address
}

// Call is one call to a deployed contract, held as plain data so that rapid can shrink it and a
// failing case can be printed verbatim. The arguments are packed when the call is drawn rather than
// when it is built, which is what lets a mutation work on the bytes the EVM will really see.
type Call struct {
	Contract int            // index into Registry
	Address  common.Address // where that contract is deployed
	Method   int            // index into that contract's Methods
	Args     []byte         // the method's arguments, ABI-encoded
	Mutation Mutation
}

// Data is the call data this call sends, which is the method's selector, its arguments, and whatever
// the mutation does to the two.
func (c *Call) Data() []byte {
	method := Registry[c.Contract].Methods[c.Method]
	return c.Mutation.apply(append(slices.Clone(method.ID), c.Args...))
}

func (c *Call) String() string {
	contract := Registry[c.Contract]
	out := fmt.Sprintf("%s.%s(%dB)", contract.Name, contract.Methods[c.Method].Name, len(c.Args))
	if c.Mutation.Kind != MutateNone {
		out += "!" + c.Mutation.Kind.String()
	}
	return out
}

// MutationKind is how a drawn call is broken. Everything but MutateNone produces call data no
// well-formed caller would send, which is the point: a contract is reached by a transaction pipeline
// that must not care whether the payload makes sense to the code at the other end.
//
//go:generate go tool stringer -type=MutationKind -trimprefix=Mutate
type MutationKind uint8

const (
	// MutateNone leaves the call valid.
	MutateNone MutationKind = iota
	// MutateSelector changes a byte of the selector, so no method of the contract matches.
	MutateSelector
	// MutateFlip changes a byte of the arguments, which can put a value outside the range its type
	// admits or point a dynamic argument's offset outside the call data.
	MutateFlip
	// MutateTruncate cuts the call data short, leaving the arguments incomplete.
	MutateTruncate
	// MutateExtend appends bytes past the end of the arguments.
	MutateExtend
	// MutateDropArgs keeps the selector and drops every argument.
	MutateDropArgs
)

// Mutation is a defect applied to a drawn call's data.
type Mutation struct {
	Kind MutationKind
	At   int  // where it applies, taken modulo what it applies to
	Byte byte // what it writes there
}

// apply breaks the call data in the way the mutation names. The data always holds at least the four
// bytes of a selector, so every position below is within it.
func (m Mutation) apply(data []byte) []byte {
	switch m.Kind {
	case MutateSelector:
		data[m.At%4] ^= m.Byte | 1
	case MutateFlip:
		data[m.At%len(data)] ^= m.Byte | 1
	case MutateTruncate:
		data = data[:m.At%len(data)]
	case MutateExtend:
		data = append(data, bytes.Repeat([]byte{m.Byte}, 1+m.At%32)...)
	case MutateDropArgs:
		data = data[:4]
	}
	return data
}
