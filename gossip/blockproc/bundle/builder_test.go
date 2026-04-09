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
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestBundleBuilder_Build_AllowsToBuildBundleAsSpecified(t *testing.T) {
	signer := types.LatestSignerForChainID(testChainID)

	key1, err := crypto.GenerateKey()
	require.NoError(t, err)

	key2, err := crypto.GenerateKey()
	require.NoError(t, err)

	keyE, err := crypto.GenerateKey()
	require.NoError(t, err)

	tx := NewBuilder(signer).
		SetFlags(EF_AllOf|EF_TolerateFailed).
		SetEarliest(12).
		SetLatest(15).
		With(
			Step(key1, &types.AccessListTx{
				Nonce: 1,
			}),
			Step(key2, &types.AccessListTx{
				Nonce: 2,
			}),
		).
		SetEnvelopeSenderKey(keyE).
		Build()

	bundle, plan, err := ValidateEnvelope(signer, tx)
	require.NoError(t, err)

	require.Equal(t, EF_AllOf|EF_TolerateFailed, plan.Flags)
	require.EqualValues(t, 12, plan.Range.Earliest)
	require.EqualValues(t, 15, plan.Range.Latest)

	txs := bundle.Transactions
	require.Len(t, txs, 2)

	sender1, err := signer.Sender(txs[0])
	require.NoError(t, err)
	require.Equal(t, crypto.PubkeyToAddress(key1.PublicKey), sender1)

	sender2, err := signer.Sender(txs[1])
	require.NoError(t, err)
	require.Equal(t, crypto.PubkeyToAddress(key2.PublicKey), sender2)

	envelopeSigner, err := signer.Sender(tx)
	require.NoError(t, err)
	require.Equal(t, crypto.PubkeyToAddress(keyE.PublicKey), envelopeSigner)
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
		types.NewTx(&types.LegacyTx{}), // = *types.Transaction
	}

	for _, input := range inputs {
		require.NotPanics(t, func() {
			Step(nil, input)
		})
	}
}

func TestBundleBuilder_Panics_WhenNestingUnsupportedTxTypes(t *testing.T) {

	cases := []types.TxData{
		&types.DynamicFeeTx{},
		&types.BlobTx{},
		&types.SetCodeTx{},
	}

	for _, txData := range cases {
		tx := types.NewTx(txData)
		t.Run(fmt.Sprintf("TxType%d", tx.Type()), func(t *testing.T) {

			require.Panics(t, func() {
				Step(nil, tx)
			}, "unsupported Tx type for Step. Only AccessListTx and LegacyTx are supported")
		})
	}
}

func TestBundleBuilder_Step_PanicsOnInvalidInput(t *testing.T) {
	require.Panics(t, func() {
		Step(nil, 12)
	}, "unsupported TxData type")
}

func TestBundleBuilder_AllOf_BuildEmptyBundle(t *testing.T) {
	signer := types.LatestSignerForChainID(testChainID)
	tx := AllOf(signer)

	_, _, err := ValidateEnvelope(signer, tx)
	require.NoError(t, err)
}

func TestBundleBuilder_AllOf_BuildBundle(t *testing.T) {
	signer := types.LatestSignerForChainID(testChainID)

	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	tx := AllOf(signer,
		Step(key, &types.AccessListTx{
			Nonce: 0,
		}),
		Step(key, &types.DynamicFeeTx{
			Nonce: 1,
		}),
		Step(key, &types.AccessListTx{
			Nonce: 2,
		}),
	)

	_, _, err = ValidateEnvelope(signer, tx)
	require.NoError(t, err)
}

func TestBundleBuilder_OneOf_BuildBundle(t *testing.T) {
	signer := types.LatestSignerForChainID(testChainID)

	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	tx := OneOf(signer,
		Step(key, &types.AccessListTx{
			Nonce: 0,
		}),
		Step(key, &types.DynamicFeeTx{
			Nonce: 1,
		}),
		Step(key, &types.AccessListTx{
			Nonce: 2,
		}),
	)

	_, _, err = ValidateEnvelope(signer, tx)
	require.NoError(t, err)
}

func TestBundleBuilder_OneOf_EmptyBundle(t *testing.T) {
	signer := types.LatestSignerForChainID(testChainID)
	tx := OneOf(signer)

	_, _, err := ValidateEnvelope(signer, tx)
	require.NoError(t, err)
}

func TestBundleBuilder_Builder_NewNestedBundle(t *testing.T) {
	signer := types.LatestSignerForChainID(testChainID)

	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	inner := OneOf(signer,
		Step(key, &types.AccessListTx{
			Nonce: 0,
		}),
		Step(key, &types.DynamicFeeTx{
			Nonce: 1,
		}),
		Step(key, &types.AccessListTx{
			Nonce: 2,
		}),
	)

	outer := AllOf(signer,
		Step(key, &types.AccessListTx{
			Nonce: 2,
		}),
		Step(key, inner),
		Step(key, &types.AccessListTx{
			Nonce: 3,
		}),
	)

	_, _, err = ValidateEnvelope(signer, inner)
	require.NoError(t, err)

	_, _, err = ValidateEnvelope(signer, outer)
	require.NoError(t, err)

	// all combined in one

	combined := AllOf(signer,
		Step(key, OneOf(signer,
			Step(key, &types.AccessListTx{}),
			Step(key, &types.DynamicFeeTx{}),
		)),
		Step(key, AllOf(signer,
			Step(key, &types.AccessListTx{}),
		)),
	)

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
				With(Step(key, types.NewTx(&types.LegacyTx{}))).
				Build()
		})
	}
}

func TestBundleBuilder_DefaultsSignerIfUnspecified(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	tx := NewBuilder(nil).
		With(Step(key, types.NewTx(&types.LegacyTx{}))).
		Build()

	signer := types.LatestSignerForChainID(big.NewInt(1))
	_, _, err = ValidateEnvelope(signer, tx)
	require.NoError(t, err)
}

func TestBundleBuilder_CanSetGasPrice(t *testing.T) {
	signer := types.LatestSignerForChainID(testChainID)

	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	for _, price := range []*big.Int{nil, big.NewInt(1), big.NewInt(1_000_000)} {
		t.Run(price.String(), func(t *testing.T) {

			tx := NewBuilder(signer).
				SetEnvelopeGasPrice(price).
				With(
					Step(key, &types.AccessListTx{
						Nonce: 0,
					}),
				).
				Build()

			_, _, err = ValidateEnvelope(signer, tx)
			require.NoError(t, err)

			if price != nil {
				require.Equal(t, 0, tx.GasPrice().Cmp(price))
			} else {
				require.Equal(t, 0, tx.GasPrice().Cmp(big.NewInt(0)))
			}
		})
	}
}

func TestBundleBuilder_DefaultsGasPriceToZero(t *testing.T) {
	signer := types.LatestSignerForChainID(testChainID)
	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	tx := NewBuilder(signer).
		With(
			Step(key, &types.AccessListTx{
				Nonce: 0,
			}),
		).
		Build()

	_, _, err = ValidateEnvelope(signer, tx)
	require.NoError(t, err)

	require.Equal(t, 0, tx.GasPrice().Cmp(big.NewInt(0)))
}
