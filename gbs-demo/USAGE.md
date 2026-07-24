# Sonic Transaction-Priorities Demo

A single Docker container that runs a complete, self-contained Sonic network of
**5 validators** with the **transaction-priorities** feature enabled. It comes
with a set of pre-funded accounts so you can try the feature immediately.

The transaction-priorities feature lets the network designate a subset of
transactions that are scheduled **ahead of everything else** in a block. Which
transactions get priority is decided by an on-chain **priority registry**
contract that anyone can configure on this demo network.

## 1. Prerequisites

- **Docker**, including the Compose plugin — install Docker Desktop (macOS /
  Windows) or Docker Engine (Linux): <https://docs.docker.com/get-docker/>.
  Compose ships with both; verify with `docker compose version`.
- **git** — to fetch the code: <https://git-scm.com/downloads>.

The [§8, Try it out](#8-try-it-out) walkthrough uses Foundry's `cast` and `jq`.
**Both are already bundled in the container**, so the simplest path needs
nothing beyond Docker: open a shell inside the running container and run the
commands there.
If you prefer to run them from your host instead? Install:

- **Foundry** (provides the `cast` CLI): <https://getfoundry.sh/>.
- **jq** — to filter the JSON that `cast` prints: <https://jqlang.github.io/jq/>.
- and use a POSIX shell — macOS Terminal (zsh/bash) and Linux work as-is; on
  **Windows**, run the snippets in **WSL**.

## 2. Get the code

Clone the repository and check out the demo branch:

```sh
git clone --branch lorenz/gbs-demo https://github.com/0xsoniclabs/sonic.git
cd sonic
```

All commands and paths in this guide are relative to this `sonic/` directory.

## 3. Start the network

From the repository root, switch into the **`gbs-demo/`** directory (where
`docker-compose.yml` lives) and start it — this builds the image and maps all
five RPC ports plus WebSocket for you:

```sh
cd gbs-demo
docker compose up --build
```

On startup the container prints a summary similar to this:

```
==============================================================================
  Sonic transaction-priorities demo network is up
==============================================================================

  Validators           : 5 (all producing blocks in this container)
  Chain ID             : 4003 (0xfa3)
  RPC (HTTP)           : http://localhost:18545   (validators 1..5 on 18545..18549)
  RPC (WebSocket)      : ws://localhost:18645
  Priority registry    : 0x7072696f72697479000000000000000000000000

  Pre-funded demo accounts (1000000000 S each):
  ...
```

The RPC endpoint is now available at **http://localhost:18545**. All five
validators serve RPC (ports `18545`–`18549`); any of them works. Leave the
container running in this terminal.

## 4. Configuration (optional)

The number of validators, the number of demo accounts, and their balance are all
configurable via environment variables:

| Variable | Default | Meaning |
|----------|---------|---------|
| `VALIDATORS` | `5` | Number of validators in the network |
| `DEMO_USERS` | `10` | Number of pre-funded demo accounts |
| `DEMO_BALANCE` | `1000000000` | Balance per demo account, in whole `S` |

Set them in whichever way suits your shell:

- **Edit the `environment:` block** in `docker-compose.yml` directly.

- **Inline prefix**, in a POSIX shell (macOS/Linux, or WSL on Windows):

  ```sh
  VALIDATORS=3 DEMO_USERS=20 DEMO_BALANCE=500 docker compose up --build
  ```

> The Compose port mapping exposes ports for 5 validators (`18545`–`18549`). With
> fewer validators the surplus mappings simply go unused; if you raise
> `VALIDATORS` above 5, add the extra `-<port>:<port>` mappings for them.

## 5. What you get

| | |
|---|---|
| **RPC (HTTP)** | `http://localhost:18545` |
| **RPC (WebSocket)** | `ws://localhost:18645` |
| **Chain ID** | `4003` |
| **Currency** | `S` (18 decimals) |
| **Priority registry** | `0x7072696f72697479000000000000000000000000` |

### Pre-funded accounts

Each account below starts with **1,000,000,000 S**. These are well-known
**test keys** — never use them on a real network.

| # | Address | Private key |
|---|---------|-------------|
| 1 | `0xB3250CbB5942c375675B3e44796e205025e82b72` | `0x6e50dbd3e81b22424cb230133b87bc9ef0f17c584a2a5dc4b212d2b83b5ee084` |
| 2 | `0xe7fe58d73407aedefF96Fe3413FEf103331E98Ad` | `0x2215aaee06a2d64ca32b201e1fb9d1e3c7a25d45a6d8b0de6300ba3a20e42ef5` |
| 3 | `0x1A4AD87873A470B8f1caA169022E11Da6a2bCA4C` | `0x1cd6fdfc633c0fa73bd306c46eecd23096365b44ab75f0e6fa04dc2adbea9583` |
| 4 | `0xa47CBDbCB7b77eeC04A06b73A1deb1C7dbB055c2` | `0x2fc91d5829f44650c32ba92c8b29d511511446b91badf03b1fd0f808b91a4b5b` |
| 5 | `0x05e71027e7d3bd6261de7634cf50F0e2142067C4` | `0x6aeeb7f09e757baa9d3935a042c3d0d46a2eda19e9b676283dce4eaf32e29dc9` |
| 6 | `0xA298Fc05bccff341f340a11FffA30567a00e651f` | `0x7d51a817ee07c3f28581c47a5072142193337fdca4d7911e58c5af2d03895d1a` |
| 7 | `0x55Ca8305745BC2cF5137452813dba9e41b1c8cB3` | `0x59963733b8a6fb1c6eeb1ce51c7e6046e652a9bcacd4cbaa3f6f26dafe7f79f7` |
| 8 | `0x28342c2826fB1B53b0C5980cb85b06563011Be7D` | `0x4cf757812428b0764a871e94b02ba026a5d3738e69f7d1d4f9f93b43ed00e820` |
| 9 | `0xf36395e2c56EDfb768Cd9c961C0ecfdB7cB9A5Fe` | `0xa80a59dc6a9be8003a696ed08a4d37d5046f66201912b40c224d4fe96b515231` |
| 10 | `0x4B576877395aD86011C7C271070738733F0f4328` | `0xa2ef6534312d205b045a94ec2e9d49191a6d17702671d51dd88a9e2837b612ce` |

The exact list (with any custom count/balance) is also printed on startup and
written to `/data/accounts.json` inside the container.

## 6. Connect a wallet

To use the network from MetaMask (or any EVM wallet), add a custom network:

- **Network name:** Sonic Priorities Demo
- **RPC URL:** `http://localhost:18545`
- **Chain ID:** `4003`
- **Currency symbol:** `S`

Then import one of the private keys above to get a funded account.

## 7. How priorities work

1. The node keeps an on-chain **priority registry** contract (at the address
   above). For every transaction it is about to include in a block, the node
   asks the registry: *"what priority does this transaction have?"*
2. The registry returns a **level** (0 = no priority, higher = more important), a
   **weight** (tie-breaker within a level), and an **entity id** (used for
   fair-share rate limiting between different entities).
3. Transactions with a level greater than zero are moved to the **front of the
   block**, ordered by `(level desc, weight desc)`. Everything else keeps its
   normal ordering behind them.

On this demo network the registry decides priority **by sender address**: once
you register an address, *every* ordinary transaction from that address is
automatically prioritized — no special transaction format is required.

> The registry deployed here is a permissionless **stand-in** for testing:
> anyone can call its configuration methods. A production registry would be
> governed/access-controlled; the node only depends on the ABI, not on who may
> call it.

### Registry interface

```solidity
// Register (or update) the priority of a sender address.
function setSenderPriority(address from, uint64 level, uint64 weight, uint128 id) external;

// Optional: transactions whose gas limit exceeds this are never prioritized (0 = no filter).
function setMaxGas(uint256 g) external;

// Optional: per-entity limits — max gas per entity per block, and per event.
function setConfig(uint256 perBlockGas, uint256 perEvent) external;

// Read the priority currently assigned to a sender.
function senderPriority(address) external view returns (uint64 level, uint64 weight, uint128 id);
```

## 8. Try it out

Interact with the network using [Foundry](https://getfoundry.sh/)'s `cast`. The
steps below register demo account #1 as a prioritized sender, generate load, and
show the prioritized transaction landing at the front of its block.

**Where to run these commands** — either works, and both target the same
`http://localhost:18545` endpoint:

- **Inside the container (no host install).** `cast` and `jq` are bundled, so
  Docker is all you need. With the network running (§3), open a shell in the
  container from `gbs-demo/`:

  ```sh
  docker compose exec sonic-priorities bash
  ```

  Then run every command below in that shell.

- **On your host.** Install Foundry and `jq` (see §1) and use a POSIX shell (WSL
  on Windows). Run the commands directly.

Steps:

1. **Set some shorthands** (uses demo account #1 as the prioritized sender):

   ```sh
   export RPC=http://localhost:18545
   export REGISTRY=0x7072696f72697479000000000000000000000000
   export PRIO_KEY=0x6e50dbd3e81b22424cb230133b87bc9ef0f17c584a2a5dc4b212d2b83b5ee084
   export PRIO_ADDR=0xB3250CbB5942c375675B3e44796e205025e82b72
   ```

2. **Discover the registry address** from the node itself (optional — it is the
   value above):

   ```sh
   cast rpc --rpc-url $RPC eth_config | jq '.current.systemContracts'
   ```

3. **Register account #1 as high-priority** (level 10, weight 1, entity id 1):

   ```sh
   cast send --rpc-url $RPC --private-key $PRIO_KEY $REGISTRY \
     "setSenderPriority(address,uint64,uint64,uint128)" $PRIO_ADDR 10 1 1
   ```

   Verify it:

   ```sh
   cast call --rpc-url $RPC $REGISTRY "senderPriority(address)(uint64,uint64,uint128)" $PRIO_ADDR
   ```

4. **Generate real load together with the prioritized transaction.** To submit
   them all at (nearly) the same time, run each `cast send` in the **background**
   with `&`, then `wait`:

   ```sh
   # ~10 ordinary transfers from account #2, all submitted concurrently
   ORD_KEY=0x2215aaee06a2d64ca32b201e1fb9d1e3c7a25d45a6d8b0de6300ba3a20e42ef5
   NONCE=$(cast nonce --rpc-url $RPC "$(cast wallet address --private-key $ORD_KEY)")
   for i in $(seq 0 9); do
     cast send --rpc-url $RPC --private-key $ORD_KEY \
       --nonce $((NONCE + i)) \
       0x000000000000000000000000000000000000dEaD --value 1wei > /dev/null &
   done

   # the prioritized transfer (account #1), fired into the same wave
   cast send --rpc-url $RPC --private-key $PRIO_KEY \
     0x000000000000000000000000000000000000dEaD --value 1wei &

   wait   # block until every transfer is mined
   ```

5. **Inspect the block order.** Take a block number that included both and list
   its transactions; the prioritized sender's transactions (usually) come first:

   ```sh
   cast block <number> --rpc-url $RPC --json | jq '.transactions'
   # then look up each tx's sender with: cast tx <hash> from --rpc-url $RPC
   ```

## 9. Stop / reset / remove

Run these from `gbs-demo/`:

- **Stop:** press `Ctrl+C` in the container's terminal to stop it in place, or
  from another terminal `docker compose stop`.
- **Remove the container:** `docker compose down`. This stops and deletes the
  container (and its default network); the built image is kept for next time.
- **Reset:** the network is recreated from a fresh genesis on every start, so
  just start it again. (The demo account addresses and keys are always the same.)
- **Full cleanup:** `docker compose down --rmi local --volumes` also removes the
  demo image and any data volume, leaving nothing behind.

## 10. Troubleshooting

- **`connection refused` on port 18545** — give the container a few seconds to
  finish importing genesis and peering; it prints the summary banner once RPC is
  ready.
- **Blocks don't advance** — the network only produces blocks when there is
  activity (plus periodic empty blocks). Send a transaction and it will proceed.
- **A demo transaction is not prioritized** — make sure the sender was
  registered with a non-zero `level`, and that the transaction's gas limit does
  not exceed `maxGas` (unset/`0` by default, i.e. no filter).
