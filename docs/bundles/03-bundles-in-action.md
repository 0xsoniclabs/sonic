# Bundles in Action

The two guarantees that bundles provide — **atomic execution** and **no interleaving** — solve a surprisingly wide range of problems. This article walks through three real-world use cases that illustrate different aspects of what bundles make possible.

> **New to bundles?** Start with [Article 1](./01-what-are-bundles.md) for the conceptual introduction, or [Article 2](./02-builders-guide.md) for the full API reference.

---

## Use Case 1: Perfect Set Acquisition

### The Problem

A collector wants to buy a team of five player NFTs as a complete set. They find all five listed on a marketplace, check the total price, and start purchasing.

With individual transactions, each purchase settles independently. By the time the third or fourth confirmation lands, the fifth item might have sold to another buyer, or the seller might have raised the price. The collector is now stuck with an incomplete set that cost them most of the budget.

The natural response — "I'll be fast" — is not a solution. Even if the collector submits all five transactions in the same block, there is no guarantee all five succeed. One reverting transaction does not undo the others.

### The Bundle Solution

Wrap all five purchases in an **AllOf** group:

```json
{
  "blockRange": { "first": "0x1234", "length": "0x0a" },
  "steps": [
    { "from": "0xCollector", "to": "0xMarket", "nonce": "0x10", "data": "0x<buyNFT_1>" },
    { "from": "0xCollector", "to": "0xMarket", "nonce": "0x11", "data": "0x<buyNFT_2>" },
    { "from": "0xCollector", "to": "0xMarket", "nonce": "0x12", "data": "0x<buyNFT_3>" },
    { "from": "0xCollector", "to": "0xMarket", "nonce": "0x13", "data": "0x<buyNFT_4>" },
    { "from": "0xCollector", "to": "0xMarket", "nonce": "0x14", "data": "0x<buyNFT_5>" }
  ]
}
```

The top-level `steps` array is implicitly an **AllOf** group. If any single purchase fails — because the item sold out or the price changed — the entire bundle reverts. None of the purchases land. The collector's funds are untouched.

### What This Demonstrates

- **AllOf atomicity**: the group is an all-or-nothing proposition.
- **No partial state**: a reverting step inside AllOf rolls back everything before it in the group.
- **Predictable economics**: the collector either gets everything at the agreed price or spends nothing.

> **Going further:** This use case also pairs naturally with the single-signature pattern in [Article 4](./04-one-signature.md), where the collector only needs to sign once even across multiple purchases from different wallets.

---

## Use Case 2: Trustless Debt Refinancing

### The Problem

A borrower has a collateralized loan on Protocol A (similar to Aave) and wants to move it to Protocol B (similar to Compound) to take advantage of a lower interest rate. The steps look straightforward:

1. Flash-loan the outstanding debt amount.
2. Repay the loan on Protocol A.
3. Withdraw the collateral from Protocol A.
4. Deposit the collateral into Protocol B.
5. Borrow from Protocol B to repay the flash loan.

In practice, this sequence is dangerous without atomicity. If step 4 succeeds but step 5 fails — because Protocol B's rates have spiked or its liquidity is insufficient — the borrower has withdrawn collateral from Protocol A but cannot open the new position. They must scramble to cover the situation manually, possibly at a loss, and under time pressure since they are holding a flash loan.

The problem is not that individual steps are risky. The problem is that the sequence has no rollback mechanism if it cannot be completed.

### The Bundle Solution

Wrap the entire sequence in an **AllOf** group. If any step fails, every step before it is undone:

```json
{
  "blockRange": { "first": "0x1234", "length": "0x05" },
  "steps": [
    {
      "from": "0xBorrower",
      "to": "0xFlashLoanProvider",
      "nonce": "0x08",
      "data": "0x<flashLoan(debtAmount)>"
    },
    {
      "from": "0xBorrower",
      "to": "0xProtocolA",
      "nonce": "0x09",
      "data": "0x<repayLoan()>"
    },
    {
      "from": "0xBorrower",
      "to": "0xProtocolA",
      "nonce": "0x0a",
      "data": "0x<withdrawCollateral()>"
    },
    {
      "from": "0xBorrower",
      "to": "0xProtocolB",
      "nonce": "0x0b",
      "data": "0x<depositCollateral()>"
    },
    {
      "from": "0xBorrower",
      "to": "0xProtocolB",
      "nonce": "0x0c",
      "data": "0x<borrowAndRepayFlashLoan()>"
    }
  ]
}
```

