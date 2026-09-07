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

// Package priorities layers transaction priorities over another domain's transactions: it draws a
// priority for every nonce slot the batch can reach, registers those in a registry that answers per
// (sender, nonce), and afterwards checks that every block the batch reached begins with exactly the
// transactions the ordering rules call for, in exactly their order. Nothing here is known to core or
// to the domain being wrapped.
//
// The priority of a transaction is not part of the transaction. It is on-chain state about a
// (sender, nonce) pair, which is why it is drawn here, in Prepare, rather than in the generator:
// what is drawn says what to do to the chain, not what to inject. rapid records and shrinks the
// draws either way -- the same reason the sponsorship funding levels are drawn where they are.
package priorities

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"slices"
	"strings"

	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/tests/transaction_properties/core"
	"github.com/0xsoniclabs/sonic/utils/signers/internaltx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"pgregory.net/rapid"
)

// The rate limits the registry answers with, both set once and beyond reach, never drawn.
//
// A perEntityBudget past what any batch can spend -- a batch above MaxEventGas is refused outright,
// and that is far below it -- is what makes every iteration checkable. Under a budget that binds, a
// prioritized transaction that never reached a block spent its entity's budget all the same if
// execution dropped it rather than the filter ahead of the ordering step, and which of the two
// happened is not observable from outside: the expected order of a block would then be unknown for
// every iteration that dropped a prioritized transaction, which most of them do. Whether the limit
// is enforced is checked where it can be decided, in the unit tests of
// gossip/blockproc/priorities -- and the model here still applies it, which its own unit tests
// cover.
//
// maxPiggybackTxs bounds what a validator eagerly puts into an event of its own, which is a path a
// forced batch never takes.
const (
	perEntityBudget = 1_000_000_000
	maxPiggybackTxs = 1_000
)

// The draws. A value appearing more than once is how these are weighted, and the order is part of it
// -- see the note on the shared generators in core/generators.go.
var (
	// levels leave a third of the slots ordinary, so that a batch mixes prioritized transactions with
	// ordinary ones and the two share a sender's nonce sequence, which is what the per-sender rule is
	// about, while most of what executes is still prioritized. Two distinct levels reach the ordering
	// between levels; the maximum reaches the boundary of the field the registry answers with.
	levels = rapid.SampledFrom([]uint64{0, 0, 1, 1, 2, math.MaxUint64})

	// weights decide the order within a level, and are mostly equal so that the hash tie-break is
	// reached rather than merely representable.
	weights = rapid.SampledFrom([]uint64{0, 0, 0, 1, 2, math.MaxUint64})

	// entities are two, so a batch usually carries both. The entity decides nothing while the budget
	// is beyond reach, which is the point of varying it anyway: what the registry answers there must
	// not reach the order, and an id leaking into the level or the weight would show up as an order
	// nothing asked for.
	entities = rapid.SampledFrom([]uint64{0, 1})
)

// slot names one transaction of a sender the way the registry does.
type slot struct {
	account common.Address
	nonce   uint64
}

// Domain layers transaction priorities over another domain: it draws that domain's batch unchanged,
// registers a priority for every nonce slot the batch can reach, and checks the order of the blocks
// it reached afterwards.
type Domain struct {
	// Inner is the domain whose transactions are prioritized. Priorities decide the order of a
	// block, never whether a transaction may execute or what it pays, so every rule of the inner
	// domain still holds unchanged.
	Inner core.Domain

	// Cfg is the generation budget, for the number of nonce slots a batch can reach.
	Cfg core.GenConfig

	registrar *Registrar
	client    *tests.PooledEhtClient

	// What the last Prepare established about the iteration now running. rapid runs iterations one at
	// a time, and every one of them calls Prepare before anything reads these.
	table map[slot]Priority
	start map[common.Address]uint64

	// What the run has seen, so a run that prioritized nothing is visible rather than merely green.
	blocks      int
	prioritized int
	hoisted     int
	ordered     int // blocks that hoisted more than one, and so ordered them against each other
	demoted     int
	unjudged    int // prioritized transactions whose place a rival for their nonce slot leaves open
}

// New builds the domain over a session whose network has transaction priorities enabled and the
// per-(sender, nonce) registry installed.
func New(
	session tests.IntegrationTestNetSession,
	client *tests.PooledEhtClient,
	network *core.Network,
	inner core.Domain,
	cfg core.GenConfig,
) (*Domain, error) {

	registrar, err := NewRegistrar(session, client, network)
	if err != nil {
		return nil, err
	}
	return &Domain{
		Inner:     inner,
		Cfg:       cfg,
		registrar: registrar,
		client:    client,
	}, nil
}

