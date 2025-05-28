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

	net := StartIntegrationTestNet(t)
	account := makeAccountWithBalance(t, net, big.NewInt(1e18))
	client, err := net.GetClient()
	require.NoError(t, err, "failed to get client")
	defer client.Close()
	actualChainID, err := client.ChainID(context.Background())
	require.NoError(t, err, "failed to get chain ID")
	differentChainId := new(big.Int).Add(actualChainID, big.NewInt(1))

	// Homestead signer is not included because it does not have a chain ID
	signerSupportedTypes := map[string]struct {
		signer  types.Signer
		txTypes []byte
	}{
		"eip155": {
			types.NewEIP155Signer(differentChainId),
			[]byte{types.LegacyTxType},
		},
		"eip2930": {
			types.NewEIP2930Signer(differentChainId),
			[]byte{types.LegacyTxType, types.AccessListTxType},
		},
		"london": {
			types.NewLondonSigner(differentChainId),
			[]byte{types.LegacyTxType, types.AccessListTxType, types.DynamicFeeTxType},
		},
		"cancun": {
			types.NewCancunSigner(differentChainId),
			[]byte{types.LegacyTxType, types.AccessListTxType, types.DynamicFeeTxType,
				types.BlobTxType},
		},
		"prague": {
			types.NewPragueSigner(differentChainId),
			[]byte{types.LegacyTxType, types.AccessListTxType, types.DynamicFeeTxType,
				types.BlobTxType, types.SetCodeTxType},
		},
	}

	// no chain id is specified because all signers used in this tests override
	// the chain ID of the transaction to the chain ID that was used to initialize
	// the signer.
	getTxsOfAllTypes := map[string]types.TxData{
		"Legacy":     &types.LegacyTx{GasPrice: big.NewInt(enoughGasPrice)},
		"AccessList": &types.AccessListTx{GasPrice: big.NewInt(enoughGasPrice)},
		"DynamicFee": &types.DynamicFeeTx{GasFeeCap: big.NewInt(enoughGasPrice)},
		"Blob":       &types.BlobTx{GasFeeCap: uint256.NewInt(enoughGasPrice)},
		"SetCode": &types.SetCodeTx{
			AuthList:  []types.SetCodeAuthorization{{}},
			GasFeeCap: uint256.NewInt(enoughGasPrice)},
	}

	for signerName, test := range signerSupportedTypes {
		for txTypeName, txData := range getTxsOfAllTypes {
			t.Run(fmt.Sprintf("%s_%s", signerName, txTypeName), func(t *testing.T) {

				tx := types.NewTx(txData)
				// if the signer does not support the transaction type,
				// it should return an error when trying to sign it.
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

func TestChainId_AcceptsLegacyTxSignedWithHomestead(t *testing.T) {
	net := StartIntegrationTestNet(t,
		IntegrationTestNetOptions{
			ModifyConfig: func(config *config.Config) {
				// The transactions signed with the Homestead are not replay protected.
				// The default configuration rejects this sort of transaction,
				// so they need to be explicitly allowed.
				config.Opera.AllowUnprotectedTxs = true
			},
		},
	)
	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()
	account := makeAccountWithBalance(t, net, big.NewInt(1e18))

	// get current nonce and sign the tx.
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

	signed, err := types.SignTx(tx, types.HomesteadSigner{}, account.PrivateKey)
	require.NoError(t, err, "failed to sign legacy transaction")

	receipt, err := net.Run(signed)
	require.NoError(t, err, "failed to run transaction")
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// get the transaction by hash and verify that it has the correct chain ID
	var json *ethapi.RPCTransaction
	err = client.Client().CallContext(context.Background(), &json,
		"eth_getTransactionByHash", signed.Hash(),
	)
	require.NoError(t, err)
	require.Equal(t, signed.Hash(), json.Hash)
	// Since HomesteadSigner does not have a chain ID, the transaction should be
	// processed and stored with chain ID 0.
	require.Equal(t, json.ChainID.ToInt().Cmp(big.NewInt(0)), 0)

	// reconstruct the transaction from the RPC response
	// and verify that it has the same hash and chain ID as the signed transaction
	decodedTx := rpcTransactionToTransaction(t, json)
	require.Equal(t, signed.Hash(), decodedTx.Hash())
	require.Equal(t, signed.ChainId().Int64(), decodedTx.ChainId().Int64())
}

func TestChainId_NonLegacyTxWithChainIdZeroAreSignedAndProcessed(t *testing.T) {
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
		IntegrationTestNetOptions{Upgrades: AsPointer(opera.GetAllegroUpgrades())},
	)
	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	actualChainID, err := client.ChainID(context.Background())
	require.NoError(t, err, "failed to get chain ID")

	for name, tx := range tests {
		t.Run(name, func(t *testing.T) {

			account := makeAccountWithBalance(t, net, big.NewInt(1e18))
			// signing transaction with any types.modernSinger will change the
			// chain ID of the signed transaction to the chain ID that was used
			// to initialize the signer.
			signed, err := types.SignTx(
				types.NewTx(tx),
				types.NewPragueSigner(actualChainID),
				account.PrivateKey)
			require.NoError(t, err)
			require.Equal(t, signed.ChainId().Int64(), actualChainID.Int64())

			receipt, err := net.Run(signed)
			require.NoError(t, err, "failed to run transaction")
			require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

			// verify that processed transaction can be retrieved by hash
			// and that it has the correct chain ID
			var json *ethapi.RPCTransaction
			err = client.Client().CallContext(context.Background(), &json,
				"eth_getTransactionByHash", signed.Hash(),
			)
			require.NoError(t, err)
			require.Equal(t, signed.Hash(), json.Hash)
			require.Equal(t, signed.ChainId().Int64(), json.ChainID.ToInt().Int64())
			require.Equal(t, json.ChainID.ToInt().Int64(), actualChainID.Int64())
			// reconstruct the transaction from the RPC response
			// and verify that it has the same hash and chain ID as the signed transaction
			decodedTx := rpcTransactionToTransaction(t, json)
			require.Equal(t, signed.Hash(), decodedTx.Hash())
			require.Equal(t, signed.ChainId().Int64(), decodedTx.ChainId().Int64())

		})
	}
}

func rpcTransactionToTransaction(t *testing.T, tx *ethapi.RPCTransaction) *types.Transaction {
	t.Helper()

	switch tx.Type {
	case types.LegacyTxType:
		return types.NewTx(&types.LegacyTx{
			Nonce:    uint64(tx.Nonce),
			Gas:      uint64(tx.Gas),
			GasPrice: tx.GasPrice.ToInt(),
			To:       tx.To,
			Value:    tx.Value.ToInt(),
			Data:     tx.Input,
			V:        tx.V.ToInt(),
			R:        tx.R.ToInt(),
			S:        tx.S.ToInt(),
		})
	case types.AccessListTxType:
		return types.NewTx(&types.AccessListTx{
			ChainID:    tx.ChainID.ToInt(),
			Nonce:      uint64(tx.Nonce),
			Gas:        uint64(tx.Gas),
			GasPrice:   tx.GasPrice.ToInt(),
			To:         tx.To,
			Value:      tx.Value.ToInt(),
			Data:       tx.Input,
			AccessList: *tx.Accesses,
			V:          tx.V.ToInt(),
			R:          tx.R.ToInt(),
			S:          tx.S.ToInt(),
		})
	case types.DynamicFeeTxType:
		return types.NewTx(&types.DynamicFeeTx{
			ChainID:    tx.ChainID.ToInt(),
			Nonce:      uint64(tx.Nonce),
			Gas:        uint64(tx.Gas),
			GasFeeCap:  tx.GasFeeCap.ToInt(),
			GasTipCap:  tx.GasTipCap.ToInt(),
			To:         tx.To,
			Value:      tx.Value.ToInt(),
			Data:       tx.Input,
			AccessList: *tx.Accesses,
			V:          tx.V.ToInt(),
			R:          tx.R.ToInt(),
			S:          tx.S.ToInt(),
		})
	case types.BlobTxType:
		return types.NewTx(&types.BlobTx{
			ChainID:    uint256.MustFromBig(tx.ChainID.ToInt()),
			Nonce:      uint64(tx.Nonce),
			Gas:        uint64(tx.Gas),
			GasFeeCap:  uint256.MustFromBig(tx.GasFeeCap.ToInt()),
			GasTipCap:  uint256.MustFromBig(tx.GasTipCap.ToInt()),
			To:         *tx.To,
			Value:      uint256.MustFromBig(tx.Value.ToInt()),
			Data:       tx.Input,
			AccessList: *tx.Accesses,
			BlobFeeCap: uint256.MustFromBig(tx.MaxFeePerBlobGas.ToInt()),
			BlobHashes: tx.BlobVersionedHashes,
			V:          uint256.MustFromBig(tx.V.ToInt()),
			R:          uint256.MustFromBig(tx.R.ToInt()),
			S:          uint256.MustFromBig(tx.S.ToInt()),
		})

	case types.SetCodeTxType:
		return types.NewTx(&types.SetCodeTx{
			ChainID:    uint256.MustFromBig(tx.ChainID.ToInt()),
			Nonce:      uint64(tx.Nonce),
			Gas:        uint64(tx.Gas),
			GasFeeCap:  uint256.MustFromBig(tx.GasFeeCap.ToInt()),
			GasTipCap:  uint256.MustFromBig(tx.GasTipCap.ToInt()),
			To:         *tx.To,
			Value:      uint256.MustFromBig(tx.Value.ToInt()),
			Data:       tx.Input,
			AccessList: *tx.Accesses,
			AuthList:   tx.AuthorizationList,
			V:          uint256.MustFromBig(tx.V.ToInt()),
			R:          uint256.MustFromBig(tx.R.ToInt()),
			S:          uint256.MustFromBig(tx.S.ToInt()),
		})
	default:
		t.Error("unsupported transaction type ", tx.Type)
		return nil
	}
}
