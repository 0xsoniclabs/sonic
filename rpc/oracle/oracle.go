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

// Package oracle suggests the tip of a new transaction from the pool and the
// gas power left to the validators, as the node's oracle did.
package oracle

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Fantom-foundation/lachesis-base/utils/piecefunc"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/utils"
)

const (
	// AsDefaultCertainty asks for the certainty of the configuration.
	AsDefaultCertainty = math.MaxUint64
	// DecimalUnit is the unit of a certainty: DecimalUnit is certain.
	DecimalUnit = piecefunc.DecimalUnit
)

const (
	percentilesPerStat = 20
	statUpdatePeriod   = 1 * time.Second
	statsBuffer        = int((15 * time.Second) / statUpdatePeriod)
	maxGasToIndex      = 40000000
	// maxTxsToIndex bounds the pool transactions one sample orders.
	maxTxsToIndex = 400
	// maxPendingHeads bounds the accounts one sample reads; it is above the
	// number of accounts the pool can hold.
	maxPendingHeads = 1 << 20
	// tipCacheSlack rounds certainties into the buckets of the tip cache.
	tipCacheSlack = DecimalUnit * 0.05
)

var (
	decimalUnitBn = big.NewInt(DecimalUnit)
	secondBn      = new(big.Int).SetUint64(uint64(time.Second))
)

// Config holds the GPO keys of the node configuration.
type Config struct {
	// MaxGasPrice is the ceiling of a suggestion.
	MaxGasPrice *big.Int
	// MinGasPrice is the floor of a suggestion.
	MinGasPrice *big.Int
	// DefaultCertainty is the certainty of a request that names none.
	DefaultCertainty uint64
}

// DefaultConfig is the one definition of the GPO defaults.
func DefaultConfig() Config {
	return Config{
		MaxGasPrice:      big.NewInt(10000000 * params.GWei),
		MinGasPrice:      new(big.Int),
		DefaultCertainty: 0.5 * DecimalUnit,
	}
}

// Source is what the oracle reads of the node: the pool heads for a sample,
// and the gas power left and the pending rules for a suggestion.
type Source interface {
	OracleInputs() core.OracleInputs
	PendingHeads(max int) []core.PendingHead
}

// OpenRules returns the rules of the open epoch, the ones the next block is
// processed with.
type OpenRules func() (*opera.Rules, error)

// Oracle suggests tips.
type Oracle struct {
	src   Source
	rules OpenRules
	cfg   Config

	stats circularStats

	cacheMu sync.Mutex
	cache   map[uint64]cachedTip
}

type cachedTip struct {
	updated time.Time
	tip     *big.Int
}

// New creates an oracle. Run samples the pool; without it the reactive part
// of a suggestion stays zero.
func New(cfg Config, src Source, rules OpenRules) *Oracle {
	cfg.MaxGasPrice = sanitizeBigInt(cfg.MaxGasPrice, nil, nil, DefaultConfig().MaxGasPrice, "MaxGasPrice")
	cfg.MinGasPrice = sanitizeBigInt(cfg.MinGasPrice, nil, nil, new(big.Int), "MinGasPrice")
	cfg.DefaultCertainty = sanitizeBigInt(new(big.Int).SetUint64(cfg.DefaultCertainty), big.NewInt(0), decimalUnitBn, big.NewInt(DecimalUnit/2), "DefaultCertainty").Uint64()
	return &Oracle{
		src:   src,
		rules: rules,
		cfg:   cfg,
		cache: make(map[uint64]cachedTip),
	}
}

