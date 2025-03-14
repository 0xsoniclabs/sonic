package provider

import (
	"fmt"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/scc"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestRetry_NewRetry_CanInitializeFromProvider(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)

	provider := NewMockProvider(ctrl)
	provider.EXPECT().Close().AnyTimes()
	retry := NewRetry(provider, 3, time.Duration(0))
	require.NotNil(retry)
}

func TestRetry_Close_ClosesProvider(t *testing.T) {
	ctrl := gomock.NewController(t)
	provider := NewMockProvider(ctrl)
	provider.EXPECT().Close()

	retry := NewRetry(provider, 3, time.Duration(0))
	retry.Close()
}

func TestRetry_GetCommitteeCertificates_RetriesWhenProviderFails(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)

	maxRetries := uint(3)

	provider := NewMockProvider(ctrl)
	provider.EXPECT().GetCommitteeCertificates(gomock.Any(), gomock.Any()).
		Return(nil, fmt.Errorf("provider failed")).Times(int(maxRetries))

	retry := NewRetry(provider, maxRetries, time.Duration(0))
	certs, err := retry.GetCommitteeCertificates(scc.Period(1), uint64(1))
	require.Error(err)
	require.Nil(certs)
}

func TestRetry_GetCommitteeCertificates_WaitsBetweenRetries(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	provider := NewMockProvider(ctrl)

	maxRetries := uint(3)
	delay := 200 * time.Millisecond

	provider.EXPECT().GetCommitteeCertificates(gomock.Any(), gomock.Any()).
		Return(nil, fmt.Errorf("provider failed")).Times(int(maxRetries))

	retryProvider := NewRetry(provider, maxRetries, delay)

	start := time.Now()
	_, _ = retryProvider.GetCommitteeCertificates(scc.Period(1), uint64(1))
	duration := time.Since(start)

	expected := time.Duration(maxRetries) * delay
	require.GreaterOrEqual(duration.Milliseconds(), expected.Milliseconds())
}
