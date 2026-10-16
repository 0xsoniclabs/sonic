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

// Package exec holds what the methods that execute transactions share:
// call arguments, state and block overrides, the EVM environment and the
// chain configuration derived from the rules.
package exec

import (
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/params"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/rpc/core"
)

// ChainConfig returns the chain configuration in force at a block height. The
// result is valid for that height only: it is the queried block's, never the
// head's.
func ChainConfig(info core.ChainInfo, height uint64) *params.ChainConfig {
	return opera.CreateTransientEvmChainConfig(info.NetworkID, info.UpgradeHeights, idx.Block(height))
}
