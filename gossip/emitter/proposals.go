// Copyright 2025 Sonic Operations Ltd
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

package emitter

import (
	"context"
	"fmt"
	"time"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/gossip/emitter/scheduler"
	"github.com/0xsoniclabs/sonic/gossip/gasprice"
	"github.com/0xsoniclabs/sonic/gossip/randao"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
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
	randaoMixer := randao.NewRandaoMixerAdapter(em.world.EventsSigner)
	return createPayload(
		adapter,
		em.config.Validator.ID,
		em.validators.Load(),
		event,
		&em.proposalTracker,
		sorted,
		scheduler.NewScheduler(adapter),
		randaoMixer,
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
	randaoMixer randao.RandaoMixer,
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
	currentEpoch := event.Epoch()
	isMyTurn, turn, err := inter.IsAllowedToPropose(
		validator,
		validators,
		incomingState,
		currentEpoch,
		currentFrame,
	)
	if err != nil {
		return inter.Payload{},
			fmt.Errorf("failed to create event payload, %w", err)
	}
	if !isMyTurn {
		return inter.Payload{
			ProposalSyncState: incomingState,
		}, nil
	}

	// Make a new proposal. For the time of the block we use the median time,
	// which is the median of all creation times of the events seen from all
	// validators.
	proposal, err := makeProposal(
		world.GetRules(),
		incomingState,
		latest,
		event.MedianTime(), // < time of the new block
		currentFrame,
		transactionScheduler,
		&transactionPriorityAdapter{sorted},
		randaoMixer,
		durationMetric,
		timeoutMetric,
	)
	if err != nil {
		return inter.Payload{},
			fmt.Errorf("failed to create event payload: %w", err)
	}

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
			LastSeenProposalTurn:  turn,
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

func (w worldAdapter) GetCurrentNetworkRules() opera.Rules {
	return w.GetRules()
}

func (w worldAdapter) GetEvmChainConfig(blockHeight idx.Block) *params.ChainConfig {
	return opera.CreateTransientEvmChainConfig(
		w.GetRules().NetworkID,
		w.GetUpgradeHeights(),
		blockHeight,
	)
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
	randaoMixer randao.RandaoMixer,
	durationMetric timerMetric,
	timeoutMetric counterMetric,
) (*inter.Proposal, error) {
	// Compute the gas limit for the next block. This is the time since the
	// previous block times the targeted network throughput.
	lastBlockTime := latestBlock.Time
	if lastBlockTime >= newBlockTime {
		// no time has passed, so no new proposal can be made
		return nil, nil
	}
	blockGasLimit := rules.Blocks.MaxBlockGas
	effectiveGasLimit := inter.GetEffectiveGasLimit(
		newBlockTime.Time().Sub(lastBlockTime.Time()),
		rules.Economy.ShortGasPower.AllocPerSec,
		blockGasLimit,
	)

	randaoReveal, randaoMix, err := randaoMixer.MixRandao(latestBlock.PrevRandao)
	if err != nil {
		return nil, fmt.Errorf("randao reveal generation failed: %w", err)
	}

	// Create the proposal for the next block.
	proposal := &inter.Proposal{
		Number:       idx.Block(latestBlock.Number) + 1,
		ParentHash:   latestBlock.Hash(),
		RandaoReveal: randaoReveal,
	}

	// Compute the base fee for the next block.
	parentBlockInfo := gasprice.ParentBlockInfo{
		BaseFee:  latestBlock.BaseFee,
		Duration: time.Duration(latestBlock.Duration),
		GasUsed:  latestBlock.GasUsed,
	}
	baseFee := gasprice.GetBaseFeeForNextBlock(parentBlockInfo, rules.Economy)
	baseFee256, overflow := uint256.FromBig(baseFee)
	if overflow {
		return nil, fmt.Errorf("required base fee %s overflows uint256", baseFee)
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
			Time:        newBlockTime,
			GasLimit:    blockGasLimit,
			MixHash:     randaoMix,
			Coinbase:    evmcore.GetCoinbase(),
			BaseFee:     *baseFee256,
			BlobBaseFee: evmcore.GetBlobBaseFee(),
		},
		candidates,
		scheduler.Limits{
			Gas:  effectiveGasLimit,
			Size: maxTotalTransactionsSizeInEventInBytes,
		},
	)

	// Track scheduling time in monitoring metrics.
	durationMetric.Update(time.Since(start))
	if ctx.Err() != nil {
		timeoutMetric.Inc(1)
	}

	return proposal, nil
}

// txScheduler is an interface for scheduling transactions in a block
// abstracting the actual scheduler implementation to facilitate testing.
type txScheduler interface {
	Schedule(
		context.Context,
		*scheduler.BlockInfo,
		scheduler.PrioritizedTransactions,
		scheduler.Limits,
	) []*types.Transaction
}

// timerMetric is an abstraction for monitoring metrics to facilitate testing.
type timerMetric interface {
	Update(time.Duration)
}

// counterMetric is an abstraction for monitoring metrics to facilitate testing.
type counterMetric interface {
	Inc(int64)
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
