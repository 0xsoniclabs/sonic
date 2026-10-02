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
	"encoding/json"
	"fmt"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/api/sonicapi"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/tests/contracts/revert"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// TestBundle_RPCSingleTxBundle_RevertLeavesNoTrace submits many single-tx
// bundles through sonic_prepareBundle and sonic_submitBundle, each calling a
// contract that reverts depending on the order of execution within a block.
// A plan whose root is the bare transaction (envelope(tx)) has no group to
// take a snapshot, so a revert drops the transaction from the block while its
// nonce increment and gas payment stay in the state. The RPC must produce
// envelope(group(tx)) instead, which rolls the reverted transaction back.
//
// Every distinct way of writing a single-tx proposal is covered; deeper
// nesting only repeats the nested cases. Before single-child groups were
// kept, all shapes without tolerateFailures collapsed into a bare root.
func TestBundle_RPCSingleTxBundle_RevertLeavesNoTrace(t *testing.T) {
	// %s is replaced by the fields of the transaction leaf.
	shapes := map[string]string{
		"flat":                    `"steps": [{%s}]`,
		"oneOf":                   `"oneOf": true, "steps": [{%s}]`,
		"tolerateFailures":        `"tolerateFailures": true, "steps": [{%s}]`,
		"nested":                  `"steps": [{"steps": [{%s}]}]`,
		"nested oneOf":            `"steps": [{"oneOf": true, "steps": [{%s}]}]`,
		"nested tolerateFailures": `"steps": [{"tolerateFailures": true, "steps": [{%s}]}]`,
		"deeply nested":           `"steps": [{"steps": [{"steps": [{%s}]}]}]`,
		"leaf tolerateFailed":     `"steps": [{"tolerateFailed": true, %s}]`,
		"leaf tolerateInvalid":    `"steps": [{"tolerateInvalid": true, %s}]`,
	}

	net := GetIntegrationTestNetWithBundlesEnabled(t)
	revertContract := tests.MustDeployContract(t, net, revert.DeployRevert)

	for name, shape := range shapes {
		t.Run(name, func(t *testing.T) {
			testSingleTxBundleRevertLeavesNoTrace(t, net, revertContract, shape)
		})
	}
}

func testSingleTxBundleRevertLeavesNoTrace(
	t *testing.T,
	net *tests.IntegrationTestNet,
	revertContract common.Address,
	shape string,
) {
	const N = 100
	require := require.New(t)

	client, err := net.GetClient()
	require.NoError(err)
	defer client.Close()

	revertAbi, err := revert.RevertMetaData.GetAbi()
	require.NoError(err)
	data := hexutil.Encode(revertAbi.Methods["probabilisticRevert"].ID)

	initialBalance := big.NewInt(1e18)
	senders := tests.MakeAccountsWithBalance(t, net, N, initialBalance)
	signer := types.LatestSignerForChainID(net.GetChainId())

	// Prepare and sign all bundles first, so that they can be submitted at
	// once and land in the same blocks, where they disturb each other.
	blockNumber, err := client.BlockNumber(t.Context())
	require.NoError(err)
	submissions := make([]sonicapi.SubmitBundleArgs, N)
	txHashes := make([]common.Hash, N)
	for i, sender := range senders {
		leaf := fmt.Sprintf(
			`"from": %q, "to": %q, "nonce": "0x0", "gas": "0x186a0", "data": %q`,
			sender.Address().Hex(), revertContract.Hex(), data,
		)
		proposal := fmt.Sprintf(`{"blockRange": {"first": %q}, %s}`,
			hexutil.EncodeUint64(blockNumber), fmt.Sprintf(shape, leaf))

		var prepared sonicapi.RPCPreparedBundle
		require.NoError(client.Client().CallContext(
			t.Context(), &prepared, "sonic_prepareBundle", json.RawMessage(proposal),
		))
		require.Len(prepared.Transactions, 1)

		unsigned, err := prepared.Transactions[0].ToTransaction()
		require.NoError(err)
		tx, err := types.SignTx(unsigned, signer, sender.PrivateKey)
		require.NoError(err)
		encoded, err := tx.MarshalBinary()
		require.NoError(err)

		submissions[i] = sonicapi.SubmitBundleArgs{
			SignedTransactions: []hexutil.Bytes{encoded},
			ExecutionPlan:      prepared.ExecutionPlan,
		}
		txHashes[i] = tx.Hash()
	}

	// Bundles failing the trial run at submission are rejected; this is fine.
	planHashes := make([]common.Hash, N)
	submitErrs := make([]error, N)
	var wg sync.WaitGroup
	for i := range submissions {
		wg.Go(func() {
			submitErrs[i] = client.Client().CallContext(
				t.Context(), &planHashes[i], "sonic_submitBundle", submissions[i],
			)
		})
	}
	wg.Wait()
	for _, err := range submitErrs {
		if err != nil {
			require.ErrorContains(err, "trial-run failed")
		}
	}

	timeout, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	infos, err := WaitForBundleExecutions(timeout, client.Client(), planHashes)
	if err != nil {
		require.ErrorIs(err, context.DeadlineExceeded)
	}

	reverted := 0
	for i, sender := range senders {
		nonce, err := client.NonceAt(t.Context(), sender.Address(), nil)
		require.NoError(err)
		balance, err := client.BalanceAt(t.Context(), sender.Address(), nil)
		require.NoError(err)

		receipt, err := client.TransactionReceipt(t.Context(), txHashes[i])
		if infos[i] != nil && (err != nil || receipt.Status == types.ReceiptStatusFailed) {
			reverted++
		}
		if err == nil {
			require.Equal(uint64(1), nonce, "sender %d", i)
			continue
		}
		require.ErrorIs(err, ethereum.NotFound)
		require.Equal(uint64(0), nonce,
			"sender %d: nonce consumed by a transaction not included in any block", i)
		require.Zero(initialBalance.Cmp(balance),
			"sender %d: gas charged for a transaction not included in any block", i)
	}
	t.Logf("%d of %d bundles reverted during block processing", reverted, N)
	require.NotZero(reverted, "no bundle was reverted during block processing")
}
