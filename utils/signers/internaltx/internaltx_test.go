// Copyright 2025 Sonic Operations Ltd
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

package internaltx

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/status-im/keycard-go/hexutils"
	"github.com/stretchr/testify/require"
)

func TestIsInternal(t *testing.T) {
	require.True(t, IsInternal(types.NewTx(&types.LegacyTx{
		V: new(big.Int),
		R: new(big.Int),
		S: new(big.Int),
	})))
	require.True(t, IsInternal(types.NewTx(&types.DynamicFeeTx{
		V: new(big.Int),
		R: new(big.Int),
		S: new(big.Int),
	})))
	require.True(t, IsInternal(types.NewTx(&types.LegacyTx{
		V: new(big.Int),
		R: new(big.Int),
		S: big.NewInt(1),
	})))
	require.True(t, IsInternal(types.NewTx(&types.LegacyTx{
		V: new(big.Int),
		R: new(big.Int),
		S: new(big.Int).SetBytes(hexutils.HexToBytes("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")),
	})))
	require.False(t, IsInternal(types.NewTx(&types.LegacyTx{
		V: big.NewInt(1),
		R: big.NewInt(1),
		S: big.NewInt(1),
	})))
	require.False(t, IsInternal(types.NewTx(&types.DynamicFeeTx{
		V: big.NewInt(1),
		R: big.NewInt(1),
		S: big.NewInt(1),
	})))
	require.False(t, IsInternal(types.NewTx(&types.LegacyTx{
		V: big.NewInt(1),
		R: new(big.Int),
		S: new(big.Int),
	})))
	require.False(t, IsInternal(types.NewTx(&types.LegacyTx{
		V: new(big.Int),
		R: big.NewInt(1),
		S: new(big.Int),
	})))
}

func TestInternalSender(t *testing.T) {
	require.Equal(t, common.Address{}, InternalSender(types.NewTx(&types.LegacyTx{
		V: new(big.Int),
		R: new(big.Int),
		S: new(big.Int),
	})))
	example := common.HexToAddress("0x0000000000000000000000000000000000000001")
	require.Equal(t, example, InternalSender(types.NewTx(&types.LegacyTx{
		V: new(big.Int),
		R: new(big.Int),
		S: new(big.Int).SetBytes(example.Bytes()),
	})))
	example = common.HexToAddress("0x0000000000000000000000000000000000000100")
	require.Equal(t, example, InternalSender(types.NewTx(&types.LegacyTx{
		V: new(big.Int),
		R: new(big.Int),
		S: new(big.Int).SetBytes(example.Bytes()),
	})))
	example = common.HexToAddress("0x1000000000000000000000000000000000000000")
	require.Equal(t, example, InternalSender(types.NewTx(&types.LegacyTx{
		V: new(big.Int),
		R: new(big.Int),
		S: new(big.Int).SetBytes(example.Bytes()),
	})))
	example = common.HexToAddress("0x1000000000000000000000000000000000000001")
	require.Equal(t, example, InternalSender(types.NewTx(&types.LegacyTx{
		V: new(big.Int),
		R: new(big.Int),
		S: new(big.Int).SetBytes(example.Bytes()),
	})))
}
