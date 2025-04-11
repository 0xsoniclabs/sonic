package tests

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

func TestEthCall_CodeLargerThanMaxInitCodeSizeIsNotAccepted(t *testing.T) {
	tests := map[string]struct {
		codeSize int
		err      error
	}{
		"max code size": {
			2 * 24576, // corresponds to the max init code size
			nil,
		},
		"max code size + 1": {
			2*24576 + 1,
			fmt.Errorf("max code size exceeded"),
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
			hugeCode := make([]byte, test.codeSize)

			params0 := map[string]string{
				"to":   "0x5555555555555555555555555555555555555555",
				"gas":  "0xffffffffffffffff",
				"data": "0x00",
			}
			params1 := "latest"
			params2 := map[string]map[string]hexutil.Bytes{
				"0x5555555555555555555555555555555555555555": {
					"code": hugeCode,
				},
			}

			var res interface{}
			err = rpcClient.Call(&res, "eth_call", params0, params1, params2)
			if test.err == nil {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, test.err.Error())
			}
		})
	}
}
