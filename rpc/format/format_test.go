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
	"crypto/ecdsa"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

var chainID = big.NewInt(4003)

func sign(t *testing.T, key *ecdsa.PrivateKey, transaction types.TxData) *types.Transaction {
	t.Helper()
	return types.MustSignNewTx(key, types.LatestSignerForChainID(chainID), transaction)
}

func TestNewTransaction_FormsEachType(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	sender := crypto.PubkeyToAddress(key.PublicKey)
	to := common.Address{0x42}
	position := Position{BlockHash: common.Hash{0x0b}, BlockNumber: 7, Index: 2, BaseFee: big.NewInt(10)}
	cases := map[string]struct {
		transaction types.TxData
		gasPrice    int64
		fields      []string // fields present beyond those of every type
		absent      []string
	}{
		"legacy": {
			transaction: &types.LegacyTx{Gas: 21000, GasPrice: big.NewInt(30), To: &to},
			gasPrice:    30,
			absent:      []string{"accessList", "maxFeePerGas", "authorizationList"},
		},
		"access list": {
			transaction: &types.AccessListTx{ChainID: chainID, Gas: 21000, GasPrice: big.NewInt(30), To: &to},
			gasPrice:    30,
			fields:      []string{"accessList"},
			absent:      []string{"maxFeePerGas"},
		},
		"dynamic fee below its cap": {
			transaction: &types.DynamicFeeTx{ChainID: chainID, Gas: 21000, GasFeeCap: big.NewInt(30), GasTipCap: big.NewInt(5), To: &to},
			gasPrice:    15,
			fields:      []string{"accessList", "maxFeePerGas", "maxPriorityFeePerGas"},
		},
		"dynamic fee at its cap": {
			transaction: &types.DynamicFeeTx{ChainID: chainID, Gas: 21000, GasFeeCap: big.NewInt(12), GasTipCap: big.NewInt(5), To: &to},
			gasPrice:    12,
		},
		"blob": {
			transaction: &types.BlobTx{ChainID: uint256.NewInt(4003), Gas: 21000, GasFeeCap: uint256.NewInt(30), GasTipCap: uint256.NewInt(5), BlobFeeCap: uint256.NewInt(1), To: to},
			gasPrice:    15,
			fields:      []string{"maxFeePerBlobGas", "blobVersionedHashes"},
		},
		"set code": {
			transaction: &types.SetCodeTx{ChainID: uint256.NewInt(4003), Gas: 21000, GasFeeCap: uint256.NewInt(30), GasTipCap: uint256.NewInt(5), To: to,
				AuthList: []types.SetCodeAuthorization{{Address: to}}},
			gasPrice: 15,
			fields:   []string{"authorizationList"},
		},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			transaction := sign(t, key, test.transaction)
			form := NewTransaction(transaction, position, chainID)
			require.Equal(t, sender, form.From)
			require.Equal(t, test.gasPrice, form.GasPrice.ToInt().Int64())
			require.Equal(t, hexutil.Uint64(2), *form.TransactionIndex)
			require.Equal(t, int64(7), form.BlockNumber.ToInt().Int64())
			fields := encodedFields(t, form)
			for _, field := range test.fields {
				require.Contains(t, fields, field)
			}
			for _, field := range test.absent {
				require.NotContains(t, fields, field)
			}
		})
	}
}

func TestNewPendingTransaction_HasNoPositionAndPaysItsFeeCap(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	transaction := sign(t, key, &types.DynamicFeeTx{ChainID: chainID, Gas: 21000, GasFeeCap: big.NewInt(30), GasTipCap: big.NewInt(5)})
	form := NewPendingTransaction(transaction, chainID)
	require.Nil(t, form.BlockHash)
	require.Nil(t, form.BlockNumber)
	require.Nil(t, form.TransactionIndex)
	require.Equal(t, int64(30), form.GasPrice.ToInt().Int64())
}

func TestNewTransaction_TakesTheChainIdOfTheNetworkForAnInternalTransaction(t *testing.T) {
	internal := types.NewTx(&types.LegacyTx{Gas: 21000, GasPrice: new(big.Int), V: new(big.Int), R: new(big.Int), S: new(big.Int)})
	form := NewPendingTransaction(internal, chainID)
	require.Equal(t, common.Address{}, form.From)
	require.Equal(t, chainID, form.ChainID.ToInt())
}

