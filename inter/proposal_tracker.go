package inter

import (
	"slices"
	"sync"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
)

// ProposalTracker is a thread-safe structure that tracks proposals seen in the
// network. It retains a list of pending proposals which are automatically
// purged after a certain timeout (defined by TurnTimeoutInFrames). At any time
// users of this utility may query whether a certain block is pending at a
// given frame height.
//
// Attention: the tracker does not keep track of the highest frame number. If
// used with a non-monotonic frame number, results are unspecified.
//
// All methods of ProposalTracker are thread-safe.
type ProposalTracker struct {
	pendingProposals []proposalTrackerEntry
	mu               sync.Mutex
}

type proposalTrackerEntry struct {
	frame idx.Frame
	block idx.Block
}

// Reset clears the list of pending proposals. After the pending proposals are
// cleared, the frame counter can start from zero again.
func (t *ProposalTracker) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.pendingProposals = nil
}

// RegisterSeenProposal informs the tracker about a fresh observation of a block
// proposal at a given frame height. This proposal is tracked until it times
// out.
func (t *ProposalTracker) RegisterSeenProposal(
	frame idx.Frame,
	block idx.Block,
) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.pendingProposals = append(t.pendingProposals, proposalTrackerEntry{
		frame: frame,
		block: block,
	})
}

// IsPending checks whether a proposal for the given block is pending at the
// given frame height. If the proposal is pending, it returns true, otherwise
// it returns false.
//
// A side effect of this function is that it purges proposals that are out-dated
// according to the given frame height. Thus, users of this function should
// ensure that the frame number is always monotonically increasing.
func (t *ProposalTracker) IsPending(
	frame idx.Frame,
	block idx.Block,
) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.pendingProposals = slices.DeleteFunc(
		t.pendingProposals,
		func(entry proposalTrackerEntry) bool {
			return entry.frame+TurnTimeoutInFrames < frame
		},
	)
	for _, entry := range t.pendingProposals {
		if entry.block == block {
			return true
		}
	}
	return false
}
