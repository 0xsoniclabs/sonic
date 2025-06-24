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
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// PrettyDuration is a combination of common.PrettyDuration and common.PrettyAge
// It is a pretty printed version of a time.Duration value that rounds
// the values up to a single most significant unit,
// while showing the least significant part if duration isn't too large.
type PrettyDuration time.Duration

// ageUnits is a list of units the age pretty printing uses.
var ageUnits = []struct {
	Size   time.Duration
	Symbol string
}{
	{12 * 30 * 24 * time.Hour, "y"},
	{30 * 24 * time.Hour, "mo"},
	{24 * time.Hour, "d"},
	{time.Hour, "h"},
	{time.Minute, "m"},
}

// String implements the Stringer interface, allowing pretty printing of duration
// values rounded to the most significant time unit.
func (t PrettyDuration) String() string {
	// Calculate the time difference and handle the 0 cornercase
	diff := time.Duration(t)
	// Accumulate a precision of 3 components before returning
	result, prec := "", 0
	if diff < 0 {
		diff = -diff
		result = "-"
	}

	for _, unit := range ageUnits {
		if diff > unit.Size {
			result = fmt.Sprintf("%s%d%s", result, diff/unit.Size, unit.Symbol)
			diff %= unit.Size

			if prec += 1; prec >= 3 {
				break
			}
		}
	}
	if prec < 3 {
		return fmt.Sprintf("%s%s", result, common.PrettyDuration(diff).String())
	}
	return result
}
