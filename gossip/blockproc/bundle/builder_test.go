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
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestBundleBuilder_Build_AllowsToBuildBundleAsSpecified(t *testing.T) {
	require := require.New(t)
	signer := types.LatestSignerForChainID(big.NewInt(1))

	key0, err := crypto.GenerateKey()
	require.NoError(err)

	key1, err := crypto.GenerateKey()
	require.NoError(err)

	key2, err := crypto.GenerateKey()
	require.NoError(err)

	keyE, err := crypto.GenerateKey()
	require.NoError(err)

	tx := NewBuilder(signer).
		SetEarliest(12).
		SetLatest(15).
		AllOf(
			Step(key0, &types.AccessListTx{
				Nonce: 0,
			}),
			Step(key1, &types.AccessListTx{
				Nonce: 1,
			}).WithFlags(EF_TolerateFailed),
			Step(key2, &types.AccessListTx{
				Nonce: 2,
			}).WithFlags(EF_TolerateInvalid),
		).
		SetEnvelopeSenderKey(keyE).
		Build()

	bundle, plan, err := ValidateEnvelope(signer, tx)
	require.NoError(err)

	// Check the block range.
	require.Equal(bundle.Plan, *plan)
	require.EqualValues(12, plan.Range.Earliest)
	require.EqualValues(15, plan.Range.Latest)

	// check the structure of the execution plan
	require.Equal(
		"AllOf(A,Step[TolerateFailed](B),Step[TolerateInvalid](C))",
		plan.Root.String(),
	)

	// Check the transactions and their senders.
	txs := bundle.Transactions
	require.Len(txs, 3)

	tx0, found := txs[*plan.Root.steps[0].txRef]
	require.True(found)
	sender0, err := signer.Sender(tx0)
	require.NoError(err)
	require.Equal(crypto.PubkeyToAddress(key0.PublicKey), sender0)
	require.EqualValues(0, tx0.Nonce())

	tx1, found := txs[*plan.Root.steps[1].txRef]
	require.True(found)
	sender1, err := signer.Sender(tx1)
	require.NoError(err)
	require.Equal(crypto.PubkeyToAddress(key1.PublicKey), sender1)
	require.EqualValues(1, tx1.Nonce())

	tx2, found := txs[*plan.Root.steps[2].txRef]
	require.True(found)
	sender2, err := signer.Sender(tx2)
	require.NoError(err)
	require.Equal(crypto.PubkeyToAddress(key2.PublicKey), sender2)
	require.EqualValues(2, tx2.Nonce())
}

func TestBundleBuilder_BuildComposedBundles(t *testing.T) {
	signer := types.LatestSignerForChainID(big.NewInt(1))

	keyA, err := crypto.GenerateKey()
	require.NoError(t, err)

	keyB, err := crypto.GenerateKey()
	require.NoError(t, err)

	keyC, err := crypto.GenerateKey()
	require.NoError(t, err)

	A := Step(keyA, &types.AccessListTx{Nonce: 1})
	B := Step(keyB, &types.AccessListTx{Nonce: 2})
	C := Step(keyC, &types.AccessListTx{Nonce: 3})
	envelope := NewBuilder(signer).
		OneOf(
			AllOf(A, B),
			AllOf(A, C),
			AllOf(B, C),
		).Build()

	bundle, _, err := ValidateEnvelope(signer, envelope)
	require.NoError(t, err)

	require.Len(t, bundle.Transactions, 3)
	require.Equal(t,
		"OneOf(AllOf(A,B),AllOf(A,C),AllOf(B,C))",
		bundle.Plan.Root.String(),
	)

	// TODO: check proper transaction details
}

func TestBundleBuilder_Step_AcceptsVariousInputTypes(t *testing.T) {
	inputs := []any{
		types.AccessListTx{},
		types.DynamicFeeTx{},
		types.BlobTx{},
		types.SetCodeTx{},
		&types.AccessListTx{},
		&types.DynamicFeeTx{},
		&types.BlobTx{},
		&types.SetCodeTx{},
		types.NewTx(&types.LegacyTx{}),
		types.NewTx(&types.AccessListTx{}),
		types.NewTx(&types.DynamicFeeTx{}),
		types.NewTx(&types.BlobTx{}),
		types.NewTx(&types.SetCodeTx{}),
	}

	for _, input := range inputs {
		require.NotPanics(t, func() {
			Step(nil, input)
		})
	}
}

