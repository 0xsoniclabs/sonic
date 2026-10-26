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
	"math/big"
	"reflect"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

// expectJsonEqual checks that a value encodes to the expected JSON.
func expectJsonEqual[T any](t testing.TB, expected string, value T) {
	t.Helper()
	encoded, err := json.Marshal(value)
	require.NoError(t, err, "failed to marshal %T to JSON", value)

	var j1, j2 T
	require.NoError(t, json.Unmarshal(encoded, &j1), "failed to unmarshal JSON back to %T", value)
	require.NoError(t, json.Unmarshal([]byte(expected), &j2), "failed to unmarshal JSON back to %T", value)
	if !reflect.DeepEqual(j1, j2) {
		expected = strings.ReplaceAll(expected, " ", "")
		expected = strings.ReplaceAll(expected, "\n", "")
		expected = strings.ReplaceAll(expected, "\t", "")
		t.Errorf("expected JSON %s, got %s", expected, encoded)
	}
}

func hexUint64(v uint64) *hexutil.Uint64 {
	return (*hexutil.Uint64)(&v)
}

func hexBig(v *big.Int) *hexutil.Big {
	return (*hexutil.Big)(v)
}
