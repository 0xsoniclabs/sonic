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
size. Calldata is **hand-encoded** with a fixed byte layout (selector ‖ from ‖ to ‖
value ‖ nonce ‖ data-offset ‖ gas ‖ data) and the response is **hand-decoded** with
strict length checks, for determinism and speed.

Priority is **orthogonal** to subsidies and bundles: a transaction may be sponsored
*and* prioritized; the two registries do not interact.

### `getPriorityConfig`

```solidity
function getPriorityConfig() external view
    returns (uint256 maxGasPerEntityPerBlock, uint256 maxPiggybackTxsPerEntityPerEvent /*, ... */);
```

Queried once per block (block formation) and opportunistically by the emitter. As
with subsidies' `getGasConfig`, the response is decoded by length so additional
fields can be appended in a backward-compatible way later.

`maxGasPerEntityPerBlock` is the **total gas budget** of prioritized transactions
of one entity in a single block: block formation spends it on the entity's
transactions in `(level desc, weight desc, hash asc)` order, but only on a
transaction whose lower-nonce predecessors from the same sender have already been
selected, so the budget is never wasted on a transaction that would be
nonce-too-high anyway (see §11). A transaction that does not fit blocks its
sender's later nonces. This lets an entity trade *many cheap* transactions
against *few expensive* ones with the same per-block cost.

`MaxPiggybackTxsPerEntityPerEvent` bounds only **foreign** prioritized
transactions — those an emitter eagerly piggybacks onto an event while it is
**not** this validator's turn (see the emitter section below). Transactions the
validator includes on its own turn are not counted against this cap.

### Rate-limit design: transaction count vs. gas

The per-entity limit could be expressed two ways:

