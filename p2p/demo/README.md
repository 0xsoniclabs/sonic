# Sonic p2p demo

A small command-line utility for running nodes on the Sonic p2p network by hand —
to see the network form and to diagnose peer connections. It runs **real** p2p
nodes (validator mesh, gossip discovery, network scan, peer-health monitoring)
on a **faked chain**, with a hard-coded, deterministic validator set.

It is a learning/diagnostic tool, not part of the production node.

## Quick start

Open a few terminals and start some validators (IDs are `0..3`):

```bash
go run ./p2p/demo validator --id 0
go run ./p2p/demo validator --id 1
go run ./p2p/demo validator --id 2
```

Optionally add an archive (no key, no consensus — it just joins and serves
discovery/scan/health):

```bash
go run ./p2p/demo archive
```

The nodes **find each other automatically** over the local network (mDNS) — no
addresses to copy. Within a few seconds the validators authenticate and form a
full mesh, and every node begins logging a table of its peers' health:

```
peer health
  PEER          ROLE              RTT      JITTER    LOSS      HEIGHT   PROBES
  WCtTbZ5yih5Y  archive         2.1ms       0.5ms      0%    79519300        3
  X5vM6gb5YQRB  validator       2.6ms       0.6ms      0%    79519299        3
```

Columns: round-trip time (EWMA), jitter, probe loss rate, the peer's reported
block height, and probes attempted.

## Scanning the network

From another terminal, crawl the running network and print a report, then exit:

```bash
go run ./p2p/demo scan
```

```
network scan complete - 3 node(s) discovered
  roles:
    validator 2
    archive   1
  client versions:
    sonic-p2p-demo/v0.1.0    3
  block heights:
    79519301     2 node(s)
    79519302     1 node(s)
```

## Useful flags

- `--debug` (global) — verbose logging: discovery, dialing, and handshakes.
  Example: `go run ./p2p/demo --debug validator --id 0`.
- `--report-interval <d>` — how often to log the health table (default `10s`).
- `--peer <multiaddr>` — bootstrap explicitly instead of / in addition to mDNS,
  e.g. for nodes on **different hosts** where mDNS does not reach. Each node
  prints its own `.../p2p/<id>` addresses on startup; pass one to another node:
  `go run ./p2p/demo validator --id 1 --peer /ip4/<host>/udp/<port>/quic-v1/p2p/<id>`.
  Repeatable.
- `scan --wait <d>` — how long to discover peers before crawling (default `5s`).
- `scan --max-nodes <n>` — cap the crawl (default `0` = unbounded).

## Notes

- The validator set has `N = 4` validators (a constant in `validators.go`). Each
  validator's consensus key is derived deterministically from its ID, so every
  process independently computes the same set.
- Validators use ephemeral libp2p host keys (a fresh network identity each run);
  their consensus identity comes from the signed handshake, not the peer ID.
- The chain is faked: block height is a shared ~1 block/sec clock plus a small
  per-node drift, so the health table shows a realistic, mostly-agreeing height.
- Run `go run ./p2p/demo <command> --help` for full usage.
