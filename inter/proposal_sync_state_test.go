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
					for blockA := range idx.Block(5) {
						for blockB := range idx.Block(5) {
							a := ProposalSyncState{
								LastSeenProposalTurn:  turnA,
								LastSeenProposalFrame: frameA,
								LastSeenProposedBlock: blockA,
							}
							b := ProposalSyncState{
								LastSeenProposalTurn:  turnB,
								LastSeenProposalFrame: frameB,
								LastSeenProposedBlock: blockB,
							}
							joined := JoinProposalSyncStates(a, b)
							require.Equal(t, max(turnA, turnB), joined.LastSeenProposalTurn)
							require.Equal(t, max(frameA, frameB), joined.LastSeenProposalFrame)
							require.Equal(t, max(blockA, blockB), joined.LastSeenProposedBlock)
						}
					}
				}
			}
		}
	}
}

func TestCalculateIncomingProposalSyncState_ProducesEpochStartStateForGenesisEvent(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	world := NewMockEventReader(ctrl)

	event := &MutableEventPayload{}
	event.SetEpoch(42)
	require.Empty(event.Parents())

	epochStartBlock := idx.Block(123)
	world.EXPECT().GetEpochStartBlock(event.Epoch()).Return(epochStartBlock)

	state := CalculateIncomingProposalSyncState(world, event)
	require.Equal(Turn(0), state.LastSeenProposalTurn)
	require.Equal(idx.Frame(0), state.LastSeenProposalFrame)
	require.Equal(epochStartBlock, state.LastSeenProposedBlock)
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
			LastSeenProposedBlock: idx.Block(0x23),
		}},
		p2: {ProposalSyncState: ProposalSyncState{
			LastSeenProposalTurn:  Turn(0x03),
			LastSeenProposalFrame: idx.Frame(0x11),
			LastSeenProposedBlock: idx.Block(0x22),
		}},
		p3: {ProposalSyncState: ProposalSyncState{
			LastSeenProposalTurn:  Turn(0x02),
			LastSeenProposalFrame: idx.Frame(0x13),
			LastSeenProposedBlock: idx.Block(0x21),
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
	require.Equal(idx.Block(0x23), state.LastSeenProposedBlock)
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
			LastSeenProposedBlock: idx.Block(4),
		},
		next.Frame,
		5, // block to be proposed
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

	validTurn := Turn(5)
	validProposer, err := GetProposer(validators, validTurn)
	require.NoError(t, err)
	invalidProposer := validatorA
	if invalidProposer == validProposer {
		invalidProposer = validatorB
	}

	type input struct {
		validator         idx.ValidatorID
		blockToBeProposed idx.Block
		currentFrame      idx.Frame
	}

	tests := map[string]func(*input){
		"wrong proposer": func(input *input) {
			input.validator = invalidProposer
		},
		"proposed block is too old": func(input *input) {
			input.blockToBeProposed = input.blockToBeProposed - 1
		},
		"proposed block is too new": func(input *input) {
			input.blockToBeProposed = input.blockToBeProposed + 1
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
				LastSeenProposedBlock: 5,
			}

			input := input{
				currentFrame:      67,
				validator:         validProposer,
				blockToBeProposed: 6,
			}

			ok, err := IsAllowedToPropose(
				input.validator,
				validators,
				ProposalState,
				input.currentFrame,
				input.blockToBeProposed,
			)
			require.NoError(err)
			require.True(ok)

			corrupt(&input)
			ok, err = IsAllowedToPropose(
				input.validator,
				validators,
				ProposalState,
				input.currentFrame,
				input.blockToBeProposed,
			)
			require.NoError(err)
			require.False(ok)
		})
	}
}

func TestIsAllowedToPropose_ForwardsTurnSelectionError(t *testing.T) {
	validators := pos.ValidatorsBuilder{}.Build()

	_, want := GetProposer(validators, Turn(0))
	require.Error(t, want)

	_, got := IsAllowedToPropose(
		idx.ValidatorID(0),
		validators,
		ProposalSyncState{},
		idx.Frame(0),
		idx.Block(1),
	)
	require.Error(t, got)
	require.Equal(t, got, want)
}
