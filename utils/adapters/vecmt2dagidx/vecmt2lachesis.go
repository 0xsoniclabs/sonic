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

package vecmt2dagidx

import (
	"github.com/Fantom-foundation/lachesis-base/abft"
	"github.com/Fantom-foundation/lachesis-base/abft/dagidx"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/vecfc"

	"github.com/0xsoniclabs/sonic/vecmt"
)

type Adapter struct {
	*vecmt.Index
}

var _ abft.DagIndex = (*Adapter)(nil)

type AdapterSeq struct {
	*vecmt.HighestBefore
}

type BranchSeq struct {
	vecfc.BranchSeq
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
	//nolint:staticcheck // QF1008: do not omit embedded field for clarity
	seq := b.HighestBefore.VSeq.Get(i)
	return &BranchSeq{seq}
}

func (v *Adapter) GetMergedHighestBefore(id hash.Event) dagidx.HighestBeforeSeq {
	return AdapterSeq{v.Index.GetMergedHighestBefore(id)}
}

func Wrap(v *vecmt.Index) *Adapter {
	return &Adapter{v}
}
