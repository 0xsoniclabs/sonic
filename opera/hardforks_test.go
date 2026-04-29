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
