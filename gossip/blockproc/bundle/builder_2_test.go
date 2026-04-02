package bundle

import (
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestBundleBuilder2_BuildAllOf(t *testing.T) {
	key1, err := crypto.GenerateKey()
	require.NoError(t, err)

	key2, err := crypto.GenerateKey()
	require.NoError(t, err)

	tx := NewBuilder2().
		AllOf(
			Step2(key1, &types.AccessListTx{
				Nonce: 1,
			}),
			Step2(key2, &types.AccessListTx{
				Nonce: 2,
			}),
		).
		Build()

	require.NotNil(t, tx)
}

func TestBuilder2_BuildOneOf(t *testing.T) {
	key1, err := crypto.GenerateKey()
	require.NoError(t, err)

	key2, err := crypto.GenerateKey()
	require.NoError(t, err)

	tx := NewBuilder2().
		OneOf(
			Step2(key1, &types.AccessListTx{
				Nonce: 1,
			}),
			Step2(key2, &types.AccessListTx{
				Nonce: 2,
			}),
		).
		Build()

	require.NotNil(t, tx)
}

func TestBuilder2_BuildNested(t *testing.T) {
	key1, err := crypto.GenerateKey()
	require.NoError(t, err)

	key2, err := crypto.GenerateKey()
	require.NoError(t, err)

	tx := NewBuilder2().
		OneOf(
			AllOf2(
				Step2(key1, &types.AccessListTx{
					Nonce: 1,
				}),
				Step2(key2, &types.AccessListTx{
					Nonce: 2,
				}),
			),
			AllOf2(
				Step2(key2, &types.AccessListTx{
					Nonce: 2,
				}),
				Step2(key1, &types.AccessListTx{
					Nonce: 1,
				}),
			),
		).
		Build()

	require.NotNil(t, tx)
}
