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

package tests

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/api/ethapi"
	"github.com/0xsoniclabs/sonic/opera"
	block_override "github.com/0xsoniclabs/sonic/tests/contracts/blockoverride"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	req "github.com/stretchr/testify/require"
)

const (
	contractFunction_getBlockParameters = "0xa3289b77"
)

func TestBlockOverride(t *testing.T) {
	require := req.New(t)
	session := getIntegrationTestNetSession(t, opera.GetSonicUpgrades())
	t.Parallel()

	// Deploy the block override observer contract.
	contract, receipt, err := DeployContract(session, block_override.DeployBlockOverride)
	require.NoError(err, "failed to deploy contract; %v", err)
	require.Equal(types.ReceiptStatusSuccessful, receipt.Status, "Deployment Unsuccessful")
	contractAddress := receipt.ContractAddress

	//Call contract method to be sure it is deployed.
	receiptLog, err := session.Apply(contract.LogBlockNumber)
	require.NoError(err, "failed to create a log")
	require.Equal(types.ReceiptStatusSuccessful, receiptLog.Status,
		"failed to create a log")

	// Need valid block number for eth_call and debug_traceCall
	blockNumber := receiptLog.BlockNumber.Uint64()

	// Set parameters to be overridden
	time := uint64(1234)
	gasLimit := uint64(567890)
	blockOverrides := &ethapi.BlockOverrides{
		Number:      (*hexutil.Big)(big.NewInt(42)),
		Difficulty:  (*hexutil.Big)(big.NewInt(1)),
		Time:        (*hexutil.Uint64)(&time),
		GasLimit:    (*hexutil.Uint64)(&gasLimit),
		Coinbase:    &common.Address{1},
		Random:      &common.Hash{2},
		BaseFee:     (*hexutil.Big)(big.NewInt(1_000)),
		BlobBaseFee: (*hexutil.Big)(big.NewInt(100)),
	}

	t.Run("eth_call block override", func(t *testing.T) {
		newClient, err := session.GetClient()
		require.NoError(err, "failed to get client; %v", err)
		defer newClient.Close()
		compareCalls(t, newClient.Client(), contractAddress, blockNumber, blockOverrides, makeEthCall)
	})

	t.Run("debug_traceCall block override", func(t *testing.T) {
		newClient, err := session.GetClient()
		require.NoError(err, "failed to get client; %v", err)
		defer newClient.Close()
		compareCalls(t, newClient.Client(), contractAddress, blockNumber, blockOverrides, makeDebugTraceCall)
	})

}

func compareCalls(t *testing.T, rpcClient *rpc.Client, contractAddress common.Address, blockNumber uint64, blockOverrides *ethapi.BlockOverrides,
	callFunc func(t *testing.T, rpcClient *rpc.Client, contractAddress common.Address, blockNumber uint64, blockOverrides *ethapi.BlockOverrides) (BlockParameters, error)) {

	require := req.New(t)
	params, err := callFunc(t, rpcClient, contractAddress, blockNumber, nil)
	require.NoError(err, "failed to make eth_call; %v", err)

	paramsOverride, err := callFunc(t, rpcClient, contractAddress, blockNumber, blockOverrides)
	require.NoError(err, "failed to make eth_call; %v", err)

	t.Logf("params: %v", params)
	t.Logf("params: %v", paramsOverride)

	err = checkAllFieldsAreDifferent(params, paramsOverride)
	require.NoError(err, "failed to compare block parameters; %v", err)

	err = checkOverrides(paramsOverride, *blockOverrides)
	require.NoError(err, "failed to compare block parameters; %v", err)
}

// BlockParameters is a struct created from a response from a call
type BlockParameters struct {
	Number      *big.Int
	Difficulty  *big.Int
	Time        *big.Int
	GasLimit    *big.Int
	Coinbase    common.Address
	Random      common.Hash
	BaseFee     *big.Int
	BlobBaseFee *big.Int
}

