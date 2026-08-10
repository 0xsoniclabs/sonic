# Priority Race Results

**TL;DR** — A prioritized transaction wins 99.8–100% of races against an identical
ordinary one at any stake, network size, or moderate load. Under heavy congestion — the
case where it matters most — it stops working: neither transaction is included in time.
The problem lives outside the replicated code and can be worked on separately, one solution
would be to use private pools for prioritized transactions.

## Setup

Every scenario runs on its own local integration test net with multiple
validators. The priority registry's per-entity rate limits are set to their
maximum so that throttling cannot interfere, and the emission intervals use
production values, since the integration test harness otherwise accelerates
event production far beyond the rate of a real network.

A single race uses two freshly created accounts, one registered in the priority
registry and one ordinary. Each sends a single nonce-0 value transfer of 21000
gas paying the same fixed fee, well above the base fee even under congestion, so
neither fee nor nonce position can separate them — only the priority lane can.
The two transactions are submitted over ordinary RPC, each to the node its variant
names, and released from a barrier that opens only once both sender goroutines have reached
it, so neither is delayed by the other's start-up. Both receipts are then awaited
independently for up to 5 s, the point past which prioritization would no longer be
meaningful; a transaction that does not make it counts as missing. Accounts are created
fresh per race so that each trial's validator-turn assignment is independent, and all of a
scenario's accounts are funded up front so that setup traffic, which pays the suggested gas
price, never competes with the congestion. Each variant runs 3000 races (500 under
congestion), up to ten of them concurrently.

## Columns

- **win rate** — share of races the prioritized transaction won, by landing in an earlier
  block, by landing in the same block at a lower transaction index, or by being included
  when the ordinary transaction was not. Races where neither was included count as losses.
- **earlier block** — races won on block number.
- **same block, earlier tx** — races where both landed in the same block and the
  prioritized one came first.
- **later** — races where both were included but the ordinary transaction came first; the
  only outright ordering losses.
- **prio missing**, **ord missing** — races where that transaction missed the 5 s deadline
  while the other one was included.
- **both missing** — races where neither was included in time.

The six count columns together sum to a row's number of races.

## Validator targeting — 32 validators, half stake 100, half stake 1

The validators are split into a high-stake and a low-stake half: the sixteen high-stake
ones hold a hundred times the stake of the sixteen low-stake ones, so they take
almost every proposing turn, while the low-stake ones rarely propose. Running all
four high/low combinations separates the effect of reaching a frequently-proposing
validator from the priority lane itself.

- **prio**, **ord** — whether that transaction was submitted to a high-stake or a low-stake
  validator.

| prio | ord | win rate | earlier block | same block, earlier tx | later | prio missing | ord missing | both missing |
|---|---|---|---|---|---|---|---|---|
| high | high | 99.9% | 1138 | 1855 | 0 | 0 | 3 | 4 |
| low | low | 100.0% | 1069 | 1930 | 1 | 0 | 0 | 0 |
| high | low | 99.9% | 973 | 2025 | 0 | 0 | 0 | 2 |
| low | high | 99.8% | 950 | 2039 | 6 | 0 | 5 | 0 |

## Validator counts — equal stakes

The network size is the variable, doubling at each step from 1 to 32. It decides how often
a given node proposes.

Each count runs two targeting variants: `same` submits both transactions to one node, so
only that node's lane ordering separates them, while `different` sends the ordinary one
elsewhere and adds gossip propagation to the race.

- **validators** — number of equally staked validators, one node each.
- **variant** — whether both transactions were submitted to the same node, or the ordinary
  one to a different node.

| validators | variant | win rate | earlier block | same block, earlier tx | later | prio missing | ord missing | both missing |
|---|---|---|---|---|---|---|---|---|
| 1 | same | 100.0% | 0 | 3000 | 0 | 0 | 0 | 0 |
| 2 | same | 100.0% | 1313 | 1687 | 0 | 0 | 0 | 0 |
| 2 | different | 100.0% | 1240 | 1760 | 0 | 0 | 0 | 0 |
| 4 | same | 99.8% | 742 | 2251 | 6 | 0 | 0 | 1 |
| 4 | different | 100.0% | 678 | 2321 | 1 | 0 | 0 | 0 |
| 8 | same | 100.0% | 1126 | 1868 | 1 | 0 | 5 | 0 |
| 8 | different | 100.0% | 1047 | 1951 | 0 | 0 | 2 | 0 |
| 16 | same | 100.0% | 690 | 2308 | 0 | 0 | 2 | 0 |
| 16 | different | 100.0% | 1173 | 1824 | 0 | 0 | 3 | 0 |
| 32 | same | 100.0% | 753 | 2243 | 0 | 0 | 4 | 0 |
| 32 | different | 99.8% | 746 | 2248 | 1 | 0 | 1 | 4 |

