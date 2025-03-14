package node

import (
	"testing"

	"github.com/0xsoniclabs/consensus/inter/idx"
	"github.com/0xsoniclabs/sonic/scc/bls"
	"github.com/0xsoniclabs/sonic/utils/objstream"
	"github.com/stretchr/testify/require"
)

/*
func TestInMemoryState_RetainsCurrentState(t *testing.T) {
	committee := scc.NewCommittee(scc.Member{})
	state := newInMemoryState(committee)
	require.Equal(t, committee, state.GetCurrentCommittee())
}
*/

func TestState_ImplementsSerializable(t *testing.T) {
	var _ objstream.Serializer = State{}
	var _ objstream.Deserializer = &State{}
}

func TestState_Json_CanBeMarshalledAndUnMarshalled(t *testing.T) {
	key := bls.NewPrivateKeyForTests(1)
	pub := key.PublicKey()
	pop := key.GetProofOfPossession()
	tests := map[string]State{
		"empty": {},
		"with validators": {
			blockHeight: 123,
			validators: map[idx.ValidatorID]validatorInfo{
				1: {
					Valid:                            true,
					Key:                              pub,
					ProofOfPossession:                pop,
					VotingPower:                      12,
					NextExpectedCommitteeAttestation: 45,
					NextExpectedBlockAttestation:     67,
				},
			},
		},
		"with multiple validators": {
			blockHeight: 47,
			validators: map[idx.ValidatorID]validatorInfo{
				1: {
					Valid:                            true,
					Key:                              pub,
					ProofOfPossession:                pop,
					VotingPower:                      12,
					NextExpectedCommitteeAttestation: 45,
					NextExpectedBlockAttestation:     67,
				},
				3: {
					Valid: false,
				},
			},
		},
	}

	for name, state := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			data, err := state.MarshalJSON()
			require.NoError(err)

			restored := State{}
			err = restored.UnmarshalJSON(data)
			require.NoError(err)
			require.Equal(state, restored)
		})
	}
}
