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

package utils

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// BigTo256 converts big number to 32 bytes array
func BigTo256(b *big.Int) common.Hash {
	return common.BytesToHash(b.Bytes())
}

// U64to256 converts uint64 to 32 bytes array
func U64to256(u64 uint64) common.Hash {
	return BigTo256(new(big.Int).SetUint64(u64))
}

// U64toBig converts uint64 to big number
func U64toBig(u64 uint64) *big.Int {
	return new(big.Int).SetUint64(u64)
}
