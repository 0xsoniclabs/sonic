package inter

import (
	"math"
	"testing"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/stretchr/testify/require"
)

func TestIsValidTurnProgression_ValidCases(t *testing.T) {
	type S = ProposalSummary
	const C = TurnTimeoutInFrames
	tests := map[string]struct {
		last ProposalSummary
		next ProposalSummary
	}{
		"next turn in next frame": {
			last: S{Turn: 1, Frame: 1},
			next: S{Turn: 2, Frame: 2},
		},
		"delayed next turn": {
			last: S{Turn: 1, Frame: 1},
			next: S{Turn: 2, Frame: 3},
		},
		"skipped turn": { // Turn 2 fails
			last: S{Turn: 1, Frame: 1},
			next: S{Turn: 3, Frame: 1 + C},
		},
		"delayed skipped turn": { // Turn 2 fails
			last: S{Turn: 1, Frame: 1},
			next: S{Turn: 3, Frame: 1 + C + 2},
		},
		"double skipped turns": { // Turns 2 and 3 fail
			last: S{Turn: 1, Frame: 1},
			next: S{Turn: 4, Frame: 1 + 2*C},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if !IsValidTurnProgression(test.last, test.next) {
				t.Errorf("expected valid turn progression")
			}
		})
	}
}

func TestIsValidTurnProgression_InvalidCases(t *testing.T) {
	type S = ProposalSummary
	const C = TurnTimeoutInFrames

	tests := map[string]struct {
		last ProposalSummary
		next ProposalSummary
	}{
		"same turn in same frame": {
			last: S{Turn: 1, Frame: 1},
			next: S{Turn: 1, Frame: 1},
		},
		"past turn in same frame": {
			last: S{Turn: 1, Frame: 1},
			next: S{Turn: 0, Frame: 2},
		},
		"same turn in next frame": {
			last: S{Turn: 1, Frame: 1},
			next: S{Turn: 1, Frame: 2},
		},
		"skipped turn too early": { // Turn 2 fails
			last: S{Turn: 1, Frame: 1},
			next: S{Turn: 3, Frame: 1 + C - 1},
		},
		"double-skipped turn too early": { // Turns 2 and 3 fail
			last: S{Turn: 1, Frame: 1},
			next: S{Turn: 4, Frame: 1 + 2*C - 1},
		},
		"inverted turn leading to zero delay if using 32-bit arithmetic": {
			last: S{Turn: 1, Frame: 1},
			next: S{Turn: invertTurn(1), Frame: 2},
		},
		"no overflow in maximum distance": {
			last: S{Turn: 0, Frame: 1},
			next: S{Turn: math.MaxUint32, Frame: 2},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if IsValidTurnProgression(test.last, test.next) {
				t.Errorf("expected invalid turn progression")
			}
		})
	}
}

func invertTurn(t Turn) Turn {
	const C = TurnTimeoutInFrames
	return Turn(uint64(1<<32)/C + uint64(t) + 1)
}

func TestInvertTurn_ProducesTurnThatCausesAZeroGap(t *testing.T) {
	for turn := range Turn(10) {
		inverted := invertTurn(turn)
		gap := TurnTimeoutInFrames * idx.Frame(inverted-turn-1)
		require.Equal(t, idx.Frame(0), gap, "expected gap to be zero")
	}
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

		// A second implementation of the function to compare results with.
		want := false
		if last.Turn+1 == next.Turn {
			want = last.Frame < next.Frame
		} else if last.Turn < next.Turn {
			gap := TurnTimeoutInFrames * uint64(next.Turn-last.Turn-1)
			if gap < TurnTimeoutInFrames {
				t.Errorf("frame gap computation underflow")
			}
			want = uint64(last.Frame)+gap <= uint64(next.Frame)
		}

		got := IsValidTurnProgression(last, next)
		if want != got {
			t.Errorf("expected %v, got %v", want, got)
		}
	})
}
