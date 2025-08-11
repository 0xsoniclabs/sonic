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

package tests

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests/block_header"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

func TestSingleProposerProtocol_CanProcessTransactions(t *testing.T) {
	upgrades := map[string]opera.Upgrades{
		"Sonic":   opera.GetSonicUpgrades(),
		"Allegro": opera.GetAllegroUpgrades(),
	}

	for name, upgrades := range upgrades {
		t.Run(name, func(t *testing.T) {
			for _, numNodes := range []int{1, 3} {
				t.Run(fmt.Sprintf("numNodes=%d", numNodes), func(t *testing.T) {
					testSingleProposerProtocol_CanProcessTransactions(t, numNodes, upgrades)
				})
			}
		})
	}
}

func testSingleProposerProtocol_CanProcessTransactions(
	t *testing.T,
	numNodes int,
	upgrades opera.Upgrades,
) {
	// This test is a general smoke test for the single-proposer protocol. It
	// checks that transactions can be processed and that the network is not
	// producing (excessive) empty blocks.
	const NumRounds = 30
	const EpochLength = 7
	const NumTxsPerRound = 5

	upgrades.SingleProposerBlockFormation = true

	require := require.New(t)
	net := StartIntegrationTestNet(t, IntegrationTestNetOptions{
		Upgrades: &upgrades,
		NumNodes: numNodes,
	})

	client, err := net.GetClient()
	require.NoError(err)
	defer client.Close()

	// --- setup network accounts ---

	// Create NumTxsPerRound accounts and send them each 1e18 wei to allow each
	// of them to send independent transactions in each round.
	accounts := make([]*Account, NumTxsPerRound)
	addresses := make([]common.Address, NumTxsPerRound)
	for i := range accounts {
		accounts[i] = NewAccount()
		addresses[i] = accounts[i].Address()
	}
	_, err = net.EndowAccounts(addresses, big.NewInt(1e18))
	require.NoError(err)

	// Check that the network is using the single-proposer protocol.
	require.Equal(3, getUsedEventVersion(t, client))

	// --- check processing of transactions ---

	chainId := net.GetChainId()
	signer := types.NewPragueSigner(chainId)
	target := common.Address{0x42}

	startBlock, err := client.BlockNumber(t.Context())
	require.NoError(err)

	// Send a sequence of transactions to the network, in several rounds,
	// across multiple epochs, and check that all get processed.
	for round := range uint64(NumRounds) {
		transactions := []*types.Transaction{}
		for sender := range NumTxsPerRound {
			transaction := types.MustSignNewTx(
				accounts[sender].PrivateKey,
				signer,
				&types.DynamicFeeTx{
					ChainID:   chainId,
					Nonce:     round,
					To:        &target,
					Value:     big.NewInt(1),
					Gas:       21000,
					GasFeeCap: big.NewInt(1e11),
					GasTipCap: big.NewInt(int64(sender) + 1),
				},
			)
			transactions = append(transactions, transaction)
		}

		receipts, err := net.RunAll(transactions)
		require.NoError(err, "failed to run transactions")
		require.Len(receipts, NumTxsPerRound, "unexpected number of receipts")
		for _, receipt := range receipts {
			require.Equal(types.ReceiptStatusSuccessful, receipt.Status)
		}

		// Start a new epoch every EpochLength rounds, but not as part of the
		// first round to avoid mixing up issued introduced by the transaction
		// processing and the epoch change. Thus, the first epoch will run for
		// EpochLength/2 rounds, and the rest for EpochLength rounds.
		if round%EpochLength == EpochLength/2 {
			net.AdvanceEpoch(t, 1)
		}
	}

	// Check that rounds have been processed fairly efficient, without the use
	// of a large number of blocks. This is a mere smoke test to check that the
	// validators are not spamming unnecessary empty proposals.
	endBlock, err := client.BlockNumber(t.Context())
	require.NoError(err)

	duration := endBlock - startBlock
	require.Less(duration, uint64(2*NumRounds))
}

