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

package ethapi

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
)

type PublicBundleAPI struct {
	b Backend
}

func NewPublicBundleAPI(b Backend) *PublicBundleAPI {
	return &PublicBundleAPI{b: b}
}

//go:generate stringer -type=BundleStatus -output bundle_status_string.go -trimprefix BundleStatus

type BundleStatus int

const (
	BundleStatusUnknown  BundleStatus = 0
	BundleStatusPending  BundleStatus = 1
	BundleStatusExecuted BundleStatus = 2
)

func (a *PublicBundleAPI) GetBundleInfo(
	ctx context.Context,
	executionPlanHash common.Hash,
) (BundleInfo, error) {

	// Since there is no global lock on the state, and a bundle can be executed
	// and removed from the pool in-between checking for the execution info and
	// the pool state, we check this twice. A valid bundle will only be removed
	// form the pool after it has been executed.
	for range 2 {

		// Check whether the given execution plan got already executed.
		info, err := a.b.GetBundleExecutionInfo(executionPlanHash)
		if err != nil {
			return BundleInfo{}, err
		}
		if info != nil {
			return BundleInfo{
				Status:   BundleStatusExecuted,
				Block:    &info.BlockNum,
				Position: &info.Position,
			}, nil
		}

		// Check whether the given execution plan is pending in the Tx Pool.
		if isInPool := a.b.IsBundleInPool(executionPlanHash); isInPool {
			return BundleInfo{
				Status: BundleStatusPending,
			}, nil
		}

	}

	// Otherwise, the state is unknown (default).
	return BundleInfo{}, nil
}

// BundleInfo is the JSON RPC message returned by the GetBundleInfo API, which
// provides information about the status of a transaction bundle.
type BundleInfo struct {
	Status   BundleStatus `json:"status"`
	Block    *uint64      `json:"block,omitempty"`
	Position *uint32      `json:"position,omitempty"`
}
