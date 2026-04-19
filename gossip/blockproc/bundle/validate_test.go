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

package bundle

import (
	"fmt"
	"maps"
	"math"
	"math/big"
	"slices"
	"strings"
	"testing"

	"github.com/0xsoniclabs/sonic/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestValidateEnvelope_ValidEnvelopes_AreAcceptedAndReturnsBundleAndPlan(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	tests := map[string]*builder{
		"empty": NewBuilder(),
		"single transaction": NewBuilder().AllOf(
			Step(key, types.AccessListTx{}),
		),
		"multiple transactions": NewBuilder().AllOf(
			Step(key, types.AccessListTx{Nonce: 1}),
			Step(key, types.AccessListTx{Nonce: 2}),
		),
		"composed bundles": NewBuilder().OneOf(
			AllOf(
				Step(key, types.AccessListTx{Nonce: 1}),
				Step(key, types.AccessListTx{Nonce: 2}),
			),
			AllOf(
				Step(key, types.AccessListTx{Nonce: 3}),
				Step(key, types.AccessListTx{Nonce: 4}),
			),
		),
		"nested bundles": NewBuilder().OneOf(
			Step(key, AllOf(
				Step(key, types.AccessListTx{Nonce: 1}),
				Step(key, types.AccessListTx{Nonce: 2}),
			).Build()),
			Step(key, AllOf(
				Step(key, types.AccessListTx{Nonce: 3}),
				Step(key, types.AccessListTx{Nonce: 4}),
			).Build()),
		),
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			signer := types.LatestSignerForChainID(big.NewInt(1))
			envelope := tc.WithSigner(signer).Build()

			bundle, plan, err := ValidateEnvelope(signer, envelope)
			require.NoError(err)

			want, err := OpenEnvelope(signer, envelope)
			require.NoError(err)

			require.Equal(want.Plan, bundle.Plan)
			require.Equal(want.Plan, *plan)

			// transactions can not be compared directly, so we compare their
			// hashes to make sure the same transactions are present
			require.Equal(len(want.Transactions), len(bundle.Transactions))
			for ref, tx := range want.Transactions {
				bundleTx, found := bundle.Transactions[ref]
				require.True(found)
				require.Equal(tx.Hash(), bundleTx.Hash())
			}
		})
	}
}

func TestValidateEnvelope_RegularTransaction_RejectedAsNotBeingAnEnvelope(t *testing.T) {
	regularTx := types.NewTx(&types.LegacyTx{
		To:   &common.Address{0x42},
		Data: []byte("this is a regular transaction, not an envelope"),
	})

	bundle, plan, err := ValidateEnvelope(nil, regularTx)
	require.ErrorContains(t, err, "not an envelope transaction")
	require.Nil(t, bundle, "expected no bundle to be returned")
	require.Nil(t, plan, "expected no execution plan to be returned")
}

func TestValidateEnvelope_NilTransaction_RejectedAsNotBeingAnEnvelope(t *testing.T) {
	bundle, plan, err := ValidateEnvelope(nil, nil)
	require.ErrorContains(t, err, "not an envelope transaction")
	require.Nil(t, bundle, "expected no bundle to be returned")
	require.Nil(t, plan, "expected no execution plan to be returned")
}

func TestValidateEnvelope_SignerNil_ReturnsError(t *testing.T) {
	envelope := AllOf().Build()
	_, _, err := ValidateEnvelope(nil, envelope)
	require.ErrorContains(t, err, "signer is nil")
}

func TestValidateEnvelope_InvalidEncoding_ReturnsError(t *testing.T) {
	signer := types.LatestSignerForChainID(big.NewInt(1))
	envelope := types.NewTx(&types.LegacyTx{
		To:   &BundleProcessor,
		Data: []byte("this is not a valid bundle encoding"),
	})

	_, _, err := ValidateEnvelope(signer, envelope)
	require.ErrorContains(t, err, "invalid bundle encoding")
}

