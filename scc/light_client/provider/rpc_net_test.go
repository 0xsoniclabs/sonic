package provider

import (
	"fmt"
	"math"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/tests/contracts/counter_event_emitter"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

////////////////////////////////////////
// integration net tests
////////////////////////////////////////

func TestRpcProvider_GetCommitteeCertificates_CanRetrieveCertificates(t *testing.T) {
	require := require.New(t)

	// start network
	net, client := startNetAndGetClient(t)

	// make providers
	providerFromClient := NewRpcProviderFromClient(client.Client())
	t.Cleanup(providerFromClient.Close)
	url := fmt.Sprintf("http://localhost:%d", net.GetPort())
	providerFromURL, err := NewRpcProviderFromURL(url)
	require.NoError(err)
	t.Cleanup(providerFromURL.Close)

	chainId := getChainId(t, client.Client())

	for _, provider := range []*RpcProvider{providerFromClient, providerFromURL} {

		// get certificates
		certs, err := provider.GetCommitteeCertificates(0, math.MaxUint64)
		require.NoError(err)

		require.NotZero(len(certs))
		for _, cert := range certs {
			require.Equal(chainId.Uint64(), cert.Subject().ChainId)
			require.NotEmpty(cert.Subject().Committee)
			require.NotZero(cert.Subject().Committee.Members()[0].PublicKey)
			require.NotZero(cert.Subject().Committee.Members()[0].ProofOfPossession)
			require.NotZero(cert.Subject().Committee.Members()[0].VotingPower)
		}
	}
}

func TestRpcProvider_GetBlockCertificates_CanRetrieveCertificates(t *testing.T) {
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

	// make providers
	providerFromClient := NewRpcProviderFromClient(client.Client())
	t.Cleanup(providerFromClient.Close)
	url := fmt.Sprintf("http://localhost:%d", net.GetPort())
	providerFromURL, err := NewRpcProviderFromURL(url)
	require.NoError(err)
	t.Cleanup(providerFromURL.Close)

	chainId := getChainId(t, client.Client())

	for _, provider := range []*RpcProvider{providerFromClient, providerFromURL} {

		// get certificates
		certs, err := provider.GetBlockCertificates(1, numBlocks-1)
		require.NoError(err)

		// get headers
		headers, err := net.GetHeaders()
		require.NoError(err)

		require.NotZero(len(certs))
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

func TestRpcProvider_CanRequestMaxNumberOfResults(t *testing.T) {
	require := require.New(t)

	// start network
	net, client := startNetAndGetClient(t)

	// make providers
	providerFromClient := NewRpcProviderFromClient(client.Client())
	t.Cleanup(providerFromClient.Close)
	url := fmt.Sprintf("http://localhost:%d", net.GetPort())
	providerFromURL, err := NewRpcProviderFromURL(url)
	require.NoError(err)
	t.Cleanup(providerFromURL.Close)

	for _, provider := range []*RpcProvider{providerFromClient, providerFromURL} {
		comCerts, err := provider.GetCommitteeCertificates(0, math.MaxUint64)
		require.NoError(err)
		require.NotZero(len(comCerts))

		blockCerts, err := provider.GetBlockCertificates(0, math.MaxUint64)
		require.NoError(err)
		require.NotZero(len(blockCerts))
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

func getChainId(t *testing.T, client *rpc.Client) *big.Int {
	t.Helper()
	var result hexutil.Big
	err := client.Call(&result, "eth_chainId")
	require.NoError(t, err)
	return result.ToInt()
}
