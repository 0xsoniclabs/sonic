package sonicapi

import (
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/0xsoniclabs/sonic/ethapi"
	testbackend "github.com/0xsoniclabs/sonic/rpc/test_backend"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func Test_TestBackend(t *testing.T) {
	address1 := common.HexToAddress("0xadd01")
	address2 := common.HexToAddress("0xadd02")
	key1, err := crypto.GenerateKey()
	require.NoError(t, err)
	hexUint64 := hexutil.Uint64(11)

	chain := testbackend.NewBlockchain()
	chain.SetAccount(address1, testbackend.AccountState{
		Nonce:   1,
		Balance: big.NewInt(1000),
	})
	signer := types.LatestSignerForChainID(chain.ChainID())

	api := NewPublicBundleAPI(chain)

	result, err := api.PrepareBundle(t.Context(),
		PrepareBundleArgs{
			Transactions: []ethapi.TransactionArgs{
				{
					From:  &address1,
					To:    &address2,
					Nonce: &hexUint64,
				},
			},
		})
	require.NoError(t, err)

	require.Len(t, result.Transactions, 1)

	readyTx := result.Transactions[0]
	require.NotNil(t, readyTx)
	require.Equal(t, address1, *readyTx.From)
	require.Equal(t, address2, *readyTx.To)
	require.Len(t, *readyTx.AccessList, 1)

	fmt.Printf("tx %#v\n", readyTx)
	fmt.Printf("acclist %#v\n", (*readyTx.AccessList)[0])
	fmt.Printf("plan %#v\n", result.Plan)

	signedTx, err := types.SignTx(readyTx.ToTransaction(), signer, key1)
	require.NoError(t, err)

	data, err := signedTx.MarshalBinary()
	require.NoError(t, err)
	_, err = api.SubmitBundle(t.Context(), SubmitBundleArgs{
		SignedTransactions: []hexutil.Bytes{hexutil.Bytes(data)},
		ExecutionPlan:      result.Plan,
	})
	require.NoError(t, err)
}

func TestBundleEstimateGas_PreArgsAreConsideredForEveryTransaction(t *testing.T) {
	ctrl := gomock.NewController(t)
	estimator := NewMockGasEstimator(ctrl)

	gasLimit := hexutil.Uint64(21000)
	numTransactions := 3

	for i := range numTransactions {
		estimator.EXPECT().EstimateGas(
			gomock.Any(),
			gomock.Any(),
		).DoAndReturn(func(args ethapi.TransactionArgs, preArgs []ethapi.TransactionArgs) (hexutil.Uint64, error) {
			require.Len(t, preArgs, i, "unexpected number of preArgs for transaction %d", i)
			return gasLimit, nil
		})
	}

	txArg := ethapi.TransactionArgs{
		From: &common.Address{0x1},
		To:   &common.Address{0x2},
	}

	args := slices.Repeat([]ethapi.TransactionArgs{txArg}, numTransactions)

	gasLimits, err := DoEstimateGasForTransactions(args, estimator)
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

	transacton := ethapi.TransactionArgs{
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
			Transactions: []ethapi.TransactionArgs{transacton},
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
