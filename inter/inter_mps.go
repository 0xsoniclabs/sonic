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
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
)

const (
	// MinAccomplicesForProof defines how many validators must have signed the same wrong vote.
	// Otherwise, wrong-signer is not liable as a protection against singular software/hardware failures
	MinAccomplicesForProof = 2
)

type EventsDoublesign struct {
	Pair [2]SignedEventLocator
}

type BlockVoteDoublesign struct {
	Block idx.Block
	Pair  [2]LlrSignedBlockVotes
}

func (p BlockVoteDoublesign) GetVote(i int) hash.Hash {
	return p.Pair[i].Val.Votes[p.Block-p.Pair[i].Val.Start]
}

type WrongBlockVote struct {
	Block      idx.Block
	Pals       [MinAccomplicesForProof]LlrSignedBlockVotes
	WrongEpoch bool
}

func (p WrongBlockVote) GetVote(i int) hash.Hash {
	return p.Pals[i].Val.Votes[p.Block-p.Pals[i].Val.Start]
}

type EpochVoteDoublesign struct {
	Pair [2]LlrSignedEpochVote
}

type WrongEpochVote struct {
	Pals [MinAccomplicesForProof]LlrSignedEpochVote
}

type MisbehaviourProof struct {
	EventsDoublesign *EventsDoublesign `rlp:"nil"`

	BlockVoteDoublesign *BlockVoteDoublesign `rlp:"nil"`

	WrongBlockVote *WrongBlockVote `rlp:"nil"`

	EpochVoteDoublesign *EpochVoteDoublesign `rlp:"nil"`

	WrongEpochVote *WrongEpochVote `rlp:"nil"`
}