func TestValidateEnvelope_InvalidBundle_IsRejected(t *testing.T) {
	// There are lots of reasons why a bundle can be invalid, we are just
	// checking a few of them to make sure that validateBundle is called.
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	signer := types.LatestSignerForChainID(big.NewInt(1))

	tests := map[string]TransactionBundle{
		"invalid block range": NewBuilder().
			SetEarliest(20).
			SetLatest(10).
			BuildBundle(),
		"missing transactions": func() TransactionBundle {
			bundle := NewBuilder().AllOf(
				Step(key, &types.AccessListTx{Nonce: 1}),
				Step(key, &types.AccessListTx{Nonce: 2}),
			).BuildBundle()
			bundle.Transactions = map[TxReference]*types.Transaction{}
			return bundle
		}(),
	}

	for name, bundle := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			issue := validateBundle(signer, bundle)
			require.Error(issue)

			encoded, err := bundle.Encode()
			require.NoError(err)

			envelope := types.NewTx(&types.AccessListTx{
				To:   &BundleProcessor,
				Data: encoded,
			})

			_, _, err = ValidateEnvelope(signer, envelope)
			require.ErrorContains(err, "invalid bundle")
			require.ErrorContains(err, issue.Error())
		})
	}
}

func TestValidateEnvelope_WrongChainId_IsRejected(t *testing.T) {
	require := require.New(t)
	key, err := crypto.GenerateKey()
	require.NoError(err)

	signer := types.LatestSignerForChainID(big.NewInt(1))

	valid := NewBuilder().AllOf(
		Step(key, &types.AccessListTx{}),
	).WithSigner(signer).Build()

	_, _, err = ValidateEnvelope(signer, valid)
	require.NoError(err)

	wrongChainIdSigner := types.LatestSignerForChainID(big.NewInt(123))
	repacked := types.MustSignNewTx(key, wrongChainIdSigner, &types.AccessListTx{
		To:   valid.To(),
		Data: valid.Data(),
	})

	_, _, err = ValidateEnvelope(signer, repacked)
	require.ErrorContains(err, "envelope signed for wrong chain ID")
}

func TestValidateEnvelope_InvalidEnvelopeGasLimit_IsRejected(t *testing.T) {
	for _, delta := range []int{-10, -1, 1, 10} {
		t.Run(fmt.Sprintf("delta=%d", delta), func(t *testing.T) {
			require := require.New(t)
			signer := types.LatestSignerForChainID(big.NewInt(1))

			validEnvelope := NewBuilder().AllOf().WithSigner(signer).Build()
			_, _, err := ValidateEnvelope(signer, validEnvelope)
			require.NoError(err)

			key, err := crypto.GenerateKey()
			require.NoError(err)

			invalidGasEnvelope := types.MustSignNewTx(key, signer,
				&types.AccessListTx{
					To:   validEnvelope.To(),
					Data: validEnvelope.Data(),
					Gas:  uint64(int(validEnvelope.Gas()) + delta),
				},
			)

			_, _, err = ValidateEnvelope(signer, invalidGasEnvelope)
			require.ErrorContains(err, "invalid gas limit")
		})
	}
}

func TestValidateEnvelope_OverflowInGasLimit_IsDetected(t *testing.T) {
	signer := types.LatestSignerForChainID(big.NewInt(1))
	require := require.New(t)
	key, err := crypto.GenerateKey()
	require.NoError(err)

	bundle := NewBuilder().AllOf(
		Step(key, &types.AccessListTx{Gas: math.MaxUint64 / 2}),
		Step(key, &types.AccessListTx{Gas: math.MaxUint64 / 2}),
		Step(key, &types.AccessListTx{Gas: math.MaxUint64 / 2}),
	).WithSigner(signer).BuildBundle()

	encoded, err := bundle.Encode()
	require.NoError(err)

	envelope := types.MustSignNewTx(key, signer, &types.AccessListTx{
		To:   &BundleProcessor,
		Data: encoded,
	})

	_, _, err = ValidateEnvelope(signer, envelope)
	require.ErrorContains(err, "invalid gas limit: failed transaction gas sum calculation")
}

