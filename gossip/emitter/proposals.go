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
	return createPayload(
		worldReaderAdapter{em.world},
		em.validators,
		em.config.Validator.ID,
		event,
		sorted,
		scheduler.NewScheduler(&em.world),
	)
}

// --- internal types and helper functions ---

// createPayload is a helper function handling the actual creation of the
// payload. It is detached from the Emitter to facilitate testing.
func createPayload(
	world worldReader,
	validators *pos.Validators,
	thisValidator idx.ValidatorID,
	event inter.EventI,
	sorted *transactionsByPriceAndNonce,
	transactionScheduler txScheduler,
) (inter.Payload, error) {

	// Get the last seen proposal information from the event's parents.
	incomingState := inter.GetIncomingProposalState(world, event)

	// Determine whether this validator is allowed to propose a new block.
	currentFrame := event.Frame()
	latest := world.GetLatestBlock()
	nextBlock := idx.Block(latest.Number + 1)
	isMyTurn, err := inter.IsAllowedToPropose(
		incomingState,
		currentFrame,
		validators,
		thisValidator,
		nextBlock,
	)
	if err != nil {
		return inter.Payload{}, err
	}
	if !isMyTurn {
		return incomingState.ToPayload(), nil
	}

	// Make a new proposal. For the time of the block we use the median time,
	// which is the median of all creation times of the events seen from all
	// validators.
	return createProposal(
		world.GetRules(),
		incomingState,
		latest,
		event.MedianTime(), // < time of the new block
		currentFrame,
		transactionScheduler,
		&transactionPriorityAdapter{sorted},
		proposalSchedulingTimer,
		proposalSchedulingTimeoutCounter,
	)
}

// worldReader is an interface for a data source providing all the information
// about the current chain state required for creating a new block proposal.
type worldReader interface {
	inter.EventReader
	GetLatestBlock() *inter.Block
	GetRules() opera.Rules
}

type worldReaderAdapter struct {
	world External
}

func (w worldReaderAdapter) GetLatestBlock() *inter.Block {
	return w.world.GetLatestBlock()
}

func (w worldReaderAdapter) GetEpochStartBlock(epoch idx.Epoch) idx.Block {
	return w.world.GetEpochStartBlock(epoch)
}

func (w worldReaderAdapter) GetEventPayload(event hash.Event) inter.Payload {
	return *w.world.GetEventPayload(event).Payload()
}

func (w worldReaderAdapter) GetRules() opera.Rules {
	return w.world.GetRules()
}

// --- proposal creation ---

// createProposal creates a new block proposal based on the given context
// information. The resulting payload contains the new block proposal.
func createProposal(
	rules opera.Rules,
	proposalState inter.ProposalSyncState,
	latestBlock *inter.Block,
	newBlockTime inter.Timestamp,
	currentFrame idx.Frame,
	transactionScheduler txScheduler,
	candidates scheduler.PrioritizedTransactions,
	durationMetric timerMetric,
	timeoutMetric counterMetric,
) (inter.Payload, error) {
	// Compute the gas limit for the next block. This is the time since the
	// previous block times the targeted network throughput.
	lastBlockTime := latestBlock.Time
	if lastBlockTime >= newBlockTime {
		// no time has passed, so no new proposal can be made
		return proposalState.ToPayload(), nil
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
			MixHash:     proposal.Randao,
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

	// Produce updated payload with the new proposal.
	return inter.Payload{
		LastSeenProposalTurn:  proposalState.LastSeenProposalTurn + 1,
		LastSeenProposedBlock: proposal.Number,
		LastSeenProposalFrame: currentFrame,
		Proposal:              proposal,
	}, nil
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

// We put a strict cap of 2 second on the accumulated gas. Thus, if the delay
// between two blocks is less than 2 seconds, gas is accumulated linearly.
// If the delay is longer than 2 seconds, we cap the gas to the maximum
// accumulation time. This is to limit the maximum block size to at most
// 2 seconds worth of gas.
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

// transactionPriorityAdapter is an adapter between the *transactionsByPriceAndNonce
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
