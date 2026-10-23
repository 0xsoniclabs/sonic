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

// Package core defines the narrow interface between the node and the RPC
// module. The module reaches the node only through [Core]; the node implements
// it in one adapter (gossip/rpc_core.go). Nothing in this interface mutates the
// node except [Pool.SendTx].
package core

import (
	"context"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
)

// ErrNotFound is returned by [ChainReader] and [StateProvider] for a block the
// node does not (yet) have, e.g. one above the committed or archived head.
var ErrNotFound = errors.New("not found")

// Core is everything the RPC module may ask the node. Every call about chain or
// state names a block by number or hash; nothing here mutates except SendTx.
// It is composed of role interfaces so a namespace depends only on the roles it
// uses and fakes stay small.
type Core interface {
	ChainReader   // ChainInfo, Header, Block, BlockNumberByHash, Receipts, Rules, TxByHash, BundleInfo
	HeadSource    // Head, SubscribeHead
	StateProvider // State
	Pool          // SendTx, PoolNonce, PoolTransaction(s), PoolContent(From), PoolStats, SubscribeTxs
	LogIndex      // FindLogs
	NodeInfo      // Progress, OracleInputs, PendingHeads
}

// ChainReader gives access to the chain history.
type ChainReader interface {
	ChainInfo() ChainInfo
	// Header returns the header of a block; its transactions are not decoded.
	Header(number uint64) (*evmcore.EvmHeader, error)
	// Block returns a block with its transactions; ErrNotFound above the
	// committed head.
	Block(number uint64) (*evmcore.EvmBlock, error)
	BlockNumberByHash(hash common.Hash) (uint64, bool)
	// Receipts returns the receipts of a block. It fails if the node does
	// not index transactions.
	Receipts(number uint64) (types.Receipts, error)
	// Rules returns the rules in force at a block. For the block after the
	// newest one it returns the rules of the open epoch, which the next block
	// is processed with.
	Rules(number uint64) (*opera.Rules, error)
	// TxByHash returns a transaction with its block and position; ErrNotFound
	// for an unknown transaction. It fails if the node does not index
	// transactions.
	TxByHash(hash common.Hash) (tx *types.Transaction, block, index uint64, err error)
	BundleInfo(executionPlanHash common.Hash) (BundleInfo, bool)
}

// HeadSource publishes the head of the chain, the newest block whose state is
// archived.
type HeadSource interface {
	// Head returns the newest block whose state is archived.
	Head() (*evmcore.EvmBlock, error)
	// SubscribeHead delivers every block that becomes available in the archive,
	// in order, together with its logs. The send blocks on ch; the receiver must
	// keep draining it until it unsubscribes.
	SubscribeHead(ch chan<- HeadEvent) Subscription
}

// StateProvider gives access to archived state: the archive at one block in
// Carmen's non-committable StateDB (inter/state.StateDB: GetProof, Copy,
// EndTransaction, Release). A state is journaled per request, released exactly
// once by the caller, and never committed.
type StateProvider interface {
	State(number uint64) (state.StateDB, error)
}

// Pool is the only part of the interface whose answers depend on "now".
type Pool interface {
	SendTx(ctx context.Context, tx *types.Transaction) error
	PoolNonce(addr common.Address) uint64
	PoolTransaction(hash common.Hash) *types.Transaction
	PoolTransactions() (types.Transactions, error)
	PoolContent() (pending, queued map[common.Address]types.Transactions)
	PoolContentFrom(addr common.Address) (pending, queued types.Transactions)
	PoolStats() (pending, queued int)
	SubscribeTxs(ch chan<- evmcore.NewTxsNotify) Subscription
}

// LogIndex is the node's address/topic index over archived blocks (topicsdb).
// It serves the indexed path of eth_getLogs until the module owns an index.
type LogIndex interface {
	// FindLogs returns the logs of the blocks [from, to] matching the pattern
	// (position 0 are the addresses, the rest the topics). A limit of 0 means
	// unlimited.
	FindLogs(ctx context.Context, from, to uint64, pattern [][]common.Hash, limit uint) ([]*types.Log, error)
}

// NodeInfo carries liveness and the inputs of the gas price oracle.
type NodeInfo interface {
	Progress() Progress
	OracleInputs() OracleInputs
	// PendingHeads returns the first pending transaction of each account that
	// has one, at most max of them.
	PendingHeads(max int) []PendingHead
}

// Subscription is the handle of a feed subscription. It is satisfied by
// go-ethereum's event.Subscription.
type Subscription interface {
	// Err returns a channel that is closed when the subscription ended.
	Err() <-chan error
	Unsubscribe()
}

// ChainInfo holds the constant facts about the chain.
type ChainInfo struct {
	ChainID        uint64
	NetworkID      uint64
	GenesisID      common.Hash
	UpgradeHeights []opera.UpgradeHeight
}

// Progress describes the synchronization state of the node.
type Progress struct {
	CurrentEpoch     uint64
	CurrentBlock     uint64
	CurrentBlockHash common.Hash
	// CurrentBlockTime is the block timestamp in nanoseconds.
	CurrentBlockTime uint64
	HighestBlock     uint64
	HighestEpoch     uint64
}

// OracleInputs are the node facts the gas price oracle samples.
type OracleInputs struct {
	// PendingRules are the rules that apply from the next epoch on (the dirty
	// rules if there are any, otherwise the current ones).
	PendingRules opera.Rules
	// Rules are the rules of the current epoch.
	Rules opera.Rules
	// TotalGasPowerLeft is the gas power the validators have available.
	TotalGasPowerLeft uint64
	// MinTip is the smallest tip the pool accepts.
	MinTip *big.Int
}

// PendingHead is the first pending transaction of an account.
type PendingHead struct {
	Tx *types.Transaction
}

// BundleInfo says where a bundle was executed.
type BundleInfo struct {
	BlockNumber uint64
	Offset      uint32
	Count       uint32
}

// HeadEvent announces a block that became available in the archive.
type HeadEvent struct {
	Block *evmcore.EvmBlock
	Logs  []*types.Log
}
