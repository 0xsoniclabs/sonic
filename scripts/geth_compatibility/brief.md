{{- /*
The brief that -prepare writes for the reviewers of a go-ethereum update. Keep
the facts about Sonic current: update them when an upgrade goes live, or when
the head introduces a new one and its documents. The group hints are keyed by
the categories of prepare.go.
*/ -}}
# go-ethereum update {{if .Verify}}verification{{else}}review{{end}}: brief

Sonic `{{.Head}}` moves its go-ethereum fork from `{{.BaseGeth}}` (Sonic
{{.Base}}) to `{{.HeadGeth}}`. The tool `scripts/geth_compatibility` listed every
go-ethereum declaration that `sonicd` and `sonictool` can reach and whose source
differs: a finding. Each finding needs a verdict, a consensus classification
and a note. You {{if .Verify}}verify{{else}}review{{end}} one group of
findings; your prompt names it. This round holds only the findings that need
{{if .Verify}}a second look{{else}}a reviewer{{end}}; the others already have
verdicts in the review file.

## Your files

All in `{{.Dir}}`:

- `<group>.findings.txt`: your findings. Each starts with
  `=== <status> <key>`, then `used by head|base through:` and a path from the
  symbol back to `main` or `init`, possibly `on the consensus path of head|base
  through:` and a path back to a consensus root, then a unified diff of the
  declaration. Comments and blank lines outside raw string literals are
  stripped from the diffed source, and constants are compared by value. A
  declaration starts with the imports it refers to, as `import <name> "<path>"`
  lines. `added` means reachable only in the head,
  `removed` only in the base; check the sources to see whether the symbol exists
  on the other side. In the paths, `called through I on T` is a call through an
  interface on a type converted to it, and `exported method ..., handed to rpc`
  is a method that the RPC server calls by reflection.
- `<group>.rows.md`: a table row per finding. Fill in its Verdict, Consensus
  and Note cells. Do not change the Symbol, Base and Head cells, and do not add
  or remove rows. Write a `|` inside a note as `\|`.
{{- if .Verify}}

## Your task

Another reviewer filled in these rows. Each one either claims that its change
can alter consensus results, with `live` or a hardfork name as Consensus, or
asks a human a question, as a `todo` row. That reviewer worked from diffs; check
every claim in the sources:

- Read both versions of the declaration and the Sonic code that reaches it, and
  confirm the gating.
- A consensus claim: does the change alter EVM semantics, gas, the state
  transition, precompile results, which transactions a block skips, or block or
  receipt contents? From which upgrade on?
- A claimed regression of output users see, such as RPC results or traces: find
  it in the code.
- A question: check whether Sonic already answers it, see
  [Resolved by Sonic](#resolved-by-sonic).

Keep a row you confirm, and sharpen its note if needed. Correct a row you
refute: its verdict, classification and note. A confirmed problem stays `todo`,
and its note says what you established.
{{- end}}

## Sources (read-only)

- Sonic at the head: `{{.HeadTree}}`
- go-ethereum at the base: `{{index .BaseModules "github.com/ethereum/go-ethereum"}}`
- go-ethereum at the head: `{{index .Modules "github.com/ethereum/go-ethereum"}}`
- Tosca, the EVM implementations: `{{index .Modules "github.com/0xsoniclabs/tosca"}}`
- Carmen, the state database: `{{index .Modules "github.com/0xsoniclabs/carmen/go"}}`
- Read first, where the head has them: the documents of the upcoming upgrade
  among its top-level Markdown files, which say which EIPs Sonic takes, the
  open items, and the fixture results.

A module row names the source directories of both versions of the module.

## Facts about Sonic

- Upgrades (`opera/rules.go`): go-ethereum forks up to Cancun are always on.
  Prague is enabled by `Upgrades.Allegro`, Osaka by `Upgrades.Brio`. Allegro
  and Brio are live (`opera/validate.go`: Brio cannot be disabled); Canto is
  the upcoming upgrade and not live anywhere. So only changes gated to a fork
  after Osaka are free of risk for existing blocks; a change of Prague or Osaka
  behaviour, or an ungated one, alters how existing blocks replay.
- Never enabled: Verkle and EIP-4762, binary tries (UBT), beacon block roots,
  blob data (Sonic blob transactions carry no blobs), the devp2p eth and snap
  protocols, the beacon chain, EIP-7928 block-level access lists.
- EVM (`opera/vm_config.go`): Tosca executes blocks, LFVM before Brio and SFVM
  from Brio on. go-ethereum's own interpreter (`core/vm` instructions, jump
  tables, gas tables, stack, memory) does not execute blocks, but it runs for
  tracing (`InterpreterForTracing`, `debug_trace*`, `trace_*`), for
  `eth_simulateV1` (`api/ethapi/simulate.go` builds a `vm.Config` without
  Interpreter and without Sonic's hooks), and for `eth_call` and
  `eth_estimateGas` with state overrides holding oversized code. Its changes
  are therefore visible in these RPCs, and it must agree with Tosca. Tosca's
  `geth_adapter` calls back into go-ethereum's EVM (`EVM.Call`, `StaticCall`,
  `DelegateCall`, `CallCode`, `Create`, `Create2`) for every nested call and
  creation, so `core/vm/evm.go`, the precompiles and `core` (state transition,
  `IntrinsicGas`, `GasPool`, `Message`) run in production.
