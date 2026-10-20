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

package sonicapi

import (
	"context"
	"fmt"

	"github.com/0xsoniclabs/carmen/go/common/amount"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// TotalSupplyResult is the result of the `sonic_totalSupply` RPC method.
// All values are valid for the block identified by BlockNumber.
type TotalSupplyResult struct {
	// BlockNumber is the block the values are valid for.
	BlockNumber hexutil.Uint64 `json:"blockNumber"`
	// TotalSupply is the sum of the balances of all accounts.
	TotalSupply *hexutil.U256 `json:"totalSupply"`
	// TotalSupplyWithoutBurned is TotalSupply reduced by the balances of the
	// zero address and the 0x000000000000000000000000000000000000dEaD address.
	TotalSupplyWithoutBurned *hexutil.U256 `json:"totalSupplyWithoutBurned"`
	// BurnedZero is the balance of the zero address.
	BurnedZero *hexutil.U256 `json:"burnedZero"`
	// BurnedDead is the balance of the 0x000000000000000000000000000000000000dEaD address.
	BurnedDead *hexutil.U256 `json:"burnedDead"`
}

// TotalSupply implements the `sonic_totalSupply` RPC method. It returns the
// total supply of the native token together with the total supply reduced by
// the balances of the burn addresses (the zero address and 0x...dEaD), and the
// number of the block the values are valid for.
//
// The values are captured during block processing, so they are consistent:
// the balances of the burn addresses reported by `eth_getBalance` for the
// returned block equal the amounts subtracted here.
func (a *PublicBundleAPI) TotalSupply(_ context.Context) (*TotalSupplyResult, error) {
	snapshot, err := a.b.GetTotalSupply()
	if err != nil {
		return nil, err
	}
	burned, overflow := amount.AddOverflow(snapshot.ZeroBalance, snapshot.DeadBalance)
	if overflow {
		return nil, fmt.Errorf("balance of burn addresses overflows")
	}
	withoutBurned, underflow := amount.SubUnderflow(snapshot.Total, burned)
	if underflow {
		return nil, fmt.Errorf("balance of burn addresses exceeds the total supply")
	}
	total := snapshot.Total.Uint256()
	rest := withoutBurned.Uint256()
	zero := snapshot.ZeroBalance.Uint256()
	dead := snapshot.DeadBalance.Uint256()
	return &TotalSupplyResult{
		BlockNumber:              hexutil.Uint64(snapshot.Block),
		TotalSupply:              (*hexutil.U256)(&total),
		TotalSupplyWithoutBurned: (*hexutil.U256)(&rest),
		BurnedZero:               (*hexutil.U256)(&zero),
		BurnedDead:               (*hexutil.U256)(&dead),
	}, nil
}
