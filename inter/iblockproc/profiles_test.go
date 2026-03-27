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
	"bytes"
	"math/big"
	"testing"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/0xsoniclabs/sonic/inter/drivertype"
	"github.com/0xsoniclabs/sonic/inter/validatorpk"
)

func TestValidatorProfiles_Copy(t *testing.T) {
	vp := ValidatorProfiles{
		idx.ValidatorID(1): drivertype.Validator{
			Weight: big.NewInt(100),
			PubKey: validatorpk.PubKey{
				Type: validatorpk.Types.Secp256k1,
				Raw:  make([]byte, 33),
			},
		},
		idx.ValidatorID(2): drivertype.Validator{
			Weight: big.NewInt(200),
			PubKey: validatorpk.PubKey{
				Type: validatorpk.Types.Secp256k1,
				Raw:  make([]byte, 33),
			},
		},
	}

	cp := vp.Copy()

	// Modify original
	p := vp[idx.ValidatorID(1)]
	p.Weight.SetInt64(999)
	vp[idx.ValidatorID(1)] = p

	// Copy should be unaffected
	if cp[idx.ValidatorID(1)].Weight.Int64() == 999 {
		t.Fatal("Copy should deep-copy weights")
	}
	if len(cp) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(cp))
	}
}

func TestValidatorProfiles_Copy_Empty(t *testing.T) {
	vp := ValidatorProfiles{}
	cp := vp.Copy()
	if len(cp) != 0 {
		t.Fatalf("expected 0 profiles, got %d", len(cp))
	}
}

func TestValidatorProfiles_SortedArray(t *testing.T) {
	vp := ValidatorProfiles{
		idx.ValidatorID(3): drivertype.Validator{
			Weight: big.NewInt(300),
			PubKey: validatorpk.PubKey{
				Type: validatorpk.Types.Secp256k1,
				Raw:  make([]byte, 33),
			},
		},
		idx.ValidatorID(1): drivertype.Validator{
			Weight: big.NewInt(100),
			PubKey: validatorpk.PubKey{
				Type: validatorpk.Types.Secp256k1,
				Raw:  make([]byte, 33),
			},
		},
	}

	arr := vp.SortedArray()
	if len(arr) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(arr))
	}
}

func TestValidatorProfiles_RLP_RoundTrip(t *testing.T) {
	vp := ValidatorProfiles{
		idx.ValidatorID(1): drivertype.Validator{
			Weight: big.NewInt(100),
			PubKey: validatorpk.PubKey{
				Type: validatorpk.Types.Secp256k1,
				Raw:  make([]byte, 33),
			},
		},
	}

	var buf bytes.Buffer
	err := rlp.Encode(&buf, vp)
	if err != nil {
		t.Fatalf("failed to encode: %v", err)
	}

	var decoded ValidatorProfiles
	err = rlp.DecodeBytes(buf.Bytes(), &decoded)
	if err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if len(decoded) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(decoded))
	}
	if decoded[idx.ValidatorID(1)].Weight.Cmp(big.NewInt(100)) != 0 {
		t.Fatal("weight mismatch after RLP round-trip")
	}
}
