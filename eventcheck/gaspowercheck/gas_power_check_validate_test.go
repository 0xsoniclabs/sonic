package gaspowercheck

import (
	"testing"
	"time"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
	"go.uber.org/mock/gomock"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/iblockproc"
)

func makeConfig() [inter.GasPowerConfigs]Config {
	return [inter.GasPowerConfigs]Config{
		{
			Idx:                inter.ShortTermGas,
			AllocPerSec:        1000000,
			MaxAllocPeriod:     inter.Timestamp(10 * time.Second),
			MinEnsuredAlloc:    1000,
			StartupAllocPeriod: inter.Timestamp(5 * time.Second),
			MinStartupGas:      500,
		},
		{
			Idx:                inter.LongTermGas,
			AllocPerSec:        2000000,
			MaxAllocPeriod:     inter.Timestamp(20 * time.Second),
			MinEnsuredAlloc:    2000,
			StartupAllocPeriod: inter.Timestamp(10 * time.Second),
			MinStartupGas:      1000,
		},
	}
}

func makeValidators() *pos.Validators {
	builder := pos.NewBuilder()
	builder.Set(idx.ValidatorID(1), 100)
	return builder.Build()
}

func makeValidationContext(epoch idx.Epoch) *ValidationContext {
	validators := makeValidators()
	return &ValidationContext{
		Epoch:      epoch,
		Configs:    makeConfig(),
		EpochStart: inter.Timestamp(1000 * time.Second),
		Validators: validators,
		ValidatorStates: []ValidatorState{
			{PrevEpochEvent: iblockproc.EventInfo{}, GasRefund: 0},
		},
	}
}

func TestCalcGasPower_WrongEpoch(t *testing.T) {
	ctrl := gomock.NewController(t)
	reader := NewMockReader(ctrl)

	ctx := makeValidationContext(5)
	reader.EXPECT().GetValidationContext().Return(ctx).AnyTimes()

	c := New(reader)

	// Event with wrong epoch.
	me := &inter.MutableEventPayload{}
	me.SetEpoch(10) // doesn't match ctx epoch 5
	me.SetCreator(1)
	e := me.Build()

	_, err := c.CalcGasPower(e, nil)
	if err == nil {
		t.Fatal("expected error for wrong epoch")
	}
}

func TestCalcGasPower_FirstEvent(t *testing.T) {
	ctrl := gomock.NewController(t)
	reader := NewMockReader(ctrl)

	ctx := makeValidationContext(1)
	reader.EXPECT().GetValidationContext().Return(ctx).AnyTimes()

	c := New(reader)

	me := &inter.MutableEventPayload{}
	me.SetEpoch(1)
	me.SetCreator(1)
	me.SetMedianTime(inter.Timestamp(1005 * time.Second)) // 5 seconds after epoch start
	e := me.Build()

	gasPower, err := c.CalcGasPower(e, nil)
	if err != nil {
		t.Fatalf("CalcGasPower failed: %v", err)
	}

	// For a first event, gas power should be at least the startup allocation.
	for i := range gasPower.Gas {
		if gasPower.Gas[i] == 0 {
			t.Errorf("expected non-zero gas power for config %d", i)
		}
	}
}

func TestCalcGasPower_WithSelfParent(t *testing.T) {
	ctrl := gomock.NewController(t)
	reader := NewMockReader(ctrl)

	ctx := makeValidationContext(1)
	reader.EXPECT().GetValidationContext().Return(ctx).AnyTimes()

	c := New(reader)

	// Self-parent event.
	parentMe := &inter.MutableEventPayload{}
	parentMe.SetEpoch(1)
	parentMe.SetCreator(1)
	parentMe.SetSeq(1)
	parentMe.SetMedianTime(inter.Timestamp(1000 * time.Second))
	parentMe.SetGasPowerLeft(inter.GasPowerLeft{Gas: [2]uint64{5000, 10000}})
	parent := parentMe.Build()

	// Child event with self-parent.
	selfParentHash := parent.ID()
	childMe := &inter.MutableEventPayload{}
	childMe.SetEpoch(1)
	childMe.SetCreator(1)
	childMe.SetSeq(2)
	childMe.SetParents(hash.Events{selfParentHash})
	childMe.SetMedianTime(inter.Timestamp(1002 * time.Second)) // 2 seconds later
	child := childMe.Build()

	gasPower, err := c.CalcGasPower(child, parent)
	if err != nil {
		t.Fatalf("CalcGasPower failed: %v", err)
	}

	// Should have gas from parent plus allocation for 2 seconds.
	for i := range gasPower.Gas {
		if gasPower.Gas[i] < 5000 {
			t.Errorf("config %d: expected gas power >= parent's left, got %d", i, gasPower.Gas[i])
		}
	}
}

