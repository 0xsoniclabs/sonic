# The RPC module

`rpc/` serves the node's JSON-RPC API over HTTP, WebSocket and IPC. It
replaces the go-ethereum `rpc.Server` the node ran before, the `api/ethapi`,
`api/sonicapi` and `gossip/filters` packages, and the gas price oracle of
`gossip/gasprice`. It reaches the node only through one interface,
`rpc/core.Core`, which `gossip/rpc_core.go` implements. `PLAN.md` holds the
design, the requirements (their IDs are cited below) and the deviations from
the reference release.

## Operator notes

### Endpoints and flags

Nothing changes in how a node is configured. `--http.*`, `--ws.*`,
`--ipcpath`, `--ipcdisable`, the HTTP server timeouts, the batch limits and
`--mode validator` keep their meaning. At start the node hands its HTTP,
WebSocket and IPC endpoints to the module (`takeOverRPC` in
`cmd/sonicd/app`); the node's own servers stay off.

- WebSocket shares the HTTP server when the ports match, whatever the hosts,
  as go-ethereum decides it; the shared server listens on the WebSocket host.
- `--http.api` and `--ws.api` select the namespaces served there; IPC serves
  every namespace.
- `--unlock` is refused with HTTP or WebSocket open unless
  `--allow-insecure-unlock` is set, as before.
- Hardware wallets derive their accounts reading the chain through the module.

### What only IPC serves

The node's own handler still serves what the module does not port, on IPC
only (`sonictool attach` reaches it):

- `admin_*`;
- `test_*` (with `--enable-test-only-api`);
- the legacy namespaces, until they are removed: `personal_*`,
  `eth_accounts`, `eth_sign`, `eth_signTransaction`, `eth_sendTransaction`,
  `eth_fillTransaction`, `eth_resend`, `dag_*`, `abft_*`, `eth_getEpochBlock`,
  `eth_getEpochStats`, `debug_chaindbProperty`, `debug_chaindbCompact`,
  `debug_setHead`, and their `ftm_` twins. Their wallet methods take the tip
  and the gas of the transactions they complete from the module.

Over HTTP and WebSocket these answer `-32601`, and `rpc_modules` does not list
their namespaces. `net_listening` and `net_peerCount`, which only the node's
p2p server knows, are answered on every transport.

The profiling methods of go-ethereum's debug handler (`debug_memStats`,
`debug_writeMemProfile`, …) answer `this method is not enabled`, as before;
in a build with the `enable_debug` tag they are served on IPC only.

### Behaviour that differs from 2.2.5

The full list, with the reasons, is section 6 of `PLAN.md`. What an operator
or a client may notice:

- `eth_blockNumber`, `latest` and every tag name the newest *archived* block
  the module has seen, the same block for every method (ADR4).
- A parameter that does not decode is answered `-32602` with
  `parameter N (name): …` instead of `invalid argument N: …`.
- `eth_pendingTransactions` answers the pool's pending transactions.
- At most 100,000 polled filters on a node (`too many filters, the limit is
  100000`), and a filter that has not polled for over 2^20 events of its kind
  is removed (`filter not found`). Unpolled filters expire 5 to 6 minutes
  after their last poll.
- At most 10,000 subscriptions per WebSocket or IPC connection. A
  subscription that loses events ends, and its connection closes with the
  reason; a client that does not read its notifications is disconnected.
- `debug_trace*` refuse a `timeout` that does not parse or is above
  `--rpc.evmtimeout` with `-32602`.
- `trace_call` and `trace_callMany` are bounded by `--rpc.maxresponsesize`
  as a whole, the calls of `trace_callMany` sharing it; a vmTrace stops at
  what is left of it (`response too large`).
- `sonic_getBundleInfo` reports its `block` as a quantity.

### Limits

Every limit of the reference keeps its flag, TOML key, default and effect
(section 8 of `PLAN.md`). The request deadline `--rpc.timeout` (5s) is applied
by the module's dispatcher to every request on every transport; a request
over it is answered `-32002`. Nothing bounds how many requests run at once
(ADR15): a `trace_filter` request replays with one worker per CPU, so a few
concurrent ones can take every core. Watch `rpc_inflight`.

### Metrics

The module registers on the default Prometheus registry, served by the
node's metrics server at `/debug/metrics/prometheus/native` (`--metrics`
with `--metrics.addr` and `--metrics.port`).

