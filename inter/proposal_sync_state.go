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

// JoinProposalSyncStates merges two proposal sync states by taking the maximum
// of each individual field. This is used to aggregate the proposal state from
// an event's parents.
func JoinProposalSyncStates(a, b ProposalSyncState) ProposalSyncState {
	return ProposalSyncState{
		LastSeenProposalTurn:  max(a.LastSeenProposalTurn, b.LastSeenProposalTurn),
		LastSeenProposedBlock: max(a.LastSeenProposedBlock, b.LastSeenProposedBlock),
		LastSeenProposalFrame: max(a.LastSeenProposalFrame, b.LastSeenProposalFrame),
	}
}

// CalculateIncomingProposalSyncState aggregates the last seen proposal information
// from the event's parents.
func CalculateIncomingProposalSyncState(
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
		current := reader.GetEventPayload(parent).ProposalSyncState
		res = JoinProposalSyncStates(res, current)
	}
	return res
}

// EventReader is an interface of an event-information data source required by
// CalculateIncomingProposalSyncState to obtain context information. In
// particular, the payload of the parent events and the block hight at the start
// of the current epoch is required.
type EventReader interface {
	// GetEpochStartBlock must be able to return the block of the current epoch.
	GetEpochStartBlock(idx.Epoch) idx.Block
	// GetEventPayload must be able to return the payload of parent events of an
	// event for which the incoming proposal sync state is being calculated.
	GetEventPayload(hash.Event) Payload
}

// --- determination of the proposal turn ---

// IsAllowedToPropose checks whether the current validator is allowed to
// propose a new block.
func IsAllowedToPropose(
	validator idx.ValidatorID,
	validators *pos.Validators,
	proposalState ProposalSyncState,
	currentFrame idx.Frame,
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
	if err != nil || proposer != validator {
		return false, err
	}

	// Check that enough time has passed for the next proposal.
	return IsValidTurnProgression(
		ProposalSummary{
			Turn:  proposalState.LastSeenProposalTurn,
			Frame: proposalState.LastSeenProposalFrame,
		},
		ProposalSummary{
			Turn:  nextTurn,
			Frame: currentFrame,
		},
	), nil
}
