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

package emitter

import (
	"testing"

	"github.com/0xsoniclabs/sonic/vecmt"
	"github.com/Fantom-foundation/lachesis-base/emitter/ancestor"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
	"github.com/Fantom-foundation/lachesis-base/kvdb/memorydb"
	"go.uber.org/mock/gomock"
)

func TestChooseParents_NoParentsForGenesisEvent(t *testing.T) {
	ctrl := gomock.NewController(t)
	external := NewMockExternal(ctrl)
	em := NewEmitter(
		DefaultConfig(),
		World{External: external},
		fixedPriceBaseFeeSource{},
		nil,
	)

	epoch := idx.Epoch(1)
	validatorId := idx.ValidatorID(1)

	external.EXPECT().GetLastEvent(epoch, validatorId)

	selfParent, parents, ok := em.chooseParents(epoch, validatorId)
	if selfParent != nil {
		t.Error("genesis event must not have self parent")
	}
	if len(parents) > 0 {
		t.Error("genesis event must not have any parents")
	}
	if !ok {
		t.Error("genesis parent assignment must always succeed")
	}
}

func TestChooseParents_NonGenesisEventMustHaveOneSelfParent(t *testing.T) {
	ctrl := gomock.NewController(t)
	external := NewMockExternal(ctrl)
	em := NewEmitter(
		DefaultConfig(),
		World{External: external},
		fixedPriceBaseFeeSource{},
		nil,
	)
	em.maxParents = 3
	em.payloadIndexer = ancestor.NewPayloadIndexer(3)

	epoch := idx.Epoch(1)
	validatorId := idx.ValidatorID(1)

	validatorIndex := vecmt.NewIndex(nil, vecmt.LiteConfig())
	validatorIndex.Reset(pos.ArrayToValidators(
		[]idx.ValidatorID{1, 2},
		[]pos.Weight{1, 1},
	), memorydb.New(), nil)

	selfParentHash := hash.Event{1}

	external.EXPECT().GetLastEvent(epoch, validatorId).Return(&selfParentHash)
	external.EXPECT().GetHeads(epoch).Return(hash.Events{{2}, {3}})
	external.EXPECT().DagIndex().Return(validatorIndex)

	selfParent, parents, ok := em.chooseParents(epoch, validatorId)
	if selfParent == nil {
		t.Error("non-genesis event must have a self parent")
	}
	// strategies sometimes choose the same parent multiple times, test for minimal amount (1 self parent + 1 random/metric)
	if wantMin, got := 2, len(parents); got < wantMin {
		t.Errorf("incorrect number of event parents, expected at least: %d, got: %d", wantMin, got)
	}
	if !ok {
		t.Error("parent assignment must succeed when no cheating is detected")
	}
}
