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

package rpc

import (
	"context"
	"encoding/json"
	"math/big"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/tests"
)

// matrixChain is the chain the matrix runs on: real state, a block with a
// transfer and a log, and a transaction in the pool.
type matrixChain struct {
	core     *tests.CarmenCore
	harness  *tests.Harness
	block    common.Hash
	mined    common.Hash
	pending  common.Hash
	raw      hexutil.Bytes
	sender   common.Address
	contract common.Address
}

func newMatrixChain(t *testing.T) matrixChain {
	t.Helper()
	signer := tests.NewSigner(t)
	contract := common.HexToAddress("0xc0")
	carmenCore := tests.NewCarmenCore(t, map[common.Address]tests.Account{
		signer.Address: {Balance: new(big.Int).Lsh(big.NewInt(1), 80)},
		contract:       {Code: []byte{0x60, 0x01, 0x60, 0x00, 0x55, 0x00}},
	})
	mined := signer.Sign(t, &types.LegacyTx{Nonce: 0, To: &contract, Gas: 50000, GasPrice: big.NewInt(2_000_000_000)})
	log := &types.Log{Address: contract, Topics: []common.Hash{{7}}}
	block := carmenCore.AppendAndPublish(tests.BlockSpec{
		Transactions: types.Transactions{mined},
		Receipts:     types.Receipts{{Status: types.ReceiptStatusSuccessful, GasUsed: 30000, Logs: []*types.Log{log}}},
	})
	pending := signer.Sign(t, &types.LegacyTx{Nonce: 1, To: &contract, Gas: 50000, GasPrice: big.NewInt(2_000_000_000)})
	carmenCore.SetPool(map[common.Address]types.Transactions{signer.Address: {pending}}, nil)
	harness := tests.NewHarness(t, carmenCore, func(dispatcher *dispatch.Dispatcher, environment *exec.Environment) {
		registerNamespaces(dispatcher, environment, carmenCore, DefaultConfig())
	})
	harness.WaitForHead(block.NumberU64())
	raw := tests.Encode(t, signer.Sign(t, &types.LegacyTx{Nonce: 2, To: &contract, Gas: 50000, GasPrice: big.NewInt(2_000_000_000)}))
	return matrixChain{core: carmenCore, harness: harness, block: block.Hash, mined: mined.Hash(), pending: pending.Hash(), raw: raw, sender: signer.Address, contract: contract}
}