func TestBundleBuilder_Step_PanicsOnInvalidInput(t *testing.T) {
	require.Panics(t, func() {
		Step(nil, 12)
	}, "unsupported TxData type")
}

func TestBundleBuilder_AllOf_BuildEmptyBundle(t *testing.T) {
	signer := types.LatestSignerForChainID(big.NewInt(1))
	tx := AllOf().Build(signer)

	_, _, err := ValidateEnvelope(signer, tx)
	require.NoError(t, err)
}

func TestBundleBuilder_AllOf_BuildBundle(t *testing.T) {
	signer := types.LatestSignerForChainID(big.NewInt(1))

	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	tx := AllOf(
		Step(key, &types.AccessListTx{
			Nonce: 0,
		}),
		Step(key, &types.DynamicFeeTx{
			Nonce: 1,
		}),
		Step(key, &types.AccessListTx{
			Nonce: 2,
		}),
	).Build(signer)

	_, _, err = ValidateEnvelope(signer, tx)
	require.NoError(t, err)
}

func TestBundleBuilder_OneOf_BuildBundle(t *testing.T) {
	signer := types.LatestSignerForChainID(big.NewInt(1))

	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	tx := OneOf(
		Step(key, &types.AccessListTx{
			Nonce: 0,
		}),
		Step(key, &types.DynamicFeeTx{
			Nonce: 1,
		}),
		Step(key, &types.AccessListTx{
			Nonce: 2,
		}),
	).Build(signer)

	_, _, err = ValidateEnvelope(signer, tx)
	require.NoError(t, err)
}

func TestBundleBuilder_OneOf_EmptyBundle(t *testing.T) {
	signer := types.LatestSignerForChainID(big.NewInt(1))
	tx := OneOf().Build(signer)

	_, _, err := ValidateEnvelope(signer, tx)
	require.NoError(t, err)
}

func TestBundleBuilder_Builder_NewNestedBundle(t *testing.T) {
	signer := types.LatestSignerForChainID(big.NewInt(1))

	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	inner := OneOf(
		Step(key, &types.AccessListTx{
			Nonce: 0,
		}),
		Step(key, &types.DynamicFeeTx{
			Nonce: 1,
		}),
		Step(key, &types.AccessListTx{
			Nonce: 2,
		}),
	).Build(signer)

	outer := AllOf(
		Step(key, &types.AccessListTx{
			Nonce: 2,
		}),
		Step(key, inner),
		Step(key, &types.AccessListTx{
			Nonce: 3,
		}),
	).Build(signer)

	_, _, err = ValidateEnvelope(signer, inner)
	require.NoError(t, err)

	_, _, err = ValidateEnvelope(signer, outer)
	require.NoError(t, err)

	// all combined in one

	combined := AllOf(
		Step(key, OneOf(
			Step(key, &types.AccessListTx{}),
			Step(key, &types.DynamicFeeTx{}),
		).Build(signer)),
		Step(key, AllOf(
			Step(key, &types.AccessListTx{}),
		).Build(signer)),
	).Build(signer)

	_, _, err = ValidateEnvelope(signer, combined)
	require.NoError(t, err)
}

func TestBundleBuilder_Regression_RespectsChainID(t *testing.T) {

	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	for _, chainId := range []int64{1, 123} {
		signer := types.LatestSignerForChainID(big.NewInt(chainId))
		require.NotPanics(t, func() {
			NewBuilder(signer).
				// the following line promotes a legacy tx (without chain id) to access list
				// the bug yielded invalid chain id panic during signing
				AllOf(Step(key, types.NewTx(&types.LegacyTx{}))).
				Build()
		})
	}
}

func TestBundleBuilder_DefaultsSignerIfUnspecified(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	tx := NewBuilder(nil).
		AllOf(Step(key, types.NewTx(&types.LegacyTx{}))).
		Build()

	signer := types.LatestSignerForChainID(big.NewInt(1))
	_, _, err = ValidateEnvelope(signer, tx)
	require.NoError(t, err)
}
