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
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/status-im/keycard-go/hexutils"
)

//go:generate solc --optimize --optimize-runs 200 --bin --bin-runtime --abi freeze_registry.sol -o build --overwrite
//go:generate abigen --bin=build/FreezeRegistry.bin --abi=build/FreezeRegistry.abi --pkg=registry --out=freeze_registry_abigen.go
//go:generate cp build/FreezeRegistry.bin-runtime freeze_contract.bin

// GetAddress returns the address of the deployed FreezeRegistry.
func GetAddress() common.Address {
	return common.Address(contractAddress[:])
}

// GetCode returns the on-chain bytecode of the FreezeRegistry contract.
func GetCode() []byte {
	return bytes.Clone(registryCode)
}

// GetImplementationAddress returns the address of the FreezeRegistry
// implementation the proxy delegates to in fake genesis.
func GetImplementationAddress() common.Address {
	return common.Address{1, 2, 3, 4, 5, 6, 9}
}

// GetFrozenSlot returns the proxy storage slot of the `frozen` entry of
// account.
func GetFrozenSlot(account common.Address) common.Hash {
	return mappingSlot(account, frozenMappingSlot)
}

// GetOptedOutSlot returns the proxy storage slot of the `optedOut` entry of
// account.
func GetOptedOutSlot(account common.Address) common.Hash {
	return mappingSlot(account, field(1))
}

// ------------------------------ Internals ------------------------------------

// contractAddress is the address of the FreezeRegistry proxy.
//
// PLACEHOLDER: the bytes spell "freeze" followed by zero padding.
var contractAddress = hexutil.MustDecode("0x667265657a650000000000000000000000000000")

// storageLocation is the ERC-7201 location of the registry's storage struct,
// erc7201("sonic.freeze.registry"); its first fields are the `frozen` and
// `optedOut` mappings.
var (
	storageLocation   = erc7201("sonic.freeze.registry")
	frozenMappingSlot = storageLocation
)

// erc7201 computes keccak256(abi.encode(uint256(keccak256(id)) - 1)) & ~0xff.
// field returns the slot of the i-th field of the registry's storage struct.
func field(i int64) common.Hash {
	return common.BigToHash(new(big.Int).Add(storageLocation.Big(), big.NewInt(i)))
}

func erc7201(id string) common.Hash {
	h := new(big.Int).SetBytes(crypto.Keccak256([]byte(id)))
	location := crypto.Keccak256(common.BigToHash(h.Sub(h, big.NewInt(1))).Bytes())
	location[31] = 0
	return common.BytesToHash(location)
}

func mappingSlot(key common.Address, slot common.Hash) common.Hash {
	var buf [64]byte
	copy(buf[12:32], key[:])
	copy(buf[32:], slot[:])
	return crypto.Keccak256Hash(buf[:])
}

//go:embed freeze_contract.bin
var registryCodeInHex string
var registryCode []byte = hexutils.HexToBytes(registryCodeInHex)
