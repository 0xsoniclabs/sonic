package tests

import (
	"math"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/tests/contracts/counter"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
)

func TestCounter_CanIncrementAndReadCounterFromHead(t *testing.T) {
	net, err := StartIntegrationTestNet(t.TempDir())
	if err != nil {
		t.Fatalf("Failed to start the fake network: %v", err)
	}
	defer net.Stop()

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
	net, err := StartIntegrationTestNet(t.TempDir())
	if err != nil {
		t.Fatalf("Failed to start the fake network: %v", err)
	}
	defer net.Stop()

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
