// Copyright 2026 Sonic Operations Ltd
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

package eventmodule

import (
	"math/big"
	"testing"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
	"go.uber.org/mock/gomock"

	"github.com/0xsoniclabs/sonic/gossip/blockproc"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/iblockproc"
	"github.com/0xsoniclabs/sonic/opera"
)

func buildValidators(ids ...idx.ValidatorID) *pos.Validators {
	builder := pos.NewBuilder()
	for _, id := range ids {
		builder.Set(id, 1)
	}
	return builder.Build()
}

func makeBlockState(numValidators int) iblockproc.BlockState {
	states := make([]iblockproc.ValidatorBlockState, numValidators)
	for i := range states {
		states[i].Originated = new(big.Int)
	}
	return iblockproc.BlockState{
		ValidatorStates: states,
	}
}

func makeEpochState(validators *pos.Validators) iblockproc.EpochState {
	valStates := make([]iblockproc.ValidatorEpochState, validators.Len())
	es := iblockproc.EpochState{}
	es.Epoch = 1
	es.EpochStart = 100
	es.Validators = validators
	es.ValidatorStates = valStates
	es.Rules = opera.Rules{}
	return es
}

func TestNew(t *testing.T) {
	m := New()
	if m == nil {
		t.Fatal("New() returned nil")
	}
}

func TestModule_ImplementsInterface(t *testing.T) {
	var _ blockproc.ConfirmedEventsModule = (*ValidatorEventsModule)(nil)
}

func TestStart_ReturnsProcessor(t *testing.T) {
	m := New()
	vals := buildValidators(1, 2)
	bs := makeBlockState(int(vals.Len()))
	es := makeEpochState(vals)

	proc := m.Start(bs, es)
	if proc == nil {
		t.Fatal("Start returned nil")
	}
}

func TestProcessConfirmedEvent_AccumulatesGas(t *testing.T) {
	ctrl := gomock.NewController(t)

	v1 := idx.ValidatorID(1)
	vals := buildValidators(v1)
	bs := makeBlockState(int(vals.Len()))
	es := makeEpochState(vals)

	m := New()
	proc := m.Start(bs, es)

	// Create a mock event with gas power used.
	e := inter.NewMockEventI(ctrl)
	e.EXPECT().Creator().Return(v1).AnyTimes()
	e.EXPECT().Seq().Return(idx.Event(1)).AnyTimes()
	e.EXPECT().GasPowerUsed().Return(uint64(1000)).AnyTimes()
	e.EXPECT().MedianTime().Return(inter.Timestamp(200)).AnyTimes()
	e.EXPECT().GasPowerLeft().Return(inter.GasPowerLeft{}).AnyTimes()
	e.EXPECT().ID().Return(hash.FakeEvent()).AnyTimes()

	proc.ProcessConfirmedEvent(e)

	block := iblockproc.BlockCtx{Idx: 1, Time: 300}
	result := proc.Finalize(block, false)

	if result.EpochGas != 1000 {
		t.Errorf("expected EpochGas=1000, got %d", result.EpochGas)
	}
}

func TestProcessConfirmedEvent_HighestEventTracking(t *testing.T) {
	ctrl := gomock.NewController(t)

	v1 := idx.ValidatorID(1)
	vals := buildValidators(v1)
	bs := makeBlockState(int(vals.Len()))
	es := makeEpochState(vals)

	m := New()
	proc := m.Start(bs, es)

	// Process two events from the same validator; the higher seq should win.
	e1 := inter.NewMockEventI(ctrl)
	e1.EXPECT().Creator().Return(v1).AnyTimes()
	e1.EXPECT().Seq().Return(idx.Event(1)).AnyTimes()
	e1.EXPECT().GasPowerUsed().Return(uint64(500)).AnyTimes()
	e1.EXPECT().MedianTime().Return(inter.Timestamp(200)).AnyTimes()
	e1.EXPECT().GasPowerLeft().Return(inter.GasPowerLeft{}).AnyTimes()
	e1.EXPECT().ID().Return(hash.FakeEvent()).AnyTimes()

	e2 := inter.NewMockEventI(ctrl)
	e2.EXPECT().Creator().Return(v1).AnyTimes()
	e2.EXPECT().Seq().Return(idx.Event(5)).AnyTimes()
	e2.EXPECT().GasPowerUsed().Return(uint64(300)).AnyTimes()
	e2.EXPECT().MedianTime().Return(inter.Timestamp(250)).AnyTimes()
	e2.EXPECT().GasPowerLeft().Return(inter.GasPowerLeft{Gas: [2]uint64{100, 200}}).AnyTimes()
	e2.EXPECT().ID().Return(hash.FakeEvent()).AnyTimes()

	proc.ProcessConfirmedEvent(e1)
	proc.ProcessConfirmedEvent(e2)

	block := iblockproc.BlockCtx{Idx: 1, Time: 300}
	result := proc.Finalize(block, false)

	if result.EpochGas != 800 {
		t.Errorf("expected EpochGas=800, got %d", result.EpochGas)
	}

	// The validator state should reflect the highest event (e2).
	vState := result.ValidatorStates[vals.GetIdx(v1)]
	if vState.LastBlock != 1 {
		t.Errorf("expected LastBlock=1, got %d", vState.LastBlock)
	}
	if vState.LastOnlineTime != 250 {
		t.Errorf("expected LastOnlineTime=250, got %d", vState.LastOnlineTime)
	}
}

