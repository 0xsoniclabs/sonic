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

package topicsdb

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPosToBytes(t *testing.T) {
	require := require.New(t)

	for i := 0xff / 0x0f; i >= 0; i-- {
		expect := uint8(0x0f * i)
		bb := posToBytes(expect)
		got := bytesToPos(bb)

		require.Equal(expect, got)
	}
}
