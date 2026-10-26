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

// Package exec holds what the namespaces share: the one set of lookups of
// blocks, headers, transactions and state at the served head, the execution
// of calls on the EVM, the send path of transactions, and the semantic limits
// of the methods.
package exec

import (
	"context"
	"math/big"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/oracle"
)

// Environment is what a namespace reaches the chain and the state through.
// It keeps the node's Core to itself, so no read bypasses its lookups.
type Environment struct {
	nodeCore core.Core
	chain    *chain.Chain
	config   Config
	oracle   *oracle.Oracle
}

// NewEnvironment returns the environment of the namespaces, with a gas price
// oracle over the node.
func NewEnvironment(nodeCore core.Core, served *chain.Chain, config Config, oracleConfig oracle.Config) *Environment {
	environment := &Environment{nodeCore: nodeCore, chain: served, config: config}
	environment.oracle = oracle.New(oracleConfig, nodeCore, environment.openRules)
	return environment
}

// Oracle returns the gas price oracle.
func (environment *Environment) Oracle() *oracle.Oracle {
	return environment.oracle
}

// openRules reads the rules of the open epoch: those of the block after the
// served head, never those of the head, which may seal an epoch.
func (environment *Environment) openRules(ctx context.Context) (opera.Rules, error) {
	head, err := environment.Latest(ctx)
	if err != nil {
		return opera.Rules{}, err
	}
	return environment.RulesAt(head, head.Number+1)
}

// Config returns the semantic limits of the methods.
func (environment *Environment) Config() Config {
	return environment.config
}

// Head is the served head as one request sees it: every lookup made with it
// answers at this head and observes the request's context.
type Head struct {
	chain.Head
	ctx context.Context
}

// Latest returns the served head for a request with the given context.
func (environment *Environment) Latest(ctx context.Context) (Head, error) {
	head, err := environment.chain.Latest()
	if err != nil {
		return Head{}, err
	}
	return Head{Head: head, ctx: ctx}, nil
}

// View returns a head for a background loop that reads at a given head.
func (environment *Environment) View(ctx context.Context, head chain.Head) Head {
	return Head{Head: head, ctx: ctx}
}

// Context returns the context of the request the head belongs to.
func (head Head) Context() context.Context {
	return head.ctx
}

// read calls Core after and before checking the request's context, so that a
// cancelled request makes at most one more call.
func read[Value any](head Head, lookup func() (Value, error)) (Value, error) {
	var zero Value
	if err := head.ctx.Err(); err != nil {
		return zero, err
	}
	value, err := lookup()
	if err != nil {
		return zero, err
	}
	if err := head.ctx.Err(); err != nil {
		return zero, err
	}
	return value, nil
}

// Resolve turns a reference into a block number at the head.
func (environment *Environment) Resolve(head Head, reference chain.BlockReference) (uint64, error) {
	if reference.Hash == nil {
		return environment.chain.Resolve(head.Head, reference)
	}
	return read(head, func() (uint64, error) {
		return environment.chain.Resolve(head.Head, reference)
	})
}

// HeaderAt returns the header of a block at or below the head, without its
// transactions root.
func (environment *Environment) HeaderAt(head Head, number uint64) (*evmcore.EvmHeader, error) {
	if number > head.Number {
		return nil, core.ErrNotFound
	}
	return read(head, func() (*evmcore.EvmHeader, error) {
		return notNil(environment.nodeCore.Header(number))
	})
}

// BlockAt returns a block at or below the head, with its transactions.
func (environment *Environment) BlockAt(head Head, number uint64) (*evmcore.EvmBlock, error) {
	if number > head.Number {
		return nil, core.ErrNotFound
	}
	return read(head, func() (*evmcore.EvmBlock, error) {
		return notNil(environment.nodeCore.Block(number))
	})
}

// FindBlock returns the block a reference names.
func (environment *Environment) FindBlock(head Head, reference chain.BlockReference) (*evmcore.EvmBlock, error) {
	number, err := environment.Resolve(head, reference)
	if err != nil {
		return nil, err
	}
	return environment.BlockAt(head, number)
}

// FindHeader returns the header of the block a reference names.
func (environment *Environment) FindHeader(head Head, reference chain.BlockReference) (*evmcore.EvmHeader, error) {
	number, err := environment.Resolve(head, reference)
	if err != nil {
		return nil, err
	}
	return environment.HeaderAt(head, number)
}

// ReceiptsAt returns the receipts of a block at or below the head.
func (environment *Environment) ReceiptsAt(head Head, number uint64) (types.Receipts, error) {
	if number > head.Number {
		return nil, core.ErrNotFound
	}
	return read(head, func() (types.Receipts, error) {
		receipts, err := environment.nodeCore.Receipts(number)
		if err == nil && receipts == nil {
			return nil, core.ErrNotFound
		}
		return receipts, err
	})
}

