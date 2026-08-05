# Transaction Priorities

> Status: **Implemented**, behind the optional `Upgrades.TransactionPriorities`
> feature flag. One item blocks activation on a public network: the registry
> address is still the placeholder `0x7072696f72697479…` (see
> [Configuration & activation](#configuration--activation)).

## Motivation

Today Sonic orders the user transactions of a block in one of two ways:

- **Legacy mode** — transactions are collected from all confirmed events and run
  through the *scrambler* (`gossip/scrambler/tx_scrambler.go`), a deterministic
  shuffle keyed by a salt derived from the set of transaction hashes. The order is
  unpredictable to any individual submitter but reproducible by every validator.
- **Single-proposer mode** — a single chosen proposer pre-selects and pre-orders
  the transactions (via `gossip/emitter/scheduler/`) and ships them in a signed
  `inter.Proposal`.

Neither mode lets the network designate a configurable subset of transactions that
must be scheduled *ahead* of the rest. **Transaction priorities** adds exactly that:
an on-chain registry contract decides, per transaction, whether it is prioritized,
how strongly, and which entity it belongs to. Prioritized transactions are placed
at the front of the block, sorted by importance, and rate-limited per entity.
Everything else keeps its current (randomized / proposer-scheduled) order.

This mirrors the existing **gas-subsidies** mechanism
(`gossip/blockproc/subsidies/`): a governed, upgradeable registry contract queried
by the node during transaction processing. We deliberately reuse that proven
pattern (hand-rolled ABI, length-versioned responses, snapshot-isolated EVM reads,
EIP-1967 proxy deployment) so the new code is easy to audit by analogy.

## Concepts

For each transaction the registry returns three values:

| Field | Type | Meaning |
|---|---|---|
| `level` | `uint64` | `0` = no priority. `> 0` = prioritized; higher levels form earlier partitions (a higher level is always scheduled before a lower one). |
| `weight` | `uint64` | Tie-breaker *within* a level — higher weight first. |
| `id` | `uint128` | Entity identifier. Transactions sharing an `id` are rate-limited together. |

Resulting block order (when the feature is enabled):

```
[ prioritized txs, in (level desc, weight desc, txhash asc) order,
  each sender's txs kept in nonce order ]
        ++
[ everything else, in its base order (scramble order / proposal order) ]
```

where "everything else" includes genuinely non-prioritized transactions, and
transactions *demoted* because they were nonce-unreachable or because their entity
exceeded the per-block gas budget.

## Registry ABI

The registry lives behind an EIP-1967 proxy at a fixed address (see
`registry.GetAddress()`), exactly like the subsidies registry. The node depends
only on the **ABI shape** (selectors + return layouts), never on the exact
bytecode — the implementation is governed and upgradeable.

### `getPriority`

```solidity
function getPriority(
    address from,
    address to,
    uint256 value,
    uint256 nonce,
    bytes   calldata data,
    uint256 gas            // tx gas limit — lets the registry exclude oversized txs
) external view returns (uint64 level, uint64 weight, uint128 id);
```

The inputs mirror the subsidies `chooseFund` call plus the transaction `gas` limit,
so a registry can base priority on sender, recipient, value, method/calldata, and
size. Calldata is **hand-encoded** with a fixed byte layout and the response
**hand-decoded** with strict length checks, for determinism and speed.

Priority is **orthogonal** to subsidies and bundles: a transaction may be sponsored
*and* prioritized; the two registries do not interact (but see §13).

### `getPriorityConfig`

```solidity
function getPriorityConfig() external view
    returns (uint256 maxGasPerEntityPerBlock, uint256 maxPiggybackTxsPerEntityPerEvent /*, ... */);
```

Queried once per block (block formation) and opportunistically by the emitter. As
with subsidies' `getGasConfig`, the response is decoded by length so additional
fields can be appended in a backward-compatible way later.

`maxGasPerEntityPerBlock` is the **total gas budget** of prioritized transactions of
one entity in a single block. Block formation spends it in `(level desc, weight desc,
hash asc)` order, but only on a transaction whose lower-nonce predecessors from the
same sender have already been selected, so it is never wasted on a transaction that
would be nonce-too-high anyway (§11). A transaction that does not fit blocks its
sender's later nonces.

`maxPiggybackTxsPerEntityPerEvent` bounds only **foreign** prioritized transactions —
those an emitter eagerly piggybacks onto an event while it is **not** this validator's
turn. Transactions the validator includes on its own turn are not counted against it.
Being per event, it bounds the size of one event, not an entity's total footprint; the
per-block gas budget is the actual rate limit.

### Rate-limit design: transaction count vs. gas

The per-entity limit could be a fixed **transaction count** per block (easy to explain:
"you get ten priority slots") or a **gas budget** per block, spent on however many
transactions fit.

Gas is chosen for two reasons. It is more flexible — the same budget accommodates many
small transactions or a few large ones, letting an entity trade *many cheap* against
*few expensive* at the same per-block cost. And decisively, a block's capacity is
itself bounded by gas, so a gas-denominated per-entity limit composes directly with a
block-level priority share: capping total prioritized gas at, say, 10 % of block
capacity is a simple sum over per-entity budgets. Expressing the same cap in
transaction counts would require assuming every transaction is maximally sized,
forcing the per-entity count low enough to be pessimistic for the common case.
Combining both metrics was rejected: it complicates what an entity can predict about
its own throughput for little gain.

The budget is charged on the transaction's **gas limit**, since block formation must
decide the order *before* executing anything. The limit is floored at intrinsic gas, so
it cannot be understated, and over-declaring only shrinks the entity's own throughput.
The budget therefore bounds *reserved* block space: realized prioritized gas is always
≤ the nominal sum of budgets.

### Failure handling

Forward/backward compatibility is handled purely by **response length** (never by
selector versioning or revert-catching), as in subsidies. The failure rules are part of
the consensus contract:

| Failure | Result |
|---|---|
| `getPriority` errors, reverts, runs out of gas, or returns a wrong-length / high-byte-overflowing result | that transaction is **level 0** |
| sender cannot be recovered | that transaction is **level 0** |
| registry absent (no code at the address) | **all** transactions level 0 |
| `getPriorityConfig` fails in any of those ways | fallback config with **both limits zero** ⇒ nothing is prioritized at all |

The zero-valued fallback config is the safest degradation: every validator that fails
to read the config produces the same, un-prioritized ordering. A block is never aborted
or skipped because of a registry failure, and failures are counted
(`chain/priorities/*`, `emitter/priorities/*`) so silent degradation is observable.

Two consequences of the strict encoding. A config word that does not fit a `uint64` is
rejected, so setting `maxGasPerEntityPerBlock = type(uint256).max` intending
"unlimited" **silently disables the feature chain-wide**; the "unlimited" encoding is
`2^64 - 1`. And the value encoder rejects a negative or wider-than-256-bit transaction
value instead of panicking on it, which is what makes classification total over every
*decodable* transaction — the guard that rejects such values earlier sits behind
`Upgrades.Brio`, which this feature does not require.

### Requirements on a registry implementation

None of these are enforceable by the node, and each one is load-bearing:

1. **`getPriority` must be O(1) in gas**, comfortably below the 100 000 cap and
   independent of registry size and attacker-writable state. If its cost can be pushed
   over the cap, priority silently switches off chain-wide (§1).
2. **The registry cannot keep state across queries.** Every query is snapshot-reverted
   (§4), so registry-side accounting or per-block quotas silently do nothing; the
   node's config fields are the only enforcement.
3. **`getPriority` is called from the zero address with an unset transaction
   context** — `msg.sender` is `0x0`, `tx.gasprice` / `tx.origin` read as zero. A
   registry that gates on `msg.sender` reverts, classifying everything as level 0.
4. **Whoever can cause a given `id` to be returned can spend that `id`'s budget** (§9).
   The node keys the rate limit purely on the returned `id` and never relates it to the
   sender.
5. **`id` must be scarce and governed.** Nothing bounds the number of distinct `id`s in
   a block and there is no block-level cap on prioritized gas, so a block's aggregate
   priority share is exactly `Σ` per-entity budgets (§8).

## Two-stage model

1. **Authoritative ordering — block formation (`gossip/c_block_callbacks.go`).** The
   single place that decides the final order, reproduced deterministically by every
   validator from the same finalized block-start state. **The only stage that affects
   consensus.**
2. **Best-effort hints — emitter.** Used only to get prioritized transactions into the
   DAG / proposal quickly. A wrong or stale hint costs at most a little bandwidth; it
   can never change the resulting block, because stage 1 re-queries authoritatively.

This separation confines the expensive and uncertain parts to the non-authoritative
stage, which is also why the emitter may use a non-total comparison order, a cache that
never invalidates, and head instead of block-start state.

## Authoritative ordering (block formation)

In `c_block_callbacks.go`, after the base order is produced (scrambler for legacy;
`proposal.Transactions` for single-proposer) **and after**
`filterNonPermissibleTransactions`, a single pure transform is applied — gated by
`Upgrades.TransactionPriorities`. It runs before `evmmodule.Start`, so the nonces it
reads are genuine block-start nonces.

```
prioritize(baseOrdered, vm@blockStartState, signer, config):
  1. classify every tx -> (level, weight, id)          // queries the registry
  2. per sender, build its chain: prioritized txs in nonce order forming a
     contiguous run from the block-start account nonce (stale nonce skipped,
     gap ends the run, lowest hash wins a duplicate nonce)
  3. walk chains greedily: repeatedly take the highest (level desc, weight desc,
     txhash asc) frontier tx whose entity budget (config.MaxGasPerEntityPerBlock)
     still fits it, and advance that chain; a frontier that does not fit blocks
     its chain, without debiting the budget
  4. emit selected txs in selection order: (level desc, weight desc, txhash asc),
     but each sender's txs kept in nonce order (a chain's frontier only advances
     in nonce order, so a higher-priority later nonce never overtakes its
     predecessor)
  5. result = [selected, in that order] ++ [base order minus the selected txs]
```

The result is a **permutation** of the input. Because demoted and non-prioritized
transactions keep their base positions, "demote to normal pool" (legacy) and "keep
proposer order" (single-proposer) both fall out of the same code. Not debiting the
budget for a transaction that does not fit means one oversized transaction blocks only
its own sender's tail, while sibling senders of the same entity keep spending the
remainder.

Chain walking uses `utils/frontierheap`, a max-heap over the *heads* of a set of
sequences, shared with the emitter. It matches nonce-constrained selection exactly —
advancing means "taken, next nonce eligible", dropping a sequence means "unusable, and
so is everything behind it", so the budget check needs no extra bookkeeping — and it
costs O(n log s) for n transactions across s senders instead of O(n·s). Consequently a
drain is **not** a sorted traversal: only frontiers are ever compared, so an element
deep in a sequence can outrank everything and still surface late.

The same transform runs for **both** modes. In single-proposer mode the proposer's
order is therefore **not trusted**: block formation re-creates the priority ordering
and overrides it. The override can **reorder** the proposed set but cannot **add**
transactions the proposer omitted — inclusion remains the proposer's prerogative,
defended by turn rotation.

### Classifier seam

`prioritize` consumes a `Classifier` interface (`Priority(tx) (Priority, error)`).
Only the per-transaction EVM classifier ships: one `getPriority` call per transaction,
each in its own state snapshot. The seam exists as a **measured escape hatch** — the
benchmark's `Native/*` arm models a fallback that fetches the classification criteria
once per block and applies them in native Go, which can be dropped in behind the
interface if the per-transaction cost ever becomes unacceptable.

Either way, *all* transactions are classified. Restricting classification to a subset
is **not** an option, as it would void priority guarantees exactly when they matter
most, under high load.

## Emitter (best-effort hints)

A priority context, built per ordering pass, classifies candidates against the
**current head block**; it is absent — and everything then non-prioritized — when the
feature is off or the required head state is unavailable.

Classifications are memoized by transaction hash, one cache slot per transaction the
pool admits, with no expiry and LRU eviction. Staleness is acceptable by construction:
a transaction that has *lost* priority keeps it here, which only works in its favour,
and one that *gains* priority later is old by then. **A registry change therefore does
not reach the emitter for already-classified transactions** — not even a proxy upgrade.
None of this reaches consensus, since block formation re-classifies every transaction
against block-start state on every block; what is observable is that an entity just
granted or just revoked priority keeps seeing the old emitter behavior for its pending
transactions (§14).

**Ordering.** Pending transactions are drained from a `FrontierHeap` per sender,
ordered by

```
(stage asc, priority level desc, priority weight desc, effective tip desc, first-seen time asc)
```

with three stages: prioritized on this validator's turn, prioritized on another
validator's turn, and not prioritized. A transaction whose effective tip cannot be
computed at the current base fee is dropped together with its sender's later nonces;
sponsorship requests bypass the base-fee check and carry a zero tip.

The stages are a *comparison key*, not three sequential passes. Because only sender
frontiers are compared, a prioritized transaction sitting behind an ordinary nonce
takes no part in the ordering until that predecessor is consumed, and so is drawn
*after* it. Consumers must cope with a prioritized transaction arriving once ordinary
ones have already spent gas power, event size, and — for foreign transactions — the
eager budget: priority raises a transaction's position only among the frontier it is
actually competing with. Turn verdicts are computed once per candidate at build time,
which is why the cached ordering may be reused for at most one second.

**Piggyback model.** A validator must **not** emit an event *solely* because it holds
prioritized transactions it does not own. The event-emit decision and all throttling
(`NoTxsThreshold`, `LimitedTpsThreshold`, stake-based suppression) are unchanged, and
transactions on this validator's turn are admitted subject to the pre-existing checks
(event size, epoch rules, gas power, one-unconfirmed-tx-per-sender, txpool freshness,
bundle validity). A transaction on *another* validator's turn is admitted eagerly only
if it is prioritized and within three bounds:

1. `maxPiggybackTxsPerEntityPerEvent` eager admissions per entity;
2. eager admissions together may claim at most **half** of the event's remaining gas
   budget — prioritized candidates are staged first, so without it a few large foreign
   transactions would starve the validator's own;
3. if the finished event carries **none** of this validator's own transactions, every
   eager admission is **rolled back**. This is what actually enforces "no priority-only
   events", as a property of the finished event rather than an ordering precondition.

Together these bound duplication across validators, prevent priority-only events, and
avoid inducing low-stake validators to emit.

In **single-proposer mode** every candidate counts as own-turn, so the whole piggyback
apparatus is inert. The same heap feeds the proposal scheduler, which trial-runs each
candidate against block state, so priority biases *which* candidates are offered first
and therefore survive the gas/size cut, while inclusion remains the proposer's
decision.

**Future: a new consensus removes the eager-admission limits.** All three bounds exist
only to keep events small in the current consensus, where events are long-lived DAG
members and one entity's prioritized transactions would otherwise be duplicated N-fold
across it (§6). Under the forthcoming consensus events are ephemeral, so their size no
longer needs constraining and the foreign-piggyback path and its bounds become
unnecessary. The authoritative per-block gas budget is independent of them and remains
the sole rate limit.

## Determinism & byte-compatibility

- **Fully gated** by `Upgrades.TransactionPriorities`. While OFF: no new state reads,
  no new bytes, **identical block hashes** to today. The flag is an optional feature
  toggled at epoch boundaries, like `SingleProposerBlockFormation` / `GasSubsidies`.
  All nodes must run a build that understands the flag before it is enabled.
- `prioritize` is a **pure total function** of (transaction set, registry state,
  block-start state, config) — total over every decodable transaction, on every rule
  set.
- All ABI encode/decode is **hand-rolled with strict length and high-byte overflow
  checks**; the per-call gas limits are compile-time client constants, never
  registry-supplied, so no contract configuration can change the cost bound. `from` is
  the recovered signer, so no classification input is attacker-supplied.
- Each registry query runs inside `Snapshot` / `RevertToSnapshot` **per query**, so
  reads leak no warm-access entries, transient storage, logs, refunds, or self-destruct
  marks into real execution, and one transaction's query cannot influence another's.
- The ordering EVM's block context uses **only consensus-derived values** (computed
  block time, computed randao, derived base fee, deterministic coinbase, max block
  gas) — never wall-clock or node-local data.
- **The hash tie-break is what makes map iteration safe.** Senders are grouped in a Go
  map, but `(level, weight, hash)` is a strict total order on the frontier —
  transaction hashes are unique within a block — so the heap root is the unique maximum
  regardless of insertion order. This is an invariant to preserve, not an incidental
  property (§5).
- The emitter's comparator is *not* a total order (its last tie-break is first-seen
  time, and its builder also iterates a map). Acceptable only because the emitter is a
  single-writer, non-consensus stage.

