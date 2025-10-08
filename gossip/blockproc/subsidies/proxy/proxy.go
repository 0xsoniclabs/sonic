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

package proxy

import (
	"bytes"
	_ "embed"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/keycard-go/hexutils"
)

//go:generate solc --optimize --optimize-runs 200 --bin --bin-runtime proxy.sol --abi proxy.sol -o build --overwrite
//go:generate abigen --bin=build/Proxy.bin --abi=build/Proxy.abi --pkg=proxy --out=proxy_abigen.go
//go:generate cp build/Proxy.bin-runtime proxy_contract.bin

// GetSlotForImplementation returns the storage slot used by the Proxy contract
// to store the address of the current implementation contract, as defined in
// EIP-1967.
func GetSlotForImplementation() common.Hash {
	return common.HexToHash("0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc")
}

// GetCode returns the on-chain bytecode of the Proxy contract.
func GetCode() []byte {
	return bytes.Clone(proxyCode)
}

// ------------------------------ Internals ------------------------------------

//go:embed proxy_contract.bin
var proxyCodeInHex string
var proxyCode []byte = hexutils.HexToBytes(proxyCodeInHex)