func getBlockParameters(data []byte) (BlockParameters, error) {

	if len(data) != 256 {
		return BlockParameters{}, fmt.Errorf("invalid data length: %d, expected 256", len(data))
	}

	return BlockParameters{
		Number:      new(big.Int).SetBytes(data[:32]),
		Difficulty:  new(big.Int).SetBytes(data[32:64]),
		Time:        new(big.Int).SetBytes(data[64:96]),
		GasLimit:    new(big.Int).SetBytes(data[96:128]),
		Coinbase:    common.BytesToAddress(data[128:160]),
		Random:      common.BytesToHash(data[160:192]),
		BaseFee:     new(big.Int).SetBytes(data[192:224]),
		BlobBaseFee: new(big.Int).SetBytes(data[224:]),
	}, nil
}

func (bp *BlockParameters) String() string {
	return fmt.Sprintf(
		"number: %v difficulty: %v time: %v gasLimit: %v coinbase: %v random: %v baseFee: %v blobbasefee: %v",
		bp.Number, bp.Difficulty, bp.Time, bp.GasLimit, bp.Coinbase, bp.Random, bp.BaseFee, bp.BlobBaseFee,
	)
}

func getFunctionCallParameters(contractAddress *common.Address) map[string]interface{} {
	return map[string]interface{}{
		"to":   contractAddress.String(),
		"data": contractFunction_getBlockParameters,
	}
}

func makeEthCall(t *testing.T, rpcClient *rpc.Client, contractAddress common.Address, blockNumber uint64, blockOverrides *ethapi.BlockOverrides) (BlockParameters, error) {
	require := req.New(t)

	var res interface{}
	err := rpcClient.Call(&res, "eth_call", getFunctionCallParameters(&contractAddress), hexutil.EncodeUint64(blockNumber), nil, blockOverrides)
	require.NoError(err, "failed to call eth_call; %v", err)

	if s, ok := res.(string); ok {
		b, err := hexutil.Decode(s)
		require.NoError(err, "failed to decode result hex; %v", err)

		params, err := getBlockParameters(b)
		require.NoError(err, "failed to decode block parameters; %v", err)

		return params, nil
	} else {
		return BlockParameters{}, fmt.Errorf("invalid result type: %T", res)
	}
}

func makeDebugTraceCall(t *testing.T, rpcClient *rpc.Client, contractAddress common.Address, blockNumber uint64, blockOverrides *ethapi.BlockOverrides) (BlockParameters, error) {
	require := req.New(t)

	traceConfig := &ethapi.TraceCallConfig{
		BlockOverrides: blockOverrides,
	}

	var res interface{}
	err := rpcClient.Call(&res, "debug_traceCall", getFunctionCallParameters(&contractAddress), hexutil.EncodeUint64(blockNumber), traceConfig)
	require.NoError(err, "failed to call eth_call; %v", err)

	if data, ok := res.(map[string]interface{}); ok {
		if s, ok := data["returnValue"].(string); ok {
			if !strings.HasPrefix(s, "0x") {
				s = "0x" + s
			}
			b, err := hexutil.Decode(s)
			require.NoError(err, "failed to decode result hex; %v", err)

			params, err := getBlockParameters(b)
			require.NoError(err, "failed to decode block parameters; %v", err)

			return params, nil
		} else {
			return BlockParameters{}, fmt.Errorf("invalid result type: %T", res)
		}
	} else {
		return BlockParameters{}, fmt.Errorf("invalid result type: %T", res)
	}
}

// checkAllFieldsAreDifferent compares two BlockParameters objects and returns an error if any fields were not overridden
func checkAllFieldsAreDifferent(params1, params2 BlockParameters) error {
	if params1.Number.Cmp(params2.Number) == 0 {
		return fmt.Errorf("Number field was not overridden: %v", params1.Number)
	}
	if params1.Difficulty.Cmp(params2.Difficulty) == 0 {
		return fmt.Errorf("Difficulty field was not overridden: %v", params1.Difficulty)
	}
	if params1.Time.Cmp(params2.Time) == 0 {
		return fmt.Errorf("Time field was not overridden: %v", params1.Time)
	}
	if params1.GasLimit.Cmp(params2.GasLimit) == 0 {
		return fmt.Errorf("GasLimit field was not overridden: %v", params1.GasLimit)
	}
	if params1.Coinbase == params2.Coinbase {
		return fmt.Errorf("Coinbase field was not overridden: %v", params1.Coinbase)
	}
	if params1.Random == params2.Random {
		return fmt.Errorf("Random field was not overridden: %v", params1.Random)
	}
	if params1.BaseFee.Cmp(params2.BaseFee) == 0 {
		return fmt.Errorf("BaseFee field was not overridden: %v", params1.BaseFee)
	}
	if params1.BlobBaseFee.Cmp(params2.BlobBaseFee) == 0 {
		return fmt.Errorf("BlobBaseFee field was not overridden: %v", params1.BlobBaseFee)
	}
	return nil
}

