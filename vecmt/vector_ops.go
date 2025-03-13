package vecmt

import (
	"github.com/0xsoniclabs/consensus/inter/dag"
	"github.com/0xsoniclabs/consensus/inter/idx"
	"github.com/0xsoniclabs/consensus/vecengine"
	"github.com/0xsoniclabs/consensus/vecfc"

	"github.com/0xsoniclabs/sonic/inter"
)

type CreationTimer interface {
	CreationTime() inter.Timestamp
}

func (b *HighestBefore) InitWithEvent(i idx.Validator, e dag.Event) {
	b.VSeq.InitWithEvent(i, e)
	b.VTime.Set(i, e.(CreationTimer).CreationTime())
}

func (b *HighestBefore) IsEmpty(i idx.Validator) bool {
	return b.VSeq.IsEmpty(i)
}

func (b *HighestBefore) IsForkDetected(i idx.Validator) bool {
	return b.VSeq.IsForkDetected(i)
}

func (b *HighestBefore) Seq(i idx.Validator) idx.Event {
	return b.VSeq.Seq(i)
}

func (b *HighestBefore) MinSeq(i idx.Validator) idx.Event {
	return b.VSeq.MinSeq(i)
}

func (b *HighestBefore) SetForkDetected(i idx.Validator) {
	b.VSeq.SetForkDetected(i)
}

func (hb *HighestBefore) CollectFrom(_other vecengine.HighestBeforeI, num idx.Validator) {
	other := _other.(*HighestBefore)
	for branchID := idx.Validator(0); branchID < num; branchID++ {
		hisSeq := other.VSeq.Get(branchID)
		if hisSeq.Seq == 0 && !hisSeq.IsForkDetected() {
			// hisSeq doesn't observe anything about this branchID
			continue
		}
		mySeq := hb.VSeq.Get(branchID)

		if mySeq.IsForkDetected() {
			// mySeq observes the maximum already
			continue
		}
		if hisSeq.IsForkDetected() {
			// set fork detected
			hb.SetForkDetected(branchID)
		} else {
			if mySeq.Seq == 0 || mySeq.MinSeq > hisSeq.MinSeq {
				// take hisSeq.MinSeq
				mySeq.MinSeq = hisSeq.MinSeq
				hb.VSeq.Set(branchID, mySeq)
			}
			if mySeq.Seq < hisSeq.Seq {
				// take hisSeq.Seq
				mySeq.Seq = hisSeq.Seq
				hb.VSeq.Set(branchID, mySeq)
				hb.VTime.Set(branchID, other.VTime.Get(branchID))
			}
		}
	}
}

func (hb *HighestBefore) GatherFrom(to idx.Validator, _other vecengine.HighestBeforeI, from []idx.Validator) {
	other := _other.(*HighestBefore)
	// read all branches to find highest event
	highestBranchSeq := vecfc.BranchSeq{}
	highestBranchTime := inter.Timestamp(0)
	for _, branchID := range from {
		vseq := other.VSeq.Get(branchID)
		vtime := other.VTime.Get(branchID)
		if vseq.IsForkDetected() {
			highestBranchSeq = vseq
			break
		}
		if vseq.Seq > highestBranchSeq.Seq {
			highestBranchSeq = vseq
			highestBranchTime = vtime
		}
	}
	hb.VSeq.Set(to, highestBranchSeq)
	hb.VTime.Set(to, highestBranchTime)
}
