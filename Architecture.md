# Sonic Client Architecture

## Overview

Sonic is an EVM-compatible blockchain client built on a DAG-based consensus protocol
(**Lachesis aBFT**). It combines asynchronous Byzantine Fault Tolerant consensus with
Ethereum-compatible transaction execution, producing a high-throughput, finality-first
Layer 1 chain.

```
Module: github.com/0xsoniclabs/sonic
```

---

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                          CLI Entry Points                           │
│                  cmd/sonicd/         cmd/sonictool/                 │
└──────────────┬──────────────────────────────┬───────────────────────┘
               │                              │
               ▼                              ▼
┌──────────────────────────┐    ┌──────────────────────────────┐
│      Integration         │    │     Tooling (sonictool)      │
│   assembly.go, db.go     │    │  genesis, chain, db, check   │
│  (wires everything up)   │    │  (offline maintenance)       │
└────────────┬─────────────┘    └──────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────────────────────────────┐
│                        Gossip Service                           │
│  ┌────────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────┐   │
│  │  Handler   │  │ Emitter  │  │  Store   │  │  Block Proc  │   │
│  │  (P2P)     │  │ (Events) │  │  (Data)  │  │  (Sealing)   │   │
│  └─────┬──────┘  └────┬─────┘  └────┬─────┘  └──────┬───────┘   │
│        │              │             │               │           │
│  ┌─────┴──────────────┴─────────────┴───────────────┘           │
│  │  Protocols (DAG stream leecher/seeder), Topology,            │
│  │  Filters, Gas Price Oracle, RANDAO, EVM Store                │
│  └──────────────────────────────────────────────────────────────│
└──────────────────────────────┬──────────────────────────────────┘
                               │
          ┌────────────────────┼────────────────────┐
          ▼                    ▼                    ▼
┌──────────────────┐ ┌──────────────────┐ ┌──────────────────────┐
│  Lachesis aBFT   │ │    EVM Core      │ │    Ethereum API      │
│  (Consensus)     │ │  (Tx Execution)  │ │  (JSON-RPC / ethapi) │
│  abft.Lachesis   │ │  StateProcessor  │ │  Backend interface   │
└────────┬─────────┘ │  TxPool          │ └──────────────────────┘
         │           └──────────────────┘
         ▼
┌──────────────────┐ ┌──────────────────┐ ┌──────────────────────┐
│  Vector Clock    │ │  Carmen StateDB  │ │   Event Validation   │
│  (vecmt)         │ │  (evmstore)      │ │   (eventcheck)       │
└──────────────────┘ └──────────────────┘ └──────────────────────┘
```

---

## Data Flow: Transaction to Finality

```
  User Tx (RPC)
       │
       ▼
  ┌─────────┐     ┌──────────┐     ┌──────────────┐     ┌────────────┐
  │ TxPool  │────▶│ Emitter  │────▶│  DAG Event   │────▶│  P2P Gossip│
  │(evmcore)│     │(emitter) │     │  (inter pkg) │     │  (handler) │
  └─────────┘     └──────────┘     └──────────────┘     └─────┬──────┘
                                                              │
                                          ┌───────────────────┘
                                          ▼
                                   ┌──────────────┐
                                   │  Lachesis    │
                                   │  Consensus   │
                                   │  (aBFT DAG)  │
                                   └──────┬───────┘
                                          │ Atropos decided
                                          ▼
                                   ┌──────────────┐     ┌──────────────┐
                                   │  Block Proc  │────▶│ EVM State    │
                                   │  (sealing)   │     │ Processor    │
                                   └──────────────┘     └──────────────┘
                                                              │
                                                              ▼
                                                        Finalized Block
```

---

## Directory Structure and Components

### `cmd/` -- CLI Entry Points

Two main binaries:

| Binary | Purpose |
|--------|---------|
| `sonicd` | Full node daemon -- runs the Sonic network node |
| `sonictool` | Offline maintenance tool -- genesis, import/export, DB heal |

```
cmd/
├── sonicd/
│   ├── app/
│   │   ├── launcher.go        # Main CLI app, flag registration, node startup
│   │   ├── app.go             # Node lifecycle (make/start/stop)
│   │   ├── accounts.go        # Account unlock helpers
│   │   ├── misccmd.go         # version, license subcommands
│   │   └── usage.go           # CLI usage formatting
│   ├── diskusage/
│   │   └── monitor.go         # Disk usage monitoring
│   └── metrics/
│       ├── flags.go           # Metrics CLI flags
│       └── disksize.go        # Disk size metrics
├── sonictool/
│   ├── app/
│   │   ├── main.go            # sonictool entry point
│   │   ├── cli.go             # JS console attach
│   │   ├── genesis.go         # Genesis import/export commands
│   │   ├── chain.go           # Chain event import/export
│   │   ├── compact.go         # DB compaction
│   │   ├── heal.go            # DB healing
│   │   ├── check.go           # DB consistency checks
│   │   ├── validator.go       # Validator management
│   │   └── account.go         # Account management
│   ├── chain/
│   │   ├── export_events.go   # Event export logic
│   │   └── import_events.go   # Event import logic
│   ├── db/
│   │   ├── heal.go            # Database healing
│   │   └── dbutils.go         # DB utility functions
│   ├── genesis/
│   │   ├── import.go          # Genesis import
│   │   ├── export.go          # Genesis export
│   │   ├── signature.go       # Genesis signature verification
│   │   └── allowed.go         # Allowed genesis hashes
│   └── check/
│       ├── live.go            # Live DB checks
│       ├── archive.go         # Archive DB checks
│       └── common.go          # Shared check utilities
└── cmdtest/
    └── test_cmd.go            # Test helpers for CLI testing
