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

package subsidies

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/tests/transaction_properties/core"
	"github.com/ethereum/go-ethereum/common"
	"pgregory.net/rapid"
)

// FundLevel is how well the iteration's chosen fund is stocked, drawn per batch. The three levels are
// what the coverage bands need: clearly short, clearly ample, and deliberately in between, where the
// model may only say the request either executed or did not.
type FundLevel uint8

const (
	// FundNothing pays nothing in, leaving the fund at whatever earlier iterations left it.
	FundNothing FundLevel = iota

	// FundExactly pays in what this batch's requests ask of it, no more.
	FundExactly

	// FundAmply pays in several times that, so coverage is beyond doubt.
	FundAmply
)

func (l FundLevel) String() string {
	switch l {
	case FundNothing:
		return "nothing"
	case FundExactly:
		return "exactly what the batch needs"
	case FundAmply:
		return "amply"
	}
	return "unknown"
}

// amplyOver is the factor FundAmply pays over what the batch asks for. It is above the four the
// coverage band uses, so an amply funded batch sits outside the band even after the base fee has
// moved between the top-up and the block.
const amplyOver = 8

// Domain layers sponsorship over another domain: it draws that domain's batch and turns part of it
// into sponsorship requests, replaces the pricing rules for those, funds one sender's sponsorship
// fund per iteration, and checks the follow-up transactions and the funds afterwards.
type Domain struct {
	// Inner is the domain whose transactions are sponsored, and whose rules still decide everything
	// that pays its own way.
	Inner core.Domain

	// Mode is how the installed registry answers, and so who pays.
	Mode Mode

	// Config is what the registry charges, read from the chain when the domain is built.
	Config GasConfig

	registry *Registry
	client   *tests.PooledEhtClient

	// What the last Prepare established about the iteration now running. rapid runs iterations one
	// at a time, and every one of them calls Prepare before anything reads these.
	senders  int
	coverage []coverage
	funds    map[common.Address]*big.Int

	// paid and charged are what the whole run put into each fund and what the follow-ups took out of
	// it, which is what the conservation check compares against the balance once the run is over and
	// the archive can answer for the head again.
	paid    map[common.Address]*big.Int
	charged map[common.Address]*big.Int

	// What the run has seen of sponsorship requests, so a run that sponsored nothing is visible
	// rather than merely green.
	observed map[core.Outcome]int
}

// New builds the domain over a session whose network has gas subsidies enabled and the registry of
// the given mode installed. It reads the registry's gas configuration once, and verifies that no fund
// other than the account funds it pays into holds anything.
func New(
	session tests.IntegrationTestNetSession,
	client *tests.PooledEhtClient,
	network *core.Network,
	inner core.Domain,
	mode Mode,
) (*Domain, error) {

	registry, err := NewRegistry(session, client, network)
	if err != nil {
		return nil, err
	}
	config, err := registry.GasConfig()
	if err != nil {
		return nil, err
	}
	// Only a fund-backed registry is asked about funds, and only it implements the queries: the
	// network-sponsored registries carry no funds and answer nothing but the four functions the
	// client calls.
	if mode == ModeFundBacked {
		if err := registry.EmptyFundsElsewhere(context.Background()); err != nil {
			return nil, err
		}
	}

	return &Domain{
		Inner:    inner,
		Mode:     mode,
		Config:   config,
		registry: registry,
		client:   client,
		funds:    map[common.Address]*big.Int{},
		paid:     map[common.Address]*big.Int{},
		charged:  map[common.Address]*big.Int{},
		observed: map[core.Outcome]int{},
	}, nil
}

// Batch draws the inner domain's transactions and sponsors part of them.
func (d *Domain) Batch() *rapid.Generator[[]core.TxSpec] {
	return Sponsoring(d.Inner.Batch(), d.unsponsorable)
}

// Skip asks the domain being wrapped: sponsorship changes who pays for a transaction, not which
// transactions have to be steered around. What this domain must not have sponsored is kept out by the
// generator instead -- see unsponsorable -- because a transaction paying its own way is worth drawing
// even where a request of the same shape is not.
func (d *Domain) Skip(spec core.TxSpec, sender core.SenderState, baseFee *big.Int) string {
	return d.Inner.Skip(spec, sender, baseFee)
}

