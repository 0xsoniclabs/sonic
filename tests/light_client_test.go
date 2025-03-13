package tests

import (
	"fmt"
	"math"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/bls"
	"github.com/0xsoniclabs/sonic/scc/light_client"
	"github.com/0xsoniclabs/sonic/scc/light_client/provider"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

////////////////////////////////////////
// TODO: expand tests to cover not only provider.Server but also light client
//       instance to sync up to the network, as well as new functionionalities
//       added to the Server through decorators.
////////////////////////////////////////

func TestServer_GetCommitteeCertificates_CanRetrieveCertificates(t *testing.T) {
	require := require.New(t)

	// start network
	net, client := startNetAndGetClient(t)

	// make providers
	providerFromClient, err := provider.NewServerFromClient(client.Client())
	require.NoError(err)
	t.Cleanup(providerFromClient.Close)
	url := fmt.Sprintf("http://localhost:%d", net.GetJsonRpcPort())
	providerFromURL, err := provider.NewServerFromURL(url)
	require.NoError(err)
	t.Cleanup(providerFromURL.Close)

	chainId := getChainIdFromClient(t, client.Client())

	for _, provider := range []*provider.Server{providerFromClient, providerFromURL} {

		// get certificates
		certs, err := provider.GetCommitteeCertificates(0, math.MaxUint64)
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
	providerFromClient, err := provider.NewServerFromClient(client.Client())
	require.NoError(err)
	t.Cleanup(providerFromClient.Close)
	url := fmt.Sprintf("http://localhost:%d", net.GetJsonRpcPort())
	providerFromURL, err := provider.NewServerFromURL(url)
	require.NoError(err)
	t.Cleanup(providerFromURL.Close)

	chainId := getChainIdFromClient(t, client.Client())

	for _, provider := range []*provider.Server{providerFromClient, providerFromURL} {

		// get certificates
		certs, err := provider.GetBlockCertificates(1, 100)
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
	providerFromClient, err := provider.NewServerFromClient(client.Client())
	require.NoError(err)
	t.Cleanup(providerFromClient.Close)
	url := fmt.Sprintf("http://localhost:%d", net.GetJsonRpcPort())
	providerFromURL, err := provider.NewServerFromURL(url)
	require.NoError(err)
	t.Cleanup(providerFromURL.Close)

	for _, provider := range []*provider.Server{providerFromClient, providerFromURL} {
		comCerts, err := provider.GetCommitteeCertificates(0, math.MaxUint64)
		require.NoError(err)
		require.NotZero(len(comCerts))

		blockCerts, err := provider.GetBlockCertificates(0, math.MaxUint64)
		require.NoError(err)
		require.NotZero(len(blockCerts))
	}
}

func TestLightClient_CanSyncToIntegrationNetwork(t *testing.T) {
	require := require.New(t)

	// setup genesis committee
	key1 := bls.NewPrivateKey()
	key2 := bls.NewPrivateKey()
	key3 := bls.NewPrivateKey()
	genesis := scc.NewCommittee(
		makeMember(key1),
		makeMember(key2),
		makeMember(key3),
	)

	// start network
	netConfig := IntegrationTestNetOptions{
		GenesisCommittee: genesis,
	}
	net := StartIntegrationTestNet(t, netConfig)
	url := fmt.Sprintf("http://localhost:%d", net.GetJsonRpcPort())

	// create light client
	config := light_client.Config{
		Provider: url,
		Genesis:  genesis,
	}
	lightClient, err := light_client.NewLightClient(config)
	require.NoError(err)
	t.Cleanup(lightClient.Close)

	// sync
	// TODO: Enable this verification once the client uses the genesis committee
	// to sign block certificates.
	// WIP: https://github.com/0xsoniclabs/sonic/pull/90
	_, err = lightClient.Sync()
	// TODO: change this check to NoError once the client signs initial blocks.
	require.ErrorContains(err, "insufficient voting power")
	// require.Equal(netHead, head)
}

////////////////////////////////////////
// helper functions
////////////////////////////////////////

func startNetAndGetClient(t *testing.T) (*IntegrationTestNet, *ethclient.Client) {
	t.Helper()
	require := require.New(t)
	// start network
	net := StartIntegrationTestNet(t)

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

func makeMember(key bls.PrivateKey) scc.Member {
	return scc.Member{
		PublicKey:         key.PublicKey(),
		ProofOfPossession: key.GetProofOfPossession(),
		VotingPower:       1,
	}
}
