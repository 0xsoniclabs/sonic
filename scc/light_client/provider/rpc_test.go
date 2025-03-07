package provider

import (
	"fmt"
	"math"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/ethapi"
	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/bls"
	"github.com/0xsoniclabs/sonic/scc/cert"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/tests/contracts/counter_event_emitter"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestRpcProvider_ReportsErrorForNilClient(t *testing.T) {
	require := require.New(t)

	provider := NewRpcProviderFromClient(nil)

	// get committee certificates
	_, err := provider.GetCommitteeCertificates(0, 1)
	require.Error(err)

	// get block certificates
	_, err = provider.GetBlockCertificates(0, 1)
	require.Error(err)
}

func TestRpcProvider_FailsToRequestAfterClose(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	client := NewMockRpcClient(ctrl)

	provider := NewRpcProviderFromClient(client)

	// close provider
	client.EXPECT().Close()
	provider.Close()

	// get committee certificates
	_, err := provider.GetCommitteeCertificates(0, 1)
	require.Error(err)

	// get block certificates
	_, err = provider.GetBlockCertificates(0, 1)
	require.Error(err)
}

func TestRpcProvider_NewRpcProvider_ReportsErrorForInvalidURL(t *testing.T) {
	require := require.New(t)

	_, err := NewRpcProviderFromURL("not-a-url")
	require.Error(err)
}

func TestRpcProvider_GetCertificates_PropagatesErrorFromClientCall(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	client := NewMockRpcClient(ctrl)

	committeeError := fmt.Errorf("committee error")
	client.EXPECT().Call(gomock.Any(), "sonic_getCommitteeCertificates",
		gomock.Any(), gomock.Any()).Return(committeeError)

	blockError := fmt.Errorf("block error")
	client.EXPECT().Call(gomock.Any(), "sonic_getBlockCertificates",
		gomock.Any(), gomock.Any()).Return(blockError)

	provider := NewRpcProviderFromClient(client)

	// get committee certificates
	_, err := provider.GetCommitteeCertificates(0, 1)
	require.ErrorIs(err, committeeError)

	// get block certificates
	_, err = provider.GetBlockCertificates(0, 1)
	require.ErrorIs(err, blockError)
}

func TestRpcProvider_GetCommitteeCertificates_ReportsCorruptedCertificatesOutOfOrder(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	client := NewMockRpcClient(ctrl)
	provider := NewRpcProviderFromClient(client)

	tests := [][]ethapi.CommitteeCertificate{
		{
			makeCommitteeCertForPeriod(1),
		},
		{
			makeCommitteeCertForPeriod(0),
			makeCommitteeCertForPeriod(2),
		},
	}

	for _, committees := range tests {

		// client setup
		client.EXPECT().Call(gomock.Any(), "sonic_getCommitteeCertificates",
			gomock.Any(), gomock.Any()).
			DoAndReturn(
				func(result *[]ethapi.CommitteeCertificate, method string, args ...interface{}) error {
					*result = committees
					return nil
				})

		// get committee certificates
		_, err := provider.GetCommitteeCertificates(0, 3)
		require.ErrorContains(err, "out of order")
	}
}

func TestRpcProvider_GetCommitteeCertificates_FailsIfMoreCertificatesThanRequestedAreReturned(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	client := NewMockRpcClient(ctrl)
	provider := NewRpcProviderFromClient(client)

	client.EXPECT().Call(gomock.Any(), "sonic_getCommitteeCertificates",
		gomock.Any(), gomock.Any()).DoAndReturn(
		func(result *[]ethapi.CommitteeCertificate, method string, args ...interface{}) error {
			*result = []ethapi.CommitteeCertificate{
				makeCommitteeCertForPeriod(0),
				makeCommitteeCertForPeriod(1),
			}
			return nil
		})

	// get committee certificates
	_, err := provider.GetCommitteeCertificates(0, 1)
	require.ErrorContains(err, "Too many certificates")
}

