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

package core

import (
	"context"
	"crypto/ecdsa"
	"flag"
	"fmt"
	"maps"
	"math/big"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// Domain is what one family of transactions contributes to an iteration: what to draw, how the
// model prices it, what has to be arranged on chain before it is injected, and what to check
// afterwards that no other domain knows about. Everything a family knows about itself lives behind
// this interface, so the generator, the model and the invariants here stay free of it.
type Domain interface {
	// Batch draws the transactions of one iteration.
	Batch() *rapid.Generator[[]TxSpec]

	// Pricing is the tail of the model's rule table, deciding whether a transaction can pay its
	// way. It is taken from the domain rather than fixed here so that the rules and the draws
	// cannot disagree about what was generated.
	Pricing() PricingRules

	// Prepare arranges whatever the drawn batch needs on chain before it is injected. It may
	// produce blocks, so the base fee is read after it returns.
	Prepare(rt *rapid.T, ctx context.Context, accounts []PooledAccount, specs []TxSpec) error

	// Skip reports why a drawn transaction must not be injected, or "" when it may be. It is where a
	// domain names what provokes a defect the client has not fixed, and it is asked once per drawn
	// transaction, with the sender's state and the base fee the batch will be judged against. The
	// policies live in each domain's skip.go; core/skip.go is what applies them.
	Skip(spec TxSpec, sender SenderState, baseFee *big.Int) string

	// Extra reports transactions that executed without having been injected, so that the accounting
	// can account for them rather than see an account move for no reason it knows of. A bundle's
	// inner transactions are the case it exists for: each one gets a receipt of its own while the
	// envelope carrying them gets none. Usually empty.
	//
	// It is called once the injected batch has been confirmed, so the receipts it looks up are there.
	Extra(rt *rapid.T, ctx context.Context) ([]ExtraExecuted, error)

	// Check verifies what only this domain can: the accounting and block invariants have already
	// run when it is called.
	Check(rt *rapid.T, ctx context.Context, obs Observation) error

	// Notes are one line per defect this domain skips the input for, logged by every run so that a
	// workaround cannot outlive the defect it was written for. Usually none.
	Notes() []string
}

// ExtraExecuted is one transaction that executed without having been injected, and which of the
// accounts this iteration claimed sent it.
type ExtraExecuted struct {
	SenderIdx int
	Tx        ExecutedTx
}

// Observation is what an iteration saw, handed to a domain's own checks. What a transaction was
// charged is priced at the base fee of the block it landed in, which is why only the blocks are here.
type Observation struct {
	Blocks []*types.Block
	Txs    []TxObservation
}

// TxObservation is the fate of one injected transaction. Receipt is nil unless Outcome is
// OutcomeExecuted.
type TxObservation struct {
	Spec    TxSpec
	Hash    common.Hash
	Nonce   uint64
	Outcome Outcome
	Receipt *types.Receipt
	Sender  PooledAccount
}

// Executed lists the transactions of an observation that produced a receipt.
func (o Observation) Executed() []TxObservation {
	out := make([]TxObservation, 0, len(o.Txs))
	for _, tx := range o.Txs {
		if tx.Outcome == OutcomeExecuted {
			out = append(out, tx)
		}
	}
	return out
}

// Network bundles a running test network with the account pool funded in its genesis and the
// configuration the model needs.
type Network struct {
	Net  *tests.IntegrationTestNet
	Pool *AccountPool
	Cfg  NetworkConfig
}

// StartNetwork starts a network running the given upgrades and stops it when the test that asked for
// it ends, so one fork's nodes and databases are gone before the next fork's are started. Every
// iteration of that fork's run shares it; none can disturb another, because each works on accounts
// nobody else holds.
func StartNetwork(t *testing.T, upgrades opera.Upgrades) *Network {
	t.Helper()

	pool, err := NewAccountPool(MaxGenesisAccounts)
	require.NoError(t, err, "failed to derive the account pool")

	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
		Upgrades: &upgrades,
		Accounts: pool.GenesisAccounts(),
	})

	t.Cleanup(func() {
		handedOut, dirtied, untouched := pool.Stats()
		t.Logf("account pool: %d handed out, %d dirtied by execution, %d still untouched",
			handedOut, dirtied, untouched)
	})

	rules := tests.GetNetworkRules(t, net)
	return &Network{
		Net:  net,
		Pool: pool,
		Cfg: NetworkConfig{
			ChainId:     net.GetChainId(),
			Upgrades:    upgrades,
			MaxEventGas: rules.Economy.Gas.MaxEventGas,
			MaxBlockGas: rules.Blocks.MaxBlockGas,
			MaxTxType:   MaxTxTypeFor(upgrades),
		},
	}
}

