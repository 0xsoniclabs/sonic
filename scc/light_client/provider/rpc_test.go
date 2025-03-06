package provider

import (
	"context"
	"math"
	"testing"

	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

func TestRPCProvider_GetCommitteeCertificate_CanRetrieveCertificates(t *testing.T) {
	require := require.New(t)

	// start network
	_, client := startNetAndGetClient(t)
	provider := NewRPCProvider(client)

	// get certificates
	certs, err := provider.GetCommitteeCertificate(0, math.MaxUint64)
	require.NoError(err)
	provider.Close()

	chainId, err := client.ChainID(context.Background())
	require.NoError(err)
	for _, cert := range certs {
		require.Equal(chainId.Uint64(), cert.Subject().ChainId)
	}
}

func TestRPCProvider_GetCommitteeCertificate_ReportsError(t *testing.T) {
	require := require.New(t)

	provider := NewRPCProvider(nil)

	// get certificates
	_, err := provider.GetCommitteeCertificate(0, 1)
	require.Error(err)
}

// helper functions
func startNetAndGetClient(t *testing.T) (*tests.IntegrationTestNet, *ethclient.Client) {
	require := require.New(t)
	// start network
	net, err := tests.StartIntegrationTestNet(t.TempDir())
	require.NoError(err)

	client, err := net.GetClient()
	require.NoError(err)
	return net, client
}
