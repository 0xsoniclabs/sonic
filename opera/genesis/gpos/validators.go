package gpos

import (
	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/ethereum/go-ethereum/common"

	"github.com/0xsoniclabs/sonic/inter/validatorpk"
)

type (
	// Validator is a helper structure to define genesis validators
	Validator struct {
		ID               consensus.ValidatorID
		Address          common.Address
		PubKey           validatorpk.PubKey
		CreationTime     inter.Timestamp
		CreationEpoch    consensus.Epoch
		DeactivatedTime  inter.Timestamp
		DeactivatedEpoch consensus.Epoch
		Status           uint64
	}

	Validators []Validator
)

// Map converts Validators to map
func (gv Validators) Map() map[consensus.ValidatorID]Validator {
	validators := map[consensus.ValidatorID]Validator{}
	for _, val := range gv {
		validators[val.ID] = val
	}
	return validators
}
