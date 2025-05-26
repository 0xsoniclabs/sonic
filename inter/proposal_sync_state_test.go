package inter

import (
	"testing"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/dag"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

func TestProposalSyncState_Join_ComputesTheMaximumForIndividualStateProperties(t *testing.T) {
	for turnA := range Turn(5) {
		for turnB := range Turn(5) {
			for frameA := range idx.Frame(5) {
				for frameB := range idx.Frame(5) {
					a := ProposalSyncState{
						LastSeenProposalTurn:  turnA,
						LastSeenProposalFrame: frameA,
					}
					b := ProposalSyncState{
						LastSeenProposalTurn:  turnB,
						LastSeenProposalFrame: frameB,
					}
					joined := JoinProposalSyncStates(a, b)
					require.Equal(t, max(turnA, turnB), joined.LastSeenProposalTurn)
					require.Equal(t, max(frameA, frameB), joined.LastSeenProposalFrame)
				}
			}
		}
	}
}

func TestCalculateIncomingProposalSyncState_ProducesDefaultStateForGenesisEvent(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	world := NewMockEventReader(ctrl)

	event := &MutableEventPayload{}
	event.SetEpoch(42)
	require.Empty(event.Parents())

	state := CalculateIncomingProposalSyncState(world, event)
	want := ProposalSyncState{}
	require.Equal(want, state)
}

func TestCalculateIncomingProposalSyncState_AggregatesParentStates(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	world := NewMockEventReader(ctrl)

	p1 := hash.Event{1}
	p2 := hash.Event{2}
	p3 := hash.Event{3}
	parents := map[hash.Event]Payload{
		p1: {ProposalSyncState: ProposalSyncState{
			LastSeenProposalTurn:  Turn(0x01),
			LastSeenProposalFrame: idx.Frame(0x12),
		}},
		p2: {ProposalSyncState: ProposalSyncState{
			LastSeenProposalTurn:  Turn(0x03),
			LastSeenProposalFrame: idx.Frame(0x11),
		}},
		p3: {ProposalSyncState: ProposalSyncState{
			LastSeenProposalTurn:  Turn(0x02),
			LastSeenProposalFrame: idx.Frame(0x13),
		}},
	}

	world.EXPECT().GetEventPayload(p1).Return(parents[p1])
	world.EXPECT().GetEventPayload(p2).Return(parents[p2])
	world.EXPECT().GetEventPayload(p3).Return(parents[p3])

	event := &dag.MutableBaseEvent{}
	event.SetParents(hash.Events{p1, p2, p3})
	state := CalculateIncomingProposalSyncState(world, event)

	require.Equal(Turn(0x03), state.LastSeenProposalTurn)
	require.Equal(idx.Frame(0x13), state.LastSeenProposalFrame)
}

func TestIsAllowedToPropose_AcceptsValidProposerTurn(t *testing.T) {
	require := require.New(t)

	validator := idx.ValidatorID(1)
	builder := pos.ValidatorsBuilder{}
	builder.Set(validator, 10)
	validators := builder.Build()

	last := ProposalSummary{
		Turn:  Turn(5),
		Frame: idx.Frame(12),
	}
	next := ProposalSummary{
		Turn:  Turn(6),
		Frame: idx.Frame(17),
	}
	require.True(IsValidTurnProgression(last, next))

	ok, err := IsAllowedToPropose(
		validator,
		validators,
		ProposalSyncState{
			LastSeenProposalTurn:  last.Turn,
			LastSeenProposalFrame: last.Frame,
		},
		idx.Epoch(42),
		next.Frame,
	)
	require.NoError(err)
	require.True(ok)
}

func TestIsAllowedToPropose_RejectsInvalidProposerTurn(t *testing.T) {
	validatorA := idx.ValidatorID(1)
	validatorB := idx.ValidatorID(2)
	builder := pos.ValidatorsBuilder{}
	builder.Set(validatorA, 10)
	builder.Set(validatorB, 20)
	validators := builder.Build()

	validEpoch := idx.Epoch(4)
	validTurn := Turn(5)
	validProposer, err := GetProposer(validators, validEpoch, validTurn)
	require.NoError(t, err)
	invalidProposer := validatorA
	if invalidProposer == validProposer {
		invalidProposer = validatorB
	}

	type input struct {
		validator    idx.ValidatorID
		currentFrame idx.Frame
	}

	tests := map[string]func(*input){
		"wrong proposer": func(input *input) {
			input.validator = invalidProposer
		},
		"invalid turn progression": func(input *input) {
			// a proposal made too late needs to be rejected
			input.currentFrame = input.currentFrame * 10
		},
	}

	for name, corrupt := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)

			ProposalState := ProposalSyncState{
				LastSeenProposalTurn:  12,
				LastSeenProposalFrame: 62,
			}

			input := input{
				currentFrame: 67,
				validator:    validProposer,
			}

			ok, err := IsAllowedToPropose(
				input.validator,
				validators,
				ProposalState,
				validEpoch,
				input.currentFrame,
			)
			require.NoError(err)
			require.True(ok)

			corrupt(&input)
			ok, err = IsAllowedToPropose(
				input.validator,
				validators,
				ProposalState,
				validEpoch,
				input.currentFrame,
			)
			require.NoError(err)
			require.False(ok)
		})
	}
}