// MaxTxTypeFor is the highest transaction type a fork accepts, mirroring epochcheck.CheckTxs.
func MaxTxTypeFor(upgrades opera.Upgrades) uint8 {
	switch {
	case upgrades.Allegro:
		return types.SetCodeTxType
	case upgrades.Sonic:
		return types.BlobTxType
	case upgrades.London:
		return types.DynamicFeeTxType
	case upgrades.Berlin:
		return types.AccessListTxType
	}
	return types.LegacyTxType
}

// Runner holds everything one fork's property test works with, and tallies what its iterations
// observed.
type Runner struct {
	Session tests.IntegrationTestNetSession
	Client  *tests.PooledEhtClient
	Network *Network
	Cfg     GenConfig
	Domain  Domain

	batch      *rapid.Generator[[]TxSpec]
	firstBlock uint64
	observed   map[Outcome]int
	skipped    map[string]int
	defects    *DefectLog
}

// Init prepares a runner for rapid.Check: it takes the domain's generator and pricing rules,
// verifies the assumptions the model makes about the network's economics -- so a change to either
// produces a clear message here rather than a puzzling failure later -- and records where the run
// starts, for the replay check.
func (r *Runner) Init(t *testing.T) {
	t.Helper()

	r.batch = r.Domain.Batch()
	r.Network.Cfg.Pricing = r.Domain.Pricing()
	r.observed = map[Outcome]int{}
	r.skipped = map[string]int{}
	r.defects = &DefectLog{}

	baseFee, err := r.baseFee(t.Context())
	require.NoError(t, err)

	require.Positive(t, MinimumViableFeeCap.Cmp(Scale(baseFee, 4, 1)),
		"MinimumViableFeeCap %v is not comfortably above the base fee %v",
		MinimumViableFeeCap, baseFee)

	dearest := new(big.Int).Mul(
		gas(r.Cfg.GasBudget), new(big.Int).Mul(MinimumViableFeeCap, big.NewInt(10)),
	)
	require.Positive(t, AccountBalance.Cmp(dearest),
		"the pooled balance %v cannot back the most expensive affordable transaction (%v)",
		AccountBalance, dearest)

	r.firstBlock, err = r.Client.BlockNumber(t.Context())
	require.NoError(t, err)
}