func TestRpcProvider_GetBlockCertificates_ReportsCorruptedCertificatesOutOfOrder(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	client := NewMockRpcClient(ctrl)
	provider := NewRpcProviderFromClient(client)

	tests := [][]ethapi.BlockCertificate{
		{
			makeBlockCertForNumber(1),
		},
		{
			makeBlockCertForNumber(0),
			makeBlockCertForNumber(2),
		},
	}

	for _, blocks := range tests {
		client.EXPECT().Call(gomock.Any(), "sonic_getBlockCertificates",
			gomock.Any(), gomock.Any()).DoAndReturn(
			func(result *[]ethapi.BlockCertificate, method string, args ...interface{}) error {
				*result = blocks
				return nil
			})

		// get block certificates
		_, err := provider.GetBlockCertificates(0, 3)
		require.ErrorContains(err, "out of order")
	}
}

func TestRpcProvider_GetBlockCertificates_FailsIfMoreCertificatesThanRequestedAreReturned(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	client := NewMockRpcClient(ctrl)
	provider := NewRpcProviderFromClient(client)

	client.EXPECT().Call(gomock.Any(), "sonic_getBlockCertificates",
		gomock.Any(), gomock.Any()).DoAndReturn(
		func(result *[]ethapi.BlockCertificate, method string, args ...interface{}) error {
			*result = []ethapi.BlockCertificate{
				makeBlockCertForNumber(0),
				makeBlockCertForNumber(1),
			}
			return nil
		})

	// get block certificates
	_, err := provider.GetBlockCertificates(0, 1)
	require.ErrorContains(err, "Too many certificates")
}

/// helper functions

func makeCommitteeCertForPeriod(period scc.Period) ethapi.CommitteeCertificate {
	return ethapi.CommitteeCertificate{
		ChainId:   1,
		Period:    uint64(period),
		Members:   []scc.Member{},
		Signers:   cert.BitSet[scc.MemberId]{},
		Signature: bls.Signature{},
	}
}

func makeBlockCertForNumber(number idx.Block) ethapi.BlockCertificate {
	return ethapi.BlockCertificate{
		ChainId:   1,
		Number:    uint64(number),
		Hash:      [32]byte{},
		StateRoot: [32]byte{},
	}
}

////////////////////////////////////////
// net tests
////////////////////////////////////////

func TestRpcProvider_GetCommitteeCertificates_CanRetrieveCertificates(t *testing.T) {
	require := require.New(t)

	// start network
	net, client := startNetAndGetClient(t)

	url := fmt.Sprintf("http://localhost:%d", net.GetPort())

	providerFromClient := NewRpcProviderFromClient(client.Client())
	providerFromURL, err := NewRpcProviderFromURL(url)
	require.NoError(err)

	chainId := getChainId(t, client.Client())

	for _, provider := range []*RpcProvider{providerFromClient, providerFromURL} {

		// get certificates
		certs, err := provider.GetCommitteeCertificates(0, math.MaxUint64)
		require.NoError(err)
		provider.Close()

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

	url := fmt.Sprintf("http://localhost:%d", net.GetPort())

	providerFromClient := NewRpcProviderFromClient(client.Client())
	providerFromURL, err := NewRpcProviderFromURL(url)
	require.NoError(err)

	chainId := getChainId(t, client.Client())

	for _, provider := range []*RpcProvider{providerFromClient, providerFromURL} {

		// get certificates
		certs, err := provider.GetBlockCertificates(1, numBlocks-1)
		require.NoError(err)
		provider.Close()

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

	url := fmt.Sprintf("http://localhost:%d", net.GetPort())

	providerFromClient := NewRpcProviderFromClient(client.Client())
	providerFromURL, err := NewRpcProviderFromURL(url)
	require.NoError(err)

	for _, provider := range []*RpcProvider{providerFromClient, providerFromURL} {
		comCerts, err := provider.GetCommitteeCertificates(0, math.MaxUint64)
		require.NoError(err)
		provider.Close()
		require.NotZero(len(comCerts))

		blockCerts, err := provider.GetBlockCertificates(0, math.MaxUint64)
		require.NoError(err)
		provider.Close()
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
