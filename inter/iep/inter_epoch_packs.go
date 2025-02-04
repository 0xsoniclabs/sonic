package iep

import (
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/ier"
)

type LlrEpochPack struct {
	Votes  []inter.LlrSignedEpochVote
	Record ier.LlrIdxFullEpochRecord
}
