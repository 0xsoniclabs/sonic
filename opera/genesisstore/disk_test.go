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

package genesisstore

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetLoggerName(t *testing.T) {
	tests := map[string]struct {
		input    string
		expected string
	}{
		// Malformed: dashes only — previously panicked with index out of range.
		"only dashes":     {input: "-", expected: "-"},
		"multiple dashes": {input: "---", expected: "---"},
		// Empty string — degenerate case, must not panic.
		"empty": {input: "", expected: ""},
		// Known unit name patterns.
		"brs0":  {input: "brs0", expected: "blocks unit 0"},
		"brs1":  {input: "brs1", expected: "blocks unit 1"},
		"brs-1": {input: "brs-1", expected: "blocks unit 1"},
		"ers0":  {input: "ers0", expected: "epochs unit 0"},
		"ers-2": {input: "ers-2", expected: "epochs unit 2"},
		"evm0":  {input: "evm0", expected: "EVM unit 0"},
		"evm-3": {input: "evm-3", expected: "EVM unit 3"},
		// Unit name without trailing digit — suffix "0" is appended internally.
		"brs no digit": {input: "brs", expected: "blocks unit 0"},
		"ers no digit": {input: "ers", expected: "epochs unit 0"},
		"evm no digit": {input: "evm", expected: "EVM unit 0"},
		// Unrecognised names are returned unchanged.
		"unknown":             {input: "unknown", expected: "unknown"},
		"unknown with dashes": {input: "foo-bar", expected: "foo-bar"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			require.NotPanics(t, func() {
				got := getLoggerName(tc.input)
				require.Equal(t, tc.expected, got)
			})
		})
	}
}
