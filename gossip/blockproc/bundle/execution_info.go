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

import "github.com/ethereum/go-ethereum/common"

// ExecutionInfo contains information about a processed bundle. It connects an
// execution plan's hash with the block number and position in which it got
// executed.
type ExecutionInfo struct {
	ExecutionPlanHash common.Hash
	BlockNumber       uint64
	Position          PositionInBlock
}

// PositionInBlock contains the position and number of transactions that ended
// up in a block as the result of the execution of a bundle.
type PositionInBlock struct {
	Offset uint32
	Count  uint32
}

// HistoryHash represents the block number and cumulative hash of the processed
// bundles history. It is used for genesis export/import serialization.
type HistoryHash struct {
	BlockNumber uint64
	Hash        common.Hash
}

// BundleGenesisHistoryHashes carries both the latest and the oldest
// cumulative history hashes for genesis export and import. The latest field
// allows importers to verify that replayed bundle entries produce the correct
// history hash for the oldest block still in the retention window, catching
// any mismatch between the exported entries and the expected history.
type BundleGenesisHistoryHashes struct {
	Latest HistoryHash
	Oldest HistoryHash
}