```

---

### `integration/` -- Assembly Layer

Wires together all major subsystems (gossip, consensus, vector clock, stores).
This is the "main()" of the node logic.

```
integration/
├── assembly.go           # rawMakeEngine() / makeEngine() -- creates Lachesis + VecClock + GossipStore
├── db.go                 # DB producer setup (LevelDB/PebbleDB)
├── metric.go             # Metrics for DB operations
├── status.go             # Node status reporting
├── makefakegenesis/      # Fake genesis for testing
└── makegenesis/          # Real genesis construction
```

Key wiring in `assembly.go`:
```go
type Configs struct {
    Opera         gossip.Config
    OperaStore    gossip.StoreConfig
    Lachesis      abft.Config
    LachesisStore abft.StoreConfig
    VectorClock   vecmt.IndexConfig
    DBs           DBsConfig
}
```

---

### `gossip/` -- Core Network Service

The largest package. Contains the P2P protocol handler, event emitter, data store,
block processing pipeline, and all sub-protocols.

```
gossip/
├── handler.go              # Main P2P protocol handler (struct handler)
├── handler_dag.go          # DAG event handling
├── handler_event.go        # Individual event handling
├── handler_peerinfo.go     # Peer info exchange
├── handler_progress.go     # Sync progress broadcasting
├── handler_stream.go       # Event streaming handlers
├── handler_tx.go           # Transaction propagation
├── protocol.go             # Protocol constants, message codes (Sonic_66)
├── config.go               # Config, ProtocolConfig, StoreConfig
├── sync.go                 # Sync loop and tx sync
├── peer.go                 # Peer representation
├── peerset.go              # Peer set management
├── p2putil.go              # P2P utility functions
├── pbcodec.go              # Protobuf codec
├── store.go                # Main gossip Store (events, blocks, epochs)
├── store_event.go          # Event storage operations
├── api.go                  # Service API registration
├── ethapi_backend.go       # ethapi.Backend implementation
├── evm_state_reader.go     # EVM state reader adapter
├── emitter_world.go        # Emitter <-> gossip world adapter
├── discover.go             # Peer discovery
├── apply_genesis.go        # Genesis application
├── c_block_callbacks.go    # Block consensus callbacks
├── c_event_callbacks.go    # Event consensus callbacks
├── c_llr_callbacks.go      # LLR (Lightweight Lachesis Reprocessing) callbacks
├── checker_helpers.go      # Event check helper functions
│
├── emitter/                # Event Emitter subsystem
│   ├── emitter.go          # Emitter struct -- creates DAG events from pending txs
│   ├── config/
│   │   └── config.go       # Emitter configuration
│   ├── control.go          # Emitter start/stop lifecycle
│   ├── hooks.go            # Emitter event hooks (callbacks)
│   ├── ordering.go         # Event ordering logic
│   ├── parents.go          # Parent event selection
│   ├── piecefuncs.go       # Piece functions for event building
│   ├── proposals.go        # Block proposal handling
│   ├── prev_action_files.go # Previous action state persistence
│   ├── sync.go             # Emitter sync state
│   ├── txs.go              # Transaction selection for events
│   ├── validators.go       # Validator-related emitter logic
│   ├── world.go            # World interface (emitter's view of the node)
│   ├── scheduler/
│   │   ├── scheduler.go    # Event emission scheduling
│   │   └── processor.go    # Scheduled event processing
│   ├── throttler/
│   │   ├── throttler.go    # Event emission throttling
│   │   └── dominant_set.go # Dominant set calculation
│   └── originatedtxs/
│       └── txs_ring_buffer.go  # Ring buffer for originated txs
│
├── blockproc/              # Block Processing Pipeline
│   ├── interface.go        # BlockProc interfaces (EVM/Event/Sealer modules)
│   ├── drivermodule/
│   │   └── driver_txs.go   # Internal driver transactions
│   ├── eventmodule/
│   │   └── confirmed_events_processor.go  # Processes confirmed events
│   ├── evmmodule/
│   │   └── evm.go          # EVM execution module
│   ├── sealmodule/
│   │   └── sealer.go       # Block sealer
│   ├── verwatcher/
│   │   ├── version_watcher.go        # Network version monitoring
│   │   ├── store.go                  # Version watcher store
│   │   ├── store_network_version.go  # Network version persistence
│   │   └── version_number.go         # Version number handling
│   ├── bundle/
│   │   ├── bundle.go       # Transaction bundle types
│   │   ├── builder.go      # Bundle builder
│   │   ├── validate.go     # Bundle validation
│   │   └── execution_flags.go  # Bundle execution flags
│   └── subsidies/
│       ├── subsidies.go    # Gas subsidy processing
│       ├── proxy/          # Subsidy proxy contract
│       └── registry/       # Subsidy registry contract
│
├── evmstore/               # EVM State Storage (Carmen)
│   ├── store.go            # Store struct -- wraps Carmen state DB
│   ├── config.go           # Store configuration
│   ├── carmen.go           # Carmen DB initialization
│   ├── statedb.go          # StateDB access
│   ├── statedb_import.go   # State import for genesis
│   ├── statedb_verify.go   # State verification
│   ├── statedb_logger.go   # StateDB logging
│   ├── apply_genesis.go    # Genesis state application
│   ├── store_block_cache.go    # Block state cache
│   ├── store_receipts.go       # Receipt storage
│   ├── store_tx.go             # Transaction storage
│   └── store_tx_position.go    # Tx position index
│
├── protocols/              # P2P Sub-Protocols
│   └── dag/
│       └── dagstream/
│           ├── types.go                # DAG stream types
│           ├── dagstreamleecher/
│           │   ├── leecher.go          # Downloads events from peers
│           │   └── config.go           # Leecher configuration
│           └── dagstreamseeder/
│               ├── seeder.go           # Serves events to peers
│               └── config.go           # Seeder configuration
│
├── filters/                # Log/Event Filtering (eth_getLogs)
│   ├── filter.go           # Filter logic
│   ├── filter_system.go    # Filter subscription system
│   └── api.go              # Filter API
│
├── gasprice/               # Gas Price Oracle
├── topology/
│   └── connection_advisor.go   # Peer connection topology advisor
├── randao/                 # RANDAO implementation
├── scrambler/              # Event scrambling
├── proclogger/             # Processing logger
├── contract/               # Pre-deployed contract ABIs
│   ├── sfc100/             # SFC (Staking) contract
│   ├── driver100/          # Driver contract
│   ├── driverauth100/      # Driver auth contract
│   └── netinit100/         # Network initializer contract
└── pb/                     # Protobuf message definitions
```

#### Key Structs

| Struct | File | Purpose |
|--------|------|---------|
| `handler` | `gossip/handler.go` | Main P2P protocol manager |
| `Store` | `gossip/store.go` | Persistent storage (events, blocks, epochs) |
| `Emitter` | `gossip/emitter/emitter.go` | Creates DAG events from pending txs |
| `Store` (evm) | `gossip/evmstore/store.go` | Carmen-backed EVM state store |
| `BlockProc` | `gossip/blockproc/interface.go` | Block processing pipeline interfaces |

---

### `opera/` -- Network Rules and Genesis

Defines the chain rules, hard-fork upgrades, VM configuration, genesis format,
and pre-deployed system contracts.

```
opera/
├── rules.go                # Rules, DagRules, EpochsRules, EconomyRules, Upgrades
├── vm_config.go            # EVM/VM configuration
├── validate.go             # Rules validation
├── marshal.go              # Rules serialization
├── legacy_serialization.go # Legacy format support
├── contracts/
│   ├── driver/             # Driver contract (epoch sealing, validator updates)
│   │   ├── driver_predeploy.go
│   │   ├── drivercall/     # Driver contract call helpers
│   │   └── driverpos/      # Driver contract storage positions
│   ├── driverauth/         # Driver authorization contract
│   ├── emitterdriver/      # Emitter-driver integration
│   ├── evmwriter/          # EVM writer precompile
│   ├── netinit/            # Network initializer contract
│   └── sfc/                # SFC (Special Fee Contract) predeploy
├── genesis/
│   ├── types.go            # Genesis data types
│   └── gpos/               # gPOS validator genesis
└── genesisstore/
    ├── store.go            # Genesis store
    ├── store_genesis.go    # Genesis store operations
    ├── disk.go             # Disk-based genesis storage
    ├── fileshash/          # File content hashing
    ├── filelog/            # File-based logging
    └── readersmap/         # Reader map utilities
