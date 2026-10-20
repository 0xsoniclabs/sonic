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

package rpcs

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/api/sonicapi"
	"github.com/0xsoniclabs/sonic/integration/makefakegenesis"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// TestTotalSupply_BurnedBalancesMatchGetBalanceOfReportedBlock verifies that the
// values reported by sonic_totalSupply are consistent with eth_getBalance of the
// burn addresses at the reported block.
func TestTotalSupply_BurnedBalancesMatchGetBalanceOfReportedBlock(t *testing.T) {
	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{})

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	sender := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))
	chainId, err := client.ChainID(t.Context())
	require.NoError(t, err)

	zero := common.Address{}
	dead := common.HexToAddress("0x000000000000000000000000000000000000dEaD")
	for _, target := range []common.Address{zero, dead} {
		tx := tests.SetTransactionDefaults(t, net, &types.LegacyTx{
			To:    &target,
			Value: big.NewInt(1234),
		}, sender)
		signed, err := types.SignNewTx(sender.PrivateKey, types.NewPragueSigner(chainId), tx)
		require.NoError(t, err)
		receipt, err := net.Run(signed)
		require.NoError(t, err)
		require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
	}

	var result sonicapi.TotalSupplyResult
	require.NoError(t, client.Client().CallContext(t.Context(), &result, "sonic_totalSupply"))
	require.NotNil(t, result.TotalSupply)
	require.NotNil(t, result.TotalSupplyWithoutBurned)

	block := rpc.BlockNumber(result.BlockNumber)
	zeroBalance, err := client.BalanceAt(t.Context(), zero, big.NewInt(block.Int64()))
	require.NoError(t, err)
	deadBalance, err := client.BalanceAt(t.Context(), dead, big.NewInt(block.Int64()))
	require.NoError(t, err)
	require.GreaterOrEqual(t, zeroBalance.Int64(), int64(1234))
	require.GreaterOrEqual(t, deadBalance.Int64(), int64(1234))

	require.Equal(t, zeroBalance, (*uint256.Int)(result.BurnedZero).ToBig())
	require.Equal(t, deadBalance, (*uint256.Int)(result.BurnedDead).ToBig())

	burned := new(big.Int).Add(zeroBalance, deadBalance)
	diff := new(big.Int).Sub((*uint256.Int)(result.TotalSupply).ToBig(), (*uint256.Int)(result.TotalSupplyWithoutBurned).ToBig())
	require.Equal(t, burned, diff)
}

// TestTotalSupply_EqualsExpectedSupplyOfReportedBlock verifies that the total
// supply reported by sonic_totalSupply equals the expected value, derived
// independently of the state: the sum of the balances in the genesis plus the
// validator stakes minted by the genesis delegations, reduced by the fees
// burned by all transactions up to the reported block. Sonic does not pay any
// fees to the coinbase, so the full fee of each transaction is burned. No epoch
// is sealed, so no rewards are minted. The check is repeated after a restart to
// cover the persisted value.
func TestTotalSupply_EqualsExpectedSupplyOfReportedBlock(t *testing.T) {
	stakes := makefakegenesis.CreateEqualValidatorStake(1)
	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		ValidatorsStake: stakes,
	})

	genesis := net.GetJsonGenesis()
	require.NotNil(t, genesis)
	genesisSupply := new(big.Int)
	for _, stake := range stakes {
		genesisSupply.Add(genesisSupply, utils.ToFtm(stake))
	}
	for _, account := range genesis.Accounts {
		if account.Balance != nil {
			genesisSupply.Add(genesisSupply, account.Balance.ToBig())
		}
	}

	// Create some balance changes: endowments of new accounts, transfers
	// between them and transfers to the burn addresses.
	accounts := tests.NewAccounts(3)
	for _, account := range accounts {
		_, err := net.EndowAccount(account.Address(), big.NewInt(1e18))
		require.NoError(t, err)
	}
	chainId := net.GetChainId()
	zero := common.Address{}
	dead := common.HexToAddress("0x000000000000000000000000000000000000dEaD")
	targets := []common.Address{accounts[1].Address(), zero, dead, accounts[0].Address()}
	for i, target := range targets {
		sender := accounts[i%len(accounts)]
		tx := tests.SetTransactionDefaults(t, net, &types.LegacyTx{
			To:    &target,
			Value: big.NewInt(int64(1000 + i)),
		}, sender)
		signed, err := types.SignNewTx(sender.PrivateKey, types.NewPragueSigner(chainId), tx)
		require.NoError(t, err)
		receipt, err := net.Run(signed)
		require.NoError(t, err)
		require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
	}

	checkTotalSupply := func() {
		t.Helper()
		client, err := net.GetClient()
		require.NoError(t, err)
		defer client.Close()

		var result sonicapi.TotalSupplyResult
		require.NoError(t, client.Client().CallContext(t.Context(), &result, "sonic_totalSupply"))
		require.NotNil(t, result.TotalSupply)

		burnedFees := new(big.Int)
		for block := int64(0); block <= int64(result.BlockNumber); block++ {
			receipts, err := client.BlockReceipts(t.Context(), rpc.BlockNumberOrHashWithNumber(rpc.BlockNumber(block)))
			require.NoError(t, err)
			for _, receipt := range receipts {
				fee := new(big.Int).SetUint64(receipt.GasUsed)
				burnedFees.Add(burnedFees, fee.Mul(fee, receipt.EffectiveGasPrice))
			}
		}
		require.Positive(t, burnedFees.Sign())

		expected := new(big.Int).Sub(genesisSupply, burnedFees)
		require.Equal(t, expected, (*uint256.Int)(result.TotalSupply).ToBig(),
			"unexpected total supply of block %d", result.BlockNumber)
	}

	checkTotalSupply()
	require.NoError(t, net.Restart())
	checkTotalSupply()
}
