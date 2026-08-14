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

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestDefectLog_SaysNothingWhenNothingWasReproduced(t *testing.T) {
	log := &core.DefectLog{}
	require.Empty(t, log.Summary("Brio"))

	// A run that reproduced nothing must not fail, whatever the flag says.
	log.Report(t, "Brio", opera.Upgrades{Allegro: true})
}

func TestDefectLog_FailsOnAnUnreceiptedChargeOnlyWhereTheDefectLives(t *testing.T) {
	defer func(previous bool) { core.FailOnKnownDefects = previous }(core.FailOnKnownDefects)
	core.FailOnKnownDefects = true

	t.Run("Allegro fixed it, so a sighting only reports", func(t *testing.T) {
		log := &core.DefectLog{}
		log.UnreceiptedCharge(common.Address{0x01}, big.NewInt(1234))
		log.Report(t, "Allegro", opera.Upgrades{Allegro: true})
	})

	t.Run("pre-Allegro a sighting fails", func(t *testing.T) {
		log := &core.DefectLog{}
		log.UnreceiptedCharge(common.Address{0x01}, big.NewInt(1234))

		fake := &testing.T{}
		log.Report(fake, "Sonic", opera.Upgrades{Allegro: false})
		require.True(t, fake.Failed())
	})
}

func TestDefectLog_NamesTheDefectTheForkAndTheEvidence(t *testing.T) {
	account := common.Address{0xab}
	log := &core.DefectLog{}
	log.UnreceiptedCharge(account, big.NewInt(1234))

	summary := log.Summary("Sonic")

	require.Contains(t, summary, "KNOWN DEFECTS REPRODUCED on Sonic")
	require.Contains(t, summary, "keeps the gas it bought")
	require.Contains(t, summary, "(1 occurrence(s))")
	require.Contains(t, summary, account.String())
	require.Contains(t, summary, "paid 1234 wei")
	require.Contains(t, summary, "end of known defects")
}

func TestDefectLog_ReportsOnlyTheDefectsItSaw(t *testing.T) {
	log := &core.DefectLog{}
	log.UnreceiptedCharge(common.Address{0x01}, big.NewInt(1234))

	summary := log.Summary("Sonic")
	require.Contains(t, summary, "keeps the gas it bought")
	require.NotContains(t, summary, "Not reproduced by this run",
		"a defect no domain had to avoid must not be reported")
}

// TestSkip_KeepsOutOnlyCreationsWhoseValueMayNotBeCovered covers the avoidance of defect 1: what is
// skipped is a creation the sender may not be able to hand the value over for, and nothing else -- a
// call of the same shape reverts, spends its nonce and is worth injecting.
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
