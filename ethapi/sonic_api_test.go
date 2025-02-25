package ethapi_test

import (
	"context"
	"fmt"
	"math"
	"slices"
	"testing"

	"github.com/0xsoniclabs/sonic/ethapi"
	"github.com/0xsoniclabs/sonic/gossip"
	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/cert"
	"github.com/0xsoniclabs/sonic/utils/result"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var _ ethapi.SccApiBackend = (*gossip.Store)(nil)

func TestSonicApi_GetCommitteeCertificates_CanProduceCertificates(t *testing.T) {
	ctrl := gomock.NewController(t)
	backend := ethapi.NewMockSccApiBackend(ctrl)
	api := ethapi.NewPublicSccApi(backend)

	certificates := []cert.CommitteeCertificate{
		cert.NewCertificate(cert.CommitteeStatement{Period: 1}),
		cert.NewCertificate(cert.CommitteeStatement{Period: 2}),
		cert.NewCertificate(cert.CommitteeStatement{Period: 3}),
	}

	results := []result.T[cert.CommitteeCertificate]{}
	for _, c := range certificates {
		results = append(results, result.New(c))
	}

	backend.EXPECT().EnumerateCommitteeCertificates(scc.Period(1)).Return(slices.Values(results))

	_, err := api.GetCommitteeCertificates(context.Background(), 1, 10)
	require.NoError(t, err)
	// TODO: check the result
}

// TODO: add checks for
// - handling errors during iteration

func TestSonicApi_GetCommitteeCertificates_CanBeCancelled(t *testing.T) {
	ctrl := gomock.NewController(t)
	backend := ethapi.NewMockSccApiBackend(ctrl)
	api := ethapi.NewPublicSccApi(backend)

	certificates := []cert.CommitteeCertificate{
		cert.NewCertificate(cert.CommitteeStatement{Period: 1}),
		cert.NewCertificate(cert.CommitteeStatement{Period: 2}),
		cert.NewCertificate(cert.CommitteeStatement{Period: 3}),
	}

	results := []result.T[cert.CommitteeCertificate]{}
	for _, c := range certificates {
		results = append(results, result.New(c))
	}

	backend.EXPECT().EnumerateCommitteeCertificates(scc.Period(1)).Return(slices.Values(results))

	context, cancel := context.WithCancel(context.Background())
	cancel()
	res, err := api.GetCommitteeCertificates(context, 1, 10)
	require.ErrorIs(t, err, context.Err())
	require.Empty(t, res)
}

func TestSonicApi_GetCommitteeCertificates_RespectsUserLimit(t *testing.T) {
	ctrl := gomock.NewController(t)
	backend := ethapi.NewMockSccApiBackend(ctrl)
	api := ethapi.NewPublicSccApi(backend)

	certificates := []cert.CommitteeCertificate{
		cert.NewCertificate(cert.CommitteeStatement{Period: 1}),
		cert.NewCertificate(cert.CommitteeStatement{Period: 2}),
		cert.NewCertificate(cert.CommitteeStatement{Period: 3}),
	}

	results := []result.T[cert.CommitteeCertificate]{}
	for _, c := range certificates {
		results = append(results, result.New(c))
	}

	backend.EXPECT().EnumerateCommitteeCertificates(scc.Period(1)).
		Return(slices.Values(results)).AnyTimes()

	context := context.Background()
	for _, limit := range []ethapi.Number{0, 1, 2, 3, math.MaxInt64} {
		res, err := api.GetCommitteeCertificates(context, 1, limit)
		require.NoError(t, err)
		want := int(limit.Int64())
		if have := len(certificates); want > have {
			want = have
		}
		require.Len(t, res, int(want))
	}
}

func TestSonicApi_GetCommitteeCertificates_ReportsFetchErrors(t *testing.T) {
	ctrl := gomock.NewController(t)
	backend := ethapi.NewMockSccApiBackend(ctrl)
	api := ethapi.NewPublicSccApi(backend)

	injected := fmt.Errorf("injected error")
	results := []result.T[cert.CommitteeCertificate]{
		result.Error[cert.CommitteeCertificate](injected),
	}

	backend.EXPECT().EnumerateCommitteeCertificates(scc.Period(1)).Return(slices.Values(results))

	res, err := api.GetCommitteeCertificates(context.Background(), 1, 10)
	require.ErrorIs(t, err, injected)
	require.Empty(t, res)
}

func TestSonicApi_GetBlockCertificate_CanProduceBlockCertificates(t *testing.T) {
	ctrl := gomock.NewController(t)
	backend := ethapi.NewMockSccApiBackend(ctrl)
	api := ethapi.NewPublicSccApi(backend)

	certificates := []cert.BlockCertificate{
		cert.NewCertificate(cert.BlockStatement{Number: 1}),
		cert.NewCertificate(cert.BlockStatement{Number: 2}),
		cert.NewCertificate(cert.BlockStatement{Number: 3}),
	}

	results := []result.T[cert.BlockCertificate]{}
	for _, c := range certificates {
		results = append(results, result.New(c))
	}

	backend.EXPECT().EnumerateBlockCertificates(idx.Block(1)).Return(slices.Values(results))

	_, err := api.GetBlockCertificates(context.Background(), 1, 10)
	require.NoError(t, err)
}

func TestPeriodNumber_Unmarshaling_HandlesMultipleFormats(t *testing.T) {
	tests := map[string]scc.Period{
		"1":      1,
		"2":      2,
		"012":    012,
		"0x12":   0x12,
		"0b1010": 0b1010,
		"0xaBc":  0xabc,
		"latest": ethapi.HeadPeriod.Period(),
	}

	for input, expected := range tests {
		t.Run(input, func(t *testing.T) {
			var p ethapi.PeriodNumber
			err := p.UnmarshalJSON([]byte(`"` + input + `"`))
			require.NoError(t, err)
			require.Equal(t, expected, p.Period())
		})
	}
}

func TestPeriodNumber_Unmarshaling_FailsOnTooLargeNumber(t *testing.T) {
	var p ethapi.PeriodNumber
	err := p.UnmarshalJSON([]byte(fmt.Sprintf("%d", uint64(math.MaxInt64)+1)))
	require.Error(t, err)
}

func TestPeriodNumber_Unmarshaling_LatestResultsInHead(t *testing.T) {
	var p ethapi.PeriodNumber
	err := p.UnmarshalJSON([]byte("latest"))
	require.NoError(t, err)
	require.Equal(t, ethapi.HeadPeriod, p)
	require.True(t, p.IsHead())
}

func TestBlockNumber_Unmarshaling_HandlesMultipleFormats(t *testing.T) {
	tests := map[string]idx.Block{
		"1":      1,
		"2":      2,
		"012":    012,
		"0x12":   0x12,
		"0b1010": 0b1010,
		"0xaBc":  0xabc,
		"latest": ethapi.HeadBlock.Block(),
	}

	for input, expected := range tests {
		t.Run(input, func(t *testing.T) {
			var p ethapi.BlockNumber
			err := p.UnmarshalJSON([]byte(`"` + input + `"`))
			require.NoError(t, err)
			require.Equal(t, expected, p.Block())
		})
	}
}

func TestBlockNumber_Unmarshaling_FailsOnTooLargeNumber(t *testing.T) {
	var p ethapi.BlockNumber
	err := p.UnmarshalJSON([]byte(fmt.Sprintf("%d", uint64(math.MaxInt64)+1)))
	require.Error(t, err)
}

func TestBlockNumber_Unmarshaling_LatestResultsInHead(t *testing.T) {
	var p ethapi.BlockNumber
	err := p.UnmarshalJSON([]byte("latest"))
	require.NoError(t, err)
	require.Equal(t, ethapi.HeadBlock, p)
	require.True(t, p.IsHead())
}

func TestNumber_Unmarshaling_HandlesMultipleFormats(t *testing.T) {
	tests := map[string]int64{
		"1":      1,
		"2":      2,
		"012":    012,
		"0x12":   0x12,
		"0b1010": 0b1010,
		"0xaBc":  0xabc,
		"max":    math.MaxInt64,
	}

	for input, expected := range tests {
		t.Run(input, func(t *testing.T) {
			var p ethapi.Number
			err := p.UnmarshalJSON([]byte(`"` + input + `"`))
			require.NoError(t, err)
			require.Equal(t, expected, p.Int64())
		})
	}
}

func TestNumber_Unmarshaling_FailsOnTooLargeNumber(t *testing.T) {
	var p ethapi.Number
	err := p.UnmarshalJSON([]byte(fmt.Sprintf("%d", uint64(math.MaxInt64)+1)))
	require.Error(t, err)
}

func TestNumber_Unmarshaling_FailsOnNegativeValue(t *testing.T) {
	var p ethapi.Number
	err := p.UnmarshalJSON([]byte("-1"))
	require.Error(t, err)
}
