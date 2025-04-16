package emitter

import (
	"fmt"
	"math/big"
	"time"

	"github.com/0xsoniclabs/sonic/eventcheck/epochcheck"
	"github.com/0xsoniclabs/sonic/evmcore"
	scrambler "github.com/0xsoniclabs/sonic/gossip/scrambler_new"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/core/types"
)

func (em *Emitter) addProposal(
	event *inter.MutableEventPayload,
	sorted *transactionsByPriceAndNonce,
) error {

	const enableProposalDebugPrints = true

	lastSeenProposalTurn := inter.Turn(0)
	lastSeenProposedBlock := idx.Block(0)
	lastSeenProposalFrame := idx.Frame(0)
	//fmt.Printf("Adding proposals to event %v in frame %d\n", event.ID(), event.Frame())
	if parents := event.Parents(); len(parents) == 0 {
		lastSeenProposedBlock = em.world.GetEpochStartBlock(event.Epoch())
	} else {
		for _, parent := range parents {
			//fmt.Printf("\tParent %v\n", parent)
			payload := em.world.GetEventPayload(parent)
			envelope := payload.ProposalEnvelope()
			turn := envelope.LastSeenProposalTurn
			block := envelope.LastSeenProposedBlock
			frame := envelope.LastSeenProposalFrame
			//fmt.Printf("\t\tLast Proposal %d/%d @ frame=%d\n", number, attempt, frame)

			if turn > lastSeenProposalTurn {
				lastSeenProposalTurn = turn
			}
			if block > lastSeenProposedBlock {
				lastSeenProposedBlock = block
			}
			if frame > lastSeenProposalFrame {
				lastSeenProposalFrame = frame
			}
		}
	}

	// By default, we fill the envelope with the latest seen proposal information.
	// By adding this to all events, event validation can track the progress of
	// the block-proposing based on events and their parents. It also enables
	// the detection if invalid proposals.
	event.SetProposalEnvelope(&inter.ProposalEnvelope{
		LastSeenProposalTurn:  lastSeenProposalTurn,
		LastSeenProposedBlock: lastSeenProposedBlock,
		LastSeenProposalFrame: lastSeenProposalFrame,
	})

	// Check whether it is this emitter's turn to propose a new block.
	nextTurn := lastSeenProposalTurn + 1
	proposer, err := inter.GetProposer(em.validators, nextTurn)
	if err != nil || proposer != em.config.Validator.ID {
		return err
	}

	// Check that enough time has passed for the next proposal.
	latest := em.world.GetLatestBlock()
	nextBlock := idx.Block(latest.Number + 1)
	valid := inter.IsValidTurnProgression(
		inter.ProposalSummary{
			Turn:  lastSeenProposalTurn,
			Block: lastSeenProposedBlock,
			Frame: lastSeenProposalFrame,
		},
		inter.ProposalSummary{
			Turn:  nextTurn,
			Block: nextBlock,
			Frame: event.Frame(),
		},
	)
	if !valid {
		return nil
	}

	// --- Build a new Proposal ---

	if enableProposalDebugPrints {
		fmt.Printf(
			"validator=%d, starting proposal for turn %d, block %d as part of epoch %d, frame %d @ t=%v (last proposal at frame %d)\n",
			em.config.Validator.ID, nextTurn, nextBlock, event.Epoch(), event.Frame(), event.CreationTime().Time(), lastSeenProposalFrame,
		)
	}

	rules := em.world.GetRules()

	// For the time of the block we use the median time, which is the median
	// of all creation times of last-seen validator times.
	nextBlockTime := event.MedianTime()

	// Compute the gas limit for the next block. This is the time since the
	// previous block times the targeted network throughput.
	lastBlockTime := latest.Time
	if lastBlockTime >= nextBlockTime {
		return nil // no time has passed, no proposal to be made
	}
	effectiveGasLimit := getEffectiveGasLimit(
		nextBlockTime.Time().Sub(lastBlockTime.Time()),
		rules.Economy.ShortGasPower.AllocPerSec, // TODO: consider using a new rule set parameter
	)

	// Create the proposal for the next block.
	proposal := &inter.Proposal{
		Number:     nextBlock,
		ParentHash: latest.Hash(),
		Time:       nextBlockTime,
		// PrevRandao: -- compute next randao mix based on predecessor --
	}

	// This step covers the actual transaction selection and sorting.
	proposal.Transactions = em.getTransactionsForProposal(
		&blockContext{
			Number:   proposal.Number,
			Time:     proposal.Time,
			GasLimit: rules.Blocks.MaxBlockGas,
		},
		sorted,
		effectiveGasLimit,
	)

	// TODO: remove
	if enableProposalDebugPrints {
		fmt.Printf(
			"validator=%d, completed proposal for turn %d, block %d with %d transactions\n",
			em.config.Validator.ID, nextTurn, nextBlock, len(proposal.Transactions),
		)
		for _, tx := range proposal.Transactions {
			sender, _ := types.Sender(em.world.TxSigner, tx)
			tip := tx.EffectiveGasTipValue(new(big.Int))
			fmt.Printf("\tTransaction with sender %v, nonce %d, effective tip %v\n", sender, tx.Nonce(), tip)
		}
	}

	// Envelop and append the new proposal to the event.
	event.SetProposalEnvelope(&inter.ProposalEnvelope{
		LastSeenProposalTurn:  nextTurn,
		LastSeenProposedBlock: nextBlock,
		LastSeenProposalFrame: event.Frame(),
		Proposal:              proposal,
	})

	return nil
}

