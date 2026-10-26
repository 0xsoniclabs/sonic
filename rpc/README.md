# The RPC module

`rpc/` is the JSON-RPC server of sonicd. It serves every namespace a client
reaches over HTTP, WebSocket and IPC, and reads the node through one narrow
interface, `core.Core`. The node's own handler serves only the namespaces that
stay with the node (the legacy wallet, `dag`, `abft` and epoch methods, `test`,
`admin`, the p2p facts of `net`), on IPC. [PLAN.md](PLAN.md) holds the
requirements, the method inventory and the decisions; this file is for the
people who run the node and for those who change the module.

## For operators

### Endpoints and flags

The flags and configuration keys are those of 2.2.x: `--http*`, `--ws*`,
`--ipcpath`, `--ipcdisable`, `--rpc.*` and the `Opera`, `Opera.FilterAPI`,
`Opera.GPO` and `Node` sections of the TOML file keep their names, defaults and
effects. The module opens the endpoints the node configuration describes and
the node opens none: an HTTP and a WebSocket endpoint on one port share one
listener, on the WebSocket host, as before.

- `--http.api` and `--ws.api` enable namespaces per transport, as before. IPC
  serves every namespace.
- `rpc_modules` lists the namespaces the transport it is called on serves.
- The legacy namespaces answer on IPC only: `eth_accounts`, `eth_sign`,
  `eth_signTransaction`, `eth_sendTransaction`, `eth_fillTransaction`,
  `eth_resend`, `eth_getEpochBlock`, `eth_getEpochStats`, `personal_*`,
  `dag_*`, `abft_*`, `debug_chaindbProperty`, `debug_chaindbCompact`,
  `debug_setHead`. Over HTTP and WebSocket they answer `-32601`. Use
  `sonictool --datadir <dir> cli` or another IPC client for them.
- `--unlock` is refused while HTTP or WebSocket is open, unless
  `--allow-insecure-unlock` is set, as before.
- `--mode validator` drops the archive and the indexes, as before; the
  endpoints stay as configured, and without an archive the module serves the
  committed head.
- `GET /health` on the HTTP endpoint answers `{"head": <served block>,
  "lag": <committed head minus served head>}`.

### Limits

Every limit of 2.2.x holds with its name and default (PLAN.md, section 8):
`--rpc.gascap`, `--rpc.evmtimeout`, `--rpc.txfeecap`, `--rpc.timeout`,
`--rpc.maxresponsesize`, `--rpc.structloglimit`, `--rpc.tracefilterrange`, the
log query limits, the batch limits and the HTTP timeouts. Two are new:

- At most 100,000 polled filters per node; a further `eth_new*Filter` answers
  `too many filters, the limit is 100000`.
- A polled filter whose events were dropped before its client polled (more
  than 2^20 events of its kind since its last poll) answers `filter not found`,
  as an expired filter does; clients install a new one.

`--rpc.timeout` is the deadline of every request. The EVM interpreters of the
chain do not poll for it, so an `eth_call`, an estimate, a simulation or a trace
runs until it ends or its gas runs out: on Sonic's interpreters `--rpc.gascap`
bounds it, and `--rpc.evmtimeout` only applies where go-ethereum's interpreter
runs.

Nothing bounds how many requests run at once (ADR15). A `trace_filter` runs one
replay worker per CPU; a few of them at once take every core. Watch
`rpc_inflight`.

A subscription whose client does not read, or whose feed lost events, closes
its connection with the reason; the client reconnects and subscribes again,
and sees no gap in what it received.

### Shutdown