// RulesAt returns the rules of the epoch of a block; for the block after the
// head, those it is processed with.
func (environment *Environment) RulesAt(head Head, number uint64) (opera.Rules, error) {
	if number > head.Number+1 {
		return opera.Rules{}, core.ErrNotFound
	}
	return read(head, func() (opera.Rules, error) {
		return environment.nodeCore.Rules(number)
	})
}

// TransactionByHash returns a transaction in a block at or below the head
// with its position.
func (environment *Environment) TransactionByHash(head Head, hash common.Hash) (*Located, error) {
	return read(head, func() (*Located, error) {
		transaction, block, index, err := environment.nodeCore.TransactionByHash(hash)
		if err != nil {
			return nil, err
		}
		if transaction == nil || block > head.Number {
			return nil, core.ErrNotFound
		}
		return &Located{Transaction: transaction, Block: block, Index: index}, nil
	})
}

// BundleInfo returns where a bundle with the given execution plan hash was
// executed, in a block at or below the head.
func (environment *Environment) BundleInfo(head Head, executionPlanHash common.Hash) (*core.BundleInfo, error) {
	return read(head, func() (*core.BundleInfo, error) {
		info, err := environment.nodeCore.BundleInfo(executionPlanHash)
		if err != nil {
			return nil, err
		}
		if info.BlockNumber > head.Number {
			return nil, core.ErrNotFound
		}
		return &info, nil
	})
}

// Located is a transaction with its position in the chain.
type Located struct {
	Transaction *types.Transaction
	Block       uint64
	Index       uint64
}

// OpenState opens the state after a block at or below the head. The caller
// releases it. A state opened by a request cancelled meanwhile is released
// here.
func (environment *Environment) OpenState(head Head, number uint64) (state.StateDB, error) {
	if number > head.Number {
		return nil, core.ErrNotFound
	}
	if err := head.ctx.Err(); err != nil {
		return nil, err
	}
	opened, err := environment.nodeCore.State(number)
	if err != nil {
		return nil, err
	}
	if opened == nil {
		return nil, core.ErrNotFound
	}
	if err := head.ctx.Err(); err != nil {
		opened.Release()
		return nil, err
	}
	return opened, nil
}

// FindState opens the state after the block a reference names. The caller
// releases it.
func (environment *Environment) FindState(head Head, reference chain.BlockReference) (state.StateDB, error) {
	number, err := environment.Resolve(head, reference)
	if err != nil {
		return nil, err
	}
	return environment.OpenState(head, number)
}

// StateAt opens the state of the block a reference names, with the block's
// header. The caller releases the state.
func (environment *Environment) StateAt(head Head, reference chain.BlockReference) (state.StateDB, *evmcore.EvmHeader, error) {
	number, err := environment.Resolve(head, reference)
	if err != nil {
		return nil, nil, err
	}
	header, err := environment.HeaderAt(head, number)
	if err != nil {
		return nil, nil, err
	}
	opened, err := environment.OpenState(head, number)
	if err != nil {
		return nil, nil, err
	}
	return opened, header, nil
}

// Heads is the feed of the blocks the served head moves to.
func (environment *Environment) Heads() *chain.Hub[core.HeadEvent] {
	return environment.chain.Heads()
}

// Transactions is the feed of the transactions that enter the pool.
func (environment *Environment) Transactions() *chain.Hub[evmcore.NewTxsNotify] {
	return environment.chain.Transactions()
}

// ChainInfo returns the facts of the chain the head loop last read.
func (environment *Environment) ChainInfo() core.ChainInfo {
	return environment.chain.ChainInfo()
}

// ChainID is the chain id as a number.
func (environment *Environment) ChainID() *big.Int {
	return new(big.Int).SetUint64(environment.chain.ChainInfo().ChainID)
}

// Signer is the signer of the chain's transactions.
func (environment *Environment) Signer() types.Signer {
	return types.LatestSignerForChainID(environment.ChainID())
}

// ChainConfig returns go-ethereum's chain configuration for a block, derived
// from the chain id and the upgrade heights.
func (environment *Environment) ChainConfig(number uint64) *params.ChainConfig {
	info := environment.chain.ChainInfo()
	return opera.CreateTransientEvmChainConfig(info.ChainID, info.UpgradeHeights, idx.Block(number))
}

// notNil turns a value Core answered without an error but nil into
// core.ErrNotFound, the one way the module says "not found".
func notNil[Value any](value *Value, err error) (*Value, error) {
	if err == nil && value == nil {
		return nil, core.ErrNotFound
	}
	return value, err
}