func TestValidateEnvelope_NestedInvalidEnvelope_IsRejected(t *testing.T) {
	signer := types.LatestSignerForChainID(big.NewInt(1))
	require := require.New(t)
	key, err := crypto.GenerateKey()
	require.NoError(err)

	invalidInner := NewBuilder().
		SetEarliest(20).
		SetLatest(10).
		WithSigner(signer).
		Build()

	outer := NewBuilder().AllOf(
		Step(key, invalidInner),
	).WithSigner(signer).Build()

	_, _, innerErr := ValidateEnvelope(signer, invalidInner)
	require.Error(innerErr)

	_, _, outerErr := ValidateEnvelope(signer, outer)
	require.ErrorContains(outerErr, "invalid nested envelope")
	require.ErrorContains(outerErr, innerErr.Error())
}

func TestValidateEnvelope_DetectsExcessiveNesting(t *testing.T) {
	require := require.New(t)
	key, err := crypto.GenerateKey()
	require.NoError(err)

	require.Equal(2, MaxBundleNestingDepth)
	signer := types.LatestSignerForChainID(big.NewInt(1))

	// Number of nested bundles: 0
	cur := NewBuilder().AllOf(
		Step(key, &types.AccessListTx{}),
	).WithSigner(signer).Build()

	_, _, err = ValidateEnvelope(signer, cur)
	require.NoError(err)

	// Number of nested bundles: 1
	cur = NewBuilder().AllOf(
		Step(key, cur),
	).WithSigner(signer).Build()

	_, _, err = ValidateEnvelope(signer, cur)
	require.NoError(err)

	// Number of nested bundles: 2
	cur = NewBuilder().AllOf(
		Step(key, cur),
	).WithSigner(signer).Build()

	_, _, err = ValidateEnvelope(signer, cur)
	require.NoError(err)

	// Number of nested bundles: 3
	cur = NewBuilder().AllOf(
		Step(key, cur),
	).WithSigner(signer).Build()

	_, _, err = ValidateEnvelope(signer, cur)
	require.ErrorContains(err, "exceeds maximum nesting depth of bundles")

	// Number of nested bundles: 4
	cur = NewBuilder().AllOf(
		Step(key, cur),
	).WithSigner(signer).Build()

	_, _, err = ValidateEnvelope(signer, cur)
	require.ErrorContains(err, "exceeds maximum nesting depth of bundles")

}

func TestValidateBundle_ValidBundles_AreAccepted(t *testing.T) {
	key1, err := crypto.GenerateKey()
	require.NoError(t, err)

	key2, err := crypto.GenerateKey()
	require.NoError(t, err)

	validBundles := []*builder{
		NewBuilder().AllOf(),
		NewBuilder().OneOf(),
		NewBuilder().AllOf(
			Step(key1, &types.AccessListTx{}),
			Step(key2, &types.AccessListTx{}),
		),
		NewBuilder().OneOf(
			AllOf(
				Step(key1, &types.AccessListTx{}),
				Step(key2, &types.AccessListTx{}),
			),
			AllOf(
				Step(key2, &types.AccessListTx{}),
				Step(key1, &types.AccessListTx{}),
			),
		),
	}

	signer := types.LatestSignerForChainID(big.NewInt(1))
	for _, builder := range validBundles {
		bundle := builder.WithSigner(signer).BuildBundle()
		require.NoError(t, validateBundle(signer, bundle))
	}
}

func TestValidateBundle_InvalidPlan_Rejected(t *testing.T) {
	bundle := TransactionBundle{}

	issue := validatePlan(bundle.Plan)
	require.Error(t, issue)

	got := validateBundle(nil, bundle)
	require.ErrorContains(t, got, "invalid execution plan")
	require.ErrorContains(t, got, issue.Error())
}

func TestValidateBundle_NilTransaction_Rejected(t *testing.T) {
	tests := map[string][]*types.Transaction{
		"single nil transaction": {nil},
		"nil and non-nil transactions": {
			types.NewTx(&types.AccessListTx{}),
			nil,
			types.NewTx(&types.AccessListTx{}),
		},
	}

	for name, transactions := range tests {
		t.Run(name, func(t *testing.T) {
			validPlan := ExecutionPlan{
				Range: BlockRange{Earliest: 10, Latest: 20},
				Root:  NewTxStep(TxReference{}),
			}
			require.NoError(t, validatePlan(validPlan))

			index := map[TxReference]*types.Transaction{}
			for i, tx := range transactions {
				index[TxReference{From: common.Address{byte(i + 1)}}] = tx
			}

			bundle := TransactionBundle{
				Plan:         validPlan,
				Transactions: index,
			}

			require.ErrorContains(t, validateBundle(nil, bundle),
				"invalid nil transaction in bundle",
			)
		})
	}
}

