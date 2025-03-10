package provider

import (
	"fmt"
	"testing"

	"github.com/0xsoniclabs/sonic/ethapi"
	"github.com/0xsoniclabs/sonic/scc"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestServer_NewServer_CanInitializeFromUrl(t *testing.T) {
	require := require.New(t)

	provider, err := NewServerFromURL("http://localhost:8545")
	t.Cleanup(provider.Close)
	require.NoError(err)
	require.NotNil(provider)
	require.False(provider.IsClosed())
}

func TestServer_NewServer_ReportsErrorForNilClient(t *testing.T) {
	require := require.New(t)

	provider, err := NewServerFromClient(nil)
	require.Error(err)
	require.Nil(provider)
}

func TestServer_NewServer_ReportsErrorForInvalidURL(t *testing.T) {
	require := require.New(t)

	provider, err := NewServerFromURL("not-a-url")
	require.Error(err)
	require.Nil(provider)
}

func TestServer_IsClosed_Reports(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)

	t.Run("provider with client is not closed", func(t *testing.T) {
		client := NewMockRpcClient(ctrl)
		client.EXPECT().Close().AnyTimes()
		provider, err := NewServerFromClient(client)
		require.NoError(err)
		require.False(provider.IsClosed())
		provider.Close()
	})

	t.Run("provider with client can be closed", func(t *testing.T) {
		client := NewMockRpcClient(ctrl)
		client.EXPECT().Close()
		provider, err := NewServerFromClient(client)
		require.NoError(err)
		require.False(provider.IsClosed())
		provider.Close()
		require.True(provider.IsClosed())
	})

	t.Run("closed provider can be re-closed", func(t *testing.T) {
		client := NewMockRpcClient(ctrl)
		client.EXPECT().Close().AnyTimes()
		provider, err := NewServerFromClient(client)
		require.NoError(err)
		provider.Close()
		require.True(provider.IsClosed())
		provider.Close()
		require.True(provider.IsClosed())
	})
}

func TestServer_FailsToRequestAfterClose(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	client := NewMockRpcClient(ctrl)

	provider, err := NewServerFromClient(client)
	require.NoError(err)

	// close provider
	client.EXPECT().Close()
	provider.Close()

	// get committee certificates
	_, err = provider.GetCommitteeCertificates(0, 1)
	require.Error(err)

	// get block certificates
	_, err = provider.GetBlockCertificates(0, 1)
	require.Error(err)
}

func TestServer_GetCertificates_PropagatesErrorFromClientCall(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	client := NewMockRpcClient(ctrl)

	committeeError := fmt.Errorf("committee error")
	client.EXPECT().Call(gomock.Any(), "sonic_getCommitteeCertificates",
		gomock.Any(), gomock.Any()).Return(committeeError)

	blockError := fmt.Errorf("block error")
	client.EXPECT().Call(gomock.Any(), "sonic_getBlockCertificates",
		gomock.Any(), gomock.Any()).Return(blockError)

	provider, err := NewServerFromClient(client)
	require.NoError(err)

	// get committee certificates
	_, err = provider.GetCommitteeCertificates(0, 1)
	require.ErrorIs(err, committeeError)

	// get block certificates
	_, err = provider.GetBlockCertificates(0, 1)
	require.ErrorIs(err, blockError)
}

func TestServer_GetCommitteeCertificates_ReportsCorruptedCertificatesOutOfOrder(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)

	tests := [][]ethapi.CommitteeCertificate{
		{
			ethapi.CommitteeCertificate{Period: uint64(1)},
		},
		{
			ethapi.CommitteeCertificate{Period: uint64(0)},
			ethapi.CommitteeCertificate{Period: uint64(2)},
		},
	}

	for _, committees := range tests {
		client := NewMockRpcClient(ctrl)
		provider, err := NewServerFromClient(client)
		require.NoError(err)

		// client setup
		client.EXPECT().Call(gomock.Any(), "sonic_getCommitteeCertificates",
			gomock.Any(), gomock.Any()).
			DoAndReturn(
				func(result *[]ethapi.CommitteeCertificate, method string, args ...interface{}) error {
					*result = committees
					return nil
				})

		// get committee certificates
		_, err = provider.GetCommitteeCertificates(0, 3)
		require.ErrorContains(err, "out of order")
	}
}

