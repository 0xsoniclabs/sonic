package driverauth

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
	expected := common.HexToAddress("0xd100ae0000000000000000000000000000000000")
	if ContractAddress != expected {
		t.Fatalf("unexpected contract address: %s", ContractAddress.Hex())
	}
}
