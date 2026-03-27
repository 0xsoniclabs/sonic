package drivercall

import (
	"math/big"
	"testing"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/validatorpk"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/opera/genesis/gpos"
)

func TestSealEpochValidators(t *testing.T) {
	validators := []idx.ValidatorID{1, 2, 3}
	data := SealEpochValidators(validators)
	if len(data) == 0 {
		t.Fatal("expected non-empty data")
	}
	// Should start with 4-byte method selector
	if len(data) < 4 {
		t.Fatal("data too short for ABI-encoded call")
	}
}

func TestSealEpochValidators_Empty(t *testing.T) {
	data := SealEpochValidators(nil)
	if len(data) == 0 {
		t.Fatal("expected non-empty data even with empty validators")
	}
}

func TestSealEpoch(t *testing.T) {
	metrics := []ValidatorEpochMetric{
		{
			Missed:          opera.BlocksMissed{},
			Uptime:          inter.Timestamp(1000),
			OriginatedTxFee: big.NewInt(100),
		},
	}
	data := SealEpoch(metrics)
	if len(data) == 0 {
		t.Fatal("expected non-empty data")
	}
	if len(data) < 4 {
		t.Fatal("data too short for ABI-encoded call")
	}
}

func TestSetGenesisValidator(t *testing.T) {
	v := gpos.Validator{
		ID:      idx.ValidatorID(1),
		Address: common.HexToAddress("0x01"),
		PubKey: validatorpk.PubKey{
			Type: validatorpk.Types.Secp256k1,
			Raw:  make([]byte, 33),
		},
		CreationTime: inter.Timestamp(1000),
	}
	data := SetGenesisValidator(v)
	if len(data) == 0 {
		t.Fatal("expected non-empty data")
	}
}

func TestSetGenesisDelegation(t *testing.T) {
	d := Delegation{
		Address:     common.HexToAddress("0x01"),
		ValidatorID: idx.ValidatorID(1),
		Stake:       big.NewInt(1000),
	}
	data := SetGenesisDelegation(d)
	if len(data) == 0 {
		t.Fatal("expected non-empty data")
	}
}

func TestDeactivateValidator(t *testing.T) {
	data := DeactivateValidator(idx.ValidatorID(1), 128)
	if len(data) == 0 {
		t.Fatal("expected non-empty data")
	}
	if len(data) < 4 {
		t.Fatal("data too short for ABI-encoded call")
	}
}