// Run is one property iteration: draw a batch, inject it, and check the result against the model and
// the invariants.
func (r *Runner) Run(rt *rapid.T) {
	ctx := rt.Context()
	specs := r.batch.Draw(rt, "batch")

	accounts, err := r.Network.Pool.Claim(r.Cfg.MaxAccountsPerBatch)
	require.NoError(rt, err)

	dirty := make([]bool, len(accounts))
	defer func() {
		for i, account := range accounts {
			r.Network.Pool.Release(account, dirty[i])
		}
	}()

	require.NoError(rt, r.Domain.Prepare(rt, ctx, accounts, specs),
		"the domain could not prepare the chain for this batch")

	before, err := r.senderStates(ctx, accounts)
	require.NoError(rt, err)

	baseFee, err := r.baseFee(ctx)
	require.NoError(rt, err)

	specs = r.dropOffending(specs, before, baseFee) // < core/skip.go
	if len(specs) == 0 {
		rt.Skip("every transaction of the batch provokes a defect the domain can only avoid")
	}

	plans, batchRejected, batchReason := PlanBatch(specs, before, baseFee, r.Network.Cfg)

	buildCtx := BuildContext{
		ChainId:     r.Network.Cfg.ChainId,
		Accounts:    accounts,
		UnfundedKey: UnfundedKey,
	}
	txs := make([]*types.Transaction, 0, len(specs))
	built := make([]TxSpec, 0, len(specs))
	kept := make([]TxPlan, 0, len(specs))
	for i, spec := range specs {
		tx, err := BuildTx(spec, plans[i].Nonce, buildCtx)
		if err != nil {
			continue
		}
		txs = append(txs, tx)
		built = append(built, spec)
		kept = append(kept, plans[i])
	}
	if len(txs) == 0 {
		rt.Skip("no transaction in the batch could be built")
	}
	specs, plans = built, kept

	for i, tx := range txs {
		if _, err := r.Client.TransactionReceipt(ctx, tx.Hash()); err == nil {
			rt.Skipf("transaction %d is identical to one already executed on this chain", i)
		}
	}

	hashes, injectErr := r.Session.ForceEmitAll(ctx, txs)

	if batchRejected {
		if injectErr == nil {
			rt.Fatalf("batch should have been rejected (%s) but was accepted\n%s",
				batchReason, Describe(specs, plans))
		}
		r.observed[OutcomeEventRejected] += len(specs)
		r.requireUnchanged(rt, ctx, accounts, before, specs, plans)
		return
	}
	if injectErr != nil {
		rt.Fatalf("batch should have been accepted but was rejected with %q\n%s",
			injectErr, Describe(specs, plans))
	}

	result, err := ConfirmOutcomes(ctx, r.Session, r.Client, hashes)
	require.NoError(rt, err)

	expectations := make([]AccountExpectation, len(accounts))
	for i := range expectations {
		expectations[i].maxUnreceiptedCharge = new(big.Int)
	}
	observation := Observation{Txs: make([]TxObservation, len(specs))}

	for i, spec := range specs {
		outcome := result.OutcomeOf(hashes[i])
		r.observed[outcome]++

		if !plans[i].Prediction.Permits(outcome) {
			rt.Fatalf("transaction %d ended as %s but the model allows %s\n%s",
				i, outcome, plans[i].Prediction, Describe(specs, plans))
		}

		senderIdx := spec.Sender() % len(accounts)
		observation.Txs[i] = TxObservation{
			Spec:    spec,
			Hash:    hashes[i],
			Nonce:   plans[i].Nonce,
			Outcome: outcome,
			Receipt: result.Receipts[hashes[i]],
			Sender:  accounts[senderIdx],
		}

		expectation := &expectations[senderIdx]
		if outcome == OutcomeExecuted {
			receipt := result.Receipts[hashes[i]]
			expectation.executed = append(expectation.executed, ExecutedTx{
				Receipt:        receipt,
				Value:          txs[i].Value(),
				TransfersValue: transfersValue(txs[i], receipt, accounts[senderIdx].Address()),
				BlobFee:        BlobFee(spec),
			})
			continue
		}

		require.NoError(rt, CheckAbsent(ctx, r.Client, hashes[i]),
			"transaction %d\n%s", i, Describe(specs, plans))

		// Known defect 2: pre-Allegro, a transaction the state processor cannot apply keeps the gas
		// it bought even though it never reaches a block.
		if !r.Network.Cfg.Upgrades.Allegro && spec.SigningMode() == SignCorrect {
			expectation.maxUnreceiptedCharge.Add(
				expectation.maxUnreceiptedCharge, GasCost(spec, Scale(baseFee, 4, 1)),
			)
		}
	}

	r.checkNonceRivalry(rt, specs, plans, hashes, result, len(accounts))

	// Whatever executed without being injected is accounted for by the same conservation check as the
	// rest, which is the point of asking: a domain that hides a transaction from it hides the only
	// oracle that would notice the wrong account paying.
	extra, err := r.Domain.Extra(rt, ctx)
	require.NoError(rt, err, "the domain could not report what executed outside the batch")
	for _, executed := range extra {
		if executed.SenderIdx < 0 || executed.SenderIdx >= len(accounts) {
			rt.Fatalf("the domain reported a transaction from sender %d, outside the %d claimed",
				executed.SenderIdx, len(accounts))
		}
		expectation := &expectations[executed.SenderIdx]
		expectation.executed = append(expectation.executed, executed.Tx)
	}

	after, err := r.senderStates(ctx, accounts)
	require.NoError(rt, err)

	for i, account := range accounts {
		change := StateChange{
			nonceBefore:   before[i].Nonce,
			nonceAfter:    after[i].Nonce,
			balanceBefore: before[i].Balance,
			balanceAfter:  after[i].Balance,
		}
		seen, err := CheckAccounting(account.Address(), change, expectations[i])
		if err != nil {
			rt.Fatalf("%v\n%s", err, Describe(specs, plans))
		}

		if seen.unreceiptedCharge.Sign() > 0 {
			r.defects.UnreceiptedCharge(account.Address(), seen.unreceiptedCharge)
		}
		dirty[i] = len(expectations[i].executed) > 0 ||
			change.nonceBefore != change.nonceAfter ||
			change.balanceBefore.Cmp(change.balanceAfter) != 0
	}

	blocks, err := r.blocks(ctx,
		result.MarkerBlocks[0]-1, result.MarkerBlocks[len(result.MarkerBlocks)-1])
	require.NoError(rt, err)
	if err := CheckBlockInvariants(ctx, r.Client, blocks); err != nil {
		rt.Fatalf("%v\n%s", err, Describe(specs, plans))
	}

	observation.Blocks = blocks
	if err := r.Domain.Check(rt, ctx, observation); err != nil {
		rt.Fatalf("%v\n%s", err, Describe(specs, plans))
	}
}

