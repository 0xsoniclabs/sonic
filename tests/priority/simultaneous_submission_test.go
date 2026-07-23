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

package priority

import (
	"context"
	"math/big"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// These tests empirically measure how effective the transaction-priorities
// feature is at making a prioritized transaction execute before a high-tip,
// non-prioritized one when the two are submitted at the same time via ordinary
// RPC (eth_sendRawTransaction).
//
// Why this is a probability and not a certainty: within a single block, block
// formation deterministically orders every prioritized transaction before every
// ordinary one regardless of tip (the authoritative stage). So a high-tip
// transaction can only "win" by landing in an *earlier block*. Which block a
// transaction lands in is decided by the best-effort emitter stage, where with
// multiple validators an ordinary transaction must wait for its assigned
// validator's ~8s "turn", while a prioritized transaction can be piggybacked by
// any validator immediately. The win probability therefore reflects the
// effectiveness of that best-effort stage under different conditions.
//
// The tests only report: each variant logs the measured probability and the
// block-gap distribution; nothing about the measured quantity is asserted.

// trialsPerVariant is the number of independent duels run per measured variant.
// Raise it for a sharper probability estimate at the cost of longer wall-clock:
// each legacy multi-validator duel can wait up to one ~8s validator turn for the
// ordinary transaction to be emitted.
const trialsPerVariant = 100

// duelSink receives the duel and congestion transfers; its value is irrelevant.
var duelSink = common.Address{0x99}

// duelOutcome is the result of a single duel between a prioritized and an
// ordinary transaction submitted simultaneously.
type duelOutcome struct {
	prioFirst bool // prio executed before ord in global (block, index) order
	sameBlock bool // both landed in the same block
	blockGap  int  // ordBlock - prioBlock (positive => prio in an earlier block)
}

// startPriorityNet starts a fresh integration test network with the Brio
// hardfork plus TransactionPriorities enabled and the given validator stakes,
// applying any extra upgrade flags via configure, and configures generous
// priority-registry limits so rate limiting never trims a prioritized
// transaction.
func startPriorityNet(t *testing.T, stakes []uint64, configure func(*opera.Upgrades)) *tests.IntegrationTestNet {
	t.Helper()
	require := require.New(t)

	upgrades := opera.GetBrioUpgrades()
	upgrades.TransactionPriorities = true
	if configure != nil {
		configure(&upgrades)
	}
	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		Upgrades:        &upgrades,
		ValidatorsStake: stakes,
	})

	client, err := net.GetClient()
	require.NoError(err)
	defer client.Close()
	configureHighPriorityLimits(t, net, client)
	return net
}

// equalStakes returns a stake distribution for n equally-weighted validators.
func equalStakes(n int) []uint64 {
	stakes := make([]uint64, n)
	for i := range stakes {
		stakes[i] = 100
	}
	return stakes
}

// newRegisteredPrioAccount creates a funded account and registers it in the
// priority registry as prioritized. The same account is reused across a net's
// trials; because duels run sequentially and prioritized transactions are
// piggybacked regardless of their own turn assignment, reuse does not bias the
// measurement.
func newRegisteredPrioAccount(t *testing.T, net *tests.IntegrationTestNet) *tests.Account {
	t.Helper()
	acc := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))
	setPrioritized(t, net, acc.Address(), 1, 1, 1)
	return acc
}

// suggestGasPrices returns a low gas price (the node's suggestion, used for the
// prioritized transaction) and a much higher one (used for the ordinary
// transaction so the feature is shown to beat a high tip).
func suggestGasPrices(t *testing.T, net *tests.IntegrationTestNet) (low, high *big.Int) {
	t.Helper()
	require := require.New(t)
	client, err := net.GetClient()
	require.NoError(err)
	defer client.Close()
	suggested, err := client.SuggestGasPrice(t.Context())
	require.NoError(err)
	low = new(big.Int).Set(suggested)
	high = new(big.Int).Add(suggested, big.NewInt(1_000_000_000_000)) // +1000 gwei
	return low, high
}

