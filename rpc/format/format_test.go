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

package format

import (
	"encoding/json"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/evmcore"
)

var chainID = big.NewInt(4003)

func sign(t *testing.T, data types.TxData) *types.Transaction {
	t.Helper()
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	signed, err := types.SignNewTx(key, types.LatestSignerForChainID(chainID), data)
	require.NoError(t, err)
	return signed
}

func encode(t *testing.T, value any) map[string]any {
	t.Helper()
	encoded, err := json.Marshal(value)
	require.NoError(t, err)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(encoded, &decoded))
	return decoded
}

func TestTransaction_HasTheFieldsOfItsType(t *testing.T) {
	recipient := common.HexToAddress("0x1")
	located := Location{BlockHash: common.Hash{1}, BlockNumber: 3, Index: 2}
	baseFee := big.NewInt(100)
	tests := map[string]struct {
		data   types.TxData
		fields []string
		absent []string
		price  string
	}{
		"legacy": {
			data:   &types.LegacyTx{To: &recipient, Gas: 21000, GasPrice: big.NewInt(150)},
			absent: []string{"accessList", "maxFeePerGas", "authorizationList"},
			price:  "0x96",
		},
		"access list": {
			data:   &types.AccessListTx{ChainID: chainID, To: &recipient, Gas: 21000, GasPrice: big.NewInt(150)},
			fields: []string{"accessList"},
			absent: []string{"maxFeePerGas"},
			price:  "0x96",
		},
		"dynamic fee": {
			data:   &types.DynamicFeeTx{ChainID: chainID, To: &recipient, Gas: 21000, GasTipCap: big.NewInt(10), GasFeeCap: big.NewInt(200)},
			fields: []string{"accessList", "maxFeePerGas", "maxPriorityFeePerGas"},
			price:  "0x6e",
		},
		"blob": {
			data: &types.BlobTx{ChainID: uint256.MustFromBig(chainID), To: recipient, Gas: 21000,
				GasTipCap: uint256.NewInt(10), GasFeeCap: uint256.NewInt(105), BlobFeeCap: uint256.NewInt(1)},
			fields: []string{"accessList", "maxFeePerGas", "maxFeePerBlobGas"},
			price:  "0x69",
		},
		"set code": {
			data: &types.SetCodeTx{ChainID: uint256.MustFromBig(chainID), To: recipient, Gas: 21000,
				GasTipCap: uint256.NewInt(10), GasFeeCap: uint256.NewInt(200), AuthList: []types.SetCodeAuthorization{{}}},
			fields: []string{"accessList", "maxFeePerGas", "authorizationList"},
			price:  "0x6e",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			transaction := sign(t, test.data)
			answer := encode(t, NewTransaction(transaction, located, baseFee, chainID))
			for _, field := range test.fields {
				require.Contains(t, answer, field)
			}
			for _, field := range test.absent {
				require.NotContains(t, answer, field)
			}
			require.Equal(t, test.price, answer["gasPrice"])
			require.Equal(t, "0x3", answer["blockNumber"])
			require.Equal(t, "0x2", answer["transactionIndex"])
			require.Equal(t, "0xfa3", answer["chainId"])
		})
	}
}

func TestTransaction_APendingDynamicFeeTransactionIsPricedAtItsCap(t *testing.T) {
	recipient := common.HexToAddress("0x1")
	transaction := sign(t, &types.DynamicFeeTx{ChainID: chainID, To: &recipient, GasTipCap: big.NewInt(10), GasFeeCap: big.NewInt(200)})
	answer := encode(t, PendingTransaction(transaction, big.NewInt(100), chainID))
	require.Equal(t, "0xc8", answer["gasPrice"])
	require.Nil(t, answer["blockHash"])
	require.Nil(t, answer["transactionIndex"])
}

func TestTransaction_AnInternalTransactionCarriesTheChainID(t *testing.T) {
	internal := types.NewTx(&types.LegacyTx{Nonce: 1})
	answer := encode(t, PendingTransaction(internal, nil, chainID))
	require.Equal(t, common.Address{}.Hex(), answer["from"])
	require.Equal(t, "0xfa3", answer["chainId"])
}

