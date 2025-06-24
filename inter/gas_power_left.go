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

package inter

import "fmt"

const (
	ShortTermGas    = 0
	LongTermGas     = 1
	GasPowerConfigs = 2
)

// GasPowerLeft is long-term gas power left and short-term gas power left
type GasPowerLeft struct {
	Gas [GasPowerConfigs]uint64
}

// Add add to all gas power lefts
func (g GasPowerLeft) Add(diff uint64) {
	for i := range g.Gas {
		g.Gas[i] += diff
	}
}

// Min returns minimum within long-term gas power left and short-term gas power left
func (g GasPowerLeft) Min() uint64 {
	min := g.Gas[0]
	for _, gas := range g.Gas {
		if min > gas {
			min = gas
		}
	}
	return min
}

// Max returns maximum within long-term gas power left and short-term gas power left
func (g GasPowerLeft) Max() uint64 {
	max := g.Gas[0]
	for _, gas := range g.Gas {
		if max < gas {
			max = gas
		}
	}
	return max
}

// Sub subtracts from all gas power lefts
func (g GasPowerLeft) Sub(diff uint64) GasPowerLeft {
	cp := g
	for i := range cp.Gas {
		cp.Gas[i] -= diff
	}
	return cp
}

// String returns string representation.
func (g GasPowerLeft) String() string {
	return fmt.Sprintf("{short=%d, long=%d}", g.Gas[ShortTermGas], g.Gas[LongTermGas])
}
