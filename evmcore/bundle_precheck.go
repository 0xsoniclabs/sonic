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

package evmcore

import (
	"crypto/rand"
	"fmt"
	"maps"
	"math/big"

	"github.com/0xsoniclabs/sonic/evmcore/core_types"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/utils"
	"github.com/Fantom-foundation/lachesis-base/common/bigendian"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/metrics"
	lru "github.com/hashicorp/golang-lru"
)

var (
	// evaluatedBundleTxsCount is a counter of the number of bundle envelopes
	// that have been trial-run during bundle pre-checks.
	evaluatedBundlesCounter = metrics.GetOrRegisterCounter("bundles/pre_check/count", nil)
	// evaluatedBundlesExecutionCostCounter is a counter of the total execution cost in gas of
	// bundle envelopes that have been trial-run during bundle pre-checks.
	// This counter reports effective execution cost accounting for rolled-back transactions,
	// so it reflects the actual resource consumption of the trial runs.
	evaluatedBundlesExecutionCostCounter = metrics.GetOrRegisterCounter("bundles/pre_check/execution_cost", nil)
)

//go:generate mockgen -source=bundle_precheck.go -destination=bundle_precheck_mock.go -package=evmcore

// MinBundleEfficiency is the minimum efficiency threshold for accepting
// bundles. The efficiency of a bundle is defined as:
// sum of usedGas of receipts / total execution cost.
const MinBundleEfficiency = 0.2

// BundleState represents the current evaluation state of a transaction bundle.
// It indicates whether the bundle is executable as is, may become executable
// at a later point or will never be executable. If the bundle is not executable
// as is, it also provides a list of reasons explaining why.
// The 3 possible cases are:
//
//  1. Executable: the bundle can be executed with the current state, and there
//     are no known issues with it.
//
//     Example: BundleState{Executable: true}
//
//  2. Temporarily blocked: the bundle is not executable in the current state,
//     but it may become executable later (e.g., because it depends on a future
//     block or on the execution of other transactions that are not included in
//     the bundle).
//
//     Example: BundleState{
//     Executable: false,
//     TemporarilyBlocked: true,
//     Reasons: []string{"bundle targets future blocks"},
//     }
//
//  3. Not executable: the bundle is not executable and there are known issues
//     with it that will never be resolved (e.g., nonce conflicts that can not be
//     resolved by waiting for other transactions to be executed).
//
//     Example: BundleState{Executable: false, Reasons: []string{"reason 1", "reason 2"}}
type BundleState struct {
	Executable         bool     // True if the bundle can be executed with the current state.
	TemporarilyBlocked bool     // True if the bundle is currently blocked but may become executable later.
	Reasons            []string // A list of human-readable strings describing why the bundle is not executable or is blocked.
	GasEfficiency      *float64 // The billed gas to execution cost ratio of the bundle. Only set if it is a bundle and considered executable, nil otherwise.
}

// GetBundleState determines the state of the bundle based on the current
// chain state, state, and the transactions in the bundle.
//
// both chain and stateDb shall be non-nil and properly initialized.
// The lifetime of the chainDb object is not managed by this function and
// callers are responsible from releasing it when it is no longer needed.
func GetBundleState(
	chain ChainStateForBundleEval,
	stateDb state.StateDB,
	envelope *types.Transaction,
) BundleState {
	return getBundleState(chain, stateDb, envelope, trialRunBundle)
}

