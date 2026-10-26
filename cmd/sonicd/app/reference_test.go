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

package app

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// The golden file holds the flags of the reference release, v2.2.5, as its
// --help lists them, but for the help flag of the command line library. A flag of the reference that is gone makes the command
// lines of its operators fail.
const referenceFlags = "testdata/reference_flags.txt"

// deprecatedFlags are flags of the reference that are still accepted but have
// no effect, each with its reason; none so far.
var deprecatedFlags = map[string]string{}

func TestFlags_KeepTheFlagsOfTheReference(t *testing.T) {
	golden, err := os.ReadFile(referenceFlags)
	require.NoError(t, err)
	app := initApp()
	var current []string
	for _, flag := range app.Flags {
		for _, name := range strings.Split(flag.GetName(), ",") {
			current = append(current, strings.TrimSpace(name))
		}
	}
	for _, name := range strings.Fields(string(golden)) {
		if _, deprecated := deprecatedFlags[name]; deprecated {
			continue
		}
		require.True(t, slices.Contains(current, name), "the flag --%s of the reference is gone", name)
	}
}
