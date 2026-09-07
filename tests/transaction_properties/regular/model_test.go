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
	"math"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/tests/transaction_properties/core"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

func TestPrediction_PermitsOnlyWhatItAllows(t *testing.T) {
	prediction := core.Allow("a reason", core.OutcomeDropped, core.OutcomeExecuted)

	require.True(t, prediction.Permits(core.OutcomeDropped))
	require.True(t, prediction.Permits(core.OutcomeExecuted))
	require.False(t, prediction.Permits(core.OutcomeEventRejected))
	require.Contains(t, prediction.String(), "Dropped or Executed")
	require.Contains(t, prediction.String(), "a reason")
}

func TestLegacyIntrinsicGas_ChargesTheScheduleOfEachComponent(t *testing.T) {
	perEntry := uint64(params.TxAccessListAddressGas + params.TxAccessListStorageKeyGas)

	tests := map[string]struct {
		spec core.TxSpec
		want uint64
	}{
		"plain call": {
			spec: &legacyTx{OptionalRecipient: core.OptionalRecipient{To: core.ToSelf}},
			want: params.TxGas,
		},
		"contract creation": {
			spec: &legacyTx{OptionalRecipient: core.OptionalRecipient{To: core.ToCreate}},
			want: params.TxGasContractCreation,
		},
		"zero payload bytes": {
			spec: &legacyTx{Payload: core.Payload{DataLen: 10}},
			want: params.TxGas + 10*params.TxDataZeroGas,
		},
		"non-zero payload bytes": {
			spec: &legacyTx{Payload: core.Payload{DataLen: 10, DataNonZero: true}},
			want: params.TxGas + 10*params.TxDataNonZeroGasEIP2028,
		},
		"access list entries": {
			spec: &accessListTx{AccessListEntries: core.AccessListEntries{AccessListLen: 3}},
			want: params.TxGas + 3*perEntry,
		},
		"a legacy transaction has no access list to charge for": {
			spec: &legacyTx{},
			want: params.TxGas,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, test.want, core.LegacyIntrinsicGas(test.spec))
		})
	}
}

func TestRevisionIntrinsicGas_AddsInitCodeWordsAndAuthorizations(t *testing.T) {
	tests := map[string]struct {
		spec core.TxSpec
		want uint64
	}{
		"a call is priced as the legacy schedule": {
			spec: &legacyTx{Payload: core.Payload{DataLen: 64}, OptionalRecipient: core.OptionalRecipient{To: core.ToSelf}},
			want: params.TxGas + 64*params.TxDataZeroGas,
		},
		"a creation pays per init code word, rounded up": {
			spec: &legacyTx{Payload: core.Payload{DataLen: 33}, OptionalRecipient: core.OptionalRecipient{To: core.ToCreate}},
			want: params.TxGasContractCreation + 33*params.TxDataZeroGas + 2*params.InitCodeWordGas,
		},
		"a set-code transaction pays per authorization": {
			spec: &setCodeTx{AuthList: core.AuthList{Entries: []core.AuthChoice{core.AuthSelf, core.AuthOther}}},
			want: params.TxGas + 2*params.CallNewAccountGas,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, test.want, core.RevisionIntrinsicGas(test.spec))
		})
	}
}

func TestFloorDataGas_PricesFourTokensPerNonZeroByte(t *testing.T) {
	tests := map[string]struct {
		spec core.TxSpec
		want uint64
	}{
		"an empty payload has no floor above the base cost": {
			spec: &legacyTx{},
			want: params.TxGas,
		},
		"a zero byte is one token": {
			spec: &legacyTx{Payload: core.Payload{DataLen: 5}},
			want: params.TxGas + 5*params.TxCostFloorPerToken,
		},
		"a non-zero byte is four tokens": {
			spec: &legacyTx{Payload: core.Payload{DataLen: 5, DataNonZero: true}},
			want: params.TxGas + 20*params.TxCostFloorPerToken,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, test.want, core.FloorDataGas(test.spec))
		})
	}
}

