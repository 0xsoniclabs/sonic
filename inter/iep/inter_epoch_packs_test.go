// Copyright 2026 Sonic Operations Ltd
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
