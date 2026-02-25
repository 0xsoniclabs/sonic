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

package rpcs

import (
	"math/big"
	"strings"
	"testing"

	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

// Local decode types for eth_simulateV1 responses.
type simulateV1BlockResult struct {
	Number   string                 `json:"number"`
	GasLimit string                 `json:"gasLimit"`
	Calls    []simulateV1CallResult `json:"calls"`
}

type simulateV1CallResult struct {
	ReturnData string               `json:"returnData"`
	Status     string               `json:"status"`
	GasUsed    string               `json:"gasUsed"`
	Logs       []simulateV1Log      `json:"logs"`
	Error      *simulateV1CallError `json:"error"`
}

type simulateV1Log struct {
	Address string   `json:"address"`
	Topics  []string `json:"topics"`
	Data    string   `json:"data"`
}

type simulateV1CallError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// TestSimulateV1 tests the eth_simulateV1 RPC endpoint.
func TestSimulateV1(t *testing.T) {
	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{})
	t.Parallel()

	t.Run("empty_block_state_calls_returns_error", func(t *testing.T) {
		client, err := net.GetClient()
		require.NoError(t, err)
		defer client.Close()

		opts := map[string]interface{}{
			"blockStateCalls": []interface{}{},
		}
		var result interface{}
		err = client.Client().Call(&result, "eth_simulateV1", opts, "latest")
		require.Error(t, err, "empty blockStateCalls must return an error")
	})

	t.Run("basic_eth_transfer_succeeds", func(t *testing.T) {
		client, err := net.GetClient()
		require.NoError(t, err)
		defer client.Close()

		sender := common.HexToAddress("0x1111111111111111111111111111111111111111")
		receiver := common.HexToAddress("0x2222222222222222222222222222222222222222")
		hundredEth := hexutil.EncodeBig(new(big.Int).Mul(big.NewInt(1e18), big.NewInt(100)))
		oneEth := hexutil.EncodeBig(new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1)))

		opts := map[string]interface{}{
			"blockStateCalls": []interface{}{
				map[string]interface{}{
					"stateOverrides": map[string]interface{}{
						sender.Hex(): map[string]interface{}{
							"balance": hundredEth,
						},
					},
					"calls": []interface{}{
						map[string]interface{}{
							"from":  sender.Hex(),
							"to":    receiver.Hex(),
							"value": oneEth,
						},
					},
				},
			},
		}
		var result []simulateV1BlockResult
		err = client.Client().Call(&result, "eth_simulateV1", opts, "latest")
		require.NoError(t, err, "eth_simulateV1 must succeed for a basic ETH transfer")
		require.Len(t, result, 1, "must return one block result")
		require.Len(t, result[0].Calls, 1, "must return one call result")
		require.Equal(t, "0x1", result[0].Calls[0].Status, "ETH transfer must succeed")
		require.NotEmpty(t, result[0].Calls[0].GasUsed, "gasUsed must be non-empty")
	})

	t.Run("code_override_returns_correct_data", func(t *testing.T) {
		client, err := net.GetClient()
		require.NoError(t, err)
		defer client.Close()

		// Runtime bytecode: PUSH1 0x42  PUSH1 0x00  MSTORE  PUSH1 0x20  PUSH1 0x00  RETURN
		// Returns the 32-byte big-endian encoding of 0x42 (= 66).
		contractAddr := common.HexToAddress("0x3333333333333333333333333333333333333333")
		const returnsFortyTwoCode = "0x604260005260206000f3"

		opts := map[string]interface{}{
			"blockStateCalls": []interface{}{
				map[string]interface{}{
					"stateOverrides": map[string]interface{}{
						contractAddr.Hex(): map[string]interface{}{
							"code": returnsFortyTwoCode,
						},
					},
					"calls": []interface{}{
						map[string]interface{}{
							"to": contractAddr.Hex(),
						},
					},
				},
			},
		}
		var result []simulateV1BlockResult
		err = client.Client().Call(&result, "eth_simulateV1", opts, "latest")
		require.NoError(t, err, "eth_simulateV1 must succeed")
		require.Len(t, result, 1)
		require.Len(t, result[0].Calls, 1)
		require.Equal(t, "0x1", result[0].Calls[0].Status, "call to overridden code must succeed")

		data, err := hexutil.Decode(result[0].Calls[0].ReturnData)
		require.NoError(t, err, "must decode returnData")
		require.Len(t, data, 32, "returnData must be 32 bytes")
		require.Equal(t, byte(0x42), data[31], "last byte of returnData must be 0x42")
	})

	t.Run("reverted_call_has_failed_status_and_error", func(t *testing.T) {
		client, err := net.GetClient()
		require.NoError(t, err)
		defer client.Close()

		// Runtime bytecode: PUSH1 0x00  DUP1  REVERT → always reverts with no data.
		contractAddr := common.HexToAddress("0x4444444444444444444444444444444444444444")
		const alwaysRevertCode = "0x600080fd"

		opts := map[string]interface{}{
			"blockStateCalls": []interface{}{
				map[string]interface{}{
					"stateOverrides": map[string]interface{}{
						contractAddr.Hex(): map[string]interface{}{
							"code": alwaysRevertCode,
						},
					},
					"calls": []interface{}{
						map[string]interface{}{
							"to": contractAddr.Hex(),
						},
					},
				},
			},
		}
		var result []simulateV1BlockResult
		err = client.Client().Call(&result, "eth_simulateV1", opts, "latest")
		require.NoError(t, err, "eth_simulateV1 must not fail for reverted calls; errors are inlined")
		require.Len(t, result, 1)
		require.Len(t, result[0].Calls, 1)
		require.Equal(t, "0x0", result[0].Calls[0].Status, "reverted call must have failed status")
		require.NotNil(t, result[0].Calls[0].Error, "reverted call must include an error object")
	})

	t.Run("reverted_call_includes_revert_reason", func(t *testing.T) {
		client, err := net.GetClient()
		require.NoError(t, err)
		defer client.Close()

		// Runtime bytecode that always reverts with reason string "fail".
		// It stores the ABI encoding of Error(string)="fail" in memory then calls REVERT(0, 100).
		//
		// Memory layout of the 100-byte revert payload:
		//   bytes[0..3]   = 0x08c379a0                     Error(string) selector
		//   bytes[4..35]  = 0x0000...0020 (uint256 = 32)   offset to string data
		//   bytes[36..67] = 0x0000...0004 (uint256 = 4)    string length
		//   bytes[68..99] = "fail" (0x6661696c) + 28 zeros string data
		//
		// Assembly:
		//   PUSH32 0x08c379a0_00..00  PUSH1 0x00  MSTORE   (bytes[0..31])
		//   PUSH32 0x00000020_00..00  PUSH1 0x20  MSTORE   (bytes[32..63])
		//   PUSH32 0x00000004_6661696c_00..00  PUSH1 0x40  MSTORE  (bytes[64..95])
		//   PUSH1 0x64  PUSH1 0x00  REVERT
		contractAddr := common.HexToAddress("0xDdDdDdDdDdDdDdDdDdDdDdDdDdDdDdDdDdDdDdDd")
		const revertsWithReasonCode = "0x7f08c379a0000000000000000000000000000000000000000000000000000000006000527f00000020000000000000000000000000000000000000000000000000000000006020527f000000046661696c00000000000000000000000000000000000000000000000060405260646000fd"

		opts := map[string]interface{}{
			"blockStateCalls": []interface{}{
				map[string]interface{}{
					"stateOverrides": map[string]interface{}{
						contractAddr.Hex(): map[string]interface{}{
							"code": revertsWithReasonCode,
						},
					},
					"calls": []interface{}{
						map[string]interface{}{
							"to": contractAddr.Hex(),
						},
					},
				},
			},
		}
		var result []simulateV1BlockResult
		err = client.Client().Call(&result, "eth_simulateV1", opts, "latest")
		require.NoError(t, err, "eth_simulateV1 must not fail for reverted calls; errors are inlined")
		require.Len(t, result, 1)
		require.Len(t, result[0].Calls, 1)
		call := result[0].Calls[0]
		require.Equal(t, "0x0", call.Status, "call must have failed status")
		require.NotNil(t, call.Error, "failed call must include an error object")
		require.Equal(t, -32000, call.Error.Code, "error code must be -32000 (execution reverted)")
		require.Contains(t, call.Error.Message, "fail",
			"error message must contain the revert reason string")

		// The returnData must start with the Error(string) selector 0x08c379a0.
		returnData, err := hexutil.Decode(call.ReturnData)
		require.NoError(t, err, "must decode returnData hex")
		require.GreaterOrEqual(t, len(returnData), 4,
			"returnData must contain at least the 4-byte Error selector")
		require.Equal(t, []byte{0x08, 0xc3, 0x79, 0xa0}, returnData[:4],
			"returnData must start with the Error(string) selector 0x08c379a0")
	})

	t.Run("gas_limit_block_override_is_reflected_in_response", func(t *testing.T) {
		client, err := net.GetClient()
		require.NoError(t, err)
		defer client.Close()

		const wantGasLimit = uint64(0x5F5E100) // 100,000,000
		opts := map[string]interface{}{
			"blockStateCalls": []interface{}{
				map[string]interface{}{
					"blockOverrides": map[string]interface{}{
						"gasLimit": hexutil.EncodeUint64(wantGasLimit),
					},
					"calls": []interface{}{},
				},
			},
		}
		var result []simulateV1BlockResult
		err = client.Client().Call(&result, "eth_simulateV1", opts, "latest")
		require.NoError(t, err, "eth_simulateV1 must succeed")
		require.Len(t, result, 1, "must return one block result")
		require.Equal(t, hexutil.EncodeUint64(wantGasLimit), result[0].GasLimit,
			"simulated block gasLimit must match the blockOverrides value")
	})

	t.Run("state_persists_across_multiple_blocks", func(t *testing.T) {
		client, err := net.GetClient()
		require.NoError(t, err)
		defer client.Close()

		// Block 1: fund `relay` by transferring 2 ETH from `sender`
		//           (sender gets a balance override so the transfer can proceed).
		// Block 2: relay forwards 1 ETH to `receiver`.
		//           No state override is applied to relay — its funds must come
		//           from the state produced by block 1.
		sender := common.HexToAddress("0x5555555555555555555555555555555555555555")
		relay := common.HexToAddress("0x6666666666666666666666666666666666666666")
		receiver := common.HexToAddress("0x7777777777777777777777777777777777777777")
		hundredEth := hexutil.EncodeBig(new(big.Int).Mul(big.NewInt(1e18), big.NewInt(100)))
		twoEth := hexutil.EncodeBig(new(big.Int).Mul(big.NewInt(1e18), big.NewInt(2)))
		oneEth := hexutil.EncodeBig(new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1)))

		opts := map[string]interface{}{
			"blockStateCalls": []interface{}{
				// Block 1
				map[string]interface{}{
					"stateOverrides": map[string]interface{}{
						sender.Hex(): map[string]interface{}{
							"balance": hundredEth,
						},
					},
					"calls": []interface{}{
						map[string]interface{}{
							"from":  sender.Hex(),
							"to":    relay.Hex(),
							"value": twoEth,
						},
					},
				},
				// Block 2 — no state override for relay
				map[string]interface{}{
					"calls": []interface{}{
						map[string]interface{}{
							"from":  relay.Hex(),
							"to":    receiver.Hex(),
							"value": oneEth,
						},
					},
				},
			},
		}
		var result []simulateV1BlockResult
		err = client.Client().Call(&result, "eth_simulateV1", opts, "latest")
		require.NoError(t, err, "eth_simulateV1 must succeed")
		require.Len(t, result, 2, "must return two block results")
		require.Equal(t, "0x1", result[0].Calls[0].Status,
			"block 1 transfer must succeed")
		require.Equal(t, "0x1", result[1].Calls[0].Status,
			"block 2 transfer must succeed — state from block 1 must persist")
	})

	t.Run("trace_transfers_emits_pseudo_transfer_log", func(t *testing.T) {
		client, err := net.GetClient()
		require.NoError(t, err)
		defer client.Close()

		sender := common.HexToAddress("0x8888888888888888888888888888888888888888")
		receiver := common.HexToAddress("0x9999999999999999999999999999999999999999")
		hundredEth := hexutil.EncodeBig(new(big.Int).Mul(big.NewInt(1e18), big.NewInt(100)))
		oneEth := hexutil.EncodeBig(new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1)))

		opts := map[string]interface{}{
			"blockStateCalls": []interface{}{
				map[string]interface{}{
					"stateOverrides": map[string]interface{}{
						sender.Hex(): map[string]interface{}{
							"balance": hundredEth,
						},
					},
					"calls": []interface{}{
						map[string]interface{}{
							"from":  sender.Hex(),
							"to":    receiver.Hex(),
							"value": oneEth,
						},
					},
				},
			},
			"traceTransfers": true,
		}
		var result []simulateV1BlockResult
		err = client.Client().Call(&result, "eth_simulateV1", opts, "latest")
		require.NoError(t, err, "eth_simulateV1 must succeed with traceTransfers=true")
		require.Len(t, result, 1)
		require.Len(t, result[0].Calls, 1)

		// ERC-7528 canonical address for native ETH pseudo-events.
		const ethPseudoAddress = "0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE"
		// keccak256("Transfer(address,address,uint256)")
		const transferEventTopic = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"

		logs := result[0].Calls[0].Logs
		require.NotEmpty(t, logs, "ETH transfer must emit at least one log when traceTransfers=true")

		foundTransferLog := false
		for _, l := range logs {
			if strings.EqualFold(l.Address, ethPseudoAddress) {
				require.NotEmpty(t, l.Topics, "ETH pseudo-log must have topics")
				require.Equal(t, transferEventTopic, l.Topics[0],
					"first topic must be the ERC-20 Transfer event signature")
				foundTransferLog = true
			}
		}
		require.True(t, foundTransferLog,
			"must find an ETH transfer pseudo-log at address %s", ethPseudoAddress)
	})

	t.Run("simulation_does_not_modify_chain_state", func(t *testing.T) {
		client, err := net.GetClient()
		require.NoError(t, err)
		defer client.Close()

		// Use an address that no test ever funds on the real chain.
		freshAddr := common.HexToAddress("0xAaAaAaAaAaAaAaAaAaAaAaAaAaAaAaAaAaAaAaAa")

		var balanceBefore string
		err = client.Client().Call(&balanceBefore, "eth_getBalance", freshAddr.Hex(), "latest")
		require.NoError(t, err, "eth_getBalance must succeed before simulation")
		require.Equal(t, "0x0", balanceBefore, "fresh address must have zero balance before simulation")

		// Simulate giving freshAddr a large balance — this must not affect the real chain.
		hundredEth := hexutil.EncodeBig(new(big.Int).Mul(big.NewInt(1e18), big.NewInt(100)))
		opts := map[string]interface{}{
			"blockStateCalls": []interface{}{
				map[string]interface{}{
					"stateOverrides": map[string]interface{}{
						freshAddr.Hex(): map[string]interface{}{
							"balance": hundredEth,
						},
					},
					"calls": []interface{}{},
				},
			},
		}
		var simResult []simulateV1BlockResult
		err = client.Client().Call(&simResult, "eth_simulateV1", opts, "latest")
		require.NoError(t, err, "eth_simulateV1 must succeed")

		var balanceAfter string
		err = client.Client().Call(&balanceAfter, "eth_getBalance", freshAddr.Hex(), "latest")
		require.NoError(t, err, "eth_getBalance must succeed after simulation")
		require.Equal(t, "0x0", balanceAfter,
			"chain state must not be modified by eth_simulateV1")
	})

	t.Run("multiple_calls_in_single_block", func(t *testing.T) {
		client, err := net.GetClient()
		require.NoError(t, err)
		defer client.Close()

		// Two calls in one block: first returns data, second reverts.
		contractAddr := common.HexToAddress("0xBbBbBbBbBbBbBbBbBbBbBbBbBbBbBbBbBbBbBbBb")
		const returnsFortyTwoCode = "0x604260005260206000f3"
		const alwaysRevertCode = "0x600080fd"
		revertAddr := common.HexToAddress("0xCcCcCcCcCcCcCcCcCcCcCcCcCcCcCcCcCcCcCcCc")

		opts := map[string]interface{}{
			"blockStateCalls": []interface{}{
				map[string]interface{}{
					"stateOverrides": map[string]interface{}{
						contractAddr.Hex(): map[string]interface{}{
							"code": returnsFortyTwoCode,
						},
						revertAddr.Hex(): map[string]interface{}{
							"code": alwaysRevertCode,
						},
					},
					"calls": []interface{}{
						map[string]interface{}{"to": contractAddr.Hex()},
						map[string]interface{}{"to": revertAddr.Hex()},
					},
				},
			},
		}
		var result []simulateV1BlockResult
		err = client.Client().Call(&result, "eth_simulateV1", opts, "latest")
		require.NoError(t, err, "eth_simulateV1 must succeed")
		require.Len(t, result, 1, "must return one block result")
		require.Len(t, result[0].Calls, 2, "must return two call results")
		require.Equal(t, "0x1", result[0].Calls[0].Status, "first call must succeed")
		require.Equal(t, "0x0", result[0].Calls[1].Status, "second call must fail")
		require.Nil(t, result[0].Calls[0].Error, "successful call must have no error")
		require.NotNil(t, result[0].Calls[1].Error, "failed call must have an error")
	})

	t.Run("contains_emited_logs", func(t *testing.T) {
		client, err := net.GetClient()
		require.NoError(t, err)
		defer client.Close()

		// Override a contract that emits an event log. The logs emitted by the overridden code must be included in the response.
		contractAddr := common.HexToAddress("0x3333333333333333333333333333333333333333")
		const contractBytecode = "0x608060405234801561000f575f80fd5b5060043610610029575f3560e01c8063a6f9dae11461002d575b5f80fd5b6100476004803603810190610042919061011e565b61005d565b6040516100549190610158565b60405180910390f35b5f8173ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff167f342827c97908e5e2f71151c08502a66d44b6f758e3ac2f1de95f02eb95f0a73560405160405180910390a3819050919050565b5f80fd5b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f6100ed826100c4565b9050919050565b6100fd816100e3565b8114610107575f80fd5b50565b5f81359050610118816100f4565b92915050565b5f60208284031215610133576101326100c0565b5b5f6101408482850161010a565b91505092915050565b610152816100e3565b82525050565b5f60208201905061016b5f830184610149565b9291505056fea264697066735822122096c65ce6729c0e854dd165928f5e47d45ace055648adf9592712a051b22e44e064736f6c63430008140033"

		opts := map[string]interface{}{
			"blockStateCalls": []interface{}{
				map[string]interface{}{
					"stateOverrides": map[string]interface{}{
						contractAddr.Hex(): map[string]interface{}{
							"code": contractBytecode,
						},
					},
					"calls": []interface{}{
						map[string]interface{}{
							"to":   contractAddr.Hex(),
							"data": "0xa6f9dae10000000000000000000000005B38Da6a701c568545dCfcB03FcB875f56beddC4",
						},
					},
				},
			},
		}
		var result []simulateV1BlockResult
		err = client.Client().Call(&result, "eth_simulateV1", opts, "latest")
		require.NoError(t, err, "eth_simulateV1 must succeed")
		require.Len(t, result, 1)
		require.Len(t, result[0].Calls, 1)
		require.Equal(t, "0x1", result[0].Calls[0].Status, "call to overridden code must succeed")

		// keccak256("OwnerSet(address,address)")
		const ownerSetTopic = "0x342827c97908e5e2f71151c08502a66d44b6f758e3ac2f1de95f02eb95f0a735"

		logs := result[0].Calls[0].Logs
		require.NotEmpty(t, logs, "call must emit at least one log")

		foundOwnerSetLog := false
		for _, l := range logs {
			if strings.EqualFold(l.Address, contractAddr.Hex()) {
				require.NotEmpty(t, l.Topics, "log must have topics")
				require.Equal(t, ownerSetTopic, l.Topics[0],
					"first topic must be the OwnerSet event signature")
				foundOwnerSetLog = true
			}
		}
		require.True(t, foundOwnerSetLog,
			"must find an OwnerSet log from the overridden contract at address %s", contractAddr.Hex())
	})
}
