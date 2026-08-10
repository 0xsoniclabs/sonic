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

package priorities

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/config"
	emitter "github.com/0xsoniclabs/sonic/gossip/emitter/config"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

// The tests in this file try to evaluate the probability of a prioritized
// transaction winning a race against an ordinary one under different conditions.
// They do not assert any behavior.
// When running with many validators, it can happen that the file watcher
// complains about too many files. In that case comment out:
// - make_node.go:136-137, unlocking the fake validator account
// - make_node.go:262, registering the keystore account backend

// The committed values keep the CI run short and are far too small to be
// meaningful. Raise them by hand to gather statistics.
const (
	numValidators            = 2
	racesPerVariant          = 1
	congestedRacesPerVariant = 1 // congested runs are significantly slower
)

const racesInParallel = 10

// raceGasPrice is paid by every transaction of these tests, high enough to
// remain sufficient even when the base fee rises under congestion.
var raceGasPrice = big.NewInt(100_000 * params.GWei)

// raceAccountBalance endows a race account with a hundred times what the single
// transaction it sends costs. Since the accounts are created per race, endowing
// them generously would exhaust the sponsor.
var raceAccountBalance = new(big.Int).Mul(raceGasPrice, big.NewInt(100*21_000))

func TestPriorities_PriorityRace_ValidatorTargeting(t *testing.T) {
	// Half the validators hold a hundred times the stake of the other half, so
	// almost every turn falls to one of the big ones.
	const half = numValidators / 2
	stakes := slices.Concat(
		slices.Repeat([]uint64{100}, half), slices.Repeat([]uint64{1}, half))
	const high, low = 0, half // first big and first small validator

	cases := map[string]struct{ prioNode, ordNode int }{
		"PrioToHigh_OrdinaryToHigh": {prioNode: high, ordNode: high},
		"PrioToLow_OrdinaryToLow":   {prioNode: low, ordNode: low},
		"PrioToHigh_OrdinaryToLow":  {prioNode: high, ordNode: low},
		"PrioToLow_OrdinaryToHigh":  {prioNode: low, ordNode: high},
	}

	net := newRaceEnv(t, stakes, nil)

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			runRaces(t, net,
				newRaceSetup(t, net, tc.prioNode, tc.ordNode, racesPerVariant))
		})
	}
}

func TestPriorities_PriorityRace_ValidatorCounts(t *testing.T) {
	for count := 1; count <= numValidators; count *= 2 {
		t.Run(fmt.Sprintf("%d_validators", count), func(t *testing.T) {
			runTargetingVariants(t,
				newRaceEnv(t, slices.Repeat([]uint64{1}, count), nil),
				racesPerVariant, 0)
		})
	}
}

func TestPriorities_PriorityRace_ProposerMode(t *testing.T) {
	cases := map[string]func(*opera.Upgrades){
		"legacy":          nil,
		"single_proposer": func(u *opera.Upgrades) { u.SingleProposerBlockFormation = true },
	}
	for name, configure := range cases {
		t.Run(name, func(t *testing.T) {
			runTargetingVariants(t,
				newRaceEnv(t, slices.Repeat([]uint64{1}, numValidators), configure),
				racesPerVariant, 0)
		})
	}
}

func TestPriorities_PriorityRace_UnderCongestion(t *testing.T) {
	cases := map[string]int{
		"1_sender":  1,
		"2_sender":  2,
		"4_sender":  4,
		"8_sender":  8,
		"16_sender": 16,
		"32_sender": 32,
	}

	// A network per case, so that the setup of one case is not slowed down by
	// the transactions the previous case's congestion left in the pools.
	for name, numSenders := range cases {
		t.Run(name, func(t *testing.T) {
			runTargetingVariants(t,
				newRaceEnv(t, slices.Repeat([]uint64{1}, numValidators), nil),
				congestedRacesPerVariant, numSenders)
		})
	}
}