- Block processing (`evmcore`, `gossip/c_block_callbacks.go`): a transaction
  that fails before execution is skipped and left out of the block, so the
  order of the checks and of the gas pool debits in `core` decides which
  transactions a block contains.
- State is Carmen (`inter/state`, `evmcore`), not go-ethereum's `core/state`,
  `trie`, `triedb` or `core/rawdb`. Check how Sonic reaches a storage symbol
  before calling it active: `sonictool`, for instance, uses go-ethereum's
  `node` and `rawdb` for account management.
- The fork carries Sonic patches: files named `sonic_*.go`, and `vm.Config`
  hooks such as `ChargeExcessGas`, `IgnoreGasFeeCap`,
  `InsufficientBalanceIsNotAnError` (set before Canto, cleared from Canto on),
  `SkipTipPaymentToCoinbase`, `StatePrecompiles`, `Interpreter`,
  `InterpreterForTracing`, `MaxTxGas`, `MaxCodeSize` and `MaxInitCodeSize`
  (the last three set from Brio on). If a diff touches code around them, check
  that they still work.

If one of these facts is wrong, say so in your reply.

## Verdicts

Exactly one per row:

- `inactive`: the change only affects behaviour Sonic does not run: a fork or
  feature Sonic never enables, or code that is reachable on paper but that you
  verified Sonic never executes. The note says why, such as "only used by
  pathdb; Sonic state is Carmen".
- `refactor`: you verified that the behaviour is unchanged: renames, extracted
  helpers, moved code, types or parameters changed without effect, or new
  parameters that every caller passes with the old behaviour.
- `adopted`: the behaviour changes and Sonic accepts it. Typically a change
  gated to the upcoming upgrade for an EIP that the documents above mark as
  taken, or a fix without consensus or user-visible impact. The note names what
  covers it: the EIP and its gate, and the test, such as the exec-spec fixtures
  of the upgrade passing on geth VM and SFVM, or a Sonic test you found.
- `todo`: a human has to decide. Mandatory for:
  1. behaviour changes that are not gated to the upcoming upgrade and affect
     consensus (EVM semantics, gas, state transition, transaction skipping,
     precompile results, block or receipt contents) or the traces of existing
     blocks;
  2. changes that bear on an open item of the upcoming upgrade, or that are
     active under it although its documents mark the EIP as not taken or
     optional;
  3. changes of RPC or tracer behaviour that users see (JSON fields, errors,
     defaults, limits), unless clearly a bug fix without compatibility concerns;
  4. module rows: always `todo`, with a note on what changed between the two
     versions;
  5. anything you could not verify. Never give a verdict on a guess.

## Consensus

The consensus path is the code whose results all nodes have to agree on: block
formation and execution, event validation and genesis. A finding is on it if
its findings entry shows `on the consensus path`; its Consensus cell is then
empty, otherwise `-`. Classify every `adopted` and every `todo` row on the
path; `refactor` and `inactive` rows need no classification.

- `live`: the change can alter consensus results under upgrades that are
  already live. This includes `adopted` rows for which Sonic compensates in its
  own code; the note says how.
- the lowercase name of a Sonic upgrade, such as `canto`: it can alter results
  only once that upgrade is active, as code gated to it that runs on the block
  path, or state that is always zero before it. Use the Sonic name, not the one
  of the go-ethereum fork it enables.
- `no`: it is on the path but cannot alter results, for instance tracing,
  performance, metrics, error strings, or go-ethereum's interpreter where Tosca
  executes blocks.

Classify a row off the path too if its consensus problem lies outside the
changed code, for instance a call that is missing on the path, or an open item
of the upcoming upgrade that the row tracks.

## Notes

