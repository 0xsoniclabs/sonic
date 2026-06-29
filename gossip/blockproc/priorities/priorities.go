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
	"math/big"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/priorities/registry"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
)

//go:generate mockgen -source=priorities.go -destination=priorities_mock.go -package=priorities

// Priority is the result of a getPriority query for a single transaction.
//
// Level zero means the transaction is not prioritized. A higher level forms an
// earlier partition (scheduled before lower levels). Weight breaks ties within a
// level (higher first). Id identifies the entity the transaction belongs to and
// is used for per-entity rate limiting. The semantics of Id are opaque to this
// code and only interpreted by the registry.
type Priority struct {
	Level  *big.Int
	Weight *big.Int
	Id     [32]byte
}

// IsPrioritized reports whether the transaction has a non-zero priority level.
func (p Priority) IsPrioritized() bool {
	return p.Level != nil && p.Level.Sign() > 0
}

// Config holds the per-entity rate limits returned by the registry's
// getPriorityConfig function.
type Config struct {
	// MaxTxsPerEntityPerBlock bounds how many transactions of one entity may be
	// prioritized within a single block (authoritative, enforced at block
	// formation).
	MaxTxsPerEntityPerBlock uint64
	// MaxTxsPerEntityPerEvent bounds how many transactions of one entity a
	// validator eagerly includes in a single emitted event (best-effort hint).
	MaxTxsPerEntityPerEvent uint64
}

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
		return zeroPriority(), nil
	}
	if tx == nil {
		return zeroPriority(), fmt.Errorf("nil transaction")
	}

	sender, err := types.Sender(signer, tx)
	if err != nil {
		return zeroPriority(), fmt.Errorf("failed to derive sender: %w", err)
	}

	input, err := createGetPriorityInput(sender, tx)
	if err != nil {
		return zeroPriority(), fmt.Errorf("failed to create input for priority registry call: %w", err)
	}

	caller := common.Address{}
	target := registry.GetAddress()
	result, _, err := vm.Call(caller, target, input, registry.GasLimitForGetPriority, uint256.NewInt(0))
	if err != nil {
		return zeroPriority(), fmt.Errorf("EVM call failed: %w", err)
	}
	if len(result) == 0 {
		return zeroPriority(), fmt.Errorf("priority registry contract not found")
	}

	return parseGetPriorityResult(result)
}

// GetConfig queries the priority registry for the current per-entity rate-limit
// configuration. If the feature is disabled it returns a zero Config.
//
// On the consensus path a returned error must be handled deterministically by
// falling back to FallbackConfig (see Prioritize callers).
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
	if len(result) == 0 {
		return Config{}, fmt.Errorf("priority registry contract not found")
	}

	return parseGetPriorityConfigResult(result)
}

// FallbackConfig is the deterministic configuration used when the registry
// configuration cannot be read while the feature is enabled. Zero limits mean
// no transaction is prioritized, the safest degradation: every validator that
// fails to read the config produces the same (un-prioritized) ordering.
var FallbackConfig = Config{
	MaxTxsPerEntityPerBlock: 0,
	MaxTxsPerEntityPerEvent: 0,
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

// parseGetPriorityResult decodes the (uint256 level, uint256 weight, bytes32 id)
// response of the getPriority call. The result must be exactly three 32-byte
// words; any other length is rejected.
func parseGetPriorityResult(data []byte) (Priority, error) {
	if len(data) != 3*32 {
		return zeroPriority(), fmt.Errorf("invalid result length from getPriority call: %d", len(data))
	}
	level := new(big.Int).SetBytes(data[0:32])
	weight := new(big.Int).SetBytes(data[32:64])
	id := [32]byte(data[64:96])
	return Priority{Level: level, Weight: weight, Id: id}, nil
}

// parseGetPriorityConfigResult decodes the
// (uint256 maxTxsPerEntityPerBlock, uint256 maxTxsPerEntityPerEvent) response of
// the getPriorityConfig call. The result must be exactly two 32-byte words, each
// fitting into a uint64.
func parseGetPriorityConfigResult(data []byte) (Config, error) {
	if len(data) != 2*32 {
		return Config{}, fmt.Errorf("invalid result length from getPriorityConfig call: %d", len(data))
	}
	type bytes24 [24]byte
	zero := bytes24{}
	if bytes24(data[0:24]) != zero || bytes24(data[32:56]) != zero {
		return Config{}, fmt.Errorf("invalid result from getPriorityConfig call, values do not fit into uint64")
	}
	return Config{
		MaxTxsPerEntityPerBlock: binary.BigEndian.Uint64(data[24:32]),
		MaxTxsPerEntityPerEvent: binary.BigEndian.Uint64(data[56:64]),
	}, nil
}

// zeroPriority returns a normalized non-prioritized Priority.
func zeroPriority() Priority {
	return Priority{Level: big.NewInt(0), Weight: big.NewInt(0)}
}
