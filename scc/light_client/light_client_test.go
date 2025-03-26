package light_client

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/bls"
	"github.com/0xsoniclabs/sonic/scc/cert"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
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
			Url:     []*url.URL{},
			Genesis: scc.NewCommittee(member),
		},
		"invalidUrl": {
			Url:     []*url.URL{{Host: "not-a-url"}},
			Genesis: scc.NewCommittee(member),
		},
		"emptyGenesisCommittee": {
			Url:     []*url.URL{{Scheme: "http", Host: "localhost:4242"}},
			Genesis: scc.NewCommittee(),
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
	require.NotNil(c.state)
	require.NotNil(c.provider)
}

func TestLightClient_Close_ClosesProvider(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	prov := NewMockprovider(ctrl)

	prov.EXPECT().close().Times(1)

	c, err := NewLightClient(testConfig())
	require.NoError(err)
	c.provider = prov
	c.Close()
}

func TestLightClient_Sync_ReturnsErrorOnProviderFailure(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	prov := NewMockprovider(ctrl)

	c, err := NewLightClient(testConfig())
	require.NoError(err)
	errStr := "failed to get block certificates"
	prov.EXPECT().getBlockCertificates(LatestBlock, uint64(1)).
		Return(nil, fmt.Errorf("%v", errStr))
	c.provider = prov
	_, err = c.Sync()
	require.ErrorContains(err, errStr)
}

func TestLightClient_Sync_ReturnsErrorOnStateSyncFailure(t *testing.T) {
	require := require.New(t)

	// setup mock provider
	ctrl := gomock.NewController(t)
	prov := NewMockprovider(ctrl)

	// setup block certificate
	blockNumber := idx.Block(scc.BLOCKS_PER_PERIOD*1 + 42)
	blockCert := cert.NewCertificate(
		cert.NewBlockStatement(0, blockNumber, common.Hash{0x1}, common.Hash{}))
	// expect to return head
	prov.EXPECT().getBlockCertificates(LatestBlock, uint64(1)).
		Return([]cert.BlockCertificate{blockCert}, nil)

	// setup committee certificate
	committeeCert := cert.NewCertificate(cert.CommitteeStatement{Period: 1})
	// expect to return committee certificates that is not signed by genesis
	prov.EXPECT().
		getCommitteeCertificates(scc.Period(1), gomock.Any()).
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
	prov := NewMockprovider(ctrl)

	// Setup test data
	key := bls.NewPrivateKey()
	blockCert, blockNumber := setupBlockCertificate(t, key)
	committeeCert := setupCommitteeCertificate(t, key)

	// Mock provider calls
	mockProviderResponses(prov, blockCert, committeeCert)

	// Create and configure LightClient
	client, err := setupLightClient(prov, key)
	require.NoError(err)

	// Perform sync
	head, err := client.Sync()
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
	// error is ignored because constant string is a url
	u, _ := url.Parse("http://localhost:4242")
	return Config{
		Url:     []*url.URL{u},
		Genesis: scc.NewCommittee(makeMember(key)),
	}
}

// setupBlockCertificate creates a block certificate for the second block of
// period 1 and signs it with the given key.
// Returns the block certificate and the block number.
func setupBlockCertificate(t *testing.T, key bls.PrivateKey) (cert.BlockCertificate, idx.Block) {
	blockNumber := idx.Block(scc.BLOCKS_PER_PERIOD*1 + 1)
	blockCert := cert.NewCertificate(
		cert.NewBlockStatement(0, blockNumber, common.Hash{0x1}, common.Hash{0x2}),
	)

	// Sign certificate
	err := blockCert.Add(scc.MemberId(0), cert.Sign(blockCert.Subject(), key))
	require.NoError(t, err)

	return blockCert, blockNumber
}

// setupCommitteeCertificate creates a committee certificate for period 1 and
// signs it with the given key.
// Returns the committee certificate.
func setupCommitteeCertificate(t *testing.T, key bls.PrivateKey) cert.CommitteeCertificate {
	member := makeMember(key)
	committeeCert := cert.NewCertificate(cert.CommitteeStatement{
		Period:    1,
		Committee: scc.NewCommittee(member),
	})

	// Sign certificate
	err := committeeCert.Add(scc.MemberId(0), cert.Sign(committeeCert.Subject(), key))
	require.NoError(t, err)

	return committeeCert
}

// mockProviderResponses mocks the provider responses for block and committee certificates
func mockProviderResponses(prov *Mockprovider, blockCert cert.BlockCertificate, committeeCert cert.CommitteeCertificate) {
	prov.EXPECT().
		getBlockCertificates(LatestBlock, uint64(1)).
		Return([]cert.BlockCertificate{blockCert}, nil)

	prov.EXPECT().
		getCommitteeCertificates(scc.Period(1), uint64(1)).
		Return([]cert.CommitteeCertificate{committeeCert}, nil)
}

// setupLightClient creates a LightClient with a committee member based on
// the given key and a used the given provider for the client.
func setupLightClient(prov *Mockprovider, key bls.PrivateKey) (*LightClient, error) {
	u, _ := url.Parse("http://localhost:4242")
	config := Config{
		Url:     []*url.URL{u},
		Genesis: scc.NewCommittee(makeMember(key)),
	}
	client, err := NewLightClient(config)
	if err != nil {
		return nil, err
	}

	client.provider = prov
	return client, nil
}
