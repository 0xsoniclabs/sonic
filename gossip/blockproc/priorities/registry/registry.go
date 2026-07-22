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

package registry

import (
	"bytes"
	_ "embed"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/keycard-go/hexutils"
)

//go:generate solc --optimize --optimize-runs 200 --bin --bin-runtime --abi priorities_registry.sol -o build --overwrite
//go:generate abigen --bin=build/PriorityRegistry.bin --abi=build/PriorityRegistry.abi --pkg=registry --out=priorities_registry_abigen.go
//go:generate cp build/PriorityRegistry.bin-runtime priorities_contract.bin

// GetAddress returns the address of the deployed PriorityRegistry.
func GetAddress() common.Address {
	return common.Address(contractAddress[:])
}

// GetCode returns the on-chain bytecode of the PriorityRegistry contract.
func GetCode() []byte {
	return bytes.Clone(registryCode)
}

// GetPriorityFunctionSelector is the function selector of the `getPriority`
// function in the PriorityRegistry contract:
// getPriority(address,address,uint256,uint256,bytes,uint256).
const GetPriorityFunctionSelector = 0xd9dceeb8

// GetPriorityConfigFunctionSelector is the function selector of the
// `getPriorityConfig` function in the PriorityRegistry contract.
const GetPriorityConfigFunctionSelector = 0x928461bd

// GasLimitForGetPriority is the fixed gas limit used when calling the
// `getPriority` function. It is a deterministic constant (not registry-supplied)
// so that the per-transaction query cost on the block-formation path is bounded
// independently of any contract configuration.
const GasLimitForGetPriority = 100_000

// GasLimitForGetPriorityConfig is the fixed gas limit used when calling the
// `getPriorityConfig` function.
const GasLimitForGetPriorityConfig = 50_000

// ------------------------------ Internals ------------------------------------

// contractAddress is the address of the deployed PriorityRegistry contract.
//
// PLACEHOLDER: the priority registry is not yet deployed on any public network.
// The bytes spell "priority" followed by zero padding so the placeholder is
// recognizable. The final mainnet proxy address must be fixed before activation.
var contractAddress = hexutil.MustDecode("0x7072696f72697479000000000000000000000000")

//go:embed priorities_contract.bin
var registryCodeInHex string
var registryCode []byte = hexutils.HexToBytes(registryCodeInHex)
