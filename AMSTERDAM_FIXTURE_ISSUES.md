# Amsterdam fixture results — issue list

Fixtures: `/home/simon/Documents/ethereum/fixtures_v21.0.1` (`state_tests`, generated
2026-10-07). Run on 2026-10-14 against:

| Component | Version |
| --- | --- |
| go-ethereum fork | `0xsoniclabs/go-ethereum v0.0.0-20261013140340-66cf338cad69` (1.17.8), unchanged |
| Tosca | `0xsoniclabs/tosca v0.0.0-20261014112021-5893ca772cf7` (branch `simon/amsterdam`) |
| Carmen | `0xsoniclabs/carmen/go v0.0.0-20260925152836-62f625a466a9` |

## Results

geth StateDB, all fork directories:

| VM | passed | failed | skipped |
| --- | --- | --- | --- |
| geth | 75 138 | 0 | 8 229 |
| sfvm | 75 138 | 0 | 8 229 |
| lfvm | 75 091 | 47 | 8 229 |

The skipped subtests belong to forks before Byzantium, plus ConstantinopleFix, which Tosca does
not support.

Carmen StateDB, `for_amsterdam` only, sfvm, which is the production pairing:

| VM | passed | failed |
| --- | --- | --- |
| sfvm | 16 871 | 84 |

## Fixed

The previous run found 209 sfvm failures in `for_amsterdam`, all in Tosca, and 1 383 more under
TangerineWhistle and SpuriousDragon. The following changes fix all of them. The Tosca changes are
folded into the Amsterdam commits of Tosca's `simon/amsterdam` branch:

* SFVM: `5893ca7` "SFVM implement amsterdam support"
* LFVM: `682abbe` "LFVM amsterdam support"
* `geth_adapter`: `bdd7c48` "Enable amsterdam in ct and geth"
* the CT specification: `7397edb` "Implement eip 8038 support"

1. **CREATE no longer skips the create-frame precheck (Tosca).** Since 1.17.8, go-ethereum runs
   the depth, balance and nonce-overflow checks in the CREATE instruction under Amsterdam, and no
   longer in `evm.create`. sfvm and lfvm `genericCreate` now run the depth and nonce checks
   themselves, before charging anything. A creator at nonce 2^64-1, or a frame at depth 1024,
   aborts the creation, as in go-ethereum.
2. **CREATE charges the account creation only for an empty destination (Tosca).** go-ethereum
   charges the 183 600 account-creation state gas only if `StateDB.Empty(contractAddr)`.
   `genericCreate` now derives the destination (`newContractAddress`) and charges only if it is
   empty, so pre-funded and colliding destinations are no longer charged. The CT specification's
   `createEffect` follows the same rule.
3. **SELFDESTRUCT to self keeps the balance (Tosca, EIP-8246).** `geth_adapter.SelfDestruct` no
   longer burns the balance of a same-transaction contract that self-destructs to itself under
   Amsterdam.
4. **Returning frames repay spilled state gas (Tosca, EIP-8037).** go-ethereum 1.17.8's
   `GasBudget.Absorb` uses the reservoir a child hands back to repay execution gas its caller
   borrowed for earlier state charges. sfvm and lfvm `absorbStateGas` now do the same.
5. **Pre-Byzantium forks are skipped (Sonic).** `convertRevision` in Tosca bottoms out at
   Istanbul, so `unsupportedForks` in `tests/eth_exec_spec/state_test.go` now lists TangerineWhistle
   and SpuriousDragon as well.

The earlier geth-side issues (EIP-8038 constants, EIP-2780 creation charge, EIP-7610) were already
gone with 1.17.8. go-ethereum needed no changes.

## Open

### Carmen does not implement EIP-8246 — 84 subtests (sfvm + Carmen)

Under EIP-8246, an account that self-destructs keeps any non-zero balance. At transaction end,
go-ethereum's `finaliseAmsterdam` turns it into a balance-only account: nonce 0, no code, no
storage. Carmen instead zeroes the balance in `Suicide`, and its `EndTransaction` deletes every
self-destructed account, including any value it received afterwards. Carmen does not know the
active fork, so fixing this needs a Carmen change plus a way for Sonic to signal Canto.

The same subtests pass on the geth StateDB. Files:

* `cancun/eip6780_selfdestruct` (63)
* `amsterdam/eip8246_selfdestruct_no_burn` (11)
* `frontier/create` (3)
* `ported_static/stCreate2` (2)
* `amsterdam/eip8037_…/call_value_to_self_destructed_same_tx_account` (2)
* one each in EIP-2780, EIP-8038 and `stInitCodeTest`

### LFVM cannot run code above 65 535 bytes (EIP-7954) — 47 subtests (lfvm)

LFVM instructions carry 16-bit arguments for jump targets, `PC` values and packed push data.
`Converter.Convert` rejects longer code, and `PC` beyond that position becomes `INVALID`. So the
64 KiB code and 128 KiB initcode limits of EIP-7954 are out of reach without widening LFVM's
instruction format. Production uses sfvm, so this is left as is.

### Not run

The Carmen StateDB has only been run on `for_amsterdam` with sfvm.

## Re-running

The geth StateDB configurations are cheap. The whole fixture set takes about 100 s per VM:

```
go test ./tests/eth_exec_spec/ -test-data /home/simon/Documents/ethereum/fixtures_v21.0.1 \
  -run 'TestBlockProcessing_EthereumExecutionSpecTests/VM:_sfvm,_StateDB:_geth/'
```

Carmen is expensive. go-ethereum's test matcher runs the subtests with `t.Parallel()`, so a
process builds as many Carmen states at once as `-test.parallel` allows, and the
`ported_static/stRandom*` files are around 157 MB each. Running several Carmen configurations side
by side exhausted 30 GB of RAM.

This setup ran `for_amsterdam` in 21 minutes, peaking at 5.8 GB:

```
go test -c -o execspec.test ./tests/eth_exec_spec/
systemd-run --user --scope -p MemoryMax=6G -p MemorySwapMax=0 ./execspec.test \
  -test-data /home/simon/Documents/ethereum/fixtures_v21.0.1 -test.parallel 4 -test.timeout 5h \
  -test.run 'TestBlockProcessing_EthereumExecutionSpecTests/VM:_sfvm,_StateDB:_carmen/for_amsterdam/'
```