```

#### Key Types

```go
type Rules struct {
    Name      string
    NetworkID uint64
    Dag       DagRules
    Emitter   EmitterRules
    Epochs    EpochsRules
    Blocks    BlocksRules
    Economy   EconomyRules
    Upgrades  Upgrades        // Hard fork flags: Berlin, London, LLR, Sonic, Allegro, Brio
}
```

Hard-fork progression: `Berlin -> London -> LLR -> Sonic -> Allegro -> Brio`

---

### `inter/` -- Shared Data Types

Core inter-node data types used across the codebase. Defines events, proposals,
block/epoch state, and validator profiles.

```
inter/
├── event.go                # EventI, EventPayloadI interfaces; Event, EventPayload structs
├── event_serializer.go     # Event RLP serialization
├── ordering.go             # Event ordering rules
├── proposal.go             # Block proposals
├── proposal_sync_state.go  # Proposal sync state, EventReader interface
├── gas_power.go            # Gas power calculations
├── drivertype/             # Driver type definitions
├── iblockproc/
│   └── decided_state.go    # BlockState, EpochState, ValidatorBlockState
├── ibr/                    # Inter-Block Records
├── iep/                    # Inter-Epoch Packs
├── ier/                    # Inter-Epoch Records
├── state/
│   └── adapter.go          # StateDB interface
├── validatorpk/            # Validator public key types
└── pb/                     # Protobuf definitions
```

---

### `evmcore/` -- EVM Execution Layer

Fork of go-ethereum's core package, adapted for Sonic's DAG-based block production.
Handles transaction execution, transaction pool, and state transitions.

```
evmcore/
├── state_processor.go      # StateProcessor -- executes txs against state
├── tx_pool.go              # Transaction pool (pending/queued management)
├── tx_validation.go        # Transaction validation rules
├── tx_list.go              # Sorted tx lists by nonce/price
├── tx_cacher.go            # Tx sender caching
├── tx_journal.go           # Tx journal (persistence)
├── tx_noncer.go            # Nonce tracking
├── evm.go                  # EVM creation helper
├── dummy_block.go          # EvmBlock/EvmHeader (Sonic's block types)
├── types.go                # Core type definitions
├── notify.go               # NewTxsNotify event types
├── apply_fake_genesis.go   # Fake genesis for testing
├── subsidies_integration.go        # Gas subsidies integration
├── subsidies_check_cache.go        # Subsidies check caching
└── core_types/             # Shared core types
```

---

### `ethapi/` -- Ethereum JSON-RPC API

Full Ethereum-compatible API layer. Implements all standard `eth_`, `debug_`,
`txpool_`, `net_` namespaces plus Sonic-specific APIs.

```
ethapi/
├── api.go                  # PublicEthereumAPI, PublicBlockChainAPI, PublicTransactionPoolAPI,
│                           # PublicAccountAPI, PrivateAccountAPI, PublicDebugAPI, PrivateDebugAPI,
│                           # PublicNetAPI
├── backend.go              # Backend interface -- abstraction over the full node
├── dag_api.go              # PublicDAGChainAPI (Sonic DAG queries)
├── abft_api.go             # PublicAbftAPI (consensus queries)
├── sonic_api.go            # PublicSccApi (Sonic Certification Chain API)
├── tx_trace.go             # PublicTxTraceAPI (transaction tracing)
├── block_overrides.go      # eth_call block overrides
├── simulate.go             # eth_simulateV1 implementation
├── transaction_args.go     # Transaction argument parsing
├── block_cert_json.go      # Block certificate JSON format
├── committee_cert_json.go  # Committee certificate JSON format
├── limit.go                # JSON result buffer limiting
├── addrlock.go             # Address-level locking
├── log_tracer.go           # Log-based EVM tracer
└── errors.go               # API error types
```

---

### `eventcheck/` -- Event Validation Pipeline

Multi-stage event validation. Each checker handles a different aspect:

```
eventcheck/
├── all.go                  # Checkers struct -- aggregates all checkers
├── basiccheck/
│   └── basic_check.go      # Structural validation (fields, sizes)
├── epochcheck/
│   └── epoch_check.go      # Epoch boundary validation
├── parentscheck/
│   └── parents_check.go    # Parent event consistency
├── parentlesscheck/
│   └── parentless_check.go # Parentless event validation
├── gaspowercheck/
│   └── gas_power_check.go  # Gas power budget validation
├── heavycheck/
│   ├── heavy_check.go      # Signature verification (expensive)
│   └── config.go           # Heavy check configuration
└── proposalcheck/
    └── proposal_check.go   # Block proposal validation
