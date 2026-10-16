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
	"fmt"
	"math"
	"math/big"
	"sort"
	"sync/atomic"
	"time"

	"github.com/Fantom-foundation/lachesis-base/utils/piecefunc"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/utils"
)

const (
	// AsDefaultCertainty asks for the certainty the oracle is configured with.
	AsDefaultCertainty = math.MaxUint64
	// DecimalUnit is the fixed point unit of certainties and ratios.
	DecimalUnit = piecefunc.DecimalUnit

	percentilesPerStat = 20
	statUpdatePeriod   = 1 * time.Second
	statsBuffer        = int((15 * time.Second) / statUpdatePeriod)
	maxGasToIndex      = 40000000
	// maxPendingHeads bounds the number of accounts sampled per tick.
	maxPendingHeads = 100_000
	// maxTxsToIndex is the number of transactions that are indexed for the
	// oracle's purposes.
	maxTxsToIndex = 400
)

var (
	decimalUnitBn = big.NewInt(DecimalUnit)
	secondBn      = new(big.Int).SetUint64(uint64(time.Second))
)

// DefaultMaxGasPrice is the recommended default maximum gas price. It returns a
// new value for each call.
func DefaultMaxGasPrice() *big.Int {
	return big.NewInt(10000000 * params.GWei)
}

// Config configures the oracle. Both fields are sanitized to defaults when
// unset or out of range.
type Config struct {
	MaxGasPrice      *big.Int
	DefaultCertainty uint64
}

// Oracle recommends gas prices based on the transactions pending in the pool
// and on the unused gas power of the validators. The inputs are the node's
// OracleInputs and PendingHeads; the oracle never reads the node otherwise.
type Oracle struct {
	node core.NodeInfo
	cfg  Config

	c      circularTxpoolStats
	tCache *lru.Cache[uint64, tipCache]
}

type tipCache struct {
	upd time.Time
	tip *big.Int
}

func sanitize(val, min, max, _default *big.Int, name string) *big.Int {
	if val == nil || (val.Sign() == 0 && _default.Sign() != 0) {
		log.Warn(fmt.Sprintf("Sanitizing invalid parameter %s of gasprice oracle", name), "provided", val, "updated", _default)
		return _default
	}
	if min != nil && val.Cmp(min) < 0 {
		log.Warn(fmt.Sprintf("Sanitizing invalid parameter %s of gasprice oracle", name), "provided", val, "updated", min)
		return min
	}
	if max != nil && val.Cmp(max) > 0 {
		log.Warn(fmt.Sprintf("Sanitizing invalid parameter %s of gasprice oracle", name), "provided", val, "updated", max)
		return max
	}
	return val
}

// New creates an oracle over the node. It samples nothing until Run is called.
func New(cfg Config, node core.NodeInfo) *Oracle {
	cfg.MaxGasPrice = sanitize(cfg.MaxGasPrice, nil, nil, DefaultMaxGasPrice(), "MaxGasPrice")
	cfg.DefaultCertainty = sanitize(new(big.Int).SetUint64(cfg.DefaultCertainty), big.NewInt(0), decimalUnitBn,
		big.NewInt(DecimalUnit/2), "DefaultCertainty").Uint64()
	return &Oracle{node: node, cfg: cfg, tCache: lru.NewCache[uint64, tipCache](100)}
}

// Run samples the pool until the context ends: every second once the oracle is
// in use, every five seconds before. It is meant to run as a dispatcher loop.
func (o *Oracle) Run(ctx context.Context) {
	ticker := time.NewTicker(statUpdatePeriod)
	defer ticker.Stop()
	for i := uint32(0); ; i++ {
		select {
		case <-ticker.C:
			// calculate more frequently after first request
			if atomic.LoadUint32(&o.c.activated) != 0 || i%5 == 0 {
				o.tick()
			}
		case <-ctx.Done():
			return
		}
	}
}

