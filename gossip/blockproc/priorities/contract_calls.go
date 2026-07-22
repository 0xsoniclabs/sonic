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

// Package priorities implements the transaction-priorities feature: querying an
// on-chain registry contract to determine, per transaction, a priority level, a
// weight, and an entity id, and using those to order transactions during block
// formation. See transaction_priorities.md for the design.
package priorities

import (
	"encoding/binary"
	"fmt"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/priorities/registry"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
)

//go:generate mockgen -source=contract_calls.go -destination=contract_calls_mock.go -package=priorities

// VirtualMachine is a minimal interface for an EVM instance that can be used to
// query the priority registry contract. It is satisfied directly by *vm.EVM.
type VirtualMachine interface {
	Call(
		from common.Address,
		to common.Address,
		input []byte,
		gas uint64,
		value *uint256.Int,
	) (
		result []byte,
		gasLeft uint64,
		err error,
	)
}

// GetPriority queries the priority registry for the given transaction. If the
// transaction-priorities feature is disabled it returns a zero (non-prioritized)
// Priority without touching the EVM.
//
// Callers on the consensus path must treat any returned error as
// "not prioritized" (see Prioritize); the error is returned for logging and
// metrics only and must never abort block formation.
func GetPriority(
	upgrades opera.Upgrades,
	vm VirtualMachine,
	signer types.Signer,
	tx *types.Transaction,
) (Priority, error) {
	if !upgrades.TransactionPriorities {
		return Priority{}, nil
	}
	if tx == nil {
		return Priority{}, fmt.Errorf("nil transaction")
	}

	sender, err := types.Sender(signer, tx)
	if err != nil {
		return Priority{}, fmt.Errorf("failed to derive sender: %w", err)
	}

	input, err := createGetPriorityInput(sender, tx)
	if err != nil {
		return Priority{}, fmt.Errorf("failed to create input for priority registry call: %w", err)
	}

	caller := common.Address{}
	target := registry.GetAddress()
	result, _, err := vm.Call(caller, target, input, registry.GasLimitForGetPriority, uint256.NewInt(0))
	if err != nil {
		return Priority{}, fmt.Errorf("EVM call failed: %w", err)
	}
	if result == nil {
		return Priority{}, fmt.Errorf("priority registry contract not found")
	}

	return parseGetPriorityResult(result)
}

// GetConfig queries the priority registry for the current per-entity rate-limit
// configuration. If the feature is disabled it returns a zero Config.
func GetConfig(upgrades opera.Upgrades, vm VirtualMachine) (Config, error) {
	if !upgrades.TransactionPriorities {
		return Config{}, nil
	}

	input := make([]byte, 4) // function selector only
	binary.BigEndian.PutUint32(input, registry.GetPriorityConfigFunctionSelector)

	caller := common.Address{}
	target := registry.GetAddress()
	result, _, err := vm.Call(caller, target, input, registry.GasLimitForGetPriorityConfig, uint256.NewInt(0))
	if err != nil {
		return Config{}, fmt.Errorf("EVM call failed: %w", err)
	}
	if result == nil {
		return Config{}, fmt.Errorf("priority registry contract not found")
	}

	return parseGetPriorityConfigResult(result)
}

// FallbackConfig is the deterministic configuration used when the registry
// configuration cannot be read while the feature is enabled. Zero limits mean
// no transaction is prioritized, the safest degradation: every validator that
// fails to read the config produces the same (un-prioritized) ordering.
var FallbackConfig = Config{
	MaxGasPerEntityPerBlock:          0,
	MaxPiggybackTxsPerEntityPerEvent: 0,
}

// GetConfigOrFallback queries the registry configuration via GetConfig and, on
// error, returns the deterministic FallbackConfig.
func GetConfigOrFallback(upgrades opera.Upgrades, vm VirtualMachine) Config {
	cfg, err := GetConfig(upgrades, vm)
	if err != nil {
		return FallbackConfig
	}
	return cfg
}