type blockContext struct {
	Number   idx.Block
	Time     inter.Timestamp
	GasLimit uint64
}

func (em *Emitter) getTransactionsForProposal(
	context *blockContext,
	sorted *transactionsByPriceAndNonce,
	gasLimit uint64,
) []*types.Transaction {
	if false {
		// This is the algorithm we think is the default in go-ethereum
		return em.scheduleByGasTipOnly(context, sorted, gasLimit)
	}
	return em.scheduleAndScramble(context, sorted, gasLimit)
}

// scheduleByGasTipOnly prioritizes transactions by their gas tip. The highest
// tips are selected first.
func (em *Emitter) scheduleByGasTipOnly(
	context *blockContext,
	sorted *transactionsByPriceAndNonce,
	gasLimit uint64,
) []*types.Transaction {
	// Create a block-processor instance to check that selected transactions
	// are indeed executable in the given order on the current state.
	block := &evmcore.EvmBlock{
		EvmHeader: evmcore.EvmHeader{
			Number:   new(big.Int).SetUint64(uint64(context.Number)),
			Time:     context.Time,
			GasLimit: context.GasLimit,
			// TODO: add missing fields like base-fee, randao, ...
		},
	}

	// TODO: align this with c_block_callbacks.go
	rules := em.world.GetRules()
	chainCfg := rules.EvmChainConfig(em.world.GetUpgradeHeights())

	stateDb := em.world.StateDB()
	defer stateDb.Release()

	var usedGas uint64 // - everything, including ignored transactions
	runTx := evmcore.NewStateProcessor(chainCfg, em.world).BeginBlock(
		block, stateDb, opera.DefaultVMConfig, &usedGas,
	)

	// sort transactions by price and nonce
	remainingGas := gasLimit
	transactions := []*types.Transaction{}
	for tx, _ := sorted.Peek(); tx != nil; tx, _ = sorted.Peek() {
		resolvedTx := tx.Resolve()

		/*
			sender, _ := types.Sender(em.world.TxSigner, resolvedTx)
			nonce := stateDb.GetNonce(sender)
			fmt.Printf("Candidate transaction from sender %s, tx-nonce %d, db-nonce %d, ...\n", sender.Hex(), resolvedTx.Nonce(), nonce)
		*/

		// check transaction epoch rules (tx type, gas price)
		if epochcheck.CheckTxs(types.Transactions{resolvedTx}, rules) != nil {
			txsSkippedEpochRules.Inc(1)
			sorted.Pop()
			continue
		}

		// make sure the transaction can be processed
		receipt, skipped, err := runTx(len(transactions), resolvedTx)
		if skipped || err != nil {
			if skipped {
				fmt.Printf("\tTransaction skipped\n")
			} else {
				fmt.Printf("\tExecution error: %v\n", err)
			}
			txsSkippedExecutionError.Inc(1)
			sorted.Pop()
			continue
		}

		if remainingGas < receipt.GasUsed {
			sorted.Pop()
			continue
		}

		// add transaction to the proposed list of transactions
		transactions = append(transactions, resolvedTx)
		remainingGas -= receipt.GasUsed
		sorted.Shift()
	}
	return transactions
}

// ---- New Scheduling Algorithm Prototype ----

// This algorithm aims to satisfy the following properties:
//  - prioritize high-tip transactions to be included in blocks
//  - use a verifiable random order of selected transactions in blocks

