# In-process RPC rewrite: requirements and plan

A new RPC module for `sonicd`, written from scratch, running in the consensus
process. Today's RPC layer is go-ethereum's server and handlers carried along
with the fork: an architecture Sonic does not own, methods reflected off
structs, a wide `Backend` interface into the node, and patches where it did not
fit. The rewrite replaces it with a module the team owns: one narrow interface
to the node, an own entry point and transports, fewer third-party packages, and
a test program that holds it to the release it replaces.

This document is the work to be done and the proof that it is done. It records
no progress: the branch, its tests and lakme say what exists. It can be executed
again from `main` by someone who has not seen the branch; every lesson a
previous execution paid for is a rule or a gate here.

**Baseline (revised 2026-10-12).**

- **Reference** is the newest `v2.2.x` tag, today `v2.2.4` (2.2.3 → 2.2.4 only
  changed bundle plans). `main` is never the reference.
- **Port base** is `main`, but `main` and `v2.2` have diverged. `v2.2` carries
  RPC fixes `main` lacks: #1177 block consistency, #1205 (`eth_getProof` key cap,
  log parameter limit on every filter path, `MaxTraceFilterRange`), #1207
  unknown block in tracing, #1199/#1200 call-object and simulate fixes,
  #1220/#1221 bundles (`git log main..v2.2.4 -- api gossip config`). `main`
  carries what `v2.2` lacks: `eth_sendRawTransactionSync` (#1008). **Step 0**
  merges the first set into `main` before any port work, so that every lakme
  difference is the port's, not the branches'.
- Wording and order rules in F2–F7 were derived on `main` before step 0; each
  is re-checked against the reference after it.
- Code pointers are `file:Symbol`, never line numbers (they drift).
- Sources: **code** the legacy handler on `main`, **new** decided here.
  Items marked *decide* are open decisions in section 9 (ADR ids; they do not
  collide with the requirement ids D1–D11).

## 1. Shape

| Principle | What it means in the module |
|---|---|
| One narrow interface | The node is reached only through `Core` (section 2), a Go interface with no setter but `SendTx`; a test enforces it |
| Stateless, block-addressed | Every request to the node names a block by number or hash; tags are resolved once per request in the module and never reach the node |
| Archive-only reads | State comes from the archive at the named block, never from live state (as `GetBlockStateDb` does today); the pending nonce from the pool is the one exception |
| One `latest` | The module defines `latest` as the newest archived block it saw on the head subscription, for everything it serves, the block number included |
| Own entry point | A dispatcher is the only place that starts request goroutines; it decodes, applies the deadline, recovers panics and counts every call on every transport; methods exist by registration, never by reflection |
| Own dependencies | go-ethereum stays for the data model, the EVM and the tracers; its `rpc`, `node` and `event` packages leave the module (section 3) |
| Monitored by design | Prometheus `client_golang`, per-method accounting, bounded cardinality, alerts named up front |
| Smaller surface | Wallet, signing and consensus-internal namespaces are not ported (X1, X2); filters and the gas price oracle move out of `gossip` (G1–G3) |
| Test-first | A step is done when lakme, the handler tests and the integration tests say so; every difference from the reference is justified in writing (section 6) |

### Layout

| Package | Holds | Imports |
|---|---|---|
| `rpc/core` | `Core` and its value types; implemented by `gossip/rpc_core.go` | evmcore, opera, inter/state |
| `rpc/dispatch` | `Dispatcher` (method table, deadline, recover, `Group`), `Args`, `BlockRef`, `Chain` (head view, selector resolution), `Config`, `Notifier` | core |
| `rpc/exec` | what execution methods share: `CallArgs`, overrides, EVM environment (`New`, `Run`), `ChainConfig` derived from rules, `RevertError` | dispatch, core, evmcore, opera |
| `rpc/oracle` | gas price oracle and its sampler | core |
| `rpc/eth`, `filters`, `trace`, `debug`, `txpool`, `net`, `web3`, `sonic` | one namespace each; `Register(d, chain[, ns])`; `eth` and `filters` register twice, as `eth_` and `ftm_` | dispatch, exec, oracle, core |
| `rpc/transport` | HTTP, WebSocket, unix-socket transports over one `Front` that routes ported methods to the dispatcher and forwards the rest to the node's legacy handler while the port is under way | dispatch |
| `rpc/tests` | `FakeCore`, `NewCarmenCore` (real in-memory Carmen), dispatcher helpers; successor of `api/rpctest` | core, dispatch |
| `rpc` | `Module`: wiring of namespaces, head loop, start and stop | all of the above |

**One import rule, one test.** Nothing under `rpc/` imports `gossip`,
`api/*` or `go-ethereum/rpc`, with one exception: `rpc/sonic` imports the leaf
package `gossip/blockproc/bundle` (plan encoding, plan hash and envelope gas
formula are consensus-critical; copying them would fork the hashing).
`gossip` imports `rpc/core` only. The composition root `cmd/sonicd/app` builds
the adapter and the `Module`. Test files may import go-ethereum's `rpc` as the
oracle for envelope compatibility (Q5).

## 2. The system interface

`Core` is implemented in `gossip` by one adapter over `Store`, `EvmStateReader`,
the tx pool and `ServiceFeed`. Every module test runs against the fakes in
`rpc/tests`.

```go
// Core is everything the RPC module may ask the node. Every call about chain
// or state names a block by number or hash; nothing here mutates except SendTx.
// It is composed of role interfaces so a namespace depends only on the roles
// it uses and fakes stay small.
type Core interface {
	ChainReader   // ChainInfo, Header, Block, BlockNumberByHash, Receipts, Rules, TxByHash, BundleInfo
	HeadSource    // Head, SubscribeHead
	StateProvider // State
	Pool          // SendTx, PoolNonce, PoolTransaction(s), PoolContent(From), PoolStats, SubscribeTxs
	LogIndex      // FindLogs
	NodeInfo      // Progress, OracleInputs, PendingHeads
}

type ChainReader interface {
	ChainInfo() ChainInfo
	Header(number uint64) (*evmcore.EvmHeader, error) // no transactions decoded
	Block(number uint64) (*evmcore.EvmBlock, error)   // ErrNotFound above the archive head
	BlockNumberByHash(hash common.Hash) (uint64, bool)
	Receipts(number uint64) (types.Receipts, error)      // fails if TxIndex is off
	Rules(number uint64) (*opera.Rules, error)           // head+1: the open epoch's rules
	TxByHash(hash common.Hash) (tx *types.Transaction, block, index uint64, err error) // ErrNotFound if unknown; fails if TxIndex is off
	BundleInfo(executionPlanHash common.Hash) (BundleInfo, bool)
}

type HeadSource interface {
	Head() (*evmcore.EvmBlock, error)                 // newest block whose state is archived
	SubscribeHead(ch chan<- HeadEvent) Subscription
}

// State: the archive at one block in Carmen's non-committable StateDB
// (inter/state.StateDB: GetProof, Copy, EndTransaction, Release).
// Journaled per request, released exactly once, never committed.
type StateProvider interface {
	State(number uint64) (state.StateDB, error)
}

// Pool: the only calls whose answer depends on "now".
type Pool interface {
	SendTx(ctx context.Context, tx *types.Transaction) error
	PoolNonce(addr common.Address) uint64
	PoolTransaction(hash common.Hash) *types.Transaction
	PoolTransactions() (types.Transactions, error)
	PoolContent() (pending, queued map[common.Address]types.Transactions)
	PoolContentFrom(addr common.Address) (pending, queued types.Transactions)
	PoolStats() (pending, queued int)
	SubscribeTxs(ch chan<- evmcore.NewTxsNotify) Subscription
}

// LogIndex: the node's address/topic index over archived blocks (topicsdb).
// Serves the indexed path of eth_getLogs until the module owns an index.
type LogIndex interface {
	FindLogs(ctx context.Context, from, to uint64, pattern [][]common.Hash, limit uint) ([]*types.Log, error)
}

// NodeInfo: liveness and the gas price oracle's inputs.
type NodeInfo interface {
	Progress() Progress
	OracleInputs() OracleInputs // PendingRules, TotalGasPowerLeft, MinTip
	PendingHeads(max int) []PendingHead
}
```

Value types (`ChainInfo{ChainID, NetworkID, GenesisID, UpgradeHeights}`,
`Progress`, `OracleInputs`, `PendingHead`, `BundleInfo{BlockNumber, Offset,
Count}`, `HeadEvent{Block, Logs}`) live in `rpc/core`; `Subscription` is a
two-method interface (`Err()`, `Unsubscribe()`) satisfied by go-ethereum's
`event.Subscription`.

Rules the adapter follows, each a design decision:

- It starts no goroutine on the module's behalf. The head snapshot is the
  synchronous `Head()`, not a first event pushed by the adapter (an unmanaged
  pusher panics on a stopped feed and takes the node down). The module keeps its
  head monotonic regardless of arrival order.
- A stopped feed answers `SubscribeHead`/`SubscribeTxs` with an already-ended
  subscription, never the nil the feed scope hands out.
- `SubscribeTxs` carries `evmcore.NewTxsNotify`, what the pool feed emits.
  `PoolNonce` has no error because the pool's `Nonce` has none.
- `Header(n)` exists because most reads (state getters, `eth_call`, estimates)
  need a header and not the block's transactions; `EvmStateReader.Header`
  already skips them. Serving them through `Block` would cost the saving
  section 7.2 measures.
- A block above the committed head is `ErrNotFound`, as #1177 makes the legacy
  reader do.
- The launcher stops the module before the node closes, so the module leaves the
  feed before the feed leaves.
- Every guard the legacy backend takes from the node's configuration is the
  adapter's. With `TxIndex` off, `Receipts` fails with `transactions index is
  disabled (enable TxIndex and re-process the DAGs)` and `TxByHash` with the
  same text ending `the DAG)` (`EthAPIBackend.GetReceiptsByNumber`,
  `GetTransaction`); a method propagates the error where its legacy
  counterpart did. A `bool` result cannot carry such a failure, which is why
  `TxByHash` returns an error. Before writing the adapter, list every
  `svc.config` read in `gossip/ethapi_backend.go`; lakme runs the default
  configuration and sees none of them, so each one has an adapter test.

Derived by the module, not asked for: `MinGasPrice` and `MaxGasLimit`, the EVM
and the chain config (from `Rules(n)`, the header and `opera.GetVmConfig`), the
head (from `SubscribeHead`).

**The current rules are the next block's.** Legacy reads the open epoch's rules
through `store.GetRules()` (`EvmStateReader.CurrentMaxGasLimit`, the oracle's
sampler, `eth_getRules` for tags). The module's equivalent is
`Rules(head+1)`, never `Rules(head)`: the head may seal an epoch, and the
header's epoch is then the old one. `MaxGasLimit` (the `eth_estimateGas` upper
bound and the Osaka cap), the sampler and `eth_getRules` all follow this rule.
When one site is found reading the wrong epoch, every other site of the shape is
checked in the same change (7.1).

Not on the interface, by decision: `GetEVM`, `AccountManager`, dag and abft,
`CalcBlockExtApi`, `ExtRPCEnabled`, the `RPC*` limit getters (RPC
configuration, not core facts), `CurrentBlock`, `test_*` (stays on the legacy
`testapi`).

`Core` grows only when a method proves it cannot be served from what is there,
and never gains a call whose answer depends on "now" unless it is the pool or
the node's liveness. A port that appears to need a new call has more often than
not misread the legacy handler; it stops and records the case in section 9
before adding anything.

**The head subscription never blocks the feed.** `event.Feed.Send` waits for
every subscriber. The module's head loop only hands events to bounded
per-consumer queues (WebSocket subscriptions, filters, the served-head
variable); a consumer that falls behind is dropped, not waited for (I15).

**One `latest`.** Every tag resolves once per request to the head the module last
received, and that number serves state, blocks, receipts, logs and
`eth_blockNumber` alike. Legacy answers `eth_blockNumber` from the committed
head (`EthAPIBackend.BlockByNumber`) and state from the archive head
(`StateAndBlockByNumberOrHash`), so a client can see a number it cannot yet
query; #1177 hides unpublished blocks and transactions but keeps the two heads.
lakme reports the difference when the archive lags; it is annotated as a fix
(ADR4).

## 3. Runtime: the dispatcher

The module owns its entry point. One `Dispatcher` receives every decoded request
and is the only place that starts a goroutine for request work. Handlers are
plain functions in a table:

```go
type Handler func(ctx context.Context, args Args) (any, error)
// registered as: d.Add("eth_getBalance", a.getBalance)
```

Methods exist only by registration, by name. Nothing is reflected off a struct,
so no lifecycle or helper method can become callable (S1).

`Dispatcher.Serve(ctx, req)`, in order: look the method up (unknown → -32601),
decode and **count** params (failure → -32602 naming the parameter; never
reaches the handler), attach the deadline, run the handler under `recover`
(panic → -32603 + stack in the log + counter), encode and count the response.
The handler's own checks follow the legacy order and wording (F2–F5). Goroutines
for parallel work come from `Dispatcher.Group(ctx)`, an `errgroup` whose children
run under the same recover. Long-lived loops (head, filter expiry, oracle
sampler) are started by `Start`, restart on panic with backoff, stop on `Stop`.

- **A restart takes everything again.** The defers of the panicking run release
  what the loop owns. A subscription taken outside the loop function is dead
  when the loop restarts: the loop keeps running and counts as recovered, but
  receives nothing. So the loop function subscribes (and the head loop reads
  `Head()`) itself on each run. A test panics each loop and checks that it still
  does its job afterwards; a restart counter alone proves nothing (P3).
- **Admission is atomic towards `Stop`.** The check that refuses new work once
  `Stop` began and the count `Stop` waits on (requests, subscriptions, `Go`,
  `Loop`) are taken under one lock. Checking an atomic flag and then calling
  `WaitGroup.Add` lets a request in after `Wait` returned: it runs on a
  cancelled context, after `Stop`, and `-race` reports the `Add` (P7).

Stream connections (WebSocket, IPC) put a `Notifier` on every request's context;
HTTP requests have none and answer `eth_subscribe` with -32601 `notifications
not supported`. The contract:

- `eth_subscribe` installs its feed **synchronously, in the handler, before the
  id is returned**. (#1162 fixed this in legacy; the port must not regress it.)
- Producers run under `Notifier.Go`, the connection's recover group; they end
  with the connection and a panic in one is an error on the connection.
- Closing a connection **cancels first, then waits** for its producers
  (`defer cancel()` registered before `defer group.Wait()` runs in the wrong
  order and deadlocks).
- Subscriptions ship in the same step as the WebSocket and IPC transports: a
  subscription forwarded to the legacy handler has no connection to notify.

| Transport | Features | Notes |
|---|---|---|
| HTTP | single and batch requests, body limit, CORS, vhosts, path prefix, server timeouts, `/health` | no subscriptions, as in geth |
| WebSocket | as HTTP plus `eth_subscribe`/`eth_unsubscribe`; origin check; per-connection subscription cap; bounded notification buffer, disconnect on overflow | `gorilla/websocket` is already in the module graph |
| IPC | as WebSocket, newline-delimited JSON on `<datadir>/sonic.ipc`, mode 0600 | `sonictool attach` and the harness dial it; `test_*` and `admin_*` are served here only |

**The HTTP write timeout is answered, not suffered.** geth's server
(`handler.handleNonBatchCall`, `batchCallBuffer` in `rpc/handler.go`) starts a
timer at the write timeout minus 100ms. When it fires, the timer cancels the
method's context and writes -32002 `request timed out` itself, flushed with a
`Content-Length`, whether or not the method returns. In a batch, every call
without an answer gets the error, the running one included, and the answers so
far go out with them. A notification is answered too. The late answer of the
method is dropped. The module does the same:

- The timeout reaches the endpoint as a value, not as a context deadline: a
  deadline cancels the method at the same instant, so the method's own error
  races the timer and wins half of the time.
- Exactly one write per request (a `sync.Once` for a call, a locked buffer for a
  batch), and the request path synchronises with the timer even when it has no
  answer of its own. Otherwise net/http finishes the response while the timer
  goroutine still writes; geth's own server has that race on a timed-out
  notification, so the differential test leaves that one case out.
- The methods that ignore cancellation (`eth_call` and `eth_estimateGas` on
  Tosca's interpreter, section 8) are exactly the ones that reach it, so the
  differential test (Q5) includes a method slower than the write timeout: a
  single call, a batch with fast and slow calls, and a notification.

**Operator surface kept as it is**: `--http.{addr,port,corsdomain,vhosts,api,
rpcprefix}`, `--ws.{addr,port,origins,api,rpcprefix}`, `--ipcpath`,
`--ipcdisable`, HTTP server timeouts (#1064), batch limits, `--mode validator`.
The node hands its listeners to the module at start (`takeOverRPC` in
`cmd/sonicd/app`, new): the node's own servers for a taken-over transport stay
off and the module listens on the configured endpoint.

- **WebSocket shares the HTTP server when the ports match**, whatever the
  hosts, and the shared server then listens on the WebSocket host (geth's
  `Node.wsServerForPort`, then `setListenAddr(WSHost, …)`). "The same endpoint"
  means the same port. Deciding by host and port binds the same port twice,
  and the node does not start.
- **Taking over rewrites the node's configuration**, so everything the node
  reads from it changes too. Before step 1, list every reader of `HTTPHost`,
  `WSHost` and `IPCPath`, and of what derives from them, and keep its
  behaviour. `Config().ExtRPCEnabled()` is false once both hosts are empty, and
  `unlockAccounts` then unlocks `--unlock` accounts with HTTP open, where the
  reference refuses with `account unlock with HTTP access is forbidden`.
  `HTTPEndpoint()` and the dump-config path report the endpoints too. The
  module, not the node configuration, answers whether external RPC is served.
- **In-process clients are clients.** `stack.Attach()` reaches the node's own
  handler, and that handler stops serving the ported namespaces at step 6.
  Before the switch, list every non-test `Attach()` caller and what it calls.
  Hardware-wallet self-derivation (`startNode`, `Wallet.SelfDerive`) reads
  `eth_getBalance` and `eth_getTransactionCount`. Such a caller goes through
  the module's dispatcher (an `ethereum.ChainStateReader` over `Serve`).
  Only the X3 forwarding stays on `Attach()` (O5).

**Codec (ADR2, decided).** An own JSON-RPC 2.0 layer on `net/http`,
`encoding/json` and gorilla, because the dispatcher must see every request
before a goroutine exists and `rpc.Server` spawns first and reflects methods off
structs. This is the largest and most security-sensitive part of the work
(envelope, batches, framing, origin and host checks, size limits). Mitigations,
all gates: geth's `rpc.Server` and `rpc.Client` are the compatibility oracle in
test files (Q5, D11); a fuzz target on the codec; the transports are the one
part of step 1 and step 4 with their own review. `Front` is scaffolding for the
methods not ported yet; at step 6 only its IPC routing of X3 remains. The `eth_stop` hole alone does not justify a rewrite (it is a
one-line fix in legacy, section 6); deadline/recover/metrics uniformity and the
measured saving do.

### go-ethereum in the module

| Keep | Why |
|---|---|
| `common`, `common/hexutil`, `common/math`, `core/types`, `rlp`, `crypto`, `crypto/kzg4844` | the data model |
| `core/vm`, `core/tracing`, `eth/tracers` (+ `logger`, `native`) | the EVM and the tracers behind `debug_*` |
| `params`, `accounts/abi`, `log` | `ChainConfig`; revert-reason decoding; the client's logger |

| Drop from the module | Replaced by |
|---|---|
| `rpc` | own `BlockNumber`/`BlockNumberOrHash`, subscription ids, server (ADR2) |
| `node` | own HTTP/WS/IPC listeners and config |
| `event` | channels owned by the dispatcher; `Subscription` is a two-method interface |
| `accounts` (wallets, `scwallet`, the manager) | the module never touches them (X1) |
| `ethdb`, `core/rawdb`, `trie`, `triedb`, `consensus*` | audit what pulls them in today; expected to go with the code they serve |

This is the *module's* import budget. The node keeps `accounts/keystore`
(`valkeystore`, `integration/assembly.go`, `sonictool account`) and the
client-side `rpc` (`sonictool attach`, `tests/`); removing them is a separate
change. A test enumerates the module's imports and fails on anything outside the
keep list.

## 4. Milestones

Two milestones, each a question the work answers and a table of how we know.

### M1. Proof of concept (steps 0 to 5)

Can the module be built on the interface of section 2 without giving anything
up? `Front` may still forward what is not ported; nothing is shippable and none
of the guarantees of M2 are claimed.

| Question | How we know | Pass |
|---|---|---|
| Is the interface sufficient? | Every method section 6 marks *port* is registered (a test diffs the registered set against the inventory); `Core` has not grown beyond section 2 during steps 1 to 5; the import test holds | No handler needed a call outside `Core`; additions are in section 2 with their reason |
| Is it feature equivalent? | lakme against the reference, every family (7.1); the `tests/` RPC suite over HTTP and WebSocket | Green, or annotated in `defects.json` with a reason section 6 accepts |
| Is performance acceptable? | The 7.2 protocol against the reference | No row slower at p < 0.05; geomean at or below |

### M2. MVP: the first release (step 6)

Same features and at least the performance of the reference, plus the guarantees
M1 does not make. The forwarding of unported methods goes and the module is the only RPC server
for everything but X3.

| Guarantee | How we know |
|---|---|
| Feature equivalent | lakme green or annotated (T1); execution-apis fixtures (T3); every geth method served or in X with a reason (Q5); Sonic extensions served and tested (Q6) |
| Performance | The 7.2 protocol: no row slower at p < 0.05, geomean at or below (T2) |
| Completion on shutdown | Q1/D10: no goroutine of the module alive after `Stop` under load |
| Cancellation | Q2: every registered method returns within the bound when cancelled |
| Nil safety | Q3: the injection matrix runs with `rpc_panics_total` at zero |
| Coverage | Q4: the CI floor holds; every registered method has a handler test |
| Standards | Q5: JSON-RPC 2.0 envelope and geth's method set match, as oracle test, fixtures and lakme observe |
| Operable | MON1–MON6; limits preserved and tested (O1); deadlines on every request (O2); `--mode validator` closes public listeners (O3); flags and TOML keys compatible, each with its effect (O4); the node's own uses of its RPC configuration and handler kept (O5); transports secure (D11); changelog and operator notes written |
| Reviewed | T7: an independent review against the reference, with 7.6 as the checklist |

### Steps

Each step ships behind nothing: the module replaces the legacy handlers for the
namespaces it has ported, `Front` forwards the rest, and lakme compares the
branch to the reference. A step is done when the lakme run is green or
annotated, its handler tests exist and the integration tests for its methods
pass. The namespace packages are independent, so steps 2 to 5 can be worked in
parallel after step 1; none may add to `Core`.

0. **Baseline** (M1). Merge the `v2.2` RPC fixes into `main` (see Baseline).
   Run lakme `main` against the reference; annotate or fix every difference.
   This is the green start every later run is compared to. Gate: lakme green on
   unmodified `main`.
1. **Skeleton** (M1). `Core` and its gossip adapter, dispatcher (recover,
   deadlines, method table), **HTTP** transport and `Front`, metrics. Ports:
   `eth_chainId`, `eth_blockNumber`, `eth_syncing`, block and header reads, tx
   lookups, `eth_getRules`/`currentEpoch`/`config`, `net_version`, `web3_*`.
   The module takes over the HTTP listener, after listing what the node reads
   of the configuration it rewrites (section 3). When the HTTP and WebSocket
   ports are the same, the WebSocket upgrade is reverse-proxied to the node's
   own server on a private endpoint until step 4 (ADR12). Gates: I1–I4, I13,
   I15, D1–D4, D9, D11, P1–P5, P7, S1, MON1–MON5, O5 (unlock check).
2. **Oracle, state and execution** (M1). The gas price oracle in the module on
   `Core.OracleInputs`, `PendingHeads` and `Rules` (I10; it runs next to the
   legacy oracle, which stays until step 6), with all three `GPO` keys in
   effect (ADR9): `eth_gasPrice`, `eth_maxPriorityFeePerGas`,
   `eth_feeHistory`, `eth_blobBaseFee`. The oracle comes first because call
   defaulting (`setDefaults`) asks it for the tip. Then `eth_get{Balance,Code,
   StorageAt,TransactionCount,Account,Proof}`, `eth_call`, `eth_estimateGas`,
   `eth_createAccessList`, `eth_simulateV1`, `sonic_estimateGasForTransactions`.
   Gates: I5–I8, I10, I12, V1–V6, V10, V11, F2–F5, O1; lakme state, call and oracle
   families.
3. **Transactions and pool** (M1). `eth_sendRawTransaction(Sync)`,
   `eth_pendingTransactions`, `txpool_*`, `sonic_submitBundle`,
   `sonic_prepareBundle`, `sonic_getBundleInfo`. Gates: V7; `rejected_tx_test.go`,
   `tests/rpcs/send_raw_transaction_sync_test.go`, bundle tests.
4. **Events** (M1). WebSocket and IPC transports (the module takes both over and
   the proxy goes), head and tx subscriptions, filters, `eth_getLogs`; one step,
   because a subscription cannot be forwarded. Gates: I9, I11, I14, I16, D4–D6,
   V8, F6, F7, P8; lakme filter family; subscription tests over WS and IPC.
5. **Tracing** (M1). `debug_*`, `trace_*`, `debug_getRaw*` (ADR7). Read
   `stateAtTransaction` before deciding where the EIP-2935 write goes (section 6).
   Gates: V6, V11, F2, F3; lakme trace families.
6. **Switch** (M2). The forwarding of unported methods goes, with the last path
   to `eth_stop` (S1); `Front` keeps only the IPC routing of X3 and the p2p
   facts; the module is the only RPC server; X3 namespaces stay on IPC
   in the minimal `api/legacy` package until their removal. Deleted here: the
   legacy oracle and its plumbing (G2), `gossip/filters` (G1), the unreachable
   parts of `api/*` (G3). Before the forwarding goes, every in-process client
   of the node's handler is moved to the module (O5). Gates: Q1–Q6, D10,
   O1–O5, P6–P8, MON1–MON6, import test, lakme and benchmark gate (T2) against
   the reference, and a review of the whole branch against the reference by
   someone who did not write it, with section 7.6 as the checklist (T7). No
   escape hatch (ADR11).

## 5. Requirements

Columns: **ID**, requirement, **Verified by** (section 7 tools:
L lakme, E execution-apis fixtures, R rpctest, I integration tests, U unit test
in the module, C CI lint or import test). I, D, P, V, F, S, X and MON gate
M1 and M2; Q, O and G gate M2 alone.

### I. Interface and state

| ID | Requirement | Verified by |
|---|---|---|
| I1 | Import rules of section 1: `rpc/` never imports `gossip` (but `rpc/sonic` → `gossip/blockproc/bundle`), `api/*`, `go-ethereum/rpc`; every core access is through `Core` | C |
| I2 | Every `Core` call about chain or state carries a `uint64` number or a hash; the module resolves `latest`, `pending`, `safe`, `finalized`, `earliest` itself, once per request | R (fake Core rejects tags), U |
| I3 | `latest` = newest archived block last seen on `SubscribeHead`; the same number serves every part of one request and `eth_blockNumber` | L (annotated), I (`block_in_archive_test.go`) |
| I4 | `earliest` resolves to the pruning cutoff (`HistoryPruningCutoff`, today 0) | R |
| I5 | `pending` state = archive head state; `pending` nonce = `Core.PoolNonce` | R, L |
| I6 | State views come from the archive only | R (fake Core has no live state), U |
| I7 | Every `StateDB` obtained is released exactly once, by the request that opened it, via `defer` before any code that may panic | R (panic injection), U |
| I8 | Methods that chain runs (`eth_call` with `preArgs`, `eth_simulateV1`, `sonic_estimateGasForTransactions`, `trace_callMany`, block replays) reuse one StateDB with `EndTransaction` between runs and `Copy()` for branches | R, L |
| I9 | The head subscription is the module's only source of head, new blocks, new logs and new pending transactions; on its loss the module keeps serving the last head and logs the lag | U, I |
| I10 | The oracle runs in the module on `Core.OracleInputs` and `Core.PendingHeads`, sampling at 1s/5s as today; call defaulting reads its tip. The sampler reads what the legacy sampler read and no more: `PendingHeads` and the open epoch's minimum gas price (`Rules(latest+1)`). `OracleInputs` walks every validator's last event for `TotalGasPowerLeft`, so it is read for a suggestion only, never per tick | L (tolerance rows), I (`gas_price_suggestion_test.go`), U (a tick does not call `OracleInputs`) |
| I11 | `eth_getLogs` and filter history: indexed queries through `Core.FindLogs`; queries with no address and no topic, and `blockHash` queries, scan blocks through `Core.Header`/`Receipts`; both keep the limits of section 8; each log gets its tx index from `TxByHash` and its timestamp from the header | L, I (`rpc_get_logs_test.go`, `indexed_logs_test.go`) |
| I12 | The EVM is built in the module from `Rules(n)`, the derived chain config and the block header; the chain config is the *queried* block's, never the head's | I, R |
| I13 | Header-only reads (state getters, `eth_call`, estimates, `eth_getLogs` timestamps) use `Core.Header`, never `Core.Block` | U (fake counts `Block` calls), benchmark |
| I14 | The module advances its served head only after that block's logs are enqueued for subscribers, so `eth_getLogs` at `latest`, `eth_blockNumber` and log subscriptions never disagree | U, I (`log_subscription_test.go`) |
| I15 | The head loop never blocks on a consumer: bounded per-consumer queues, a lagging consumer is dropped | U (stalled consumer, head still advances) |
| I16 | Work that depends only on the event is done once per event, not once per consumer: the `newHeads` notification (the block's receipts and the encoded header) is built once per block and shared by every subscription. The reference did this in its filter system; a handler per subscription that reads `Receipts` multiplies block loads by the subscriber count. The benchmarks of 7.2 do not cover subscriptions, so this is a rule | U (N subscribers, one `Receipts` call per block) |

### D. Dispatcher and transports

| ID | Requirement | Verified by |
|---|---|---|
| D1 | All request goroutines are created by the dispatcher or its `Group`; a lint test fails on a bare `go` statement elsewhere in the module | C |
| D2 | Lookup, decode, validation, deadline, recover, metrics and encoding happen in the dispatcher, in that order, for every method on every transport | U (table-driven per stage), R |
| D3 | HTTP: single and batch requests, body cap, batch item and response caps, CORS/vhosts/path prefix as configured today, `/health` reporting head and lag; at the write timeout minus 100ms a running call is answered -32002 `request timed out` and every unanswered call of a batch with it, as geth does (section 3) | I, L, U (differential test with a slow method) |
| D4 | WebSocket: requests, batches, subscriptions (`newHeads`, `logs`, `newPendingTransactions`), subscription cap per connection, bounded buffer per subscription, disconnect of a client that does not read | I (`log_subscription_test.go`, `pending_transactions_subscription_test.go`), U |
| D5 | IPC unix socket at `<datadir>/sonic.ipc`, mode 0600, same features as WebSocket; the only transport that serves `admin_*`, `test_*` and the legacy namespaces | I (harness client), U |
| D6 | Filters installed over HTTP expire after 5 minutes without a poll; ids are unguessable and per node | L, U |
| D7 | The `eth` namespace is mirrored as `ftm` | L |
| D8 | Namespaces are enabled per transport by `--http.api`, `--ws.api`; defaults as today | I |
| D9 | Unknown method → -32601; malformed JSON → -32700; a parameter the dispatcher cannot decode or count → -32602 naming it; internal → -32603; a handler's own check keeps the code and words of its legacy counterpart (-32000 for a plain error, revert data on `eth_call`, the simulate codes) | L (error rows), E, U |
| D10 | `Stop` drains in-flight requests up to the deadline, closes listeners and subscriptions, and returns with no goroutine of the module alive | U (goroutine count), I (`node_restart_test.go`) |
| D11 | Transport security matches geth's: Host/vhost check, CORS, WebSocket origin check, body and message caps, read/write/idle timeouts. Tested with a disallowed Host (DNS rebinding), a disallowed origin, an oversize body, a stalled client | I, U (geth `rpc.Client` as oracle) |

### P. Panics, goroutines and locks

| ID | Requirement | Verified by |
|---|---|---|
| P1 | A panic in any handler or child is recovered by the dispatcher, answered -32603, logged once with the stack, counted in `rpc_panics_total{method}`; the process never exits because of RPC work | R (panic injection on every `Core` method), U |
| P2 | Child goroutines run in a `Group`: a panicking child becomes an error, cancels its siblings, and the parent waits for all children | U (goroutine count), R |
| P3 | Long-lived loops restart on panic with backoff and a counter; a loop that panics repeatedly is reported, never silently dead. A restarted loop takes its subscriptions again: what it acquired outside its function was released by the panic (section 3) | U (panic injected into each loop; afterwards the loop still does its job: the head advances, the pool feed delivers) |
| P4 | Handlers hold no locks. Shared structures (filter registry, subscription registry, oracle cache) expose small methods that lock and `defer` unlock in the same function and never call out to `Core`, a handler or the EVM while locked | review rule, U (blocked `Core` does not stall other requests) |
| P5 | Resources with a lifetime (state views, subscriptions, tracer buffers) are released by a `defer` on the line after acquisition | R (panic injection; budget back at zero) |
| P6 | Module tests run under `-race`; the RPC integration tests run under `-race` nightly | C |
| P7 | Admission is atomic towards `Stop`: the stopping check and the count `Stop` waits on are taken under one lock, for requests, subscriptions, `Go` and `Loop`; no work starts after `Stop` returned | U (stress: requests and `Go` racing `Stop`, under `-race`) |
| P8 | What the node hands out is shared and read-only: the blocks and logs of a `HeadEvent` reach every consumer, and receipts, headers and logs come from the node's caches. Normalising for an answer (nil topics to `[]`, a log's tx index) works on a copy. The reference's `returnLogs` writes topics in place; the port does not inherit it | U (a shared log is unchanged after `eth_getFilterChanges`), P6 |

### Q. Release guarantees (M2)

| ID | Requirement | Verified by |
|---|---|---|
| Q1 | Shutdown completes: `Stop` cancels in-flight requests at their deadline, waits for them, closes listeners, subscriptions and loops; the node's shutdown under RPC load finishes within the node's own timeout | U (goroutines after `Stop` under load = before `Start`), I (node stopped while a load client runs) |
| Q2 | Cancellation is observed on every path: a cancelled request returns within one `Core` call or one interpreter poll; every loop checks the context; no handler waits on a channel without selecting on the context | U (every method through `FakeCore` latency injection, cancelled) |
| Q3 | Nil safety: every method answers without panic when any `Core` method returns nil, not-found or an error, and for any `null` or absent optional argument; `rpc_panics_total` stays zero over the matrix | R (`FakeCore` failure modes × methods × argument shapes) |
| Q4 | Line coverage of every module package at or above the floor (ADR10), measured in CI; every registered method has a handler test (T4) | C |
| Q5 | Compliance with geth: every method in geth's OpenRPC specification for `eth`, `debug`, `txpool`, `net`, `web3` is served or listed in X with its reason (a test diffs the sets); the JSON-RPC 2.0 envelope (ids, batches, notifications, error codes, `null` results, timeouts) matches geth's `rpc.Server`, checked by a differential test that sends the same requests to both. The test includes the cases that only show under a configured limit: a method slower than the write timeout (call and batch), a batch over each cap | C, E, L, U |
| Q6 | Sonic extensions are served and listed by origin in section 6: `ftm` mirror, `sonic_*`, OpenEthereum `trace_*`, `eth_getRules`, `eth_currentEpoch`, `eth_config`, `eth_sendRawTransactionSync`; each has the handler tests and lakme rows geth methods have | L, I, R |

### V. Validation (every method, before the handler)

| ID | Requirement | Verified by |
|---|---|---|
| V1 | Block selectors: a known tag, a number in `[cutoff, head]`, or a known hash; anything else → -32602 (unknown height) or "block not found" where the code says so | L, E, R |
| V2 | Ranges: `from ≤ to`, both resolvable, span within the method's limit (`eth_getLogs`, `eth_feeHistory` count ≤ 1024 clamped, `trace_filter` ≤ `MaxTraceFilterRange`) | L, E, R |
| V3 | Quantities are canonical hex, fit their width; `uint256` fields bounded; gas ≤ `RPCGasCap`; prices and values non-negative | E, R |
| V4 | Addresses 20 bytes, hashes and topics 32; topic nesting depth ≤ 2; total topic and address parameters ≤ `LogQueryParameterLimit`, checked on every filter path (`eth_getLogs`, `eth_newFilter`, `logs` subscriptions) | L, R |
| V5 | Call arguments: `data` and `input` agree when both given; blob, access-list, authorization-list shapes; `state` and `stateDiff` mutually exclusive per account; block overrides sane; `preArgs` count bounded | R (rows of `transaction_args_test.go`), E, L |
| V6 | Tracer requests: tracer name from the allow-list, config decodable, `timeout` parses and ≤ `RPCEVMTimeout`, `trace_*` types from `{trace, vmTrace, stateDiff}` | L, R |
| V7 | Raw transactions: RLP decodes to a known type, chain id matches, EIP-155 protected unless `UnprotectedAllowed`, fee ≤ `RPCTxFeeCap`; the pool's verdict is returned verbatim; `sendRawTransactionSync` timeout ≤ `TxSyncMaxTimeout` | I, R |
| V8 | Filter and subscription ids exist and belong to the calling connection | L, U |
| V9 | Validation never allocates proportionally to a client-chosen number before the bound is checked (block count, storage keys, topics, batch items) | U |
| V10 | `eth_getProof`: at most 1024 storage keys, strict key decoding, both → -32602 before any state access | L, R |
| V11 | No value a client sets decides how far a lookup walks or how often a loop runs, beyond what a limit of section 8 bounds. A block override `number` changes which blocks BLOCKHASH may ask for, never what one BLOCKHASH costs: it is answered by one lookup by number (`Core.Header`), with the zero hash at or above the block of the call, never by walking the parent hashes back from that block (`evmcore.GetHashFn`), which a `number` above the head turns into one header read per block down to genesis. Applies to every block context built from a header of the chain (`exec.Env.BlockContext`): `eth_call`, `eth_estimateGas` (once per step of its search), `eth_createAccessList`, `debug_traceCall`, `trace_*`, `sonic_estimateGasForTransactions`. `eth_simulateV1` is the one walk left: its blocks are not in `Core`, its block number is that of the header it walks from, so it walks at most 256 headers | U (`Core.Header` calls counted under overrides above and below the head) |

### F. Fidelity to the reference

The module replaces a node clients have adapted to for years. Wherever a client
can observe a difference, the reference is right by definition and the module
matches it bit for bit; every difference lakme finds is justified in writing, as
a row of the deviations table (section 6) with its `defects.json` entry, or it is
a defect of the port and is fixed. A difference is never left unexplained and
never explained by `main`. F2–F7 are the classes a port produces when it reads
the legacy handler's intent instead of its code; each is a rule because one
execution paid for it.

| ID | Requirement | Verified by |
|---|---|---|
| F1 | Every observable answer — result, error code, wording, and which of several failing checks is reported — matches the reference unless a deviations row says otherwise | L |
| F2 | Checks run in the legacy order: block first, then state overrides, then the call object; a tracer's name, config and `timeout` are checked where the tracer is built, after its subject is found. A request naming a missing block and carrying a bad fee pair reports the block (`DoCall`, `AccessList`, `TraceCall`, `PublicTxTraceAPI.Call`, `DoEstimateGas`) | L, U |
| F3 | A missing block is worded as each legacy handler worded it: `header not found` (state getters, `eth_call`); `block does not exists` or `block with hash %s not found` (`eth_estimateGas`, via its gas cap); `block #%d not found` (`debug_*`); `cannot trace block %d got invalid block for tracing` (`trace_block`); the tracing wording of #1207. One uniform resolver error is not acceptable where lakme can see it | L, U (one assertion per wording) |
| F4 | A handler's own check (fee fields, `data` vs `input`, `state` vs `stateDiff`, reward percentiles) is a plain error with the legacy text, so it arrives as -32000; -32602 is only what the dispatcher cannot decode or count; an unknown subscription type is -32601 `no "x" subscription in eth namespace` | L, U |
| F5 | `err: %w (supplied gas %d)` wraps a failed message only where legacy went through `DoCall` (`eth_call`, `eth_estimateGas`, `eth_simulateV1`); `eth_createAccessList` reports the bare error under `failed to apply transaction from sender …` | L, U |
| F6 | `eth_getLogs` over a range ends at the first block it cannot read and returns what it has; range limits are checked on the requested bounds before the walk | L, U |
| F7 | `eth_newPendingTransactionFilter(true)` serves whole transactions with no block fields, each priced by its own gas price | L, U |
| F8 | Known bugs of the reference (section 6) are reproduced, not fixed, until a release fixes them; each has an issue and a branch with one failing test | L |

### S. Surface

| ID | Requirement | Verified by |
|---|---|---|
| S1 | A method is callable only because a line in a `Register` function names it. Legacy registers every exported method of its API structs and so serves `eth_stop`/`ftm_stop` from `PublicFilterAPI.Stop()` (exported for `gossip.Service`): per the code, one unauthenticated call stops the filter event loop and every later filter or subscription install blocks. The module never registers it and `Front` stops forwarding a namespace the module serves | U (-32601 on every transport), L |

### O. Limits inherited from the reference (preserved and tested)

| ID | Requirement | Verified by |
|---|---|---|
| O1 | Semantic caps keep their names, defaults and effect (section 8) | L (rows that hit each cap), I |
| O2 | `RPCTimeout` is applied by the dispatcher to every request context; the fork's `SetExecutionTimeLimit` is no longer used by the module | U, C |
| O3 | `--mode validator` keeps disabling public listeners | I |
| O4 | Every CLI flag and TOML key under `config` keeps its effect or is listed as deprecated (accepted, warns, no effect) for one release. The loader rejects unknown fields, so removal breaks configs: this covers `--unlock`, `--password`, `--allow-insecure-unlock` (X1) and the `GPO` section (ADR9). "Keeps its effect" is tested by changing the key and observing the change. A key that still parses but no longer reaches its consumer passes a parse test (`GPO.MinGasPrice` did); so does a node-configuration key nothing reads any more. Every `if cfg.X != nil` or `!= 0` mapping into the module's `Config` has a row that sets it | U (test enumerates flags and keys; one effect test per key the module consumes) |
| O5 | What the node derives from its own RPC configuration and handler keeps its effect after `takeOverRPC`: `--unlock` with HTTP or WebSocket open and no `--allow-insecure-unlock` is refused; hardware-wallet self-derivation reads the chain through the module; the reported endpoints are the module's. Every non-test `stack.Attach()` caller is listed with the methods it calls, and served (section 3) | I (`cmd/sonicd/app` exec test of the unlock refusal), U (chain reader over the dispatcher) |

### MON. Monitoring

| ID | Requirement | Verified by |
|---|---|---|
| MON1 | `client_golang` on the default registry, served by the node's metrics endpoint; no go-ethereum metrics registry in the module | C |
| MON2 | Per method: `rpc_requests_total{method,result}`, `rpc_duration_seconds{method}`, `rpc_response_bytes{method}`; per transport: `rpc_connections{transport}`, `rpc_inflight`; subscriptions: `rpc_subscriptions{kind}`, `rpc_subscription_events_total{kind,outcome}`; WebSocket and IPC counted like HTTP | U |
| MON3 | Head: `rpc_head_served`, `rpc_head_lag` (committed minus served) | U |
| MON4 | Faults: `rpc_panics_total{method}`, `rpc_loop_restarts_total{loop}` | R |
| MON5 | Cardinality bounded by construction: unknown method names map to `other`; never label by address, block, client or error text | U |
| MON6 | Alerts defined from day one: panics > 0, head lag over a block for a minute, p99 per method over its budget | operations |

### X. Marked for deletion

| ID | Requirement |
|---|---|
| X1 | Not ported: `personal_*`, `eth_accounts`, `eth_sign`, `eth_signTransaction`, `eth_sendTransaction`, `eth_fillTransaction`, `eth_resend`, `debug_chaindbProperty`, `debug_chaindbCompact`, `debug_setHead`; with them `AddrLocker` and the RPC use of the account manager. The CLI flags that feed them follow O4 |
| X2 | Not ported: `dag_*`, `abft_*`, `eth_getEpochBlock`, `eth_getEpochStats`; `eth_getRules`, `eth_currentEpoch`, `eth_config` are served from block rules, epoch and `ChainInfo` |
| X3 | Until removed, X1 and X2 methods are served by the legacy `api/legacy` (and `api/testapi`) registered on IPC only (ADR5); lakme rows for them are annotated; integration tests using `dag_*` switch to the IPC client |
| X4 | `CalcBlockExtApi`, `RPCBlockExt` and the gossip `PublicEthereumAPI` shims (`gossip/api.go`) are not carried over; `eth_chainId` has one implementation |
| X5 | `MinGasPrice`, `MaxGasLimit` are derived in the module, not exposed by `Core`; `MaxGasLimit` from `Rules(head+1)`, the open epoch (section 2) |

### G. Moves out of gossip (step 6)

| ID | Requirement |
|---|---|
| G1 | `gossip/filters` is rewritten in the module on `SubscribeHead`, `SubscribeTxs`, `FindLogs`; the package is deleted |
| G2 | The oracle half of `gossip/gasprice` (`Oracle`, reactive, constructive, tip cache) is deleted once the module's oracle serves the four price methods and `setDefaults` has no legacy caller left; `base_fee.go`, `gas_tip.go`, `gaspricelimits` stay for block processing and the emitter; `gossip/gpo_backend.go`, `Service.gpo`, `EvmStateReader.gpo`, `EthAPIBackend.SuggestGasTipCap` go; the `GPO` section follows ADR9 |
| G3 | `gossip/ethapi_backend.go` shrinks to the `Core` adapter; `api/ethapi`, `api/sonicapi`, `api/testapi` are deleted except what X3 keeps on IPC |
| G4 | `txtrace`, `evmcore`, `gossip/evmstore` and `gossip/blockproc/bundle` stay and are used as libraries |

### T. Testing (section 7)

| ID | Requirement | Verified by |
|---|---|---|
| T1 | lakme against the reference is green on every PR touching the module; every difference is fixed or an entry in `defects.json` with `introduced`/`fixed_in` bounds and a one-line reason | L |
| T2 | lakme benchmarks per release by the 7.2 protocol, compared with `benchstat`; no method's median regresses against the reference at p < 0.05 | L |
| T3 | The execution-apis fixtures run against the node on every PR for the methods they cover; failures are fixed or annotated | E |
| T4 | `rpc/tests` holds `FakeCore` (panic injection per method) and `NewCarmenCore`; every ported method has handler tests through the dispatcher | R |
| T5 | The `tests/` RPC integration tests pass in HTTP and WebSocket modes, plus the IPC path for `test_*` and `admin_*` | I |
| T6 | Module unit tests cover dispatcher stages, transports, filters and the oracle; run with `-race` | U |
| T7 | Before step 6 lands, the branch is reviewed against the reference by someone who did not write it, using the classes of 7.6 as the checklist. Every finding is fixed with a test that fails without the fix, or becomes a deviations row | review |

## 6. Method inventory

Decision: **port** (new handler), **local** (answered without `Core`),
**delete** (X1/X2; legacy on IPC until removed), **core** (node-local namespace,
IPC only). Coverage: **L** lakme, **E** execution-apis fixture, **I**
integration test. Every ported method also has an R test (T4).

### eth (mirrored as ftm)

| Method | Decision | Inherits from | Cov. | Notes |
|---|---|---|---|---|
| eth_chainId | local | `PublicBlockChainAPI.ChainId` | L E | one implementation (X4) |
| eth_blockNumber | local | `PublicBlockChainAPI.BlockNumber` | E | archive head (I3) |
| eth_syncing | port | `PublicEthereumAPI.Syncing` | E | `Core.Progress` |
| eth_gasPrice, eth_maxPriorityFeePerGas, eth_feeHistory, eth_blobBaseFee | port | `PublicEthereumAPI.*` | L E | oracle in module (I10); count clamp 1024 (V2) |
| eth_getBalance, eth_getTransactionCount, eth_getCode, eth_getStorageAt | port | `PublicBlockChainAPI.Get*`, `PublicTransactionPoolAPI.GetTransactionCount` | L E | pending nonce via pool (I5) |
| eth_getAccount | port | `GetAccount` | L | storage root from empty-storage flag today |
| eth_getProof | port | `GetProof` | L E | `State.GetProof`; V10 |
| eth_call | port | `Call`, `DoCall` | L E I | revert data in error; `preArgs` (I8); overrides (V5) |
| eth_estimateGas | port | `EstimateGas`, `DoEstimateGas` | L E I | binary search bounded by `RPCGasCap`, `MaxGasLimit` |
| eth_createAccessList | port | `CreateAccessList`, `AccessList` | L E | reproduces F8 bugs 1–2 |
| eth_simulateV1 | port | `simulate.go` | L E I | block count cap (ADR6) |
| eth_getBlockByNumber/Hash, eth_getHeaderByNumber/Hash | port | `PublicBlockChainAPI` | L E | |
| eth_getBlockReceipts | port | `GetBlockReceipts` | L E | |
| eth_getBlockTransactionCountBy{Number,Hash}, eth_getUncle*, eth_getTransactionByBlock{Number,Hash}AndIndex, eth_getRawTransactionByBlock{Number,Hash}AndIndex | port | `PublicBlockChainAPI`, `PublicTransactionPoolAPI` | L E | uncles are always empty |
| eth_getTransactionByHash, eth_getRawTransactionByHash, eth_getTransactionReceipt | port | `PublicTransactionPoolAPI` | L E | pool fallback for pending txs |
| eth_sendRawTransaction | port | `SendRawTransaction`, `SubmitTransaction` | E I | V7 |
| eth_sendRawTransactionSync | port | `SendRawTransactionSync` | I | main only; waits on the head subscription, not polling |
| eth_pendingTransactions | port | `PendingTransactions` | | pool view |
| eth_newFilter, eth_newBlockFilter, eth_newPendingTransactionFilter, eth_getFilterChanges, eth_getFilterLogs, eth_uninstallFilter | port | `gossip/filters/api.go` | L | D6, V8 |
| eth_getLogs | port | `PublicFilterAPI.GetLogs` | L E I | I11, V2, V4 |
| eth_subscribe / eth_unsubscribe (newHeads, logs, newPendingTransactions) | port | `PublicFilterAPI` | I | WS and IPC only |
| eth_getRules, eth_currentEpoch, eth_config | port | `GetRules`, `CurrentEpoch`, `config.go` | L | from block rules and `ChainInfo` (X2) |
| eth_etherbase, eth_coinbase, eth_hashrate | local | `gossip/api.go` | | constants |
| eth_stop | delete | `PublicFilterAPI.Stop` | | S1 |
| eth_accounts, eth_sign, eth_signTransaction, eth_sendTransaction, eth_fillTransaction, eth_resend | delete | `PublicAccountAPI`, `PublicTransactionPoolAPI` | | X1 |
| eth_getEpochBlock, eth_getEpochStats | delete | `PublicBlockChainAPI`, `dag_api.go` | L | X2; lakme annotated |

### txpool, sonic

| Method | Decision | Inherits from | Cov. |
|---|---|---|---|
| txpool_content, txpool_contentFrom, txpool_status, txpool_inspect | port | `PublicTxPoolAPI` | I |
| sonic_estimateGasForTransactions, sonic_prepareBundle | port | `api/sonicapi/estimate_gas.go`, `prepare_bundle.go` | I (`tests/rpcs/bundle_test.go`) |
| sonic_submitBundle, sonic_getBundleInfo | port | `submit_bundle.go`, `get_bundle_info.go` | I |

### debug, trace

| Method | Decision | Inherits from | Cov. | Notes |
|---|---|---|---|---|
| debug_getBlockRlp, debug_printBlock | port | `PublicDebugAPI` | L | printBlock is a spew dump (lakme annotated) |
| debug_traceTransaction | port | `PublicDebugAPI.TraceTransaction` | L I | replays the block up to the tx (I8); `MaxResponseSize`, `StructLogLimit` |
| debug_traceBlockByNumber/Hash | port | `TraceBlockByNumber`, `traceBlock` | L I | |
| debug_traceCall | port | `TraceCall` | L I | |
| trace_transaction, trace_get, trace_block | port | `tx_trace.go` | L | |
| trace_call, trace_callMany | port | `tx_trace.go` | L I | |
| trace_filter | port | `tx_trace.go` | L | parallel replay via `Group` (P2); range cap `MaxTraceFilterRange` (1000, `--rpc.tracefilterrange`) is inherited (V2); result order and error message vary today (lakme annotated) |
| debug_getRaw{Block,Header,Receipts,Transaction} | *decide* | not implemented | E | ADR7 |
| debug_chaindbProperty, debug_chaindbCompact, debug_setHead | delete | `PublicDebugAPI` | | X1 |
| debug_* profiling (geth built-ins) | core | node | | IPC |

### dag, abft, personal, test, net, admin, web3

| Method | Decision | Notes |
|---|---|---|
| dag_getEvent, dag_getEventPayload, dag_getHeads | delete | X2, X3; lakme draws `dag_getHeads`: annotated |
| abft_* (4 methods) | delete | X2, X3; lakme draws two: annotated |
| personal_* | delete | X1 |
| test_proposeTransactions, test_addTransactions | core | legacy `testapi` on IPC only |
| net_version | local | chain id |
| net_listening, net_peerCount | core | p2p facts, as today |
| admin_* | core | IPC only |
| web3_clientVersion, web3_sha3 | local | |

### Deviations the port introduces

Each row is a deliberate difference from the reference. Anything not here
matches it bit for bit (F1). A row lakme can see names its `defects.json`
entry; a new row is added with its entry in the same change.

| Method | Legacy | Here | Why |
|---|---|---|---|
| any, undecodable parameter | `-32602` `invalid argument N: …` | `-32602` `parameter N (name): …` | D9; wording differs, code does not |
| `eth_pendingTransactions` | only transactions of accounts the account manager holds, so `[]` | the pool's pending transactions | X1 removes the account manager |
| `eth_sendRawTransactionSync` | polls every 100 ms | waits on the head subscription | I9; the legacy message about an indexed but unarchived block cannot occur |
| `eth_getAccount`, `eth_getProof` | block selector required; propagate `state.Error()` | selector optional (`latest`); do not propagate | consistent with the other state getters |
| `debug_trace*` `timeout` | any duration accepted | `-32602` above `RPCEVMTimeout` or malformed; checked where the tracer is built | V6; lakme `rpc-module-tracer-timeout-limit` |
| `debug_traceCall` chain config | `ChainConfig(overridden block number)` | the queried block's rules | only differs if an override crosses an upgrade height |
| `trace_*` replay with a short receipt list | indexes the list unguarded, panics | `no receipt found for transaction <hash>` | latent panic (F8) |
| `debug_traceBlock*` message conversion error | ignored, then nil dereference | returned | same |
| `sonic_getBundleInfo` `block` | `rpc.BlockNumber` | `hexutil.Uint64` | no go-ethereum `rpc` import; identical JSON for reachable values |
| `sonic_prepareBundle`, `sonic_submitBundle` "current block" | the node's live head | the archived head | I3; can shift the default range and base fee by a block |
| `eth_getRules` | an epoch number, `null` for an epoch with no state | the rules in force at a block; the tags that select the head answer the rules the next block is processed with, as the open epoch did | X2; lakme `rpc-module-getRules-block-selector` |
| `eth_blockNumber` | committed head | archived head | I3 (ADR4) |

### Bugs of the reference the port treats (F8)

Confirmed by running or reading the code on `main` and `v2.2.4`. The module
reproduces the first two so lakme stays green; it fixes the two panics, which
the table above records, and the BLOCKHASH walk, which changes no answer. Each
is fixed deliberately in a release, with a `defects.json` entry (none for the
walk: lakme cannot see it) and an issue holding one failing test.

| Where | Bug | Fix | Tracking |
|---|---|---|---|
| `AccessList` (`api.go`) | when the caller omits `gas`, the per-round `setDefaults` gas estimate runs *before* `args.AccessList = &accessList`, so the round that finds an entry runs with the previous round's gas and fails `intrinsic gas too low` | assign the access list before the re-estimate | sonic-admin#904, branch `luis/repro-createaccesslist-intrinsic-gas` |
| `setDefaults` (`transaction_args.go`) | the call object for the gas estimate copies `Data` and not `Input`: a call that sent only `input` is estimated with no calldata and reverts | copy `Input` too, or use `args.data()` | sonic-admin#905, branch `luis/repro-createaccesslist-ignores-input` |
| `PublicFilterAPI.Stop` (`filters/api.go`) | exported, so served as `eth_stop`/`ftm_stop` | stop registering it: wrap the API in `Service.APIs`, since `gossip` calls `Stop` from another package. **File and fix in the 2.2 line now**, not at step 6 | issue to file (S1) |
| `replayBlock` (`tx_trace.go`) | the untraced branch indexes `receipts[i]` unguarded: a short list panics | hoist the guard the traced branch has | issue to file |
| `traceBlock` (`api.go`) | ignores the error from `TxAsMessage`, then dereferences the nil message | return it | issue to file |
| `GetHashFn` (`evmcore/evm.go`) | lacks geth's `ref.Number <= n` guard and walks the parent hashes back from the block of the call. A block override `number` above the head lets BLOCKHASH ask for the head or later: the walk reads every header down to genesis without finding it, and keeps 32 bytes per block. On a 20,000-block chain one `eth_call` read 20,000 headers and `eth_estimateGas` 460,000; on mainnet one unauthenticated request costs minutes and GBs, and neither the request deadline nor `evm.Cancel` stops it inside the opcode. A `number` far below the head walks from the head down to it | the guard in `GetHashFn` (block processing never asks at or above its block, so consensus is unchanged); the module answers BLOCKHASH by a lookup by number (`exec.Env.BlockContext`). Same answers, see V11 | issue to file |

Considered and rejected, to stay identical to the reference: erroring on an
`eth_feeHistory` count above 1024 (legacy clamps, lakme draws up to 1100), and
dropping the legacy fee and gas defaulting of `eth_createAccessList` (changes
`gasUsed`, loses an "insufficient funds" answer).

Inherited quirks kept as they are, not deviations:

- `TxSyncMaxTimeout` (30s) is unreachable while `RPCTimeout` (5s) bounds every
  request; an operator who wants long sync waits raises `--rpc.timeout`.
- `RPCEVMTimeout` only bites where geth's interpreter runs (section 8).
- The EIP-2935 history contract is written where legacy replays wrote it:
  `stateAtTransaction` runs it, so `debug_traceTransaction`, `debug_traceCall`,
  `trace_transaction` and `trace_call` see it once before the traced
  transaction; `traceBlock` and `trace_block` never run it. The untraced branch
  of `replayBlock` calls it per replayed transaction. `exec.Options.
  SkipParentBlockHash` selects the behaviour. Read the legacy path before
  deciding; the intuitive answer is wrong for the transaction tracers.
- `eth_simulateV1` does not build on `rpc/exec`: a simulated block's number is
  usually above the head so `Rules(n)` fails, BLOCKHASH must see simulated
  predecessors, validation mode needs the base fee, and the simulator needs its
  own precompile set and a state wrapper for the log tracer.

## 7. Testing program

Four tools, each with one job. No parity harness against a second process; the
reference is the newest `v2.2.x` tag.

### 7.1 Compatibility: lakme

`../lakme` starts two sonic versions from the same exported genesis and draws
queries over the whole history with `rapid`, comparing answers field by field.

- Run: `-lakme.a <reference> -lakme.b <branch>`, `-rapid.checks 500` on PRs,
  3000 nightly. Against any 2.2.x use `-lakme.genesis testdata/synth_no_canto.g`
  (that line predates Canto and cannot import the default genesis).
- One reference, always. Every difference is the port's until justified: fixed
  in the module, or an entry in `defects.json`. Deliberate deviations carry no
  version bounds and an id starting `rpc-module-`; release differences carry
  `introduced`/`fixed_in`. Where one side answers a result and the other an
  error the diff path is empty, so such an entry needs
  `^(|error\\.(code|message))$`.
- An uncommitted tree is compared as a dangling commit (`GIT_INDEX_FILE` on a
  scratch index, `git commit-tree`), the sha as `-lakme.b`.
- rapid stops a family at its first failure. Before the next round fix the
  *class*: list every site of the same shape and check each against its legacy
  counterpart. One round per instance is the slow way.
- Entries this plan expects: `rpc-module-tracer-timeout-limit` (V6),
  `rpc-module-getRules-block-selector` (X2), `eth_blockNumber` archive head (I3,
  only if lakme's chain lags), uniform `MaxResponseSize` (O1, only if a reply
  exceeds it). Already in `defects.json` for 2.2.3 and not to be redone:
  `trace_filter-range-limit`, `log-filter-parameter-limit-upfront`,
  `getProof-storage-key-validation`, `trace-call-missing-block-panic`.
- lakme does not draw `eth_subscribe`, `eth_sendRawTransaction*`, `txpool_*`,
  `sonic_*`, `eth_syncing`; those are integration territory (7.5). Gap to close
  in lakme: a WebSocket side (ADR8).
- A green lakme run says nothing about what lakme cannot provoke: one node
  configuration (the defaults), one transport, requests that finish in time,
  no panics, no shutdown, no concurrency between consumers. lakme green is
  necessary, not sufficient: section 7.6 lists the classes it misses and their
  gates.

### 7.2 Performance: lakme benchmarks

Nine methods, `latest` and `archive` variants, plus `debug_traceBlockByNumber`
on the fullest blocks; side `a` is timed, one version per sweep:

    go test -run '^$' -bench . -benchtime 1000x -count 5 \
      -lakme.a <ref> -lakme.b <ref> -lakme.genesis testdata/synth_no_canto.g

Protocol, because a single sweep per version is not a measurement:

- One genesis for both versions.
- Sweeps interleaved in balanced order, `new, release, release, new`, twice, so
  each version's mean position in time is the same (a laptop CPU drifts by half
  within an hour; the first sweep is the fastest).
- Each version is first compared with itself: `benchstat` over its own sweeps
  must agree within 5% geomean; an interrupted or disturbed sweep is discarded.
  When one version fails and the other passes, compare over the pair in which
  both reproduce and say so.
- Then `benchstat release.txt new.txt`, all sweeps concatenated; gate per T2.

Reference numbers (2026-09-21, Ryzen 7 PRO 7840U, 20 samples, measured against
v2.2.2; re-measure against the current reference at step 2): geomean 405.7µs →
344.6µs (−15.1%). Cheap state reads gain 20–32% (`GetStorageAt`, `GetBalance`,
`GetLogs`), `GetBlock` 11–17%, `Call` 13%, `TraceCall` 8–11%, `GetProof` 7%,
`EstimateGas` and `TraceBlock/busiest` 1–4%. The saving is a roughly constant
30–45µs per request in the RPC layer; nothing in the module makes execution
faster, so an `EstimateGas` or `TraceBlock/busiest` moving by more than a few
percent means something other than the RPC layer changed.

### 7.3 Ethereum execution APIs

The `ethereum/execution-apis` fixtures (`tests/<method>/*.io`, 29 methods today)
run through a Sonic runner built on `rpctestgen`, a separate private project.
Covers the eth methods marked E and `debug_getRaw*`. To do: publish the runner
and run it on every PR (T3); a Sonic import of `chain.rlp`/`genesis.json` and one
allow-list of Sonic-specific divergences (no uncles, no blobs, base fee rules).
(`tests/eth_exec_spec` here runs the execution-spec state tests, a different suite.)

### 7.4 Handler tests: `rpc/tests` on `Core`

The successor of `api/rpctest`: a `FakeCore` with a builder for accounts, blocks
and transactions and panic injection per `Core` method, and `NewCarmenCore`, a
real in-memory Carmen (its `EndBlock` returns a nil channel when there is no
archive; do not wait on it). A mockgen mock of `Core` (the repository's
convention; Jenkins checks generated mocks) serves error-injection rows; the
fake serves stateful flows. Handler tests live next to their namespace and call
through the dispatcher. Every handler lands with its rows; every wording in
F3–F5 has one assertion; every `Core` method a handler touches has a
panic-injection row.

### 7.5 Integration tests: `tests/`

The RPC tests in `tests/` (`eth_call_test.go`, `rpc_get_logs_test.go`, `rpcs/*`,
`log_subscription_test.go`, `pending_transactions_subscription_test.go`,
`node_restart_test.go`, `rpc_replay_test.go`, …) run unchanged against the
module. The harness's core client uses the IPC socket for `test_*`, `admin_*`
and `dag_*`.

### 7.6 What the tools do not see

lakme, the fixtures and the handler tests compare answers under one default
configuration. Every class below passed them all, so each has a gate of its own
and is a checklist item of the review (T7). For each class, list the sites from
the legacy code before porting, not from the port afterwards.

| Class | How it hides | Gate | Instance an execution shipped |
|---|---|---|---|
| Guards that depend on node configuration | lakme runs the defaults | an adapter test per `svc.config` read in the legacy backend (section 2) | `TxIndex` off: receipts and lookups answered empty instead of the error |
| Config keys that parse but do nothing | a parse test passes | an effect test per key (O4) | `GPO.MinGasPrice` floor dropped (ADR9) |
| The node's own readers of the configuration `takeOverRPC` rewrites | nobody queries them over RPC | O5, listed before step 1 (section 3) | `--unlock` allowed with HTTP open |
| In-process clients of the node's handler | they are not on any transport | O5, every `Attach()` caller listed before step 6 | hardware-wallet derivation lost `eth_getBalance` |
| Endpoint rules of geth's `node` package | tests use one endpoint layout | a `takeOverRPC` row per layout geth distinguishes | same WebSocket port with another host did not start |
| Behaviour under a configured limit or timeout | requests finish in time | the differential test with a slow method and each cap (Q5, D3) | no -32002; batches lost their unanswered calls |
| The current-epoch rules | the head seals an epoch only at epoch ends | `Rules(head+1)` at every such site (section 2), a row where the head seals an epoch | `MaxGasLimit` read the sealed epoch |
| Restart after a panic | nothing panics in a green run | P3: the loop works after the restart | the head froze while the restart counter said it had recovered |
| Shutdown and admission races | requests do not race `Stop` | P7, a stress test under `-race` | a request admitted after `Stop` returned |
| Mutation of shared values | one consumer per value | P8, a shared value is unchanged after the call, `-race` | `returnLogs` wrote topics into a log shared by all subscribers |
| Cost multiplied by consumers | the benchmarks have one client | I16, `Core` calls counted with N consumers | `newHeads` loaded receipts once per subscriber |
| Cost of a coarse `Core` call | the answer is right | read what the legacy code read, per caller (I10) | the oracle tick walked every validator's last event |
| Defaults written twice | they agree today | one definition (section 8) | the oracle's maximum gas price default in three places |
| Cost a client chooses | the answers are right, and fast for the values honest clients send | V11: for every number a request carries (overrides, indexes, range ends), count the `Core` calls with hostile values: above the head, far below it, the maximum of the type | a block override `number` above the head made one BLOCKHASH read every header down to genesis (F8 table) |

## 8. Limits inherited from the reference

The module changes no limit the reference enforces. Every knob keeps its name,
default and effect (O1); the only change of mechanism is `RPCTimeout`, applied by
the dispatcher (O2). Each default has one definition, in the module package that
consumes it; `gossip.DefaultConfig`, `config` and `dispatch.DefaultConfig` refer
to it and never repeat the literal (`gossip/gasprice` delegates to `rpc/oracle`).
A copy agrees until someone changes one of them. Nothing here bounds how many requests run at once; that is
out of this plan (the dispatcher is where a limiter would go, `rpc_inflight`
shows the need).

| Knob | Default | Kind | Where |
|---|---|---|---|
| `RPCGasCap` | 50,000,000 | cap on `eth_call`/`estimateGas`/traces | `gossip.DefaultConfig` |
| `RPCEVMTimeout` | 5s | EVM interpreter cancel; only geth's interpreter polls it (at jumps), Tosca's LFVM/SFVM never do, so on the chain interpreter the bound is `RPCGasCap` | `DoCall`, tracers, `simulate.go`; timeout test in `rpc/eth` |
| `RPCTimeout` | 5s | per-request deadline; today a fork patch (`SetExecutionTimeLimit`) | `gossip.NewService` |
| `RPCTxFeeCap` | 100 FTM | refuse expensive sends | `checkTxFee` |
| `MaxResponseSize` | 25 MiB | only `debug_*` and `trace_*` check it | `api.go` |
| `StructLogLimit` | 2000 | struct-log length | `gossip.DefaultConfig` |
| `MaxTraceFilterRange` | 1000 (`--rpc.tracefilterrange`, 0 = unlimited) | widest `trace_filter` span, since 2.2.3 | `gossip.DefaultConfig` |
| `eth_getProof` keys | 1024 | storage keys per request, since 2.2.3 | `maxGetProofKeys` |
| `LogQueryParameterLimit` / `LogQueryResultLimit` | 1000 / 5000 | parameters (every filter path since 2.2.3) / results | `filters.Config` |
| `IndexedLogsBlockRangeLimit` / `UnindexedLogsBlockRangeLimit` | effectively none / 100 | block span of indexed / block-scan log queries | `filters.Config` |
| filter `deadline` | 5 min | unpolled filters are dropped | `filters/api.go` |
| `BatchRequestLimit` / `BatchResponseMaxSize` | 1000 / 25 MB | batch limits | `config/node_config.go` |
| HTTP body, WS message | 5 MiB, 32 MiB | transport limits | the fork's `rpc` package |
| HTTP timeouts | read 30s, write 30s, idle 120s; at write − 100ms a running call is answered -32002 (section 3) | `net/http` | `config/defaults.go` |
| `GPO.MaxGasPrice` / `MinGasPrice` / `DefaultCertainty` | 10,000,000 gwei / 0 / 0.5 | ceiling and floor of the oracle's suggestion; certainty of a request that names none | `gossip/gasprice/config.go`, ADR9 |
| `TxSyncDefaultTimeout` / `TxSyncMaxTimeout` | 5s / 30s | `eth_sendRawTransactionSync` wait; **`main` only**, not in 2.2.x, no lakme row | `gossip.DefaultConfig` |
| `--mode validator` | | disables public RPC on validators | `config/config.go` |

How each is tested: lakme draws requests that hit `RPCGasCap`,
`MaxResponseSize`, `StructLogLimit`, `MaxTraceFilterRange`, the `LogQuery*` and
log-range limits, the `getProof` cap and the `eth_feeHistory` clamp (T1);
`RPCTxFeeCap`, `TxSync*`, the batch limits, body and message sizes and
`--mode validator` by integration tests in `tests/`, added where missing (T5);
the timeout test in `rpc/eth` covers `RPCTimeout` and shows where `RPCEVMTimeout`
does and does not apply (T6).

## 9. Open decisions

Decided: ADR1 module path `rpc/`; ADR2 an own JSON-RPC layer (section 3);
ADR3 uniform `MaxResponseSize` (O1); ADR5 the legacy namespaces of X3 live in
`api/legacy` and `api/testapi`, served by the node's own handler on IPC only,
its wallet methods defaulting fees and gas through the module's oracle and
`eth_estimateGas`; ADR11 the legacy engine is deleted with the switch: no
`--rpc.engine` flag, no escape hatch, `Front` keeps only the IPC routing of X3
and the p2p facts.

| ID | Decision | Recommendation |
|---|---|---|
| ADR4 | One `latest` (archive head) for `eth_blockNumber` | Yes (I3). Annotate in lakme; a client that reads the number can always query it |
| ADR6 | New semantic caps other clients have and we lack | `eth_simulateV1` block count: 256 (reth, nethermind). `trace_filter` is already capped (inherited); `eth_getLogs` stays unlimited-but-result-capped for compatibility |
| ADR7 | `debug_getRaw{Block,Header,Receipts,Transaction}` | Add in step 5; fixtures exist and they are block reads |
| ADR8 | lakme WebSocket side | Add after step 4, so subscriptions and filters over WS are compared; until then integration tests carry them |
| ADR9 | The `GPO` config section (`MaxGasPrice`, `MinGasPrice`, `DefaultCertainty`; TOML only) once the legacy oracle is deleted | Keep all three and feed them to the module's oracle through `Config`. `MinGasPrice` is not ignored: it is the floor of the suggestion (`utils.BigMax(gpo.cfg.MinGasPrice, reactive, constructive)` in `Oracle.suggestTip`, `gossip/gasprice/gasprice.go`). An earlier execution read it as ignored, kept the key and dropped its effect, which a parse test does not see (O4). The loader rejects unknown fields, so removing a key is breaking; that is why the section stays |
| ADR10 | Coverage floor for Q4 | 80% line coverage per module package, measured before step 6 lands so the number comes from evidence; raised, never lowered |
| ADR12 | Shared HTTP/WebSocket endpoint during steps 1 to 3 | Reverse-proxy the WebSocket upgrade (`httputil.ReverseProxy` handles it) to the node's own server on a private loopback endpoint; the alternative, taking over WS at step 1, drags subscriptions into the skeleton |
