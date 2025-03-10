package provider

import (
	"fmt"
	"testing"

	"github.com/0xsoniclabs/sonic/ethapi"
	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/bls"
	"github.com/0xsoniclabs/sonic/scc/cert"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestRpcProvider_CanInitializeFromUrl(t *testing.T) {
	require := require.New(t)

	provider, err := NewRpcProviderFromURL("http://localhost:8545")
	t.Cleanup(provider.Close)
	require.NoError(err)
	require.NotNil(provider)
	require.False(provider.IsClosed())
}

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

func TestRpcProvider_IsClosed_Reports(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)

	t.Run("provider with nil client is closed", func(t *testing.T) {
		provider := NewRpcProviderFromClient(nil)
		require.True(provider.IsClosed())
	})

	t.Run("provider with client is not closed", func(t *testing.T) {
		client := NewMockRpcClient(ctrl)
		client.EXPECT().Close().AnyTimes()
		provider := NewRpcProviderFromClient(client)
		require.False(provider.IsClosed())
		provider.Close()
	})

	t.Run("provider with client can be closed", func(t *testing.T) {
		client := NewMockRpcClient(ctrl)
		client.EXPECT().Close()
		provider := NewRpcProviderFromClient(client)
		require.False(provider.IsClosed())
		provider.Close()
		require.True(provider.IsClosed())
	})

	t.Run("closed provider can be re-closed", func(t *testing.T) {
		client := NewMockRpcClient(ctrl)
		client.EXPECT().Close().AnyTimes()
		provider := NewRpcProviderFromClient(client)
		provider.Close()
		require.True(provider.IsClosed())
		provider.Close()
		require.True(provider.IsClosed())
	})
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

	provider, err := NewRpcProviderFromURL("not-a-url")
	require.Error(err)
	require.Nil(provider)
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

func TestRpcProvider_GetCommitteeCertificates_DropsExcessCertificates(t *testing.T) {
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
	certs, err := provider.GetCommitteeCertificates(0, 1)
	require.NoError(err)
	require.Len(certs, 1)
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

func TestRpcProvider_GetBlockCertificates_DropsExcessCertificates(t *testing.T) {
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
	certs, err := provider.GetBlockCertificates(0, 1)
	require.NoError(err)
	require.Len(certs, 1)
}

func TestRpcProvider_GetBlockCertificates_CanFetchLatestBlock(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	client := NewMockRpcClient(ctrl)
	provider := NewRpcProviderFromClient(client)

	latestBlockNumber := idx.Block(1024)
	// block certificates
	client.EXPECT().Call(gomock.Any(), "sonic_getBlockCertificates",
		"latest", "0x1").DoAndReturn(
		func(result *[]ethapi.BlockCertificate, method string, args ...interface{}) error {
			*result = []ethapi.BlockCertificate{
				makeBlockCertForNumber(latestBlockNumber),
			}
			return nil
		})

	// get block certificates
	blockCerts, err := provider.GetBlockCertificates(LatestBlock, 1)
	require.NoError(err)
	require.Len(blockCerts, 1)
	require.Equal(latestBlockNumber, blockCerts[0].Subject().Number)
}

func TestRpcProvider_GetCertificates_ReturnsCertificates(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	client := NewMockRpcClient(ctrl)
	provider := NewRpcProviderFromClient(client)

	// committee certificates
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
	comCerts, err := provider.GetCommitteeCertificates(0, 2)
	require.NoError(err)
	require.Len(comCerts, 2)
	require.Equal(scc.Period(0), comCerts[0].Subject().Period)
	require.Equal(scc.Period(1), comCerts[1].Subject().Period)

	// block certificates
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
	blockCerts, err := provider.GetBlockCertificates(0, 2)
	require.NoError(err)
	require.Len(blockCerts, 2)
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
