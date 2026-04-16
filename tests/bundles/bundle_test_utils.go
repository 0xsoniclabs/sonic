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
	"testing"

	"github.com/0xsoniclabs/sonic/api/sonicapi"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/tests/contracts/revert"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
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

// deployContract deploys a contract using the provided deploy function and
// returns its address.
//
//nolint:unused
func deployContract[T any](
	t testing.TB,
	session tests.IntegrationTestNetSession,
	deployFunc tests.ContractDeployer[T],
) common.Address {
	t.Helper()

	_, receipt, err := tests.DeployContract(session, deployFunc)
	require.NoError(t, err, "failed to deploy contract; %v", err)
	require.Equal(t, receipt.Status, types.ReceiptStatusSuccessful)

	return receipt.ContractAddress
}

// getMethodParameters retrieves the ABI of a contract and packs the input
// parameters for a specified method and returns the packed input data.
//
//nolint:unused
func getMethodParameters(
	t testing.TB,
	bindMetadata *bind.MetaData,
	methodName string,
) []byte {
	t.Helper()

	abi, err := bindMetadata.GetAbi()
	require.NoError(t, err, "failed to get counter abi; %v", err)
	input, err := abi.Pack(methodName)
	require.NoError(t, err, "failed to pack input for method %s; %v", methodName, err)

	return input
}

// prepareRevertContract deploys the Revert contract and prepares the input
// for calling the doCrash method, which always reverts.
// It returns the address of the deployed contract and the input data.
//
//nolint:unused
func deployRevertContractAndGetMethodCallParameters(t testing.TB, session tests.IntegrationTestNetSession) (common.Address, []byte) {
	addr := deployContract(t, session, revert.DeployRevert)
	input := getMethodParameters(t, revert.RevertMetaData, "doCrash")
	return addr, input
}
