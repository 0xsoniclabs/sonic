# Network Sponsorship Modes

## Motivation

The existing gas-subsidy mechanism allows external parties to pre-fund a pool inside the
subsidies registry contract. When a user submits a zero-gas-price transaction, the node
queries the registry at block-processing time: if the registry returns a non-zero `FundId`,
the transaction is executed and a synthetic `deductFees(fundId, fee)` transaction is
appended to the block, charging the fee to the identified fund.

That model covers one class of sponsorship — *explicit fund-backed* — but leaves two
important use-cases unserved:

**Off-chain rate-limited network sponsorship.** Some transactions should be sponsored
directly by the network (i.e. the gas cost is absorbed by the block producers) without
any on-chain fund to deduct from. Access control and rate limiting are handled entirely
off-chain: the registry is governed to return the network-sponsored signal only for
permitted senders, recipients, or call patterns. No on-chain state needs to change as
a result of the sponsorship.

**On-chain rate-limited network sponsorship with tracking.** Other scenarios require the
network to sponsor transactions while still maintaining an on-chain record of how much
gas was consumed. This lets the registry enforce on-chain rate limits, quotas, or
analytics: after each sponsored execution the registry is notified with a tracking
identifier and the fee that would have been charged, and the registry can reject future
requests once limits are exceeded.

---

## Design

### Sponsorship modes

`chooseFund` is extended to return two words: a `mode` (`uint256`) and a `payload`
(`bytes32`). The mode is an explicit discriminator; the payload's meaning depends on
the mode.

| Mode | Name | Payload | Behaviour |
|---|---|---|---|
| 0 | not covered | zero (ignored) | Reject the sponsorship request |
| 1 | fund-backed | `fundId` | Execute; insert `deductFees(fundId, fee)` (unchanged) |
| 2 | network sponsored | zero (ignored) | Execute; no post-execution transaction |
| 3 | network sponsored + tracking | `trackingId` | Execute; insert `track(trackingId, fee)` |

### Execution pipeline

#### Mode 1 — fund-backed (existing, unchanged)

1. `chooseFund` returns `(1, fundId)`.
2. The sponsored transaction is executed with `NoBaseFee = true`.
3. A `deductFees(fundId, fee)` transaction is appended, charging the fund.

#### Mode 2 — network sponsored

1. `chooseFund` returns `(2, <ignored>)`.
2. The sponsored transaction is executed with `NoBaseFee = true`.
3. No post-execution transaction is appended. All gas costs are absorbed by the network.

#### Mode 3 — network sponsored with tracking

1. `chooseFund` returns `(3, trackingId)`.
2. The sponsored transaction is executed with `NoBaseFee = true`.
3. A `track(trackingId, fee)` transaction is appended, where
   `fee = (gasUsed + overheadToCharge) × baseFee`. It is constructed identically to
   the fee-deduction transaction (sent from the zero address, gas price zero, internal).

The fee reported to `track` carries sufficient information for the registry to recover
`gasUsed` if needed, by dividing by the block's base fee.

### Updated registry interface

`chooseFund` is extended to return `(mode uint256, payload bytes32)` instead of the
single `bytes32 fundId`. The node detects the registry version by the length of the
response:

| Response length | Interpretation |
|---|---|
| 32 bytes | Legacy registry — parse as `fundId`; map non-zero to mode 1 |
| 64 bytes | Extended registry — parse as `(mode, payload)` |
| any other length | Error |

The 32-byte legacy shape is unambiguous: old registries only ever returned a raw `fundId`
(0 = not covered, non-zero = fund-backed), which maps exactly to modes 0 and 1. They
never return mode 2 or 3, so no backward-compatibility hazard exists.

```solidity
// Extended return: (mode, payload)
// payload is fundId for mode 1, trackingId for mode 3, zero otherwise.
function chooseFund(
    address from,
    address to,
    uint256 value,
    uint256 nonce,
    bytes calldata data,
    uint256 maxFee
) external view returns (uint256 mode, bytes32 payload);

// Reports the gas fee consumed by a network-sponsored tracked transaction.
// Called from the zero address as an internal transaction.
function track(bytes32 trackingId, uint256 fee) external;
```

### Gas configuration extension

The registry exposes a `getGasConfig()` function (selector `0x4b5c54c0`) that provides
the gas limits and block-resource overhead needed to process each sponsorship mode.

