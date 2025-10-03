// Copyright 2025 Sonic Operations Ltd
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

//go:generate solc --optimize --optimize-runs 200 --bin --bin-runtime subsidies_registry.sol --abi subsidies_registry.sol -o build --overwrite
//go:generate abigen --bin=build/SubsidiesRegistry.bin --abi=build/SubsidiesRegistry.abi --pkg=registry --out=subsidies_registry_abigen.go
//go:generate cp build/SubsidiesRegistry.bin-runtime subsidies_contract.bin

// GetAddress returns the address of the deployed SubsidiesRegistry.
func GetAddress() common.Address {
	return common.Address(contractAddress[:])
}

// GetCode returns the on-chain bytecode of the SubsidiesRegistry contract.
func GetCode() []byte {
	return bytes.Clone(registryCode)
}

// GetGasConfigFunctionSelector is the function selector of the `getGasConfig`
// function in the SubsidiesRegistry contract.
const GetGasConfigFunctionSelector = 0x4b5c54c0

// ChooseFundFunctionSelector is the function selector of the `chooseFund`
// function in the SubsidiesRegistry contract.
const ChooseFundFunctionSelector = 0x399f59ca

// DeductFeesFunctionSelector is the function selector of the `deductFees`
// function in the SubsidiesRegistry contract.
const DeductFeesFunctionSelector = 0xb9ed9f26

// GasLimitForGetGasConfig is the gas limit to be used when calling the
// `getGasConfig` function in the SubsidiesRegistry contract.
const GasLimitForGetGasConfig = 50_000

// ------------------------------ Internals ------------------------------------

// contractAddress is the address of the deployed SubsidiesRegistry contract.
// On the Sonic mainnet, a proxy contract for the registry was deployed by
// https://sonicscan.org//address/0x7d0E23398b6CA0eC7Cdb5b5Aad7F1b11215012d2
var contractAddress = hexutil.MustDecode("0x7d0E23398b6CA0eC7Cdb5b5Aad7F1b11215012d2")

//go:embed subsidies_contract.bin
var registryCodeInHex string
var registryCode []byte = hexutils.HexToBytes(registryCodeInHex)