func TestPayloadBytes_CountsZeroAndNonZeroApart(t *testing.T) {
	zero, nonZero := core.PayloadBytes(&legacyTx{Payload: core.Payload{DataLen: 7}})
	require.Equal(t, uint64(7), zero)
	require.Equal(t, uint64(0), nonZero)

	zero, nonZero = core.PayloadBytes(&legacyTx{Payload: core.Payload{DataLen: 7, DataNonZero: true}})
	require.Equal(t, uint64(0), zero)
	require.Equal(t, uint64(7), nonZero)
}

func TestEffectiveGasPrice_IsTheLowerOfTheCapAndTheTipPlusBaseFee(t *testing.T) {
	baseFee := big.NewInt(100)

	tests := map[string]struct {
		spec core.TxSpec
		want int64
	}{
		"one price field ignores the base fee": {
			spec: &legacyTx{SinglePrice: core.SinglePrice{GasPrice: big.NewInt(7)}},
			want: 7,
		},
		"the tip plus the base fee, when that is lower": {
			spec: &dynamicFeeTx{WidePrices: core.WidePrices{
				GasFeeCap: big.NewInt(1000), GasTipCap: big.NewInt(5),
			}},
			want: 105,
		},
		"the fee cap, when the tip would exceed it": {
			spec: &dynamicFeeTx{WidePrices: core.WidePrices{
				GasFeeCap: big.NewInt(120), GasTipCap: big.NewInt(90),
			}},
			want: 120,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, big.NewInt(test.want), core.EffectiveGasPrice(test.spec, baseFee))
		})
	}
}

func TestGasCostAndAffordability_DifferOnBlobGas(t *testing.T) {
	spec := &blobTx{
		Envelope:     core.Envelope{GasLimit: 1000},
		NarrowPrices: core.NarrowPrices{GasFeeCap: big.NewInt(10), GasTipCap: big.NewInt(10)},
		BlobHashList: core.BlobHashList{BlobHashLen: 1},
	}
	baseFee := big.NewInt(0)
	blobGas := int64(params.BlobTxBlobGasPerBlob)

	// Charged at the blob base fee, which is the protocol minimum on this chain.
	require.Equal(t,
		big.NewInt(1000*10+blobGas*params.BlobTxMinBlobGasprice),
		core.GasCost(spec, baseFee),
	)

	// Required at the transaction's own cap, which is what makes an enormous cap unaffordable.
	require.Equal(t, big.NewInt(1000*10+blobGas*10), core.Affordability(spec, baseFee))
}

func TestDeclaredCost_IsTheGasAtTheCapPlusTheValue(t *testing.T) {
	spec := &dynamicFeeTx{
		Envelope:   core.Envelope{GasLimit: 21_000},
		WidePrices: core.WidePrices{GasFeeCap: big.NewInt(2), GasTipCap: big.NewInt(1)},
		WideValue:  core.WideValue{Value: big.NewInt(500)},
	}
	require.Equal(t, big.NewInt(21_000*2+500), core.DeclaredCost(spec))
}

func TestBlobGas_IsOnlyChargedForABlobTransaction(t *testing.T) {
	require.Zero(t, core.BlobGasUsed(&legacyTx{}))
	require.Zero(t, core.BlobFee(&legacyTx{}).Sign())

	spec := &blobTx{BlobHashList: core.BlobHashList{BlobHashLen: 2}}
	require.Equal(t, 2*uint64(params.BlobTxBlobGasPerBlob), core.BlobGasUsed(spec))
}

func TestSaturatingAdd_StopsAtTheMaximum(t *testing.T) {
	require.Equal(t, uint64(7), core.SaturatingAdd(3, 4))
	require.Equal(t, uint64(math.MaxUint64), core.SaturatingAdd(math.MaxUint64, 1))
	require.Equal(t, uint64(math.MaxUint64), core.SaturatingAdd(math.MaxUint64-1, 5))
}

