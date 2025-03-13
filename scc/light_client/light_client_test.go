package light_client

import (
	"fmt"
	"testing"

	"github.com/0xsoniclabs/consensus/inter/idx"
	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/bls"
	"github.com/0xsoniclabs/sonic/scc/cert"
	"github.com/0xsoniclabs/sonic/scc/light_client/provider"
	"github.com/ethereum/go-ethereum/common"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestLightClient_NewLightClient_ReportsInvalidConfig(t *testing.T) {
	require := require.New(t)
	key := bls.NewPrivateKey()
	member := scc.Member{
		PublicKey:         key.PublicKey(),
		ProofOfPossession: key.GetProofOfPossession(),
		VotingPower:       1,
	}

	tests := map[string]Config{
		"emptyStringProvider": {
			Provider: "",
			Genesis:  scc.NewCommittee(member),
		},
		"invalidProviderURL": {
			Provider: "not-a-url",
			Genesis:  scc.NewCommittee(member),
		},
		"emptyGenesisCommittee": {
			Provider: "http://localhost:4242",
			Genesis:  scc.NewCommittee(),
		},
	}

	for name, config := range tests {
		t.Run(name, func(t *testing.T) {
			c, err := NewLightClient(config)
			require.Error(err)
			require.Nil(c)
		})
	}
}

func TestLightClient_NewLightClient_CreatesLightClientFromValidConfig(t *testing.T) {
	require := require.New(t)
	c, err := NewLightClient(testConfig())
	require.NoError(err)
	require.NotNil(c)
}

func TestLightClient_Close_ClosesProvider(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	prov := provider.NewMockProvider(ctrl)
	prov.EXPECT().Close().Times(1)
	prov.EXPECT().GetBlockCertificates(gomock.Any(), gomock.Any()).
		Return(nil, fmt.Errorf("provider is closed"))
	c, err := NewLightClient(testConfig())
	require.NoError(err)
	c.provider = prov
	c.Close()
	_, err = c.Sync()
	require.ErrorContains(err, "provider is closed")
}

func TestLightClient_Sync_InitializesState(t *testing.T) {
	require := require.New(t)
	c, err := NewLightClient(testConfig())
	require.NoError(err)
	require.NotNil(c.state)
}

func TestLightClient_Sync_ReturnsErrorOnProviderFailure(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	prov := provider.NewMockProvider(ctrl)

	c, err := NewLightClient(testConfig())
	require.NoError(err)
	errStr := "failed to get block certificates"
	prov.EXPECT().GetBlockCertificates(provider.LatestBlock, uint64(1)).
		Return(nil, fmt.Errorf("%v", errStr))
	c.provider = prov
	_, err = c.Sync()
	require.ErrorContains(err, errStr)
}

func TestLightClient_Sync_ReturnsErrorOnStateSyncFailure(t *testing.T) {
	require := require.New(t)

	// setup mock provider
	ctrl := gomock.NewController(t)
	prov := provider.NewMockProvider(ctrl)

	// setup block certificate
	blockNumber := idx.Block(scc.BLOCKS_PER_PERIOD*1 + 42)
	blockCert := cert.NewCertificate(
		cert.NewBlockStatement(0, blockNumber, common.Hash{0x1}, common.Hash{}))
	// expect to return head
	prov.EXPECT().GetBlockCertificates(provider.LatestBlock, uint64(1)).
		Return([]cert.BlockCertificate{blockCert}, nil)

	// setup committee certificate
	committeeCert := cert.NewCertificate(cert.CommitteeStatement{Period: 1})
	// expect to return committee certificates that is not signed by genesis
	prov.EXPECT().
		GetCommitteeCertificates(scc.Period(1), gomock.Any()).
		Return([]cert.CommitteeCertificate{committeeCert}, nil)

	// create LightClient
	c, err := NewLightClient(testConfig())
	require.NoError(err)

	// set provider
	c.provider = prov

	// sync
	_, err = c.Sync()
	require.ErrorContains(err, "invalid committee")
}

func TestLightClientState_Sync_UpdatesStateToHead(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	prov := provider.NewMockProvider(ctrl)

	// setup block for period 1.
	blockNumber := idx.Block(scc.BLOCKS_PER_PERIOD*1 + 1)
	blockCert := cert.NewCertificate(
		cert.NewBlockStatement(0, blockNumber, common.Hash{0x1}, common.Hash{0x2}))

	// setup committee certificate for period 1.
	key := bls.NewPrivateKey()
	member := makeMember(key)
	committeeCert1 := cert.NewCertificate(cert.CommitteeStatement{
		Period:    1,
		Committee: scc.NewCommittee(member),
	})

	// member signs the certificates
	err := committeeCert1.Add(scc.MemberId(0), cert.Sign(committeeCert1.Subject(), key))
	require.NoError(err)
	err = blockCert.Add(scc.MemberId(0), cert.Sign(blockCert.Subject(), key))
	require.NoError(err)

	// provider calls
	prov.EXPECT().
		GetBlockCertificates(provider.LatestBlock, uint64(1)).
		Return([]cert.BlockCertificate{blockCert}, nil)
	prov.EXPECT().
		GetCommitteeCertificates(scc.Period(1), uint64(1)).
		Return([]cert.CommitteeCertificate{committeeCert1}, nil)

	// sync
	config := Config{
		Provider: "http://localhost:4242",
		Genesis:  scc.NewCommittee(member),
	}
	c, err := NewLightClient(config)
	require.NoError(err)
	c.provider = prov
	head, err := c.Sync()
	require.NoError(err)

	// check state
	require.Equal(blockNumber, head)
}

/////////////////////////////////////////////////////
// Helper functions for testing
/////////////////////////////////////////////////////

func makeMember(key bls.PrivateKey) scc.Member {
	return scc.Member{
		PublicKey:         key.PublicKey(),
		ProofOfPossession: key.GetProofOfPossession(),
		VotingPower:       1,
	}
}

func testConfig() Config {
	key := bls.NewPrivateKey()
	return Config{
		Provider: "http://localhost:4242",
		Genesis:  scc.NewCommittee(makeMember(key)),
	}
}