// unsponsorable reports what this domain will not ask to have sponsored.
func (d *Domain) unsponsorable(spec core.TxSpec) bool {
	// Known defect [4]: chooseFund packs the value of a request into 32 bytes, which panics on a value
	// too wide to fit. Such a transaction is still worth injecting, so it is only kept from asking to
	// be sponsored. See Notes.
	if spec.Amount().BitLen() > 256 {
		return true
	}

	// A registry that sponsors every sender sponsors a stranger too, so a signature recovering to an
	// address holding nothing executes rather than being refused for want of gas. It then moves an
	// account nobody here claimed, which is beyond what this harness observes: the accounting watches
	// the pooled sender the spec names, and the nonce that transaction spends is the stranger's. A
	// fund-backed registry has no such case -- no fund covers a stranger -- so it keeps them.
	return d.Mode != ModeFundBacked && spec.SigningMode().RecoversToStranger()
}

// Pricing prices a sponsorship request from its fund and everything else by the inner domain's rules.
func (d *Domain) Pricing() core.PricingRules {
	return d.PricingRules(d.Inner.Pricing())
}

// Prepare funds one sender's sponsorship fund for the batch about to be injected, then reads back
// what every claimed sender's fund holds, which is what the coverage rules and the fund accounting
// both need. The fund level and the sender to pay for are drawn here rather than in the generator
// because they say what to do to the chain, not what to inject: the generator stays pure, and rapid
// still records and shrinks both draws.
func (d *Domain) Prepare(
	rt *rapid.T,
	ctx context.Context,
	accounts []core.PooledAccount,
	specs []core.TxSpec,
) error {

	if err := d.Inner.Prepare(rt, ctx, accounts, specs); err != nil {
		return err
	}

	d.senders = len(accounts)
	d.coverage = make([]coverage, len(accounts))
	d.funds = make(map[common.Address]*big.Int, len(accounts))

	requiredGas := make([]uint64, len(accounts))
	for _, spec := range requests(specs) {
		index := spec.Sender() % len(accounts)
		requiredGas[index] = core.SaturatingAdd(requiredGas[index],
			core.SaturatingAdd(spec.Gas(), d.Config.MaxOverhead()))
	}

	// Modes 2 and 3 need no fund at all, so no block is spent on one.
	if d.Mode == ModeFundBacked {
		baseFee, err := d.baseFee(ctx)
		if err != nil {
			return err
		}
		donations := make([]Donation, 0, len(accounts))
		for i, account := range accounts {
			if requiredGas[i] == 0 {
				continue // nothing of this sender's asks to be sponsored
			}
			amount := donation(fundLevels.Draw(rt, label("fundLevel", i)), requiredGas[i], baseFee)
			if amount.Sign() == 0 {
				continue // the unfunded case, which costs nothing at all
			}
			donations = append(donations, Donation{Account: account.Address(), Amount: amount})
			add(d.paid, account.Address(), amount)
		}
		if _, err := d.registry.TopUp(ctx, donations); err != nil {
			return err
		}
	}

	// What each fund holds is what this run paid into it less what the follow-ups took out, and both
	// of those are the chain's own numbers: a payment confirmed by its receipt, a charge parsed out of
	// the follow-up transaction that made it. It is not read back from the chain here, because reading
	// it means a call, a call resolves state through the archive, and the archive stops advancing on a
	// fork that reproduces known defect 2 -- where the state no longer follows from the blocks it is
	// derived from. CheckFunds compares this ledger against the chain once, at the end, where a fork
	// that can be asked at all will answer.
	for i, account := range accounts {
		funds := d.held(account.Address())
		d.funds[account.Address()] = funds
		d.coverage[i] = coverage{funds: funds, requiredGas: requiredGas[i]}
	}
	return nil
}

// fundLevels are the levels a fund is stocked to, weighted towards covering the batch: a run whose
// requests are mostly unfunded spends its iterations on the one answer that needs no fund at all.
// The ample level comes first because rapid's index draw favours the front of the list, so the order
// is what makes the weighting land where the comment says -- see the note on the shared generators in
// core/generators.go.
var fundLevels = rapid.SampledFrom([]FundLevel{FundAmply, FundAmply, FundExactly, FundNothing})

// donation is what to pay into a fund, priced at the base fee the requests will be judged at.
func donation(level FundLevel, requiredGas uint64, baseFee *big.Int) *big.Int {
	if level == FundNothing {
		return new(big.Int)
	}
	required := new(big.Int).Mul(new(big.Int).SetUint64(requiredGas), baseFee)
	if level == FundAmply {
		return required.Mul(required, big.NewInt(amplyOver))
	}
	return required
}

