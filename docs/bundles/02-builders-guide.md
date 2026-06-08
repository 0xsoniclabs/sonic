# Transaction Bundles: A Builder's Guide

This article covers everything you need to integrate transaction bundles into your application: how bundles are structured, the three-step API workflow, execution flags, validity constraints, and how the network validates and executes bundles under the hood.

> **Not a developer?** The technical details here are aimed at dApp and tooling builders. If you want to understand what bundles can do for your users, start with [Article 1](./01-what-are-bundles.md) and the use-case walkthrough in [Article 3](./03-bundles-in-action.md).

---

## Anatomy of a Bundle

A bundle has two parts:

1. **The transactions** — the actual signed Ethereum transactions to be executed.
2. **The execution plan** — a tree structure that tells the network in what order to run them, how to group them, and what to do if something fails.

These two parts are tightly coupled by a **plan hash** that is embedded in every transaction's access list before signing. This binding is what makes the pattern secure: a user's signature over a transaction implicitly authorizes the entire plan, because any change to the plan would change the hash, invalidating the signatures.

### Execution Plan Structure

The execution plan is a tree of **steps**. Each step is either:

- A **transaction step** — a leaf node that references one transaction (by sender address and hash).
- A **group step** — an internal node containing child steps, with `AllOf` or `OneOf` semantics.

```
AllOf
├── tx_A
├── tx_B
└── OneOf
    ├── tx_C
    └── tx_D
```

The plan also carries a **block range** that specifies which blocks the bundle is eligible to be included in.

### The BundleOnly Marker

Before any signing takes place, each transaction in the bundle has an entry injected into its access list:

- **Address:** `0x00000000000000000000000000000000000B0D1E`
- **Storage key:** the plan hash

This marker does two things. First, it signals to the network that this transaction must not be included in a block on its own — only as part of a bundle. Second, it binds the transaction to its execution plan. Because the access list is covered by the transaction signature, signing a bundled transaction is signing a commitment to the specific plan.

### The Envelope

When a bundle is submitted, it does not enter the mempool as individual transactions. Instead, it is wrapped in a single **envelope transaction** sent to the special `BundleProcessor` address:

```
0x00000000000000000000000000000000B0D1EADD
```

The envelope's data field contains an RLP-encoded payload with all bundled transactions and the execution plan. The network unpacks this envelope during block execution and runs the steps according to the plan.

---

## The API Workflow

Integrating bundles involves three RPC calls: **prepare**, **submit**, and optionally **query**.

### Step 1 — Prepare: `sonic_prepareBundle`

Call `sonic_prepareBundle` with your unsigned transactions and the structure you want. The node estimates gas limits, suggests gas prices, computes the plan hash, and injects the BundleOnly marker into each transaction's access list. It returns the transactions ready to sign.

**Request:**

```json
{
  "blockRange": {
    "first": "0x1234",
    "length": "0x0a"
  },
  "steps": [
    {
      "from": "0xAlice",
      "to": "0xNFTMarket",
      "nonce": "0x5",
      "data": "0xbuyNFT42...",
      "chainId": "0xfa"
    },
    {
      "from": "0xAlice",
      "to": "0xNFTMarket",
      "nonce": "0x6",
      "data": "0xbuyNFT43...",
      "chainId": "0xfa"
    }
  ]
}
```

Each leaf step is a transaction argument object — the same fields accepted by `eth_call` — plus two optional flags:

| Field             | Type    | Default | Meaning |
|-------------------|---------|---------|---------|
| `tolerateFailed`  | boolean | `false` | Treat a reverted transaction as successful (state changes are still rolled back). |
| `tolerateInvalid` | boolean | `false` | Skip an invalid transaction (bad nonce, insufficient funds) and treat it as successful. |

Group steps use a different shape:

```json
{
  "oneOf": true,
  "tolerateFailures": false,
  "steps": [ ... ]
}
```

| Field               | Type    | Default | Meaning |
|---------------------|---------|---------|---------|
| `oneOf`             | boolean | `false` | `true` = OneOf semantics; `false` = AllOf semantics. |
| `tolerateFailures`  | boolean | `false` | Treat failure of the whole group as successful. |
| `steps`             | array   | —       | Child steps, which can be transactions or nested groups. |

**Response:**

