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
	"maps"
	"slices"

	"github.com/0xsoniclabs/sonic/tests/transaction_properties/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"pgregory.net/rapid"
)

// txTypes declares every transaction type that can be drawn.
var txTypes = map[uint8]func() core.Drawable{
	types.LegacyTxType:     func() core.Drawable { return &legacyTx{} },
	types.AccessListTxType: func() core.Drawable { return &accessListTx{} },
	types.DynamicFeeTxType: func() core.Drawable { return &dynamicFeeTx{} },
	types.BlobTxType:       func() core.Drawable { return &blobTx{} },
	types.SetCodeTxType:    func() core.Drawable { return &setCodeTx{} },
}

// knownTxTypes are the types above in ascending order, so drawing from them is reproducible.
var knownTxTypes = slices.Sorted(maps.Keys(txTypes))

// GenBatch draws a batch of transactions to inject. Everything here is pure -- no network, no clock,
// no global state -- which is what lets rapid replay and shrink a failing case faithfully.
func GenBatch(t *rapid.T, cfg core.GenConfig) []core.TxSpec {
	count := rapid.IntRange(1, cfg.MaxTxsPerBatch).Draw(t, "txCount")
	oversized := core.OversizedBatch.Draw(t, "core.OversizedBatch")

	specs := make([]core.TxSpec, count)
	for i := range specs {
		specs[i] = GenTxSpec(t, core.GenEnv{
			Network:     cfg.Network,
			Senders:     cfg.MaxAccountsPerBatch,
			GasCeiling:  cfg.GasBudget / uint64(count),
			ClaimAllGas: oversized && i == 0,
			Index:       i,
		})
	}
	return specs
}

// GenTxSpec draws one transaction, letting the drawn type fill its own fields.
func GenTxSpec(t *rapid.T, env core.GenEnv) core.TxSpec {
	spec := txTypes[GenTxType(t, env)]()
	spec.Draw(t, env)
	return spec
}

// supportedTypeWeight is how many times the types a fork supports are offered against a single type
// above its maximum, and so how rare the unsupported draw is: one entry in nine, and about one draw
// in ten, since it sits last and rapid's index draw favours the front. It has to be
// rare because a batch containing one is refused whole, which makes it the largest single drain on the
// coverage of every check downstream of event validation -- at even weight it costs Sonic, the one
// fork where such a type exists, roughly half of that coverage.
const supportedTypeWeight = 2

// GenTxType draws a transaction type: the ones the fork supports, plus one above its maximum so that
// epochcheck's unsupported-type rejection stays reachable.
func GenTxType(t *rapid.T, env core.GenEnv) uint8 {
	return rapid.SampledFrom(TxTypeCandidates(env.Network.MaxTxType)).Draw(t, env.Label("txType"))
}

// TxTypeCandidates is what a type is drawn from, each supported type appearing supportedTypeWeight
// times against a single type above the fork's maximum, where one exists.
func TxTypeCandidates(maxTxType uint8) []uint8 {
	supported := make([]uint8, 0, len(knownTxTypes))
	for _, txType := range knownTxTypes {
		if txType <= maxTxType {
			supported = append(supported, txType)
		}
	}

	candidates := slices.Repeat(supported, supportedTypeWeight)
	if unsupported := maxTxType + 1; txTypes[unsupported] != nil {
		candidates = append(candidates, unsupported)
	}
	return candidates
}

// DrawGasLimit draws a gas limit relative to the payload's own intrinsic cost, so that every
// interesting boundary is reachable, weighted towards limits that suffice because a generator
// producing mostly unusable ones never exercises what happens when a transaction succeeds. The result
// is capped at the batch's share of the gas budget so the batch does not overrun MaxEventGas.
func DrawGasLimit(t *rapid.T, env core.GenEnv, spec core.TxSpec) uint64 {
	if env.ClaimAllGas {
		return env.Network.MaxEventGas
	}

	legacy := core.LegacyIntrinsicGas(spec)
	revision := core.RevisionIntrinsicGas(spec)
	floor := core.FloorDataGas(spec)
	highest := max(legacy, revision, floor)
	sufficient := core.SaturatingAdd(highest, 30_000)

	choice := rapid.OneOf(
		rapid.Just(sufficient),
		rapid.Just(sufficient),
		rapid.Just(sufficient),
		rapid.Just(sufficient),
		rapid.SampledFrom([]uint64{
			0,
			params.TxGas - 1, params.TxGas,
			legacy - 1, legacy,
			revision - 1, revision,
			floor - 1, floor,
			highest, highest + 1,
		}),
		rapid.Uint64Range(0, core.SaturatingAdd(highest, 100_000)),
	).Draw(t, env.Label("gas"))

	return min(choice, env.GasCeiling)
}