// getBundleState is the internal version of GetBundleState, allowing to inject
// a custom trial-run function to simplify testing.
func getBundleState(
	chain ChainStateForBundleEval,
	stateDb state.StateDB,
	envelope *types.Transaction,
	trialRunner func(*types.Transaction, ChainStateForBundleEval, state.StateDB) (*float64, bool),
) BundleState {
	chainId := big.NewInt(int64(chain.GetCurrentNetworkRules().NetworkID))
	signer := types.LatestSignerForChainID(chainId)

	// Check that bundled transactions are enabled.
	if !chain.GetCurrentNetworkRules().Upgrades.TransactionBundles {
		return makePermanentlyBlockedState("transaction bundles are not enabled on this network")
	}

	// Verify that the bundle is valid.
	bundle, _, err := bundle.ValidateEnvelope(signer, envelope)
	if err != nil {
		return makePermanentlyBlockedState(fmt.Sprintf("invalid bundle: %v", err))
	}

	// Quickest filter: check if the bundle is in the valid block range.
	currentBlock := chain.GetLatestHeader().Number.Uint64()
	if bundle.Plan.Range.IsAfterRange(currentBlock) {
		return makePermanentlyBlockedState("bundle has expired (block constraint)")
	}
	if bundle.Plan.Range.IsBeforeRange(currentBlock) {
		return makeTemporaryBlockedState("bundle targets future blocks")
	}

	// Check the time period limit of the bundle.
	currentTime := chain.GetLatestHeader().Time
	if bundle.Plan.Period.IsAfterPeriod(currentTime) {
		return makePermanentlyBlockedState("bundle has expired (time constraint)")
	}
	if bundle.Plan.Period.IsBeforePeriod(currentTime) {
		return makeTemporaryBlockedState("bundle targets future time period")
	}

	// Check that the bundle has not already been processed.
	if stateDb.HasBundleRecentlyBeenProcessed(bundle.Plan.Hash()) {
		return makePermanentlyBlockedState("bundle already processed")
	}

	// Next, check whether there are any nonce conflicts in the execution of
	// the bundle. This is a quicker check than actually running the bundle in
	// full to determine whether it can succeed or not.
	state := checkForNonceConflicts(bundle, signer, stateDb)
	if !state.Executable {
		return state
	}

	// Trial-run the bundle to check whether it can succeed or not. This is the
	// most expensive check, so it is performed at the end after all the cheaper
	// checks have passed. If we reach this point, nonces are aligned, so if it
	// fails, it means that there is something else wrong with the bundle (e.g.,
	// a missing pre-condition) that will never be resolved, and we can consider
	// the bundle as non-executable.

	// Make sure to revert all changes to enable re-using the same StateDB for
	// multiple calls to GetBundleState without having to create a new StateDB.
	snapshot := stateDb.InterTxSnapshot()
	defer stateDb.RevertToInterTxSnapshot(snapshot)

	gasEfficiency, valid := trialRunner(envelope, chain, stateDb)
	if !valid {
		return makePermanentlyBlockedState("bundle trial-run failed")
	}
	return makeRunnableState(gasEfficiency)
}

// ChainStateForBundleEval is an extension of the ChainState interface providing
// extra chain state information for trial-running bundles.
type ChainStateForBundleEval interface {
	ChainState

	// GetLatestHeader returns the latest block header of the chain.
	GetLatestHeader() *EvmHeader
}

type NonceSource interface {
	GetNonce(addr common.Address) uint64
}

// checkForNonceConflicts checks whether there are any nonce conflicts in the
// execution of the bundle.
//
// It returns a BundleState with Executable=false and a reason if there is a
// nonce conflict that will never be resolved.
//
// It returns a BundleState with Executable=false and TemporarilyBlocked=true
// if there is a nonce conflict that may be resolved in the future.
//
// It returns a BundleState with Executable=true if there are no nonce conflicts
// right now.
func checkForNonceConflicts(
	txBundle *bundle.TransactionBundle,
	signer types.Signer,
	nonceSource NonceSource,
) BundleState {

	// Step 1: run with current nonces to check whether the bundle is ready to
	// run right now. The runner does not allow nonce-gaps.
	strictRunner := &dryRunner{
		signer:       signer,
		nonceTracker: &nonceTracker{source: nonceSource},
	}
	if bundle.RunBundle(txBundle, strictRunner) {
		return makeRunnableState(nil)
	}

	// Step 2: check with future nonces, to check whether the bundle may become
	// executable in the future. The runner allows nonce-gaps.
	looseRunner := &dryRunner{
		signer:       signer,
		nonceTracker: &nonceTracker{source: nonceSource},
		allowGaps:    true, // for each account, a future nonce may be chosen once
	}
	if bundle.RunBundle(txBundle, looseRunner) {
		return makeTemporaryBlockedState("gapped nonce")
	}

	// Step 3: if it is still not successful, it means that there are nonce
	// conflicts that cannot be resolved by waiting for other transactions to
	// be executed, and we can consider the bundle as non-executable.
	return makePermanentlyBlockedState("bundle nonce check execution failed")
}

