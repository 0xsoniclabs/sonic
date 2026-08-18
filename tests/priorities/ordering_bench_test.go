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
	"crypto/ecdsa"
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"testing"

	carmen "github.com/0xsoniclabs/carmen/go/state"
	"github.com/0xsoniclabs/carmen/go/state/gostate"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/priorities"
	priorityregistry "github.com/0xsoniclabs/sonic/gossip/blockproc/priorities/registry"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/proxy"
	"github.com/0xsoniclabs/sonic/gossip/evmstore"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// These benchmarks compare how long the two candidate priority classifiers take:
//   - EvmClassifier, the implemented one, runs one getPriority EVM call per
//     transaction, each isolated by a state snapshot/revert.
//   - nativeClassifier models the alternative of fetching the criteria once per
//     block and classifying in pure Go with one map lookup per transaction. Only
//     the per-transaction lookup is measured; the once-per-block fetch is not,
//     since it is amortized across all transactions of the block. Fetching it in
//     bulk would need a different registry storage layout, so it is not modeled
//     here.
//
// The queries run against a real in-memory Carmen state pre-populated with
// benchStateAccounts dummy accounts rather than against a mock, so a lookup
// traverses an account trie of some depth. The numbers are still a lower bound:
// production state is far larger and disk-backed, and the registry deployed here
// is the stand-in contract, whose getPriority is a single mapping read that
// ignores the calldata.

const (
	// benchStateAccounts is the number of dummy accounts pre-populated into the
	// benchmark state so registry lookups traverse a non-trivial account trie.
	// Raise this to stress deeper tries.
	benchStateAccounts = 10_000

	// benchMaxBlockTxs is the largest realistic number of transactions in a
	// block; scenarios beyond this are not exercised.
	benchMaxBlockTxs = 10_000

	// benchPrioritizedSenders is how many of the dummy accounts are configured
	// with a non-zero priority in the registry.
	benchPrioritizedSenders = 256
)

func BenchmarkPrioritize(b *testing.B) {
	env := setupBenchEnv(b, benchStateAccounts, benchPrioritizedSenders)
	var failures countingMeter
	evmClassifier := priorities.NewEvmClassifier(env.upgrades, env.evm, env.signer, env.statedb, &failures)
	nativeClassifier := env.nativeClassifier()
	b.Cleanup(func() { require.Zero(b, int64(failures)) })

	requireBothClassifiersReorder(b, env, evmClassifier, nativeClassifier)

	// Default arm vs. native arm across realistic block sizes, using a realistic
	// mix (~10% prioritized) of empty-calldata transfers.
	for _, n := range []int{10, 100, 1000, benchMaxBlockTxs} {
		txs := env.makeTxs(b, n, 10, 0)
		b.Run(fmt.Sprintf("EvmClassifier/n=%d", n), func(b *testing.B) {
			runPrioritize(b, env, txs, evmClassifier)
		})
		b.Run(fmt.Sprintf("Native/n=%d", n), func(b *testing.B) {
			runPrioritize(b, env, txs, nativeClassifier)
		})
	}

	// Result-mix sensitivity at a full block: the EVM query cost is paid for
	// every transaction regardless of outcome, so this mainly moves the (cheap)
	// ordering passes.
	for _, m := range []struct {
		name      string
		oneInEach int // 1 => all prioritized; 0 => none; k => every k-th
	}{
		{"all-normal", 0},
		{"mixed-10pct", 10},
		{"all-prioritized", 1},
	} {
		txs := env.makeTxs(b, benchMaxBlockTxs, m.oneInEach, 0)
		b.Run(fmt.Sprintf("Mix/%s", m.name), func(b *testing.B) {
			runPrioritize(b, env, txs, evmClassifier)
		})
	}

	// Calldata-size sensitivity at a full block: calldata is ABI-encoded into
	// the getPriority input, so larger payloads cost more per query.
	for _, dataLen := range []int{0, 1024} {
		txs := env.makeTxs(b, benchMaxBlockTxs, 10, dataLen)
		b.Run(fmt.Sprintf("Calldata/bytes=%d", dataLen), func(b *testing.B) {
			runPrioritize(b, env, txs, evmClassifier)
		})
	}
}

func runPrioritize(b *testing.B, env *benchEnv, txs types.Transactions, classifier priorities.Classifier) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		got := priorities.Prioritize(txs, classifier, env.signer, env.statedb, env.cfg)
		if len(got) != len(txs) {
			b.Fatalf("Prioritize returned %d txs, want %d", len(got), len(txs))
		}
	}
}

