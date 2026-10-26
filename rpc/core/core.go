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

// Package core defines the one interface through which the RPC module reaches
// the node, and the values that cross it.
package core

import (
	"context"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
)

// ErrNotFound is how every lookup of Core reports a missing item. A lookup
// never answers a nil value without an error.
var ErrNotFound = errors.New("not found")

// Core is everything the RPC module may ask the node. Every call about chain
// or state names a block by number or hash; nothing here mutates except SendTx.
// It is composed of role interfaces so a namespace depends only on the roles
// it uses. A lookup reports a missing item as ErrNotFound.
type Core interface {
	ChainReader
	HeadSource
	StateProvider
	Pool
	LogIndex
	NodeInfo
}

// ChainReader reads the committed chain by block number.
type ChainReader interface {
	// ChainInfo returns the facts of the chain that do not depend on a block.
	ChainInfo() ChainInfo
	// Header returns the header of a block, without decoding its transactions.
	Header(number uint64) (*evmcore.EvmHeader, error)
	// Block returns a block with its transactions; ErrNotFound above the
	// committed head.
	Block(number uint64) (*evmcore.EvmBlock, error)
	// BlockNumberByHash maps a block hash to its number; ErrNotFound for an
	// unknown hash.
	BlockNumberByHash(hash common.Hash) (uint64, error)
	// Receipts returns the receipts of a block; it fails if TxIndex is off.
	Receipts(number uint64) (types.Receipts, error)
	// Rules returns the rules a block is processed with; for the block after
	// the committed head, the rules of the open epoch.
	Rules(number uint64) (*opera.Rules, error)
	// TxByHash locates a committed transaction; ErrNotFound if unknown, and
	// it fails if TxIndex is off.
	TxByHash(hash common.Hash) (tx *types.Transaction, block, index uint64, err error)
	// BundleInfo locates the execution of a bundle; ErrNotFound for an
	// unknown plan.
	BundleInfo(executionPlanHash common.Hash) (BundleInfo, error)
}

// HeadSource delivers the newest block whose state is archived.
type HeadSource interface {
	// Head returns the newest block whose state is archived.
	Head() (*evmcore.EvmBlock, error)
	// SubscribeHead delivers every new archived block with its logs. A
	// stopped node answers with a subscription that has already ended.
	SubscribeHead(ch chan<- HeadEvent) Subscription
}

// StateProvider opens the archive at one block in Carmen's non-committable
// StateDB. A state is journaled per request, released exactly once, and never
// committed.
type StateProvider interface {
	State(number uint64) (state.StateDB, error)
}

// Pool holds the only calls whose answer depends on "now".
type Pool interface {
	SendTx(ctx context.Context, tx *types.Transaction) error
	PoolNonce(addr common.Address) uint64
	// PoolTransaction returns a transaction of the pool; ErrNotFound if it is
	// not in the pool.
	PoolTransaction(hash common.Hash) (*types.Transaction, error)
	PoolTransactions() (types.Transactions, error)
	PoolContent() (pending, queued map[common.Address]types.Transactions)
	PoolContentFrom(addr common.Address) (pending, queued types.Transactions)
	PoolStats() (pending, queued int)
	// SubscribeTxs delivers the transactions entering the pool. A stopped
	// node answers with a subscription that has already ended.
	SubscribeTxs(ch chan<- evmcore.NewTxsNotify) Subscription
}

// LogIndex is the node's address and topic index over archived blocks. It
// serves the indexed path of eth_getLogs.
type LogIndex interface {
	FindLogs(ctx context.Context, from, to uint64, pattern [][]common.Hash, limit uint) ([]*types.Log, error)
}

// NodeInfo reports the node's liveness and the gas price oracle's inputs.
type NodeInfo interface {
	Progress() Progress
	// OracleInputs walks the last event of every validator; it is read for a
	// suggestion only.
	OracleInputs() OracleInputs
	// PendingHeads returns the first pending transaction of up to max
	// accounts of the pool.
	PendingHeads(max int) []PendingHead
}

// Subscription is a feed subscription; go-ethereum's event.Subscription
// satisfies it.
type Subscription interface {
	Err() <-chan error
	Unsubscribe()
}

// ChainInfo holds the facts of the chain that no block selects.
type ChainInfo struct {
	ChainID        *big.Int
	NetworkID      uint64
	GenesisID      common.Hash
	UpgradeHeights []opera.UpgradeHeight
}

// Progress is the synchronization status of the node and its peers.
type Progress struct {
	CurrentEpoch     uint64
	CurrentBlock     uint64
	CurrentBlockHash common.Hash
	CurrentBlockTime inter.Timestamp
	HighestBlock     uint64
	HighestEpoch     uint64
}

// OracleInputs are the facts of the open epoch the gas price oracle weighs.
type OracleInputs struct {
	PendingRules      opera.Rules
	TotalGasPowerLeft uint64
	MinTip            *big.Int
}

// PendingHead describes the first pending transaction of one account.
type PendingHead struct {
	Gas       uint64
	GasTipCap *big.Int
	GasFeeCap *big.Int
}

// BundleInfo locates the transactions a bundle left in a block.
type BundleInfo struct {
	BlockNumber uint64
	Offset      uint32
	Count       uint32
}

// HeadEvent is one new archived block and its logs.
type HeadEvent struct {
	Block *evmcore.EvmBlock
	Logs  []*types.Log
}
