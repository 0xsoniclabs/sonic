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

package core

import (
	"bytes"
	"cmp"
	"fmt"
	"math"
	"math/big"
	"slices"
	"strings"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/params"
)

// PricingRules are the rules deciding whether a transaction can pay its way, tried after the
// structural ones and in the order given. A domain provides its own, so that whatever prices it drew
// are judged by whoever knows what they mean.
type PricingRules func(
	spec TxSpec,
	sender SenderState,
	baseFee *big.Int,
	cfg NetworkConfig,
) []Rule

// NetworkConfig is what the model needs to know about a network. It is read once per network.
type NetworkConfig struct {
	ChainId     *big.Int
	Upgrades    opera.Upgrades
	MaxEventGas uint64
	MaxBlockGas uint64
	MaxTxType   uint8

	// Pricing is the tail of PredictTx's rule table, taken from the domain under test.
	Pricing PricingRules
}

// Prediction is what the model expects to observe for one transaction. Allowed usually holds a
// single outcome, but naming one where the pipeline has two legal answers would only produce false
// failures.
type Prediction struct {
	Allowed []Outcome
	Reason  string
}

func Allow(reason string, outcomes ...Outcome) Prediction {
	return Prediction{Allowed: outcomes, Reason: reason}
}

func (p Prediction) Permits(outcome Outcome) bool {
	return slices.Contains(p.Allowed, outcome)
}

func (p Prediction) String() string {
	names := make([]string, 0, len(p.Allowed))
	for _, outcome := range p.Allowed {
		names = append(names, outcome.String())
	}
	return "{" + strings.Join(names, " or ") + "} because " + p.Reason
}

// Dropped and Either are the outcome sets a rule can allow: a transaction that must not execute, and
// one whose fate the model cannot pin down.
var (
	Dropped = []Outcome{OutcomeDropped}
	Either  = []Outcome{OutcomeDropped, OutcomeExecuted}
)

// TxPlan is everything decided about one transaction before it is built.
type TxPlan struct {
	Nonce      uint64
	Prediction Prediction
}

// SenderState is what the model knows about a sender.
type SenderState struct {
	Nonce   uint64
	Balance *big.Int
}

// PlanBatch resolves the nonce of every transaction in a batch and predicts each outcome, returning
// whether the batch as a whole is expected to be refused. The nonce is decided here because only
// this sees the whole batch, and the batch verdict is separate because event validation is
// all-or-nothing: one malformed transaction must stop every other transaction in its batch from
// having any effect.
//
// The batch is taken to be reorderable, which an injected one is: the scrambler puts one sender's
// transactions in nonce order whatever order they arrived in. Use PlanBatchInGivenOrder for a batch
// that executes in the order it is written.
func PlanBatch(
	specs []TxSpec,
	senders []SenderState,
	baseFee *big.Int,
	cfg NetworkConfig,
) (plans []TxPlan, batchRejected bool, reason string) {
	return planBatch(specs, senders, baseFee, cfg, true)
}

// PlanBatchInGivenOrder is PlanBatch for a batch nothing reorders, where a sender's nonces have to
// ascend in the order the batch itself gives rather than merely all be present. A bundle is that: its
// steps are executed in the order its execution plan references them, and they reach the block in that
// order too, so a run of nonces given back to front executes in an ordinary batch and fails in a
// bundle.
func PlanBatchInGivenOrder(
	specs []TxSpec,
	senders []SenderState,
	baseFee *big.Int,
	cfg NetworkConfig,
) (plans []TxPlan, batchRejected bool, reason string) {
	return planBatch(specs, senders, baseFee, cfg, false)
}

func planBatch(
	specs []TxSpec,
	senders []SenderState,
	baseFee *big.Int,
	cfg NetworkConfig,
	reorderable bool,
) (plans []TxPlan, batchRejected bool, reason string) {

	totalGas := uint64(0)
	for _, spec := range specs {
		if failure := EventLevelFailure(spec, cfg); failure != "" {
			return rejectAll(specs, senders, failure), true, failure
		}
		totalGas = SaturatingAdd(totalGas, spec.Gas())
	}
	if totalGas >= cfg.MaxEventGas {
		const reason = "total gas of the batch reaches MaxEventGas"
		return rejectAll(specs, senders, reason), true, reason
	}

	nonces := AssignNonces(specs, senders)

	plans = make([]TxPlan, len(specs))
	for i, spec := range specs {
		plans[i] = TxPlan{
			Nonce:      nonces[i],
			Prediction: PredictTx(spec, nonces[i], senders[SenderOf(spec, senders)], baseFee, cfg),
		}
	}

	ResolveNonceOrder(specs, senders, baseFee, plans, reorderable)
	return plans, false, ""
}

