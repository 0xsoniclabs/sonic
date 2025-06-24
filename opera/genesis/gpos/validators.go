// Copyright 2025 Sonic Operations Ltd
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
