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

package bundle

import (
	"bytes"
	"fmt"

	"github.com/0xsoniclabs/sonic/utils/checked"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

// calculateEnvelopeGas calculates the gas limit for an envelope transaction
// based on the given payload and access list.
func calculateEnvelopeGas(
	bundle TransactionBundle,
	payload []byte,
	accessList types.AccessList,
	authList []types.SetCodeAuthorization,
) (uint64, error) {
	dataSize := uint64(len(payload))
	z := uint64(bytes.Count(payload, []byte{0}))
	nz := dataSize - z
	return calculateEnvelopeGasInternal(
		bundle.GetTransactionsInReferencedOrder(),
		z, nz,
		uint64(len(accessList)),
		uint64(accessList.StorageKeys()),
		uint64(len(authList)),
	)
}

// calculateEnvelopeGasInternal is the internal implementation of
// calculateEnvelopeGas that takes pre-calculated values object sizes instead
// of the actual objects. This allows for testing the gas calculation logic
// without needing to construct large objects in memory.
func calculateEnvelopeGasInternal(
	transactions []*types.Transaction,
	numZeroBytesInData uint64,
	numNonZeroBytesInData uint64,
	numAccessListAddresses uint64,
	numAccessListSlots uint64,
	numSetCodeAuthorizations uint64,
) (uint64, error) {
	intrinsic, err := calculateIntrinsicGas(
		numZeroBytesInData,
		numNonZeroBytesInData,
		numAccessListAddresses,
		numAccessListSlots,
		numSetCodeAuthorizations,
	)
	if err != nil {
		return 0, fmt.Errorf("failed intrinsic gas calculation: %w", err)
	}

	floorDataGas, err := calculateFloorDataGas(
		numZeroBytesInData, numNonZeroBytesInData,
	)
	if err != nil {
		return 0, fmt.Errorf("failed floor data gas calculation: %w", err)
	}

	txGas, err := calculateTxGasSum(transactions)
	if err != nil {
		return 0, fmt.Errorf("failed transaction gas sum calculation: %w", err)
	}

	return max(intrinsic, floorDataGas, txGas), nil
}

// calculateIntrinsicGas calculates the intrinsic gas costs of a transaction
// with the given properties. This is a testable version of the
// core.IntrinsicGas function that doesn't require actual data leading to
// overflows to be constructed in memory for tests (which is not possible).
func calculateIntrinsicGas(
	numZeroBytesInData uint64,
	numNonZeroBytesInData uint64,
	numAccessListAddresses uint64,
	numAccessListSlots uint64,
	numSetCodeAuthorizations uint64,
) (uint64, error) {
	// Set the starting gas for the raw transaction
	gas := checked.Uint64(params.TxGas)

	// Add costs for non-zero data bytes.
	gas = checked.Add(gas, checked.Mul(numNonZeroBytesInData, params.TxDataNonZeroGasEIP2028))

	// Add costs for zero data bytes.
	gas = checked.Add(gas, checked.Mul(numZeroBytesInData, params.TxDataZeroGas))

	// Add costs for access list entries.
	gas = checked.Add(gas, checked.Mul(numAccessListAddresses, params.TxAccessListAddressGas))
	gas = checked.Add(gas, checked.Mul(numAccessListSlots, params.TxAccessListStorageKeyGas))

	// Add costs for authorization list entries.
	gas = checked.Add(gas, checked.Mul(numSetCodeAuthorizations, params.CallNewAccountGas))

	return gas.Unwrap()
}

// calculateFloorDataGas calculates the floor data gas costs for a payload with
// the given number of zero and non-zero bytes. This is a testable version of
// the core.FloorDataGas function that doesn't require actual data leading to
// overflows to be constructed in memory for tests (which is not possible).
func calculateFloorDataGas(
	numZeroBytesInData uint64,
	numNonZeroBytesInData uint64,
) (uint64, error) {
	tokens := checked.Add(
		checked.Mul(numNonZeroBytesInData, params.TxTokenPerNonZeroByte),
		numZeroBytesInData,
	)

	// Minimum gas required for a transaction based on its data tokens (EIP-7623).
	return checked.Add(
		params.TxGas, // basic costs
		checked.Mul(tokens, params.TxCostFloorPerToken), // costs per token
	).Unwrap()
}

// calculateTxGasSum sums up the gas limits of the given transactions. An
// error is returned if an overflow occurred.
func calculateTxGasSum(transactions []*types.Transaction) (uint64, error) {
	sum := checked.Uint64(0)
	for _, tx := range transactions {
		if tx != nil {
			sum = checked.Add(sum, tx.Gas())
		}
	}
	return sum.Unwrap()
}
