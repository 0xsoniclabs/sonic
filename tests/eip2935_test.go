package tests

import (
	"context"
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

var (
	historyStorageAddress = common.HexToAddress("0x0000F90827F1C53a10cb7A02335B175320002935")
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
