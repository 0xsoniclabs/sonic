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
	"crypto/sha256"

	"github.com/Fantom-foundation/lachesis-base/common/bigendian"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
)

type LlrBlockVotes struct {
	Start idx.Block
	Epoch idx.Epoch
	Votes []hash.Hash
}

func (bvs LlrBlockVotes) LastBlock() idx.Block {
	return bvs.Start + idx.Block(len(bvs.Votes)) - 1
}

type LlrEpochVote struct {
	Epoch idx.Epoch
	Vote  hash.Hash
}

type LlrSignedBlockVotes struct {
	Signed                       SignedEventLocator
	TxsAndMisbehaviourProofsHash hash.Hash
	EpochVoteHash                hash.Hash
	Val                          LlrBlockVotes
}

type LlrSignedEpochVote struct {
	Signed                       SignedEventLocator
	TxsAndMisbehaviourProofsHash hash.Hash
	BlockVotesHash               hash.Hash
	Val                          LlrEpochVote
}

func (r SignedEventLocator) Size() uint64 {
	return uint64(len(r.Sig)) + 3*32 + 4*4
}

func (bvs LlrSignedBlockVotes) Size() uint64 {
	return bvs.Signed.Size() + uint64(len(bvs.Val.Votes))*32 + 32*2 + 8 + 4
}

func (ers LlrEpochVote) Hash() hash.Hash {
	hasher := sha256.New()
	hasher.Write(ers.Epoch.Bytes())
	hasher.Write(ers.Vote.Bytes())
	return hash.BytesToHash(hasher.Sum(nil))
}

func (bvs LlrBlockVotes) Hash() hash.Hash {
	hasher := sha256.New()
	hasher.Write(bvs.Start.Bytes())
	hasher.Write(bvs.Epoch.Bytes())
	hasher.Write(bigendian.Uint32ToBytes(uint32(len(bvs.Votes))))
	for _, bv := range bvs.Votes {
		hasher.Write(bv.Bytes())
	}
	return hash.BytesToHash(hasher.Sum(nil))
}

func (bvs LlrSignedBlockVotes) CalcPayloadHash() hash.Hash {
	return hash.Of(bvs.TxsAndMisbehaviourProofsHash.Bytes(), hash.Of(bvs.EpochVoteHash.Bytes(), bvs.Val.Hash().Bytes()).Bytes())
}

func (ev LlrSignedEpochVote) CalcPayloadHash() hash.Hash {
	return hash.Of(ev.TxsAndMisbehaviourProofsHash.Bytes(), hash.Of(ev.Val.Hash().Bytes(), ev.BlockVotesHash.Bytes()).Bytes())
}

func (ev LlrSignedEpochVote) Size() uint64 {
	return ev.Signed.Size() + 32 + 32*2 + 4 + 4
}
