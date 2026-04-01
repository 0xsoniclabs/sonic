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
	"math"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func Test_Account_NewAccount(t *testing.T) {
	acc, err := NewAccount()
	require.NoError(t, err)
	require.NotNil(t, acc)
	require.NotNil(t, acc.PrivateKey)
}

func Test_Account_Address(t *testing.T) {
	acc, err := NewAccount()
	require.NoError(t, err)

	expectedAddr := crypto.PubkeyToAddress(acc.PrivateKey.PublicKey)
	require.Equal(t, &expectedAddr, acc.Address())
}

func Test_ToHexUint64(t *testing.T) {
	tests := []struct {
		name string
		val  uint64
	}{
		{"zero", 0},
		{"normal", 1234},
		{"max", math.MaxUint64},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hexVal := ToHexUint64(tt.val)
			require.Equal(t, tt.val, uint64(*hexVal))
		})
	}
}

func Test_ToHexUint(t *testing.T) {
	tests := []struct {
		name string
		val  uint
	}{
		{"zero", 0},
		{"normal", 5678},
		{"max", math.MaxUint},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hexVal := ToHexUint(tt.val)
			require.Equal(t, tt.val, uint(*hexVal))
		})
	}
}

func Test_ToHexBigInt(t *testing.T) {
	tests := []struct {
		name string
		val  *big.Int
	}{
		{"zero", big.NewInt(0)},
		{"normal", big.NewInt(1e18)},
		{"nil", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hexVal := ToHexBigInt(tt.val)
			if tt.val == nil {
				require.Nil(t, hexVal)
			} else {
				require.Equal(t, tt.val, (*big.Int)(hexVal))
			}
		})
	}
}

func Test_ToHexBytes(t *testing.T) {
	tests := []struct {
		name string
		val  []byte
	}{
		{"normal", []byte{0xde, 0xad, 0xbe, 0xef}},
		{"empty", []byte{}},
		{"nil", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hexVal := ToHexBytes(tt.val)
			require.Equal(t, tt.val, []byte(*hexVal))
		})
	}
}

func Test_ToEvmHeader(t *testing.T) {
	tests := []struct {
		name  string
		block TestBlock
	}{
		{"genesis", TestBlock{Number: 0, Hash: common.Hash{1}, ParentHash: common.Hash{2}}},
		{"standard", TestBlock{Number: 10, Hash: common.Hash{1}, ParentHash: common.Hash{2}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := ToEvmHeader(tt.block)
			require.Equal(t, big.NewInt(int64(tt.block.Number)), header.Number)
			require.Equal(t, tt.block.Hash, header.Hash)
			require.Equal(t, tt.block.ParentHash, header.ParentHash)
		})
	}
}