func TestValidateBundle_InconsistentChainIds_Rejected(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	signer12 := types.LatestSignerForChainID(big.NewInt(12))
	signer14 := types.LatestSignerForChainID(big.NewInt(14))
	signer16 := types.LatestSignerForChainID(big.NewInt(16))

	tests := map[string][]*types.Transaction{
		"two different chain IDs": {
			types.MustSignNewTx(key, signer12, &types.AccessListTx{Nonce: 1}),
			types.MustSignNewTx(key, signer14, &types.AccessListTx{Nonce: 2}),
		},
		"multiple different chain IDs": {
			types.MustSignNewTx(key, signer12, &types.AccessListTx{Nonce: 1}),
			types.MustSignNewTx(key, signer14, &types.AccessListTx{Nonce: 2}),
			types.MustSignNewTx(key, signer16, &types.AccessListTx{Nonce: 3}),
		},
		"first different": {
			types.MustSignNewTx(key, signer14, &types.AccessListTx{Nonce: 1}),
			types.MustSignNewTx(key, signer12, &types.AccessListTx{Nonce: 2}),
			types.MustSignNewTx(key, signer12, &types.AccessListTx{Nonce: 3}),
		},
		"middle different": {
			types.MustSignNewTx(key, signer12, &types.AccessListTx{Nonce: 1}),
			types.MustSignNewTx(key, signer14, &types.AccessListTx{Nonce: 2}),
			types.MustSignNewTx(key, signer12, &types.AccessListTx{Nonce: 3}),
		},
		"last different": {
			types.MustSignNewTx(key, signer12, &types.AccessListTx{Nonce: 1}),
			types.MustSignNewTx(key, signer12, &types.AccessListTx{Nonce: 2}),
			types.MustSignNewTx(key, signer14, &types.AccessListTx{Nonce: 3}),
		},
	}

	for name, transactions := range tests {
		t.Run(name, func(t *testing.T) {
			validPlan := ExecutionPlan{
				Range: BlockRange{Earliest: 10, Latest: 20},
				Root:  NewTxStep(TxReference{}),
			}
			require.NoError(t, validatePlan(validPlan))

			signer := signer12
			sender := crypto.PubkeyToAddress(key.PublicKey)

			index := map[TxReference]*types.Transaction{}
			for _, tx := range transactions {
				stripped, err := removeBundleOnlyMark(tx)
				require.NoError(t, err)
				hash := signer.Hash(stripped)

				ref := TxReference{
					From: sender,
					Hash: hash,
				}
				index[ref] = tx
			}

			bundle := TransactionBundle{
				Plan:         validPlan,
				Transactions: index,
			}

			require.ErrorContains(t, validateBundle(signer12, bundle),
				"invalid transaction in bundle: invalid chain id",
			)
		})
	}
}

func TestValidateBundle_MissingSigner_ProducesAnError(t *testing.T) {
	validPlan := ExecutionPlan{
		Range: BlockRange{Earliest: 10, Latest: 20},
		Root:  NewTxStep(TxReference{}),
	}

	bundle := TransactionBundle{
		Plan: validPlan,
	}

	require.ErrorContains(t, validateBundle(nil, bundle), "signer is nil")
}

