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
	"testing"

	"github.com/0xsoniclabs/sonic/tests/contracts/basefee"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/stretchr/testify/require"
)

func TestBaseFee_CanReadBaseFeeFromHeadAndBlockAndHistory(t *testing.T) {
	net := StartIntegrationTestNet(t)

	// Deploy the base fee contract.
	contract, _, err := DeployContract(net, basefee.DeployBasefee)
	require.NoError(t, err)

	// Collect the current base fee from the head state.
	receipt, err := net.Apply(contract.LogCurrentBaseFee)
	require.NoError(t, err)
	require.Len(t, receipt.Logs, 1, "expected exactly one log entry for the base fee")

	entry, err := contract.ParseCurrentFee(*receipt.Logs[0])
	require.NoError(t, err)
	fromLog := entry.Fee

	// Collect the base fee from the block header.
	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	block, err := client.BlockByNumber(t.Context(), receipt.BlockNumber)
	require.NoError(t, err)
	fromBlock := block.BaseFee()

	// Collect the base fee from the archive.
	fromArchive, err := contract.GetBaseFee(&bind.CallOpts{BlockNumber: receipt.BlockNumber})
	require.NoError(t, err)

	require.Positive(t, fromLog.Int64(),
		"base fee should be non-negative",
	)
	require.Equal(t, fromLog, fromBlock,
		"base fee from log should match base fee from block header",
	)
	require.Equal(t, fromLog, fromArchive,
		"base fee from log should match base fee from archive",
	)
}
