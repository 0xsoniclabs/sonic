// Copyright 2025 Sonic Operations Ltd
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

package gasprice

import (
	"math/big"
	"testing"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/opera"
)

type fakeTx struct {
	gas uint64
	tip *big.Int
	cap *big.Int
}

type TestBackend struct {
	block             idx.Block
	totalGasPowerLeft uint64
	rules             opera.Rules
	pendingRules      opera.Rules
	pendingTxs        []fakeTx
}

func (t TestBackend) GetLatestBlockIndex() idx.Block {
	return t.block
}

func (t TestBackend) TotalGasPowerLeft() uint64 {
	return t.totalGasPowerLeft
}

func (t TestBackend) GetRules() opera.Rules {
	return t.rules
}

func (t TestBackend) GetPendingRules() opera.Rules {
	return t.pendingRules
}

func (t TestBackend) PendingTxs() map[common.Address]types.Transactions {
	txs := make(map[common.Address]types.Transactions, len(t.pendingTxs))
	for i, tx := range t.pendingTxs {
		txs[common.BytesToAddress(big.NewInt(int64(i)).Bytes())] = types.Transactions{
			types.NewTx(&types.DynamicFeeTx{
				Nonce:     uint64(i),
				GasTipCap: tx.tip,
				GasFeeCap: tx.cap,
				Gas:       tx.gas,
			}),
		}
	}
	return txs
}

func (t TestBackend) MinGasTip() *big.Int {
	return big.NewInt(0)
}

func TestOracle_constructiveGasPrice(t *testing.T) {
	backend := &TestBackend{
		totalGasPowerLeft: 0,
		rules:             opera.FakeNetRules(opera.GetSonicUpgrades()),
		pendingRules:      opera.FakeNetRules(opera.GetSonicUpgrades()),
	}

	gpo := NewOracle(Config{}, backend)
	gpo.cfg.MaxGasPrice = math.MaxBig256
	gpo.cfg.MinGasPrice = new(big.Int)

	// all the gas is consumed, price should be high
	backend.totalGasPowerLeft = 0
	require.Equal(t, "2500", gpo.constructiveGasPrice(0, 0, big.NewInt(100)).String())
	require.Equal(t, "2500", gpo.constructiveGasPrice(0, 0.1*DecimalUnit, big.NewInt(100)).String())
	require.Equal(t, "2500", gpo.constructiveGasPrice(10080000, 0, big.NewInt(100)).String())
	require.Equal(t, "2500", gpo.constructiveGasPrice(gpo.maxTotalGasPower().Uint64()*2, 2*DecimalUnit, big.NewInt(100)).String())

	// all the gas is free, price should be low
	backend.totalGasPowerLeft = gpo.maxTotalGasPower().Uint64()
	require.Equal(t, "100", gpo.constructiveGasPrice(0, 0, big.NewInt(100)).String())
	require.Equal(t, "120", gpo.constructiveGasPrice(0, 0.1*DecimalUnit, big.NewInt(100)).String())
	require.Equal(t, "110", gpo.constructiveGasPrice(10080000, 0, big.NewInt(100)).String())
	require.Equal(t, "2500", gpo.constructiveGasPrice(gpo.maxTotalGasPower().Uint64()*2, 2*DecimalUnit, big.NewInt(100)).String())

	// half of the gas is free, price should be 3.75x
	backend.totalGasPowerLeft = gpo.maxTotalGasPower().Uint64() / 2
	require.Equal(t, "375", gpo.constructiveGasPrice(0, 0, big.NewInt(100)).String())
	require.Equal(t, "637", gpo.constructiveGasPrice(0, 0.1*DecimalUnit, big.NewInt(100)).String())
	require.Equal(t, "551", gpo.constructiveGasPrice(10080000, 0, big.NewInt(100)).String())
	require.Equal(t, "2500", gpo.constructiveGasPrice(gpo.maxTotalGasPower().Uint64()*2, 2*DecimalUnit, big.NewInt(100)).String())

	// third of the gas is free, price should be higher
	backend.totalGasPowerLeft = gpo.maxTotalGasPower().Uint64() / 3
	require.Equal(t, "812", gpo.constructiveGasPrice(0, 0, big.NewInt(100)).String())
	require.Equal(t, "1255", gpo.constructiveGasPrice(0, 0.1*DecimalUnit, big.NewInt(100)).String())
	require.Equal(t, "1080", gpo.constructiveGasPrice(10080000, 0, big.NewInt(100)).String())
	require.Equal(t, "2500", gpo.constructiveGasPrice(gpo.maxTotalGasPower().Uint64()*2, 2*DecimalUnit, big.NewInt(100)).String())

}