- **By transaction count** — a fixed number of prioritized transactions per
  entity per block (e.g. ten). Easy to explain ("you get ten priority slots per
  block").
- **By gas (chosen)** — a total gas-limit budget per entity per block (e.g.
  1M gas, `MaxGasPerEntityPerBlock`). The entity spends the budget on however many
  transactions fit, trading *many cheap* against *few expensive* ones.

Gas is chosen for two reasons. First, it is more flexible: the same budget
accommodates many small transactions or a few large ones, without forcing the
entity into a one-size-fits-all slot. Second — and decisively — a block's capacity
is itself bounded by gas, so a gas-denominated per-entity limit composes directly
with a block-level priority share: capping total prioritized gas at, say, 10 % of
block capacity is a simple sum over per-entity budgets, which is what makes
governance of the aggregate tractable. Expressing the same cap in transaction
counts would require assuming every transaction is maximally sized, forcing the

Combining both metrics, or introducing others, is possible but was rejected: it
complicates what an entity can predict about its own throughput for little gain.

### Versioning & failure handling

Forward/backward compatibility is handled purely by **response length** (never by
selector versioning or revert-catching), as in subsidies.

The failure rules are part of the consensus contract:

| Failure | Result |
|---|---|
| `getPriority` errors, reverts, runs out of gas, or returns a wrong-length / high-byte-overflowing result | that transaction is **level 0** |
| sender cannot be recovered | that transaction is **level 0** |
| registry absent (no code at the address) | the call returns `nil`, reported as `"priority registry contract not found"` ⇒ **all** transactions level 0 |
| `getPriorityConfig` fails in any of the above ways | `priorities.FallbackConfig` = **both limits zero** ⇒ nothing is prioritized at all |

`FallbackConfig` is zero-valued on purpose: it is the safest degradation, because
every validator that fails to read the config produces the same, un-prioritized
ordering. A block is never aborted or skipped because of a registry failure. Failures
are counted (`chain/priorities/{config,txs}/failed`,
`emitter/priorities/{config,txs}/failed`) so silent degradation is observable.

Note the consequence for the `Config` encoding: because `parseGetPriorityConfigResult`
rejects any word that does not fit a `uint64`, a governance action that sets
`maxGasPerEntityPerBlock = type(uint256).max` intending "unlimited" **silently
disables the feature chain-wide**. The "unlimited" encoding is `2^64 - 1`.

### Requirements on a registry implementation

These are not enforceable by the node, and every one of them is load-bearing. They
are collected here because a registry that violates them breaks the feature or a
third party.

1. **`getPriority` must be O(1) in gas**, with a worst case comfortably below
   100 000 and independent of registry size and of attacker-writable state. If its
   cost can be pushed over the cap, priority silently switches off chain-wide (see
   [C-2](#c-2-pushing-getpriority-over-its-gas-cap)).
2. **The registry cannot keep state across queries.** Each query is wrapped in a
   state snapshot that is immediately reverted (see
   [§4](#4-per-query-isolation--state-residue)), so any storage write, log, or
   counter a registry performs during `getPriority` is undone. Per-block quotas or
   accounting must not be attempted registry-side; the node's `Config` fields are
   the only enforcement.
3. **`getPriority` is called from the zero address with an unset transaction
   context.** `msg.sender` is `0x0`, and `tx.gasprice` / `tx.origin` read as zero. A
   registry that gates on `msg.sender` will revert and thereby classify everything as
   level 0.
4. **Whoever can cause a given `id` to be returned can spend that `id`'s budget.**
   The node keys the rate limit purely on the returned `id` and never relates it to
   the sender. A registry that derives `id` from attacker-suppliable fields hands
   third parties a budget-drain primitive (see
   [B-2](#b-2-imitating-the-classification-criteria-of-an-entity) and §9).
5. **`id` must be scarce and governed.** Nothing bounds the number of distinct `id`s
   in a block, and there is no block-level cap on prioritized gas, so the aggregate
   priority share of a block is exactly `Σ` per-entity budgets (see §8).

## Two-stage model

The feature has two clearly separated stages with different trust levels:

1. **Authoritative ordering — block formation (`gossip/c_block_callbacks.go`).**
   The single place that decides the final order. Every validator reproduces it
   deterministically from the same finalized block-start state. **This is the only
   stage that affects consensus.**

2. **Best-effort hints — txpool & emitter.** Used only to get prioritized
   transactions into the DAG / proposal quickly. A wrong or stale hint costs at
   most a little bandwidth; it can never change the block that results, because
   stage 1 re-queries authoritatively.

This separation is what makes the feature safe: the expensive/uncertain parts are
confined to the non-authoritative stage.

## Authoritative ordering (block formation)

In `c_block_callbacks.go`, after the base order is produced (scrambler for legacy;
`proposal.Transactions` for single-proposer) **and after**
`filterNonPermissibleTransactions`, a single pure transform is applied — gated by
`Upgrades.TransactionPriorities`:

```
prioritize(baseOrdered, vm@blockStartState, signer, config):
  1. classify every tx -> (level, weight, id)          // queries the registry
  2. per sender, build its chain: prioritized txs in nonce order forming a
     contiguous run from the block-start account nonce (stale nonce skipped,
     gap ends the run)
  3. walk chains greedily: repeatedly take the highest (level desc, weight desc,
     txhash asc) frontier tx whose entity budget (config.MaxGasPerEntityPerBlock)
     still fits it, and advance that chain; a frontier that does not fit blocks
     its chain
  4. emit selected txs in selection order: (level desc, weight desc, txhash asc),
     but each sender's txs kept in nonce order (a chain's frontier only advances
     in nonce order, so a higher-priority later nonce never overtakes its
     predecessor)
  5. result = [selected, in that order] ++ [base order minus the selected txs]
```

Because demoted/overflow and non-prioritized transactions stay in their original
base positions, "demote to normal pool" (legacy) and "keep proposer order"
(single-proposer) both fall out of the same code.

The same transform runs for **both** modes. In single-proposer mode this means the
proposer's order is **not trusted**: `c_block_callbacks.go` re-creates the priority
ordering and overrides it (hoisting prioritized txs to the front). The override can
**reorder** the proposed set but cannot **add** transactions the proposer omitted —
inclusion remains the proposer's prerogative, defended by turn rotation.

### Classifier seam

`prioritize` consumes a `Classifier` interface (`Priority(tx) (Priority, error)`),
allowing two interchangeable implementations selected by benchmark results:

- **Per-tx call (default):** one `getPriority` EVM call per transaction.
- **Native-filter fallback:** one call per block to fetch the filter criteria
  (accepted senders / targets / methods + weights/ids), then classify all
  transactions in native Go.

Both classify *all* transactions — restricting classification to a subset is **not**
an option, as it would void priority guarantees under high load.

## Emitter (best-effort hints)

A cached priority evaluator in the txpool (mirroring `subsidiesCheckerCache`)
provides a `(prioritized, id)` lookup against current head state.

**Piggyback model.** A validator must **not** emit an event *solely* because it
holds prioritized transactions it does not own under `isMyTxTurn`. The event-emit
decision and all throttling (`NoTxsThreshold`, `LimitedTpsThreshold`, stake-based
suppression) are unchanged. Prioritized transactions the validator is not the
turn-owner of are only *added* to an event that is already being emitted for other
reasons — capped per entity at `MaxPiggybackTxsPerEntityPerEvent`. This bounds duplication
across validators, prevents priority-only events, and avoids inducing low-stake
validators to emit.

In single-proposer mode the proposer's scheduler is biased so prioritized
transactions are offered first (and thus survive the gas/size cut), subject to the
same per-entity-per-event cap.

**Future: a new consensus removes the piggyback cap.** The
`MaxPiggybackTxsPerEntityPerEvent` cap exists only to bound the size of events in
the current consensus, where events are long-lived DAG members and one entity's
prioritized transactions would otherwise be duplicated N-fold across it (§6). Under
the forthcoming consensus, events are ephemeral — discarded after a few seconds —
so their size no longer needs to be constrained and the foreign-piggyback path and
its per-entity-per-event cap become unnecessary. Once that consensus lands, this
best-effort emitter limit can be dropped entirely; the authoritative per-block gas
budget (`MaxGasPerEntityPerBlock`) is independent of it and remains the sole rate
limit.

## Determinism & byte-compatibility

- **Fully gated** by `Upgrades.TransactionPriorities`. While OFF: no new state
  reads, no new bytes, **identical block hashes** to today. The flag is an optional
  feature toggled at epoch boundaries, like `SingleProposerBlockFormation` /
  `GasSubsidies`. All nodes must run a build that understands the flag before it is
  enabled.
- `prioritize` is a **pure total-order function** of (transaction set, registry
  state). Tie-break by transaction hash guarantees a total order.
- All ABI encode/decode is **hand-rolled with strict length and high-byte overflow
  checks**; fixed per-call gas caps.
- Each registry query runs inside `Snapshot` / `RevertToSnapshot` **per query**, so
  reads leak no warm-access entries, transient storage, refunds, or self-destruct
  marks into real execution, and one transaction's query cannot influence another's.
- The ordering EVM's block context uses **only consensus-derived values** (computed
  block time, computed randao, derived base fee, deterministic coinbase) — never
  wall-clock or node-local data.
- No output-affecting pass depends on Go **map iteration order**: senders are
  grouped via a map, but selection always takes the globally highest-priority
  eligible frontier, and transaction-hash tie-breaking makes that a strict total
  order.

## Security & risk analysis

This section enumerates the issues identified during design review, how each is
addressed, and the residual / accepted risk.

### 1. Per-tx registry query on the consensus critical path (DoS / liveness)

*Issue.* Classifying every transaction with an EVM call adds work to block
formation — the hot consensus path. Unlike subsidies (which only queries the small
zero-gas-price subset), this touches *all* transactions, so a flood of cheap
transactions multiplies the cost.

*Addressed.* Per-call gas is capped at a small fixed limit, bounding worst-case
work at `numTxs × cap`. The strategy is **benchmark-gated**: if the per-tx-call
latency is unacceptable at realistic block sizes, we switch to the native-filter
fallback (one call per block + Go classification) behind the `Classifier` seam.

*Residual / accepted.* Restricting classification to a subset is explicitly
rejected (it would void high-load guarantees), so we accept an O(numTxs) classifier
whose cost is held down by the chosen strategy.

*Measured.* `BenchmarkPrioritize` (in `ordering_bench_test.go`) runs the whole
`Prioritize` pass against a real in-memory Carmen state pre-populated with 10,000
dummy accounts (so the account trie has a representative depth) and a registry
deployed behind the production EIP-1967 proxy. Realistic blocks are bounded at
10,000 transactions; larger scenarios are not exercised. On an Intel i7-6600U
(2.60 GHz, single-threaded):

| transactions | per-tx EVM call (default) | native-filter (fallback) |
|---|---|---|
| 10     | 0.22 ms      | 0.008 ms |
| 100    | 1.36 ms      | 0.047 ms |
| 1,000  | 23.6 ms      | 0.35 ms  |
| 10,000 | 247 ms       | 4.9 ms   |

So the default classifier costs ≈ **25 µs per transaction** (≈ 250 ms for a
maximally full 10,000-tx block); the native fallback is ≈ 0.5 µs per transaction
(≈ 50× cheaper). Result mix barely moves the total (all-normal 311 ms, 10 % mix
278 ms, all-prioritized 242 ms at 10,000 txs) — the EVM query is paid for every
transaction regardless of outcome, confirming that the ordering passes are
negligible next to the query. 1 KiB of calldata per transaction adds ≈ 12 %
(236 ms → 265 ms at 10,000 txs).

*Decision.* Keep the **per-tx-call classifier as the default**: typical blocks are
far below the ceiling, where the cost is single-digit milliseconds, and it needs no
additional registry ABI. The ≈ 250 ms worst case only materializes for a block
saturated with 10,000 transactions; if blocks routinely approach that ceiling, the
native-filter fallback (≈ 5 ms) should be adopted. The `Classifier` seam is already
in place to switch without touching the ordering logic, and the benchmark's
`Native/*` arm tracks the fallback's lower bound.

### 2. Non-deterministic failure handling (chain split)

*Issue.* If validators disagree on what a failed/malformed query means, they
produce different blocks → fork.

*Addressed.* Hard rule: **any** per-tx query error, revert, malformed/wrong-length
result, or out-of-gas ⇒ the transaction is treated as **level 0 (non-prioritized)**;
the block is never aborted or skipped because of it. A `getPriorityConfig` failure
⇒ a fixed, documented fallback config. Registry absent while the flag is ON ⇒ all
transactions level 0. Because every node runs the query against the same
state/contract, all nodes reach the same outcome.

*Residual.* None expected; covered by tests.

### 3. EVM-context determinism

*Issue.* If the ordering EVM's block context contains any node-local value, equal
state could yield different priorities.

*Addressed.* The context is built solely from consensus-derived block fields. No
`time.Now()` or other local input.

*Residual.* None.

### 4. Per-query isolation / state residue

*Issue.* A registry read could leave residue (warm slots, transient storage,
refunds, self-destructs) that perturbs subsequent real execution, or an earlier
query could influence a later one.

*Addressed.* Snapshot + revert around **each individual** query (not once around
the loop), mirroring subsidies. A dedicated test asserts execution is byte-identical
with and without the ordering queries.

*Residual.* None expected; explicitly tested.

### 5. Single-proposer reorder vs. proposal consistency

*Issue.* In single-proposer mode we execute a reordered list while the signed
proposal hash covers the proposal order.

*Addressed.* The block hash derives from the executed (reordered) list, which is
intended. During implementation we verify nothing asserts "executed block txs ==
`proposal.Transactions` order" (LLR records, receipt/proposal cross-checks).

*Residual.* The proposer still controls *inclusion*; the override only reorders.
Accepted (turn rotation defends against a censoring proposer).

### 6. Eager-emit bandwidth amplification

*Issue.* Letting every validator emit prioritized transactions duplicates them
N-fold across the DAG and could be used to stress the network.

*Addressed.* The piggyback model forbids priority-only events and preserves all
existing throttling; per-entity-per-event caps bound a single entity's footprint in
any one event.

*Residual.* Some cross-validator duplication remains by design (that is the point —
faster inclusion). Bounded by the caps and unchanged emit-decision logic.

### 7. Registry admin as a consensus-critical trust anchor

*Issue.* The upgradeable, governed registry can grant any transaction front-of-block
placement and push everyone else behind it — i.e. sanctioned front-running and
de-facto reordering/censorship power over the chain.

*Addressed.* This is an inherent property of a governed ordering oracle, the same
trust model as the subsidies registry but with ordering power. Mitigations are
governance controls and transparency; documented prominently so operators
understand the trust placed in the registry admin.

*Residual / accepted.* The registry admin is trusted for ordering. Accepted as a
deliberate design choice.

### 8. Normal-transaction starvation

*Issue.* With no cap on the priority share of a block, prioritized transactions can
fill an entire block and push normal traffic to later blocks.

*Addressed.* Bounded by per-entity limits and registry governance.

*Residual / accepted.* No reserved space for normal traffic in this version
(explicit decision). Could be added later as a configurable priority-share cap.

### 9. Rate-limit bypass via `id` minting

*Issue.* The per-entity limit keys on the registry-returned `id`; an entity that can
induce distinct `id`s evades the limit.

*Addressed.* `id` integrity is the registry's responsibility; the node enforces the
limit faithfully given the returned `id`s.

*Residual / accepted.* Inherited from registry policy; documented assumption.

### 10. Tie-break grinding

*Issue.* Ties within equal `(level, weight)` are broken by transaction hash, which a
submitter can grind.

*Addressed.* Weight is the primary, registry-controlled lever; hash only orders
exact `(level, weight)` ties. Low impact.

*Residual / accepted.* Minor; documented.

### 11. Per-sender nonce vs. hoisting

*Issue.* Hoisting a high-nonce prioritized transaction ahead of a same-sender
lower-nonce non-prioritized one makes the hoisted one fail (nonce-too-high → skipped),
so only the earlier one lands — prioritizing it is what keeps it out.

*Addressed.* `prioritize` selects priorities per sender in nonce order (see
`selectPrioritized`): only the contiguous run of prioritized nonces starting at
the block-start account nonce can keep its priority, and the per-entity gas budget
is spent on a transaction only once its lower-nonce predecessors are selected. A
transaction whose predecessor is not prioritized (or does not fit the budget)
drops to its base-order position, where the predecessor runs first; a stale
prioritized nonce (below the account nonce) cannot execute and is skipped without
blocking later nonces. Because the budget follows nonce order, it is never spent
on a transaction that would only be nonce-too-high.

The prefix is emitted in selection order, which keeps each sender's selected
transactions in nonce order, so a sender's own run is never reordered into a
nonce-too-high position either.

*Residual / accepted.* None: a selected prioritized transaction always has its
lower-nonce predecessors selected and ordered ahead of it.

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
