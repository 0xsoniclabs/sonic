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

// ToFtm number of FTM to Wei
func ToFtm(ftm uint64) *big.Int {
	return new(big.Int).Mul(new(big.Int).SetUint64(ftm), big.NewInt(1e18))
}

// ToFtmU256 number of FTM to Wei using the uint256 type
func ToFtmU256(ftm uint64) *uint256.Int {
	return BigIntToUint256(ToFtm(ftm))
}
