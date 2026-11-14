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

	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"
)

// OverrideAccount is what a call replaces of an account before it runs.
// State replaces the whole storage, StateDiff the slots it names; an account
// has at most one of them.
type OverrideAccount struct {
	Nonce     *hexutil.Uint64              `json:"nonce"`
	Code      *hexutil.Bytes               `json:"code"`
	Balance   **hexutil.U256               `json:"balance"`
	State     *map[common.Hash]common.Hash `json:"state"`
	StateDiff *map[common.Hash]common.Hash `json:"stateDiff"`
}

// StateOverride replaces accounts of the state a call runs on.
type StateOverride map[common.Address]OverrideAccount

// Apply writes the overrides into the state, which then ends a transaction,
// so that calls see them as committed state.
func (o *StateOverride) Apply(stateDB state.StateDB) error {
	if o == nil {
		return nil
	}
	for address, account := range *o {
		if account.Nonce != nil {
			stateDB.SetNonce(address, uint64(*account.Nonce), tracing.NonceChangeUnspecified)
		}
		if account.Code != nil {
			stateDB.SetCode(address, *account.Code, tracing.CodeChangeUnspecified)
		}
		if account.Balance != nil {
			stateDB.SetBalance(address, (*uint256.Int)(*account.Balance))
		}
		if account.State != nil && account.StateDiff != nil {
			return fmt.Errorf("account %s has both 'state' and 'stateDiff'", address.Hex())
		}
		if account.State != nil {
			stateDB.SetStorage(address, *account.State)
		}
		if account.StateDiff != nil {
			for key, value := range *account.StateDiff {
				stateDB.SetState(address, key, value)
			}
		}
	}
	stateDB.EndTransaction()
	return nil
}

// BlockOverrides replaces fields of the block context a call runs in.
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

// apply writes the overrides into a block context.
func (o *BlockOverrides) apply(blockContext *vm.BlockContext) {
	if o == nil {
		return
	}
	if o.Number != nil {
		blockContext.BlockNumber = o.Number.ToInt()
	}
	if o.Difficulty != nil {
		blockContext.Difficulty = o.Difficulty.ToInt()
	}
	if o.Time != nil {
		blockContext.Time = uint64(*o.Time)
	}
	if o.GasLimit != nil {
		blockContext.GasLimit = uint64(*o.GasLimit)
	}
	if o.Coinbase != nil {
		blockContext.Coinbase = *o.Coinbase
	}
	if o.Random != nil {
		blockContext.Random = o.Random
	}
	if o.BaseFee != nil {
		blockContext.BaseFee = o.BaseFee.ToInt()
	}
	if o.BlobBaseFee != nil {
		blockContext.BlobBaseFee = o.BlobBaseFee.ToInt()
	}
}