// baseFee reads the base fee of the latest block, which is what a fee a fund must cover is priced at.
func (d *Domain) baseFee(ctx context.Context) (*big.Int, error) {
	header, err := d.client.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to read the latest header: %w", err)
	}
	if header.BaseFee == nil {
		return nil, fmt.Errorf("the latest header carries no base fee")
	}
	return header.BaseFee, nil
}

// Extra reports nothing: the follow-up transactions a sponsorship appends are internal ones from the
// zero address, which is no account this harness watches, and the sponsored transaction itself is
// injected like any other.
func (d *Domain) Extra(rt *rapid.T, ctx context.Context) ([]core.ExtraExecuted, error) {
	return d.Inner.Extra(rt, ctx)
}

// Check verifies the follow-up transactions the sponsorships called for and that the funds fell by
// exactly what those follow-ups charged.
func (d *Domain) Check(rt *rapid.T, ctx context.Context, obs core.Observation) error {
	if err := d.Inner.Check(rt, ctx, obs); err != nil {
		return err
	}

	for _, tx := range obs.Txs {
		if IsSponsorshipRequest(tx.Spec) {
			d.observed[tx.Outcome]++
		}
	}

	charged, err := d.CheckPostTransactions(obs)
	if err != nil {
		return err
	}
	for account, amount := range charged {
		add(d.charged, account, amount)
	}
	return nil
}

// coverageOf is what the model knows about the fund backing a spec's sender this iteration.
func (d *Domain) coverageOf(spec core.TxSpec) coverage {
	if d.senders == 0 {
		return coverage{funds: new(big.Int)}
	}
	return d.coverage[spec.Sender()%d.senders]
}

// String renders what the run saw of sponsorship requests, which is what says whether the sponsored
// path was reached at all rather than merely not violated.
func (d *Domain) String() string {
	total := 0
	for _, count := range d.observed {
		total += count
	}
	if total == 0 {
		return "no sponsorship requests observed"
	}

	var out strings.Builder
	fmt.Fprintf(&out, "%d sponsorship requests (%s):", total, d.Mode)
	for _, outcome := range []core.Outcome{
		core.OutcomeEventRejected, core.OutcomeDropped, core.OutcomeExecuted,
	} {
		count := d.observed[outcome]
		fmt.Fprintf(&out, " %s=%d (%d%%)", outcome, count, count*100/total)
	}
	return out.String()
}

// CheckFunds is the fund half of the accounting, checked once the run is over: every fund holds
// exactly what the run paid into it less what the follow-ups took out. No wei may leave a fund
// unaccounted for, and none may appear in one.
//
// It is one check at the end rather than one per iteration because a fund balance can only be read
// from the archive, and the archive is hundreds of blocks behind while the run is hot. Idle, it
// catches up in under a millisecond.
func (d *Domain) CheckFunds(ctx context.Context) error {
	if d.Mode != ModeFundBacked {
		return nil // no fund is charged, and nothing was paid into one
	}

	for account, paid := range d.paid {
		held, err := d.registry.FundsOf(ctx, account, 0)
		if err != nil {
			return err
		}

		charged := d.charged[account]
		if charged == nil {
			charged = new(big.Int)
		}
		expected := new(big.Int).Sub(paid, charged)
		if held.Cmp(expected) != 0 {
			return fmt.Errorf(
				"the fund of %v was paid %v and charged %v, so it should hold %v, but it holds %v "+
					"(off by %v)",
				account, paid, charged, expected, held, new(big.Int).Sub(held, expected))
		}
	}
	return nil
}

// add accumulates an amount per account.
func add(totals map[common.Address]*big.Int, account common.Address, amount *big.Int) {
	total, ok := totals[account]
	if !ok {
		total = new(big.Int)
		totals[account] = total
	}
	total.Add(total, amount)
}

// held is what a fund holds by the run's own reckoning: paid in, less taken out.
func (d *Domain) held(account common.Address) *big.Int {
	held := new(big.Int)
	if paid, ok := d.paid[account]; ok {
		held.Set(paid)
	}
	if charged, ok := d.charged[account]; ok {
		held.Sub(held, charged)
	}
	return held
}
