package iep

import (
	"testing"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/ier"
)

func TestLlrEpochPack(t *testing.T) {
	pack := LlrEpochPack{
		Votes:  []inter.LlrSignedEpochVote{},
		Record: ier.LlrIdxFullEpochRecord{Idx: idx.Epoch(1)},
	}
	if pack.Record.Idx != 1 {
		t.Fatalf("expected epoch 1, got %d", pack.Record.Idx)
	}
	if len(pack.Votes) != 0 {
		t.Fatal("expected empty votes")
	}
}

func TestLlrEpochPack_WithVotes(t *testing.T) {
	pack := LlrEpochPack{
		Votes: []inter.LlrSignedEpochVote{
			{},
			{},
		},
		Record: ier.LlrIdxFullEpochRecord{Idx: idx.Epoch(2)},
	}
	if len(pack.Votes) != 2 {
		t.Fatalf("expected 2 votes, got %d", len(pack.Votes))
	}
}