// Report logs what the run observed, and reports every known defect it reproduced along with every
// one the domain under test had to avoid.
func (r *Runner) Report(t *testing.T, fork string) {
	t.Helper()
	t.Logf("outcome distribution: %s", r)
	r.defects.Note(r.Domain.Notes()...)
	r.defects.Report(t, fork)
}

// checkNonceRivalry asserts that no two transactions spent the same nonce, which the accounting
// cannot see on its own: it only reads where the sender's nonce ended up, not how many transactions
// claimed each step along the way.
func (r *Runner) checkNonceRivalry(
	rt *rapid.T,
	specs []TxSpec,
	plans []TxPlan,
	hashes []common.Hash,
	result Outcomes,
	senderCount int,
) {
	type nonceKey struct {
		sender int
		Nonce  uint64
	}
	executed := map[nonceKey]int{}

	for i, spec := range specs {
		if result.OutcomeOf(hashes[i]) != OutcomeExecuted {
			continue
		}
		executed[nonceKey{spec.Sender() % senderCount, plans[i].Nonce}]++
	}

	for key, count := range executed {
		if count > 1 {
			rt.Fatalf(
				"%d transactions on sender %d executed with nonce %d, so at least two spent the "+
					"same nonce\n%s",
				count, key.sender, key.Nonce, Describe(specs, plans),
			)
		}
	}
}