func TestScale_BuildsABandAroundAValue(t *testing.T) {
	require.Equal(t, big.NewInt(400), core.Scale(big.NewInt(100), 4, 1))
	require.Equal(t, big.NewInt(25), core.Scale(big.NewInt(100), 1, 4))
}

func TestAssignNonces_ResolvesEachChoiceAgainstItsSender(t *testing.T) {
	senders := []core.SenderState{{Nonce: 5, Balance: big.NewInt(0)}}
	specs := []core.TxSpec{
		&legacyTx{Envelope: core.Envelope{Nonce: core.NonceCorrect}},
		&legacyTx{Envelope: core.Envelope{Nonce: core.NonceCorrect}},
		&legacyTx{Envelope: core.Envelope{Nonce: core.NonceTooLow}},
		&legacyTx{Envelope: core.Envelope{Nonce: core.NonceGap, NonceGapSize: 3}},
		&legacyTx{Envelope: core.Envelope{Nonce: core.NonceMax}},
	}

	// Successive correct nonces form a run; the gap counts from the end of that run.
	require.Equal(t,
		[]uint64{5, 6, 4, 10, math.MaxUint64},
		core.AssignNonces(specs, senders),
	)
}

func TestAssignNonces_TooLowOnAFreshAccountIsTheCorrectNonce(t *testing.T) {
	senders := []core.SenderState{{Nonce: 0, Balance: big.NewInt(0)}}
	specs := []core.TxSpec{&legacyTx{Envelope: core.Envelope{Nonce: core.NonceTooLow}}}

	// There is no nonce below zero, so this is a known limit of a run on fresh accounts rather than a
	// too-low nonce being tested.
	require.Equal(t, []uint64{0}, core.AssignNonces(specs, senders))
}

func TestAssignNonces_KeepsSendersApart(t *testing.T) {
	senders := []core.SenderState{
		{Nonce: 1, Balance: big.NewInt(0)},
		{Nonce: 9, Balance: big.NewInt(0)},
	}
	specs := []core.TxSpec{
		&legacyTx{Envelope: core.Envelope{SenderIdx: 0, Nonce: core.NonceCorrect}},
		&legacyTx{Envelope: core.Envelope{SenderIdx: 1, Nonce: core.NonceCorrect}},
		&legacyTx{Envelope: core.Envelope{SenderIdx: 0, Nonce: core.NonceCorrect}},
	}

	require.Equal(t, []uint64{1, 9, 2}, core.AssignNonces(specs, senders))
}

// affordableSpec is a transaction the model expects to execute, so that a test can vary one field
// without every rule firing at once.
func affordableSpec(sender int, nonce core.NonceChoice) *legacyTx {
	return &legacyTx{
		Envelope:          core.Envelope{SenderIdx: sender, Nonce: nonce, GasLimit: 100_000},
		OptionalRecipient: core.OptionalRecipient{To: core.ToSelf},
		WideValue:         core.WideValue{Value: big.NewInt(0)},
		SinglePrice:       core.SinglePrice{GasPrice: new(big.Int).Set(core.MinimumViableFeeCap)},
	}
}

func testNetwork(upgrades opera.Upgrades) core.NetworkConfig {
	return core.NetworkConfig{
		ChainId:     big.NewInt(1),
		Upgrades:    upgrades,
		MaxEventGas: 10_000_000,
		MaxBlockGas: 20_000_000,
		MaxTxType:   core.MaxTxTypeFor(upgrades),
		Pricing:     PricingRules,
	}
}

