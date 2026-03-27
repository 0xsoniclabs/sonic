package gpos

import (
	"testing"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"

	"github.com/0xsoniclabs/sonic/inter/validatorpk"
)

func TestValidators_Map(t *testing.T) {
	vals := Validators{
		{
			ID:      idx.ValidatorID(1),
			Address: common.HexToAddress("0x01"),
			PubKey: validatorpk.PubKey{
				Type: validatorpk.Types.Secp256k1,
				Raw:  make([]byte, 33),
			},
		},
		{
			ID:      idx.ValidatorID(2),
			Address: common.HexToAddress("0x02"),
			PubKey: validatorpk.PubKey{
				Type: validatorpk.Types.Secp256k1,
				Raw:  make([]byte, 33),
			},
		},
	}

	m := vals.Map()
	if len(m) != 2 {
		t.Fatalf("expected 2 validators, got %d", len(m))
	}
	if m[idx.ValidatorID(1)].Address != common.HexToAddress("0x01") {
		t.Fatal("unexpected address for validator 1")
	}
	if m[idx.ValidatorID(2)].Address != common.HexToAddress("0x02") {
		t.Fatal("unexpected address for validator 2")
	}
}

func TestValidators_Map_Empty(t *testing.T) {
	vals := Validators{}
	m := vals.Map()
	if len(m) != 0 {
		t.Fatalf("expected 0 validators, got %d", len(m))
	}
}

func TestValidators_Map_DuplicateIDs(t *testing.T) {
	vals := Validators{
		{ID: idx.ValidatorID(1), Address: common.HexToAddress("0x01")},
		{ID: idx.ValidatorID(1), Address: common.HexToAddress("0x02")}, // duplicate
	}
	m := vals.Map()
	// Last one wins
	if len(m) != 1 {
		t.Fatalf("expected 1 validator (deduped), got %d", len(m))
	}
	if m[idx.ValidatorID(1)].Address != common.HexToAddress("0x02") {
		t.Fatal("expected last duplicate to win")
	}
}

func TestValidator_Fields(t *testing.T) {
	v := Validator{
		ID:               idx.ValidatorID(5),
		Address:          common.HexToAddress("0xaabb"),
		CreationEpoch:    idx.Epoch(10),
		DeactivatedEpoch: idx.Epoch(20),
		Status:           1,
	}
	if v.ID != 5 {
		t.Fatal("unexpected ID")
	}
	if v.CreationEpoch != 10 {
		t.Fatal("unexpected CreationEpoch")
	}
	if v.DeactivatedEpoch != 20 {
		t.Fatal("unexpected DeactivatedEpoch")
	}
	if v.Status != 1 {
		t.Fatal("unexpected Status")
	}
}
