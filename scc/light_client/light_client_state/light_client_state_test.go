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
				Return(nil, fmt.Errorf("Failed to get block certificates"))
		},
		"GettingCommitteeCertificates": func(prov *provider.MockProvider) {
			blockCert := cert.NewCertificate(cert.BlockStatement{
				Number: 1234,
			})
			prov.EXPECT().
				GetBlockCertificates(provider.LatestBlock, uint64(1)).
				Return([]cert.BlockCertificate{blockCert}, nil)
			prov.EXPECT().
				GetCommitteeCertificates(scc.Period(1), uint64(1)).
				Return(nil, fmt.Errorf("Failed to get committee certificates"))
		},
	}

	for name, expectedCalls := range tests {
		t.Run(name, func(t *testing.T) {
			prov := provider.NewMockProvider(gomock.NewController(t))
			state := NewState(scc.Committee{})
			expectedCalls(prov)
			_, err := state.Sync(prov)
			require.Error(err)
		})
	}
}

func TestLightClientState_Sync_ChangesNothingWhenLatestBlockIsEmpty(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	prov := provider.NewMockProvider(ctrl)

	state := NewState(scc.Committee{})
	prov.EXPECT().
		GetBlockCertificates(provider.LatestBlock, uint64(1)).
		Return([]cert.BlockCertificate{}, nil)

	block, err := state.Sync(prov)
	require.NoError(err)
	want := NewState(scc.Committee{})
	require.Equal(idx.Block(0), block)
	compareStates(t, want, state)
}

func TestLightClientState_Sync_ChangesNothingWhenUpdatingToCurrentPeriod(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	prov := provider.NewMockProvider(ctrl)

	// setup state, block and committee for block verification
	blockNumber := idx.Block(scc.BLOCKS_PER_PERIOD*1 + 1)
	key := bls.NewPrivateKey()
	member := makeMember(key)
	blockCert := cert.NewCertificate(
		cert.NewBlockStatement(0, blockNumber, common.Hash{}, common.Hash{}))
	blockCert.Add(scc.MemberId(0), cert.Sign(blockCert.Subject(), key))

	state := NewState(scc.NewCommittee(member))
	state.period = 1

	prov.EXPECT().
		GetBlockCertificates(provider.LatestBlock, uint64(1)).
		Return([]cert.BlockCertificate{blockCert}, nil)

	block, err := state.Sync(prov)
	require.NoError(err)
	require.Equal(idx.Block(blockNumber), block)
	want := testState(scc.Period(1), blockNumber, scc.NewCommittee(member))
	compareStates(t, want, state)
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

func testState(period scc.Period, blockNumber idx.Block, committee scc.Committee) *State {
	return &State{
		committee:  committee,
		period:     period,
		headNumber: blockNumber,
	}
}