// checkOverrides checks that the given BlockParameters are the same as the overrides
func checkOverrides(params BlockParameters, overrides ethapi.BlockOverrides) error {
	if want, got := overrides.Number, params.Number; got.Cmp((*big.Int)(want)) != 0 {
		return fmt.Errorf("Number override incorrect, wanted %v, got %v", want, got)
	}
	if want, got := overrides.Time, params.Time; got.Uint64() != (uint64(*want)) {
		return fmt.Errorf("Time override incorrect, wanted %v, got %v", want, got)
	}
	if want, got := overrides.GasLimit, params.GasLimit; got.Uint64() != (uint64(*want)) {
		return fmt.Errorf("GasLimit override incorrect, wanted %v, got %v", want, got)
	}
	if want, got := overrides.Coinbase, params.Coinbase; *want != got {
		return fmt.Errorf("Coinbase override incorrect, wanted %v, got %v", want, got)
	}
	if want, got := overrides.Random, params.Random; *want != got {
		return fmt.Errorf("Random override incorrect, wanted %v, got %v", want, got)
	}
	if want, got := overrides.BaseFee, params.BaseFee; got.Cmp((*big.Int)(want)) != 0 {
		return fmt.Errorf("BaseFee override incorrect, wanted %v, got %v", want, got)
	}
	if want, got := overrides.BlobBaseFee, params.BlobBaseFee; got.Cmp((*big.Int)(want)) != 0 {
		return fmt.Errorf("BlobBaseFee override incorrect, wanted %v, got %v", want, got)
	}
	return nil
}