On shutdown the module stops accepting connections, waits for the requests in
flight up to `--rpc.timeout`, cancels the rest and waits for them. A request the
cancel cannot reach (an EVM run on Sonic's interpreters, bounded by its gas) is
logged once the grace period is over, with its method and age:
`RPC requests still running after the grace period`.

### Metrics

On the node's metrics endpoint (`--metrics`, `/debug/metrics/prometheus/native`):

| Metric | Labels | Meaning |
|---|---|---|
| `rpc_requests_total` | `method`, `result` | requests by method and outcome (`ok`, `error`, `invalid_params`, `not_found`, `panic`, `rejected`) |
| `rpc_duration_seconds` | `method` | time to answer |
| `rpc_response_bytes` | `method` | size of the answer |
| `rpc_inflight` | | requests being answered |
| `rpc_connections` | `transport` | open WebSocket and IPC connections |
| `rpc_subscriptions` | `kind` | live subscriptions and polled filters |
| `rpc_subscription_events_total` | `kind`, `outcome` | notifications delivered or dropped |
| `rpc_head_served` | | block the module serves as `latest` |
| `rpc_head_lag` | | committed head minus served head, in blocks |
| `rpc_panics_total` | `method` | panics recovered; a handler, loop or child |
| `rpc_loop_restarts_total` | `loop` | background loops restarted after a panic |

A method name the module does not serve is counted as `other`; no label holds
an address, a block or a client.

### Alerts

MON6: operations installs these; the release does not depend on them.

| Alert | Expression | Why |
|---|---|---|
| `RPCPanics` | `increase(rpc_panics_total[5m]) > 0` | every panic is a defect; the stack is in the log |
| `RPCLoopRestarting` | `increase(rpc_loop_restarts_total[10m]) > 0` | a feed loop that restarts drops its subscribers each time |
| `RPCHeadLag` | `min_over_time(rpc_head_lag[1m]) > 1` | the served head stays more than a block behind the committed one for a minute: the archive or the head feed is stuck |
| `RPCSlowMethod` | `histogram_quantile(0.99, sum by (method, le) (rate(rpc_duration_seconds_bucket[5m]))) > <budget of the method>` | p99 of a method over its budget; start from 0.5 s for reads, 5 s (`--rpc.timeout`) for calls and traces |

## For developers

### Layout

| Package | Holds |
|---|---|
| `rpc/core` | `Core`, the interface to the node, and its value types |
| `rpc/dispatch` | the method table, decoding, deadline, recover, metrics, `Group`, loops, `Stop`, subscriptions |
| `rpc/chain` | the served head, its feeds, block references and their resolution |
| `rpc/exec` | `Env`: the lookups every namespace uses, call arguments, overrides, the EVM environment, the send path |
| `rpc/format` | the JSON forms of blocks, transactions, receipts and logs |
| `rpc/oracle` | the gas price oracle |
| `rpc/eth`, `filters`, `debug`, `trace`, `txpool`, `sonic`, `net`, `web3` | one namespace each |
| `rpc/transport` | HTTP, WebSocket, IPC, the envelope, and `Front`, which routes per transport |
| `rpc/tests` | test support: `FakeCore`, `NewCarmenCore`, the replay chain, the harness |
| `rpc` | `Module`, which wires them |

The node side is `gossip/rpc_core.go` (the adapter behind `Core`) and
`cmd/sonicd/app/rpc.go` (`takeOverRPC`, the module's configuration, the
forwarder to the node's handler). `api/legacy` and `api/testapi` are the
node's handler.

### Rules and the tests that hold them

Each rule here is checked by the named test; a rule no test checks is not
written here.

| Rule | Test |
|---|---|
| The imports follow the Layout table of PLAN.md: no namespace imports another, `dispatch` imports no node package, nothing imports `gossip` or `api` but the listed leaves, go-ethereum within its keep list | `TestImports_FollowTheLayoutAndTheBudget` (`rpc/imports_test.go`) |
| Goroutines start through the dispatcher only | `TestLint_NoBareGoStatementOutsideTheDispatcher` |
| The fork's `SetExecutionTimeLimit` is not used; the dispatcher sets the deadline | `TestLint_NoExecutionTimeLimit` |
| A namespace takes a role of `Core`, never `core.Core`; `Env` exports no field | `TestLint_NamespacesReachCoreThroughRoles` |
| `Core`'s methods are frozen; only `SendTx` changes the node | `TestCore_MethodSetIsFrozen`, `TestCore_SendTxIsTheOnlySetter` |
| An exported symbol has a caller outside tests | `TestH4_ExportedSymbolsHaveCallers`; unexported: `staticcheck -checks U1000 ./rpc/...` (2026.2 or later, built with the module's Go) |
| The module serves exactly the methods of section 6 | `TestInventory_TheServedSetIsSection6` |
| Every method of go-ethereum's specification is served or listed in section X | `TestInventory_EveryMethodOfGoEthereumIsServedOrListed` |
| Every served method has a handler test in its namespace's package | `TestHandlerTests_NameEveryMethod` |
| No `Core` failure, in any mode, makes a request panic; a `Core` panic is contained and counted | `TestMatrix_EveryCoreFailureIsAnsweredWithoutPanic` |
| No argument shape makes a request panic | `TestMatrix_EveryArgumentShapeIsAnsweredWithoutPanic` |
| A cancelled request returns within one `Core` call | `TestCancellation_EveryMethodReturnsWithinOneCoreCall` |
| `Stop` leaves no goroutine and logs what it waits for | `TestStop_UnderLoadLeavesNoGoroutine`, `TestStop_DuringAnEVMRunAtTheGasCap`, `TestStop_DuringABlockedCoreCall` |
| The envelope matches go-ethereum's server, limits included | `TestEnvelope_MatchesGoEthereum`, `FuzzFrame` (`rpc/transport`) |
| The node's handler serves exactly the legacy methods | `TestNodeHandler_ServesExactlyTheLegacyMethods` (`tests/`) |
| Flags and TOML keys of the reference keep their effect | `TestFlags_*`, `TestTOML_*` (`cmd/sonicd/app/config_golden_test.go`) |

Run the module's tests with `go test -race ./rpc/...`. The integration tests
in `tests/` reach the nodes over HTTP; with `SONIC_TESTS_RPC_TRANSPORT=ws` their
clients use WebSocket instead (T5). The IPC client of the harness
(`GetIpcClientConnectedToNode`) reaches the namespaces only IPC serves.

### Writing a handler

- Take the head once: `head := env.Latest(ctx)`, and pass it to every lookup.
  The lookups check the request's context before and after each `Core` call
  and answer absence as `core.ErrNotFound` or a sentinel of `chain.Resolve`;
  map it at the call site to the wording of the legacy handler, or to `null`
  with `exec.NullIfNotFound`.
- Register the method in the package's `Register`; add its request to the
  matrices in `rpc/module_test.go` and its handler tests next to it, on
  `rpc/tests`.
- A difference a client can see from 2.2.x is a row of the deviations table
  in PLAN.md, with its lakme `defects.json` entry.
