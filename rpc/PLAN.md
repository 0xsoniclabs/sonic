# RPC module: requirements and plan

A new RPC stack for `sonicd`, written from scratch, running in the consensus
process and selected by a flag next to the current one. Today's RPC layer is
go-ethereum's server and handlers carried along with the fork: methods
reflected off structs, a wide `Backend` interface into the node, and patches
where it did not fit. The module is one the team owns: one narrow interface to
the node, its own entry point and transports, fewer third-party packages, and a
test program that holds it to the release it replaces.

The legacy stack stays in the code base, unchanged, and stays the default. The
module depends on nothing of it but the log query code (ADR15), so that the
legacy stack can be deleted later by the deletions, the one move and the edits
of the removal list (section 3). Code the two stacks would share is copied into
the module; the duplication is accepted.

This document is the work to be done and the proof that it is done. It records
no progress, is executed from `main`, lives on a branch of its own and is part
of neither the work branch nor the pull requests of M2 (H12).

**Baseline.**

- **Reference** is what the module answers like: legacy mode of the same
  commit, which answers as `main` does (SW4). lakme compares the two directly
  (9.1), so every difference it finds is the module's.
- **Release** is the newest release tag (v2.2.6 when this was written), the
  reference of the release run only (9.1). What `main` answers differently from
  it (today `eth_sendRawTransactionSync`, #1008) is a release difference,
  annotated at step 0. Version-specific facts below are re-checked at step 0.
- **Port base** is the current `main`. Fixes reach `main` first; the work
  branch takes no cherry-picks. It is rebased onto `main` at the end of each
  step. After a rebase every commit is built and tested again (`git rebase
  --exec` with the build and the tests of `rpc/`), and the legacy run (SW4) is
  repeated. A change to the legacy RPC that a rebase brings is ported to the
  module in that rebase; the same-commit lakme run shows one that was missed.
- **Legacy stack** is what serves RPC on `main`: `api` (`GetAPIs`),
  `api/ethapi`, `api/sonicapi`,
  `api/testapi`, `api/rpctest`, `gossip/filters`, the oracle half of
  `gossip/gasprice` with `gossip/gpo_backend.go`, `gossip/ethapi_backend.go`,
  `gossip/api.go`, `Service.APIs`, `txtrace`, and the endpoints the node opens
  itself (go-ethereum's HTTP, WebSocket and IPC servers). **Module** is
  everything under `rpc/` and the adapter `gossip/rpc_core.go`. The **node
  handler** (`stack.RPCHandler()`, go-ethereum's in-process server with the
  namespaces it registers itself) is neither: module mode reaches it over IPC
  (X2), and it stays when the legacy stack is deleted.
- Code pointers are `file:Symbol`, never line numbers. Sources: **code** the
  legacy handler on `main`, **new** decided here; open decisions go to
  section 11.

**Rule for everything the work produces.** Code, comments, documentation, tests
and commit messages never mention this plan, its sections, its requirement or
decision ids, or any branch. They say what and why in their own words (H12).

## 1. Milestones

| Milestone | Outcome | Ends when |
|---|---|---|
| M1 | The module is complete on one work branch, behind `--rpc.stack module`; `legacy` stays the default | every requirement of section 7 and every line of the M1 gate (9.7) is green, and the user approves |
| M2 | The work of M1 is on `main`, through stacked pull requests | the last pull request is merged |

**Not part of this work.** Neither milestone makes `module` the default, and
neither deletes or moves any part of the legacy stack. Both are later
projects. This work only keeps the deletion small: the removal list
(section 3) stays complete, held by SW3 and SW6.

**M1** is one branch cut from `main`, built commit by commit in the steps of
section 6. No pull request is opened. On every commit, legacy mode answers as
`main` does, and module mode serves only what is ported so far: nothing is
forwarded to the legacy stack, and an unported method answers -32601. The
user is the person who directs this work. The user approves and at the
end of step 7, which ends M1. Nothing of M2 starts before that approval.

**M2** cuts the work branch into stacked pull requests, so that the work is
reviewed in public and reaches `main` with a history that can be read. Each
pull request is a contiguous run of the commits of M1 and is merged by
squash, in order. Their number is not fixed here; it is kept reasonably
small, with each pull request still reviewable in full. Everything else
about M2, and any change to these points, is decided before M2 starts.

**After M2, while both stacks exist**, a change to an RPC answer lands in
both stacks in the same pull request, or the module's side becomes a
deviations row (section 8). The same-commit lakme run is part of the review
of every pull request that touches either stack.

## 2. Shape

| Principle | What it means |
|---|---|
| Two stacks, one active | `--rpc.stack` picks one at start; the other registers no API and starts no goroutine (SW1, SW2). One legacy type is built in both modes, for the emitter's signer (section 3) |
| Independent | the module imports nothing of the legacy stack and the legacy stack nothing of the module (SW3); one exception, the log queries (ADR15) |
| One narrow interface | the node is reached only through `Core` (section 4); a test freezes it |
| Stateless, keyed | no `Core` call takes a tag: tags are resolved once per request, in the module, and chain and state are read by block number, by hash or by id; the one exception is the bounds of a log query (section 4) |
| Archive-only reads | account state comes from the archive at the named block, never from live state. What is not in a block yet is read from the node as it is now, by the calls section 4 lists: the pool (with the pending nonce) and the live consensus calls |
| One `latest` | the newest archived block seen on the head subscription, for everything served, `eth_blockNumber` included |
| Own entry point | one dispatcher starts every request goroutine and, for every method the module registers, decodes, applies the deadline, recovers panics and counts the call on every transport (the node handler's methods on IPC are forwarded, X2); methods exist by typed registration (`dispatch.Add`), never by reflection over a struct's methods |
| Own dependencies | go-ethereum for the data model, the EVM and the tracers; its `rpc`, `node` and `event` packages stay out (section 5) |
| Monitored | Prometheus `client_golang`, per-method accounting, bounded cardinality, alerts named up front |
| Smaller surface | wallet, signing and chain-database methods are not served (X1) |
| Test-first | a step is done when lakme, the handler tests and the integration tests say so; every difference from the reference is justified in writing (section 8) |

### Layout

| Package | Holds | Imports |
|---|---|---|
| `rpc/core` | `Core`, its role interfaces and value types; implemented by `gossip/rpc_core.go` | evmcore, opera, inter, inter/state, inter/iblockproc |
| `rpc/dispatch` | the JSON-RPC runtime, nothing of the chain: `Dispatcher` (method table, deadline, recover, `Group`, loops, `Stop`), `Add` and `Handler` (D13), `Error`, `Encoded` (H3), `Notifier`, metrics; `Config` holds the request deadline only | no node package |
| `rpc/chain` | the served head: `Head`, `Chain` (head loop; head and pool feeds through bounded `Hub`s, I15, I18), `BlockNumber`/`BlockReference` (client input, H6), `Resolve` (H1); `Config` holds the pruning cutoff | core, dispatch, evmcore |
| `rpc/exec` | what methods share: the lookups of `Environment` (H1, H2); `CallArguments`, overrides, EVM environment, `ChainConfig` from rules, `RevertError`; the send path (V7); `Config`, the semantic limits and switches (section 10) | chain, dispatch, core, oracle, evmcore, opera |
| `rpc/format` | JSON forms of blocks, headers, transactions, receipts and logs used by several namespaces | core, evmcore |
| `rpc/oracle` | gas price oracle, its sampler and `Config` (the `GPO` keys). It imports neither `chain` nor `dispatch`: the `Module` passes it the served head as a function and starts its sampler as a loop of the dispatcher | core |
| `rpc/eth`, `filters`, `trace`, `debug`, `txpool`, `net`, `web3`, `sonic`, `dag`, `abft`, `testonly` | one namespace each (`testonly` serves `test_`); `Register(dispatcher, environment[, role])` (H8). `rpc/trace` holds the module's copy of the OpenEthereum tracer of `txtrace`; `rpc/debug` the module's copy of the profiling stubs of a build without `enable_debug` (X2) | dispatch, chain, exec, format, oracle, core (role interfaces) |
| `rpc/transport` | HTTP, WebSocket and IPC over one `Front` that routes every method the module registers to the dispatcher and, on IPC only, the other `admin_*` and `debug_*` methods to the node handler, an `http.Handler` (X2) | dispatch |
| `rpc/tests` | test support (9.4): `FakeCore`, `NewCarmenCore`, the replay chain, accounts, contract code, the harness | core, dispatch, chain, exec, oracle |
| `rpc` | `Module`: wiring of namespaces, start and stop | all of the above except `rpc/tests` |

**The Imports column is a test.** The test holds the table as data and fails on
an import between module packages that the row does not list. A change to the
table changes the test in the same commit. What two namespaces share moves to
`exec` or `format` when the second one needs it; no namespace imports another.
Beyond the module, a row may import go-ethereum's keep list (section 5), the
node libraries `evmcore`, `opera`, `inter`, `inter/state`, `inter/iblockproc`,
`utils`, lachesis-base for the index and hash types those libraries hand out,
and the leaf exceptions below.

Nothing under `rpc/` imports `gossip`, a legacy package or `go-ethereum/rpc`.
The exceptions are leaf packages of consensus-critical definitions that must not
be copied, each allowed in its one package; a new one goes into this list and
the test in the same change:

| Package | Allowed in | Why not copied |
|---|---|---|
| `gossip/blockproc/bundle` | `rpc/sonic` | plan encoding, plan hash, envelope gas formula |
| `gossip/blockproc/priorities/registry`, `gossip/blockproc/subsidies/registry` | `rpc/eth` | system contract addresses of `eth_config` |
| `gossip/blockproc/subsidies` | `rpc/format` | `IsSponsorshipRequest` decides a zero `effectiveGasPrice` as block processing does |
| `gossip/gasprice/gaspricelimits` | `rpc/exec` | the suggested price of a new transaction, as pool and emitter apply it |
| `gossip/evmstore` | `rpc/tests` | the Carmen state wrapper, test support only |

`gossip` imports `rpc/core` for the adapter and nothing else of the module;
the node configuration keeps its own types and literals until the legacy stack
is deleted (section 10).
`Core` is declared with its user, the module, and not in `gossip`: the module
then builds and is tested without `gossip`, and the import test can hold that
nothing under `rpc/` reaches the node except through `Core`.
The composition root `cmd/sonicd/app` builds the adapter and the `Module`, and is
the one place that names both stacks (SW2). Test files may import
go-ethereum's `rpc` as the oracle for envelope compatibility (Q5).

## 3. Stack selection

- **The flag** is SW1.
- **Legacy mode** is `main`: no module package is constructed and no module
  goroutine runs.
- **Module mode** registers no legacy API and starts none of the legacy
  stack's RPC components. Before step 1, list what in `gossip.Service` serves
  only the legacy RPC (`Service.APIs`, the filter API, the oracle and its
  sampler, `netRPCService`) and what else reads each one; module mode skips
  what only RPC reads, and `Service.Stop` skips stopping it
  (`StopPublicFilterAPI`, `gpo.Stop()`). `EthAPIBackend` is built in both
  modes and starts nothing: the emitter's `TransactionSigner` is
  `EthAPI.signer`; the adapter does not use it (SW3). The node
  opens no RPC server of its own: the module listens on every configured
  endpoint (`takeOverRPC`, new, in `cmd/sonicd/app`). Until step 4 module mode
  serves HTTP, and IPC without subscriptions; a configured WebSocket endpoint
  is not opened, and a warning names it.
- **Legacy stays as it is.** No node change the module needs alters a legacy
  answer; the legacy-mode lakme run proves it (SW4). Where the module wants
  different behaviour from shared code, it copies the code (the vmTrace
  accounting of `txtrace`, V12) instead of changing it.

**Rules of `takeOverRPC`** (module mode), each found by a node that failed:

- **WebSocket shares the HTTP server when the ports match**, whatever the hosts;
  the shared server listens on the WebSocket host (geth's
  `Node.wsServerForPort`, then `setListenAddr(WSHost, …)`). Deciding by host and
  port binds the port twice and the node does not start.
- **Taking over rewrites the node configuration**, so everything the node
  derives from it changes. Before step 1, list every reader of `HTTPHost`,
  `WSHost`, `IPCPath` and what derives from them, and keep its behaviour:
  `Config().ExtRPCEnabled()` turns false, which `unlockAccounts` reads;
  `HTTPEndpoint()` and `--dump-config` report the endpoints. The module, not
  the node configuration, answers whether external RPC is served (O5).
- **Module mode unlocks no account.** No method the module serves uses an
  unlocked or derived account (X1). `--unlock`, `--password` and
  `--allow-insecure-unlock` are accepted, warn and have no effect in module
  mode: `unlockAccounts` is not called, so the refusal `account unlock with HTTP
  access is forbidden` has nothing left to guard (O5).
- **In-process clients are clients.** In module mode `stack.Attach()` reaches a
  node handler that serves only go-ethereum's own namespaces. Its one non-test
  caller on `main` is the wallet event loop of `startNode`
  (`Wallet.SelfDerive`, which reads `eth_getBalance`, `eth_getTransactionCount`,
  `eth_getCode` and `eth_getStorageAt`); module mode does not start that loop,
  for the same reason. Before step 7, list the non-test `Attach()` callers
  again; a new one is served through `Dispatcher.Serve` or recorded in
  section 11.

**Removal list.** Not part of this work: no item is done in M1 or M2.
Deleting the legacy stack later takes the deletions, the one move and the
edits below. SW3 and SW6 keep the list complete from the first commit of M1,
and step 7 reviews it:

- move the log query code the adapter reuses (`gossip/filters/filter.go` and
  the `Config` of `gossip/filters/api.go`, ADR15) into a node package, in a
  move commit; the adapter's `Logs` and its backend import the moved package;
- delete the legacy stack (Baseline), the legacy-only tests
  (`*_legacy_test.go`, SW6) and the legacy CI stage;
- remove the legacy RPC wiring of `gossip/service.go`: the fields `gpo`,
  `EthAPI`, `netRPCService` and `filterAPI`, their start and stop calls,
  `rpc.SetExecutionTimeLimit`, and the module-mode conditions that skip them;
  the emitter's `TransactionSigner` takes the service's signer instead of
  `EthAPI.signer`; delete the unused `EvmStateReader.gpo`;
- remove the switch of SW2 from the composition root; `--rpc.stack` becomes
  deprecated for one release (accepted, warn, no effect; O4); the X1 flags
  (`--unlock`, `--password`, `--allow-insecure-unlock`) keep the deprecation
  they have in module mode (O5), and the wallet event loop of `startNode` is
  deleted;
- `GPO` takes `oracle.Config` and `FilterAPI` the configuration type of the
  moved log query package, with the same keys; `gossip.DefaultConfig` refers to
  the module's defaults; delete the equality tests of section 10;
- delete `SONIC_TESTS_RPC_STACK` and `RequireRPCStack` from `tests/`.

Nothing else changes. If something must, SW3 has a gap: the import test is
fixed first.

## 4. The system interface

`Core` is implemented in `gossip/rpc_core.go` by one adapter over `Store`,
`EvmStateReader`, the tx pool and `ServiceFeed`. Every test under `rpc/` runs
against the fakes of `rpc/tests`; the adapter's tests run on a real in-memory
store (T8).

```go
// Core is everything the RPC module may ask the node. No call takes a tag:
// chain and state are read by block number, by hash or by id. Only
// SendTransaction and the TestOnly calls change anything. A lookup reports a
// missing item as ErrNotFound. A namespace depends only on the roles it uses
// (H8).
type Core interface {
	ChainReader     // ChainInfo, Header, Block, BlockNumberByHash, Receipts, Rules, TransactionByHash, BundleInfo
	HeadSource      // Head, SubscribeHead
	StateProvider   // State
	Pool            // SendTransaction, PoolNonce, PoolTransaction, PoolNextTransactions, PoolContent(From), PoolStats, SubscribeTransactions
	LogIndex        // Logs
	NodeInfo        // Progress, OracleInputs, PeerCount
	ConsensusReader // EpochOf, EpochState, LiveEpochState, LiveEpochHeads, FindEvents, Event, EventPayload
	TestOnly        // SendTransactions, EmitTransactions
}

type ChainReader interface {
	ChainInfo() ChainInfo
	Header(number uint64) (*evmcore.EvmHeader, error)   // no transactions decoded
	Block(number uint64) (*evmcore.EvmBlock, error)     // ErrNotFound above the committed head
	BlockNumberByHash(hash common.Hash) (uint64, error) // ErrNotFound for an unknown hash
	Receipts(number uint64) (types.Receipts, error)     // fails if TxIndex is off
	Rules(number uint64) (*opera.Rules, error)          // a stored block: its epoch's rules; above the committed head: the open epoch's
	TransactionByHash(hash common.Hash) (transaction *types.Transaction, block, index uint64, err error) // fails if TxIndex is off
	BundleInfo(executionPlanHash common.Hash) (BundleInfo, error)
}

type HeadSource interface {
	Head() (*evmcore.EvmBlock, error) // newest block whose state is archived; without an archive, the committed head (O3)
	SubscribeHead(events chan<- HeadNotify) Subscription
}

// State: the archive at one block in Carmen's non-committable StateDB
// (inter/state.StateDB). Journaled per request, released once, never committed.
type StateProvider interface {
	State(number uint64) (state.StateDB, error)
}

// Pool: the transactions the node holds outside blocks. Every answer depends
// on "now".
type Pool interface {
	SendTransaction(ctx context.Context, transaction *types.Transaction) error
	PoolNonce(address common.Address) uint64
	PoolTransaction(hash common.Hash) (*types.Transaction, error)
	PoolNextTransactions() types.Transactions // the next executable transaction of every sender, unordered: the oracle's sample
	PoolContent() (pending, queued map[common.Address]types.Transactions)
	PoolContentFrom(address common.Address) (pending, queued types.Transactions)
	PoolStats() (pending, queued int)
	SubscribeTransactions(events chan<- evmcore.NewTxsNotify) Subscription
}

// LogIndex: log queries over archived blocks, answered by the node's own
// query code on its address/topic index (topicsdb), so that both stacks find
// the same logs (ADR15).
type LogIndex interface {
	Logs(ctx context.Context, query LogQuery) ([]*types.Log, error)
}

// NodeInfo: liveness, peers and what the gas price oracle reads per
// suggestion. Every answer depends on "now".
type NodeInfo interface {
	Progress() Progress
	OracleInputs() OracleInputs // NextEpochRules, TotalGasPowerLeft, MinTip
	PeerCount() int
}

// ConsensusReader: epochs, validators and DAG events. EpochOf and EpochState
// are keyed like the chain reads. The two Live calls answer from the node's
// current state. The event lookups read the DAG, which runs ahead of the
// blocks, so the served head does not bound them.
type ConsensusReader interface {
	EpochOf(number uint64) (uint64, error)                                // a stored block: its epoch; above the committed head: the open epoch
	EpochState(epoch uint64) (EpochSnapshot, error)                       // the states stored when the epoch opened; ErrNotFound without them
	LiveEpochState() EpochSnapshot                                        // the node's current block and epoch state, read together
	LiveEpochHeads(epoch uint64) ([]common.Hash, error)                   // events without descendants; ErrNotFound unless the epoch is open
	FindEvents(epoch uint64, lamport uint32, prefix []byte) []common.Hash // the events a short id names
	Event(id common.Hash) (*inter.Event, error)
	EventPayload(id common.Hash) (*inter.EventPayload, error)
}

// EpochSnapshot: the node's block state and epoch state of one moment.
type EpochSnapshot struct {
	BlockState iblockproc.BlockState
	EpochState iblockproc.EpochState
}

// TestOnly: the calls of test_*. The namespace is always registered, as in
// legacy; while EnableTestOnlyApi is off, its methods answer
// "test-only API is not enabled" without calling the node.
type TestOnly interface {
	SendTransactions(transactions types.Transactions) error // test_addTransactions: into the pool in one call, then validated and ordered by the emitter as usual
	EmitTransactions(transactions types.Transactions) error // test_proposeTransactions: on a validator, emitted in a new event as given, without validation
}
```

`ConsensusReader` serves `dag_*`, `abft_*`, `eth_getRules`, `eth_currentEpoch`
and `eth_getEpoch{Block,Stats}`. Its seven calls replace the nine those legacy
handlers (`dag_api.go`, `abft_api.go`, `PublicBlockChainAPI.GetRules`,
`CurrentEpoch`, `GetEpochBlock`) make on `EthAPIBackend`. What the backend
computes from the two states (validators, downtime, uptime, originated fee,
epoch timing) and the decoding of a short event id are module code, copied
with their wording (SW3). The interface is frozen at step 1 with the rest of
`Core`; its adapter half and the handlers land at step 6.

| Method | Calls | Answers from "now" |
|---|---|---|
| `eth_currentEpoch` | `EpochOf(head+1)` | no |
| `eth_getRules`, `eth_getEpochBlock`, `abft_getValidators` | the epoch tag, then `EpochState`; `pending` reads `LiveEpochState` | `pending` only |
| `eth_getEpochStats` | `EpochOf(head+1)`, then `EpochState` of that epoch for its two epoch starts | no |
| `abft_getDowntime`, `abft_getEpochUptime`, `abft_getOriginatedEpochFee` | `LiveEpochState` | yes: they take a validator and no epoch |
| `dag_getHeads` | the epoch tag, then `LiveEpochHeads` | yes: heads are kept for the open epoch only |
| `dag_getEvent`, `dag_getEventPayload` | `FindEvents` for a short id, then `Event` or `EventPayload` | yes: an event may not be in a block yet |

`eth_getRules` takes an epoch, as the reference. Epoch tags are resolved in
the module, with the current epoch `EpochOf(head+1)` in place of the store's
(the current rules, below). The reference has two rules, and each method keeps
its own: `GetEpochBlockState` reads `latest`, `safe` and `finalized` as the
current epoch (`eth_getRules`, `eth_getEpochBlock`, `abft_getValidators`);
`epochWithDefault` reads them as the epoch before it and refuses a number
above the current epoch with `epoch is not in range` (`dag_getHeads`). In
both, `earliest` is the pruning cutoff and `pending` is the open epoch as the
node has it now. The live answers are the consensus exception to the rule on
`Core`'s growth below.

**`Logs` reuses the node's log query code** (ADR15). The adapter answers
with `gossip/filters.NewRangeFilter` or `NewBlockFilter` and `Filter.Logs`, on
the node's `FilterAPI` configuration, over a `filters.Backend` of the adapter's
own on the store, in `gossip/rpc_core.go`. It implements the calls `Filter`
makes (listed at step 4 from `filter.go`) with copies of the lookups of
`EthAPIBackend` (SW3); the calls only `PublicFilterAPI` makes (the three
`Subscribe*`) answer an ended subscription. Two calls are bounded by
`LogQuery.Head`, the module's served head: `HeaderByNumber` of a tag answers
the served head, and `HeaderByHash` of a block above it answers nothing.
`LogQuery` carries the requested bounds as
the reference reads them (`int64`, a tag or a missing bound negative), because
the legacy filter decides on them: a tag bound is the head, and equal bounds
lift the result limit. It is the one place a tag reaches the node (I2), so the
module's tags have go-ethereum's values (`rpc.BlockNumber`); the legacy filter
reads every tag, `earliest` included, as the head. The legacy code keeps its
order of checks, wording, limits, tx indexes and timestamps. Two parts are
module code, copied from the legacy code with its wording: decoding the
criteria (`FilterCriteria.UnmarshalJSON`: `blockHash` with
`fromBlock`/`toBlock`, the address and topic forms) and the normalisation of
`returnLogs` (no null answer, no null topics), on a copy (P8).

Only queries over stored blocks are reused. Events as they arrive are the
module's: polled filters and `logs` subscriptions take their logs from the
head subscription and match them with a copy of the reference's `filterLogs`
(I11, I17). `PublicFilterAPI` and its event system are typed on go-ethereum's
`rpc` and go with the legacy stack.

Value types (`ChainInfo{ChainID, NetworkID, GenesisID, UpgradeHeights}`,
`Progress`, `OracleInputs`, `BundleInfo{BlockNumber, Offset, Count}`,
`HeadNotify{Block, Logs}`, `LogQuery`, `EpochSnapshot`) live in `rpc/core`.
`EpochSnapshot` holds the node's own `iblockproc` states, as `Rules` returns
the node's `opera.Rules`: the module copies no consensus type.
`Subscription` is `Err()` and `Unsubscribe()`, satisfied by the subscriptions
of the node's feeds (`notify.Subscription`).

Rules of the adapter:

- **It does not use the legacy stack.** It calls neither `EthAPIBackend` nor
  anything under `api/`; logic it needs from them is copied (SW3). The one
  legacy package it imports is `gossip/filters`, for `Filter` (above).
- It starts no goroutine for the module. The head snapshot is the synchronous
  `Head()`, never a first event pushed by the adapter (an unmanaged pusher
  panics on a stopped feed). The module keeps its head monotonic.
- **A block and its logs are one event.** `ServiceFeed` sends them on two
  feeds (`newBlock`, then `newLogs`), which the adapter cannot join without a
  goroutine. `ServiceFeed` gains a third feed that sends the pair from the same
  place; `SubscribeHead` subscribes to it. The two legacy feeds are unchanged
  (SW4).
- A stopped feed answers `Subscribe*` with an ended subscription, never `nil`.
- `Header(number)` exists because most reads need no transactions (I13).
- A block above the committed head is `ErrNotFound`, as #1177 makes legacy do.
- **One way to say "not found"**: `(value, error)` with `ErrNotFound`; no
  `bool`, no `nil` without an error. The lookups of `exec.Environment` keep the
  rule (H1).
- The launcher stops the module before the node closes.
- **Every guard the legacy backend takes from the node configuration is the
  adapter's.** With `TxIndex` off, `Receipts` fails with `transactions index is
  disabled (enable TxIndex and re-process the DAGs)` and `TransactionByHash`
  with the same text ending `the DAG)`; a method propagates the error where its
  legacy counterpart did. Before writing the adapter, list every `svc.config`
  read in `gossip/ethapi_backend.go`; each gets an adapter test, because lakme
  runs only the default configuration.

**Derived by the module**, not asked for: `MinGasPrice`, `MaxGasLimit`, the EVM
and chain config (from `Rules(number)`, the header and `opera.GetVmConfig`), the
head (from `SubscribeHead`).

**The current rules and the current epoch are the next block's.** They are the
ones block `head+1` runs under, where `head` is the served head:
`Rules(head+1)` and `EpochOf(head+1)`, never those of `head`, which may seal an
epoch. This keeps them a function of the served head, like every other read,
and not of the node's live state, which legacy reads (`store.GetRules()`,
`store.GetEpoch()`). When the archive lags, block `head+1` is stored and the
calls answer from its epoch. When it does not, `head+1` is above the committed
head and the calls answer with the open epoch, which the blocks up to `head`
have fixed: it is the one case in which a keyed call reads the node's current
state, and it is what legacy answers. `MaxGasLimit`, the oracle's sampler
(I10) and the epoch tags follow the rule; when one site is found wrong, every
site of the shape is checked.

**What answers from "now".** `Head` and the subscriptions, `Pool`, `NodeInfo`,
`TestOnly`, and of `ConsensusReader` the two `Live` calls and the event
lookups. Every other call is keyed by a block number, a hash, an epoch or a
query bounded by the served head (`Logs`), and its answer does not change once
the item is stored; `Block`, `Rules` and `EpochOf` say in their comments what
they answer above the committed head.

**`pending` has four meanings, each used with its noun.** The `pending` block
tag is the served head, as `latest`: there is no pending block (I5). The
pending nonce is the pool's next nonce of a sender (`PoolNonce`). Pending
transactions are the pool's executable ones, as opposed to queued
(`PoolContent`, `PoolStats`). The `pending` epoch is the open epoch as the node
has it now (`LiveEpochState`). Nothing else is called pending: the oracle's
pool sample is `PoolNextTransactions`, and the rules staged for the next epoch
are `OracleInputs.NextEpochRules`.

**Not on the interface**: `GetEVM`, `AccountManager`, `CalcBlockExtApi`,
`ExtRPCEnabled`, the `RPC*` limit getters, `CurrentBlock`. `Core` grows only
when a method cannot be served from what is there, and never by a call whose
answer depends on "now" unless it is the pool, liveness or one of the live
consensus calls named above. A port that seems to
need a new call has more often misread the legacy handler; it stops and records
the case in section 11.

**The head subscription never blocks the feed.** `event.Feed.Send` waits for
every subscriber, so the head loop only hands events to bounded per-consumer
queues; a consumer that falls behind is dropped and told, and so is its client
(I15, I18).

**One `latest`.** Every tag resolves once per request to the head the module
last received, for state, blocks, receipts, logs and `eth_blockNumber` alike.
Legacy answers `eth_blockNumber` from the committed head and state from the
archive head, so a client can see a number it cannot yet query. The difference
shows when the archive lags and is annotated as a fix (ADR4).

## 5. Runtime: dispatcher and transports

```go
// A handler takes its parameters as one struct and returns a typed result.
type Handler[Params, Result any] func(ctx context.Context, params Params) (Result, error)

// Add registers a method. A package function, because Go methods take no
// type parameters; both types are inferred from the handler.
func Add[Params, Result any](dispatcher *Dispatcher, method string, handler Handler[Params, Result])

// The positional parameters of eth_getBalance, in order.
type getBalanceParams struct {
	Address common.Address
	Block   *chain.BlockReference // a pointer field is optional
}

// registered as: dispatch.Add(dispatcher, "eth_getBalance", handlers.getBalance)
```

**Typed registration (D13).** A method's parameters are a struct: its fields
are the positions of the JSON-RPC array, in order. A pointer field is optional
and all optional fields trail, which is geth's rule for a pointer argument, so
the count and `null` rules of D9 are decided by the type and written once in
the dispatcher. `Add` reads the struct by reflection once, at registration, and
refuses a shape it cannot serve (a field that is not exported, a required field
after an optional one). A request is decoded from that prepared layout, with no
handler code: the dispatcher still reaches each field through `reflect`, one
field address per parameter, which is small beside the JSON decoding of the
value and is held to that by a benchmark (D13). A handler never sees raw
arguments, never counts them and never checks one for `null`. A method without parameters takes `struct{}`. A position whose
form depends on an earlier one (the second parameter of `eth_subscribe`) is a
`json.RawMessage` field that the handler decodes. The result is the handler's
own type; `any` only where the reference answers in more than one shape
(`eth_syncing`, a block with hashes or with transactions), and
`dispatch.Encoded` where H3 applies.

`Dispatcher.Serve(ctx, request)` runs under one `recover` (panic → -32603,
stack in the log, counter), which covers decoding and encoding as well as the
handler: a custom `UnmarshalJSON` or `MarshalJSON` can panic too. In order:
look the method up (unknown → -32601), decode and count params (failure →
-32602 in geth's wording, D9), attach the deadline, run the handler, encode and
count the response (D2). The handler's own checks
follow the legacy order and wording (F2–F5). Parallel work runs in
`Dispatcher.Group(ctx)`, an `errgroup` under the same recover. Long-lived loops
(head, filter expiry, oracle sampler) start with `Start`, restart on panic with
backoff and stop on `Stop`. A restarted loop subscribes again and drops the
consumers of its feed (P3, I18). Admission is atomic towards `Stop` (P7).
`Stop` waits for work that ignores cancellation and logs what it waits for
(Q1).

Stream connections (WebSocket, IPC) put a `Notifier` on every request's
context; HTTP has none and answers `eth_subscribe` with -32601 `notifications
not supported`.

- `eth_subscribe` installs its feed synchronously, before the id is returned
  (#1162 fixed this in legacy).
- Producers run under `Notifier.Go`, the connection's recover group, and end
  with the connection.
- Closing a connection cancels first, then waits for its producers; the reverse
  order deadlocks.

The transports are HTTP, WebSocket and IPC (D3–D5, D11). `gorilla/websocket`
is already in the module graph. IPC serves requests from step 1 and
subscriptions from step 4; `sonictool attach` and the test harness dial it.

**The HTTP write timeout is answered, not suffered.** geth's server
(`handler.handleNonBatchCall`, `batchCallBuffer` in `rpc/handler.go`) starts a
timer at the write timeout minus 100ms; when it fires it cancels the method
and writes -32002 `request timed out` itself, with a `Content-Length`. In a
batch every unanswered call gets the error and the answers so far go out with
them; a notification is answered too; the method's late answer is dropped. The
module does the same:

- The timeout reaches the endpoint as a value, not a context deadline, which
  would let the method's own error race the timer.
- One write per request (`sync.Once` for a call, a locked buffer for a batch);
  the request path synchronises with the timer even without an answer of its
  own. geth itself races on a timed-out notification, so the differential test
  leaves that one case out.
- The methods that ignore cancellation (EVM runs on Tosca, section 10) are the
  ones that reach it; the differential test (Q5) includes a method slower than
  the write timeout as a call, a batch and a notification.

**Operator surface kept**: `--http.{addr,port,corsdomain,vhosts,api,rpcprefix}`,
`--ws.{addr,port,origins,api,rpcprefix}`, `--ipcpath`, `--ipcdisable`, HTTP
server timeouts (#1064), batch limits, `--mode validator`.

**Codec (ADR2).** An own JSON-RPC 2.0 layer on `net/http`, `encoding/json` and
gorilla, because the dispatcher must see every request before a goroutine
exists, and `rpc.Server` spawns first and reflects methods. It is the largest
and most security-sensitive part (envelope, batches, framing, origin and host
checks, size limits). Gates: geth's `rpc.Server` and `rpc.Client` as the
compatibility oracle in test files (Q5, D11), a fuzz target on the codec, and
an own review of the transport commits of steps 1 and 4.

### go-ethereum in the module

| Keep | Why |
|---|---|
| `common`, `common/hexutil`, `common/math`, `core/types`, `rlp`, `crypto`, `crypto/kzg4844` | the data model |
| `core`, `core/vm`, `core/tracing`, `eth/tracers` (+ `logger`, `native`) | messages and their application (`Message`, `ApplyMessage`, `GasPool`; `evmcore.TxAsMessage` returns a `*core.Message`), the EVM and the tracers |
| `trie` | the stack trie that hashes the transactions of a simulated block (`eth_simulateV1`), in `rpc/eth` only |
| `params`, `accounts/abi`, `log` | `ChainConfig`; revert-reason decoding; the logger |

Dropped from the module: `rpc` (own `BlockNumber`, `BlockNumberOrHash`,
subscription ids, server), `node` (own listeners), `event` (channels owned by
the dispatcher), `accounts` (X1), `ethdb`, `core/rawdb`, `triedb` and
`consensus*`. The budget is on direct imports; what a kept package imports in
turn is not counted. This is the module's import budget; the node
and the legacy stack keep theirs. A test enumerates the non-test imports of the
packages under `rpc/` and fails on a go-ethereum package outside the keep list;
the adapter is a node file and outside it. JavaScript tracers stay registered by
the launcher's import of `eth/tracers/js`, as today; the module only checks
`AllowJSTracers` (V6).

## 6. Work

This section is M1: one branch cut from `main`, in the order of the steps. No
pull request is opened during M1; every commit is written so that M2 can make
a pull request from it, alone or with its neighbours.

**Commits.**

- One reviewable unit each. The subject says what changes; the body says why,
  how it was checked, and what the author's self-review changed (H11). No plan
  references (H12).
- Dependency order: the branch up to any commit is a valid base for the next.
- Every commit builds and passes its tests and those of the packages it touches,
  under `-race` for `rpc/`. Legacy mode is unchanged (SW4). Module mode serves
  what is ported, with lakme green or annotated for the families the commit
  touches; an unported method answers -32601.
- Tests land with the code they test, in the same commit.
- A fix belongs to the commit it fixes: `git commit --fixup` and `git rebase
  --autosquash` while the fixed commit is only on its author's branch, a commit
  naming the fixed one after.
- Moves and deletions are commits of their own.

**Steps.** The work branch is linear. Steps 2, 4 and 6 depend only on step 1:
they may be written at the same time, each on a branch of its own cut from the
end of step 1, and each joins the work branch by rebase, in step order. Steps
3 and 5 start after step 2: they use its call arguments, overrides and EVM environment
(`sonic_prepareBundle`, `debug_traceCall`, `trace_call`). Each step has its own
commits; none may add to `Core` beyond section 4. A step is done when its last
commit passes the gates it names.

0. **Baseline.** Commits: `main`'s flags and TOML keys as golden files,
   with the tests that compare the node against them (O4). Outside the
   repository: publish the Sonic runner of the execution-apis fixtures (9.3);
   give lakme one flag per side that passes node arguments to that side
   (`-lakme.a.args`, `-lakme.b.args`), so that either side runs with
   `--rpc.stack module` (the benchmarks time side `a`, 9.2); file the issues
   section 8 still marks `issue to file`. Run lakme and the
   fixtures on unmodified `main` against the release (the release run, 9.1),
   and annotate every difference as a release difference in `defects.json`
   (among them the optional block selector of the state getters, if the release
   requires one). Gate: lakme green or annotated, fixtures green or
   allow-listed.
1. **Skeleton.** The module is built first and selected last: until the
   commit that adds `--rpc.stack` only its tests reach it, and from that
   commit module mode has a listener and a method to answer. Commits:
   - `rpc/core`: `Core` with every role as section 4 defines it, its value
     types, the test that freezes its method set, the import test with the
     Layout table and the legacy package list (I1, SW3); no generated mock
     (ADR13)
   - `rpc/tests`: `FakeCore` with panic injection per method, `NewCarmenCore`,
     accounts (9.4); later support lands with its first user
   - the combined head feed of `ServiceFeed`, then the adapter with its unit
     tests (T8). The adapter is complete except `Logs` (step 4) and the
     `ConsensusReader` half (step 6); until then those calls return an error
     that names the missing call, and a test lists them
   - `rpc/dispatch`, with typed registration (D1, D2, D9, D13, P1–P3, P7, Q1,
     O2)
   - `rpc/chain` (H1, H6, I2–I4, I9, I15, I18)
   - `rpc/exec`: `Environment` with its lookups, and `Config` (H1, H2, H8); the
     oracle and execution join it at step 2
   - metrics (MON1–MON5)
   - `rpc/transport`: message layer, HTTP, IPC without subscriptions, and
     `Front` with the IPC routing to the node handler, with the differential
     envelope test against geth's server and the codec fuzz target (D3, D5,
     D11, Q5, X2); the profiling stubs of `rpc/debug` in the same commit, so
     that no build without `enable_debug` reaches geth's `debug.Handler`
   - the `Module`, with `rpc_modules`
   - `--rpc.stack` and the module-mode start path in the composition root,
     after listing the legacy RPC components and the readers of the rewritten
     configuration (section 3); `takeOverRPC` for HTTP and IPC; the exec test
     of `cmd/sonicd/app` starts a node in each mode and reads `rpc_modules`
     (SW1, SW2, O5)
   - the stack selection of the integration test framework, with the `tests/`
     test of a node in each mode (9.5, SW1, SW5, SW6)
   - block, header and transaction reads with their forms in `rpc/format`;
     `eth_chainId`, `eth_blockNumber`, `eth_syncing`, `eth_config`, `net_*`,
     `web3_*` (I13, S1, H8)

   Gates: SW1–SW3, SW6 (with its list of files still importing legacy types),
   the `Core` surface test, the import table, H1, H3, H6, H8–H12, I1–I4, I13,
   I15, D1–D3, D5 without subscriptions, D9, D11, D13, P1–P5, P7, S1, X2,
   MON1–MON5, O3, O5, T8 for the adapter's calls so far.
2. **Oracle, state and execution.** The oracle comes first: call defaulting
   asks it for the tip. Commits:
   - `rpc/oracle` and `eth_gasPrice`, `eth_maxPriorityFeePerGas`,
     `eth_feeHistory`, `eth_blobBaseFee` (I10, ADR9)
   - `rpc/exec`: call arguments, overrides, the EVM environment, BLOCKHASH by
     lookup (I12, V5, V11, H2)
   - `eth_get{Balance,Code,StorageAt,TransactionCount,Account,Proof}` (I5–I7, V10)
   - `eth_call`, `eth_estimateGas`, `eth_createAccessList` (I8, F2–F5, V1–V6)
   - `eth_simulateV1` (its 256-block cap, section 10),
     `sonic_estimateGasForTransactions`

   Gates: H2, I5–I8, I10, I12, V1–V6, V10, V11, F2–F5, O1; lakme state, call
   and oracle families.
3. **Transactions and pool.** Commits:
   - `eth_sendRawTransaction`, `eth_sendRawTransactionSync` on the send path of
     `rpc/exec`, which `sonic_submitBundle` uses too (V7)
   - `eth_pendingTransactions`, `txpool_*`, `test_*`
   - `sonic_submitBundle`, `sonic_prepareBundle`, `sonic_getBundleInfo`

   Gates: V7; `rejected_tx_test.go`,
   `tests/rpcs/send_raw_transaction_sync_test.go`, the bundle tests in module
   mode.
4. **Events.** Commits:
   - `Core.Logs` in the adapter over `gossip/filters.Filter` and the adapter's
     own `filters.Backend`, then `eth_getLogs` (I11, F6, V4, ADR15)
   - the polled filters and their journal (D6, V8, I17, I18, F7, P8)
   - the WebSocket transport, the `Notifier` and subscriptions on WebSocket
     and IPC, checked against geth's client (D4, D5, D11)
   - `eth_subscribe`/`eth_unsubscribe` (`newHeads`, `logs`,
     `newPendingTransactions`), `takeOverRPC` for WebSocket with the shared
     HTTP port (I9, I14, I16, I18)

   Gates: I9, I11, I14, I16–I18, D4–D6, V8, F6, F7, P8; lakme filter family;
   subscription tests over WebSocket and IPC.
5. **Tracing.** The `debug_*` and `trace_*` methods of section 8. The replay
   follows `stateAtTransaction`, with the EIP-2935 write where section 8 puts
   it. Commits:
   - the module's copy of the OpenEthereum tracer of `txtrace` in `rpc/trace`
     (copied unchanged, in a commit of its own), then its size accounting (V12)
   - `debug_traceTransaction`, `debug_traceBlockByNumber`/`Hash`,
     `debug_traceCall`, `debug_getBlockRlp`, `debug_printBlock` (V6, F3)
   - `trace_transaction`, `trace_get`, `trace_block`, `trace_filter` (V2)
   - `trace_call`, `trace_callMany`, bounded as a whole (V12)

   Gates: V6, V11, V12, F2, F3; lakme trace families.
6. **DAG and epochs.** Commits:
   - the adapter half of `ConsensusReader` (section 4)
   - `dag_getEvent`, `dag_getEventPayload`, `dag_getHeads`
   - `abft_*`, `eth_getRules`, `eth_currentEpoch`, `eth_getEpochBlock`,
     `eth_getEpochStats`

   Gates: Q6 for these methods; lakme dag and abft rows;
   `event_extra_data_test.go`, `single_proposer` and `validator_stakes` in
   module mode.
7. **Completion.** Commits:
   - the non-test `Attach()` callers listed again; a new one served (O5)
   - the module CI stages of the integration tests, over HTTP and over
     WebSocket (SW5, T5)
   - the last legacy type import of `tests/` moved; the list of SW6 is empty
   - changelog, operator notes (how to select the stack, what module mode does
     not serve, ADR14) and the alerts of MON6

   Then the review of the whole branch against the reference by someone who did
   not write it, with 9.6 as the checklist (T7); each finding is fixed in the
   commit it belongs to, with a test that fails without the fix. Gate: every
   requirement of section 7 and every line of the M1 gate (9.7). With it
   green, the user approves the end of M1 (section 1).

## 7. Requirements

**Verified by**: L lakme, E execution-apis fixtures, R handler tests on
`rpc/tests`, I integration tests, U unit test (module packages and the
adapter), C CI lint or
import test. Every id gates M1; a step names those it must already meet. Rows
marked *Beyond the reference* make the module better where the reference has a
bug, a missing bound or a latent cost; three of them (I17, V11, V12) close a
denial of service. Where a client can see one, section 8 has a deviations row.

### SW. Stack selection

| ID | Requirement | Verified by |
|---|---|---|
| SW1 | `--rpc.stack legacy\|module` (TOML `Opera.RPCStack`) selects the stack at start; default `legacy`; another value refuses to start, naming both | U (each value and its effect, O4), I (a node in each mode answers `rpc_modules` as that stack) |
| SW2 | One stack runs, chosen in one function of the composition root. What each mode constructs, skips and owns is section 3; no method opens a node RPC server (X1) | U (registered APIs and goroutines per mode), I |
| SW3 | The module (`rpc/`, `gossip/rpc_core.go`) imports no legacy package and calls no `EthAPIBackend` method; the legacy stack imports nothing under `rpc/`; no helper is shared between them. What both need is copied, or is node code both use as a library. One exception, held by the test: the adapter's `Logs` uses `gossip/filters.Filter`, over a backend of its own (ADR15) | C (import test, the legacy package list and the legacy files of `gossip` as data) |
| SW4 | Legacy mode answers as `main` does | L (legacy run against the merge-base with `main`, 9.1: no difference, nothing annotated), I (legacy CI stage) |
| SW5 | Every integration test passes on each stack it applies to (9.5) | I |
| SW6 | A test for one stack says so with `RequireRPCStack` and skips on the other. Legacy-only tests live in `*_legacy_test.go`, the only files under `tests/` that may import a legacy package. Until step 7 the import test also allows the files that still import a legacy package for its types, listed as data; each commit that moves a type shortens the list, and it is empty at step 7 (9.5) | C (import test over `tests/`), review |

### I. Interface and state

| ID | Requirement | Verified by |
|---|---|---|
| I1 | The import rules and the Layout table of section 2; every core access is through `Core` | C |
| I2 | No `Core` call takes a tag: the module resolves `latest`, `pending`, `safe`, `finalized`, `earliest` once per request, and a call about chain or state carries a number, a hash or an id. Exception: the bounds of `LogQuery`, with the served head (section 4) | R (fake Core rejects a tag cast to a number, on every call that takes a block number), U |
| I3 | `latest` is the newest archived block last seen on `SubscribeHead`; the same number serves every part of a request and `eth_blockNumber` | L (annotated), I (`block_in_archive_test.go`) |
| I4 | `earliest` resolves to the pruning cutoff (`HistoryPruningCutoff`, today 0). Exception: a bound of a log query, which the legacy filter reads as the head (section 4) | R |
| I5 | `pending` state is the archive head's; the `pending` nonce is `Core.PoolNonce` | R, L |
| I6 | State views come from the archive only | R (fake Core has no live state), U |
| I7 | Every `StateDB` is released exactly once, by the request that opened it, by a `defer` before any code that may panic | R (panic injection), U |
| I8 | Methods that chain runs (`eth_call` with `preArgs`, `eth_simulateV1`, `sonic_estimateGasForTransactions`, `trace_callMany`, block replays) reuse one StateDB with `EndTransaction` between runs and `Copy()` for branches | R, L |
| I9 | The head subscription is the only source of the head, new blocks and logs, and the pool subscription (`SubscribeTransactions`) the only source of pending transactions; on the loss of the head subscription the module keeps serving the last head and logs the lag | U, I |
| I10 | The oracle runs on `OracleInputs` and `PoolNextTransactions`, sampling at 1s/5s as today; call defaulting reads its tip. A tick of the sampler reads the next executable transaction of every sender (`PoolNextTransactions`, what the legacy sampler takes from `PendingTxs`) and the minimum gas price of the current rules (`Rules(latest+1)`, section 4), then sorts and cuts the sample as the legacy sampler does. Without archive lag these are the legacy sampler's inputs; while the archive lags across an epoch end, the price is the served head's (deviations). `OracleInputs` walks every validator's last event, so it is read per suggestion, never per tick | L (tolerance rows), I (`gas_price_suggestion_test.go`), U (a tick does not call `OracleInputs`; `MinGasPrice` floors the suggestion) |
| I11 | `eth_getLogs` and `eth_getFilterLogs` are answered by `Core.Logs`, which runs the legacy query code (section 4, ADR15). The module decodes the criteria, passes the bounds as the reference reads them with its served head, and normalises the answer on a copy (P8). A tag bound, and a `blockHash` above the served head, answer as at the served head (I3). Polled filters and `logs` subscriptions match events with the module's own matcher | L, I (`rpc_get_logs_test.go`, `indexed_logs_test.go`; a `logs` subscription and `eth_getLogs` with the same criteria return the same logs), U (adapter: tag bounds and a hash above the served head; each decoding error of the criteria against the legacy text) |
| I12 | The EVM is built from `Rules(number)` and the header of the *queried* block, never the head's. Its chain config follows the block context's number after block overrides, as the reference's `GetEVM` does; the tracer's chain config and the VM config follow the queried block, as `traceTx` does | I, R (a block override `number` across an upgrade height) |
| I13 | Header-only reads (state getters, `eth_call`, estimates) use `Header`, never `Block` | U (fake counts `Block` calls), benchmark |
| I14 | The served head advances only after the block's logs are enqueued for subscribers, so `eth_getLogs` at `latest`, `eth_blockNumber` and log subscriptions agree | U, I (`log_subscription_test.go`) |
| I15 | The head loop never blocks on a consumer: bounded queues per consumer; a lagging consumer is dropped | U (stalled consumer, head still advances) |
| I16 | Work that depends only on the event is done once per event: the `newHeads` notification (receipts, encoded header) is built once per block for every subscription | U (N subscribers, one `Receipts` call per block) |
| I17 | *Beyond the reference.* Polled filters hold only events that arrive between two polls; `eth_getFilterLogs` is a query and goes to `Core.Logs` (I11). A polled filter costs nothing per event, and their number is bounded. Events of a kind live once in a journal shared by the filters of that kind; a filter keeps only its poll position, and its criteria apply when it polls, outside the lock. The journal keeps at most what the oldest live filter has not polled (at most `FilterTimeout` of events) and at most 2^20 events; a filter whose events were dropped is removed and answers `filter not found`. At most 100,000 polled filters per node | U (one copy of an event for 1000 filters; the cap; a filter that fell behind is removed, one that polled gets every event; the journal shrinks) |
| I18 | A consumer that loses events is told, and so is its client; no consumer goes on with a gap. Events are lost when a consumer falls behind (I15) and when a feed loop restarts (section 5). A dropped subscription ends its connection with the reason. When the polled filters' feed is dropped, every filter of the kind answers `filter not found` and the feed subscribes again. A poll never returns a list with a gap | U (each consumer made to fall behind and each loop panicked; a filter installed afterwards gets every event) |

### D. Dispatcher and transports

| ID | Requirement | Verified by |
|---|---|---|
| D1 | All request goroutines are created by the dispatcher or its `Group`; a bare `go` statement elsewhere in the module fails a lint test | C |
| D2 | Lookup, decode, deadline, handler, encoding and metrics happen in the dispatcher, in that order, under one recover, for every method the module registers, on every transport. A call forwarded to the node handler passes none of these stages (X2) | U (per stage; a panic in decoding and one in encoding answer -32603), R |
| D3 | HTTP: single and batch requests, body cap, batch item and response caps, CORS, vhosts and path prefix as today, gzip as geth's server (ADR12), no subscriptions; the write timeout answered as geth does (section 5) | I, L, U (differential test with a slow method) |
| D4 | WebSocket: requests, batches, subscriptions (`newHeads`, `logs`, `newPendingTransactions`) installed before their id is returned, a subscription cap per connection, a bounded buffer per subscription, disconnect of a client that does not read | I (`log_subscription_test.go`, `pending_transactions_subscription_test.go`), U |
| D5 | IPC at `<datadir>/sonic.ipc` (`--ipcpath`, off with `--ipcdisable`), mode 0600, newline-delimited JSON, the features of WebSocket. IPC has no API list: it serves every namespace the module registers, as geth's IPC serves every registered API, and is the only transport of the node handler's methods (X2) | I (harness IPC client), U (every registered method reachable over IPC) |
| D6 | Filters expire 5 to 6 minutes after their last poll: the sweep runs five times per `FilterTimeout` (deviations); ids are unguessable and per node | L, U |
| D7 | No `ftm` namespace: `ftm_*` answers -32601 on every transport, and `ftm` in `--http.api` or `--ws.api` is ignored with a module-mode warning that says so; legacy mode keeps the launcher's deprecation warning unchanged | U, I |
| D8 | Namespaces are enabled per transport by `--http.api` and `--ws.api`, defaults as today. `admin` in those lists is ignored, with a warning; `debug` there enables the module's `debug_*` methods, never the node handler's | I |
| D9 | Unknown method -32601; malformed JSON -32700; a parameter the dispatcher cannot decode or count -32602 in geth's words (`invalid argument N: …`, `missing value for required argument N`, `too many arguments, want at most N`, `non-array args`); internal -32603; a handler's own check keeps the code and words of its legacy counterpart | L (error rows), E, U (each wording against geth's server, Q5) |
| D11 | Transport security as geth's: Host/vhost check, CORS, origin check, body and message caps, read/write/idle timeouts | I, U (a disallowed Host, a disallowed origin, an oversize body, a stalled client; geth's `rpc.Client` as oracle) |
| D13 | Typed registration (section 5). A method is `dispatch.Add(dispatcher, name, handler)` with a handler `func(ctx, Params) (Result, error)`; `Params` is a struct whose fields are the positional parameters, a pointer field optional and trailing. The dispatcher decodes, counts and applies the `null` rules from the type, in the words of D9; no handler takes raw arguments. `Add` panics at start on a `Params` shape it cannot serve and on a method registered twice. The field layout is read by reflection once at registration; a request costs one field address per parameter through `reflect` and no layout work, and typed decoding is no slower than decoding the same parameters by hand | U (table over parameter shapes: missing, extra, `null` for a required and for an optional field, each against geth's server, Q5; `Add` refusing each bad shape), benchmark (typed decoding of `eth_getBalance` and `eth_call` parameters against hand decoding of the same: time within noise, no added allocation per request), C (a test fails on a handler that names `json.RawMessage` parameters outside a listed set) |

### P. Panics, goroutines and locks

| ID | Requirement | Verified by |
|---|---|---|
| P1 | A panic in a handler or child is recovered, answered -32603, logged once with the stack, counted in `rpc_panics_total{method}`; RPC work never ends the process | R (panic injection on every `Core` method), U |
| P2 | Child goroutines run in a `Group`: a panicking child becomes an error, cancels its siblings, and the parent waits for all | U, R |
| P3 | Loops restart on panic with backoff and a counter; a loop that panics repeatedly is reported. A restarted loop subscribes again (the head loop also reads `Head()`) and drops the consumers of its feed (I18) | U (each loop panicked does its job afterwards: the head advances, the pool feed delivers) |
| P4 | Handlers hold no locks. Shared structures expose small methods that lock and `defer` unlock in the same function and never call `Core`, a handler or the EVM while locked | U (a blocked `Core` call does not stall other requests), review |
| P5 | Resources with a lifetime (state views, subscriptions, tracer buffers) are released by a `defer` on the line after acquisition | R (panic injection; budget back at zero) |
| P6 | Module tests run under `-race` | C |
| P7 | Admission is atomic towards `Stop`: the stopping check and the count `Stop` waits on are taken under one lock, for requests, subscriptions, `Go` and `Loop` | U (requests and `Go` racing `Stop`, `-race`) |
| P8 | *Beyond the reference.* What the node hands out is shared and read-only: blocks and logs of a `HeadNotify`, cached receipts, headers and logs. Normalising for an answer (nil topics to `[]`, a log's tx index) works on a copy | U (a shared log is unchanged after `eth_getFilterChanges`), P6 |

### Q. Release guarantees

| ID | Requirement | Verified by |
|---|---|---|
| Q1 | Shutdown completes within the node's timeout under RPC load. `Stop` closes listeners and subscriptions, drains in-flight requests up to the deadline, and returns with no module goroutine alive (`node_restart_test.go`). After the grace period `Stop` cancels and waits; that wait is bounded by the longest work cancellation does not reach (an EVM run on Tosca to `RPCGasCap`, one `Core` call), and `Stop` logs each request it still waits for, with method and age | U (goroutines after `Stop` under load equal those before `Start`; `Stop` timed during a Tosca run at the gas cap and during a blocked `Core` call, with the log line), I (node stopped under load) |
| Q2 | Cancellation is observed on every path: a cancelled request returns within one `Core` call or one interpreter poll; no handler waits on a channel without the context. Exception: Tosca never polls, so its EVM runs are bounded by `RPCGasCap` | U (every method cancelled under `FakeCore` latency) |
| Q3 | Every method answers without panic when any `Core` method returns nil, not-found or an error, and for any `null` or absent optional argument (the dispatcher hands the handler a nil pointer field, D13) | R (`FakeCore` failure modes × methods × argument shapes; `rpc_panics_total` zero) |
| Q4 | Line coverage of every module package at the floor (ADR10), measured in CI; every registered method has handler tests covering the minimum set of 9.4, checked against its legacy handler | C (a test fails on a registered method no test names), review (the set, per method) |
| Q5 | Every method of the `ethereum/execution-apis` OpenRPC specification (`eth`, `debug`; the document pinned at step 0 and kept as test data) is served or listed in X with its reason; `txpool`, `net` and `web3` have no specification and follow the inventory of section 8; the JSON-RPC 2.0 envelope (ids, batches, notifications, error codes, `null` results, timeouts) matches geth's `rpc.Server`, including a method slower than the write timeout and a batch over each cap | C (served set diffed against the pinned specification), E, L, U (differential test, fuzz target) |
| Q6 | Sonic extensions are served, tested like geth methods, and listed in section 8: `sonic_*`, `trace_*`, `dag_*`, `abft_*`, `test_*`, `eth_getRules`, `eth_currentEpoch`, `eth_config`, `eth_getEpochBlock`, `eth_getEpochStats`, `eth_sendRawTransactionSync` | R for every method; L and I as the inventory lists per method (section 8): lakme draws no `sonic_*`, `test_*` or `eth_sendRawTransaction*` (9.1) |

### V. Validation

| ID | Requirement | Verified by |
|---|---|---|
| V1 | Block selectors: a known tag, a number in `[cutoff, head]`, or a known hash; anything else -32602 (unknown height) or "block not found" where the code says so | L, E, R |
| V2 | Ranges: `from ≤ to`, both resolvable, span within the method's limit (`eth_feeHistory` count clamped at 1024, `trace_filter` ≤ `MaxTraceFilterRange`). The range of a log query is checked by the reused query code, not by the module (F6) | L, E, R |
| V3 | Quantities are canonical hex and fit their width; gas ≤ `RPCGasCap`; prices and values non-negative | E, R |
| V4 | Addresses 20 bytes, hashes and topics 32; topic nesting ≤ 2; topic and address parameters ≤ `LogQueryParameterLimit` on every filter path | L, R |
| V5 | Call arguments: `data` and `input` agree; blob, access-list and authorization-list shapes; `state` and `stateDiff` exclusive per account; block overrides sane; `preArgs` bounded | R, E, L |
| V6 | Tracer requests: name from the allow-list unless `AllowJSTracers` (else `custom tracer is not permitted`), config decodable, `timeout` parses and ≤ `RPCEVMTimeout`, `trace_*` types from `{trace, vmTrace, stateDiff}` | L, R |
| V7 | Raw transactions: RLP decodes to a known type, chain id matches, EIP-155 unless `AllowUnprotectedTxs`, fee ≤ `RPCTxFeeCap`; the pool's verdict returned verbatim; sync timeout ≤ `TxSyncMaxTimeout` | I, R |
| V8 | A polled filter id exists; filters are per node, as in the reference, so any connection may poll one (D6). A subscription id exists and belongs to the calling connection | L, U |
| V9 | No allocation proportional to a client-chosen number before its bound is checked (block count, storage keys, topics, batch items) | U |
| V10 | `eth_getProof`: at most 1024 storage keys, strict key decoding, both -32602 before any state access | L, R |
| V11 | *Beyond the reference.* No client value decides how far a lookup walks, beyond a limit of section 10. BLOCKHASH is one lookup by number (`Core.Header`), the zero hash at or above the block of the call, never a walk of parent hashes (`evmcore.GetHashFn`), which a block override `number` above the head turns into one header read per block down to genesis. Applies to every block context built from a chain header (`exec.Environment.BlockContext`). `eth_simulateV1` builds its own block context and reads at most 256 headers per BLOCKHASH, simulated or stored (section 8) | U (`Core.Header` calls counted under overrides above and below the head, and at the type's maximum; in a simulation, for a number below its base) |
| V12 | *Beyond the reference.* A response that collects traces is bounded by `MaxResponseSize` as a whole: `debug_trace*` and `trace_filter` as in the reference, and `trace_call` and `trace_callMany`, whose calls share one budget and encode each result when it is there. A vmTrace stops at what is left of the budget, counting all it keeps in the larger of its memory and response cost: per operation, per pushed value (`SWAP16` and `DUP16` push 17 words for 3 gas), code and memory as hex | U (a vmTrace over the limit fails before the run ends; two calls that fit alone fail together; the accounting of pushed values, code and memory) |

### F. Fidelity to the reference

Wherever a client can observe a difference, the reference is right: the module
matches it bit for bit, and every difference lakme finds is a deviations row
(section 8) with its `defects.json` entry, or a defect that is fixed. F2–F7 are
the classes a port produces when it reads the legacy handler's intent instead
of its code.

| ID | Requirement | Verified by |
|---|---|---|
| F1 | Every observable answer (result, error code, wording, and which of several failing checks is reported) matches the reference unless a deviations row says otherwise | L |
| F2 | Checks run in the legacy order: block, then state overrides, then the call object; a tracer's name, config and `timeout` are checked where the tracer is built, after its subject is found | L, U |
| F3 | A missing block is worded as each legacy handler words it: `header not found` (state getters, `eth_call`); `block does not exists` or `block with hash %s not found` (`eth_estimateGas`); `block #%d not found` (`debug_*`); `cannot trace block %d got invalid block for tracing` (`trace_block`); the tracing wording of #1207 | L, U (one assertion per wording) |
| F4 | A handler's own check (fee fields, `data` vs `input`, `state` vs `stateDiff`, reward percentiles) is a plain error with the legacy text, so -32000; an unknown subscription type is -32601 `no "x" subscription in eth namespace` | L, U |
| F5 | `err: %w (supplied gas %d)` wraps a failed message only where legacy went through `DoCall` (`eth_call`, `eth_estimateGas`, `eth_simulateV1`); `eth_createAccessList` reports the bare error under `failed to apply transaction from sender …` | L, U |
| F6 | A log query over a range answers as the reused query code does (ADR15), and the module adds no range logic of its own: the bounds are clamped to the served head, the range limits are checked on the clamped bounds, and a walk ends at the first block it cannot read and returns what it has | L, U (adapter) |
| F7 | `eth_newPendingTransactionFilter(true)` serves whole transactions with no block fields, each priced by its own gas price | L, U |
| F8 | Known bugs of the reference (section 8) are reproduced, not fixed, unless the table says otherwise; each has an issue with one test that fails on `main`. The test stays with its issue and is not on the work branch, where every commit passes its tests (section 6) | L |

### H. Code economy and readability

None of these show in an answer, so no tool of section 9 sees them. Before
porting a package, list the repeated shapes of the legacy code (9.6) and write
each once.

| ID | Requirement | Verified by |
|---|---|---|
| H1 | One path from a reference to a block. `Chain.Resolve(head, reference)` turns a number, tag or hash into a number and reports a missing selector, an unknown hash and an invalid tag as sentinels. One set of lookups on `exec.Environment` (`HeaderAt`, `BlockAt`, `FindBlock`, `TransactionByHash`, `OpenState`, `StateAt`) serves every namespace; the number/hash switch is written once. Lookups return `(value, error)`, never `nil, nil`, and never turn an error into "no value". A method maps a sentinel to its wording (F3) or to `null` through one helper (H5) | U (table over `Resolve`; no lookup answers `nil, nil`), review |
| H2 | Read the least that answers. A method that uses no header reads none (`OpenState`, not `StateAt`). An adapter method needing a stored block's field reads the block, not the EVM header. What a request needs about its block is read once and held | U (fake counts `Header` calls), profile (9.2) |
| H3 | Encode once. A handler that encodes to check a size returns `dispatch.Encoded`, passed on unchanged. `json.RawMessage` only for bytes the module did not write (a tracer's output) | U |
| H4 | No dead code. An exported symbol has a non-test caller or implements an interface; an unexported one has a caller. A refactor deletes what it orphans. No forwarding-only wrapper. Test-only helpers live in test files or `rpc/tests` | C (unused-symbol check with the module's Go version; test-only exports listed and justified), review |
| H5 | One definition of a fact: the chain id as `*big.Int`, the signer, the head's base fee, a shared error wording (`exec.UnknownHashError`), the optional-timeout context, the mapping of `ErrNotFound` to no value. A conversion seen a third time becomes a function | review |
| H6 | Block numbers are `uint64` inside the module. `BlockNumber` is for client input and holds tags as negatives with go-ethereum's values, so a `uint64` above `MaxInt64` cast to it becomes a tag. Where the reference casts (`debug_getBlockRlp`), F1 wins and the cast stays in that handler with a comment | U, review |
| H7 | The same shape twice is a function, within a function and across handlers (the subscription loops) | review |
| H8 | Dependencies narrow and held once. `exec.Environment` keeps `Core` unexported and offers the lookups, head, limits and oracle. A namespace needing a node fact beyond it receives the one role interface that has it in `Register`, never `core.Core`. A handler struct holds each dependency once. An interface is declared where used and lists all its user calls; no capability is found by type assertion. An unused role interface is deleted | C (no namespace names `core.Core`; no exported `Core` field on `Environment`), review |
| H9 | Names follow Go's conventions and the repository's: the wider a name's scope, the more it says. Exported names, fields and package-level names are whole words in the domain's terms (`dispatcher`, `environment`), and no name hides a package. A parameter is never a single letter; the conventional `t` and `b` of tests are the exception. A method receiver may be one letter. A local of a few lines may be short (`i`, `ok`, `err`), and abbreviations the code base already uses (`ctx`, `tx`, `cfg`, `id`) are fine anywhere. Names owned elsewhere keep their spelling (go-ethereum, Carmen, node packages, JSON-RPC names, flags, TOML keys, metrics). Flat control flow: early returns, no `else` after `return`. Clean under `.golangci.yml`. A comment says why | C (`golangci-lint`), review; no test of its own |
| H10 | Small functions doing one thing, named for it. A handler reads as its steps (resolve, check, read, answer); a step with logic is a named helper. A body needing comments to mark its parts is split there; nesting over three levels is flattened. A handler lives in the file of its feature, beside its helpers; the file with `Register` holds only the package doc, the handler struct and `Register` | review |
| H11 | Self-review: before a commit, its author reads the whole diff as a reviewer and checks it against H1–H10, H12 and the error rules (section 4, D9, F3–F5); findings are fixed in the same commit, and the body says what the review changed. It does not replace T7 | review (commit body) |
| H12 | Code, comments, documentation, tests and commit messages do not refer to this plan (its sections, requirement or decision ids) or to any branch: the plan is not in the repository, so such a reference explains nothing to a reader of the code | review (H11, T7); no test of its own |

### S. Surface

| ID | Requirement | Verified by |
|---|---|---|
| S1 | A method is callable only because a `dispatch.Add` line in a namespace's `Register` names it; no method is found by reflecting over a struct (D13 reads the fields of a parameter type, not methods). The node handler's methods on IPC are the exception (X2) | U (a method no `Add` names answers -32601 on every transport, the forwarded ones on IPC aside), L |

### O. Limits and the operator surface

| ID | Requirement | Verified by |
|---|---|---|
| O1 | Semantic caps keep their names, defaults and effect (section 10) | L (rows that hit each cap), I |
| O2 | `RPCTimeout` is applied by the dispatcher to every request it serves (a call forwarded to the node handler is X2); the module does not use the fork's `SetExecutionTimeLimit` | U, C (lint on `SetExecutionTimeLimit`) |
| O3 | `--mode validator` keeps its effect: in the reference it drops the archive and indexes and leaves the endpoints as configured; module mode takes them over the same way. Without an archive the served head is the committed head (the feed delivers every block, lag zero), block and receipt reads work, and `State` fails with the reference's error (read at step 1 from `LastBlockWithArchiveState` and the state lookup of `EthAPIBackend`), so no read falls back to live state (I6) | U (adapter on a store without archive), I |
| O4 | Every CLI flag and TOML key keeps its effect, or is deprecated (accepted, warns, no effect) for one release; the loader rejects unknown fields, so removal breaks configs. "Keeps its effect" is tested by changing the key and observing the change in each stack that reads it; every mapping into the module's `Config` has a row that sets it | U (flags and keys against the golden files; one effect test per key) |
| O5 | What the node derives from its RPC configuration and handler is settled for module mode: the reported endpoints are the module's; `--unlock`, `--password` and `--allow-insecure-unlock` are accepted, warn and unlock nothing; the wallet event loop is not started; every non-test `Attach()` caller is listed, and a new one is served (section 3) | I (`cmd/sonicd/app` exec test: a module-mode node with `--unlock` and HTTP open starts and warns), U (one `takeOverRPC` test per endpoint layout geth distinguishes) |

### MON. Monitoring

| ID | Requirement | Verified by |
|---|---|---|
| MON1 | `client_golang` on the default registry, served by the node's metrics endpoint; no go-ethereum metrics registry in the module | C |
| MON2 | `rpc_requests_total{method,result}`, `rpc_duration_seconds{method}`, `rpc_response_bytes{method}`, `rpc_connections{transport}`, `rpc_inflight`, `rpc_subscriptions{kind}`, `rpc_subscription_events_total{kind,outcome}`; every transport counted alike. Calls forwarded to the node handler are not counted (X2) | U |
| MON3 | `rpc_head_served`, `rpc_head_lag` (committed minus served) | U |
| MON4 | `rpc_panics_total{method}`, `rpc_loop_restarts_total{loop}` | R |
| MON5 | Bounded cardinality: unknown methods map to `other`; no label by address, block, client or error text | U |
| MON6 | Alerts named in `rpc/README.md`: panics > 0, head lag over a block for a minute, p99 per method over its budget. Installing them is operations' work, not a gate | review |

### X. Not served, not added

| ID | Requirement |
|---|---|
| X1 | Not served in module mode: `personal_*`, `eth_accounts`, `eth_sign`, `eth_signTransaction`, `eth_sendTransaction`, `eth_fillTransaction`, `eth_resend`, `debug_chaindbProperty`, `debug_chaindbCompact`, `debug_setHead`, `admin_startHTTP`, `admin_startRPC`, `admin_startWS`, `admin_stopHTTP`, `admin_stopRPC`, `admin_stopWS` (the module owns every endpoint, SW2; these are not routed to the node handler), and the deprecated `ftm_*` mirror of `eth_*` (D7); they answer -32601. The module has no `AddrLocker` and no account manager. Operators who need them run legacy mode |
| X2 | In module mode IPC routes every `admin_*` and `debug_*` method the module does not register, except those of X1, in process to the node handler (`stack.RPCHandler()`); no other transport reaches it. Routing is per method, because `debug` holds the module's tracing methods and the node's profiling methods. Profiling answers as in the reference: a build without `enable_debug` registers the module's copy of the reference's stubs (`this method is not enabled`, on every transport that enables `debug`), so geth's `debug.Handler` is reached only in a build with `enable_debug`. `net_*` and `web3_*` are the module's own. A forwarded call is outside the dispatcher's stages (D2): the node handler decodes it, recovers its panics and bounds it with the fork's own execution limit, so it has no typed registration, no `RPCTimeout` and no module metrics. It runs in a goroutine the dispatcher starts (`Go`), so D1 and `Stop` (P7, Q1) hold for it |
| X3 | Not carried over: `CalcBlockExtApi` and `RPCBlockExt`, which nothing on `main` reads (the key stays accepted, O4), and the `PublicEthereumAPI` type of `gossip/api.go`, whose methods `rpc/eth` answers as constants; `eth_chainId` has one implementation where legacy has two |
| X5 | Not added: `debug_getRaw{Block,Header,Receipts,Transaction}`, `debug_getBadBlocks`. geth serves them, the reference does not (F1); listed for Q5, fixtures allow-listed (ADR7) |

### T. Testing

| ID | Requirement | Verified by |
|---|---|---|
| T1 | The same-commit lakme run (legacy against module, 9.1) is green on every commit touching the module: for the ported method families until step 7, for all of them from then on; every difference is fixed or a `defects.json` entry with a one-line reason | L |
| T2 | Module-mode benchmarks by the protocol of 9.2: geomean at or below legacy mode of the same commit, no row slower after its re-run | L |
| T3 | The execution-apis fixtures pass in module mode, or are allow-listed | E |
| T4 | `rpc/tests` holds the test support of 9.4 and the tests follow TS1–TS13; every method has handler tests through the dispatcher | R, review |
| T5 | The `tests/` integration tests pass on both stacks over HTTP in CI (SW5), and on the module stack over WebSocket in CI (`SONIC_TESTS_RPC_TRANSPORT=ws`) | I |
| T6 | Module unit tests cover dispatcher stages, transports, filters and the oracle, under `-race` | U |
| T7 | Before step 7 ends, someone who did not write the branch reviews it against the reference with 9.6 as the checklist; every finding is fixed with a test that fails without the fix, or becomes a deviations row | review |
| T8 | The adapter has unit tests for every `Core` method on a real in-memory store: a found item, `ErrNotFound`, the `TxIndex`-off errors, `Rules(head+1)` and `EpochOf(head+1)` when the head seals an epoch, a stopped feed, and `Logs` with tag bounds and a hash above the served head; and one test per `svc.config` read (section 4) | U |

## 8. Method inventory

Decision: **port** (new handler), **local** (answered without `Core`), **node**
(the node handler, IPC only, X2), **no** (not served in module mode, X1),
**absent** (geth has it, the reference does not, X5). Coverage: R handler
tests with the minimum set of 9.4, L lakme, E fixture, I integration test.
Every `port` and `local` row has R; a row without it is a gap.

### eth

| Method | Decision | Legacy | Cov. | Notes |
|---|---|---|---|---|
| eth_chainId | local | `PublicBlockChainAPI.ChainId` | R L E | one implementation (X3) |
| eth_blockNumber | local | `PublicBlockChainAPI.BlockNumber` | R L E I | archive head (I3) |
| eth_syncing | port | `PublicEthereumAPI.Syncing` | R E | `Progress` |
| eth_gasPrice, eth_maxPriorityFeePerGas, eth_feeHistory, eth_blobBaseFee | port | `PublicEthereumAPI.*` | R L E | module oracle (I10); count clamp 1024 (V2) |
| eth_getBalance, eth_getTransactionCount, eth_getCode, eth_getStorageAt | port | `PublicBlockChainAPI.Get*`, `PublicTransactionPoolAPI.GetTransactionCount` | R L E | pending nonce from pool (I5) |
| eth_getAccount | port | `GetAccount` | R L | storage root from the empty-storage flag |
| eth_getProof | port | `GetProof` | R L E | `State.GetProof`; V10 |
| eth_call | port | `Call`, `DoCall` | R L E I | revert data in error; `preArgs` (I8); overrides (V5); V11 |
| eth_estimateGas | port | `EstimateGas`, `DoEstimateGas` | R L E I | binary search bounded by `RPCGasCap`, `MaxGasLimit`; V11 |
| eth_createAccessList | port | `CreateAccessList`, `AccessList` | R L E | reproduces F8 bugs 1–2 |
| eth_simulateV1 | port | `simulate.go` | R L E I | 256-block cap, as the reference (section 10) |
| eth_getBlockByNumber/Hash, eth_getHeaderByNumber/Hash, eth_getBlockReceipts | port | `PublicBlockChainAPI` | R L E | |
| eth_getBlockTransactionCountBy{Number,Hash}, eth_getUncle*, eth_getTransactionByBlock{Number,Hash}AndIndex, eth_getRawTransactionByBlock{Number,Hash}AndIndex | port | `PublicBlockChainAPI`, `PublicTransactionPoolAPI` | R L E | no uncles |
| eth_getTransactionByHash, eth_getRawTransactionByHash, eth_getTransactionReceipt | port | `PublicTransactionPoolAPI` | R L E | pool fallback for pending |
| eth_sendRawTransaction, eth_sendRawTransactionSync | port | `SendRawTransaction`, `SubmitTransaction`, `SendRawTransactionSync` | R E I | V7; sync waits on the head subscription |
| eth_pendingTransactions | local | `PendingTransactions` | R | `[]`: the reference's answer without managed accounts (X1) |
| eth_newFilter, eth_newBlockFilter, eth_newPendingTransactionFilter, eth_getFilterChanges, eth_getFilterLogs, eth_uninstallFilter | port | `gossip/filters/api.go` | R L | D6, V8, I17; `eth_getFilterLogs` on `Core.Logs` (I11) |
| eth_getLogs | port | `PublicFilterAPI.GetLogs` | R L E I | `Core.Logs` (I11, ADR15); V2, V4 |
| eth_subscribe, eth_unsubscribe | port | `PublicFilterAPI` | R I | WebSocket and IPC |
| eth_getRules, eth_currentEpoch | port | `GetRules`, `CurrentEpoch` | R L | an epoch, as the reference; `ConsensusReader` |
| eth_config | port | `config.go` | R L | block rules and `ChainInfo` |
| eth_getEpochBlock, eth_getEpochStats | port | `GetEpochBlock`, `dag_api.go` | R L | `ConsensusReader` |
| eth_etherbase, eth_coinbase, eth_hashrate | local | `gossip/api.go` | R | constants (X3) |
| eth_accounts, eth_sign, eth_signTransaction, eth_sendTransaction, eth_fillTransaction, eth_resend | no | `PublicAccountAPI`, `PublicTransactionPoolAPI` | | X1 |

### Other namespaces

| Method | Decision | Legacy | Cov. | Notes |
|---|---|---|---|---|
| txpool_content, txpool_contentFrom, txpool_status, txpool_inspect | port | `PublicTxPoolAPI` | R I | |
| sonic_estimateGasForTransactions, sonic_prepareBundle, sonic_submitBundle, sonic_getBundleInfo | port | `api/sonicapi` | R I | `tests/rpcs/bundle_test.go`, `tests/bundles` |
| dag_getEvent, dag_getEventPayload, dag_getHeads | port | `dag_api.go` | R L I | `ConsensusReader` |
| abft_getValidators, abft_getDowntime, abft_getEpochUptime, abft_getOriginatedEpochFee | port | `abft_api.go` | R L | `ConsensusReader` |
| test_proposeTransactions, test_addTransactions | port | `api/testapi` | R I | `TestOnly`; always registered, `test-only API is not enabled` while `EnableTestOnlyApi` is off |
| debug_getBlockRlp, debug_printBlock | port | `PublicDebugAPI` | R L | printBlock is a spew dump (annotated) |
| debug_traceTransaction, debug_traceBlockByNumber/Hash, debug_traceCall | port | `PublicDebugAPI` | R L I | replay up to the transaction (I8); `MaxResponseSize`, `StructLogLimit` |
| trace_transaction, trace_get, trace_block | port | `tx_trace.go` | R L | |
| trace_call, trace_callMany | port | `tx_trace.go` | R L I | V12 |
| trace_filter | port | `tx_trace.go` | R L | replay in a `Group` (P2); `MaxTraceFilterRange` (V2); order and message vary today (annotated) |
| debug_getRaw{Block,Header,Receipts,Transaction} | absent | | | X5 |
| debug_chaindbProperty, debug_chaindbCompact, debug_setHead, personal_* | no | `PublicDebugAPI` | | X1 |
| debug_* profiling, build without `enable_debug` | local | `debugOverride.go` | R | the reference's stubs (X2) |
| debug_* profiling with `enable_debug`, admin_* except X1 | node | go-ethereum | I | IPC only (X2) |
| admin_startHTTP, admin_startRPC, admin_startWS, admin_stopHTTP, admin_stopRPC, admin_stopWS | no | go-ethereum | | X1 |
| net_version, net_listening, net_peerCount | local, port | `PublicNetAPI` | R | chain id; constant; `PeerCount` |
| web3_clientVersion, web3_sha3 | local | go-ethereum `web3API` | R | the module's own (X2) |
| rpc_modules | local | go-ethereum `RPCService` | R I | per transport (deviations) |

### Deviations of the module

Each row is a deliberate difference from the reference; anything else matches
it bit for bit (F1). A row lakme can see names its `defects.json` entry, added
in the same change.

| Method | Reference | Module | Why |
|---|---|---|---|
| X1 methods | served where their namespace is enabled | -32601 | X1; legacy mode serves them |
| `ftm_*` | mirror of `eth_*` and the filter methods, deprecated | -32601; `ftm` in an API list warns and is ignored | D7; removed with the legacy stack |
| `--unlock`, `--password`, `--allow-insecure-unlock`; hardware wallets | accounts unlocked at start, refused with HTTP open; wallets opened and self-derived | accepted, warned, no effect; no wallet event loop | X1, O5: no served method uses an account |
| `admin_*`; with `enable_debug`, the `debug_*` profiling methods | served on every transport that enables the namespace | IPC only, by the node handler | X2; `sonictool attach` reaches them |
| `admin_start{HTTP,RPC,WS}`, `admin_stop{HTTP,RPC,WS}` | start or stop the node's own servers | -32601 | X1: the module owns every endpoint (SW2) |
| `admin_subscribe` (`peerEvents`) on IPC | served | `no "peerEvents" subscription in admin namespace` | a subscription cannot be routed to the node handler |
| `rpc_modules` | every namespace the node's server registered | per transport, the module's namespaces; on IPC also the node handler's | X1, X2 |
| `eth_pendingTransactions` on a node with keystore accounts | the pending transactions those accounts sent | `[]` | X1: the module has no account manager; `[]` is the reference's answer on a node without accounts |
| `eth_newFilter`, `eth_newBlockFilter`, `eth_newPendingTransactionFilter` | unbounded | `too many filters, the limit is 100000` | I17 |
| `eth_getFilterChanges` | every event since the last poll | `filter not found` for a filter whose events left the journal, and for every filter of a dropped feed | I17, I18 |
| polled filter expiry | ticker every 5 minutes, so 5 to 10 minutes after the last poll | sweep five times per `FilterTimeout`, so 5 to 6 minutes | D6 |
| `eth_sendRawTransactionSync` | polls every 100 ms | waits on the head subscription | I9 |
| `debug_trace*` `timeout` | any duration | `-32602` above `RPCEVMTimeout` or malformed | V6; `rpc-module-tracer-timeout-limit` |
| `trace_*` replay with a short receipt list | panics | `no receipt found for transaction <hash>` | F8 |
| `debug_traceBlock*` message conversion error | ignored, then nil dereference | returned | F8 |
| `sonic_getBundleInfo` `block` | `rpc.BlockNumber` | `hexutil.Uint64` | no go-ethereum `rpc`; same JSON for reachable values |
| `sonic_prepareBundle`, `sonic_submitBundle` current block | live head | archived head | I3 |
| `eth_blockNumber` | committed head | archived head | I3, ADR4 |
| `eth_currentEpoch`, `eth_getEpochStats` and the epoch tags, while the archive lags across an epoch end | the store's current epoch | the epoch after the served head, `EpochOf(latest+1)` | I3; no difference without lag |
| gas price suggestions, while the archive lags across an epoch end | the sampler prices the pool with the open epoch's minimum gas price | with the minimum gas price after the served head, `Rules(latest+1)` | I10; no difference without lag |
| a subscription that loses events | not detected | ends, its connection closes with the reason | I18 |
| subscriptions per connection | unbounded | `too many subscriptions on this connection, the limit is 10000` | D4 |
| `trace_call`, `trace_callMany` | unbounded response; vmTrace stops near 256 MiB per call counting 128 bytes per operation | `ErrResponseTooLarge` above `MaxResponseSize` for the whole response | V12, ADR3 |

### Bugs of the reference (F8)

Confirmed on `main` and the release. The module reproduces the first two so
lakme stays green, and fixes the two panics (deviations above) and the
BLOCKHASH walk (V11, no answer changes). Legacy keeps all five; fixing them in
legacy is a change of its own, outside this work.

| Where | Bug | Fix | Tracking |
|---|---|---|---|
| `AccessList` (`api.go`) | without `gas`, the per-round `setDefaults` estimate runs before `args.AccessList = &accessList`, so the round that finds an entry runs with the previous gas and fails `intrinsic gas too low` | assign first, then re-estimate | sonic-admin#904, branch `luis/repro-createaccesslist-intrinsic-gas` |
| `setDefaults` (`transaction_args.go`) | the estimate's call object copies `Data`, not `Input`: an `input`-only call is estimated without calldata | copy `Input`, or use `args.data()` | sonic-admin#905, branch `luis/repro-createaccesslist-ignores-input` |
| `replayBlock` (`tx_trace.go`) | the untraced branch indexes `receipts[i]` unguarded | the traced branch's guard | issue to file |
| `traceBlock` (`api.go`) | ignores the error of `TxAsMessage`, then dereferences the nil message | return it | issue to file |
| `GetHashFn` (`evmcore/evm.go`) | lacks geth's `ref.Number <= n` guard; an override `number` above the head makes BLOCKHASH read every header down to genesis (20,000 headers for one `eth_call` on a 20,000-block chain, 460,000 for `eth_estimateGas`), and neither deadline nor `evm.Cancel` stops it | the guard (consensus unchanged); the module answers by lookup (V11) | issue to file |

Rejected, to stay identical: an error for an `eth_feeHistory` count above 1024
(legacy clamps; lakme draws up to 1100), and dropping the legacy fee and gas
defaulting of `eth_createAccessList` (changes `gasUsed`).

Not deviations (inherited behaviour, or structure no client sees):

- `TxSyncMaxTimeout` (30s) is unreachable while `RPCTimeout` (5s) bounds every
  request.
- `RPCEVMTimeout` bites only on geth's interpreter (section 10).
- The EIP-2935 history contract is written once on the parent state before the
  first transaction of every replay, as `main` does (#1160,
  `applyPreBlockSystemCalls`), in `exec.Environment.ReplayState`.
- `eth_simulateV1` takes call arguments and overrides from `rpc/exec` and
  builds its own block context, not `exec.Environment.BlockContext`: its block
  numbers are above the head, BLOCKHASH must see simulated predecessors,
  validation needs the base fee, and it has its own precompile set and
  log-tracer state wrapper. A simulated header carries the number of its block
  context, so BLOCKHASH reads at most 256 headers, simulated or stored (V11).

## 9. Testing program

### 9.1 Compatibility: lakme

`../lakme` starts two sonic nodes (two versions, or one commit in two modes)
from one exported genesis and draws queries over the whole history with
`rapid`, comparing answers field by field.

- **Same-commit run** (T1): `-lakme.a <commit> -lakme.b <commit>`, side A in
  legacy mode and side B with `--rpc.stack module` (the step 0 flags);
  `-rapid.checks 500` per commit, 3000 at the end of each step and at the M1
  gate. Both sides share every node change, so every difference is the
  module's.
- **Legacy run** (SW4): `-lakme.a <merge-base with main> -lakme.b <commit>`,
  both in legacy mode, at the end of each step and after each rebase: no
  difference, nothing annotated. Both sides are `main` but for the work, so a
  rebase never makes its baseline stale.
- **Release run**: `-lakme.a <release> -lakme.b <main>`, both in legacy mode,
  at step 0. Its differences are release differences. Against a
  2.2.x release use `-lakme.genesis testdata/synth_no_canto.g`.
- Deliberate deviations have no version bounds and an id starting
  `rpc-module-`; release differences, seen only by the release run, carry
  `introduced`/`fixed_in`.
  Where one side answers a result and the other an error, the diff path is
  empty: the entry needs `^(|error\\.(code|message))$`.
- An uncommitted tree is compared as a dangling commit (`GIT_INDEX_FILE` on a
  scratch index, `git commit-tree`).
- rapid stops a family at its first failure. Before the next round fix the
  class: every site of the same shape, checked against its legacy counterpart.
- Expected entries of the same-commit run: `rpc-module-tracer-timeout-limit`,
  `eth_blockNumber` archive head (if the chain lags), uniform `MaxResponseSize`
  (if a reply exceeds it). The release run keeps the entries already present
  (`trace_filter-range-limit`, `log-filter-parameter-limit-upfront`,
  `getProof-storage-key-validation`, `trace-call-missing-block-panic`) and the
  release differences of `main` (step 0): `getLogs-range-end-clamped` (#1177),
  `trace_call-pre-evm-error-refused`.
- lakme does not draw `eth_subscribe`, `eth_sendRawTransaction*`, `txpool_*`,
  `sonic_*`, `test_*`, `eth_syncing`: integration territory (9.5). It sees one
  configuration, one transport, no slow request, no panic, no shutdown and no
  concurrency: necessary, not sufficient (9.6).

### 9.2 Performance

Nine methods, two rows each (`latest` and `archive`; `debug_traceBlockByNumber`
on the busiest block and on the 50 busiest), so 18 rows. Side `a` is timed; a
sweep times one mode of the commit, passed to side `a` by the step 0 flag:

    go test -run '^$' -bench . -benchtime 1000x -count 5 \
      -lakme.a <commit> -lakme.b <commit> -lakme.a.args '--rpc.stack <mode>' \
      -lakme.genesis testdata/synth_no_canto.g

- One genesis for both modes; sweeps in balanced order `module, legacy,
  legacy, module`, twice.
- Each mode first agrees with itself within 5% geomean (`benchstat` over its
  sweeps); a disturbed sweep is discarded.
- Then `benchstat legacy.txt module.txt`; gate T2. With 18 rows, a run with no
  change shows one slower row at p < 0.05 in about one run in three, so each
  such row is re-measured alone, interleaved, with at least 60 samples per
  side; only a row slower again fails.
- The module saves a roughly constant cost per request; an `EstimateGas` or
  `TraceBlock/busiest` row moving more than a few percent means something else
  changed.
- **Profile before optimising.** A change for speed lands with the profile line
  of its cost and the `benchstat` of its gain. `-lakme.pprof 20` samples side
  `a`; read with `go tool pprof -top -cum` and `-peek <handler>`, looking for
  unused lookups (H2) and double encoding (H3). Profile each class of method at
  the end of steps 2 and 5.

### 9.3 Ethereum execution APIs

The `ethereum/execution-apis` fixtures (`tests/<method>/*.io`) run through a
Sonic runner built on `rpctestgen`, published at step 0, with a Sonic import of
`chain.rlp`/`genesis.json` and one allow-list: no uncles, no blobs, base fee
rules, the `debug_getRaw*` fixtures (X5). (`tests/eth_exec_spec` runs the
execution-spec state tests, a different suite.)

### 9.4 Handler tests: `rpc/tests`

Handler tests live next to their namespace and call through the dispatcher.
They test each method's logic and edge cases; lakme, the fixtures and the
integration tests do not replace them. The minimum set of every method, each
case a row (TS13), read from its legacy handler:

- the answer at `latest`, at a past block and at each tag the method accepts;
- a number above the head, an unknown hash and a missing selector, each with
  its wording (F3);
- each validation branch, one row per check (V1–V10) and per wording (F4, F5);
- each limit it applies, at the limit and one past it (section 10);
- the legacy order of checks when two checks fail (F2);
- the `Core` reads that must not happen (H2, I13);
- a panic-injection row for every `Core` method it touches (P1, TS8).

`rpc/tests` is the module's own test support, independent of `api/rpctest`,
and also serves `cmd/sonicd/app`.

| Piece | What it is | For |
|---|---|---|
| `FakeCore` | in-memory `Core`: a builder for blocks, receipts, pool and consensus facts; head and pool feeds; per method a mode (normal, panic, fail, not found, nil), a call count and a gate; one delay for every call | every handler test; failure, panic, cancellation; reads that must not happen |
| `NewCarmenCore` | a `FakeCore` whose `State(number)` copies one real in-memory Carmen state with an in-memory S5 archive, so witness proofs work (its `EndBlock` returns a nil channel without an archive; do not wait on it); `TrackStates` counts opened and released states (I7) | state getters and execution |
| the replay chain | a funded account, a contract writing storage, a block with a transfer and a call, over real state | `debug_trace*`, `trace_*` |
| accounts and code | signing on the fake chain id, a new account, binary encoding; code that spins and code that writes storage | tests needing a transaction or contract |
| the harness | a dispatcher and a `Chain` over a core, defaults changed by mutate functions; a variant with `exec.Environment` and an oracle; call, decode and expect-failure helpers; the recovered-panic count; a wait for the served head | calling a namespace as a transport does |

| ID | Rule |
|---|---|
| TS1 | `FakeCore` is the only double of `Core`; no generated mock (ADR13) |
| TS2 | A namespace test gets its harness from one call; the package helper only names what it registers |
| TS3 | A configuration change is a mutate function on the defaults, never a copy of them |
| TS4 | A fixture two packages use moves to `rpc/tests` when the second appears |
| TS5 | Tests wait for the served head with the harness's wait; every waiting helper has a deadline and fails the test at it |
| TS6 | A `Core` method named by a string (injection, counts, gates) must exist; the name list is derived from the interface |
| TS7 | An assertion compares a value read before the action with one read after it; a fixture built must be queried |
| TS8 | A matrix row checks its fault was reached (`rpc_panics_total` grew); a `Core` method no generic shape reaches gets a targeted request, or is named with the reason (only a loop calls it) |
| TS9 | The fake hands out copies and is changed only through its setters |
| TS10 | Metrics are read from a registry of the test's own through one scrape helper per package; modules and clients started by a test stop with it |
| TS11 | Test support adds no module dependency; tests of `core`, `dispatch`, `chain`, `exec`, `oracle` that use `rpc/tests` are external test packages |
| TS12 | Test support has no dead code (H4) |
| TS13 | Tests are table-driven: the cases of one behaviour are rows of a table (name, input, expected answer or error) run as `t.Run` subtests, so a new edge case is a row, not a new function. A case that needs its own setup or flow is a test of its own. The rule holds for the adapter's tests and for new tests in `tests/` whose cases share a network |

### 9.5 Integration tests: `tests/`

Every integration test runs against both stacks. The framework
(`tests/integration_test_net.go`, `session_dispenser.go`, `test_utils.go`)
gains:

- **Stack selection.** `SONIC_TESTS_RPC_STACK=legacy|module` (default
  `legacy`, another value fails the run) sets the stack of every network the
  run starts; `StartIntegrationTestNet` passes `--rpc.stack`.
  `IntegrationTestNetOptions.RPCStack` overrides it for a test that needs one
  stack; the session dispenser keys its shared networks by stack as well as by
  upgrades.
- **Per-stack tests.** `RequireRPCStack(t, stack)` skips a test on the other
  stack and logs why. Legacy-only tests live in `*_legacy_test.go` (SW6):
  `TestRPCApis`, which reflects over `api/ethapi`, and tests of X1 methods.
  Module-only tests (module limits) call it with `module`.
- **One client set for both.** The harness gets an IPC client per node
  (`GetIpcClient`), and `admin_*` calls (peer wiring in
  `integration_test_net.go`, `connect_p2p_test.go`, `bundles/pool_cleanup_test.go`)
  use it on both stacks (X2). `--http.api` keeps `dag`, `test` and `admin`;
  module mode ignores `admin` there (D8).
- **Stack-neutral types.** The types tests decode into (transactions, call
  arguments, block overrides, trace configuration, bundle info) come from the
  module's packages or the test package, never from `api/ethapi` or
  `api/sonicapi` outside `*_legacy_test.go`. Each moves in the commit that
  adds the namespace defining it, and its files leave the import test's list
  (SW6).
- **Served head.** In module mode `latest` is the archived head (I3) and may
  trail the committed head. A test that needs a block served waits for it with
  the existing `WaitForBlock`: a block above the served head is not found. A
  test that predicts the next block from `eth_blockNumber`
  (`trace_consistency_test.go`) retries until the prediction holds, as it does
  today. The module adds no endpoint for tests; the lag is the metric
  `rpc_head_lag` (MON3).
- **Transport.** `SONIC_TESTS_RPC_TRANSPORT=ws` makes the harness's clients use
  WebSocket, with `--ws.api` equal to `--http.api` (T5).
- **CI.** Jenkins runs `go test ./tests/...` once per stack, in parallel
  stages, and once more on the module stack over WebSocket (the module stages
  added at step 7); the unit tests run once.
- **New tests.** A node started in each mode serves its stack's
  `rpc_modules` and refuses an unknown `--rpc.stack` (SW1); an IPC console test
  (`sonictool attach` reaches the module and the node handler); a node stopped
  under RPC load (Q1).

### 9.6 What the tools do not see

lakme, the fixtures and the handler tests compare answers under one default
configuration. Each class below passes them all, so it has a gate of its own and
is a checklist item of T7. List each class's sites from the legacy code before
porting.

| Class (why the tools miss it) | Gate |
|---|---|
| Guards depending on node configuration (lakme runs the defaults) | an adapter test per `svc.config` read (section 4) |
| Config keys that parse but do nothing (a parse test passes) | an effect test per key (O4) |
| Node readers of the configuration `takeOverRPC` rewrites; in-process clients of the node handler (not on a transport) | O5, listed before steps 1 and 7 (section 3) |
| geth's endpoint rules (tests use one layout) | a `takeOverRPC` test per layout |
| Behaviour under a limit or timeout (requests finish in time) | the differential test with a slow method and each cap (Q5, D3) |
| The current-epoch rules (the head seals an epoch only at epoch ends) | `Rules(head+1)` and `EpochOf(head+1)` at every site, a row where the head seals an epoch |
| Restart after a panic; events lost without the consumer knowing (nothing panics or falls behind in a green run) | P3, I18 |
| Shutdown and admission races; work ignoring cancellation at shutdown | P7, Q1 |
| Mutation of shared values; cost multiplied by consumers (one consumer, one client) | P8, I16 |
| State a client makes the node keep (a test installs a few filters) | I17: every structure a request leaves behind has a cap, its per-event cost measured with many |
| Cost of a coarse `Core` call; work the answer does not use (the answer is right) | I10, H2, H3, the profiles of 9.2 |
| Cost a client chooses (honest values are fast) | V11: `Core` calls counted with hostile values: above the head, far below it, the type's maximum |
| A size limit counting the wrong thing (tested with the trace it was written for) | V12: count everything kept in the limit's unit, test with the most expensive operation per gas |
| Defaults written twice (they agree today) | one definition (section 10) |
| Copies, dead code, wide interfaces, absence said several ways (the tests pass anyway) | H1, H4, H5, H7, H8, section 4 "not found" |
| A package holding what its users own; a documented rule no test checks | the Layout table; each README rule names its test |
| Assertions that cannot fail (green whatever the code does) | TS6–TS8 and a review of every test helper |
| A dependency of the module on the legacy stack (both stacks work) | SW3, SW6: the import tests, with the log query exception (ADR15) as data; the removal list (section 3) reviewed at step 7 |

### 9.7 M1 gate

Every requirement's check in section 7 is green, and:

- `go test -race ./rpc/...`; `go test ./...`, with `tests/` once per stack (T5,
  SW5) and once in WebSocket mode on the module stack.
- the same-commit lakme run with `-rapid.checks 3000`, every difference in
  `defects.json` (T1); the legacy run without a difference (SW4).
- The benchmark protocol (T2); the execution-apis runner (T3).
- The unused-symbol check (H4), `golangci-lint` (H9), coverage at the floor
  (Q4).
- The self-review line in every commit body (H11); the alerts in `rpc/README.md`
  (MON6); the changelog and operator notes.
- Every finding of T7 fixed with a test or a row of section 8.

## 10. Limits inherited from the reference

The module changes no limit the reference enforces: every knob keeps its name,
default and effect (O1); only `RPCTimeout` changes mechanism (O2). Each module
default has one definition in the module, in the "Module definition" column.
The node configuration keeps its own literals (`gossip.DefaultConfig`,
`gossip/filters`, `gossip/gasprice`), so that a module change never changes a
legacy default; a test in `cmd/sonicd/app` checks each pair equal until the
legacy stack is deleted, when the node configuration refers to the module's
definitions. A knob of the
node configuration lives in the `Config` that carries it: `dispatch.Config` the
request deadline, `exec.Config` every semantic limit and switch, which the
namespaces read through `Environment` (H8), `oracle.Config` the `GPO` keys,
`chain.Config` the pruning cutoff; `rpc.Config` composes them and the
composition root fills each from the node configuration. A constant no key
sets (`maxGetProofKeys`, `maxFilters`, the transport caps) is defined in the
package that applies it. A knob the node configuration holds for the transport
(batch limits, HTTP timeouts) goes to `rpc/transport` as it is: the transport
has no default of its own for it, and its zero value means what it means in
the node configuration. Nothing bounds the requests running at once
(ADR14).

| Knob | Default | Kind | Legacy site | Module definition |
|---|---|---|---|---|
| `RPCGasCap` | 50,000,000 | cap on calls, estimates, traces | `gossip.DefaultConfig` | `exec.DefaultConfig` |
| `RPCEVMTimeout` | 5s | interpreter cancel; only geth's interpreter polls it (at jumps), Tosca's LFVM/SFVM never do, so there the bound is `RPCGasCap` | `DoCall`, tracers, `simulate.go` | `exec.DefaultConfig`; timeout test in `rpc/eth` |
| `RPCTimeout` | 5s | per-request deadline; legacy uses a fork patch (`SetExecutionTimeLimit`) | `gossip.NewService` | `dispatch.DefaultConfig` |
| `RPCTxFeeCap` | 100 FTM | refuse expensive sends | `checkTxFee` | `exec.DefaultConfig` |
| `MaxResponseSize` | 25 MiB (`--rpc.maxresponsesize`) | `debug_trace*`, `trace_filter`; in the module also `trace_call`, `trace_callMany` (V12) | `api/ethapi/api.go`, `tx_trace.go` | `exec.DefaultConfig` |
| `StructLogLimit` | 2000 | default tracer output, counted in bytes by go-ethereum although the flag says logs | `gossip.DefaultConfig` | `exec.DefaultConfig` |
| `MaxTraceFilterRange` | 1000 (`--rpc.tracefilterrange`, 0 = unlimited) | widest `trace_filter` span | `gossip.DefaultConfig` | `exec.DefaultConfig` |
| `eth_getProof` keys | 1024 | storage keys per request | `maxGetProofKeys` | `maxGetProofKeys` in `rpc/eth` |
| `LogQueryParameterLimit` / `LogQueryResultLimit` | 1000 / 5000 | filter parameters on every path / results | `gossip/filters.Config` | `gossip/filters.Config` (`FilterAPI`), applied by the reused `Filter` (ADR15); the parameter limit also in `exec.DefaultConfig` for `eth_newFilter` and `logs` subscriptions |
| `IndexedLogsBlockRangeLimit` / `UnindexedLogsBlockRangeLimit` | effectively none / 100 | block span of indexed / scanned log queries | `gossip/filters.Config` | `gossip/filters.Config` (`FilterAPI`), applied by the reused `Filter` (ADR15) |
| filter deadline | 5 min | unpolled filters dropped | `gossip/filters/api.go` | `exec.DefaultConfig` (`FilterTimeout`) |
| polled filters, journaled events | 100,000 per node, 2^20 per kind | new (I17, ADR6) | none | `maxFilters`, `journalLimit` in `rpc/filters` |
| `BatchRequestLimit` / `BatchResponseMaxSize` | 0 / 0, unlimited (`--rpc.batch-request-limit`, `--rpc.batch-response-max-size`); `NodeDefaultConfig` sets neither, and 1000 / 25 MB are geth's defaults, not Sonic's | batch limits | `config/node_config.go` | node configuration |
| HTTP body, WS message | 5 MiB, 32 MiB | transport limits | go-ethereum's `rpc` | `rpc/transport` |
| HTTP timeouts | read 30s, write 30s, idle 120s; -32002 at write − 100ms | `net/http` | `config/defaults.go` | node configuration |
| `GPO.MaxGasPrice` / `MinGasPrice` / `DefaultCertainty` | 10,000,000 gwei / 0 / 0.5 | ceiling and floor of the suggestion; certainty when none is named | `gossip/gasprice/gasprice.go` | `rpc/oracle`; ADR9 |
| `TxSyncDefaultTimeout` / `TxSyncMaxTimeout` | 5s / 30s | `eth_sendRawTransactionSync` wait; `main` only | `gossip.DefaultConfig` | `exec.DefaultConfig` |
| `AllowJSTracers` | false (`--rpc.allow-js-tracers`) | JavaScript tracers in `debug_trace*`; only built-in ones when false | `PublicDebugAPI` | `exec.DefaultConfig` |
| `AllowUnprotectedTxs` | false (TOML only) | accept raw transactions without EIP-155 | `EthAPIBackend.UnprotectedAllowed` | `exec.DefaultConfig` |
| `EnableTestOnlyApi` | false (`--enable-test-only-api`) | `test_*` do their work | `EthAPIBackend.IsTestOnlyApiEnabled` | `exec.DefaultConfig` |
| `eth_simulateV1` blocks | 256 | blocks per simulation | `maxSimulateBlocks` (`simulate.go`) | `maxSimulateBlocks` in `rpc/eth` |
| subscriptions per connection | 10,000 | new (D4, ADR6) | none | `maxSubscriptionsPerConnection` in `rpc/transport` |
| notification buffer | per subscription; size set at step 4 and named in `rpc/README.md` | new (D4) | none | `rpc/transport` |
| `--mode validator` | | O3 | `config/config.go` | node configuration |

lakme hits `RPCGasCap`, `MaxResponseSize`, `StructLogLimit`,
`MaxTraceFilterRange`, the log limits, the `getProof` cap and the
`eth_feeHistory` clamp (T1); integration tests cover `RPCTxFeeCap`, `TxSync*`,
batch limits, body and message sizes and `--mode validator`, added where
missing (T5); the timeout test in `rpc/eth` covers `RPCTimeout` and where
`RPCEVMTimeout` applies (T6).

## 11. Decisions

Decided: ADR1 module path `rpc/`; ADR2 an own JSON-RPC layer (section 5); ADR3
uniform `MaxResponseSize` (V12); ADR4 one `latest`, the archive head, for
`eth_blockNumber` too (I3); ADR5 the legacy-only namespaces are ported, X1
methods are not (X1, X2, Q6); ADR6 the new caps (I17, D4); ADR7 no
`debug_getRaw*` before M1 ends (X5); ADR11 the stack selection (section 1).
Open, after M1, waited for by nothing: ADR12, ADR14.

| ID | Subject | Decision |
|---|---|---|
| ADR8 | lakme WebSocket side | After M1, a lakme change; until then integration tests carry subscriptions and filters over WebSocket |
| ADR9 | The `GPO` section (`MaxGasPrice`, `MinGasPrice`, `DefaultCertainty`) | Feeds the legacy oracle in legacy mode and the module's oracle through `oracle.Config` in module mode. `MinGasPrice` floors the suggestion (`Oracle.suggestTip`). The loader rejects unknown keys, so the section stays |
| ADR10 | Coverage floor | 80% line coverage per module package, raised, never lowered; `rpc/core`, `rpc/tests` and generated files excluded |
| ADR12 | Compression of small responses | Open. The gzip writer's reset is a large share of a small response's CPU, in the reference too (profile at step 2); M1 compresses as the reference does (F1) |
| ADR13 | Test double of `Core` | `FakeCore` only (TS1). The repository generates a mock per interface; not for `Core`: strict expectations describe call sequences that H2 changes, background loops call `Core` at times a test does not control, and a failure row is one `FakeCore` mode where a mock needs an expectation per method. Mocks stay for one-collaborator tests (a logger) |
| ADR14 | A bound on concurrent requests | Open. The dispatcher counts `rpc_inflight` and limits nothing, as the reference; a few concurrent `trace_filter` requests take every core. The bound would be a semaphore per method class at admission, answered at once when full and counted `rejected`; decide when `rpc_inflight` or CPU shows the need. The operator notes say so |
| ADR15 | Log queries | Reused, not rewritten: the log index and the query code on `main` stay as they are. `Core.Logs` runs `gossip/filters.Filter` in the adapter, so the query, its limits and its wording match the reference by construction (F6, V2); decoding the criteria and normalising the answer are module copies, tested against the legacy texts (section 4). The cost: the one exception to SW3, the limits of the reused code stay in `filters.Config`, and the removal of the legacy stack moves `filter.go` before deleting the rest (removal list, section 3) |
