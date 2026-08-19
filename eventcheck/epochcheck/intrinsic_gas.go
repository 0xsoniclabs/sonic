// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package epochcheck

import (
	"math"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"

	"github.com/0xsoniclabs/sonic/opera"
)

// cantoGasRules are the go-ethereum revision flags of Canto consulted by the
// intrinsic gas computation.
var cantoGasRules = params.Rules{
	IsHomestead: true,
	IsIstanbul:  true,
	IsShanghai:  true,
	IsAmsterdam: true,
}

// minIntrinsicGas returns a lower bound of the intrinsic gas of the given
// transaction, whoever sent it. From Canto on, EIP-2780 prices the base cost
// by sender and value, which are not known here, so the cheapest case, a
// transfer to self, is assumed.
func minIntrinsicGas(tx *types.Transaction, rules opera.Rules) (uint64, error) {
	if !rules.Upgrades.Canto {
		return intrinsicGasLegacy(tx.Data(), tx.AccessList(), tx.To() == nil)
	}
	var from common.Address
	if to := tx.To(); to != nil {
		from = *to
	}
	return core.IntrinsicGas(tx.Data(), tx.AccessList(), tx.SetCodeAuthorizations(), from, tx.To(), nil, cantoGasRules)
}

// intrinsicGasLegacy computes the 'intrinsic gas' for a message with the given data.
// This is an outdated version of
// https://github.com/ethereum/go-ethereum/blob/master/core/state_transition.go#L71
// and is only kept for history reproduction.
// It does not account for init data or authorization lists.
// It also assumes, that the Homestead, Istanbul, and Shanghai forks are always active.
func intrinsicGasLegacy(data []byte, accessList types.AccessList, isContractCreation bool) (uint64, error) {
	// Set the starting gas for the raw transaction
	var gas uint64
	if isContractCreation {
		gas = params.TxGasContractCreation
	} else {
		gas = params.TxGas
	}
	// Bump the required gas by the amount of transactional data
	if len(data) > 0 {
		// Zero and non-zero bytes are priced differently
		var nz uint64
		for _, byt := range data {
			if byt != 0 {
				nz++
			}
		}
		// Make sure we don't exceed uint64 for all data combinations
		if (math.MaxUint64-gas)/params.TxDataNonZeroGasEIP2028 < nz {
			return 0, vm.ErrOutOfGas
		}
		gas += nz * params.TxDataNonZeroGasEIP2028

		z := uint64(len(data)) - nz
		if (math.MaxUint64-gas)/params.TxDataZeroGas < z {
			return 0, vm.ErrGasUintOverflow
		}
		gas += z * params.TxDataZeroGas
	}
	if accessList != nil {
		gas += uint64(len(accessList)) * params.TxAccessListAddressGas
		gas += uint64(accessList.StorageKeys()) * params.TxAccessListStorageKeyGas
	}
	return gas, nil
}
