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

package tests

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/tests/contracts/prevrandao"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/stretchr/testify/require"
)

func TestPrevRandao(t *testing.T) {
	net := StartIntegrationTestNet(t)

	// Deploy the contract.
	contract, _, err := DeployContract(net, prevrandao.DeployPrevrandao)
	require.NoError(t, err)
	// Collect the current PrevRandao fee from the head state.
	receipt, err := net.Apply(contract.LogCurrentPrevRandao)
	require.NoError(t, err)
	require.Len(t, receipt.Logs, 1, "expected exactly one log entry")

	entry, err := contract.ParseCurrentPrevRandao(*receipt.Logs[0])
	require.NoError(t, err, "failed to parse log entry")
	fromLog := entry.Prevrandao

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	block, err := client.BlockByNumber(t.Context(), receipt.BlockNumber)
	require.NoError(t, err)
	fromLatestBlock := block.MixDigest().Big() // MixDigest == MixHash == PrevRandao
	require.Zero(t, block.Difficulty().Uint64(), "block difficulty should be zero")

	// Collect the prevrandao from the archive.
	fromArchive, err := contract.GetPrevRandao(&bind.CallOpts{BlockNumber: receipt.BlockNumber})
	require.NoError(t, err)
	require.Greater(t, fromArchive.Sign(), 0, "prevrandao from archive should be positive")

	require.Equal(t, fromLatestBlock, fromLog, "prevrandao from log should match prevrandao from latest block")
	require.Equal(t, fromLatestBlock, fromArchive, "prevrandao from archive should match prevrandao from latest block")

	fromSecondLastBlock, err := contract.GetPrevRandao(&bind.CallOpts{BlockNumber: big.NewInt(receipt.BlockNumber.Int64() - 1)})
	require.NoError(t, err)
	require.NotEqual(t, fromSecondLastBlock, fromLatestBlock, "prevrandao from second last block should not match prevrandao from latest block")
}