// Batch draws the inner domain's transactions, which a priority does not change.
func (d *Domain) Batch() *rapid.Generator[[]core.TxSpec] {
	return d.Inner.Batch()
}

// Pricing is the inner domain's: what a transaction pays is no business of its priority.
func (d *Domain) Pricing() core.PricingRules {
	return d.Inner.Pricing()
}

// Prepare registers a priority for every nonce slot the batch can reach.
//
// The window of one sender starts at the nonce it holds now and is as long as a batch has
// transactions, which is every slot a transaction of the batch can occupy and be prioritized in: a
// nonce below the window has been spent, and one above it sits behind a hole, and neither can extend
// the sender's contiguous run -- so whatever an earlier iteration may have left registered there
// cannot be hoisted.
func (d *Domain) Prepare(
	rt *rapid.T,
	ctx context.Context,
	accounts []core.PooledAccount,
	specs []core.TxSpec,
) error {

	if err := d.Inner.Prepare(rt, ctx, accounts, specs); err != nil {
		return err
	}

	d.table = map[slot]Priority{}
	d.start = map[common.Address]uint64{}

	windows := make([]Window, 0, len(accounts))
	for i, account := range accounts {
		nonce, err := d.client.NonceAt(ctx, account.Address(), nil)
		if err != nil {
			return fmt.Errorf("failed to read the nonce of %v: %w", account.Address(), err)
		}
		d.start[account.Address()] = nonce

		slots := make([]Priority, d.Cfg.MaxTxsPerBatch)
		for s := range slots {
			slots[s] = Priority{
				Level:  levels.Draw(rt, label("priorityLevel", i, s)),
				Weight: weights.Draw(rt, label("priorityWeight", i, s)),
				Entity: entities.Draw(rt, label("priorityEntity", i, s)),
			}
			d.table[slot{account.Address(), nonce + uint64(s)}] = slots[s]
		}
		windows = append(windows, Window{
			Account:    account.Address(),
			FirstNonce: nonce,
			Slots:      slots,
		})
	}
	return d.registrar.Register(ctx, windows)
}

// Skip is the inner domain's: a priority provokes nothing this one has to step around.
func (d *Domain) Skip(spec core.TxSpec, sender core.SenderState, baseFee *big.Int) string {
	return d.Inner.Skip(spec, sender, baseFee)
}

func (d *Domain) Extra(rt *rapid.T, ctx context.Context) ([]core.ExtraExecuted, error) {
	return d.Inner.Extra(rt, ctx)
}

func (d *Domain) Notes() []string {
	return d.Inner.Notes()
}

// Check verifies the order of every block the batch reached: it must begin with the transactions the
// ordering rules hoist, in their order, and nothing that was not hoisted may sit among them.
//
// The ordering step judges every transaction the block's events carried, not only the ones that go on
// to execute: it runs before execution, on what is left after a filter that asks nothing about nonces,
// balances or prices. So the batch's dropped transactions are part of the model's input too -- one
// dropped for what it costs still occupied its sender's nonce slot in the sequence the rules built,
// and a rival for that slot is then not the sender's own next nonce at all.
//
// What the model cannot decide is a slot two of the batch's transactions both claim: the two carry
// the same registered priority, being the same slot, and which of them the ordering step took is
// settled by their hashes among the transactions it was given -- a set a filter ahead of it may have
// thinned in a way nothing here can observe. The place of such a transaction is left unjudged, and
// reported as such.
func (d *Domain) Check(rt *rapid.T, ctx context.Context, obs core.Observation) error {
	if err := d.Inner.Check(rt, ctx, obs); err != nil {
		return err
	}

	injected := make(map[common.Hash]core.TxObservation, len(obs.Txs))
	executed := 0
	for _, tx := range obs.Txs {
		injected[tx.Hash] = tx
		if tx.Outcome == core.OutcomeExecuted {
			executed++
		}
	}
	dropped := d.dropped(obs)

	found := 0
	seen := map[common.Address]uint64{}
	for _, block := range obs.Blocks {
		mine, positions := d.ours(block, injected)
		if len(mine) == 0 {
			continue
		}
		found += len(mine)

		// Where each sender stood when this block started: what it held before the batch, plus what
		// it has already spent in the blocks before this one.
		starts := map[common.Address]uint64{}
		for sender, before := range d.start {
			starts[sender] = before + seen[sender]
		}

		candidates := append(slices.Clone(mine), dropped...)
		contested := contestedSlots(candidates)
		hoisted, err := d.compare(
			block, mine, positions, PrioritizedPrefix(candidates, starts, perEntityBudget), contested)
		if err != nil {
			return err
		}

		for _, tx := range mine {
			if injected[tx.Hash].Spec.SigningMode() == core.SignCorrect {
				seen[tx.Sender]++
			}
		}
		prioritized := countPrioritized(mine)
		unjudged := 0
		for _, tx := range mine {
			if tx.Priority.IsPrioritized() && contested[slot{tx.Sender, tx.Nonce}] {
				unjudged++
			}
		}
		d.prioritized += prioritized
		d.unjudged += unjudged
		d.hoisted += len(hoisted)
		d.demoted += prioritized - unjudged - len(hoisted)
		if len(hoisted) > 1 {
			d.ordered++
		}
		d.blocks++
	}

	if found != executed {
		return fmt.Errorf(
			"%d transactions of the batch executed but %d of them appear in the blocks of this "+
				"iteration, so the order of a block was judged against an incomplete batch",
			executed, found)
	}
	return nil
}

