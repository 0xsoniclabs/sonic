package tests

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestSetStorage(t *testing.T) {
	require := require.New(t)

	// start network
	net, err := StartIntegrationTestNet(t.TempDir())
	require.NoError(err)
	defer net.Stop()

	// create a client
	client, err := net.GetClient()
	require.NoError(err, "failed to get client")
	defer client.Close()

	contractAddress := common.Address{1}.String()

	var result string
	rpcClient := client.Client()
	defer rpcClient.Close()

	err = rpcClient.Call(&result, "eth_call",
		map[string]interface{}{
			"to":   contractAddress,
			"data": "0x2e64cec1",
		},
		"latest",
		map[string]interface{}{
			contractAddress: map[string]interface{}{
				"code": contractCode,
				"state": map[string]interface{}{
					"0x0000000000000000000000000000000000000000000000000000000000000000": "0x000000000000000000000000000000000000000000000000000000000000002a",
				},
			},
		},
	)
	require.NoError(err)

	num, ok := big.NewInt(0).SetString(result, 0)
	require.True(ok)
	require.Equal(uint64(42), num.Uint64(), "Storage was not overridden")
}

var contractCode = "0x608060405234801561000f575f80fd5b5060043610610034575f3560e01c80632e64cec1146100385780636057361d14610056575b5f80fd5b610040610072565b60405161004d919061009b565b60405180910390f35b610070600480360381019061006b91906100e2565b61007a565b005b5f8054905090565b805f8190555050565b5f819050919050565b61009581610083565b82525050565b5f6020820190506100ae5f83018461008c565b92915050565b5f80fd5b6100c181610083565b81146100cb575f80fd5b50565b5f813590506100dc816100b8565b92915050565b5f602082840312156100f7576100f66100b4565b5b5f610104848285016100ce565b9150509291505056fea26469706673582212204e8daff0172cba88c37063e26299240060c3abfa2b021697bb2f7443e44c4c3864736f6c634300081a0033"

// // Simple storage contract with one number
// pragma solidity >=0.7.0 <0.9.0;
// contract Storage {
//
//     uint256 number;
//
//     function store(uint256 num) public {
//         number = num;
//     }
//
//     function retrieve() public view returns (uint256){
//         return number;
//     }
// }
