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

import "math/big"

// BigMin returns the smallest of the provided big.Ints.
// None of the arguments must be nil. If no arguments
// are provided, nil is returned.
func BigMin(values ...*big.Int) *big.Int {
	if len(values) == 0 {
		return nil
	}
	res := values[0]
	for _, b := range values[1:] {
		if res.Cmp(b) > 0 {
			res = b
		}
	}
	return res
}

// BigMax returns the largest of the provided big.Ints.
// None of the arguments must be nil. If no arguments
// are provided, nil is returned.
func BigMax(values ...*big.Int) *big.Int {
	if len(values) == 0 {
		return nil
	}
	res := values[0]
	for _, b := range values[1:] {
		if res.Cmp(b) < 0 {
			res = b
		}
	}
	return res
}
