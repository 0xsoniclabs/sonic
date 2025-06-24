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

package scc

import "github.com/Fantom-foundation/lachesis-base/inter/idx"

// BLOCKS_PER_PERIOD is the number of blocks in a period.
const BLOCKS_PER_PERIOD = 1024

// Period is an identifier for a range of blocks certified by the same committee.
// Periods have a fixed length of BLOCKS_PER_PERIOD blocks.
type Period uint64

// GetPeriod returns the period of the given block number.
func GetPeriod(number idx.Block) Period {
	return Period(number / BLOCKS_PER_PERIOD)
}

// IsFirstBlockOfPeriod returns true if the given block number is the first
// block of its period.
func IsFirstBlockOfPeriod(number idx.Block) bool {
	return number%BLOCKS_PER_PERIOD == 0
}

// IsLastBlockOfPeriod returns true if the given block number is the last block
// of its period.
func IsLastBlockOfPeriod(number idx.Block) bool {
	return number%BLOCKS_PER_PERIOD == BLOCKS_PER_PERIOD-1
}
