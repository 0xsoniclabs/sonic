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

package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCartesianProduct_CountInstantiations(t *testing.T) {

	count := func(_ int, modifier []int) int {
		var count int
		for _, v := range modifier {
			count += v
		}
		return count
	}

	countInstances := func(pieces [][]int, modifier func(int, []int) int) int {
		var count int
		makeZero := func() int { return 0 }
		for range GenerateTestDataBasedOnModificationCombinations(makeZero, pieces, modifier) {
			count++
		}
		return count
	}

	assert.Equal(t, 1, countInstances(nil, count))
	assert.Equal(t, 1, countInstances([][]int{{1}}, count))
	assert.Equal(t, 2, countInstances([][]int{{1}, {1, 2}}, count))
	assert.Equal(t, 4, countInstances([][]int{{1, 2}, {1, 2}}, count))
	assert.Equal(t, 4, countInstances([][]int{{1}, {1, 2}, {1, 2}}, count))
	assert.Equal(t, 6, countInstances([][]int{{1}, {1, 2, 3}, {1, 2}}, count))
	assert.Equal(t, 6, countInstances([][]int{{1}, {1, 2}, {1, 2, 3}}, count))
}

func TestCartesianProduct_noPiecesReturnOriginalObject(t *testing.T) {
	makeOriginal := func() int { return 36 }
	it := GenerateTestDataBasedOnModificationCombinations(
		makeOriginal,
		nil,
		func(v int, pieces []int) int {
			require.Len(t, pieces, 0)
			return v
		},
	)

	versions := make([]int, 0)
	for i := range it {
		versions = append(versions, i)
	}

	require.Equal(t, 1, len(versions))
	require.Equal(t, makeOriginal(), versions[0])
}

func TestCartesianProduct_AcceptsFunctionsAsPieces(t *testing.T) {

	type TestType struct {
		A int
		B int
	}
	type modFunc func(t *TestType)

	setA := func(a int) modFunc {
		return func(t *TestType) {
			t.A = a
		}
	}
	setB := func(a int) modFunc {
		return func(t *TestType) {
			t.B = a
		}
	}

	instances := make([]TestType, 0)
	for v := range GenerateTestDataBasedOnModificationCombinations(
		func() TestType { return TestType{} },
		[][]modFunc{
			{setA(1), setA(2)},
			{setB(1), setB(2)},
		},
		func(t TestType, modifiers []modFunc) TestType {
			for _, m := range modifiers {
				m(&t)
			}
			return t
		}) {
		instances = append(instances, v)
	}

	assert.Len(t, instances, 4)
	assert.Contains(t, instances, TestType{1, 1})
	assert.Contains(t, instances, TestType{1, 2})
	assert.Contains(t, instances, TestType{2, 1})
	assert.Contains(t, instances, TestType{2, 2})
}