// requireBothClassifiersReorder ensures that with both classifiers, some
// transactions get classified as prioritized and are reordered.
func requireBothClassifiersReorder(b *testing.B, env *benchEnv, evm, native priorities.Classifier) {
	txs := env.makeTxs(b, 100, 10, 0)
	base := hashesOf(txs)
	evmOrder := hashesOf(priorities.Prioritize(txs, evm, env.signer, env.statedb, env.cfg))
	nativeOrder := hashesOf(priorities.Prioritize(txs, native, env.signer, env.statedb, env.cfg))
	require.NotEqual(b, base, evmOrder)
	require.NotEqual(b, base, nativeOrder)
	require.Equal(b, evmOrder, nativeOrder)
}

// hashesOf returns the transaction hashes of the transactions.
func hashesOf(txs types.Transactions) []common.Hash {
	hashes := make([]common.Hash, len(txs))
	for i, tx := range txs {
		hashes[i] = tx.Hash()
	}
	return hashes
}

// benchEnv is a fully populated in-memory benchmark environment.
type benchEnv struct {
	statedb  *evmstore.CarmenStateDB
	evm      *vm.EVM
	signer   types.Signer
	upgrades opera.Upgrades
	cfg      priorities.Config

	keys           []*ecdsa.PrivateKey
	prioByAddr     map[common.Address]priorities.Priority
	numPrioritized int
}

// nativeClassifier returns a Classifier that resolves priorities from an
// in-memory map (the criteria fetched once per block), modeling the native
// classifier alternative.
func (e *benchEnv) nativeClassifier() priorities.Classifier {
	return &nativeClassifier{signer: e.signer, byAddr: e.prioByAddr}
}

type nativeClassifier struct {
	signer types.Signer
	byAddr map[common.Address]priorities.Priority
}

func (c *nativeClassifier) Priority(tx *types.Transaction) (priorities.Priority, error) {
	sender, err := types.Sender(c.signer, tx)
	if err != nil {
		return priorities.Priority{}, err
	}
	if p, ok := c.byAddr[sender]; ok {
		return p, nil
	}
	return priorities.Priority{}, nil
}

// makeTxs builds n signed transactions. Every oneInEach-th transaction is sent
// from a prioritized account (oneInEach == 0 disables prioritized senders,
// oneInEach == 1 makes all of them prioritized). Each carries dataLen bytes of
// calldata. Nonces are assigned per sender, counting up from the block-start
// account nonce of zero, so that prioritized senders form the contiguous nonce
// sequences Prioritize requires to hoist them.
func (e *benchEnv) makeTxs(b *testing.B, n, oneInEach, dataLen int) types.Transactions {
	to := common.Address{0xaa}
	data := make([]byte, dataLen)
	gas := 21000 + uint64(dataLen)*params.TxDataZeroGas
	normalStart := e.numPrioritized
	normalCount := len(e.keys) - normalStart
	require.Positive(b, normalCount)

	nonces := make([]uint64, len(e.keys))
	txs := make(types.Transactions, n)
	for i := 0; i < n; i++ {
		k := normalStart + i%normalCount
		if oneInEach > 0 && i%oneInEach == 0 && e.numPrioritized > 0 {
			k = i % e.numPrioritized
		}
		txs[i] = types.MustSignNewTx(e.keys[k], e.signer, &types.LegacyTx{
			Nonce:    nonces[k],
			To:       &to,
			Gas:      gas,
			GasPrice: big.NewInt(1),
			Data:     data,
		})
		nonces[k]++
	}

	// Resolve senders and hashes up front. Both are cached on the transaction and
	// are already resolved by the time Prioritize runs in production, so only the
	// first measured iteration would otherwise pay for them.
	for _, tx := range txs {
		_, err := types.Sender(e.signer, tx)
		require.NoError(b, err)
		tx.Hash()
	}
	return txs
}