func TestValidateBundle_InvalidIndex_Rejected(t *testing.T) {
	signer := types.LatestSignerForChainID(big.NewInt(123))
	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	bundle := NewBuilder().AllOf(
		Step(key, &types.AccessListTx{Nonce: 1}),
	).WithSigner(signer).BuildBundle()

	require.NoError(t, validateBundle(signer, bundle))

	ref := slices.Collect(maps.Keys(bundle.Transactions))[0]
	validTxData := utils.GetTxData(bundle.Transactions[ref]).(*types.AccessListTx)

	// using an unsigned transaction in the index is detected
	validTxData.R = nil
	unsignedTx := types.NewTx(validTxData)
	bundle.Transactions[ref] = unsignedTx
	require.ErrorContains(t, validateBundle(signer, bundle),
		"invalid transaction in bundle",
	)

	// Changing the signer of the transaction is detected
	otherKey, err := crypto.GenerateKey()
	require.NoError(t, err)
	otherTx := types.MustSignNewTx(otherKey, signer, validTxData)
	bundle.Transactions[ref] = otherTx

	require.ErrorContains(t, validateBundle(signer, bundle),
		"sender in transaction reference does not match actual sender",
	)

	// Change in the transaction data is detected
	validTxData.Value = big.NewInt(1234)
	changedDataTx := types.MustSignNewTx(key, signer, validTxData)
	bundle.Transactions[ref] = changedDataTx

	require.ErrorContains(t, validateBundle(signer, bundle),
		"content of transaction does not match transaction hash",
	)
}

func TestValidateBundle_UsageOfLegacyTransaction_Rejected(t *testing.T) {
	validPlan := ExecutionPlan{
		Range: BlockRange{Earliest: 10, Latest: 20},
		Root:  NewTxStep(TxReference{}),
	}
	require.NoError(t, validatePlan(validPlan))

	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	signer := types.LatestSignerForChainID(big.NewInt(123))
	tx := types.MustSignNewTx(key, signer, &types.LegacyTx{})

	ref := TxReference{
		From: crypto.PubkeyToAddress(key.PublicKey),
		Hash: signer.Hash(tx),
	}

	index := map[TxReference]*types.Transaction{
		ref: tx,
	}

	bundle := TransactionBundle{
		Plan:         validPlan,
		Transactions: index,
	}

	require.ErrorContains(t, validateBundle(signer, bundle),
		"invalid transaction in bundle: unsupported transaction type: 0",
	)
}

func TestValidateBundle_TransactionNotAgreeingToPlan_Rejected(t *testing.T) {
	signer := types.LatestSignerForChainID(big.NewInt(123))
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	bundle := NewBuilder().AllOf(
		Step(key, &types.AccessListTx{}),
	).WithSigner(signer).BuildBundle()

	require.NoError(t, validateBundle(signer, bundle))

	originalTx := bundle.GetTransactionsInReferencedOrder()[0]

	// removing the agreement to the execution plan is detected
	noAgreementData := utils.GetTxData(originalTx).(*types.AccessListTx)
	noAgreementData.AccessList = []types.AccessTuple{{Address: BundleOnly}}
	noAgreement := types.MustSignNewTx(key, signer, noAgreementData)
	require.True(t, IsBundleOnly(noAgreement))

	ref := slices.Collect(maps.Keys(bundle.Transactions))[0]
	bundle.Transactions[ref] = noAgreement

	require.ErrorContains(t, validateBundle(signer, bundle),
		"contains transaction not approving the execution plan",
	)

	// restore the valid bundle
	bundle.Transactions[ref] = originalTx
	require.NoError(t, validateBundle(signer, bundle))

	// replacing the agreement with another execution hash is also detected
	otherAgreementData := utils.GetTxData(originalTx).(*types.AccessListTx)
	otherAgreementData.AccessList = []types.AccessTuple{{
		Address:     BundleOnly,
		StorageKeys: []common.Hash{{0x99}},
	}}
	otherAgreement := types.MustSignNewTx(key, signer, otherAgreementData)
	require.True(t, IsBundleOnly(otherAgreement))

	bundle.Transactions[ref] = otherAgreement

	require.ErrorContains(t, validateBundle(signer, bundle),
		"contains transaction not approving the execution plan",
	)
}

func TestValidateBundle_MissingTransactionInIndex_Rejected(t *testing.T) {
	signer := types.LatestSignerForChainID(big.NewInt(123))
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	bundle := NewBuilder().AllOf(
		Step(key, &types.AccessListTx{Nonce: 1}),
		Step(key, &types.AccessListTx{Nonce: 2}),
	).WithSigner(signer).BuildBundle()

	require.NoError(t, validateBundle(signer, bundle))

	ref1 := slices.Collect(maps.Keys(bundle.Transactions))[0]
	delete(bundle.Transactions, ref1)
	require.ErrorContains(t, validateBundle(signer, bundle),
		"missing transaction referenced by the execution plan",
	)
}

