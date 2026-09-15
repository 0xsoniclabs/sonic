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
	"math/big"
	"reflect"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"pgregory.net/rapid"
)

// DrawCall draws a call to one of the deployed contracts: which contract, which of its methods, what
// it is passed, and how the result is broken, if at all.
func DrawCall(t *rapid.T, label string, deployed []Deployed) *Call {
	target := rapid.SampledFrom(deployed).Draw(t, label)
	contract := Registry[target.Contract]
	method := rapid.IntRange(0, len(contract.Methods)-1).Draw(t, label+"Method")

	return &Call{
		Contract: target.Contract,
		Address:  target.Address,
		Method:   method,
		Args:     drawArgs(t, label, contract.Methods[method]),
		Mutation: drawMutation(t, label),
	}
}

// Mutations are weighted towards leaving the call valid, because a call that never reaches the code
// it names exercises nothing the payload of any other transaction does not already exercise.
var Mutations = rapid.SampledFrom([]MutationKind{
	MutateNone, MutateNone, MutateNone, MutateNone, MutateNone,
	MutateSelector, MutateFlip, MutateTruncate, MutateExtend, MutateDropArgs,
})

func drawMutation(t *rapid.T, label string) Mutation {
	return Mutation{
		Kind: Mutations.Draw(t, label+"Mutation"),
		At:   rapid.IntRange(0, 1<<16).Draw(t, label+"MutationAt"),
		Byte: rapid.Byte().Draw(t, label+"MutationByte"),
	}
}

// drawArgs draws one value per input of a method and encodes them. A method this cannot fill is left
// with no arguments at all, which is a malformed call rather than a broken generator.
func drawArgs(t *rapid.T, label string, method abi.Method) []byte {
	values := make([]any, len(method.Inputs))
	for i, input := range method.Inputs {
		value := reflect.New(input.Type.GetType()).Elem()
		fill(t, input.Type, value, label+"Arg")
		values[i] = value.Interface()
	}

	packed, err := method.Inputs.Pack(values...)
	if err != nil {
		return nil
	}
	return packed
}

// bigIntType is what every ABI integer too wide for a machine word maps to.
var bigIntType = reflect.TypeOf((*big.Int)(nil))

// fill draws a value of the given ABI type into the Go type the bindings map it to, so a tuple or a
// nested array is filled without this package knowing what is in it. Both halves are needed: the Go
// type says where a value goes, and the ABI type says which values the encoder will accept there --
// a negative drawn into a uint256 is refused, not truncated.
//
// An address argument is drawn as bytes like anything else, which is what keeps it off the accounts
// under observation: several of these contracts forward value to an address they are handed, and an
// account the harness checks to the wei must not be able to receive any that way.
func fill(t *rapid.T, typ abi.Type, v reflect.Value, label string) {
	switch typ.T {
	case abi.IntTy, abi.UintTy:
		switch {
		case v.Type() == bigIntType:
			v.Set(drawBig(t, typ, label))
		case typ.T == abi.IntTy:
			v.SetInt(rapid.Int64().Draw(t, label)) // truncated to the field's own width
		default:
			v.SetUint(rapid.Uint64().Draw(t, label))
		}

	case abi.BoolTy:
		v.SetBool(rapid.Bool().Draw(t, label))

	case abi.StringTy:
		v.SetString(rapid.StringN(0, 16, 64).Draw(t, label))

	case abi.BytesTy:
		v.SetBytes(rapid.SliceOfN(rapid.Byte(), 0, 64).Draw(t, label))

	case abi.SliceTy:
		length := rapid.IntRange(0, 3).Draw(t, label+"Len")
		v.Set(reflect.MakeSlice(v.Type(), length, length))
		for i := range length {
			fill(t, *typ.Elem, v.Index(i), label)
		}

	case abi.ArrayTy:
		for i := range v.Len() {
			fill(t, *typ.Elem, v.Index(i), label)
		}

	case abi.TupleTy:
		for i, elem := range typ.TupleElems {
			fill(t, *elem, v.Field(i), label)
		}

	default: // an address, a fixed byte array, a function: bytes, whatever they mean
		for i := range v.Len() {
			v.Index(i).SetUint(uint64(rapid.Byte().Draw(t, label)))
		}
	}
}

// drawBig draws a wide integer within what its type admits, straddling what a contract is likely to
// check: nothing, one, a plausible amount, and the largest value there is.
func drawBig(t *rapid.T, typ abi.Type, label string) reflect.Value {
	limit := new(big.Int).Lsh(big.NewInt(1), uint(typ.Size)) // one past the largest unsigned value
	if typ.T == abi.IntTy {
		limit.Rsh(limit, 1)
	}
	highest := new(big.Int).Sub(limit, big.NewInt(1))

	value := rapid.OneOf(
		rapid.Just(big.NewInt(0)),
		rapid.Just(big.NewInt(1)),
		rapid.Just(highest),
		rapid.Custom(func(t *rapid.T) *big.Int {
			drawn := new(big.Int).SetBytes(rapid.SliceOfN(rapid.Byte(), 32, 32).Draw(t, "wideValue"))
			return drawn.Mod(drawn, limit)
		}),
	).Draw(t, label)

	if typ.T == abi.IntTy && rapid.Bool().Draw(t, label+"Negative") {
		value = new(big.Int).Neg(value)
	}
	return reflect.ValueOf(value)
}
