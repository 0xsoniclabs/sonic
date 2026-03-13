package bundle

import (
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func Test_Builder_AllowsToBuildBundleAsSpecified(t *testing.T) {
	key1, err := crypto.GenerateKey()
	require.NoError(t, err)

	key2, err := crypto.GenerateKey()
	require.NoError(t, err)

	tx := NewBuilder().
		WithFlags(EF_AllOf|EF_TolerateFailed).
		Earliest(12).
		Latest(15).
		With(
			Step(key1, &types.AccessListTx{
				Nonce: 1,
			}),
			Step(key2, &types.AccessListTx{
				Nonce: 2,
			}),
		).Build()

	bundle, plan, err := ValidateTransactionBundle(tx)
	require.NoError(t, err)

	require.Equal(t, EF_AllOf|EF_TolerateFailed, plan.Flags)
	require.EqualValues(t, 12, plan.Earliest)
	require.EqualValues(t, 15, plan.Latest)

	txs := bundle.Bundle
	require.Len(t, txs, 2)
	signer := types.LatestSignerForChainID(txs[0].ChainId())

	sender1, err := signer.Sender(txs[0])
	require.NoError(t, err)
	require.Equal(t, crypto.PubkeyToAddress(key1.PublicKey), sender1)

	sender2, err := signer.Sender(txs[1])
	require.NoError(t, err)
	require.Equal(t, crypto.PubkeyToAddress(key2.PublicKey), sender2)
}

func Test_AllOf_BuildEmptyBundle(t *testing.T) {
	tx := AllOf()

	_, _, err := ValidateTransactionBundle(tx)
	require.NoError(t, err)
}

func Test_AllOf_BuildBundle(t *testing.T) {

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
	)

	_, _, err = ValidateTransactionBundle(tx)
	require.NoError(t, err)
}

func Test_OneOf_BuildBundle(t *testing.T) {

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
	)

	_, _, err = ValidateTransactionBundle(tx)
	require.NoError(t, err)
}

func Test_Builder_NewNestedBundle(t *testing.T) {

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
	)

	outer := AllOf(
		Step(key, &types.AccessListTx{
			Nonce: 2,
		}),
		Step(key, inner),
		Step(key, &types.AccessListTx{
			Nonce: 3,
		}),
	)

	_, _, err = ValidateTransactionBundle(inner)
	require.NoError(t, err)

	_, _, err = ValidateTransactionBundle(outer)
	require.NoError(t, err)

	// all combined in one

	combined := AllOf(
		Step(key, OneOf(
			Step(key, &types.AccessListTx{}),
			Step(key, &types.DynamicFeeTx{}),
		)),
		Step(key, AllOf(
			Step(key, &types.AccessListTx{}),
		)),
	)

	_, _, err = ValidateTransactionBundle(combined)
	require.NoError(t, err)
}
