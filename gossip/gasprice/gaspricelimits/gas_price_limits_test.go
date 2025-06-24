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

package gaspricelimits

import (
	"math"
	"math/big"
	"testing"
)

func TestAddPercentage_AddsRequestedAmountOfPercent(t *testing.T) {
	for v := range 10000 {
		for p := range 100 {
			res := int(addPercentage(big.NewInt(int64(v)), p).Int64())
			expected := v + v*p/100
			if res != expected {
				t.Log("v: ", v, "p: ", p)
				t.Errorf("Expected %d, got %d", expected, res)
			}
		}
	}
}

func TestAddPercentage_TreatsNilLikeZero(t *testing.T) {
	res := addPercentage(nil, 10)
	if res.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("Expected 0, got %d", res)
	}
}

func TestAddPercentage_CanHandleValuesLargerThanMaxUint64(t *testing.T) {
	value := big.NewInt(math.MaxInt64)
	value = value.Mul(value, value)

	extra := new(big.Int).Div(value, big.NewInt(10))
	want := new(big.Int).Add(value, extra)
	got := addPercentage(value, 10)

	if got.Cmp(want) != 0 {
		t.Errorf("Expected %d, got %d", want, got)
	}
}

func TestGetSuggestedGasPriceForNewTransactions_ReturnsValue10PercentHigherThanBaseFee(t *testing.T) {
	for v := range []int{0, 1, 5, 10, 100, 1000, 10000} {
		baseFee := big.NewInt(int64(v))
		want := addPercentage(baseFee, 10)
		got := GetSuggestedGasPriceForNewTransactions(baseFee)
		if got.Cmp(want) != 0 {
			t.Errorf("Expected %d, got %d", want, got)
		}
	}
}

func TestGetMinimumFeeCapForTransactionPool_ReturnsValue5PercentHigherThanBaseFee(t *testing.T) {
	for v := range []int{0, 1, 5, 10, 100, 1000, 10000} {
		baseFee := big.NewInt(int64(v))
		want := addPercentage(baseFee, 5)
		got := GetMinimumFeeCapForTransactionPool(baseFee)
		if got.Cmp(want) != 0 {
			t.Errorf("Expected %d, got %d", want, got)
		}
	}
}

func TestGetMinimumFeeCapForEventEmitterPool_ReturnsValue2PercentHigherThanBaseFee(t *testing.T) {
	for v := range []int{0, 1, 5, 10, 100, 1000, 10000} {
		baseFee := big.NewInt(int64(v))
		want := addPercentage(baseFee, 5)
		got := GetMinimumFeeCapForEventEmitter(baseFee)
		if got.Cmp(want) != 0 {
			t.Errorf("Expected %d, got %d", want, got)
		}
	}
}