// requireUnchanged asserts that a rejected batch changed nothing anywhere.
func (r *Runner) requireUnchanged(
	rt *rapid.T,
	ctx context.Context,
	accounts []PooledAccount,
	before []SenderState,
	specs []TxSpec,
	plans []TxPlan,
) {
	after, err := r.senderStates(ctx, accounts)
	require.NoError(rt, err)

	for i, account := range accounts {
		change := StateChange{
			nonceBefore:   before[i].Nonce,
			nonceAfter:    after[i].Nonce,
			balanceBefore: before[i].Balance,
			balanceAfter:  after[i].Balance,
		}
		if _, err := CheckAccounting(account.Address(), change, AccountExpectation{}); err != nil {
			rt.Fatalf("a rejected batch must change nothing: %v\n%s", err, Describe(specs, plans))
		}
	}
}

// senderStates reads the on-chain nonce and balance of every account.
func (r *Runner) senderStates(
	ctx context.Context,
	accounts []PooledAccount,
) ([]SenderState, error) {
	states := make([]SenderState, len(accounts))
	for i, account := range accounts {
		nonce, err := r.Client.NonceAt(ctx, account.Address(), nil)
		if err != nil {
			return nil, fmt.Errorf("failed to read the nonce of %v: %w", account.Address(), err)
		}
		balance, err := r.Client.BalanceAt(ctx, account.Address(), nil)
		if err != nil {
			return nil, fmt.Errorf("failed to read the balance of %v: %w", account.Address(), err)
		}
		states[i] = SenderState{Nonce: nonce, Balance: balance}
	}
	return states, nil
}

// baseFee reads the base fee of the latest block, which decides both what a transaction pays per unit
// of gas and the floor its fee cap must clear.
func (r *Runner) baseFee(ctx context.Context) (*big.Int, error) {
	header, err := r.Client.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to read the latest header: %w", err)
	}
	if header.BaseFee == nil {
		return nil, fmt.Errorf("the latest header carries no base fee")
	}
	return header.BaseFee, nil
}

// blocks reads an inclusive range of blocks.
func (r *Runner) blocks(ctx context.Context, from, to uint64) ([]*types.Block, error) {
	blocks := make([]*types.Block, 0, to-from+1)
	for number := from; number <= to; number++ {
		block, err := r.Client.BlockByNumber(ctx, new(big.Int).SetUint64(number))
		if err != nil {
			return nil, fmt.Errorf("failed to read block %d: %w", number, err)
		}
		blocks = append(blocks, block)
	}
	return blocks, nil
}

// String renders the outcome distribution of a whole run.
func (r *Runner) String() string {
	total := 0
	for _, count := range r.observed {
		total += count
	}
	if total == 0 {
		return "no transactions observed"
	}

	var out strings.Builder
	fmt.Fprintf(&out, "%d transactions:", total)
	for _, outcome := range []Outcome{OutcomeEventRejected, OutcomeDropped, OutcomeExecuted} {
		count := r.observed[outcome]
		fmt.Fprintf(&out, " %s=%d (%d%%)", outcome, count, count*100/total)
	}
	for _, reason := range slices.Sorted(maps.Keys(r.skipped)) {
		fmt.Fprintf(&out, "\n    %4d kept out of their batch: %s", r.skipped[reason], reason)
	}
	return out.String()
}

// transfersValue reports whether an executed transaction's value left its sender, which it does only
// when it went elsewhere and execution did not revert. A contract creation does transfer, to the new
// contract.
func transfersValue(tx *types.Transaction, receipt *types.Receipt, sender common.Address) bool {
	return receipt.Status == types.ReceiptStatusSuccessful &&
		(tx.To() == nil || *tx.To() != sender)
}

