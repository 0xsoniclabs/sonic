# P2P Network-Layer Protections

Sonic's P2P layer runs on a permissionless, adversarial network. This document
catalogues the protections implemented across the layer as
`threat → mechanism → where implemented`. The `guard` package holds the reusable
primitives; the remaining entries live in the core (`p2p`) and the higher-level
networks (`p2p/networks`).

## Traffic & resource exhaustion

| Threat | Mechanism | Where |
|---|---|---|
| A peer floods messages/bytes | **Per-peer traffic rate limiting** — bytes/sec + msgs/sec token buckets, checked on every framed stream read and every gossip validation | `guard/ratelimiter.go` (`RateLimiter.Check`), enforced in `stream.go` and `node.go` `topicValidator` |
| Sustained traffic abuse | **Disconnect + temporary ban** — a per-peer violation bucket flags sustained abuse; the peer is disconnected and banned for a cooldown | `guard/ratelimiter.go` (violation bucket) → `node.go` `DisconnectAndBan` |
| Oversized messages exhaust memory/CPU | **Per-protocol / per-message-type size caps** — length-delimited framing validates the length prefix and rejects an oversized frame *before* reading its body; each read passes its own cap | `codec.go` (`ReadMessage`), plus a gossip message-size cap in `networks/validatordirectory.go` |
| Too many connections/streams/memory | **Resource limits** — system and per-peer caps on connections, streams, and memory | `guard/resources.go` (libp2p resource manager) |
| Excess connections | **Connection manager** watermarks trim surplus connections | configured in `node.go` |

## Authentication & mesh integrity

| Threat | Mechanism | Where |
|---|---|---|
| A non-validator lingers on the validator mesh, or floods failed handshakes | **Handshake-failure limiting** — every failed validator handshake disconnects the peer; sustained failures additionally ban it for a cooldown (a short burst is tolerated for epoch-boundary membership skew). Scoped to peers that *open the handshake stream*; archives/observers are untouched | `guard/failurelimiter.go` (`FailureLimiter`) → `networks/validatormesh.go` (`HandshakeProtocol.Handle`) → `node.go` `DisconnectAndBan` |
| An impostor claims a validator identity | **Binding-proof handshake** — a peer proves it operates its PeerID by signing `(peerID ‖ epoch ‖ nonce)` with the consensus key; verified against the current membership | `networks/authenticator.go`, `networks/validatormesh.go` |
| Signature reuse across contexts | **Domain separation** — distinct domain tags per signing context (handshake vs. directory advertisement), and length-prefixed digests to remove field-boundary ambiguity | `networks/authenticator.go`, `networks/validatordirectory.go` |

## Gossip / discovery spam

| Threat | Mechanism | Where |
|---|---|---|
| Spam on a gossip topic | **Anti-spam validate gate** — a topic's `Validate` runs before propagation and drops (and score-penalises) invalid messages. For the validator directory: membership + signature + freshness (monotonic sequence) + size checks | `networks/validatordirectory.go`, `networks/archivedirectory.go` (`Validate`) |
| Misbehaving/spamming gossip peers | **gossipsub peer scoring** — penalises invalid/duplicate publishers and greylists low-scoring peers | `guard/scoring.go` |
| Replayed advertisements | **Monotonic sequence numbers** (seeded from wall-clock nanos so they survive restarts); receivers accept only strictly newer advertisements | `networks/validatordirectory.go` |

## Identity & connection gating

| Threat | Mechanism | Where |
|---|---|---|
| Banned/known-bad peers reconnecting | **Connection gater** — allow/deny with a timed (cooldown) ban list, refused at dial and at the secured-connection stage | `guard/gater.go` (`Gater`, `BanUntil`) |
| Validator deanonymization / targeting | **Decoupled, ephemeral identity** — the libp2p host key is per-node and separate from the consensus key (in-memory by default; persisted only for archives), so validator identities are not trivially mapped to network endpoints | `identity.go` |

## Known gaps / future work

- **Spoofed-identity connection floods** (a fresh peer ID per attempt) are bounded
  today only by the resource-manager inbound caps. A deeper defense is an
  `InterceptAccept`-level per-IP connection-rate limit (before the peer ID is
  known). Tracked in `HANDOFF.md`.