func sanitizeBigInt(val, min, max, def *big.Int, name string) *big.Int {
	if val == nil || (val.Sign() == 0 && def.Sign() != 0) {
		log.Warn(fmt.Sprintf("Sanitizing invalid parameter %s of gasprice oracle", name), "provided", val, "updated", def)
		return def
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

// Run samples the pool once a second after the first suggestion, and every
// five seconds before it, until ctx ends.
func (o *Oracle) Run(ctx context.Context) {
	ticker := time.NewTicker(statUpdatePeriod)
	defer ticker.Stop()
	for i := uint32(0); ; i++ {
		select {
		case <-ticker.C:
			if o.stats.activated.Load() || i%5 == 0 {
				o.sample(ctx)
			}
		case <-ctx.Done():
			return
		}
	}
}

// errNoRules is a rules lookup that answered nothing.
var errNoRules = errors.New("no rules of the open epoch")

// openRules reads the rules of the open epoch, unless ctx ended.
func (o *Oracle) openRules(ctx context.Context) (*opera.Rules, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	rules, err := o.rules()
	if err == nil && rules == nil {
		err = errNoRules
	}
	if err != nil {
		return nil, err
	}
	return rules, nil
}

// SuggestTip suggests the tip of a new transaction for a certainty between 0
// and DecimalUnit, or AsDefaultCertainty.
func (o *Oracle) SuggestTip(ctx context.Context, certainty uint64) (*big.Int, error) {
	if certainty == AsDefaultCertainty {
		certainty = o.cfg.DefaultCertainty
	}
	bucket := certainty / tipCacheSlack
	o.cacheMu.Lock()
	cached, ok := o.cache[bucket]
	o.cacheMu.Unlock()
	if ok && time.Since(cached.updated) < statUpdatePeriod {
		return new(big.Int).Set(cached.tip), nil
	}
	tip, err := o.suggestTip(ctx, certainty)
	if err != nil {
		return nil, err
	}
	o.cacheMu.Lock()
	o.cache[bucket] = cachedTip{updated: time.Now(), tip: tip}
	o.cacheMu.Unlock()
	return new(big.Int).Set(tip), nil
}

func (o *Oracle) suggestTip(ctx context.Context, certainty uint64) (*big.Int, error) {
	rules, err := o.openRules(ctx)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	inputs := o.src.OracleInputs()
	minPrice := rules.Economy.MinGasPrice
	adjustedMinPrice := minPrice
	if pending := inputs.PendingRules.Economy.MinGasPrice; pending != nil {
		adjustedMinPrice = utils.BigMax(minPrice, pending)
	}

	reactive := o.reactiveGasPrice(certainty)
	constructive := constructiveGasPrice(rules, inputs.TotalGasPowerLeft, o.stats.totalGas(), 0.005*DecimalUnit+certainty/25, adjustedMinPrice)

	combined := utils.BigMin(
		utils.BigMax(o.cfg.MinGasPrice, reactive, constructive),
		o.cfg.MaxGasPrice,
	)
	tip := new(big.Int).Sub(combined, minPrice)
	if inputs.MinTip != nil && tip.Cmp(inputs.MinTip) < 0 {
		return new(big.Int).Set(inputs.MinTip), nil
	}
	return tip, nil
}

// constructive

func maxTotalGasPower(rules *opera.Rules) *big.Int {
	alloc := new(big.Int).SetUint64(rules.Economy.LongGasPower.AllocPerSec)
	period := new(big.Int).SetUint64(uint64(rules.Economy.LongGasPower.MaxAllocPeriod))
	total := new(big.Int).Mul(alloc, period)
	return total.Div(total, secondBn)
}

func constructiveGasPrice(rules *opera.Rules, gasPowerLeft, gasOffsetAbs, gasOffsetRatio uint64, adjustedMinPrice *big.Int) *big.Int {
	maxPower := maxTotalGasPower(rules)
	current64 := gasPowerLeft
	if current64 > gasOffsetAbs {
		current64 -= gasOffsetAbs
	} else {
		current64 = 0
	}
	current := new(big.Int).SetUint64(current64)

	freeRatioBn := current.Mul(current, decimalUnitBn)
	if maxPower.Sign() != 0 {
		freeRatioBn.Div(freeRatioBn, maxPower)
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
	multiplier := new(big.Int).SetUint64(freeRatioToConstructiveGasRatio(freeRatio))
	price := multiplier.Mul(multiplier, adjustedMinPrice)
	return price.Div(price, decimalUnitBn)
}

var freeRatioToConstructiveGasRatio = piecefunc.NewFunc([]piecefunc.Dot{
	{X: 0, Y: 25 * DecimalUnit},
	{X: 0.3 * DecimalUnit, Y: 9 * DecimalUnit},
	{X: 0.5 * DecimalUnit, Y: 3.75 * DecimalUnit},
	{X: 0.8 * DecimalUnit, Y: 1.5 * DecimalUnit},
	{X: 0.95 * DecimalUnit, Y: 1.05 * DecimalUnit},
	{X: DecimalUnit, Y: DecimalUnit},
})

// reactive

var certaintyToGasAbove = piecefunc.NewFunc([]piecefunc.Dot{
	{X: 0, Y: 50000000},
	{X: 0.2 * DecimalUnit, Y: 20000000},
	{X: 0.5 * DecimalUnit, Y: 8000000},
	{X: DecimalUnit, Y: 0},
})

func (o *Oracle) reactiveGasPrice(certainty uint64) *big.Int {
	return o.stats.priceForGasAbove(certaintyToGasAbove(certainty))
}

type poolStat struct {
	totalGas    uint64
	percentiles [percentilesPerStat]*big.Int
}

// circularStats keeps the samples of the last statsBuffer seconds. Only Run
// writes stats and i; readers load the average.
type circularStats struct {
	stats     [statsBuffer]poolStat
	i         int
	activated atomic.Bool
	avg       atomic.Pointer[poolStat]
}

// sample reads the heads of the pool and the minimum gas price of the open
// epoch, and nothing else: OracleInputs walks every validator's last event.
func (o *Oracle) sample(ctx context.Context) {
	rules, err := o.openRules(ctx)
	if err != nil {
		log.Warn("Gas price oracle skips a sample", "err", err)
		return
	}
	c := &o.stats
	c.stats[c.i] = calcPoolStat(o.src.PendingHeads(maxPendingHeads), rules.Economy.MinGasPrice)
	c.i = (c.i + 1) % len(c.stats)
	avg := c.calcAvg()
	c.avg.Store(&avg)
}

func (c *circularStats) dec(v int) int {
	if v == 0 {
		return len(c.stats) - 1
	}
	return v - 1
}

func (c *circularStats) calcAvg() poolStat {
	avg := poolStat{}
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

	// the maximum of the 4 most recent samples, plus 5%
	rec := poolStat{}
	for p := range rec.percentiles {
		rec.percentiles[p] = new(big.Int)
	}
	recI1 := c.dec(c.i)
	recI2 := c.dec(recI1)
	recI3 := c.dec(recI2)
	recI4 := c.dec(recI3)
	for _, s := range []poolStat{c.stats[recI1], c.stats[recI2], c.stats[recI3], c.stats[recI4]} {
		for p := range s.percentiles {
			if s.percentiles[p] == nil {
				continue
			}
			if rec.percentiles[p].Cmp(s.percentiles[p]) < 0 {
				rec.percentiles[p].Set(s.percentiles[p])
			}
		}
	}
	for p := range rec.percentiles {
		rec.percentiles[p].Mul(rec.percentiles[p], big.NewInt(21))
		rec.percentiles[p].Div(rec.percentiles[p], big.NewInt(20))
	}

	// the minimum of the recent maximum and the average
	res := poolStat{totalGas: avg.totalGas}
	for _, s := range []poolStat{avg, rec} {
		for p := range s.percentiles {
			if res.percentiles[p] == nil || res.percentiles[p].Cmp(s.percentiles[p]) > 0 {
				res.percentiles[p] = s.percentiles[p]
			}
		}
	}
	return res
}

func (c *circularStats) priceForGasAbove(gas uint64) *big.Int {
	c.activated.Store(true)
	avg := c.avg.Load()
	if avg == nil || avg.totalGas == 0 {
		return new(big.Int)
	}
	if gas > maxGasToIndex {
		// extrapolate linearly
		v := new(big.Int).Mul(avg.percentiles[len(avg.percentiles)-1], new(big.Int).SetUint64(maxGasToIndex))
		return v.Div(v, new(big.Int).SetUint64(gas+1))
	}
	p0 := gas * uint64(len(avg.percentiles)) / maxGasToIndex
	if p0 >= uint64(len(avg.percentiles))-1 {
		return avg.percentiles[len(avg.percentiles)-1]
	}
	// interpolate linearly
	p1 := p0 + 1
	x0, x1 := p0*maxGasToIndex/uint64(len(avg.percentiles)), p1*maxGasToIndex/uint64(len(avg.percentiles))
	y0, y1 := avg.percentiles[p0], avg.percentiles[p1]
	sum := new(big.Int).Add(new(big.Int).Mul(y0, new(big.Int).SetUint64(x1-gas)), new(big.Int).Mul(y1, new(big.Int).SetUint64(gas-x0)))
	return sum.Div(sum, new(big.Int).SetUint64(x1-x0))
}

func (c *circularStats) totalGas() uint64 {
	c.activated.Store(true)
	avg := c.avg.Load()
	if avg == nil {
		return 0
	}
	return avg.totalGas
}

// effectiveTip is min(tipCap, feeCap - baseFee), negative when the fee cap is
// below the base fee, as go-ethereum's EffectiveGasTipValue.
func effectiveTip(h core.PendingHead, baseFee *big.Int) *big.Int {
	tip := new(big.Int).Sub(h.GasFeeCap, baseFee)
	if h.GasTipCap.Cmp(tip) < 0 {
		tip.Set(h.GasTipCap)
	}
	return tip
}

func calcPoolStat(heads []core.PendingHead, minGasPrice *big.Int) poolStat {
	s := poolStat{}
	if len(heads) == 0 {
		return s
	}
	// the highest effective tip first, then the most gas; a head without its
	// prices is no sample
	sorted := make([]core.PendingHead, 0, len(heads))
	for _, h := range heads {
		if h.GasFeeCap != nil && h.GasTipCap != nil {
			sorted = append(sorted, h)
		}
	}
	sort.Slice(sorted, func(i, j int) bool {
		a, b := sorted[i], sorted[j]
		cmp := effectiveTip(a, minGasPrice).Cmp(effectiveTip(b, minGasPrice))
		if cmp == 0 {
			return a.Gas > b.Gas
		}
		return cmp > 0
	})
	for i, h := range sorted {
		s.totalGas += h.Gas
		if s.totalGas > maxGasToIndex || i > maxTxsToIndex {
			sorted = sorted[:i+1]
			break
		}
	}

	gasCounter := uint64(0)
	p := uint64(0)
	for _, h := range sorted {
		for p < uint64(len(s.percentiles)) && gasCounter >= p*maxGasToIndex/uint64(len(s.percentiles)) {
			tip := effectiveTip(h, minGasPrice)
			if tip.Sign() < 0 {
				s.percentiles[p] = minGasPrice
			} else {
				s.percentiles[p] = tip.Add(tip, minGasPrice)
			}
			p++
		}
		gasCounter += h.Gas
	}
	return s
}