// requests are, for each method, the parameter arrays the matrix sends: a
// valid request, and the absent and null shapes of its optional parameters.
func (chain matrixChain) requests() map[string][]string {
	quote := func(value any) string {
		encoded, _ := json.Marshal(value)
		return string(encoded)
	}
	block, mined, pending := quote(chain.block), quote(chain.mined), quote(chain.pending)
	byNumber := []string{`["0x1", true]`, `["latest", false]`, `["0x5", true]`}
	byHash := []string{`[` + block + `, true]`, `[` + mined + `, false]`}
	numberIndex := []string{`["0x1", "0x0"]`, `["latest", "0x5"]`, `["0x5", "0x0"]`}
	hashIndex := []string{`[` + block + `, "0x0"]`, `[` + mined + `, "0x0"]`}
	number := []string{`["0x1"]`, `["latest"]`, `["earliest"]`, `["0x5"]`}
	hash := []string{`[` + block + `]`, `[` + mined + `]`}
	transaction := []string{`[` + mined + `]`, `[` + pending + `]`, `[` + block + `]`}
	sender, contract, raw := quote(chain.sender), quote(chain.contract), quote(chain.raw)
	account := []string{`[` + sender + `]`, `[` + sender + `, "latest"]`, `[` + sender + `, null]`, `[` + sender + `, ` + block + `]`, `[` + sender + `, "0x5"]`}
	message := `{"from": ` + sender + `, "to": ` + contract + `}`
	priced := `{"from": ` + sender + `, "to": ` + contract + `, "maxFeePerGas": "0x3b9aca00"}`
	created := `{"from": ` + sender + `, "input": "0x6001600055"}`
	calls := []string{
		`[` + message + `]`, `[` + message + `, "latest", null, null]`, `[` + priced + `, ` + block + `]`,
		`[` + message + `, "0x5"]`, `[` + message + `, "latest", {` + contract + `: {"balance": "0x1"}}, {"number": "0x10"}]`,
	}
	requests := map[string][]string{
		"chainId": {`[]`}, "blockNumber": {`[]`}, "syncing": {`[]`}, "currentEpoch": {`[]`}, "config": {`[]`},
		"gasPrice": {`[]`}, "maxPriorityFeePerGas": {`[]`}, "blobBaseFee": {`[]`},
		"feeHistory": {`["0x2", "latest", [10, 90]]`, `["0x1", "0x5", null]`, `["0x0", "latest", null]`},
		"etherbase":  {`[]`}, "coinbase": {`[]`}, "hashrate": {`[]`},
		"getRules":                               number,
		"getBlockByNumber":                       byNumber,
		"getBlockByHash":                         byHash,
		"getHeaderByNumber":                      number,
		"getHeaderByHash":                        hash,
		"getBlockReceipts":                       {`["0x1"]`, `[` + block + `]`, `[{}]`, `["0x5"]`},
		"getBlockTransactionCountByNumber":       number,
		"getBlockTransactionCountByHash":         hash,
		"getUncleByBlockNumberAndIndex":          numberIndex,
		"getUncleByBlockHashAndIndex":            hashIndex,
		"getUncleCountByBlockNumber":             number,
		"getUncleCountByBlockHash":               hash,
		"getTransactionByBlockNumberAndIndex":    numberIndex,
		"getTransactionByBlockHashAndIndex":      hashIndex,
		"getRawTransactionByBlockNumberAndIndex": numberIndex,
		"getRawTransactionByBlockHashAndIndex":   hashIndex,
		"getTransactionByHash":                   transaction,
		"getRawTransactionByHash":                transaction,
		"getTransactionReceipt":                  transaction,
		"getBalance":                             account,
		"getCode":                                account,
		"getTransactionCount":                    append(account, `[`+sender+`, "pending"]`),
		"getAccount":                             account,
		"getStorageAt":                           {`[` + contract + `, "0x0"]`, `[` + contract + `, "0x0", "latest"]`, `[` + contract + `, "0x0", "0x5"]`},
		"getProof":                               {`[` + contract + `, ["0x0"]]`, `[` + contract + `, [], "latest"]`, `[` + contract + `, ["0x0"], "0x5"]`},
		"call":                                   calls,
		"estimateGas":                            append(calls, `[`+created+`]`),
		"simulateV1": {
			`[{"blockStateCalls": [{"calls": [` + message + `]}, {}]}]`, `[{"blockStateCalls": [{"calls": [` + created + `]}]}, "latest"]`,
			`[{"blockStateCalls": []}]`, `[{"blockStateCalls": [{}]}, "0x5"]`, `[{"blockStateCalls": [{"calls": [` + message + `]}], "returnFullTransactions": true}, ` + block + `]`,
		},
		"sendRawTransaction":          {`[` + raw + `]`, `["0x01"]`},
		"sendRawTransactionSync":      {`[` + raw + `, "0x1"]`, `["0x01"]`},
		"pendingTransactions":         {`[]`},
		"getLogs":                     {`[{"fromBlock": "0x1", "address": ` + contract + `}]`, `[{"blockHash": ` + block + `}]`, `[{}]`, `[{"fromBlock": "0x0", "toBlock": "0x5"}]`, `[{"blockHash": ` + mined + `}]`},
		"newFilter":                   {`[{"address": ` + contract + `}]`, `[{"fromBlock": "pending", "toBlock": "pending"}]`, `[{"fromBlock": "0x5", "toBlock": "0x1"}]`},
		"newBlockFilter":              {`[]`},
		"newPendingTransactionFilter": {`[]`, `[true]`, `[null]`},
		"getFilterChanges":            {`["0x1"]`},
		"getFilterLogs":               {`["0x1"]`},
		"uninstallFilter":             {`["0x1"]`},
		"createAccessList":            {`[` + message + `]`, `[` + priced + `, "latest"]`, `[` + created + `]`, `[` + message + `, "0x5"]`},
	}
	proposal := `{"steps": [{"from": ` + sender + `, "to": ` + contract + `, "nonce": "0x1"}]}`
	all := map[string][]string{
		"txpool_content":      {`[]`},
		"txpool_contentFrom":  {`[` + sender + `]`},
		"txpool_status":       {`[]`},
		"txpool_inspect":      {`[]`},
		"sonic_prepareBundle": {`[` + proposal + `]`, `[{"blockRange": {"first": "0x9", "length": "0x2"}, "steps": [{"from": ` + sender + `, "to": ` + contract + `, "nonce": "0x1", "gas": "0x5208"}]}]`},
		"sonic_submitBundle":  {`[{"signedTransactions": [` + raw + `], "executionPlan": {"blockRange": {"first": "0x9", "length": "0x2"}, "steps": [{"from": ` + sender + `, "hash": ` + pending + `}]}}]`, `[{"signedTransactions": []}]`},
		"sonic_getBundleInfo": {`[` + mined + `]`},
		"sonic_estimateGasForTransactions": {
			`[[` + message + `, ` + message + `]]`, `[[` + message + `], "latest", null, null]`, `[[` + created + `], "0x5"]`, `[[]]`,
		},
		"net_version":        {`[]`},
		"web3_clientVersion": {`[]`},
		"web3_sha3":          {`["0x01"]`},
	}
	for name, shapes := range requests {
		all["eth_"+name] = shapes
		all["ftm_"+name] = shapes
	}
	return all
}

// oracleCachePeriod is a little more than the period the oracle caches a
// suggestion for.
const oracleCachePeriod = 1100 * time.Millisecond

