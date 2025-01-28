// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package gasprice

import (
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/utils/piecefunc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/types"
	lru "github.com/hashicorp/golang-lru"

	"github.com/0xsoniclabs/sonic/opera"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

var (
	DefaultMaxGasPrice = big.NewInt(10000000 * params.GWei)
	DecimalUnitBn      = big.NewInt(DecimalUnit)
	secondBn           = new(big.Int).SetUint64(uint64(time.Second))
)

const (
	AsDefaultCertainty = math.MaxUint64
	DecimalUnit        = piecefunc.DecimalUnit
)

type Config struct {
	MaxGasPrice      *big.Int `toml:",omitempty"`
	MinGasPrice      *big.Int `toml:",omitempty"`
	DefaultCertainty uint64   `toml:",omitempty"`
}

type Reader interface {
	GetLatestBlockIndex() idx.Block
	TotalGasPowerLeft() uint64
	GetRules() opera.Rules
	GetPendingRules() opera.Rules
	PendingTxs() map[common.Address]types.Transactions
	MinGasTip() *big.Int
}

type tipCache struct {
	upd time.Time
	tip *big.Int
}

// Oracle recommends gas prices based on the content of recent
// blocks. Suitable for both light and full clients.
type Oracle struct {
	backend Reader

	c circularTxpoolStats

	cfg Config

	tCache *lru.Cache

	wg   sync.WaitGroup
	quit chan struct{}
}

func sanitizeBigInt(val, min, max, _default *big.Int, name string) *big.Int {
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

// NewOracle returns a new gasprice oracle which can recommend suitable
// gasprice for newly created transaction.
func NewOracle(params Config, backend Reader) *Oracle {
	params.MaxGasPrice = sanitizeBigInt(params.MaxGasPrice, nil, nil, DefaultMaxGasPrice, "MaxGasPrice")
	params.MinGasPrice = sanitizeBigInt(params.MinGasPrice, nil, nil, new(big.Int), "MinGasPrice")
	params.DefaultCertainty = sanitizeBigInt(new(big.Int).SetUint64(params.DefaultCertainty), big.NewInt(0), DecimalUnitBn, big.NewInt(DecimalUnit/2), "DefaultCertainty").Uint64()
	tCache, _ := lru.New(100)
	return &Oracle{
		cfg:     params,
		tCache:  tCache,
		quit:    make(chan struct{}),
		backend: backend,
	}
}

func (gpo *Oracle) SetReader(backend Reader) {
	gpo.backend = backend
}

func (gpo *Oracle) Start() {
	gpo.wg.Add(1)
	go func() {
		defer gpo.wg.Done()
		gpo.txpoolStatsLoop()
	}()
}

func (gpo *Oracle) Stop() {
	close(gpo.quit)
	gpo.wg.Wait()
}

func (gpo *Oracle) suggestTip(certainty uint64) *big.Int {
	minPrice := gpo.backend.GetRules().Economy.MinGasPrice
	pendingMinPrice := gpo.backend.GetPendingRules().Economy.MinGasPrice
	adjustedMinGasPrice := math.BigMax(minPrice, pendingMinPrice)

	reactive := gpo.reactiveGasPrice(certainty)
	constructive := gpo.constructiveGasPrice(gpo.c.totalGas(), 0.005*DecimalUnit+certainty/25, adjustedMinGasPrice)

	combined := math.BigMax(reactive, constructive)
	if combined.Cmp(gpo.cfg.MinGasPrice) < 0 {
		combined = gpo.cfg.MinGasPrice
	}
	if combined.Cmp(gpo.cfg.MaxGasPrice) > 0 {
		combined = gpo.cfg.MaxGasPrice
	}

	tip := new(big.Int).Sub(combined, minPrice)
	minGasTip := gpo.backend.MinGasTip()
	if tip.Cmp(minGasTip) < 0 {
		return minGasTip
	}
	return tip
}

// SuggestTip returns a tip cap so that newly created transaction can have a
// very high chance to be included in the following blocks.
//
// Note, for legacy transactions and the legacy eth_gasPrice RPC call, it will be
// necessary to add the basefee to the returned number to fall back to the legacy
// behavior.
func (gpo *Oracle) SuggestTip(certainty uint64) *big.Int {
	if gpo.backend == nil {
		return new(big.Int)
	}
	if certainty == AsDefaultCertainty {
		certainty = gpo.cfg.DefaultCertainty
	}

	const cacheSlack = DecimalUnit * 0.05
	roundedCertainty := certainty / cacheSlack
	if cached, ok := gpo.tCache.Get(roundedCertainty); ok {
		cache := cached.(tipCache)
		if time.Since(cache.upd) < statUpdatePeriod {
			return new(big.Int).Set(cache.tip)
		} else {
			gpo.tCache.Remove(roundedCertainty)
		}
	}

	tip := gpo.suggestTip(certainty)

	gpo.tCache.Add(roundedCertainty, tipCache{
		upd: time.Now(),
		tip: tip,
	})
	return new(big.Int).Set(tip)
}