```

Validation order: `basic -> epoch -> parents -> gaspower -> heavy (sig verify) -> proposal`

---

### `scc/` -- Sonic Certification Chain

BLS-based light client protocol for cross-chain verification.

```
scc/
├── committee.go            # Committee struct (BLS public keys + weights)
├── member.go               # Committee member
├── bls/
│   └── bls.go              # BLS key types (PrivateKey, PublicKey, Signature)
├── cert/
│   ├── statement.go        # Statement interface, BlockStatement, CommitteeStatement
│   ├── bitset.go           # Signer bitset for aggregated signatures
│   ├── serialize.go        # Certificate serialization
│   └── pb/                 # Protobuf: BlockCertificate, CommitteeCertificate
├── node/
│   ├── node.go             # SCC Node struct
│   ├── store.go            # SCC Store interface
│   └── state.go            # SCC State interface
└── light_client/
    ├── light_client.go     # LightClient struct
    ├── light_client_state.go  # Light client state tracking
    ├── server.go           # Light client RPC server
    ├── client.go           # RPC client interface
    ├── provider.go         # Data provider interface
    ├── multiplexer.go      # Multi-provider multiplexer
    └── retry.go            # Retry logic
```

---

### `vecmt/` -- Vector Clock (Median Time)

DAG vector clock implementation used by Lachesis consensus for
causal ordering and median timestamp calculation.

```
vecmt/
├── index.go            # Index struct -- main vector clock index
├── index_test.go       # Tests
├── vector_ops.go       # CreationTimer interface, vector operations
├── median_time.go      # Median time calculation from DAG
├── backed_map.go       # Disk-backed map for vector data
└── vecflushable.go     # Flushable vector storage
```

---

### `config/` -- Node Configuration

```
config/
├── config.go           # Config struct -- top-level node configuration
└── flags/              # CLI flag definitions
```

---

### `valkeystore/` -- Validator Key Store

Manages validator private keys with file-based encrypted storage.

```
valkeystore/
├── keystore.go         # RawKeystoreI, KeystoreI interfaces
├── signer.go           # SignerAuthority interface
├── files.go            # FileKeystore -- file-based key storage
├── mem.go              # MemKeystore -- in-memory (testing)
├── cache.go            # CachedKeystore -- with caching
├── sync.go             # SyncedKeystore -- thread-safe wrapper
└── encryption/
    ├── encryption.go   # Key encryption/decryption
    └── migration.go    # Key format migration
```

---

### `topicsdb/` -- Log Topics Index

Efficient log event indexing using a leap-join algorithm for `eth_getLogs`.

```
topicsdb/
├── topicsdb.go         # Index interface
├── index.go            # index struct implementation
├── leap_join.go        # Leap-join algorithm for topic matching
└── dummy.go            # No-op implementation (when indexing disabled)
```

---

### `txtrace/` -- Transaction Tracing

```
txtrace/
└── ...                 # Transaction tracing utilities
```

---

### `utils/` -- Utility Packages

```
utils/
├── adapters/           # Adapter types (vecmt2dagidx)
├── bits/               # Bit manipulation
├── caution/            # Error handling helpers
├── concurrent/         # Concurrent data structures
├── cser/               # Compact serialization
├── dbutil/             # Database utilities
├── devnullfile/        # /dev/null file implementation
├── errlock/            # Error locking
├── eventid/            # Event ID cache
├── fast/               # Fast math utilities
├── iodb/               # IO database utilities
├── ioread/             # IO read helpers
├── jsonhex/            # JSON hex encoding
├── leap/               # Leap algorithm utilities
├── memory/             # Memory utilities
├── migration/          # Data migration helpers
├── objstream/          # Object streaming
├── prompt/             # User prompts
├── result/             # Result type
├── rlpstore/           # RLP store helpers
├── signers/            # Transaction signers
├── txtime/             # Transaction timestamp tracking
└── wgmutex/            # WaitGroup-based mutex
```

---

## External Dependencies

| Dependency | Purpose |
|------------|---------|
| `github.com/Fantom-foundation/lachesis-base` | Lachesis aBFT consensus engine, DAG primitives, KV store abstractions |
| `github.com/ethereum/go-ethereum` | go-ethereum: EVM, p2p networking, RPC, accounts, crypto |
| `github.com/0xsoniclabs/carmen` | Carmen: high-performance state database for EVM storage |

---

## Component Interaction Diagram

```
                    ┌─────────────┐
                    │   sonicd    │
                    │  (CLI app)  │
                    └──────┬──────┘
                           │ creates
                           ▼
                    ┌─────────────┐
                    │ Integration │──────────────────────────────────┐
                    │  Assembly   │                                  │
                    └──────┬──────┘                                  │
                           │ creates                                 │ creates
              ┌────────────┼────────────────┐                        │
              ▼            ▼                ▼                        ▼
     ┌──────────────┐ ┌────────┐  ┌───────────────┐     ┌──────────────┐
     │ gossip.Store │ │Lachesis│  │  vecmt.Index  │     │ abft.Store   │
     │              │ │ (aBFT) │  │ (Vector Clock)│     │              │
     └───────┬──────┘ └───┬────┘  └───────────────┘     └──────────────┘
             │            │
             │            │ consensus decisions
             ▼            ▼
     ┌──────────────────────────┐
     │     gossip.handler       │◀──── P2P Network
     │  (protocol manager)      │
     ├──────────────────────────┤
     │  dagLeecher / dagSeeder  │ ── event streaming
     │  dagProcessor            │ ── DAG event processing
     │  dagFetcher / txFetcher  │ ── item fetching
     │  peers (peerSet)         │ ── peer management
     │  connectionAdvisor       │ ── topology optimization
     └───────────┬──────────────┘
                 │
        ┌────────┼────────┐
        ▼        │        ▼
  ┌──────────┐   │   ┌──────────────┐
  │ Emitter  │   │   │  eventcheck  │
  │          │   │   │  (validation)│
  │ Creates  │   │   └──────────────┘
  │  events  │   │
  │ from txs │   │
  └──────────┘   │
                 ▼
          ┌──────────────┐
          │  blockproc   │
          │  (block      │
          │  processing) │
          ├──────────────┤
          │ eventmodule  │ ── confirmed events
          │ evmmodule    │ ── EVM execution ──▶ evmcore.StateProcessor
          │ drivermodule │ ── internal txs     ──▶ Carmen StateDB
          │ sealmodule   │ ── block sealing
          │ bundle       │ ── tx bundles
          │ subsidies    │ ── gas subsidies
          └──────────────┘
