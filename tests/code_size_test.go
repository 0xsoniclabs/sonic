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
	"math/big"
	"strings"
	"testing"

	"github.com/0xsoniclabs/sonic/api/ethapi"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests/contracts/contractcreator"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

func TestCodeSizeLimit_MaxInitCodeSizeIsUpdatedWithBrio(t *testing.T) {
	tests := map[string]struct {
		upgrades     opera.Upgrades
		initCodeSize uint64
		errorMessage string
	}{
		"Allegro below limit": {
			upgrades:     opera.GetAllegroUpgrades(),
			initCodeSize: MAX_INIT_CODE_SIZE,
		},
		"Allegro above limit": {
			upgrades:     opera.GetAllegroUpgrades(),
			initCodeSize: MAX_INIT_CODE_SIZE + 1,
			errorMessage: "max initcode size exceeded: code size 49153, limit 49152",
		},
		"Brio below limit": {
			upgrades:     opera.GetBrioUpgrades(),
			initCodeSize: opera.SonicPostAllegroMaxInitCodeSize,
		},
		"Brio above limit": {
			upgrades:     opera.GetBrioUpgrades(),
			initCodeSize: opera.SonicPostAllegroMaxInitCodeSize + 1,
			errorMessage: "max initcode size exceeded: code size 98305, limit 98304",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			session := getIntegrationTestNetSession(t, test.upgrades)
			chainId := session.GetChainId()
			sender := session.GetSessionSponsor()

			txsPayload := &types.AccessListTx{
				ChainID: chainId,
				Nonce:   0,
				Gas:     20_000_000,
				To:      nil,                             // contract creation
				Data:    make([]byte, test.initCodeSize), // init code
			}

			signedTx := CreateTransaction(t, session, txsPayload, sender)
			receipt, err := session.Run(signedTx)

			if test.errorMessage != "" {
				require.ErrorContains(t, err, test.errorMessage)
				require.Nil(t, receipt)
			} else {
				require.NoError(t, err)
				require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
			}
		})
	}
}

func TestCodeSizeLimits_MaxContractCodeSizeIsUpdatedWithBrio(t *testing.T) {
	tests := map[string]struct {
		upgrades      opera.Upgrades
		codeSize      uint64
		receiptStatus uint64
	}{
		"Allegro below limit": {
			upgrades:      opera.GetAllegroUpgrades(),
			codeSize:      MAX_CODE_SIZE,
			receiptStatus: types.ReceiptStatusSuccessful,
		},
		"Allegro above limit": {
			upgrades:      opera.GetAllegroUpgrades(),
			codeSize:      MAX_CODE_SIZE + 1,
			receiptStatus: types.ReceiptStatusFailed,
		},
		"Brio below limit": {
			upgrades:      opera.GetBrioUpgrades(),
			codeSize:      opera.SonicPostAllegroMaxCodeSize,
			receiptStatus: types.ReceiptStatusSuccessful,
		},
		"Brio above limit": {
			upgrades:      opera.GetBrioUpgrades(),
			codeSize:      opera.SonicPostAllegroMaxCodeSize + 1,
			receiptStatus: types.ReceiptStatusFailed,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			session := getIntegrationTestNetSession(t, test.upgrades)
			chainId := session.GetChainId()
			sender := session.GetSessionSponsor()

			// The code to be deployed is the return of the init code,
			// so the init code just needs to return a byte array of the specified size.
			initCode := []byte{
				byte(vm.PUSH3), byte(test.codeSize >> 16), // push code size
				byte(test.codeSize >> 8), byte(test.codeSize), // push code size
				byte(vm.PUSH1), 0x00, // push memory offset
				byte(vm.RETURN), // return
			}

			txsPayload := &types.AccessListTx{
				ChainID: chainId,
				Nonce:   0,
				Gas:     20_000_000,
				To:      nil, // contract creation
				Data:    initCode,
			}

			signedTx := CreateTransaction(t, session, txsPayload, sender)
			receipt, err := session.Run(signedTx)
			require.NoError(t, err)
			require.Equal(t, test.receiptStatus, receipt.Status)
		})
	}
}

