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

package priorities

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPriority_IsPrioritized_ChecksIfLevelIsGreater0(t *testing.T) {
	var with9ByteLevel Priority
	with9ByteLevel.Level.SetUint64(math.MaxUint64)
	with9ByteLevel.Level.AddUint64(&with9ByteLevel.Level, 1)

	tests := map[string]struct {
		prio Priority
		want bool
	}{
		"zero level":         {Prio(0, 0, 0), false},
		"zero level, weight": {Prio(0, 5, 0), false},
		"non-zero level":     {Prio(1, 0, 0), true},
		"non-zero 9 byte":    {with9ByteLevel, true},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.want, tc.prio.IsPrioritized())
		})
	}
}

func TestPriority_Cmp(t *testing.T) {
	tests := map[string]struct {
		a, b Priority
		want int
	}{
		"higher level wins":                        {Prio(2, 0, 0), Prio(1, 1, 0), 1},
		"lower level loses":                        {Prio(1, 1, 0), Prio(2, 0, 0), -1},
		"equal non-zero level, higher weight wins": {Prio(1, 2, 0), Prio(1, 1, 0), 1},
		"equal non-zero level, lower weight loses": {Prio(1, 1, 0), Prio(1, 2, 0), -1},
		"equal non-zero level and weight":          {Prio(1, 1, 0), Prio(1, 1, 0), 0},
		"zero level ignores weight":                {Prio(0, 1, 0), Prio(0, 2, 0), 0},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.want, tc.a.Cmp(tc.b))
		})
	}
}

// Prio builds a Priority for tests. Level and weight are limited to uint64 and
// id to a single byte for simplicity.
func Prio(level uint64, weight uint64, id byte) Priority {
	p := Priority{}
	p.Level.SetUint64(level)
	p.Weight.SetUint64(weight)
	p.ID[31] = id
	return p
}
