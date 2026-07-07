## Plan: RPC Conformance Tests from execution-apis

Add a test suite in `tests/execution_api/` that loads external test fixtures from `github.com/ethereum/execution-apis` (cloned out-of-source), replays a chain using the `api/rpc_test` infrastructure, then validates JSON-RPC responses against `.io` test vectors. A CLI flag (`-execution.apis`) points to the cloned repo.

---

**Steps**

### Phase 1: Test Data Loading Infrastructure

1. **Add CLI test flag** — Define `var executionAPIsDir = flag.String("execution.apis", "", "...")` in a new `tests/execution_api/conformance_test.go`. Skip all conformance tests when flag is empty. *(Pattern from `tests/state_test.go` with external repo check)*

2. **Genesis loader** (`tests/execution_api/genesis_loader.go`) — Parse `tests/genesis.json` using go-ethereum's `core.Genesis` type. Convert `alloc` entries to `AccountState` map for state initialization. Extract chain config for EVM fork rules.

3. **Chain.rlp loader** (`tests/execution_api/chain_loader.go`) — Decode RLP stream of `types.Block` (standard go-ethereum format). Convert to the existing `rpctest.Block` structs with transactions.

### Phase 2: Populate Fake Backend from Fixtures

1. **Wire loaders into the fake backend builder** — The genesis loader feeds accounts via `WithAccount()` for each alloc entry. The chain loader feeds blocks via `WithBlockHistory()`. No EVM execution or chain replay; the test only exercises the RPC handler layer against the pre-populated fake backend.

2. **Extend fake backend if needed** — Minor additions to `api/rpc_test/backend.go` (e.g., supporting more fields in the `Block` struct for gas, timestamp, etc.) to serve the data the RPC handlers need. Receipt-dependent methods (`eth_getTransactionReceipt`, `eth_getLogs`) are skipped initially since receipts require execution. *depends on steps 2, 3*

### Phase 3: Test Vector Parser & Runner

1. **`.io` file parser** (`tests/execution_api/io_parser.go`) — Walk `tests/<method>/*.io`, parse `>>` lines as requests and `<<` lines as expected responses. Handle multi-roundtrip files (sequential pairs).

2. **JSON-RPC dispatcher** (`tests/execution_api/dispatcher.go`) — Register all API services (`GetAPIs()` from `api/api.go`) on an in-process `rpc.Server` (no HTTP). Route raw JSON-RPC request bytes → response bytes. *depends on step 5*

3. **Conformance test runner** (in `tests/execution_api/conformance_test.go`) — Load fixtures, replay chain, sub-test per method/file. Send each request through dispatcher, compare JSON response semantically. *depends on steps 6, 7*

### Phase 4: Divergence Handling

1. **Skip list** — Define `skipMethods` (e.g., `eth_getProof`, `eth_simulateV1`, `testing_buildBlockV1`) and `skipTests` for known Sonic-specific divergences. *parallel with step 8*

---

**Relevant files**

- `api/rpc_test/backend.go` — Fake backend builder; extend `Block` struct if needed for extra header fields
- `api/rpc_test/tools.go` — `Block` struct, conversion helpers
- `api/ethapi/backend.go` — `Backend` interface reference (methods the fake backend must satisfy)
- `api/api.go` — `GetAPIs()` for registering services on in-process RPC server
- `tests/state_test.go` — Reference pattern for external-repo-gated tests

---

**Verification**

1. `go test ./tests/execution_api/ -v -execution.apis=/path/to/execution-apis` — runs conformance suite
2. `go test ./tests/execution_api/ -v` (without flag) — skips conformance, existing tests pass
3. Early smoke: `eth_blockNumber`, `eth_chainId`, `eth_getBalance`, `eth_getBlockByNumber` should pass first
4. CI remains green — tests gated behind the flag

---

**Implementation Notes**

- The `execution-apis` repo is checked out at `/home/luis/code/execution-apis`. Use this path during development/testing.
- **Goal is test infrastructure only** — do NOT modify the existing Sonic RPC handlers or backend code to make tests pass. Tests are expected to fail initially; the value is in having the conformance harness in place to surface gaps.

---

**Decisions**

- **In-process RPC server** (not HTTP): no port issues, faster, sufficient for conformance
- **No chain replay / no Carmen execution**: the test exercises only the RPC handler logic against a pre-populated fake backend; receipt/log-dependent tests are skipped until a later phase
- **Flag-based** (not submodule): user controls execution-apis version
- **Reuse existing fake backend builder** (`api/rpc_test`): populate via `WithAccount()` and `WithBlockHistory()`, no new state infrastructure
- **Tests may fail**: the objective is the test infrastructure, not fixing conformance gaps in the current code

---

**Further Considerations**

1. **Incremental rollout** — Start with stateless methods (`eth_blockNumber`, `eth_chainId`) and simple state queries (`eth_getBalance`, `eth_getBlockByNumber`). Receipt-dependent methods (`eth_getTransactionReceipt`, `eth_getLogs`) are out of scope until a chain-replay layer is added in a future phase.

2. **JSON comparison** — execution-apis may include fields Sonic omits or vice versa. Recommend "expected fields must match" (subset comparison) rather than byte-exact match.

3. **Multi-roundtrip `.io` files** — Some tests (e.g., send tx then query it) need sequential execution with side effects. Process `>>` / `<<` pairs in order against the same live backend state.
