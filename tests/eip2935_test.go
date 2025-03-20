package tests

import (
	"context"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/config"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

var (
	historyStorageAddress = common.HexToAddress("0x0000F90827F1C53a10cb7A02335B175320002935")
	senderAddr            = common.HexToAddress("0x3462413Af4609098e1E27A490f554f260213D685")
)

func TestEIP2935_IsAutomaticallyDeployedWithFakeNet(t *testing.T) {

	tests := map[string]func(t *testing.T) *IntegrationTestNet{
		"json genesis": func(t *testing.T) *IntegrationTestNet {
			return StartIntegrationTestNetWithJsonGenesis(t,
				IntegrationTestNetOptions{
					FeatureSet: opera.AllegroFeatures,
				})
		},
		"fake genesis": func(t *testing.T) *IntegrationTestNet {
			return StartIntegrationTestNetWithFakeGenesis(t,
				IntegrationTestNetOptions{
					FeatureSet: opera.AllegroFeatures,
				})
		},
	}

	for name, netConstructor := range tests {
		t.Run(name, func(t *testing.T) {
			net := netConstructor(t)

			client, err := net.GetClient()
			require.NoError(t, err)
			defer client.Close()

			code, err := client.CodeAt(context.Background(), historyStorageAddress, nil)
			require.NoError(t, err)
			require.Equal(t, params.HistoryStorageCode, code)

			nonce, err := client.NonceAt(context.Background(), historyStorageAddress, nil)
			require.NoError(t, err)
			require.Equal(t, uint64(1), nonce)
		})
	}
}

func TestEIP2935_HistoryContractIsNotDeployedBeforePrague(t *testing.T) {

	tests := map[string]func(t *testing.T) *IntegrationTestNet{
		"json genesis": func(t *testing.T) *IntegrationTestNet {
			return StartIntegrationTestNetWithJsonGenesis(t)
		},
		"fake genesis": func(t *testing.T) *IntegrationTestNet {
			return StartIntegrationTestNetWithFakeGenesis(t)
		},
	}

	for name, netConstructor := range tests {
		t.Run(name, func(t *testing.T) {
			net := netConstructor(t)

			client, err := net.GetClient()
			require.NoError(t, err)
			defer client.Close()

			code, err := client.CodeAt(context.Background(), historyStorageAddress, nil)
			require.NoError(t, err)
			require.Empty(t, code)

			nonce, err := client.NonceAt(context.Background(), historyStorageAddress, nil)
			require.NoError(t, err)
			require.Equal(t, uint64(0), nonce)
		})
	}
}

func TestEIP2935_DeployContract(t *testing.T) {

	net := StartIntegrationTestNet(t,
		IntegrationTestNetOptions{
			// < Allegro automatically deploys the history storage contract
			// < To test deployment, we need to use a feature set that does not already have the contract
			FeatureSet: opera.SonicFeatures,
			ModifyConfig: func(config *config.Config) {
				// the transaction to deploy the contract is not replay protected
				// This has the benefit that the same tx will work in both ethereum and sonic.
				// Nevertheless the default RPC configuration rejects this sort of transaction.
				config.Opera.AllowUnprotectedTxs = true
			},
		},
	)

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	// Deploy transaction as described in EIP-2935
	// https://eips.ethereum.org/EIPS/eip-2935
	// {
	// 	"type": "0x0",
	// 	"nonce": "0x0",
	// 	"to": null,
	// 	"gas": "0x3d090",
	// 	"gasPrice": "0xe8d4a51000",
	// 	"maxPriorityFeePerGas": null,
	// 	"maxFeePerGas": null,
	// 	"value": "0x0",
	// 	"input": "0x60538060095f395ff33373fffffffffffffffffffffffffffffffffffffffe14604657602036036042575f35600143038111604257611fff81430311604257611fff9006545f5260205ff35b5f5ffd5b5f35611fff60014303065500",
	// 	"v": "0x1b",
	// 	"r": "0x539",
	// 	"s": "0xaa12693182426612186309f02cfe8a80a0000",
	// 	"hash": "0x67139a552b0d3fffc30c0fa7d0c20d42144138c8fe07fc5691f09c1cce632e15"
	//   }

	v, ok := new(big.Int).SetString("0x1b", 0)
	require.True(t, ok)
	r, ok := new(big.Int).SetString("0x539", 0)
	require.True(t, ok)
	s, ok := new(big.Int).SetString("0xaa12693182426612186309f02cfe8a80a0000", 0)
	require.True(t, ok)

	payload := &types.LegacyTx{
		Nonce:    0,
		Gas:      0x3d090,
		GasPrice: new(big.Int).SetUint64(0xe8d4a51000),
		Value:    new(big.Int).SetUint64(0),
		Data:     common.Hex2Bytes("60538060095f395ff33373fffffffffffffffffffffffffffffffffffffffe14604657602036036042575f35600143038111604257611fff81430311604257611fff9006545f5260205ff35b5f5ffd5b5f35611fff60014303065500"),
		V:        v,
		R:        r,
		S:        s,
	}

	tx := types.NewTx(payload)

	// The transaction is pre EIP-155, (the chain ID is not included in the signature)
	sender, err := types.HomesteadSigner{}.Sender(tx)
	require.NoError(t, err)
	require.Equal(t, senderAddr, sender)

	_, err = net.EndowAccount(senderAddr, big.NewInt(1e18))
	require.NoError(t, err)

	receipt, err := net.Run(tx)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	code, err := client.CodeAt(context.Background(), historyStorageAddress, nil)
	require.NoError(t, err)
	require.Equal(t, params.HistoryStorageCode, code)

	nonce, err := client.NonceAt(context.Background(), historyStorageAddress, nil)
	require.NoError(t, err)
	require.Equal(t, uint64(1), nonce)
}
