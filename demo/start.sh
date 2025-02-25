#!/usr/bin/env bash
cd $(dirname $0)
. ./_params.sh

set -e

echo -e "\nStart $N nodes:\n"
pushd ..
make all && mv ./build/sonicd ./build/demo_sonicd
popd

rm -f ./transactions.rlp
for ((i=0;i<$N;i+=1))
do
    DATADIR="${PWD}/sonic$i.datadir"
    PORT=$(($PORT_BASE+$i))
    RPCP=$(($RPCP_BASE+$i))
    WSP=$(($WSP_BASE+$i))
    ACC=$(($i+1))

    if [ ! -e ${DATADIR} ]
    then
        echo "Import fake genesis ${ACC}/${N}"
        mkdir -p ${DATADIR}
        ../build/sonictool --datadir=${DATADIR} genesis fake ${N} --mode=rpc
    fi

    (../build/demo_sonicd \
	--datadir=${DATADIR} \
	--mode=rpc \
	--fakenet=${ACC}/$N \
	--port=${PORT} \
	--nat extip:127.0.0.1 \
	--http --http.addr="127.0.0.1" --http.port=${RPCP} --http.corsdomain="*" --http.api="eth,debug,net,admin,web3,personal,txpool,ftm,dag,sonic" \
	--ws --ws.addr="127.0.0.1" --ws.port=${WSP} --ws.origins="*" --ws.api="eth,debug,net,admin,web3,personal,txpool,ftm,dag,sonic" \
	--metrics --metrics.addr=127.0.0.1 --metrics.port=$(($RPCP+1100)) \
	--verbosity=3 >> sonicd$i.log 2>&1)&

    echo -e "\tnode$i ok"
done

echo -e "\nConnect nodes to ring:\n"
for ((i=0;i<$N;i+=1))
do
    for ((n=0;n<$M;n+=1))
    do
        j=$(((i+n+1) % N))

	enode=$(attach_and_exec $j 'admin.nodeInfo.enode')
        echo "    p2p address = ${enode}"

        echo " connecting node-$i to node-$j:"
        res=$(attach_and_exec $i "admin.addPeer(${enode})")
        echo "    result = ${res}"
    done
done

for ((i=0;i<$N;i+=1))
do
  echo "Node $i peers:"
  attach_and_exec $i 'admin.peers'
done
