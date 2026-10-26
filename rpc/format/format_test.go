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

package format_test

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/rpc/format"
)

var (
	chainID  = big.NewInt(4003)
	signer   = types.LatestSignerForChainID(chainID)
	to       = common.HexToAddress("0x42")
	baseFee  = big.NewInt(100)
	blockOne = common.HexToHash("0xb1")
)

func sign(t *testing.T, data types.TxData) (*types.Transaction, common.Address) {
	t.Helper()
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	tx, err := types.SignNewTx(key, signer, data)
	require.NoError(t, err)
	return tx, crypto.PubkeyToAddress(key.PublicKey)
}

func TestTransaction_EveryType(t *testing.T) {
	list := types.AccessList{{Address: to, StorageKeys: []common.Hash{{1}}}}
	cases := map[string]struct {
		data    types.TxData
		price   int64 // in a block
		dynamic bool
		blob    bool
		setCode bool
	}{
		"legacy":      {data: &types.LegacyTx{To: &to, Gas: 21000, GasPrice: big.NewInt(500)}, price: 500},
		"access list": {data: &types.AccessListTx{ChainID: chainID, To: &to, Gas: 21000, GasPrice: big.NewInt(500), AccessList: list}, price: 500},
		"dynamic fee": {data: &types.DynamicFeeTx{ChainID: chainID, To: &to, Gas: 21000, GasFeeCap: big.NewInt(1000), GasTipCap: big.NewInt(7), AccessList: list}, price: 107, dynamic: true},
		"capped":      {data: &types.DynamicFeeTx{ChainID: chainID, To: &to, Gas: 21000, GasFeeCap: big.NewInt(103), GasTipCap: big.NewInt(7)}, price: 103, dynamic: true},
		"blob":        {data: &types.BlobTx{ChainID: uint256.MustFromBig(chainID), To: to, Gas: 21000, GasFeeCap: uint256.NewInt(1000), GasTipCap: uint256.NewInt(7), BlobFeeCap: uint256.NewInt(9)}, price: 107, dynamic: true, blob: true},
		"set code":    {data: &types.SetCodeTx{ChainID: uint256.MustFromBig(chainID), To: to, Gas: 50000, GasFeeCap: uint256.NewInt(1000), GasTipCap: uint256.NewInt(7), AuthList: []types.SetCodeAuthorization{{Address: to}}}, price: 107, dynamic: true, setCode: true},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			tx, from := sign(t, tc.data)
			mined := format.NewTransaction(tx, blockOne, 1, 3, baseFee, chainID)
			require.Equal(t, from, mined.From)
			require.Equal(t, tx.Hash(), mined.Hash)
			require.Equal(t, blockOne, *mined.BlockHash)
			require.Equal(t, int64(1), mined.BlockNumber.ToInt().Int64())
			require.Equal(t, hexutil.Uint64(3), *mined.TransactionIndex)
			require.Equal(t, tc.price, mined.GasPrice.ToInt().Int64())
			require.Equal(t, chainID, mined.ChainID.ToInt())
			require.Equal(t, tc.dynamic, mined.GasFeeCap != nil)
			require.Equal(t, tc.blob, mined.MaxFeePerBlobGas != nil)
			if tc.blob {
				require.NotNil(t, mined.BlobVersionedHashes, "never null")
			}
			require.Equal(t, tc.setCode, len(mined.AuthorizationList) > 0)

			pending := format.NewPendingTransaction(tx, baseFee, chainID)
			require.Nil(t, pending.BlockHash)
			require.Nil(t, pending.TransactionIndex)
			if tc.dynamic {
				require.Equal(t, tx.GasFeeCap(), pending.GasPrice.ToInt(), "a pending transaction pays its fee cap")
			}
		})
	}
}

func TestTransaction_UnprotectedAndInternal(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	tx, err := types.SignNewTx(key, types.HomesteadSigner{}, &types.LegacyTx{To: &to, Gas: 21000, GasPrice: big.NewInt(1)})
	require.NoError(t, err)
	res := format.NewPendingTransaction(tx, nil, chainID)
	require.Equal(t, crypto.PubkeyToAddress(key.PublicKey), res.From)

	internal := types.NewTx(&types.LegacyTx{To: &to, Gas: 21000, GasPrice: big.NewInt(0), V: new(big.Int), R: new(big.Int), S: new(big.Int).SetBytes(common.Address{7}.Bytes())})
	res = format.NewPendingTransaction(internal, nil, chainID)
	require.Equal(t, common.Address{7}, res.From, "the sender of an internal transaction is in its S")

	// The node's own transactions come from the zero address and carry the
	// chain's id, as the reference answered them.
	system := types.NewTx(&types.LegacyTx{To: &to, Gas: 21000, GasPrice: big.NewInt(0), V: new(big.Int), R: new(big.Int), S: new(big.Int)})
	res = format.NewPendingTransaction(system, nil, chainID)
	require.Equal(t, common.Address{}, res.From)
	require.Equal(t, chainID, res.ChainID.ToInt())
}