func TestServer_GetCommitteeCertificates_DropsExcessCertificates(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	client := NewMockRpcClient(ctrl)
	provider, err := NewServerFromClient(client)
	require.NoError(err)

	client.EXPECT().Call(gomock.Any(), "sonic_getCommitteeCertificates",
		gomock.Any(), gomock.Any()).DoAndReturn(
		func(result *[]ethapi.CommitteeCertificate, method string, args ...interface{}) error {
			*result = []ethapi.CommitteeCertificate{
				ethapi.CommitteeCertificate{Period: uint64(0)},
				ethapi.CommitteeCertificate{Period: uint64(1)},
			}
			return nil
		})

	// get committee certificates
	certs, err := provider.GetCommitteeCertificates(0, 1)
	require.NoError(err)
	require.Len(certs, 1)
}

func TestServer_GetBlockCertificates_ReportsCorruptedCertificatesOutOfOrder(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	client := NewMockRpcClient(ctrl)
	provider, err := NewServerFromClient(client)
	require.NoError(err)

	tests := [][]ethapi.BlockCertificate{
		{
			ethapi.BlockCertificate{Number: uint64(1)},
		},
		{
			ethapi.BlockCertificate{Number: uint64(0)},
			ethapi.BlockCertificate{Number: uint64(2)},
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

func TestServer_GetBlockCertificates_DropsExcessCertificates(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	client := NewMockRpcClient(ctrl)
	provider, err := NewServerFromClient(client)
	require.NoError(err)

	client.EXPECT().Call(gomock.Any(), "sonic_getBlockCertificates",
		gomock.Any(), gomock.Any()).DoAndReturn(
		func(result *[]ethapi.BlockCertificate, method string, args ...interface{}) error {
			*result = []ethapi.BlockCertificate{
				ethapi.BlockCertificate{Number: uint64(0)},
				ethapi.BlockCertificate{Number: uint64(1)},
			}
			return nil
		})

	// get block certificates
	certs, err := provider.GetBlockCertificates(0, 1)
	require.NoError(err)
	require.Len(certs, 1)
}

func TestServer_GetBlockCertificates_CanFetchLatestBlock(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	client := NewMockRpcClient(ctrl)
	provider, err := NewServerFromClient(client)
	require.NoError(err)

	latestBlockNumber := idx.Block(1024)
	// block certificates
	client.EXPECT().Call(gomock.Any(), "sonic_getBlockCertificates",
		"latest", "0x1").DoAndReturn(
		func(result *[]ethapi.BlockCertificate, method string, args ...interface{}) error {
			*result = []ethapi.BlockCertificate{
				ethapi.BlockCertificate{Number: uint64(latestBlockNumber)},
			}
			return nil
		})

	// get block certificates
	blockCerts, err := provider.GetBlockCertificates(LatestBlock, 1)
	require.NoError(err)
	require.Len(blockCerts, 1)
	require.Equal(latestBlockNumber, blockCerts[0].Subject().Number)
}

func TestServer_GetCertificates_ReturnsCertificates(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	client := NewMockRpcClient(ctrl)
	provider, err := NewServerFromClient(client)
	require.NoError(err)

	// committee certificates
	client.EXPECT().Call(gomock.Any(), "sonic_getCommitteeCertificates",
		gomock.Any(), gomock.Any()).DoAndReturn(
		func(result *[]ethapi.CommitteeCertificate, method string, args ...interface{}) error {
			*result = []ethapi.CommitteeCertificate{
				ethapi.CommitteeCertificate{Period: uint64(0)},
				ethapi.CommitteeCertificate{Period: uint64(1)},
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
				ethapi.BlockCertificate{Number: uint64(0)},
				ethapi.BlockCertificate{Number: uint64(1)},
			}
			return nil
		})

	// get block certificates
	blockCerts, err := provider.GetBlockCertificates(0, 2)
	require.NoError(err)
	require.Len(blockCerts, 2)
}
