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

package gaspowercheck

import (
	"testing"
	"time"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"

	"github.com/0xsoniclabs/sonic/inter"
)

func TestNew(t *testing.T) {
	c := New(nil)
	if c == nil {
		t.Fatal("expected non-nil Checker")
	}
}

func TestCalcValidatorGasPowerPerSec_ZeroStake(t *testing.T) {
	builder := pos.NewBuilder()
	builder.Set(idx.ValidatorID(1), 100)
	validators := builder.Build()

	config := Config{
		Idx:                0,
		AllocPerSec:        1000,
		MaxAllocPeriod:     inter.Timestamp(10 * time.Second),
		MinEnsuredAlloc:    100,
		StartupAllocPeriod: inter.Timestamp(5 * time.Second),
		MinStartupGas:      50,
	}

	// Validator 2 doesn't exist in the validator set
	perSec, maxGas, startup := CalcValidatorGasPowerPerSec(idx.ValidatorID(2), validators, config)
	if perSec != 0 || maxGas != 0 || startup != 0 {
		t.Fatalf("expected all zeros for unknown validator, got perSec=%d maxGas=%d startup=%d", perSec, maxGas, startup)
	}
}

func TestCalcValidatorGasPowerPerSec_ValidStake(t *testing.T) {
	builder := pos.NewBuilder()
	builder.Set(idx.ValidatorID(1), 100)
	validators := builder.Build()

	config := Config{
		Idx:                0,
		AllocPerSec:        1000000,
		MaxAllocPeriod:     inter.Timestamp(10 * time.Second),
		MinEnsuredAlloc:    100,
		StartupAllocPeriod: inter.Timestamp(5 * time.Second),
		MinStartupGas:      50,
	}

	perSec, maxGas, startup := CalcValidatorGasPowerPerSec(idx.ValidatorID(1), validators, config)
	if perSec == 0 {
		t.Fatal("expected non-zero perSec for valid validator")
	}
	if maxGas == 0 {
		t.Fatal("expected non-zero maxGas")
	}
	if startup == 0 {
		t.Fatal("expected non-zero startup")
	}
}

func TestCalcValidatorGasPowerPerSec_MinEnsuredAlloc(t *testing.T) {
	builder := pos.NewBuilder()
	builder.Set(idx.ValidatorID(1), 1)
	validators := builder.Build()

	config := Config{
		Idx:                0,
		AllocPerSec:        1, // very low alloc
		MaxAllocPeriod:     inter.Timestamp(1 * time.Second),
		MinEnsuredAlloc:    1000, // high min ensured
		StartupAllocPeriod: inter.Timestamp(1 * time.Second),
		MinStartupGas:      500,
	}

	_, maxGas, startup := CalcValidatorGasPowerPerSec(idx.ValidatorID(1), validators, config)
	if maxGas < 1000 {
		t.Fatalf("expected maxGas >= MinEnsuredAlloc, got %d", maxGas)
	}
	if startup < 500 {
		t.Fatalf("expected startup >= MinStartupGas, got %d", startup)
	}
}

func TestErrWrongGasPowerLeft(t *testing.T) {
	if ErrWrongGasPowerLeft == nil {
		t.Fatal("ErrWrongGasPowerLeft should not be nil")
	}
}
