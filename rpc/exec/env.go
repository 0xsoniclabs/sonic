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

package exec

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/oracle"
)

// ErrHeaderNotFound is the answer of the methods that read state at a block the
// node does not have.
var ErrHeaderNotFound = errors.New("header not found")

// Env is what the methods that execute transactions share: the node interface,
// the served head, the limits and the gas price oracle. It is created once and
// is safe for concurrent use.
type Env struct {
	Core   core.Core
	Chain  *dispatch.Chain
	Cfg    dispatch.Config
	Oracle *oracle.Oracle
}

// NewEnv creates the environment of the execution methods.
func NewEnv(chain *dispatch.Chain, o *oracle.Oracle) *Env {
	return &Env{Core: chain.Core(), Chain: chain, Cfg: chain.Config(), Oracle: o}
}

// ChainID returns the chain id.
func (e *Env) ChainID() *big.Int { return new(big.Int).SetUint64(e.Core.ChainInfo().ChainID) }

// ChainConfig returns the chain configuration in force at a block height.
func (e *Env) ChainConfig(height uint64) *params.ChainConfig {
	return ChainConfig(e.Core.ChainInfo(), height)
}

// MinGasPrice is the price the node suggests for new transactions, derived
// from the base fee of the head.
func (e *Env) MinGasPrice(head dispatch.Head) *big.Int {
	var baseFee *big.Int
	if head.Header != nil {
		baseFee = head.Header.BaseFee
	}
	return oracle.SuggestedGasPrice(baseFee)
}

// MaxGasLimit is the largest gas limit of a transaction under the rules the
// block after the head is processed with: the event gas limit less the gas an
// empty event costs. An epoch sealed by the head applies at once.
func (e *Env) MaxGasLimit(head dispatch.Head) uint64 {
	rules, err := e.Core.Rules(head.Number + 1)
	if err != nil || rules == nil {
		return 0
	}
	maxEmptyEventGas := rules.Economy.Gas.EventGas +
		uint64(rules.Dag.MaxParents-rules.Dag.MaxFreeParents)*rules.Economy.Gas.ParentGas +
		uint64(rules.Dag.MaxExtraData)*rules.Economy.Gas.ExtraDataGas
	if rules.Economy.Gas.MaxEventGas < maxEmptyEventGas {
		return 0
	}
	return rules.Economy.Gas.MaxEventGas - maxEmptyEventGas
}

// NetworkRules returns the rules in force at a block.
func (e *Env) NetworkRules(number uint64) (*opera.Rules, error) {
	rules, err := e.Core.Rules(number)
	if errors.Is(err, core.ErrNotFound) {
		return nil, nil
	}
	return rules, err
}

// VMConfig resolves the VM configuration of a block from its rules.
func (e *Env) VMConfig(number uint64) (vm.Config, error) {
	rules, err := e.NetworkRules(number)
	if err != nil {
		return vm.Config{}, err
	}
	if rules == nil {
		return vm.Config{}, fmt.Errorf("no network rules found for block height %d", number)
	}
	return opera.GetVmConfig(*rules), nil
}

// ResolveNumber resolves the block number or tag of a reference against the
// head; a hash resolves to the number of its block. The second result is
// false if the reference names no block of the served chain.
func (e *Env) ResolveNumber(head dispatch.Head, ref dispatch.BlockRef) (uint64, bool) {
	switch {
	case ref.Number != nil:
		n, err := e.Chain.Number(head, *ref.Number)
		if err != nil || n > head.Number {
			return 0, false
		}
		return n, true
	case ref.Hash != nil:
		n, ok := e.Core.BlockNumberByHash(*ref.Hash)
		if !ok || n > head.Number {
			return 0, false
		}
		return n, true
	}
	return 0, false
}

// header returns the header of a block of the served chain; nil if there is none.
func (e *Env) header(head dispatch.Head, number uint64) (*evmcore.EvmHeader, error) {
	if number > head.Number {
		return nil, nil
	}
	if number == head.Number && head.Header != nil {
		h := *head.Header
		return &h, nil
	}
	h, err := e.Core.Header(number)
	if errors.Is(err, core.ErrNotFound) {
		return nil, nil
	}
	return h, err
}

