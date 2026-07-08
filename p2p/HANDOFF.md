# Sonic P2P Layer — Integration Hand-off

This package (`./p2p`) is intentionally **not wired into the running node**. It
is self-contained and independently tested. This document describes the
follow-up task: integrating it into `sonicd`, connecting its interfaces to real
node state, and the roadmap of protocols to add next.

## 1. Configuration

Add a `P2P` section to the node-wide config and default it:

- In `config/config.go` (`type Config struct`, around line 74) add a field:
  ```go
  P2P p2p.Config
  ```
- In `config/defaults.go`, initialise it from `p2p.DefaultConfig()`.
- Optionally add CLI flags in `config/flags/flags.go` and map them in
  `config/config.go` (mirroring how `gossip.Config` is handled).

Because the loader (`config.LoadAllConfigs` / `SaveAllConfigs`) uses
`naoina/toml` with `MissingField` errors, the struct field must exist before the
section can appear in a TOML file. Once added, it flows through
`sonictool dumpconfig` automatically.

`p2p.Config` fields worth surfacing: `ListenAddresses`, `BootstrapPeers`,
`HostKeyPath` (set for **archives**; leave empty for validators/observers),
`RateLimit`, `Resources`, `ConnectionManager`.

## 2. Service registration

The `p2p.Node` already exposes `Start() error` / `Stop() error` matching
go-ethereum's `node.Lifecycle`. Wrap and register it beside the gossip service:

- In `config/make_node.go`, near the existing registration block
  (`stack.RegisterProtocols` / `stack.RegisterLifecycle`, ~lines 201–205),
  construct the node and `stack.RegisterLifecycle(p2pService)`, appending its
  teardown to the `cleanup` slice.
- Run it on its **own listen port**, independent of the devp2p `p2p.Server`, so
  the new stack coexists with the existing `"opera"` gossip during the
  transition.

Register the protocols/networks the node should run before `Start`:
`node.RegisterStreamProtocol(...)`, `node.RegisterGossipTopic(...)`.

## 3. Data-source wiring

The networks and the scan protocol depend only on interfaces
(`p2p/networks/sources.go`, `p2p/protocols/scanner.go`). Provide real
implementations:

- **`networks.Membership`** — from epoch state. The current validators and their
  public keys are read today in `readEpochPubKeys`
  (`gossip/checker_helpers.go:122`); expose them as `[]Member{ID, PublicKey}` plus
  an `OnChange` feed on epoch transitions. This is the **only** external input to
  the validator network — addresses are discovered internally by the
  `ValidatorDirectory` gossip topic; callers never supply peer IDs or multiaddrs.
- **`networks.Signer`** (advertisement + binding proof) — back it with
  `valkeystore.SignerAuthority` (`valkeystore/signer.go`), which is already
  unlocked during node start (`config/make_node.go:117-146`). Its `Sign` returns
  the 64-byte R||S the authenticator expects.

### Validator network wiring

The directory, mesh, and handshake are composed by `networks.NewValidatorNetwork`
into one unit. `*p2p.Node` satisfies its `ValidatorNode` interface. Because it
registers a gossip topic and a stream protocol, **construct it before
`node.Start()`, then call `network.Start(ctx)` after**:

```go
vn := networks.NewValidatorNetwork(node, membership, signer,
        networks.NewSecp256k1Verifier(), validatorID, networks.ValidatorNetworkConfig{})
// ... register other protocols ...
node.Start()          // node starts serving; gossip topic + handshake are live
vn.Start(ctx)         // begins advertising this node and maintaining the mesh
// on shutdown: vn.Stop() then node.Stop()
```

`ValidatorNetworkConfig` carries the directory tuning (`Directory`) plus the
handshake-abuse policy (`HandshakeFailures` burst/rate and `HandshakeBanDuration`);
zero values get sensible defaults. A peer that fails the validator handshake is
disconnected, and a sustained flood of failures is banned for the cooldown — this
is scoped to peers that open the handshake stream, so archives/observers that only
gossip are unaffected.

Only validators run a `ValidatorNetwork`; observers/archives do not. Leave
`Config.HostKeyPath` empty for validators (ephemeral key; identity comes from the
signed advertisement + binding proof, not the peer ID).
- **`networks.NodeStatusSource`** — role from the node's configured role, client
  version from the `version` package, block height from the gossip store head.
- **`networks.PeerSource`** — from the P2P host's peerstore / current
  connections.

## 4. Observability

- Pass the node's Prometheus registry into `p2p.New(cfg, log, registerer)`
  instead of the default registerer, so P2P metrics land in the same registry
  exported by `cmd/sonicd/metrics`.
- Pass a named `logger.Logger` (e.g. `logger.New("p2p")`).

## 5. Identity

- **Archives** should set `Config.HostKeyPath` (e.g. `<datadir>/p2p/nodekey`) for
  a stable, cacheable peer ID.
- **Validators** and **observers** should leave it empty (ephemeral in-memory
  key); validators are identified by the binding proof, not the peer ID.

## 6. Roadmap — protocols to add next

Each is an independent follow-up, added via the open/closed registry without
touching the core:

1. **Validator-mesh consensus protocols** — the actual consensus message
   exchange as `StreamProtocol`s over the authenticated mesh.
2. **Finalized-block dissemination** — a `GossipTopic` (via `GossipNetwork`) on
   which validators publish finalized blocks; subscribers verify the aggregated
   BLS `Certificate` (reuse `scc/bls` `AggregatePublicKeys`/`VerifyAll` and the
   `scc/cert` `Certificate[T]` machinery) before re-propagating.
3. **Archive block-history fetch** — a `StreamProtocol` to pull block ranges from
   archives discovered via `ArchiveDirectory` (discovery already exists here;
   the transfer protocol is new).
4. **Standalone network-scan CLI tool** — a small command that stands up an
   observer node, runs `protocols.Scanner`, and prints a network-state report
   (node count, client-version breakdown, sync heights, role counts).
5. **Retire devp2p `opera`** — once the new stack carries production traffic,
   remove the old gossip transport.
6. **`InterceptAccept` per-IP connection-rate limiting** — a deeper defense
   against spoofed-identity connection floods (a fresh peer ID per attempt),
   applied before the peer ID is known. Today such floods are bounded only by the
   resource-manager inbound caps. See `guard/PROTECTIONS.md`.
```
