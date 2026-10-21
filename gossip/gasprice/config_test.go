// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package gasprice

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

func TestConfig_DefaultMaxGasPrice_ProducesFreshPointers(t *testing.T) {
	referenceValue := big.NewInt(10000000 * params.GWei)
	typePtr := reflect.TypeOf(referenceValue)

	v1 := DefaultMaxGasPrice()
	ptr1 := reflect.ValueOf(v1).Pointer()

	v2 := DefaultMaxGasPrice()
	ptr2 := reflect.ValueOf(v2).Pointer()

	// Check pointer uniqueness
	if ptr1 == ptr2 {
		t.Errorf("DefaultMaxGasPrice returned a duplicate pointer")
	}
	// Check type
	if reflect.TypeOf(v1) != typePtr {
		t.Errorf("Returned value is not of type *big.Int")
	}
	// Check value
	require.Zero(t, v1.Cmp(referenceValue),
		"Returned value has incorrect value: got %v, want %v", v1, referenceValue)
	require.Zero(t, v2.Cmp(referenceValue),
		"Returned value has incorrect value: got %v, want %v", v2, referenceValue)
}
