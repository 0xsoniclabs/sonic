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
}

// JoinProposalSyncStates merges two proposal sync states by taking the maximum
// of each individual field. This is used to aggregate the proposal state from
// an event's parents.
func JoinProposalSyncStates(a, b ProposalSyncState) ProposalSyncState {
	return ProposalSyncState{
		LastSeenProposalTurn:  max(a.LastSeenProposalTurn, b.LastSeenProposalTurn),
		LastSeenProposalFrame: max(a.LastSeenProposalFrame, b.LastSeenProposalFrame),
	}
}

// CalculateIncomingProposalSyncState aggregates the last seen proposal information
// from the event's parents.
func CalculateIncomingProposalSyncState(
	reader EventReader,
	event dag.Event,
) ProposalSyncState {
	// The last seen proposal information of the parents needs to be aggregated.
	res := ProposalSyncState{}
	for _, parent := range event.Parents() {
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
	// GetEventPayload must be able to return the payload of parent events of an
	// event for which the incoming proposal sync state is being calculated.
	GetEventPayload(hash.Event) Payload
}

// --- determination of the proposal turn ---

// IsAllowedToPropose checks whether the current validator is allowed to
// propose a new block. If so, the turn for the allowed proposal is returned.
func IsAllowedToPropose(
	validator idx.ValidatorID,
	validators *pos.Validators,
	proposalState ProposalSyncState,
	currentEpoch idx.Epoch,
	currentFrame idx.Frame,
) (bool, Turn, error) {
	// Check whether it is this emitter's turn to propose a new block.
	nextTurn := getCurrentTurn(proposalState, currentFrame) + 1
	proposer, err := GetProposer(validators, currentEpoch, nextTurn)
	if err != nil || proposer != validator {
		return false, 0, err
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
	), nextTurn, nil
}

// getCurrentTurn calculates the current turn based on the last seen proposal
// state and the current frame. This function considers the implicit turn
// progression that occurs if no proposals are made within the timeout period.
func getCurrentTurn(
	proposalState ProposalSyncState,
	currentFrame idx.Frame,
) Turn {
	if currentFrame <= proposalState.LastSeenProposalFrame {
		return proposalState.LastSeenProposalTurn
	}
	delta := currentFrame - proposalState.LastSeenProposalFrame - 1
	return proposalState.LastSeenProposalTurn + Turn(delta/TurnTimeoutInFrames)
}
