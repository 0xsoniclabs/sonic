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
)

func TestCounter_CanIncrementAndReadCounterFromHead(t *testing.T) {
	net := StartIntegrationTestNet(t)

	// Deploy the counter contract.
	contract, _, err := DeployContract(net, counter.DeployCounter)
	if err != nil {
		t.Fatalf("failed to deploy contract; %v", err)
	}

	// Increment the counter a few times and check that the value is as expected.
	for i := 0; i < 10; i++ {
		counter, err := contract.GetCount(nil)
		if err != nil {
			t.Fatalf("failed to get counter value; %v", err)
		}

		if counter.Cmp(new(big.Int).SetInt64(int64(i))) != 0 {
			t.Fatalf("unexpected counter value; expected %d, got %v", i, counter)
		}

		_, err = net.Apply(contract.IncrementCounter)
		if err != nil {
			t.Fatalf("failed to increment counter; %v", err)
		}
	}
}

func TestCounter_CanReadHistoricCounterValues(t *testing.T) {
	net := StartIntegrationTestNet(t)

	// Deploy the counter contract.
	contract, receipt, err := DeployContract(net, counter.DeployCounter)
	if err != nil {
		t.Fatalf("failed to deploy contract; %v", err)
	}

	// Increment the counter a few times and record the block height.
	updates := map[int]int{}                       // block height -> counter
	updates[int(receipt.BlockNumber.Uint64())] = 0 // contract deployed
	for i := 0; i < 10; i++ {
		receipt, err := net.Apply(contract.IncrementCounter)
		if err != nil {
			t.Fatalf("failed to increment counter; %v", err)
		}
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
		if err != nil {
			t.Fatalf("failed to get counter value at block %d; %v", i, err)
		}
		if got.Cmp(big.NewInt(int64(want))) != 0 {
			t.Errorf("unexpected counter value at block %d; expected %d, got %v", i, want, got)
		}
	}
}
