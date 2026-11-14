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
	"math/big"
	"sort"
	"sync/atomic"
	"time"

	"github.com/0xsoniclabs/sonic/utils"
	"github.com/Fantom-foundation/lachesis-base/utils/piecefunc"
	"github.com/ethereum/go-ethereum/core/types"
)

const (
	percentilesPerStat = 20
	// statsKept is how many samples the average is taken over.
	statsKept = int((15 * time.Second) / samplePeriod)
	// maxGasToIndex is the gas of the pool's sample that is indexed.
	maxGasToIndex = 40000000
	// maxTransactionsToIndex bounds the transactions a sample indexes.
	maxTransactionsToIndex = 400
)

// poolStat is a sample of the pool: the gas it holds, and the prices at
// which each twentieth of that gas is bid, from the highest bid.
type poolStat struct {
	totalGas    uint64
	percentiles [percentilesPerStat]*big.Int
}

// newPoolStat samples the next transaction of every sender, by tip from the
// highest, and by gas among equal tips.
func newPoolStat(transactions types.Transactions, minGasPrice *big.Int) poolStat {
	stat := poolStat{}
	if len(transactions) == 0 {
		return stat
	}
	minGasPriceU256 := utils.BigIntToUint256Clamped(minGasPrice)
	sorted := transactions
	sort.Slice(sorted, func(i, j int) bool {
		a, b := sorted[i], sorted[j]
		cmp := a.EffectiveGasTipCmp(b, minGasPriceU256)
		if cmp == 0 {
			return a.Gas() > b.Gas()
		}
		return cmp > 0
	})
	for i, transaction := range sorted {
		stat.totalGas += transaction.Gas()
		if stat.totalGas > maxGasToIndex || i > maxTransactionsToIndex {
			sorted = sorted[:i+1]
			break
		}
	}
	gasCounter := uint64(0)
	p := uint64(0)
	for _, transaction := range sorted {
		for p < uint64(len(stat.percentiles)) && gasCounter >= p*maxGasToIndex/uint64(len(stat.percentiles)) {
			price := transaction.EffectiveGasTipValue(minGasPrice)
			if price.Sign() < 0 {
				price = minGasPrice
			} else {
				price.Add(price, minGasPrice)
			}
			stat.percentiles[p] = price
			p++
		}
		gasCounter += transaction.Gas()
	}
	return stat
}

// sampleStats keeps the last samples and their average. The sampling loop
// adds samples; suggestions read the average.
type sampleStats struct {
	stats  [statsKept]poolStat
	next   int
	active atomic.Bool
	// average is the poolStat suggestions read.
	average atomic.Value
}

// isActive tells whether a suggestion has been asked for.
func (s *sampleStats) isActive() bool {
	return s.active.Load()
}

func (s *sampleStats) add(stat poolStat) {
	s.stats[s.next] = stat
	s.next = (s.next + 1) % len(s.stats)
	s.average.Store(s.calcAverage())
}

func (s *sampleStats) previous(i int) int {
	if i == 0 {
		return len(s.stats) - 1
	}
	return i - 1
}

// calcAverage takes per percentile the smaller of the average of the
// non-empty samples and the largest of the last four samples plus 5%.
func (s *sampleStats) calcAverage() poolStat {
	average := poolStat{}
	for p := range average.percentiles {
		average.percentiles[p] = new(big.Int)
	}
	nonZero := uint64(0)
	for _, stat := range s.stats {
		if stat.totalGas == 0 {
			continue
		}
		nonZero++
		average.totalGas += stat.totalGas
		for p := range stat.percentiles {
			if stat.percentiles[p] != nil {
				average.percentiles[p].Add(average.percentiles[p], stat.percentiles[p])
			}
		}
	}
	if nonZero == 0 {
		return average
	}
	average.totalGas /= nonZero
	count := new(big.Int).SetUint64(nonZero)
	for p := range average.percentiles {
		average.percentiles[p].Div(average.percentiles[p], count)
	}

	recent := poolStat{}
	for p := range recent.percentiles {
		recent.percentiles[p] = new(big.Int)
	}
	last1 := s.previous(s.next)
	last2 := s.previous(last1)
	last3 := s.previous(last2)
	last4 := s.previous(last3)
	for _, stat := range []poolStat{s.stats[last1], s.stats[last2], s.stats[last3], s.stats[last4]} {
		for p := range stat.percentiles {
			if stat.percentiles[p] != nil && recent.percentiles[p].Cmp(stat.percentiles[p]) < 0 {
				recent.percentiles[p].Set(stat.percentiles[p])
			}
		}
	}
	for p := range recent.percentiles {
		recent.percentiles[p].Mul(recent.percentiles[p], big.NewInt(21))
		recent.percentiles[p].Div(recent.percentiles[p], big.NewInt(20))
	}

	result := poolStat{totalGas: average.totalGas}
	for _, stat := range []poolStat{average, recent} {
		for p := range stat.percentiles {
			if result.percentiles[p] == nil || result.percentiles[p].Cmp(stat.percentiles[p]) > 0 {
				result.percentiles[p] = stat.percentiles[p]
			}
		}
	}
	return result
}

// loadAverage reads the average and marks the stats active.
func (s *sampleStats) loadAverage() (poolStat, bool) {
	s.active.Store(true)
	average, found := s.average.Load().(poolStat)
	return average, found
}

// priceForGasAbove is the price at which the gas of the sample above the
// given amount is bid, interpolated between percentiles.
func (s *sampleStats) priceForGasAbove(gas uint64) *big.Int {
	average, found := s.loadAverage()
	if !found || average.totalGas == 0 {
		return new(big.Int)
	}
	last := average.percentiles[len(average.percentiles)-1]
	if gas > maxGasToIndex {
		price := new(big.Int).Mul(last, new(big.Int).SetUint64(maxGasToIndex))
		return price.Div(price, new(big.Int).SetUint64(gas+1))
	}
	p0 := gas * uint64(len(average.percentiles)) / maxGasToIndex
	if p0 >= uint64(len(average.percentiles))-1 {
		return last
	}
	p1 := p0 + 1
	x0, x1 := p0*maxGasToIndex/uint64(len(average.percentiles)), p1*maxGasToIndex/uint64(len(average.percentiles))
	y0, y1 := average.percentiles[p0], average.percentiles[p1]
	price := new(big.Int).Add(times(y0, x1-gas), times(y1, gas-x0))
	return price.Div(price, new(big.Int).SetUint64(x1-x0))
}

func times(value *big.Int, factor uint64) *big.Int {
	return new(big.Int).Mul(value, new(big.Int).SetUint64(factor))
}

// totalGas is the average gas of the samples.
func (s *sampleStats) totalGas() uint64 {
	average, _ := s.loadAverage()
	return average.totalGas
}

// certaintyToGasAbove is the gas of the sample a suggestion of a certainty
// outbids.
var certaintyToGasAbove = piecefunc.NewFunc([]piecefunc.Dot{
	{X: 0, Y: 50000000},
	{X: 0.2 * DecimalUnit, Y: 20000000},
	{X: 0.5 * DecimalUnit, Y: 8000000},
	{X: DecimalUnit, Y: 0},
})
