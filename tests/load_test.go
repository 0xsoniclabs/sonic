package tests

import (
	"encoding/json"
	"math/big"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/config"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestLoadStressTest(t *testing.T) {
	t.Run("SingleProposer", func(t *testing.T) {
		testLoadStressTest(t, true)
	})
	t.Run("DistributedProposer", func(t *testing.T) {
		testLoadStressTest(t, false)
	})
}

func testLoadStressTest(t *testing.T, singleProposer bool) {
	require := require.New(t)
	const (
		NumNodes               = 1
		NumAccounts            = 50_000
		TransactionsPerAccount = 1
		Rate                   = 500 // transactions per second
	)

	t.Logf("Created %d accounts", NumAccounts)

	accounts := make([]*Account, NumAccounts)
	addresses := make([]common.Address, NumAccounts)
	for i := range accounts {
		accounts[i] = NewAccount()
		addresses[i] = accounts[i].Address()
	}

	t.Logf("Starting test network")
	upgrades := opera.GetSonicUpgrades()
	upgrades.SingleProposerBlockFormation = singleProposer
	net := StartIntegrationTestNet(t, IntegrationTestNetOptions{
		Upgrades: &upgrades,
		NumNodes: NumNodes,
		ModifyConfig: func(config *config.Config) {
			config.Emitter.MaxTxsPerAddress = 1 << 16
			config.TxPool.AccountSlots = 1 << 16
			config.TxPool.AccountQueue = 1 << 16
			config.TxPool.GlobalSlots = 1 << 16
		},
	})

	t.Logf("Endowing %d accounts", len(addresses))
	_, err := net.EndowAccounts(addresses, big.NewInt(1e18))
	require.NoError(err)

	t.Logf("Update network rules")
	rules := getNetworkRules(t, net)
	rules.Emitter.Interval = inter.Timestamp(510 * time.Millisecond)
	rules.Economy.ShortGasPower.AllocPerSec = 1_000_000_000
	rules.Economy.LongGasPower.AllocPerSec = 1_000_000_000
	updateNetworkRules(t, net, rules)

	require.NoError(net.AdvanceEpoch(1))

	if false {
		rules = getNetworkRules(t, net)
		prettyJson, err := json.MarshalIndent(rules, "", "  ")
		require.NoError(err, "Failed to marshal network rules to JSON")
		t.Logf("Network rules updated: %+v", string(prettyJson))
	}

	signer := types.LatestSignerForChainID(net.GetChainId())

	// Pre-signing all transactions
	t.Logf("Pre-signing %d transactions for %d accounts", TransactionsPerAccount, NumAccounts)
	transactions := make([]*types.Transaction, NumAccounts*TransactionsPerAccount)
	var wg sync.WaitGroup
	wg.Add(NumAccounts)
	for i, account := range accounts {
		go func(i int, account *Account) {
			defer wg.Done()
			for n := range TransactionsPerAccount {
				tx := types.MustSignNewTx(
					account.PrivateKey,
					signer,
					&types.DynamicFeeTx{
						Nonce:     uint64(n),
						To:        &addresses[i],
						Value:     big.NewInt(0),
						Gas:       21000,
						GasFeeCap: big.NewInt(1e12),
						GasTipCap: big.NewInt(1),
					},
				)
				tx.Hash() // warm up the hash cash
				transactions[n*NumAccounts+i] = tx
			}
		}(i, account)
	}
	wg.Wait()

	start := time.Now()

	allDone := make(chan struct{})
	endTimes := map[common.Hash]time.Time{}
	go func() {
		defer close(allDone)
		client, err := net.GetClient()
		require.NoError(err)
		defer client.Close()

		last, err := client.BlockNumber(t.Context())
		require.NoError(err)
		for len(endTimes) < len(transactions) {
			number, err := client.BlockNumber(t.Context())
			require.NoError(err)
			if number > last {
				now := time.Now()
				block, err := client.BlockByNumber(t.Context(), big.NewInt(int64(number)))
				require.NoError(err)
				for _, tx := range block.Transactions() {
					endTimes[tx.Hash()] = now
				}
			} else {
				time.Sleep(time.Millisecond)
			}
		}
	}()

	pacer := &pacer{rate: Rate}
	pacer.Start()
	defer pacer.Stop()

	t.Logf("Starting load generators ...")
	startTimes := make([]time.Time, len(transactions))
	var wg2 sync.WaitGroup
	workers := 2 * Rate
	wg2.Add(workers)
	var counter atomic.Uint32
	for range workers {
		go func() {
			defer wg2.Done()
			client, err := net.GetClient()
			require.NoError(err)
			defer client.Close()
			for {
				next := counter.Add(1) - 1
				if next >= uint32(len(transactions)) {
					break // all transactions have been sent
				}
				tx := transactions[next]

				pacer.Wait()
				startTimes[next] = time.Now()
				require.NoError(client.SendTransaction(t.Context(), tx))
			}
		}()
	}
	wg2.Wait()

	<-allDone

	samples := make([]time.Duration, 0, len(transactions))
	for i, tx := range transactions {
		startTime := startTimes[i]
		endTime, ok := endTimes[tx.Hash()]
		if !ok {
			t.Fatalf("Transaction %s not found in end times", tx.Hash())
		}
		samples = append(samples, endTime.Sub(startTime))
	}

	/*
		last := time.Time{}
		for _, time := range startTimes {
			fmt.Printf("Send at %v - delta %v\n", time, time.Sub(last))
			last = time
		}
	*/

	/*
		t.Logf("Collected delays: %d samples", len(samples))
		for _, sample := range samples {
			fmt.Printf("%d\n", sample.Milliseconds())
		}
	*/

	duration := time.Since(start)
	t.Logf("Load test completed in %v", duration)
	t.Logf("Total transactions: %d", NumAccounts*TransactionsPerAccount)
	t.Logf("Average transactions per second: %.2f", float64(NumAccounts*TransactionsPerAccount)/duration.Seconds())

	sum := time.Duration(0)
	for _, sample := range samples {
		sum += sample
	}
	avg := sum / time.Duration(len(samples))

	slices.Sort(samples)
	t.Logf("Avg: %v", avg)
	t.Logf("P50: %v", samples[len(samples)*50/100])
	t.Logf("P90: %v", samples[len(samples)*90/100])
	t.Logf("P95: %v", samples[len(samples)*95/100])
	t.Logf("Max: %v", samples[len(samples)-1])
}

type pacer struct {
	rate float64

	next <-chan struct{}
	quit chan<- struct{}
	done <-chan struct{}
}

func (p *pacer) Start() {
	quit := make(chan struct{}, 1)
	p.quit = quit
	next := make(chan struct{}, 1)
	p.next = next
	done := make(chan struct{}, 1)
	p.done = done
	go func() {
		defer close(done)
		pending := 0.0
		last := time.Now()
		for {
			now := time.Now()
			new := now.Sub(last)
			last = now
			pending += new.Seconds() * p.rate
			if pending < 1 {
				time.Sleep(time.Second / time.Duration(p.rate))
			}
			for pending >= 1 {
				select {
				case next <- struct{}{}:
					pending--
				case <-quit:
					return
				}
			}
		}
	}()
}

func (p *pacer) Stop() {
	close(p.quit)
	<-p.done
}

func (p *pacer) Wait() {
	<-p.next
}