// dryRunner is an implementation of the TransactionRunner interface enabling
// the RunBundle function to be used for checking nonce conflicts without having
// to trial-run the bundle on the EVM nor having to duplicate the bundle
// execution logic in a separate function.
//
// It is only to be used by the checkForNonceConflicts function, which performs
// the proper lifecycle management of the dryRunner.
type dryRunner struct {
	signer       types.Signer
	nonceTracker *nonceTracker
	allowGaps    bool
	nonceLocked  map[common.Address]struct{}
	undo         []func()
}

func (r *dryRunner) Run(tx *types.Transaction) core_types.TransactionResult {
	// if the transaction is a nested bundle, process it as such
	if bundle.IsEnvelope(tx) {
		txBundle, err := bundle.OpenEnvelope(r.signer, tx)
		if err != nil {
			return core_types.TransactionResultInvalid
		}

		if bundle.RunBundle(&txBundle, r) {
			return core_types.TransactionResultSuccessful
		}

		return core_types.TransactionResultFailed
	}

	// check for nonce conflicts
	sender, err := types.Sender(r.signer, tx)
	if err != nil {
		return core_types.TransactionResultInvalid
	}
	if r.allowGaps && r.nonceLocked == nil {
		r.nonceLocked = make(map[common.Address]struct{})
	}

	got := tx.Nonce()
	want := r.nonceTracker.getNonce(sender)
	if got != want {
		// If gaps are allowed, we can skip over one nonce gap for each account.
		if got > want && r.allowGaps {
			if _, found := r.nonceLocked[sender]; !found {
				r.nonceLocked[sender] = struct{}{}
				r.nonceTracker.setNonce(sender, got+1)
				return core_types.TransactionResultSuccessful
			}
		}
		return core_types.TransactionResultInvalid
	}

	// This account has used the first nonce now, no more gaps allowed for it.
	if r.allowGaps {
		r.nonceLocked[sender] = struct{}{}
	}

	// if there are no nonce conflicts, consume the nonce for the sender and
	// continue with the next transaction in the bundle
	r.nonceTracker.setNonce(sender, got+1)
	return core_types.TransactionResultSuccessful
}

func (r *dryRunner) CreateSnapshot() int {
	nonceBackup := r.nonceTracker.backup()
	nonceLockedBackup := maps.Clone(r.nonceLocked)
	r.undo = append(r.undo, func() {
		r.nonceTracker.restore(nonceBackup)
		r.nonceLocked = nonceLockedBackup
	})
	return len(r.undo) - 1
}

func (r *dryRunner) RevertToSnapshot(id int) {
	for len(r.undo) > id {
		lastIndex := len(r.undo) - 1
		lastUndo := r.undo[lastIndex]
		r.undo = r.undo[:lastIndex]
		lastUndo()
	}
}

// nonceTracker is keeping track of consumed nonces during the execution of a
// bundle in dry-run mode. It allows to check for nonce conflicts without having
// to trial-run the bundle on the EVM.
type nonceTracker struct {
	source NonceSource
	nonces map[common.Address]uint64 // overrides
}

func (t *nonceTracker) getNonce(addr common.Address) uint64 {
	if nonce, ok := t.nonces[addr]; ok {
		return nonce
	}
	nonce := t.source.GetNonce(addr)
	t.setNonce(addr, nonce)
	return nonce
}

func (t *nonceTracker) setNonce(addr common.Address, nonce uint64) {
	if t.nonces == nil {
		t.nonces = make(map[common.Address]uint64)
	}
	t.nonces[addr] = nonce
}

