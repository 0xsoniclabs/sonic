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

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests/contracts/transientstorage"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/stretchr/testify/require"
)

func TestTransientStorage_TransientStorageIsValidInTransaction(t *testing.T) {
	session := getIntegrationTestNetSession(t, opera.GetSonicUpgrades())
	t.Parallel()

	// Deploy the transient storage contract
	contract, _, err := DeployContract(session, transientstorage.DeployTransientstorage)
	require.NoError(t, err, "failed to deploy contract")

	// Get the value from the contract before changing it
	valueBefore, err := contract.GetValue(nil)
	require.NoError(t, err, "failed to get value")

	// Store the value in transient storage value
	receipt, err := session.Apply(contract.StoreValue)
	require.NoError(t, err, "failed to store value")

	// Check that the value was stored during transaction and emitted to logs
	require.Equal(t, 1, len(receipt.Logs), "unexpected number of logs; expected 1")

	// Get the value from the log
	logValue, err := contract.ParseStoredValue(*receipt.Logs[0])
	require.NoError(t, err, "failed to parse log")
	fromLog := logValue.Value

	// Get the value from the archive at time of store transaction
	fromArchive, err := contract.GetValue(&bind.CallOpts{BlockNumber: receipt.BlockNumber})
	require.NoError(t, err, "failed to get transient value from archive")

	// Get the value from the archive from head
	fromArchiveHead, err := contract.GetValue(nil)
	require.NoError(t, err, "failed to get transient value from archive at head time")

	// Check that all non log values are zero
	require.Zero(t, valueBefore.Uint64(), "value before should be zero")
	require.Zero(t, fromArchive.Uint64(), "from archive should be zero")
	require.Zero(t, fromArchiveHead.Uint64(), "from archive head should be zero")

	// Check that the log value is the same as set in contract
	require.Equal(t, uint64(42), fromLog.Uint64(), "unexpected log value")
}
