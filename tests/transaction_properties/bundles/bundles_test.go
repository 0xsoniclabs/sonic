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
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/api/sonicapi"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests/transaction_properties/core"
	"github.com/0xsoniclabs/sonic/tests/transaction_properties/regular"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

func testConfig(upgrades opera.Upgrades) core.NetworkConfig {
	return core.NetworkConfig{
		ChainId:     big.NewInt(1),
		Upgrades:    upgrades,
		MaxEventGas: 10_000_000,
		MaxBlockGas: 20_000_000,
		MaxTxType:   core.MaxTxTypeFor(upgrades),
		Pricing:     regular.PricingRules,
	}
}

func testBatch() *rapid.Generator[[]core.TxSpec] {
	return regular.Domain{Cfg: core.GenConfig{
		Network:             testConfig(opera.GetBrioUpgrades()),
		MaxTxsPerBatch:      4,
		MaxAccountsPerBatch: 3,
		GasBudget:           1_000_000,
	}}.Batch()
}

// TestBundling_CarriesABatchInsideABatch uses rapid on the generator itself: whatever is drawn, the
// batch keeps its shape and the envelope holds something an envelope can hold.
func TestBundling_CarriesABatchInsideABatch(t *testing.T) {
	inner := testBatch()

	rapid.Check(t, func(rt *rapid.T) {
		specs := Bundling(inner).Draw(rt, "batch")
		require.NotEmpty(rt, specs)

		envelopes := envelopesIn(specs)
		require.LessOrEqual(rt, len(envelopes), 1,
			"two bundles in one batch would compete for one nonce")

		for _, envelope := range envelopes {
			require.NotEmpty(rt, envelope.Contents, "an envelope with nothing in it is not a bundle")
			require.LessOrEqual(rt, len(envelope.Contents), 3)
			if envelope.Root == RootSingle {
				require.Len(rt, envelope.Contents, 1, "a bare step carries one transaction")
			}

			// The capabilities an envelope answers for itself.
			require.False(rt, envelope.IsCreate())
			require.Equal(rt, core.ToOther, envelope.Recipient())
			require.Zero(rt, envelope.Amount().Sign())
			require.Equal(rt, uint8(types.LegacyTxType), envelope.TxType())
			require.Equal(rt, core.NonceCorrect, envelope.NonceChoice(),
				"an envelope's nonce is never spent, so it is not drawn")
			require.Positive(rt, envelope.FeeCap().Sign())

			// Nothing about it is known before Prepare has built the bundle.
			require.Nil(rt, envelope.Data())
			require.Zero(rt, envelope.Gas())
			_, err := envelope.TxData(0, core.BuildContext{})
			require.Error(rt, err, "an envelope with no bundle built cannot be assembled")
		}
	})
}

func TestBundling_TakesOverTheEnvelopeOfWhatItReplaces(t *testing.T) {
	inner := testBatch()

	rapid.Check(t, func(rt *rapid.T) {
		specs := Bundling(inner).Draw(rt, "batch")
		for _, envelope := range envelopesIn(specs) {
			require.Less(rt, envelope.Sender(), 3, "the carrier sends from the batch's own window")
		}
	})
}

func TestMarkable_GivesALegacyPayloadSomewhereToPutTheMarker(t *testing.T) {
	to := common.Address{0x42}
	legacy := &types.LegacyTx{
		Nonce: 7, GasPrice: big.NewInt(11), Gas: 21_000,
		To: &to, Value: big.NewInt(3), Data: []byte{1, 2},
	}

	marked, ok := markable(legacy).(*types.AccessListTx)
	require.True(t, ok, "a legacy payload has no access list, so it becomes one that has")
	require.Equal(t, legacy.Nonce, marked.Nonce)
	require.Equal(t, legacy.GasPrice, marked.GasPrice)
	require.Equal(t, legacy.Gas, marked.Gas)
	require.Equal(t, legacy.To, marked.To)
	require.Equal(t, legacy.Value, marked.Value)
	require.Equal(t, legacy.Data, marked.Data)

	// Everything else already has one and is left alone.
	dynamic := &types.DynamicFeeTx{Nonce: 1}
	require.Same(t, dynamic, markable(dynamic))
}

