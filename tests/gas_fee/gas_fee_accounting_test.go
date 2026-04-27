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

package gasfee

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

import (
	"maps"
	"math/big"
	"slices"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/drivermodule"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/opera/contracts/driver/drivercall"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/utils/signers/internaltx"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestTxFeeAccounting_EpochSealingReportsAggregatedFees(t *testing.T) {
	testCases := map[string]bool{
		"distributed_block_formation": false,
		"single_proposer":             true,
	}

	for name, mode := range testCases {
		t.Run(name, func(t *testing.T) {
			upgrades := opera.GetBrioUpgrades()
			upgrades.GasSubsidies = true
			upgrades.TransactionBundles = true
			upgrades.SingleProposerBlockFormation = mode
			testTxFeeAccounting_EpochSealingReportsAggregatedFees(t, tests.IntegrationTestNetOptions{
				Upgrades: &upgrades,
				NumNodes: 3,
			})
		})
	}
}

func testTxFeeAccounting_EpochSealingReportsAggregatedFees(
	t *testing.T,
	options tests.IntegrationTestNetOptions,
) {
	net := tests.StartIntegrationTestNet(t, options)

	const numEpochs = 3
	const numTransactions = 100
	const submissionDelay = 10 * time.Millisecond

	// Create a slice of transactions to create background load on the net.
	txs := createTransactionMix(t, net, numTransactions)
	backgroundLoadDone := make(chan struct{})
	go func() {
		defer close(backgroundLoadDone)

		// Gradually submit transactions in the background.
		for _, tx := range txs {
			_, err := net.Send(tx)
			require.NoError(t, err)
			time.Sleep(submissionDelay)
		}

		// Wait for all of those to complete.
		waitForTransactionMixToBeComplete(t, net, txs)
	}()

	// Advance epochs every now and then.
	interEpochDelay := numTransactions * submissionDelay / (numEpochs + 1)
	for range numEpochs {
		net.AdvanceEpoch(t, 1)
		time.Sleep(interEpochDelay)
	}

	<-backgroundLoadDone

	// create a final epoch to cover all remaining transactions and a few
	// empty blocks.
	net.AdvanceEpoch(t, 1)

	// --- verification ---

	// Fetch all blocks with their transactions and receipts and check that the
	// sum of the reported fees in the sealing match the collected tx fees.
	blocks, err := net.GetBlocks(t.Context())
	require.NoError(t, err)

	epochSealers := []*types.Transaction{}

	totalFees := big.NewInt(0)
	for _, b := range blocks {
		for _, tx := range b.Transactions() {

			receipt, err := net.GetReceipt(tx.Hash())
			require.NoError(t, err)

			// --- mitigation for reporting bug ---
			// see https://github.com/0xsoniclabs/sonic-admin/issues/743

			// There is a bug in the system causing the effective gas price for
			// internal transactions to be non-zero. Until fixed, those prices
			// need to be corrected here.
			if internaltx.IsInternal(tx) {
				receipt.EffectiveGasPrice = big.NewInt(0)
			}
			// Same problem for sponsored transactions.
			if subsidies.IsSponsorshipRequest(tx) {
				receipt.EffectiveGasPrice = big.NewInt(0)
			}
			// --- end of reporting issue mitigation ---

			// Compute the effect fees charged for this transaction.
			txFees, err := drivermodule.ComputeEffectiveFee(tx, receipt)
			require.NoError(t, err)

			// Keep a running total.
			totalFees = new(big.Int).Add(totalFees, txFees.ToBig())

			// Check if the current transaction is sealing an epoch. If so, the
			// reported gas fees should match the running total.
			metrics, err := drivercall.ParseSealEpochArgs(tx)
			if err != nil {
				continue
			}

			epochSealers = append(epochSealers, tx)

			sumReportedFees := big.NewInt(0)
			for _, cur := range metrics {
				sumReportedFees.Add(sumReportedFees, cur.OriginatedTxFee)
			}

			// Check that the reported and total fees match.
			diff := new(big.Int).Sub(sumReportedFees, totalFees)
			require.Zero(t, diff.Sign(), "Difference in reported fees: %v", diff)
		}
	}

	// Verify that in each epoch sealing, every validator got the fees assigned
	// according to the transactions that have been suggested.
}

type Event struct {
	Epoch        uint64
	Creator      uint64
	Transactions []*types.Transaction
}

func getAllEvents(
	t *testing.T,
	net *tests.IntegrationTestNet,
) []Event {
	t.Helper()

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	currentEpoch := tests.GetCurrentEpoch(t, client)

	res := []Event{}
	for epoch := range currentEpoch + 1 {
		res = append(res, getEventsFromEpoch(t, client, epoch)...)
	}
	return res
}

func getEventsFromEpoch(
	t *testing.T,
	client *tests.PooledEhtClient,
	epoch uint64,
) []Event {
	// Start with the heads
	heads := tests.GetEventHeadsForEpoch(t, client, epoch)

	missingEvents := heads
	resolved := map[hash.Event]Event{}
	for len(missingEvents) > 0 {
		eventID := missingEvents[len(missingEvents)-1]
		missingEvents = missingEvents[:len(missingEvents)-1]

		if _, resolved := resolved[eventID]; resolved {
			continue
		}

		// Fetch information from the header.
		var result struct {
			Creator hexutil.Uint64  `json:"creator"`
			Parents []hexutil.Bytes `json:"parents"`
			// other fields are ignored
		}
		err := client.Client().Call(&result, "dag_getEvent", eventID.Hex())
		require.NoError(t, err)

		// Fetch transactions from the payload.
		var payloadResult struct {
			Transactions []*types.Transaction `json:"transactions"`
			// other fields are ignored
		}
		err = client.Client().Call(&payloadResult, "dag_getEventPayload", eventID.Hex(), true)
		require.NoError(t, err)

		event := Event{
			Epoch:        epoch,
			Creator:      uint64(result.Creator),
			Transactions: payloadResult.Transactions,
		}

		parents := []hash.Event{}
		for _, parent := range result.Parents {
			parents = append(parents, hash.Event(common.HexToHash(parent.String())))
		}

		// store event as resolved and add parents to missing list
		resolved[eventID] = event
		missingEvents = append(missingEvents, parents...)
	}

	return slices.Collect(maps.Values(resolved))

}
