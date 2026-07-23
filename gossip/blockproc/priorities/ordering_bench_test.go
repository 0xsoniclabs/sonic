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
	"math/big"
	"testing"

	carmen "github.com/0xsoniclabs/carmen/go/state"
	"github.com/0xsoniclabs/carmen/go/state/gostate"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/priorities/registry"
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

// These benchmarks answer the query-strategy gate (design doc §1, "Critical-path
// query cost"): is the default per-transaction EvmClassifier — one getPriority
// EVM call per transaction, each isolated by a state snapshot/revert — fast
// enough at realistic block sizes, or must we fall back to fetching the
// classification criteria once per block and applying them natively in Go?
//
// To keep the registry lookups representative they run against a real in-memory
// Carmen state pre-populated with benchStateAccounts dummy accounts so the
// account trie has a realistic depth. The native classifier (Native/* sub
// benchmarks) is the comparison arm: it models the fallback's lower bound (one
// criteria fetch amortized to ~0, then a pure-Go map lookup per transaction).
const (
	// benchStateAccounts is the number of dummy accounts pre-populated into the
	// benchmark state so registry lookups traverse a realistically deep trie.
	// Raise this to stress deeper tries.
	benchStateAccounts = 10_000

	// benchMaxBlockTxs is the largest realistic number of transactions in a
	// block; scenarios beyond this are not exercised.
	benchMaxBlockTxs = 10_000

	// benchPrioritizedSenders is how many of the dummy accounts are configured
	// with a non-zero priority in the registry.
	benchPrioritizedSenders = 256
)

// benchEnv is a fully populated in-memory benchmark environment.
type benchEnv struct {
	statedb  *evmstore.CarmenStateDB
	evm      *vm.EVM
	signer   types.Signer
	upgrades opera.Upgrades
	cfg      Config

	keys           []*ecdsa.PrivateKey
	prioByAddr     map[common.Address]Priority
	numPrioritized int
}

func BenchmarkPrioritize(b *testing.B) {
	env := setupBenchEnv(b, benchStateAccounts, benchPrioritizedSenders)
	evmClassifier := NewEvmClassifier(env.upgrades, env.evm, env.signer, env.statedb)
	nativeClassifier := env.nativeClassifier()

	// Default arm vs. native fallback arm across realistic block sizes, using a
	// realistic mix (~10% prioritized) of empty-calldata transfers.
	for _, n := range []int{10, 100, 1000, benchMaxBlockTxs} {
		txs := env.makeTxs(b, n, 10, 0)
		b.Run(fmt.Sprintf("EvmClassifier/n=%d", n), func(b *testing.B) {
			runPrioritize(b, txs, evmClassifier, env.signer, env.statedb, env.cfg)
		})
		b.Run(fmt.Sprintf("Native/n=%d", n), func(b *testing.B) {
			runPrioritize(b, txs, nativeClassifier, env.signer, env.statedb, env.cfg)
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
			runPrioritize(b, txs, evmClassifier, env.signer, env.statedb, env.cfg)
		})
	}

	// Calldata-size sensitivity at a full block: calldata is ABI-encoded into
	// the getPriority input, so larger payloads cost more per query.
	for _, dataLen := range []int{0, 1024} {
		txs := env.makeTxs(b, benchMaxBlockTxs, 10, dataLen)
		b.Run(fmt.Sprintf("Calldata/bytes=%d", dataLen), func(b *testing.B) {
			runPrioritize(b, txs, evmClassifier, env.signer, env.statedb, env.cfg)
		})
	}
}

func runPrioritize(b *testing.B, txs types.Transactions, classifier Classifier, signer types.Signer, state NonceReader, cfg Config) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if got := Prioritize(txs, classifier, signer, state, cfg); len(got) != len(txs) {
			b.Fatalf("Prioritize returned %d txs, want %d", len(got), len(txs))
		}
	}
}

// nativeClassifier returns a Classifier that resolves priorities from an
// in-memory map (the criteria fetched once per block), modeling the
// native-filter fallback.
func (e *benchEnv) nativeClassifier() Classifier {
	return &mapClassifier{signer: e.signer, byAddr: e.prioByAddr}
}

type mapClassifier struct {
	signer types.Signer
	byAddr map[common.Address]Priority
}

func (c *mapClassifier) Priority(tx *types.Transaction) (Priority, error) {
	sender, err := types.Sender(c.signer, tx)
	if err != nil {
		return Priority{}, err
	}
	if p, ok := c.byAddr[sender]; ok {
		return p, nil
	}
	return Priority{}, nil
}