## Proposer mode — 32 equal validators

Both modes run on an otherwise identical network. Legacy mode derives blocks from the event
DAG all validators emit into; single proposer has one validator assemble the block, so both
transactions almost always land in the same one and the lane decides only their index.

- **mode** — block formation mode: every validator proposes in turn (legacy), or one
  validator builds the block (single proposer).

| mode | variant | win rate | earlier block | same block, earlier tx | later | prio missing | ord missing | both missing |
|---|---|---|---|---|---|---|---|---|
| legacy | same | 100.0% | 1230 | 1769 | 0 | 0 | 1 | 0 |
| legacy | different | 100.0% | 1161 | 1831 | 1 | 0 | 7 | 0 |
| single proposer | same | 77.2% | 2 | 2310 | 4 | 6 | 4 | 674 |
| single proposer | different | 82.8% | 0 | 2481 | 0 | 7 | 3 | 509 |

## Under congestion — 32 equal validators, 500 races

The same two targeting variants run while extra accounts, spread across the nodes, flood
the network with ordinary transfers at the racers' gas price for the whole duration. More
senders means more load, and past some point inclusion itself becomes contested rather than
just ordering.

- **senders** — number of accounts flooding the network while the races run.

| senders | variant | win rate | earlier block | same block, earlier tx | later | prio missing | ord missing | both missing |
|---|---|---|---|---|---|---|---|---|
| 1 | same | 100.0% | 207 | 293 | 0 | 0 | 0 | 0 |
| 1 | different | 99.2% | 152 | 324 | 4 | 0 | 20 | 0 |
| 2 | same | 100.0% | 184 | 301 | 0 | 0 | 15 | 0 |
| 2 | different | 99.8% | 133 | 347 | 0 | 0 | 19 | 1 |
| 4 | same | 100.0% | 118 | 322 | 0 | 0 | 60 | 0 |
| 4 | different | 99.8% | 1 | 7 | 0 | 0 | 491 | 1 |
| 8 | same | 98.0% | 112 | 254 | 0 | 0 | 124 | 10 |
| 8 | different | 98.4% | 5 | 11 | 4 | 0 | 476 | 4 |
| 16 | same | 1.8% | 0 | 1 | 0 | 0 | 8 | 491 |
| 16 | different | 38.8% | 0 | 7 | 0 | 0 | 187 | 306 |
| 32 | same | 0.0% | 0 | 0 | 0 | 0 | 0 | 500 |
| 32 | different | 0.0% | 0 | 0 | 0 | 0 | 0 | 500 |

## Observations

- **Targeting and network size do not matter.** 99.8–100% in every legacy-mode variant,
  from 1 to 32 validators and in all four high/low stake combinations. Most wins are
  intra-block.
- **Ordering is near-perfect wherever both transactions are included.** Across all
  scenarios the ordinary transaction came first in well under 0.5% of such races.
- **Single-proposer mode loses only on latency.** Cross-block wins all but vanish, as
  expected, and only 4 races out of 6000 are ordering losses — but in 17–22% neither
  transaction was included within 5 s, dragging the win rate to 77–83%.
- **Congestion hits the ordinary transaction first, and across nodes.** At 4–8 senders it
  missed the deadline in ~490 of 500 cross-node races against 60–124 same-node, while the
  prioritized one still made it in ~99% either way. The different_validators variants
  also ran ~250 s against ~159–173 s, pointing at gossip propagation.
- **Past 8 senders prioritization delivers nothing, exactly where it would be worth the
  most.** At 16 senders most races end with both transactions missing, at 32 senders every
  single one does. Heavy load is the case a priority lane is meant to cut through, and
  instead it degrades with the network: the lane cannot order what no block includes.