One line, at most about 300 characters, specific: the EIP, the gating
condition, the Sonic caller that makes the code run, or the test. A `todo` note
starts with `OPEN:` and the question, followed by `Found:` and what you
established; without `OPEN:` the row counts as not reviewed. Related rows, such
as a refactor spanning many functions, get consistent verdicts and may share a
note. A note that starts with `was <verdict> before the last change:` is an
earlier review of a previous version of the finding; use it, but check the
current diff.

## How to work

- Read the diff of each finding. If it is not conclusive, read the full
  declaration in the base and the head, check the gating condition in the head,
  and check how Sonic reaches and uses it: the path, and a search of the Sonic
  head.
- Work read-only: read and search files, and compare them. Do not build, test
  or run anything, including `geth_compatibility`, and do not change any file except
  your rows file.

## Resolved by Sonic

Before you leave a row `todo`, check whether Sonic's own code already answers
its question:

- a fix on the head, with a test that pins the behaviour;
- a setting that makes the changed code unreachable, such as the forks Sonic
  enables, its VM configuration, or the way it decodes and builds transactions.

Look at the Sonic callers in the path, at the tests next to them, and at the
fork settings. If the code answers the question, give the row its final verdict
and name the code and the test in the note. If it answers part of it, the note
says what is resolved and what remains `OPEN:`. A question that is a decision
stays `todo` even if the change is harmless: accepting a new behaviour, an RPC
format, or a dependency.

## When you are done

Check your rows file: every row has a verdict, every verdict other than `todo`
has a note, every `todo` note starts with `OPEN:`, and every `adopted` row on
the consensus path has a classification. Then reply with a short summary: the
number of rows per verdict, every `todo` row with its question, and anything
that concerns other groups or contradicts this brief.

## Group hints

### core

go-ethereum `core` (state transition, `IntrinsicGas`, `FloorDataGas`,
`GasPool`, `Message`, `TransactionToMessage`, `Transfer` and `CanTransfer`),
`core/types` (headers, blocks, receipts, blooms, transactions),
`core/types/bal` and `core/stateless`. Much of `core` runs in production:
Sonic's `evmcore` uses `core.ApplyMessage`, `IntrinsicGas`, `FloorDataGas` and
`GasPool`; check the Sonic callers in `evmcore`, `gossip/blockproc` and
`api/ethapi`. A transaction that fails before execution is skipped, so check
whether a change moves a check before or after the gas pool debit or the gas
purchase, and how Sonic sets the `Message` flags. Changes to header or block
fields or hashing matter for block hashes and RPC output.

### misc

go-ethereum packages without a group of their own, such as `p2p/*`,
`common/lru` and `rlp`. Sonic runs its own protocols but uses go-ethereum's p2p
server, RLPx, discovery v5 (v4 only with `--discovery.v4`), enode, ENR and NAT,
so these run in production; changes to discovery or the handshake can affect
connectivity with nodes running older Sonic versions.

### modules

Rows `module <path>` compare the versions of dependency modules, other than
go-ethereum, that provide reachable code. They are always `todo`. Diff the two
source directories the row names, and note the base and head version and a
short summary of what changed in the code Sonic reaches: major version changes,
removed or replaced modules, notable behaviour changes. Keep it proportionate, a
few lines of investigation per module.

### params-tracing

go-ethereum `params` (`ChainConfig`, `Rules`, fork checks, protocol constants,
embedded chainspec JSON files), `params/forks` and `core/tracing` (the `Hooks`
struct and the gas and balance change reasons). Sonic builds its own
`ChainConfig` in `opera/rules.go`; check which fields and `Rules` flags Sonic
sets, and whether new fields default safely (nil, false, zero) for Sonic.
Changed constants matter if code Sonic runs uses them under the forks Sonic
enables. The embedded chainspecs are Ethereum networks: check whether Sonic
loads them at all. For `core/tracing`, check Sonic's tracers and hooks
(`txtrace`, `evmcore`, `api/ethapi`, the tracers it registers) for changed hook
signatures or semantics.

### rpc-api

go-ethereum `rpc` (server, client, codecs, websocket and HTTP handling, limits,
error codes), `node` (the node stack, HTTP and WS servers, auth, config),
`ethclient`, `internal/web3ext` (console extensions), `accounts/*`, `signer/*`
and `common/hexutil`. Sonic serves its JSON-RPC through go-ethereum's `rpc`
server and `node`, so changes in request handling, batching, limits, timeouts,
error messages or JSON encoding are visible to Sonic's users: they are `todo`
unless clearly a bug fix without compatibility concerns. Check Sonic's node
setup (`config/`, `cmd/sonicd/`) for the node features and flags it uses.
`accounts/usbwallet` and external signers are reachable through the account
manager; check whether Sonic enables them.

