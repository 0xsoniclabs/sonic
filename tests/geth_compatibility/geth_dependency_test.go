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

package geth_compatibility_tests

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestGethDependency_EffectiveGasTipProducesUnchangedResults(t *testing.T) {
	tests := map[string]struct {
		baseFee     int64
		gasTipCap   int64
		gasFeeCap   int64
		expectedTip int64
	}{
		"all zero": {
			baseFee:     0,
			gasTipCap:   0,
			gasFeeCap:   0,
			expectedTip: 0,
		},
		"all equal": {
			baseFee:     50,
			gasTipCap:   50,
			gasFeeCap:   50,
			expectedTip: 0,
		},
		"tip limited by tip cap": {
			baseFee:     50,
			gasTipCap:   20,
			gasFeeCap:   100,
			expectedTip: 20,
		},
		"tip limited by fee cap": {
			baseFee:     50,
			gasTipCap:   100,
			gasFeeCap:   70,
			expectedTip: 20,
		},
		"tip cap equal to fee cap minus base fee": {
			baseFee:     50,
			gasTipCap:   50,
			gasFeeCap:   100,
			expectedTip: 50,
		},
		"fee cap equal to base fee": {
			baseFee:     50,
			gasTipCap:   10,
			gasFeeCap:   50,
			expectedTip: 0,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			txData := &types.DynamicFeeTx{
				GasTipCap: big.NewInt(test.gasTipCap),
				GasFeeCap: big.NewInt(test.gasFeeCap),
			}
			tx := types.NewTx(txData)
			effectiveTip, err := tx.EffectiveGasTip(big.NewInt(test.baseFee))
			require.NoError(t, err)
			require.Zero(t, big.NewInt(test.expectedTip).Cmp(effectiveTip))
		})
	}
}

func TestGethDependency_EffectiveGasTipReturnsUnchangedErrors(t *testing.T) {
	overflow := big.NewInt(0).Lsh(big.NewInt(1), 256) // 2^256 overflows uint256
	tests := map[string]struct {
		baseFee   *big.Int
		gasTipCap *big.Int
		gasFeeCap *big.Int

		// Although returned values should be ignored when an error is expected,
		// we want to see all catch all behavior changes.
		expectedTip *big.Int
	}{
		"fee cap less than base fee": {
			baseFee:     big.NewInt(50),
			gasTipCap:   big.NewInt(10),
			gasFeeCap:   big.NewInt(40),
			expectedTip: big.NewInt(10),
		},
		"base fee overflow": {
			baseFee:     overflow,
			expectedTip: nil,
		},
		"tip cap overflow": {
			gasTipCap:   overflow,
			expectedTip: big.NewInt(0),
		},
		"fee cap overflow": {
			gasFeeCap:   overflow,
			expectedTip: big.NewInt(0),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			txData := &types.DynamicFeeTx{
				GasTipCap: test.gasTipCap,
				GasFeeCap: test.gasFeeCap,
			}
			tx := types.NewTx(txData)
			tip, err := tx.EffectiveGasTip(test.baseFee)
			require.Error(t, err)

			if test.expectedTip == nil {
				require.Nil(t, tip)
			} else {
				require.Zero(t, test.expectedTip.Cmp(tip))
			}
		})
	}
}