func (em *Emitter) scheduleAndScramble(
	context *blockContext,
	sorted *transactionsByPriceAndNonce,
	gasLimit uint64,
) []*types.Transaction {

	rules := em.world.GetRules()
	candidates := []*types.Transaction{}

	growCandidates := func(size int) bool {
		hasGrown := false
		for len(candidates) < size {
			tx, _ := sorted.Peek()
			if tx == nil {
				break
			}
			resolvedTx := tx.Resolve()

			// check transaction epoch rules (tx type, gas price)
			if epochcheck.CheckTxs(types.Transactions{resolvedTx}, rules) != nil {
				txsSkippedEpochRules.Inc(1)
				sorted.Pop()
				continue
			}
			candidates = append(candidates, resolvedTx)
			hasGrown = true
			sorted.Shift()
		}

		/*
			fmt.Printf("New candidates:\n")
			for _, tx := range candidates {
				fmt.Printf("\tTransaction with tip %d\n", tx.EffectiveGasTipValue(new(big.Int)))
			}
		*/

		return hasGrown
	}

	start := time.Now()

	var bestOrder []*types.Transaction
	var bestGasUsed uint64
	var bestCandidateSize int

	// Phase one: find an upper bound for the number of transactions to consider.
	reachedGasLimit := false
	numEvals := 0
	for i := 1; ; i *= 2 {
		if !growCandidates(i) {
			break
		}
		//fmt.Printf("Testing %d transactions\n", len(candidates))
		numEvals++
		order, gasUsed, reachedLimit := em.evaluateTransactions(&sorted.signer, context, candidates, gasLimit)
		if gasUsed > bestGasUsed {
			bestGasUsed = gasUsed
			bestOrder = order
			bestCandidateSize = len(candidates)
		}
		if reachedLimit {
			reachedGasLimit = true
			break
		}
	}

	//fmt.Printf("Tested up to %d candidates, best contains %d transactions, exceeded gas limit %t\n", len(candidates), len(bestOrder), reachedGasLimit)

	// Phase two: fine-tune if not all transactions were used.
	if reachedGasLimit {
		// binary search the interval between the best solution and the candidate list
		low := len(bestOrder)
		high := len(candidates)
		for low < high {
			mid := (low + high) / 2
			//fmt.Printf("Testing %d transactions (low: %d, high: %d)\n", mid, low, high)
			numEvals++
			order, gasUsed, reachedLimit := em.evaluateTransactions(&sorted.signer, context, candidates[:mid], gasLimit)
			// We replace our current best if we can get more gas used with a smaller candidate list size.
			// This is a trade-off between us honoring the priority expressed by tips and the maximum
			// gas we can pack into a block.
			if gasUsed > bestGasUsed || (gasUsed == bestGasUsed && mid < bestCandidateSize) {
				bestGasUsed = gasUsed
				bestOrder = order
				bestCandidateSize = mid
			}
			if reachedLimit {
				high = mid
			} else {
				low = mid + 1
			}
		}
	}

	fmt.Printf("Scheduling took %v, %d runs, %.1f%% full\n", time.Since(start), numEvals, 100*float64(bestGasUsed)/float64(gasLimit))

	return bestOrder
}

func (em *Emitter) evaluateTransactions(
	signer *types.Signer,
	context *blockContext,
	transactions []*types.Transaction,
	gasLimit uint64,
) (
	[]*types.Transaction, // list of selected transactions
	uint64, // gas used by resulting transaction list
	bool, // true if the gas limit was reached and candidates were ignored
) {

	// TODO: add a context supporting a time-out

	// Create a block-processor instance to check that selected transactions
	// are indeed executable in the given order on the current state.
	block := &evmcore.EvmBlock{
		EvmHeader: evmcore.EvmHeader{
			Number:   new(big.Int).SetUint64(uint64(context.Number)),
			Time:     context.Time,
			GasLimit: context.GasLimit,
			// TODO: add missing fields like base-fee, randao, ...
		},
	}

	// TODO: align this with c_block_callbacks.go
	rules := em.world.GetRules()
	chainCfg := rules.EvmChainConfig(em.world.GetUpgradeHeights())

	stateDb := em.world.StateDB()
	defer stateDb.Release()

	var _usedGas uint64 // - everything, including ignored transactions
	runTx := evmcore.NewStateProcessor(chainCfg, em.world).BeginBlock(
		block, stateDb, opera.DefaultVMConfig, &_usedGas,
	)

	// sort transactions by price and nonce
	reachedLimit := false
	remainingGas := gasLimit

	// TODO: pass randao as seed
	scrambled := scramble(signer, 42, transactions)
	selected := make([]*types.Transaction, 0, len(scrambled))
	for _, tx := range scrambled {

		// test whether this transaction can be processed
		receipt, skipped, err := runTx(len(selected), tx)
		if skipped || err != nil {
			// TODO: add some monitoring
			//txsSkippedExecutionError.Inc(1)
			continue
		}

		if remainingGas < receipt.GasUsed {
			// TODO: add some monitoring
			reachedLimit = true
			continue
		}

		// add transaction to the proposed list of transactions
		selected = append(selected, tx)
		remainingGas -= receipt.GasUsed
	}
	return selected, gasLimit - remainingGas, reachedLimit
}

// scramble returns a random permutation of the given transactions. The result
// is a copy of the input slice, so the input is not modified.
func scramble(signer *types.Signer, seed uint64, transactions []*types.Transaction) []*types.Transaction {
	return scrambler.Scramble(*signer, seed, transactions)
}

func getEffectiveGasLimit(
	delta time.Duration,
	targetedThroughput uint64,
) uint64 {
	// We put a strict cap of 2 second on the accumulated gas. Thus, if the delay
	// between two blocks is less than 2 seconds, gas is accumulated linearly.
	// If the delay is longer than 2 seconds, we cap the gas to the maximum
	// accumulation time. This is to limit the maximum block size to at most
	// 2 seconds worth of gas.
	const maxAccumulationTime = 2 * time.Second
	if delta > maxAccumulationTime {
		delta = maxAccumulationTime
	}
	if delta <= 0 {
		return 0
	}
	return new(big.Int).Div(
		new(big.Int).Mul(
			big.NewInt(int64(targetedThroughput)),
			big.NewInt(int64(delta.Nanoseconds())),
		),
		big.NewInt(int64(time.Second.Nanoseconds())),
	).Uint64()
}