func TestValidateBundle_AdditionalTransactionInIndex_Rejected(t *testing.T) {
	signer := types.LatestSignerForChainID(big.NewInt(123))
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	bundle := NewBuilder().AllOf(
		Step(key, &types.AccessListTx{Nonce: 1}),
		Step(key, &types.AccessListTx{Nonce: 2}),
	).WithSigner(signer).BuildBundle()

	require.NoError(t, validateBundle(signer, bundle))

	key2, err := crypto.GenerateKey()
	require.NoError(t, err)

	ref1 := slices.Collect(maps.Keys(bundle.Transactions))[0]
	validTx := bundle.Transactions[ref1]

	extraTx := types.MustSignNewTx(key2, signer, utils.GetTxData(validTx))
	refExtra := TxReference{
		From: crypto.PubkeyToAddress(key2.PublicKey),
		Hash: ref1.Hash,
	}
	bundle.Transactions[refExtra] = extraTx

	require.ErrorContains(t, validateBundle(signer, bundle),
		"contains transaction not referenced by the execution plan",
	)
}

func TestValidatePlan_AcceptsValidPlans(t *testing.T) {
	validPlans := []ExecutionPlan{
		{
			Root:  NewTxStep(TxReference{}),
			Range: BlockRange{Earliest: 10, Latest: 10},
		},
		{
			Root:  NewAllOfStep(NewTxStep(TxReference{}), NewTxStep(TxReference{})),
			Range: BlockRange{Earliest: 10, Latest: 20},
		},
		{
			Root:  NewOneOfStep(NewTxStep(TxReference{}), NewTxStep(TxReference{})),
			Range: BlockRange{Earliest: 0, Latest: MaxBlockRange - 1},
		},
	}

	for _, plan := range validPlans {
		require.NoError(t, validatePlan(plan))
	}
}

func TestValidatePlan_DetectsInvalidPlans(t *testing.T) {
	tests := map[string]struct {
		plan  ExecutionPlan
		issue string
	}{
		"empty plan": {
			plan:  ExecutionPlan{},
			issue: "invalid execution plan",
		},
		"invalid root step": {
			plan: ExecutionPlan{
				Root:  ExecutionStep{}, // invalid step
				Range: BlockRange{Earliest: 10, Latest: 20},
			},
			issue: "invalid execution plan",
		},
		"invalid block range": {
			plan: ExecutionPlan{
				Root:  NewTxStep(TxReference{}),
				Range: BlockRange{Earliest: 20, Latest: 10}, // invalid range
			},
			issue: "invalid block range",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require.ErrorContains(t, validatePlan(test.plan), test.issue)
		})
	}

	invalidPlanAndRange := ExecutionPlan{
		Root:  ExecutionStep{},                      // invalid step
		Range: BlockRange{Earliest: 20, Latest: 10}, // invalid range
	}

	require.Error(t, validateStep(invalidPlanAndRange.Root))
	require.Error(t, validateRange(invalidPlanAndRange.Range))
	require.Error(t, validatePlan(invalidPlanAndRange))
}

func TestValidateStep_AcceptsValidSteps(t *testing.T) {
	validSteps := []ExecutionStep{
		// -- atomic steps --
		NewTxStep(TxReference{}),
		NewTxStep(TxReference{}).WithFlags(EF_Default),
		NewTxStep(TxReference{}).WithFlags(EF_TolerateFailed),
		NewTxStep(TxReference{}).WithFlags(EF_TolerateInvalid),
		NewTxStep(TxReference{}).WithFlags(EF_TolerateFailed | EF_TolerateInvalid),

		// -- all-of steps --
		NewAllOfStep(),
		NewAllOfStep(
			NewTxStep(TxReference{}),
			NewTxStep(TxReference{}),
		),
		NewAllOfStep(
			NewTxStep(TxReference{}),
			NewTxStep(TxReference{}),
		).WithFlags(EF_TolerateFailed),

		// -- one-of steps --
		NewOneOfStep(),
		NewOneOfStep(
			NewTxStep(TxReference{}),
			NewTxStep(TxReference{}),
		),
		NewOneOfStep(
			NewTxStep(TxReference{}),
			NewTxStep(TxReference{}),
		).WithFlags(EF_TolerateFailed),

		// -- combined steps --
		NewOneOfStep(
			NewTxStep(TxReference{}),
			NewAllOfStep(
				NewTxStep(TxReference{}),
			),
		),
	}

	for _, step := range validSteps {
		require.NoError(t, validateStep(step))
	}
}

