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

package ethapi

import (
	"math/big"
	"testing"

	rpctest "github.com/0xsoniclabs/sonic/api/rpc_test"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func Test_TraceSimpleTransfer(t *testing.T) {
	acc1, err := rpctest.NewWallet()
	require.NoError(t, err)
	acc2, err := rpctest.NewWallet()
	require.NoError(t, err)
	transferBalance := big.NewInt(1e17)

	be := rpctest.NewBackendBuilder(t).
		WithAccount(*acc1.Address(), rpctest.Account{Balance: big.NewInt(1e18)}).
		WithAccount(*acc2.Address(), rpctest.Account{}).
		WithBlockHistory(
			[]rpctest.Block{
				{
					Number: 1,
					Hash:   common.HexToHash("0x1"),
				},
				{
					Number:     2,
					Hash:       common.HexToHash("0x2"),
					ParentHash: common.HexToHash("0x1"),
				},
			},
		).
		Build()

	api := NewPublicTxTraceAPI(be, 100_000)

	txRequest1 := TransactionArgs{
		From:     acc1.Address(),
		To:       acc2.Address(),
		Nonce:    rpctest.ToHexUint64(0),
		Gas:      rpctest.ToHexUint64(8_000_000),
		GasPrice: rpctest.ToHexBigInt(big.NewInt(1)),
		Value:    rpctest.ToHexBigInt(transferBalance),
	}

	res, err := api.Call(
		t.Context(),
		txRequest1,
		[]string{"stateDiff"},
		rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber),
		&TraceCallConfig{},
	)
	require.NoError(t, err)
	require.NotNil(t, res)

	// check state diff for correct balance changes
	require.NotNil(t, res.StateDiff)
	require.NotNil(t, res.StateDiff[*acc1.Address()])
	require.NotNil(t, res.StateDiff[*acc2.Address()])

	balanceDiff1, ok1 := res.StateDiff[*acc1.Address()].Balance.(map[string]any)
	require.True(t, ok1)
	balanceDiff2, ok2 := res.StateDiff[*acc2.Address()].Balance.(map[string]any)
	require.True(t, ok2)

	require.NotNil(t, balanceDiff1["*"])
	require.NotNil(t, balanceDiff2["+"])

	balanceAmount2, ok := balanceDiff2["+"].(*hexutil.Big)
	require.True(t, ok)
	require.Equal(t, transferBalance, (*big.Int)(balanceAmount2))

	// real state should not be mutated after trace call
	state, _, err := be.StateAndBlockByNumberOrHash(t.Context(), rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber))
	require.NoError(t, err)
	require.NotNil(t, state)
	require.Equal(t, uint256.NewInt(0), state.GetBalance(*acc2.Address()))
}
