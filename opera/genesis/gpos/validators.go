package gpos

import (
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/ethereum/go-ethereum/common"

	"github.com/0xsoniclabs/sonic/inter/validatorpk"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
)

type (
	// Validator is a helper structure to define genesis validators
	Validator struct {
		ID               idx.ValidatorID
		Address          common.Address
		PubKey           validatorpk.PubKey
		CreationTime     inter.Timestamp
		CreationEpoch    idx.Epoch
		DeactivatedTime  inter.Timestamp
		DeactivatedEpoch idx.Epoch
		Status           uint64
	}

	Validators []Validator
)

// Map converts Validators to map
func (gv Validators) Map() map[idx.ValidatorID]Validator {
	validators := map[idx.ValidatorID]Validator{}
	for _, val := range gv {
		validators[val.ID] = val
	}
	return validators
}
