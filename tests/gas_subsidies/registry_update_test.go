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

package gas_subsidies

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies/proxy"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies/registry"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/tests/contracts/sponsor_everything"
	"github.com/0xsoniclabs/sonic/utils/signers/internaltx"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestRegistryUpdate_UpdateContract_SponsoredTransactionsCanBePerformed(t *testing.T) {

	// Enable a revision with allegro to support setCode transactions.
	upgrades := opera.GetBrioUpgrades()
	upgrades.GasSubsidies = true

	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		Upgrades: &upgrades,
	})

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	// Install the replacement contract for the default registry.
	_, receipt, err := tests.DeployContract(net, sponsor_everything.DeploySponsorEverything)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// Get the initial config to compare later.
	reg, err := registry.NewRegistry(registry.GetAddress(), client)
	require.NoError(t, err)
	initialConfig, err := reg.GetGasConfig(nil)
	require.NoError(t, err)

	// Update the registry to point to the new contract.
	proxy, err := proxy.NewProxy(registry.GetAddress(), client)
	require.NoError(t, err)
	receipt, err = net.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return proxy.Update(opts, receipt.ContractAddress)
	})
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// Check that now the replaced contract is in use.
	updatedConfig, err := reg.GetGasConfig(nil)
	require.NoError(t, err)
	require.EqualValues(t, 1_234_567, updatedConfig.ChooseFundLimit.Uint64())
	require.EqualValues(t, 654_321, updatedConfig.DeductFeesLimit.Uint64())
	require.NotEqual(t,
		updatedConfig.OverheadCharge.Uint64(),
		initialConfig.OverheadCharge.Uint64(),
	)

	// Run an example sponsored transaction to verify that the new registry
	// works as expected.
	sponsee := tests.NewAccount()
	Fund(t, net, sponsee.Address(), big.NewInt(1e18))

	tx := &types.LegacyTx{Gas: 21000, To: &common.Address{0x42}}
	signedTx := makeSponsorRequestTransaction(t, tx, net.GetChainId(), sponsee)
	require.NoError(t, client.SendTransaction(t.Context(), signedTx))

	// Check that the sponsored transaction was included and paid for according
	// to the new registry contract's rules (in particular gas costs).
	validateSponsoredTxInBlock(t, &net.Session, signedTx.Hash())
}

func TestRegistryUpdate_UpdatesAreEffectiveImmediately(t *testing.T) {
	require := require.New(t)

	// Enable a revision with allegro to support setCode transactions.
	upgrades := opera.GetBrioUpgrades()
	upgrades.GasSubsidies = true

	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		Upgrades: &upgrades,
	})

	client, err := net.GetClient()
	require.NoError(err)
	defer client.Close()

	// Install the replacement contract for the default registry.
	_, receipt, err := tests.DeployContract(net, sponsor_everything.DeploySponsorEverything)
	require.NoError(err)
	require.Equal(types.ReceiptStatusSuccessful, receipt.Status)
	replacementAddress := receipt.ContractAddress

	// Now we create 4 transactions:
	// - two sponsorship request to run before the update
	// - a transaction to perform the update
	// - a sponsorship request to run after the update
	//
	// Ideally, all 4 are included in the same block (but not strictly necessary
	// for this test, and not verified). The first is charged for according to
	// the old rules, the last according to the new rules.

	sponsee := tests.NewAccount()

	// provide some funds to the sponsee for transaction 1 and 3
	Fund(t, net, sponsee.Address(), big.NewInt(1e18))

	// provide some funds to the sponsee for transaction 2 (the update)
	_, err = net.EndowAccount(sponsee.Address(), big.NewInt(1e18))
	require.NoError(err)

	signer := types.LatestSignerForChainID(net.GetChainId())
	tx1 := types.MustSignNewTx(sponsee.PrivateKey, signer, &types.LegacyTx{
		Gas:      21000,
		To:       &common.Address{0x42},
		GasPrice: big.NewInt(0),
		Nonce:    0,
	})
	require.True(subsidies.IsSponsorshipRequest(tx1))

	tx2 := types.MustSignNewTx(sponsee.PrivateKey, signer, &types.LegacyTx{
		Gas:      21000,
		To:       &common.Address{0x42},
		GasPrice: big.NewInt(0),
		Nonce:    1,
	})
	require.True(subsidies.IsSponsorshipRequest(tx2))

	proxy, err := proxy.NewProxy(registry.GetAddress(), client)
	require.NoError(err)

	opts := new(bind.TransactOpts)
	opts.From = sponsee.Address()
	opts.Signer = func(address common.Address, tx *types.Transaction) (*types.Transaction, error) {
		return types.SignTx(tx, signer, sponsee.PrivateKey)
	}
	opts.NoSend = true
	opts.Nonce = big.NewInt(2)
	tx3, err := proxy.Update(opts, replacementAddress)
	require.NoError(err)

	tx4 := types.MustSignNewTx(sponsee.PrivateKey, signer, &types.LegacyTx{
		Gas:      21000,
		To:       &common.Address{0x43},
		GasPrice: big.NewInt(0),
		Nonce:    3,
	})
	require.True(subsidies.IsSponsorshipRequest(tx4))

	// Run the 4 transactions.
	receipts, err := net.RunAll([]*types.Transaction{tx1, tx2, tx3, tx4})
	require.NoError(err)
	require.Len(receipts, 4)
	for _, r := range receipts {
		require.Equal(types.ReceiptStatusSuccessful, r.Status)
	}

	// Check that all sponsored transactions used the same amount of gas.
	require.Equal(receipts[0].GasUsed, receipts[1].GasUsed)
	require.Equal(receipts[0].GasUsed, receipts[3].GasUsed)

	// Fetch the payment transaction for this sponsored transaction.
	payments := []*types.Transaction{}
	for _, tx := range []*types.Transaction{tx1, tx2, tx4} {
		receipt, err := client.TransactionReceipt(t.Context(), tx.Hash())
		require.NoError(err)

		block, err := client.BlockByNumber(t.Context(), receipt.BlockNumber)
		require.NoError(err)
		require.LessOrEqual(int(receipt.TransactionIndex), len(block.Transactions()))
		payment := block.Transactions()[receipt.TransactionIndex+1]
		require.True(internaltx.IsInternal(payment))
		substractReceipt, err := client.TransactionReceipt(t.Context(), payment.Hash())
		require.NoError(err)
		require.Equal(types.ReceiptStatusSuccessful, substractReceipt.Status)

		payments = append(payments, payment)
	}

	// The first two transactions are charged according to the old rules. Since
	// both are identical, the data for the payment Tx must be identical.
	require.Equal(payments[0].Data(), payments[1].Data())

	// The last transaction is charged according to the new rules. Thus, the
	// call data for the payment transaction must be different.
	require.NotEqual(payments[0].Data(), payments[2].Data())

}
