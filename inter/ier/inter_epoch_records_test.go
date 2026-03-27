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

package ier

import (
	"math/big"
	"testing"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"

	"github.com/0xsoniclabs/sonic/inter/drivertype"
	"github.com/0xsoniclabs/sonic/inter/iblockproc"
	"github.com/0xsoniclabs/sonic/inter/validatorpk"
	"github.com/0xsoniclabs/sonic/opera"
)

func makeTestRecord() LlrFullEpochRecord {
	validators := pos.NewBuilder()
	validators.Set(idx.ValidatorID(1), 100)

	profiles := iblockproc.ValidatorProfiles{
		idx.ValidatorID(1): drivertype.Validator{
			Weight: big.NewInt(100),
			PubKey: validatorpk.PubKey{
				Type: validatorpk.Types.Secp256k1,
				Raw:  make([]byte, 33),
			},
		},
	}

	return LlrFullEpochRecord{
		BlockState: iblockproc.BlockState{
			LastBlock: iblockproc.BlockCtx{Idx: 1},
			ValidatorStates: []iblockproc.ValidatorBlockState{
				{Originated: big.NewInt(0)},
			},
			NextValidatorProfiles: profiles,
		},
		EpochState: iblockproc.EpochState{
			Epoch:             1,
			Validators:        validators.Build(),
			ValidatorStates:   []iblockproc.ValidatorEpochState{{}},
			ValidatorProfiles: profiles,
			Rules:             opera.FakeNetRules(opera.Upgrades{London: true}),
		},
	}
}

func TestLlrFullEpochRecord_Hash(t *testing.T) {
	r := makeTestRecord()
	h := r.Hash()
	if h == (hash.Hash{}) {
		t.Fatal("expected non-zero hash")
	}

	// Deterministic
	h2 := r.Hash()
	if h != h2 {
		t.Fatal("expected deterministic hash")
	}
}

func TestLlrFullEpochRecord_Hash_DifferentRecords(t *testing.T) {
	r1 := makeTestRecord()
	r2 := makeTestRecord()
	r2.BlockState.EpochGas = 999

	h1 := r1.Hash()
	h2 := r2.Hash()
	if h1 == h2 {
		t.Fatal("different records should produce different hashes")
	}
}

func TestLlrIdxFullEpochRecord(t *testing.T) {
	r := LlrIdxFullEpochRecord{
		LlrFullEpochRecord: makeTestRecord(),
		Idx:                idx.Epoch(5),
	}
	if r.Idx != 5 {
		t.Fatalf("expected Idx 5, got %d", r.Idx)
	}
	h := r.Hash()
	if h == (hash.Hash{}) {
		t.Fatal("expected non-zero hash")
	}
}
