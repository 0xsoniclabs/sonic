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

package max_block_size

import (
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/integration/makefakegenesis"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"
)

// This test ensures that the maximum block size introduced in EIP-7934 is
// enforced in all proposer modes.
func TestMaxBlockSizeIsEnforced(t *testing.T) {
	mode := map[string]bool{
		"DistributedProposer": false,
		"SingleProposer":      true,
	}

	upgrades := map[string]opera.Upgrades{
		// Although this test increases the network rules way beyond safe limits,
		// it is not possible to guarantee that blocks exceeding the block size limit
		// will be created on every hardware. The tests purpose, of failing in case
		// of an exceeded limit is still served by running it with the limiting enabled.
		// "preBrio": opera.GetAllegroUpgrades(),
		"brio": opera.GetBrioUpgrades(),
	}

	transactionsPerAccount := 10
	numAccounts := 10
	genesisAccounts := make([]makefakegenesis.Account, 0, numAccounts)
	accounts := make([]*tests.Account, 0, numAccounts)
	for range numAccounts {
		account := tests.NewAccount()
		accounts = append(accounts, account)

		genesisAccount := makefakegenesis.Account{
			Address: account.Address(),
			Balance: big.NewInt(1e18),
			Nonce:   0,
		}
		genesisAccounts = append(genesisAccounts, genesisAccount)
	}

	for modeName, singleProposer := range mode {
		for upgradeName, upgrade := range upgrades {
			t.Run(fmt.Sprintf("%s_%s", modeName, upgradeName), func(t *testing.T) {

				upgrade.SingleProposerBlockFormation = singleProposer
				net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
					Upgrades: &upgrade,
					Accounts: genesisAccounts,
					NumNodes: 3,
				})
				increaseLimits(t, net)
				client, err := net.GetClient()
				require.NoError(t, err)
				defer client.Close()

				transactions := make([]*types.Transaction, transactionsPerAccount*numAccounts)
				input := make([]byte, 125_000) // large payload to exceed block size
				cost := uint64(len(input))*params.TxCostFloorPerToken + params.TxGas

				for i := range transactionsPerAccount {
					for accountIdx, account := range accounts {
						txsPayload := &types.LegacyTx{
							Nonce: uint64(i),
							Gas:   cost,
							To:    &common.Address{0x42},
							Value: big.NewInt(0),
							Data:  input,
						}
						signedTx := tests.CreateTransaction(t, net, txsPayload, account)
						transactions[i*numAccounts+accountIdx] = signedTx // save txs with the same nonce next to each other
					}
				}

				slices.Reverse(transactions) // reverse to increase pressure on block size
				receipts, err := net.RunAll(transactions)
				require.NoError(t, err)

				greaterThanMaxBlockSize := false
				for _, receipt := range receipts {
					require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

					block, err := client.BlockByNumber(t.Context(), receipt.BlockNumber)
					require.NoError(t, err)
					require.NotNil(t, block)

					if block.Size() > uint64(params.MaxBlockSize) {
						greaterThanMaxBlockSize = true
					}

					if upgrade.Brio {
						// Starting with Brio both proposer modes enforce MaxBlockSize
						require.LessOrEqual(t, block.Size(), uint64(params.MaxBlockSize), "block size is below MaxBlockSize")
					}
				}
				if modeName == "DistributedProposer" && !upgrade.Brio {
					// Ensure that the test produces at least one block exceeding MaxBlockSize
					require.True(t, greaterThanMaxBlockSize, "expected at least one block to exceed MaxBlockSize")
				}
			})
		}
	}
}

func increaseLimits(t *testing.T, net *tests.IntegrationTestNet) {
	// Increase the gas limit to allow for larger transactions in blocks. These
	// limits are beyond safe limits acceptable for production.
	current := tests.GetNetworkRules(t, net)

	modified := current.Copy()

	modified.Economy.Gas.MaxEventGas = 50_000_000_000
	modified.Economy.Gas.EventGas = 200_000_000
	modified.Economy.ShortGasPower.AllocPerSec = 50_000_000_000
	modified.Economy.ShortGasPower.MaxAllocPeriod = 50_000_000_000
	modified.Economy.LongGasPower = modified.Economy.ShortGasPower
	modified.Emitter.Interval = inter.Timestamp(1 * time.Second)
	tests.UpdateNetworkRules(t, net, modified)
	net.AdvanceEpoch(t, 1)

	// Check that the modification was applied.
	current = tests.GetNetworkRules(t, net)
	require.Equal(t, modified, current)
}
