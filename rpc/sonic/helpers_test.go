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

package sonic

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func expectJsonEqual[T any](t testing.TB, expected string, value T) {
	t.Helper()
	encoded, err := json.Marshal(value)
	require.NoError(t, err, "failed to marshal BundleRPCInfo to JSON")

	var j1, j2 T
	err = json.Unmarshal(encoded, &j1)
	require.NoError(t, err, "failed to unmarshal JSON back to %T", value)
	err = json.Unmarshal([]byte(expected), &j2)
	require.NoError(t, err, "failed to unmarshal JSON back to %T", value)
	v := reflect.DeepEqual(j1, j2)
	if !v {
		expected = strings.ReplaceAll(expected, " ", "")
		expected = strings.ReplaceAll(expected, "\n", "")
		expected = strings.ReplaceAll(expected, "\t", "")
		t.Logf("Expected JSON: %s", expected)
		t.Logf("Actual JSON:   %s", string(encoded))
		t.FailNow()
	}
}
