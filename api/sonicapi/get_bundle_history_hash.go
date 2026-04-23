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
)

// GetBundleHistoryHash implements the `sonic_getBundleHistoryHash` RPC method,
// which returns the block number of the last bundle history update and the
// cumulative history hash over all processed bundles up to that block.
//
// The history hash is computed incrementally after every block in which at
// least one bundle has been executed, using:
//
//	newHash = Keccak256(oldHash || XOR(executedPlanHashes) || blockNum)
//
// The hash remains zero until the first bundle is executed. It can be used by
// validators to cross-check that they have processed the same set of bundles
// in the same order.
func (a *PublicBundleAPI) GetBundleHistoryHash(
	_ context.Context,
) (*RPCBundleHistoryHash, error) {
	blockNum, hash := a.b.GetProcessedBundleHistoryHash()
	return &RPCBundleHistoryHash{
		Block: hexutil.Uint64(blockNum),
		Hash:  hash,
	}, nil
}

// RPCBundleHistoryHash is the JSON RPC response returned by GetBundleHistoryHash.
// It contains the block number of the last processed bundle history update and
// the cumulative history hash.
type RPCBundleHistoryHash struct {
	Block hexutil.Uint64 `json:"block"`
	Hash  common.Hash    `json:"hash"`
}