func TestPlanBatch_RefusesTheWholeBatchForOneUnsupportedType(t *testing.T) {
	senders := []core.SenderState{{Nonce: 0, Balance: new(big.Int).Set(core.AccountBalance)}}
	specs := []core.TxSpec{
		affordableSpec(0, core.NonceCorrect),
		&blobTx{}, // above the maximum type before Sonic's successor forks
	}

	plans, rejected, reason := core.PlanBatch(specs, senders, big.NewInt(1), core.NetworkConfig{
		ChainId:     big.NewInt(1),
		MaxEventGas: 10_000_000,
		MaxBlockGas: 20_000_000,
		MaxTxType:   types.DynamicFeeTxType,
	})

	require.True(t, rejected)
	require.Contains(t, reason, "above the maximum")
	for _, plan := range plans {
		require.True(t, plan.Prediction.Permits(core.OutcomeEventRejected),
			"every transaction of a refused batch must expect the refusal")
		require.False(t, plan.Prediction.Permits(core.OutcomeExecuted))
	}
}

func TestPlanBatch_RefusesTheWholeBatchOverTheEventGasAllowance(t *testing.T) {
	senders := []core.SenderState{{Nonce: 0, Balance: new(big.Int).Set(core.AccountBalance)}}
	cfg := testNetwork(opera.GetAllegroUpgrades())

	spec := affordableSpec(0, core.NonceCorrect)
	spec.GasLimit = cfg.MaxEventGas

	_, rejected, reason := core.PlanBatch([]core.TxSpec{spec}, senders, big.NewInt(1), cfg)

	require.True(t, rejected)
	require.Contains(t, reason, "MaxEventGas")
}

func TestPlanBatch_AcceptsAnAffordableBatch(t *testing.T) {
	senders := []core.SenderState{{Nonce: 0, Balance: new(big.Int).Set(core.AccountBalance)}}

	plans, rejected, reason := core.PlanBatch(
		[]core.TxSpec{affordableSpec(0, core.NonceCorrect)},
		senders, big.NewInt(1), testNetwork(opera.GetAllegroUpgrades()),
	)

	require.False(t, rejected)
	require.Empty(t, reason)
	require.True(t, plans[0].Prediction.Permits(core.OutcomeExecuted), plans[0].Prediction.String())
	require.False(t, plans[0].Prediction.Permits(core.OutcomeDropped), plans[0].Prediction.String())
}

func TestResolveNonceOrder_AdmitsARunGivenOutOfOrder(t *testing.T) {
	senders := []core.SenderState{{Nonce: 0, Balance: new(big.Int).Set(core.AccountBalance)}}
	specs := []core.TxSpec{
		affordableSpec(0, core.NonceCorrect),
		affordableSpec(0, core.NonceCorrect),
		affordableSpec(0, core.NonceCorrect),
	}

	plans, rejected, _ := core.PlanBatch(specs, senders, big.NewInt(1), testNetwork(opera.GetAllegroUpgrades()))
	require.False(t, rejected)

	// The scrambler orders a sender's transactions by nonce, so a whole run executes whatever order it
	// was injected in.
	require.Equal(t, []uint64{0, 1, 2}, []uint64{plans[0].Nonce, plans[1].Nonce, plans[2].Nonce})
	for i, plan := range plans {
		require.True(t, plan.Prediction.Permits(core.OutcomeExecuted), "transaction %d: %s", i, plan.Prediction)
	}
}

func TestResolveNonceOrder_LeavesRivalsForOneNonceUndecided(t *testing.T) {
	senders := []core.SenderState{{Nonce: 0, Balance: new(big.Int).Set(core.AccountBalance)}}
	specs := []core.TxSpec{
		affordableSpec(0, core.NonceCorrect),
		affordableSpec(0, core.NonceTooLow), // resolves to 0 as well, so the two compete
	}

	plans, rejected, _ := core.PlanBatch(specs, senders, big.NewInt(1), testNetwork(opera.GetAllegroUpgrades()))
	require.False(t, rejected)
	require.Equal(t, plans[0].Nonce, plans[1].Nonce)

	for i, plan := range plans {
		require.True(t, plan.Prediction.Permits(core.OutcomeExecuted), "transaction %d", i)
		require.True(t, plan.Prediction.Permits(core.OutcomeDropped), "transaction %d", i)
		require.Contains(t, plan.Prediction.Reason, "competes for a nonce")
	}
}