## Risks & residual

The issues identified during design review, how each is addressed, and what remains.
Open items, in the order they deserve attention:

1. **`id` policy** (§9) — the hinge for both the cheapest attack and the aggregate
   priority share of §8. Needs a written registry policy.
2. **The `getPriority` gas cap** (§1) — a silent, chain-wide off-switch, worsened by
   the emitter's negative caching (§14).
3. **Tie-break grinding** (§10) — free and deterministic; one cheap fix also closes
   intra-entity starvation (§8).
4. **Unstable sort on duplicates** (§5) — a latent split whose safety currently rests
   on a property of the skip path rather than of the ordering code.
5. **The unprotected stand-in proxy** (§7) — a release gate, and part of why activation
   is still blocked.
6. **Sender conflict in legacy mode** (§12) — cheaply negates the feature for a
   targeted sender.

### 1. Critical-path query cost (DoS / liveness)

*Issue.* Classifying every transaction with an EVM call adds work to block formation —
the hot consensus path. Unlike subsidies (which only queries the small zero-gas-price
subset), this touches *all* transactions, so a flood of cheap transactions multiplies
the cost, and the work is unmetered and uncompensated: a minimal 21 000-gas transaction
buys roughly 5× its own gas in replicated work nobody pays for.

*Addressed.* Per-call gas is capped at a fixed client constant (100 000), bounding
worst-case work at `numTxs × cap` regardless of what the registry does. The strategy is
**benchmark-gated**: if per-transaction latency becomes unacceptable, the native-filter
fallback drops in behind the `Classifier` seam.

