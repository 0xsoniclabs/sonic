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

package bundle

import (
	"testing"
	"unsafe"

	"github.com/stretchr/testify/require"
)

func TestExecutionFlags_Valid_AllCombinationsPass(t *testing.T) {
	// explicit combination tests
	tests := []ExecutionFlags{
		0,
		EF_TolerateInvalid,
		EF_TolerateFailed,
		EF_TolerateInvalid | EF_TolerateFailed,
	}

	for _, flags := range tests {
		require.True(t, flags.Valid())
	}
}

func TestExecutionFlags_Valid_OutOfRangeFails(t *testing.T) {
	require.EqualValues(t, 1, unsafe.Sizeof(ExecutionFlags(0)))
	for f := range 256 {
		flags := ExecutionFlags(f)
		require.Equal(t, flags.Valid(), f < (1<<numUsedBits))
	}
}

func TestExecutionFlags_TolerateInvalid(t *testing.T) {
	require := require.New(t)

	flags := []ExecutionFlags{
		EF_Default,
		EF_Default | EF_TolerateFailed,
	}

	for _, flags := range flags {
		require.False(flags.TolerateInvalid())
		require.True((flags | EF_TolerateInvalid).TolerateInvalid())
	}
}

func TestExecutionFlags_TolerateFailed(t *testing.T) {
	require := require.New(t)

	flags := []ExecutionFlags{
		EF_Default,
		EF_Default | EF_TolerateInvalid,
	}

	for _, flags := range flags {
		require.False(flags.TolerateFailed())
		require.True((flags | EF_TolerateFailed).TolerateFailed())
	}
}
