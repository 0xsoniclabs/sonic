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
	"testing"

	"github.com/0xsoniclabs/sonic/integration/makefakegenesis"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/tosca/go/tosca/vm"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestRevisionIsForwardedCorrectly_DelegationDesignationAddressAccessIsConsideredInAllegro(t *testing.T) {
	gas := uint64(21_000) // transaction base
	gas += 7 * 3          // 7 push instructions
	gas += 2_600          // cold access to recipient
	gas += 10             // gas in recursive call (is fully consumed due to failed execution)

	tests := map[string]struct {
		upgrades opera.Upgrades
		gas      uint64
	}{
		"Sonic": {
			upgrades: opera.GetSonicUpgrades(),
			gas:      gas, // delegate designator ignored, no address access.
		},
		"Allegro": {
			upgrades: opera.GetAllegroUpgrades(),
			gas:      gas + 2_600, // cold access to delegate billed in interpreter.
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			net := StartIntegrationTestNetWithJsonGenesis(t, IntegrationTestNetOptions{
				Upgrades: &test.upgrades,
				Accounts: accountsToDeploy(),
			})

			client, err := net.GetClient()
			require.NoError(t, err)
			defer client.Close()

			sender := MakeAccountWithBalance(t, net, big.NewInt(1e18))

			gasPrice, err := client.SuggestGasPrice(t.Context())
			require.NoError(t, err)

			chainId, err := client.ChainID(t.Context())
			require.NoError(t, err)

			recipient := common.HexToAddress("0x44")
			txData := &types.AccessListTx{
				ChainID:    chainId,
				Nonce:      0,
				GasPrice:   gasPrice,
				Gas:        test.gas + 1, // +1 to ensure there was no error which consumed the gas
				To:         &recipient,
				Value:      big.NewInt(0),
				Data:       []byte{},
				AccessList: types.AccessList{},
			}
			tx := SignTransaction(t, chainId, txData, sender)

			receipt, err := net.Run(tx)
			require.NoError(t, err)

			require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
			require.Equal(t, test.gas, receipt.GasUsed)
		})
	}
}

func accountsToDeploy() []makefakegenesis.Account {
	// account 0x42 code: single invalid instruction (0xee)
	// account 0x43 code: delegation designation to 0x42: 0xef0100...042
	// account 0x44 code: code that calls 0x43

	account42 := makefakegenesis.Account{
		Name:    "account42",
		Address: common.HexToAddress("0x42"),
		Code:    []byte{0xee},
		Nonce:   1,
	}

	account43 := makefakegenesis.Account{
		Name:    "account43",
		Address: common.HexToAddress("0x43"),
		Code:    append([]byte{0xef, 0x01, 0x00}, common.HexToAddress("0x42").Bytes()...),
		Nonce:   1,
	}

	code44 := []byte{
		byte(vm.PUSH1), 0x00, // retSize
		byte(vm.PUSH1), 0x00, // retOffset
		byte(vm.PUSH1), 0x00, // argSize
		byte(vm.PUSH1), 0x00, // argOffset
		byte(vm.PUSH1), 0x00, // value
		byte(vm.PUSH20),
	}
	code44 = append(code44, common.HexToAddress("0x43").Bytes()...) // address
	code44 = append(code44,
		byte(vm.PUSH1), 0x0a, // gas
		byte(vm.CALL), // call
		byte(vm.STOP), // return
	)

	account44 := makefakegenesis.Account{
		Name:    "account44",
		Address: common.HexToAddress("0x44"),
		Code:    code44,
		Nonce:   1,
	}

	return []makefakegenesis.Account{account42, account43, account44}
}

func TestRevisionIsForwardedCorrectly_BrioEnablesOsakaInBlockProcessing(t *testing.T) {
	code := []byte{
		byte(vm.PUSH1), 0x00, // offset
		byte(vm.CALLDATALOAD), // load input data
		byte(vm.CLZ),          // count leading zeros
		byte(vm.PUSH1), 0x00,  // size of log
		byte(vm.PUSH1), 0x00, // offset of log
		byte(vm.LOG1), // log the CLZ result as topic
		byte(vm.STOP), // stop
	}
	account := makefakegenesis.Account{
		Name:    "account",
		Address: common.HexToAddress("0x42"),
		Code:    code,
	}

	tests := map[string]struct {
		upgrades       opera.Upgrades
		expectedReturn []byte
	}{
		"Sonic": {
			upgrades: opera.GetSonicUpgrades(),
		},
		"Allegro": {
			upgrades: opera.GetAllegroUpgrades(),
		},
		"Brio": {
			upgrades: opera.GetBrioUpgrades(),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			net := StartIntegrationTestNetWithJsonGenesis(t, IntegrationTestNetOptions{
				Upgrades: &test.upgrades,
				Accounts: []makefakegenesis.Account{account},
			})
			client, err := net.GetClient()
			require.NoError(t, err)
			defer client.Close()
			sender := MakeAccountWithBalance(t, net, big.NewInt(1e18))

			txData := &types.LegacyTx{
				Gas: 100_000,
				To:  &account.Address,
				Data: []byte{
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // 8 leading zero bytes
					0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
					0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
					0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
				},
			}
			tx := CreateTransaction(t, net, txData, sender)
			receipt, err := net.Run(tx)
			require.NoError(t, err)

			if !test.upgrades.Brio {
				require.Equal(t, types.ReceiptStatusFailed, receipt.Status)
			} else {
				require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

				logs := receipt.Logs
				require.Len(t, logs, 1, "expected exactly one log from CLZ contract")
				topics := logs[0].Topics
				require.Len(t, topics, 1)
				expected := common.Hash(append(make([]byte, 31), 64)) // 64 leading zero bits
				require.Equal(t, expected, topics[0], "CLZ log topic mismatch")
			}
		})
	}
}

