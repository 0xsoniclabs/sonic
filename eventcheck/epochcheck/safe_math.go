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

package epochcheck

import (
	"math"

	gomath "github.com/ethereum/go-ethereum/common/math"
)

// safeAdd returns the sum of all arguments, or math.MaxUint64 if any addition overflows.
func safeAdd(vals ...uint64) uint64 {
	sum := uint64(0)
	for _, v := range vals {
		var overflow bool
		if sum, overflow = gomath.SafeAdd(sum, v); overflow {
			return math.MaxUint64
		}
	}
	return sum
}

// safeMul returns a*b, or math.MaxUint64 if the multiplication overflows.
func safeMul(a, b uint64) uint64 {
	product, overflow := gomath.SafeMul(a, b)
	if overflow {
		return math.MaxUint64
	}
	return product
}
