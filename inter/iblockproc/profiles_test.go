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