```

---

## P2P Protocol

Protocol name: `opera` (version 66, protobuf wire encoding)

| Message Code | Name | Direction | Purpose |
|-------------|------|-----------|---------|
| 0 | `HandshakeMsg` | Bidirectional | Initial peer handshake |
| 1 | `ProgressMsg` | Broadcast | Sync progress status |
| 2 | `EvmTxsMsg` | Push | Transaction propagation |
| 3 | `NewEvmTxHashesMsg` | Push | New transaction hash announcement |
| 4 | `GetEvmTxsMsg` | Request | Request transactions by hash |
| 5 | `NewEventIDsMsg` | Push | New event ID announcement |
| 6 | `GetEventsMsg` | Request | Request events by ID |
| 7 | `EventsMsg` | Response/Push | Event batch delivery |
| 8 | `RequestEventsStream` | Request | Stream events by selector |
| 9 | `EventsStreamResponse` | Response | Streamed events response |
| 10 | `GetPeerInfosMsg` | Request | Request known peer info |
| 11 | `PeerInfosMsg` | Response | Known peer information |
| 12 | `GetEndPointMsg` | Request | Request peer endpoint |
| 13 | `EndPointUpdateMsg` | Response | Peer endpoint update |

---

## Consensus Model

Sonic uses **Lachesis aBFT** (from `lachesis-base`), a leaderless, asynchronous
Byzantine Fault Tolerant consensus protocol:

1. **Events** -- Validators create DAG events containing transactions and parent references
2. **DAG** -- Events form a Directed Acyclic Graph through parent links
3. **Atropos** -- The consensus algorithm elects "Atropos" events that define finality
4. **Epochs** -- The validator set is updated at epoch boundaries
5. **Blocks** -- Finalized events are ordered and sealed into EVM-compatible blocks

```
  Validator A:  ●────●────●────●────●
                 ╲  ╱ ╲  ╱      ╲
  Validator B:    ●────●────●────●────●
                 ╱  ╲    ╱ ╲      ╲
  Validator C:  ●────●────●────●────●
                              ▲
                          Atropos
                       (finality point)
```

---

## Storage Architecture

### Overview

Sonic uses a multi-layer storage architecture with three distinct store types,
all coordinated through a single `kvdb.FlushableDBProducer` that manages
underlying PebbleDB instances.

```
┌──────────────────────────────────────────────────────────────────────────┐
│                        integration/db.go                                 │
│                     (DB Producer Pipeline)                               │
│                                                                          │
│  PebbleDB ──▶ dbcounter ──▶ flaggedproducer ──▶ cachedproducer ──▶ skipkeys │
│  (raw disk)   (metrics)     (flush IDs)         (DB caching)     (metadata) │
└──────────────────────┬───────────────────────────────────────────────────┘
                       │ kvdb.FullDBProducer
          ┌────────────┼──────────────────┐
          ▼            ▼                  ▼
   ┌─────────────┐ ┌────────────┐ ┌──────────────────┐
   │gossip.Store │ │abft.Store  │ │ Per-Epoch Stores  │
   │ DB: "gossip"│ │DB:"lachesis"│ │DB:"gossip-{N}"   │
   │             │ │            │ │DB:"lachesis-{N}"  │
   │ ┌─────────┐ │ └────────────┘ └──────────────────┘
   │ │evmstore │ │
   │ │ (shares │ │
   │ │ gossip  │ │
   │ │  DB)    │ │
   │ └─────────┘ │
   └─────────────┘
          │
          ▼ (separate filesystem)
   ┌──────────────┐
   │  Carmen      │
   │  StateDB     │
   │ (EVM state)  │
   └──────────────┘
