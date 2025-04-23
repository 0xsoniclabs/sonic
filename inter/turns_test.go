package inter

import (
	"testing"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
)

func TestIsValidTurnProgression_ExampleCasesAndResults(t *testing.T) {
	type S = ProposalSummary
	const C = TurnTimeoutInFrames
	tests := map[string]struct {
		last  ProposalSummary
		next  ProposalSummary
		valid bool
	}{
		// -- past turn proposals --
		"past turn in same frame": {
			last:  S{Turn: 1, Frame: 1},
			next:  S{Turn: 0, Frame: 1},
			valid: false,
		},
		"past turn in next frame": {
			last:  S{Turn: 1, Frame: 1},
			next:  S{Turn: 0, Frame: 2},
			valid: false,
		},
		"past turn in previous frame": {
			last:  S{Turn: 1, Frame: 1},
			next:  S{Turn: 0, Frame: 0},
			valid: false,
		},
		"past turn in long future frame": {
			last:  S{Turn: 1, Frame: 1},
			next:  S{Turn: 0, Frame: 300},
			valid: false,
		},
		"past turn in long past frame": {
			last:  S{Turn: 1, Frame: 5*C},
			next:  S{Turn: 0, Frame: 1},
			valid: false,
		},

		// -- same turn proposals --
		"same turn in same frame": {
			last:  S{Turn: 1, Frame: 1},
			next:  S{Turn: 1, Frame: 1},
			valid: false,
		},
		"same turn in next frame": {
			last:  S{Turn: 1, Frame: 1},
			next:  S{Turn: 1, Frame: 2},
			valid: false,
		},
		"same turn in previous frame": {
			last:  S{Turn: 1, Frame: 1},
			next:  S{Turn: 1, Frame: 0},
			valid: false,
		},
		"same turn in long future frame": {
			last:  S{Turn: 1, Frame: 1},
			next:  S{Turn: 1, Frame: 1 + 2*C},
			valid: false,
		},

		// -- next turn proposals --
		"next turn in next frame": {
			last:  S{Turn: 1, Frame: 1},
			next:  S{Turn: 2, Frame: 2},
			valid: true,
		},
		"next turn in same frame": {
			last:  S{Turn: 1, Frame: 1},
			next:  S{Turn: 2, Frame: 1},
			valid: false,
		},
		"next turn in previous frame": {
			last:  S{Turn: 1, Frame: 1},
			next:  S{Turn: 2, Frame: 0},
			valid: false,
		},
		"next turn in delayed frame": {
			last:  S{Turn: 1, Frame: 1},
			next:  S{Turn: 2, Frame: 3},
			valid: true,
		},
		"next turn in frame just before timeout": {
			last:  S{Turn: 1, Frame: 1},
			next:  S{Turn: 2, Frame: 1 + C},
			valid: true,
		},
		"next turn in frame after timeout": {
			last:  S{Turn: 1, Frame: 1},
			next:  S{Turn: 2, Frame: 1 + (C + 1)},
			valid: false, // after the timeout, the attempt should be blocked
		},
		"next turn in long future frame": {
			last:  S{Turn: 1, Frame: 1},
			next:  S{Turn: 2, Frame: 1 + 2*C},
			valid: false,
		},

		// -- skipped turn proposals --
		// In these scenarios, turn 2 has not been consumed, enabling turn 3 to
		// be used between (C, 2*C] frames after the last proposal.
		"skipped turn just before being enabled": {
			last:  S{Turn: 1, Frame: 1},
			next:  S{Turn: 3, Frame: 1 + C},
			valid: false,
		},
		"skipped turn just when enabled": {
			last:  S{Turn: 1, Frame: 1},
			next:  S{Turn: 3, Frame: 1 + (C + 1)},
			valid: true,
		},
		"skipped turn delayed in time window": {
			last:  S{Turn: 1, Frame: 1},
			next:  S{Turn: 3, Frame: 1 + (C + 3)},
			valid: true,
		},
		"skipped turn just before end of time window": {
			last:  S{Turn: 1, Frame: 1},
			next:  S{Turn: 3, Frame: 1 + 2*C},
			valid: true,
		},
		"skipped turn just after end of time window": {
			last:  S{Turn: 1, Frame: 1},
			next:  S{Turn: 3, Frame: 1 + (2*C + 1)},
			valid: false,
		},
		"skipped turn long after its own window": {
			last:  S{Turn: 1, Frame: 1},
			next:  S{Turn: 3, Frame: 1 + (2*C + 100)},
			valid: false,
		},

		// -- double-skipped turn proposals --
		// In these scenarios, turns 2 and 3 have not been consumed, enabling
		// turn 4 to be used between (2*C, 3*C] frames after the last proposal.
		"double skipped turn long before being enabled": {
			last:  S{Turn: 1, Frame: 1},
			next:  S{Turn: 4, Frame: 1 + (C + 3)},
			valid: false,
		},
		"double skipped turn just before being enabled": {
			last:  S{Turn: 1, Frame: 1},
			next:  S{Turn: 4, Frame: 1 + 2*C},
			valid: false,
		},
		"double skipped turn just when enabled": {
			last:  S{Turn: 1, Frame: 1},
			next:  S{Turn: 4, Frame: 1 + 2*C + 1},
			valid: true,
		},
		"double skipped turn delayed in time window": {
			last:  S{Turn: 1, Frame: 1},
			next:  S{Turn: 4, Frame: 1 + 2*C + 3},
			valid: true,
		},
		"double skipped turn just before end of time window": {
			last:  S{Turn: 1, Frame: 1},
			next:  S{Turn: 4, Frame: 1 + 3*C},
			valid: true,
		},
		"double skipped turn just after end of time window": {
			last:  S{Turn: 1, Frame: 1},
			next:  S{Turn: 4, Frame: 1 + 3*C + 1},
			valid: false,
		},
		"double skipped turn long after its own window": {
			last:  S{Turn: 1, Frame: 1},
			next:  S{Turn: 4, Frame: 1 + 3*C + 100},
			valid: false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := IsValidTurnProgression(test.last, test.next)
			if got != test.valid {
				t.Errorf("expected %v, got %v", test.valid, got)
			}
		})
	}
}

