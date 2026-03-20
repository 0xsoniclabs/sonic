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
	"fmt"
	"maps"
	"math"
	big "math/big"

	"github.com/0xsoniclabs/sonic/evmcore/core_types"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	state "github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	params "github.com/ethereum/go-ethereum/params"
)

//go:generate mockgen -source=bundle_precheck.go -destination=bundle_precheck_mock.go -package=evmcore

// BundleState represents the current evaluation state of a transaction bundle.
// It indicates whether the bundle is executable, if it is temporarily blocked
// from execution. If the bundle is not executable, it also provides a list of
// reasons explaining why.
//
// Fields:
//   - Executable:         True if the bundle can be executed.
//   - TemporarilyBlocked: True if the bundle is currently blocked but may become executable later.
//   - Reasons:            A list of human-readable strings describing why the bundle is not executable or is blocked.
type BundleState struct {
	Executable         bool
	TemporarilyBlocked bool
	Reasons            []string
}

// GetBundleState determines the state of the bundle based on the current state
// of the blockchain and the transactions in the bundle.
func GetBundleState(
	chain ChainState,
	envelop *types.Transaction,
) BundleState {
	return getBundleState(chain, envelop, trialRunBundle)
}

// getBundleState is the internal version of GetBundleState, allowing to inject
// a custom trial-run function to simplify testing.
func getBundleState(
	chain ChainState,
	envelop *types.Transaction,
	trialRunner func(*types.Transaction, ChainState, state.StateDB) bool,
) BundleState {

	// Verify that the bundle is valid.
	bundle, _, err := bundle.ValidateTransactionBundle(envelop)
	if err != nil {
		return BundleState{
			Executable: false,
			Reasons:    []string{fmt.Sprintf("invalid bundle: %v", err)},
		}
	}

	// Quickest filter: check if the bundle is in the valid block range.
	currentBlock := chain.GetLatestHeader().Number.Uint64()
	if bundle.Latest < currentBlock {
		return BundleState{
			Executable: false,
			Reasons:    []string{ErrBundleLatestPassed.Error()},
		}
	}
	if bundle.Earliest > currentBlock {
		return BundleState{
			Executable:         true,
			TemporarilyBlocked: true,
		}
	}

	// Next, check whether there are any nonce conflicts in the execution of
	// the bundle. This is a quick check than actually running the bundle in
	// full to determine whether it can succeed or not.
	chainId := big.NewInt(int64(chain.GetCurrentNetworkRules().NetworkID))
	signer := types.LatestSignerForChainID(chainId)
	stateDb := chain.StateDB()
	state := checkForNonceConflicts(bundle, signer, stateDb)
	if !state.Executable {
		state.Reasons = append([]string{"nonce conflict check failed"}, state.Reasons...)
		return state
	}

	if state.TemporarilyBlocked {
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
	defer func() {
		// TODO: follow-up task: deal with this error or update the function to
		// not return an error at all if it can not be handled properly.
		stateDb.RevertToInterTxSnapshot(snapshot)
	}()

	if success := trialRunner(envelop, chain, stateDb); !success {
		return BundleState{
			Executable: false,
			Reasons:    []string{"bundle trial-run failed. Revise transactions in the plan"},
		}
	}
	return BundleState{Executable: true}
}

type ChainState interface {
	// DummyChain needs to be implemented in order to resolve past block hashes.
	DummyChain

	// GetCurrentNetworkRules returns the current network rules for the EVM.
	GetCurrentNetworkRules() opera.Rules

	// GetEvmChainConfig returns the chain configuration for the EVM at the
	// given block height
	GetEvmChainConfig(blockHeight idx.Block) *params.ChainConfig

	// StateDB returns a context for running transactions on the head state of
	// the chain. A non-committable state-DB instance is sufficient.
	StateDB() state.StateDB

	// GetLatestHeader returns the latest block header of the chain.
	GetLatestHeader() *EvmHeader
}

type NonceSource interface {
	GetNonce(addr common.Address) uint64
}

// checkForNonceConflicts checks whether there are any nonce conflicts in the
// execution of the bundle.
// It returns a BundleState with Executable=false and a reason if there is a nonce conflict
// that will never be resolved.
// It returns a BundleState with Executable=true and TemporarilyBlocked=true if there is a nonce conflict that may
// be resolved in the future.
// It returns a BundleState with Executable=true if there are no nonce conflicts right now.
func checkForNonceConflicts(
	txBundle *bundle.TransactionBundle,
	signer types.Signer,
	nonceSource NonceSource,
) BundleState {
	// We start by collecting the lowest nonces referenced for each sender in
	// the bundle.
	lowest, err := getLowestReferencedNonces(txBundle, signer)
	if err != nil {
		// If we fail to derive the lowest referenced nonces, it means that the
		// bundle is malformed (e.g., contains invalid transactions) and we can
		// consider it as non-executable.
		return BundleState{
			Executable: false,
			Reasons:    []string{fmt.Sprintf("could not get lowest nonce for all accounts: %v", err)},
		}
	}

	// We correct the lowest nonces to be at least as high as the current nonces
	// for each sender. Lower nonces are no longer available.
	for sender, lowestNonce := range lowest {
		lowest[sender] = max(lowestNonce, nonceSource.GetNonce(sender))
	}

	// With those lowest nonces as a start, we attempt to run the bundle.
	runner := &dryRunner{
		signer:         signer,
		nonceTracker:   &nonceTracker{nonces: maps.Clone(lowest)},
		acceptedSender: make(map[common.Address]struct{}),
	}

	// If this execution failed, the bundle is non-executable.
	if success := bundle.RunBundle(txBundle, runner); !success {
		return BundleState{
			Executable: false,
			Reasons:    []string{"bundle nonce check execution failed"},
		}
	}

	// If it succeeded, it depends on whether there is a gap between the lowest
	// and the current nonces for any sender of an accepted transaction.
	for sender := range runner.acceptedSender {
		if nonceSource.GetNonce(sender) < lowest[sender] {
			return BundleState{
				Executable:         true,
				TemporarilyBlocked: true,
			}
		}
	}
	return BundleState{Executable: true}
}

// getLowestReferencedNonces returns the lowest nonce referenced for each sender
// in the bundle. If the bundle is malformed (e.g., contains invalid signatures)
// an error is returned.
func getLowestReferencedNonces(
	txBundle *bundle.TransactionBundle,
	signer types.Signer,
) (map[common.Address]uint64, error) {
	res := make(map[common.Address]uint64)
	for _, tx := range txBundle.Transactions {
		if bundle.IsEnvelope(tx) {
			bundle, err := bundle.OpenEnvelope(tx)
			if err != nil {
				return nil, fmt.Errorf("invalid nested bundle: %w", err)
			}
			innerRes, err := getLowestReferencedNonces(bundle, signer)
			if err != nil {
				return nil, err
			}
			for addr, nonce := range innerRes {
				if existingNonce, ok := res[addr]; !ok || nonce < existingNonce {
					res[addr] = nonce
				}
			}
		} else {
			sender, err := types.Sender(signer, tx)
			if err != nil {
				return nil, fmt.Errorf("failed to derive sender: %w", err)
			}
			if nonce, ok := res[sender]; !ok || tx.Nonce() < nonce {
				res[sender] = tx.Nonce()
			}
		}
	}
	return res, nil
}

// dryRunner is an implementation of the TransactionRunner interface enabling
// the RunBundle function to be used for checking nonce conflicts without having
// to trial-run the bundle on the EVM nor having to duplicate the bundle
// execution logic in a separate function.
//
// It is only to be used by the checkForNonceConflicts function, which performs
// the proper lifecycle management of the dryRunner.
type dryRunner struct {
	signer         types.Signer
	nonceTracker   *nonceTracker
	acceptedSender map[common.Address]struct{}
}

func (r *dryRunner) Run(tx *types.Transaction) core_types.TransactionResult {

	// if the transaction is a nested bundle, process it as such
	if bundle.IsEnvelope(tx) {
		txBundle, err := bundle.OpenEnvelope(tx)
		if err != nil {
			return core_types.TransactionResultInvalid
		}
		acceptedBackup := maps.Clone(r.acceptedSender)
		backup := r.nonceTracker.backup()
		if bundle.RunBundle(txBundle, r) {
			return core_types.TransactionResultSuccessful
		}
		r.nonceTracker.restore(backup)
		r.acceptedSender = acceptedBackup
		return core_types.TransactionResultFailed
	}

	// check for nonce conflicts
	sender, err := types.Sender(r.signer, tx)
	if err != nil {
		return core_types.TransactionResultInvalid
	}
	want := r.nonceTracker.getNonce(sender)
	if tx.Nonce() < want {
		return core_types.TransactionResultInvalid
	}
	if tx.Nonce() > want {
		return core_types.TransactionResultInvalid
	}

	// if there are no nonce conflicts, consume the nonce for the sender and
	// continue with the next transaction in the bundle
	r.nonceTracker.consumeNonce(sender)
	r.acceptedSender[sender] = struct{}{}
	return core_types.TransactionResultSuccessful
}

// nonceTracker is keeping track of consumed nonces during the execution of a
// bundle, recording the lowest required nonce per account.
type nonceTracker struct {
	nonces map[common.Address]uint64
}

func (t *nonceTracker) getNonce(addr common.Address) uint64 {
	return t.nonces[addr]
}

func (t *nonceTracker) consumeNonce(addr common.Address) {
	t.nonces[addr]++
}

func (t *nonceTracker) backup() *nonceTracker {
	return &nonceTracker{
		nonces: maps.Clone(t.nonces),
	}
}

func (t *nonceTracker) restore(backup *nonceTracker) {
	t.nonces = backup.nonces
}

// --- Trial Run Logic ---

func trialRunBundle(
	envelop *types.Transaction,
	chain ChainState,
	stateDb state.StateDB,
) bool {

	latestHeader := chain.GetLatestHeader()
	blobBaseFee := GetBlobBaseFee()

	// Create next block header state to trail-run against.
	nextBlock := &EvmBlock{
		EvmHeader: EvmHeader{
			Number:      new(big.Int).Add(latestHeader.Number, big.NewInt(1)),
			Time:        latestHeader.Time + 1,
			GasLimit:    math.MaxInt64, // < assume limit high enough
			Coinbase:    GetCoinbase(),
			PrevRandao:  common.Hash{1, 2, 3}, // < can not be predicted
			BaseFee:     latestHeader.BaseFee, // < assume base fee is not changing much
			BlobBaseFee: blobBaseFee.ToBig(),
		},
	}

	// TODO: follow-up task - align this with c_block_callbacks.go and single
	// proposer scheduler code. Ideally, they would share a common code base.
	chainCfg := chain.GetEvmChainConfig(idx.Block(nextBlock.Header().Number.Uint64()))
	vmConfig := opera.GetVmConfig(chain.GetCurrentNetworkRules())

	gasLimit := uint64(math.MaxUint64)
	stateProcessor := NewStateProcessor(
		chainCfg, chain, chain.GetCurrentNetworkRules().Upgrades,
	)
	transactionProcessor := stateProcessor.BeginBlock(nextBlock, stateDb, vmConfig, gasLimit, nil)
	summary := transactionProcessor.Run(0, envelop)

	// Check if the bundle lead to any accepted transactions. If so, it is
	// a success, otherwise it is a failure.
	for _, tx := range summary.ProcessedTransactions {
		if tx.Receipt != nil {
			return true
		}
	}
	return false
}