func TestValidateStep_DetectsInvalidSteps(t *testing.T) {
	tests := map[string]struct {
		step  ExecutionStep
		issue string
	}{
		"empty step": {
			step:  ExecutionStep{},
			issue: "malformed execution step",
		},
		"step with both single and group set": {
			step: ExecutionStep{
				single: &single{},
				group:  &group{},
			},
			issue: "malformed execution step",
		},
		"invalid execution flags": {
			step:  NewTxStep(TxReference{}).WithFlags(0xFF),
			issue: "invalid execution flags in step",
		},
		"malformed nested all-of step": {
			step: NewAllOfStep(
				ExecutionStep{}, // invalid step
			),
			issue: "malformed execution step",
		},
		"malformed nested one-of step": {
			step: NewOneOfStep(
				ExecutionStep{}, // invalid step
			),
			issue: "malformed execution step",
		},
		"invalid nested execution flags": {
			step: NewAllOfStep(
				NewTxStep(TxReference{}),
				NewOneOfStep(
					NewTxStep(TxReference{}),
					NewTxStep(TxReference{}).WithFlags(0xFF),
					NewTxStep(TxReference{}),
				),
				NewTxStep(TxReference{}),
			),
			issue: "invalid execution flags in step",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require.ErrorContains(t, validateStep(test.step), test.issue)
		})
	}
}

func TestValidateStep_DetectsExcessiveNesting(t *testing.T) {
	require.NoError(t, validateStep(NewAllOfStep(
		wrapInNested(NewTxStep(TxReference{}), MaxGroupNestingDepth-1),
		wrapInNested(NewTxStep(TxReference{}), MaxGroupNestingDepth-1),
		wrapInNested(NewTxStep(TxReference{}), MaxGroupNestingDepth-1),
	)))

	require.ErrorContains(t, validateStep(NewAllOfStep(
		wrapInNested(NewTxStep(TxReference{}), MaxGroupNestingDepth-1),
		wrapInNested(NewTxStep(TxReference{}), MaxGroupNestingDepth),
		wrapInNested(NewTxStep(TxReference{}), MaxGroupNestingDepth-1),
	)), "exceeds maximum nesting depth")

	for depth := range MaxGroupNestingDepth + 2 {
		step := wrapInNested(NewTxStep(TxReference{}), depth)
		if depth <= MaxGroupNestingDepth {
			require.NoError(t, validateStep(step))
		} else {
			require.ErrorContains(t, validateStep(step), "exceeds maximum nesting depth")
		}
	}
}

func TestValidateStep_MaximumNestingDepthMatchesConstant(t *testing.T) {
	inner := NewTxStep(TxReference{})
	allowed := wrapInNested(inner, MaxGroupNestingDepth)
	invalid := wrapInNested(inner, MaxGroupNestingDepth+1)

	// make sure wrapInNested produces the correct number of nested groups
	count := strings.Count(allowed.String(), "OneOf")
	require.Equal(t, MaxGroupNestingDepth, count)

	count = strings.Count(invalid.String(), "OneOf")
	require.Equal(t, MaxGroupNestingDepth+1, count)

	require.NoError(t, validateStep(allowed))
	require.ErrorContains(t, validateStep(invalid), "exceeds maximum nesting depth")

}

func wrapInNested(inner ExecutionStep, depth int) ExecutionStep {
	if depth == 0 {
		return inner
	}
	return NewOneOfStep(
		wrapInNested(inner, depth-1),
	)
}

