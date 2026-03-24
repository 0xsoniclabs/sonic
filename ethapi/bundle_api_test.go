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

package ethapi

import (
	"encoding/json"
	"math/big"
	reflect "reflect"
	"slices"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestBundleEstimateGas_PreArgsAreConsideredForEveryTransaction(t *testing.T) {
	ctrl := gomock.NewController(t)
	estimator := NewMockGasEstimator(ctrl)

	gasLimit := hexutil.Uint64(21000)
	numTransactions := 3

	for i := range numTransactions {
		estimator.EXPECT().EstimateGas(
			gomock.Any(),
			gomock.Any(),
		).DoAndReturn(func(args TransactionArgs, preArgs []TransactionArgs) (hexutil.Uint64, error) {
			require.Len(t, preArgs, i, "unexpected number of preArgs for transaction %d", i)
			return gasLimit, nil
		})
	}

	txArg := TransactionArgs{
		From: &common.Address{0x1},
		To:   &common.Address{0x2},
	}

	args := slices.Repeat([]TransactionArgs{txArg}, numTransactions)

	gasLimits, err := doEstimateGasForTransactions(args, estimator)
	require.NoError(t, err)
	bundleGasLimit := gasLimit + hexutil.Uint64(params.TxAccessListAddressGas) +
		hexutil.Uint64(params.TxAccessListStorageKeyGas)
	require.Equal(t, slices.Repeat([]hexutil.Uint64{bundleGasLimit}, numTransactions), gasLimits)
}

func TestBundleRPC_JsonIsEthereumConformant(t *testing.T) {
	// This test ensures that the JSON encoding of the RPC arguments and results
	// conforms to the Ethereum JSON-RPC specification

	blockNum := rpc.BlockNumber(123)
	hexUint := hexutil.Uint(10)
	hexUint64 := hexutil.Uint64(11)
	big := hexutil.Big(*big.NewInt(12))
	address := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")
	hash := common.HexToHash("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")

	expectJsonEqual(t, `{
		  "block":"0x7b",
		  "position":"0xa",
		  "count":"0xa"
	}`,
		RPCBundleInfo{
			Block:    &blockNum,
			Position: &hexUint,
			Count:    &hexUint,
		})

	plan := RPCExecutionPlan{
		Flags: 3,
		Steps: []RPCExecutionStep{
			{
				From: address,
				Hash: hash,
			},
		},
		Earliest: blockNum,
		Latest:   blockNum,
	}
	expectJsonEqual(t, `{
		  "flags": 3,
		  "steps": [
			{
		  		"from":"0x1234567890abcdef1234567890abcdef12345678",
				"hash":"0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
			}
		  ],
		  "earliest":"0x7b",
		  "latest":"0x7b"
	}`, plan)

	transacton := TransactionArgs{
		From:     &address,
		To:       &address,
		Nonce:    &hexUint64,
		Gas:      &hexUint64,
		GasPrice: &big,
		Value:    &big,
		Data:     (*hexutil.Bytes)(&[]byte{0x1, 0x2, 0x3}),
	}
	expectJsonEqual(t, `{
		"transactions":[
			{
				"from":"0x1234567890abcdef1234567890abcdef12345678",
				"to":"0x1234567890abcdef1234567890abcdef12345678",
				"gas":"0xb",
				"gasPrice":"0xc",
				"value":"0xc",
				"nonce":"0xb",
				"data":"0x010203"
			}
		],
		"plan": {
		  "flags": 3,
		  "steps": [
			{
		  		"from":"0x1234567890abcdef1234567890abcdef12345678",
				"hash":"0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
			}
		  ],
		  "earliest":"0x7b",
		  "latest":"0x7b"
		}
	}`,
		RPCPreparedBundle{
			Transactions: []TransactionArgs{transacton},
			Plan:         plan,
		})

}

func expectJsonEqual[T any](t testing.TB, expected string, value T) {
	t.Helper()
	encoded, err := json.Marshal(value)
	require.NoError(t, err, "failed to marshal BundleRPCInfo to JSON")

	var j1, j2 T
	err = json.Unmarshal(encoded, &j1)
	require.NoError(t, err, "failed to unmarshal JSON back to %T", value)
	err = json.Unmarshal([]byte(expected), &j2)
	require.NoError(t, err, "failed to unmarshal JSON back to %T", value)
	v := reflect.DeepEqual(j1, j2)
	if !v {
		expected = strings.ReplaceAll(expected, " ", "")
		expected = strings.ReplaceAll(expected, "\n", "")
		expected = strings.ReplaceAll(expected, "\t", "")
		t.Logf("Expected JSON: %s", expected)
		t.Logf("Actual JSON:   %s", string(encoded))
		t.FailNow()
	}
}
