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
	"testing"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/stretchr/testify/require"
)

func TestProposalTracker_Reset_ResetsPendingProposals(t *testing.T) {
	require := require.New(t)
	tracker := &ProposalTracker{}

	tracker.RegisterSeenProposal(0, 0)
	tracker.RegisterSeenProposal(1, 1)

	require.True(tracker.IsPending(2, 0))
	require.True(tracker.IsPending(2, 1))

	tracker.Reset()

	require.False(tracker.IsPending(2, 0))
	require.False(tracker.IsPending(2, 1))
}

func TestProposalTracker_RegisterSeenProposal_RegistersProposals(t *testing.T) {
	require := require.New(t)
	tracker := &ProposalTracker{}

	// Initially, no proposals are pending
	require.False(tracker.IsPending(0, 0))
	require.False(tracker.IsPending(0, 1))

	// Register a proposal for block 0 at frame 0
	tracker.RegisterSeenProposal(0, 0)
	require.True(tracker.IsPending(0, 0))
	require.False(tracker.IsPending(0, 1))

	// Register a proposal for block 1 at frame 1
	tracker.RegisterSeenProposal(1, 1)
	require.True(tracker.IsPending(1, 1))
	require.True(tracker.IsPending(0, 0))
}

func TestProposalTracker_RegisterSeenProposal_CanHandleMultipleProposalsInSameFrame(t *testing.T) {
	require := require.New(t)
	tracker := &ProposalTracker{}

	now := idx.Frame(5)
	require.False(tracker.IsPending(now, 0))
	require.False(tracker.IsPending(now, 1))

	tracker.RegisterSeenProposal(now, 0)
	tracker.RegisterSeenProposal(now, 1)

	require.True(tracker.IsPending(now, 0))
	require.True(tracker.IsPending(now, 1))

	now++
	require.True(tracker.IsPending(now, 0))
	require.True(tracker.IsPending(now, 1))

	now += TurnTimeoutInFrames - 1
	require.True(tracker.IsPending(now, 0))
	require.True(tracker.IsPending(now, 1))

	now++
	require.False(tracker.IsPending(now, 0))
	require.False(tracker.IsPending(now, 1))
}

func TestProposalTracker_IsPending_InitiallyNoProposalsArePending(t *testing.T) {
	tracker := &ProposalTracker{}
	for b := range idx.Block(10) {
		for f := range idx.Frame(10) {
			require.False(
				t, tracker.IsPending(f, b),
				"block %d should not be pending in frame %d", b, f,
			)
		}
	}
}

func TestProposalTracker_IsPending_PurgesOutdatedProposals(t *testing.T) {
	require := require.New(t)
	tracker := &ProposalTracker{}

	block := idx.Block(123)
	initialFrame := idx.Frame(12)
	currentFrame := initialFrame
	require.False(tracker.IsPending(currentFrame, block))
	tracker.RegisterSeenProposal(currentFrame, block)
	require.True(tracker.IsPending(currentFrame, block))

	for range TurnTimeoutInFrames {
		currentFrame++
		require.True(tracker.IsPending(currentFrame, block))
	}
	require.Equal(initialFrame+TurnTimeoutInFrames, currentFrame)

	currentFrame++
	require.False(tracker.IsPending(currentFrame, block))
}