The canonical format returns five `uint64` values ABI-encoded as five 32-byte words
(160 bytes), **gas limits first, then per-mode overhead charges**:

```
chooseFundGasLimit | deductFeesGasLimit | trackGasLimit | fundBackedOverheadCharge | networkTrackedOverheadCharge
```

`fundBackedOverheadCharge` is the total extra gas to reserve for a mode-1 (fund-backed)
sponsorship — it covers the cost of the `chooseFund` and `deductFees` calls and the
`getGasConfig` call itself. `networkTrackedOverheadCharge` covers the same base costs but
replaces `deductFees` with `track`, so it uses `trackGasLimit` instead of
`deductFeesGasLimit`.

These two values are used separately:

- **Before `chooseFund`**: since the mode is not yet known, the node uses
  `max(fundBackedOverheadCharge, networkTrackedOverheadCharge)` to compute the fee
  estimate passed to `chooseFund`.
- **After `chooseFund`**: the mode-specific overhead is stored in the `Sponsorship` so
  that `Overhead()` and `GetPostTransactions()` use the exact value for that mode.

Modes 0 and 2 append no post-execution transaction and therefore carry zero overhead;
no per-mode field is needed for them.

The node detects the registry version by the length of the returned data:

| Response length | Interpretation |
|---|---|
| 96 bytes | Legacy registry — `trackGasLimit` defaults to zero; both overhead fields default to the single shared overhead value |
| 160 bytes | Full registry — all five fields present |
| any other length | Error |

Legacy 96-byte registries predate tracking support. Defaulting both overhead fields to
the shared overhead value is conservative: mode-3 sponsorships were not possible on
those registries, so the value is only ever used for mode 1.

---

## Design decision justifications

### Why a separate mode field rather than sentinel FundId values?

The original approach reserved `fundId == 1` and `fundId == 2` as special signals inside
the existing `bytes32` return value. This keeps the response to 32 bytes for the common
case, but pollutes the `FundId` namespace: a fund-backed sponsor can no longer use those
values as legitimate fund identifiers, and any future mode extension must reserve further
values from the same space. Separating `mode` and `payload` into two distinct words
gives each field a single, unambiguous purpose. The mode is always an integer enum; the
payload is always either a `fundId` or a `trackingId` depending on context. No value in
either field is special-cased or reserved for a different purpose.

Backward compatibility is preserved by the same length-detection mechanism used
elsewhere: a 32-byte response is parsed with the legacy rule (0 = not covered,
non-zero = fund-backed, mode 1); a 64-byte response uses the new `(mode, payload)` rule.
Old registries that return 32 bytes never produce a mode-2 or mode-3 result, so no
misinterpretation is possible.

### Why return the tracking ID from `chooseFund` rather than a separate `getTrackingId` call?

The registry already has all the information it needs to compute a tracking identifier
at `chooseFund` time — it receives the full transaction parameters (`from`, `to`,
`value`, `nonce`, `calldata`, `maxFee`). Returning the tracking ID as the `payload` in
the same response eliminates one round-trip EVM call from the block-processing pipeline
(the mode-3 path drops from `chooseFund → getTrackingId → execute → track` to
`chooseFund → execute → track`), removes a registry function from the interface, and
removes one gas limit entry from the gas config. The payload word is simply ignored by
the node for modes 0 and 2, so no overhead is introduced for those paths.

### Why reuse the `getGasConfig` and `chooseFund` selectors and detect version by response length?

Solidity function selectors do not encode return types — they are `keccak256` of
`"name(paramType,…)"[:4]`. Two functions with the same name and parameter types but
different return types collide on the same selector and cannot coexist in a single
contract. A versioned function name (e.g. `getGasConfigV2`) avoids the collision but
requires the node to attempt the new call first, catch a revert, and then fall back to
the old call — two round-trips with conditional error handling.

Instead, the existing functions are extended in the upgraded registry to return
additional words. The node detects the version by checking `len(result)`. This is
unambiguous because all return types here are fixed-size (`uint64`, `uint256`, `bytes32`),
ABI-encoded as fixed 32-byte words — there is no variable-length encoding that could
produce an intermediate length. The result is a single call, straightforward
length-based branching, and no revert-catch logic. The same pattern applies to both
`chooseFund` (32 vs 64 bytes) and `getGasConfig` (96 vs 160 bytes).