// VerifyChainReplays replays the chain on an independent state database and requires it to agree with
// the node, which is how correctness is checked rather than mere liveness: a node staying up says
// nothing about whether it computed the right thing, while a replay reproducing every state root and
// block hash from the same inputs says the execution was deterministic and the accounting
// self-consistent.
//
// It is skipped for a run that reproduced known defect 2, where the state no longer follows from the
// blocks alone.
func (r *Runner) VerifyChainReplays(t *testing.T) {
	t.Helper()

	if n := len(r.defects.unreceiptedCharges); n > 0 {
		t.Logf(
			"skipping the replay check: %d account(s) were charged by transactions that "+
				"produced no receipt, so the blocks alone do not determine the state", n)
		return
	}

	genesis := r.Network.Net.GetJsonGenesis()
	require.NotNil(t, genesis, "the network must be started with a JSON genesis")

	head, err := r.Client.BlockNumber(t.Context())
	require.NoError(t, err)
	require.Greater(t, head, r.firstBlock, "the run should have produced blocks to verify")

	blocks := make([]*types.Block, 0, head+1)
	for number := uint64(0); number <= head; number++ {
		block, err := r.Client.BlockByNumber(t.Context(), new(big.Int).SetUint64(number))
		require.NoError(t, err)
		blocks = append(blocks, block)
	}

	tests.VerifyBlocks(t, genesis, blocks)
}

// UnfundedKey signs the SignUnfundedKey case: derived from a fixed seed and never funded, so a
// transaction it signs has a recoverable sender that can pay for nothing.
var UnfundedKey = mustDeriveUnfundedKey()

func mustDeriveUnfundedKey() *ecdsa.PrivateKey {
	key, err := crypto.ToECDSA(crypto.Keccak256([]byte("txprop-unfunded")))
	if err != nil {
		panic(fmt.Sprintf("failed to derive the unfunded key: %v", err))
	}
	return key
}

// Describe renders a batch and its predictions for a failure report.
func Describe(specs []TxSpec, plans []TxPlan) string {
	var out strings.Builder
	out.WriteString("batch:")
	for i, spec := range specs {
		fmt.Fprintf(&out, "\n  [%d] nonce=%d %s\n      expected %s",
			i, plans[i].Nonce, Format(spec), plans[i].Prediction)
	}
	return out.String()
}

// VerbosityEnv is the client's own log level, which StartIntegrationTestNet leaves alone when it is
// already set. A property test silences the nodes by default: every batch it deliberately makes too
// large for one event is logged at ERROR as "Self-event connection failed", hundreds of times a run,
// and the test reports what it found itself. Set it to see the nodes again -- 1 is error, 3 is the
// client's own default.
const VerbosityEnv = "SONIC_VERBOSITY"

// ApplyTestFlags prepares a property test's binary: it silences the nodes unless the environment says
// otherwise, and raises rapid's default number of checks. It is called from TestMain, because rapid
// reads its own environment before any test runs.
func ApplyTestFlags(checksPerScenario string) error {
	if _, set := os.LookupEnv(VerbosityEnv); !set {
		if err := os.Setenv(VerbosityEnv, "0"); err != nil {
			return fmt.Errorf("failed to silence the nodes: %w", err)
		}
	}

	flag.Parse()

	if !isFlagSet("rapid.checks") {
		if err := flag.Set("rapid.checks", checksPerScenario); err != nil {
			return fmt.Errorf("failed to set rapid.checks: %w", err)
		}
	}
	return nil
}

// init keeps rapid from leaving a fail file behind, for every test binary that reaches this harness.
// rapid otherwise writes each counterexample to testdata/rapid/ and replays every file it finds there
// before drawing anything new -- so a failure from an earlier run of a harness that has since changed
// pollutes the next run, and a run that starts by reproducing yesterday's case is not the run that was
// asked for. The seed in the failure message is the durable artifact; feed it back with -rapid.seed.
//
// rapid registers its flags in an init of its own, which runs first because this package imports it,
// and an explicit -rapid.nofailfile=false on the command line still wins because flag.Parse comes
// after.
func init() {
	if err := flag.Set("rapid.nofailfile", "true"); err != nil {
		panic(fmt.Sprintf("failed to turn off rapid's fail files: %v", err))
	}
}

func isFlagSet(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		found = found || f.Name == name
	})
	return found
}