func TestPricingRules_JudgeTheCarrierByTheRegime(t *testing.T) {
	brio := opera.GetBrioUpgrades()
	bundlesOn := brio
	bundlesOn.TransactionBundles = true

	tests := map[string]struct {
		upgrades opera.Upgrades
		spec     core.TxSpec
		want     []core.Outcome
	}{
		"an envelope where bundles run is a carrier and nothing else": {
			upgrades: bundlesOn,
			spec:     &Envelope{},
			want:     core.Dropped,
		},
		"an envelope where bundles are disabled is skipped": {
			upgrades: brio,
			spec:     &Envelope{},
			want:     core.Dropped,
		},
		"an ordinary transaction is priced by the domain being wrapped": {
			upgrades: bundlesOn,
			spec:     &testSpec{},
			want:     nil,
		},
	}

	domain := &Domain{Inner: regular.Domain{}, cfg: testConfig(bundlesOn)}
	rules := domain.PricingRules(func(
		core.TxSpec, core.SenderState, *big.Int, core.NetworkConfig,
	) []core.Rule {
		return []core.Rule{{Applies: true, Reason: "the fallback was asked", Outcomes: core.Either}}
	})

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			cfg := testConfig(test.upgrades)
			got := rules(test.spec, core.SenderState{Balance: new(big.Int)}, big.NewInt(1), cfg)

			require.Len(t, got, 1)
			require.True(t, got[0].Applies)
			if test.want == nil {
				require.Equal(t, "the fallback was asked", got[0].Reason)
				return
			}
			require.Equal(t, test.want, got[0].Outcomes)
			require.NotEqual(t, "the fallback was asked", got[0].Reason)
		})
	}
}

func TestPricingRules_HandBackEverythingBeforeBrio(t *testing.T) {
	domain := &Domain{Inner: regular.Domain{}, cfg: testConfig(opera.GetAllegroUpgrades())}
	asked := false
	rules := domain.PricingRules(func(
		core.TxSpec, core.SenderState, *big.Int, core.NetworkConfig,
	) []core.Rule {
		asked = true
		return nil
	})

	rules(&Envelope{}, core.SenderState{}, big.NewInt(1), testConfig(opera.GetAllegroUpgrades()))
	require.True(t, asked, "before Brio an envelope is an ordinary call and is priced like one")
}

func TestCheckInfo_RequiresTheRecordToMatchTheBlock(t *testing.T) {
	domain := &Domain{cfg: testConfig(opera.GetBrioUpgrades())}
	envelope := &Envelope{Built: &Built{
		Inner: []*types.Transaction{
			types.NewTx(&types.LegacyTx{Nonce: 1}),
			types.NewTx(&types.LegacyTx{Nonce: 2}),
		},
	}}
	receipts := []*types.Receipt{
		{BlockNumber: big.NewInt(9), TransactionIndex: 3},
		{BlockNumber: big.NewInt(9), TransactionIndex: 4},
	}
	info := &sonicapi.RPCBundleInfo{
		Block: rpc.BlockNumber(9), Position: hexutil.Uint(3), Count: hexutil.Uint(2),
	}

	require.NoError(t, domain.checkInfo(envelope, info, receipts, 2, core.Observation{}))

	t.Run("a count that is not what reached the block", func(t *testing.T) {
		wrong := *info
		wrong.Count = 1
		require.ErrorContains(t,
			domain.checkInfo(envelope, &wrong, receipts, 2, core.Observation{}),
			"reports 1 of its transactions in the block, but 2 have a receipt")
	})

	t.Run("a position that is not where the transactions are", func(t *testing.T) {
		wrong := *info
		wrong.Position = 7
		require.ErrorContains(t,
			domain.checkInfo(envelope, &wrong, receipts, 2, core.Observation{}),
			"but its bundle info puts it at 7")
	})

	t.Run("a block that is not the one they are in", func(t *testing.T) {
		wrong := *info
		wrong.Block = rpc.BlockNumber(11)
		require.ErrorContains(t,
			domain.checkInfo(envelope, &wrong, receipts, 2, core.Observation{}),
			"but its bundle info names block 11")
	})

	t.Run("a bundle that gave up records nothing of its transactions", func(t *testing.T) {
		none := &sonicapi.RPCBundleInfo{Block: rpc.BlockNumber(9)}
		require.NoError(t, domain.checkInfo(envelope, none, []*types.Receipt{nil, nil}, 0,
			core.Observation{}))
	})
}

