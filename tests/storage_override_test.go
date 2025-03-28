package tests

import (
	"math/big"
	"strings"
	"testing"

	"github.com/0xsoniclabs/sonic/tests/contracts/storage"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestSetStorage_PreExisting_Contract_Storage_Temporarily_Overridden(t *testing.T) {
	net, err := StartIntegrationTestNet(t.TempDir())
	require.NoError(t, err, "Failed to start the fake network: %v", err)

	defer net.Stop()

	// Deploy the contract.
	contract, receipt, err := DeployContract(net, storage.DeployStorage)
	require.NoError(t, err, "failed to deploy contract; %v", err)

	checkStorage := func(t *testing.T) {
		t.Helper()

		valA, err := contract.GetA(nil)
		require.NoError(t, err, "failed to get A value; %v", err)
		require.Equal(t, big.NewInt(1), valA, "unexpected A value")

		valB, err := contract.GetB(nil)
		require.NoError(t, err, "failed to get B value; %v", err)
		require.Equal(t, big.NewInt(2), valB, "unexpected B value")

		valC, err := contract.GetC(nil)
		require.NoError(t, err, "failed to get C value; %v", err)
		require.Equal(t, big.NewInt(3), valC, "unexpected C value")
	}

	// check the initial storage values
	checkStorage(t)

	address := receipt.ContractAddress
	addressStr := address.String()

	// get the client to call RPC methods
	client, err := net.GetClient()
	require.NoError(t, err, "failed to get client")
	defer client.Close()

	rpcClient := client.Client()
	defer rpcClient.Close()

	// Parse the ABI
	parsedABI, err := abi.JSON(strings.NewReader(storage.StorageMetaData.ABI))
	require.NoError(t, err, "failed to parse ABI; %v", err)

	data, err := parsedABI.Pack("sumABC")
	require.NoError(t, err, "failed to pack function call; %v", err)

	// call eth_call of the contract to override the storage
	var result string
	err = rpcClient.Call(&result, "eth_call",
		map[string]interface{}{
			"to":   addressStr,
			"data": hexutil.Encode(data),
		},
		"latest",
		map[string]interface{}{
			addressStr: map[string]interface{}{
				"state": map[string]interface{}{
					"0x0000000000000000000000000000000000000000000000000000000000000000": "0x000000000000000000000000000000000000000000000000000000000000000a", // 10
					"0x0000000000000000000000000000000000000000000000000000000000000001": "0x000000000000000000000000000000000000000000000000000000000000000b", // 11
					"0x0000000000000000000000000000000000000000000000000000000000000002": "0x000000000000000000000000000000000000000000000000000000000000000c", // 12
				},
			},
		},
	)
	require.NoError(t, err)
	num, ok := big.NewInt(0).SetString(result, 0)
	require.True(t, ok)
	require.Equal(t, uint64(33), num.Uint64(), "storage was not overridden")

	// check the storage values after the call stays as it was before
	checkStorage(t)
}

func TestSetStorage_Contract_Not_On_Blockchain_Executed_With_Extra_Storage(t *testing.T) {
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