// SenderOf indexes the sender a spec draws from, taken modulo the number claimed so that a generator
// need not know how many there are.
func SenderOf(spec TxSpec, senders []SenderState) int {
	return spec.Sender() % len(senders)
}

// AssignNonces gives every transaction of a batch the nonce it will carry, depending only on the
// senders' on-chain nonces and never on what the model expects, which is what keeps it free of
// circularity. Successive correct nonces from one sender get successive values, so a batch from a
// single sender forms a usable run rather than a pile of duplicates.
func AssignNonces(specs []TxSpec, senders []SenderState) []uint64 {
	next := make([]uint64, len(senders))
	for i := range next {
		next[i] = senders[i].Nonce
	}

	nonces := make([]uint64, len(specs))
	for i, spec := range specs {
		index := SenderOf(spec, senders)
		switch spec.NonceChoice() {
		case NonceCorrect:
			nonces[i] = next[index]
			next[index]++
		case NonceTooLow:
			nonces[i] = senders[index].Nonce - min(senders[index].Nonce, 1)
		case NonceGap:
			nonces[i] = SaturatingAdd(next[index], max(spec.GapSize(), 1))
		case NonceMax:
			nonces[i] = math.MaxUint64
		}
	}
	return nonces
}

// ResolveNonceOrder decides, per sender, which transactions the nonce sequence admits.
//
// A reorderable batch is walked in ascending nonce order rather than in the order it was injected: the
// scrambler reorders a block's transactions and guarantees one sender's run in nonce order, so a batch
// carrying nonces 0, 3, 1, 2 executes all four, and judging them as injected would wrongly call 3 a
// gap. A batch nothing reorders is walked as written, where those same nonces leave everything after
// the 3 behind a hole.
func ResolveNonceOrder(
	specs []TxSpec,
	senders []SenderState,
	baseFee *big.Int,
	plans []TxPlan,
	reorderable bool,
) {
	for index := range senders {
		order := make([]int, 0, len(specs))
		for i, spec := range specs {
			if SenderOf(spec, senders) == index && plans[i].Prediction.Permits(OutcomeExecuted) {
				order = append(order, i)
			}
		}
		if reorderable {
			slices.SortStableFunc(order, func(a, b int) int {
				return cmp.Compare(plans[a].Nonce, plans[b].Nonce)
			})
		}

		rivals := map[uint64]int{}
		for _, i := range order {
			rivals[plans[i].Nonce]++
		}

		balance := new(big.Int).Set(senders[index].Balance)
		expected := senders[index].Nonce
		uncertain := false

		for _, i := range order {
			required := Affordability(specs[i], Scale(baseFee, 4, 1))

			// What an earlier transaction of this sender may already have taken with it decides this
			// one too, since a sender that has parted with its balance can no longer buy gas. This is
			// only reachable when something spends nothing on gas itself: an ordinary transaction
			// pays for its gas before it transfers, so a transfer that would leave its sender unable
			// to pay reverts instead, while a sponsored one pays nothing and can hand over the lot.
			drained := balance.Cmp(required) < 0

			balance.Sub(balance, required)
			balance.Sub(balance, specs[i].Amount())

			switch {
			case uncertain:
				plans[i].Prediction = Allow(
					"an earlier transaction from this sender may or may not have executed, so the "+
						"sequence position is unknown",
					Either...,
				)

			case rivals[plans[i].Nonce] > 1:
				plans[i].Prediction = Allow(
					"it competes for a nonce with another transaction of the batch, so at most "+
						"one of them can execute",
					Either...,
				)
				if plans[i].Nonce == expected {
					expected++
				}

			case plans[i].Nonce != expected:
				plans[i].Prediction = Allow(fmt.Sprintf(
					"its nonce %d is not the %d the sender's sequence has reached",
					plans[i].Nonce, expected,
				), Dropped...)

			case drained:
				plans[i].Prediction = Allow(
					"an earlier transaction from this sender may have taken away what this one needs "+
						"to buy its gas",
					Either...,
				)
				expected++

			default:
				expected++
			}

			uncertain = uncertain || drained
		}
	}
}

// rejectAll plans a batch that event validation refuses outright, where nothing executes and so
// every nonce is simply its sender's current one.
func rejectAll(specs []TxSpec, senders []SenderState, reason string) []TxPlan {
	nonces := AssignNonces(specs, senders)
	plans := make([]TxPlan, len(specs))
	for i := range specs {
		plans[i] = TxPlan{Nonce: nonces[i], Prediction: Allow(reason, OutcomeEventRejected)}
	}
	return plans
}

