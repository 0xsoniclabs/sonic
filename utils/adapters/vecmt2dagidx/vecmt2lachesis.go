package vecmt2dagidx

import (
	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/consensus/consensus/consensusengine"
	"github.com/0xsoniclabs/consensus/dagidx"
	"github.com/0xsoniclabs/consensus/vecengine"

	"github.com/0xsoniclabs/sonic/vecmt"
)

type Adapter struct {
	*vecmt.Index
}

var _ consensusengine.DagIndex = (*Adapter)(nil)

type AdapterSeq struct {
	*vecmt.HighestBefore
}

type BranchSeq struct {
	vecengine.BranchSeq
}

// Seq is a maximum observed e.Seq in the branch
func (b *BranchSeq) Seq() consensus.Seq {
	return b.BranchSeq.Seq
}

// MinSeq is a minimum observed e.Seq in the branch
func (b *BranchSeq) MinSeq() consensus.Seq {
	return b.BranchSeq.MinSeq
}

// Size of the vector clock
func (b AdapterSeq) Size() int {
	return b.VSeq.Size()
}

// Get i's position in the byte-encoded vector clock
func (b AdapterSeq) Get(i consensus.ValidatorIndex) dagidx.Seq {
	seq := b.HighestBefore.VSeq.Get(i)
	return &BranchSeq{seq}
}

func (v *Adapter) GetMergedHighestBefore(id consensus.EventHash) dagidx.HighestBeforeSeq {
	return AdapterSeq{v.Index.GetMergedHighestBefore(id)}
}

func Wrap(v *vecmt.Index) *Adapter {
	return &Adapter{v}
}