func TestOracle_reactiveGasPrice(t *testing.T) {
	backend := &TestBackend{
		totalGasPowerLeft: 0,
		rules:             opera.FakeNetRules(opera.GetSonicUpgrades()),
		pendingRules:      opera.FakeNetRules(opera.GetSonicUpgrades()),
	}

	gpo := NewOracle(Config{}, backend)
	gpo.cfg.MaxGasPrice = math.MaxBig256
	gpo.cfg.MinGasPrice = new(big.Int)

	// no stats -> zero price
	gpo.c = circularTxpoolStats{}
	require.Equal(t, "0", gpo.reactiveGasPrice(0).String())
	require.Equal(t, "0", gpo.reactiveGasPrice(DecimalUnit).String())
	gpo.txpoolStatsTick()
	require.Equal(t, "0", gpo.reactiveGasPrice(0).String())
	require.Equal(t, "0", gpo.reactiveGasPrice(DecimalUnit).String())

	// one tx
	gpo.c = circularTxpoolStats{}
	backend.pendingTxs = append(backend.pendingTxs, fakeTx{
		gas: 50000,
		tip: big.NewInt(0),
		cap: big.NewInt(1e9),
	})
	require.Equal(t, "0", gpo.reactiveGasPrice(0).String())
	require.Equal(t, "0", gpo.reactiveGasPrice(DecimalUnit).String())
	gpo.txpoolStatsTick()
	require.Equal(t, "0", gpo.reactiveGasPrice(0).String())
	require.Equal(t, "0", gpo.reactiveGasPrice(0.8*DecimalUnit).String())
	require.Equal(t, "200000000", gpo.reactiveGasPrice(0.9*DecimalUnit).String())
	require.Equal(t, "600000000", gpo.reactiveGasPrice(0.95*DecimalUnit).String())
	require.Equal(t, "920000000", gpo.reactiveGasPrice(0.99*DecimalUnit).String())
	require.Equal(t, "1000000000", gpo.reactiveGasPrice(DecimalUnit).String())

	// add one more tx
	backend.pendingTxs = append(backend.pendingTxs, fakeTx{
		gas: 25000,
		tip: big.NewInt(3 * 1e9),
		cap: big.NewInt(3.5 * 1e9),
	})

	require.Equal(t, "0", gpo.reactiveGasPrice(0).String())
	require.Equal(t, "1000000000", gpo.reactiveGasPrice(DecimalUnit).String())
	gpo.txpoolStatsTick()
	require.Equal(t, "0", gpo.reactiveGasPrice(0).String())
	require.Equal(t, "0", gpo.reactiveGasPrice(0.8*DecimalUnit).String())
	require.Equal(t, "450000000", gpo.reactiveGasPrice(0.9*DecimalUnit).String())
	require.Equal(t, "1350000000", gpo.reactiveGasPrice(0.95*DecimalUnit).String())
	require.Equal(t, "2070000000", gpo.reactiveGasPrice(0.99*DecimalUnit).String())
	require.Equal(t, "2250000000", gpo.reactiveGasPrice(DecimalUnit).String())

	// add two more txs
	backend.pendingTxs = append(backend.pendingTxs, fakeTx{
		gas: 2500000,
		tip: big.NewInt(1 * 1e9),
		cap: big.NewInt(3.5 * 1e9),
	})
	backend.pendingTxs = append(backend.pendingTxs, fakeTx{
		gas: 2500000,
		tip: big.NewInt(0 * 1e9),
		cap: big.NewInt(3.5 * 1e9),
	})

	gpo.txpoolStatsTick()
	require.Equal(t, "0", gpo.reactiveGasPrice(0).String())
	require.Equal(t, "333333333", gpo.reactiveGasPrice(0.8*DecimalUnit).String())
	require.Equal(t, "799999999", gpo.reactiveGasPrice(0.9*DecimalUnit).String())
	require.Equal(t, "1733333332", gpo.reactiveGasPrice(0.95*DecimalUnit).String())
	require.Equal(t, "2479999999", gpo.reactiveGasPrice(0.99*DecimalUnit).String())
	require.Equal(t, "2666666666", gpo.reactiveGasPrice(DecimalUnit).String())
	// price gets closer to latest state
	gpo.txpoolStatsTick()
	require.Equal(t, "500000000", gpo.reactiveGasPrice(0.8*DecimalUnit).String())
	require.Equal(t, "2875000000", gpo.reactiveGasPrice(DecimalUnit).String())
	gpo.txpoolStatsTick()
	require.Equal(t, "600000000", gpo.reactiveGasPrice(0.8*DecimalUnit).String())
	require.Equal(t, "3000000000", gpo.reactiveGasPrice(DecimalUnit).String())
	gpo.txpoolStatsTick()
	require.Equal(t, "666666666", gpo.reactiveGasPrice(0.8*DecimalUnit).String())
	require.Equal(t, "3083333333", gpo.reactiveGasPrice(DecimalUnit).String())
	for i := 0; i < statsBuffer-5; i++ {
		gpo.txpoolStatsTick()
	}
	require.Equal(t, "933333333", gpo.reactiveGasPrice(0.8*DecimalUnit).String())
	require.Equal(t, "3500000000", gpo.reactiveGasPrice(DecimalUnit).String())
	gpo.txpoolStatsTick()
	require.Equal(t, "1000000000", gpo.reactiveGasPrice(0.8*DecimalUnit).String())
	require.Equal(t, "3500000000", gpo.reactiveGasPrice(DecimalUnit).String())
	gpo.txpoolStatsTick()
	require.Equal(t, "1000000000", gpo.reactiveGasPrice(0.8*DecimalUnit).String())
	require.Equal(t, "3500000000", gpo.reactiveGasPrice(DecimalUnit).String())

	// change minGasPrice
	backend.rules.Economy.MinGasPrice = big.NewInt(100)
	gpo.txpoolStatsTick()
	require.Equal(t, "933333340", gpo.reactiveGasPrice(0.8*DecimalUnit).String())
	require.Equal(t, "3466666673", gpo.reactiveGasPrice(DecimalUnit).String())
	gpo.txpoolStatsTick()
	require.Equal(t, "866666680", gpo.reactiveGasPrice(0.8*DecimalUnit).String())
	require.Equal(t, "3433333346", gpo.reactiveGasPrice(DecimalUnit).String())
	gpo.txpoolStatsTick()
	require.Equal(t, "800000020", gpo.reactiveGasPrice(0.8*DecimalUnit).String())
	require.Equal(t, "3400000020", gpo.reactiveGasPrice(DecimalUnit).String())
	gpo.txpoolStatsTick()
	// recent gas price plus 5%
	require.Equal(t, "105", gpo.reactiveGasPrice(0.8*DecimalUnit).String())
	require.Equal(t, "3150000105", gpo.reactiveGasPrice(DecimalUnit).String())
	for i := 0; i < statsBuffer-5; i++ {
		gpo.txpoolStatsTick()
	}
	require.Equal(t, "105", gpo.reactiveGasPrice(0.8*DecimalUnit).String())
	require.Equal(t, "3033333426", gpo.reactiveGasPrice(DecimalUnit).String())
	gpo.txpoolStatsTick()
	require.Equal(t, "100", gpo.reactiveGasPrice(0.8*DecimalUnit).String())
	require.Equal(t, "3000000100", gpo.reactiveGasPrice(DecimalUnit).String())
	gpo.txpoolStatsTick()
	require.Equal(t, "100", gpo.reactiveGasPrice(0.8*DecimalUnit).String())
	require.Equal(t, "3000000100", gpo.reactiveGasPrice(DecimalUnit).String())

	// half of txs are confirmed now
	backend.pendingTxs = backend.pendingTxs[:2]
	gpo.txpoolStatsTick()
	require.Equal(t, "93", gpo.reactiveGasPrice(0.8*DecimalUnit).String())
	require.Equal(t, "3000000100", gpo.reactiveGasPrice(DecimalUnit).String())
	gpo.txpoolStatsTick()
	require.Equal(t, "86", gpo.reactiveGasPrice(0.8*DecimalUnit).String())
	require.Equal(t, "3000000100", gpo.reactiveGasPrice(DecimalUnit).String())
	for i := 0; i < statsBuffer-3; i++ {
		gpo.txpoolStatsTick()
	}
	require.Equal(t, "0", gpo.reactiveGasPrice(0.8*DecimalUnit).String())
	require.Equal(t, "3000000100", gpo.reactiveGasPrice(DecimalUnit).String())
	gpo.txpoolStatsTick()
	require.Equal(t, "0", gpo.reactiveGasPrice(0.8*DecimalUnit).String())
	require.Equal(t, "3000000100", gpo.reactiveGasPrice(DecimalUnit).String())
	gpo.txpoolStatsTick()
	require.Equal(t, "0", gpo.reactiveGasPrice(0.8*DecimalUnit).String())
	require.Equal(t, "3000000100", gpo.reactiveGasPrice(DecimalUnit).String())

	// all txs are confirmed now
	backend.pendingTxs = backend.pendingTxs[:0]
	gpo.txpoolStatsTick()
	require.Equal(t, "0", gpo.reactiveGasPrice(0.8*DecimalUnit).String())
	require.Equal(t, "3000000100", gpo.reactiveGasPrice(DecimalUnit).String())
	gpo.txpoolStatsTick()
	require.Equal(t, "0", gpo.reactiveGasPrice(0.8*DecimalUnit).String())
	require.Equal(t, "3000000100", gpo.reactiveGasPrice(DecimalUnit).String())
	for i := 0; i < statsBuffer-3; i++ {
		gpo.txpoolStatsTick()
	}
	require.Equal(t, "0", gpo.reactiveGasPrice(0.8*DecimalUnit).String())
	require.Equal(t, "0", gpo.reactiveGasPrice(DecimalUnit).String())
	gpo.txpoolStatsTick()
	require.Equal(t, "0", gpo.reactiveGasPrice(0.8*DecimalUnit).String())
	require.Equal(t, "0", gpo.reactiveGasPrice(DecimalUnit).String())
	gpo.txpoolStatsTick()
	require.Equal(t, "0", gpo.reactiveGasPrice(0.8*DecimalUnit).String())
	require.Equal(t, "0", gpo.reactiveGasPrice(DecimalUnit).String())
}