// EventLevelFailure reports why an event carrying this transaction must be refused outright, or ""
// when nothing at this level objects. That is only epochcheck: an event from a peer also goes
// through basiccheck and heavycheck, but a self-emitted one skips both, so an absurd gas limit or an
// unrecoverable signature is not refused up front and is dealt with further down.
func EventLevelFailure(spec TxSpec, cfg NetworkConfig) string {
	if spec.TxType() > cfg.MaxTxType {
		return "transaction type is above the maximum this fork supports"
	}
	return ""
}

// Rule is one reason the model has for expecting something other than plain execution.
type Rule struct {
	Applies  bool
	Reason   string
	Outcomes []Outcome
}

// PredictTx predicts the outcome of a transaction that entered the chain as part of an accepted
// event. The rules are tried in order, so the first that applies gives the reason. They are written
// from the specification rather than by calling the validators under test, which would assert
// nothing.
//
// The rules here are the structural ones, which hold whoever pays for the transaction. The ones
// deciding whether it can pay its way come last, from cfg.Pricing, because that answer belongs to
// the domain that drew the prices: an ordinary transaction pays from its sender's balance, while a
// sponsorship request pays from a fund and cannot be judged by the same table.
func PredictTx(
	spec TxSpec,
	nonce uint64,
	sender SenderState,
	baseFee *big.Int,
	cfg NetworkConfig,
) Prediction {

	var (
		feeCap          = spec.FeeCap()
		_, carriesAuths = spec.(WithAuths)
	)

	for _, rule := range append([]Rule{
		{spec.SigningMode() != SignCorrect,
			"the signature is not the sender's own", Dropped},
		{spec.Gas() < LegacyIntrinsicGas(spec),
			"gas limit is below the legacy intrinsic gas of the payload", Dropped},
		{spec.SeparateTipCap() && feeCap.Cmp(spec.TipCap()) < 0,
			"gas tip cap exceeds gas fee cap", Dropped},
		{cfg.Upgrades.Allegro && len(BlobHashesOf(spec)) > 0,
			"blob transactions carrying blob hashes are not permissible from Allegro onwards", Dropped},
		{carriesAuths && len(AuthsOf(spec)) == 0,
			"a set-code transaction needs a non-empty authorization list", Dropped},
		{spec.Amount().BitLen() > 256,
			"value does not fit in 256 bits", Dropped},
		{feeCap.BitLen() > 256,
			"gas fee cap does not fit in 256 bits", Dropped},
		{spec.TipCap().BitLen() > 256,
			"gas tip cap does not fit in 256 bits", Dropped},
		{cfg.Upgrades.Brio && DeclaredCost(spec).BitLen() > 256,
			"cost does not fit in 256 bits, which ValidateTxStatic rejects from Brio onwards", Dropped},
		{nonce == math.MaxUint64,
			"nonce is the maximum uint64, which ValidateTxStatic rejects", Dropped},
		{spec.Gas() < RevisionIntrinsicGas(spec),
			"gas limit is below the intrinsic gas of the current revision", Dropped},
		{cfg.Upgrades.Allegro && spec.Gas() < FloorDataGas(spec),
			"gas limit is below the EIP-7623 floor data gas", Dropped},
		{spec.Gas() > cfg.MaxBlockGas,
			"gas limit exceeds the block gas limit", Dropped},
	}, cfg.Pricing(spec, sender, baseFee, cfg)...) {
		if rule.Applies {
			return Allow(rule.Reason, rule.Outcomes...)
		}
	}

	return Allow("all checks the model knows about are satisfied", OutcomeExecuted)
}

// MinimumViableFeeCap is the fee cap the generator gives transactions meant to pay their way: a
// hundred times the network's minimum gas price, which leaves ample room above any base fee an
// almost idle network reaches. checkModelAssumptions verifies that at run time.
var MinimumViableFeeCap = new(big.Int).Mul(big.NewInt(1e9), big.NewInt(100))

// EffectiveGasPrice is the price a transaction pays per unit of gas, mirroring
// core.TransactionToMessage: the fee cap, or the tip plus the base fee if that is lower.
func EffectiveGasPrice(spec TxSpec, baseFee *big.Int) *big.Int {
	feeCap := spec.FeeCap()
	withBaseFee := new(big.Int).Add(spec.TipCap(), baseFee)
	if !spec.SeparateTipCap() || withBaseFee.Cmp(feeCap) > 0 {
		return feeCap
	}
	return withBaseFee
}

// GasCost is what buyGas debits before execution starts, gas and blob gas only. The value is not
// part of it because Sonic sets vm.Config.InsufficientBalanceIsNotAnError, so a transaction
// transferring more than its sender owns executes, reverts and pays for the gas it burned.
func GasCost(spec TxSpec, baseFee *big.Int) *big.Int {
	total := new(big.Int).Mul(EffectiveGasPrice(spec, baseFee), gas(spec.Gas()))
	return total.Add(total, BlobFee(spec))
}

