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

package legacy

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

// Defaults are where the wallet methods take the tip and the gas of a
// transaction they complete: the RPC module's eth_maxPriorityFeePerGas and
// eth_estimateGas on the pending block (ADR5). The composition root sets
// them, and nothing else.
type Defaults interface {
	MaxPriorityFeePerGas(ctx context.Context) (*big.Int, error)
	EstimateGas(ctx context.Context, args TransactionArgs) (hexutil.Uint64, error)
}
