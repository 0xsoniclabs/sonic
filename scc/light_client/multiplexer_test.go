package light_client

import (
	"fmt"
	"testing"

	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/cert"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestMultiplexer_newMultiplexer_ReturnsErrorIfNoProviders(t *testing.T) {
	_, err := newMultiplexer()
	require.Error(t, err)
}

func TestMultiplexer_Close_closesAllProviders(t *testing.T) {
	ctrl := gomock.NewController(t)

	prov1 := NewMockprovider(ctrl)
	prov1.EXPECT().close().Times(1)

	prov2 := NewMockprovider(ctrl)
	prov2.EXPECT().close().Times(1)

	m, err := newMultiplexer(prov1, prov2)
	require.NoError(t, err)
	m.close()
}

func TestMultiplexer_tryAllProviders_TriesAllProvidersOnFails(t *testing.T) {
	ctrl := gomock.NewController(t)

	p1 := NewMockprovider(ctrl)
	p1.EXPECT().getBlockCertificates(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("error")).Times(1)

	p2 := NewMockprovider(ctrl)
	p2.EXPECT().getBlockCertificates(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("error")).Times(1)

	m, err := newMultiplexer(p1, p2)
	require.NoError(t, err)
	_, err = tryAll(m.providers, func(p provider) ([]cert.BlockCertificate, error) {
		return p.getBlockCertificates(idx.Block(0), uint64(1))
	})
	require.ErrorContains(t, err, "all providers failed")
}

func TestMultiplexer_tryAllProviders_ReturnsFirstSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)

	p1 := NewMockprovider(ctrl)
	p1.EXPECT().getBlockCertificates(gomock.Any(), gomock.Any()).Return([]cert.BlockCertificate{}, nil).Times(1)

	p2 := NewMockprovider(ctrl)

	m, err := newMultiplexer(p1, p2)
	require.NoError(t, err)
	_, err = tryAll(m.providers, func(p provider) ([]cert.BlockCertificate, error) {
		return p.getBlockCertificates(idx.Block(0), uint64(1))
	})
	require.NoError(t, err)
}

func TestMultiplexer_GetCertificates_PropagatesError(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	p1 := NewMockprovider(ctrl)
	p2 := NewMockprovider(ctrl)

	m, err := newMultiplexer(p1, p2)
	require.NoError(err)

	// fail to get block certificates
	p1.EXPECT().getBlockCertificates(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("error1")).Times(1)
	p2.EXPECT().getBlockCertificates(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("error2")).Times(1)

	_, err = m.getBlockCertificates(idx.Block(0), uint64(1))
	require.ErrorContains(err, "all providers failed")
	require.ErrorContains(err, "error1")
	require.ErrorContains(err, "error2")

	// fail to get committee certificates
	p1.EXPECT().getCommitteeCertificates(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("error3")).Times(1)
	p2.EXPECT().getCommitteeCertificates(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("error4")).Times(1)

	_, err = m.getCommitteeCertificates(scc.Period(0), uint64(1))
	require.ErrorContains(err, "all providers failed")
	require.ErrorContains(err, "error3")
	require.ErrorContains(err, "error4")

	p1.EXPECT().getAccountProof(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("error5")).Times(1)
	p2.EXPECT().getAccountProof(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("error6")).Times(1)

	_, err = m.getAccountProof(common.Address{0x01}, idx.Block(0))
	require.ErrorContains(err, "all providers failed")
	require.ErrorContains(err, "error5")
	require.ErrorContains(err, "error6")
}
