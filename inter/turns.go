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
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
)

// Turn is the turn number of a proposal. Turns are used to orchestrate the
// sequence of block proposals in the consensus protocol. Turns are processed
// in order. A turn ends with a proposer making a proposal or a timeout.
type Turn uint32

// TurnTimeoutInFrames is the number of frames after which a turn is considered
// failed. Hence, if for the given number of frames no proposal is made, the
// current turn times out and the next turn is started.
//
// The value is set to 3 frames after empirical testing of the network has shown
// an average latency of 3 frames.
//
// ATTENTION: All nodes on the network must agree on the same value for this
// constant. Thus, changing this value requires a hard fork.
const TurnTimeoutInFrames = 3

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
	// Turns and frames must strictly increase in each progression step.
	if last.Turn >= next.Turn || last.Frame >= next.Frame {
		return false
	}

	// Every turn has a window of frames after the last successful turn during
	// which it is valid to make a proposal. Let l be the frame number of the
	// last successful turn t, and q the attempted turn to be made. Then q is
	// valid for the frames f if
	//
	//                       d * C < f - l <= (d + 1) * C
	// where
	//   - d = q - t - 1 ... number of failed turns between t and q
	//   - C = TurnTimeoutInFrames ... number of frames after which a turn is
	//     considered to have timed out
	//
	// Thus, the immediate successor turn q = t+1 is valid for the frames f iff
	//                              0 < f - l <= C
	// which is equivalent to
	//                              l < f <= l + C
	// If the turn t+1 is not materializing, the next turn q = t+2 is valid for
	// the frames f iff
	//                              C < f - l <= 2 * C
	// which is equivalent to
	//                           l + C < f <= l + 2 * C
	//
	// This rules partition future frames into intervals of size C, each
	// associated to a specific turn. Thus, for no future frame the last seen
	// turn allows more than one proposal to be made. This is important to make
	// sure that validators are not saving up proposals by not using their turns
	// and then proposing all of them in a burst.
	delta := uint32(next.Frame - last.Frame - 1)
	return delta/TurnTimeoutInFrames == uint32(next.Turn-last.Turn-1)
}

// ProposalSummary is a summary of the metadata of a proposal made in a turn.
type ProposalSummary struct {
	// Turn is the turn number the proposal was made in.
	Turn Turn
	// Frame is the frame number the proposal was made in.
	Frame idx.Frame
}
