package drivertype

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/inter/validatorpk"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
)

func TestDoublesignBit(t *testing.T) {
	if DoublesignBit != 128 {
		t.Fatalf("expected DoublesignBit == 128, got %d", DoublesignBit)
	}
}

func TestOkStatus(t *testing.T) {
	if OkStatus != 0 {
		t.Fatalf("expected OkStatus == 0, got %d", OkStatus)
	}
}

func TestValidator(t *testing.T) {
	v := Validator{
		Weight: big.NewInt(100),
		PubKey: validatorpk.PubKey{
			Type: validatorpk.Types.Secp256k1,
			Raw:  make([]byte, 33),
		},
	}
	if v.Weight.Cmp(big.NewInt(100)) != 0 {
		t.Fatal("unexpected weight")
	}
	if v.PubKey.Type != validatorpk.Types.Secp256k1 {
		t.Fatal("unexpected pubkey type")
	}
}

func TestValidatorAndID(t *testing.T) {
	v := ValidatorAndID{
		ValidatorID: idx.ValidatorID(1),
		Validator: Validator{
			Weight: big.NewInt(50),
			PubKey: validatorpk.PubKey{
				Type: validatorpk.Types.Secp256k1,
				Raw:  make([]byte, 33),
			},
		},
	}
	if v.ValidatorID != 1 {
		t.Fatal("unexpected validator ID")
	}
	if v.Validator.Weight.Cmp(big.NewInt(50)) != 0 {
		t.Fatal("unexpected weight")
	}
}