// ours picks the batch's transactions out of a block, in the order the block puts them in, along
// with the index each one sits at.
func (d *Domain) ours(
	block *types.Block,
	injected map[common.Hash]core.TxObservation,
) ([]BlockTx, []int) {

	mine := make([]BlockTx, 0, len(injected))
	positions := make([]int, 0, len(injected))
	for index, tx := range block.Transactions() {
		observed, ok := injected[tx.Hash()]
		if !ok {
			continue
		}
		sender := observed.Sender.Address()
		mine = append(mine, BlockTx{
			Hash:     tx.Hash(),
			Sender:   sender,
			Nonce:    tx.Nonce(),
			Gas:      tx.Gas(),
			Priority: d.priorityOf(observed.Spec, sender, tx.Nonce()),
		})
		positions = append(positions, index)
	}
	return mine, positions
}

// dropped is what the ordering step was given of the batch and no block kept, so that the sequences
// it built are modelled from the same transactions rather than from the survivors alone. The nonce is
// the one the transaction was built with and the gas its drawn limit, both being what it carried into
// the block it was dropped from.
//
// Some of these the filter ahead of the ordering step took out instead, which cannot be told apart
// from here. Naming one too many is harmless: alone at its nonce slot it displaces nothing, and no
// block kept it, so it is dropped from the prefix to be looked for -- and a later nonce of its sender
// cannot be in a block either, that slot never having been spent. Where it does displace something it
// shares the slot with it, which is what leaves such a slot unjudged.
func (d *Domain) dropped(obs core.Observation) []BlockTx {
	out := make([]BlockTx, 0, len(obs.Txs))
	for _, tx := range obs.Txs {
		if tx.Outcome == core.OutcomeExecuted {
			continue
		}
		sender := tx.Sender.Address()
		out = append(out, BlockTx{
			Hash:     tx.Hash,
			Sender:   sender,
			Nonce:    tx.Nonce,
			Gas:      tx.Spec.Gas(),
			Priority: d.priorityOf(tx.Spec, sender, tx.Nonce),
		})
	}
	return out
}

// priorityOf is the priority the ordering step gives one of the batch's transactions: what the
// registry holds for its sender's nonce slot, and nothing at all unless the signature recovers to the
// sender the transaction was drawn for. The registry answers per (sender, nonce), so a transaction the
// chain attributes to a stranger holds no slot of its sender's -- and none of its own, nothing being
// registered for an address the pool never handed out.
func (d *Domain) priorityOf(spec core.TxSpec, sender common.Address, nonce uint64) Priority {
	if spec.SigningMode() != core.SignCorrect {
		return Priority{}
	}
	return d.table[slot{sender, nonce}]
}

// contestedSlots are the nonce slots more than one of the batch's transactions claims. The ordering
// step took exactly one of them into its sender's sequence, by their hashes among the transactions it
// was given, and that set is not the one observable here -- so which it was, and with it the place of
// every one of them, is not the model's to decide.
func contestedSlots(candidates []BlockTx) map[slot]bool {
	claims := map[slot]int{}
	for _, tx := range candidates {
		if tx.Priority.IsPrioritized() {
			claims[slot{tx.Sender, tx.Nonce}]++
		}
	}
	contested := map[slot]bool{}
	for at, claimed := range claims {
		if claimed > 1 {
			contested[at] = true
		}
	}
	return contested
}

