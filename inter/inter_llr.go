package inter

import (
	"crypto/sha256"

	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/consensus/utils/byteutils"
)

type LlrBlockVotes struct {
	Start consensus.BlockID
	Epoch consensus.Epoch
	Votes []consensus.Hash
}

func (bvs LlrBlockVotes) LastBlock() consensus.BlockID {
	return bvs.Start + consensus.BlockID(len(bvs.Votes)) - 1
}

type LlrEpochVote struct {
	Epoch consensus.Epoch
	Vote  consensus.Hash
}

type LlrSignedBlockVotes struct {
	Signed                       SignedEventLocator
	TxsAndMisbehaviourProofsHash consensus.Hash
	EpochVoteHash                consensus.Hash
	Val                          LlrBlockVotes
}

type LlrSignedEpochVote struct {
	Signed                       SignedEventLocator
	TxsAndMisbehaviourProofsHash consensus.Hash
	BlockVotesHash               consensus.Hash
	Val                          LlrEpochVote
}

func (r SignedEventLocator) Size() uint64 {
	return uint64(len(r.Sig)) + 3*32 + 4*4
}

func (bvs LlrSignedBlockVotes) Size() uint64 {
	return bvs.Signed.Size() + uint64(len(bvs.Val.Votes))*32 + 32*2 + 8 + 4
}

func (ers LlrEpochVote) Hash() consensus.Hash {
	hasher := sha256.New()
	hasher.Write(ers.Epoch.Bytes())
	hasher.Write(ers.Vote.Bytes())
	return consensus.BytesToHash(hasher.Sum(nil))
}

func (bvs LlrBlockVotes) Hash() consensus.Hash {
	hasher := sha256.New()
	hasher.Write(bvs.Start.Bytes())
	hasher.Write(bvs.Epoch.Bytes())
	hasher.Write(byteutils.Uint32ToBigEndian(uint32(len(bvs.Votes))))
	for _, bv := range bvs.Votes {
		hasher.Write(bv.Bytes())
	}
	return consensus.BytesToHash(hasher.Sum(nil))
}

func (bvs LlrSignedBlockVotes) CalcPayloadHash() consensus.Hash {
	return consensus.Of(bvs.TxsAndMisbehaviourProofsHash.Bytes(), consensus.Of(bvs.EpochVoteHash.Bytes(), bvs.Val.Hash().Bytes()).Bytes())
}

func (ev LlrSignedEpochVote) CalcPayloadHash() consensus.Hash {
	return consensus.Of(ev.TxsAndMisbehaviourProofsHash.Bytes(), consensus.Of(ev.Val.Hash().Bytes(), ev.BlockVotesHash.Bytes()).Bytes())
}

func (ev LlrSignedEpochVote) Size() uint64 {
	return ev.Signed.Size() + 32 + 32*2 + 4 + 4
}
