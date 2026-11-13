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

// Package core declares what the RPC module may ask the node, and nothing
// else: the module reaches the node only through Core, and the node
// implements it with one adapter. Declaring it here, with its user, lets the
// module build and be tested without the node.
package core

import (
	"context"
	"errors"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// ErrNotFound reports a missing item. A lookup answers it, never a nil value
// without an error.
var ErrNotFound = errors.New("not found")

// Core is everything the RPC module may ask the node. No call takes a tag:
// chain and state are read by block number, by hash or by id; the bounds of a
// log query are the one exception. Only SendTransaction and the TestOnly
// calls change anything. A namespace depends only on the roles it uses.
type Core interface {
	ChainReader
	HeadSource
	StateProvider
	Pool
	LogIndex
	NodeInfo
	ConsensusReader
	TestOnly
}

// ChainReader reads the stored chain. Its answers do not change once an item
// is stored.
type ChainReader interface {
	ChainInfo() ChainInfo
	// Header reads a block without decoding its transactions.
	Header(number uint64) (*evmcore.EvmHeader, error)
	// Block answers ErrNotFound above the committed head.
	Block(number uint64) (*evmcore.EvmBlock, error)
	// BlockNumberByHash answers ErrNotFound for an unknown hash.
	BlockNumberByHash(hash common.Hash) (uint64, error)
	// Receipts fails if the node does not index transactions.
	Receipts(number uint64) (types.Receipts, error)
	// Rules answers the rules of the epoch of a stored block, and above the
	// committed head the rules of the open epoch.
	Rules(number uint64) (*opera.Rules, error)
	// TransactionByHash fails if the node does not index transactions.
	TransactionByHash(hash common.Hash) (transaction *types.Transaction, block, index uint64, err error)
	BundleInfo(executionPlanHash common.Hash) (BundleInfo, error)
}

// HeadSource tells the newest block whose state is archived. Without an
// archive, it is the committed head.
type HeadSource interface {
	Head() (*evmcore.EvmBlock, error)
	SubscribeHead(events chan<- HeadNotify) Subscription
}

// StateProvider opens the archive at one block, as Carmen's non-committable
// StateDB: journaled per request, released once, never committed.
type StateProvider interface {
	State(number uint64) (state.StateDB, error)
}

// Pool holds the transactions the node has outside blocks. Every answer
// depends on "now".
type Pool interface {
	SendTransaction(ctx context.Context, transaction *types.Transaction) error
	// PoolNonce is the pending nonce: the next nonce of the sender after the
	// transactions the pool holds.
	PoolNonce(address common.Address) uint64
	PoolTransaction(hash common.Hash) (*types.Transaction, error)
	// PoolNextTransactions is the next executable transaction of every
	// sender, unordered: the sample of the gas price oracle.
	PoolNextTransactions() types.Transactions
	PoolContent() (pending, queued map[common.Address]types.Transactions)
	PoolContentFrom(address common.Address) (pending, queued types.Transactions)
	PoolStats() (pending, queued int)
	SubscribeTransactions(events chan<- evmcore.NewTxsNotify) Subscription
}

// LogIndex answers log queries over archived blocks with the node's own query
// code on its address and topic index, so that both RPC stacks of the node
// find the same logs.
type LogIndex interface {
	Logs(ctx context.Context, query LogQuery) ([]*types.Log, error)
}

// NodeInfo tells liveness, peers and what the gas price oracle reads per
// suggestion. Every answer depends on "now".
type NodeInfo interface {
	Progress() Progress
	OracleInputs() OracleInputs
	PeerCount() int
}

// ConsensusReader reads epochs, validators and DAG events. EpochOf and
// EpochState are keyed like the chain reads. The two Live calls answer from
// the node's current state. The event lookups read the DAG, which runs ahead
// of the blocks, so the served head does not bound them.
type ConsensusReader interface {
	// EpochOf answers the epoch of a stored block, and above the committed
	// head the open epoch.
	EpochOf(number uint64) (uint64, error)
	// EpochState answers the states stored when the epoch opened, and
	// ErrNotFound without them.
	EpochState(epoch uint64) (EpochSnapshot, error)
	// LiveEpochState reads the node's current block and epoch state together.
	LiveEpochState() EpochSnapshot
	// LiveEpochHeads answers the events without descendants, and ErrNotFound
	// unless the epoch is open.
	LiveEpochHeads(epoch uint64) ([]common.Hash, error)
	// FindEvents answers the events a short id names.
	FindEvents(epoch uint64, lamport uint32, prefix []byte) []common.Hash
	Event(id common.Hash) (*inter.Event, error)
	EventPayload(id common.Hash) (*inter.EventPayload, error)
}

// TestOnly holds the calls of the test namespace, which change the node
// outside the rules for tests.
type TestOnly interface {
	// SendTransactions puts the transactions into the pool in one call; the
	// emitter then validates and orders them as usual.
	SendTransactions(transactions types.Transactions) error
	// EmitTransactions emits the transactions, as given and without
	// validation, in a new event of this validator.
	EmitTransactions(transactions types.Transactions) error
}

// Subscription is a feed subscription of the node.
type Subscription interface {
	Err() <-chan error
	Unsubscribe()
}