// newRaceEnv starts a priority net with the given validator stakes and upgrade
// configuration. Since the timing of event production decides most races, the
// two intervals the integration test harness accelerates, Min and Confirming,
// are restored to their production values.
func newRaceEnv(
	t *testing.T,
	stakes []uint64,
	configure func(*opera.Upgrades),
) *tests.IntegrationTestNet {
	t.Helper()

	upgrades := opera.GetBrioUpgrades()
	upgrades.TransactionPriorities = true
	if configure != nil {
		configure(&upgrades)
	}

	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		Upgrades:        &upgrades,
		ValidatorsStake: stakes,
		ModifyConfig: func(c *config.Config) {
			production := emitter.DefaultConfig().EmitIntervals
			c.Emitter.EmitIntervals.Min = production.Min
			c.Emitter.EmitIntervals.Confirming = production.Confirming

			// Each validator runs exactly once, so the emitters need not wait
			// out the doublesign protection period before their first event.
			c.Emitter.EmitIntervals.DoublesignProtection = 0
		},
	})
	setPriorityConfig(t, net, math.MaxUint64, math.MaxUint64)

	// The emission interval is a network rule, so it cannot be set through the
	// node configuration above. The stall thresholds stay as the harness set
	// them, being failure timers rather than part of the emission cadence.
	var rules struct{ Emitter struct{ Interval uint64 } }
	rules.Emitter.Interval = uint64(opera.DefaultEmitterRules().Interval)
	tests.UpdateNetworkRules(t, net, rules)
	net.AdvanceEpoch(t, 1)

	return net
}

// runTargetingVariants runs `races` races with both transactions submitted to
// the same node and, unless the network has a single node, with the ordinary
// transaction submitted to a different one. If congestionSenders is positive,
// that many senders flood the network while the races run.
func runTargetingVariants(
	t *testing.T,
	net *tests.IntegrationTestNet,
	races int,
	congestionSenders int,
) {
	t.Helper()

	variants := []struct {
		name              string
		prioNode, ordNode int
	}{
		{"same_validator", 0, 0},
		{"different_validators", 0, 1},
	}
	if net.NumNodes() < 2 {
		variants = variants[:1]
	}

	// All accounts are created up front, so that their setup transactions,
	// which pay the network's suggested gas price, do not have to compete with
	// the congestion.
	setups := make([]raceSetup, len(variants))
	for i, variant := range variants {
		setups[i] = newRaceSetup(t, net, variant.prioNode, variant.ordNode, races)
	}

	if congestionSenders > 0 {
		stop := startCongestion(t, net, congestionSenders)
		defer stop()
	}

	for i, variant := range variants {
		t.Run(variant.name, func(t *testing.T) {
			runRaces(t, net, setups[i])
		})
	}
}

// raceSetup holds the clients and accounts of one targeting variant.
type raceSetup struct {
	prioClient, ordClient *tests.PooledEhtClient
	prioAccs, ordAccs     []*tests.Account
}

// newRaceSetup connects to the nodes the prioritized and the ordinary
// transaction are submitted to and creates one account per race for each of
// them. Fresh accounts per race keep the trials from sharing a sender's
// validator-turn assignment.
func newRaceSetup(
	t *testing.T,
	net *tests.IntegrationTestNet,
	prioNode, ordNode int,
	races int,
) raceSetup {
	t.Helper()
	require := require.New(t)

	prioClient, err := net.GetClientConnectedToNode(prioNode)
	require.NoError(err)
	t.Cleanup(prioClient.Close)
	ordClient, err := net.GetClientConnectedToNode(ordNode)
	require.NoError(err)
	t.Cleanup(ordClient.Close)

	return raceSetup{
		prioClient: prioClient,
		ordClient:  ordClient,
		prioAccs: newFundedPrioritizedAccounts(
			t, net, prioClient, races, raceAccountBalance, 1, 1, 1),
		ordAccs: newFundedAccounts(
			t, net, ordClient, races, raceAccountBalance),
	}
}