func TestIsAllowedToPropose_ForwardsTurnSelectionError(t *testing.T) {
	validators := pos.ValidatorsBuilder{}.Build()

	_, want := GetProposer(validators, idx.Epoch(0), Turn(0))
	require.Error(t, want)

	_, got := IsAllowedToPropose(
		idx.ValidatorID(0),
		validators,
		ProposalSyncState{},
		idx.Epoch(0),
		idx.Frame(0),
	)
	require.Error(t, got)
	require.Equal(t, got, want)
}

func TestGetCurrentTurn_ForKnownExamples_ProducesCorrectTurn(t *testing.T) {
	tests := map[string]struct {
		lastTurn     Turn
		lastFrame    idx.Frame
		currentFrame idx.Frame
		want         Turn
	}{
		"same frame": {
			lastTurn:     4,
			lastFrame:    5,
			currentFrame: 5,
			want:         4,
		},
		"previous frame should not decrease the turn": {
			lastTurn:     4,
			lastFrame:    5,
			currentFrame: 4,
			want:         4,
		},
		"next frame should not increase the turn": {
			lastTurn:     4,
			lastFrame:    5,
			currentFrame: 6,
			want:         4,
		},
		"a timeout should increase the turn": {
			lastTurn:     4,
			lastFrame:    5,
			currentFrame: 5 + TurnTimeoutInFrames,
			want:         5,
		},
		"multiple timeouts should increase the turn": {
			lastTurn:     4,
			lastFrame:    5,
			currentFrame: 5 + 2*TurnTimeoutInFrames,
			want:         6,
		},
		"multiple timeouts should increase the turn (2)": {
			lastTurn:     4,
			lastFrame:    5,
			currentFrame: 5 + 3*TurnTimeoutInFrames,
			want:         7,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := getCurrentTurn(
				ProposalSyncState{
					LastSeenProposalTurn:  test.lastTurn,
					LastSeenProposalFrame: test.lastFrame,
				},
				test.currentFrame,
			)
			require.Equal(t, test.want, got)
		})
	}
}

func TestGetCurrentTurn_ForCartesianProductOfInputs_ProducesResultsConsideringTimeouts(t *testing.T) {
	for turn := range Turn(3) {
		for start := range idx.Frame(5) {
			for currentFrame := range idx.Frame(5 * TurnTimeoutInFrames) {
				got := getCurrentTurn(
					ProposalSyncState{
						LastSeenProposalTurn:  turn,
						LastSeenProposalFrame: start,
					},
					currentFrame,
				)

				want := turn
				if currentFrame > start {
					delta := currentFrame - start
					want += Turn(delta / TurnTimeoutInFrames)
				}
				require.Equal(t, want, got)
			}
		}
	}
}
