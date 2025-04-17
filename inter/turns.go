package inter

import (
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
)

// Turn is the turn number of a proposal. Turns are used to orchestrate the
// sequence of block proposals in the consensus protocol. Turns are processed
// in order. A turn ends with a proposer making a proposal or a timeout.
type Turn uint32

// TurnTimeoutInFrames is the number of frames after which a turn is considered
// failed. Hence, if for the given number of frames no proposal is made, the a
// turn times out and the next turn is started.
//
// The value is set to 8 frames after empirical testing of the network has shown
// an average latency of 3 frames. The timeout is set to 8 frames to account for
// network latency, processing time, and other factors that may cause delays.
//
// ATTENTION: All nodes on the network must agree on the same value for this
// constant. Thus, changing this value requires a hard fork.
const TurnTimeoutInFrames = 8

// IsValidTurnProgression determines whether `next` is a valid successor of
// `last`. This is used during event validation to identify valid proposals and
// discard invalid ones.
//
// ATTENTION: this code is consensus critical. All nodes on the network must
// agree on the same logic. Thus, changing this code requires a hard fork.
func IsValidTurnProgression(
	last ProposalSummary,
	next ProposalSummary,
) bool {
	// Turns must strictly increase in each progression step.
	if last.Turn >= next.Turn {
		return false
	}

	// In the good case, the subsequent proposal is for the succeeding turn
	// in a subsequent frame. This does not require a minimum waiting period.
	if last.Turn+1 == next.Turn {
		return last.Frame < next.Frame
	}

	// If there is a failed turn (either not proposed or not accepted), the
	// next turn must be at least ProposalTimeoutInterval frames after the
	// previous turn. This is to give a proposer enough time to propose
	// a its block before being declared failed.
	gap := TurnTimeoutInFrames * uint64(next.Turn-last.Turn-1)
	return uint64(last.Frame)+gap <= uint64(next.Frame)
}

// ProposalSummary is a summary of the metadata of a proposal made in a turn.
type ProposalSummary struct {
	// Turn is the turn number the proposal was made in.
	Turn Turn
	// Frame is the frame number the proposal was made in.
	Frame idx.Frame
}
