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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
)

// GetBundleInfo implements the `sonic_getBundleInfo` RPC method, which retrieves
// information about the execution of a transaction bundle.
//
// Since bundles are not stored in the blockchain like regular transactions,
// this method provides information about bundles executed in the recent past.
// The sonic client is not capable of tracking bundles indefinitely, and may return
// null for bundles executed too far in the past.
//
// In the same fashion as `eth_getTransactionReceipt`, this method returns a
// non-error response with null payload if the bundle hasn't been executed yet.
//
// If the bundle has been executed, it returns the block number, position of the
// first transaction of the bundle in the block, and the total number of transactions
// that got included in the block due to the execution of the bundle.
func (a *PublicBundleAPI) GetBundleInfo(
	ctx context.Context,
	executionPlanHash common.Hash,
) (*RPCBundleInfo, error) {

	// Check whether the given execution plan got already executed.
	info := a.b.GetBundleExecutionInfo(executionPlanHash)
	if info != nil {
		return &RPCBundleInfo{
			Block:    rpc.BlockNumber(info.BlockNumber),
			Position: hexutil.Uint(info.Position.Offset),
			Count:    hexutil.Uint(info.Position.Count),
		}, nil
	}

	// Otherwise, the state is unknown (default).
	return nil, nil
}

// RPCBundleInfo is the JSON RPC message returned by the GetBundleInfo API, which
// provides information about the effect of the execution of a transaction bundle.
type RPCBundleInfo struct {
	Block    rpc.BlockNumber `json:"block,omitempty"`
	Position hexutil.Uint    `json:"position,omitempty"`
	Count    hexutil.Uint    `json:"count,omitempty"`
}
