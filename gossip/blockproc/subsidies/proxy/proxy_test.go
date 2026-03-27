package proxy

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestGetSlotForImplementation(t *testing.T) {
	slot := GetSlotForImplementation()
	expected := common.HexToHash("0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc")
	if slot != expected {
		t.Fatalf("unexpected slot: %s", slot.Hex())
	}
}

func TestGetCode(t *testing.T) {
	code := GetCode()
	if len(code) == 0 {
		t.Fatal("expected non-empty code")
	}

	// Verify it returns a copy, not the original
	code2 := GetCode()
	code[0] = 0xff
	if code2[0] == 0xff {
		t.Fatal("GetCode should return a copy")
	}
}