```json
{
  "transactions": [
    {
      "from": "0xAlice",
      "to": "0xNFTMarket",
      "nonce": "0x5",
      "data": "0xbuyNFT42...",
      "gas": "0x8d20",
      "maxFeePerGas": "0x...",
      "accessList": [
        {
          "address": "0x00000000000000000000000000000000000B0D1E",
          "storageKeys": ["0xplanHash..."]
        }
      ]
    },
    { ... }
  ],
  "executionPlan": {
    "blockRange": { "first": "0x1234", "length": "0x0a" },
    "steps": [
      { "from": "0xAlice", "hash": "0x..." },
      { "from": "0xAlice", "hash": "0x..." }
    ]
  }
}
```

> **Important:** Do not modify any field of the returned transactions. The plan hash is embedded in the access list and the transaction hash is part of the execution plan. Any change invalidates the binding.

If you omit `gas`, the node estimates it for each transaction accounting for the state changes produced by earlier transactions in depth-first order. If you omit gas price fields, the node fills in a suggested value based on the current base fee. Both auto-filling features require an all-AllOf plan with no tolerance flags (because gas estimation assumes all prior steps succeeded and left state that later steps can depend on).

### Step 2 — Sign

Each returned transaction must be signed by the address in its `from` field, in the order they appear in the `transactions` array. The order is the depth-first traversal of the execution plan tree.

If multiple addresses appear in the bundle — which is common in the [single-signature pattern](./04-one-signature.md) — each address signs its own subset of transactions. Collect all signed transactions and preserve their order.

Encode each signed transaction as its binary representation (RLP-encoded, the same encoding used in `eth_sendRawTransaction`).

### Step 3 — Submit: `sonic_submitBundle`

Call `sonic_submitBundle` with the signed transactions and the execution plan returned by `prepareBundle`.

**Request:**

```json
{
  "signedTransactions": [
    "0x02f8...",
    "0x02f8..."
  ],
  "executionPlan": {
    "blockRange": { "first": "0x1234", "length": "0x0a" },
    "steps": [
      { "from": "0xAlice", "hash": "0x..." },
      { "from": "0xAlice", "hash": "0x..." }
    ]
  }
}
```

The `executionPlan` value should be passed through exactly as returned by `prepareBundle`.

**Response:**

```
"0xplanHash..."
```

The response is the execution plan hash — a unique identifier for the bundle. Keep this hash if you want to track execution.

Internally, the node wraps everything into a single envelope transaction, signs it with an ephemeral one-time key, and submits it to the transaction pool. From that point on, validators treat it as an ordinary pending transaction.

### Step 4 — Query: `sonic_getBundleInfo`

Once you have submitted a bundle, you can poll for its execution status using the plan hash.

**Request:** the plan hash returned by `submitBundle`.

**Response:**

```json
{
  "blockNumber": "0x1240",
  "positionInBlock": "0x02",
  "transactionCount": "0x02"
}
```

The node keeps this record for up to 1024 blocks after execution. After that window, the entry is pruned.

---

## Execution Flags Reference

### Transaction-Level Flags

| Flag              | When to use |
|-------------------|-------------|
| `tolerateFailed`  | The transaction may revert on the EVM, but that outcome is acceptable — you do not want the entire group to fail because of it. The transaction's state changes are still rolled back; only the failure status is absorbed. |
| `tolerateInvalid` | The transaction may be structurally invalid at execution time (wrong nonce, insufficient funds). Use this for optional steps that might be redundant depending on prior state. The transaction is skipped entirely; no state changes. |

### Group-Level Flag

| Flag               | When to use |
|--------------------|-------------|
| `tolerateFailures` | The group might fail, but you do not want that to propagate to the parent. Useful when the group is an optional sub-workflow within a larger bundle. |

---

## Block Range and Time Period

Every bundle has a **block range** that controls which blocks it is eligible to be included in.

```json
"blockRange": {
  "first": "0x1234",
  "length": "0x0a"
}
```

- `first` — The first block in which the bundle can be included.
- `length` — The number of consecutive blocks (maximum: **1024**).

A bundle with `first = 0x1234` and `length = 0x0a` is eligible for blocks 0x1234 through 0x123d inclusive.

**Timing behavior:**
- If the current block is before `first`, the bundle waits in the mempool.
- If the current block is at or after `first + length`, the bundle has expired and is removed.