// judged sorts out what a comparison holds against each other: the batch's transactions of the
// block whose place the rules decide, in block order and with where the block puts them, and the
// ones of the expected prefix to be looked for among them. Two of the expected ones are left out --
// one the ordering step hoisted and execution then dropped, which is in no block to be found in, and
// one at a contested slot, which is left out of both sides.
func judged(
	mine []BlockTx,
	positions []int,
	expected []BlockTx,
	kept map[common.Hash]struct{},
	contested map[slot]bool,
) (want, got []BlockTx, at []int) {

	want = make([]BlockTx, 0, len(expected))
	for _, tx := range expected {
		_, inBlock := kept[tx.Hash]
		if inBlock && !contested[slot{tx.Sender, tx.Nonce}] {
			want = append(want, tx)
		}
	}

	got = make([]BlockTx, 0, len(mine))
	at = make([]int, 0, len(mine))
	for i, tx := range mine {
		if contested[slot{tx.Sender, tx.Nonce}] {
			continue
		}
		got = append(got, tx)
		at = append(at, positions[i])
	}
	return want, got, at
}

// compare holds the block's order against the expected prefix, in the two ways it can be wrong: the
// hoisted transactions in the wrong order or the wrong set of them, and something else placed among
// them. It returns the hoisted transactions it judged.
//
// Two of the expected ones are passed over rather than looked for: one the ordering step hoisted and
// execution then dropped, which is in no block to be found in, and one at a contested slot, whose
// place is unjudged on both sides. An internal transaction is not something else: a sponsorship
// appends its follow-up immediately behind the transaction it belongs to, which is inside the prefix
// when that transaction was hoisted, and it is placed there by execution rather than by the ordering
// step.
func (d *Domain) compare(
	block *types.Block,
	mine []BlockTx,
	positions []int,
	expected []BlockTx,
	contested map[slot]bool,
) ([]BlockTx, error) {

	kept := hashesOf(mine)
	want, got, at := judged(mine, positions, expected, kept, contested)

	for i, tx := range want {
		if got[i].Hash != tx.Hash {
			return nil, fmt.Errorf(
				"block %d puts %s at position %d of the batch's transactions whose place the "+
					"priority rules decide, where the rules call for %s\n%s",
				block.NumberU64(), got[i], i, tx, d.describe(mine, expected, want))
		}
	}

	if len(want) == 0 {
		return want, nil
	}
	for index, tx := range block.Transactions() {
		if index >= at[len(want)-1] {
			break
		}
		if internaltx.IsInternal(tx) {
			continue
		}
		if _, ours := kept[tx.Hash()]; !ours {
			return nil, fmt.Errorf(
				"block %d places %v at position %d, ahead of the prioritized transaction %s the "+
					"ordering must have hoisted past it\n%s",
				block.NumberU64(), tx.Hash().TerminalString(), index,
				want[len(want)-1], d.describe(mine, expected, want))
		}
	}
	return want, nil
}

// describe renders what the block held and what was expected of it, for a failure report. The
// prefix the rules call for is given twice: as the ordering step decided it, transactions no block
// kept included, and as what is left of it to look for.
func (d *Domain) describe(mine, expected, want []BlockTx) string {
	var out strings.Builder
	out.WriteString("the batch's transactions, in block order:")
	for _, tx := range mine {
		fmt.Fprintf(&out, "\n  %s", tx)
	}
	out.WriteString("\nthe prioritized prefix the rules call for, in this order:")
	for _, tx := range expected {
		fmt.Fprintf(&out, "\n  %s", tx)
	}
	out.WriteString("\nof which these are the ones to be found in the block, in this order:")
	for _, tx := range want {
		fmt.Fprintf(&out, "\n  %s", tx)
	}
	return out.String()
}

func hashesOf(txs []BlockTx) map[common.Hash]struct{} {
	out := make(map[common.Hash]struct{}, len(txs))
	for _, tx := range txs {
		out[tx.Hash] = struct{}{}
	}
	return out
}

func countPrioritized(txs []BlockTx) int {
	count := 0
	for _, tx := range txs {
		if tx.Priority.IsPrioritized() {
			count++
		}
	}
	return count
}

// String renders what the run saw of priorities, which is what says whether the ordering was
// exercised at all rather than merely not violated.
func (d *Domain) String() string {
	if d.blocks == 0 {
		return "no block of a prioritized batch was checked"
	}
	return fmt.Sprintf(
		"%d blocks checked: %d prioritized transactions, %d of them hoisted, %d demoted by a nonce "+
			"they could not extend, %d left unjudged by a rival for their nonce slot; %d blocks "+
			"hoisted more than one, and so ordered them against each other",
		d.blocks, d.prioritized, d.hoisted, d.demoted, d.unjudged, d.ordered)
}

// label names a draw after the slot it belongs to, matching how the generators in core label theirs.
func label(name string, account, slot int) string {
	return fmt.Sprintf("%s_%d_%d", name, account, slot)
}