```

### On-Disk Layout

A running Sonic node produces the following data directory structure:

```
db/
├── chaindata/                          # PebbleDB databases
│   ├── gossip/                         # Main gossip store (events, blocks, state)
│   │   ├── *.sst                       # PebbleDB sorted string tables
│   │   ├── *.log                       # PebbleDB write-ahead log
│   │   ├── MANIFEST-*                  # PebbleDB manifest
│   │   └── CURRENT, OPTIONS-*, LOCK
│   ├── gossip-{epoch}/                 # Per-epoch gossip data (heads, last events, DAG index)
│   │   └── ...                         # Dropped when epoch advances
│   ├── lachesis/                       # Main Lachesis consensus state
│   │   └── ...
│   └── lachesis-{epoch}/               # Per-epoch Lachesis consensus data
│       └── ...
├── carmen/                             # Carmen EVM state (separate from PebbleDB)
│   └── live/                           # Live state trie
│       ├── meta.json                   # Carmen metadata
│       ├── forest.json                 # MPT forest config
│       ├── codes.dat                   # Contract bytecodes
│       ├── accounts/                   # Account state (balance, nonce, codehash)
│       │   ├── values.dat, freelist.dat, meta.json
│       ├── values/                     # Storage slot values
│       │   ├── values.dat, freelist.dat, meta.json
│       ├── branches/                   # MPT branch nodes
│       │   └── ...
│       └── extensions/                 # MPT extension nodes
│           └── ...
├── p2p/                                # Peer-to-peer node data
│   ├── nodekey                         # Node private key
│   └── nodes/                          # LevelDB peer database
│       └── ...
├── keystore/                           # Validator encrypted keys
└── transactions.rlp                    # Transaction journal (tx pool persistence)
```

---

### DB Producer Pipeline (`integration/db.go`)

The database producer is assembled as a decorator chain:

```go
// 1. Raw PebbleDB producer
raw := pebble.NewProducer(chaindataDir, cacher)

// 2. Wrap with read/write counters for metrics
raw = dbcounter.Wrap(raw, true)

// 3. Scoped producer with flush ID tracking (atomicity)
scoped := flaggedproducer.Wrap(raw, FlushIDKey)

// 4. DB instance caching (reuse open DB handles)
cached := cachedproducer.WrapAll(scoped)

