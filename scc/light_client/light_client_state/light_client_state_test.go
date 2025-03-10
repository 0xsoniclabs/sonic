package lc_state

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/bls"
	"github.com/0xsoniclabs/sonic/scc/cert"
	"github.com/0xsoniclabs/sonic/scc/light_client/provider"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestLightClientState_PropagatesErrorsFrom(t *testing.T) {
	require := require.New(t)

	tests := map[string]func(prov *provider.MockProvider){
		"GettingFirstBlockCertificate": func(prov *provider.MockProvider) {
			prov.EXPECT().
				GetBlockCertificates(provider.LatestBlock, uint64(1)).
				Return(nil, fmt.Errorf("failed to get block certificates"))
		},
		"GettingCommitteeCertificates": func(prov *provider.MockProvider) {
			expectBlockForPeriod(prov, 1)
			prov.EXPECT().
				GetCommitteeCertificates(scc.Period(1), uint64(1)).
				Return(nil, fmt.Errorf("failed to get committee certificates"))
		},
	}

	for name, expectedCalls := range tests {
		t.Run(name, func(t *testing.T) {
			prov := provider.NewMockProvider(gomock.NewController(t))
			state := NewState(scc.Committee{})
			expectedCalls(prov)
			_, err := state.Sync(prov)
			require.ErrorContains(err, "failed to get")
		})
	}
}

func TestLightClientState_Sync_ChangesNothingWhen_LatestBlockIsEmpty(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	prov := provider.NewMockProvider(ctrl)

	prov.EXPECT().
		GetBlockCertificates(provider.LatestBlock, uint64(1)).
		Return([]cert.BlockCertificate{}, nil)

	state := NewState(scc.Committee{})
	_, err := state.Sync(prov)
	want := State{}
	compareStates(t, &want, state)
	require.ErrorContains(err, "zero block certificates")
}

func TestLightClientState_Sync_UpdatesOnlyHeadWhen_SyncToCurrentPeriod(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	prov := provider.NewMockProvider(ctrl)

	// setup state with non empty committee
	key := bls.NewPrivateKey()
	member := makeMember(key)
	state := NewState(scc.NewCommittee(member))
	state.period = 1

	// setup block for period 1.
	blockNumber := idx.Block(scc.BLOCKS_PER_PERIOD*1 + 1)
	blockCert := cert.NewCertificate(
		cert.NewBlockStatement(0, blockNumber, common.Hash{0x1}, common.Hash{}))

	// sigh the block certificate with the committee member
	err := blockCert.Add(scc.MemberId(0), cert.Sign(blockCert.Subject(), key))
	require.NoError(err)

	prov.EXPECT().
		GetBlockCertificates(provider.LatestBlock, uint64(1)).
		Return([]cert.BlockCertificate{blockCert}, nil)

	_, err = state.Sync(prov)
	require.NoError(err)
	want := State{
		period:     scc.Period(1),
		committee:  scc.NewCommittee(member),
		headNumber: blockCert.Subject().Number,
		headHash:   common.Hash{0x1},
	}
	compareStates(t, &want, state)
}

func TestLightClientState_Sync_CanNotSyncToPastPeriod(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	prov := provider.NewMockProvider(ctrl)

	// setup block for period 1.
	expectBlockForPeriod(prov, 1)

	// setup synced to period 2
	state := State{period: 2}
	_, err := state.Sync(prov)
	require.ErrorContains(err, "cannot sync to a previous period")
}

func TestLightClientState_Sync_ReportsFailedVerificationOfLatestBlock(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	prov := provider.NewMockProvider(ctrl)

	// setup state with non empty committee
	key := bls.NewPrivateKey()
	member := makeMember(key)
	state := NewState(scc.NewCommittee(member))
	state.period = 1

	// setup unsigned block for period 1.
	expectBlockForPeriod(prov, 1)

	_, err := state.Sync(prov)
	require.ErrorContains(err, "insufficient voting power")
}

func TestLightClientState_Sync_FailsWithNilProvider(t *testing.T) {
	require := require.New(t)
	state := NewState(scc.Committee{})
	_, err := state.Sync(nil)
	require.ErrorContains(err, "cannot update with nil provider")
}

