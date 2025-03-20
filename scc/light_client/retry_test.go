package light_client

import (
	"fmt"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/cert"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestRetry_NewRetry_CanInitializeFromProvider(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	provider := NewMockprovider(ctrl)

	retry := newRetry(provider, 3, time.Duration(0))
	require.NotNil(retry)
}

func TestRetry_NewRetry_CorrectlyHandlesZeroTimeout(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	provider := NewMockprovider(ctrl)

	retry := newRetry(provider, 3, time.Duration(0))
	require.Equal(time.Second*10, retry.timeout)
	retry = newRetry(provider, 3, 1*time.Second)
	require.Equal(time.Second*1, retry.timeout)
}

func TestRetry_Close_ClosesProvider(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := NewMockprovider(ctrl)
	provider.EXPECT().close().Times(1)

	retry := newRetry(provider, 3, time.Duration(0))
	retry.close()
}

func TestRetry_retry_RetriesWhenProviderFails(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)

	retries := uint(3)

	provider := NewMockprovider(ctrl)
	provider.EXPECT().getCommitteeCertificates(gomock.Any(), gomock.Any()).
		Return(nil, fmt.Errorf("provider failed")).Times(int(retries + 1))

	certs, err := retry(retries, time.Second*10, func() (any, error) {
		return provider.getCommitteeCertificates(scc.Period(1), uint64(1))
	})
	require.Error(err)
	require.Nil(certs)
}

func TestRetry_retry_WaitsBetweenRetries(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	provider := NewMockprovider(ctrl)

	retries := uint(3)
	timeout := time.Second * 10

	provider.EXPECT().getCommitteeCertificates(gomock.Any(), gomock.Any()).
		Return(nil, fmt.Errorf("provider failed")).Times(int(retries + 1))

	start := time.Now()
	_, _ = retry(retries, timeout, func() (any, error) {
		return provider.getCommitteeCertificates(scc.Period(1), uint64(1))
	})
	duration := time.Since(start)

	require.LessOrEqual(duration.Seconds(), timeout.Seconds())
}

func TestRetry_retry_ReportsTimeoutError(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	provider := NewMockprovider(ctrl)

	retries := uint(3)
	timeout := time.Millisecond * 1

	provider.EXPECT().getCommitteeCertificates(gomock.Any(), gomock.Any()).
		Return(nil, fmt.Errorf("provider failed")).AnyTimes()

	_, err := retry(retries, timeout, func() (any, error) {
		return provider.getCommitteeCertificates(scc.Period(1), uint64(1))
	})
	require.ErrorContains(err, "exceeded timeout")
}

func TestRetry_retry_ReturnsResultWhenProviderSucceeds(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	provider := NewMockprovider(ctrl)

	// fail once
	provider.EXPECT().getCommitteeCertificates(gomock.Any(), gomock.Any()).
		Return(nil, fmt.Errorf("some error")).Times(1)
	// then succeed
	provider.EXPECT().getCommitteeCertificates(gomock.Any(), gomock.Any()).
		Return([]cert.CommitteeCertificate{{}}, nil).Times(1)

	certs, err := retry(3, 10*time.Second, func() (any, error) {
		return provider.getCommitteeCertificates(scc.Period(1), uint64(1))
	})

	require.NoError(err)
	require.NotNil(certs)
}

func TestRetry_GetCertificates_PropagatesError(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	provider := NewMockprovider(ctrl)

	retries := uint(3)

	provider.EXPECT().getCommitteeCertificates(gomock.Any(), gomock.Any()).
		Return(nil, fmt.Errorf("provider failed")).Times(int(retries + 1))

	retryProvider := newRetry(provider, retries, time.Duration(0))
	ccerts, err := retryProvider.getCommitteeCertificates(scc.Period(1), uint64(1))
	require.Error(err)
	require.Nil(ccerts)

	provider.EXPECT().getBlockCertificates(gomock.Any(), gomock.Any()).
		Return(nil, fmt.Errorf("provider failed")).Times(int(retries) + 1)
	bcerts, err := retryProvider.getBlockCertificates(idx.Block(1), uint64(1))
	require.Error(err)
	require.Nil(bcerts)
}

func TestRetry_GetCertificates_ReceivesCertificates(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	provider := NewMockprovider(ctrl)

	committeeCert := []cert.CommitteeCertificate{
		cert.NewCertificate(cert.NewCommitteeStatement(1, 1, scc.Committee{})),
	}
	provider.EXPECT().getCommitteeCertificates(gomock.Any(), gomock.Any()).
		Return(committeeCert, nil).Times(1)

	retryProvider := newRetry(provider, 1, time.Duration(0))
	ccerts, err := retryProvider.getCommitteeCertificates(scc.Period(1), uint64(1))
	require.NoError(err)
	require.Equal(committeeCert, ccerts)

	blockCert := []cert.BlockCertificate{
		cert.NewCertificate(cert.NewBlockStatement(1, 1, common.Hash{}, common.Hash{})),
	}
	provider.EXPECT().getBlockCertificates(gomock.Any(), gomock.Any()).
		Return(blockCert, nil).Times(1)

	bcerts, err := retryProvider.getBlockCertificates(idx.Block(1), uint64(1))
	require.NoError(err)
	require.Equal(blockCert, bcerts)
}