func TestEnvelope_RendersThePlanItCarries(t *testing.T) {
	plan := bundle.ExecutionPlan{
		Root: bundle.NewAllOfStep(
			bundle.NewTxStep(bundle.TxReference{}),
			bundle.NewTxStep(bundle.TxReference{Hash: common.Hash{1}}),
		),
		Range: bundle.MakeMaxRangeStartingAt(100),
	}
	envelope := &Envelope{
		Root:     RootAllOf,
		Contents: []core.TxSpec{&testSpec{}, &testSpec{}},
		Built: &Built{
			Plan:  plan,
			Plans: []core.TxPlan{{Nonce: 7}, {Nonce: 8}},
		},
	}

	rendered := envelope.String()
	require.Contains(t, rendered, "plan AllOf(A,B) over blocks "+plan.Range.String())
	require.Contains(t, rendered, "carrying A nonce=7")
	require.Contains(t, rendered, "carrying B nonce=8")
}

// testSpec is a transaction these tests shape directly, since the ordinary transaction types belong to
// the regular domain and are its own. It carries what the predicates below read.
type testSpec struct {
	core.Envelope
	core.Payload
	core.OptionalRecipient
	core.WideValue
	core.SinglePrice
}

func (testSpec) TxType() uint8 { return types.LegacyTxType }

func (s *testSpec) TxData(uint64, core.BuildContext) (types.TxData, error) {
	return nil, nil // never assembled: these tests are about the model, not the builder
}

func TestCannotFail_ClaimsOnlyThePlainCase(t *testing.T) {
	sender := core.SenderState{Balance: big.NewInt(1e18)}
	baseFee := big.NewInt(1)

	plain := &testSpec{
		Envelope:          core.Envelope{GasLimit: 21_000 + ampleGas},
		OptionalRecipient: core.OptionalRecipient{To: core.ToOther},
		WideValue:         core.WideValue{Value: big.NewInt(1)},
		SinglePrice:       core.SinglePrice{GasPrice: big.NewInt(1)},
	}
	require.True(t, cannotFail(plain, sender, baseFee))

	creation := *plain
	creation.OptionalRecipient = core.OptionalRecipient{To: core.ToCreate}
	require.False(t, cannotFail(&creation, sender, baseFee),
		"a creation runs its data as init code, which the model does not read")

	tight := *plain
	tight.Envelope.GasLimit = 21_000
	require.False(t, cannotFail(&tight, sender, baseFee),
		"a gas limit at the intrinsic cost can still run out further in")

	poor := *plain
	poor.WideValue = core.WideValue{Value: big.NewInt(1e18)}
	require.False(t, cannotFail(&poor, sender, baseFee),
		"a transfer the sender cannot back once it has bought its gas reverts")
}

func TestNearAGasBoundary_CatchesWhatTheMarkerCanMove(t *testing.T) {
	intrinsic := uint64(21_000)

	at := &testSpec{Envelope: core.Envelope{GasLimit: intrinsic}}
	require.False(t, nearAGasBoundary(at), "a limit that clears the boundary stays clear of it")

	just := &testSpec{Envelope: core.Envelope{GasLimit: intrinsic - 1}}
	require.True(t, nearAGasBoundary(just),
		"a limit one below it is one the marker's cost carries over")

	far := &testSpec{Envelope: core.Envelope{GasLimit: intrinsic - markerGas - 1}}
	require.False(t, nearAGasBoundary(far), "and one that far below stays below")
}

func TestBlocksBeyondReach_IsBeyondWhatAnIterationProduces(t *testing.T) {
	// The range choices are only meaningful if a range starting this far ahead cannot be reached by the
	// handful of blocks one iteration drives.
	require.Greater(t, uint64(blocksBeyondReach), uint64(core.ConfirmationBlocks)*10)
	require.Less(t, uint64(blocksBeyondReach), bundle.MaxBlockRangeLength)
}