func TestIsValidTurnProgression_EnumerationTests(t *testing.T) {
	type S = ProposalSummary
	const C = TurnTimeoutInFrames
	for lastTurn := range Turn(10) {
		for lastFrame := range idx.Frame(10) {
			for nextTurn := range Turn(C * 10) {
				for nextFrame := range idx.Frame(C * 10) {
					last := S{
						Turn:  lastTurn,
						Frame: lastFrame,
					}
					next := S{
						Turn:  nextTurn,
						Frame: nextFrame,
					}
					want := isValidTurnProgressionForTests(last, next)
					got := IsValidTurnProgression(last, next)
					if want != got {
						t.Errorf("expected %v, got %v", want, got)
					}
				}
			}
		}
	}
}

// isValidTurnProgressionForTests is an alternative implementation of the function to
// compare the results with the production version. It is intended to be a
// reference implementations in tests.
func isValidTurnProgressionForTests(
	last ProposalSummary,
	next ProposalSummary,
) bool {
	// A straightforward implementation of the logic in IsValidTurnProgression
	// explicitly computing the (dC,(d+1)C] intervals.
	if last.Turn >= next.Turn {
		return false
	}
	delta := uint64(next.Turn - last.Turn - 1)
	min := uint64(last.Frame) + delta*TurnTimeoutInFrames
	max := min + TurnTimeoutInFrames
	return min < uint64(next.Frame) && uint64(next.Frame) <= max
}

func FuzzIsValidTurnProgression(f *testing.F) {

	f.Fuzz(func(t *testing.T, lastTurn, nextTurn, lastFrame, nextFrame uint32) {
		last := ProposalSummary{
			Turn:  Turn(lastTurn),
			Frame: idx.Frame(lastFrame),
		}
		next := ProposalSummary{
			Turn:  Turn(nextTurn),
			Frame: idx.Frame(nextFrame),
		}

		want := isValidTurnProgressionForTests(last, next)
		got := IsValidTurnProgression(last, next)
		if want != got {
			t.Errorf("expected %v, got %v", want, got)
		}
	})
}

var validTurnBenchInput Turn = 12
var validTurnBenchResult bool

func BenchmarkIsValidTurnProgression_Production(b *testing.B) {
	last := false
	for range b.N {
		last = IsValidTurnProgression(
			ProposalSummary{Turn: 1, Frame: 1},
			ProposalSummary{Turn: validTurnBenchInput, Frame: 2},
		)
	}
	validTurnBenchResult = last // needed to avoid compiler optimization
}

func BenchmarkIsValidTurnProgression_Tests(b *testing.B) {
	last := false
	for range b.N {
		last = isValidTurnProgressionForTests(
			ProposalSummary{Turn: 1, Frame: 1},
			ProposalSummary{Turn: validTurnBenchInput, Frame: 2},
		)
	}
	validTurnBenchResult = last // needed to avoid compiler optimization
}
