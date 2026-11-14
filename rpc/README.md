# The RPC module

The RPC module serves the node's JSON-RPC over HTTP, WebSocket and IPC. It
reaches the node only through the interface `rpc/core.Core`, which
`gossip/rpc_core.go` implements, and it answers as the node's legacy RPC
stack does, except where this file says otherwise.

## Selecting the stack

The node serves one of two RPC stacks:

- `--rpc.stack legacy`, the default, or `RPCStack = "legacy"` in the
  `[Opera]` section of the TOML configuration: the legacy stack.
- `--rpc.stack module`, or `RPCStack = "module"`: this module.

Every RPC flag and key keeps its effect in both stacks: `--http.*`, `--ws.*`,
`--ipcpath`, `--ipcdisable`, the batch limits, the HTTP timeouts,
`--rpc.gascap`, `--rpc.evmtimeout`, `--rpc.txfeecap`, `--rpc.timeout`, the
`GPO` section, `--rpc.maxresponsesize`, `--rpc.structloglimit`,
`--rpc.tracefilterrange`, `--rpc.allow-js-tracers` and
`--enable-test-only-api`. `cmd/sonicd/app/rpc_stack_test.go` holds the
mapping of each into the module.

## What the module does not serve

- The `ftm` namespace: its methods answer -32601, and `ftm` in `--http.api`
  or `--ws.api` is ignored with a warning.
- The methods that use the node's accounts: `eth_accounts`, `eth_sign`,
  `eth_signTransaction`, `eth_sendTransaction`, `eth_fillTransaction`,
  `eth_resend` and `personal_*`. `--unlock`, `--password` and
  `--allow-insecure-unlock` are accepted, warn, and unlock nothing, and the
  wallets are not opened. `eth_pendingTransactions` answers `[]`.
- `admin_*` over HTTP and WebSocket: the node's administration is served on
  IPC only, where `sonictool attach` reaches it, as are the profiling
  methods of `debug_*` in a build with `enable_debug`. `admin` in an API list
  is ignored with a warning. `admin_startHTTP`, `admin_stopHTTP` and their
  WebSocket and RPC forms answer -32601: the module owns every endpoint.
- `debug_chaindbProperty`, `debug_chaindbCompact`, `debug_setHead` and
  `debug_getRaw*`.

`rpc/layout_test.go` holds that no package of the module imports the legacy
stack, and `TestLayout_TheNodesInProcessClientsAreKnown` that the one
in-process client of the node's own handler is the wallet loop, which the
module's stack does not start.

## Where the module answers differently

- **One latest.** Every tag, for blocks, state, receipts, logs and
  `eth_blockNumber` alike, names the newest block whose state is archived.
  The legacy stack answers `eth_blockNumber` from the committed head, so a
  client may see a number it cannot query yet. The current epoch and the
  current rules are those of the block after that head.
- **Tracer timeouts** above `--rpc.evmtimeout` (5s by default), and ones
  that do not parse, are refused with -32602.
- **Traced calls are bounded as a whole.** `trace_call` and `trace_callMany`
  answer at most `--rpc.maxresponsesize` bytes, all calls together, and a
  vmTrace stops when it would keep more than is left.
- **Polled filters** live in one journal per kind; at most 100,000 filters
  live, and one that falls behind 2^20 events is removed and answers
  `filter not found`. Filters expire 5 to 6 minutes after their last poll.
- **Subscriptions**: at most 10,000 per connection. Each holds at most 128
  events it has not sent; a subscription that falls behind, or whose feed is
  lost, ends its connection with the reason, so that its client does not go
  on with a gap.
- `eth_sendRawTransactionSync` waits for the block that holds the
  transaction instead of polling.
- BLOCKHASH in calls is one lookup of the block's header; a block override
  of the number cannot make it walk the chain.
- `trace_*` answers `no receipt found for transaction <hash>` where the
  legacy replay of a block with a short receipt list panics.

## Load

Nothing bounds the requests running at once, as in the legacy stack: a few
concurrent `trace_filter` requests can take every core. `rpc_inflight` shows
the load; a bound per class of method is to be decided when it shows the
need.

## Metrics

On the node's metrics endpoint:

| Metric | Labels | Meaning |
|---|---|---|
| `rpc_requests_total` | `method`, `result` | calls served; unknown methods count as `other` |
| `rpc_duration_seconds` | `method` | time to serve a call |
| `rpc_response_bytes` | `method` | size of the results |
| `rpc_inflight` | | calls being served |
| `rpc_connections` | `transport` | open connections |
| `rpc_subscriptions` | `kind` | live subscriptions |
| `rpc_subscription_events_total` | `kind`, `outcome` | notifications sent or failed |
| `rpc_panics_total` | `method` | panics recovered while serving a call |
| `rpc_loop_restarts_total` | `loop` | restarts of a loop after a panic |
| `rpc_head_served` | | the head the module serves |
| `rpc_head_lag` | | committed head minus served head |

## Alerts

Operations installs these on the metrics above:

- **Panics**: `increase(rpc_panics_total[5m]) > 0`. A method crashed; its
  call was answered -32603 and the stack is in the log.
- **Head lag**: `rpc_head_lag > 1` for one minute. The archive trails the
  committed head, so clients see an old latest block.
- **Slow methods**: the 99th percentile of `rpc_duration_seconds` of a
  method above its budget for ten minutes, per method:
  `histogram_quantile(0.99, sum by (method, le) (rate(rpc_duration_seconds_bucket[5m])))`.
