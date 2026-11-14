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
	"errors"
	"fmt"
	"math/big"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/gasprice/gaspricelimits"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/ethereum/go-ethereum/core/vm"
)

// errHeaderNotFound words a block without state, as the state reads of the
// legacy stack word it.
var errHeaderNotFound = errors.New("header not found")

// OpenState opens the archive at the block a reference names, for a read
// that needs no header of it. The caller releases it.
func (e *Environment) OpenState(head *evmcore.EvmBlock, reference chain.BlockReference) (state.StateDB, error) {
	number, err := e.resolveState(head, reference)
	if err != nil {
		return nil, err
	}
	return e.openState(number, reference)
}

// StateAt opens the archive at the block a reference names and reads its
// header. The caller releases the state.
func (e *Environment) StateAt(head *evmcore.EvmBlock, reference chain.BlockReference) (state.StateDB, *evmcore.EvmHeader, error) {
	number, err := e.resolveState(head, reference)
	if err != nil {
		return nil, nil, err
	}
	header, err := e.HeaderAt(head, number)
	if errors.Is(err, core.ErrNotFound) {
		return nil, nil, errHeaderNotFound
	}
	if err != nil {
		return nil, nil, err
	}
	stateDB, err := e.openState(number, reference)
	if err != nil {
		return nil, nil, err
	}
	return stateDB, header, nil
}

// resolveState resolves a reference of a state read, wording a block that is
// not there as "header not found".
func (e *Environment) resolveState(head *evmcore.EvmBlock, reference chain.BlockReference) (uint64, error) {
	number, err := e.Resolve(head, reference)
	if errors.Is(err, chain.ErrUnknownHash) {
		return 0, errHeaderNotFound
	}
	if err == nil && number > head.NumberU64() {
		return 0, errHeaderNotFound
	}
	return number, err
}

// openState opens the archive at a block. On a node without an archive a
// read of the head by tag fails as the legacy stack's, which looks for the
// newest archived block first.
func (e *Environment) openState(number uint64, reference chain.BlockReference) (state.StateDB, error) {
	stateDB, err := e.reader.State(number)
	switch {
	case errors.Is(err, core.ErrNotFound):
		return nil, errHeaderNotFound
	case errors.Is(err, core.ErrNoArchive) && namesTheHead(reference):
		return nil, fmt.Errorf("failed to get latest block number; failed to get latest archive block; %v", err)
	}
	return stateDB, err
}

// namesTheHead tells whether a reference is a tag of the head.
func namesTheHead(reference chain.BlockReference) bool {
	return reference.Number != nil && *reference.Number < 0 && *reference.Number != chain.EarliestBlockNumber
}

// Rules are the network rules of a block; above the committed head those of
// the open epoch.
func (e *Environment) Rules(number uint64) (*opera.Rules, error) {
	rules, err := e.reader.Rules(number)
	if errors.Is(err, core.ErrNotFound) {
		return nil, fmt.Errorf("no network rules found for block height %d", number)
	}
	return rules, err
}

// VMConfig is the configuration of the EVM running on top of a block.
func (e *Environment) VMConfig(number uint64) (vm.Config, error) {
	rules, err := e.Rules(number)
	if err != nil {
		return vm.Config{}, err
	}
	return opera.GetVmConfig(*rules), nil
}

// CurrentRules are the rules the block after the head runs under.
func (e *Environment) CurrentRules(head *evmcore.EvmBlock) (*opera.Rules, error) {
	return e.Rules(head.NumberU64() + 1)
}

// MaxGasLimit is the most gas a transaction of the block after the head may
// use: the gas of an event less the gas of an empty one.
func (e *Environment) MaxGasLimit(head *evmcore.EvmBlock) (uint64, error) {
	rules, err := e.CurrentRules(head)
	if err != nil {
		return 0, err
	}
	gas := rules.Economy.Gas
	emptyEventGas := gas.EventGas +
		uint64(rules.Dag.MaxParents-rules.Dag.MaxFreeParents)*gas.ParentGas +
		uint64(rules.Dag.MaxExtraData)*gas.ExtraDataGas
	if gas.MaxEventGas < emptyEventGas {
		return 0, nil
	}
	return gas.MaxEventGas - emptyEventGas, nil
}

// SuggestedGasPrice is the gas price suggested for a new transaction: the
// head's base fee with a margin for the next blocks.
func (e *Environment) SuggestedGasPrice(head *evmcore.EvmBlock) *big.Int {
	return gaspricelimits.GetSuggestedGasPriceForNewTransactions(head.Header().BaseFee)
}