// 5. Skip metadata keys (hide internal keys from application)
final := skipkeys.WrapAllProducer(cached, MetadataPrefix)
```

The `FlushIDKey` and `MetadataPrefix` are internal markers stored in every DB
to track flush consistency. `MetadataPrefix` is a 76-byte constant; the
application never sees these keys.

**Flush mechanism**: The `gossip.Store` periodically commits all dirty data
via `dbs.Flush(flushID)`. The flush is triggered when either:
- Time since last flush exceeds `MaxNonFlushedPeriod` (default: 30 min, randomized ~90-100%)
- Estimated dirty data exceeds `MaxNonFlushedSize` (default: ~21-23 MiB, randomized)

The randomization (per-epoch) desynchronizes flushes across validators.

---

### gossip.Store -- Main Chain Data

Opened as the `"gossip"` database. Uses a **table prefix** pattern where each
logical table gets a single-byte prefix prepended to all its keys.

```go
// gossip/store.go -- table struct tags define the prefix bytes
table struct {
    Version                kvdb.Store `table:"_"`   // prefix '_'
    BlockEpochState        kvdb.Store `table:"D"`   // prefix 'D'
    BlockEpochStateHistory kvdb.Store `table:"h"`   // prefix 'h'
    Events                 kvdb.Store `table:"e"`   // prefix 'e'
    Blocks                 kvdb.Store `table:"b"`   // prefix 'b'
    EpochBlocks            kvdb.Store `table:"P"`   // prefix 'P'
    Genesis                kvdb.Store `table:"g"`   // prefix 'g'
    UpgradeHeights         kvdb.Store `table:"U"`   // prefix 'U'
    CommitteeCertificates  kvdb.Store `table:"C"`   // prefix 'C'
    BlockCertificates      kvdb.Store `table:"c"`   // prefix 'c'
    HighestLamport         kvdb.Store `table:"l"`   // prefix 'l'
    NetworkVersion         kvdb.Store `table:"V"`   // prefix 'V'
    BlockHashes            kvdb.Store `table:"B"`   // prefix 'B'
}
```

The `table.MigrateTables()` utility from lachesis-base uses reflection to read
the struct tags and wraps the underlying `kvdb.Store` so that every `Put`/`Get`
automatically prepends the prefix byte.

#### Table Details

| Prefix | Table | Key | Value | Description |
|--------|-------|-----|-------|-------------|
| `_` | Version | migration ID string | version bytes | DB migration version tracking |
| `D` | BlockEpochState | `"s"` (singleton) | RLP(`BlockEpochState`) | Current block + epoch state (latest) |
| `h` | BlockEpochStateHistory | `epoch.Bytes()` (4B big-endian) | RLP(`BlockEpochState`) | Historical block+epoch state per epoch |
| `e` | Events | `event.ID.Bytes()` (32B = epoch ++ lamport ++ hash) | RLP(`EventPayload`) | Full DAG events with transactions |
| `b` | Blocks | `block.Bytes()` (8B big-endian) | RLP(`inter.Block`) | Sealed blocks (hashes, time, gas, epoch) |
| `P` | EpochBlocks | `(MaxUint64 - block).Bytes()` (inverted) | `epoch.Bytes()` | Block-to-epoch reverse lookup (inverted for efficient latest-first iteration) |
| `g` | Genesis | `"g"` | genesis hash (32B) | Genesis identification |
| `g` | Genesis | `"i"` | `block.Bytes()` | Genesis block index |
| `U` | UpgradeHeights | `[]byte{}` (singleton) | RLP(`[]UpgradeHeight`) | Hard fork activation heights + times |
| `C` | CommitteeCertificates | `period` (8B big-endian) | serialized `Certificate` | SCC committee certificates |
| `c` | BlockCertificates | `block` (8B big-endian) | serialized `Certificate` | SCC block certificates |
| `l` | HighestLamport | `"k"` (singleton) | `lamport.Bytes()` (4B) | Highest observed Lamport timestamp |
| `V` | NetworkVersion | `"v"` | `uint64` (8B big-endian) | Network protocol version |
| `V` | NetworkVersion | `"m"` | `uint64` (8B big-endian) | Missed (unsupported) network version |
| `B` | BlockHashes | `hash.Bytes()` (32B) | `block.Bytes()` (8B) | Block hash to block number index |

#### Event Key Schema

Events are keyed by their `hash.Event` (32 bytes), which encodes:

```
+----------+----------+------------------------+
| Epoch    | Lamport  | Hash (remaining bytes)  |
| (4 bytes)| (4 bytes)| (24 bytes)              |
+----------+----------+------------------------+
```

This layout enables efficient iteration by epoch (`NewIterator(epoch.Bytes(), nil)`)
and by epoch+lamport prefix for event lookups.

#### LRU Caches

Every table has an associated in-memory LRU cache:

| Cache | Type | Default Size | Key | Value |
|-------|------|-------------|-----|-------|
| Events | weighted LRU | 5000 items / 6 MiB | `hash.Event` | `*EventPayload` (pointer) |
| EventsHeaders | weighted LRU | 5000 items | `hash.Event` | `*Event` (pointer) |
| EventIDs | eventid.Cache | 100,000 items | `hash.Event` | existence flag |
| Blocks | weighted LRU | varies | `idx.Block` | `*inter.Block` (pointer) |
| BlockHashes | weighted LRU | same as blocks | `common.Hash` | `idx.Block` (value) |
| BRHashes | weighted LRU | same as blocks | key | hash (value) |
| BlockEpochStateHistory | weighted LRU | varies | `idx.Epoch` | `*BlockEpochState` (pointer) |
| BlockEpochState | atomic.Value | 1 (singleton) | -- | `*BlockEpochState` (value) |
| HighestLamport | atomic.Value | 1 (singleton) | -- | `idx.Lamport` (value) |
| UpgradeHeights | atomic.Value | 1 (singleton) | -- | `[]UpgradeHeight` (pointer) |
| Genesis | atomic.Value | 1 (singleton) | -- | `hash.Hash` (value) |

---

### Per-Epoch Store (`gossip/store_epoch.go`)

Each epoch gets a dedicated PebbleDB database named `"gossip-{epoch}"`.
When the epoch advances, the old epoch DB is dropped.

```go
// Per-epoch table prefixes
table struct {
    LastEvents kvdb.Store `table:"t"`   // prefix 't'
    Heads      kvdb.Store `table:"H"`   // prefix 'H'
    DagIndex   kvdb.Store `table:"v"`   // prefix 'v'
}
```

| Prefix | Table | Key | Value | Description |
|--------|-------|-----|-------|-------------|
| `t` | LastEvents | `[]byte{}` (singleton) | packed `[validatorID(4B) ++ eventHash(32B)]...` | Last event per validator in current epoch |
| `H` | Heads | `[]byte{}` (singleton) | packed `[eventHash(32B)]...` | DAG head events (no descendants) |
| `v` | DagIndex | varies | vector clock data | Lachesis DAG indexing data |

Both Heads and LastEvents are stored as a single concatenated byte blob
(sorted for determinism) and cached in `atomic.Value`.

---

### evmstore.Store -- EVM Index Data

Shares the same `"gossip"` PebbleDB as gossip.Store (passed via `mainDB`).
Has its own table prefixes for EVM-specific index data.

```go
// gossip/evmstore/store.go
table struct {
    Receipts    kvdb.Store `table:"r"`   // prefix 'r'
    TxPositions kvdb.Store `table:"x"`   // prefix 'x'
    Txs         kvdb.Store `table:"X"`   // prefix 'X'
}
```

| Prefix | Table | Key | Value | Description |
|--------|-------|-----|-------|-------------|
| `r` | Receipts | `block.Bytes()` (8B) | RLP(`[]*ReceiptForStorage`) | All receipts for a block (batch) |
| `x` | TxPositions | `txHash.Bytes()` (32B) | RLP(`TxPosition{Block, Event, EventOffset, BlockOffset}`) | Transaction position index |
| `X` | Txs | `txHash.Bytes()` (32B) | RLP(`*types.Transaction`) | Full transaction data |

The `TxPosition` struct maps a transaction hash to its location:

```go
type TxPosition struct {
    Block       idx.Block    // which block
    Event       hash.Event   // which DAG event originated it
    EventOffset uint32       // offset within the event
    BlockOffset uint32       // offset within the block
}
```

#### EVM Store Caches

| Cache | Default Size | Key | Value |
|-------|-------------|-----|-------|
| Receipts | 4000 blocks / 4 MiB | `idx.Block` | `types.Receipts` (value) |
| TxPositions | 20,000 items | `txHash` string | `*TxPosition` (pointer) |
| EvmBlocks | 5000 blocks / 6 MiB | `idx.Block` | `*EvmBlock` (pointer, in-memory only) |

#### Log Topics Index (`topicsdb`)

EVM logs are indexed via `topicsdb.Index`, which also shares the gossip
PebbleDB. Uses a **leap-join algorithm** for efficient multi-topic queries.

Key schema for topic entries:

```
+------------------+----------+------------------------------+
|   Topic Hash     | Position |        Log Record ID         |
|   (32 bytes)     | (1 byte) |        (48 bytes)            |
+------------------+----------+------------------------------+
                                     |
                                     v
                          +----------+----------+----------+
                          | Block    | TxHash   | LogIndex |
                          | (8B)     | (32B)    | (8B)     |
                          +----------+----------+----------+