func TestCodeSizeLimits_CreateTransactionsAndCreateOpcodesHaveTheirOwnInitCodeSizeLimit(t *testing.T) {
	const (
		transactionLimit = opera.SonicPostAllegroMaxInitCodeSize
		opcodeLimit      = opera.SonicPostAllegroVMCreateMaxInitCodeSize
		gas              = 2_000_000
	)
	maxInitCodeSizeExceeded := vm.ErrMaxInitCodeSizeExceeded.Error()

	for name, upgrades := range opera.GetAllHardForksInOrder() {
		if !upgrades.Brio {
			continue
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			// The pool rejects creation transactions above the limit on its
			// own; without its validation they reach the block processing.
			net := StartIntegrationTestNet(t, IntegrationTestNetOptions{
				Upgrades:             &upgrades,
				ClientExtraArguments: []string{"--disable-txPool-validation"},
			})

			client, err := net.GetClient()
			require.NoError(t, err)
			defer client.Close()
			rpcClient := client.Client()
			sender := net.GetSessionSponsor()

			_, receipt, err := DeployContract(net, contractcreator.DeployContractcreator)
			require.NoError(t, err)
			require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
			creator := receipt.ContractAddress
			creatorAbi, err := contractcreator.ContractcreatorMetaData.GetAbi()
			require.NoError(t, err)

			creationTx := func(initCodeSize uint64) *types.LegacyTx {
				return &types.LegacyTx{Gas: gas, Data: make([]byte, initCodeSize)}
			}
			creatorTx := func(method string, initCodeSize uint64) *types.LegacyTx {
				data, err := creatorAbi.Pack(method, new(big.Int).SetUint64(initCodeSize))
				require.NoError(t, err)
				return &types.LegacyTx{To: &creator, Gas: gas, Data: data}
			}

			executed := []struct {
				name   string
				tx     *types.LegacyTx
				status uint64
				frames int
			}{
				{"transaction at limit", creationTx(transactionLimit), types.ReceiptStatusSuccessful, 1},
				{"CREATE at limit", creatorTx("CreatetWith", opcodeLimit), types.ReceiptStatusSuccessful, 2},
				{"CREATE above limit", creatorTx("CreatetWith", opcodeLimit+1), types.ReceiptStatusFailed, 1},
				{"CREATE2 at limit", creatorTx("Create2With", opcodeLimit), types.ReceiptStatusSuccessful, 2},
				{"CREATE2 above limit", creatorTx("Create2With", opcodeLimit+1), types.ReceiptStatusFailed, 1},
			}
			for _, test := range executed {
				t.Run(test.name, func(t *testing.T) {
					tx := CreateTransaction(t, net, test.tx, sender)
					receipt, err := net.Run(tx)
					require.NoError(t, err)
					require.Equal(t, test.status, receipt.Status)
					args := callArgs(tx, sender)

					// Calls replay the transaction on the state it ran on, where
					// the contract it creates does not exist yet.
					block := hexutil.EncodeBig(receipt.BlockNumber)
					txIndex := hexutil.Uint(receipt.TransactionIndex)
					for method, trace := range traceWithAllMethods(rpcClient, tx.Hash(), args, block, &txIndex) {
						t.Run(method, func(t *testing.T) {
							require.NoError(t, trace.err)
							require.Equal(t, test.frames, countFrames(trace.frames))
							if test.status == types.ReceiptStatusFailed {
								require.Contains(t, trace.frames[0].Error, maxInitCodeSizeExceeded)
							} else {
								requireNoErrorFrames(t, trace.frames)
							}
							if strings.HasPrefix(method, "debug_") {
								require.EqualValues(t, receipt.GasUsed, trace.frames[0].GasUsed)
							}
						})
					}
				})
			}

			t.Run("transaction above limit is skipped", func(t *testing.T) {
				other := MakeAccountWithBalance(t, net, big.NewInt(1e18))
				tx := CreateTransaction(t, net, creationTx(transactionLimit+1), sender)
				_, err := net.Send(tx)
				require.NoError(t, err)

				// A later transaction of another sender gets processed
				// while the skipped one is left without a receipt.
				receipt, err := net.Run(CreateTransaction(t, net, &types.LegacyTx{To: &common.Address{}}, other))
				require.NoError(t, err)
				require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
				_, err = client.TransactionReceipt(t.Context(), tx.Hash())
				require.ErrorIs(t, err, ethereum.NotFound)

				evictFromPool(t, net, tx, sender)
				args := callArgs(tx, sender)

				wantErrors := map[string]string{
					"debug_traceCall":        maxInitCodeSizeExceeded,
					"debug_traceTransaction": "not found",
					"trace_transaction":      "not found",
				}
				for method, trace := range traceWithAllMethods(rpcClient, tx.Hash(), args, "latest", nil) {
					t.Run(method, func(t *testing.T) {
						if method == "trace_call" {
							// trace_call reports failures before EVM execution in the trace.
							require.NoError(t, trace.err)
							require.Len(t, trace.frames, 1)
							require.Contains(t, trace.frames[0].Error, maxInitCodeSizeExceeded)
						} else {
							require.ErrorContains(t, trace.err, wantErrors[method])
						}
					})
				}
			})
		})
	}
}

