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

package bundles

import (
	"context"
	"errors"
	"slices"

	"github.com/0xsoniclabs/sonic/api/sonicapi"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
)

// GetBundleInfo calls the sonic_getBundleInfo RPC method to retrieve
// information about the execution of a transaction bundle.
func GetBundleInfo(
	ctxt context.Context,
	client *rpc.Client,
	executionPlanHash common.Hash,
) (*sonicapi.RPCBundleInfo, error) {
	var info *sonicapi.RPCBundleInfo
	err := client.CallContext(
		ctxt,
		&info,
		"sonic_getBundleInfo",
		executionPlanHash,
	)
	if err == nil && info == nil {
		return nil, ethereum.NotFound
	}
	return info, err
}

// WaitForBundleExecution waits until the bundle execution information of a
// transaction bundle becomes available through the sonic_getBundleInfo RPC
// method. The waiting time can be limited by the provided context.
func WaitForBundleExecution(
	ctxt context.Context,
	client *rpc.Client,
	executionPlanHash common.Hash,
) (*sonicapi.RPCBundleInfo, error) {
	infos, err := WaitForBundleExecutions(
		ctxt, client,
		[]common.Hash{executionPlanHash},
	)
	return infos[0], err
}

// WaitForBundleExecutions waits until the bundle execution information of a
// list of execution plans becomes available through the sonic_getBundleInfo RPC
// method. The waiting time can be limited by the provided context.
func WaitForBundleExecutions(
	ctxt context.Context,
	client *rpc.Client,
	executionPlanHashes []common.Hash,
) ([]*sonicapi.RPCBundleInfo, error) {

	infos := make([]*sonicapi.RPCBundleInfo, len(executionPlanHashes))
	err := tests.WaitFor(ctxt, func(innerCtx context.Context) (bool, error) {
		for i, plan := range executionPlanHashes {
			if infos[i] != nil {
				continue
			}

			info, err := GetBundleInfo(innerCtx, client, plan)
			if err != nil {
				if errors.Is(err, ethereum.NotFound) {
					continue
				}
				return false, err
			}

			if info != nil {
				infos[i] = info
			}
		}
		return !slices.Contains(infos, nil), nil
	})
	return infos, err
}
