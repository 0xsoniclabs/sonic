package emitter

import (
	"context"
	"math/big"
	"time"

	"github.com/0xsoniclabs/sonic/gossip/emitter/scheduler"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

//go:generate mockgen -source=proposals.go -destination=proposals_mock.go -package=emitter

// createPayload creates payload to be attached to the given event. The result
// may include a new block proposal if the current validator is allowed to make
// a proposal. Otherwise, the payload contains meta-data required to track the
// progress of the proposal process.
//
// This function is supposed to be called during the event creation process. The
// provided sorted and indexed list of transactions is the set of candidate
// transactions that should be included in a new block. These transactions are
// executed during the scheduling step of the block-proposal process. If their
// preconditions (nonce, balance, gas-price-limit, etc.) are met, making them
// eligible for an inclusion in the proposed block, they are included.
// Non-eligible transactions are ignored.
//
// The process may fail if the current set of validators is empty and no
// proposer for the current turn can be determined.
func (em *Emitter) createPayload(
	event inter.EventI,
	sorted *transactionsByPriceAndNonce,
) (inter.Payload, error) {
	adapter := worldAdapter{External: em.world}
	return createPayload(
		adapter,
		em.config.Validator.ID,
		em.validators,
		event,
		&em.proposalTracker,
		sorted,
		scheduler.NewScheduler(adapter),
		proposalSchedulingTimer,
		proposalSchedulingTimeoutCounter,
	)
}

// createPayload is a helper function which constructs the payload for every
// event if the single-proposer mode is enabled. It performs the following
// operations:
//   - it determines the current ProposalSyncState based on the event's parents
//   - it checks if the current validator is allowed to propose a new block
//   - if allowed, it creates a new block proposal and returns it in the payload
//
// The resulting payload contains valid ProposalSyncState information and
// optionally a new block proposal, if all preconditions are met.
func createPayload(
	world worldReader,
	validator idx.ValidatorID,
	validators *pos.Validators,
	event inter.EventI,
	proposalTracker proposalTracker,
	sorted *transactionsByPriceAndNonce,
	transactionScheduler txScheduler,
	durationMetric timerMetric,
	timeoutMetric counterMetric,
) (inter.Payload, error) {

	// Get the last seen proposal information from the event's parents.
	incomingState := inter.CalculateIncomingProposalSyncState(world, event)

	// Do not re-propose a pending proposal.
	currentFrame := event.Frame()
	latest := world.GetLatestBlock()
	nextBlock := idx.Block(latest.Number + 1)
	if proposalTracker.IsPending(currentFrame, nextBlock) {
		return inter.Payload{
			ProposalSyncState: incomingState,
		}, nil
	}

	// Determine whether this validator is allowed to propose a new block.
	isMyTurn, err := inter.IsAllowedToPropose(
		validator,
		validators,
		incomingState,
		currentFrame,
	)
	if err != nil {
		return inter.Payload{}, err
	}
	if !isMyTurn {
		return inter.Payload{
			ProposalSyncState: incomingState,
		}, nil
	}

	// Make a new proposal. For the time of the block we use the median time,
	// which is the median of all creation times of the events seen from all
	// validators.
	proposal := makeProposal(
		world.GetRules(),
		incomingState,
		latest,
		event.MedianTime(), // < time of the new block
		currentFrame,
		transactionScheduler,
		&transactionPriorityAdapter{sorted},
		durationMetric,
		timeoutMetric,
	)

	// If no new proposal was created, the payload remains empty.
	if proposal == nil {
		return inter.Payload{
			ProposalSyncState: incomingState,
		}, nil
	}

	// If a proposal was made, the sync state is updated to reflect the
	// new proposal.
	return inter.Payload{
		ProposalSyncState: inter.ProposalSyncState{
			LastSeenProposalTurn:  incomingState.LastSeenProposalTurn + 1,
			LastSeenProposalFrame: currentFrame,
		},
		Proposal: proposal,
	}, nil
}

// proposalTracker is an interface for tracking proposals to avoid double
// proposals.
type proposalTracker interface {
	// IsPending checks whether there is a pending proposal for the given frame
	// and block. If the proposal is pending, it returns true, otherwise it
	// returns false.
	IsPending(frame idx.Frame, block idx.Block) bool
}

// worldReader is an interface for a data source providing all the information
// about the current chain state required for creating a new block proposal.
type worldReader interface {
	inter.EventReader
	GetLatestBlock() *inter.Block
	GetRules() opera.Rules
}

// worldAdapter is an adapter of the External interface to the worldReader
// and the scheduler.Chain interface.
type worldAdapter struct {
	External
}

func (w worldAdapter) GetEventPayload(event hash.Event) inter.Payload {
	return *w.External.GetEventPayload(event).Payload()
}

func (w worldAdapter) GetEvmChainConfig() *params.ChainConfig {
	return w.GetRules().EvmChainConfig(w.GetUpgradeHeights())
}

// --- proposal creation ---

// makeProposal creates a new block proposal based on the given context
// information. The resulting proposal may be nil if the preconditions for
// making a new proposal are not met (e.g., if no time has passed since the
// last block).
func makeProposal(
	rules opera.Rules,
	incomingSyncState inter.ProposalSyncState,
	latestBlock *inter.Block,
	newBlockTime inter.Timestamp,
	currentFrame idx.Frame,
	transactionScheduler txScheduler,
	candidates scheduler.PrioritizedTransactions,
	durationMetric timerMetric,
	timeoutMetric counterMetric,
) *inter.Proposal {
	// Compute the gas limit for the next block. This is the time since the
	// previous block times the targeted network throughput.
	lastBlockTime := latestBlock.Time
	if lastBlockTime >= newBlockTime {
		// no time has passed, so no new proposal can be made
		return nil
	}
	effectiveGasLimit := getEffectiveGasLimit(
		newBlockTime.Time().Sub(lastBlockTime.Time()),
		rules.Economy.ShortGasPower.AllocPerSec, // TODO: consider using a new rule set parameter
	)

	// Create the proposal for the next block.
	proposal := &inter.Proposal{
		Number:     idx.Block(latestBlock.Number) + 1,
		ParentHash: latestBlock.Hash(),
		Time:       newBlockTime,
		// PrevRandao: -- compute next randao mix based on predecessor --
	}

	// This step covers the actual transaction selection and sorting.
	start := time.Now()
	ctx, cancel := context.WithDeadline(
		context.Background(),
		start.Add(100*time.Millisecond),
	)
	defer cancel()
	proposal.Transactions = transactionScheduler.Schedule(
		ctx,
		&scheduler.BlockInfo{
			Number:      proposal.Number,
			Time:        proposal.Time,
			GasLimit:    rules.Blocks.MaxBlockGas,
			MixHash:     common.Hash{},    // TODO: integrate randao reveal
			Coinbase:    common.Address{}, // TODO: integrate coinbase address
			BaseFee:     uint256.Int{},    // TODO: integrate base fee
			BlobBaseFee: uint256.Int{},    // TODO: integrate blob base fee
		},
		candidates,
		effectiveGasLimit,
	)

	// Track scheduling time in monitoring metrics.
	durationMetric.Update(time.Since(start))
	if ctx.Err() != nil {
		timeoutMetric.Inc(1)
	}

	return proposal
}

// txScheduler is an interface for scheduling transactions in a block
// abstracting the actual scheduler implementation to facilitate testing.
type txScheduler interface {
	Schedule(context.Context, *scheduler.BlockInfo, scheduler.PrioritizedTransactions, uint64) []*types.Transaction
}

// timerMetric is an abstraction for monitoring metrics to facilitate testing.
type timerMetric interface {
	Update(time.Duration)
}

// counterMetric is an abstraction for monitoring metrics to facilitate testing.
type counterMetric interface {
	Inc(int64)
}

// We put a strict cap of 2 second on the maximum time gas can be accumulated
// for a single block. Thus, if the delay between two blocks is less than 2
// seconds, gas is accumulated linearly. If the delay is longer than 2 seconds,
// we cap the gas to the maximum accumulation time. This is to limit the maximum
// block size to at most 2 seconds worth of gas.
const maxAccumulationTime = 2 * time.Second

// getEffectiveGasLimit computes the effective gas limit for the next block.
// This is the time since the last block times the targeted network throughput.
// The result is capped to the gas that corresponds to a maximum accumulation
// time of maxAccumulationTime.
func getEffectiveGasLimit(
	delta time.Duration,
	targetedThroughput uint64,
) uint64 {
	if delta <= 0 {
		return 0
	}
	if delta > maxAccumulationTime {
		delta = maxAccumulationTime
	}
	return new(big.Int).Div(
		new(big.Int).Mul(
			big.NewInt(int64(targetedThroughput)),
			big.NewInt(int64(delta.Nanoseconds())),
		),
		big.NewInt(int64(time.Second.Nanoseconds())),
	).Uint64()
}

// transactionPriorityAdapter is an adapter between the transactionsByPriceAndNonce
// and the scheduler's PrioritizedTransactions interface.
type transactionPriorityAdapter struct {
	sorted transactionIndex
}

func (a *transactionPriorityAdapter) Current() *types.Transaction {
	tx, _ := a.sorted.Peek()
	if tx == nil {
		return nil
	}
	return tx.Resolve()
}

func (a *transactionPriorityAdapter) Accept() {
	a.sorted.Shift()
}

func (a *transactionPriorityAdapter) Skip() {
	a.sorted.Pop()
}

type transactionIndex interface {
	Peek() (*txpool.LazyTransaction, *uint256.Int)
	Shift()
	Pop()
}
