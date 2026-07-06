# Sonic RPC Namespace Methods

This document describes all JSON-RPC methods available in the `sonic_` namespace.

All examples use JSON-RPC 2.0 and hex-encoded values.

## Shared Field Formats

These field formats are reused by multiple methods.

### Transaction Object Fields

Used by:

- `sonic_estimateGasForTransactions` as `transactions[]`
- `sonic_prepareBundle` inside each transaction step

| Field | Type | Required | Short description |
| --- | --- | --- | --- |
| `from` | address | Yes for bundle workflows | Sender account. |
| `to` | address | No | Receiver account. Omit for contract creation. |
| `gas` | hex quantity | No | Gas limit for this transaction. |
| `gasPrice` | hex quantity | No | Legacy gas price. Do not combine with EIP-1559 fee fields. |
| `maxFeePerGas` | hex quantity | No | Maximum total gas price for EIP-1559 style fees. |
| `maxPriorityFeePerGas` | hex quantity | No | Miner tip for EIP-1559 style fees. |
| `value` | hex quantity | No | Amount of native token to transfer. |
| `nonce` | hex quantity | Yes for bundle workflows | Sender nonce for ordering. |
| `data` | hex bytes | No | Transaction input data (legacy name). |
| `input` | hex bytes | No | Transaction input data (preferred name). |
| `accessList` | array | No | Optional pre-declared storage/address access list. |
| `chainId` | hex quantity | No | Chain ID for signing context. |
| `maxFeePerBlobGas` | hex quantity | No | Blob gas fee cap (blob tx only). |
| `blobVersionedHashes` | array of hashes | No | Blob hashes (blob tx only). |
| `blobs` | array | No | Blob payloads (blob tx with sidecar). |
| `commitments` | array | No | Blob commitments (blob tx with sidecar). |
| `proofs` | array | No | Blob proofs (blob tx with sidecar). |
| `authorizationList` | array | No | Authorizations for set-code transactions. |

Notes:

- Use `input` instead of `data` when possible.
- Do not set `gasPrice` together with `maxFeePerGas` or `maxPriorityFeePerGas`.

### Block Range Object

Used by:

- `sonic_prepareBundle` request
- `sonic_prepareBundle` response execution plan
- `sonic_submitBundle` request execution plan

| Field | Type | Required | Short description |
| --- | --- | --- | --- |
| `first` | hex quantity | No in prepare request, Yes in execution plans | First block where execution is allowed. |
| `length` | hex quantity | No in prepare request, Yes in execution plans | Number of blocks in the allowed window. |

### Proposal/Plan Group Fields (Recursive)

Groups are used to nest steps.

| Field | Type | Required | Short description |
| --- | --- | --- | --- |
| `steps` | array | Yes | Child steps or child groups. |
| `oneOf` | boolean | No | Try alternatives; one successful branch is enough. |
| `tolerateFailures` | boolean | No | Continue even if a child branch fails. |

### Proposal Leaf Step Fields (Prepare Request)

A proposal leaf is a transaction step inside `sonic_prepareBundle`.

| Field | Type | Required | Short description |
| --- | --- | --- | --- |
| `tolerateFailed` | boolean | No | Keep bundle flow even if this transaction fails. |
| `tolerateInvalid` | boolean | No | Keep bundle flow even if this transaction is invalid. |
| transaction object fields | object | See Transaction Object Fields | The transaction details for this step. |

### Composable Plan Leaf Fields (Prepare Response / Submit Request)

A composable plan leaf references a transaction by sender and hash.

| Field | Type | Required | Short description |
| --- | --- | --- | --- |
| `from` | address | Yes | Sender account for the referenced transaction. |
| `hash` | hash | Yes | Transaction reference hash used by the plan. |
| `tolerateFailed` | boolean | No | Keep flow even if this step fails. |
| `tolerateInvalid` | boolean | No | Keep flow even if this step is invalid. |

## sonic_estimateGasForTransactions

This method estimates gas for a list of transactions as a sequence, so each estimate can consider the state updates from earlier transactions in the same list. It is useful when transactions depend on each other.

### Parameters

Position 1: `transactions` (array of transaction objects, required)