// makeTxs builds n signed transactions. Every oneInEach-th transaction is sent
// from a prioritized account (oneInEach == 0 disables prioritized senders,
// oneInEach == 1 makes all of them prioritized). Each carries dataLen bytes of
// calldata.
func (e *benchEnv) makeTxs(b *testing.B, n, oneInEach, dataLen int) types.Transactions {
	to := common.Address{0xaa}
	data := make([]byte, dataLen)
	gas := uint64(21000)
	if dataLen > 0 {
		gas += uint64(dataLen) * 16
	}
	normalStart := e.numPrioritized
	normalCount := len(e.keys) - normalStart
	require.Positive(b, normalCount, "need at least one non-prioritized account")

	txs := make(types.Transactions, n)
	for i := 0; i < n; i++ {
		var key *ecdsa.PrivateKey
		if oneInEach > 0 && i%oneInEach == 0 && e.numPrioritized > 0 {
			key = e.keys[i%e.numPrioritized]
		} else {
			key = e.keys[normalStart+(i%normalCount)]
		}
		txs[i] = types.MustSignNewTx(key, e.signer, &types.LegacyTx{
			Nonce:    uint64(i),
			To:       &to,
			Gas:      gas,
			GasPrice: big.NewInt(1),
			Data:     data,
		})
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

	upgrades := enabledUpgrades()
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

	statedb.CreateAccount(registry.GetAddress())
	statedb.SetCode(registry.GetAddress(), proxy.GetCode(), tracing.CodeChangeUnspecified)
	statedb.SetNonce(registry.GetAddress(), 1, tracing.NonceChangeUnspecified)
	statedb.SetState(registry.GetAddress(), proxy.GetSlotForImplementation(), implSlotValue)

	statedb.CreateAccount(implAddr)
	statedb.SetCode(implAddr, registry.GetCode(), tracing.CodeChangeUnspecified)
	statedb.SetNonce(implAddr, 1, tracing.NonceChangeUnspecified)

	// Pre-populate dummy accounts to deepen the account trie. The first
	// numPrioritized of them become prioritized senders.
	balance := uint256.NewInt(1_000_000_000_000_000_000)
	keys := make([]*ecdsa.PrivateKey, numAccounts)
	for i := 0; i < numAccounts; i++ {
		keys[i] = benchKey(i)
		addr := crypto.PubkeyToAddress(keys[i].PublicKey)
		statedb.CreateAccount(addr)
		statedb.SetBalance(addr, balance)
	}

	regABI, err := registry.RegistryMetaData.GetAbi()
	require.NoError(err)

	callRegistry := func(method string, args ...any) {
		input, err := regABI.Pack(method, args...)
		require.NoError(err)
		_, _, err = evm.Call(common.Address{}, registry.GetAddress(), input, 5_000_000, uint256.NewInt(0))
		require.NoError(err)
	}

	// Generous per-entity limits so rate limiting does not distort the timing.
	callRegistry("setConfig", big.NewInt(1_000_000), big.NewInt(1_000_000))

	prioByAddr := make(map[common.Address]Priority, numPrioritized)
	for i := 0; i < numPrioritized; i++ {
		addr := crypto.PubkeyToAddress(keys[i].PublicKey)
		level := uint64(1 + i%4)    // a few distinct levels
		weight := uint64(1 + i%100) // spread of weights
		var id [16]byte
		binary.BigEndian.PutUint64(id[8:], uint64(i)) // distinct entity per sender
		callRegistry("setSenderPriority", addr, level, weight, new(big.Int).SetBytes(id[:]))

		prioByAddr[addr] = Priority{Level: level, Weight: weight, ID: id}
	}

	if ch := statedb.EndBlock(1); ch != nil {
		require.NoError(<-ch)
	}

	// --- Block 2: serve the priority queries. ---
	statedb.BeginBlock(2)
	b.Cleanup(func() { statedb.EndBlock(2) })

	cfg, err := GetConfig(upgrades, evm)
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

// benchKey derives a deterministic private key from an index so the benchmark is
// reproducible and setup is fast.
func benchKey(i int) *ecdsa.PrivateKey {
	seed := crypto.Keccak256(binary.BigEndian.AppendUint64(nil, uint64(i)+1))
	for {
		key, err := crypto.ToECDSA(seed)
		if err == nil {
			return key
		}
		seed = crypto.Keccak256(seed)
	}
}
