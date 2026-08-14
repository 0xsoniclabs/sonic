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

package regular

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/tests/transaction_properties/core"
	"github.com/stretchr/testify/require"
)

func TestSkip_KeepsOutOnlyCreationsWhoseValueMayNotBeCovered(t *testing.T) {
	baseFee := big.NewInt(1)
	sender := core.SenderState{Balance: new(big.Int).Set(core.AccountBalance)}

	creation := func(value *big.Int) *legacyTx {
		spec := affordableSpec(0, core.NonceCorrect)
		spec.To = core.ToCreate
		spec.Value = value
		return spec
	}
	gas := core.Affordability(creation(big.NewInt(0)), core.Scale(baseFee, 4, 1))
	left := new(big.Int).Sub(core.AccountBalance, gas)

	tests := map[string]struct {
		spec    core.TxSpec
		skipped bool
	}{
		"a creation transferring nothing":            {spec: creation(big.NewInt(0))},
		"a creation whose value fits beside its gas": {spec: creation(left)},
		"a creation one wei beyond that": {
			spec: creation(new(big.Int).Add(left, big.NewInt(1))), skipped: true,
		},
		"a creation transferring more than anyone holds": {
			spec: creation(new(big.Int).Lsh(big.NewInt(1), 300)), skipped: true,
		},
		"a call transferring more than its sender holds": {
			spec: func() core.TxSpec {
				spec := affordableSpec(0, core.NonceCorrect)
				spec.Value = new(big.Int).Lsh(big.NewInt(1), 300)
				return spec
			}(),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			reason := Domain{}.Skip(test.spec, sender, baseFee)
			require.Equal(t, test.skipped, reason != "", reason)
		})
	}
}
