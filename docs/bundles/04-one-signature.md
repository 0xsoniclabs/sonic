# One Signature, Many Transactions

The use cases in [Article 3](./03-bundles-in-action.md) each require the user to sign multiple transactions — one per step in the bundle. For a five-step workflow that means five wallet confirmations. That is better than the uncoordinated alternative, but it is still friction.

This article shows a pattern that reduces any multi-step workflow to a **single user signature**, regardless of how many transactions are involved. The key insight is that bundles support transactions from multiple signers. A web service can handle most of the signing — as long as the user's one signature is enough to authorize the complete plan.

---

## The Core Idea: A Temporary Account

The pattern works by introducing a **temporary account** controlled by the web service. Here is the shape of the interaction:

1. The web service creates a fresh key pair for a temporary account. This account has no prior history and no funds.
2. The user sends funds to the temporary account **as part of the bundle** — not before it.
3. The temporary account, also as part of the same bundle, uses those funds to carry out all the steps and delivers the results back to the user.
4. At the end, the temporary account is empty and its key can be discarded.

Because the fund transfer and all subsequent steps are in the same bundle, nothing can go wrong in the middle. Either the whole workflow completes — the user's funds move to the temporary account, all actions execute, and the results arrive back — or the fund transfer itself is rolled back. The user is never in a position where they have sent funds but the desired outcome did not materialise.

The user only needs to sign **one transaction**: the fund transfer to the temporary account.

---

## Why This Is Safe

A natural concern: does the user have to trust the web service?

The answer is no — and the reason is the **BundleOnly marker** described in [Article 2](./02-builders-guide.md).

When `sonic_prepareBundle` processes the proposal, it computes a hash of the entire execution plan and injects it into the access list of every transaction in the bundle. The user's fund-transfer transaction carries this hash. When the user signs that transaction, their signature cryptographically covers the plan hash.

This means:
- The user's signature is only valid for the specific plan that was presented to them.
- The web service cannot change any step in the bundle after the user has signed. Any modification would change the plan hash, which would change the unsigned transaction hash, which would invalidate the signature.
- If the user examines the execution plan before signing — which a good wallet UI will surface — they can verify exactly what will happen.

The user's single signature is not a blank cheque. It is a precise commitment to a specific execution plan.

---

## Walkthrough: Perfect Set Acquisition

Let us trace through the full workflow for the NFT set purchase from [Article 3](./03-bundles-in-action.md), using the temporary account pattern.

### Setup

The web service maintains a pool of temporary signing keys, or generates a fresh one per request. Call this address `0xTemp`.

The web service also knows:
- Which NFTs the user wants (IDs 1–5 on `0xNFTMarket`)
- The total price for all five (e.g. 5 S)
- The user's address (`0xUser`)

### Building the Proposal

The web service calls `sonic_prepareBundle` with a proposal that describes the full workflow:

```json
{
  "blockRange": { "first": "0x1234", "length": "0x0a" },
  "steps": [
    {
      "from": "0xUser",
      "to": "0xTemp",
      "nonce": "0x10",
      "value": "0x4563918244f40000",
      "chainId": "0xfa"
    },
    {
      "from": "0xTemp",
      "to": "0xNFTMarket",
      "nonce": "0x00",
      "data": "0x<buyNFT(1)>"
    },
    {
      "from": "0xTemp",
      "to": "0xNFTMarket",
      "nonce": "0x01",
      "data": "0x<buyNFT(2)>"
    },
    {
      "from": "0xTemp",
      "to": "0xNFTMarket",
      "nonce": "0x02",
      "data": "0x<buyNFT(3)>"
    },
    {
      "from": "0xTemp",
      "to": "0xNFTMarket",
      "nonce": "0x03",
      "data": "0x<buyNFT(4)>"
    },
    {
      "from": "0xTemp",
      "to": "0xNFTMarket",
      "nonce": "0x04",
      "data": "0x<buyNFT(5)>"
    },
    {
      "from": "0xTemp",
      "to": "0xUser",
      "nonce": "0x05",
      "data": "0x<transferNFT(1)>"
    },
    {
      "from": "0xTemp",
      "to": "0xUser",
      "nonce": "0x06",
      "data": "0x<transferNFT(2)>"
    },
    {
      "from": "0xTemp",
      "to": "0xUser",
      "nonce": "0x07",
      "data": "0x<transferNFT(3)>"
    },
    {
      "from": "0xTemp",
      "to": "0xUser",
      "nonce": "0x08",
      "data": "0x<transferNFT(4)>"
    },
    {
      "from": "0xTemp",
      "to": "0xUser",
      "nonce": "0x09",
      "data": "0x<transferNFT(5)>"
    }
  ]
}
```

