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
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"

	"github.com/0xsoniclabs/sonic/inter/state"
)

// OverrideAccount is what a call changes of an account before it runs. State
// replaces the whole storage, StateDiff changes some slots; an account may
// not have both.
type OverrideAccount struct {
	Nonce     *hexutil.Uint64              `json:"nonce"`
	Code      *hexutil.Bytes               `json:"code"`
	Balance   **hexutil.U256               `json:"balance"`
	State     *map[common.Hash]common.Hash `json:"state"`
	StateDiff *map[common.Hash]common.Hash `json:"stateDiff"`
}

// StateOverride are the accounts a call overrides.
type StateOverride map[common.Address]OverrideAccount

// Apply writes the overrides into the state and ends a transaction, so that
// the call sees them as the committed state of a preceding one. Absent
// overrides leave the state as it is.
func (overrides StateOverride) Apply(statedb state.StateDB) error {
	if overrides == nil {
		return nil
	}
	for address, account := range overrides {
		if account.Nonce != nil {
			statedb.SetNonce(address, uint64(*account.Nonce), tracing.NonceChangeUnspecified)
		}
		if account.Code != nil {
			statedb.SetCode(address, *account.Code, tracing.CodeChangeUnspecified)
		}
		if account.Balance != nil {
			statedb.SetBalance(address, (*uint256.Int)(*account.Balance))
		}
		if account.State != nil && account.StateDiff != nil {
			return fmt.Errorf("account %s has both 'state' and 'stateDiff'", address.Hex())
		}
		if account.State != nil {
			statedb.SetStorage(address, *account.State)
		}
		if account.StateDiff != nil {
			for key, value := range *account.StateDiff {
				statedb.SetState(address, key, value)
			}
		}
	}
	statedb.EndTransaction()
	return nil
}

// BlockOverrides are the fields of the block a call changes. The fields have
// no JSON names, as in the reference: they match any case of their names.
type BlockOverrides struct {
	Number      *hexutil.Big
	Difficulty  *hexutil.Big
	Time        *hexutil.Uint64
	GasLimit    *hexutil.Uint64
	Coinbase    *common.Address
	Random      *common.Hash
	BaseFee     *hexutil.Big
	BlobBaseFee *hexutil.Big
}

// Apply writes the overrides into a block context.
func (overrides *BlockOverrides) Apply(context *vm.BlockContext) {
	if overrides == nil {
		return
	}
	if overrides.Number != nil {
		context.BlockNumber = overrides.Number.ToInt()
	}
	if overrides.Difficulty != nil {
		context.Difficulty = overrides.Difficulty.ToInt()
	}
	if overrides.Time != nil {
		context.Time = uint64(*overrides.Time)
	}
	if overrides.GasLimit != nil {
		context.GasLimit = uint64(*overrides.GasLimit)
	}
	if overrides.Coinbase != nil {
		context.Coinbase = *overrides.Coinbase
	}
	if overrides.Random != nil {
		context.Random = overrides.Random
	}
	if overrides.BaseFee != nil {
		context.BaseFee = overrides.BaseFee.ToInt()
	}
	if overrides.BlobBaseFee != nil {
		context.BlobBaseFee = overrides.BlobBaseFee.ToInt()
	}
}