### state-storage

go-ethereum `core/state`, `trie`, `trie/bintrie`, `triedb/*`, `core/rawdb`,
`ethdb/*` and `core/filtermaps`. Sonic's state is Carmen, so most of this
should be `inactive`, but verify each path: the `used by` path shows how Sonic
reaches it, for instance `sonictool`'s account management through
go-ethereum's `node.Node` and `rawdb`, RPC handlers, or the transaction pool.
Mark a row `inactive` only with a concrete reason, such as "bintrie is only
used by UBT and Verkle, never enabled", or "the rawdb freezer is only opened by
node.OpenDatabase*, which Sonic never calls" after a search. Where go-ethereum
storage code does run in Sonic (`memorydb`, `rawdb` accessors used by
`sonictool` or the keystore), assess the change normally.

### tracers-log

go-ethereum `eth/tracers/*` (native tracers such as `callTracer`,
`prestateTracer` and `4byte`, the struct logger, live tracers, `TraceConfig`),
`log` (handlers, formatting, verbosity), `metrics` and `internal/telemetry`.
Sonic exposes go-ethereum's tracers through its `debug_trace*` RPCs (check
`api/ethapi` and how Sonic registers and calls tracers), so a change in tracer
output is visible to users: `todo` unless clearly a bug fix without
compatibility concerns. Tracer changes for features of the upcoming upgrade are
`adopted` if they only show once it is active. Log handler changes affect
Sonic's log format and flags; check how `cmd/sonicd` configures logging.

### vm-evm

go-ethereum `core/vm` outside the instruction and gas tables: `evm.go` (`Call`,
`Create`, precompile dispatch, block and transaction context), `contract.go`,
`interpreter.go`, the gas accounting types, errors, jump destinations, and the
fork's Sonic files (`sonic_*.go`). `evm.go` runs in production, because Tosca's
adapter calls back into it, so check every behaviour change for fork gating and
for the Sonic hooks in `vm.Config`. Where a signature or semantics changed,
check that Tosca's `geth_adapter` and Sonic's `evmcore` use it consistently.

### vm-gas

go-ethereum `core/vm` `gas_table.go`, `operations_acl.go`,
`operations_verkle.go` and `gas.go`. go-ethereum's interpreter does not execute
blocks, but check whether Tosca or Sonic call any of these gas functions
directly. The question per change: is it gated to the upcoming upgrade, or does
it alter gas for earlier forks, which shows in traces and `eth_simulateV1`?
`operations_verkle.go` is EIP-4762, which Sonic never enables. The documents of
the upcoming upgrade say which EIPs Tosca implements and give the fixture
results.

### vm-instructions

go-ethereum `core/vm` `instructions.go`, `eips.go`, `jump_table.go` and
`opcodes.go`. go-ethereum's interpreter does not execute blocks (Tosca does),
but runs for tracing, `eth_simulateV1`, and `eth_call` with oversized code
overrides. So a change matters if it changes the traced or simulated behaviour
of existing blocks (not gated to the upcoming upgrade), or if it diverges from
what Tosca does under the same fork. Check gated changes against the documents
of the upcoming upgrade and whether Tosca's LFVM and SFVM implement the same.
Also check whether instruction code is shared with Tosca or called outside the
interpreter (the exports in `sonic_tosca_integration.go`).

### vm-precompiles

go-ethereum `core/vm` `contracts*.go` and `precompile_cache.go`, and the
`crypto` and `consensus` packages. Precompiles run in production, because Tosca
calls back into go-ethereum's EVM for calls, so any change of a precompile's
result, gas or error, or of which precompiles are active under a fork, matters
for consensus unless it is gated to the upcoming upgrade. Check how Sonic
selects precompiles: search the head for `ActivePrecompiledContracts`,
`PrecompiledContracts*`, `SetPrecompiles` and `StatePrecompiles`. Sonic adds its
own `evmwriter` state precompile, whose gas on revert must stay as before. A
precompile cache must give identical results, including errors and gas, across
forks. Embedded files such as the KZG trusted setup appear as `file` rows.

### vm-stack-memory

go-ethereum `core/vm` `stack.go`, `memory.go` and `memory_table.go`.
go-ethereum's interpreter does not execute blocks, but the fork exports some of
these to Tosca and Sonic (`core/vm/sonic_tosca_integration.go`: `NewStack`,
`Stack.Len`, `Stack.Push`, `MemoryGasCost`, ...), so check every exported or
Sonic-used function for callers in Tosca and the Sonic head. Confirm that the
behaviour (values, limits, errors, memory expansion cost) is unchanged before
calling a change a refactor.
