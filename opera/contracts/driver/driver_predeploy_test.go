package driver

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestGetContractBin(t *testing.T) {
	bin := GetContractBin()
	if len(bin) == 0 {
		t.Fatal("expected non-empty contract binary")
	}
}

func TestContractAddress(t *testing.T) {
	expected := common.HexToAddress("0xd100a01e00000000000000000000000000000000")
	if ContractAddress != expected {
		t.Fatalf("unexpected contract address: %s", ContractAddress.Hex())
	}
}

func TestContractAddress_NotZero(t *testing.T) {
	if ContractAddress == (common.Address{}) {
		t.Fatal("contract address should not be zero")
	}
}
