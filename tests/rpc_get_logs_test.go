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
	"strings"
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests/contracts/indexed_logs"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestRpc_GetLogs_BlockTimeStampHexEncoded(t *testing.T) {

	session := getIntegrationTestNetSession(t, opera.GetAllegroUpgrades())

	// deploy a contract
	contract, receipt, err := DeployContract(session, indexed_logs.DeployIndexedLogs)
	require.NoError(t, err)
	require.Equal(t, receipt.Status, types.ReceiptStatusSuccessful)
	contractAddress := receipt.ContractAddress

	// Create logs
	receipt, err = session.Apply(contract.EmitEvents)
	require.NoError(t, err)
	require.Equal(t, receipt.Status, types.ReceiptStatusSuccessful)
	// ensure there is at least one log generated
	require.Greater(t, len(receipt.Logs), 0)

	// log structs as updated by
	// https://github.com/ethereum/go-ethereum/pull/32129/badae50d0316f299665bc2dae3daf6349f5abe44fe141ac9eeda0eaacf040c55R25
	// from go-ethereum v1.16.1 onwards
	var result []struct {
		BlockTimestamp any
	}

	arg := map[string]interface{}{
		"address":   []common.Address{contractAddress},
		"topics":    [][]common.Hash{{receipt.Logs[0].Topics[0]}},
		"fromBlock": hexutil.EncodeBig(receipt.BlockNumber),
		"toBlock":   hexutil.EncodeBig(receipt.BlockNumber),
	}

	client, err := session.GetClient()
	require.NoError(t, err)
	defer client.Close()

	err = client.Client().CallContext(t.Context(), &result, "eth_getLogs", arg)
	require.NoError(t, err)

	require.Greater(t, len(result), 0)
	require.NotNil(t, result[0].BlockTimestamp)

	genericTimeStamp, ok := result[0].BlockTimestamp.(string)
	require.True(t, ok)
	require.True(t, strings.HasPrefix(genericTimeStamp, "0x"))
}
