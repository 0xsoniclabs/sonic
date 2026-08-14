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

package subsidies

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/tests/transaction_properties/core"
	"github.com/0xsoniclabs/sonic/tests/transaction_properties/regular"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// testBatch is the generator the property test uses, at the shape it draws it.
func testBatch() *rapid.Generator[[]core.TxSpec] {
	return rapid.Custom(func(t *rapid.T) []core.TxSpec {
		return regular.GenBatch(t, core.GenConfig{
			Network:             testNetwork(opera.GetBrioUpgrades()),
			MaxTxsPerBatch:      4,
			MaxAccountsPerBatch: 3,
			GasBudget:           1_000_000,
		})
	})
}

func testNetwork(upgrades opera.Upgrades) core.NetworkConfig {
	upgrades.GasSubsidies = true
	return core.NetworkConfig{
		ChainId:     big.NewInt(1),
		Upgrades:    upgrades,
		MaxEventGas: 10_000_000,
		MaxBlockGas: 20_000_000,
		MaxTxType:   core.MaxTxTypeFor(upgrades),
		Pricing:     regular.PricingRules,
	}
}

func testBuildContext(t *testing.T) core.BuildContext {
	t.Helper()

	accounts := make([]core.PooledAccount, 3)
	for i := range accounts {
		key, err := core.DeriveKey(i)
		require.NoError(t, err)
		accounts[i] = core.PooledAccount{Account: &tests.Account{PrivateKey: key}, Index: i}
	}
	return core.BuildContext{
		ChainId:     big.NewInt(1),
		Accounts:    accounts,
		UnfundedKey: core.UnfundedKey,
	}
}

// TestSponsoring_LeavesEveryDrawnSpecCoherent applies rapid to the transform itself: whatever the
// inner generator drew, what comes out is a sponsorship request or is not, and either way it is still
// a transaction that can be built and judged.
func TestSponsoring_LeavesEveryDrawnSpecCoherent(t *testing.T) {
	ctx := testBuildContext(t)
	generator := Sponsoring(testBatch(), nothing)

	rapid.Check(t, func(rt *rapid.T) {
		for _, spec := range generator.Draw(rt, "batch") {
			if IsSponsorshipRequest(spec) {
				require.Zero(rt, spec.FeeCap().Sign(), "a request offers nothing for its gas")
				require.False(rt, spec.IsCreate(), "a creation cannot be sponsored")

				// The tip is zeroed along with the fee cap wherever the transform did the zeroing.
				// It can still be higher, because the ordinary tip caps include one above the fee
				// cap and that draw makes a request of a zero fee cap all by itself -- which the
				// model drops for the tip alone, whoever was going to pay.
				if tip := spec.TipCap(); tip.Sign() != 0 {
					require.Positive(rt, tip.Cmp(spec.FeeCap()),
						"a request may only carry a tip that is itself a defect: %s", core.Format(spec))
				}
			}

			_, err := core.BuildTx(spec, 0, ctx)
			require.NoError(rt, err, "a sponsored spec must still be buildable: %s", core.Format(spec))
		}
	})
}

// TestSponsoring_SponsorsEveryTypeItCan pins that the transform reaches every transaction type rather
// than only the one whose price capability it happens to know.
func TestSponsoring_SponsorsEveryTypeItCan(t *testing.T) {
	generator := Sponsoring(testBatch(), nothing)

	sponsored := map[uint8]bool{}
	rapid.Check(t, func(rt *rapid.T) {
		for _, spec := range generator.Draw(rt, "batch") {
			if IsSponsorshipRequest(spec) {
				sponsored[spec.TxType()] = true
			}
		}
	})

	for _, txType := range []uint8{0, 1, 2, 3, 4} {
		require.True(t, sponsored[txType], "no transaction of type %d was ever sponsored", txType)
	}
}

// TestSponsoring_LeavesTheSharedGeneratorValuesAlone guards the one way this transform could poison
// the whole run: a drawn *big.Int is shared between iterations, so zeroing a price has to replace the
// field rather than write through the pointer.
func TestSponsoring_LeavesTheSharedGeneratorValuesAlone(t *testing.T) {
	before := new(big.Int).Set(core.MinimumViableFeeCap)
	generator := Sponsoring(testBatch(), nothing)

	rapid.Check(t, func(rt *rapid.T) {
		generator.Draw(rt, "batch")
		require.Equal(rt, before, core.MinimumViableFeeCap,
			"the shared fee cap value was modified in place")
	})
}

// TestSponsoring_DropsWhatMustNotBeSponsored covers the shapes the domain refuses, including the ones
// the inner generator makes into requests by itself: a zero fee cap is one of the values it draws, so
// declining to sponsor a transaction is not enough to keep it out of the sponsored set.
func TestSponsoring_DropsWhatMustNotBeSponsored(t *testing.T) {
	everything := func(core.TxSpec) bool { return true }
	generator := Sponsoring(testBatch(), everything)

	rapid.Check(t, func(rt *rapid.T) {
		for _, spec := range generator.Draw(rt, "batch") {
			require.False(rt, IsSponsorshipRequest(spec),
				"nothing may be sponsored when everything is unsponsorable: %s", core.Format(spec))
		}
	})
}

func TestUnsponsorable_KeepsStrangersOutOfNetworkSponsoredRunsOnly(t *testing.T) {
	stranger := request(0, 21_000)
	stranger.Signing = core.SignForeign

	// Known defect [4]: a request carrying this value panics the node, whoever pays for it.
	wide := request(0, 21_000)
	wide.Value = new(big.Int).Lsh(big.NewInt(1), 300)

	for mode, wantStranger := range map[Mode]bool{
		ModeFundBacked:     false,
		ModeNetwork:        true,
		ModeNetworkTracked: true,
	} {
		domain := &Domain{Mode: mode}
		require.Equal(t, wantStranger, domain.unsponsorable(stranger),
			"%s and a signature recovering to a stranger", mode)
		require.True(t, domain.unsponsorable(wide),
			"%s must not sponsor a value wider than 256 bits", mode)
		require.False(t, domain.unsponsorable(request(0, 21_000)),
			"%s must sponsor an ordinary transaction", mode)
	}
}

// nothing declares no shape unsponsorable, which is what the transform's own tests want: they are
// about the transform, not about what any one domain refuses.
func nothing(core.TxSpec) bool { return false }
