// Copyright 2025 Sonic Operations Ltd
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

package tx_max_gas

import (
	"math"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/utils/signers/internaltx"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestTxMaxGas(t *testing.T) {
	// This test verifies the effects of https://eips.ethereum.org/EIPS/eip-7825

	net := tests.StartIntegrationTestNet(t,
		tests.IntegrationTestNetOptions{
			Upgrades: tests.AsPointer(opera.GetBrioUpgrades()),
		})

	type rulesType struct {
		Economy struct {
			Gas struct {
				MaxEventGas int64
			}
		}
	}

	client, err := net.GetClient()
	require.NoError(t, err)

	var rules rulesType
	err = client.Client().Call(&rules, "eth_getRules", "latest")
	require.NoError(t, err)

	client.Close()

	originalMaxCap := rules.Economy.Gas.MaxEventGas

	t.Run("transaction with gas over network limit is rejected", func(t *testing.T) {
		tx := tests.CreateTransaction(t, net, &types.LegacyTx{Gas: uint64(originalMaxCap + 1)}, net.GetSessionSponsor())
		receipt, err := net.Run(tx)
		require.ErrorContains(t, err, "gas limit too high")
		require.Nil(t, receipt)
	})

	t.Run("internal transactions can execute even if over network limit", func(t *testing.T) {

		var epochBefore hexutil.Uint64
		err = client.Client().Call(&epochBefore, "eth_currentEpoch")
		require.NoError(t, err)

		// internal transactions use a fixed gas limit defined in drivermodule/driver_txs.go
		// of 500_000_000 so we need to set a max to less than that.
		// As enforced by rules validation a change on MaxEventGas can not be lower than 1_000_000
		// so we set it to 1_000_000 and see that internal transactions still work.
		rules.Economy.Gas.MaxEventGas = int64(opera.UpperBoundForRuleChangeGasCosts())

		tests.UpdateNetworkRules(t, net, rules)

		// internal transactions are executed as part of epoch sealing
		tests.AdvanceEpochAndWaitForBlocks(t, net)

		// since the previous epoch seal would have executed with the old rules to apply the new ones
		// a new epoch advancement is needed to ensure a seal epoch is executed with the new limits.
		tests.AdvanceEpochAndWaitForBlocks(t, net)

		var epochAfter hexutil.Uint64
		err = client.Client().Call(&epochAfter, "eth_currentEpoch")
		require.NoError(t, err)

		// at least two epochs should have passed
		require.Greater(t, epochAfter, epochBefore+1, "Epoch should have advanced")

		// look for a block from latest backwards, looking for a block that starts with an internal transaction.
		internalTransaction := lookForEpochSeal(t, net)
		require.NotNil(t, internalTransaction, "Should find an internal transaction")
		require.Greater(t, internalTransaction.Gas(), opera.UpperBoundForRuleChangeGasCosts(),
			"Internal transaction gas should be over the max event gas limit")
	})

	t.Run("high gas transaction accepted into the pool is never executed after rules change", func(t *testing.T) {

		// reset max gas.
		rules.Economy.Gas.MaxEventGas = int64(2_000_000)
		tests.UpdateNetworkRules(t, net, rules)
		tests.AdvanceEpochAndWaitForBlocks(t, net)

		client, err := net.GetClient()
		require.NoError(t, err)
		defer client.Close()

		err = client.Client().Call(&rules, "eth_getRules", "latest")
		require.NoError(t, err)
		require.Equal(t, int64(2_000_000), rules.Economy.Gas.MaxEventGas, "MaxEventGas should be updated")

		account := tests.MakeAccountWithBalance(t, net, big.NewInt(math.MaxInt64))

		// create a transaction with high gas which is accepted into the pool
		// but cannot be executed because of gapped nonce.
		gappedTx := tests.CreateTransaction(t, net, &types.LegacyTx{Nonce: 1, Gas: 1_500_000}, account)
		err = client.SendTransaction(t.Context(), gappedTx)
		require.NoError(t, err, "Transaction should be accepted into the pool")

		// update rules to lower max gas below the transaction's gas
		rules.Economy.Gas.MaxEventGas = int64(1_100_000)
		tests.UpdateNetworkRules(t, net, rules)
		tests.AdvanceEpochAndWaitForBlocks(t, net)

		err = client.Client().Call(&rules, "eth_getRules", "latest")
		require.NoError(t, err)
		require.Equal(t, int64(1_100_000), rules.Economy.Gas.MaxEventGas, "MaxEventGas should be updated")

		// send a transaction with the missing nonce and gas under new limit
		lowGasTx := tests.CreateTransaction(t, net, &types.LegacyTx{Nonce: 0, Gas: 500_000}, account)
		receipt, err := net.Run(lowGasTx)
		require.NoError(t, err, "Transaction should be executed")
		require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status, "Transaction should be successful")

		// wait 3 blocks
		for range 3 {
			receipt, err = net.EndowAccount(common.Address{1}, big.NewInt(1)) // trigger block creation
			require.NoError(t, err, "Block creation should succeed")
			require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status, "Transaction should be successful")
		}

		// verify the high gas transaction was never executed
		_, err = client.TransactionReceipt(t.Context(), gappedTx.Hash())
		require.ErrorIs(t, err, ethereum.NotFound, "Transaction should not be executed")
	})
}

// lookForEpochSeal looks for the most recent epoch seal transaction by scanning blocks backwards
func lookForEpochSeal(t *testing.T, net *tests.IntegrationTestNet) *types.Transaction {

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	blockNumber, err := client.BlockNumber(t.Context())
	require.NoError(t, err)

	var sealEpoch *types.Transaction

	for ; sealEpoch == nil || blockNumber != 0; blockNumber-- {
		block, err := client.BlockByNumber(t.Context(), big.NewInt(int64(blockNumber)))
		require.NoError(t, err)

		if len(block.Transactions()) == 0 {
			continue
		}

		// if the first transaction is an internal transaction, we found the epoch seal block
		if internaltx.IsInternal(block.Transactions()[0]) {
			sealEpoch = block.Transactions()[0]
		}
	}
	return sealEpoch
}
