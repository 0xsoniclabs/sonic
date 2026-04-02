package trace

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum/common"
)

func TestTraceChain(t *testing.T) {

	net := tests.StartIntegrationTestNet(t)

	for range 10 {
		_, _ = net.EndowAccount(common.Address{24}, big.NewInt(1))
	}
}
