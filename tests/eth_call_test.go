package tests

import (
	"math"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

func TestEthCall_CodeLargerThanMaxInitCodeSizeIsAccepted(t *testing.T) {
	tests := map[string]struct {
		codeSize int
	}{
		"max code size": {
			params.MaxCodeSize,
		},
		"max code size + 1": {
			params.MaxCodeSize + 1,
		},
		"max code size lfvm": {
			math.MaxUint16, // max code size supported by the LFVM
		},
		"max code size lfvm + 1": {
			math.MaxUint16 + 1,
		},
	}
	net := StartIntegrationTestNet(t)

	client, err := net.GetClient()
	if err != nil {
		t.Fatalf("Failed to connect to the integration test network: %v", err)
	}
	defer client.Close()

	rpcClient := client.Client()

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			accountWithHugeCode := "0x5555555555555555555555555555555555555555"

			txArguments := map[string]string{
				"to":   accountWithHugeCode,
				"gas":  "0xffffffffffffffff",
				"data": "0x00",
			}
			requestedBlock := "latest"
			stateOverrides := map[string]map[string]hexutil.Bytes{
				accountWithHugeCode: {
					"code": make([]byte, test.codeSize),
				},
			}

			var res interface{}
			err = rpcClient.Call(&res, "eth_call", txArguments, requestedBlock, stateOverrides)
			require.NoError(t, err)
		})
	}
}
