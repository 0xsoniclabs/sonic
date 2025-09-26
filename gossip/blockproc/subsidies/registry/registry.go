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

// ChooseFundFunctionSelector is the function selector of the `chooseFund`
// function in the SubsidiesRegistry contract.
const ChooseFundFunctionSelector = 0x399f59ca

// DeductFeesFunctionSelector is the function selector of the `deductFees` function
// in the SubsidiesRegistry contract.
const DeductFeesFunctionSelector = 0xb9ed9f26

// GasLimitForChooseFundCall is the gas limit to be used when calling the
// `chooseFund` function in the SubsidiesRegistry contract.
// TODO: reevaluate this value once contract is settled.
const GasLimitForChooseFundCall = 100_000

// GasLimitForDeductFeesCall is the gas limit to be used when calling the
// `deductFees` function in the SubsidiesRegistry contract.
// TODO: reevaluate this value once contract is settled.
const GasLimitForDeductFeesCall = 100_000

// ------------------------------ Internals ------------------------------------

// contractAddress is the address of the deployed SubsidiesRegistry contract.
// On the Sonic mainnet, a proxy contract for the registry was deployed by
// https://sonicscan.org/tx/0xef62408baae42d01e9ef0a44fcb0dffbbd5531f48bc47c1e411e15d6719c7229
var contractAddress = hexutil.MustDecode("0x000DE63B7f02E16DF54F1Bf2147729591b296F77")

//go:embed subsidies_contract.bin
var registryCodeInHex string
var registryCode []byte = hexutils.HexToBytes(registryCodeInHex)
