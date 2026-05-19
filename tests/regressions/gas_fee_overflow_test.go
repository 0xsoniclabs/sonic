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

package regressions

import (
	"context"
	"math"
	"math/big"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/config"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

// The tests in this file reproduce a bug that was reported by an external code
// audit, identifying an overflow of the gas fee field beyond the 256-bit limit
// as a potential attack vector for panicking the client.
//
// While such transactions are filtered by the TxPool and the RPC, a modified
// validator could still attempt to include them in a block, causing the block
// processor to panic when processing the transaction.
//
// This bug never reached a release, but these tests are kept as a regression
// test to ensure that the issue is properly fixed and does not regress in the
// future.

func TestGasFeeOverflow_TransactionsWithOverflowingFeeCapsAreIgnoredByBlockProcessor(t *testing.T) {
	net := tests.StartIntegrationTestNet(t)

	account := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))

	// Create and sign a transaction with excessive gas fees (beyond 256 bits).
	signer := types.LatestSignerForChainID(net.GetChainId())
	tx := types.MustSignNewTx(account.PrivateKey, signer, &types.DynamicFeeTx{
		To:        &common.Address{},
		Gas:       21000,
		GasFeeCap: new(big.Int).Lsh(big.NewInt(1), 256),
		GasTipCap: new(big.Int).Lsh(big.NewInt(1), 256),
	})

	// Send the transaction to the network, by-passing the transaction pool.
	_, err := net.ForceEmit(t.Context(), tx)
	require.NoError(t, err)

	// Let the network progress. The bug to be reproduced by this test causes
	// the client to panic when processing the invalid transaction.
	net.AdvanceEpoch(t, 1)

	// The transaction did not get accepted in any block.
	_, err = net.TryGetReceipt(10*time.Second, tx.Hash())
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestGasFeeOverflow_TransactionsAreRejected(t *testing.T) {
	// This test makes sure that transactions with gas fees exceeding the
	// 256-bit limit are rejected by both the RPC and the TxPool when using
	// the default configuration.

	cases := map[string]struct {
		options                 tests.IntegrationTestNetOptions
		expectedRejectionReason string
	}{
		"rejected by RPC": {
			options:                 tests.IntegrationTestNetOptions{},
			expectedRejectionReason: "exceeds the configured cap (100.00 FTM)",
		},
		"rejected by TxPool": {
			options: tests.IntegrationTestNetOptions{
				// Disable the fee-cap check in the RPC
				ModifyConfig: func(config *config.Config) {
					config.Opera.RPCTxFeeCap = math.Pow(2, 256) // (FTM)
				},
			},
			expectedRejectionReason: "max fee per gas higher than 2^256-1",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			net := tests.StartIntegrationTestNet(t, tc.options)

			key, err := crypto.GenerateKey()
			require.NoError(t, err)

			// Create and sign a transaction with excessive gas fees (beyond 256 bits).
			signer := types.LatestSignerForChainID(net.GetChainId())
			tx := types.MustSignNewTx(key, signer, &types.DynamicFeeTx{
				To:        &common.Address{},
				Gas:       21000,
				GasFeeCap: new(big.Int).Lsh(big.NewInt(1), 256),
				GasTipCap: new(big.Int).Lsh(big.NewInt(1), 256),
			})

			// Send the transaction to the network. This should be rejected
			// either by the RPC or the TxPool, depending on the test case.
			_, err = net.Send(tx)
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.expectedRejectionReason)
		})
	}
}

func TestGasFeeOverflow_TransactionWithFeeOverflowInBundleAreIgnoredByBlockProcessor(t *testing.T) {

	upgrades := opera.GetBrioUpgrades()
	upgrades.TransactionBundles = true

	// In this setup, all TxPool and RPC checks are enabled.
	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		Upgrades: &upgrades,
	})

	account := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))

	// Create a bundle with a transaction exceeding gas fee limits (beyond 256 bits).
	signer := types.LatestSignerForChainID(net.GetChainId())
	envelope, bundle, _ := bundle.NewBuilder().AllOf(
		bundle.Step(account.PrivateKey, &types.DynamicFeeTx{
			To:        &common.Address{},
			Gas:       21000,
			GasFeeCap: new(big.Int).Lsh(big.NewInt(1), 256),
			GasTipCap: new(big.Int).Lsh(big.NewInt(1), 256),
		}),
	).WithSigner(signer).BuildEnvelopeBundleAndPlan()

	// Send the transaction to the network. Since checks in the pool are
	// disabled, this should be accepted.
	_, err := net.ForceEmit(t.Context(), envelope)
	require.NoError(t, err)

	// Let the network progress. The bug to be reproduced by this test causes
	// the client to panic when processing the invalid transaction.
	net.AdvanceEpoch(t, 1)

	// The transaction did not get accepted in any block.
	inner := bundle.GetTransactionsInReferencedOrder()[0]
	_, err = net.TryGetReceipt(10*time.Second, inner.Hash())
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestGasFeeOverflow_TransactionWithFeeOverflowInBundleIsRejectedByThePool(t *testing.T) {

	upgrades := opera.GetBrioUpgrades()
	upgrades.TransactionBundles = true

	// In this setup, all TxPool and RPC checks are enabled.
	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		Upgrades: &upgrades,
	})

	account := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))

	// Create a bundle with a transaction exceeding gas fee limits (beyond 256 bits).
	signer := types.LatestSignerForChainID(net.GetChainId())
	envelope := bundle.NewBuilder().AllOf(
		bundle.Step(account.PrivateKey, &types.DynamicFeeTx{
			To:        &common.Address{},
			Gas:       21000,
			GasFeeCap: new(big.Int).Lsh(big.NewInt(1), 256),
			GasTipCap: new(big.Int).Lsh(big.NewInt(1), 256),
		}),
	).WithSigner(signer).Build()

	// Send the transaction to the network. This should be rejected by the
	// TxPool, with a proper error message.
	_, err := net.Send(envelope)
	require.Error(t, err)
	require.ErrorContains(t, err, "trial-run failed")
}
