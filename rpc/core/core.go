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

// Package core defines Core, the one interface through which the RPC module
// reaches the node, together with its role interfaces and value types.
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
// or state names a block by number or hash; nothing here mutates except
// SendTransaction. It is composed of role interfaces so that a namespace
// depends only on the roles it uses.
type Core interface {
	ChainReader
	HeadSource
	StateProvider
	Pool
	LogIndex
	NodeInfo
}

// ChainReader reads the committed chain. A block above the committed head is
// reported as ErrNotFound.
type ChainReader interface {
	// ChainInfo returns the facts of the chain that do not depend on a block.
	ChainInfo() (ChainInfo, error)
	// Header returns the header of a block without decoding its transactions,
	// so its transactions root is not set.
	Header(number uint64) (*evmcore.EvmHeader, error)
	// Block returns a block with its transactions.
	Block(number uint64) (*evmcore.EvmBlock, error)
	// BlockNumberByHash returns the number of the block with the given hash.
	BlockNumberByHash(hash common.Hash) (uint64, error)
	// Receipts returns the receipts of a block. It fails if the transaction
	// index is disabled.
	Receipts(number uint64) (types.Receipts, error)
	// Rules returns the rules of the epoch a block belongs to. For the block
	// after the committed head it returns the rules of the open epoch.
	Rules(number uint64) (opera.Rules, error)
	// TransactionByHash returns a committed transaction and its position. It
	// fails if the transaction index is disabled.
	TransactionByHash(hash common.Hash) (transaction *types.Transaction, block uint64, index uint64, err error)
	// BundleInfo returns where a bundle with the given execution plan hash was
	// executed.
	BundleInfo(executionPlanHash common.Hash) (BundleInfo, error)
}

// HeadSource tells the newest block whose state is archived.
type HeadSource interface {
	// Head returns the newest block whose state is archived.
	Head() (*evmcore.EvmBlock, error)
	// SubscribeHead delivers each new archived block with its logs. A stopped
	// node answers with a subscription that has already ended.
	SubscribeHead(events chan<- HeadEvent) Subscription
}

// StateProvider opens the archived state of a block. The state is a
// non-committable view, journaled per request; the caller releases it
// exactly once.
type StateProvider interface {
	State(number uint64) (state.StateDB, error)
}

// Pool is the transaction pool, the only part of Core whose answers depend on
// the current moment.
type Pool interface {
	SendTransaction(ctx context.Context, transaction *types.Transaction) error
	PoolNonce(address common.Address) uint64
	PoolTransaction(hash common.Hash) (*types.Transaction, error)
	PoolTransactions() (types.Transactions, error)
	PoolContent() (pending, queued map[common.Address]types.Transactions)
	PoolContentFrom(address common.Address) (pending, queued types.Transactions)
	PoolStats() (pending, queued int)
	SubscribeTransactions(events chan<- evmcore.NewTxsNotify) Subscription
}

// LogIndex is the node's index of log addresses and topics over the archived
// blocks.
type LogIndex interface {
	FindLogs(ctx context.Context, first, last uint64, pattern [][]common.Hash, limit uint) ([]*types.Log, error)
}

// NodeInfo reports the liveness of the node and the inputs of the gas price
// oracle.
type NodeInfo interface {
	Progress() Progress
	// OracleInputs walks the last event of every validator; it is read for a
	// suggestion, never per sampling tick.
	OracleInputs() OracleInputs
	// PendingHeads returns the first pending transaction of every account in
	// the pool.
	PendingHeads() types.Transactions
}

// Subscription is a feed subscription. go-ethereum's event.Subscription
// satisfies it.
type Subscription interface {
	Err() <-chan error
	Unsubscribe()
}

// ChainInfo holds the facts of the chain that do not depend on a block.
type ChainInfo struct {
	ChainID        uint64
	GenesisID      common.Hash
	UpgradeHeights []opera.UpgradeHeight
}

// Progress is the synchronization status of the node.
type Progress struct {
	CurrentEpoch     uint64
	CurrentBlock     uint64
	CurrentBlockHash common.Hash
	CurrentBlockTime inter.Timestamp
	HighestBlock     uint64
	HighestEpoch     uint64
}

// OracleInputs are the facts the gas price oracle combines into a suggestion.
type OracleInputs struct {
	// PendingRules are the rules the open epoch ends with, including the
	// changes made in it.
	PendingRules opera.Rules
	// TotalGasPowerLeft is the gas power the validators have left.
	TotalGasPowerLeft uint64
	// MinTip is the smallest tip the pool accepts.
	MinTip *big.Int
}

// BundleInfo tells where a bundle was executed.
type BundleInfo struct {
	BlockNumber uint64
	Offset      uint32
	Count       uint32
}

// HeadEvent is a new archived block with its logs. Both are shared by every
// consumer and must not be modified.
type HeadEvent struct {
	Block *evmcore.EvmBlock
	Logs  []*types.Log
}
