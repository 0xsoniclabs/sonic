# Amsterdam / Glamsterdam support in Sonic

Assessment of the [Glamsterdam network upgrade (EIP-7773)](https://eips.ethereum.org/EIPS/eip-7773)
against Sonic's `Canto` hard fork.

Assessed against:

| Component | Version |
| --- | --- |
| go-ethereum fork | `0xsoniclabs/go-ethereum v0.0.0-20261013140340-66cf338cad69` (1.17.8) |
| Tosca | `0xsoniclabs/tosca v0.0.0-20261014112021-5893ca772cf7` (branch `simon/amsterdam`) |
| Carmen | `0xsoniclabs/carmen/go v0.0.0-20260925152836-62f625a466a9` |

`Upgrades.Canto` maps to `cfg.AmsterdamTime`, so everything below is reachable
once the fork is activated.

## Scheduled EIPs

| EIP | Sonic? | What's still missing |
| --- | --- | --- |
| **2780** Reduce intrinsic transaction gas | Yes — done | Live via `core.IntrinsicGas` and the `amsterdam` network-rule flag. `gossip/blockproc/bundle/gas.go` now delegates to `core.IntrinsicGas` instead of re-implementing it, and the envelope's value is threaded through so the EIP-2780 base cost applies. |
| **7610** Revert creation in case of non-empty storage | No | go-ethereum removed its implementation in 1.17.8 (#35581): a collision is a non-zero nonce or non-empty code again. The v21.0.1 fixtures pass without it. Nothing to do on the Sonic side. |
| **7688** Forward compatible consensus data structures | No | Beacon-chain SSZ containers. |
| **7708** ETH transfers emit a log | Yes — done | `evmcore` installs go-ethereum's `core.Transfer`/`core.CanTransfer` as the block context's hooks instead of re-implementing them, so transaction, `CALL`/`CALLCODE` and `CREATE`/`CREATE2` value transfers emit the log under Canto. Tosca emits the SELFDESTRUCT log in both LFVM and SFVM. `evmwriter.setBalance` changes a balance without a counterparty and is out of the EIP's scope, so SFC-driven balance changes remain invisible to log consumers. |
| **7732** Enshrined Proposer-Builder Separation | No | Consensus layer; Sonic uses Lachesis. |
| **7778** Block Gas Accounting without Refunds | Yes | go-ethereum's `GasPool` implements it. Sonic's header `GasUsed` is a sum of post-refund receipt gas (`gossip/blockproc/evmmodule/evm.go`), not `GasPool.Used()` = `max(execution, state)`. The interaction with Sonic's `ChargeExcessGas` needs a decision. |
| **7843** SLOTNUM opcode | Yes | Tosca implements the opcode and reads `evm.Context.SlotNum`. Sonic never sets `SlotNum`, so SLOTNUM returns 0. Needs a Sonic definition of "slot", plus a header field if it has to be verifiable. |
| **7928** Block-Level Access Lists | Optional | Carmen's `Finalise` returns `nil` and ignores the block-access index; Sonic's block format has no BAL hash. Full support means Carmen-side BAL construction plus a Sonic header and consensus change. Only needed if Sonic wants to expose BALs. |
| **7954** Increase Maximum Contract Size | Yes — done | `GetVmConfig` clears `MaxCodeSize`/`MaxInitCodeSize` under Canto so go-ethereum's 64 KiB / 128 KiB limits apply. `sfvm.Config.MaxCachedCodeSize` is still 48 KiB, so contracts between 48 and 64 KiB skip the analysis cache (performance only). LFVM cannot run code larger than 65 535 bytes, since its instructions carry 16-bit arguments; production uses SFVM. |
| **7976** Increase Calldata Floor Cost | Yes — done | Live via `core.FloorDataGas`, which `bundle/gas.go` now calls directly. |
| **7981** Increase Access List Cost | Yes — done | Live via `core.IntrinsicGas`, which `bundle/gas.go` now calls directly. The bundle-only access-list marker is repriced with it. |
| **7997** Deterministic Factory Predeploy | No | go-ethereum 1.17.8 dropped the fork-time insertion of the factory (#35458); it only remains in `core.SystemContractAllocs`, the genesis alloc of dev mode and the simulated backend. Sonic follows upstream and does not insert it. |
| **8024** Backward compatible SWAPN, DUPN, EXCHANGE | Yes — done | Tosca implements all three in LFVM and SFVM, including gas and stack usage. Nothing on the Sonic side. |
| **8037** State Creation Gas Cost Increase | Yes | `vm.GasBudget` is plumbed through, `CostPerStateByte` is set, and Tosca passes `StateGas`/`StateGasCharged` through its adapter. `evmcore.ProcessParentBlockHash` now provisions the same system-call reservoir as go-ethereum. Missing: block-level state-gas accounting (see EIP-7778). The subsidies and priorities registry queries keep an empty reservoir by design, and the `evmwriter` precompile keeps charging its storage and code writes as execution gas, see below. Tosca's CREATE charges the account only for an empty destination and runs the create-frame precheck itself, and returning frames repay the caller's spilled state gas, matching go-ethereum 1.17.8. Note that this repricing made the fake-genesis network initialization transaction jump from under 3M to about 7.8M gas, since depositing contract code and creating storage slots are now charged state gas. |
| **8038** State-access gas cost increase | Yes | Handled in go-ethereum and Tosca gas tables. Sonic's hardcoded `evmwriter` gas constants do not track the repricing, by design, see below; `bundle/gas.go` no longer hardcodes any. |
| **8045** Exclude slashed validators from proposing | No | Consensus layer. |
| **8061** Increase exit and consolidation churn | No | Consensus layer. |
| **8246** Remove SELFDESTRUCT Burn | Yes | Handled in go-ethereum's interpreter and in Tosca's `geth_adapter`; the obsolete `EmitLogsForBurnAccounts` stub was removed from `CarmenStateDB`. Missing: Carmen zeroes the balance on `Suicide` and deletes self-destructed accounts at `EndTransaction`, where go-ethereum keeps a non-zero balance as a balance-only account, see `AMSTERDAM_FIXTURE_ISSUES.md`. |
| **8282** Builder Execution Requests | No | Consensus layer / PBS. |

## Networking and informational EIPs

| EIP | Sonic? | Why |
| --- | --- | --- |
| **7975** eth/70 - partial block receipt lists | No | devp2p; Sonic has its own p2p stack. |
| **8070** eth/72 - Sparse Blobpool | No | devp2p; Sonic blob transactions carry no blobs. |
| **8136** Cell-Level Deltas for Data Column Broadcast | No | Blob data columns. |
| **8159** eth/71 - Block Access List Exchange | No | Depends on EIP-7928. |
| **8189** snap/2 - BAL-Based State Healing | No | Sonic syncs via Carmen, not snap. |
| **7904** Compute Gas Cost Analysis | No | Informational, no implementation. |
| **8261** Gas Limit Schedule | No | Informational; Sonic sets block gas limits through `opera.Rules`. |

## Work items, by risk

1. **EIP-7843 SLOTNUM.** Cheap to wire once "what is a slot on Sonic" is
   decided. The candidates and a recommendation are collected below.
2. **EIP-8246 in Carmen.** With the Carmen StateDB, 84 Amsterdam subtests still
   fail because Carmen deletes self-destructed accounts together with their
   balance. Needs a Carmen change and a way to signal Canto to it.

`evmcore.PreExecution` mirrors go-ethereum's `core.PreExecution`, reduced to the
part Sonic has: the EIP-2935 system call. Sonic has no beacon block root and
builds no block access lists. The sites that used to guard
`ProcessParentBlockHash` with their own `IsPrague` check --
`StateProcessor.ProcessWithDifficulty`, `StateProcessor.BeginBlock`, the
tracing helper `applyPreBlockSystemCalls` (used by `traceBlock`,
`stateAtTransaction` and `replayBlock`) and the `eth_simulateV1` block loop --
now call `PreExecution` instead, matching where go-ethereum calls its own.
`DoCall` runs no system call, as in go-ethereum.

`evmcore.ProcessParentBlockHash` is Sonic's only EIP-style system call, and it
now derives its budget from a local `systemCallGasBudget` that mirrors
go-ethereum's function of the same name: 30M execution gas, plus a
`SYSTEM_MAX_SSTORES_PER_CALL * CostPerStateByte * StorageCreationSize`
state-gas reservoir once Amsterdam is active. Without it the history storage
contract's slot write spilled into the call's execution gas.

The remaining direct `evm.Call` sites, the subsidies registry's `getGasConfig`
and `chooseFund` queries and the priority registry's `getPriority` and
`getPriorityConfig` queries, keep an empty reservoir on purpose. Their gas
limits are work bounds rather than forwarded budgets: the subsidies limits are
also what `Sponsorship.Overhead` charges to the sponsored transaction, so a
reservoir on top would let the registry consume more than is billed, and the
priority limits bound per-transaction work on the block-formation path. Both
registries are `view`, so no state gas arises in the first place; with an empty
reservoir a registry that did write state would pay for it out of the same
limit instead of beyond it.

The `evmwriter` precompile's gas summation is deliberately left on the
pre-Amsterdam schedule. It charges `CallValueTransferGas` for `setBalance` and
`incNonce`, `CreateGas` plus `CreateDataGas + MemoryGas` per code byte for
`copyCode` and `swapCode`, and `SstoreSetGasEIP2200` for `setStorage`, all as
execution gas. So the accounts, code and storage slots it creates are not charged
EIP-8037 state gas, and none of the constants track the EIP-8038 repricing.

This is intentional rather than a gap. The precompile reverts for every caller
other than `driver.ContractAddress`, and the `NodeDriver` only forwards calls
from its foundation-held owner, so the prices are not reachable by arbitrary
callers and cannot be used to grow state cheaply. Repricing them would instead
be a consensus change to a Sonic-specific contract with no EIP mandate, and one
that could make an operation such as `upgradeCode` on a large contract cost more
than the per-transaction gas cap allows: charging a 24 KiB code body at
`CostPerStateByte` adds about 37.6M state gas, well past the default
`MaxEventGas` of about 25M. The behaviour is documented on
`evmwriter.PreCompiledContract`.

`gossip/blockproc/bundle/gas.go` appeared under EIP-2780, EIP-7976 and EIP-7981
and has been fixed once, by deleting the duplicated formulas rather than adding
Amsterdam branches to them: `CalculateEnvelopeGas` now calls `core.IntrinsicGas`
and `core.FloorDataGas`, and only the bundle-specific sum over the bundled
transactions' gas limits remains local. `opera.Upgrades` is threaded through
`ValidateEnvelope` down to the gas computation, which maps `Upgrades.Canto` to
`params.Rules.IsAmsterdam`. The envelope's value is threaded through as well,
since EIP-2780 prices it. The tests no longer pin a revision: they run against
both schedules.

`evmcore.Transfer` and `evmcore.CanTransfer` were deleted rather than extended
with an Amsterdam branch: they were copies of go-ethereum's `core.Transfer` and
`core.CanTransfer` with the `*params.Rules` argument dropped, and the block
context now points at the originals. Receipt blooms and log indices needed no
change, since both are derived from `statedb.GetLogs`. The two consumers of
`onNewLog`, the driver transaction listener and the version watcher, filter on
`driver.ContractAddress` and therefore ignore the new logs from
`params.SystemAddress`. `tests/eip7708_test.go` covers transactions, contract
creations, nested calls, self transfers, zero-value transfers, reverts, and the
`eth_getLogs` path, against both the Brio and the Canto schedule.

One overlap is left as-is: `eth_simulateV1` with `traceTransfers` synthesises its
own ERC-7528 transfer logs from the canonical `0xEeee...eEeE` address, so under
Canto a simulated value transfer reports both that log and the real EIP-7708 one.
go-ethereum's simulator behaves the same way, and the two are told apart by their
log address.

Enabling Canto in the integration test network exposed one unrelated gap:
`IntegrationTestNetSession.EndowAccounts` hardcoded a gas limit of 21000 for its
value transfers. EIP-2780 keeps a plain transfer at 21000, but EIP-8037 charges
state gas for creating the recipient account, which takes a transfer to a fresh
account to about 204600 gas. The helper now estimates the gas limit through the
node once per call, using a fresh recipient so the estimate is an upper bound for
every address in the batch.

Remaining bundle-side follow-ups:

- The integration tests under `tests/bundles`, `tests/gas_fee`,
  `tests/priorities` and `tests/regressions` build bundles with the builder's
  default (pre-Canto) gas schedule. They pass today because none of those
  networks enable Canto; enabling it requires passing the network's upgrades via
  `builder.WithUpgrades`.

## What a slot could be on Sonic

EIP-7843 is thin. `SLOTNUM` (0x4b) pushes one `uint64` for a fixed 2 gas, and
the EIP states no monotonicity or validity rule for the value. Its motivation is
that contracts otherwise have to hardcode a slot length and derive the slot from
`TIMESTAMP`, or pay for a beacon-root lookup; its rationale adds that the value
should be a header field rather than an external input, so that zkVM proving
keeps working. The open question for Sonic is therefore whether `SLOTNUM` should
be a clock, ticking even when no block is produced, or a label, a per-block
index. Everything else follows from that choice.

| Candidate | Where it lives today | Monotone | New information | Cost |
| --- | --- | --- | --- | --- |
| `0`, the status quo | -- | n/a | None, and a silent trap for ported contracts | None |
| Block number | `EvmHeader.Number`, set on every path | Strictly | None | One line |
| Lachesis epoch | `inter.Block.Epoch`, `EvmHeader.Epoch`, RPC `epoch` | Non-decreasing | The epoch for 2 gas instead of a cold SFC call | One line, plus two fixes |
| Proposal turn | `Payload.LastSeenProposalTurn` | Within an epoch only | Missed proposal opportunities | Plumbing, plus composition with the epoch |
| Atropos frame | Not stored on the block | Within an epoch only | None meaningful | High |
| Atropos Lamport time | `blockCtx.Atropos.Lamport()` | Within an epoch only | None meaningful | One line |
| `(time - genesis) / SLOT_DURATION` | Derivable | Non-decreasing, and repeats | None | Medium |
| Nanosecond block time | `inter.Timestamp`, hidden in `extraData` | Strictly | Sub-second block time | One line |
| Global cumulative turn counter | Does not exist | Strictly | True slot semantics | Consensus change |

The **block number** is the only candidate that is correct by construction on
every header-building path and under both block-formation modes. It reads as
"Sonic produces one block per slot and never misses one": strictly increasing,
already committed to the block hash, and free of any divergence between the
proposer's speculative execution and block execution. It adds no information; it
only stops `SLOTNUM` from being a zero-valued trap for contracts ported from
Ethereum.

The **Lachesis epoch** is already computed for every block, carried in
`EvmHeader.Epoch`, and exposed over RPC as `"epoch"`. Contracts reach it today
only through `SFC.currentEpoch()`, which costs a cold account access plus a cold
`SLOAD` plus the call overhead, and EIP-8038 makes that more expensive under
Canto, not less. Handing the same number over for 2 gas is exactly the cost
argument the EIP itself makes. Epochs are the unit of SFC reward accounting,
validator-set changes and rules changes, so "has the epoch rolled over" is a
real on-chain question. Three points would have to be settled first: whether the
value means `SFC.currentEpoch()` or `currentSealedEpoch`, which diverge inside
the sealing block, since `sealEpoch` runs as an internal transaction of a block
whose header epoch is still the old one; that the epoch is not part of the
hashed header today, so it is replay-derivable but not header-verifiable; and
that the mapping inverts Ethereum's hierarchy, where the slot is the fine unit
and the epoch the coarse one, so anything computing `genesis + 12 * slot`
breaks.

The **proposal turn** is structurally Sonic's slot. `inter.GetProposer` assigns
exactly one proposer per turn, and a turn ends either with a proposal or with a
timeout after `TurnTimeoutInFrames` frames; `IsValidTurnProgression` already
enforces a strict increase. It is also the only candidate that keeps ticking
while the chain is idle: mainnet sets `MaxEmptyBlockSkipPeriod` to one minute,
so the block number is explicitly not a clock during quiet periods, while turns
continue to advance. Three things block it. The DAG resets at epoch boundaries
and `GetProposer` mixes `epoch || turn`, so the turn is monotone only within an
epoch and would have to be composed as `epoch << 32 | turn`, which destroys
arithmetic on the value. It only exists while `SingleProposerBlockFormation` is
enabled, and `opera/validate.go` documents that flag as freely toggleable. And
it is not plumbed: `extractProposalForNextBlock` returns the proposal, the
proposer and the time, and drops the turn, which lives in the payload of the
carrying event.

A **time-derived slot** rejects itself. It reintroduces the hardcoded slot
length the EIP exists to abstract away, and Sonic has no fixed cadence in the
first place: the emitter `Interval`, 600 ms by default, is a mutable rules
parameter. At a one-second slot several blocks share a slot number.

The **nanosecond block time** is the only candidate that exposes something
contracts cannot obtain today. `inter.Timestamp` is nanoseconds since the Unix
epoch and block times are forced to strictly increase, but the Ethereum header
truncates to whole seconds and `inter.EncodeExtraData` stashes the remainder in
`extraData`, which the EVM cannot read. `TIMESTAMP` therefore repeats across
consecutive blocks on a chain producing roughly one block per second. That is
real value, but it is a timestamp and not a slot index, and shipping it under
the name `SLOTNUM` would mislead every piece of Ethereum tooling. If sub-second
time is wanted, it deserves its own opcode or precompile.

The recommendation is to ship the **block number** with Canto and to treat
exposing the epoch as a separate decision. The block number is the only
definition that holds under both block-formation modes, is defined on every
header-building path, and needs no new header field or consensus surface. The
epoch has the best value per line of the lot and is worth exposing, but not
under the name `SLOTNUM`: binding the coarse unit to the opcode Ethereum uses
for the fine one reads fine in a specification and bites integrators later, and
a Sonic-specific accessor costs about the same. The global cumulative turn
counter is the only faithful implementation of what EIP-7843 means, and is worth
building only if `SingleProposerBlockFormation` becomes permanent.

Whatever is chosen, three implementation notes apply. `SlotNum` is plumbed but
never set: `evmcore.NewEVMBlockContextWithDifficulty` builds the
`vm.BlockContext` without it, and Tosca reads it through
`geth_adapter/adapter.go` as `SlotNumber`, so one field in that struct literal
is the whole EVM-side change. For any value other than the block number,
`scheduler.BlockInfo.toEvmBlock` and `priorityQueryHeader` both build an
`EvmHeader` without an `Epoch`, so the field feeding `SlotNum` has to be
populated there as well; otherwise the proposer's scheduling pass and the
priority queries observe a different `SLOTNUM` than block execution does, which
changes which transactions fit into the block. And making the value
header-verifiable means adding `SlotNumber` to `inter.Block.GetEthereumHeader`,
which changes block hashes and therefore has to be Canto-gated, along with a
matching optional field in `blockData` and in `ibr.LlrFullBlockRecord`, and in
the header reconstruction in `tests/block_verifiability_utils.go`.