- Full transaction fields are documented in Transaction Object Fields.
- Maximum number of transactions per call: 16.

Position 2: `blockNumberOrHash` (object or tag, optional)

- Block to run estimation against.
- If omitted, latest block is used.

Position 3: `stateOverrides` (object, optional)

- Temporary account changes used only during estimation.

`stateOverrides` value format:

- Top-level keys: account addresses.
- Top-level values: account override object.

Account override object fields:

| Field | Type | Required | Short description |
| --- | --- | --- | --- |
| `nonce` | hex quantity | No | Temporary nonce for this account. |
| `code` | hex bytes | No | Temporary contract code. |
| `balance` | hex quantity | No | Temporary account balance. |
| `state` | object | No | Full temporary storage replacement. |
| `stateDiff` | object | No | Partial temporary storage updates. |

Storage objects (`state` and `stateDiff`):

- Keys: 32-byte storage slot hashes.
- Values: 32-byte storage values.
- Do not provide both `state` and `stateDiff` for the same account.

Position 4: `blockOverrides` (object, optional)

- Temporary block header values used only during estimation.

`blockOverrides` fields:

| Field | Type | Required | Short description |
| --- | --- | --- | --- |
| `number` | hex quantity | No | Temporary block number. |
| `difficulty` | hex quantity | No | Temporary difficulty value. |
| `time` | hex quantity | No | Temporary block timestamp. |
| `gasLimit` | hex quantity | No | Temporary block gas limit. |
| `coinbase` | address | No | Temporary block beneficiary address. |
| `random` | hash | No | Temporary random/mix value. |
| `baseFee` | hex quantity | No | Temporary base fee. |
| `blobBaseFee` | hex quantity | No | Temporary blob base fee. |

### Returns

Object with one field:

| Field | Type | Short description |
| --- | --- | --- |
| `gasLimits` | array of hex quantity | Estimated gas limit for each input transaction in the same order. |

### Example Request

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "sonic_estimateGasForTransactions",
  "params": [
    [
      {
        "from": "0x1111111111111111111111111111111111111111",
        "to": "0x2222222222222222222222222222222222222222",
        "nonce": "0x1",
        "value": "0x0",
        "input": "0x",
        "maxFeePerGas": "0x3b9aca00",
        "maxPriorityFeePerGas": "0x3b9aca00"
      },
      {
        "from": "0x1111111111111111111111111111111111111111",
        "to": "0x3333333333333333333333333333333333333333",
        "nonce": "0x2",
        "value": "0x0",
        "input": "0x",
        "maxFeePerGas": "0x3b9aca00",
        "maxPriorityFeePerGas": "0x3b9aca00"
      }
    ],
    "latest",
    null,
    null
  ]
}
```

### Example Response

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "gasLimits": ["0x6f10", "0x7530"]
  }
}
```

## sonic_getBundleInfo

This method checks whether a bundle execution plan has been seen and executed, then returns where it landed in a block. If the bundle is unknown or not yet available, the result is `null`.

### Parameters

Position 1: `executionPlanHash` (hash, required)

- Execution plan hash returned by `sonic_submitBundle`.

### Returns

Result is either `null` or an object with:

| Field | Type | Short description |
| --- | --- | --- |
| `block` | hex quantity | Block number where the bundle was executed. |
| `position` | hex quantity | Index of the first included transaction in that block. |
| `count` | hex quantity | Number of transactions included from the bundle. |

### Example Request

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "sonic_getBundleInfo",
  "params": [
    "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
  ]
}
```

### Example Response (Executed)

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": {
    "block": "0xbc614f",
    "position": "0x5",
    "count": "0x2"
  }
}
```

