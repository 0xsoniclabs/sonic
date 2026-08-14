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

package bundles

import (
	"math"
	"math/big"

	"github.com/0xsoniclabs/sonic/tests/transaction_properties/core"
	"github.com/ethereum/go-ethereum/params"
)

// PricingRules decide what becomes of an envelope, which is not a matter of price at all: from Brio
// an envelope is a carrier rather than a transaction, and a carrier never enters a block. Before
// Brio it is an ordinary call to an address with no code, and the rules of the domain being wrapped
// price it like any other. Anything that is not an envelope is theirs in every regime.
func (d *Domain) PricingRules(fallback core.PricingRules) core.PricingRules {
	return func(
		spec core.TxSpec,
		sender core.SenderState,
		baseFee *big.Int,
		cfg core.NetworkConfig,
	) []core.Rule {

		if _, ok := spec.(*Envelope); !ok || !cfg.Upgrades.Brio {
			return fallback(spec, sender, baseFee, cfg)
		}

		reason := "an envelope carrying a bundle never enters a block itself, whatever the bundle did"
		if !cfg.Upgrades.TransactionBundles {
			reason = "bundles are disabled, so an envelope is skipped and its contents never run"
		}
		return []core.Rule{{Applies: true, Reason: reason, Outcomes: core.Dropped}}
	}
}

// Expectation is what the model expects of a bundle: whether its contents must reach a block, and why.
// Where the answer is Uncertain the atomicity and contiguity checks still apply -- they hold whatever
// the bundle did.
type Expectation struct {
	Fate      Fate
	Uncertain bool
	Reason    string
}

func (e Expectation) permits(fate Fate) bool {
	return e.Uncertain || e.Fate == fate
}

func (e Expectation) String() string {
	if e.Uncertain {
		return "{Executed or Skipped} because " + e.Reason
	}
	return "{" + e.Fate.String() + "} because " + e.Reason
}

// markerGas is what the bundle-only marker costs, being one access-list entry with one storage key.
// The builder adds exactly this to every transaction of a bundle, and the marker adds exactly this to
// the intrinsic gas the transaction must cover, so predicting from the drawn gas limit gives the same
// verdict on both sides -- except within this much of a gas boundary, where the two sides can fall
// apart and the prediction has to widen.
const markerGas = params.TxAccessListAddressGas + params.TxAccessListStorageKeyGas

// Expect predicts what becomes of one bundle. The gates are the ones ValidateEnvelope and the block
// processor apply, in the order they apply them, written from the specification rather than by calling
// them -- and then, since the root is a group that must succeed as a whole or a bare step, the bundle
// executes exactly when every transaction of it would.
func (d *Domain) Expect(envelope *Envelope, baseFee *big.Int) Expectation {
	switch {
	case !d.cfg.Upgrades.Brio:
		return Expectation{Fate: FateSkipped, Reason: "bundles need Brio to mean anything"}
	case !d.cfg.Upgrades.TransactionBundles:
		return Expectation{Fate: FateSkipped, Reason: "bundles are disabled"}
	case envelope.Built == nil:
		return Expectation{Fate: FateSkipped, Reason: "the bundle was never built"}
	case envelope.Declared != GasExact:
		return Expectation{Fate: FateSkipped,
			Reason: "the envelope does not declare the gas limit its contents require"}
	case envelope.SigningMode() == core.SignWrongChainID:
		return Expectation{Fate: FateSkipped, Reason: "the envelope is signed for another chain"}
	case envelope.Range != RangeCovering:
		return Expectation{Fate: FateSkipped,
			Reason: "the execution plan's block range does not cover this block"}
	case envelope.Built.GasLimit > d.cfg.MaxBlockGas:
		return Expectation{Fate: FateSkipped, Reason: "the envelope asks for more than a block holds"}

	}

	// The carrier faces the block formation filter like any other transaction, and is deleted from the
	// proposal before anything of the bundle is opened -- so the structural rules of the model apply to
	// it, all of them, minus its price, because nothing charges an envelope.
	//
	// Its signature is not beside the point, though nothing on this path recovers the sender of an
	// envelope: the scrambler that orders the block does, and drops what it cannot recover. So a
	// carrier that recovers to a stranger runs its bundle, and one that recovers to nobody does not.
	if !envelope.SigningMode().Recovers() {
		return Expectation{Fate: FateSkipped,
			Reason: "the sender of the carrier cannot be recovered, so the scrambler leaves it out"}
	}
	if carrier := d.carrierPrediction(envelope, baseFee); !carrier.Permits(core.OutcomeExecuted) {
		return Expectation{Fate: FateSkipped,
			Reason: "the carrier does not survive block formation: " + carrier.Reason}
	}

	// Every transaction of the bundle has to make it, since the root tolerates no failure: a group that
	// must succeed as a whole, or a single step with nothing to catch it. Reaching a block is not
	// enough for one of them -- a transaction that executes and reverts is a failure to the group that
	// carries it, though it would be an ordinary receipt with a zero status on its own.
	if envelope.Built.Refused {
		return Expectation{Fate: FateSkipped,
			Reason: "the contents are not all permissible, so the carrier is deleted with them"}
	}

	uncertain := ""
	for i, spec := range envelope.Contents {
		sender := d.stateOf(spec)
		prediction := envelope.Built.Plans[i].Prediction

		switch {
		case markerPushesCostOver256Bits(spec):
			return Expectation{Fate: FateSkipped,
				Reason: "the marker's gas pushes a transaction's declared cost over 256 bits, which " +
					"ValidateTxStatic refuses"}
		case nearAGasBoundary(spec):
			uncertain = "a transaction of the bundle has a gas limit within the marker's own cost of " +
				"a gas boundary, so the marker may put it on either side"
		case !prediction.Permits(core.OutcomeExecuted):
			return Expectation{Fate: FateSkipped,
				Reason: "a transaction of the bundle cannot execute: " + prediction.Reason}
		case !prediction.Permits(core.OutcomeDropped) && cannotFail(spec, sender, baseFee):
			// This one reaches a block and cannot revert once there; keep looking.
		default:
			uncertain = "whether a transaction of the bundle would execute and succeed is not " +
				"something the model claims to know: " + prediction.Reason
		}
	}
	if uncertain != "" {
		return Expectation{Fate: FateExecuted, Uncertain: true, Reason: uncertain}
	}
	return Expectation{
		Fate:   FateExecuted,
		Reason: "every transaction of the bundle executes and cannot revert",
	}
}