func (t *nonceTracker) backup() *nonceTracker {
	return &nonceTracker{
		source: t.source,
		nonces: maps.Clone(t.nonces),
	}
}

func (t *nonceTracker) restore(backup *nonceTracker) {
	t.source = backup.source
	t.nonces = backup.nonces
}

// --- Trial Run Logic ---

// trialRunBundle performs a trial run of the bundle on the EVM to check whether it can
// succeed or not. It returns the gas efficiency (ratio of billed gas to execution cost)
// and a boolean indicating if the trial run results in at least one accepted transaction.
func trialRunBundle(
	envelope *types.Transaction,
	chain ChainStateForBundleEval,
	stateDb state.StateDB,
) (*float64, bool) {
	return trialRunBundleInternal(
		envelope,
		chain,
		stateDb,
		realTransactionProcessorFactory{},
		rand.Read,
		evaluatedBundlesCounter,
		evaluatedBundlesExecutionCostCounter,
	)
}

// trialRunBundleInternal is the internal version of trialRunBundle, allowing to
// inject a custom transaction processor factory to simplify testing.
func trialRunBundleInternal(
	envelope *types.Transaction,
	chain ChainStateForBundleEval,
	stateDb state.StateDB,
	factory transactionProcessorFactory,
	readRandom func([]byte) (int, error),
	evaluatedBundlesCount utils.MetricsCounter,
	evaluatedBundlesExecutionCost utils.MetricsCounter,
) (*float64, bool) {
	latestHeader := chain.GetLatestHeader()
	blobBaseFee := GetBlobBaseFee()

	// Create a random fake-PrevRandao for the trial run.
	var fakePrevRandao common.Hash
	if n, err := readRandom(fakePrevRandao[:]); n != len(fakePrevRandao) || err != nil {
		return nil, false
	}

	// Make sure that everything this function does is reverted at the end.
	snapshost := stateDb.InterTxSnapshot()
	defer stateDb.RevertToInterTxSnapshot(snapshost)

	// Create next block header state to trail-run against. This must cover
	// all fields that are accessible within the EVM.
	nextBlock := &EvmBlock{
		EvmHeader: EvmHeader{
			Number:      new(big.Int).Add(latestHeader.Number, big.NewInt(1)),
			Time:        latestHeader.Time + 1, // < minimum increase in each block
			GasLimit:    latestHeader.GasLimit, // < assume approximately uniform blocks
			Coinbase:    GetCoinbase(),
			PrevRandao:  fakePrevRandao,
			BaseFee:     latestHeader.BaseFee, // < assume base fee is not changing much
			BlobBaseFee: blobBaseFee.ToBig(),
		},
	}

	transactionProcessor := factory.newTransactionProcessor(chain, stateDb, nextBlock)
	summary := transactionProcessor.Run(0, envelope)

	usedGas := uint64(0)
	for _, tx := range summary.ProcessedTransactions {
		if tx.Receipt != nil {
			usedGas += tx.Receipt.GasUsed
		}
	}

	evaluatedBundlesCount.Inc(1)
	evaluatedBundlesExecutionCost.Inc(int64(summary.ExecutionCost))

	// Calculate the gas efficiency of the bundle and check if it meets the minimum threshold.
	gasEfficiency := new(float64)
	if summary.ExecutionCost > 0 {
		efficiency := float64(usedGas) / float64(summary.ExecutionCost)
		*gasEfficiency = efficiency
	}
	if summary.ExecutionCost == 0 || *gasEfficiency < MinBundleEfficiency {
		return gasEfficiency, false
	}

	// Check if the bundle lead to any accepted transactions. If so, it is
	// a success, otherwise it is a failure.
	for _, tx := range summary.ProcessedTransactions {
		if tx.Receipt != nil {
			return gasEfficiency, true
		}
	}
	return gasEfficiency, false
}

type transactionProcessorFactory interface {
	newTransactionProcessor(ChainState, state.StateDB, *EvmBlock) transactionProcessor
}