// TestResolveNonceOrder_WithPrioritiesARivalNonceUnsettlesTheRestOfTheSequence is what transaction
// priorities do to a run behind a contested nonce, and what a run of the priorities property found:
// the ordering step picks between the rivals for a nonce before anything executes, and hoists what
// comes after behind the one it picked -- past the rival that ends up spending the nonce.
func TestResolveNonceOrder_WithPrioritiesARivalNonceUnsettlesTheRestOfTheSequence(t *testing.T) {
	batch := func() []core.TxSpec {
		return []core.TxSpec{
			affordableSpec(0, core.NonceCorrect),
			affordableSpec(0, core.NonceTooLow), // resolves to 0 as well, so the two compete
			affordableSpec(0, core.NonceCorrect),
		}
	}
	senders := []core.SenderState{{Nonce: 0, Balance: new(big.Int).Set(core.AccountBalance)}}

	ordinary, rejected, _ := core.PlanBatch(
		batch(), senders, big.NewInt(1), testNetwork(opera.GetAllegroUpgrades()))
	require.False(t, rejected)
	require.Equal(t, []uint64{0, 0, 1},
		[]uint64{ordinary[0].Nonce, ordinary[1].Nonce, ordinary[2].Nonce})
	require.False(t, ordinary[2].Prediction.Permits(core.OutcomeDropped),
		"the scrambler leaves a sender's run in nonce order, so whichever rival spends nonce 0, "+
			"the transaction behind it executes: %s", ordinary[2].Prediction)

	upgrades := opera.GetAllegroUpgrades()
	upgrades.TransactionPriorities = true
	hoistable, rejected, _ := core.PlanBatch(batch(), senders, big.NewInt(1), testNetwork(upgrades))
	require.False(t, rejected)
	require.True(t, hoistable[2].Prediction.Permits(core.OutcomeExecuted))
	require.True(t, hoistable[2].Prediction.Permits(core.OutcomeDropped),
		"nonce 1 may be hoisted past the rival that spends nonce 0: %s", hoistable[2].Prediction)
}

func TestResolveNonceOrder_RejectsATransactionBehindAGap(t *testing.T) {
	senders := []core.SenderState{{Nonce: 0, Balance: new(big.Int).Set(core.AccountBalance)}}
	specs := []core.TxSpec{affordableSpec(0, core.NonceGap)}
	specs[0].(*legacyTx).NonceGapSize = 2

	plans, rejected, _ := core.PlanBatch(specs, senders, big.NewInt(1), testNetwork(opera.GetAllegroUpgrades()))
	require.False(t, rejected)

	require.False(t, plans[0].Prediction.Permits(core.OutcomeExecuted))
	require.Contains(t, plans[0].Prediction.Reason, "is not the 0 the sender's sequence has reached")
}

