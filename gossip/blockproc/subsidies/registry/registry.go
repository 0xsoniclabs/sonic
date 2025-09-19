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
	"math/big"

	"github.com/0xsoniclabs/sonic/opera/contracts/sfc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/status-im/keycard-go/hexutils"
)

//go:generate solc --optimize --optimize-runs 200 --bin --bin-runtime subsidies_registry.sol --abi subsidies_registry.sol -o build --overwrite
//go:generate abigen --bin=build/SubsidiesRegistry.bin --abi=build/SubsidiesRegistry.abi --pkg=registry --out=subsidies_registry.go
//go:generate cp build/SubsidiesRegistry.bin-runtime subsidies_contract.bin

const IsCoveredFunctionSelector = 0x36a656a7
const DeductFeesFunctionSelector = 0x944557d6

// The deployment transaction was generated to be issued by an EOA account
// which's private key got discarded afterwards. The contract is thus
// deployed at an address that cannot be pre-occupied by anybody.
//
// For developers: if you need to re-generate the deployment transaction
// (e.g. because the contract got modified), you can use the unit test
// TestGenerateDeploymentTransaction to get fresh values for those variables.

var creatorAddress = hexutil.MustDecode("0x3aA7d3bdc309070181d1cE381D964d33C6Ad9Eb8")
var contractAddress = hexutil.MustDecode("0x71331479A7bB85B37e6210b0e4E4BF4F681261e6")
var deploymentV = hexutil.MustDecode("0x1c")
var deploymentR = hexutil.MustDecode("0x7b29e78247c868230d57ad0657c6ebbe8d543d462237f6d55eb4a2b6c44a4ff1")
var deploymentS = hexutil.MustDecode("0x56d0f0dfb231c1b93bb23b4d7ce466bdcecdf92513c6882147616afe3d08d425")

// GetAddress returns the address of the deployed SubsidiesRegistry.
func GetAddress() common.Address {
	return common.Address(contractAddress[:])
}

// GetCode returns the on-chain bytecode of the SubsidiesRegistry contract.
func GetCode() []byte {
	return bytes.Clone(registryCode)
}

// GetDeploymentTransaction returns a pre-signed transaction that deploys the
// SubsidiesRegistry contract. The transaction was signed with a random private
// key that was discarded afterwards. The contract is thus deployed at an
// address that cannot be pre-occupied by anybody.
//
// Before running the transaction, make sure to provide enough funds to the
// creator address returned by this function.
func GetDeploymentTransaction() (
	tx *types.Transaction,
	creator common.Address,
) {
	raw := getUnsignedDeploymentTransaction()
	raw.V = new(big.Int).SetBytes(deploymentV)
	raw.R = new(big.Int).SetBytes(deploymentR)
	raw.S = new(big.Int).SetBytes(deploymentS)
	return types.NewTx(raw), common.Address(creatorAddress)
}

func getUnsignedDeploymentTransaction() *types.LegacyTx {
	sfcAddress := sfc.ContractAddress
	sfcParameter := [32]byte{}
	copy(sfcParameter[12:], sfcAddress.Bytes())

	initCode := hexutil.MustDecode(RegistryMetaData.Bin)
	return &types.LegacyTx{
		Gas:      2_500_000,
		GasPrice: big.NewInt(1e12),
		Data:     append(initCode, sfcParameter[:]...),
	}
}

//go:embed subsidies_contract.bin
var registryCodeInHex string
var registryCode []byte = hexutils.HexToBytes(registryCodeInHex)
