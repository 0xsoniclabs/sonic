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

// TestDropOffending_KeepsOutACreationTheTransferAHEADOfItDrains is the failure this policy's ordering
// was written for, reduced to two transactions: a creation drawn first but sitting behind a transfer
// in its sender's sequence. The scrambler runs the transfer first, so the creation meets a balance it
// cannot hand its value out of and provokes defect 1 -- a receipt without a nonce -- while a walk in
// the order the batch was drawn sees it against the full balance and lets it through.
func TestDropOffending_KeepsOutACreationTheTransferAheadOfItDrains(t *testing.T) {
	baseFee := big.NewInt(1)
	half := new(big.Int).Div(core.AccountBalance, big.NewInt(2))

	creation := affordableSpec(0, core.NonceGap)
	creation.NonceGapSize = 1
	creation.To = core.ToCreate
	creation.Value = half

	transfer := affordableSpec(0, core.NonceCorrect)
	transfer.To = core.ToOther
	transfer.Value = half

	specs := []core.TxSpec{creation, transfer}
	senders := []core.SenderState{{Balance: new(big.Int).Set(core.AccountBalance)}}

	skipped := map[string]int{}
	kept := core.DropOffending(specs, senders, baseFee, Domain{}.Skip,
		func(reason string) { skipped[reason]++ })

	require.Equal(t, []core.TxSpec{transfer}, kept,
		"the creation runs behind the transfer, so it must not be injected")
	require.Equal(t, 1, skipped[defect1], "the reason must be tallied for the report")

	// The same two inside a bundle, which nothing reorders: there the creation really does run first,
	// against the whole balance, and there is nothing to keep out.
	kept = core.DropOffendingInGivenOrder(specs, senders, baseFee, Domain{}.Skip,
		func(string) { t.Error("nothing runs ahead of the creation here") })
	require.Equal(t, specs, kept)
}
