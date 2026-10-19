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
	"github.com/0xsoniclabs/sonic/gossip/scrambler"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBundle_FailingRootStepLeavesNoStateBehind(t *testing.T) {
	upgrades := opera.GetCantoUpgrades()
	upgrades.TransactionBundles = true

	// The transaction pool would reject a bundle failing at the head state, so
	// the envelope is emitted directly, like a validator not pre-checking it.
	net := NewNoIntakeAndEmissionValidationTestNet(t, upgrades)

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	signer := types.LatestSignerForChainID(net.GetChainId())
	revertAddress, revertInput := tests.MustDeployRevertContractAndGetMethodCallParameters(t, net)

	initialBalance := big.NewInt(1e18)

	cases := map[string]struct {
		wrapInGroup bool
	}{
		"BareRoot":   {wrapInGroup: false},
		"AllOfGroup": {wrapInGroup: true},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			sender := tests.MakeAccountsWithBalance(t, net, 1, initialBalance)[0]

			blockNumber, err := client.BlockNumber(t.Context())
			require.NoError(t, err)

			root := Step(t, net, sender, &types.AccessListTx{
				To:   &revertAddress,
				Gas:  1_000_000,
				Data: revertInput,
			})
			if c.wrapInGroup {
				root = bundle.AllOf(root)
			}

			envelope, txBundle, plan := bundle.NewBuilder().
				WithSigner(signer).
				SetEarliest(blockNumber).
				With(root).
				BuildEnvelopeBundleAndPlan()

			t.Log("execution plan:", plan.Root.String())

			_, err = net.Send(envelope)
			require.NoError(t, err)

			info, err := WaitForBundleExecution(t.Context(), client.Client(), plan.Hash())
			require.NoError(t, err)
			require.Zero(t, int(info.Count))

			// The failing transaction reaches no block, ...
			block := big.NewInt(info.Block.Int64())
			failingTx := txBundle.GetTransactionsInReferencedOrder()[0]
			_, err = client.TransactionReceipt(t.Context(), failingTx.Hash())
			require.ErrorIs(t, err, ethereum.NotFound)
			require.NotContains(t, getBlockTxsHashes(t, client, block), failingTx.Hash())

			// ... so the block must not account for any of its effects either.
			nonce, err := client.NonceAt(t.Context(), sender.Address(), block)
			require.NoError(t, err)
			assert.Zero(t, nonce)

			balance, err := client.BalanceAt(t.Context(), sender.Address(), block)
			require.NoError(t, err)
			assert.Equal(t, initialBalance, balance)
		})
	}
}

func TestBundle_FailingBareRootBeforeCantoDoesNotInflateNextReceipt(t *testing.T) {
	net := GetIntegrationTestNetWithBundlesEnabled(t)

	revertAddress, revertInput := tests.MustDeployRevertContractAndGetMethodCallParameters(t, net)
	accounts := tests.MakeAccountsWithBalance(t, net, 2, big.NewInt(1e18))
	sender, marker := accounts[0], accounts[1]

	client, err := net.GetClient()
	require.NoError(t, err)
	blockNumber, err := client.BlockNumber(t.Context())
	require.NoError(t, err)
	client.Close()

	signer := types.LatestSignerForChainID(net.GetChainId())
	envelope, _, _ := bundle.NewBuilder().
		WithSigner(signer).
		SetEarliest(blockNumber).
		With(Step(t, net, sender, &types.AccessListTx{
			To:   &revertAddress,
			Gas:  1_000_000,
			Data: revertInput,
		})).
		BuildEnvelopeBundleAndPlan()

	// The marker must follow the failing bundle in the same block, so its
	// value is varied until the scrambler orders it after the envelope.
	var markerTx *types.Transaction
	for value := int64(1); ; value++ {
		markerTx = tests.CreateTransaction(t, net, &types.LegacyTx{
			To:    &common.Address{},
			Value: big.NewInt(value),
		}, marker)
		order := scrambler.GetExecutionOrder(types.Transactions{envelope, markerTx}, signer, true)
		if order[1] == markerTx {
			break
		}
	}
	_, err = net.ForceEmitAll(t.Context(), []*types.Transaction{envelope, markerTx})
	require.NoError(t, err)

	requireMarkerGasUsed := func() {
		t.Helper()
		client, err := net.GetClient()
		require.NoError(t, err)
		defer client.Close()

		receipt, err := net.GetReceipt(markerTx.Hash())
		require.NoError(t, err)
		require.EqualValues(t, params.TxGas, receipt.GasUsed)

		// The bundle's gas leaks into the block, but into no receipt.
		header, err := client.HeaderByNumber(t.Context(), receipt.BlockNumber)
		require.NoError(t, err)
		receipts, err := client.BlockReceipts(t.Context(),
			rpc.BlockNumberOrHashWithNumber(rpc.BlockNumber(receipt.BlockNumber.Int64())))
		require.NoError(t, err)
		receiptsGas := uint64(0)
		for _, r := range receipts {
			receiptsGas += r.GasUsed
		}
		require.Greater(t, header.GasUsed, receiptsGas)
	}

	// Live receipts are served from the cache, ...
	requireMarkerGasUsed()

	// ... after a restart they are derived from storage.
	require.NoError(t, net.Restart())
	requireMarkerGasUsed()
}