The structure is simple: first the user funds the temporary account, then the temporary account buys each NFT, then it transfers each NFT to the user. All of this is one flat AllOf sequence — if any step fails, nothing happens.

### Presenting the Plan to the User

`sonic_prepareBundle` returns:
- A flat list of 11 prepared transactions, each with the plan hash in its access list.
- The execution plan (a structured description of what will happen).

The web service presents the plan to the user — ideally rendered in a human-readable form by the wallet UI. The user can see exactly what they are authorizing: the amounts, the recipients, the NFT IDs.

The user signs **one transaction**: the first one, which sends 5 S to `0xTemp`. They approve this in their wallet, and the web service receives the signed transaction.

### Signing the Rest

The web service signs the remaining 10 transactions (the purchases and transfers) with the private key for `0xTemp`. This key is controlled by the web service and can sign without user interaction.

### Submitting the Bundle

With all 11 signed transactions in hand, the web service calls `sonic_submitBundle`:

```json
{
  "signedTransactions": [
    "0x02f8...",  // signed by 0xUser
    "0x02f8...",  // signed by 0xTemp
    "0x02f8...",  // signed by 0xTemp
    ...
  ],
  "executionPlan": { ... }
}
```

The node wraps everything in an envelope, submits it to the mempool, and returns the plan hash.

### What the User Experiences

From the user's perspective:
1. They see a single confirmation request in their wallet showing the payment of 5 S.
2. They confirm it.
3. After the bundle executes, all five NFTs appear in their wallet.

One click. One signature. Complete set.

---

## The Security Guarantee, Restated

The temporary account pattern is trustless in a precise sense:

- The user's signed transaction commits them to the **exact plan hash** that was shown to them.
- The web service cannot change any step — including the transfer destinations, the NFT IDs, or the amounts — without producing a different plan hash, which would render the user's signature invalid.
- The bundle's AllOf semantics mean the web service cannot run only the fund-transfer step without also completing the purchases and transfers. All steps succeed together or none of them do.

The web service can fail to submit the bundle, or submit it and have it expire without inclusion. In either case the user's signed transaction is never executed on its own (the BundleOnly marker prevents it), and the user loses nothing.

---

## Beyond NFTs: Where This Pattern Applies

The temporary account pattern is general. Anywhere a workflow requires:
- **Actions on behalf of the user** — buying, swapping, bridging, staking
- **Multiple protocol interactions** — any sequence that touches more than one contract
- **User-facing simplicity** — reducing any number of confirmations to one

...the same structure applies: the user funds a temporary account as the first step in an AllOf bundle, and the web service carries out the rest.

Some examples:

**Automated DeFi portfolio rebalancing.** A web service proposes a multi-step rebalance — sell asset A, buy B and C in specific proportions, stake the result. The user approves the plan with one signature; the web service executes it atomically.

**Gasless onboarding.** A new user with only a token balance and no native S for gas can authorize a bundle in which a relayer covers gas. The relayer is reimbursed from the user's token balance within the same bundle.

**Cross-protocol yield optimization.** A yield aggregator moves funds between protocols in a sequence that may touch three or four contracts. The user sees one confirmation; the aggregator manages all the complexity.

---

## Key Takeaways

- Bundles support transactions from multiple signers in a single atomic unit.
- A web service can sign most transactions in a bundle; the user only needs to sign the one that transfers their funds.
- The BundleOnly marker cryptographically binds the user's signature to the exact execution plan — no trust assumptions beyond what the user can verify before signing.
- The temporary account pattern turns any multi-step workflow into a single-click user experience without sacrificing the atomicity or safety guarantees that bundles provide.

---

## Further Reading

- [Article 1: What Are Bundles and Why Do They Matter?](./01-what-are-bundles.md)
- [Article 2: Transaction Bundles — A Builder's Guide](./02-builders-guide.md)
- [Article 3: Bundles in Action](./03-bundles-in-action.md)