func (o *Oracle) tick() {
	c := &o.c
	// calculate txpool statistic and push into the circular buffer
	c.stats[c.i] = o.calcTxpoolStat()
	c.i = (c.i + 1) % len(c.stats)
	// calculate average of statistics in the circular buffer
	c.avg.Store(c.calcAvg())
}

// SuggestTip returns a tip cap so that a newly created transaction has a very
// high chance to be included in the following blocks.
//
// Note, for legacy transactions and the legacy eth_gasPrice RPC call, it will
// be necessary to add the base fee to the returned number to fall back to the
// legacy behavior.
func (o *Oracle) SuggestTip(certainty uint64) *big.Int {
	if o.node == nil {
		return new(big.Int)
	}
	if certainty == AsDefaultCertainty {
		certainty = o.cfg.DefaultCertainty
	}

	const cacheSlack = DecimalUnit * 0.05
	roundedCertainty := certainty / cacheSlack
	if cache, ok := o.tCache.Get(roundedCertainty); ok {
		if time.Since(cache.upd) < statUpdatePeriod {
			return new(big.Int).Set(cache.tip)
		}
		o.tCache.Remove(roundedCertainty)
	}

	tip := o.suggestTip(certainty)

	o.tCache.Add(roundedCertainty, tipCache{upd: time.Now(), tip: tip})
	return new(big.Int).Set(tip)
}

func (o *Oracle) suggestTip(certainty uint64) *big.Int {
	in := o.node.OracleInputs()
	minPrice := orZero(in.Rules.Economy.MinGasPrice)
	pendingMinPrice := orZero(in.PendingRules.Economy.MinGasPrice)
	adjustedMinGasPrice := utils.BigMax(minPrice, pendingMinPrice)

	reactive := o.reactiveGasPrice(certainty)
	constructive := o.constructiveGasPrice(in, o.c.totalGas(), 0.005*DecimalUnit+certainty/25, adjustedMinGasPrice)

	combined := utils.BigMin(
		utils.BigMax(new(big.Int), reactive, constructive),
		o.cfg.MaxGasPrice,
	)

	tip := new(big.Int).Sub(combined, minPrice)
	if in.MinTip != nil && tip.Cmp(in.MinTip) < 0 {
		return new(big.Int).Set(in.MinTip)
	}
	return tip
}

// orZero is zero for an absent number.
func orZero(v *big.Int) *big.Int {
	if v == nil {
		return new(big.Int)
	}
	return v
}

// -- constructive price: from the gas power the validators have left --

func maxTotalGasPower(in core.OracleInputs) *big.Int {
	rules := in.Rules
	allocBn := new(big.Int).SetUint64(rules.Economy.LongGasPower.AllocPerSec)
	periodBn := new(big.Int).SetUint64(uint64(rules.Economy.LongGasPower.MaxAllocPeriod))
	maxTotalGasPowerBn := new(big.Int).Mul(allocBn, periodBn)
	maxTotalGasPowerBn.Div(maxTotalGasPowerBn, secondBn)
	return maxTotalGasPowerBn
}

func (o *Oracle) constructiveGasPrice(in core.OracleInputs, gasOffsetAbs uint64, gasOffsetRatio uint64, adjustedMinPrice *big.Int) *big.Int {
	max := maxTotalGasPower(in)

	current64 := in.TotalGasPowerLeft
	if current64 > gasOffsetAbs {
		current64 -= gasOffsetAbs
	} else {
		current64 = 0
	}
	current := new(big.Int).SetUint64(current64)

	freeRatioBn := current.Mul(current, decimalUnitBn)
	if max.Sign() > 0 {
		freeRatioBn.Div(freeRatioBn, max)
	} else {
		freeRatioBn.SetUint64(0)
	}
	freeRatio := freeRatioBn.Uint64()
	if freeRatio > gasOffsetRatio {
		freeRatio -= gasOffsetRatio
	} else {
		freeRatio = 0
	}
	if freeRatio > DecimalUnit {
		freeRatio = DecimalUnit
	}
	return constructiveGasPriceOf(freeRatio, adjustedMinPrice)
}

