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

package opera

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetAllHardForksInOrder_ReturnsOrderedEntries(t *testing.T) {
	expectedOrder := []string{
		"Sonic",
		"Allegro",
		"Brio",
	}

	var actualOrder []string
	GetAllHardForksInOrder()(func(name string, _ Upgrades) bool {
		actualOrder = append(actualOrder, name)
		return true
	})
	require.Equal(t, expectedOrder, actualOrder, "Expected hard forks to be returned in the correct order")
}

func TestGetAllHardForksInOrder_IterationCanBeInterrupted(t *testing.T) {
	expectedOrder := []string{
		"Sonic",
	}

	var actualOrder []string
	GetAllHardForksInOrder()(func(name string, _ Upgrades) bool {
		actualOrder = append(actualOrder, name)
		return false // Interrupt after the first entry
	})
	require.Equal(t, expectedOrder, actualOrder, "Expected iteration to be interrupted after the first entry")
}
