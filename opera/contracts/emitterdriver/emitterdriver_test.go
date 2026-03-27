package emitterdriver

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestContractAddress(t *testing.T) {
	expected := common.HexToAddress("0xee00d10000000000000000000000000000000000")
	if ContractAddress != expected {
		t.Fatalf("unexpected contract address: %s", ContractAddress.Hex())
	}
}

func TestContractAddress_NotZero(t *testing.T) {
	zero := common.Address{}
	if ContractAddress == zero {
		t.Fatal("contract address should not be zero")
	}
}
