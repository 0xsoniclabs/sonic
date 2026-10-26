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

// Package exec holds what the methods share: the one set of lookups of
// blocks, headers, transactions and state, the arguments and environment of
// an EVM run, and the send path.
package exec

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/gasprice/gaspricelimits"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/oracle"
)

// Env is what a namespace reaches the chain and the state through. It keeps
// Core to itself: every lookup goes through the methods below, which resolve
// against the head a request holds and report absence as core.ErrNotFound or
// a sentinel of chain.Resolve.
type Env struct {
	core   core.Core
	chain  *chain.Chain
	oracle *oracle.Oracle
	d      *dispatch.Dispatcher
	cfg    Config
}

// NewEnv creates the environment of the namespaces.
func NewEnv(c core.Core, ch *chain.Chain, o *oracle.Oracle, d *dispatch.Dispatcher, cfg Config) *Env {
	return &Env{core: c, chain: ch, oracle: o, d: d, cfg: cfg}
}

// Config returns the limits of the methods.
func (e *Env) Config() Config {
	return e.cfg
}

// Chain returns the served head and its feeds.
func (e *Env) Chain() *chain.Chain {
	return e.chain
}

// Oracle returns the gas price oracle.
func (e *Env) Oracle() *oracle.Oracle {
	return e.oracle
}

// Dispatcher returns the runtime, for the children of a request.
func (e *Env) Dispatcher() *dispatch.Dispatcher {
	return e.d
}

// Head is what one request reads the chain under: the served head, read once,
// and the request's context. Every lookup checks the context before it asks
// the node and after, so a cancelled request returns within one Core call and
// no EVM run starts after the cancel.
type Head struct {
	*chain.Head
	ctx context.Context
}

// Context is the context of the request.
func (h *Head) Context() context.Context {
	return h.ctx
}

// Latest is the head a request resolves its tags against; it reads it once.
func (e *Env) Latest(ctx context.Context) *Head {
	return &Head{Head: e.chain.Latest(), ctx: ctx}
}

// View is the head of work that is not a request, such as a feed loop, under
// that work's context.
func (e *Env) View(ctx context.Context, head *chain.Head) *Head {
	return &Head{Head: head, ctx: ctx}
}

