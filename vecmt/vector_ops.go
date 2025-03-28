package vecmt

import (
	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/consensus/vecengine"

	"github.com/0xsoniclabs/sonic/inter"
)

type CreationTimer interface {
	CreationTime() inter.Timestamp
}

func (b *HighestBefore) InitWithEvent(i consensus.ValidatorIndex, e consensus.Event) {
	b.VSeq.InitWithEvent(i, e)
	b.VTime.Set(i, e.(CreationTimer).CreationTime())
}

func (b *HighestBefore) IsEmpty(i consensus.ValidatorIndex) bool {
	return b.VSeq.IsEmpty(i)
}

func (b *HighestBefore) IsForkDetected(i consensus.ValidatorIndex) bool {
	return b.VSeq.IsForkDetected(i)
}

func (b *HighestBefore) Seq(i consensus.ValidatorIndex) consensus.Seq {
	return b.VSeq.Seq(i)
}

func (b *HighestBefore) MinSeq(i consensus.ValidatorIndex) consensus.Seq {
	return b.VSeq.MinSeq(i)
}

func (b *HighestBefore) SetForkDetected(i consensus.ValidatorIndex) {
	b.VSeq.SetForkDetected(i)
}

func (hb *HighestBefore) CollectFrom(_other vecengine.HighestBeforeI, num consensus.ValidatorIndex) {
	other := _other.(*HighestBefore)
	for branchID := consensus.ValidatorIndex(0); branchID < num; branchID++ {
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

func (hb *HighestBefore) GatherFrom(to consensus.ValidatorIndex, _other vecengine.HighestBeforeI, from []consensus.ValidatorIndex) {
	other := _other.(*HighestBefore)
	// read all branches to find highest event
	highestBranchSeq := vecengine.BranchSeq{}
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
