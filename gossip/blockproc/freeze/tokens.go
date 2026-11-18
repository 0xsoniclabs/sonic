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

package freeze

import (
	"bytes"
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	transferSelector     = crypto.Keccak256([]byte("transfer(address,uint256)"))[:4]
	transferFromSelector = crypto.Keccak256([]byte("transferFrom(address,address,uint256)"))[:4]
)

// isFrozenTokenCall reports whether input is an ERC-20 transfer to, or
// transferFrom from or to, a frozen account, sent to a token that is not
// opted out. Tokens moved by other functions, such as mints, burnFrom or
// EIP-3009 transferWithAuthorization, are not checked.
func isFrozenTokenCall(db StateReader, token common.Address, input []byte) bool {
	var accounts []byte // the address arguments
	switch {
	case bytes.HasPrefix(input, transferSelector) && len(input) >= 4+32:
		accounts = input[4 : 4+32] // to
	case bytes.HasPrefix(input, transferFromSelector) && len(input) >= 4+64:
		accounts = input[4 : 4+64] // from, to
	default:
		return false
	}
	for account := range slices.Chunk(accounts, 32) {
		if IsFrozen(db, common.BytesToAddress(account)) {
			return !IsOptedOut(db, token)
		}
	}
	return false
}