With the bundle, the refinancing either succeeds completely or leaves the borrower exactly where they started. There is no in-between state where the old loan is closed but the new one could not be opened.

Crucially, the **no-interleaving** guarantee also matters here. Even if another transaction in the same block modifies Protocol B's interest rates, that other transaction cannot execute between the deposit in step 4 and the borrow in step 5. The bundle's steps are a single uninterrupted unit within the block.

### What This Demonstrates

- **AllOf atomicity across multiple protocols**: five contracts, one outcome.
- **No-interleaving guarantee**: other transactions in the same block cannot insert between bundle steps.
- **Safety without trust**: the borrower does not need to trust that conditions will hold between steps. If they do not, the entire operation is simply cancelled.

---

## Use Case 3: Prioritized Fallback Trading

### The Problem

A trader wants to convert a token and has identified three possible execution venues — three decentralised exchanges or cross-chain bridges — ranked by preference. Exchange A offers the best rate; Exchange B is acceptable; Exchange C is the last resort with the least favorable terms.

Without bundles, the trader must attempt each one sequentially: submit a transaction to A, wait for the receipt, check if it succeeded, then try B if not, and so on. In a fast-moving market, this iterative approach can take ten or more blocks. Prices change. Liquidity shifts. The rate on B may no longer be acceptable by the time the trader gets there.

Ideally, the trader would like to declare all three attempts at once, have them tried in order, and guarantee that **at most one** trade executes.

### The Bundle Solution

Wrap the three trade attempts in a **OneOf** group:

```json
{
  "blockRange": { "first": "0x1234", "length": "0x05" },
  "steps": [
    {
      "oneOf": true,
      "steps": [
        {
          "from": "0xTrader",
          "to": "0xExchangeA",
          "nonce": "0x15",
          "data": "0x<swap(tokenIn, tokenOut, bestRate)>"
        },
        {
          "from": "0xTrader",
          "to": "0xExchangeB",
          "nonce": "0x15",
          "data": "0x<swap(tokenIn, tokenOut, acceptableRate)>"
        },
        {
          "from": "0xTrader",
          "to": "0xExchangeC",
          "nonce": "0x15",
          "data": "0x<swap(tokenIn, tokenOut, fallbackRate)>"
        }
      ]
    }
  ]
}
```

> Note that all three transactions share the same nonce `0x15`. This is intentional: only one of them can execute; the others are discarded. The OneOf group ensures exactly this.

The network tries Exchange A first. If that transaction succeeds, the group stops — Exchange B and C are never attempted. If A fails (insufficient liquidity, slippage too high), Exchange B is tried. If B also fails, C is tried. The failed attempts are rolled back and leave no trace.

All of this happens in a **single block**. The trader does not wait through multiple block confirmations to learn the outcome. They either get the best available rate or nothing at all, and the result is known as soon as the bundle's block is confirmed.

### What This Demonstrates

- **OneOf semantics**: at most one branch executes; the rest are abandoned cleanly.
- **Single-block settlement**: the entire decision tree resolves in one block, not one per confirmation.
- **Shared nonce across branches**: since only one branch can execute, it is valid and intentional for competing transactions to share a nonce.

---

## Choosing the Right Pattern

| Goal | Pattern |
|------|---------|
| All steps must succeed, or none | `AllOf` (default) |
| Try options in preference order, stop at first success | `OneOf` |
| A step may fail without cancelling the parent | `tolerateFailed` on the step or parent group |
| A step may not exist at execution time | `tolerateInvalid` on the step |
| Complex workflow with both mandatory and optional branches | Nest `AllOf` and `OneOf` |

These patterns can be composed freely. A bundle for a sophisticated DeFi protocol might include a mandatory setup step in AllOf, followed by a OneOf for the core operation with several fallback venues, followed by a mandatory cleanup step — all in a single atomic unit.

---

## Next Up

[Article 4: One Signature, Many Transactions](./04-one-signature.md) shows how to combine bundles with a temporary signing key so that users can authorize any of these multi-step workflows with a single wallet confirmation.