func TestFinalize_CheaterEventsNulled(t *testing.T) {
	ctrl := gomock.NewController(t)

	v1 := idx.ValidatorID(1)
	v2 := idx.ValidatorID(2)
	vals := buildValidators(v1, v2)
	bs := makeBlockState(int(vals.Len()))
	bs.EpochCheaters = []idx.ValidatorID{v1} // v1 is a cheater
	es := makeEpochState(vals)

	m := New()
	proc := m.Start(bs, es)

	// Process events from both validators.
	e1 := inter.NewMockEventI(ctrl)
	e1.EXPECT().Creator().Return(v1).AnyTimes()
	e1.EXPECT().Seq().Return(idx.Event(1)).AnyTimes()
	e1.EXPECT().GasPowerUsed().Return(uint64(100)).AnyTimes()
	e1.EXPECT().MedianTime().Return(inter.Timestamp(200)).AnyTimes()
	e1.EXPECT().GasPowerLeft().Return(inter.GasPowerLeft{}).AnyTimes()
	e1.EXPECT().ID().Return(hash.FakeEvent()).AnyTimes()

	e2 := inter.NewMockEventI(ctrl)
	e2.EXPECT().Creator().Return(v2).AnyTimes()
	e2.EXPECT().Seq().Return(idx.Event(1)).AnyTimes()
	e2.EXPECT().GasPowerUsed().Return(uint64(200)).AnyTimes()
	e2.EXPECT().MedianTime().Return(inter.Timestamp(200)).AnyTimes()
	e2.EXPECT().GasPowerLeft().Return(inter.GasPowerLeft{}).AnyTimes()
	e2.EXPECT().ID().Return(hash.FakeEvent()).AnyTimes()

	proc.ProcessConfirmedEvent(e1)
	proc.ProcessConfirmedEvent(e2)

	block := iblockproc.BlockCtx{Idx: 1, Time: 300}
	result := proc.Finalize(block, false)

	// Cheater v1 should have no last event set; v2 should.
	v1State := result.ValidatorStates[vals.GetIdx(v1)]
	if v1State.LastBlock != 0 {
		t.Errorf("cheater v1 should have LastBlock=0, got %d", v1State.LastBlock)
	}

	v2State := result.ValidatorStates[vals.GetIdx(v2)]
	if v2State.LastBlock != 1 {
		t.Errorf("v2 should have LastBlock=1, got %d", v2State.LastBlock)
	}
}

func TestFinalize_UptimeCalculation_Berlin(t *testing.T) {
	ctrl := gomock.NewController(t)

	v1 := idx.ValidatorID(1)
	vals := buildValidators(v1)
	bs := makeBlockState(int(vals.Len()))
	es := makeEpochState(vals)
	es.Rules.Upgrades.Berlin = true
	es.EpochStart = inter.Timestamp(100)
	// BlockMissedSlack must be >= block.Idx - info.LastBlock for uptime to be counted.
	es.Rules.Economy.BlockMissedSlack = 10

	m := New()
	proc := m.Start(bs, es)

	e := inter.NewMockEventI(ctrl)
	e.EXPECT().Creator().Return(v1).AnyTimes()
	e.EXPECT().Seq().Return(idx.Event(1)).AnyTimes()
	e.EXPECT().GasPowerUsed().Return(uint64(0)).AnyTimes()
	e.EXPECT().MedianTime().Return(inter.Timestamp(250)).AnyTimes()
	e.EXPECT().GasPowerLeft().Return(inter.GasPowerLeft{}).AnyTimes()
	e.EXPECT().ID().Return(hash.FakeEvent()).AnyTimes()

	proc.ProcessConfirmedEvent(e)

	block := iblockproc.BlockCtx{Idx: 1, Time: 300}
	result := proc.Finalize(block, false)

	vState := result.ValidatorStates[vals.GetIdx(v1)]
	// With Berlin upgrade, prevOnlineTime = max(LastOnlineTime=0, EpochStart=100) = 100.
	// Uptime = MedianTime(250) - prevOnlineTime(100) = 150.
	if vState.Uptime != 150 {
		t.Errorf("expected Uptime=150, got %d", vState.Uptime)
	}
}

