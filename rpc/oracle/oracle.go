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

// Package oracle suggests the tip of a new transaction from the content of
// the pool and the gas power the validators have left, as the node's gas
// price oracle did.
package oracle

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"sync"
	"time"

	"github.com/Fantom-foundation/lachesis-base/utils/piecefunc"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/utils"
)

const (
	// DecimalUnit is the unit of a certainty: DecimalUnit is certain.
	DecimalUnit = piecefunc.DecimalUnit
	// DefaultCertainty asks for the certainty the configuration sets.
	DefaultCertainty = math.MaxUint64
)

var decimalUnit = big.NewInt(DecimalUnit)

// Config holds the GPO keys of the node configuration.
type Config struct {
	// MaxGasPrice is the ceiling of a suggestion.
	MaxGasPrice *big.Int `toml:",omitempty"`
	// MinGasPrice is the floor of a suggestion.
	MinGasPrice *big.Int `toml:",omitempty"`
	// DefaultCertainty is the certainty of a request that names none.
	DefaultCertainty uint64 `toml:",omitempty"`
}

// DefaultMaxGasPrice is the default ceiling of a suggestion, a new value on
// each call.
func DefaultMaxGasPrice() *big.Int {
	return big.NewInt(10_000_000 * params.GWei)
}

// DefaultConfig returns the defaults of the reference.
func DefaultConfig() Config {
	return Config{
		MaxGasPrice:      DefaultMaxGasPrice(),
		MinGasPrice:      new(big.Int),
		DefaultCertainty: 0.5 * DecimalUnit,
	}
}

// Node is what the oracle reads of the node.
type Node interface {
	OracleInputs() core.OracleInputs
	PendingHeads() types.Transactions
}

// OpenRules reads the rules of the open epoch, those of the block after the
// served head.
type OpenRules func(ctx context.Context) (opera.Rules, error)

// Oracle suggests tips. Its sampler reads the head transactions of the pool
// and the open epoch's minimum gas price once a second; the gas power the
// validators have left, which walks the last event of every validator, is
// read for a suggestion only.
type Oracle struct {
	config    Config
	node      Node
	openRules OpenRules

	stats circularStats

	cacheMutex sync.Mutex
	cache      map[uint64]cachedTip
}

type cachedTip struct {
	updated time.Time
	tip     *big.Int
}

// New returns an oracle with the configuration sanitized as the node's
// oracle sanitized it.
func New(config Config, node Node, openRules OpenRules) *Oracle {
	config.MaxGasPrice = sanitize(config.MaxGasPrice, nil, nil, DefaultMaxGasPrice(), "MaxGasPrice")
	config.MinGasPrice = sanitize(config.MinGasPrice, nil, nil, new(big.Int), "MinGasPrice")
	config.DefaultCertainty = sanitize(new(big.Int).SetUint64(config.DefaultCertainty), big.NewInt(0), decimalUnit, big.NewInt(DecimalUnit/2), "DefaultCertainty").Uint64()
	return &Oracle{config: config, node: node, openRules: openRules, cache: map[uint64]cachedTip{}}
}

func sanitize(value, lowest, highest, fallback *big.Int, name string) *big.Int {
	switch {
	case value == nil || (value.Sign() == 0 && fallback.Sign() != 0):
		log.Warn(fmt.Sprintf("Sanitizing invalid parameter %s of gasprice oracle", name), "provided", value, "updated", fallback)
		return fallback
	case lowest != nil && value.Cmp(lowest) < 0:
		log.Warn(fmt.Sprintf("Sanitizing invalid parameter %s of gasprice oracle", name), "provided", value, "updated", lowest)
		return lowest
	case highest != nil && value.Cmp(highest) > 0:
		log.Warn(fmt.Sprintf("Sanitizing invalid parameter %s of gasprice oracle", name), "provided", value, "updated", highest)
		return highest
	}
	return value
}

// cacheSlack is the width of the band of certainties one cached tip answers.
const cacheSlack = DecimalUnit * 0.05

// SuggestTip returns a tip cap that gives a new transaction a chance to be
// included in the next blocks with the given certainty. For a legacy
// transaction the base fee is added to it.
func (oracle *Oracle) SuggestTip(ctx context.Context, certainty uint64) (*big.Int, error) {
	if certainty == DefaultCertainty {
		certainty = oracle.config.DefaultCertainty
	}
	band := certainty / cacheSlack
	if tip, found := oracle.cached(band); found {
		return tip, nil
	}
	tip, err := oracle.suggestTip(ctx, certainty)
	if err != nil {
		return nil, err
	}
	oracle.remember(band, tip)
	return new(big.Int).Set(tip), nil
}

func (oracle *Oracle) cached(band uint64) (*big.Int, bool) {
	oracle.cacheMutex.Lock()
	defer oracle.cacheMutex.Unlock()
	entry, found := oracle.cache[band]
	if !found {
		return nil, false
	}
	if time.Since(entry.updated) >= samplePeriod {
		delete(oracle.cache, band)
		return nil, false
	}
	return new(big.Int).Set(entry.tip), true
}

func (oracle *Oracle) remember(band uint64, tip *big.Int) {
	oracle.cacheMutex.Lock()
	defer oracle.cacheMutex.Unlock()
	oracle.cache[band] = cachedTip{updated: time.Now(), tip: tip}
}

func (oracle *Oracle) suggestTip(ctx context.Context, certainty uint64) (*big.Int, error) {
	rules, err := oracle.openRules(ctx)
	if err != nil {
		return nil, err
	}
	inputs := oracle.node.OracleInputs()
	minPrice := orZero(rules.Economy.MinGasPrice)
	adjustedMinPrice := utils.BigMax(minPrice, orZero(inputs.PendingRules.Economy.MinGasPrice))

	reactive := oracle.stats.priceForGasAbove(certaintyToGasAbove(certainty))
	constructive := constructivePrice(rules, inputs.TotalGasPowerLeft, oracle.stats.totalGas(), 0.005*DecimalUnit+certainty/25, adjustedMinPrice)
	combined := utils.BigMin(utils.BigMax(oracle.config.MinGasPrice, reactive, constructive), oracle.config.MaxGasPrice)

	tip := new(big.Int).Sub(combined, minPrice)
	if inputs.MinTip != nil && tip.Cmp(inputs.MinTip) < 0 {
		return new(big.Int).Set(inputs.MinTip), nil
	}
	return tip, nil
}

// orZero reads a price that rules always carry as zero when they do not.
func orZero(price *big.Int) *big.Int {
	if price == nil {
		return new(big.Int)
	}
	return price
}