func TestLightClientState_Sync_FailsWhenHeadProvidedIsSmallerThanCurrent(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	prov := provider.NewMockProvider(ctrl)

	expectBlockForPeriod(prov, 0)

	state := NewState(scc.NewCommittee())
	lastBlockOfPeriod := idx.Block(3)
	state.headNumber = lastBlockOfPeriod

	_, err := state.Sync(prov)
	require.ErrorContains(err, "invalid block number")
	want := State{
		headNumber: lastBlockOfPeriod,
	}
	compareStates(t, &want, state)
}

func TestLightClientState_Sync_FailsWithUnorderedCommitteeCertificates(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	prov := provider.NewMockProvider(ctrl)

	// setup block.
	expectBlockForPeriod(prov, 2)

	// setup committee certificates for period 2.
	committeeCert2 := cert.NewCertificate(cert.CommitteeStatement{
		Period: 2,
	})

	// return list of certificates missing certificate for period 1
	prov.EXPECT().
		GetCommitteeCertificates(scc.Period(1), uint64(2)).
		Return([]cert.CommitteeCertificate{committeeCert2}, nil)

	state := NewState(scc.Committee{})
	_, err := state.Sync(prov)
	require.ErrorContains(err, "unexpected committee certificate period")
}

func TestLightClientState_Sync_FailsWithInvalidCommitteeCertificate(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	prov := provider.NewMockProvider(ctrl)

	// setup block
	expectBlockForPeriod(prov, 1)

	// setup committee certificates for period 1.
	committeeCert1 := cert.NewCertificate(cert.CommitteeStatement{
		Period: 1,
	})

	// return certificate without a valid committee
	prov.EXPECT().
		GetCommitteeCertificates(scc.Period(1), uint64(1)).
		Return([]cert.CommitteeCertificate{committeeCert1}, nil)

	state := NewState(scc.Committee{})
	_, err := state.Sync(prov)
	require.ErrorContains(err, "invalid committee")
}

func TestLightClientState_Sync_ReportsCurrentCommitteeFailsToVerifyNextCommittee(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	prov := provider.NewMockProvider(ctrl)

	// setup block for period 1.
	expectBlockForPeriod(prov, 1)

	// setup committee certificate for period 1.
	key := bls.NewPrivateKey()
	member := makeMember(key)
	committeeCert1 := cert.NewCertificate(cert.CommitteeStatement{
		Period:    1,
		Committee: scc.NewCommittee(member),
	})

	// return certificate for period 1 that has not been sign
	prov.EXPECT().
		GetCommitteeCertificates(scc.Period(1), uint64(1)).
		Return([]cert.CommitteeCertificate{committeeCert1}, nil)

	state := NewState(scc.NewCommittee(member))
	_, err := state.Sync(prov)
	require.ErrorContains(err, "committee certificate verification")
}

func TestLightClientState_Sync_UpdatesState(t *testing.T) {
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
	state := NewState(scc.NewCommittee(member))
	_, err = state.Sync(prov)
	require.NoError(err)

	want := State{
		period:     scc.Period(1),
		committee:  scc.NewCommittee(member),
		headNumber: blockNumber,
		headHash:   common.Hash{0x1},
	}
	compareStates(t, &want, state)

}

// /////////////////////////
// Helper functions
// /////////////////////////

func compareStates(t *testing.T, expected, actual *State) {
	require := require.New(t)
	require.Equal(expected.Head(), actual.Head())
	require.Equal(expected.period, actual.period)
	require.Equal(expected.headHash, actual.headHash)
	require.True(reflect.DeepEqual(expected.committee, actual.committee))
}

func expectBlockForPeriod(prov *provider.MockProvider, period scc.Period) cert.BlockCertificate {
	blockNumber := idx.Block(scc.BLOCKS_PER_PERIOD*period + 1)
	blockCert := cert.NewCertificate(
		cert.NewBlockStatement(0, blockNumber, common.Hash{}, common.Hash{}))

	prov.EXPECT().
		GetBlockCertificates(provider.LatestBlock, uint64(1)).
		Return([]cert.BlockCertificate{blockCert}, nil)

	return blockCert
}