func TestSingleProposerProtocol_CanBeEnabledAndDisabled(t *testing.T) {
	upgrades := map[string]opera.Upgrades{
		"Sonic":   opera.GetSonicUpgrades(),
		"Allegro": opera.GetAllegroUpgrades(),
	}

	for name, upgrades := range upgrades {
		t.Run(name, func(t *testing.T) {
			for _, numNodes := range []int{1, 3} {
				t.Run(fmt.Sprintf("numNodes=%d", numNodes), func(t *testing.T) {
					testSingleProposerProtocol_CanBeEnabledAndDisabled(t, numNodes, upgrades)
				})
			}
		})
	}
}

func testSingleProposerProtocol_CanBeEnabledAndDisabled(
	t *testing.T,
	numNodes int,
	mode opera.Upgrades,
) {
	require := require.New(t)

	// The network is initially started using the distributed protocol.
	mode.SingleProposerBlockFormation = false
	net := StartIntegrationTestNet(t, IntegrationTestNetOptions{
		NumNodes: numNodes,
		Upgrades: &mode,
	})

	// Test that before the switch transactions can be processed.
	address := common.Address{0x42}
	_, err := net.EndowAccount(address, big.NewInt(50))
	require.NoError(err)

	// Initially, Version 2 of the event protocol should be used.
	client, err := net.GetClient()
	require.NoError(err)
	defer client.Close()
	require.Equal(2, getUsedEventVersion(t, client))

	type upgrades struct {
		SingleProposerBlockFormation bool
	}
	type rulesType struct {
		Upgrades upgrades
	}

	// Make sure that the switch can be performed multiple times.
	for range 2 {
		steps := []struct {
			versionBefore int
			flagValue     bool
			versionAfter  int
		}{
			{2, true, 3},  // Enable single-proposer protocol
			{3, false, 2}, // Disable single-proposer protocol
		}
		for _, step := range steps {
			// Send the network rule update.
			rulesDiff := rulesType{
				Upgrades: upgrades{SingleProposerBlockFormation: step.flagValue},
			}
			UpdateNetworkRules(t, net, rulesDiff)

			// The rules only take effect after the epoch change. Make sure that
			// until then, transactions can be processed.
			_, err = net.EndowAccount(address, big.NewInt(50))
			require.NoError(err)

			// At this point, the old version should still be used.
			require.Equal(step.versionBefore, getUsedEventVersion(t, client))

			// Advance the epoch by one, enabling the single-proposer protocol.
			net.AdvanceEpoch(t, 1)

			// Check that transactions can still be processed after the epoch change.
			for range 5 {
				_, err = net.EndowAccount(address, big.NewInt(50))
				require.NoError(err)
			}

			// At this point, the new version should be used.
			require.Equal(step.versionAfter, getUsedEventVersion(t, client))
		}
	}

	// Run some consistency checks after the test.
	headers, err := net.GetHeaders()
	require.NoError(err)

	// Test parent/child relation properties.
	block_header.HeadersParentChildProperties(t, headers)
}

// getUsedEventVersion retrieves the current event version used by the network.
func getUsedEventVersion(
	t *testing.T,
	client *PooledEhtClient,
) int {
	t.Helper()
	require := require.New(t)

	// Get the current epoch.
	block := struct {
		Epoch hexutil.Uint64
	}{}
	err := client.Client().Call(&block, "eth_getBlockByNumber", rpc.BlockNumber(-1), false)
	require.NoError(err)

	// Get the head events of the current epoch.
	heads := []hexutil.Bytes{}
	err = client.Client().Call(&heads, "dag_getHeads", rpc.BlockNumber(block.Epoch))
	require.NoError(err)

	// Download one of the head events and fetch the version.
	event := struct {
		Version hexutil.Uint64
	}{}
	err = client.Client().Call(&event, "dag_getEvent", heads[0].String())
	require.NoError(err)

	return int(event.Version)
}