| Metric | Labels | What |
|---|---|---|
| `rpc_requests_total` | `method`, `result` (`success`, `error`, `invalid`, `panic`, `rejected`) | requests answered |
| `rpc_duration_seconds` | `method` | time to answer, a histogram |
| `rpc_response_bytes` | `method` | size of the encoded result, a histogram |
| `rpc_inflight` | | requests being served |
| `rpc_connections` | `transport` (`http`, `ws`, `ipc`) | open connections |
| `rpc_subscriptions` | `kind` | live subscriptions (`newHeads`, `logs`, `newPendingTransactions`) and polled filters (`blockFilter`, `pendingTransactionFilter`, `logFilter`) |
| `rpc_subscription_events_total` | `kind`, `outcome` (`delivered`, `failed`, `lost`) | events written or polled, writes that failed, subscriptions and filters ended by a loss |
| `rpc_panics_total` | `method` | panics recovered, a request's or a loop's |
| `rpc_loop_restarts_total` | `loop` | restarts of the module's long-lived loops |
| `rpc_head_served` | | the block the module serves as `latest` |
| `rpc_head_lag` | | committed head minus served head, in blocks |

A method name no namespace registers is counted as `other`; no label carries
an address, a block, a client or an error text (MON5).

### Alerts (MON6)

Install these; the thresholds are a starting point.

| Alert | Expression | Why |
|---|---|---|
| `RPCPanics` | `increase(rpc_panics_total[5m]) > 0` | a panic is a bug: the request or loop recovered, the cause did not go away |
| `RPCHeadLag` | `min_over_time(rpc_head_lag[1m]) > 1` | the module serves a head over a block behind the node for a minute: its head loop is stuck or failing |
| `RPCSlowMethod` | `histogram_quantile(0.99, sum by (method, le) (rate(rpc_duration_seconds_bucket[5m]))) > budget` | p99 of a method over its budget; suggested budgets: 0.1s for block, transaction, receipt and state reads, 1s for `eth_call`, `eth_estimateGas` and `eth_getLogs`, 5s (`--rpc.timeout`) for traces and `eth_simulateV1` |

Worth a dashboard without an alert: `rpc_loop_restarts_total` (a loop that
keeps failing restarts with a growing backoff), `rpc_subscription_events_total`
with `outcome="lost"` (consumers too slow for the feeds), `rpc_inflight` (see
Limits) and `rpc_requests_total` with `result="rejected"` (requests during
shutdown).

## Developer notes

Each rule below names the test that holds it.

- **Layout.** The packages, what each holds and what it may import are the
  Layout table of `PLAN.md`, kept as data in `lint_test.go`:
  `TestLayout_EveryPackageIsInTheTable`, `TestLayout_ImportsFollowTheTable`,
  `TestLayout_TestsImportNoForbiddenPackage`. A namespace imports no other
  namespace; what two share moves to `exec` or `format`. Nothing under `rpc/`
  imports `gossip`, `api/*` or go-ethereum's `rpc`, except the leaf packages
  the table lists.
- **One way in.** The node is reached through `Core` only, and a namespace
  reaches it through `exec.Environment`, never directly:
  `TestLint_NamespacesReachNoCore`.
- **Goroutines.** Only the dispatcher starts goroutines (`Go`, `Group`,
  `Loop`, `context.AfterFunc` elsewhere): `TestLint_GoroutinesStartInTheDispatcher`.
- **Deadlines.** The request deadline is the dispatcher's; nothing uses the
  EVM's execution time limit: `TestLint_NoExecutionTimeLimit`.
- **Registration.** A method exists because a `Register` names it, nothing is
  reflected: `TestFront_AMethodNoRegisterNamesIsNotFoundOnEveryTransport`.
  What the node's own handler serves is pinned:
  `TestNodeHandler_ServesExactlyTheLegacyMethods` (`tests/`).
- **Faults.** Every registered method is sent through the injection matrix,
  which makes each `Core` method fail, return nothing and panic, and checks
  that every state opened is released: `TestMatrix_NamesEveryRegisteredMethod`,
  `TestMatrix_EveryCoreFailureIsContained`.
- **Cancellation.** A cancelled request makes at most one more `Core` call:
  `TestMatrix_ACancelledRequestReturnsWithinOneCall`.
- **Names.** Whole words, no name shorter than three letters but the usual
  ones, none hiding a package: `TestLint_NamesAreWholeWords`.

Adding a method: register it in its namespace's `Register` with its
parameters, answer through `exec.Environment`, add its request shapes to the
injection matrix (`matrix_test.go`), test it against `rpc/tests.FakeCore` or
`NewCarmenCore`, and, where it differs from the reference, add a row to
section 6 of `PLAN.md`.
