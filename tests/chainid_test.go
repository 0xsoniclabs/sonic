package tests

import (
	"context"
	"fmt"
	"math/big"
	"slices"
	"testing"

	"github.com/0xsoniclabs/sonic/config"
	"github.com/0xsoniclabs/sonic/ethapi"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestChainId_RejectsAllTxSignedWithWrongChainId(t *testing.T) {

	// Homestead signer is not included because it does not have a chain ID
	signerSupportedTypes := map[string]struct {
		signer  types.Signer
		txTypes []byte
	}{
		"eip155": {
			types.NewEIP155Signer(big.NewInt(1)),
			[]byte{types.LegacyTxType},
		},
		"eip2930": {
			types.NewEIP2930Signer(big.NewInt(1)),
			[]byte{types.LegacyTxType, types.AccessListTxType},
		},
		"london": {
			types.NewLondonSigner(big.NewInt(1)),
			[]byte{types.LegacyTxType, types.AccessListTxType, types.DynamicFeeTxType},
		},
		"cancun": {
			types.NewCancunSigner(big.NewInt(1)),
			[]byte{types.LegacyTxType, types.AccessListTxType, types.DynamicFeeTxType,
				types.BlobTxType},
		},
		"prague": {
			types.NewPragueSigner(big.NewInt(1)),
			[]byte{types.LegacyTxType, types.AccessListTxType, types.DynamicFeeTxType,
				types.BlobTxType, types.SetCodeTxType},
		},
	}

	getTxsOfAllTypes := map[string]types.TxData{
		"Legacy":     &types.LegacyTx{GasPrice: big.NewInt(enoughGasPrice)},
		"AccessList": &types.AccessListTx{GasPrice: big.NewInt(enoughGasPrice)},
		"DynamicFee": &types.DynamicFeeTx{GasFeeCap: big.NewInt(enoughGasPrice)},
		"Blob":       &types.BlobTx{GasFeeCap: uint256.NewInt(enoughGasPrice)},
		"SetCode": &types.SetCodeTx{
			AuthList:  []types.SetCodeAuthorization{{}},
			GasFeeCap: uint256.NewInt(enoughGasPrice)},
	}

	net := StartIntegrationTestNet(t)
	account := makeAccountWithBalance(t, net, big.NewInt(1e18))

	for signerName, test := range signerSupportedTypes {
		for txTypeName, txData := range getTxsOfAllTypes {
			t.Run(fmt.Sprintf("%s_%s", signerName, txTypeName), func(t *testing.T) {

				tx := types.NewTx(txData)
				if !slices.Contains(test.txTypes, tx.Type()) {
					_, err := types.SignTx(tx, test.signer, account.PrivateKey)
					require.Error(t, err)
					return
				}

				signedTx, err := types.SignTx(tx, test.signer, account.PrivateKey)
				require.NoError(t, err, "failed to sign transaction")

				receipt, err := net.Run(signedTx)
				require.ErrorContains(t, err, "invalid sender")
				require.Nil(t, receipt, "expected nil receipt")
			})
		}
	}
}

func TestChainId_AcceptsLegacyTxSignedWith(t *testing.T) {
	// Test that Sonic client can process a transaction signed with the Homestead signer
	// and that it returns the correct chain ID (0).

	net := StartIntegrationTestNet(t,
		IntegrationTestNetOptions{
			ModifyConfig: func(config *config.Config) {
				// the transaction to deploy the contract is not replay protected
				// This has the benefit that the same tx will work in both ethereum and sonic.
				// Nevertheless the default configuration rejects this sort of transaction.
				config.Opera.AllowUnprotectedTxs = true
			},
		},
	)
	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()
	account := makeAccountWithBalance(t, net, big.NewInt(1e18))

	actualChainID, err := client.ChainID(context.Background())
	require.NoError(t, err, "failed to get chain ID")

	signers := map[string]types.Signer{
		"Homestead": types.HomesteadSigner{},
		"EIP155":    types.NewEIP155Signer(actualChainID),
	}

	for name, signer := range signers {
		t.Run(name, func(t *testing.T) {

			nonce, err := client.NonceAt(context.Background(), account.Address(), nil)
			require.NoError(t, err, "failed to get nonce")

			to := &common.Address{42}
			tx := types.NewTx(&types.LegacyTx{
				Nonce:    nonce,
				To:       to,
				Value:    big.NewInt(1),
				Gas:      1e6,
				GasPrice: big.NewInt(enoughGasPrice),
				Data:     []byte("some"),
			})

			signed, err := types.SignTx(tx, signer, account.PrivateKey)
			require.NoError(t, err, "failed to create legacy transaction")

			receipt, err := net.Run(signed)
			require.NoError(t, err, "failed to run transaction")
			require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

			var json *ethapi.RPCTransaction
			err = client.Client().CallContext(context.Background(), &json,
				"eth_getTransactionByHash", signed.Hash(),
			)
			require.NoError(t, err)
			require.Equal(t, signed.Hash(), json.Hash)
		})
	}
}

func TestChainId_AcceptsNonLegacyTxWithChainIdZero(t *testing.T) {
	zeroBig := new(big.Int).Sub(big.NewInt(1), big.NewInt(1))
	zeroUint := uint256.NewInt(0)
	enoughGas := uint64(1e6)

	tests := map[string]types.TxData{
		"AccessList": &types.AccessListTx{
			ChainID:  zeroBig,
			GasPrice: big.NewInt(enoughGasPrice),
			Gas:      enoughGas,
		},
		"DynamicFee": &types.DynamicFeeTx{
			ChainID:   zeroBig,
			GasFeeCap: big.NewInt(enoughGasPrice),
			Gas:       enoughGas,
		},
		"Blob": &types.BlobTx{
			ChainID:   zeroUint,
			GasFeeCap: uint256.NewInt(enoughGasPrice),
			Gas:       enoughGas,
		},
		"SetCode": &types.SetCodeTx{
			ChainID:   zeroUint,
			AuthList:  []types.SetCodeAuthorization{{}},
			GasFeeCap: uint256.NewInt(enoughGasPrice),
			Gas:       enoughGas,
		},
	}

	net := StartIntegrationTestNet(t,
		IntegrationTestNetOptions{FeatureSet: opera.AllegroFeatures},
	)
	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	actualChainID, err := client.ChainID(context.Background())
	require.NoError(t, err, "failed to get chain ID")

	for name, tx := range tests {
		t.Run(name, func(t *testing.T) {

			account := makeAccountWithBalance(t, net, big.NewInt(1e18))
			signed := signTransaction(t, actualChainID, tx, account)

			receipt, err := net.Run(signed)
			require.NoError(t, err, "failed to run transaction")
			require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
		})
	}
}
