package ier

import (
	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/sonic/inter/iblockproc"
)

type LlrFullEpochRecord struct {
	BlockState iblockproc.BlockState
	EpochState iblockproc.EpochState
}

type LlrIdxFullEpochRecord struct {
	LlrFullEpochRecord
	Idx consensus.Epoch
}

func (er LlrFullEpochRecord) Hash() consensus.Hash {
	return consensus.Of(er.BlockState.Hash().Bytes(), er.EpochState.Hash().Bytes())
}
