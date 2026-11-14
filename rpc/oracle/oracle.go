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

// Package oracle suggests gas prices for the RPC module, as the legacy oracle
// of the node does: from a sample of the pool, taken in a loop, and from the
// gas power the validators have left, read per suggestion. It knows the
// served head only as a function, and its loop is started by the module.
package oracle

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/utils"
	"github.com/Fantom-foundation/lachesis-base/utils/piecefunc"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	lru "github.com/hashicorp/golang-lru/v2"
)

const (
	// DecimalUnit is the certainty of a sure suggestion; certainties are
	// fractions of it.
	DecimalUnit = piecefunc.DecimalUnit
	// DefaultCertainty asks for the certainty of the configuration.
	DefaultCertainty = math.MaxUint64
)

var decimalUnit = big.NewInt(DecimalUnit)

// Config is the configuration of the oracle, the GPO section of the node's.
type Config struct {
	// MaxGasPrice is the ceiling of a suggested price.
	MaxGasPrice *big.Int
	// MinGasPrice is the floor of a suggested price.
	MinGasPrice *big.Int
	// DefaultCertainty is the certainty of a suggestion that names none.
	DefaultCertainty uint64
}

// DefaultConfig is the configuration of the oracle unless the node sets one.
func DefaultConfig() Config {
	return Config{MaxGasPrice: defaultMaxGasPrice(), MinGasPrice: new(big.Int), DefaultCertainty: DecimalUnit / 2}
}

func defaultMaxGasPrice() *big.Int {
	return big.NewInt(10000000 * params.GWei)
}

// Source is what the oracle reads from the node.
type Source interface {
	OracleInputs() core.OracleInputs
	PoolNextTransactions() types.Transactions
	Rules(number uint64) (*opera.Rules, error)
}

// cachedTip is a suggestion and when it was made.
type cachedTip struct {
	made time.Time
	tip  *big.Int
}

// Oracle suggests gas tips.
type Oracle struct {
	config Config
	source Source
	// head is the served head.
	head  func() uint64
	stats sampleStats
	tips  *lru.Cache[uint64, cachedTip]
}

// New creates an oracle; Sample must run for it to see the pool. A config
// value out of its range is replaced by its default, with a warning.
func New(config Config, source Source, head func() uint64) *Oracle {
	defaults := DefaultConfig()
	config.MaxGasPrice = sanitize(config.MaxGasPrice, nil, nil, defaults.MaxGasPrice, "MaxGasPrice")
	config.MinGasPrice = sanitize(config.MinGasPrice, nil, nil, defaults.MinGasPrice, "MinGasPrice")
	config.DefaultCertainty = sanitize(new(big.Int).SetUint64(config.DefaultCertainty), new(big.Int), decimalUnit,
		new(big.Int).SetUint64(defaults.DefaultCertainty), "DefaultCertainty").Uint64()
	tips, _ := lru.New[uint64, cachedTip](100)
	return &Oracle{config: config, source: source, head: head, tips: tips}
}

func sanitize(value, minimum, maximum, fallback *big.Int, name string) *big.Int {
	switch {
	case value == nil || value.Sign() == 0 && fallback.Sign() != 0:
		log.Warn(fmt.Sprintf("Sanitizing invalid parameter %s of gasprice oracle", name), "provided", value, "updated", fallback)
		return fallback
	case minimum != nil && value.Cmp(minimum) < 0:
		log.Warn(fmt.Sprintf("Sanitizing invalid parameter %s of gasprice oracle", name), "provided", value, "updated", minimum)
		return minimum
	case maximum != nil && value.Cmp(maximum) > 0:
		log.Warn(fmt.Sprintf("Sanitizing invalid parameter %s of gasprice oracle", name), "provided", value, "updated", maximum)
		return maximum
	default:
		return value
	}
}

// SuggestTip suggests a tip that gets a new transaction into the next blocks
// with the certainty, a fraction of DecimalUnit. A suggestion is kept for a
// second per twentieth of certainty.
func (o *Oracle) SuggestTip(certainty uint64) (*big.Int, error) {
	if certainty == DefaultCertainty {
		certainty = o.config.DefaultCertainty
	}
	const cacheSlack = DecimalUnit * 0.05
	rounded := certainty / cacheSlack
	if cached, found := o.tips.Get(rounded); found && time.Since(cached.made) < samplePeriod {
		return new(big.Int).Set(cached.tip), nil
	}
	tip, err := o.suggestTip(certainty)
	if err != nil {
		return nil, err
	}
	o.tips.Add(rounded, cachedTip{made: time.Now(), tip: tip})
	return new(big.Int).Set(tip), nil
}

// suggestTip combines the price the pool sample asks for at the certainty
// with the price the gas power left asks for, within the configured bounds,
// and answers its excess over the minimum gas price, at least the pool's
// smallest tip.
func (o *Oracle) suggestTip(certainty uint64) (*big.Int, error) {
	rules, err := o.currentRules()
	if err != nil {
		return nil, err
	}
	inputs := o.source.OracleInputs()
	minPrice := rules.Economy.MinGasPrice
	adjustedMinPrice := utils.BigMax(minPrice, inputs.NextEpochRules.Economy.MinGasPrice)

	reactive := o.stats.priceForGasAbove(certaintyToGasAbove(certainty))
	gasOffset := o.stats.totalGas()
	constructive := constructivePrice(rules, inputs.TotalGasPowerLeft, gasOffset, 0.005*DecimalUnit+certainty/25, adjustedMinPrice)

	combined := utils.BigMin(utils.BigMax(o.config.MinGasPrice, reactive, constructive), o.config.MaxGasPrice)
	tip := new(big.Int).Sub(combined, minPrice)
	if tip.Cmp(inputs.MinTip) < 0 {
		return inputs.MinTip, nil
	}
	return tip, nil
}

// currentRules are the rules of the block after the served head.
func (o *Oracle) currentRules() (*opera.Rules, error) {
	rules, err := o.source.Rules(o.head() + 1)
	if err != nil {
		return nil, fmt.Errorf("failed to read the current rules: %w", err)
	}
	return rules, nil
}

// samplePeriod is how often the pool is sampled once suggestions are asked
// for, and how long a suggestion is kept.
const samplePeriod = 1 * time.Second

// idleSamples is how many sample periods pass between two samples before the
// first suggestion is asked for.
const idleSamples = 5

// Sample samples the pool until the context ends: every second once
// suggestions are asked for, every five seconds before.
func (o *Oracle) Sample(ctx context.Context) {
	ticker := time.NewTicker(samplePeriod)
	defer ticker.Stop()
	for i := 0; ; i++ {
		select {
		case <-ticker.C:
			if o.stats.isActive() || i%idleSamples == 0 {
				o.sample()
			}
		case <-ctx.Done():
			return
		}
	}
}

func (o *Oracle) sample() {
	rules, err := o.currentRules()
	if err != nil {
		log.Warn("Gas price oracle skips a sample", "err", err)
		return
	}
	o.stats.add(newPoolStat(o.source.PoolNextTransactions(), rules.Economy.MinGasPrice))
}
