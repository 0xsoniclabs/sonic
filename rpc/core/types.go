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

package core

import (
	"math/big"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/iblockproc"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// ChainInfo describes the chain the node follows.
type ChainInfo struct {
	// ChainID is the EIP-155 chain id, which is also the network id.
	ChainID        uint64
	GenesisID      common.Hash
	UpgradeHeights []opera.UpgradeHeight
}

// Progress is the synchronisation state of the node and of its best peer.
type Progress struct {
	CurrentEpoch     uint64
	CurrentBlock     uint64
	CurrentBlockHash common.Hash
	CurrentBlockTime inter.Timestamp
	HighestBlock     uint64
	HighestEpoch     uint64
}

// OracleInputs is what the gas price oracle reads per suggestion. Reading it
// walks the last event of every validator.
type OracleInputs struct {
	// NextEpochRules are the rules staged for the next epoch.
	NextEpochRules opera.Rules
	// TotalGasPowerLeft is the gas power the validators have obtained.
	TotalGasPowerLeft uint64
	// MinTip is the smallest tip the pool accepts.
	MinTip *big.Int
}

// BundleInfo tells where the transactions of an executed bundle are.
type BundleInfo struct {
	BlockNumber uint64
	// Offset is the index of the bundle's first transaction in its block.
	Offset uint32
	// Count is the number of the bundle's transactions in its block.
	Count uint32
}

// HeadNotify is a new head and its logs, sent together. Both are shared
// with every subscriber and must not be changed.
type HeadNotify struct {
	Block *evmcore.EvmBlock
	Logs  []*types.Log
}

// LogQuery is a log query over stored blocks. It is the one place a block tag
// reaches the node: the node's query code decides on the bounds as the client
// gave them.
type LogQuery struct {
	// BlockHash names one block; the bounds are then ignored.
	BlockHash *common.Hash
	// Begin and End are block numbers, or tags with go-ethereum's negative
	// values; a missing bound is a tag too.
	Begin, End int64
	Addresses  []common.Address
	Topics     [][]common.Hash
	// Head is the served head, which bounds the query.
	Head uint64
}

// EpochSnapshot is the node's block state and epoch state of one moment.
type EpochSnapshot struct {
	BlockState iblockproc.BlockState
	EpochState iblockproc.EpochState
}
