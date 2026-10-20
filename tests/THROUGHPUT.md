# Throughput Benchmarks

`tests/throughput_bench_test.go` and `gossip/throughput_bench_test.go` measure value transfers per second:

- `BenchmarkThroughput_Processing` runs blocks through the state processor only. It times transaction processing and leaves out the state commit and block creation.
- `BenchmarkThroughput_ProcessingWithStateRoot` runs the same blocks and also times the end of each block in the state, its commit and its state root.
- `BenchmarkThroughput_Blocks` (in `gossip`) hands each block as a single confirmed event to the node's consensus callbacks. It times block formation, the EVM, the state commit with its state root, and storing the block and its receipts, without the emitter, consensus, tx pool and RPC.
- `BenchmarkThroughput_Network` runs a single-node test network under load and counts the transactions included over a 20 s window.

Each benchmark has these cases:

- `native`: native value transfers.
- `erc20`: one ERC20 transfer is an `approve` followed by a `transferFrom`.

## Run

Each case runs on its own into its own file, 8 times for processing and blocks and 4 times for the network. The case is removed from the benchmark names, so that `benchstat` compares the cases side by side, with the first one as the baseline:

```sh
# processing only, 20 blocks per run
go test ./tests -run '^$' -bench '^BenchmarkThroughput_Processing$/^native$' -benchtime 20x -count 8 | sed 's|/native||' > native.txt
go test ./tests -run '^$' -bench '^BenchmarkThroughput_Processing$/^erc20$' -benchtime 20x -count 8 | sed 's|/erc20||' > erc20.txt

# processing with state root, 20 blocks per run
go test ./tests -run '^$' -bench '^BenchmarkThroughput_ProcessingWithStateRoot$/^native$' -benchtime 20x -count 8 | sed 's|/native||' >> native.txt
go test ./tests -run '^$' -bench '^BenchmarkThroughput_ProcessingWithStateRoot$/^erc20$' -benchtime 20x -count 8 | sed 's|/erc20||' >> erc20.txt

# blocks without consensus, 20 blocks per run
go test ./gossip -run '^$' -bench '^BenchmarkThroughput_Blocks$/^native$' -benchtime 20x -count 8 | sed 's|/native||' >> native.txt
go test ./gossip -run '^$' -bench '^BenchmarkThroughput_Blocks$/^erc20$' -benchtime 20x -count 8 | sed 's|/erc20||' >> erc20.txt

# network targeting ~3 blocks/s, one network per run
go test ./tests -run '^$' -bench '^BenchmarkThroughput_Network$/^native$' -benchtime 1x -count 4 | sed 's|/native||' >> native.txt
go test ./tests -run '^$' -bench '^BenchmarkThroughput_Network$/^erc20$' -benchtime 1x -count 4 | sed 's|/erc20||' >> erc20.txt

benchstat native=native.txt erc20=erc20.txt
```

## Block rate

`networkEmitterInterval` in `throughput_bench_test.go` controls the block rate of the network benchmark. One block takes between one and three events, depending on the load, so the interval does not translate directly into blocks/s. Check the reported `blocks/s` and adjust the interval until it is about 3.