// runDuel submits one prioritized transaction (low tip, nonce prioNonce from
// prioAcc) and one ordinary transaction (high tip, from a fresh account)
// simultaneously via ordinary RPC, prio to node prioNode and ord to node
// ordNode, and returns which one executed first. A fresh ordinary account per
// call keeps each trial's validator-turn assignment independent.
func runDuel(
	t *testing.T,
	net *tests.IntegrationTestNet,
	signer types.Signer,
	prioAcc *tests.Account,
	prioNonce uint64,
	prioNode, ordNode int,
	lowGasPrice, highGasPrice *big.Int,
) duelOutcome {
	t.Helper()
	require := require.New(t)

	ordAcc := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))

	prioTx := types.MustSignNewTx(prioAcc.PrivateKey, signer, &types.LegacyTx{
		Nonce:    prioNonce,
		To:       &duelSink,
		Value:    big.NewInt(1),
		Gas:      21000,
		GasPrice: lowGasPrice,
	})
	ordTx := types.MustSignNewTx(ordAcc.PrivateKey, signer, &types.LegacyTx{
		Nonce:    0,
		To:       &duelSink,
		Value:    big.NewInt(1),
		Gas:      21000,
		GasPrice: highGasPrice,
	})

	prioClient, err := net.GetClientConnectedToNode(prioNode)
	require.NoError(err)
	defer prioClient.Close()
	ordClient, err := net.GetClientConnectedToNode(ordNode)
	require.NoError(err)
	defer ordClient.Close()

	// Release both submissions at the same instant. Errors are collected and
	// asserted on the test goroutine, never inside the sender goroutines.
	start := make(chan struct{})
	var wg sync.WaitGroup
	var prioErr, ordErr error
	wg.Go(func() {
		defer wg.Done()
		<-start
		prioErr = prioClient.SendTransaction(t.Context(), prioTx)
	})
	wg.Go(func() {
		defer wg.Done()
		<-start
		ordErr = ordClient.SendTransaction(t.Context(), ordTx)
	})
	close(start)
	wg.Wait()
	require.NoError(prioErr)
	require.NoError(ordErr)

	receipts, err := net.TryGetReceipts(1*time.Minute, []common.Hash{prioTx.Hash(), ordTx.Hash()})
	require.NoError(err)
	prioR, ordR := receipts[0], receipts[1]
	require.Equal(types.ReceiptStatusSuccessful, prioR.Status)
	require.Equal(types.ReceiptStatusSuccessful, ordR.Status)

	prioBlk := prioR.BlockNumber.Uint64()
	ordBlk := ordR.BlockNumber.Uint64()
	prioFirst := prioBlk < ordBlk ||
		(prioBlk == ordBlk && prioR.TransactionIndex < ordR.TransactionIndex)
	return duelOutcome{
		prioFirst: prioFirst,
		sameBlock: prioBlk == ordBlk,
		blockGap:  int(ordBlk) - int(prioBlk),
	}
}

// reportOutcomes logs the empirical win probability and block-gap distribution
// of a variant's duels. It is report-only: nothing is asserted.
func reportOutcomes(t *testing.T, outcomes []duelOutcome) {
	t.Helper()
	n := len(outcomes)
	if n == 0 {
		return
	}
	wins, sameBlock, prioEarlier, ordEarlier := 0, 0, 0, 0
	gaps := make([]int, 0, n)
	sumGap := 0.0
	for _, o := range outcomes {
		if o.prioFirst {
			wins++
		}
		switch {
		case o.sameBlock:
			sameBlock++
		case o.blockGap > 0:
			prioEarlier++
		default:
			ordEarlier++
		}
		gaps = append(gaps, o.blockGap)
		sumGap += float64(o.blockGap)
	}
	sort.Ints(gaps)
	t.Logf("prio-wins=%d/%d p=%.3f | sameBlock=%d prioEarlierBlock=%d ordEarlierBlock=%d | blockGap(ord-prio) mean=%.2f median=%d",
		wins, n, float64(wins)/float64(n),
		sameBlock, prioEarlier, ordEarlier,
		sumGap/float64(n), gaps[n/2])
}

// TestPriority_SimultaneousSubmission_ValidatorTargeting measures the prio-win
// probability as a function of which validator each transaction is submitted to,
// covering both the same-vs-different-validator and low-vs-high-stake dimensions
// on a single network. Nodes 0 and 1 hold most of the stake; nodes 2 and 3 are
// low-stake.
func TestPriority_SimultaneousSubmission_ValidatorTargeting(t *testing.T) {
	net := startPriorityNet(t, []uint64{100, 100, 1, 1}, nil)
	signer := types.LatestSignerForChainID(net.GetChainId())
	low, high := suggestGasPrices(t, net)
	prioAcc := newRegisteredPrioAccount(t, net)

	cases := map[string]struct{ prioNode, ordNode int }{
		"same_high_stake":           {prioNode: 0, ordNode: 0},
		"same_low_stake":            {prioNode: 2, ordNode: 2},
		"different_prioHigh_ordLow": {prioNode: 0, ordNode: 2},
		"different_prioLow_ordHigh": {prioNode: 2, ordNode: 0},
	}

	var nonce uint64
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			outcomes := make([]duelOutcome, trialsPerVariant)
			for i := range outcomes {
				outcomes[i] = runDuel(t, net, signer, prioAcc, nonce, tc.prioNode, tc.ordNode, low, high)
				nonce++
			}
			reportOutcomes(t, outcomes)
		})
	}
}