// HeaderByNumber returns the header a number or tag names; nil if there is none.
func (e *Env) HeaderByNumber(head dispatch.Head, bn dispatch.BlockNumber) (*evmcore.EvmHeader, error) {
	n, err := e.Chain.Number(head, bn)
	if err != nil {
		return nil, nil
	}
	return e.header(head, n)
}

// HeaderByHash returns the header with a hash; unknown hashes are an error.
func (e *Env) HeaderByHash(head dispatch.Head, hash common.Hash) (*evmcore.EvmHeader, error) {
	n, ok := e.Core.BlockNumberByHash(hash)
	if !ok {
		return nil, fmt.Errorf("block with hash %s not found", hash.String())
	}
	return e.header(head, n)
}

// StateAt opens the archived state of the block a reference names, together
// with the header of the block. The caller owns the state and must release it
// exactly once, on the line after the call.
func (e *Env) StateAt(head dispatch.Head, ref dispatch.BlockRef) (state.StateDB, *evmcore.EvmHeader, error) {
	n, ok := e.ResolveNumber(head, ref)
	if !ok {
		if ref.Number == nil && ref.Hash == nil {
			return nil, nil, errors.New("unknown header selector")
		}
		return nil, nil, ErrHeaderNotFound
	}
	header, err := e.header(head, n)
	if err != nil {
		return nil, nil, err
	}
	if header == nil {
		return nil, nil, ErrHeaderNotFound
	}
	st, err := e.Core.State(n)
	if errors.Is(err, core.ErrNotFound) {
		return nil, nil, ErrHeaderNotFound
	}
	if err != nil {
		return nil, nil, err
	}
	if st == nil {
		return nil, nil, ErrHeaderNotFound
	}
	return st, header, nil
}

// hashChain gives the EVM the hashes of the blocks BLOCKHASH asks for.
type hashChain struct {
	e    *Env
	head dispatch.Head
}

// Header implements evmcore.DummyChain. It returns nil if the block is unknown
// or its hash is not the given one.
func (c hashChain) Header(hash common.Hash, number uint64) *evmcore.EvmHeader {
	header, err := c.e.header(c.head, number)
	if header == nil || err != nil || header.Hash != hash {
		return nil
	}
	return header
}

// BlockContext returns the block context of the EVM for a header.
func (e *Env) BlockContext(head dispatch.Head, header *evmcore.EvmHeader) vm.BlockContext {
	return evmcore.NewEVMBlockContext(header, hashChain{e, head}, nil)
}

// NewEVM builds the EVM of a block. The chain configuration is that of the
// block number in the block context, which a block override may change. A nil
// VM configuration is resolved from the rules of the header's block.
func (e *Env) NewEVM(head dispatch.Head, statedb vm.StateDB, header *evmcore.EvmHeader, vmConfig *vm.Config, blockCtx *vm.BlockContext) (*vm.EVM, error) {
	if vmConfig == nil {
		number := header.Number.Uint64()
		rules, err := e.NetworkRules(number)
		if err != nil {
			return nil, fmt.Errorf("failed to get network rules for block %d: %w", number, err)
		}
		if rules == nil {
			return nil, fmt.Errorf("network rules for block %d are not found", number)
		}
		config := opera.GetVmConfig(*rules)
		vmConfig = &config
	}
	var context vm.BlockContext
	if blockCtx == nil {
		context = e.BlockContext(head, header)
	} else {
		context = *blockCtx
	}
	config := e.ChainConfig(context.BlockNumber.Uint64())
	return vm.NewEVM(context, statedb, config, *vmConfig), nil
}

// NewEVMWithConfig builds an EVM with the given chain configuration instead of
// the one of the block number in the block context.
func (e *Env) NewEVMWithConfig(statedb vm.StateDB, vmConfig vm.Config, blockCtx vm.BlockContext, chainConfig *params.ChainConfig) *vm.EVM {
	return vm.NewEVM(blockCtx, statedb, chainConfig, vmConfig)
}

// Cancel makes the EVM stop when the context ends. The returned function
// releases the registration and must be called when the EVM is done.
func Cancel(ctx context.Context, evm *vm.EVM) (stop func() bool) {
	return context.AfterFunc(ctx, evm.Cancel)
}
