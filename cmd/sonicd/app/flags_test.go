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
	"bufio"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// O4: every flag of the reference release keeps being accepted. The list was
// taken from config/flags of v2.2.4. A flag may change its effect only by being
// documented as deprecated; it is never removed within a release.
func TestFlags_EveryFlagOfTheReferenceReleaseIsStillAccepted(t *testing.T) {
	file, err := os.Open("testdata/flags_v2.2.4.txt")
	require.NoError(t, err)
	defer file.Close()

	accepted := map[string]bool{}
	for _, flag := range initApp().Flags {
		name, _, _ := strings.Cut(flag.GetName(), ",")
		accepted[name] = true
	}

	scanner := bufio.NewScanner(file)
	count := 0
	for scanner.Scan() {
		name := strings.TrimSpace(scanner.Text())
		if name == "" {
			continue
		}
		if name == "disable-txPool-validation" || name == "enable-test-only-api" {
			continue // registered by tests only
		}
		count++
		require.True(t, accepted[name], "flag --%s of v2.2.4 is not accepted any more", name)
	}
	require.NoError(t, scanner.Err())
	require.Greater(t, count, 70)
}
