package light_client

import (
	"fmt"
	"testing"

	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/bls"
	"github.com/0xsoniclabs/sonic/scc/cert"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestLightClientState_PropagatesErrorsFrom(t *testing.T) {
	require := require.New(t)

	tests := map[string]func(prov *Mockprovider){
		"GettingFirstBlockCertificate": func(prov *Mockprovider) {
			prov.EXPECT().
				getBlockCertificates(LatestBlock, uint64(1)).
				Return(nil, fmt.Errorf("failed to get block certificates"))
		},
		"GettingCommitteeCertificates": func(prov *Mockprovider) {
			expectQueryForBlockOfPeriod(prov, 1)
			prov.EXPECT().
				getCommitteeCertificates(scc.Period(1), uint64(1)).
				Return(nil, fmt.Errorf("failed to get committee certificates"))
		},
	}

	for name, expectedCalls := range tests {
		t.Run(name, func(t *testing.T) {
			prov := NewMockprovider(gomock.NewController(t))
			s := newState(scc.Committee{})
			expectedCalls(prov)
			_, err := s.sync(prov)
			require.ErrorContains(err, "failed to get")
		})
	}
}

func TestLightClientState_Head_StateRoot_ReturnsFalseForUnsyncedState(t *testing.T) {
	require := require.New(t)
	s := newState(scc.Committee{})
	head, synced := s.head()
	require.False(synced)
	require.Zero(head)
	root, synced := s.stateRoot()
	require.False(synced)
	require.Zero(root)
}

func TestLightClientState_Sync_ChangesNothingWhen_LatestBlockCanNotBeObtained(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	prov := NewMockprovider(ctrl)

	prov.EXPECT().
		getBlockCertificates(LatestBlock, uint64(1)).
		Return([]cert.BlockCertificate{}, nil)

	s := newState(scc.Committee{})
	_, err := s.sync(prov)
	require.ErrorContains(err, "zero block certificates")
	want := state{}
	require.Equal(&want, s)
}

func TestLightClientState_Sync_UpdatesOnlyHeadWhen_SyncToCurrentPeriod(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	prov := NewMockprovider(ctrl)

	// setup state with non empty committee
	key := bls.NewPrivateKey()
	member := makeMember(key)
	s := newState(scc.NewCommittee(member))
	s.period = 1

	// setup block for period 1.
	blockNumber := idx.Block(scc.BLOCKS_PER_PERIOD*1 + 1)
	blockCert := cert.NewCertificate(
		cert.NewBlockStatement(0, blockNumber, common.Hash{0x1}, common.Hash{0x2}))

	// sigh the block certificate with the committee member
	err := blockCert.Add(scc.MemberId(0), cert.Sign(blockCert.Subject(), key))
	require.NoError(err)

	prov.EXPECT().
		getBlockCertificates(LatestBlock, uint64(1)).
		Return([]cert.BlockCertificate{blockCert}, nil)

	_, err = s.sync(prov)
	require.NoError(err)
	want := state{
		period:     scc.Period(1),
		committee:  scc.NewCommittee(member),
		headNumber: blockCert.Subject().Number,
		headHash:   common.Hash{0x1},
		headRoot:   common.Hash{0x2},
		hasSynced:  true,
	}
	require.Equal(&want, s)
}

func TestLightClientState_Sync_CanNotSyncToPastPeriod(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	prov := NewMockprovider(ctrl)

	// setup block for period 1.
	expectQueryForBlockOfPeriod(prov, 1)

	// setup synced to period 2
	s := state{period: 2}
	_, err := s.sync(prov)
	require.ErrorContains(err, "cannot sync to a previous period")
}

func TestLightClientState_Sync_ReportsFailedVerificationOfLatestBlock(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	prov := NewMockprovider(ctrl)

	// setup state with non empty committee
	key := bls.NewPrivateKey()
	member := makeMember(key)
	s := newState(scc.NewCommittee(member))
	s.period = 1

	// setup unsigned block for period 1.
	expectQueryForBlockOfPeriod(prov, 1)

	_, err := s.sync(prov)
	require.ErrorContains(err, "failed to authenticate block certificate")
}

func TestLightClientState_Sync_FailsWithNilProvider(t *testing.T) {
	require := require.New(t)
	s := newState(scc.Committee{})
	_, err := s.sync(nil)
	require.ErrorContains(err, "cannot update with nil provider")
}

func TestLightClientState_Sync_IgnoresSameBlockOrPeriod(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	prov := NewMockprovider(ctrl)

	expectQueryForBlockOfPeriod(prov, 0)

	s := newState(scc.NewCommittee())
	lastBlockSyncedTo := idx.Block(1)
	s.headNumber = lastBlockSyncedTo

	_, err := s.sync(prov)
	require.NoError(err)
	want := state{
		headNumber: lastBlockSyncedTo,
	}
	require.Equal(&want, s)
}

func TestLightClientState_Sync_FailsWithOlderHead(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	prov := NewMockprovider(ctrl)

	expectQueryForBlockOfPeriod(prov, 0)

	s := newState(scc.NewCommittee())
	lastBlockSyncedTo := idx.Block(3)
	s.headNumber = lastBlockSyncedTo

	_, err := s.sync(prov)
	require.ErrorContains(err, "provider returned old block head")
	want := state{
		headNumber: lastBlockSyncedTo,
	}
	require.Equal(&want, s)
}

func TestLightClientState_Sync_FailsWithUnorderedCommitteeCertificates(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	prov := NewMockprovider(ctrl)

	// setup block.
	expectQueryForBlockOfPeriod(prov, 2)

	// setup committee certificates for period 2.
	committeeCert2 := cert.NewCertificate(cert.CommitteeStatement{
		Period: 2,
	})

	// return list of certificates missing certificate for period 1
	prov.EXPECT().
		getCommitteeCertificates(scc.Period(1), uint64(2)).
		Return([]cert.CommitteeCertificate{committeeCert2}, nil)

	s := newState(scc.Committee{})
	_, err := s.sync(prov)
	require.ErrorContains(err, "unexpected committee certificate period")
}

func TestLightClientState_Sync_FailsWithInvalidCommitteeCertificate(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	prov := NewMockprovider(ctrl)

	// setup block
	expectQueryForBlockOfPeriod(prov, 1)

	// setup committee certificates for period 1.
	committeeCert1 := cert.NewCertificate(cert.CommitteeStatement{
		Period: 1,
	})

	// return certificate without a valid committee
	prov.EXPECT().
		getCommitteeCertificates(scc.Period(1), uint64(1)).
		Return([]cert.CommitteeCertificate{committeeCert1}, nil)

	s := newState(scc.Committee{})
	_, err := s.sync(prov)
	require.ErrorContains(err, "invalid committee")
}

func TestLightClientState_Sync_ReportsIfCurrentCommitteeFailsToVerifyNextCommittee(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	prov := NewMockprovider(ctrl)

	// setup block for period 1.
	expectQueryForBlockOfPeriod(prov, 1)

	// setup committee certificate for period 1.
	key := bls.NewPrivateKey()
	member := makeMember(key)
	committeeCert1 := cert.NewCertificate(cert.CommitteeStatement{
		Period:    1,
		Committee: scc.NewCommittee(member),
	})

	// return certificate for period 1 that has not been sign
	prov.EXPECT().
		getCommitteeCertificates(scc.Period(1), uint64(1)).
		Return([]cert.CommitteeCertificate{committeeCert1}, nil)

	s := newState(scc.NewCommittee(member))
	_, err := s.sync(prov)
	require.ErrorContains(err, "committee certificate verification")
}

func TestLightClientState_Sync_UpdatesState(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	prov := NewMockprovider(ctrl)

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
		getBlockCertificates(LatestBlock, uint64(1)).
		Return([]cert.BlockCertificate{blockCert}, nil)
	prov.EXPECT().
		getCommitteeCertificates(scc.Period(1), uint64(1)).
		Return([]cert.CommitteeCertificate{committeeCert1}, nil)

	// sync
	s := newState(scc.NewCommittee(member))
	_, err = s.sync(prov)
	require.NoError(err)

	want := state{
		period:     scc.Period(1),
		committee:  scc.NewCommittee(member),
		headNumber: blockNumber,
		headHash:   common.Hash{0x1},
		headRoot:   common.Hash{0x2},
		hasSynced:  true,
	}
	require.Equal(&want, s)

}

// /////////////////////////
// Helper functions
// /////////////////////////

func expectQueryForBlockOfPeriod(prov *Mockprovider, period scc.Period) {
	blockNumber := idx.Block(scc.BLOCKS_PER_PERIOD*period + 1)
	blockCert := cert.NewCertificate(
		cert.NewBlockStatement(0, blockNumber, common.Hash{}, common.Hash{}))

	prov.EXPECT().
		getBlockCertificates(LatestBlock, uint64(1)).
		Return([]cert.BlockCertificate{blockCert}, nil)
}