// evictFromPool replaces a transaction that block processing skips. Such a
// transaction stays in the pool, where it keeps being re-emitted and blocks
// the later nonces of its sender.
func evictFromPool(t *testing.T, net IntegrationTestNetSession, tx *types.Transaction, sender *Account) {
	t.Helper()
	replacement := SignTransaction(t, net.GetChainId(), &types.LegacyTx{
		Nonce:    tx.Nonce(),
		To:       &common.Address{},
		Gas:      21_000,
		GasPrice: new(big.Int).Mul(tx.GasPrice(), big.NewInt(2)),
	}, sender)
	receipt, err := net.Run(replacement)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
}

type callFrame struct {
	Error   string         `json:"error"`
	GasUsed hexutil.Uint64 `json:"gasUsed"`
	Calls   []callFrame    `json:"calls"`
}

func countFrames(frames []callFrame) int {
	count := len(frames)
	for _, frame := range frames {
		count += countFrames(frame.Calls)
	}
	return count
}

func requireNoErrorFrames(t *testing.T, frames []callFrame) {
	t.Helper()
	for _, frame := range frames {
		require.Empty(t, frame.Error)
		requireNoErrorFrames(t, frame.Calls)
	}
}

func callArgs(tx *types.Transaction, sender *Account) ethapi.TransactionArgs {
	return ethapi.TransactionArgs{
		From: AsPointer(sender.Address()),
		To:   tx.To(),
		Gas:  AsPointer(hexutil.Uint64(tx.Gas())),
		Data: AsPointer(hexutil.Bytes(tx.Data())),
	}
}

type traceResult struct {
	frames []callFrame
	err    error
}

// traceWithAllMethods traces the transaction by its hash and as a call at
// the given position, in both the debug and the trace namespace.
func traceWithAllMethods(
	rpcClient *rpc.Client,
	txHash common.Hash,
	args ethapi.TransactionArgs,
	block string,
	txIndex *hexutil.Uint,
) map[string]traceResult {
	debugCallConfig := callTracerConfig()
	debugCallConfig.TxIndex = txIndex
	var debugCall, debugTransaction callFrame
	debugCallErr := rpcClient.Call(&debugCall, "debug_traceCall", args, block, debugCallConfig)
	debugTransactionErr := rpcClient.Call(&debugTransaction, "debug_traceTransaction", txHash, callTracerConfig())

	var call struct {
		Trace []callFrame `json:"trace"`
	}
	callConfig := ethapi.TraceCallConfig{TxIndex: txIndex}
	callErr := rpcClient.Call(&call, "trace_call", args, []string{"trace"}, block, callConfig)
	var transaction []callFrame
	transactionErr := rpcClient.Call(&transaction, "trace_transaction", txHash)

	return map[string]traceResult{
		"debug_traceCall":        {[]callFrame{debugCall}, debugCallErr},
		"debug_traceTransaction": {[]callFrame{debugTransaction}, debugTransactionErr},
		"trace_call":             {call.Trace, callErr},
		"trace_transaction":      {transaction, transactionErr},
	}
}
