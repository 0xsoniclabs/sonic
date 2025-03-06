package tests

import (
	"context"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/tests/contracts/counter"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestIntegrationTestNet_CanStartRestartAndStopIntegrationTestNet(t *testing.T) {
	net := StartIntegrationTestNet(t)
	if err := net.Restart(); err != nil {
		t.Fatalf("Failed to restart the test network: %v", err)
	}
	net.Stop()
}

func TestIntegrationTestNet_CanStartMultipleConsecutiveInstances(t *testing.T) {
	for range 2 {
		net := StartIntegrationTestNet(t)
		net.Stop()
	}
}

func TestIntegrationTestNet_CanFetchInformationFromTheNetwork(t *testing.T) {
	net := StartIntegrationTestNet(t)

	client, err := net.GetClient()
	if err != nil {
		t.Fatalf("Failed to connect to the integration test network: %v", err)
	}
	defer client.Close()

	block, err := client.BlockNumber(context.Background())
	if err != nil {
		t.Fatalf("Failed to get block number: %v", err)
	}

	if block == 0 || block > 1000 {
		t.Errorf("Unexpected block number: %v", block)
	}
}

func TestIntegrationTestNet_CanEndowAccountsWithTokens(t *testing.T) {
	net := StartIntegrationTestNet(t)

	client, err := net.GetClient()
	if err != nil {
		t.Fatalf("Failed to connect to the integration test network: %v", err)
	}

	address := common.Address{0x01}
	balance, err := client.BalanceAt(context.Background(), address, nil)
	if err != nil {
		t.Fatalf("Failed to get balance for account: %v", err)
	}

	for i := 0; i < 10; i++ {
		increment := int64(1000)

		receipt, err := net.EndowAccount(address, increment)
		if err != nil {
			t.Fatalf("Failed to endow account 1: %v", err)
		}
		if want, got := types.ReceiptStatusSuccessful, receipt.Status; want != got {
			t.Fatalf("Expected status %v, got %v", want, got)
		}

		want := balance.Add(balance, big.NewInt(int64(increment)))
		balance, err = client.BalanceAt(context.Background(), address, nil)
		if err != nil {
			t.Fatalf("Failed to get balance for account: %v", err)
		}
		if want, got := want, balance; want.Cmp(got) != 0 {
			t.Fatalf("Unexpected balance for account, got %v, wanted %v", got, want)
		}
		balance = want
	}
}

func TestIntegrationTestNet_CanDeployContracts(t *testing.T) {
	net := StartIntegrationTestNet(t)

	_, receipt, err := DeployContract(net, counter.DeployCounter)
	if err != nil {
		t.Fatalf("Failed to deploy contract: %v", err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		t.Errorf("Contract deployment failed: %v", receipt)
	}
}

func TestIntegrationTestNet_CanInteractWithContract(t *testing.T) {
	net := StartIntegrationTestNet(t)

	contract, _, err := DeployContract(net, counter.DeployCounter)
	if err != nil {
		t.Fatalf("Failed to deploy contract: %v", err)
	}

	receipt, err := net.Apply(contract.IncrementCounter)
	if err != nil {
		t.Fatalf("Failed to send transaction: %v", err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		t.Errorf("Contract deployment failed: %v", receipt)
	}
}