### Example Response (Not Yet Known)

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": null
}
```

## sonic_prepareBundle

This method takes a proposed execution tree of unsigned transactions, fills missing defaults such as gas and fees when possible, builds the final execution plan, and returns both the prepared transactions and plan. Sign and submit exactly what this method returns.

### Parameters

Position 1: `proposal` (object, required)

`proposal` top-level fields:

| Field | Type | Required | Short description |
| --- | --- | --- | --- |
| `blockRange` | object | No | Allowed block window. If omitted, server chooses a default future range. |
| `steps` | array | Yes | Root execution steps or groups. |
| `oneOf` | boolean | No | Root-level alternative-branch behavior. |
| `tolerateFailures` | boolean | No | Root-level continue-on-failure behavior. |

For nested groups, use Proposal/Plan Group Fields.

For transaction leaf steps, use Proposal Leaf Step Fields plus full Transaction Object Fields.

### Returns

Object with:

| Field | Type | Short description |
| --- | --- | --- |
| `transactions` | array of transaction objects | Prepared transaction args in depth-first order. |
| `executionPlan` | object | Composable execution plan matching `transactions`. |

`executionPlan` fields:

- `blockRange` object (see Block Range Object)
- recursive `steps`, `oneOf`, `tolerateFailures` (see Proposal/Plan Group Fields)
- leaf nodes with Composable Plan Leaf Fields

### Example Request

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "sonic_prepareBundle",
  "params": [
    {
      "blockRange": {
        "first": "0xbc614f",
        "length": "0x10"
      },
      "steps": [
        {
          "from": "0x1111111111111111111111111111111111111111",
          "to": "0x2222222222222222222222222222222222222222",
          "nonce": "0x10",
          "value": "0x0",
          "input": "0x",
          "maxFeePerGas": "0x3b9aca00",
          "maxPriorityFeePerGas": "0x3b9aca00"
        },
        {
          "oneOf": true,
          "steps": [
            {
              "tolerateFailed": true,
              "from": "0x1111111111111111111111111111111111111111",
              "to": "0x3333333333333333333333333333333333333333",
              "nonce": "0x11",
              "value": "0x0",
              "input": "0x",
              "maxFeePerGas": "0x3b9aca00",
              "maxPriorityFeePerGas": "0x3b9aca00"
            }
          ]
        }
      ]
    }
  ]
}
```

### Example Response

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "transactions": [
      {
        "from": "0x1111111111111111111111111111111111111111",
        "to": "0x2222222222222222222222222222222222222222",
        "nonce": "0x10",
        "gas": "0x7530",
        "maxFeePerGas": "0x3b9aca00",
        "maxPriorityFeePerGas": "0x3b9aca00",
        "input": "0x",
        "accessList": [
          {
            "address": "0x0000000000000000000000000000000000000000",
            "storageKeys": [
              "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
            ]
          }
        ]
      }
    ],
    "executionPlan": {
      "blockRange": {
        "first": "0xbc614f",
        "length": "0x10"
      },
      "steps": [
        {
          "from": "0x1111111111111111111111111111111111111111",
          "hash": "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
        }
      ]
    }
  }
}
```

## sonic_submitBundle

This method submits a prepared bundle to the network. It expects signed transactions and the matching execution plan returned from `sonic_prepareBundle`. On success, it returns the execution plan hash used for tracking.

### Parameters

Position 1: `bundle` (object, required)

| Field | Type | Required | Short description |
| --- | --- | --- | --- |
| `signedTransactions` | array of hex bytes | Yes | Signed raw transactions, ordered to match the plan references. |
| `executionPlan` | object | Yes | Execution plan returned by `sonic_prepareBundle`. |

`executionPlan` full structure:

- `blockRange` object (see Block Range Object)
- recursive group fields: `steps`, `oneOf`, `tolerateFailures` (see Proposal/Plan Group Fields)
- leaf fields: `from`, `hash`, `tolerateFailed`, `tolerateInvalid` (see Composable Plan Leaf Fields)

### Returns

A single hash value:

- execution plan hash (`0x...`) used by `sonic_getBundleInfo`.

### Example Request

```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "method": "sonic_submitBundle",
  "params": [
    {
      "signedTransactions": [
        "0x02f86c82053910843b9aca00843b9aca008275309422222222222222222222222222222222222222228080c0"
      ],
      "executionPlan": {
        "blockRange": {
          "first": "0xbc614f",
          "length": "0x10"
        },
        "steps": [
          {
            "from": "0x1111111111111111111111111111111111111111",
            "hash": "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
          }
        ]
      }
    }
  ]
}
```

### Example Response

```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "result": "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
}
```
