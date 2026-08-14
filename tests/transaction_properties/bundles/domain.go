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

package bundles

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"math/big"
	"slices"
	"strings"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/tests/transaction_properties/core"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"pgregory.net/rapid"
)

// Domain layers bundles over another domain: it draws that domain's batch, replaces one transaction
// with an envelope carrying a batch of its own, builds the bundle, and afterwards checks that the
// bundle either executed whole or not at all.
type Domain struct {
	// Inner is the domain whose transactions are bundled, and whose rules still price everything that
	// enters a block on its own account -- including every transaction inside a bundle, which pays
	// its own gas at the base fee like any other.
	Inner core.Domain

	// Offset is where the contents' senders start among the accounts an iteration claimed. The batch
	// around the bundle uses the accounts below it, so the two nonce sequences never meet.
	Offset int

	client *tests.PooledEhtClient
	cfg    core.NetworkConfig

	// What the last Prepare established about the iteration now running. rapid runs iterations one at
	// a time, and every one of them calls Prepare before anything reads these.
	envelopes []*Envelope
	contents  []core.PooledAccount
	inner     []core.SenderState

	// refused counts the contents the model refused outright, by reason, alongside demoted below.
	refused map[string]int

	// skipped counts the contents the inner domain would not have injected, by reason.
	skipped map[string]int

	// demoted counts the bundles whose bare root had to become a group, working around defect3.
	demoted int

	// What the run has seen of bundles, so a run that executed none is visible rather than green, and
	// what kept the rest from running, so the draws can be aimed at what is worth drawing.
	observed map[Fate]int
	reasons  map[string]int
}

// Fate is what became of a bundle, which is what the domain counts.
type Fate uint8

const (
	// FateExecuted means every transaction of the bundle reached the block.
	FateExecuted Fate = iota
	// FateSkipped means none did.
	FateSkipped
)

func (f Fate) String() string {
	if f == FateExecuted {
		return "Executed"
	}
	return "Skipped"
}

// New builds the domain over a session whose network runs the upgrades the scenario asked for. The
// offset must leave room for both windows: the batch's senders below it and the contents' above.
func New(
	client *tests.PooledEhtClient,
	network *core.Network,
	inner core.Domain,
	offset int,
) (*Domain, error) {

	if offset <= 0 {
		return nil, fmt.Errorf("the contents need a sender window of their own, got offset %d", offset)
	}
	return &Domain{
		Inner:    inner,
		Offset:   offset,
		client:   client,
		cfg:      network.Cfg,
		observed: map[Fate]int{},
		reasons:  map[string]int{},
		refused:  map[string]int{},
		skipped:  map[string]int{},
	}, nil
}

// Batch draws the inner domain's transactions and turns one of them into an envelope.
func (d *Domain) Batch() *rapid.Generator[[]core.TxSpec] {
	return Bundling(d.Inner.Batch())
}

// Pricing prices an envelope by what a bundle does to it and everything else by the inner rules.
func (d *Domain) Pricing() core.PricingRules {
	return d.PricingRules(d.Inner.Pricing())
}

// Prepare builds the bundle of every envelope in the batch. Nothing has to be arranged on chain --
// a bundle costs no block -- but the envelope's payload and gas limit follow from its contents, and
// those can only be assembled once the accounts and the block number are known.
func (d *Domain) Prepare(
	rt *rapid.T,
	ctx context.Context,
	accounts []core.PooledAccount,
	specs []core.TxSpec,
) error {

	if err := d.Inner.Prepare(rt, ctx, accounts, specs); err != nil {
		return err
	}

	d.envelopes = envelopesIn(specs)
	if len(d.envelopes) == 0 {
		return nil
	}
	if len(accounts) < 2*d.Offset {
		return fmt.Errorf(
			"a bundle needs %d accounts for its contents above the %d of the batch, but %d were claimed",
			d.Offset, d.Offset, len(accounts))
	}

	block, err := d.client.BlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("failed to read the block number: %w", err)
	}
	header, err := d.client.HeaderByNumber(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to read the latest header: %w", err)
	}

	// The contents' senders, read once: their nonces are theirs alone, since nothing outside a bundle
	// draws from this window.
	d.contents = accounts[d.Offset:]
	d.inner = make([]core.SenderState, len(d.contents))
	for i, account := range d.contents {
		nonce, err := d.client.NonceAt(ctx, account.Address(), nil)
		if err != nil {
			return fmt.Errorf("failed to read the nonce of %v: %w", account.Address(), err)
		}
		balance, err := d.client.BalanceAt(ctx, account.Address(), nil)
		if err != nil {
			return fmt.Errorf("failed to read the balance of %v: %w", account.Address(), err)
		}
		d.inner[i] = core.SenderState{Nonce: nonce, Balance: balance}
	}

	for _, envelope := range d.envelopes {
		built, err := d.build(envelope, d.contents, block, header.BaseFee)
		if err != nil {
			return err
		}
		envelope.Built = built
	}
	return nil
}

