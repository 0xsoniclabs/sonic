# Transaction Priorities: Guarantees

The transaction-priorities feature spans two stages with different trust levels
and guarantees:

- **Emitter / txpool** — a best-effort stage that tries to get prioritized
  transactions into events, and so into blocks, quickly.
- **Block formation** — a deterministic stage that guarantees prioritized
  transactions in a block execute before non-prioritized ones, provided they
  meet all requirements.

In short: the feature guarantees the *order* of prioritized transactions within
a block, but only makes a best effort at *including* them.

## Best-effort: reaching an event, and a block

The emitter sits outside the replicated, consensus-critical code — a validator
is free to choose which transactions its events carry, withholding or reordering
them without being Byzantine — so nothing it does can be a guarantee.

In legacy mode every validator carries transactions in its own events, filling
each in three stages:

1. **My-turn prioritized** — prioritized transactions whose turn is this
   validator's, in priority-then-tip order (not subject to the per-entity cap).
2. **Foreign prioritized (piggyback)** — prioritized transactions whose turn
   belongs to another validator, added eagerly but bounded three ways: at most
   `MaxPiggybackTxsPerEntityPerEvent` per entity, at most **half** the event's
   remaining gas budget for all of them together, and the*finished* event must
   carry at least one of this validator's own transactions, or every eager
   admission is rolled back. An event is therefore never emitted solely to carry
   other validators' prioritized transactions.
3. **My-turn ordinary** — the remaining non-prioritized transactions whose turn
   is this validator's.

The three stages are not separate passes but a single ordering:
`(stage, priority level desc, priority weight desc, effective tip desc,
first-seen asc)`. The turn verdict is computed once, when the ordering is built,
which is why the ordering may only be reused for one second.

Because it is one ordering over sender *frontiers* rather than three passes, the
transactions actually come out **non-monotonically in stage**. Each transaction
keeps the priority it was classified with — unlike block formation, the emitter
never demotes one because an earlier nonce was ordinary — but a sender's
transactions are still taken in nonce order, so a transaction sitting behind an
earlier nonce does not compete at all until that predecessor has been consumed,
and is then promoted ahead of whatever is left. A sender whose nonce 0 is
ordinary and whose nonce 1 is prioritized yields nonce 0 among the ordinary
traffic and nonce 1, still prioritized, after it. So stage 1 does not mean
"offered before everything else" — it means "wins against the frontier it
competes with", and a prioritized transaction can reach the event only after
ordinary ones have already spent gas power and event size, and — if it is a
foreign one — the eager budget too. 

In single-proposer mode only one validator emits transactions at all, and the
turn check is switched off entirely — every candidate counts as this proposer's
turn, so the piggyback bounds and the rollback never engage. That proposer runs a
scheduler over the same priority-ordered candidates — prioritized first, then by
tip, each sender in nonce order — and in place of the turn and piggyback checks it
*trial-runs* each candidate against the block state, keeping those that succeed
and fit the gas and size limits and skipping the rest, until the block is full,
the candidates run out, or a ~100 ms deadline expires. Inclusion is entirely this
proposer's call, and the order it produces is still only a proposal — block
formation re-applies the authoritative reordering on top.

Getting one into a block then means surviving a chain of hops, any of which can
drop it:

- **Into the event.** A transaction is admitted only if it passes the per-tx
  checks: size, gas power, epoch rules, sender conflict, txpool freshness, and
  bundle validity.
- **From event to block.** A block is assembled from the events DAG consensus
  confirms into its frame, and the carrying event may instead be confirmed into
  a later block — which events a block draws from is a consensus outcome the
  emitter does not control.
- **Past the proposer.** In single-proposer mode, inclusion is the proposer's
  call: the authoritative reordering can shuffle the proposed set but cannot add
  a transaction the proposer omitted (turn rotation defends against a censoring
  proposer).
- **Keeping its priority.** The priority the emitter acted on is only a hint,
  classified against current head state and memoized by transaction hash in a
  cache that never expires. Block formation re-classifies against block-start
  state, so a registry change in between can demote the transaction — or promote
  one the emitter had treated as ordinary. The cache's staleness is deliberately
  one-sided: a transaction that has lost its priority keeps the stale hint, which
  only works in its favour, and one that gains priority later is old by then, so
  promoting it in the emitter would no longer help.
