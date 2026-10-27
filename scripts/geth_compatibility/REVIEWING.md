# Reviewing a go-ethereum upgrade

`geth_compatibility` writes `review.md`, a row per go-ethereum declaration that
Sonic reaches and whose source differs between the go-ethereum versions of the
base and the head (see `README.md`). Reviewers, subagents of any coding agent
or people, give every row a verdict, a consensus classification and a note.
This document is for the coordinator of a review and for the reviewers.

## Workflow

A review alternates between the tool and the reviewers: run the tool, review
the rows that need it, run the tool again. Only the tool and the coordinator
write `review.md`; reviewers work in parallel and write their rows to files of
their own. When the head changes, for instance after a rebase or with a new
go-ethereum version, the same loop covers only the rows the tool reopened.

## Coordinating a review

1. Run the tool, see `README.md`. Check out the head separately for the
   reviewers, for instance with `git worktree add --detach <dir> <head>`, and
   note the merge base of the base and the head, `git merge-base <base> <head>`,
   which the tool compares.
2. Check that the [facts about Sonic](#facts-about-sonic) below still hold, such
   as which upgrades are live. Correct this document where they changed on the
   base; facts that differ only on the head go to the reviewers in step 5.
3. Select the rows that need a reviewer: `todo` rows whose note does not start
   with `OPEN:`, and `adopted` and `OPEN:` rows with an empty Consensus cell.
   On a new review file, these are all rows.
4. Split them into groups by [area](#hints-by-area), by the package that starts
   the Symbol cell. Split an area with more than about 60 rows into several
   groups of related symbols, such as `core/vm` by source file. Write the table
   lines of each group, copied from `review.md`, to a file of its own.
5. Give each group to its own reviewer, all at once, with this prompt:

   > Review these rows of a go-ethereum upgrade of Sonic. Follow the section
   > "Reviewing" of `scripts/geth_compatibility/REVIEWING.md`.

   Add the file of the group's rows, the file to write the reviewed rows to,
   the path of the head checkout, the merge base, and the facts that differ on
   the head. The source directories of go-ethereum, Tosca and Carmen save each
   reviewer the lookup.
6. Merge the reviewed rows. Check each reviewer's file: every row of its group
   once, the Symbol, Base and Head cells unchanged, and the checks of
   [When you are done](#when-you-are-done). Replace the rows in `review.md` with
   them, matching on the Symbol cell, and run the tool again. It regroups the
   rows and stops on a malformed or duplicate one.
7. Give the rows classified `live` or with a hardfork name, and the `OPEN:`
   rows other than `module` rows, to second reviewers, split by area as in
   step 4, with this prompt:

   > Verify these rows of a go-ethereum upgrade of Sonic. Follow the section
   > "Second look" of `scripts/geth_compatibility/REVIEWING.md`.

   Add what step 5 gives, and the concerns that the replies of step 6 raised
   about these rows, so that they are checked as well. Merge the rows as in
   step 6, and run the tool again.
8. Report the `todo` rows left: they are questions for humans. Check what the
   replies found outside the rows, such as a problem in Sonic's own code that a
   reviewer came across, and report it separately. Correct this document where
   a reply showed one of its facts about the base to be wrong, and remove the
   head checkout.

## Reviewing

You review a group of rows. Fill in the Verdict, Consensus and Note cells of
each, and write the rows to the file your prompt names. Do not change the
Symbol, Base and Head cells, and do not edit `review.md` yourself.

### Sources (read-only)

- Sonic at the head: the checkout your prompt names. Sonic at the base:
  `git show <merge base>:<path>`, with the merge base your prompt names.
- go-ethereum at the base and at the head: the header of `review.md` names both
  module versions, as in `github.com/0xsoniclabs/go-ethereum v0.0.0-...`.
  `go mod download -json <module>@<version>` prints the source directory as
  `Dir`. A version without a module path is a directory relative to the Sonic
  checkout.
- Tosca, the EVM implementations, and Carmen, the state database: the header
  of `review.md` names their versions at the base and at the head after those
  of go-ethereum; find their directories with `go mod download -json` as
  well.
- Read first, where the head has them: the documents of the upcoming upgrade
  among its top-level Markdown files, which say which EIPs Sonic takes, the
  open items, and the fixture results.

### Reading a row

The Symbol cell is `<package> <kind> <name>`. The package is relative to the
go-ethereum module, `go-ethereum` for its root, or a full path for other
modules. The kind is `func` (methods as `Type.Method`), `type`, `var`,
`const`, `file` (a C, assembly or embedded file, relative to the package
directory), `cgo` (the cgo preamble of the named Go file, with its `#cgo`
directives) or `module` (a dependency module, other than go-ethereum, Tosca and
Carmen, that provides reachable code or a value it reads).

Find the declaration in both go-ethereum directories, or for a row starting
with `github.com/0xsoniclabs/tosca/` or `github.com/0xsoniclabs/carmen/go/` in
both directories of that module, for instance with
`grep -n 'func (.*Type) Method(' <dir>/<package>/*.go`, and compare the two
versions. `<package> func init` stands for all `init` functions of the
package, and `<package> var _` for all its variables named `_`. In Go
declarations the tool ignores blank lines outside raw string literals and
comments other than `//go:embed` directives, compares constants by value, and
counts the imports a declaration refers to as part of it. So if the code reads
the same, look at the imports the declaration uses, at the values of its
constants and at the files a variable embeds. `file` and `cgo` rows compare
the text as it is, comments included. `-` in the Base or Head cell means that
side lacks the declaration, or rarely that its package could not be loaded
there; check whether it exists there.

The tool does not say how Sonic reaches a declaration. Search the head checkout
for callers, and follow them through go-ethereum if there are none in Sonic.
`README.md` lists the rules by which the tool counts code as reachable, such as
methods called through interfaces, methods the RPC server calls by reflection,
and package-level variables initialized at startup. An empty Consensus cell
means the finding is on the consensus path, possibly through the initializer
of a package variable that code on the path reads or an init function writing
that variable, such as an ABI parsed at startup; `-` means it is not.

For a `module` row, find the two versions with `git show <base>:go.mod` and
the `go.mod` of the head, and their directories with `go mod download -json`.

### Facts about Sonic

- Upgrades (`opera/rules.go`): go-ethereum forks up to Cancun are always on.
  Prague is enabled by `Upgrades.Allegro`, Osaka by `Upgrades.Brio`. Allegro
  and Brio are live (`opera/validate.go`: Brio cannot be disabled); Canto is
  the upcoming upgrade and not live anywhere. So only changes gated to a fork
  after Osaka are free of risk for existing blocks; a change of Prague or Osaka
  behaviour, or an ungated one, alters how existing blocks replay.
- Never enabled: Verkle and EIP-4762, binary tries (UBT), beacon block roots,
  blob data (Sonic blob transactions carry no blobs), the devp2p eth and snap
  protocols, the beacon chain, EIP-7928 block-level access lists.
- EVM (`opera/vm_config.go`): Tosca's SFVM executes blocks under all
  upgrades. go-ethereum's own interpreter (`core/vm` instructions, jump tables,
  gas tables, stack, memory) does not execute blocks, but it runs for tracing
  (`InterpreterForTracing`, `debug_trace*`, `trace_*`) and for `eth_simulateV1`
  (`api/ethapi/simulate.go` builds a `vm.Config` without Interpreter and
  without Sonic's hooks). Its changes are therefore visible in these RPCs, and
  it must agree with SFVM. Tosca's
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
  before calling it inactive: the import of legacy genesis files
  (`gossip/evmstore/statedb_import.go`) runs go-ethereum's `rawdb`, `trie` and
  hash-based `triedb`, `trie.StackTrie` hashes the transactions and receipts of
  every block, and `sonictool` uses go-ethereum's `node` for account management
  and its `ethdb/pebble` for `dbs analyze`.
- The fork carries Sonic patches: files such as `core/types/sonic_extensions.go`
  and `core/vm/tosca_integration.go`, and `vm.Config` hooks such as
  `ChargeExcessGas`, `IgnoreGasFeeCap`, `InsufficientBalanceIsNotAnError` (set
  before Canto, cleared from Canto on), `SkipTipPaymentToCoinbase`,
  `StatePrecompiles`, `Interpreter`, `InterpreterForTracing`, `MaxTxGas`,
  `MaxCodeSize` and `MaxInitCodeSize` (the last three set from Brio on). If a
  change touches code around them, check that they still work.

If one of these facts is wrong, say so in your reply.

### Verdicts

Exactly one per row:

- `inactive`: the change only affects behaviour Sonic does not run: a fork or
  feature Sonic never enables, or code that is reachable on paper but that you
  verified Sonic never executes. The note says why, such as "only used by
  pathdb; Sonic state is Carmen".
- `refactor`: you verified that the behaviour is unchanged: renames, extracted
  helpers, moved code, types or parameters changed without effect, or new
  parameters that every caller passes with the old behaviour.
- `adopted`: the behaviour changes and Sonic accepts it. Typically a change
  gated to the upcoming upgrade for an EIP that its documents mark as taken, or
  a fix without consensus or user-visible impact. The note names what covers
  it: the EIP and its gate, and the test, such as the exec-spec fixtures of the
  upgrade passing on geth VM and SFVM, or a Sonic test you found.
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

### Consensus

The consensus path is the code whose results all nodes have to agree on: block
formation and execution, event validation and genesis. Classify every
`adopted` and every `todo` row whose Consensus cell is empty; `refactor` and
`inactive` rows need no classification.

- `live`: the change can alter consensus results under upgrades that are
  already live. This includes `adopted` rows for which Sonic compensates in its
  own code; the note says how.
- the lowercase name of a Sonic upgrade, such as `canto`: it can alter results
  only once that upgrade is active, as code gated to it that runs on the block
  path, or state that is always zero before it. Use the Sonic name, a field of
  `opera.Upgrades`, not the one of the go-ethereum fork it enables.
- `no`: it is on the path but cannot alter results, for instance tracing,
  performance, metrics, error strings, or go-ethereum's interpreter where Tosca
  executes blocks.

Classify a row off the path too, replacing its `-`, if its consensus problem
lies outside the changed code, for instance a call that is missing on the path,
or an open item of the upcoming upgrade that the row tracks.

### Notes

One line, at most about 300 characters, specific: the EIP, the gating
condition, the Sonic caller that makes the code run, or the test. A `todo` note
starts with `OPEN:` and the question, followed by `Found:` and what you
established; without `OPEN:` the row counts as not reviewed. Related rows, such
as a refactor spanning many functions, get consistent verdicts and may share a
note. A note that starts with `was <verdict> before the last change:` is an
earlier review of a previous version of the finding; use it, but check the
current code. Write a `|` inside a note as `\|`.

### How to work

- Compare both versions of each declaration, check the gating condition in the
  head, and check how Sonic reaches and uses it: its callers, and a search of
  the Sonic head.
- Work read-only: read and search files, and compare them. Do not build, test
  or run anything other than the `go` commands above that locate sources, and
  do not change any file other than the one for your rows.

### Resolved by Sonic

Before you leave a row `todo`, check whether Sonic's own code already answers
its question:

- a fix on the head, with a test that pins the behaviour;
- a setting that makes the changed code unreachable, such as the forks Sonic
  enables, its VM configuration, or the way it decodes and builds transactions.

Look at the Sonic callers, at the tests next to them, and at the fork settings.
If the code answers the question, give the row its final verdict and name the
code and the test in the note. If it answers part of it, the note says what is
resolved and what remains `OPEN:`. A question that is a decision stays `todo`
even if the change is harmless: accepting a new behaviour, an RPC format, or a
dependency.

### When you are done

Check your rows: every row has a verdict, every verdict other than `todo` has a
note, every `todo` note starts with `OPEN:`, and every `adopted` and `todo`
row on the consensus path has a classification. Then write your rows to the file your
prompt names, one Markdown table line with all six cells per row of your group,
in the format of `review.md` and without a header. Reply with a short summary:
the number of rows per verdict, every `todo` row with its question, and
anything that concerns other areas or contradicts this document.

## Second look

Another reviewer filled in these rows. Each one either claims that its change
can alter consensus results, with `live` or a hardfork name as Consensus, or
asks a human a question, as a `todo` row. Work as in
[Reviewing](#reviewing), and check every claim in the sources:

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
and its note says what you established. Finish as in
[When you are done](#when-you-are-done), and list in your summary every row you
changed, with its old and new verdict and classification and the reason.

## Hints by area

### core: `core`, `core/types`, `core/stateless`

The state transition, `IntrinsicGas`, `FloorDataGas`, `GasPool`, `Message`,
`TransactionToMessage`, `Transfer` and `CanTransfer`; headers, blocks,
receipts, blooms and transactions. Much of `core` runs in production: Sonic's
`evmcore` uses `core.ApplyMessage`, `IntrinsicGas`, `FloorDataGas` and
`GasPool`; check the Sonic callers in `evmcore`, `gossip/blockproc` and
`api/ethapi`. A transaction that fails before execution is skipped, so check
whether a change moves a check before or after the gas pool debit or the gas
purchase, and how Sonic sets the `Message` flags. Changes to header or block
fields or hashing matter for block hashes and RPC output.

### core/vm

`evm.go` (`Call`, `Create`, precompile dispatch, block and transaction
context), `contract.go`, `interpreter.go`, the gas accounting types, errors and
jump destinations run in production, because Tosca's adapter calls back into
`evm.go`. Check every behaviour change for fork gating and for the Sonic hooks
in `vm.Config`. Where a signature or semantics changed, check that Tosca's
`geth_adapter` and Sonic's `evmcore` use it consistently.

The instructions (`instructions.go`, `eips.go`, `jump_table.go`,
`opcodes.go`), the gas functions (`gas.go`, `gas_table.go`,
`operations_acl.go`, `operations_verkle.go`) and the stack and memory
(`stack.go`, `memory.go`, `memory_table.go`) belong to go-ethereum's
interpreter, which does not execute blocks. A change there matters if it
changes the traced or simulated behaviour of existing blocks (not gated to the
upcoming upgrade), or if it diverges from what Tosca's SFVM does under the
same fork; check gated changes against the documents of the upcoming
upgrade. `operations_verkle.go` is EIP-4762, which Sonic never enables. The
fork exports parts of the interpreter to Tosca and Sonic in
`core/vm/tosca_integration.go`, such as `NewStack`, `Stack.Push` and
`MemoryGasCost`: check callers of every exported or Sonic-used function in
Tosca and the Sonic head, and confirm that values, limits, errors and memory
expansion costs are unchanged before calling a change a refactor.

### Precompiles: `core/vm` `contracts*.go`, `crypto`, `consensus`

Precompiles run in production, because Tosca calls back into go-ethereum's EVM
for calls, so any change of a precompile's result, gas or error, or of which
precompiles are active under a fork, matters for consensus unless it is gated
to the upcoming upgrade. Check how Sonic selects precompiles: search the head
for `ActivePrecompiledContracts`, `PrecompiledContracts`, `SetPrecompiles` and
`StatePrecompiles`. Sonic adds its own `evmwriter` state precompile
(`opera/contracts/evmwriter`), whose gas on revert must stay as before.
Embedded files such as the KZG trusted setup appear as `file` rows.

### params and tracing hooks: `params`, `core/tracing`

`ChainConfig`, `Rules`, fork checks, protocol constants, embedded chainspec
JSON files, `params/forks`, and the `Hooks` struct with the gas and balance
change reasons. Sonic builds its own `ChainConfig` in `opera/rules.go`; check
which fields and `Rules` flags Sonic sets, and whether new fields default
safely (nil, false, zero) for Sonic. Changed constants matter if code Sonic runs
uses them under the forks Sonic enables. The embedded chainspecs are Ethereum
networks: check whether Sonic loads them at all. For `core/tracing`, check
Sonic's tracers and hooks (`txtrace`, `evmcore`, `api/ethapi`, the tracers it
registers) for changed hook signatures or semantics.

### State and storage: `core/state`, `core/rawdb`, `core/filtermaps`, `trie`, `triedb`, `ethdb`

Sonic's state is Carmen, so most of this should be `inactive`, but verify each
case by how Sonic reaches it: for instance the import of legacy genesis files,
`sonictool`, RPC handlers, or the transaction pool. Mark a row `inactive` only
with a concrete reason, such as "bintrie is only used by UBT and Verkle, never
enabled", or "the rawdb freezer is only opened by node.OpenDatabase*, which
Sonic never calls" after a search. Where go-ethereum storage code does run in
Sonic (the legacy genesis import, `StackTrie`, the `ethdb/pebble` database of
`sonictool dbs analyze`), assess the change normally.

### RPC and APIs: `rpc`, `node`, `ethclient`, `accounts`, `signer`, `common/hexutil`, `internal/web3ext`, `internal/ethapi`

Sonic serves its JSON-RPC through go-ethereum's `rpc` server and `node`, so
changes in request handling, batching, limits, timeouts, error messages or JSON
encoding are visible to Sonic's users: they are `todo` unless clearly a bug fix
without compatibility concerns. Check Sonic's node setup (`config/`,
`cmd/sonicd/`) for the node features and flags it uses. `accounts/usbwallet`
and external signers are reachable through the account manager
(`config/make_node.go`); check whether Sonic enables them.

### Tracers and logging: `eth/tracers`, `log`, `metrics`, `internal/telemetry`

Sonic exposes go-ethereum's tracers, such as `callTracer`, `prestateTracer`,
`4byte` and the struct logger, through its `debug_trace*` RPCs; check
`api/ethapi` and how Sonic registers and calls tracers. A change in tracer
output is visible to users: `todo` unless clearly a bug fix without
compatibility concerns. Tracer changes for features of the upcoming upgrade
are `adopted` if they only show once it is active. Log handler changes affect
Sonic's log format and flags; check how `cmd/sonicd` configures logging.

### Modules: `module <path>`

These rows compare the versions of dependency modules other than go-ethereum.
They are always `todo`. Compare the two source directories, and note the base
and head version and a short summary of what changed in the code Sonic
reaches or the values it reads: major version changes, removed or replaced
modules, notable behaviour changes. Keep it proportionate, a few lines of investigation per module.

### Everything else, such as `p2p`, `common`, `rlp`

Sonic runs its own protocols but uses go-ethereum's p2p server, RLPx,
discovery v5 (v4 only with `--discovery.v4`), enode, ENR and NAT, so these run
in production; changes to discovery or the handshake can affect connectivity
with nodes running older Sonic versions.