func TestTransactionAt_IsNilOutsideTheBlock(t *testing.T) {
	transaction := sign(t, &types.LegacyTx{Gas: 21000})
	block := evmcore.NewEvmBlock(&evmcore.EvmHeader{Number: big.NewInt(1), Hash: common.Hash{1}}, types.Transactions{transaction})
	require.NotNil(t, TransactionAt(block, 0, chainID))
	require.Nil(t, TransactionAt(block, 1, chainID))
	encoded, err := transaction.MarshalBinary()
	require.NoError(t, err)
	require.Equal(t, encoded, []byte(RawTransactionAt(block, 0)))
	require.Nil(t, RawTransactionAt(block, 1))
}

func TestBlock_ListsTransactionsAsHashesOrInFull(t *testing.T) {
	transaction := sign(t, &types.LegacyTx{Gas: 21000})
	block := evmcore.NewEvmBlock(&evmcore.EvmHeader{Number: big.NewInt(1), Hash: common.Hash{1}, BaseFee: big.NewInt(1)}, types.Transactions{transaction})
	hashes := encode(t, Block(block, types.Receipts{}, false, chainID))
	require.Equal(t, []any{transaction.Hash().Hex()}, hashes["transactions"])
	require.Equal(t, types.EmptyRootHash.Hex(), hashes["receiptsRoot"])
	full := encode(t, Block(block, nil, true, chainID))
	require.Equal(t, transaction.Hash().Hex(), full["transactions"].([]any)[0].(map[string]any)["hash"])
	require.Equal(t, []any{}, full["uncles"])
}

func TestReceipt_PricesAsBlockProcessingCharges(t *testing.T) {
	recipient := common.HexToAddress("0x1")
	header := &evmcore.EvmHeader{Number: big.NewInt(1), Hash: common.Hash{1}, BaseFee: big.NewInt(100)}
	receipt := &types.Receipt{Status: types.ReceiptStatusSuccessful, GasUsed: 21000}
	tests := map[string]struct {
		transaction *types.Transaction
		header      *evmcore.EvmHeader
		price       string
	}{
		"dynamic fee": {transaction: sign(t, &types.DynamicFeeTx{ChainID: chainID, To: &recipient, GasTipCap: big.NewInt(10), GasFeeCap: big.NewInt(200)}), header: header, price: "0x6e"},
		"sponsored":   {transaction: sign(t, &types.DynamicFeeTx{ChainID: chainID, To: &recipient}), header: header, price: "0x0"},
		"internal":    {transaction: types.NewTx(&types.LegacyTx{To: &recipient, GasPrice: big.NewInt(5)}), header: header, price: "0x0"},
		"without base fee": {
			transaction: sign(t, &types.LegacyTx{To: &recipient, GasPrice: big.NewInt(150)}),
			header:      &evmcore.EvmHeader{Number: big.NewInt(1), Hash: common.Hash{1}},
			price:       "0x96",
		},
	}
	for name, test := range tests {
		answer := encode(t, Receipt(test.header, test.transaction, 0, receipt, chainID))
		require.Equal(t, test.price, answer["effectiveGasPrice"], name)
		require.Equal(t, []any{}, answer["logs"], name)
		require.Equal(t, "0x1", answer["status"], name)
	}
}

func TestReceipt_OfAContractCreationNamesTheContract(t *testing.T) {
	header := &evmcore.EvmHeader{Number: big.NewInt(1), Hash: common.Hash{1}, BaseFee: big.NewInt(1)}
	creation := sign(t, &types.LegacyTx{GasPrice: big.NewInt(2), Data: []byte{1}})
	receipt := &types.Receipt{ContractAddress: common.HexToAddress("0xc"), PostState: []byte{1}, Logs: []*types.Log{{Address: common.HexToAddress("0xc")}}}
	answer := encode(t, Receipt(header, creation, 4, receipt, chainID))
	require.Equal(t, strings.ToLower(common.HexToAddress("0xc").Hex()), answer["contractAddress"])
	require.Equal(t, "0x01", answer["root"])
	require.NotContains(t, answer, "status")
	logs := answer["logs"].([]any)
	require.Equal(t, "0x4", logs[0].(map[string]any)["transactionIndex"])
	require.Equal(t, []any{}, logs[0].(map[string]any)["topics"])
	require.Equal(t, uint(0), receipt.Logs[0].TxIndex, "the shared log is not modified")
}
