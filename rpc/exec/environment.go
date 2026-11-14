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

// Package exec holds what the namespaces of the RPC module share: the
// lookups of the served chain, bounded by the served head, the execution of
// calls and the send path.
package exec

import (
	"context"
	"errors"
	"math/big"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/oracle"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

// Reader is what the environment reads from the node, and the pool it sends
// transactions to.
type Reader interface {
	core.ChainReader
	core.StateProvider
	PoolNonce(address common.Address) uint64
	SendTransaction(ctx context.Context, transaction *types.Transaction) error
	EpochOf(number uint64) (uint64, error)
	EpochState(epoch uint64) (core.EpochSnapshot, error)
	LiveEpochState() core.EpochSnapshot
}

// Environment offers the namespaces the served chain, the execution of calls
// on it, the limits of the module and the gas price oracle. Every lookup
// takes the head a request read once, finds nothing above it, and reports a
// missing item as core.ErrNotFound or a sentinel of chain.Resolve.
type Environment struct {
	config  Config
	chain   *chain.Chain
	reader  Reader
	oracle  *oracle.Oracle
	chainID *big.Int
}

// NewEnvironment creates the environment of a served chain.
func NewEnvironment(config Config, served *chain.Chain, reader Reader, suggester *oracle.Oracle) *Environment {
	return &Environment{
		config:  config,
		chain:   served,
		reader:  reader,
		oracle:  suggester,
		chainID: new(big.Int).SetUint64(reader.ChainInfo().ChainID),
	}
}

// Config holds the limits and switches of the module.
func (e *Environment) Config() Config {
	return e.config
}

// Oracle is the gas price oracle.
func (e *Environment) Oracle() *oracle.Oracle {
	return e.oracle
}

// NextHead is closed once a head newer than the served one is served.
func (e *Environment) NextHead() <-chan struct{} {
	return e.chain.NextHead()
}

// Head is the served head, which a request reads once. It is shared and must
// not be changed.
func (e *Environment) Head() *evmcore.EvmBlock {
	return e.chain.Head()
}

// ChainID is the chain id, the same value on every call; it must not be
// changed.
func (e *Environment) ChainID() *big.Int {
	return e.chainID
}

// ChainInfo describes the chain.
func (e *Environment) ChainInfo() core.ChainInfo {
	return e.reader.ChainInfo()
}

// ChainConfig is go-ethereum's chain configuration for a block, which tells
// the upgrades active at its height.
func (e *Environment) ChainConfig(number uint64) *params.ChainConfig {
	info := e.reader.ChainInfo()
	return opera.CreateTransientEvmChainConfig(info.ChainID, info.UpgradeHeights, idx.Block(number))
}

// Resolve turns a block reference into a block number against the head. A
// hash of a block above the head is unknown, as it is to the node before the
// block is stored.
func (e *Environment) Resolve(head *evmcore.EvmBlock, reference chain.BlockReference) (uint64, error) {
	number, err := e.chain.Resolve(head.NumberU64(), reference)
	if err == nil && reference.Hash != nil && number > head.NumberU64() {
		return 0, chain.ErrUnknownHash
	}
	return number, err
}

// HeaderAt reads the header of a block up to the head, without its
// transactions.
func (e *Environment) HeaderAt(head *evmcore.EvmBlock, number uint64) (*evmcore.EvmHeader, error) {
	if number > head.NumberU64() {
		return nil, core.ErrNotFound
	}
	return e.reader.Header(number)
}

// BlockAt reads a block up to the head.
func (e *Environment) BlockAt(head *evmcore.EvmBlock, number uint64) (*evmcore.EvmBlock, error) {
	if number > head.NumberU64() {
		return nil, core.ErrNotFound
	}
	return e.reader.Block(number)
}

// FindHeader reads the header of the block a reference names.
func (e *Environment) FindHeader(head *evmcore.EvmBlock, reference chain.BlockReference) (*evmcore.EvmHeader, error) {
	number, err := e.Resolve(head, reference)
	if err != nil {
		return nil, err
	}
	return e.HeaderAt(head, number)
}

// FindBlock reads the block a reference names.
func (e *Environment) FindBlock(head *evmcore.EvmBlock, reference chain.BlockReference) (*evmcore.EvmBlock, error) {
	number, err := e.Resolve(head, reference)
	if err != nil {
		return nil, err
	}
	return e.BlockAt(head, number)
}

// ReceiptsAt reads the receipts of a block up to the head.
func (e *Environment) ReceiptsAt(head *evmcore.EvmBlock, number uint64) (types.Receipts, error) {
	if number > head.NumberU64() {
		return nil, core.ErrNotFound
	}
	return e.reader.Receipts(number)
}

// TransactionByHash finds a transaction of a block up to the head, and tells
// its block and index.
func (e *Environment) TransactionByHash(head *evmcore.EvmBlock, hash common.Hash) (transaction *types.Transaction, block, index uint64, err error) {
	transaction, block, index, err = e.reader.TransactionByHash(hash)
	if err != nil {
		return nil, 0, 0, err
	}
	if block > head.NumberU64() {
		return nil, 0, 0, core.ErrNotFound
	}
	return transaction, block, index, nil
}

// IsMissing tells whether a lookup failed for want of the item: a block,
// state or transaction that is not there, or a hash of no block.
func IsMissing(err error) bool {
	return errors.Is(err, core.ErrNotFound) || errors.Is(err, chain.ErrUnknownHash)
}

// NoneIfMissing answers a missing item as no value without an error, which
// is how most methods answer it.
func NoneIfMissing[Value any](value Value, err error) (Value, error) {
	if IsMissing(err) {
		var none Value
		return none, nil
	}
	return value, err
}