// build assembles one bundle with the production builder, which is what stamps every transaction with
// the bundle-only marker, raises its gas limit by that marker's own cost, computes the plan hash and
// signs. Rebuilding any of that here would only test this file against itself.
func (d *Domain) build(
	envelope *Envelope,
	accounts []core.PooledAccount,
	block uint64,
	baseFee *big.Int,
) (*Built, error) {

	signer := types.LatestSignerForChainID(d.cfg.ChainId)
	buildCtx := core.BuildContext{
		ChainId:     d.cfg.ChainId,
		Accounts:    accounts,
		UnfundedKey: core.UnfundedKey,
	}

	// The contents are handed to the builder unsigned, because the builder signs them itself once it
	// has stamped the marker on them: a signature made here would either be thrown away or, when the
	// spec asked for the wrong chain, refused by the builder's own signer. So a signing defect drawn
	// for one of them is not what the transaction will carry, and it is the spec that is corrected
	// rather than the prediction -- a spec that lies about what it became is a model nobody can trust.
	for _, spec := range envelope.Contents {
		spec.(core.SignedElsewhere).SetSigning(core.SignCorrect)
	}

	envelope.Contents = d.dropOffending(envelope.Contents, baseFee) // < bundles/skip.go
	if len(envelope.Contents) == 0 {
		return nil, nil // nothing left to carry, so there is no bundle and no envelope to inject
	}

	// The contents are planned as the batch they are, by the model that plans every other batch: that
	// is what gives each one its nonce and decides, with the whole set in view, which of them the
	// sender's sequence admits.
	plans, refused, reason := core.PlanBatch(envelope.Contents, d.inner, baseFee, d.pricingCfg())
	if refused {
		d.refused[reason]++
	}

	contents := make([]core.TxSpec, 0, len(envelope.Contents))
	kept := make([]core.TxPlan, 0, len(envelope.Contents))
	steps := make([]bundle.BuilderStep, 0, len(envelope.Contents))
	for i, spec := range envelope.Contents {
		txData, err := spec.TxData(plans[i].Nonce, buildCtx)
		if err != nil {
			continue // not representable at all, which is a harness limit rather than a finding
		}
		contents = append(contents, spec)
		kept = append(kept, plans[i])
		steps = append(steps, bundle.Step(buildCtx.Sender(spec).PrivateKey, markable(txData)))
	}
	if len(steps) == 0 {
		return nil, nil // nothing to carry, so there is no bundle and no envelope to inject
	}
	envelope.Contents, plans = contents, kept

	builder := bundle.NewBuilder().WithSigner(signer)
	switch envelope.Range {
	case RangeCovering:
		builder = builder.SetEarliest(block)
	case RangeBefore:
		// A range that ended before the block the bundle can land in. Block numbers only grow, so
		// starting at zero with a length that stops short of the current block is late for good.
		builder = builder.SetEarliest(0).SetRangeLength(max(block, 1))
	case RangeAfter:
		builder = builder.SetEarliest(block + blocksBeyondReach)
	}
	if envelope.Root == RootSingle {
		envelope.Contents, plans = envelope.Contents[:1], plans[:1]
	}
	if envelope.Root == RootSingle && !cannotFail(envelope.Contents[0], d.stateOf(envelope.Contents[0]), baseFee) {
		envelope.Root = RootAllOf // defect3, see bundles/skip.go
		d.demoted++
	}
	if envelope.Root == RootSingle {
		builder = builder.With(steps[0])
	} else {
		builder = builder.AllOf(steps[:len(envelope.Contents)]...)
	}

	envelopeTx, txBundle, plan := builder.BuildEnvelopeBundleAndPlan()
	payload := envelopeTx.Data()

	gasLimit := envelopeTx.Gas()
	switch envelope.Declared {
	case GasTooLow:
		gasLimit--
	case GasTooHigh:
		gasLimit++
	}

	return &Built{
		Payload:  payload,
		GasLimit: gasLimit,
		PlanHash: plan.Hash(),
		Plan:     plan,
		Inner:    txBundle.GetTransactionsInReferencedOrder(),
		Plans:    plans,
		Refused:  refused,
	}, nil
}

// blocksBeyondReach is how far past the current block a range must start to be out of reach for the
// blocks this iteration produces, which is a handful.
const blocksBeyondReach = 1000

