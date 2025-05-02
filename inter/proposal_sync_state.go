package inter

import (
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/dag"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
)

//go:generate mockgen -source=proposal_sync_state.go -destination=proposal_sync_state_mock.go -package=inter

// ProposalSyncState is a structure holding a summary of the state tracked by
// events on the DAG to facilitate the proposal selection.
type ProposalSyncState struct {
	LastSeenProposalTurn  Turn
	LastSeenProposalFrame idx.Frame
	LastSeenProposedBlock idx.Block
}

func (s *ProposalSyncState) FromPayload(payload Payload) {
	s.LastSeenProposalTurn = payload.LastSeenProposalTurn
	s.LastSeenProposedBlock = payload.LastSeenProposedBlock
	s.LastSeenProposalFrame = payload.LastSeenProposalFrame
}

func (s ProposalSyncState) ToPayload() Payload {
	return Payload{
		LastSeenProposalTurn:  s.LastSeenProposalTurn,
		LastSeenProposedBlock: s.LastSeenProposedBlock,
		LastSeenProposalFrame: s.LastSeenProposalFrame,
	}
}

// JoinProposalState merges two proposal states by taking the maximum of each
// field. This is used to aggregate the proposal state from an event's parents.
func JoinProposalState(a, b ProposalSyncState) ProposalSyncState {
	return ProposalSyncState{
		LastSeenProposalTurn:  max(a.LastSeenProposalTurn, b.LastSeenProposalTurn),
		LastSeenProposedBlock: max(a.LastSeenProposedBlock, b.LastSeenProposedBlock),
		LastSeenProposalFrame: max(a.LastSeenProposalFrame, b.LastSeenProposalFrame),
	}
}

// GetIncomingProposalState aggregates the last seen proposal information
// from the event's parents.
func GetIncomingProposalState(
	reader EventReader,
	event dag.Event,
) ProposalSyncState {
	res := ProposalSyncState{}
	parents := event.Parents()

	// For genesis events without errors, there is no last seen proposal.
	// However, we need to set the last seen proposed block to the start block
	// of the epoch to retain progress.
	if len(parents) == 0 {
		res.LastSeenProposedBlock = reader.GetEpochStartBlock(event.Epoch())
		return res
	}

	// For all other events, the last seen proposal information of the parents
	// needs to be aggregated.
	for _, parent := range parents {
		current := ProposalSyncState{}
		current.FromPayload(reader.GetEventPayload(parent))
		res = JoinProposalState(res, current)
	}
	return res
}

type EventReader interface {
	GetEpochStartBlock(idx.Epoch) idx.Block
	GetEventPayload(hash.Event) Payload
}

// --- determination of the proposal turn ---

// IsAllowedToPropose checks whether the current validator is allowed to
// propose a new block.
func IsAllowedToPropose(
	proposalState ProposalSyncState,
	currentFrame idx.Frame,
	validators *pos.Validators,
	thisValidator idx.ValidatorID,
	blockToPropose idx.Block,
) (bool, error) {
	// Check that the block about to be proposed is the next expected block.
	// TODO: show that this throttling mechanism is safe
	// see https://github.com/0xsoniclabs/sonic-admin/issues/182
	if proposalState.LastSeenProposedBlock+1 != blockToPropose {
		return false, nil
	}

	// Check whether it is this emitter's turn to propose a new block.
	nextTurn := proposalState.LastSeenProposalTurn + 1
	proposer, err := GetProposer(validators, nextTurn)
	if err != nil || proposer != thisValidator {
		return false, err
	}

	// Check that enough time has passed for the next proposal.
	valid := IsValidTurnProgression(
		ProposalSummary{
			Turn:  proposalState.LastSeenProposalTurn,
			Frame: proposalState.LastSeenProposalFrame,
		},
		ProposalSummary{
			Turn:  nextTurn,
			Frame: currentFrame,
		},
	)
	if !valid {
		return false, nil
	}

	// It is indeed this validator's turn to propose a new block.
	return true, nil
}
