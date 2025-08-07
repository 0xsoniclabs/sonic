// Copyright 2025 Sonic Operations Ltd
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

package tests

import (
	"fmt"
	"math/big"
	"slices"
	"testing"

	"github.com/0xsoniclabs/sonic/config"
	"github.com/0xsoniclabs/sonic/ethapi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestChainId(t *testing.T) {
	net := StartIntegrationTestNet(t,
		IntegrationTestNetOptions{
			ModifyConfig: func(config *config.Config) {
				// The transactions signed with the Homestead are not replay protected.
				// The default configuration rejects this sort of transaction,
				// so they need to be explicitly allowed.
				config.Opera.AllowUnprotectedTxs = true
			},
		})

	account := MakeAccountWithBalance(t, net, big.NewInt(1e18))

	t.Run("RejectsAllTxsSignedWithWrongChainId", func(t *testing.T) {
		t.Parallel()
		testChainId_RejectsAllTxSignedWithWrongChainId(t, net, account)
	})

	t.Run("AcceptsLegacyTxSignedWithHomestead", func(t *testing.T) {
		t.Parallel()
		testChainId_AcceptsLegacyTxSignedWithHomestead(t, net, account)
	})
}

func testChainId_RejectsAllTxSignedWithWrongChainId(
	t *testing.T,
	net *IntegrationTestNet,
	account *Account,
) {

	client, err := net.GetClient()
	require.NoError(t, err, "failed to get client")
	t.Cleanup(client.Close)

	actualChainID, err := client.ChainID(t.Context())
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
				t.Parallel()

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

func testChainId_AcceptsLegacyTxSignedWithHomestead(
	t *testing.T,
	net *IntegrationTestNet,
	account *Account) {

	client, err := net.GetClient()
	require.NoError(t, err, "failed to get client")
	t.Cleanup(client.Close)

	// get current nonce and sign the tx.
	nonce, err := client.NonceAt(t.Context(), account.Address(), nil)
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
	err = client.Client().CallContext(t.Context(), &json,
		"eth_getTransactionByHash", signed.Hash(),
	)
	require.NoError(t, err)
	require.Equal(t, signed.Hash(), json.Hash)
	// Since HomesteadSigner does not have a chain ID, the transaction should be
	// processed and stored with chain ID 0.
	require.Equal(t, int64(0), json.ChainID.ToInt().Int64())

	// reconstruct the transaction from the RPC response
	// and verify that it has the same hash and chain ID as the signed transaction
	decodedTx := rpcTransactionToTransaction(t, json)
	require.Equal(t, signed.Hash(), decodedTx.Hash())
	require.Equal(t, int64(0), decodedTx.ChainId().Int64())
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