*Measured.* `BenchmarkPrioritize` runs the whole ordering pass against a real in-memory
Carmen state pre-populated with 10 000 dummy accounts (so the account trie has a
representative depth) and the registry deployed behind the production EIP-1967 proxy.
On an Intel i7-6600U (2.60 GHz, single-threaded):

| transactions | per-tx EVM call (default) | native-filter (fallback) |
|---|---|---|
| 10     | 0.22 ms      | 0.008 ms |
| 100    | 1.36 ms      | 0.047 ms |
| 1,000  | 23.6 ms      | 0.35 ms  |
| 10,000 | 247 ms       | 4.9 ms   |

So the default costs ≈ **25 µs per transaction**, the native fallback ≈ 0.5 µs (≈ 50×
cheaper). Result mix barely moves the total (all-normal 311 ms, 10 % mix 278 ms,
all-prioritized 242 ms at 10 000 txs) — the query is paid for every transaction
regardless of outcome, so the ordering passes are negligible next to it. 1 KiB of
calldata per transaction adds ≈ 12 %.

*Decision.* Keep the per-tx-call classifier as the default: typical blocks are far
below the ceiling, where the cost is single-digit milliseconds, and it needs no
additional registry ABI.

*Residual.* The benchmark measures the stand-in registry (one `SLOAD`), not the bound:
a registry using its full allowance costs roughly an order of magnitude more per
transaction, so the honest worst case is `numTxs × 100 000` gas of uncompensated work on
every validator. `numTxs` is not bounded by the benchmark's 10 000 either — steady state
is governed by the gas economy (~700 minimal tx/s sustained, single-digit milliseconds
per block), but block assembly spills events up to the 5 Ggas maximum block gas, on the
order of 200 000 minimal transactions, and ordering runs on all of them because it
precedes the per-block gas limit applied during execution. The largest classification
bill therefore arrives exactly when the network is recovering from a stall. Hardening
options: a deterministic aggregate per-block classification budget (transactions beyond
it ⇒ level 0), or the native-filter fallback.

