#!/bin/bash

REPETITIONS=30

SIZE_FACTOR=1   go test -run=none -bench "^(BenchmarkDagProcessorQueue_.*)$" ./ -v  -count $REPETITIONS | tee vanilla.txt
SIZE_FACTOR=2   go test -run=none -bench "^(BenchmarkDagProcessorQueue_.*)$" ./ -v  -count $REPETITIONS | tee factor2.txt
SIZE_FACTOR=5   go test -run=none -bench "^(BenchmarkDagProcessorQueue_.*)$" ./ -v  -count $REPETITIONS | tee factor5.txt

benchstat  vanilla.txt factor2.txt factor5.txt  