func TestBlockOverride_CallAsNextBlock_KnowsBaseBlockHash(t *testing.T) {
	session := getIntegrationTestNetSession(t, opera.GetCantoUpgrades())
	t.Parallel()

	client, err := session.GetClient()
	req.NoError(t, err)
	defer client.Close()

	// Produce a few blocks so the base block has ancestors.
	for range 3 {
		_, err := session.EndowAccount(common.Address{0x42}, big.NewInt(1))
		req.NoError(t, err)
	}
	base, err := client.BlockByNumber(t.Context(), nil)
	req.NoError(t, err)
	baseNumber := base.NumberU64()
	parent, err := client.BlockByNumber(t.Context(), new(big.Int).SetUint64(baseNumber-1))
	req.NoError(t, err)

	// Code returning BLOCKHASH of the number given as the 32-byte input:
	// PUSH1 0, CALLDATALOAD, BLOCKHASH, PUSH1 0, MSTORE, PUSH1 32, PUSH1 0, RETURN
	blockHashReader := common.Address{0xbe, 0xef}
	stateOverrides := map[common.Address]map[string]any{
		blockHashReader: {"code": hexutil.Bytes{0x60, 0x00, 0x35, 0x40, 0x60, 0x00, 0x52, 0x60, 0x20, 0x60, 0x00, 0xf3}},
	}

	type caller func(t *testing.T, to common.Address, input []byte, number uint64) common.Hash
	callers := map[string]caller{
		"eth_call": func(t *testing.T, to common.Address, input []byte, number uint64) common.Hash {
			t.Helper()
			var res hexutil.Bytes
			err := client.Client().Call(&res, "eth_call",
				map[string]any{"to": to, "data": hexutil.Bytes(input)},
				hexutil.EncodeUint64(baseNumber), stateOverrides, numberOverride(number))
			req.NoError(t, err)
			return common.BytesToHash(res)
		},
		"debug_traceCall": func(t *testing.T, to common.Address, input []byte, number uint64) common.Hash {
			t.Helper()
			var res struct {
				Failed      bool   `json:"failed"`
				ReturnValue string `json:"returnValue"`
			}
			err := client.Client().Call(&res, "debug_traceCall",
				map[string]any{"to": to, "data": hexutil.Bytes(input)},
				hexutil.EncodeUint64(baseNumber),
				map[string]any{"stateOverrides": stateOverrides, "blockOverrides": numberOverride(number)})
			req.NoError(t, err)
			req.False(t, res.Failed, "call failed")
			out, err := hexutil.Decode("0x" + strings.TrimPrefix(res.ReturnValue, "0x"))
			req.NoError(t, err)
			return common.BytesToHash(out)
		},
	}

	blockHash := func(call caller, t *testing.T, of, number uint64) common.Hash {
		return call(t, blockHashReader, common.BigToHash(new(big.Int).SetUint64(of)).Bytes(), number)
	}
	historyHash := func(call caller, t *testing.T, of, number uint64) common.Hash {
		return call(t, params.HistoryStorageAddress, common.BigToHash(new(big.Int).SetUint64(of)).Bytes(), number)
	}

	for name, call := range callers {
		t.Run(name, func(t *testing.T) {
			t.Run("next block: BLOCKHASH of the base block", func(t *testing.T) {
				req.Equal(t, base.Hash(), blockHash(call, t, baseNumber, baseNumber+1))
			})
			t.Run("next block: BLOCKHASH of the base block's parent", func(t *testing.T) {
				req.Equal(t, parent.Hash(), blockHash(call, t, baseNumber-1, baseNumber+1))
			})
			t.Run("next block: history contract holds the base block", func(t *testing.T) {
				req.Equal(t, base.Hash(), historyHash(call, t, baseNumber, baseNumber+1))
			})
			t.Run("two blocks ahead: BLOCKHASH of the base block is unknown", func(t *testing.T) {
				req.Equal(t, common.Hash{}, blockHash(call, t, baseNumber, baseNumber+2))
				req.Equal(t, common.Hash{}, blockHash(call, t, baseNumber+1, baseNumber+2))
			})
			t.Run("two blocks ahead: history contract does not hold the base block", func(t *testing.T) {
				req.Equal(t, common.Hash{}, historyHash(call, t, baseNumber, baseNumber+2))
			})
			t.Run("two blocks ahead: BLOCKHASH of the base block's parent", func(t *testing.T) {
				req.Equal(t, parent.Hash(), blockHash(call, t, baseNumber-1, baseNumber+2))
			})
		})
	}
}

func numberOverride(number uint64) *ethapi.BlockOverrides {
	return &ethapi.BlockOverrides{Number: (*hexutil.Big)(new(big.Int).SetUint64(number))}
}

func TestBlockOverride_CallAsNextBlock_TraceTimeoutCoversHistorySystemCall(t *testing.T) {
	session := getIntegrationTestNetSession(t, opera.GetCantoUpgrades())
	t.Parallel()

	client, err := session.GetClient()
	req.NoError(t, err)
	defer client.Close()

	base, err := client.BlockByNumber(t.Context(), nil)
	req.NoError(t, err)
	baseNumber := base.NumberU64()

	// The history contract is replaced by a loop calling the BLAKE2F
	// precompile with 10,000 rounds, which takes seconds to use up the
	// 30,000,000 gas of the system call that runs before the call:
	//   PUSH4 10000, PUSH1 224, SHL, PUSH0, MSTORE       // rounds
	//   loop: JUMPDEST, PUSH0, PUSH0, PUSH1 213, PUSH0, PUSH1 9, GAS,
	//         STATICCALL, POP, PUSH1 loop, JUMP
	slowHistory := hexutil.MustDecode("0x630000271060e01b5f525b5f5f60d55f60095afa50600a56")
	stateOverrides := map[common.Address]map[string]any{
		params.HistoryStorageAddress: {"code": hexutil.Bytes(slowHistory)},
	}

	start := time.Now()
	var res json.RawMessage
	err = client.Client().Call(&res, "debug_traceCall",
		map[string]any{"to": common.Address{0x42}},
		hexutil.EncodeUint64(baseNumber),
		map[string]any{
			"stateOverrides": stateOverrides,
			"blockOverrides": numberOverride(baseNumber + 1),
			"timeout":        "50ms",
		})
	elapsed := time.Since(start)

	req.Error(t, err, "trace succeeded after %v", elapsed)
	req.Less(t, elapsed, 2*time.Second)
}