var freeRatioToConstructiveGasRatio = piecefunc.NewFunc([]piecefunc.Dot{
	{X: 0, Y: 25 * DecimalUnit},
	{X: 0.3 * DecimalUnit, Y: 9 * DecimalUnit},
	{X: 0.5 * DecimalUnit, Y: 3.75 * DecimalUnit},
	{X: 0.8 * DecimalUnit, Y: 1.5 * DecimalUnit},
	{X: 0.95 * DecimalUnit, Y: 1.05 * DecimalUnit},
	{X: DecimalUnit, Y: DecimalUnit},
})

func constructiveGasPriceOf(freeRatio uint64, adjustedMinPrice *big.Int) *big.Int {
	multiplier := new(big.Int).SetUint64(freeRatioToConstructiveGasRatio(freeRatio))

	// gas price = multiplier * adjustedMinPrice
	price := multiplier.Mul(multiplier, adjustedMinPrice)
	return price.Div(price, decimalUnitBn)
}

// -- reactive price: from the pending transactions --

type txpoolStat struct {
	totalGas    uint64
	percentiles [percentilesPerStat]*big.Int
}

type circularTxpoolStats struct {
	stats     [statsBuffer]txpoolStat
	i         int
	activated uint32
	avg       atomic.Value
}

var certaintyToGasAbove = piecefunc.NewFunc([]piecefunc.Dot{
	{X: 0, Y: 50000000},
	{X: 0.2 * DecimalUnit, Y: 20000000},
	{X: 0.5 * DecimalUnit, Y: 8000000},
	{X: DecimalUnit, Y: 0},
})

func (o *Oracle) reactiveGasPrice(certainty uint64) *big.Int {
	gasAbove := certaintyToGasAbove(certainty)
	return o.c.getGasPriceForGasAbove(gasAbove)
}

func (c *circularTxpoolStats) dec(v int) int {
	if v == 0 {
		return len(c.stats) - 1
	}
	return v - 1
}

// calcAvg calculates the average of the statistics in the circular buffer.
func (c *circularTxpoolStats) calcAvg() txpoolStat {
	avg := txpoolStat{}
	for p := range avg.percentiles {
		avg.percentiles[p] = new(big.Int)
	}
	nonZero := uint64(0)
	for _, s := range c.stats {
		if s.totalGas == 0 {
			continue
		}
		nonZero++
		avg.totalGas += s.totalGas
		for p := range s.percentiles {
			if s.percentiles[p] == nil {
				continue
			}
			avg.percentiles[p].Add(avg.percentiles[p], s.percentiles[p])
		}
	}
	if nonZero == 0 {
		return avg
	}
	avg.totalGas /= nonZero
	nonZeroBn := new(big.Int).SetUint64(nonZero)
	for p := range avg.percentiles {
		avg.percentiles[p].Div(avg.percentiles[p], nonZeroBn)
	}

	// take maximum from previous 4 stats plus 5%
	rec := txpoolStat{}
	for p := range rec.percentiles {
		rec.percentiles[p] = new(big.Int)
	}
	recI1 := c.dec(c.i)
	recI2 := c.dec(recI1)
	recI3 := c.dec(recI2)
	recI4 := c.dec(recI3)
	for _, s := range []txpoolStat{c.stats[recI1], c.stats[recI2], c.stats[recI3], c.stats[recI4]} {
		for p := range s.percentiles {
			if s.percentiles[p] == nil {
				continue
			}
			if rec.percentiles[p].Cmp(s.percentiles[p]) < 0 {
				rec.percentiles[p].Set(s.percentiles[p])
			}
		}
	}
	// increase by 5%
	for p := range rec.percentiles {
		rec.percentiles[p].Mul(rec.percentiles[p], big.NewInt(21))
		rec.percentiles[p].Div(rec.percentiles[p], big.NewInt(20))
	}

	// return minimum from max(recent two stats * 1.05) and avg stats
	res := txpoolStat{}
	res.totalGas = avg.totalGas
	for _, s := range []txpoolStat{avg, rec} {
		for p := range s.percentiles {
			if res.percentiles[p] == nil || res.percentiles[p].Cmp(s.percentiles[p]) > 0 {
				res.percentiles[p] = s.percentiles[p]
			}
		}
	}
	return res
}

