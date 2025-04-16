package tests

import (
	"fmt"
	"math"
	"testing"

	"github.com/0xsoniclabs/sonic/config"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

func TestEthCall_CodeLargerThanMaxInitCodeSizeIsNotAccepted(t *testing.T) {

	net := StartIntegrationTestNet(t)
	client, err := net.GetClient()
	if err != nil {
		t.Fatalf("Failed to connect to the integration test network: %v", err)
	}
	defer client.Close()

	netWithFlag := StartIntegrationTestNet(t, IntegrationTestNetOptions{
		ModifyConfig: func(config *config.Config) {
			config.Opera.RPCEVMSimulation = true
		},
	})
	clientWithEVM, err := netWithFlag.GetClient()
	if err != nil {
		t.Fatalf("Failed to connect to the integration test network: %v", err)
	}
	defer clientWithEVM.Close()

	tests := map[string]struct {
		rpcClient *rpc.Client
		codeSize  int
		err       error
	}{
		"max code size": {
			client.Client(),
			math.MaxUint16, // max code size supported by the LFVM
			nil,
		},
		"max code size + 1": {
			client.Client(),
			math.MaxUint16 + 1,
			fmt.Errorf("max code size exceeded"),
		},
		"max code size + 1 with EVM": {
			clientWithEVM.Client(),
			math.MaxUint16 + 1,
			nil,
		},
	}

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
			err = test.rpcClient.Call(&res, "eth_call", txArguments, requestedBlock, stateOverrides)
			if test.err == nil {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, test.err.Error())
			}
		})
	}
}