// runRaces runs one race per account pair of the given setup, at most
// racesInParallel of them at a time, and logs the empirical win probability.
func runRaces(
	t *testing.T,
	net *tests.IntegrationTestNet,
	setup raceSetup,
) {
	t.Helper()

	races := len(setup.prioAccs)
	prioReceipts := make([]*types.Receipt, races)
	ordReceipts := make([]*types.Receipt, races)
	errs := make([]error, races)

	var wg sync.WaitGroup
	slots := make(chan struct{}, racesInParallel)
	for i := range races {
		slots <- struct{}{}
		wg.Go(func() {
			defer func() { <-slots }()
			prioReceipts[i], ordReceipts[i], errs[i] = runRace(t, net,
				setup.prioAccs[i], setup.ordAccs[i], setup.prioClient, setup.ordClient)
		})
	}
	wg.Wait()

	require.NoError(t, errors.Join(errs...))

	earlierBlock, earlierTx, later := 0, 0, 0
	prioMissing, ordMissing, bothMissing := 0, 0, 0
	for i := range races {
		prio, ord := prioReceipts[i], ordReceipts[i]
		switch {
		case prio == nil && ord == nil:
			bothMissing++
		case prio == nil:
			prioMissing++
		case ord == nil:
			ordMissing++
		case prio.BlockNumber.Cmp(ord.BlockNumber) < 0:
			earlierBlock++
		case prio.BlockNumber.Cmp(ord.BlockNumber) == 0 &&
			prio.TransactionIndex < ord.TransactionIndex:
			earlierTx++
		default:
			later++
		}
	}

	// A transaction that is not included in time loses against one that is.
	// Races in which neither was included count as a loss as well.
	wins := earlierBlock + earlierTx + ordMissing
	_, _ = fmt.Fprintf(t.Output(),
		"wins=%d/%d winRate=%.1f%% "+
			"earlierBlock=%d sameBlockEarlierTx=%d later=%d "+
			"prioMissing=%d ordMissing=%d bothMissing=%d\n",
		wins, races,
		float64(wins)/float64(races)*100,
		earlierBlock, earlierTx, later,
		prioMissing, ordMissing, bothMissing)
}

// runRace submits one prioritized transaction (from prioAcc) and one ordinary
// transaction (from ordAcc) simultaneously via ordinary RPC, prio through
// prioClient and ord through ordClient. Both accounts are unused and both
// transactions pay the same gas price, so neither fee nor nonce position can
// separate them. It returns the receipt of each transaction, or nil for a
// transaction which was not included in time. Since it runs on its own
// goroutine, it reports failures as an error rather than through `t`.
func runRace(
	t *testing.T,
	net *tests.IntegrationTestNet,
	prioAcc, ordAcc *tests.Account,
	prioClient, ordClient *tests.PooledEhtClient,
) (prioReceipt, ordReceipt *types.Receipt, err error) {
	prioTx := newSignedTx(t, net, prioAcc, 0, 1, 21000, raceGasPrice)
	ordTx := newSignedTx(t, net, ordAcc, 0, 1, 21000, raceGasPrice)

	// Release both submissions at the same instant. The barrier is only opened
	// once both senders have reached it, so neither is delayed by the other
	// goroutine's start-up.
	start := make(chan struct{})
	var ready, wg sync.WaitGroup
	ready.Add(2)
	var prioErr, ordErr error
	wg.Go(func() {
		ready.Done()
		<-start
		prioErr = prioClient.SendTransaction(t.Context(), prioTx)
	})
	wg.Go(func() {
		ready.Done()
		<-start
		ordErr = ordClient.SendTransaction(t.Context(), ordTx)
	})
	ready.Wait()
	close(start)
	wg.Wait()
	if err := errors.Join(prioErr, ordErr); err != nil {
		return nil, nil, err
	}

	// Await both receipts independently, so that a transaction which is not
	// included in time does not hide the other one's outcome.
	wg.Go(func() { prioReceipt, prioErr = tryGetReceipt(net, prioTx.Hash()) })
	wg.Go(func() { ordReceipt, ordErr = tryGetReceipt(net, ordTx.Hash()) })
	wg.Wait()
	if err := errors.Join(prioErr, ordErr); err != nil {
		return nil, nil, err
	}

	if prioReceipt != nil && prioReceipt.Status != types.ReceiptStatusSuccessful {
		return nil, nil, errors.New("prioritized transaction failed")
	}
	if ordReceipt != nil && ordReceipt.Status != types.ReceiptStatusSuccessful {
		return nil, nil, errors.New("ordinary transaction failed")
	}

	return prioReceipt, ordReceipt, nil
}

