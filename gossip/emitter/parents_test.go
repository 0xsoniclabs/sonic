package emitter

import (
	"github.com/0xsoniclabs/consensus/dagindexer"
	"testing"

	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/kvdb/memorydb"
	"github.com/0xsoniclabs/sonic/emitter/ancestor"
	"github.com/0xsoniclabs/sonic/gossip/emitter/mock"
	"github.com/golang/mock/gomock"
)

func TestChooseParents_NoParentsForGenesisEvent(t *testing.T) {
	ctrl := gomock.NewController(t)
	external := mock.NewMockExternal(ctrl)
	em := NewEmitter(
		DefaultConfig(),
		World{External: external},
		fixedPriceBaseFeeSource{},
	)

	epoch := consensus.Epoch(1)
	validatorId := consensus.ValidatorID(1)

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
	external := mock.NewMockExternal(ctrl)
	em := NewEmitter(
		DefaultConfig(),
		World{External: external},
		fixedPriceBaseFeeSource{},
	)
	em.maxParents = 3
	em.payloadIndexer = ancestor.NewPayloadIndexer(3)

	epoch := consensus.Epoch(1)
	validatorId := consensus.ValidatorID(1)

	validatorIndex := dagindexer.NewIndex(nil, dagindexer.LiteConfig())
	validatorIndex.Reset(consensus.ArrayToValidators(
		[]consensus.ValidatorID{1, 2},
		[]consensus.Weight{1, 1},
	), memorydb.New(), nil)

	selfParentHash := consensus.EventHash{1}

	external.EXPECT().GetLastEvent(epoch, validatorId).Return(&selfParentHash)
	external.EXPECT().GetHeads(epoch).Return(consensus.EventHashes{{2}, {3}})
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