type transactionProcessor interface {
	Run(int, *types.Transaction) ProcessSummary
}

type realTransactionProcessorFactory struct{}

func (realTransactionProcessorFactory) newTransactionProcessor(
	chain ChainState,
	stateDb state.StateDB,
	block *EvmBlock,
) transactionProcessor {
	return NewTransactionProcessorForBlock(chain, stateDb, block)
}

func makeRunnableState(gasEfficiency *float64) BundleState {
	return BundleState{Executable: true, GasEfficiency: gasEfficiency}
}

func makeTemporaryBlockedState(reason string) BundleState {
	return BundleState{
		TemporarilyBlocked: true,
		Reasons:            []string{reason},
	}
}

func makePermanentlyBlockedState(reason string) BundleState {
	return BundleState{Reasons: []string{reason}}
}

// BundleEvaluator is an interface which exposes the GetBundleState function,
// The signature of GetBundleState is identical to the free standing function
// GetBundleState, this allows to inject caches in functions using the function.
type BundleEvaluator interface {
	GetBundleState(
		chain ChainStateForBundleEval,
		stateDb state.StateDB,
		envelope *types.Transaction,
	) BundleState
}

type bundleEvaluationCache struct {
	cache *lru.Cache
}

// NewBundleEvaluationCache creates a new instance of bundleEvaluationCache,
// which is an LRU cache for storing bundle evaluation results.
//
// Bundles are cached by their execution plan hash and the block number at
// which they were evaluated, to ensure that further evaluations in the same
// block can hit the cache, sparing the need to re-run the same bundle
// multiple times within the same block.
func NewBundleEvaluationCache() *bundleEvaluationCache {

	// The cache size of 100k entries is chosen arbitrarily and can be tuned
	// Evaluation of block current-1 are never used again, so we can consider
	// that the cache is only useful for the current block. This means that the
	// cache contains evaluations for up to 100k envelopes pending of execution.
	//
	// The cache does not proactively purge entries from previous blocks, instead
	// it relies on its LRU eviction policy to evict old entries when the cache
	// is full, older block evaluations are expected to be evicted first.
	//
	// Maximum memory consumption is a constant factor of the following formula:
	// number of bundles in the cache * (size of bundle plan hash + size of block number + size of BundleState)
	// 32  + 8 + ( 1+1 + 24, aligned to 32) = 72 bytes per entry, so ~7.2MiB for 100k entries.
	// Notice that the reasons in the bundle state point to static strings,
	// so they do not contribute to the memory consumption of each entry.
	cache, _ := lru.New(100_000)
	return &bundleEvaluationCache{cache: cache}
}

func (c *bundleEvaluationCache) GetBundleState(
	chain ChainStateForBundleEval,
	stateDb state.StateDB,
	envelope *types.Transaction,
) BundleState {

	if !bundle.IsEnvelope(envelope) {
		return makeRunnableState(nil)
	}

	chainId := big.NewInt(int64(chain.GetCurrentNetworkRules().NetworkID))
	signer := types.LatestSignerForChainID(chainId)

	// Caching evaluations by plan hash instead of envelope hash allows to
	// maximize cache hits, covering cases where different envelopes share
	// the same execution plan.
	// OpenEnvelope may be an expensive operation, this step relies on optimizations
	// done in OpenEnvelope itself to be efficient.
	txBundle, err := bundle.OpenEnvelope(signer, envelope)
	if err != nil {
		return makePermanentlyBlockedState(err.Error())
	}
	planHash := txBundle.Plan.Hash()

	blockNumber := uint64(0)
	if block := chain.GetLatestHeader(); block != nil {
		blockNumber = block.Number.Uint64()
	}

	key := string(append(
		bigendian.Uint64ToBytes(blockNumber),
		planHash.Bytes()...,
	))
	if state, ok := c.cache.Get(key); ok {
		return state.(BundleState)
	}

	state := GetBundleState(chain, stateDb, envelope)
	c.cache.Add(key, state)
	return state
}