func TestValidate_Matching(t *testing.T) {
	ctrl := gomock.NewController(t)
	reader := NewMockReader(ctrl)

	ctx := makeValidationContext(1)
	reader.EXPECT().GetValidationContext().Return(ctx).AnyTimes()

	c := New(reader)

	// First calculate what the gas power should be.
	me := &inter.MutableEventPayload{}
	me.SetEpoch(1)
	me.SetCreator(1)
	me.SetMedianTime(inter.Timestamp(1005 * time.Second))
	e := me.Build()

	gasPower, err := c.CalcGasPower(e, nil)
	if err != nil {
		t.Fatalf("CalcGasPower failed: %v", err)
	}

	// Now build an event with matching GasPowerLeft (gasPower - GasPowerUsed).
	me2 := &inter.MutableEventPayload{}
	me2.SetEpoch(1)
	me2.SetCreator(1)
	me2.SetMedianTime(inter.Timestamp(1005 * time.Second))
	me2.SetGasPowerUsed(100)
	me2.SetGasPowerLeft(inter.GasPowerLeft{Gas: [2]uint64{
		gasPower.Gas[0] - 100,
		gasPower.Gas[1] - 100,
	}})
	e2 := me2.Build()

	err = c.Validate(e2, nil)
	if err != nil {
		t.Fatalf("Validate should pass for correct gas power: %v", err)
	}
}

func TestValidate_Mismatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	reader := NewMockReader(ctrl)

	ctx := makeValidationContext(1)
	reader.EXPECT().GetValidationContext().Return(ctx).AnyTimes()

	c := New(reader)

	me := &inter.MutableEventPayload{}
	me.SetEpoch(1)
	me.SetCreator(1)
	me.SetMedianTime(inter.Timestamp(1005 * time.Second))
	me.SetGasPowerUsed(100)
	me.SetGasPowerLeft(inter.GasPowerLeft{Gas: [2]uint64{999999, 999999}}) // wrong
	e := me.Build()

	err := c.Validate(e, nil)
	if err != ErrWrongGasPowerLeft {
		t.Fatalf("expected ErrWrongGasPowerLeft, got %v", err)
	}
}

func TestCalcValidatorGasPower_WithStartup(t *testing.T) {
	ctrl := gomock.NewController(t)
	reader := NewMockReader(ctrl)

	ctx := makeValidationContext(1)
	reader.EXPECT().GetValidationContext().Return(ctx).AnyTimes()

	c := New(reader)

	me := &inter.MutableEventPayload{}
	me.SetEpoch(1)
	me.SetCreator(1)
	me.SetMedianTime(ctx.EpochStart) // same time as epoch start
	e := me.Build()

	gasPower, err := c.CalcGasPower(e, nil)
	if err != nil {
		t.Fatalf("CalcGasPower failed: %v", err)
	}

	// For a first event with zero elapsed time, should get at least startup gas.
	for i, cfg := range ctx.Configs {
		if gasPower.Gas[i] < cfg.MinStartupGas {
			t.Errorf("config %d: gas power %d < MinStartupGas %d", i, gasPower.Gas[i], cfg.MinStartupGas)
		}
	}
}

func TestCalcValidatorGasPowerPerSec_MultipleValidators(t *testing.T) {
	builder := pos.NewBuilder()
	builder.Set(idx.ValidatorID(1), 75)
	builder.Set(idx.ValidatorID(2), 25)
	validators := builder.Build()

	config := Config{
		Idx:                0,
		AllocPerSec:        1000000,
		MaxAllocPeriod:     inter.Timestamp(10 * time.Second),
		MinEnsuredAlloc:    100,
		StartupAllocPeriod: inter.Timestamp(5 * time.Second),
		MinStartupGas:      50,
	}

	perSec1, _, _ := CalcValidatorGasPowerPerSec(idx.ValidatorID(1), validators, config)
	perSec2, _, _ := CalcValidatorGasPowerPerSec(idx.ValidatorID(2), validators, config)

	// Validator 1 has 75% stake, so should get ~3x the gas of validator 2 (25%).
	if perSec1 <= perSec2 {
		t.Errorf("validator with more stake should get more gas: %d vs %d", perSec1, perSec2)
	}
	ratio := float64(perSec1) / float64(perSec2)
	if ratio < 2.5 || ratio > 3.5 {
		t.Errorf("expected ratio ~3.0, got %.2f", ratio)
	}
}
