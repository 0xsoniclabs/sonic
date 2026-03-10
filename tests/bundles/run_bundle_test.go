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
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/ethapi"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

func TestBundle_CanBeProcessedByTheNetwork(t *testing.T) {
	const NumNodes = 3

	upgrades := opera.GetBrioUpgrades()
	upgrades.TransactionBundles = true

	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		Upgrades: &upgrades,
		NumNodes: NumNodes,
	})

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	coordinator := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))
	senderA := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))
	senderB := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))

	addrA := senderA.Address()
	addrB := senderB.Address()

	chainId := net.GetChainId()
	signer := types.LatestSignerForChainID(chainId)

	// Create a bundle where sender A and B exchange 1 token each.
	block, err := client.BlockNumber(t.Context())
	require.NoError(t, err)
	txToSign, plan := prepareBundle(
		chainId, block,
		[]UnsignedTransaction{
			{
				Sender: addrA,
				Transaction: tests.SetTransactionDefaults(
					t, net,
					&types.AccessListTx{
						To:    &addrB,
						Gas:   30_000,
						Value: big.NewInt(1),
					},
					senderA,
				),
			},
			{
				Sender: addrB,
				Transaction: tests.SetTransactionDefaults(
					t, net,
					&types.AccessListTx{
						To:    &addrA,
						Gas:   30_000,
						Value: big.NewInt(1),
					},
					senderB,
				),
			},
		},
	)

	// Sign the individual transactions
	signedTxs := []*types.Transaction{
		types.MustSignNewTx(senderA.PrivateKey, signer, txToSign[0].Transaction),
		types.MustSignNewTx(senderB.PrivateKey, signer, txToSign[1].Transaction),
	}

	// Create the bundle transaction
	bundleTx := types.MustSignNewTx(
		coordinator.PrivateKey, signer,
		makeBundle(signedTxs, plan),
	)

	// Check bundle construction.
	require.True(t, bundle.IsTransactionBundle(bundleTx))
	recoveredBundle, recoveredPlan, err := bundle.ValidateTransactionBundle(bundleTx, signer)
	require.NoError(t, err)
	require.NotNil(t, recoveredBundle)
	require.NotNil(t, recoveredPlan)
	require.Equal(t, plan, *recoveredPlan)
	require.EqualValues(t, 0, bundleTx.GasFeeCap().Uint64())

	// Check bundle status before submission.
	info, err := getBundleInfo(t.Context(), client.Client(), plan.Hash())
	require.NoError(t, err)
	require.Equal(t, ethapi.BundleStatusUnknown, info.Status)

	// Run the bundle.
	require.NoError(t, client.SendTransaction(t.Context(), bundleTx))

	// Wait for the bundle to be processed.
	info, err = waitForBundleExecution(t.Context(), client.Client(), plan.Hash())
	require.NoError(t, err)
	require.Equal(t, ethapi.BundleStatusExecuted, info.Status)

	// Check the block and position in which the bundle was included.
	require.NotNil(t, info.Block)
	require.NotNil(t, info.Position)
	require.NotNil(t, info.Count)

	// Check that the transactions are in the block as advertised.
	receipts, err := net.GetReceipts([]common.Hash{signedTxs[0].Hash(), signedTxs[1].Hash()})
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

	// Check that all nodes have the same block state.
	latestBlock, err := client.BlockByNumber(t.Context(), nil)
	require.NoError(t, err)
	for i := range NumNodes {
		client, err := net.GetClientConnectedToNode(i)
		require.NoError(t, err)
		defer client.Close()

		block, err := client.BlockByNumber(t.Context(), latestBlock.Number())
		require.NoError(t, err)
		require.Equal(t, latestBlock.Hash(), block.Hash())
	}
}

type UnsignedTransaction struct {
	Sender      common.Address
	Transaction *types.AccessListTx
}

func prepareBundle(
	chainId *big.Int,
	targetBlock uint64,
	txs []UnsignedTransaction,
) ([]UnsignedTransaction, bundle.ExecutionPlan) {

	signer := types.LatestSignerForChainID(chainId)

	var steps []bundle.ExecutionStep
	for _, unsignedTx := range txs {
		steps = append(steps, bundle.ExecutionStep{
			From: unsignedTx.Sender,
			Hash: signer.Hash(types.NewTx(unsignedTx.Transaction)),
		})
	}

	// build execution plan
	plan := bundle.ExecutionPlan{
		Steps:    steps,
		Earliest: targetBlock,
		Latest:   targetBlock + 10,
	}

	planHash := plan.Hash()

	// amend transactions with the execution plan hash in the access list
	for _, unsignedTx := range txs {
		txData := unsignedTx.Transaction
		txData.AccessList = append(txData.AccessList, types.AccessTuple{
			Address: bundle.BundleOnly,
			StorageKeys: []common.Hash{
				planHash,
			},
		})
	}

	return txs, plan
}

func makeBundle(
	txs []*types.Transaction,
	plan bundle.ExecutionPlan,
) types.TxData {
	data := bundle.Encode(bundle.TransactionBundle{
		Version:  bundle.BundleV1,
		Bundle:   txs,
		Flags:    plan.Flags,
		Earliest: plan.Earliest,
		Latest:   plan.Latest,
	})
	neededGas := uint64(0)
	for _, tx := range txs {
		neededGas += tx.Gas()
	}
	intrGas, _ := core.IntrinsicGas(
		data,
		nil,   // access list is set in the individual transactions
		nil,   // code auth is not used in the bundle transaction
		false, // bundle transaction is not a contract creation
		true,
		true,
		true,
	)
	if neededGas < intrGas {
		neededGas = intrGas
	}
	return &types.DynamicFeeTx{
		To:   &bundle.BundleAddress,
		Data: data,
		Gas:  neededGas,
	}
}

func getBundleInfo(
	ctxt context.Context,
	client *rpc.Client,
	executionPlanHash common.Hash,
) (ethapi.BundleInfo, error) {
	var info ethapi.BundleInfo
	err := client.CallContext(
		ctxt,
		&info,
		"sonic_getBundleInfo",
		executionPlanHash,
	)
	return info, err
}

func waitForBundleExecution(
	ctxt context.Context,
	client *rpc.Client,
	executionPlanHash common.Hash,
) (ethapi.BundleInfo, error) {
	var info ethapi.BundleInfo
	var err error
	err = tests.WaitFor(ctxt, func(innerCtx context.Context) (bool, error) {
		info, err = getBundleInfo(innerCtx, client, executionPlanHash)
		if err != nil {
			return false, err
		}
		return info.Status != ethapi.BundleStatusPending, nil
	})
	return info, err
}
