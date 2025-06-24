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

package inter

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCalcMisbehaviourProofsHash(t *testing.T) {
	v := []MisbehaviourProof{
		{
			EventsDoublesign: &EventsDoublesign{
				Pair: [2]SignedEventLocator{
					test_signed_event_locator,
					test_signed_event_locator,
				},
			},
		},
		{
			BlockVoteDoublesign: &BlockVoteDoublesign{
				Block: test_block_votes.Start,
				Pair: [2]LlrSignedBlockVotes{
					test_llr_signed_block_votes,
					test_llr_signed_block_votes,
				},
			},
		},
		{
			WrongBlockVote: &WrongBlockVote{
				Block: test_block_votes.Start,
				Pals: [2]LlrSignedBlockVotes{
					test_llr_signed_block_votes,
					test_llr_signed_block_votes,
				},
				WrongEpoch: true,
			},
		},
		{
			EpochVoteDoublesign: &EpochVoteDoublesign{
				Pair: [2]LlrSignedEpochVote{
					test_llr_signed_epoch_vote,
					test_llr_signed_epoch_vote,
				},
			},
		},
		{
			WrongEpochVote: &WrongEpochVote{
				Pals: [2]LlrSignedEpochVote{
					test_llr_signed_epoch_vote,
					test_llr_signed_epoch_vote,
				},
			},
		},
	}

	require.Equal(t, "85834ef7fc1d75d65832b1f9b45b43c4f5677811bb2d384208553d32ab49def1", hex.EncodeToString(CalcMisbehaviourProofsHash(v).Bytes()))
}

func TestEvent_Version3_PayloadHashMatchesHashOfPayload(t *testing.T) {
	require := require.New(t)

	for turn := range Turn(2) {
		payload := Payload{ProposalSyncState: ProposalSyncState{
			LastSeenProposalTurn: turn,
		}}

		builder := MutableEventPayload{}
		builder.SetVersion(3)
		builder.SetPayload(payload)
		event := builder.Build()

		require.Equal(CalcPayloadHash(event), event.PayloadHash())
		require.Equal(payload.Hash(), event.PayloadHash())
	}
}