The **gas cap is also an attack surface**: if `getPriority`'s cost depends on global
registry state — any iteration over a registered-entity list, any growable mapping
walk — an attacker who can grow that state pushes every query over the cap and silently
demotes the whole network's priority traffic to level 0, observable only as the failure
counter pegging at 100 %. The structural defense is registry requirement 1; raising the
cap is possible, but any variable cap must remain consensus-derived.

The emitter classifies every pending transaction per tick while holding a state handle.
The per-pool-slot cache means each transaction is classified once per node with no
thrash, but with a registry using its full gas allowance the sweep could approach the
600 ms emission interval — worth watching, not worth redesigning.

### 2. Non-deterministic failure handling (chain split)

*Issue.* If validators disagree on what a failed or malformed query means, they produce
different blocks → fork.

*Addressed.* The hard rules are tabulated under failure handling above: any per-tx
failure ⇒ level 0; any config failure ⇒ zero limits; registry absent ⇒ all transactions
level 0. The block is never aborted or skipped, and every node runs the query against
the same state and contract.

*Residual.* None. Sender recovery is likewise deterministic — a pure function of the
transaction bytes.

### 3. EVM-context determinism

*Issue.* If the ordering EVM's block context contains any node-local value, equal state
could yield different priorities.

*Addressed.* The context is built solely from consensus-derived block fields. No
`time.Now()` or other local input.