func TestPredictTx_AppliesTheFirstRuleThatFits(t *testing.T) {
	sender := core.SenderState{Nonce: 0, Balance: new(big.Int).Set(core.AccountBalance)}
	// A base fee large enough that the bands the model builds around it do not round to zero.
	baseFee := big.NewInt(1000)
	allegro := testNetwork(opera.GetAllegroUpgrades())

	tests := map[string]struct {
		spec     core.TxSpec
		cfg      core.NetworkConfig
		Nonce    uint64
		Outcomes []core.Outcome
		Reason   string
	}{
		"a signature that is not the sender's": {
			spec: func() core.TxSpec {
				spec := affordableSpec(0, core.NonceCorrect)
				spec.Signing = core.SignZeroR
				return spec
			}(),
			cfg: allegro, Outcomes: core.Dropped, Reason: "signature is not the sender's own",
		},
		"gas below the legacy intrinsic cost": {
			spec: func() core.TxSpec {
				spec := affordableSpec(0, core.NonceCorrect)
				spec.GasLimit = params.TxGas - 1
				return spec
			}(),
			cfg: allegro, Outcomes: core.Dropped, Reason: "legacy intrinsic gas",
		},
		"a tip above the fee cap": {
			spec: &dynamicFeeTx{
				Envelope:   core.Envelope{GasLimit: 100_000},
				WidePrices: core.WidePrices{GasFeeCap: big.NewInt(10), GasTipCap: big.NewInt(11)},
				WideValue:  core.WideValue{Value: big.NewInt(0)},
			},
			cfg: allegro, Outcomes: core.Dropped, Reason: "tip cap exceeds gas fee cap",
		},
		"blob hashes from Allegro onwards": {
			spec: &blobTx{
				Envelope:     core.Envelope{GasLimit: 100_000},
				NarrowPrices: core.NarrowPrices{GasFeeCap: new(big.Int).Set(core.MinimumViableFeeCap), GasTipCap: big.NewInt(0)},
				NarrowValue:  core.NarrowValue{Value: big.NewInt(0)},
				BlobHashList: core.BlobHashList{BlobHashLen: 1},
			},
			cfg: allegro, Outcomes: core.Dropped, Reason: "blob hashes are not permissible",
		},
		"an empty authorization list": {
			spec: &setCodeTx{
				Envelope:     core.Envelope{GasLimit: 100_000},
				NarrowPrices: core.NarrowPrices{GasFeeCap: new(big.Int).Set(core.MinimumViableFeeCap), GasTipCap: big.NewInt(0)},
				NarrowValue:  core.NarrowValue{Value: big.NewInt(0)},
			},
			cfg: allegro, Outcomes: core.Dropped, Reason: "non-empty authorization list",
		},
		"a value wider than 256 bits": {
			spec: func() core.TxSpec {
				spec := affordableSpec(0, core.NonceCorrect)
				spec.Value = new(big.Int).Lsh(big.NewInt(1), 257)
				return spec
			}(),
			cfg: allegro, Outcomes: core.Dropped, Reason: "value does not fit in 256 bits",
		},
		"the maximum nonce": {
			spec: affordableSpec(0, core.NonceMax), cfg: allegro, Nonce: math.MaxUint64,
			Outcomes: core.Dropped, Reason: "maximum uint64",
		},
		"gas above the block limit": {
			spec: func() core.TxSpec {
				spec := affordableSpec(0, core.NonceCorrect)
				spec.GasLimit = allegro.MaxBlockGas + 1
				return spec
			}(),
			cfg: allegro, Outcomes: core.Dropped, Reason: "exceeds the block gas limit",
		},
		"a fee cap below the base fee": {
			spec: func() core.TxSpec {
				spec := affordableSpec(0, core.NonceCorrect)
				spec.GasPrice = big.NewInt(0)
				return spec
			}(),
			cfg: allegro, Outcomes: core.Dropped, Reason: "below the block's base fee",
		},
		"a balance that cannot back the gas": {
			spec: func() core.TxSpec {
				spec := affordableSpec(0, core.NonceCorrect)
				spec.GasLimit = 10_000_000
				spec.GasPrice = new(big.Int).Mul(core.MinimumViableFeeCap, big.NewInt(1e9))
				return spec
			}(),
			cfg: allegro, Outcomes: core.Dropped, Reason: "cannot back the gas",
		},
		"everything the model knows about is satisfied": {
			spec: affordableSpec(0, core.NonceCorrect), cfg: allegro,
			Outcomes: []core.Outcome{core.OutcomeExecuted}, Reason: "all checks the model knows about",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			prediction := core.PredictTx(test.spec, test.Nonce, sender, baseFee, test.cfg)

			require.Equal(t, test.Outcomes, prediction.Allowed, prediction.String())
			require.Contains(t, prediction.Reason, test.Reason)
		})
	}
}