func block(t *testing.T, txs types.Transactions) *evmcore.EvmBlock {
	t.Helper()
	header := &evmcore.EvmHeader{Number: big.NewInt(1), Hash: blockOne, BaseFee: baseFee, Time: inter.FromUnix(5), GasLimit: 1 << 30}
	return evmcore.NewEvmBlock(header, txs)
}

func TestBlockAndTransactionAt(t *testing.T) {
	tx, _ := sign(t, &types.DynamicFeeTx{ChainID: chainID, To: &to, Gas: 21000, GasFeeCap: big.NewInt(1000)})
	b := block(t, types.Transactions{tx})
	enc, err := json.Marshal(format.Block(b, nil, false, chainID))
	require.NoError(t, err)
	var hashes struct {
		Transactions []common.Hash `json:"transactions"`
		Uncles       []common.Hash `json:"uncles"`
		Size         hexutil.Uint64
	}
	require.NoError(t, json.Unmarshal(enc, &hashes))
	require.Equal(t, []common.Hash{tx.Hash()}, hashes.Transactions)
	require.NotNil(t, hashes.Uncles)

	enc, err = json.Marshal(format.Block(b, nil, true, chainID))
	require.NoError(t, err)
	var full struct {
		Transactions []format.Transaction `json:"transactions"`
	}
	require.NoError(t, json.Unmarshal(enc, &full))
	require.Equal(t, tx.Hash(), full.Transactions[0].Hash)

	require.Nil(t, format.TransactionAt(b, 1, chainID))
	require.Nil(t, format.RawTransactionAt(b, 1))
	raw, err := tx.MarshalBinary()
	require.NoError(t, err)
	require.Equal(t, hexutil.Bytes(raw), format.RawTransactionAt(b, 0))
}

func TestReceipt(t *testing.T) {
	tx, from := sign(t, &types.DynamicFeeTx{ChainID: chainID, To: &to, Gas: 21000, GasFeeCap: big.NewInt(1000), GasTipCap: big.NewInt(7)})
	b := block(t, types.Transactions{tx})
	shared := &types.Log{Address: to}
	receipt := &types.Receipt{Status: 1, GasUsed: 21000, CumulativeGasUsed: 21000, Logs: []*types.Log{shared}}
	fields := format.Receipt(b.Header(), tx, 0, receipt, signer)
	require.Equal(t, from, fields["from"])
	require.Equal(t, hexutil.Uint64(107), fields["effectiveGasPrice"])
	require.Equal(t, hexutil.Uint(1), fields["status"])
	require.Nil(t, fields["contractAddress"])
	logs := fields["logs"].([]*types.Log)
	require.Equal(t, []common.Hash{}, logs[0].Topics)
	require.Equal(t, blockOne, logs[0].BlockHash)
	require.Nil(t, shared.Topics, "the shared log is unchanged")
	require.Equal(t, common.Hash{}, shared.BlockHash)

	creation, _ := sign(t, &types.LegacyTx{Gas: 50000, GasPrice: big.NewInt(300), Data: []byte{1}})
	created := common.Address{9}
	fields = format.Receipt(b.Header(), creation, 0, &types.Receipt{PostState: []byte{1}, ContractAddress: created}, signer)
	require.Equal(t, created, fields["contractAddress"])
	require.Equal(t, hexutil.Bytes{1}, fields["root"])
	require.Equal(t, [][]*types.Log{}, fields["logs"])
	require.Equal(t, hexutil.Uint64(300), fields["effectiveGasPrice"])

	sponsored, _ := sign(t, &types.LegacyTx{To: &to, Gas: 21000, GasPrice: big.NewInt(0)})
	fields = format.Receipt(b.Header(), sponsored, 0, &types.Receipt{}, signer)
	require.Equal(t, hexutil.Uint64(0), fields["effectiveGasPrice"], "a sponsorship request pays nothing")

	noBaseFee := &evmcore.EvmHeader{Number: big.NewInt(1), Hash: blockOne}
	fields = format.Receipt(noBaseFee, creation, 0, &types.Receipt{}, signer)
	require.Equal(t, hexutil.Uint64(300), fields["effectiveGasPrice"])
}

func TestLogsAndHashesAreNeverNull(t *testing.T) {
	shared := &types.Log{Address: to}
	out := format.Logs([]*types.Log{shared, {Topics: []common.Hash{{1}}}})
	require.Equal(t, []common.Hash{}, out[0].Topics)
	require.Nil(t, shared.Topics)
	require.NotSame(t, shared, out[0])
	require.Equal(t, []*types.Log{}, format.Logs(nil))
	require.Equal(t, []common.Hash{}, format.Hashes(nil))
	require.Equal(t, []common.Hash{{1}}, format.Hashes([]common.Hash{{1}}))
}
