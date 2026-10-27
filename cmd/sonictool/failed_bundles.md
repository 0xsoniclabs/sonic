# Failed bundle export and replay

Before Canto, a failed Sonic bundle can change the state without leaving a tx or receipt in the block. Without extra data, Bertha's replay of those blocks fails. The Sonic exporter recovers the missing envelopes from the node's events, the block service stores them per chain, and Bertha's replay runs them.

## The problem

- Before Canto, `RunBundle` does not revert a failed bundle as a whole. Only groups (`AllOf`, `OneOf`) revert themselves.
- If an envelope's root is a **single tx** that fails, the tx's effects stay: nonce, balance and gas. Its processed list is discarded, so the block gets no tx and no receipt for it.
- The same happens for a nested envelope that fails inside an outer bundle that succeeds.
- The header's `gasUsed` is the processor's counter, so it includes the leaked gas.
- Canto fixes this: from Canto on, a failed bundle is reverted as a whole.

What breaks in Bertha's replay without the data:

| Check | Symptom |
| --- | --- |
| Receipts | `receipt cumulative gas used mismatch` (mainnet block 80174946) |
| Later txs of the same sender | nonce gap, skipped txs |
| State root | differs, because the bundle senders' nonce and balance changes are missing |
| Block hash | `gasUsed` from the last receipt differs from the header |

## File format

```json
[
  {"block": 80174946, "pos": 1, "tx": "0x01f9033b..."},
  {"block": 80174946, "pos": 2, "tx": "0x01f90250..."}
]
```

- `block`: block number.
- `pos`: number of the block's txs executed before the envelope. `0` means before the first tx, and `len(txs)` means after the last.
- `tx`: hex of the envelope's `MarshalBinary()` (typed tx encoding).
- Entries are sorted by block, then by execution order. Several entries may share a `pos`.
- For a nested failure, `tx` is the **inner** envelope as it ran inside the outer bundle: signed by the step's key and carrying the bundle-only marker. The envelope tx itself never executes, only its payload, so this copy has the same effects as the original.

## Export

### Which envelopes are exported

Only envelopes whose **own root is a single tx** that failed. Groups revert themselves.

- Every valid bundle is still run, including groups and successful ones, because:
  - a failure can be nested inside it;
  - the number of block txs it consumed sets the positions of later entries;
  - its plan feeds the processed set (rescheduled bundles) and the history hash check.
- A failed root tx that was merely invalid (e.g. bad nonce) is exported too. It has no effect in the replay.
- A bundle that succeeded with all its txs invalid but tolerated is exported too. Bertha replays it as a bundle without receipts.

### How it works

1. **Block range:** from the activation of TransactionBundles until the block before Canto, or the latest block.
2. **Per epoch:** epochs without envelopes in their events are not replayed; only their history hash is checked.
3. **Consensus replay:** Lachesis runs in memory on the stored events and yields the blocks (Atropos time, confirmed events). Empty blocks without cheaters are skipped within the skip period, as in Sonic. The rebuilt block time must match the stored one.
4. **Tx order:** events are spilled by block gas, txs are put in scrambler order, and non-permissible txs are dropped. Epochs with single-proposer block formation or tx priorities are not supported and fail.
5. **Bundle replay:** the ordered txs are walked along the block's included txs, without an EVM.
6. **Checks:** every included tx is reproduced, the history hash matches per block, and each epoch's seal hash matches.

Bundle replay (step 5) uses Sonic's own `bundle.RunBundle` with a runner that only matches tx hashes:

- **Plain tx:** if it is the next included tx, consume it and take its status from the stored receipt. Otherwise count it as *missing*.
- **Sponsored tx:** like a plain tx; its internal post-txs (fee charge), which follow it in the block, are consumed with it.
- **Envelope:**
  - Skip it if invalid, out of range or period, or its plan was already processed (rescheduled).
  - Otherwise run its plan, with missing txs resulting as failed. Snapshots and reverts restore the position and the missing count.
  - A missing tx may also have been invalid without being reverted, e.g. in a step that tolerates only invalid txs. If the next included tx belongs to the bundle or to one nested in it, the run was wrong: it is repeated with missing txs resulting as invalid.
  - If it fails with missing txs of its own, export it at the position where it started.

### Incremental export (`--append`)

