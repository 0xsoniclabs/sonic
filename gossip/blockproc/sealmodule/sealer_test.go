package sealmodule

import (
	"math/big"
	"testing"
	"time"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/drivertype"
	"github.com/0xsoniclabs/sonic/inter/iblockproc"
	"github.com/0xsoniclabs/sonic/inter/validatorpk"
	"github.com/0xsoniclabs/sonic/opera"
)

func makeValidators() *pos.Validators {
	builder := pos.NewBuilder()
	builder.Set(idx.ValidatorID(1), 100)
	return builder.Build()
}

func makeProfiles() iblockproc.ValidatorProfiles {
	return iblockproc.ValidatorProfiles{
		idx.ValidatorID(1): drivertype.Validator{
			Weight: big.NewInt(100),
			PubKey: validatorpk.PubKey{
				Type: validatorpk.Types.Secp256k1,
				Raw:  make([]byte, 33),
			},
		},
	}
}

func makeBlockState() iblockproc.BlockState {
	return iblockproc.BlockState{
		LastBlock: iblockproc.BlockCtx{Idx: 1, Time: inter.Timestamp(1000)},
		ValidatorStates: []iblockproc.ValidatorBlockState{
			{Originated: big.NewInt(0)},
		},
		NextValidatorProfiles: makeProfiles(),
	}
}

func makeEpochState() iblockproc.EpochState {
	return iblockproc.EpochState{
		Epoch:             1,
		EpochStart:        inter.Timestamp(1000),
		PrevEpochStart:    inter.Timestamp(500),
		Validators:        makeValidators(),
		ValidatorStates:   []iblockproc.ValidatorEpochState{{}},
		ValidatorProfiles: makeProfiles(),
		Rules:             opera.FakeNetRules(opera.Upgrades{London: true}),
	}
}

func TestNew(t *testing.T) {
	m := New()
	if m == nil {
		t.Fatal("expected non-nil module")
	}
}

func TestStart(t *testing.T) {
	m := New()
	block := iblockproc.BlockCtx{Idx: 1, Time: inter.Timestamp(2000)}
	bs := makeBlockState()
	es := makeEpochState()

	sealer := m.Start(block, bs, es)
	if sealer == nil {
		t.Fatal("expected non-nil sealer")
	}
}

func TestEpochSealing_NotSealing(t *testing.T) {
	m := New()
	block := iblockproc.BlockCtx{Idx: 1, Time: inter.Timestamp(1001)}
	bs := makeBlockState()
	bs.EpochGas = 0
	bs.AdvanceEpochs = 0
	es := makeEpochState()

	sealer := m.Start(block, bs, es)
	if sealer.EpochSealing() {
		t.Fatal("should not be sealing with low gas and short duration")
	}
}

func TestEpochSealing_MaxGas(t *testing.T) {
	m := New()
	block := iblockproc.BlockCtx{Idx: 1, Time: inter.Timestamp(1001)}
	bs := makeBlockState()
	es := makeEpochState()
	bs.EpochGas = es.Rules.Epochs.MaxEpochGas // at max

	sealer := m.Start(block, bs, es)
	if !sealer.EpochSealing() {
		t.Fatal("should be sealing when gas exceeds max")
	}
}

func TestEpochSealing_MaxDuration(t *testing.T) {
	m := New()
	es := makeEpochState()
	block := iblockproc.BlockCtx{
		Idx:  1,
		Time: es.EpochStart + es.Rules.Epochs.MaxEpochDuration,
	}
	bs := makeBlockState()

	sealer := m.Start(block, bs, es)
	if !sealer.EpochSealing() {
		t.Fatal("should be sealing when duration exceeds max")
	}
}

func TestEpochSealing_AdvanceEpochs(t *testing.T) {
	m := New()
	block := iblockproc.BlockCtx{Idx: 1, Time: inter.Timestamp(1001)}
	bs := makeBlockState()
	bs.AdvanceEpochs = 1
	es := makeEpochState()

	sealer := m.Start(block, bs, es)
	if !sealer.EpochSealing() {
		t.Fatal("should be sealing when AdvanceEpochs > 0")
	}
}

func TestEpochSealing_Cheaters(t *testing.T) {
	m := New()
	block := iblockproc.BlockCtx{Idx: 1, Time: inter.Timestamp(1001)}
	bs := makeBlockState()
	bs.EpochCheaters = append(bs.EpochCheaters, idx.ValidatorID(99))
	es := makeEpochState()

	sealer := m.Start(block, bs, es)
	if !sealer.EpochSealing() {
		t.Fatal("should be sealing when there are cheaters")
	}
}

func TestUpdate(t *testing.T) {
	m := New()
	block := iblockproc.BlockCtx{Idx: 1, Time: inter.Timestamp(1001)}
	bs := makeBlockState()
	es := makeEpochState()

	sealer := m.Start(block, bs, es)

	bs2 := makeBlockState()
	bs2.EpochGas = 999
	es2 := makeEpochState()
	es2.Epoch = 2
	sealer.Update(bs2, es2)
	// Should not panic
}

func TestSealEpoch(t *testing.T) {
	m := New()
	blockTime := inter.Timestamp(2000 + uint64(time.Second))
	block := iblockproc.BlockCtx{Idx: 2, Time: blockTime}
	bs := makeBlockState()
	bs.AdvanceEpochs = 1
	es := makeEpochState()

	sealer := m.Start(block, bs, es)
	newBs, newEs := sealer.SealEpoch()

	if newEs.Epoch != 2 {
		t.Fatalf("expected new epoch 2, got %d", newEs.Epoch)
	}
	if newBs.EpochGas != 0 {
		t.Fatalf("expected EpochGas to be reset, got %d", newBs.EpochGas)
	}
	if newBs.AdvanceEpochs != 0 {
		t.Fatalf("expected AdvanceEpochs to be decremented, got %d", newBs.AdvanceEpochs)
	}
	if newEs.PrevEpochStart != es.EpochStart {
		t.Fatal("expected PrevEpochStart to be set to old EpochStart")
	}
	if newEs.EpochStart != blockTime {
		t.Fatal("expected EpochStart to be set to block time")
	}
	if len(newBs.EpochCheaters) != 0 {
		t.Fatal("expected EpochCheaters to be reset")
	}
}

func TestSealEpoch_WithDirtyRules(t *testing.T) {
	m := New()
	block := iblockproc.BlockCtx{Idx: 2, Time: inter.Timestamp(2000)}
	bs := makeBlockState()
	rules := opera.FakeNetRules(opera.Upgrades{London: true, Berlin: true})
	bs.DirtyRules = &rules
	es := makeEpochState()

	sealer := m.Start(block, bs, es)
	newBs, newEs := sealer.SealEpoch()

	if newBs.DirtyRules != nil {
		t.Fatal("expected DirtyRules to be nil after seal")
	}
	if !newEs.Rules.Upgrades.Berlin {
		t.Fatal("expected new rules to include Berlin upgrade")
	}
}
