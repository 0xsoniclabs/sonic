# Sonic P2P Layer — Architecture

This package implements Sonic's libp2p-based peer-to-peer communication layer.
It provides the transport, identity, framing, rate limiting, and
protocol-registration machinery on which the higher-level networks and future
consensus protocols are built. Concrete exchange protocols are added
incrementally; this package ships the framework and one worked example (the
network scan).

> Status: the package is self-contained and independently testable. Wiring it
> into the running node (config, service registration, real data sources) is a
> deliberate follow-up — see [HANDOFF.md](./HANDOFF.md).

## Node roles

The network is permissionless and composed of three roles:

| Role | Purpose | Identity |
|---|---|---|
| **Validator** | Produces consensus. Forms a **full mesh** with all other validators. | Per-node libp2p key **plus** an application-layer *binding proof* signed by the consensus key. |
| **Archive** | Serves historic block data and end-user RPC APIs. **Advertises** itself for discovery. | Per-node libp2p key, typically **persisted** for a stable, cacheable peer ID. |
| **Observer** | Neither validator nor archive (e.g. a node running the network scan). | Ephemeral in-memory libp2p key; a fresh identity each run. |

The layer offers, correspondingly, a fully-meshed validator network, a
gossipsub dissemination network (for finalized blocks — a follow-up protocol),
and an archive-discovery rendezvous.

## Transport & identity

- **Transport:** QUIC is primary (TLS 1.3, stream multiplexing, no
  head-of-line blocking, 1-RTT handshake). TCP + Noise is the fallback for
  QUIC-hostile networks. Both are assembled in `transport` wiring inside
  `node.go`.
- **Identity:** the libp2p **host key is per-node and separate from the
  consensus key** (`identity.go`). It is generated in memory by default, and
  persisted only when `Config.HostKeyPath` is set (archives). This keeps the
  high-value consensus key off the transport surface, allows independent
  rotation, and avoids making validators trivially identifiable on the network.
- **Validator authentication** happens at the application layer: on the
  validator-mesh handshake, a node proves it operates a peer ID by signing
  `(peer_id ‖ epoch ‖ nonce)` with its consensus key (`networks/authenticator.go`).
  Binding to the connection's remote peer and the live epoch defeats replay.

## Protocol architecture — open/closed

The core (`p2p.Node`) is **closed for modification, open for extension**. New
protocols are added by implementing one of two interfaces and registering it —
the core is never edited (`registry.go`):

```
StreamProtocol   // request/response or streaming over a libp2p protocol ID
GossipTopic      // publish/subscribe over gossipsub
```

Registration is done before `Start`:

```go
node.RegisterStreamProtocol(myProtocol) // installs a libp2p stream handler
node.RegisterGossipTopic(myTopic)       // joins + validates + delivers on a topic
```

Every higher-level network is just an implementation of these:

- **ValidatorNetwork** (`networks/validatornetwork.go`) composes the three parts
  below into one unit and is the sole entry point. Its only external input is
  `Membership` (`{ID, PublicKey}` from consensus); addresses, discovery, and
  authentication are internal.
- **ValidatorDirectory** (`networks/validatordirectory.go`) is a `GossipTopic` on
  which validators publish signed `pubkey → PeerID + addresses` advertisements
  (domain-separated, length-prefixed digest; membership/signature/freshness
  gated). It exposes discovered addresses to the mesh as an `AddressResolver`,
  and publishes on join / on new-member discovery / periodically.
- **ValidatorMesh** (`networks/validatormesh.go`) maintains the full mesh,
  dialing each member whose address the directory has resolved and re-dialing as
  membership or known addresses change; the `HandshakeProtocol` is a
  `StreamProtocol` that authenticates peers.
- **GossipNetwork** (`networks/gossipnetwork.go`) turns a validate/deliver pair
  into a registrable `GossipTopic` — the substrate for finalized-block
  dissemination and the archive directory.
- **ArchiveDirectory** (`networks/archivedirectory.go`) is a `GossipTopic`
  implementing the archive-discovery rendezvous.

### Framing & per-message-type size caps

Messages are **protobuf**, length-delimited (`codec.go`). `ReadMessage` and
`WriteMessage` take a `maxSize` **per call**, so each protocol — and each
message type within it — enforces its own cap. The length prefix is validated
*before* the body is read, so an oversized frame never triggers a large
allocation.

### Adversarial protections (`guard/`)

| Guard | Enforces |
|---|---|
| `RateLimiter` | Per-peer **bytes/sec and messages/sec** token buckets, checked on every framed read and every gossip validation; sustained abuse → disconnect + temporary ban. |
| `FailureLimiter` | Per-peer tolerance for repeated failures (e.g. failed validator handshakes); sustained failures → ban. |
| `Gater` | libp2p `ConnectionGater`: allow/deny + timed (cooldown) ban list, rejected at dial/accept. |
| `ResourceManager` | Aggregate & per-peer connection/stream/memory caps. |
| `GossipScoreParams` | gossipsub peer scoring: penalises spam/invalid publishes. |

See **[guard/PROTECTIONS.md](./guard/PROTECTIONS.md)** for the authoritative,
end-to-end catalogue of network-layer protections (threat → mechanism → where
implemented), including the framing size caps, gossip anti-spam gate, signature
domain separation, and identity decoupling that live outside `guard/`.

### Adding a new protocol — worked example (network scan)

The network scan (`protocols/networkscan.go`, `protocols/scanner.go`) is the
template. It is a `StreamProtocol` on `/sonic/scan/1` that reports a node's
role, client version, synced height, and known peers, and a `Scanner` that
crawls the graph and aggregates a `NetworkReport`. Note it couples to the rest
of the node **only** through the injected `NodeStatusSource` and `PeerSource`
interfaces (`networks/sources.go`) — in tests these are fakes/mocks. To add your
own protocol:

1. Define your wire messages in `pb/*.proto` and regenerate (`go generate ./p2p/pb/...`).
2. Implement `StreamProtocol` or `GossipTopic`, taking any node-specific data as
   an injected interface.
3. Register it on the node before `Start`.

Nothing in the core changes.

## Observability

- **Metrics** use the Prometheus `client_golang` API directly (`metrics.go`),
  with best-practice names/units and labels bounded to
  protocol/topic/direction/result — **never** per peer ID. The registerer is
  injectable so the node can supply its own registry.
- **Logging** is via the injected `logger.Logger` interface (mockable), threaded
  to every component. Key events are logged: dialing, connection open/close
  (with a reason), rate-limit exceedance, bans, validator-set reconciliation,
  and handshake errors.
```
