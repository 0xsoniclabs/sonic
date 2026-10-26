// Copyright 2026 Sonic Operations Ltd
// This file is part of the Sonic Client
//
// Sonic is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Sonic is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Sonic. If not, see <http://www.gnu.org/licenses/>.

package oracle

import (
	"context"
	"math/big"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Fantom-foundation/lachesis-base/utils/piecefunc"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/0xsoniclabs/sonic/utils"
)

const (
	percentilesPerStat = 20
	samplePeriod       = 1 * time.Second
	samplesKept        = int((15 * time.Second) / samplePeriod)
	maxGasToIndex      = 40_000_000
	// idleSampleEvery is how many periods the sampler waits between samples
	// while nobody asked for a suggestion.
	idleSampleEvery = 5
	// maxTransactionsToIndex bounds the transactions one sample sorts in.
	maxTransactionsToIndex = 400
)

// sample is the distribution of the tips of the pool at one moment: the
// gas of its head transactions and the price at each percentile of that gas.
type sample struct {
	totalGas    uint64
	percentiles [percentilesPerStat]*big.Int
}

// circularStats keeps the last samples and their combination.
type circularStats struct {
	mutex     sync.Mutex
	samples   [samplesKept]sample
	next      int
	activated atomic.Bool
	combined  atomic.Pointer[sample]
}

var certaintyToGasAbove = piecefunc.NewFunc([]piecefunc.Dot{
	{X: 0, Y: 50_000_000},
	{X: 0.2 * DecimalUnit, Y: 20_000_000},
	{X: 0.5 * DecimalUnit, Y: 8_000_000},
	{X: DecimalUnit, Y: 0},
})

// Sample is the run of the oracle's sampling loop: once a period, or once in
// idleSampleEvery periods while nobody asked for a suggestion, it samples the
// pool.
func (oracle *Oracle) Sample(ctx context.Context) error {
	ticker := time.NewTicker(samplePeriod)
	defer ticker.Stop()
	for tick := uint64(0); ; tick++ {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
		if !oracle.stats.activated.Load() && tick%idleSampleEvery != 0 {
			continue
		}
		if err := oracle.takeSample(ctx); err != nil {
			return err
		}
	}
}

func (oracle *Oracle) takeSample(ctx context.Context) error {
	rules, err := oracle.openRules(ctx)
	if err != nil {
		return err
	}
	oracle.stats.add(sampleOf(oracle.node.PendingHeads(), orZero(rules.Economy.MinGasPrice)))
	return nil
}

// sampleOf computes the distribution of the tips of the head transactions of
// the pool, each tip with the minimum gas price added.
func sampleOf(heads types.Transactions, minGasPrice *big.Int) sample {
	result := sample{}
	if len(heads) == 0 {
		return result
	}
	minGasPriceValue := utils.BigIntToUint256Clamped(minGasPrice)
	sorted := append(types.Transactions(nil), heads...)
	sort.Slice(sorted, func(first, second int) bool {
		comparison := sorted[first].EffectiveGasTipCmp(sorted[second], minGasPriceValue)
		if comparison == 0 {
			return sorted[first].Gas() > sorted[second].Gas()
		}
		return comparison > 0
	})
	for index, transaction := range sorted {
		result.totalGas += transaction.Gas()
		if result.totalGas > maxGasToIndex || index > maxTransactionsToIndex {
			sorted = sorted[:index+1]
			break
		}
	}
	gasCounter := uint64(0)
	percentile := uint64(0)
	for _, transaction := range sorted {
		for percentile < uint64(len(result.percentiles)) && gasCounter >= percentile*maxGasToIndex/uint64(len(result.percentiles)) {
			tip := transaction.EffectiveGasTipValue(minGasPrice)
			if tip.Sign() < 0 {
				result.percentiles[percentile] = minGasPrice
			} else {
				result.percentiles[percentile] = tip.Add(tip, minGasPrice)
			}
			percentile++
		}
		gasCounter += transaction.Gas()
	}
	return result
}

