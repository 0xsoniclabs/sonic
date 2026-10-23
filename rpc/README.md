# RPC module

The `rpc` package serves the JSON-RPC interfaces of the node (HTTP, WebSocket
and the IPC socket) in the node's process. It reaches the node only through the
narrow `core.Core` interface; `gossip/rpc_core.go` is the one adapter. The design
and the requirement ids (I, D, P, Q, V, F, S, O, MON, X, G, T) are in
[PLAN.md](PLAN.md).

## Layout

| Package | Content |
|---|---|
| `core` | `Core`, the interface to the node |
| `dispatch` | method table, typed registration, deadlines, panic containment, served head (`Chain`), subscriptions (`Notifier`), metrics |
| `transport` | JSON-RPC envelope, HTTP, WebSocket, IPC, `Front` (dispatcher first, legacy handler second) |
| `eth`, `filters`, `trace`, `debug`, `txpool`, `sonic`, `net`, `web3` | the namespaces |
| `exec` | call, estimate, access list and replay on archived state |
| `oracle` | gas price oracle |
| `tests` | `FakeCore`, `CarmenCore`, `ReplayChain`, accounts, contract code and a harness to test handlers without a node |

## Operator notes

- This module is the only RPC server of the node. There is no switch back to
  the handlers of the previous releases.
- Every RPC flag and configuration key keeps its name and effect. The module
  reads the limits from the node configuration at start: `RPCTimeout`,
  `RPCEVMTimeout`, `RPCGasCap`, `RPCTxFeeCap`, the log filter limits, the batch
  limits and the `GPO` section.
- The wallet methods, `eth_getEpochBlock`, `eth_getEpochStats`,
  `debug_chaindbProperty`, `debug_chaindbCompact`, `debug_setHead` and the
  `admin`, `dag`, `abft`, `personal` and `test` namespaces are answered on the IPC
  socket only (by the node's own handler, `api/legacy` and `api/testapi`).
- The Tosca interpreter does not observe `--rpc.evmtimeout`. A call that does
  not run on the go-ethereum interpreter stops at the gas cap
  (`--rpc.gascap`). Keep the gas cap set on public nodes.
- The node holds at most 100,000 polled filters (`eth_newFilter`,
  `eth_newBlockFilter`, `eth_newPendingTransactionFilter`); beyond that a new
  filter is refused. Their events are kept once for all of them, at most 2^20
  of a kind; a filter not polled before its events were dropped is answered
  `filter not found`, as an expired one.
- `trace_call` and `trace_callMany` are bounded by `--rpc.maxresponsesize` as
  `debug_trace*` and `trace_filter` are: above it they answer `response too
  large`. The calls of `trace_callMany` share the limit.
- A request that panics is answered with `-32603` and counted in
  `rpc_panics_total`. A loop of the module that panics is restarted and counted
  in `rpc_loop_restarts_total`.

## Metrics

All metrics are on the default Prometheus registry, which the metrics endpoint of
the node serves. Labels are bounded: unknown method names are counted as `other`.

| Metric | Labels |
|---|---|
| `rpc_requests_total` | `method`, `result` (`ok`, `error`, `invalid_params`, `not_found`, `panic`, `timeout`, `cancelled`, `rejected`) |
| `rpc_duration_seconds`, `rpc_response_bytes` | `method` |
| `rpc_connections` | `transport` (`http`, `ws`, `ipc`) |
| `rpc_inflight` | |
| `rpc_subscriptions` | `kind` |
| `rpc_subscription_events_total` | `kind`, `outcome` |
| `rpc_head_served`, `rpc_head_lag` | |
| `rpc_panics_total` | `method` |
| `rpc_loop_restarts_total` | `loop` |

Alerts to define (MON6):

- `increase(rpc_panics_total[5m]) > 0`
- `rpc_head_lag > 1` for one minute
- the 99th percentile of `rpc_duration_seconds` of a method over the budget of
  that method

## Developer notes

- Add a method with `dispatch.Add0` … `Add4`. A parameter name ending in `?` is
  optional. Handlers get a context that carries the request deadline; check it in
  every loop.
- Take the head once per request (`chain.Head()`) and use it for every part of
  the answer.
- Resolve a block reference with `chain.Resolve` and look blocks, headers,
  transactions and state up with the `exec.Env` methods; do not write a private
  copy in a namespace. Use `env.OpenState` when the header is not used, and keep
  block numbers `uint64` (`dispatch.BlockNumber` is for client input).
- Return `dispatch.Encoded` for a result you encoded with `json.Marshal`; a
  `json.RawMessage` is compacted again.
- Delete what a change leaves without a caller in the same commit.
- A namespace imports `rpc/core`, `rpc/dispatch`, `rpc/exec` and go-ethereum
  value types only; `rpc/lint_test.go` checks the rule.
- Test a handler on `tests.FakeCore` (no node) or on `tests.NewCarmenCore` (real
  state), served by `tests.NewHarness` or, for a namespace on `exec.Env`,
  `tests.NewEnvHarness`. Take accounts, transactions, contract code and the
  replay chain from `rpc/tests`; write a helper of your own only for what one
  package needs. Use `FakeCore`, not a generated mock of `Core`: tests state
  what a request answers, not which `Core` calls it makes. The test of
  `rpc/matrix_test.go` injects failures, nil values and panics into every
  method of the module.
- Compare with the reference release with `lakme` (`defects.json` lists the
  accepted differences).
