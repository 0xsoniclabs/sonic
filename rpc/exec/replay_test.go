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

package exec_test

import (
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/tests"
)

func TestStateAtTransaction_ReplaysTheTransactionsBefore(t *testing.T) {
	signer := tests.NewSigner(t)
	carmenCore := tests.NewCarmenCore(t, map[common.Address]tests.Account{
		signer.Address: {Balance: new(big.Int).Lsh(big.NewInt(1), 100)},
		counterAddress: {Code: counter},
	})
	sent := types.Transactions{
		signer.Sign(t, &types.LegacyTx{Nonce: 0, To: &counterAddress, Gas: 100_000, GasPrice: big.NewInt(2_000_000_000)}),
		signer.Sign(t, &types.LegacyTx{Nonce: 1, To: &counterAddress, Gas: 100_000, GasPrice: big.NewInt(2_000_000_000)}),
	}
	carmenCore.AppendBlock(tests.BlockSpec{Transactions: sent})
	carmenCore.PublishHead(1)
	harness := tests.NewHarness(t, carmenCore.FakeCore, noMethods)
	harness.WaitForHead(1)
	environment := harness.Environment
	head, err := environment.Latest(t.Context())
	require.NoError(t, err)
	block, err := environment.BlockAt(head, 1)
	require.NoError(t, err)

	message, statedb, _, err := environment.StateAtTransaction(head, block, 1)
	require.NoError(t, err)
	require.Equal(t, uint64(1), message.Nonce)
	require.Equal(t, common.BigToHash(big.NewInt(1)), statedb.GetState(counterAddress, common.Hash{}), "the first transaction ran")
	statedb.Release()

	message, statedb, _, err = environment.StateAtTransaction(head, block, 2)
	require.EqualError(t, err, "transaction index 2 out of range for block "+block.Hash.String())
	require.Nil(t, message)
	require.Nil(t, statedb)
	_, _, _, err = environment.StateAtTransaction(head, block, 3)
	require.ErrorContains(t, err, "transaction index 3 out of range")
	genesis, err := environment.BlockAt(head, 0)
	require.NoError(t, err)
	_, _, _, err = environment.StateAtTransaction(head, genesis, 0)
	require.EqualError(t, err, "no transaction in genesis")

	carmenCore.SetMode("State", tests.Fail)
	_, _, err = environment.ReplayState(head, block)
	var parent *exec.ParentStateError
	require.True(t, errors.As(err, &parent))
	require.ErrorIs(t, err, tests.ErrInjected)
	carmenCore.SetMode("State", tests.Normal)
	carmenCore.SetMode("Rules", tests.Fail)
	_, _, err = environment.ReplayState(head, block)
	require.ErrorContains(t, err, "failed to get vm config")
	states := carmenCore.States()
	require.Equal(t, states.Opened, states.Released, "every state is released")
}