// TestPriority_SimultaneousSubmission_ValidatorCountSweep measures how the
// prio-win probability changes with the number of validators. With a single
// validator the turn mechanism is trivial, so both transactions land in the same
// block and the prioritized one always wins; more validators introduce the turn
// wait that the feature's piggyback path is designed to bypass.
func TestPriority_SimultaneousSubmission_ValidatorCountSweep(t *testing.T) {
	cases := map[string]int{
		"1_validator":  1,
		"3_validators": 3,
		"7_validators": 7,
	}
	for name, count := range cases {
		t.Run(name, func(t *testing.T) {
			net := startPriorityNet(t, equalStakes(count), nil)
			signer := types.LatestSignerForChainID(net.GetChainId())
			low, high := suggestGasPrices(t, net)
			prioAcc := newRegisteredPrioAccount(t, net)

			outcomes := make([]duelOutcome, trialsPerVariant)
			for i := range outcomes {
				outcomes[i] = runDuel(t, net, signer, prioAcc, uint64(i), 0, 0, low, high)
			}
			reportOutcomes(t, outcomes)
		})
	}
}

// TestPriority_SimultaneousSubmission_SingleProposerVsLegacy compares the
// prio-win probability under legacy multi-proposer emission against
// single-proposer block formation, submitting the two transactions to different
// validators in both modes.
func TestPriority_SimultaneousSubmission_SingleProposerVsLegacy(t *testing.T) {
	cases := map[string]func(*opera.Upgrades){
		"legacy":          nil,
		"single_proposer": func(u *opera.Upgrades) { u.SingleProposerBlockFormation = true },
	}
	for name, configure := range cases {
		t.Run(name, func(t *testing.T) {
			net := startPriorityNet(t, equalStakes(4), configure)
			signer := types.LatestSignerForChainID(net.GetChainId())
			low, high := suggestGasPrices(t, net)
			prioAcc := newRegisteredPrioAccount(t, net)

			outcomes := make([]duelOutcome, trialsPerVariant)
			for i := range outcomes {
				outcomes[i] = runDuel(t, net, signer, prioAcc, uint64(i), 0, 1, low, high)
			}
			reportOutcomes(t, outcomes)
		})
	}
}

// TestPriority_SimultaneousSubmission_UnderCongestion measures the prio-win
// probability while the network is flooded with high-tip ordinary traffic,
// stressing the best-effort stage (block space and piggyback throughput). The
// congestion is generated from a pool of pre-funded accounts submitting via
// ordinary RPC, spread across all nodes, for the duration of the trials.
func TestPriority_SimultaneousSubmission_UnderCongestion(t *testing.T) {
	net := startPriorityNet(t, equalStakes(4), nil)
	signer := types.LatestSignerForChainID(net.GetChainId())
	low, high := suggestGasPrices(t, net)
	prioAcc := newRegisteredPrioAccount(t, net)

	congestion := tests.MakeAccountsWithBalance(t, net, 8, new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1000)))
	stop := make(chan struct{})
	var genWg sync.WaitGroup
	for i, acc := range congestion {
		genWg.Add(1)
		go func(acc *tests.Account, node int) {
			defer genWg.Done()
			client, err := net.GetClientConnectedToNode(node)
			if err != nil {
				return
			}
			defer client.Close()
			for {
				select {
				case <-stop:
					return
				default:
				}
				nonce, err := client.PendingNonceAt(context.Background(), acc.Address())
				if err != nil {
					continue
				}
				tx := types.MustSignNewTx(acc.PrivateKey, signer, &types.LegacyTx{
					Nonce:    nonce,
					To:       &duelSink,
					Value:    big.NewInt(1),
					Gas:      21000,
					GasPrice: high,
				})
				_ = client.SendTransaction(context.Background(), tx)
				time.Sleep(3 * time.Millisecond)
			}
		}(acc, i%net.NumNodes())
	}

	outcomes := make([]duelOutcome, trialsPerVariant)
	for i := range outcomes {
		outcomes[i] = runDuel(t, net, signer, prioAcc, uint64(i), 0, 1, low, high)
	}
	close(stop)
	genWg.Wait()
	reportOutcomes(t, outcomes)
}
