package scc

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/0xsoniclabs/sonic/scc/bls"
	"github.com/stretchr/testify/require"
)

func TestCommittee_NewCommittee_EmptyCommitteeUsesNilMembers(t *testing.T) {
	committee := NewCommittee()
	require.Nil(t, committee.members)
}

func TestCommittee_Members_EnumeratesMembers(t *testing.T) {
	tests := [][]Member{
		{},
		{newTestMember(1, 10)},
		{newTestMember(1, 10), newTestMember(2, 20)},
	}

	for _, members := range tests {
		committee := Committee{members: members}
		require.Equal(t, members, committee.Members())
	}
}

func TestCommittee_GetMember_ReturnsCorrectMember(t *testing.T) {
	members := []Member{
		newTestMember(1, 10),
		newTestMember(2, 20),
		newTestMember(3, 15),
	}
	committee := Committee{members: members}

	for i, m := range members {
		got, found := committee.GetMember(MemberId(i))
		require.True(t, found)
		require.Equal(t, m, got)
	}
}

func TestCommittee_GetMember_ReturnsNotFoundIfOutOfBounds(t *testing.T) {
	committee := Committee{members: []Member{newTestMember(1, 10)}}
	_, found := committee.GetMember(1)
	require.False(t, found)
	_, found = committee.GetMember(2)
	require.False(t, found)
}

func TestCommittee_GetMemberId_ReturnsCorrectId(t *testing.T) {
	members := []Member{
		newTestMember(1, 10),
		newTestMember(2, 20),
		newTestMember(3, 15),
	}
	committee := Committee{members: members}

	for i, m := range members {
		id, found := committee.GetMemberId(m.PublicKey)
		require.True(t, found)
		require.Equal(t, MemberId(i), id)
	}
}

func TestCommittee_GetMemberId_ReturnsNotFoundIfNotPresent(t *testing.T) {
	committee := Committee{members: []Member{newTestMember(1, 10)}}
	_, found := committee.GetMemberId(bls.NewPrivateKeyForTests(2).PublicKey())
	require.False(t, found)
}

func TestCommittee_GetTotalVotingPower_ReturnsCorrectTotal(t *testing.T) {
	total, overflow := Committee{}.GetTotalVotingPower()
	require.False(t, overflow)
	require.Equal(t, uint64(0), total)

	total, overflow = Committee{[]Member{
		newTestMember(1, 10),
		newTestMember(2, 20),
		newTestMember(3, 15),
	}}.GetTotalVotingPower()
	require.False(t, overflow)
	require.Equal(t, uint64(45), total)
}

func TestCommittee_GetTotalVotingPower_DetectsOverflow(t *testing.T) {
	_, overflow := Committee{[]Member{
		newTestMember(1, 10),
		newTestMember(2, 20),
		newTestMember(3, math.MaxUint64-15),
	}}.GetTotalVotingPower()
	require.True(t, overflow)
}

func TestCommittee_Validate_AcceptsMaximumVotingPower(t *testing.T) {
	members := []Member{
		newTestMember(1, math.MaxUint64),
	}
	committee := Committee{members: members}
	require.NoError(t, committee.Validate())
}

func TestCommittee_Validate_DetectsInvalidCommittee(t *testing.T) {
	tests := map[string]struct {
		members []Member
		err     string
	}{
		"empty": {
			members: nil,
			err:     "at least one member",
		},
		"too many members": {
			members: make([]Member, MaxCommitteeSize+1),
			err:     "committee size exceeds the maximum",
		},
		"invalid member": {
			members: []Member{
				newTestMember(1, 10),
				{},
				newTestMember(3, 15),
			},
			err: "invalid member",
		},
		"duplicate members": {
			members: []Member{
				newTestMember(1, 10),
				newTestMember(2, 20),
				newTestMember(1, 15),
			},
			err: "duplicate members",
		},
		"voting power overflow": {
			members: []Member{
				newTestMember(1, 10),
				newTestMember(2, 20),
				newTestMember(3, math.MaxUint64-10),
			},
			err: "voting power overflow",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := Committee{members: test.members}.Validate()
			require.Error(t, err)
			require.Contains(t, err.Error(), test.err)
		})
	}
}

func TestCommittee_String_ReturnsHumanReadableSummary(t *testing.T) {
	members := []Member{
		newTestMember(1, 10),
		newTestMember(2, 20),
		newTestMember(3, 15),
	}
	committee := Committee{members: members}

	print := committee.String()
	require.Contains(t, print, "0: 0x84fe..1f52 -> 10")
	require.Contains(t, print, "1: 0x8b85..e766 -> 20")
	require.Contains(t, print, "2: 0xb526..7132 -> 15")
	require.Contains(t, print, "Valid: true")
}

func TestCommittee_Serialize_DeserializeRoundTrip(t *testing.T) {
	members := []Member{
		newTestMember(1, 10),
		newTestMember(2, 20),
		newTestMember(3, 15),
	}
	committee := Committee{members: members}

	data := committee.Serialize()
	got, err := DeserializeCommittee(data)
	require.NoError(t, err)
	require.Equal(t, committee, got)
}

func TestCommittee_Serialize_EmptyCommitteeCanBeSerializedAndDeserialized(t *testing.T) {
	committee := Committee{}
	data := committee.Serialize()
	got, err := DeserializeCommittee(data)
	require.NoError(t, err)
	require.Equal(t, committee, got)
}

func TestCommittee_Deserialize_HandlesInvalidInput(t *testing.T) {
	tests := map[string]struct {
		data []byte
		err  string
	}{
		"invalid data length": {
			data: make([]byte, 1),
			err:  "invalid committee data length",
		},
		"invalid member": {
			data: make([]byte, len(EncodedMember{})),
			err:  "invalid member",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := DeserializeCommittee(test.data)
			require.Error(t, err)
			require.Contains(t, err.Error(), test.err)
		})
	}
}

func newTestMember(i byte, power uint64) Member {
	key := bls.NewPrivateKeyForTests(i)
	return Member{
		PublicKey:         key.PublicKey(),
		ProofOfPossession: key.GetProofOfPossession(),
		VotingPower:       power,
	}
}

func TestCommittee_CanEncodeDecodeJson(t *testing.T) {
	members := []Member{
		newTestMember(1, 10),
		newTestMember(2, 20),
		newTestMember(3, 15),
	}
	committee := Committee{members: members}

	data, err := json.Marshal(committee)
	require.NoError(t, err)
	require.NotEqual(t, "{}", string(data))

	var committee2 Committee
	err = json.Unmarshal(data, &committee2)
	require.NoError(t, err)

	require.Equal(t, committee, committee2)
}
