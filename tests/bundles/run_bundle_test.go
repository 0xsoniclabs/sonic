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
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestBundle_CanBeProcessedByTheNetwork(t *testing.T) {
	t.Parallel()
	upgrades := opera.GetBrioUpgrades()
	upgrades.TransactionBundles = true

	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		Upgrades: &upgrades,
	})

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	senderA := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))
	senderB := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))

	addrA := senderA.Address()
	addrB := senderB.Address()

	block, err := client.BlockNumber(t.Context())
	require.NoError(t, err)

	signer := types.LatestSignerForChainID(net.GetChainId())

	// Create a bundle where sender A and B exchange 1 token each.
	envelope, bundle, plan := bundle.NewBuilder(signer).
		SetEarliest(block).
		With(
			bundle.Step(
				senderA.PrivateKey,
				tests.SetTransactionDefaults(t, net, &types.AccessListTx{
					To:    &addrB,
					Value: big.NewInt(1),
				}, senderA),
			),
			bundle.Step(
				senderB.PrivateKey,
				tests.SetTransactionDefaults(t, net, &types.AccessListTx{
					To:    &addrA,
					Value: big.NewInt(1),
				}, senderB),
			),
		).
		BuildEnvelopeBundleAndPlan()

	// Check bundle status before submission.
	_, err = GetBundleInfo(t.Context(), client.Client(), plan.Hash())
	require.ErrorIs(t, err, ethereum.NotFound)

	// Run the bundle.
	require.NoError(t, client.SendTransaction(t.Context(), envelope))

	// Wait for the bundle to be processed.
	info, err := WaitForBundleExecution(t.Context(), client.Client(), plan.Hash())
	require.NoError(t, err)

	// Check that the transactions are in the block as advertised.
	receipts, err := net.GetReceipts([]common.Hash{bundle.Transactions[0].Hash(), bundle.Transactions[1].Hash()})
	require.NoError(t, err)
	require.Len(t, receipts, 2)
	for _, receipt := range receipts {
		require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
		require.EqualValues(t, info.Block, receipt.BlockNumber.Uint64())
	}

	require.EqualValues(t, info.Position, receipts[0].TransactionIndex)
	require.EqualValues(t, info.Position+1, receipts[1].TransactionIndex)

	// Verify that there is no receipt for the envelope itself.
	_, err = client.TransactionReceipt(t.Context(), envelope.Hash())
	require.ErrorIs(t, err, ethereum.NotFound)
}