// add pushes a sample into the circular buffer and recombines the samples.
func (stats *circularStats) add(taken sample) {
	stats.mutex.Lock()
	defer stats.mutex.Unlock()
	stats.samples[stats.next] = taken
	stats.next = (stats.next + 1) % len(stats.samples)
	combined := stats.combine()
	stats.combined.Store(&combined)
}

func (stats *circularStats) previous(position int) int {
	if position == 0 {
		return len(stats.samples) - 1
	}
	return position - 1
}

// combine returns, per percentile, the smaller of the average of the samples
// and 5% above the largest of the last four; the caller holds the lock.
func (stats *circularStats) combine() sample {
	average := sample{}
	for percentile := range average.percentiles {
		average.percentiles[percentile] = new(big.Int)
	}
	nonZero := uint64(0)
	for _, kept := range stats.samples {
		if kept.totalGas == 0 {
			continue
		}
		nonZero++
		average.totalGas += kept.totalGas
		for percentile := range kept.percentiles {
			if kept.percentiles[percentile] != nil {
				average.percentiles[percentile].Add(average.percentiles[percentile], kept.percentiles[percentile])
			}
		}
	}
	if nonZero == 0 {
		return average
	}
	average.totalGas /= nonZero
	count := new(big.Int).SetUint64(nonZero)
	for percentile := range average.percentiles {
		average.percentiles[percentile].Div(average.percentiles[percentile], count)
	}

	recent := sample{}
	for percentile := range recent.percentiles {
		recent.percentiles[percentile] = new(big.Int)
	}
	position := stats.next
	for range 4 {
		position = stats.previous(position)
		for percentile, price := range stats.samples[position].percentiles {
			if price != nil && recent.percentiles[percentile].Cmp(price) < 0 {
				recent.percentiles[percentile].Set(price)
			}
		}
	}
	for percentile := range recent.percentiles {
		recent.percentiles[percentile].Mul(recent.percentiles[percentile], big.NewInt(21))
		recent.percentiles[percentile].Div(recent.percentiles[percentile], big.NewInt(20))
	}

	result := sample{totalGas: average.totalGas}
	for _, candidate := range []sample{average, recent} {
		for percentile, price := range candidate.percentiles {
			if result.percentiles[percentile] == nil || result.percentiles[percentile].Cmp(price) > 0 {
				result.percentiles[percentile] = price
			}
		}
	}
	return result
}

// priceForGasAbove is the price a transaction must pay to have no more than
// the given gas of the pool above it.
func (stats *circularStats) priceForGasAbove(gas uint64) *big.Int {
	stats.activated.Store(true)
	combined := stats.combined.Load()
	if combined == nil || combined.totalGas == 0 {
		return new(big.Int)
	}
	last := combined.percentiles[len(combined.percentiles)-1]
	if gas > maxGasToIndex {
		// Beyond the indexed gas, extrapolate linearly.
		price := new(big.Int).Mul(last, new(big.Int).SetUint64(maxGasToIndex))
		return price.Div(price, new(big.Int).SetUint64(gas+1))
	}
	lower := gas * uint64(len(combined.percentiles)) / maxGasToIndex
	if lower >= uint64(len(combined.percentiles))-1 {
		return last
	}
	upper := lower + 1
	lowerGas := lower * maxGasToIndex / uint64(len(combined.percentiles))
	upperGas := upper * maxGasToIndex / uint64(len(combined.percentiles))
	// Interpolate linearly between the two percentiles.
	price := new(big.Int).Mul(combined.percentiles[lower], new(big.Int).SetUint64(upperGas-gas))
	price.Add(price, new(big.Int).Mul(combined.percentiles[upper], new(big.Int).SetUint64(gas-lowerGas)))
	return price.Div(price, new(big.Int).SetUint64(upperGas-lowerGas))
}

// totalGas is the average gas of the head transactions of the pool.
func (stats *circularStats) totalGas() uint64 {
	stats.activated.Store(true)
	combined := stats.combined.Load()
	if combined == nil {
		return 0
	}
	return combined.totalGas
}
