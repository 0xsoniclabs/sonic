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

package tests

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

// ToHexUint64 converts a uint64 to a hexutil.Uint64.
func ToHexUint64(i uint64) *hexutil.Uint64 {
	hu := hexutil.Uint64(i)
	return &hu
}

// ToHexUint converts a uint to a hexutil.Uint.
func ToHexUint(i uint) *hexutil.Uint {
	hu := hexutil.Uint(i)
	return &hu
}

// ToHexBigInt converts a big.Int to a hexutil.Big.
func ToHexBigInt(i *big.Int) *hexutil.Big {
	if i == nil {
		return nil
	}
	hu := hexutil.Big(*i)
	return &hu
}

// ToHexBytes converts a byte slice to a hexutil.Bytes.
func ToHexBytes(b []byte) *hexutil.Bytes {
	hb := hexutil.Bytes(b)
	return &hb
}
