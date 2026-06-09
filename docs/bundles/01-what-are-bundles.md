# What Are Bundles and Why Do They Matter?

Decentralized applications carry a hidden tax: every on-chain interaction demands a wallet confirmation. Approve tokens -- click. Confirm the swap -- click. Stake the result -- click. For multi-step workflows the ritual can run through five confirmations, and each pause is an invitation for the world to change underneath you.

Sonic **transaction bundles** address both problems -- the friction of repeated signing, and the fragility of operations that span multiple blocks -- through a single mechanism: a group of transactions that the network treats as one.

---

## Problem One: The Multi-Signature Tax

Consider buying a complete five-card set of onchain collectible cards. Each purchase is its own transaction. By the time you confirm the fourth, the fifth item may have sold out or the seller may have changed the price. You are now the proud owner of four-fifths of a set you no longer want (an incomplete set is worth far less than the cards in it cost), with no clean way to undo the preceding purchases.

The same pattern appears throughout DeFi. Moving a loan from one protocol to another typically requires at least five steps. Each step is a separate transaction. Each transaction is a separate confirmation. Each confirmation is a moment where the market can move, a price can spike, or liquidity can disappear.

Even when nothing goes wrong, the signing ceremony is exhausting. For protocols that want to attract mainstream users, every extra wallet popup is a conversion risk.

## Problem Two: The World Moves Between Your Steps

The multi-signature problem is partly about UX. The deeper problem is about correctness.

Imagine you are refinancing a loan: you close your position on Protocol A, and then open a new one on Protocol B. Between those two on-chain steps -- even if they are in the same block -- other transactions can execute. A price oracle can update. Liquidity can shift. The conditions that made the refinancing worthwhile may no longer hold at the moment you try to open the new position.

With ordinary transactions, you cannot prevent this. Your sequence of actions has gaps, and those gaps are exploitable -- by MEV bots, by market movement, or simply by bad timing.

**Atomicity** closes those gaps. If all steps execute as an uninterrupted unit -- or not at all -- the mid-sequence exposure disappears.

---

## Bundles: Transactions as a Script

A bundle is a group of transactions with an **execution plan** that tells the network how to run them. The plan can be as simple as "run these five transactions in order, and revert all of them if any one fails." It can also be more expressive: "try option A first; if that fails, try option B."

From the network's point of view, a bundle arrives as a single envelope transaction. The network unpacks it, validates the structure, runs the steps in order within one block, and either applies all the resulting state changes or reverts all of them.

No interleaving. No partial execution. No mid-sequence exposure.

---

## Two Ways to Group: AllOf and OneOf

Bundles support two grouping semantics, which can be combined and nested freely.

### AllOf -- Everything or Nothing

An **AllOf** group succeeds only if every step inside it succeeds. If any step fails or reverts, the entire group is rolled back as though it never happened.

```
AllOf(
  buy_card_1,
  buy_card_2,
  buy_card_3,
  buy_card_4,
  buy_card_5
)
```

If `buy_card_3` reverts because the item sold out, the purchases of cards 1 and 2 are also undone. You either get the complete set or nothing.

### OneOf -- First Success Wins

A **OneOf** group tries its steps in order and succeeds as soon as one of them succeeds. Subsequent steps are skipped.

```
OneOf(
  trade_on_exchange_A,  // best rate
  trade_on_exchange_B,  // acceptable rate
  trade_on_exchange_C   // fallback rate
)
```

If Exchange A has enough liquidity, the trade executes there and Exchange B and C are never touched. If A fails, its state changes are reverted and B is tried next. Failed branches that were attempted still consume their nonce and gas, and they appear in the block with a revert status -- only their state changes are undone. Branches that were never reached leave no trace at all.

The one exception: if **all** branches fail, none of the transactions land in the block and no nonces are consumed.

### Nesting

AllOf and OneOf groups can be nested. A complex workflow might combine both:

```
AllOf(
  flash_loan,
  close_position_on_protocol_A,
  withdraw_collateral,
  OneOf(
    open_position_on_protocol_B,
    open_position_on_protocol_C
  ),
  repay_flash_loan
)
```

This reads as: "Do all of these steps, in order -- but for step four, try Protocol B first, then fall back to Protocol C."

---

## Bundles in Practice

Both motivation stories -- the UX friction and the interleaving risk -- are resolved by the same underlying property: **atomic execution within a single block**.

Because the entire bundle executes atomically:

- A collector can buy a full set of cards with a single on-chain commitment, knowing they either get everything or spend nothing.
- A borrower can refinance across protocols without any moment of unsafe exposure in between.
- A trader can express a ranked list of preferred execution venues and be guaranteed that at most one trade fires, regardless of how the market moves in the same block.

And, as we will see in [Article 4](./04-one-signature.md), there is a pattern that goes further: by combining bundles with a temporary signing key managed by a web service, the user can authorize an entire multi-step workflow with a **single wallet confirmation**.

Bundled transactions appear in blocks as ordinary transactions. Block explorers, indexers, wallets, and any other tooling that reads block data require no modification -- the only place bundles touch the existing workflow is at creation time.

---

## What Comes Next

- **[Article 2: The Builder's Guide](./02-builders-guide.md)** -- The full technical picture: how to construct and submit a bundle via the `sonic_prepareBundle` and `sonic_submitBundle` APIs, how the network validates and executes bundles, and a complete reference for all configuration options.

- **[Article 3: Bundles in Action](./03-bundles-in-action.md)** -- Three concrete use cases, with bundle structures for each: Perfect Set Acquisition, Trustless Debt Refinancing, and Prioritized Fallback Trading.

- **[Article 4: One Signature, Many Transactions](./04-one-signature.md)** -- How to build workflows that require only a single user signature, using a temporary account managed by your web service.