// ask runs one Core call of a request: not after the request was cancelled,
// and a nil answer without error, which Core never gives, is absence.
func ask[T any](ctx context.Context, get func() (*T, error)) (*T, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	v, err := get()
	if err == nil && v == nil {
		err = core.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return v, nil
}

// Resolve turns a block reference into a block number.
func (e *Env) Resolve(head *Head, ref chain.BlockRef) (uint64, error) {
	if err := head.ctx.Err(); err != nil {
		return 0, err
	}
	return e.chain.Resolve(head.Head, ref)
}

// HeaderAt returns the header of a block up to the head, without its
// transactions.
func (e *Env) HeaderAt(head *Head, n uint64) (*evmcore.EvmHeader, error) {
	if n > head.Number() {
		return nil, core.ErrNotFound
	}
	return ask(head.ctx, func() (*evmcore.EvmHeader, error) { return e.core.Header(n) })
}

// BlockAt returns a block up to the head, with its transactions.
func (e *Env) BlockAt(head *Head, n uint64) (*evmcore.EvmBlock, error) {
	if n > head.Number() {
		return nil, core.ErrNotFound
	}
	return ask(head.ctx, func() (*evmcore.EvmBlock, error) { return e.core.Block(n) })
}

// FindHeader resolves a reference and returns the header of its block.
func (e *Env) FindHeader(head *Head, ref chain.BlockRef) (*evmcore.EvmHeader, error) {
	n, err := e.Resolve(head, ref)
	if err != nil {
		return nil, err
	}
	return e.HeaderAt(head, n)
}

// FindBlock resolves a reference and returns its block.
func (e *Env) FindBlock(head *Head, ref chain.BlockRef) (*evmcore.EvmBlock, error) {
	n, err := e.Resolve(head, ref)
	if err != nil {
		return nil, err
	}
	return e.BlockAt(head, n)
}

// ReceiptsAt returns the receipts of a block up to the head.
func (e *Env) ReceiptsAt(head *Head, n uint64) (types.Receipts, error) {
	if n > head.Number() {
		return nil, core.ErrNotFound
	}
	receipts, err := ask(head.ctx, func() (*types.Receipts, error) {
		r, err := e.core.Receipts(n)
		return &r, err
	})
	if err != nil {
		return nil, err
	}
	return *receipts, nil
}

// position is where a transaction is in the chain.
type position struct {
	tx           *types.Transaction
	block, index uint64
}

// TxByHash locates a transaction of a block up to the head.
func (e *Env) TxByHash(head *Head, hash common.Hash) (tx *types.Transaction, block, index uint64, err error) {
	p, err := ask(head.ctx, func() (*position, error) {
		tx, block, index, err := e.core.TxByHash(hash)
		if tx == nil {
			return nil, err
		}
		return &position{tx: tx, block: block, index: index}, err
	})
	if err != nil {
		return nil, 0, 0, err
	}
	if p.block > head.Number() {
		return nil, 0, 0, core.ErrNotFound
	}
	return p.tx, p.block, p.index, nil
}

// BundleInfo locates the execution of a bundle in a block up to the head.
func (e *Env) BundleInfo(head *Head, planHash common.Hash) (core.BundleInfo, error) {
	info, err := ask(head.ctx, func() (*core.BundleInfo, error) {
		info, err := e.core.BundleInfo(planHash)
		return &info, err
	})
	if err != nil {
		return core.BundleInfo{}, err
	}
	if info.BlockNumber > head.Number() {
		return core.BundleInfo{}, core.ErrNotFound
	}
	return *info, nil
}

// OpenState resolves a reference and opens the archive at its block. The
// caller releases the state on the line after.
func (e *Env) OpenState(head *Head, ref chain.BlockRef) (state.StateDB, error) {
	n, err := e.Resolve(head, ref)
	if err != nil {
		return nil, err
	}
	return e.StateAtNumber(head, n)
}

// StateAtNumber opens the archive at a block up to the head.
func (e *Env) StateAtNumber(head *Head, n uint64) (state.StateDB, error) {
	if n > head.Number() {
		return nil, core.ErrNotFound
	}
	if err := head.ctx.Err(); err != nil {
		return nil, err
	}
	st, err := e.core.State(n)
	if err == nil && st == nil {
		err = core.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := head.ctx.Err(); err != nil {
		st.Release()
		return nil, err
	}
	return st, nil
}

// StateAt resolves a reference and returns its header and the archive at its
// block, for the methods that use the header too.
func (e *Env) StateAt(head *Head, ref chain.BlockRef) (state.StateDB, *evmcore.EvmHeader, error) {
	header, err := e.FindHeader(head, ref)
	if err != nil {
		return nil, nil, err
	}
	st, err := e.StateAtNumber(head, header.Number.Uint64())
	if err != nil {
		return nil, nil, err
	}
	return st, header, nil
}

// Rules returns the rules a block is processed with.
func (e *Env) Rules(ctx context.Context, n uint64) (*opera.Rules, error) {
	return ask(ctx, func() (*opera.Rules, error) { return e.core.Rules(n) })
}

// OpenRules returns the rules the block after the head is processed with: the
// rules of the open epoch, which is not the head's when the head sealed one.
func (e *Env) OpenRules(head *Head) (*opera.Rules, error) {
	return e.Rules(head.ctx, head.Number()+1)
}

// ChainInfo returns the facts of the chain no block selects, as of the served
// head.
func (e *Env) ChainInfo() core.ChainInfo {
	return e.chain.Info()
}

// ChainID is the EIP-155 chain id.
func (e *Env) ChainID() *big.Int {
	return e.chain.Info().ChainID
}

// Signer is the signer of the chain: the latest one, since signers accept the
// transactions of their predecessors.
func (e *Env) Signer() types.Signer {
	return types.LatestSignerForChainID(e.ChainID())
}

// ChainConfig is go-ethereum's configuration of the chain at a block: the
// upgrades in force at that height.
func (e *Env) ChainConfig(n uint64) *params.ChainConfig {
	info := e.chain.Info()
	return opera.CreateTransientEvmChainConfig(info.NetworkID, info.UpgradeHeights, idx.Block(n))
}

// VMConfig is the EVM configuration of the rules of a block.
func (e *Env) VMConfig(ctx context.Context, n uint64) (vm.Config, error) {
	rules, err := e.Rules(ctx, n)
	if err != nil {
		return vm.Config{}, fmt.Errorf("no network rules found for block height %d: %w", n, err)
	}
	return opera.GetVmConfig(*rules), nil
}

// SuggestedGasPrice is the gas price the node suggests for a new transaction
// at the head: its base fee with a margin for short-term fluctuations.
func (e *Env) SuggestedGasPrice(head *Head) *big.Int {
	return gaspricelimits.GetSuggestedGasPriceForNewTransactions(head.Block.BaseFee)
}

// MaxGasLimit is the largest gas limit of a transaction in the open epoch.
func (e *Env) MaxGasLimit(head *Head) (uint64, error) {
	rules, err := e.OpenRules(head)
	if err != nil {
		return 0, err
	}
	return opera.MaxGasLimit(*rules), nil
}

// BlockContext is the EVM's block context of a header of the chain.
// BLOCKHASH is answered by one lookup by number, with the zero hash at and
// above the block, whatever block number an override sets: no client value
// decides how far a lookup walks.
func (e *Env) BlockContext(header *evmcore.EvmHeader) vm.BlockContext {
	ctx := evmcore.NewEVMBlockContext(header, nil, nil)
	ctx.GetHash = e.blockHashes(header.Number.Uint64())
	return ctx
}

func (e *Env) blockHashes(below uint64) vm.GetHashFunc {
	return func(n uint64) common.Hash {
		if n >= below {
			return common.Hash{}
		}
		header, err := e.core.Header(n)
		if err != nil {
			return common.Hash{}
		}
		return header.Hash
	}
}

// NullIfNotFound turns absence into the null answer, and passes every other
// value and error on.
func NullIfNotFound[T any](v T, err error) (any, error) {
	if errors.Is(err, core.ErrNotFound) || errors.Is(err, chain.ErrUnknownHash) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return v, nil
}

// IsMissing reports whether an error is absence: a missing item or an unknown
// hash.
func IsMissing(err error) bool {
	return errors.Is(err, core.ErrNotFound) || errors.Is(err, chain.ErrUnknownHash)
}

// UnknownHashError is the wording of a block hash the node does not know, as
// several legacy handlers phrased it.
func UnknownHashError(hash common.Hash) error {
	return fmt.Errorf("block with hash %s not found", hash.String())
}

// ErrHeaderNotFound is the wording of a missing block of the state getters and
// of eth_call.
var ErrHeaderNotFound = errors.New("header not found")

// HeaderNotFound maps absence to "header not found", the wording of the state
// getters, and passes every other error on.
func HeaderNotFound(err error) error {
	if IsMissing(err) {
		return ErrHeaderNotFound
	}
	return err
}
