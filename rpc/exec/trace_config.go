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

package exec

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/eth/tracers"
)

// TraceCallConfig is how a traced call runs: the tracer and its settings,
// the overrides of the state and the block, and the index of the
// transaction of the block whose state the call runs on, 0 if none.
type TraceCallConfig struct {
	tracers.TraceConfig
	StateOverrides *StateOverride
	BlockOverrides *BlockOverrides
	TxIndex        *hexutil.Uint
}
