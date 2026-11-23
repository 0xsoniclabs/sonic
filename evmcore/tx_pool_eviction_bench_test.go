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

package evmcore_test

import (
	"crypto/ecdsa"
	"encoding/binary"
	"fmt"
	"math/big"
	"testing"

	carmen "github.com/0xsoniclabs/carmen/go/state"
	"github.com/0xsoniclabs/carmen/go/state/gostate"
	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	freezeregistry "github.com/0xsoniclabs/sonic/gossip/blockproc/freeze/registry"
	subsidiesregistry "github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies/registry"
	"github.com/0xsoniclabs/sonic/gossip/evmstore"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// BenchmarkTxPool_PerBlockEviction compares the per-block scan evicting
// transactions of frozen senders with the per-block demotion, which checks
// sponsorships and bundles, on a real in-memory Carmen state. Each pool holds
// one pending transaction per sender. No sender is frozen, every sponsorship
// is covered and every bundle is executable, so nothing is evicted and every
// iteration does the same work. Cold runs drop the check caches before each
// iteration: bundle evaluations are cached per block, so they are always cold
// on a new block; sponsorship checks are cached up to 15s.
func BenchmarkTxPool_PerBlockEviction(b *testing.B) {
	log.SetDefault(log.NewLogger(log.DiscardHandler()))
	for _, senders := range []int{1_000, 10_000} {
		env := newEvictionBenchEnv(b, senders)
		for _, c := range []struct {
			name string
			txs  func(*ecdsa.PrivateKey) *types.Transaction
			cold bool
			op   func(*evmcore.TxPool)
		}{
			{"dropFrozenSenders", env.plainTx, false, (*evmcore.TxPool).DropFrozenSenders_forTesting},
			{"evictBundleOnly", env.plainTx, false, (*evmcore.TxPool).EvictBundleOnlyTransactionsOfProcessedBundles_forTesting},
			{"demote/plain", env.plainTx, false, (*evmcore.TxPool).DemoteUnexecutables_forTesting},
			{"demote/sponsored/cached", env.sponsoredTx, false, (*evmcore.TxPool).DemoteUnexecutables_forTesting},
			{"demote/sponsored/cold", env.sponsoredTx, true, (*evmcore.TxPool).DemoteUnexecutables_forTesting},
			{"demote/bundles/cold", env.bundleTx, true, (*evmcore.TxPool).DemoteUnexecutables_forTesting},
		} {
			b.Run(fmt.Sprintf("senders=%d/%s", senders, c.name), func(b *testing.B) {
				pool := env.newPool(b, c.txs)
				b.ReportAllocs()
				b.ResetTimer()
				for range b.N {
					if c.cold {
						b.StopTimer()
						pool.ClearCheckCaches_forTesting()
						b.StartTimer()
					}
					c.op(pool)
				}
				b.StopTimer()
				pending, _ := pool.Stats()
				require.Equal(b, senders, pending)
			})
		}
	}
}

type evictionBenchEnv struct {
	chain  *evictionBenchChain
	signer types.Signer
	keys   []*ecdsa.PrivateKey
}

func newEvictionBenchEnv(b *testing.B, senders int) *evictionBenchEnv {
	st, err := carmen.NewState(carmen.Parameters{
		Variant:   gostate.VariantGoMemory,
		Schema:    carmen.Schema(5),
		Archive:   carmen.NoArchive,
		Directory: b.TempDir(),
	})
	require.NoError(b, err)
	b.Cleanup(func() { _ = st.Close() })
	db := evmstore.CreateCarmenStateDb(carmen.CreateStateDBUsing(st), nil)

	upgrades := opera.GetBrioUpgrades()
	upgrades.GasSubsidies = true
	upgrades.TransactionBundles = true
	upgrades.AccountFreeze = true
	rules := opera.FakeNetRules(upgrades)
	config := opera.CreateTransientEvmChainConfig(rules.NetworkID, []opera.UpgradeHeight{opera.MakeUpgradeHeight(upgrades, 0)}, 1)
	chain := &evictionBenchChain{
		header: evmcore.EvmHeader{Number: big.NewInt(1), GasLimit: 1_000_000_000, BaseFee: big.NewInt(1), PrevRandao: common.Hash{1}},
		db:     db,
		config: config,
		rules:  rules,
	}

	db.BeginBlock(1)
	balance := uint256.NewInt(1e18)
	keys := make([]*ecdsa.PrivateKey, senders)
	for i := range keys {
		keys[i], err = crypto.ToECDSA(crypto.Keccak256(binary.BigEndian.AppendUint64(nil, uint64(i)+1)))
		require.NoError(b, err)
		db.SetBalance(crypto.PubkeyToAddress(keys[i].PublicKey), balance)
	}

	// Frozen accounts other than the senders give the registry some storage.
	db.CreateAccount(freezeregistry.GetAddress())
	db.SetCode(freezeregistry.GetAddress(), freezeregistry.GetCode(), tracing.CodeChangeUnspecified)
	for i := range 100 {
		db.SetState(freezeregistry.GetAddress(), freezeregistry.GetFrozenSlot(common.Address{0xf, byte(i)}), common.Hash{31: 1})
	}

	// A global sponsorship fund covers every sponsorship request, after all
	// more specific funds have been checked.
	db.CreateAccount(subsidiesregistry.GetAddress())
	db.SetCode(subsidiesregistry.GetAddress(), subsidiesregistry.GetCode(), tracing.CodeChangeUnspecified)
	evm := vm.NewEVM(evmcore.NewEVMBlockContext(&chain.header, chain, nil), db, config, opera.GetVmConfig(rules))
	sponsor := crypto.PubkeyToAddress(keys[0].PublicKey)
	input := append(crypto.Keccak256([]byte("sponsor(bytes32)"))[:4], crypto.Keccak256([]byte("g"))...)
	_, _, err = evm.Call(sponsor, subsidiesregistry.GetAddress(), input, vm.NewGasBudget(1_000_000, 0), uint256.NewInt(1e17))
	require.NoError(b, err)

	require.NoError(b, evmstore.EndBlockAndCommit(db, 1))
	db.BeginBlock(2)

	return &evictionBenchEnv{
		chain:  chain,
		signer: types.LatestSignerForChainID(config.ChainID),
		keys:   keys,
	}
}

// newPool returns a stopped pool holding one pending transaction per sender.
func (e *evictionBenchEnv) newPool(b *testing.B, makeTx func(*ecdsa.PrivateKey) *types.Transaction) *evmcore.TxPool {
	config := evmcore.DefaultTxPoolConfig
	config.Journal = ""
	config.NoLocals = true
	config.GlobalSlots = uint64(2 * len(e.keys))
	pool := evmcore.NewTxPool(config, e.chain.config, e.chain, evmcore.NewBundleEvaluationCache())

	txs := make([]*types.Transaction, len(e.keys))
	for i, key := range e.keys {
		txs[i] = makeTx(key)
	}
	for _, err := range pool.AddRemotesSync(txs) {
		require.NoError(b, err)
	}
	pool.Stop()
	pending, queued := pool.Stats()
	require.Equal(b, len(e.keys), pending)
	require.Zero(b, queued)
	return pool
}

var evictionBenchRecipient = common.Address{0xaa}

func (e *evictionBenchEnv) plainTx(key *ecdsa.PrivateKey) *types.Transaction {
	return types.MustSignNewTx(key, e.signer, &types.LegacyTx{
		To:       &evictionBenchRecipient,
		Gas:      21_000,
		GasPrice: big.NewInt(10),
	})
}

func (e *evictionBenchEnv) sponsoredTx(key *ecdsa.PrivateKey) *types.Transaction {
	return types.MustSignNewTx(key, e.signer, &types.LegacyTx{
		To:  &evictionBenchRecipient,
		Gas: 21_000,
	})
}

func (e *evictionBenchEnv) bundleTx(key *ecdsa.PrivateKey) *types.Transaction {
	return bundle.NewBuilder().
		WithSigner(e.signer).
		SetEnvelopeSenderKey(key).
		SetEnvelopeGasPrice(big.NewInt(10)).
		AllOf(bundle.Step(key, &types.AccessListTx{
			To:       &evictionBenchRecipient,
			Gas:      21_000,
			GasPrice: big.NewInt(10),
		})).
		Build()
}

// evictionBenchChain is a chain whose head stays at a fixed block.
type evictionBenchChain struct {
	header evmcore.EvmHeader
	db     state.StateDB
	config *params.ChainConfig
	rules  opera.Rules
	feed   event.Feed
}

func (c *evictionBenchChain) CurrentBlock() *evmcore.EvmBlock {
	return &evmcore.EvmBlock{EvmHeader: c.header}
}
func (c *evictionBenchChain) Block(common.Hash, uint64) *evmcore.EvmBlock { return c.CurrentBlock() }
func (c *evictionBenchChain) Header(common.Hash, uint64) *evmcore.EvmHeader {
	return &c.header
}
func (c *evictionBenchChain) CurrentStateDB() (state.StateDB, error) { return c.db, nil }
func (c *evictionBenchChain) CurrentBaseFee() *big.Int               { return c.header.BaseFee }
func (c *evictionBenchChain) CurrentMaxGasLimit() uint64             { return c.header.GasLimit }
func (c *evictionBenchChain) CurrentConfig() *params.ChainConfig     { return c.config }
func (c *evictionBenchChain) CurrentRules() opera.Rules              { return c.rules }
func (c *evictionBenchChain) SubscribeNewBlock(ch chan<- evmcore.ChainHeadNotify) event.Subscription {
	return c.feed.Subscribe(ch)
}