*Residual.* Two footnotes rather than risks. The query EVM is a full EVM whose `GetHash`
returns the zero hash when the local store lacks a header, so a registry using
`BLOCKHASH` inherits the same node-local-history assumption as ordinary execution. And
the transaction context is left unset, which is registry requirement 3.

### 4. Per-query isolation / state residue

*Issue.* A registry read could leave residue (warm slots, transient storage, logs,
refunds, self-destructs) that perturbs subsequent real execution, or an earlier query
could influence a later one.

*Addressed.* Snapshot + revert around **each individual** query (not once around the
loop), mirroring subsidies. Carmen's undo log covers storage, access-list entries and
logs, so the revert is complete and the snapshot depth stays flat across a whole block's
queries.

*Residual.* The queries are ordinary calls, not static calls, so a registry *may* write
during classification — those writes are simply undone, which is why registry
requirement 2 exists. Isolation is verified structurally (every query is asserted to be
snapshot-wrapped); a byte-identity test would be a worthwhile addition.

### 5. Single-proposer reorder & duplicate transactions

*Issue.* In single-proposer mode we execute a reordered list while the signed proposal
hash covers the proposal order.

*Addressed.* The block hash derives from the executed (reordered) list, which is
intended. Proposal validation is order-agnostic (it checks only for nil transactions and
total size), and nothing anywhere cross-checks executed order against
`proposal.Transactions`.

