package scc

import (
	"testing"

	"github.com/0xsoniclabs/consensus/consensus"

	"github.com/stretchr/testify/require"
)

func TestPeriod_GetPeriod_MapsBlocksToCorrectPeriod(t *testing.T) {
	tests := []struct {
		block  consensus.BlockID
		period Period
	}{
		{0, 0},
		{1, 0},
		{BLOCKS_PER_PERIOD - 1, 0},
		{BLOCKS_PER_PERIOD, 1},
		{BLOCKS_PER_PERIOD + 1, 1},
		{BLOCKS_PER_PERIOD*2 - 1, 1},
		{BLOCKS_PER_PERIOD * 2, 2},
		{BLOCKS_PER_PERIOD*2 + 1, 2},
	}

	for _, test := range tests {
		require.Equal(t, test.period, GetPeriod(test.block))
	}
}

func TestPeriod_IsFirstBlockInPeriod_IdentifiesFirstBlock(t *testing.T) {
	for i := consensus.BlockID(0); i < BLOCKS_PER_PERIOD*10; i++ {
		cur := GetPeriod(i)
		next := GetPeriod(i + 1)
		if cur != next {
			require.True(t, IsFirstBlockOfPeriod(i+1))
		}
	}
}

func TestPeriod_IsLastBlockInPeriod_IdentifiesLastBlock(t *testing.T) {
	for i := consensus.BlockID(0); i < BLOCKS_PER_PERIOD*10; i++ {
		cur := GetPeriod(i)
		next := GetPeriod(i + 1)
		if cur != next {
			require.True(t, IsLastBlockOfPeriod(i))
		}
	}
}
