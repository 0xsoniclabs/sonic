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
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests/transaction_properties/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

func TestTxTypeCandidates_OffersOneTypeAboveTheForkMaximum(t *testing.T) {
	tests := map[string]struct {
		MaxTxType      uint8
		supported      int
		unsupported    uint8
		hasUnsupported bool
	}{
		"a fork below the highest known type offers one above its maximum": {
			MaxTxType: types.BlobTxType, supported: 4,
			unsupported: types.SetCodeTxType, hasUnsupported: true,
		},
		"a fork at the highest known type has none to offer": {
			MaxTxType: types.SetCodeTxType, supported: 5, hasUnsupported: false,
		},
		"an early fork": {
			MaxTxType: types.LegacyTxType, supported: 1,
			unsupported: types.AccessListTxType, hasUnsupported: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			candidates := TxTypeCandidates(test.MaxTxType)

			unsupported := 0
			for _, candidate := range candidates {
				if candidate > test.MaxTxType {
					unsupported++
					require.Equal(t, test.unsupported, candidate,
						"only the type just above the maximum may be offered")
				}
			}

			require.Equal(t, test.supported*supportedTypeWeight+unsupported, len(candidates))
			if !test.hasUnsupported {
				require.Zero(t, unsupported)
				return
			}
			require.Equal(t, 1, unsupported, "only one entry may be above the maximum")
		})
	}
}

// TestTxTypeCandidates_MakesTheUnsupportedTypeRare pins the weighting on the one fork it applies to,
// as the composition of the list rather than as a measured frequency: rapid draws the index with a
// bias towards the front, so the entry sitting last comes out somewhat rarer than its ninth.
// A batch containing an unsupported type is refused whole, so at even weight this single draw would
// cost about half the coverage of every check downstream of event validation.
func TestTxTypeCandidates_MakesTheUnsupportedTypeRare(t *testing.T) {
	candidates := TxTypeCandidates(types.BlobTxType) // Sonic's maximum

	unsupported := 0
	for _, candidate := range candidates {
		if candidate == types.SetCodeTxType {
			unsupported++
		}
	}

	require.Len(t, candidates, 9)
	require.Equal(t, 1, unsupported, "one entry in nine")
}

func TestTxTypeCandidates_OffersOnlyTypesThatCanBeBuilt(t *testing.T) {
	for _, upgrades := range opera.GetAllHardForksInOrder() {
		for _, candidate := range TxTypeCandidates(core.MaxTxTypeFor(upgrades)) {
			require.NotNil(t, txTypes[candidate],
				"a drawn type must have a spec to build, or generation would panic")
		}
	}
}

// The batch sizes the property test uses, mirrored here so the generators are checked at the shape
// they are drawn at.
const (
	maxTxsPerBatch      = 4
	maxAccountsPerBatch = 3
)

func testGenEnv(upgrades opera.Upgrades) core.GenEnv {
	return core.GenEnv{
		Network:    testNetwork(upgrades),
		Senders:    maxAccountsPerBatch,
		GasCeiling: 1_000_000,
	}
}

// TestGenTxSpec_DrawsOnlyWhatTheTypeCanCarry uses rapid on the generators themselves: whatever is
// drawn, the spec must be internally consistent and buildable.
func TestGenTxSpec_DrawsOnlyWhatTheTypeCanCarry(t *testing.T) {
	for name, upgrades := range opera.GetAllHardForksInOrder() {
		t.Run(name, func(t *testing.T) {
			env := testGenEnv(upgrades)

			rapid.Check(t, func(rt *rapid.T) {
				spec := GenTxSpec(rt, env)

				require.NotNil(rt, txTypes[spec.TxType()])
				require.Less(rt, spec.Sender(), env.Senders)
				require.NotNil(rt, spec.Amount())
				require.NotNil(rt, spec.FeeCap())
				require.NotNil(rt, spec.TipCap())
				require.LessOrEqual(rt, len(spec.Data()), core.MaxDataLen)
				require.LessOrEqual(rt, spec.Gas(), env.GasCeiling)

				// A capability the type lacks must be absent rather than empty.
				if _, ok := spec.(core.WithAccessList); !ok {
					require.Nil(rt, core.AccessListOf(spec))
				}
				require.LessOrEqual(rt, len(core.AccessListOf(spec)), core.MaxAccessListLen)
				require.LessOrEqual(rt, len(core.BlobHashesOf(spec)), 2)
				require.LessOrEqual(rt, len(core.AuthsOf(spec)), 3)

				// A type holding its recipient by value can never be drawn as a creation.
				if _, ok := spec.(core.WithBlobHashes); ok {
					require.False(rt, spec.IsCreate())
				}
			})
		})
	}
}

func TestGenTxSpec_EveryDrawnSpecCanBeBuilt(t *testing.T) {
	ctx := testBuildContext(t)
	env := testGenEnv(opera.GetAllegroUpgrades())

	rapid.Check(t, func(rt *rapid.T) {
		spec := GenTxSpec(rt, env)

		tx, err := core.BuildTx(spec, 0, ctx)
		require.NoError(rt, err, "a drawn spec must be representable: %s", core.Format(spec))
		require.Equal(rt, spec.TxType(), tx.Type())
	})
}

func TestGenBatch_StaysWithinItsLimits(t *testing.T) {
	cfg := core.GenConfig{
		Network:             testNetwork(opera.GetAllegroUpgrades()),
		MaxTxsPerBatch:      maxTxsPerBatch,
		MaxAccountsPerBatch: maxAccountsPerBatch,
		GasBudget:           1_000_000,
	}

	rapid.Check(t, func(rt *rapid.T) {
		specs := GenBatch(rt, cfg)

		require.NotEmpty(rt, specs)
		require.LessOrEqual(rt, len(specs), cfg.MaxTxsPerBatch)

		total := uint64(0)
		for _, spec := range specs {
			require.Less(rt, spec.Sender(), cfg.MaxAccountsPerBatch)
			total = core.SaturatingAdd(total, spec.Gas())
		}

		// Either the batch fits the budget, or one transaction was deliberately made to claim the
		// whole event allowance so that the event-level rejection stays reachable.
		if total > cfg.GasBudget {
			require.Equal(rt, cfg.Network.MaxEventGas, specs[0].Gas())
		}
	})
}

func TestDrawGasLimit_ClaimsTheWholeAllowanceWhenAsked(t *testing.T) {
	env := testGenEnv(opera.GetAllegroUpgrades())
	env.ClaimAllGas = true

	rapid.Check(t, func(rt *rapid.T) {
		spec := GenTxSpec(rt, env)
		require.Equal(rt, env.Network.MaxEventGas, spec.Gas())
	})
}