*Residual.* The proposer still controls *inclusion*; the override only reorders
(accepted — turn rotation defends against a censoring proposer, §12). Proposals are also
**not deduplicated**, which makes a latent split reachable: with two entries holding the
same transaction the nonce comparison returns 0 and the sort is not stable, so which
index enters the sender sequence — and hence where the other one sits in the
remainder — is unspecified. It is harmless *today* only because the losing duplicate is
nonce-too-low, and skipped transactions consume no gas or block size and produce no
receipt, so the resulting block is identical either way. That argument rests on a
property of the skip path rather than of the ordering code, and the no-duplicates
precondition otherwise comes only from the scrambler, which is gated on `Upgrades.Sonic`.
Two one-line fixes: dedup by hash before ordering, or give the nonce comparison an
entry-index fallback.

### 6. Eager-emit bandwidth amplification

*Issue.* Letting every validator emit prioritized transactions duplicates them N-fold
across the DAG and could be used to stress the network.

*Addressed.* The piggyback model forbids priority-only events (enforced by the rollback)
and preserves all existing throttling; the per-entity count cap and the half-gas share
bound eager admissions within any one event.

*Residual.* Some cross-validator duplication remains by design — that is the point,
faster inclusion. Three qualifications on how tight the per-entity bound is:

- The half-gas share is a **single global counter**, not per entity, so one entity's
  large prioritized transaction can consume an event's whole eager gas budget.
- Own-turn admissions are **not bounded at all**, intentionally, so turn ownership is
  what limits a validator's own footprint. Turn ownership is a function of the sender
  address, so an entity can grind addresses until its own validator owns round 0 —
  making it deterministic rather than probabilistic. That buys **inclusion latency
  only**; the per-block gas budget is enforced at block formation and is unaffected.
- A **bundle envelope** is one transaction carrying N inner transactions from N senders,
  and the classifier sees only the envelope. Envelope validation requires the envelope
  gas limit to cover the sum of inner gas limits, so the authoritative gas budget is
  charged correctly, but the piggyback cap counts *transactions* and is incremented once
  per envelope — as are the one-unconfirmed-tx-per-sender guard and the nonce-contiguity
  rule. Low severity, confined to the best-effort stage; one envelope also means one
  `getPriority` call, so bundles *reduce* classification cost. If it matters, express
  the piggyback cap in gas rather than count.

Finally, each emitter skip site increments its counter by one and then drops an entire
sender sequence, so the skip counters undercount by an unbounded factor and cannot
diagnose a §12-style attack. Fix by counting sequence lengths.

### 7. Registry admin & stand-in deployment as trust anchors

*Issue.* The upgradeable, governed registry can grant any transaction front-of-block
placement and push everyone else behind it — i.e. sanctioned front-running and de-facto
reordering/censorship power over the chain.

*Addressed.* An inherent property of a governed ordering oracle, the same trust model as
the subsidies registry but with ordering power. Mitigations are governance controls and
transparency, documented prominently so operators understand the trust placed in the
registry admin.

*Residual / accepted.* The admin is trusted for ordering. Three powers that are easy to
overlook: it can **revoke** priority as freely as grant it; it controls `id` assignment
and therefore decides whose budget a third party can drain (§9); and an upgrade that
makes `getPriority` exceed its gas cap turns the feature off chain-wide with no visible
governance action (§1). Because queries are snapshot-reverted the registry cannot
implement its own accounting, so all enforcement is node-side and limited to the two
config fields.

*Residual — release gate.* The stand-in artifacts are unprotected: the proxy's
`update(address)` has **no access control** and the stand-in registry's setters are plain
`external`. Both are self-documented as local-testing artifacts installed only by the
fake-genesis path (the same pattern already ships for gas subsidies), and the feature is
inert until a real registry is deployed — the placeholder address holds no code, so every
call fails ⇒ all transactions level 0. But if a production genesis were ever generated
from the fake-genesis path, anyone could `update()` the implementation and own block
ordering for the whole chain. The activation checklist must assert: (a) a real, deployed
proxy address replacing the placeholder; (b) an access-controlled proxy; (c) an
access-controlled implementation.

### 8. Starvation of normal traffic, and of siblings inside an entity

*Issue.* Per-entity budgets bound one entity, not their **sum**. Nothing bounds how many
entities exist and there is no block-level cap on total prioritized gas, so a
sufficiently large entity set can fill an entire block while every individual entity
stays inside its own limit.

*Addressed by governance, not by the node.* The registry sets both the entity set and
each entity's budget, so `Σ` budgets is a governed quantity and keeping it within an
acceptable share of the block gas limit is a registry governance responsibility — the
gas denomination is what makes that a plain sum. No node-enforced block-level share cap
is planned.

