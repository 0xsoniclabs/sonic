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

package internaltx

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func IsInternal(tx *types.Transaction) bool {
	v, r, _ := tx.RawSignatureValues()
	return v.Sign() == 0 && r.Sign() == 0
}

func InternalSender(tx *types.Transaction) common.Address {
	_, _, s := tx.RawSignatureValues()
	return common.BytesToAddress(s.Bytes())
}

func Sender(signer types.Signer, tx *types.Transaction) (common.Address, error) {
	if !IsInternal(tx) {
		return types.Sender(signer, tx)
	}
	return InternalSender(tx), nil
}