func TestNewTransaction_ReadsAnUnprotectedTransactionWithTheHomesteadSigner(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	transaction := types.MustSignNewTx(key, types.HomesteadSigner{}, &types.LegacyTx{Gas: 21000, GasPrice: big.NewInt(1)})
	require.Equal(t, crypto.PubkeyToAddress(key.PublicKey), NewPendingTransaction(transaction, chainID).From)
}

func encodedFields(t *testing.T, value any) map[string]json.RawMessage {
	t.Helper()
	encoded, err := json.Marshal(value)
	require.NoError(t, err)
	var fields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(encoded, &fields))
	return fields
}

func TestBlock_FormsTransactionsAsHashesOrInFull(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	transaction := sign(t, key, &types.LegacyTx{Gas: 21000, GasPrice: big.NewInt(30)})
	block := evmcore.NewEvmBlock(&evmcore.EvmHeader{Number: big.NewInt(3), Hash: common.Hash{0x03}, BaseFee: big.NewInt(1)}, types.Transactions{transaction})
	receipts := types.Receipts{{Status: 1, Logs: []*types.Log{}}}

	hashes := Block(block, receipts, false, chainID)
	require.Equal(t, []any{transaction.Hash()}, hashes.Txs)
	require.Equal(t, []common.Hash{}, hashes.Uncles)
	require.Equal(t, types.MergeBloom(receipts), hashes.Bloom)

	full := Block(block, receipts, true, chainID)
	require.Equal(t, common.Hash{0x03}, *full.Txs[0].(*Transaction).BlockHash)
}

func TestReceipt_PricesAndWordsAsBlockProcessing(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	to := common.Address{0x42}
	header := &evmcore.EvmHeader{Number: big.NewInt(3), Hash: common.Hash{0x03}, BaseFee: big.NewInt(10)}
	internal := types.NewTx(&types.LegacyTx{To: &to, Gas: 21000, GasPrice: big.NewInt(30), V: new(big.Int), R: new(big.Int), S: big.NewInt(1)})
	cases := map[string]struct {
		transaction *types.Transaction
		header      *evmcore.EvmHeader
		price       hexutil.Uint64
	}{
		"dynamic fee":      {sign(t, key, &types.DynamicFeeTx{ChainID: chainID, To: &to, Gas: 21000, GasFeeCap: big.NewInt(30), GasTipCap: big.NewInt(5)}), header, 15},
		"sponsored":        {sign(t, key, &types.LegacyTx{To: &to, Gas: 21000, GasPrice: new(big.Int)}), header, 0},
		"internal":         {internal, header, 0},
		"without base fee": {sign(t, key, &types.LegacyTx{To: &to, Gas: 21000, GasPrice: big.NewInt(30)}), &evmcore.EvmHeader{Number: big.NewInt(3)}, 30},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			fields := Receipt(test.header, test.transaction, 0, &types.Receipt{Status: 1}, chainID)
			require.Equal(t, test.price, fields["effectiveGasPrice"])
			require.Equal(t, [][]*types.Log{}, fields["logs"], "a receipt without logs answers an empty list")
			require.Equal(t, hexutil.Uint(1), fields["status"])
			require.Nil(t, fields["contractAddress"])
		})
	}
}

func TestReceipt_CopiesSharedLogsAndNamesTheirTransaction(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	transaction := sign(t, key, &types.LegacyTx{Gas: 53000, GasPrice: big.NewInt(30)})
	shared := &types.Log{Address: common.Address{0x01}}
	receipt := &types.Receipt{PostState: []byte{0x01}, Logs: []*types.Log{shared}, ContractAddress: common.Address{0x09}}
	header := &evmcore.EvmHeader{Number: big.NewInt(3), Hash: common.Hash{0x03}, BaseFee: big.NewInt(10)}

	fields := Receipt(header, transaction, 4, receipt, chainID)
	logs := fields["logs"].([]*types.Log)
	require.Equal(t, transaction.Hash(), logs[0].TxHash)
	require.Equal(t, uint(4), logs[0].TxIndex)
	require.Equal(t, []common.Hash{}, logs[0].Topics)
	require.Equal(t, types.Log{Address: common.Address{0x01}}, *shared, "the shared log is unchanged")
	require.Equal(t, hexutil.Bytes{0x01}, fields["root"])
	require.NotContains(t, fields, "status")
	require.Equal(t, common.Address{0x09}, fields["contractAddress"])
}
