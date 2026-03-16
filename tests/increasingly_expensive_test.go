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

package tests

import (
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests/contracts/increasingly_expensive"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestIncreasinglyExpensive_ContractIsIncreasinglyExpensive(t *testing.T) {

	session := getIntegrationTestNetSession(t, opera.GetSonicUpgrades())
	t.Parallel()

	// Deploy the increasingly expensive contract.
	contract, receipt, err := DeployContract(session, increasingly_expensive.DeployIncreasinglyExpensive)
	require.NoError(t, err, "failed to deploy contract; %v", err)
	require.Equal(t, receipt.Status, types.ReceiptStatusSuccessful)

	// The first call to the contract is the most expensive, because the storage address is written for the first time.
	receipt, err = session.Apply(contract.IncrementAndLoop)
	require.NoError(t, err, "failed to apply increment counter contract")
	require.Equal(t, receipt.Status, types.ReceiptStatusSuccessful)

	lastGasUsed := uint64(0)
	for i := 0; i < 10; i++ {
		receipt, err = session.Apply(contract.IncrementAndLoop)
		require.NoError(t, err, "failed to apply increment counter contract")
		require.Equal(t, receipt.Status, types.ReceiptStatusSuccessful)

		require.Greater(t, receipt.GasUsed, lastGasUsed, "gas used should be greater than previous iteration")
		lastGasUsed = receipt.GasUsed
	}
}