- **Through execution.** To avoid hoisting a transaction ahead of an
  earlier-nonce one from the same sender — which would leave it nonce-too-high
  and skipped — block formation instead demotes it: a prioritized transaction
  keeps its priority only while it extends the sender's contiguous run of
  prioritized nonces from the block-start account nonce. So a transaction whose
  earlier nonce is not itself prioritized loses the ordering guarantee (it still
  executes in its base-order position, after that earlier nonce). In
  single-proposer mode it can be dropped even earlier: if it fails the
  proposer's trial-run (e.g. a nonce gap or insufficient balance) it never
  enters the proposal, reaching no event or block at all.

In legacy mode an entity can raise its odds of inclusion by running its own
validator: because the emitter is not consensus-critical, that validator can
emit the entity's prioritized transactions in its own events unconditionally,
removing the hops that depend on other validators — turn ownership, the
piggyback cap, propagation. The only risk that still remains is that the event
might not be included in the next block. This however only improves inclusion
latency, never the amount of priority an entity gets.

## Hard guarantees: ordering within a block

Block formation decides the final order of transactions in a block with a
deterministic reordering. Within a block, every prioritized transaction precedes
every non-prioritized one, in `(level desc, weight desc, txhash asc)` order —
except that each sender's transactions stay in nonce order, so a higher-priority
later nonce never overtakes its own predecessor (which would leave it
nonce-too-high). In single-proposer mode this reordering also overrides the order
the proposer chose.

Two things keep a transaction out of this prefix.

**The per-entity rate limit.** An entity may spend at most
`MaxGasPerEntityPerBlock` gas (charged on the gas *limit*, since the order must be
decided before anything executes) on prioritized transactions per block. Its
transactions are selected in priority order until the next would exceed the
budget; that transaction — and every later nonce of the same sender — is demoted
to its base-order position, while **other senders of the same entity keep
packing** whatever budget is left. The budget is not debited by a transaction that
did not fit, so one oversized transaction blocks only its own sender's tail.

**Nonce reachability.** A prioritized transaction is selected only once its
lower-nonce predecessors from the same sender are selected, so one sitting behind
a non-prioritized or unselected earlier nonce is demoted too. A prioritized nonce
*below* the account nonce cannot execute at all and is skipped without ending the
run. Among several prioritized transactions from one sender sharing a nonce, the
one with the lowest hash keeps its priority and the others are demoted; the run
continues past them.

Because the budget is spent in nonce order, these two rules compose rather than
conflict: the budget is never consumed by a transaction that could only have been
nonce-too-high.

This governs only how a block's transactions are *ordered*, never *which* are
included — that is the best-effort stage's job.

The ordering does have one indirect effect on inclusion: a prioritized
transaction keeps its priority only while it extends its sender's contiguous run
of prioritized nonces from the block-start account nonce. One whose earlier
nonce is not itself prioritized is demoted to its base-order position rather than
being hoisted ahead of that earlier nonce and left nonce-too-high; it still lands
in the block, just without the ordering guarantee.

The guarantee bounds each entity, not their sum. Nothing in the node caps the total
prioritized gas of a block, so how much block space priority consumes in aggregate
is a property of the registry's configuration, not of this code.

## Why these hops cannot be closed

None of the hops above is an implementation gap that more code could remove.
They follow from what block formation can rely on: consensus can only order
transactions **already present** in the confirmed events (legacy) or the
proposal (single-proposer). It cannot conjure a transaction that never arrived,
and getting one there crosses an unreliable, asynchronous, partly adversarial
distributed system.

- **Network transport is unreliable.** Transaction and event propagation can be
  dropped, delayed, duplicated, or reordered. A validator includes only what it
  has received; since delivery cannot be guaranteed, neither can the inclusion
  of any given event — let alone a specific transaction within it.
- **Emission is distributed and rotated.** The originating validator for a
  transaction follows a rotating turn schedule; if it is offline or partitioned,
  inclusion waits for a later turn or the bounded piggyback path. No single node
  can unilaterally force a transaction in.
- **Resources are finite and node-local.** Event gas power, event byte size, and
  block gas all bound what fits, and each node's mempool holds a different
  subset of the pending transactions.
- **State moves.** Priority depends on registry state, which can change between
  the emitter's hint and block formation; only the block-start snapshot is
  authoritative.

So the design confines the uncertain work to the best-effort stage: it can make
prioritized transactions reach a block *sooner*, but the block that results —
and the order within it — is fixed entirely by the authoritative stage.