*Residual / accepted.* The node reserves no space for ordinary traffic and will neither
prevent nor report starvation by an over-generous registry. For scale: at the stand-in
default of 10 M gas per entity per block, on the order of a dozen entities accounts for
a full block's realistic capacity.

*Residual — inside an entity.* Senders sharing an `id` share one budget counter, and
selection always serves the globally best frontier, so a sender with higher level/weight
and many cheap transactions can drain the shared budget before its sibling's frontier is
ever peeked; there is no per-sender fair share. Not third-party controllable, since
level and weight come from the registry — except via tie-break grinding: when two
siblings tie on `(level, weight)`, the one with the lower hash starves the other, and
that hash is free to grind (§10). Dropping an over-budget frontier without debiting
limits the damage: one oversized transaction blocks only its own sender.

### 9. Rate-limit bypass and third-party drain via `id`

*Issue.* The per-entity limit keys on the registry-returned `id`, so an entity that can
induce distinct `id`s evades the limit — and a registry that derives `id` per
transaction, or lets a client influence it, gives every transaction a fresh full budget.
Nothing in the node constrains this; the decoder accepts any 16 bytes.

*Addressed.* Delegated to the registry as hard requirements 4 and 5: `id` must be a
function of an attacker-unforgeable authority, and the number of live `id`s must be
governed. The node enforces the limit faithfully given the returned `id`s.

*Residual / accepted by delegation.* Fully open at the node level, and combined with §8
`id` scarcity is the *only* thing standing between the feature and a block entirely
filled with prioritized traffic.

The same delegation enables **denial**, not just bypass. If the registry classifies on
transaction shape — target, selector, value — rather than on the signer, any party can
craft a matching transaction: it gains priority it never signed up for **and** spends
the imitated entity's budget. Since the budget is debited on the gas limit before
execution, minimal 21 000-gas junk priced at base fee suffices and need not even succeed,
which makes this **the cheapest attack in this document**. Criteria-based classification
is nonetheless a legitimate intended use ("all calls to contract X are prioritized");
the property a registry must decide on is requirement 4. Sender allowlists, or a signed
permit carried in `data`, keep priority and budget coupled; keying on `to` does not.
Registries that intend to prioritize a **bundler** should note the same shape: keying on
`from` lets anyone who gets into that bundler's envelope inherit front-of-block
placement, and `to` is always the bundle processor, so the real target cannot be keyed
on without parsing `data`.

### 10. Tie-break grinding

*Issue.* Ties within equal `(level, weight)` are broken by transaction hash, which a
submitter can grind.

*Addressed.* Weight is the primary, registry-controlled lever; hash only orders exact
`(level, weight)` ties.

*Residual — open.* Grinding is **free**: the classifier input excludes the signature, so
re-signing an identical payload yields a different hash with an identical
classification, at the cost of one ECDSA signature and no gas. Whoever grinds wins every
tie deterministically, computed offline before submission. That is weaker than the status
quo it sits next to — the scrambler's salt derives from the block's transaction-hash set
and is unpredictable, whereas the priority prefix is pre-computable — and it matters most
for the obvious way to write a registry, one uniform `(level, weight)` for a whole
customer class. It is also the mechanism behind intra-entity starvation (§8). A cheap fix
preserves determinism: tie-break on `keccak(txHash ‖ prevRandao)` instead of the raw
hash, since `randao` is already available to the ordering pass and is unknown when the
transaction is signed.

### 11. Per-sender nonce vs. hoisting

*Issue.* Hoisting a high-nonce prioritized transaction ahead of a same-sender
lower-nonce non-prioritized one makes the hoisted one fail (nonce-too-high → skipped),
so only the earlier one lands — prioritizing it is what keeps it out.

*Addressed.* Priorities are selected per sender in nonce order: only the contiguous run
of prioritized nonces starting at the block-start account nonce keeps its priority, and
the budget is spent on a transaction only once its lower-nonce predecessors are selected.
A transaction whose predecessor is not prioritized (or does not fit the budget) drops to
its base-order position, where the predecessor runs first; a stale prioritized nonce
cannot execute and is skipped without blocking later nonces; among same-nonce siblings
the lowest hash keeps priority and the run continues past them. The prefix is emitted in
selection order, so a sender's own run is never reordered into a nonce-too-high position
either.

*Residual.* Not exploitable against a third party — every grouping and nonce decision is
keyed on the recovered sender, so it requires the victim's key. Against oneself it is a
sharp footgun: a sender that **mixes ordinary and prioritized traffic on one address**
loses its whole prioritized run behind the first ordinary nonce, since the gap demotes
the tail authoritatively and the emitter drops a whole sequence whose head is
non-prioritized on a foreign turn. Guidance for entities: **use dedicated addresses for
prioritized traffic.**

