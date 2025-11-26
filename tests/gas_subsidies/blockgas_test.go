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

	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies/registry"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/tests/contracts/revert"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/v2"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestGasSubsidies_TooLargeForBlock(t *testing.T) {

	singleProposerOption := map[string]bool{
		"singleProposer": true,
		"distributed":    false,
	}

	for name, enabled := range singleProposerOption {
		upgrades := opera.GetSonicUpgrades()
		upgrades.SingleProposerBlockFormation = enabled
		t.Run(name, func(t *testing.T) {
			testGasSubsidies_tooLargeForBlock(t, upgrades)
		})
	}
}

func testGasSubsidies_tooLargeForBlock(t *testing.T, upgrades opera.Upgrades) {
	// Step 1: Create a network with a single block proposer
	upgrades.GasSubsidies = true
	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		Upgrades: &upgrades,
	})

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	revertContract, receipt, err := tests.DeployContract(net, revert.DeployRevert)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	sender := tests.NewAccount()

	reg, err := registry.NewRegistry(registry.GetAddress(), client)
	require.NoError(t, err)

	// Step 2: Fund a global sponsorship
	_, id, err := reg.GlobalSponsorshipFundId(nil)
	require.NoError(t, err)

	receipt, err = net.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
		opts.Value = big.NewInt(1e18)
		return reg.Sponsor(opts, id)
	})
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// Step 3: Update rules to have maximum block gas limit of 3 million
	current := tests.GetNetworkRules(t, net)
	modified := current.Copy()
	modified.Blocks.MaxBlockGas = 3_000_000
	tests.UpdateNetworkRules(t, net, modified)
	net.AdvanceEpoch(t, 1)
	updatedRules := tests.GetNetworkRules(t, net)
	require.Equal(t, uint64(3_000_000), updatedRules.Blocks.MaxBlockGas)

	// Step 4: Send a sponsored transaction that uses almost all the gas in the block

	opts, err := net.GetTransactOptions(sender)
	require.NoError(t, err)
	opts.GasLimit = 2_800_000
	opts.GasPrice = big.NewInt(0)
	tx, err := revertContract.DoCrash(opts)
	require.NoError(t, err)

	// wait for 3 blocks
	_, _ = net.EndowAccount(sender.Address(), big.NewInt(1e18))
	_, _ = net.EndowAccount(sender.Address(), big.NewInt(1e18))
	_, _ = net.EndowAccount(sender.Address(), big.NewInt(1e18))

	// Transaction shall not be executed because there was no gas left to pay for the execution
	// of the funds deduction transaction
	_, err = client.TransactionReceipt(t.Context(), tx.Hash())
	require.ErrorContains(t, err, ethereum.NotFound.Error())
}
