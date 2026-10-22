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
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

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
