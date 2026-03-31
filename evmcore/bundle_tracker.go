package evmcore

import (
	"maps"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type bundleTrackerEntry struct {
	envelopeHash common.Hash
	// if not nil, the block number when the bundle was sunsetted.
	blockNum *uint64
}

type BundleTracker struct {
	state StateReader
	// map from execution plan hash to the corresponding envelope hash and the
	// block number when the bundle was sunsetted (if it was sunsetted)
	index map[common.Hash]bundleTrackerEntry
}

func NewBundleTracker(state StateReader) *BundleTracker {
	return &BundleTracker{
		state: state,
		index: make(map[common.Hash]bundleTrackerEntry),
	}
}

// TrackTransaction starts tracking execution of a potential bundle by its envelope transaction.
// if the transaction is not an envelope, it will be ignored.
func (t *BundleTracker) TrackTransaction(tx *types.Transaction) {
	if !bundle.IsEnvelope(tx) {
		return
	}

	plan, err := bundle.ExtractExecutionPlan(tx)
	if err != nil {
		return
	}

	execPlanHash := plan.Hash()
	t.index[execPlanHash] = bundleTrackerEntry{envelopeHash: tx.Hash()}
}

// SunsetTransaction marks the bundle corresponding to the given transaction as
// sunsetted at the current block number.
//
// The tracker will not forget this bundle until the sunset is final, which
// hapens after a tolerance of blocks have been executed. Allowing to execute
// any instances of this bundle on flight in the consensus dag.
func (t *BundleTracker) SunsetTransaction(tx *types.Transaction) {
	blockNum := t.state.CurrentBlock().NumberU64()
	defer t.cleanup(blockNum)

	if !bundle.IsEnvelope(tx) {
		return
	}

	plan, err := bundle.ExtractExecutionPlan(tx)
	if err != nil {
		return
	}

	execPlanHash := plan.Hash()
	if entry, ok := t.index[execPlanHash]; ok {
		entry.blockNum = &blockNum
		t.index[execPlanHash] = entry
	}
}

// IsBundlePending checks whether the bundle with the given execution plan hash
// is pending.
// A bundle is pending if we are still waiting for its execution result, which
// is the case if:
//   - we know about this bundle and it was not sunsetted yet, or
//   - we know about this bundle, it was sunsetted but the sunset is not final
//     yet, and it has not been executed.
func (t *BundleTracker) IsBundlePending(execPlanHash common.Hash) bool {
	blockNum := t.state.CurrentBlock().NumberU64()
	defer t.cleanup(blockNum)

	if entry, ok := t.index[execPlanHash]; ok {

		if entry.blockNum == nil {
			// if the bundle is not sunsetted, it's pending
			return true
		} else if t.isSunsetFinal(blockNum, entry.blockNum) {
			// if sunset is final, it's not pending anymore. It has been discarded
			return false
		}
		// if sunset is not final, it may have been executed
		return !t.state.HasBundleBeenProcessed(execPlanHash)
	}
	// never heard of this bundle before or already forgotten
	return false
}

func (t *BundleTracker) cleanup(blockHeight uint64) {
	maps.DeleteFunc(t.index, func(_ common.Hash, entry bundleTrackerEntry) bool {
		return t.isSunsetFinal(blockHeight, entry.blockNum)
	})
}

// inFlightBlocksTolerance is the number of blocks after the sunset of a bundle
// during which the tracker will still consider it pending, to allow any
// in-flight instances of this bundle in the consensus dag to be executed.
const inFlightBlocksTolerance = 3

func (t *BundleTracker) isSunsetFinal(currentBlockNum uint64, sunsetBlockNum *uint64) bool {
	if sunsetBlockNum == nil {
		return false
	}
	return currentBlockNum >= *sunsetBlockNum+inFlightBlocksTolerance
}