// Extra reports the transactions of every bundle that executed. They reached a block without having
// been injected -- the envelope carrying them is what was injected, and that never enters a block --
// so this is the only way the accounting learns of them.
func (d *Domain) Extra(rt *rapid.T, ctx context.Context) ([]core.ExtraExecuted, error) {
	extra, err := d.Inner.Extra(rt, ctx)
	if err != nil {
		return nil, err
	}

	for _, envelope := range d.envelopes {
		if envelope.Built == nil {
			continue
		}
		for i, tx := range envelope.Built.Inner {
			receipt, err := d.client.TransactionReceipt(ctx, tx.Hash())
			if err == ethereum.NotFound {
				continue // this bundle did not execute, so nothing of it is accounted for
			}
			if err != nil {
				return nil, fmt.Errorf("failed to query the receipt of %v: %w", tx.Hash(), err)
			}

			window := envelope.Contents[i].Sender() % len(d.contents)
			extra = append(extra, core.ExtraExecuted{
				SenderIdx: d.Offset + window,
				Tx: core.ExecutedTx{
					Receipt:        receipt,
					Value:          tx.Value(),
					TransfersValue: transfersValue(tx, receipt, d.contents[window]),
					BlobFee:        core.BlobFee(envelope.Contents[i]),
				},
			})
		}
	}
	return extra, nil
}

// Check verifies what only this domain can: that a bundle is all or nothing, and that what the chain
// says it did agrees with what the model expected of it.
func (d *Domain) Check(rt *rapid.T, ctx context.Context, obs core.Observation) error {
	if err := d.Inner.Check(rt, ctx, obs); err != nil {
		return err
	}
	for _, envelope := range d.envelopes {
		if err := d.checkBundle(ctx, envelope, obs); err != nil {
			return err
		}
	}
	return nil
}

// String renders what the run saw of bundles, which is what says whether the bundle path was reached
// at all rather than merely not violated.
func (d *Domain) String() string {
	total := 0
	for _, count := range d.observed {
		total += count
	}
	if total == 0 {
		return "no bundles observed"
	}

	var out strings.Builder
	fmt.Fprintf(&out, "%d bundles:", total)
	for _, fate := range []Fate{FateExecuted, FateSkipped} {
		count := d.observed[fate]
		fmt.Fprintf(&out, " %s=%d (%d%%)", fate, count, count*100/total)
	}

	for _, reason := range byCount(d.reasons) {
		fmt.Fprintf(&out, "\n    %4d %s", d.reasons[reason], reason)
	}
	for _, reason := range byCount(d.refused) {
		fmt.Fprintf(&out, "\n    %4d contents refused before the bundle was opened: %s",
			d.refused[reason], reason)
	}
	for _, reason := range byCount(d.skipped) {
		fmt.Fprintf(&out, "\n    %4d contents kept out of their bundle: %s", d.skipped[reason], reason)
	}
	if d.demoted > 0 {
		fmt.Fprintf(&out,
			"\n    %4d bare roots turned into AllOf(A) groups, steering around the defect Notes reports",
			d.demoted)
	}
	return out.String()
}

// byCount orders the keys of a tally by how often each was seen, the commonest first.
func byCount(tally map[string]int) []string {
	return slices.SortedFunc(maps.Keys(tally), func(a, b string) int {
		return cmp.Or(cmp.Compare(tally[b], tally[a]), cmp.Compare(a, b))
	})
}

// transfersValue reports whether an inner transaction's value left its sender, on the same terms as
// the runner applies to an injected one.
func transfersValue(tx *types.Transaction, receipt *types.Receipt, sender core.PooledAccount) bool {
	return receipt.Status == types.ReceiptStatusSuccessful &&
		(tx.To() == nil || *tx.To() != sender.Address())
}

// markable is a payload the bundle-only marker can be put on. A legacy transaction has no access list
// to hold one, so it becomes an access-list transaction carrying the same call -- which is what
// bundle.Step does for a signed transaction, and what the builder's own gas adjustment assumes.
func markable(data types.TxData) types.TxData {
	legacy, ok := data.(*types.LegacyTx)
	if !ok {
		return data
	}
	return &types.AccessListTx{
		Nonce:    legacy.Nonce,
		GasPrice: legacy.GasPrice,
		Gas:      legacy.Gas,
		To:       legacy.To,
		Value:    legacy.Value,
		Data:     legacy.Data,
	}
}

// stateOf is the state of the account behind one transaction of a bundle, as Prepare read it.
func (d *Domain) stateOf(spec core.TxSpec) core.SenderState {
	return d.inner[spec.Sender()%len(d.inner)]
}