func (c *circularTxpoolStats) getGasPriceForGasAbove(gas uint64) *big.Int {
	atomic.StoreUint32(&c.activated, 1)
	avgC := c.avg.Load()
	if avgC == nil {
		return new(big.Int)
	}
	avg := avgC.(txpoolStat)
	if avg.totalGas == 0 {
		return new(big.Int)
	}
	if gas > maxGasToIndex {
		// extrapolate linearly
		v := new(big.Int).Mul(avg.percentiles[len(avg.percentiles)-1], new(big.Int).SetUint64(maxGasToIndex))
		v.Div(v, new(big.Int).SetUint64(gas+1))
		return v
	}
	p0 := gas * uint64(len(avg.percentiles)) / maxGasToIndex
	if p0 >= uint64(len(avg.percentiles))-1 {
		return avg.percentiles[len(avg.percentiles)-1]
	}
	// interpolate linearly
	p1 := p0 + 1
	x := gas
	x0, x1 := p0*maxGasToIndex/uint64(len(avg.percentiles)), p1*maxGasToIndex/uint64(len(avg.percentiles))
	y0, y1 := avg.percentiles[p0], avg.percentiles[p1]
	return div64I(addBigI(mul64N(y0, x1-x), mul64N(y1, x-x0)), x1-x0)
}

func mul64N(a *big.Int, b uint64) *big.Int { return new(big.Int).Mul(a, new(big.Int).SetUint64(b)) }

func div64I(a *big.Int, b uint64) *big.Int { return a.Div(a, new(big.Int).SetUint64(b)) }

func addBigI(a, b *big.Int) *big.Int { return a.Add(a, b) }

func (c *circularTxpoolStats) totalGas() uint64 {
	atomic.StoreUint32(&c.activated, 1)
	avgC := c.avg.Load()
	if avgC == nil {
		return 0
	}
	return avgC.(txpoolStat).totalGas
}

// calcTxpoolStat retrieves the pending transactions and calculates statistics.
func (o *Oracle) calcTxpoolStat() txpoolStat {
	heads := o.node.PendingHeads(maxPendingHeads)
	s := txpoolStat{}
	if len(heads) == 0 {
		// short circuit if empty txpool
		return s
	}
	// take only one tx from each account
	txs := make(types.Transactions, 0, len(heads))
	for _, h := range heads {
		if h.Tx != nil {
			txs = append(txs, h.Tx)
		}
	}
	if len(txs) == 0 {
		return s
	}

	minGasPrice := orZero(o.node.OracleInputs().Rules.Economy.MinGasPrice)
	minGasPriceU256 := utils.BigIntToUint256Clamped(minGasPrice)
	// txs are sorted from large price to small
	sorted := txs
	sort.Slice(sorted, func(i, j int) bool {
		a, b := sorted[i], sorted[j]
		cmp := a.EffectiveGasTipCmp(b, minGasPriceU256)
		if cmp == 0 {
			return a.Gas() > b.Gas()
		}
		return cmp > 0
	})

	// don't index more transactions than needed for the oracle's purposes
	for i, tx := range sorted {
		s.totalGas += tx.Gas()
		if s.totalGas > maxGasToIndex || i > maxTxsToIndex {
			sorted = sorted[:i+1]
			break
		}
	}

	gasCounter := uint64(0)
	p := uint64(0)
	for _, tx := range sorted {
		for p < uint64(len(s.percentiles)) && gasCounter >= p*maxGasToIndex/uint64(len(s.percentiles)) {
			gasTip := tx.EffectiveGasTipValue(minGasPrice)
			s.percentiles[p] = gasTip
			if s.percentiles[p].Sign() < 0 {
				s.percentiles[p] = minGasPrice
			} else {
				s.percentiles[p].Add(s.percentiles[p], minGasPrice)
			}
			p++
		}
		gasCounter += tx.Gas()
	}
	return s
}
