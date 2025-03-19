package light_client

import (
	"fmt"
	"math"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/tests"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

////////////////////////////////////////
// TODO: expand tests to cover not only Server but also light client
//       instance to sync up to the network, as well as new functionionalities
//       added to the Server through decorators.
////////////////////////////////////////

func TestServer_GetCommitteeCertificates_CanRetrieveCertificates(t *testing.T) {
	require := require.New(t)

	// start network
	net, client := startNetAndGetClient(t)

	// make providers
	providerFromClient, err := newServerFromClient(client.Client())
	require.NoError(err)
	t.Cleanup(providerFromClient.close)
	url := fmt.Sprintf("http://localhost:%d", net.GetJsonRpcPort())
	providerFromURL, err := newServerFromURL(url)
	require.NoError(err)
	t.Cleanup(providerFromURL.close)

	chainId := getChainIdFromClient(t, client.Client())

	for _, provider := range []*server{providerFromClient, providerFromURL} {

		// get certificates
		certs, err := provider.getCommitteeCertificates(0, math.MaxUint64)
		require.NoError(err)

		require.NotZero(len(certs))
		for _, cert := range certs {
			require.Equal(chainId.Uint64(), cert.Subject().ChainId)
		}
	}
}

func TestServer_GetBlockCertificates_CanRetrieveCertificates(t *testing.T) {
	require := require.New(t)

	// start network
	net, client := startNetAndGetClient(t)

	// make providers
	providerFromClient, err := newServerFromClient(client.Client())
	require.NoError(err)
	t.Cleanup(providerFromClient.close)
	url := fmt.Sprintf("http://localhost:%d", net.GetJsonRpcPort())
	providerFromURL, err := newServerFromURL(url)
	require.NoError(err)
	t.Cleanup(providerFromURL.close)

	chainId := getChainIdFromClient(t, client.Client())

	for _, provider := range []*server{providerFromClient, providerFromURL} {

		// get certificates
		certs, err := provider.getBlockCertificates(1, 100)
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

func TestServer_CanRequestMaxNumberOfResults(t *testing.T) {
	require := require.New(t)

	// start network
	net, client := startNetAndGetClient(t)

	// make providers
	providerFromClient, err := newServerFromClient(client.Client())
	require.NoError(err)
	t.Cleanup(providerFromClient.close)
	url := fmt.Sprintf("http://localhost:%d", net.GetJsonRpcPort())
	providerFromURL, err := newServerFromURL(url)
	require.NoError(err)
	t.Cleanup(providerFromURL.close)

	for _, provider := range []*server{providerFromClient, providerFromURL} {
		comCerts, err := provider.getCommitteeCertificates(0, math.MaxUint64)
		require.NoError(err)
		require.NotZero(len(comCerts))

		blockCerts, err := provider.getBlockCertificates(0, math.MaxUint64)
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

func getChainIdFromClient(t *testing.T, client *rpc.Client) *big.Int {
	t.Helper()
	var result hexutil.Big
	err := client.Call(&result, "eth_chainId")
	require.NoError(t, err)
	return result.ToInt()
}
