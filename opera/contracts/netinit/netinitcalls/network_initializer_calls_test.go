package netinitcall

import (
	"math/big"
	"testing"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
)

func TestInitializeAll(t *testing.T) {
	data := InitializeAll(
		idx.Epoch(1),
		big.NewInt(1000000),
		common.HexToAddress("0x01"),
		common.HexToAddress("0x02"),
		common.HexToAddress("0x03"),
		common.HexToAddress("0x04"),
		common.HexToAddress("0x05"),
	)
	if len(data) == 0 {
		t.Fatal("expected non-empty data")
	}
	if len(data) < 4 {
		t.Fatal("data too short for ABI-encoded call")
	}
}
