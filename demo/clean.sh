#!/usr/bin/env bash
cd "$(dirname "$0")" || exit

rm -rf ./sonic*.datadir tool.datadir
rm ./*.log
rm ../build/demo_sonicd