func setupBenchEnv(b *testing.B, numAccounts, numPrioritized int) *benchEnv {
	require := require.New(b)
	require.LessOrEqual(numPrioritized, numAccounts)

	st, err := carmen.NewState(carmen.Parameters{
		Variant:   gostate.VariantGoMemory,
		Schema:    carmen.Schema(5),
		Archive:   carmen.NoArchive,
		Directory: b.TempDir(),
	})
	require.NoError(err)
	b.Cleanup(func() { _ = st.Close() })

	statedb := evmstore.CreateCarmenStateDb(carmen.CreateStateDBUsing(st), nil)

	upgrades := opera.GetBrioUpgrades()
	upgrades.TransactionPriorities = true
	rules := opera.FakeNetRules(upgrades)
	chainConfig := opera.CreateTransientEvmChainConfig(rules.NetworkID, nil, 1)
	signer := types.LatestSigner(chainConfig)

	blockContext := vm.BlockContext{
		CanTransfer: func(vm.StateDB, common.Address, *uint256.Int) bool { return true },
		Transfer:    func(vm.StateDB, common.Address, common.Address, *uint256.Int, *params.Rules) {},
		GetHash:     func(uint64) common.Hash { return common.Hash{} },
		BlockNumber: big.NewInt(2),
		Time:        1,
		GasLimit:    1_000_000_000,
		BaseFee:     big.NewInt(1),
		Random:      &common.Hash{},
	}
	evm := vm.NewEVM(blockContext, statedb, chainConfig, opera.GetVmConfig(rules))

	// --- Block 1: deploy the registry and populate state. ---
	statedb.BeginBlock(1)

	// Deploy the priority registry exactly as genesis does: an EIP-1967 proxy at
	// the fixed address delegating to the implementation.
	implAddr := common.Address{1, 2, 3, 4, 5, 6, 8}
	implSlotValue := common.Hash{}
	copy(implSlotValue[12:], implAddr[:])

	statedb.CreateAccount(priorityregistry.GetAddress())
	statedb.SetCode(priorityregistry.GetAddress(), proxy.GetCode(), tracing.CodeChangeUnspecified)
	statedb.SetNonce(priorityregistry.GetAddress(), 1, tracing.NonceChangeUnspecified)
	statedb.SetState(priorityregistry.GetAddress(), proxy.GetSlotForImplementation(), implSlotValue)

	statedb.CreateAccount(implAddr)
	statedb.SetCode(implAddr, priorityregistry.GetCode(), tracing.CodeChangeUnspecified)
	statedb.SetNonce(implAddr, 1, tracing.NonceChangeUnspecified)

	// Pre-populate dummy accounts to deepen the account trie. The first
	// numPrioritized of them become prioritized senders. Their private keys are
	// derived deterministically so the benchmark is reproducible.
	balance := uint256.NewInt(1_000_000_000_000_000_000)
	keys := make([]*ecdsa.PrivateKey, numAccounts)
	for i := 0; i < numAccounts; i++ {
		key, err := crypto.ToECDSA(crypto.Keccak256(binary.BigEndian.AppendUint64(nil, uint64(i)+1)))
		require.NoError(err)
		keys[i] = key
		addr := crypto.PubkeyToAddress(key.PublicKey)
		statedb.CreateAccount(addr)
		statedb.SetBalance(addr, balance)
	}

	regABI, err := priorityregistry.RegistryMetaData.GetAbi()
	require.NoError(err)

	callRegistry := func(method string, args ...any) {
		input, err := regABI.Pack(method, args...)
		require.NoError(err)
		_, _, err = evm.Call(common.Address{}, priorityregistry.GetAddress(), input, 5_000_000, uint256.NewInt(0))
		require.NoError(err)
	}

	// Effectively unlimited per-entity limits so rate limiting does not distort
	// the timing.
	maxLimit := new(big.Int).SetUint64(math.MaxUint64)
	callRegistry("setConfig", maxLimit, maxLimit)

	prioByAddr := make(map[common.Address]priorities.Priority, numPrioritized)
	for i := 0; i < numPrioritized; i++ {
		addr := crypto.PubkeyToAddress(keys[i].PublicKey)
		level := uint64(1 + i%4)    // a few distinct levels
		weight := uint64(1 + i%100) // spread of weights
		var id priorities.PriorityID
		binary.BigEndian.PutUint64(id[8:], uint64(i)) // distinct entity per sender
		callRegistry("setSenderPriority", addr, level, weight, new(big.Int).SetBytes(id[:]))

		prioByAddr[addr] = priorities.Priority{Level: level, Weight: weight, ID: id}
	}

	if ch := statedb.EndBlock(1); ch != nil {
		require.NoError(<-ch)
	}

	// --- Block 2: serve the priority queries. ---
	statedb.BeginBlock(2)
	b.Cleanup(func() {
		if ch := statedb.EndBlock(2); ch != nil {
			require.NoError(<-ch)
		}
	})

	snapshot := statedb.InterTxSnapshot()
	cfg, err := priorities.GetConfig(upgrades, evm)
	statedb.RevertToInterTxSnapshot(snapshot)
	require.NoError(err)

	return &benchEnv{
		statedb:        statedb,
		evm:            evm,
		signer:         signer,
		upgrades:       upgrades,
		cfg:            cfg,
		keys:           keys,
		prioByAddr:     prioByAddr,
		numPrioritized: numPrioritized,
	}
}

// countingMeter accumulates the registry-query failures, which must stay zero.
type countingMeter int64

func (c *countingMeter) Mark(n int64) { *c += countingMeter(n) }