```

- **Topic Hash** (32B): The Keccak-256 topic value (e.g., `Transfer` event signature)
- **Position** (1B): Topic position in the log (0-4, matching LOG0..LOG4 opcodes)
- **Log Record ID** (48B): Composite of block number + tx hash + log index

This layout allows efficient prefix scans: given a topic hash + position,
iterate all matching log record IDs in block order.

---

### Carmen StateDB -- EVM World State

Carmen is a **separate high-performance state database** (not PebbleDB).
It stores account balances, nonces, contract code, and storage slots
using a custom file-based MPT (Merkle Patricia Trie) implementation.

```go
// Configuration (evmstore/config.go)
StateDb: carmen.Parameters{
    Variant:      "go-file",          // file-based Go implementation
    Schema:       carmen.Schema(5),    // schema version 5
    Archive:      carmen.S5Archive,    // archive mode (for RPC nodes)
    LiveCache:    1940 * MiB,          // live state cache
    ArchiveCache: 1940 * MiB,          // archive state cache
}
```

Carmen stores data in a dedicated `carmen/` directory with this structure:

| Directory | Contents | Purpose |
|-----------|----------|---------|
| `live/accounts/` | `values.dat`, `freelist.dat` | Account state (balance, nonce, code hash, storage root) |
| `live/values/` | `values.dat`, `freelist.dat` | Storage slot values |
| `live/branches/` | `values.dat`, `freelist.dat` | MPT branch nodes |
| `live/extensions/` | `values.dat`, `freelist.dat` | MPT extension nodes |
| `live/codes.dat` | flat file | Contract bytecode storage |
| `live/forest.json` | JSON | MPT forest configuration |
| `live/meta.json` | JSON | Database metadata |

**Archive mode**: RPC nodes additionally maintain an `archive/` directory that
stores historical state snapshots. Validator nodes run with `NoArchive` to save
disk space. The node refuses to start if the archive mode doesn't match the
existing data directory to prevent inconsistencies.

The `CarmenStateDB` wrapper (`evmstore/carmen.go`) implements Sonic's
`state.StateDB` interface by delegating to Carmen's native `VmStateDB`:

```
 go-ethereum EVM
       |
       v
 state.StateDB interface (inter/state/adapter.go)
       |
       v
 CarmenStateDB (evmstore/carmen.go)
       |  wraps calls: GetBalance -> carmen.GetBalance, etc.
       v
 carmen.VmStateDB / carmen.StateDB
       |
       v
 Carmen file-based MPT (disk)
```

Key operations mapped:
- `GetBalance(addr)` / `AddBalance(addr, val)` -- Carmen account operations
- `GetState(addr, key)` / `SetState(addr, key, val)` -- Carmen storage slot operations
- `GetCode(addr)` / `SetCode(addr, code)` -- Carmen code storage
- `BeginBlock(num)` / `EndBlock(num)` -- Carmen block lifecycle
- `Snapshot()` / `RevertToSnapshot(id)` -- Carmen transaction-level snapshots
- `GetHash()` -- Carmen state root hash (Merkle root of the entire state trie)

---

### abft.Store -- Lachesis Consensus State

Managed by the `lachesis-base` library. Uses two PebbleDB databases:

| Database | Contents |
|----------|----------|
| `"lachesis"` | Main consensus state (frames, roots, decided state) |
| `"lachesis-{epoch}"` | Per-epoch consensus data (election state, votes). Dropped on epoch advance. |

This store is opaque to the Sonic codebase -- it's managed entirely by the
`abft.Lachesis` engine from lachesis-base.

---

### kvdb Abstraction Layer

All KV access goes through the `kvdb` interfaces from `lachesis-base`:

```go
// Core interface -- basic key-value operations
type Store interface {
    Get(key []byte) ([]byte, error)
    Has(key []byte) (bool, error)
    Put(key, value []byte) error
    Delete(key []byte) error
    NewIterator(prefix, start []byte) ethdb.Iterator
    // ...
}

// Flush coordination
type FlushableDBProducer interface {
    OpenDB(name string) (Store, error)
    Flush(id []byte) error
    NotFlushedSizeEst() int
    // ...
}
```

The **table prefix pattern** is central to the design:

```go
// table.MigrateTables reads struct tags and wraps each field
// so that "e" prefix is auto-prepended to all Events table operations
table.MigrateTables(&s.table, s.mainDB)

// This means the actual key on disk for an event is:
//   "e" + event.ID.Bytes()
// And for a block:
//   "b" + block.Bytes()
```

This allows multiple logical tables to share a single PebbleDB instance,
keeping the number of open file descriptors low while maintaining logical
separation.

---

### Data Serialization

| Format | Used By | Description |
|--------|---------|-------------|
| **RLP** | Events, Blocks, Receipts, Txs, TxPositions, BlockEpochState, UpgradeHeights | Ethereum's Recursive Length Prefix encoding. Used via `rlpstore.Helper` which wraps encode/decode with error handling and caching. |
| **Protobuf** | P2P wire messages, SCC certificates, event payloads (v3) | Used for newer data formats. Certificates are serialized via `cert.Certificate.Serialize()`. |
| **Big-endian uint** | Block numbers, epoch numbers, Lamport timestamps, periods | Fixed-width big-endian encoding for sortable keys. |
| **Raw bytes** | Heads, LastEvents | Concatenated fixed-width entries packed into a single value. |
| **Carmen native** | EVM state (accounts, storage, code) | Custom binary format in `.dat` files with freelist-based allocation. |

---

### DB Migration

The `gossip/store_migration.go` file tracks a linear migration chain:

```
opera-gossip-store
  -> "used gas recovery"
  -> "tx hashes recovery"
  -> "DAG heads recovery"
  -> "DAG last events recovery"
  -> "BlockState recovery"
  -> "LlrState recovery"
  -> "erase gossip-async db"
  -> "erase SFC API table"
  -> "erase legacy genesis DB"
  -> "calculate upgrade heights"
  -> "add time into upgrade heights"    <- current
```

Migration state is tracked in the `Version` table (prefix `_`).
Older migrations are marked `unsupportedMigration` -- if the DB is too old,
the node must be restarted from a fresh genesis.

---

## Network IDs

| Network | ID |
|---------|----|
| Mainnet | `0xfa` (250) |
| Testnet | `0xfa2` (4002) |
| Fakenet | `0xfa3` (4003) |
