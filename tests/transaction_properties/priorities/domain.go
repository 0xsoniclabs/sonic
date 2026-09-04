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
// What became of the rest of the batch cannot change the answer, which is what the unreachable
// budget buys: a transaction that did not execute is the last of its sender's run, since the nonces
// behind it cannot execute either, and dropping the tail of a run leaves both the run and the order
// of what is left alone.
func (d *Domain) Check(rt *rapid.T, ctx context.Context, obs core.Observation) error {
	if err := d.Inner.Check(rt, ctx, obs); err != nil {
		return err
	}

	senderOf := make(map[common.Hash]common.Address, len(obs.Txs))
	executed := 0
	for _, tx := range obs.Txs {
		senderOf[tx.Hash] = tx.Sender.Address()
		if tx.Outcome == core.OutcomeExecuted {
			executed++
		}
	}

	found := 0
	seen := map[common.Address]uint64{}
	for _, block := range obs.Blocks {
		mine, positions := d.ours(block, senderOf)
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

		expected := PrioritizedPrefix(mine, starts, perEntityBudget)
		if err := d.compare(block, mine, positions, expected); err != nil {
			return err
		}

		for _, tx := range mine {
			seen[tx.Sender]++
		}
		d.prioritized += countPrioritized(mine)
		d.demoted += countPrioritized(mine) - len(expected)
		d.hoisted += len(expected)
		if len(expected) > 1 {
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
	senderOf map[common.Hash]common.Address,
) ([]BlockTx, []int) {

	mine := make([]BlockTx, 0, len(senderOf))
	positions := make([]int, 0, len(senderOf))
	for index, tx := range block.Transactions() {
		sender, ok := senderOf[tx.Hash()]
		if !ok {
			continue
		}
		mine = append(mine, BlockTx{
			Hash:     tx.Hash(),
			Sender:   sender,
			Nonce:    tx.Nonce(),
			Gas:      tx.Gas(),
			Priority: d.table[slot{sender, tx.Nonce()}],
		})
		positions = append(positions, index)
	}
	return mine, positions
}

// compare holds the block's order against the expected prefix, in the two ways it can be wrong: the
// hoisted transactions in the wrong order or the wrong set of them, and something else placed among
// them.
//
// An internal transaction is not something else: a sponsorship appends its follow-up immediately
// behind the transaction it belongs to, which is inside the prefix when that transaction was
// hoisted, and it is placed there by execution rather than by the ordering step.
func (d *Domain) compare(
	block *types.Block,
	mine []BlockTx,
	positions []int,
	expected []BlockTx,
) error {

	for i, want := range expected {
		if mine[i].Hash != want.Hash {
			return fmt.Errorf(
				"block %d puts %s at position %d of the batch's transactions, where the priority "+
					"rules call for %s\n%s",
				block.NumberU64(), mine[i], i, want, d.describe(mine, expected))
		}
	}

	if len(expected) == 0 {
		return nil
	}
	hoisted := hashesOf(expected)
	for index, tx := range block.Transactions() {
		if index >= positions[len(expected)-1] {
			break
		}
		if internaltx.IsInternal(tx) {
			continue
		}
		if _, ok := hoisted[tx.Hash()]; !ok {
			return fmt.Errorf(
				"block %d places %v at position %d, ahead of the prioritized transaction %s the "+
					"ordering must have hoisted past it\n%s",
				block.NumberU64(), tx.Hash().TerminalString(), index,
				expected[len(expected)-1], d.describe(mine, expected))
		}
	}
	return nil
}

// describe renders what the block held and what was expected of it, for a failure report.
func (d *Domain) describe(mine, expected []BlockTx) string {
	var out strings.Builder
	out.WriteString("the batch's transactions, in block order:")
	for _, tx := range mine {
		fmt.Fprintf(&out, "\n  %s", tx)
	}
	out.WriteString("\nexpected to be hoisted, in this order:")
	for _, tx := range expected {
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
			"they could not extend; %d blocks hoisted more than one, and so ordered them against "+
			"each other",
		d.blocks, d.prioritized, d.hoisted, d.demoted, d.ordered)
}

// label names a draw after the slot it belongs to, matching how the generators in core label theirs.
func label(name string, account, slot int) string {
	return fmt.Sprintf("%s_%d_%d", name, account, slot)
}