One ordering-policy change worth recording: within a block a prioritized transaction
wins a same-nonce race against a non-prioritized one **regardless of gas price**,
overriding the scrambler's `(nonce, gasPrice desc, hash)` replacement rule, so a
prioritized sender can replace its own in-flight transaction at a lower fee.

### 12. Censorship of a prioritized sender

*Issue.* A validator withholds a victim's prioritized transactions from its own events;
a single proposer omits them entirely.

*Addressed.* Turn rotation means no single validator owns a transaction's turn for long,
the piggyback path lets any validator carry a prioritized transaction it does not own,
and an entity may run its own validator to remove the dependency altogether. In
single-proposer mode the authoritative override can reorder the proposed set but cannot
add to it, so inclusion is the proposer's alone.

*Residual / accepted.* Withholding or reordering within a validator's own events is
non-Byzantine by construction, and out-of-turn emission is confirmed not slashable. A
censoring single proposer denies priority for the whole of its turn; only rotation
defends. This remains the largest accepted risk in single-proposer mode.

*Residual — open, legacy mode.* The emitter drops a sender's **entire** sequence while
that sender has unconfirmed transactions in connected events. That counter is
incremented for every transaction in every connected event **from any validator** and
decremented only on confirmation, so a malicious validator that repeatedly emits events
containing any transaction from the victim's prioritized sender — including
already-executed duplicates, which nothing on this path rejects — makes every honest
emitter on the network refuse all of that sender's transactions. Bounded by the counter
being cleared at epoch start, by LRU eviction at 20 000 addresses, and by the attacker's
own gas power, and it **does not apply in single-proposer mode**, where events carry no
individual transactions. A pre-existing mechanism that priorities make worth exploiting:
it negates the feature for a targeted victim at modest cost, and is a real asymmetry
between the two block-formation modes.

### 13. Cross-feature: priority decides the subsidy race

*Issue.* Sponsored transactions have `gasPrice == 0` and a zero tip. Priority is compared
before tip in the emitter and dominates everything in the authoritative prefix, so a
prioritized *and* sponsored entity systematically reaches a shared subsidy fund ahead of
everyone else, every block, at zero cost to itself.

*Addressed.* Fund-side limits are the subsidies feature's own concern; nothing here is
incorrect.

*Residual.* "Priority is orthogonal to subsidies" is true of the registries but too
strong as a statement about outcomes: priority removes the fee-based deterrent that
previously moderated both subsidy consumption and the budget-burning of §9.

### 14. Stale emitter hints

*Issue.* The emitter's cache is keyed by transaction hash and never invalidated, so a
classification can outlive the registry state it came from.

*Addressed.* The blast radius is event bandwidth only: block formation re-classifies
against block-start state, so a transaction whose priority was revoked lands as
ordinary, and the eager path is bounded by the count cap, the half-gas share and the
rollback. Evicting the cache to force fresh classification only helps the victim.

*Residual — the inverse direction, worth fixing.* A *failed* classification is memoized
as level 0 with no TTL and no invalidation, so a single transient failure costs a
transaction the emitter fast path for the rest of its pool lifetime. This compounds §1's
gas-cap exposure: a transient registry problem leaves a durable emitter-side deficit.
Not caching failures, or caching them with a short TTL, resolves it.

## Known limitations

- **Placeholder registry address.** `registry.GetAddress()` returns
  `0x7072696f72697479…` ("priority" in ASCII). The final address must be fixed, and the
  proxy and implementation access-controlled, before activation on a public network
  (§7).
- **Single-proposer inclusion:** the authoritative override reorders only the proposed
  set; it cannot add transactions the proposer omitted (§5, §12).
- **Nonce vs. hoisting:** ordering is sound, but mixing ordinary and prioritized traffic
  on one address forfeits the run (§11).
- **Emitter hints lag the registry:** classifications are memoized per transaction hash
  and never expire (§14).
- **No block-level priority share cap:** the aggregate is `Σ` per-entity budgets, a
  registry governance responsibility (§8).
- **Free tie-break grinding** within a `(level, weight)` class (§10).

## Configuration & activation

`Upgrades.TransactionPriorities` is an optional feature flag. Changes take 
effect at the start of the next epoch, applied via an ordinary network-rules-update
transaction.

Validation imposes **no constraints** on the flag: it may be freely enabled *and*
disabled, requires no other upgrade, and is not checked against whether a registry is
deployed. It must only be enabled when all nodes run V2.3 or later, because a node that ignores
the flag cannot reproduce the resulting blocks and will drop off the network. 
