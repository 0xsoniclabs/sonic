package provider

import (
	"context"
	"fmt"
	"math"
	"testing"

	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/tests/contracts/counter_event_emitter"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

func TestRPCProvider_GetCommitteeCertificates_CanRetrieveCertificates(t *testing.T) {
	require := require.New(t)

	// start network
	net, client := startNetAndGetClient(t)

	providerFromClient := NewRPCProviderFromClient(client)
	providerFromURL, err := NewRPCProviderFromURL(fmt.Sprintf("http://localhost:%d", net.GetPort()))
	require.NoError(err)

	for _, provider := range []*RPCProvider{providerFromClient, providerFromURL} {

		// get certificates
		certs, err := provider.GetCommitteeCertificates(0, math.MaxUint64)
		require.NoError(err)
		provider.Close()

		chainId, err := client.ChainID(context.Background())
		require.NoError(err)
		for _, cert := range certs {
			require.Equal(chainId.Uint64(), cert.Subject().ChainId)

		}
	}
}

func TestRPCProvider_GetCommitteeCertificates_ReportsErrorForNilClient(t *testing.T) {
	require := require.New(t)

	provider := NewRPCProviderFromClient(nil)

	// get certificates
	_, err := provider.GetCommitteeCertificates(0, 1)
	require.Error(err)
}

func TestRPCProvider_GetBlockCertificate_ReportsErrorForNilClient(t *testing.T) {
	require := require.New(t)

	provider := NewRPCProviderFromClient(nil)

	// get certificates
	_, err := provider.GetBlockCertificates(0, 1)
	require.Error(err)
}

func TestRPCProvider_NewRPCProvider_ReportsErrorForInvalidURL(t *testing.T) {
	require := require.New(t)

	_, err := NewRPCProviderFromURL("not-a-url")
	require.Error(err)
}

func TestRPCProvider_GetBlockCertificates_CanRetrieveCertificates(t *testing.T) {
	require := require.New(t)

	// start network
	net, client := startNetAndGetClient(t)

	// Produce a few blocks on the network. We use the counter contract since
	// it is also producing events.
	const numBlocks = 10
	counter, _, err := tests.DeployContract(net, counter_event_emitter.DeployCounterEventEmitter)
	require.NoError(err)
	for range numBlocks {
		_, err := net.Apply(counter.Increment)
		require.NoError(err, "failed to increment counter")
	}

	providerFromClient := NewRPCProviderFromClient(client)
	providerFromURL, err := NewRPCProviderFromURL(fmt.Sprintf("http://localhost:%d", net.GetPort()))
	require.NoError(err)

	for _, provider := range []*RPCProvider{providerFromClient, providerFromURL} {

		// get certificates
		certs, err := provider.GetBlockCertificates(1, 10)
		require.NoError(err)
		provider.Close()

		// get headers
		headers, err := net.GetHeaders()
		require.NoError(err)

		chainId, err := client.ChainID(context.Background())
		require.NoError(err)
		for _, cert := range certs {
			require.Equal(chainId.Uint64(), cert.Subject().ChainId)
			if cert.Subject().Number >= idx.Block(len(headers)) {
				continue
			}
			header := headers[cert.Subject().Number]
			require.Equal(chainId.Uint64(), cert.Subject().ChainId, "chain ID mismatch")
			require.Equal(header.Hash(), cert.Subject().Hash, "block hash mismatch")
			require.Equal(header.Root, cert.Subject().StateRoot, "state root mismatch")
		}
	}
}

func TestRPCProvider_CanRequestMaxNumberOfResults(t *testing.T) {
	require := require.New(t)

	// start network
	net, client := startNetAndGetClient(t)

	providerFromClient := NewRPCProviderFromClient(client)
	providerFromURL, err := NewRPCProviderFromURL(fmt.Sprintf("http://localhost:%d", net.GetPort()))
	require.NoError(err)

	for _, provider := range []*RPCProvider{providerFromClient, providerFromURL} {

		// get certificates
		certs, err := provider.GetCommitteeCertificates(0, math.MaxUint64)
		require.NoError(err)
		provider.Close()

		require.Len(certs, 1)
	}
}

////////////////////////////////////////
// helper functions
////////////////////////////////////////

func startNetAndGetClient(t *testing.T) (*tests.IntegrationTestNet, *ethclient.Client) {
	t.Helper()
	require := require.New(t)
	// start network
	net := tests.StartIntegrationTestNet(t)

	client, err := net.GetClient()
	require.NoError(err)
	return net, client
}