// cannotFail reports whether execution has nothing left to object to once the transaction is applied,
// which is what a group demands of everything it carries. The model claims this only for the plain
// case, because everything else turns on what the EVM makes of a payload:
//
//   - a contract creation runs its data as init code, and a byte of 0x01 is an ADD with nothing to add;
//   - a transfer the sender cannot back after buying its gas reverts, since Sonic's EVM treats an
//     insufficient balance as a revert rather than as an error;
//   - a gas limit merely above the intrinsic cost can still run out inside a precompile.
//
// A call to an account with no code, with gas to spare and a value its sender clearly holds, has none
// of those left to go wrong.
func cannotFail(spec core.TxSpec, sender core.SenderState, baseFee *big.Int) bool {
	if spec.IsCreate() {
		return false
	}
	if spec.Gas() < core.SaturatingAdd(gasBoundary(spec), ampleGas) {
		return false
	}
	// What is left of the balance once the gas is bought, priced at the top of the band the base fee
	// may have moved into.
	left := new(big.Int).Sub(sender.Balance, core.GasCost(spec, core.Scale(baseFee, 4, 1)))
	return left.Sign() > 0 && spec.Amount().Cmp(left) <= 0
}

// ampleGas is the headroom above every intrinsic bound that the generator's own sufficient gas limit
// leaves, and enough for the precompiles this test calls.
const ampleGas = 30_000

// gasBoundary is the highest intrinsic bound a transaction has to clear.
func gasBoundary(spec core.TxSpec) uint64 {
	return max(
		core.LegacyIntrinsicGas(spec),
		core.RevisionIntrinsicGas(spec),
		core.FloorDataGas(spec),
	)
}

// carrierPrediction judges the envelope by the structural rules alone. The copy is what drops the
// signature from the question: a spec is plain data, so a spec that differs only in being correctly
// signed is one assignment away.
func (d *Domain) carrierPrediction(envelope *Envelope, baseFee *big.Int) core.Prediction {
	carrier := *envelope
	carrier.Envelope.Signing = core.SignCorrect

	nonce := uint64(0)
	if envelope.NonceChoice() == core.NonceMax {
		nonce = math.MaxUint64
	}

	cfg := d.cfg
	cfg.Pricing = func(core.TxSpec, core.SenderState, *big.Int, core.NetworkConfig) []core.Rule {
		return nil // an envelope pays nothing, so no price of its own can stop it
	}
	return core.PredictTx(&carrier, nonce, core.SenderState{Balance: new(big.Int)}, baseFee, cfg)
}

// pricingCfg is the network as the contents are priced: they pay their own gas at the base fee, by the
// rules of the domain being wrapped, since being inside a bundle changes nothing about that.
func (d *Domain) pricingCfg() core.NetworkConfig {
	cfg := d.cfg
	cfg.Pricing = d.Inner.Pricing()
	return cfg
}

// markerPushesCostOver256Bits reports whether the transaction the builder makes of this spec declares a
// cost too wide for ValidateTxStatic, although the spec as drawn does not. Cost is priced off the gas
// limit, and the builder raises the gas limit of every transaction of a bundle by the marker's own
// cost, so a fee cap that leaves less than that much headroom under the bound is over it once the
// marker is on. Certain rather than uncertain: the marker is on every one of them.
func markerPushesCostOver256Bits(spec core.TxSpec) bool {
	gas := core.SaturatingAdd(core.SaturatingAdd(spec.Gas(), markerGas), core.BlobGasUsed(spec))
	cost := new(big.Int).Mul(spec.FeeCap(), new(big.Int).SetUint64(gas))
	return cost.Add(cost, spec.Amount()).BitLen() > 256
}

// nearAGasBoundary reports whether the marker's cost could move a transaction across one of the gas
// limits it has to clear, which is where the model has to stop naming one outcome.
func nearAGasBoundary(spec core.TxSpec) bool {
	gas := spec.Gas()
	for _, boundary := range []uint64{
		core.LegacyIntrinsicGas(spec),
		core.RevisionIntrinsicGas(spec),
		core.FloorDataGas(spec),
	} {
		if gas < boundary && boundary-gas <= markerGas {
			return true
		}
	}
	return false
}