func TestValidateRange_AcceptsValidRanges(t *testing.T) {
	tests := []BlockRange{
		{0, 0},
		{0, 100},
		{0, MaxBlockRange - 1},
		{10, 10},
		{10, 100},
		{10, MaxBlockRange + 9},
	}

	for _, tc := range tests {
		require.NoError(t, validateRange(tc))
	}
}

func TestValidateRange_DetectsInvalidRanges(t *testing.T) {
	tests := []struct {
		blockRange BlockRange
		issue      string
	}{
		{
			blockRange: BlockRange{1, 0},
			issue:      "empty block range",
		},
		{
			blockRange: BlockRange{10, 9},
			issue:      "empty block range",
		},
		{
			blockRange: BlockRange{MaxBlockRange, 0},
			issue:      "empty block range",
		},
		{
			blockRange: BlockRange{0, MaxBlockRange},
			issue:      "invalid block range",
		},
		{
			blockRange: BlockRange{10, MaxBlockRange + 10},
			issue:      "invalid block range",
		},
		{
			blockRange: BlockRange{10, MaxBlockRange + 100},
			issue:      "invalid block range",
		},
	}

	for _, tc := range tests {
		require.ErrorContains(t, validateRange(tc.blockRange), tc.issue)
	}
}

func TestValidateRange_ComprehensiveRangeChecks(t *testing.T) {
	for earliest := range 2 * MaxBlockRange {
		for latest := range 2 * MaxBlockRange {
			r := BlockRange{earliest, latest}
			if size := r.Size(); size > 0 && size <= MaxBlockRange {
				require.NoError(t, validateRange(r),
					"earliest=%d, latest=%d", earliest, latest,
				)
			} else {
				require.Error(t, validateRange(r),
					"earliest=%d, latest=%d", earliest, latest,
				)
			}
		}
	}
}

func TestBelongsToExecutionPlan_IdentifiesTransactionsWhichSignTheExecutionPlan(t *testing.T) {

	executionPlanHash := common.Hash{0x01, 0x02, 0x03}

	tests := map[string]struct {
		tx                types.TxData
		executionPlanHash common.Hash
		expected          bool
	}{
		"transaction without access list": {
			tx:       &types.LegacyTx{},
			expected: false,
		},
		"transaction with bundle-only but no plan hash": {
			tx: &types.AccessListTx{
				AccessList: types.AccessList{
					{
						Address: BundleOnly,
					},
				},
			},
			executionPlanHash: executionPlanHash,
			expected:          false,
		},
		"fragmented access list": {
			tx: &types.AccessListTx{
				AccessList: types.AccessList{
					{
						Address: BundleOnly,
					},
					{
						Address:     common.HexToAddress("0x0000000000000000000000000000000000000001"),
						StorageKeys: []common.Hash{executionPlanHash},
					},
				},
			},
			executionPlanHash: executionPlanHash,
			expected:          false,
		},
		"transaction with bundle-only and matching plan hash": {
			tx: &types.AccessListTx{
				AccessList: types.AccessList{
					{
						Address:     BundleOnly,
						StorageKeys: []common.Hash{executionPlanHash},
					},
				},
			},
			executionPlanHash: executionPlanHash,
			expected:          true,
		},
		"transaction with multiple accepted plans": {
			tx: &types.AccessListTx{
				AccessList: types.AccessList{
					{
						Address:     BundleOnly,
						StorageKeys: []common.Hash{{0x0A}},
					},
					{
						Address:     BundleOnly,
						StorageKeys: []common.Hash{executionPlanHash},
					},
					{
						Address:     BundleOnly,
						StorageKeys: []common.Hash{{0x0B}},
					},
				},
			},
			executionPlanHash: executionPlanHash,
			expected:          true,
		},
		"transaction with multiple accepted plans compact": {
			tx: &types.AccessListTx{
				AccessList: types.AccessList{
					{
						Address:     BundleOnly,
						StorageKeys: []common.Hash{{0x0A}, executionPlanHash, {0x0B}},
					},
				},
			},
			executionPlanHash: executionPlanHash,
			expected:          true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, test.expected,
				belongsToExecutionPlan(types.NewTx(test.tx), test.executionPlanHash))
		})
	}
}