func TestPredictTx_GatesRulesOnTheFork(t *testing.T) {
	sender := core.SenderState{Nonce: 0, Balance: new(big.Int).Set(core.AccountBalance)}

	spec := &blobTx{
		Envelope:     core.Envelope{GasLimit: 100_000},
		NarrowPrices: core.NarrowPrices{GasFeeCap: new(big.Int).Set(core.MinimumViableFeeCap), GasTipCap: big.NewInt(0)},
		NarrowValue:  core.NarrowValue{Value: big.NewInt(0)},
		BlobHashList: core.BlobHashList{BlobHashLen: 1},
	}

	// Only from Allegro does the block formation filter refuse a blob transaction carrying hashes.
	onSonic := core.PredictTx(spec, 0, sender, big.NewInt(1), testNetwork(opera.GetSonicUpgrades()))
	require.True(t, onSonic.Permits(core.OutcomeExecuted), onSonic.String())

	onAllegro := core.PredictTx(spec, 0, sender, big.NewInt(1), testNetwork(opera.GetAllegroUpgrades()))
	require.False(t, onAllegro.Permits(core.OutcomeExecuted), onAllegro.String())
}

func TestEventLevelFailure_ObjectsOnlyToTheTransactionType(t *testing.T) {
	cfg := core.NetworkConfig{MaxTxType: types.DynamicFeeTxType}

	require.Empty(t, core.EventLevelFailure(&legacyTx{}, cfg))
	require.Empty(t, core.EventLevelFailure(&dynamicFeeTx{}, cfg))
	require.NotEmpty(t, core.EventLevelFailure(&blobTx{}, cfg))

	// An absurd gas limit is not refused here: a self-emitted event skips basiccheck, so it is dealt
	// with further down.
	require.Empty(t, core.EventLevelFailure(&legacyTx{Envelope: core.Envelope{GasLimit: math.MaxUint64}}, cfg))
}

// TestPlanBatchInGivenOrder_AdmitsNoRunGivenOutOfOrder is the difference a bundle makes: an injected
// batch is put in nonce order by the scrambler, while a bundle's steps execute in the order its plan
// references them, so the same two transactions execute in one and fail in the other.
func TestPlanBatchInGivenOrder_AdmitsNoRunGivenOutOfOrder(t *testing.T) {
	senders := []core.SenderState{{Nonce: 1, Balance: new(big.Int).Set(core.AccountBalance)}}
	batch := func() []core.TxSpec {
		ahead := affordableSpec(0, core.NonceGap)
		ahead.NonceGapSize = 1 // nonce 2, one beyond the sender's next
		return []core.TxSpec{ahead, affordableSpec(0, core.NonceCorrect)}
	}
	cfg := testNetwork(opera.GetAllegroUpgrades())

	reordered, rejected, _ := core.PlanBatch(batch(), senders, big.NewInt(1), cfg)
	require.False(t, rejected)
	require.Equal(t, []uint64{2, 1}, []uint64{reordered[0].Nonce, reordered[1].Nonce})
	for i, plan := range reordered {
		require.True(t, plan.Prediction.Permits(core.OutcomeExecuted),
			"transaction %d is admitted by the sequence once reordered: %s", i, plan.Prediction)
	}

	asWritten, rejected, _ := core.PlanBatchInGivenOrder(batch(), senders, big.NewInt(1), cfg)
	require.False(t, rejected)
	require.False(t, asWritten[0].Prediction.Permits(core.OutcomeExecuted),
		"the first one is ahead of the sender's sequence and nothing will reorder it: %s",
		asWritten[0].Prediction)
	require.True(t, asWritten[1].Prediction.Permits(core.OutcomeExecuted),
		"%s", asWritten[1].Prediction)
}
