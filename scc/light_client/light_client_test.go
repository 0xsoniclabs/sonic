// Copyright 2025 Sonic Operations Ltd
// This file is part of the Sonic Client
//
// Sonic is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Sonic is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Sonic. If not, see <http://www.gnu.org/licenses/>.

package light_client

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/0xsoniclabs/carmen/go/carmen"
	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/bls"
	"github.com/0xsoniclabs/sonic/scc/cert"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
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
	u, _ := url.Parse("http://localhost:4242")
	config := Config{
		Url:     []*url.URL{u},
		Genesis: scc.NewCommittee(member),
	}
	c, err := NewLightClient(config)
	require.NoError(err)
	c.provider = prov
	head, err := c.Sync()
	require.NoError(err)

	// check state
	require.Equal(blockNumber, head)
}

func TestLightClient_getAccountProof_ReportsErrorsFrom(t *testing.T) {
	require := require.New(t)
	address := common.Address{0x01}

	tests := map[string]struct {
		mockProvider func(*Mockprovider)
		expectedErr  string
	}{
		"ProviderError": {
			mockProvider: func(prov *Mockprovider) {
				prov.EXPECT().getAccountProof(address, gomock.Any()).
					Return(nil, fmt.Errorf("some error"))
			},
			expectedErr: "failed to get account proof",
		},
		"NilProof": {
			mockProvider: func(prov *Mockprovider) {
				prov.EXPECT().getAccountProof(address, gomock.Any()).
					Return(nil, nil)
			},
			expectedErr: "nil account proof",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// setup
			ctrl := gomock.NewController(t)
			prov := NewMockprovider(ctrl)
			tt.mockProvider(prov)
			c, err := NewLightClient(testConfig())
			require.NoError(err)
			c.provider = prov
			c.state.hasSynced = true

			_, err = c.getAccountProof(address)
			require.ErrorContains(err, tt.expectedErr)
		})
	}
}

func TestLightClient_getAccountProof_ReturnsProof(t *testing.T) {
	require := require.New(t)
	// setup
	ctrl := gomock.NewController(t)
	prov := NewMockprovider(ctrl)
	c, err := NewLightClient(testConfig())
	require.NoError(err)
	c.provider = prov
	c.state.hasSynced = true

	want := carmen.CreateWitnessProofFromNodes()
	prov.EXPECT().getAccountProof(common.Address{0x01}, c.state.headNumber).
		Return(want, nil)

	got, err := c.getAccountProof(common.Address{0x01})
	require.NoError(err)
	require.Equal(want, got)
}

func TestLightClient_GetBalance_getAccountProof_ReturnsErrorIfNotSynced(t *testing.T) {
	require := require.New(t)
	c, err := NewLightClient(testConfig())
	require.NoError(err)
	_, err = c.GetBalance(common.Address{0x01})
	require.ErrorContains(err, "light client has not yet synced")
	_, err = c.getAccountProof(common.Address{0x01})
	require.ErrorContains(err, "light client has not yet synced")
}

func TestLightClient_GetBalance_PropagatesErrorFrom(t *testing.T) {
	tests := map[string]struct {
		mockExpect  func(*Mockprovider, *carmen.MockWitnessProof)
		expectedErr string
	}{
		"getAccountProof": {
			mockExpect: func(prov *Mockprovider, _ *carmen.MockWitnessProof) {
				prov.EXPECT().getAccountProof(gomock.Any(), gomock.Any()).
					Return(nil, fmt.Errorf("some error"))
			},
			expectedErr: "failed to get account proof",
		},
		"proofGetBalance": {
			mockExpect: func(prov *Mockprovider, proof *carmen.MockWitnessProof) {
				prov.EXPECT().getAccountProof(gomock.Any(), gomock.Any()).
					Return(proof, nil)
				proof.EXPECT().GetBalance(gomock.Any(), gomock.Any()).
					Return(carmen.NewAmountFromUint256(uint256.NewInt(0)), false, fmt.Errorf("some error"))
			},
			expectedErr: "failed to get balance from proof",
		},
		"balanceNotProven": {
			mockExpect: func(prov *Mockprovider, proof *carmen.MockWitnessProof) {
				prov.EXPECT().getAccountProof(gomock.Any(), gomock.Any()).
					Return(proof, nil)
				proof.EXPECT().GetBalance(gomock.Any(), gomock.Any()).
					Return(carmen.NewAmountFromUint256(uint256.NewInt(0)), false, nil)
			},
			expectedErr: "balance could not be proven",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)

			// setup
			ctrl := gomock.NewController(t)
			prov := NewMockprovider(ctrl)
			c, err := NewLightClient(testConfig())
			require.NoError(err)
			c.provider = prov
			c.state.hasSynced = true

			proof := carmen.NewMockWitnessProof(ctrl)

			tt.mockExpect(prov, proof)

			_, err = c.GetBalance(common.Address{0x01})
			require.ErrorContains(err, tt.expectedErr)
		})
	}
}

func TestLightClient_GetBalance_ReturnsBalance(t *testing.T) {
	require := require.New(t)
	// setup
	ctrl := gomock.NewController(t)
	prov := NewMockprovider(ctrl)
	c, err := NewLightClient(testConfig())
	require.NoError(err)
	c.provider = prov
	c.state.hasSynced = true
	c.state.headNumber = idx.Block(42)
	wantBalance := uint256.NewInt(42)

	proof := carmen.NewMockWitnessProof(ctrl)
	proof.EXPECT().GetBalance(gomock.Any(), carmen.Address{0x01}).
		Return(carmen.NewAmountFromUint256(wantBalance), true, nil)

	// setup rpc provider to return proof
	prov.EXPECT().getAccountProof(common.Address{0x01}, idx.Block(42)).
		Return(proof, nil)

	// get balance function receives the proof and uses the state root to verify
	balance, err := c.GetBalance(common.Address{0x01})
	require.NoError(err)
	require.Equal(wantBalance, balance)
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
