package scc

import "github.com/0xsoniclabs/consensus/consensus"

// BLOCKS_PER_PERIOD is the number of blocks in a period.
const BLOCKS_PER_PERIOD = 1024

// Period is an identifier for a range of blocks certified by the same committee.
// Periods have a fixed length of BLOCKS_PER_PERIOD blocks.
type Period uint64

// GetPeriod returns the period of the given block number.
func GetPeriod(number consensus.BlockID) Period {
	return Period(number / BLOCKS_PER_PERIOD)
}

// IsFirstBlockOfPeriod returns true if the given block number is the first
// block of its period.
func IsFirstBlockOfPeriod(number consensus.BlockID) bool {
	return number%BLOCKS_PER_PERIOD == 0
}

// IsLastBlockOfPeriod returns true if the given block number is the last block
// of its period.
func IsLastBlockOfPeriod(number consensus.BlockID) bool {
	return number%BLOCKS_PER_PERIOD == BLOCKS_PER_PERIOD-1
}
