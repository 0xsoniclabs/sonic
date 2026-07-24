#!/usr/bin/env bash
#
# Boots a self-contained Sonic fake-net with N validators, all running inside
# this single container, with the transaction-priorities feature enabled and a
# set of pre-funded demo accounts. Prints the RPC endpoints and account keys,
# then stays in the foreground streaming the validator logs.
set -euo pipefail

N="${VALIDATORS:-5}"
USERS="${DEMO_USERS:-10}"
BALANCE="${DEMO_BALANCE:-1000000000}"
WORKDIR="${WORKDIR:-/data}"
HTTP_BASE="${HTTP_PORT_BASE:-18545}"
WS_BASE="${WS_PORT_BASE:-18645}"
P2P_BASE="${P2P_PORT_BASE:-5050}"
RING_PEERS="${RING_PEERS:-2}"
API="eth,net,web3,admin,txpool,dag"

GENESIS="$WORKDIR/genesis.json"
ACCOUNTS="$WORKDIR/accounts.json"
PRIORITY_REGISTRY="0x7072696f72697479000000000000000000000000"

mkdir -p "$WORKDIR"

rpc() { # rpc <port> <json-body>
    curl -s -X POST "http://127.0.0.1:$1" -H 'Content-Type: application/json' --data "$2"
}

pids=()
cleanup() {
    echo -e "\nShutting down validators..."
    kill "${pids[@]}" 2>/dev/null || true
    wait 2>/dev/null || true
}
trap cleanup SIGINT SIGTERM

# 1. Generate the genesis and the demo account list (once per data dir).
if [ ! -f "$GENESIS" ]; then
    echo "Generating genesis for $N validators and $USERS demo accounts..."
    genesisgen -validators "$N" -users "$USERS" -balance "$BALANCE" \
        -out "$GENESIS" -accounts "$ACCOUNTS"
fi

# 2. Initialize a data dir per node and launch it.
for ((i = 0; i < N; i++)); do
    datadir="$WORKDIR/node$i"
    http=$((HTTP_BASE + i)); ws=$((WS_BASE + i)); p2p=$((P2P_BASE + i))
    validator=$((i + 1))

    if [ ! -d "$datadir" ]; then
        echo "Importing genesis for node $i (validator $validator/$N)..."
        sonictool --datadir "$datadir" genesis json --experimental --mode=rpc "$GENESIS" \
            > "$WORKDIR/node$i.import.log" 2>&1
    fi

    sonicd --datadir "$datadir" --mode rpc --fakenet "$validator/$N" \
        --port "$p2p" --nat extip:127.0.0.1 \
        --http --http.addr 0.0.0.0 --http.port "$http" --http.corsdomain '*' --http.vhosts '*' --http.api "$API" \
        --ws --ws.addr 0.0.0.0 --ws.port "$ws" --ws.origins '*' --ws.api "$API" \
        --verbosity 2 > "$WORKDIR/node$i.log" 2>&1 &
    pids+=($!)
done

# 3. Wait for every RPC endpoint to answer.
echo "Waiting for validators to come up..."
for ((i = 0; i < N; i++)); do
    port=$((HTTP_BASE + i))
    for attempt in $(seq 60); do
        if rpc "$port" '{"jsonrpc":"2.0","id":1,"method":"web3_clientVersion","params":[]}' | grep -q '"result"'; then
            break
        fi
        if [ "$attempt" -eq 60 ]; then
            echo "node $i did not become ready; see $WORKDIR/node$i.log" >&2
            cleanup; exit 1
        fi
        sleep 1
    done
done

# 4. Connect the validators into a ring so consensus can proceed.
declare -a enodes
for ((i = 0; i < N; i++)); do
    enodes[i]=$(rpc $((HTTP_BASE + i)) '{"jsonrpc":"2.0","id":1,"method":"admin_nodeInfo","params":[]}' | jq -r '.result.enode')
done
for ((i = 0; i < N; i++)); do
    for ((k = 1; k <= RING_PEERS; k++)); do
        j=$(((i + k) % N))
        rpc $((HTTP_BASE + i)) \
            "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"admin_addPeer\",\"params\":[\"${enodes[j]}\"]}" > /dev/null
    done
done

# 5. Print a summary for the user.
chain_id=$(rpc "$HTTP_BASE" '{"jsonrpc":"2.0","id":1,"method":"eth_chainId","params":[]}' | jq -r '.result')
chain_id_dec=$((chain_id))

cat <<EOF

==============================================================================
  Sonic transaction-priorities demo network is up
==============================================================================

  Validators           : $N (all producing blocks in this container)
  Chain ID             : $chain_id_dec ($chain_id)
  RPC (HTTP)           : http://localhost:$HTTP_BASE   (validators 1..$N on $HTTP_BASE..$((HTTP_BASE + N - 1)))
  RPC (WebSocket)      : ws://localhost:$WS_BASE
  Priority registry    : $PRIORITY_REGISTRY

  Pre-funded demo accounts ($BALANCE S each):
------------------------------------------------------------------------------
EOF
jq -r '.[] | "  \(.name)\n    address     : \(.address)\n    private key : \(.privateKey)"' "$ACCOUNTS"
cat <<EOF
------------------------------------------------------------------------------
  These are well-known TEST keys. Never use them on a real network.

  Next steps: see the usage guide (USAGE.md) shipped with this demo.
==============================================================================

EOF

# 6. Stream logs and keep the container alive until the nodes exit.
tail -n 0 -F "$WORKDIR"/node*.log &
pids+=($!)
wait -n "${pids[@]}"
cleanup