func TestFinalize_BlockMissedSlack(t *testing.T) {
	ctrl := gomock.NewController(t)

	v1 := idx.ValidatorID(1)
	vals := buildValidators(v1)
	bs := makeBlockState(int(vals.Len()))
	es := makeEpochState(vals)
	es.Rules.Economy.BlockMissedSlack = 5

	// Set the validator's LastBlock to a high value so
	// block.Idx > info.LastBlock + BlockMissedSlack.
	bs.ValidatorStates[vals.GetIdx(v1)].LastBlock = 1
	bs.ValidatorStates[vals.GetIdx(v1)].Originated = new(big.Int)

	m := New()
	proc := m.Start(bs, es)

	e := inter.NewMockEventI(ctrl)
	e.EXPECT().Creator().Return(v1).AnyTimes()
	e.EXPECT().Seq().Return(idx.Event(1)).AnyTimes()
	e.EXPECT().GasPowerUsed().Return(uint64(0)).AnyTimes()
	e.EXPECT().MedianTime().Return(inter.Timestamp(200)).AnyTimes()
	e.EXPECT().GasPowerLeft().Return(inter.GasPowerLeft{}).AnyTimes()
	e.EXPECT().ID().Return(hash.FakeEvent()).AnyTimes()

	proc.ProcessConfirmedEvent(e)

	// Block far beyond LastBlock + BlockMissedSlack.
	block := iblockproc.BlockCtx{Idx: 100, Time: 300}
	result := proc.Finalize(block, false)

	vState := result.ValidatorStates[vals.GetIdx(v1)]
	// block.Idx(100) > info.LastBlock(1) + BlockMissedSlack(5), so uptime should NOT increase.
	if vState.Uptime != 0 {
		t.Errorf("expected Uptime=0 when block missed slack exceeded, got %d", vState.Uptime)
	}
	// But LastBlock and LastOnlineTime should still be updated.
	if vState.LastBlock != 100 {
		t.Errorf("expected LastBlock=100, got %d", vState.LastBlock)
	}
}

func TestMultipleValidators(t *testing.T) {
	ctrl := gomock.NewController(t)

	v1 := idx.ValidatorID(1)
	v2 := idx.ValidatorID(2)
	v3 := idx.ValidatorID(3)
	vals := buildValidators(v1, v2, v3)
	bs := makeBlockState(int(vals.Len()))
	es := makeEpochState(vals)

	m := New()
	proc := m.Start(bs, es)

	for _, vid := range []idx.ValidatorID{v1, v2, v3} {
		e := inter.NewMockEventI(ctrl)
		e.EXPECT().Creator().Return(vid).AnyTimes()
		e.EXPECT().Seq().Return(idx.Event(1)).AnyTimes()
		e.EXPECT().GasPowerUsed().Return(uint64(100)).AnyTimes()
		e.EXPECT().MedianTime().Return(inter.Timestamp(200)).AnyTimes()
		e.EXPECT().GasPowerLeft().Return(inter.GasPowerLeft{}).AnyTimes()
		e.EXPECT().ID().Return(hash.FakeEvent()).AnyTimes()
		proc.ProcessConfirmedEvent(e)
	}

	block := iblockproc.BlockCtx{Idx: 1, Time: 300}
	result := proc.Finalize(block, false)

	if result.EpochGas != 300 {
		t.Errorf("expected EpochGas=300, got %d", result.EpochGas)
	}

	for _, vid := range []idx.ValidatorID{v1, v2, v3} {
		vState := result.ValidatorStates[vals.GetIdx(vid)]
		if vState.LastBlock != 1 {
			t.Errorf("validator %d: expected LastBlock=1, got %d", vid, vState.LastBlock)
		}
	}
}