When you call `sonic_prepareBundle`, the `blockRange` is validated and sanitized against the current block number. If you omit `blockRange`, the node fills in a default range starting from the current block with the maximum length.

---

## How the Network Handles Bundles

Understanding the validation and execution pipeline helps you write bundles that behave predictably.

### Mempool Pre-check

When an envelope arrives in the transaction pool, the node runs a series of checks to classify it:

1. **Feature flag** — transaction bundles must be enabled on the network.
2. **Structure validation** — the execution plan must be well-formed, nesting depths must be within limits, and every transaction must carry the BundleOnly marker with the correct plan hash.
3. **Block range check** — if the range has expired, the bundle is rejected permanently; if it targets future blocks, it waits.
4. **Deduplication** — if the same plan hash has been processed within the last 1024 blocks, the bundle is rejected.
5. **Nonce check** — a lightweight dry run verifies that nonces are consistent. If they are inconsistent in a way that could resolve later (e.g. a gap that another transaction could fill), the bundle is marked temporarily blocked rather than rejected.
6. **Trial run** — the bundle is executed against a simulated next block to confirm it would succeed. If the trial run fails, the bundle is permanently rejected.
7. **Efficiency check** — the ratio of gas used by transactions accepted in blocks to the total execution cost must be at least **20%**. The total execution cost includes gas spent running transactions that were subsequently rolled back (e.g. failed OneOf branches), not just the billed gas limits. This prevents bundles that trigger many expensive failing branches from consuming disproportionate network resources.

If all checks pass, the bundle (in its envelope) enters the mempool like any other transaction.

### Block Execution

When a validator includes the envelope in a block, the EVM encounters it during transaction processing. The execution engine unpacks the bundle and runs the steps according to the execution plan:

- For **AllOf** groups: a snapshot is taken before the group runs. If any step fails, the snapshot is restored and all state changes from the group are discarded.
- For **OneOf** groups: steps are tried in order. When one succeeds, the group stops and later branches are skipped entirely. Earlier failed attempts have their state changes rolled back, but they still appear in the block and consume their sender's nonce and gas. If **all** branches fail, none of the transactions are included in the block and no nonces are consumed.
- Tolerance flags (`tolerateFailed`, `tolerateInvalid`) cause the execution engine to accept failure outcomes at the step or group level without propagating failure to the parent.

### Deduplication

After a bundle is executed in block N, the network stores its plan hash with a reference to block N. That record is kept for 1024 blocks. Any attempt to execute the same plan during that window is rejected, preventing replay.

---

## Limits at a Glance

| Constraint               | Limit  | Notes |
|--------------------------|--------|-------|
| Block range length       | 1024 blocks | Hard limit; cannot be changed without a hard fork. |
| Bundle nesting depth     | 2 levels | A bundle can contain a bundle, but not a bundle inside a bundle inside a bundle. |
| Group nesting depth      | 8 levels | Maximum depth of AllOf/OneOf tree within one bundle. |
| Minimum gas efficiency   | 20% | Ratio of gas used by transactions accepted in blocks to total execution cost (gas spent on accepted and rolled-back transactions). |

---

## Tips and Gotchas

**Do not modify prepared transactions.** The node signs references by hash. Any modification changes the hash and invalidates the plan binding.

**Legacy transactions are not supported.** The BundleOnly marker requires an access list, which legacy (type 0) transactions do not support. The prepare step automatically promotes legacy transactions to type 1, but be aware of this if you are building the proposal programmatically.

**Automatic gas estimation requires strict AllOf semantics.** The node can only estimate gas for transactions in strict sequential order where every prior step is assumed to have succeeded. Using `oneOf`, `tolerateFailed`, or `tolerateInvalid` at any level disables auto gas estimation. In those cases you must supply gas limits manually.

**The block range starts now.** If you call `prepareBundle` and then wait several blocks before calling `submitBundle`, the block range that was set during preparation may have already advanced or expired. Build in enough slack, or re-prepare if the submission is delayed.

---

## Next Steps

- [Article 3: Bundles in Action](./03-bundles-in-action.md) — Three worked examples showing how to structure bundles for real-world use cases.
- [Article 4: One Signature, Many Transactions](./04-one-signature.md) — How to build multi-step workflows that require only a single wallet confirmation from the end user.