// Affordability is what buyGas requires the sender to hold, which differs from what it charges: blob
// gas is charged at the blob base fee but checked against the transaction's own blob fee cap, so an
// enormous cap demands an enormous balance to pay the floor rate. The check is made in uint256, so a
// requirement wider than 256 bits overflows and reports insufficient funds outright.
func Affordability(spec TxSpec, baseFee *big.Int) *big.Int {
	total := new(big.Int).Mul(EffectiveGasPrice(spec, baseFee), gas(spec.Gas()))
	return total.Add(total, new(big.Int).Mul(gas(BlobGasUsed(spec)), spec.FeeCap()))
}

// DeclaredCost mirrors types.Transaction.Cost, the quantity ValidateTxStatic range-checks. What the
// sender must be able to pay is Affordability.
func DeclaredCost(spec TxSpec) *big.Int {
	total := new(big.Int).Mul(spec.FeeCap(), gas(SaturatingAdd(spec.Gas(), BlobGasUsed(spec))))
	return total.Add(total, spec.Amount())
}

// BlobFee is what a transaction pays for its blob gas, charged on top of the ordinary gas fee,
// priced at the blob base fee rather than the transaction's own cap, and absent from the receipt.
// The blob base fee is the protocol minimum, which holds because this chain never consumes blob gas.
func BlobFee(spec TxSpec) *big.Int {
	return new(big.Int).Mul(gas(BlobGasUsed(spec)), big.NewInt(params.BlobTxMinBlobGasprice))
}

// BlobGasUsed is the blob gas a transaction reserves, mirroring stateTransition.BlobGasUsed.
func BlobGasUsed(spec TxSpec) uint64 {
	return uint64(len(BlobHashesOf(spec))) * params.BlobTxBlobGasPerBlob
}

// LegacyIntrinsicGas is the pre-Berlin intrinsic gas schedule, which basiccheck applies to every
// transaction whatever its type. It cannot overflow, because the generator bounds the payload and
// the access list far below what would be needed.
func LegacyIntrinsicGas(spec TxSpec) uint64 {
	intrinsic := uint64(params.TxGas)
	if spec.IsCreate() {
		intrinsic = params.TxGasContractCreation
	}
	zero, nonZero := PayloadBytes(spec)
	perEntry := uint64(params.TxAccessListAddressGas + params.TxAccessListStorageKeyGas)

	return intrinsic +
		zero*params.TxDataZeroGas +
		nonZero*params.TxDataNonZeroGasEIP2028 +
		uint64(len(AccessListOf(spec)))*perEntry
}

// PayloadBytes counts the payload's zero and non-zero bytes, which are priced apart.
func PayloadBytes(spec TxSpec) (zero, nonZero uint64) {
	data := spec.Data()
	zeroCount := bytes.Count(data, []byte{0})
	return uint64(zeroCount), uint64(len(data) - zeroCount)
}

// RevisionIntrinsicGas is the intrinsic gas the state processor charges: the legacy schedule plus the
// two components the active revisions price and basiccheck's older one does not, being EIP-3860's
// per-word init code cost and a set-code transaction's per-authorization cost.
func RevisionIntrinsicGas(spec TxSpec) uint64 {
	intrinsic := LegacyIntrinsicGas(spec)
	if spec.IsCreate() {
		words := (uint64(len(spec.Data())) + 31) / 32
		intrinsic += words * params.InitCodeWordGas
	}
	return intrinsic + uint64(len(AuthsOf(spec)))*params.CallNewAccountGas
}

// FloorDataGas is the EIP-7623 lower bound on a transaction's gas, driven purely by the size and
// content of its payload.
func FloorDataGas(spec TxSpec) uint64 {
	zero, nonZero := PayloadBytes(spec)
	return params.TxGas + (zero+4*nonZero)*params.TxCostFloorPerToken
}

// gas lifts a gas amount into big.Int arithmetic.
func gas(amount uint64) *big.Int {
	return new(big.Int).SetUint64(amount)
}

// scale multiplies by numerator and divides by denominator, for building a band around a value that
// is only approximately known, such as a base fee that moves between being read and being applied.
func Scale(value *big.Int, numerator, denominator int64) *big.Int {
	out := new(big.Int).Mul(value, big.NewInt(numerator))
	return out.Div(out, big.NewInt(denominator))
}

// SaturatingAdd adds without wrapping.
func SaturatingAdd(a, b uint64) uint64 {
	return min(a, math.MaxUint64-b) + b
}
