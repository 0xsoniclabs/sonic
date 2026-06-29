# Transaction Priorities — Implementation Hand-off

This is a self-contained snapshot of the **Transaction Priorities** work so any
engineer (or a fresh session) can resume without re-deriving context. The
engineer-facing design rationale lives in
[`transaction_priorities.md`](./transaction_priorities.md); this file is the
implementation map and status.

## Goal

Let the network designate a configurable subset of transactions to be scheduled
ahead of others. An on-chain registry (like the subsidies registry) returns, per
transaction, `(level, weight, id)`:

- `level` — 0 = no priority, higher = earlier partition.
- `weight` — tie-break within a level (higher first).
- `id` — entity identifier, used for per-entity rate limiting.

Block order when enabled:
`[prioritized sorted by (level desc, weight desc, txhash asc)] ++ [rest in base order]`.

## Locked design decisions

1. **Both modes.** Authoritative ordering is in `gossip/c_block_callbacks.go` for
   legacy *and* single-proposer. The single proposer's order is **not trusted** —
   `c_block_callbacks.go` re-creates and overrides it.
2. **Single-proposer override = hoist priority only.** Prioritized txs move to the
   front (canonical order); non-prioritized txs keep the proposer's order. Override
   reorders but cannot force-*include* omitted txs.
3. **Rate limit via registry config call** `getPriorityConfig()` returning
   `maxTxsPerEntityPerBlock` (block formation) and `maxTxsPerEntityPerEvent`
   (emitter).
4. **Overflow = demote to normal pool.** Beyond the per-entity per-block limit, the
   lowest-weight excess txs lose priority and stay in base order.
5. **Query input** mirrors subsidies `chooseFund` (`from, to, value, nonce,
   calldata`) **plus `gas`** (tx gas limit). Priority is **orthogonal** to
   subsidies/bundles.
6. **Tie-break = transaction hash.**
7. **Classify ALL txs** (subset filtering rejected). **Best-effort cache** in
   txpool/emitter; block formation always re-queries authoritatively.
8. **Query strategy is benchmark-gated** behind a `Classifier` seam: default =
   per-tx `getPriority` call; fallback = one criteria fetch per block + native Go
   classification.
9. **No cap on the priority share of a block** (starvation bounded only by
   per-entity limits + governance).
10. **Emitter piggyback model:** never emit an event solely for non-owned priority
    txs; only add them to events already being emitted; preserve all throttling;
    cap per entity per event.

## Implementation order (with status)

1. [in progress] **Docs** — this file + `transaction_priorities.md` (with the
   Security & risk analysis section).
2. [todo] **Upgrade flag** `Upgrades.TransactionPriorities`.
3. [todo] **Registry package + query** (+ representative test contract).
4. [todo] **Benchmark gate** — pick per-tx-call vs. native-filter.
5. [todo] **Authoritative ordering** in `c_block_callbacks.go`.
6. [todo] **Emitter hints** (piggyback).
7. [todo] **Tests**, ending with the `tests/priority/` end-to-end demo.

## Files

### Add

- `gossip/blockproc/priorities/HANDOFF.md` (this file)
- `gossip/blockproc/priorities/transaction_priorities.md` (design doc)
- `gossip/blockproc/priorities/priorities.go` — query + `prioritize` building
  blocks. Mirror `gossip/blockproc/subsidies/subsidies.go`:
  - `type VirtualMachine interface { Call(from, to common.Address, input []byte, gas uint64, value *uint256.Int) ([]byte, uint64, error) }` (satisfied by `*vm.EVM`).
  - `type Priority struct { Level, Weight *big.Int; Id [32]byte }`; `IsPrioritized()`.
  - `type Config struct { MaxTxsPerEntityPerBlock, MaxTxsPerEntityPerEvent uint64 }`.
  - `GetPriority(upgrades, vm, signer, tx) (Priority, error)`,
    `GetConfig(upgrades, vm) (Config, error)` — hand-rolled ABI, strict length
    checks, fixed gas caps.
  - `Classifier` interface `Priority(tx) (Priority, error)` (per-tx vs. native-filter).
- `gossip/blockproc/priorities/registry/{registry.go, priorities_registry.sol,
  priorities_registry_abigen.go, priorities_contract.bin}` — mirror
  `gossip/blockproc/subsidies/registry/`. New fixed `GetAddress()`, selectors,
  per-call gas-limit constants, embedded `bin-runtime`, `//go:generate` directives.
  Reuse `gossip/blockproc/subsidies/proxy/` for deployment.