func TestRevisionIsForwardedCorrectly_RPCCall_BrioEnablesOsaka(t *testing.T) {

	// This test verifies that RPC configuration of block processing correctly
	// forwards the revision to the EVM interpreter.
	// This test uses a smart contract that uses the CLZ opcode,
	// which is only available starting from the Brio upgrade.

	code := []byte{
		byte(vm.PUSH1), 0x00, // offset
		byte(vm.CALLDATALOAD), // load input data
		byte(vm.CLZ),          // count leading zeros

		// if execution reaches here, Brio is enabled.
		// do a call to produce tracer output.
		byte(vm.PUSH1), 0x00, // retSize
		byte(vm.PUSH1), 0x00, // retOffset
		byte(vm.PUSH1), 0x00, // argSize
		byte(vm.PUSH1), 0x00, // argOffset
		byte(vm.PUSH1), 0x00, // value
		byte(vm.PUSH1), 0x01, // address
		byte(vm.PUSH1), 0x0a, // gas
		byte(vm.CALL), // call
		byte(vm.STOP), // stop
	}
	brioOnlyContract := makefakegenesis.Account{
		Name:    "account",
		Address: common.HexToAddress("0x42"),
		Code:    code,
		Nonce:   1,
	}

	// 1)  start net with Allegro

	net := StartIntegrationTestNet(t, IntegrationTestNetOptions{
		Upgrades: AsPointer(opera.GetAllegroUpgrades()),
		Accounts: []makefakegenesis.Account{brioOnlyContract},
	})

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	blockBeforeUpgrade, err := client.BlockByNumber(t.Context(), nil)
	require.NoError(t, err)

	// 1.1)  contract cannot be executed before Brio

	err = doRpcCall(client, brioOnlyContract.Address, blockBeforeUpgrade.Hash())
	require.ErrorContains(t, err, "execution unsuccessful", "expected eth_call to fail before Brio upgrade")
	trace, err := doTraceCall(client, brioOnlyContract.Address, blockBeforeUpgrade.Hash())
	require.NoError(t, err, "expected trace_call to succeed even if execution fails")
	require.Contains(t, trace["error"].(string), "invalid opcode", "expected invalid opcode error in trace_call before Brio upgrade")

	// 2)  upgrade to Brio

	type rulesType struct {
		Upgrades struct{ Brio bool }
	}
	rulesDiff := rulesType{
		Upgrades: struct{ Brio bool }{Brio: true},
	}

	UpdateNetworkRules(t, net, rulesDiff)
	AdvanceEpochAndWaitForBlocks(t, net)

	blockAfterUpgrade, err := client.BlockByNumber(t.Context(), nil)
	require.NoError(t, err)

	// 2.1)  contract can be executed after Brio

	err = doRpcCall(client, brioOnlyContract.Address, blockAfterUpgrade.Hash())
	require.NoError(t, err, "expected eth_call to execute with Brio upgrade")
	trace, err = doTraceCall(client, brioOnlyContract.Address, blockAfterUpgrade.Hash())
	require.NoError(t, err, "expected trace_call to execute with Brio upgrade")
	_, failed := trace["error"]
	require.False(t, failed, "did not expect error in trace_call after Brio upgrade")
	require.Len(t, trace["calls"].([]any), 1, "expected one traced call entry from CLZ contract after Brio upgrade")

	// 2.2)  expect rcp calls failing if using older than fork block

	err = doRpcCall(client, brioOnlyContract.Address, blockBeforeUpgrade.Hash())
	require.ErrorContains(t, err, "execution unsuccessful", "expected eth_call to fail before Brio upgrade")
	trace, err = doTraceCall(client, brioOnlyContract.Address, blockBeforeUpgrade.Hash())
	require.NoError(t, err, "expected trace_call to succeed even if execution fails")
	require.Contains(t, trace["error"].(string), "invalid opcode", "expected invalid opcode error in trace_call before Brio upgrade")
}

// doTraceCall invokes debug_traceCall on the given contract at the given block hash
// it returns a map with the entries of the json response, or an error
func doTraceCall(client *PooledEhtClient, contractAddress common.Address, blockHash common.Hash) (map[string]any, error) {
	// debug_traceCall serves to test functions using StateTransition RPC method
	config := map[string]any{
		"tracer": "callTracer",
	}
	result, err := InvokeRpcCallMethod("debug_traceCall", contractAddress, client, blockHash, config)
	return result.(map[string]any), err

}

// doRpcCall invokes eth_call on the given contract at the given block hash
// it returns a map with the entries of the json response, or an error
func doRpcCall(client *PooledEhtClient, contractAddress common.Address, blockHash common.Hash) error {
	// eth_call servers to test functions using the DoCall RPC method
	_, err := InvokeRpcCallMethod("eth_call", contractAddress, client, blockHash, nil, nil)
	return err
}

func InvokeRpcCallMethod(
	method string,
	contractAddress common.Address,
	client *PooledEhtClient,
	blockHash common.Hash,
	args ...any,
) (any, error) {
	data := []byte{
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // 8 leading zero bytes
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	}
	txArguments := map[string]string{
		"to":   contractAddress.Hex(),
		"data": fmt.Sprintf("0x%s", common.Bytes2Hex(data)),
	}
	var res any
	err := client.Client().Call(&res, method, append([]any{txArguments, blockHash}, args...)...)
	return res, err
}
