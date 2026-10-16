---
name: verify
description: Run a single-validator sonicd fakenet and drive block production over RPC to verify changes in block processing (gossip callbacks, ledger, EVM, epoch sealing).
---

# Verify on a live fakenet

```bash
make sonicd sonictool
W=$(mktemp -d)
./build/sonictool --datadir $W/db genesis fake 1 --upgrades brio   # or sonic|allegro
./build/sonicd --datadir $W/db --fakenet 1/1 --mode rpc \
  --http --http.addr 127.0.0.1 --http.port 28545 --http.api eth,web3,net,txpool,debug,ftm \
  --ws --ws.addr 127.0.0.1 --ws.port 28546 --ws.api eth \
  --port 0 --maxpeers 0 --nodiscover > $W/node.log 2>&1 &
go build -o build/vd ./.claude/skills/verify/driver   # build/ is gitignored
```

Driver (`vd <http-url> <cmd>`, signs with fake validator key 1, which owns the driver contracts):

- `info`: head block, epoch, balance, state root
- `transfer N`: N sequential transfers, checks the recipient balance delta
- `logs <ws-url>`: deploys a LOG1 contract, calls it 3x, checks receipts, `eth_getLogs` and the WS subscription
- `burst`: 20 txs in one block, one of them failing
- `advance-epoch`: `advanceEpochs(1)` on driverauth, runs the epoch-sealing path
- `rules '<json diff>'`: `updateNetworkRules`

Also worth checking: parentHash linkage over all blocks via `eth_getBlockByNumber`, `grep -E "CRIT|ERROR|panic" node.log`, and a restart with `kill -INT` followed by a relaunch on the same datadir.

## Gotchas

- `SingleProposerBlockFormation` is a feature flag that is independent of hardforks, so no `--upgrades` value turns it on. To exercise the proposal path, run `vd rules '{"Upgrades":{"SingleProposerBlockFormation":true}}'`, then `advance-epoch` twice, and confirm with `eth_getRules ["latest"]`.
- `sfc` in `--http.api` logs a harmless "Unavailable modules" ERROR.
- `kill -9` leaves the DB dirty. `sonictool heal` repairs it, but heal rolls back to the epoch start, and the single validator then errlocks ("does not contain last emitted event", double-sign guard). This is known and intended (#446, closed unmerged), so on one validator unclean-shutdown recovery can't be driven.
