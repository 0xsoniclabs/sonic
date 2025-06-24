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

package result

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResult_New_ProducesASuccessfulResult(t *testing.T) {
	res := New[int](42)
	require.False(t, res.IsError())
	v, err := res.Unwrap()
	require.Equal(t, 42, v)
	require.Nil(t, err)
}

func TestResult_Error_ProducesAFailedResult(t *testing.T) {
	require.True(t, Error[int](fmt.Errorf("fail")).IsError())
}

func TestResult_Error_NilResultsInZeroValue(t *testing.T) {
	res := Error[int](nil)
	require.False(t, res.IsError())
	v, err := res.Unwrap()
	require.Zero(t, v)
	require.Nil(t, err)
}
