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

package dispatch_test

import (
	"regexp"
	"testing"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/stretchr/testify/require"
)

func TestNewID_IsHexWithoutLeadingZerosAndUnique(t *testing.T) {
	seen := map[dispatch.ID]bool{}
	for range 1000 {
		id := dispatch.NewID()
		require.Regexp(t, regexp.MustCompile(`^0x([1-9a-f][0-9a-f]{0,31}|0)$`), string(id))
		require.False(t, seen[id])
		seen[id] = true
	}
}
