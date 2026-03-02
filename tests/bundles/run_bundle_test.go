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
	"math/big"
	"slices"
	"testing"

	"github.com/0xsoniclabs/sonic/api/ethapi"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

func TestBundle_CanBeProcessedByTheNetwork(t *testing.T) {
	t.Parallel()
	upgrades := opera.GetBrioUpgrades()
	upgrades.TransactionBundles = true

	session := sharedNetwork.GetIntegrationTestNetSession(t, upgrades)
	signer := types.LatestSignerForChainID(session.GetChainId())

	client, err := session.GetClient()
	require.NoError(t, err)
	defer client.Close()

	coordinator := tests.MakeAccountWithBalance(t, session, big.NewInt(1e18))
	senderA := tests.MakeAccountWithBalance(t, session, big.NewInt(1e18))
	senderB := tests.MakeAccountWithBalance(t, session, big.NewInt(1e18))

	addrA := senderA.Address()
	addrB := senderB.Address()

	block, err := client.BlockNumber(t.Context())
	require.NoError(t, err)

	// Create a bundle where sender A and B exchange 1 token each.
	bundleTx, bundle, plan := bundle.NewBuilder(signer).
		SetEarliest(block).
		With(
			bundle.Step(
				senderA.PrivateKey,
				tests.SetTransactionDefaults(t, session, &types.AccessListTx{
					To:    &addrB,
					Gas:   30_000,
					Value: big.NewInt(1),
				}, senderA),
			),
			bundle.Step(
				senderB.PrivateKey,
				tests.SetTransactionDefaults(t, session, &types.AccessListTx{
					To:    &addrA,
					Gas:   30_000,
					Value: big.NewInt(1),
				}, senderB),
			),
		).
		BuildEnvelopeBundleAndPlan()

	// Check bundle status before submission.
	_, err = getBundleInfo(t.Context(), client.Client(), plan.Hash())
	require.ErrorIs(t, err, ethereum.NotFound)

	// Run the bundle.
	require.NoError(t, client.SendTransaction(t.Context(), bundleTx))

	// Wait for the bundle to be processed.
	info, err := waitForBundleExecution(t.Context(), client.Client(), plan.Hash())
	require.NoError(t, err)

	// Check the block and position in which the bundle was included.
	require.NotNil(t, info.Block)
	require.NotNil(t, info.Position)
	require.NotNil(t, info.Count)

	// Check that the transactions are in the block as advertised.
	receipts, err := session.GetReceipts([]common.Hash{bundle.Transactions[0].Hash(), bundle.Transactions[1].Hash()})
	require.NoError(t, err)
	require.Len(t, receipts, 2)
	for _, receipt := range receipts {
		require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
		require.EqualValues(t, *info.Block, receipt.BlockNumber.Uint64())
	}

	require.EqualValues(t, *info.Position, receipts[0].TransactionIndex)
	require.EqualValues(t, *info.Position+1, receipts[1].TransactionIndex)

	// Verify that there is no receipt for the bundle transaction itself.
	_, err = client.TransactionReceipt(t.Context(), bundleTx.Hash())
	require.ErrorIs(t, err, ethereum.NotFound)

	// Also, the nonce of the bundle creator is zero.
	nonce, err := client.NonceAt(t.Context(), coordinator.Address(), big.NewInt(int64(*info.Block)))
	require.NoError(t, err)
	require.Zero(t, nonce)
}

func getBundleInfo(
	ctxt context.Context,
	client *rpc.Client,
	executionPlanHash common.Hash,
) (*ethapi.RPCBundleInfo, error) {
	var info *ethapi.RPCBundleInfo
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

func waitForBundleExecution(
	ctxt context.Context,
	client *rpc.Client,
	executionPlanHash common.Hash,
) (*ethapi.RPCBundleInfo, error) {
	infos, err := waitForBundlesExecution(
		ctxt, client,
		[]common.Hash{executionPlanHash},
	)
	return infos[0], err
}

func waitForBundlesExecution(
	ctxt context.Context,
	client *rpc.Client,
	executionPlanHashes []common.Hash,
) ([]*ethapi.RPCBundleInfo, error) {

	infos := make([]*ethapi.RPCBundleInfo, len(executionPlanHashes))
	err := tests.WaitFor(ctxt, func(innerCtx context.Context) (bool, error) {
		for i, plan := range executionPlanHashes {
			if infos[i] != nil {
				continue
			}

			info, err := getBundleInfo(innerCtx, client, plan)
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
