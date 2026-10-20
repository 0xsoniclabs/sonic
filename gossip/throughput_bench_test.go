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

package gossip

import (
	"math/big"
	"testing"
	"time"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/lachesis"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/integration/makefakegenesis"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests/throughput"
)

// BenchmarkThroughput_Blocks runs blocks of one transfer per sender through the
// node's block processing, from the confirmed event to the stored block: block
// formation, the EVM, the state commit with its state root, and the storage of
// the block and its receipts. Each block is a single event handed to the
// consensus callbacks, so the emitter, consensus, tx pool and RPC are left out.
//
//	go test ./gossip -run '^$' -bench '^BenchmarkThroughput_Blocks$' -benchtime 20x
func BenchmarkThroughput_Blocks(b *testing.B) {
	for _, load := range throughput.Loads {
		b.Run(load.Name, func(b *testing.B) {
			cfg := MemTestStoreConfig(b.TempDir())
			cfg.EVM.StateDb.LiveCache = 1 << 30
			cfg.EVM.StateDb.ArchiveCache = 1 << 30
			// An event evicted from the cache is decoded again on its next
			// read, and its transactions lose their recovered senders.
			cfg.Cache.EventsSize = 1 << 30
			env := newTestEnvWithStore(newInMemoryStoreWithGenesisDataAndConfig(b, opera.GetDolceUpgrades(), 1, 1, cfg), b)
			b.Cleanup(func() { require.NoError(b, env.Close()) })
			store := env.store
			issuer := throughput.NewIssuer(new(big.Int).SetUint64(store.GetRules().NetworkID), throughput.NewKeys())
			validator := store.GetValidators().GetID(0)

			var lamport idx.Lamport
			apply := func(txs types.Transactions) time.Duration {
				for _, tx := range txs { // < nodes recover senders before processing
					_, err := types.Sender(issuer.Signer, tx)
					require.NoError(b, err)
				}
				lamport++
				builder := &inter.MutableEventPayload{}
				builder.SetVersion(2)
				builder.SetEpoch(store.GetEpoch())
				builder.SetSeq(idx.Event(lamport))
				builder.SetLamport(lamport)
				builder.SetCreator(validator)
				builder.SetMedianTime(store.GetBlockState().LastBlock.Time + inter.Timestamp(time.Second))
				builder.SetTxs(txs)
				event := builder.Build()
				store.SetEvent(event)

				start := time.Now()
				callbacks := env.GetConsensusCallbacks().BeginBlock(&lachesis.Block{Atropos: event.ID()})
				callbacks.ApplyEvent(event)
				callbacks.EndBlock()
				env.WaitBlockEnd()
				elapsed := time.Since(start)

				receipts, _ := store.evm.GetRawReceipts(store.GetLatestBlockIndex())
				require.GreaterOrEqual(b, len(receipts), len(txs))
				for _, receipt := range receipts {
					require.Equal(b, types.ReceiptStatusSuccessful, receipt.Status)
				}
				return elapsed
			}

			// The validator funds the senders.
			key := makefakegenesis.FakeKey(validator)
			nonce := env.State().GetNonce(crypto.PubkeyToAddress(key.PublicKey))
			balance := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1e9))
			funding := make(types.Transactions, len(issuer.Senders))
			for i, sender := range issuer.Senders {
				funding[i] = types.MustSignNewTx(key, issuer.Signer, &types.LegacyTx{
					Nonce: nonce + uint64(i), To: &sender, Value: balance, Gas: 21_000, GasPrice: big.NewInt(throughput.GasPrice),
				})
			}
			apply(funding)
			if txs := issuer.Setup(b, load); len(txs) > 0 {
				apply(txs)
			}

			var elapsed time.Duration
			transfers := 0
			for b.Loop() {
				var txs types.Transactions
				for sender := range issuer.Senders {
					txs = append(txs, issuer.Transfers(b, load, sender)...)
				}
				elapsed += apply(txs)
				transfers += len(issuer.Senders)
			}

			state := env.State()
			for i, sender := range issuer.Senders {
				require.Equal(b, issuer.Nonce(i), state.GetNonce(sender))
			}

			seconds := elapsed.Seconds()
			b.ReportMetric(float64(transfers)/seconds, "transfers/s")
			b.ReportMetric(float64(transfers*load.TxsPerTransfer)/seconds, "tx/s")
		})
	}
}
