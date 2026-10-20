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

package tests

import (
	"context"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/config"
	"github.com/0xsoniclabs/sonic/integration/makefakegenesis"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests/throughput"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// Value transfer throughput, once for transaction processing alone and once on
// a network targeting about 3 blocks per second:
//
//	go test ./tests -run '^$' -bench '^BenchmarkThroughput_Processing$' -benchtime 20x
//	go test ./tests -run '^$' -bench '^BenchmarkThroughput_Network$' -benchtime 1x

const (
	// networkEmitterInterval sets the block rate. A block takes one to three
	// events depending on the load, so check the reported blocks/s.
	networkEmitterInterval = 200 * time.Millisecond

	networkWarmup   = 5 * time.Second
	networkWindow   = 20 * time.Second
	networkSendJobs = 16
	// networkRoundsInFlight is the number of rounds a sender may be ahead of the chain.
	networkRoundsInFlight = 16
)

func fundedAccounts(senders []common.Address) []makefakegenesis.Account {
	balance := new(uint256.Int).Mul(uint256.NewInt(1e18), uint256.NewInt(1e9))
	res := make([]makefakegenesis.Account, len(senders))
	for i, sender := range senders {
		res[i] = makefakegenesis.Account{Address: sender, Balance: balance}
	}
	return res
}

// BenchmarkThroughput_Processing runs blocks of one transfer per sender through
// the state processor and reports the rate of the processing alone, leaving out
// the commit of the state and the creation of the block.
func BenchmarkThroughput_Processing(b *testing.B) {
	benchmarkProcessing(b, false)
}

// BenchmarkThroughput_ProcessingWithStateRoot is BenchmarkThroughput_Processing
// with the end of each block in the state, its commit and its state root in the
// measured time.
func BenchmarkThroughput_ProcessingWithStateRoot(b *testing.B) {
	benchmarkProcessing(b, true)
}

func benchmarkProcessing(b *testing.B, withStateRoot bool) {
	for _, load := range throughput.Loads {
		b.Run(load.Name, func(b *testing.B) {
			rules := opera.FakeNetRules(opera.GetDolceUpgrades())

			state, err := NewState(b.TempDir())
			require.NoError(b, err)
			defer func() { require.NoError(b, state.Close()) }()

			issuer := throughput.NewIssuer(big.NewInt(int64(rules.NetworkID)), throughput.NewKeys())
			require.NoError(b, state.ApplyGenesis(&makefakegenesis.GenesisJson{Accounts: fundedAccounts(issuer.Senders)}))

			number := int64(1)
			var blockTime time.Duration
			apply := func(txs []*types.Transaction) {
				for _, tx := range txs { // < nodes recover senders before processing
					_, err := types.Sender(issuer.Signer, tx)
					require.NoError(b, err)
				}
				block := types.NewBlockWithHeader(&types.Header{
					Number:   big.NewInt(number),
					Time:     uint64(number),
					GasLimit: rules.Blocks.MaxBlockGas,
					BaseFee:  rules.Economy.MinBaseFee,
				}).WithBody(types.Body{Transactions: txs})
				start := time.Now()
				receipts, err := state.ApplyBlock(rules, block)
				state.GetStateRoot()
				blockTime += time.Since(start)
				require.NoError(b, err)
				require.Len(b, receipts, len(txs))
				for _, receipt := range receipts {
					require.Equal(b, types.ReceiptStatusSuccessful, receipt.Status)
				}
				number++
			}

			if txs := issuer.Setup(b, load); len(txs) > 0 {
				apply(txs)
			}
			state.processTime = 0
			blockTime = 0

			transfers := 0
			for b.Loop() {
				var txs []*types.Transaction
				for sender := range issuer.Senders {
					txs = append(txs, issuer.Transfers(b, load, sender)...)
				}
				apply(txs)
				transfers += len(issuer.Senders)
			}

			seconds := state.processTime.Seconds()
			if withStateRoot {
				seconds = blockTime.Seconds()
			}
			b.ReportMetric(float64(transfers)/seconds, "transfers/s")
			b.ReportMetric(float64(transfers*load.TxsPerTransfer)/seconds, "tx/s")
		})
	}
}

// BenchmarkThroughput_Network loads a single node network targeting about
// 3 blocks per second with transfers, each sender at most networkRoundsInFlight
// transfers ahead of the chain, and reports the rate of the included
// transactions over a window of blocks.
func BenchmarkThroughput_Network(b *testing.B) {
	for _, load := range throughput.Loads {
		b.Run(load.Name, func(b *testing.B) {
			issuer := throughput.NewIssuer(big.NewInt(int64(opera.FakeNetworkID)), throughput.NewKeys())
			net := StartIntegrationTestNetWithJsonGenesis(b, IntegrationTestNetOptions{
				Upgrades:         AsPointer(opera.GetDolceUpgrades()),
				Accounts:         fundedAccounts(issuer.Senders),
				LiveCacheSize:    AsPointer(1 << 30),
				ArchiveCacheSize: AsPointer(1 << 30),
				ModifyRules: func(rules *opera.Rules) {
					rules.Emitter.Interval = inter.Timestamp(networkEmitterInterval)
					rules.Economy.Gas.MaxEventGas = 2_000_000_000
					rules.Economy.ShortGasPower.AllocPerSec = 10_000_000_000
					rules.Economy.LongGasPower.AllocPerSec = 10_000_000_000
					rules.Epochs.MaxEpochGas = 1 << 62
				},
				ModifyConfig: func(cfg *config.Config) {
					cfg.TxPool.GlobalSlots = 1 << 17
					cfg.TxPool.GlobalQueue = 1 << 17
				},
			})

			client, err := net.GetClient()
			require.NoError(b, err)
			defer client.Close()

			if setup := issuer.Setup(b, load); len(setup) > 0 {
				for _, tx := range setup {
					require.NoError(b, client.SendTransaction(b.Context(), tx))
				}
				for _, tx := range setup {
					receipt, err := net.GetReceipt(tx.Hash())
					require.NoError(b, err)
					require.Equal(b, types.ReceiptStatusSuccessful, receipt.Status)
				}
			}

			// Each job sends the transfers of its senders round by round, so
			// that nonces arrive in order, until the window is over. The pool
			// accepts all local transactions, so the jobs keep at most
			// networkRoundsInFlight rounds ahead of the chain.
			ctx, cancel := context.WithCancel(b.Context())
			var wg sync.WaitGroup
			for job := range networkSendJobs {
				wg.Go(func() {
					client, err := net.GetClient()
					if err != nil {
						b.Error(err)
						return
					}
					defer client.Close()
					for ctx.Err() == nil && !b.Failed() {
						nonce, err := client.NonceAt(ctx, issuer.Senders[job], nil)
						if err != nil {
							if ctx.Err() == nil {
								b.Error(err)
							}
							return
						}
						if nonce+uint64(networkRoundsInFlight*load.TxsPerTransfer) < issuer.Nonce(job) {
							select {
							case <-ctx.Done():
							case <-time.After(10 * time.Millisecond):
							}
							continue
						}
						var txs []*types.Transaction
						for sender := job; sender < len(issuer.Senders); sender += networkSendJobs {
							txs = append(txs, issuer.Transfers(b, load, sender)...)
						}
						sendInOrder(ctx, b, client.Client(), txs)
					}
				})
			}

			time.Sleep(networkWarmup)
			first := blockAt(b, client.Client(), "latest")
			time.Sleep(networkWindow)
			last := blockAt(b, client.Client(), "latest")
			cancel()
			wg.Wait()
			require.Greater(b, last.Number, first.Number)

			txs := 0
			for number := int64(first.Number) + 1; number <= int64(last.Number); number++ {
				receipts, err := client.BlockReceipts(b.Context(), rpc.BlockNumberOrHashWithNumber(rpc.BlockNumber(number)))
				require.NoError(b, err)
				for _, receipt := range receipts {
					require.Equal(b, types.ReceiptStatusSuccessful, receipt.Status)
				}
				txs += len(receipts)
			}
			blocks := float64(last.Number - first.Number)
			seconds := time.Duration(last.TimestampNano - first.TimestampNano).Seconds()
			b.ReportMetric(float64(txs)/float64(load.TxsPerTransfer)/seconds, "transfers/s")
			b.ReportMetric(float64(txs)/seconds, "tx/s")
			b.ReportMetric(blocks/seconds, "blocks/s")
			b.ReportMetric(float64(txs)/blocks, "tx/block")
		})
	}
}

// sendInOrder sends the transactions in batches until all are sent or the
// context is done, and fails the benchmark on any rejected transaction.
func sendInOrder(ctx context.Context, b *testing.B, client *rpc.Client, txs []*types.Transaction) {
	for len(txs) > 0 {
		batch := make([]rpc.BatchElem, min(100, len(txs)))
		for i := range batch {
			raw, err := txs[i].MarshalBinary()
			if err != nil {
				panic(err)
			}
			batch[i] = rpc.BatchElem{Method: "eth_sendRawTransaction", Args: []any{hexutil.Encode(raw)}, Result: new(common.Hash)}
		}
		if err := client.BatchCallContext(ctx, batch); err != nil {
			if ctx.Err() == nil {
				b.Error(err)
			}
			return
		}
		for _, elem := range batch {
			if elem.Error != nil {
				b.Error(elem.Error)
				return
			}
		}
		txs = txs[len(batch):]
	}
}

type blockSummary struct {
	Number        hexutil.Uint64
	TimestampNano hexutil.Uint64
}

func blockAt(b *testing.B, client *rpc.Client, number string) blockSummary {
	var block blockSummary
	require.NoError(b, client.Call(&block, "eth_getBlockByNumber", number, false))
	return block
}
