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

	"github.com/holiman/uint256"
)

func BigIntToUint256(value *big.Int) *uint256.Int {
	if value.Sign() < 0 {
		panic("unable to convert negative big.Int to uint256")
	}
	bytes := value.Bytes()
	if len(bytes) > 32 {
		panic("unable to convert big.Int exceeding 32 bytes to uint256")
	}
	return new(uint256.Int).SetBytes(bytes)
}

func Uint256ToBigInt(value *uint256.Int) *big.Int {
	return new(big.Int).SetBytes(value.Bytes())
}
