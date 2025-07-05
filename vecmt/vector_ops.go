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

package vecmt

import (
	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/consensus/vecengine"

	"github.com/0xsoniclabs/sonic/inter"
)

type CreationTimer interface {
	CreationTime() inter.Timestamp
}

func (hb *HighestBefore) InitWithEvent(i consensus.ValidatorIndex, e consensus.Event) {
	hb.VSeq.InitWithEvent(i, e)
	hb.VTime.Set(i, e.(CreationTimer).CreationTime())
}

func (hb *HighestBefore) IsEmpty(i consensus.ValidatorIndex) bool {
	return hb.VSeq.IsEmpty(i)
}

func (hb *HighestBefore) IsForkDetected(i consensus.ValidatorIndex) bool {
	return hb.VSeq.IsForkDetected(i)
}

func (hb *HighestBefore) Seq(i consensus.ValidatorIndex) consensus.Seq {
	return hb.VSeq.Seq(i)
}

func (hb *HighestBefore) MinSeq(i consensus.ValidatorIndex) consensus.Seq {
	return hb.VSeq.MinSeq(i)
}

func (hb *HighestBefore) SetForkDetected(i consensus.ValidatorIndex) {
	hb.VSeq.SetForkDetected(i)
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
