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

package sonic

import (
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/tests"
)

// newBundleChain is a fake core with five blocks served and the namespace
// over it.
func newBundleChain(t *testing.T) (*tests.FakeCore, *tests.Harness) {
	t.Helper()
	fake := tests.NewFakeCore(t)
	for range 5 {
		fake.AppendAndPublish(tests.BlockSpec{})
	}
	harness := tests.NewHarness(t, fake, func(dispatcher *dispatch.Dispatcher, environment *exec.Environment) {
		Register(dispatcher, environment)
	})
	harness.WaitForHead(5)
	return fake, harness
}

// buildSubmitBundleArguments is the reference test's helper: a valid bundle of
// the steps, each with the flags, for the block range.
func buildSubmitBundleArguments(t *testing.T, signer types.Signer, flags bundle.ExecutionFlags, first, length uint64, steps ...bundle.BuilderStep) SubmitBundleArguments {
	t.Helper()
	flagged := make([]bundle.BuilderStep, len(steps))
	for index, step := range steps {
		flagged[index] = step.WithFlags(flags)
	}
	transactionBundle, plan := bundle.NewBuilder().
		WithSigner(signer).
		SetEarliest(first).
		SetRangeLength(length).
		AllOf(flagged...).
		BuildBundleAndPlan()
	ordered := transactionBundle.GetTransactionsInReferencedOrder()
	signed := make([]hexutil.Bytes, len(ordered))
	for index, transaction := range ordered {
		encoded, err := transaction.MarshalBinary()
		require.NoError(t, err)
		signed[index] = encoded
	}
	planForm, err := NewRPCExecutionPlanComposable(plan)
	require.NoError(t, err)
	return SubmitBundleArguments{SignedTransactions: signed, ExecutionPlan: planForm}
}

func transferSteps(t *testing.T, count int, gas uint64) []bundle.BuilderStep {
	t.Helper()
	recipient := common.Address{2}
	steps := make([]bundle.BuilderStep, count)
	for index := range steps {
		key, err := crypto.GenerateKey()
		require.NoError(t, err)
		steps[index] = bundle.Step(key, &types.DynamicFeeTx{To: &recipient, Nonce: uint64(index), Gas: gas})
	}
	return steps
}

func TestSubmitBundle_SendsAnEnvelopeOfThePlan(t *testing.T) {
	cases := map[string]struct {
		flags bundle.ExecutionFlags
		steps int
	}{
		"single tx, default flags":   {bundle.EF_Default, 1},
		"three txs, default flags":   {bundle.EF_Default, 3},
		"two txs, TolerateFailed":    {bundle.EF_TolerateFailed, 2},
		"single tx, TolerateInvalid": {bundle.EF_TolerateInvalid, 1},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			fake, harness := newBundleChain(t)
			signer := harness.Environment.Signer()
			arguments := buildSubmitBundleArguments(t, signer, test.flags, 6, 100, transferSteps(t, test.steps, 200_000)...)
			hash := tests.Decode[common.Hash](t, harness.Result("sonic_submitBundle", arguments))

			sent := fake.Sent()
			require.Len(t, sent, 1)
			envelope := sent[0]
			require.True(t, bundle.IsEnvelope(envelope))
			_, plan, err := bundle.ValidateEnvelope(signer, envelope)
			require.NoError(t, err)
			require.Equal(t, plan.Hash(), hash)
			require.EqualValues(t, 6, plan.Range.First)
			require.EqualValues(t, 100, plan.Range.Length)

			opened, err := bundle.OpenEnvelope(signer, envelope)
			require.NoError(t, err)
			require.Len(t, opened.Transactions, test.steps)
			require.GreaterOrEqual(t, envelope.Gas(), uint64(test.steps)*200_000, "the envelope's gas covers the bundled transactions")
		})
	}
}

func TestSubmitBundle_RefusesWhatIsNoBundle(t *testing.T) {
	fake, harness := newBundleChain(t)
	signer := harness.Environment.Signer()
	valid := func() SubmitBundleArguments {
		return buildSubmitBundleArguments(t, signer, bundle.EF_Default, 6, 10, transferSteps(t, 2, params.TxGas)...)
	}

	empty := valid()
	empty.SignedTransactions = nil
	require.Equal(t, "signedTransactions must not be empty", harness.Failure("sonic_submitBundle", empty).Message)

	tooLong := valid()
	tooLong.ExecutionPlan.BlockRange.Length = hexutil.Uint64(bundle.MaxBlockRangeLength + 1)
	require.Contains(t, harness.Failure("sonic_submitBundle", tooLong).Message, "invalid block range")

	past := valid()
	past.ExecutionPlan.BlockRange = RPCRange{First: 0x1, Length: 0x2}
	require.Contains(t, harness.Failure("sonic_submitBundle", past).Message, "is not in the future relative to current block 5", "the served head is the current block")

	mismatch := valid()
	mismatch.SignedTransactions = mismatch.SignedTransactions[:1]
	require.Equal(t, "executionPlan steps count (2) must match signedTransactions count (1)", harness.Failure("sonic_submitBundle", mismatch).Message)

	undecodable := valid()
	undecodable.SignedTransactions[0] = hexutil.Bytes{0x01}
	require.Contains(t, harness.Failure("sonic_submitBundle", undecodable).Message, "failed to decode bundled transaction 0")

	require.Empty(t, fake.Sent(), "nothing reaches the pool")

	refused := errors.New("pool refused")
	fake.SetSendError(refused)
	require.Equal(t, "failed to submit bundle transaction: pool refused", harness.Failure("sonic_submitBundle", valid()).Message)
}

func TestGetBundleInfo_AnswersWhereABundleOfAServedBlockRan(t *testing.T) {
	fake, harness := newBundleChain(t)
	served, unserved, unknown := common.Hash{1}, common.Hash{2}, common.Hash{3}
	fake.SetBundle(served, core.BundleInfo{BlockNumber: 5, Offset: 10, Count: 11})
	fake.SetBundle(unserved, core.BundleInfo{BlockNumber: 6, Offset: 1, Count: 1})

	require.JSONEq(t, `{"block":"0x5","position":"0xa","count":"0xb"}`, string(harness.Result("sonic_getBundleInfo", served)))
	require.JSONEq(t, `null`, string(harness.Result("sonic_getBundleInfo", unknown)))
	require.JSONEq(t, `null`, string(harness.Result("sonic_getBundleInfo", unserved)), "a block above the head is not served yet")

	fake.SetMode("BundleInfo", tests.Fail)
	require.Equal(t, tests.ErrInjected.Error(), harness.Failure("sonic_getBundleInfo", served).Message)
}

func TestRPCBundleInfo_IsEthereumConformantJSON(t *testing.T) {
	expectJsonEqual(t, `{"block":"0x7b","position":"0xa","count":"0xb"}`, RPCBundleInfo{Block: 123, Position: 10, Count: 11})
	expectJsonEqual(t, `{}`, RPCBundleInfo{})
}
