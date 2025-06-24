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
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPrettyDuration_String(t *testing.T) {
	for _, testcase := range []struct {
		str string
		val time.Duration
	}{
		{"0s", 0},
		{"1ns", time.Nanosecond},
		{"1µs", time.Microsecond},
		{"1ms", time.Millisecond},
		{"1s", time.Second},
		{"1.000s", time.Second + time.Microsecond + time.Nanosecond},
		{"1.001s", time.Second + time.Millisecond + time.Microsecond + time.Nanosecond},
		{"1m1.001s", time.Minute + time.Second + time.Millisecond + time.Microsecond + time.Nanosecond},
		{"1h1m1.001s", time.Hour + time.Minute + time.Second + time.Millisecond + time.Microsecond + time.Nanosecond},
		{"1d1h1m", 24*time.Hour + time.Hour + time.Minute + time.Second + time.Millisecond + time.Microsecond + time.Nanosecond},
		{"1mo1d1h", 30*24*time.Hour + 24*time.Hour + time.Hour + time.Minute + time.Second + time.Millisecond + time.Microsecond + time.Nanosecond},
		{"26y4mo23d", time.Duration(9503.123456789 * 24 * float64(time.Hour))},
	} {
		require.Equal(t, testcase.str, PrettyDuration(testcase.val).String())
		if testcase.val > 0 {
			require.Equal(t, "-"+testcase.str, PrettyDuration(-testcase.val).String())
		}
	}
}