// --- ABI encoding / decoding (hand-rolled for determinism) ---

// createGetPriorityInput encodes the calldata for the getPriority call:
// getPriority(address from, address to, uint256 value, uint256 nonce,
// bytes data, uint256 gas).
func createGetPriorityInput(sender common.Address, tx *types.Transaction) ([]byte, error) {
	if tx == nil {
		return nil, fmt.Errorf("nil transaction")
	}

	to := common.Address{}
	if tx.To() != nil {
		to = *tx.To()
	}

	addressPadding := [12]byte{}
	uint64Padding := [24]byte{}

	input := []byte{}
	input = binary.BigEndian.AppendUint32(input, registry.GetPriorityFunctionSelector)

	// from, to (left-padded addresses)
	input = append(input, addressPadding[:]...)
	input = append(input, sender[:]...)
	input = append(input, addressPadding[:]...)
	input = append(input, to[:]...)

	// value
	input = append(input, tx.Value().FillBytes(make([]byte, 32))...)

	// nonce
	input = append(input, uint64Padding[:]...)
	input = binary.BigEndian.AppendUint64(input, tx.Nonce())

	// data: offset of the dynamic parameter (6 head slots × 32 bytes)
	input = append(input, uint64Padding[:]...)
	input = binary.BigEndian.AppendUint64(input, 32*6)

	// gas (the transaction gas limit)
	input = append(input, uint64Padding[:]...)
	input = binary.BigEndian.AppendUint64(input, tx.Gas())

	// dynamic data: length prefix + padded bytes
	input = append(input, uint64Padding[:]...)
	input = binary.BigEndian.AppendUint64(input, uint64(len(tx.Data())))
	input = append(input, tx.Data()...)
	if rem := len(tx.Data()) % 32; rem != 0 {
		input = append(input, make([]byte, 32-rem)...)
	}

	return input, nil
}

// parseGetPriorityResult decodes the (uint64 level, uint64 weight, uint128 id)
// response of the getPriority call. The result must be exactly three 32-byte
// words with level and weight fitting into a uint64 and id into a uint128; any
// other shape is rejected.
func parseGetPriorityResult(data []byte) (Priority, error) {
	if len(data) != 3*32 {
		return Priority{}, fmt.Errorf("invalid result length from getPriority call: %d", len(data))
	}
	if !allZero(data[0:24]) || !allZero(data[32:56]) || !allZero(data[64:80]) {
		return Priority{}, fmt.Errorf("invalid result from getPriority call")
	}
	return Priority{
		Level:  binary.BigEndian.Uint64(data[24:32]),
		Weight: binary.BigEndian.Uint64(data[56:64]),
		ID:     [16]byte(data[80:96]),
	}, nil
}

// parseGetPriorityConfigResult decodes the
// (uint256 maxGasPerEntityPerBlock, uint256 maxPiggybackTxsPerEntityPerEvent) response of
// the getPriorityConfig call. The result must be exactly two 32-byte words, each
// fitting into a uint64.
func parseGetPriorityConfigResult(data []byte) (Config, error) {
	if len(data) != 2*32 {
		return Config{}, fmt.Errorf("invalid result length from getPriorityConfig call: %d", len(data))
	}
	if !allZero(data[0:24]) || !allZero(data[32:56]) {
		return Config{}, fmt.Errorf("invalid result from getPriorityConfig call, values do not fit into uint64")
	}
	return Config{
		MaxGasPerEntityPerBlock:          binary.BigEndian.Uint64(data[24:32]),
		MaxPiggybackTxsPerEntityPerEvent: binary.BigEndian.Uint64(data[56:64]),
	}, nil
}

// allZero reports whether every byte in b is zero.
func allZero(b []byte) bool {
	for _, x := range b {
		if x != 0 {
			return false
		}
	}
	return true
}
