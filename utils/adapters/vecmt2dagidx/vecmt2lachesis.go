package vecmt2dagidx

import (
	"github.com/0xsoniclabs/consensus/abft"
	"github.com/0xsoniclabs/consensus/abft/dagidx"
	"github.com/0xsoniclabs/consensus/dagindexer"
	"github.com/0xsoniclabs/consensus/hash"
	"github.com/0xsoniclabs/consensus/inter/idx"
)

type Adapter struct {
	*dagindexer.Index
}

var _ abft.DagIndex = (*Adapter)(nil)

type AdapterSeq struct {
	*dagindexer.HighestBefore
}

type BranchSeq struct {
	dagindexer.BranchSeq
}

// Seq is a maximum observed e.Seq in the branch
func (b *BranchSeq) Seq() idx.Event {
	return b.BranchSeq.Seq
}

// MinSeq is a minimum observed e.Seq in the branch
func (b *BranchSeq) MinSeq() idx.Event {
	return b.BranchSeq.MinSeq
}

// Size of the vector clock
func (b AdapterSeq) Size() int {
	return b.VSeq.Size()
}

// Get i's position in the byte-encoded vector clock
func (b AdapterSeq) Get(i idx.Validator) dagidx.Seq {
	seq := b.HighestBefore.VSeq.Get(i)
	return &BranchSeq{seq}
}

func (v *Adapter) GetMergedHighestBefore(id hash.Event) dagidx.HighestBeforeSeq {
	return AdapterSeq{v.Index.GetMergedHighestBefore(id)}
}

func Wrap(v *dagindexer.Index) *Adapter {
	return &Adapter{v}
}
