package bundle

import (
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func Test_Builder_BuildEmptyBundle(t *testing.T) {
	tx := NewAllOf()

	_, _, err := ValidateTransactionBundle(tx)
	require.NoError(t, err)
}

func Test_NewAllOf_BuildBundle(t *testing.T) {

	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	tx := NewAllOf(
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

func Test_NewOneOf_BuildBundle(t *testing.T) {

	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	tx := NewOneOf(
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

func Test_NewNestedBundle(t *testing.T) {

	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	inner := NewOneOf(
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

	outer := NewAllOf(
		Step(key, &types.AccessListTx{
			Nonce: 2,
		}),
		Nested(key, inner),
		Step(key, &types.AccessListTx{
			Nonce: 3,
		}),
	)

	_, _, err = ValidateTransactionBundle(inner)
	require.NoError(t, err)

	_, _, err = ValidateTransactionBundle(outer)
	require.NoError(t, err)

}