// unreached are the methods of Core no request reaches, each with the reason.
var unreached = map[string]string{
	"ChainInfo":             "read by the head loop only; requests read its last answer",
	"Head":                  "read at the start and by the head loop only",
	"SubscribeHead":         "the head loop's subscription",
	"SubscribeTransactions": "the pool loop's subscription",
	"PendingHeads":          "read by the oracle's sampler loop only",
}

// TestMatrix_EveryCoreFailureIsContained holds Q3 and P1: whatever a Core
// method answers, every request is answered, a panic is recovered, answered
// -32603 and counted, and nothing else panics.
func TestMatrix_EveryCoreFailureIsContained(t *testing.T) {
	chain := newMatrixChain(t)
	requests := chain.requests()
	for _, method := range tests.CoreMethods {
		if _, skipped := unreached[method]; skipped {
			continue
		}
		for _, mode := range tests.Modes {
			if method == "OracleInputs" {
				// The oracle answers a second suggestion of a band from
				// its cache; only an expired cache reaches the node.
				time.Sleep(oracleCachePeriod)
			}
			chain.core.SetMode(method, mode)
			callsBefore := chain.core.Calls(method)
			panicsBefore := chain.harness.Panics()
			crashed := 0
			for name, shapes := range requests {
				for _, shape := range shapes {
					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					_, err := chain.harness.Dispatcher.Serve(ctx, name, json.RawMessage(shape))
					cancel()
					if err != nil && err.Code == dispatch.CodeInternalError && err.Message == dispatch.ErrPanic.Message {
						crashed++
					}
				}
			}
			chain.core.SetMode(method, tests.Normal)
			require.Greater(t, chain.core.Calls(method), callsBefore, "no request reached %s in mode %s", method, mode)
			panics := chain.harness.Panics() - panicsBefore
			if mode == tests.Panic {
				require.Positive(t, crashed, "a panic of %s was not answered", method)
				require.Equal(t, float64(crashed), panics, "every panic of %s is counted once", method)
				continue
			}
			require.Zero(t, panics, "a request panicked when %s answered %s", method, mode)
		}
	}
	states := chain.core.States()
	require.Positive(t, states.Opened)
	require.Equal(t, states.Opened, states.Released, "every state opened is released once (I7)")
}

// TestMatrix_NamesEveryRegisteredMethod holds T4 for the matrix: a method
// the matrix does not call is a method whose failures nothing checks.
func TestMatrix_NamesEveryRegisteredMethod(t *testing.T) {
	chain := newMatrixChain(t)
	requests := chain.requests()
	for _, method := range chain.harness.Dispatcher.Methods() {
		require.Contains(t, requests, method, "the matrix sends no request to %s", method)
	}
	for method := range requests {
		require.True(t, chain.harness.Dispatcher.Has(method), "the matrix names %s, which nothing registers", method)
	}
}

// TestMatrix_UnreachedMethodsAreCoreMethods holds TS6 for the list of
// unreached methods.
func TestMatrix_UnreachedMethodsAreCoreMethods(t *testing.T) {
	for method, reason := range unreached {
		require.Contains(t, tests.CoreMethods, method)
		require.NotEmpty(t, strings.TrimSpace(reason))
	}
	require.True(t, slices.IsSorted(tests.CoreMethods))
}

// TestMatrix_ACancelledRequestReturnsWithinOneCall holds Q2: under a slow
// node, a request cancelled while it waits makes at most one more call and
// returns.
func TestMatrix_ACancelledRequestReturnsWithinOneCall(t *testing.T) {
	chain := newMatrixChain(t)
	const latency = 100 * time.Millisecond
	chain.core.SetDelay(latency)
	for name, shapes := range chain.requests() {
		for _, shape := range shapes {
			ctx, cancel := context.WithTimeout(context.Background(), latency/2)
			start := time.Now()
			_, _ = chain.harness.Dispatcher.Serve(ctx, name, json.RawMessage(shape))
			cancel()
			require.Less(t, time.Since(start), 2*latency+latency/2, "%s %s did not return within one call of its cancel", name, shape)
		}
	}
}

// TestMatrix_ABlockedCallStallsNoOtherRequest holds P4: a request waiting
// for the node holds no lock another request needs.
func TestMatrix_ABlockedCallStallsNoOtherRequest(t *testing.T) {
	chain := newMatrixChain(t)
	release := chain.core.Hold("Block")
	blocked := make(chan struct{})
	go func() {
		defer close(blocked)
		_, _ = chain.harness.Call("eth_getBlockByNumber", "0x1", false)
	}()
	tests.Eventually(t, func() bool { return chain.core.Calls("Block") > 0 }, "the request to reach the node")
	result := chain.harness.Result("eth_chainId")
	require.JSONEq(t, `"0xfa3"`, string(result))
	result = chain.harness.Result("eth_getTransactionByHash", chain.mined)
	require.Contains(t, string(result), chain.mined.Hex())
	release()
	<-blocked
}
