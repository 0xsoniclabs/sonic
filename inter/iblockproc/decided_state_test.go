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

package iblockproc

import (
	"math/big"
	"testing"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
	"github.com/Fantom-foundation/lachesis-base/lachesis"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/drivertype"
	"github.com/0xsoniclabs/sonic/inter/validatorpk"
	"github.com/0xsoniclabs/sonic/opera"
)

func makeTestValidators() *pos.Validators {
	builder := pos.NewBuilder()
	builder.Set(idx.ValidatorID(1), 100)
	return builder.Build()
}

func makeTestProfiles() ValidatorProfiles {
	return ValidatorProfiles{
		idx.ValidatorID(1): drivertype.Validator{
			Weight: big.NewInt(100),
			PubKey: validatorpk.PubKey{
				Type: validatorpk.Types.Secp256k1,
				Raw:  make([]byte, 33),
			},
		},
	}
}

func makeTestBlockState() BlockState {
	return BlockState{
		LastBlock: BlockCtx{
			Idx:     1,
			Time:    inter.Timestamp(1000),
			Atropos: hash.ZeroEvent,
		},
		FinalizedStateRoot: hash.Hash{},
		EpochGas:           100,
		EpochCheaters:      lachesis.Cheaters{},
		ValidatorStates: []ValidatorBlockState{
			{
				Originated: big.NewInt(50),
			},
		},
		NextValidatorProfiles: makeTestProfiles(),
	}
}

func makeTestEpochState() EpochState {
	return EpochState{
		Epoch:             1,
		EpochStart:        inter.Timestamp(1000),
		PrevEpochStart:    inter.Timestamp(500),
		Validators:        makeTestValidators(),
		ValidatorStates:   []ValidatorEpochState{{}},
		ValidatorProfiles: makeTestProfiles(),
		Rules:             opera.FakeNetRules(opera.Upgrades{London: true}),
	}
}

func TestBlockState_Copy(t *testing.T) {
	bs := makeTestBlockState()
	cp := bs.Copy()

	// Modify original
	bs.EpochGas = 999
	bs.ValidatorStates[0].Originated.SetInt64(999)

	// Copy should be unaffected
	if cp.EpochGas == 999 {
		t.Fatal("Copy should be independent - EpochGas was shared")
	}
	if cp.ValidatorStates[0].Originated.Int64() == 999 {
		t.Fatal("Copy should deep-copy Originated")
	}
}

func TestBlockState_Copy_WithDirtyRules(t *testing.T) {
	bs := makeTestBlockState()
	rules := opera.FakeNetRules(opera.Upgrades{London: true})
	bs.DirtyRules = &rules

	cp := bs.Copy()
	if cp.DirtyRules == nil {
		t.Fatal("expected DirtyRules to be copied")
	}
}

func TestBlockState_Copy_NilDirtyRules(t *testing.T) {
	bs := makeTestBlockState()
	bs.DirtyRules = nil

	cp := bs.Copy()
	if cp.DirtyRules != nil {
		t.Fatal("expected nil DirtyRules in copy")
	}
}

func TestBlockState_GetValidatorState(t *testing.T) {
	bs := makeTestBlockState()
	validators := makeTestValidators()

	vs := bs.GetValidatorState(idx.ValidatorID(1), validators)
	if vs == nil {
		t.Fatal("expected non-nil validator state")
	}
	if vs.Originated.Int64() != 50 {
		t.Fatalf("expected Originated 50, got %d", vs.Originated.Int64())
	}
}

func TestBlockState_Hash(t *testing.T) {
	bs := makeTestBlockState()
	h := bs.Hash()
	if h == (hash.Hash{}) {
		t.Fatal("expected non-zero hash")
	}

	// Same state should produce same hash
	h2 := bs.Hash()
	if h != h2 {
		t.Fatal("expected deterministic hash")
	}

	// Different state should produce different hash
	bs.EpochGas = 999
	h3 := bs.Hash()
	if h == h3 {
		t.Fatal("different state should produce different hash")
	}
}

func TestEpochState_Duration(t *testing.T) {
	es := makeTestEpochState()
	d := es.Duration()
	expected := es.EpochStart - es.PrevEpochStart
	if d != expected {
		t.Fatalf("expected duration %d, got %d", expected, d)
	}
}

func TestEpochState_GetValidatorState(t *testing.T) {
	es := makeTestEpochState()
	vs := es.GetValidatorState(idx.ValidatorID(1), es.Validators)
	if vs == nil {
		t.Fatal("expected non-nil validator epoch state")
	}
}

func TestEpochState_Hash_London(t *testing.T) {
	es := makeTestEpochState()
	es.Rules.Upgrades.London = true

	h := es.Hash()
	if h == (hash.Hash{}) {
		t.Fatal("expected non-zero hash")
	}

	h2 := es.Hash()
	if h != h2 {
		t.Fatal("expected deterministic hash")
	}
}

func TestEpochState_Hash_PreLondon(t *testing.T) {
	es := makeTestEpochState()
	es.Rules.Upgrades.London = false

	h := es.Hash()
	if h == (hash.Hash{}) {
		t.Fatal("expected non-zero hash")
	}
}

func TestEpochState_Copy(t *testing.T) {
	es := makeTestEpochState()
	cp := es.Copy()

	// Modify original
	es.Epoch = 999

	// Copy should be unaffected
	if cp.Epoch == 999 {
		t.Fatal("Copy should be independent")
	}
}

func TestEpochState_Copy_WithProfiles(t *testing.T) {
	es := makeTestEpochState()
	cp := es.Copy()

	// Modify profile in original
	p := es.ValidatorProfiles[idx.ValidatorID(1)]
	p.Weight.SetInt64(999)
	es.ValidatorProfiles[idx.ValidatorID(1)] = p

	// Copy should be unaffected
	cpWeight := cp.ValidatorProfiles[idx.ValidatorID(1)].Weight
	if cpWeight.Int64() == 999 {
		t.Fatal("Copy should deep-copy ValidatorProfiles")
	}
}