- `evmcore/priorities_integration.go` — cached priority checker, mirror
  `evmcore/subsidies_integration.go` (`newPriorityChecker(rules, chain, state,
  signer)` → cached `(prioritized, id)` lookup).
- `tests/priority/` — end-to-end acceptance test (+ a configurable priority
  registry test contract under `tests/contracts/`, modeled on
  `network_sponsor_configurable`).

### Modify

- `opera/rules.go` — add `TransactionPriorities bool` to `Upgrades` (after
  `TransactionBundles`, ~line 211) with doc comment.
- `opera/legacy_serialization.go` — serialize the flag (mirror `GasSubsidies`).
- `opera/validate.go` — allow toggling (mirror `GasSubsidies`).
- `gossip/c_block_callbacks.go` — apply `prioritize` after
  `filterNonPermissibleTransactions` (~line 277), gated by the flag, for both
  branches. Build the ordering EVM from `statedb` (~line 158) + consensus-only block
  context; per-query snapshot/revert; `GetConfig` once per block.
- `integration/makefakegenesis/json.go` — deploy proxy + implementation behind the
  flag (mirror the `if upgrades.GasSubsidies {…}` block ~lines 156-178).
- `api/ethapi/config.go` — `sysContracts["TRANSACTION_PRIORITY_REGISTRY_ADDRESS"]`
  (next to `GAS_SUBSIDY_REGISTRY_ADDRESS`, ~line 88).
- `gossip/emitter/txs.go` — piggyback eager emission in `addTxs`; per-entity
  per-event cap; do not change emit-trigger logic.
- `gossip/emitter/ordering.go` (+ `proposals.go` / `scheduler/` as needed) — bias
  candidate order so prioritized txs are offered first under the per-event cap.
- `evmcore/tx_pool.go` — wire the priority checker/cache (mirror subsidies wiring
  ~lines 355-362).

### Model from (read these first)

- `gossip/blockproc/subsidies/subsidies.go` — query pattern (`IsCovered`,
  hand-rolled ABI, `VirtualMachine`).
- `gossip/blockproc/subsidies/registry/registry.go` — registry package layout.
- `evmcore/subsidies_integration.go` — cached checker pattern.
- `gossip/scrambler/tx_scrambler.go` — base ordering this layers on top of.
- `gossip/c_block_callbacks.go:240-280` — the two-mode branch + base ordering seam.

## `prioritize` algorithm (authoritative, in c_block_callbacks)

```
prioritize(base []*types.Transaction, classifier, signer, cfg) []*types.Transaction:
  meta := []                         // iterate base order; no map-order dependence
  for tx in base:
     (level, weight, id) := classifier.Priority(tx)   // errors => level 0
     meta.append({tx, level, weight, id})
  prioritized := [m in meta if m.level > 0]
  // rate limit per id: keep top cfg.MaxTxsPerEntityPerBlock by weight (tie: txhash)
  group prioritized by id; per group sort by (weight desc, txhash asc); keep first N
  kept := union of kept-per-group
  sort kept by (level desc, weight desc, txhash asc)
  rest := [m.tx in base order if m.tx not in kept]    // demoted overflow + non-prio
  return kept.txs ++ rest
```

## Hard invariants (consensus-critical — do not violate)

- Gated by `Upgrades.TransactionPriorities`; OFF ⇒ byte-identical to today.
- `prioritize` is a pure, total-order function of (tx set, registry state).
- Any query error / malformed result / OOG ⇒ tx treated as level 0; never abort or
  skip the block. `GetConfig` failure ⇒ fixed fallback. Registry absent ⇒ all level 0.
- Ordering EVM block context = consensus-derived values only (no wall-clock).
- Snapshot + revert per **individual** query; verify no residue (warm list,
  transient storage, refunds, self-destructs) leaks into execution.
- No Go map-iteration-order dependence in any output-affecting pass.
- Single-proposer: block hash derives from the executed (reordered) list; verify no
  code asserts executed-order == proposal order.

## Open items

- Benchmark numbers → decide per-tx-call vs. native-filter; record in design doc §1.
- Confirm exact `getPriorityConfig` field set (room for future fields via
  length-versioned decode).
- Final fixed registry address + implementation address for genesis.
