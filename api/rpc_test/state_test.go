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
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func Test_newTestState(t *testing.T) {
	ts := NewTestState(t)
	require.NotNil(t, ts)
	require.NotNil(t, ts.StateDB)
}

func Test_testState_setAccount(t *testing.T) {
	ts := NewTestState(t)
	addr := common.HexToAddress("0x1")

	store := map[common.Hash]common.Hash{
		common.HexToHash("0x1"): common.HexToHash("0x2"),
		common.HexToHash("0x3"): common.HexToHash("0x4"),
	}

	acc := Account{
		Nonce:   10,
		Balance: big.NewInt(1000),
		Code:    []byte{0x60, 0x01},
		Store:   store,
	}

	ts.setAccount(addr, acc)

	require.Equal(t, acc.Nonce, ts.GetNonce(addr))
	require.Equal(t, uint256.NewInt(acc.Balance.Uint64()), ts.GetBalance(addr))
	require.Equal(t, acc.Code, ts.GetCode(addr))
	require.Equal(t, store[common.HexToHash("0x1")], ts.GetState(addr, common.HexToHash("0x1")))
	require.Equal(t, store[common.HexToHash("0x3")], ts.GetState(addr, common.HexToHash("0x3")))
}
