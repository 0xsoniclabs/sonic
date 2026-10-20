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

	// The marker must follow the failing bundle in the same block. The
	// scrambler orders two transactions by each other's hash, so both the
	// envelope and the marker are rebuilt until the marker follows.
	var envelope, markerTx *types.Transaction
	for value := int64(1); ; value++ {
		require.LessOrEqual(t, value, int64(64), "scrambler never ordered the marker after the envelope")
		envelope, _, _ = bundle.NewBuilder().
			WithSigner(signer).
			SetEarliest(blockNumber).
			With(Step(t, net, sender, &types.AccessListTx{
				To:   &revertAddress,
				Gas:  1_000_000,
				Data: revertInput,
			})).
			BuildEnvelopeBundleAndPlan()
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

	// The failed bundle leaked its gas right before the marker, ...
	requireLeakBeforeMarker(t, net, markerTx.Hash())

	// Live receipts are served from the cache, ...
	requireMarkerGasUsed(t, net, markerTx.Hash())

	// ... after a restart they are derived from storage.
	require.NoError(t, net.Restart())
	requireMarkerGasUsed(t, net, markerTx.Hash())

	// ... and after an export/import, they are derived from the genesis.
	require.NoError(t, net.RestartWithExportImport())
	requireMarkerGasUsed(t, net, markerTx.Hash())
}

// requireLeakBeforeMarker checks that the failed bundle leaked its gas right
// before the marker transaction. The failed bundle leaves no transaction in
// the block, so the marker is its only one, and the bundle's gas is all its
// cumulative gas holds beyond the marker's own.
func requireLeakBeforeMarker(t *testing.T, net tests.IntegrationTestNetSession, markerHash common.Hash) {
	t.Helper()
	receipt, err := net.GetReceipt(markerHash)
	require.NoError(t, err)
	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()
	block, err := client.BlockByNumber(t.Context(), receipt.BlockNumber)
	require.NoError(t, err)
	require.Len(t, block.Transactions(), 1,
		"the marker must be the only transaction in its block, or its cumulative gas may hold the gas of other transactions")
	require.Greater(t, receipt.CumulativeGasUsed, params.TxGas,
		"the marker's cumulative gas must hold the gas leaked by the failed bundle")
}

// requireMarkerGasUsed checks that the receipt of the marker transaction
// reports only its own gas, not the gas leaked by the failed bundle before it.
func requireMarkerGasUsed(t *testing.T, net tests.IntegrationTestNetSession, markerHash common.Hash) {
	t.Helper()
	receipt, err := net.GetReceipt(markerHash)
	require.NoError(t, err)
	require.EqualValues(t, params.TxGas, receipt.GasUsed,
		"the marker's receipt must report only its own gas, not the gas leaked by the failed bundle")
}