- Reads the existing file and keeps its entries. The search starts at the **last entry's block + 1**.
- Sonic's store deletes processed bundles and history hashes after 32768 blocks, so the resume state is rebuilt:
  - **Warm-up:** replay from the epoch that contains `start − 1023` (a plan's maximum range is 1024 blocks). This rebuilds the set of processed plans, so rescheduled bundles are recognised.
  - If that epoch also contains `start`, begin one epoch earlier, so that every exported block is checked against a history hash.
  - **History hash:** unknown until the end of the warm-up epoch. It is taken from that epoch's seal and verified from then on.
- Only entries with `block ≥ start` are added.
- The file is replaced (temp file + rename) only after a successful search.
- The resume point is the last *exported* block, not the last *scanned* one. After a stretch without failures, the export rescans more blocks than needed. That is still correct, just slower.

### Operations

```sh
# sonicd is synced; stop it gracefully
sonictool --datadir <datadir> events export-failed-bundles failed_bundles.json           # first run
sonictool --datadir <datadir> events export-failed-bundles --append failed_bundles.json  # later runs
# restart sonicd
blockservice import-failed-bundles sonic failed_bundles.json
```

- Needs the events since bundle activation (first run), or since the warm-up epoch (`--append`).
- **Read-only:**
  - Opens only `chaindata` and never opens Carmen. The last gossip DB migration dates from v2.0.0, so a current node has none pending.
  - Any write would persist a dirty flag, and sonicd would then refuse to start (`dirty state`). The exporter test checks that a hash of all databases is unchanged after the export.
- The DB must be clean: after a crash (dirty flag set), the export refuses to run.

## Import and replay in Bertha

- **Storage:** like the corrections, the block DB holds the exporter's JSON as one value per chain, separate from the corrections. It is not served or fetched over gRPC.
- **Import** (`blockservice import-failed-bundles <chain> <file>`, `blockservice/src/cmd/import_metadata.rs`): stores the file as is, replacing the previously imported one. It does not check the entries against the blocks.
- **Loading** (`go/replay/failed_bundles.go`, `go/replay/metadata.go`): the metadata store parses the JSON when the replay starts:
  - decodes each tx with `UnmarshalBinary`;
  - requires `block` and `pos` in every entry, and the entries sorted by block, then by `pos`;
  - groups the entries by block.
- **Applying a block** (`go/replay/state.go`):
  - Each envelope is inserted before the tx at its `pos`; `pos > len(txs)` is an error.
  - The processor (`NewStateProcessorForReplay`) runs the envelope. Its failed bundle leaves the state changes and adds no receipt.
  - The block time includes the nanoseconds from the header's extra data, as Sonic checks a bundle's time period against it (`go/convert/convert.go`).
  - Every skipped tx is an error, including a skipped envelope, whose effects would be missing. A failed bundle that adds a receipt is an error too.
- **Header:** `gasUsed` comes from the processor's `usedGas`, not from the last receipt, so it includes the leaked gas and the block hash matches (`go/replay/replay.go`).
- **Archive:** archive verification applies the same failed bundles.
- **Verify:** `bertha verify` and `blockservice verify` compute a block's `gasUsed` from its last receipt, so the hash of a block with failed bundles does not match. Both read the blocks with failed bundles from the stored JSON. On any chain, mismatches of these blocks are tolerated with a warning on the first one. Other mismatches still fail.
- `blockservice import-gfile --verify` does not tolerate them, as it does not read the failed bundles. Import such g-files without `--verify`, then import the failed bundles and run `blockservice verify`.

## Verification and limits

- On a local integration net:
  - single failed bundle in one block;
  - failed bundles at the beginning, middle and end of blocks;
  - nested failures: bare root, inside a `OneOf`, after a tx with `TolerateFailed`, and inside a group that reverts.

  In every case, the Bertha replay fails without the envelopes and passes with them, in pipeline, sequential and archive mode. This was not yet repeated with the envelopes imported through `blockservice import-failed-bundles`.
- Exporter tests: export cases, resume across epochs with a rescheduled bundle, databases unchanged by the export. Each has a negative control: the broken variant fails the test.
- Not covered by the exporter tests: sponsored txs, steps that tolerate only invalid txs, and empty blocks with cheaters.
- Exported for Sonic mainnet: Bertha's `data/failed_bundles/sonic.json`.
