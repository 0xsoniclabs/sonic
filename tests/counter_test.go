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
	"math"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/tests/contracts/counter"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/stretchr/testify/require"
)

func TestCounter_CanIncrementAndReadCounterFromHead(t *testing.T) {
	net := StartIntegrationTestNet(t)

	// Deploy the counter contract.
	contract, _, err := DeployContract(net, counter.DeployCounter)
	require.NoError(t, err)

	// Increment the counter a few times and check that the value is as expected.
	for i := 0; i < 10; i++ {
		counter, err := contract.GetCount(nil)
		require.NoError(t, err, "failed to get counter value")

		require.Equal(t, int64(i), counter.Int64(), "unexpected counter value")

		_, err = net.Apply(contract.IncrementCounter)
		require.NoError(t, err, "failed to apply increment counter contract")
	}
}

func TestCounter_CanReadHistoricCounterValues(t *testing.T) {
	net := StartIntegrationTestNet(t)

	// Deploy the counter contract.
	contract, receipt, err := DeployContract(net, counter.DeployCounter)
	require.NoError(t, err, "failed to deploy contract")

	// Increment the counter a few times and record the block height.
	updates := map[int]int{}                       // block height -> counter
	updates[int(receipt.BlockNumber.Uint64())] = 0 // contract deployed
	for i := 0; i < 10; i++ {
		receipt, err := net.Apply(contract.IncrementCounter)
		require.NoError(t, err, "failed to apply increment counter contract")

		updates[int(receipt.BlockNumber.Uint64())] = i + 1
	}

	minHeight := math.MaxInt
	maxHeight := 0
	for height := range updates {
		if height < minHeight {
			minHeight = height
		}
		if height > maxHeight {
			maxHeight = height
		}
	}

	// Check that the counter value at each block height is as expected.
	want := 0
	for i := minHeight; i <= maxHeight; i++ {
		if v, found := updates[i]; found {
			want = v
		}
		got, err := contract.GetCount(&bind.CallOpts{BlockNumber: big.NewInt(int64(i))})
		require.NoError(t, err, "failed to get counter value at block %d", i)
		require.Equal(t, int64(want), got.Int64(), "unexpected counter value at block %d", i)
	}
}