// tryGetReceipt waits up to five seconds for the receipt of the given
// transaction. It returns a nil receipt if the transaction was not included
// in time.
func tryGetReceipt(
	net *tests.IntegrationTestNet,
	txHash common.Hash,
) (*types.Receipt, error) {
	// after 5s delay, priority becomes useless
	receipt, err := net.TryGetReceipt(5*time.Second, txHash)
	if errors.Is(err, context.DeadlineExceeded) {
		return nil, nil
	}
	return receipt, err
}

// congestionThreshold is the number of pending transactions in a pool for it
// to be considered congested. The threshold is set to 90% of the pool capacity.
const congestionThreshold = 1280 * 90 / 100

// startCongestion floods the network with ordinary transactions from
// `numSenders` fresh accounts spread over all nodes, until the returned stop
// function is called. It returns only once every node reports a pending
// backlog of at least congestionThreshold transactions.
func startCongestion(
	t *testing.T,
	net *tests.IntegrationTestNet,
	numSenders int,
) (stop func()) {
	t.Helper()
	require := require.New(t)

	clients := make([]*tests.PooledEhtClient, net.NumNodes())
	for i := range clients {
		client, err := net.GetClientConnectedToNode(i)
		require.NoError(err)
		clients[i] = client
	}

	// Funds for ten million transfers at raceGasPrice, so that no sender runs
	// dry and stops contributing to the congestion unnoticed. Each group's
	// endowment is awaited on the node its senders submit to below, so that
	// their first transaction does not race the endowment's propagation.
	balance := new(big.Int).Mul(raceGasPrice, big.NewInt(21_000*10_000_000))
	senders := make([]*tests.Account, numSenders)
	for i, client := range clients {
		count := numSenders / len(clients)
		if i < numSenders%len(clients) {
			count++
		}
		for j, sender := range newFundedAccounts(t, net, client, count, balance) {
			senders[i+j*len(clients)] = sender
		}
	}

	done := make(chan struct{})
	var wg sync.WaitGroup
	for i, sender := range senders {
		client := clients[i%len(clients)]
		wg.Go(func() {
			for nonce := uint64(0); ; {
				select {
				case <-done:
					return
				default:
				}
				tx := newSignedTx(t, net, sender, nonce, 1, 21000, raceGasPrice)
				if client.SendTransaction(t.Context(), tx) == nil {
					// advance nonce iff tx was accepted
					nonce++
					continue
				}
				// only sleep if tx was rejected; otherwise busy loop
				time.Sleep(5 * time.Millisecond)
				// A rejection usually means a full pool, but a previously
				// accepted transaction may also have been evicted from it.
				// Resynchronizing keeps the resulting nonce gap from stopping
				// this sender for the rest of the run.
				if pending, err := client.PendingNonceAt(
					t.Context(), sender.Address()); err == nil {
					nonce = pending
				}
			}
		})
	}

	// Wait for the flood to build up, so that every node's pool is saturated
	// before the first race starts.
	for _, client := range clients {
		var status struct{ Pending hexutil.Uint }
		require.NoError(tests.WaitFor(t.Context(),
			func(ctx context.Context) (bool, error) {
				err := client.Client().CallContext(ctx, &status, "txpool_status")
				return uint64(status.Pending) >= congestionThreshold, err
			}))
	}

	return func() {
		close(done)
		wg.Wait()
		for _, client := range clients {
			client.Close()
		}
	}
}
