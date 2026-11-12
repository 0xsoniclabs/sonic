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
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/api/ethapi"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests/contracts/contractcreator"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
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
			initCodeSize: opera.SonicBrioMaxInitCodeSize,
		},
		"Brio above limit": {
			upgrades:     opera.GetBrioUpgrades(),
			initCodeSize: opera.SonicBrioMaxInitCodeSize + 1,
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
			codeSize:      opera.SonicBrioMaxCodeSize,
			receiptStatus: types.ReceiptStatusSuccessful,
		},
		"Brio above limit": {
			upgrades:      opera.GetBrioUpgrades(),
			codeSize:      opera.SonicBrioMaxCodeSize + 1,
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

			_, receipt, err := DeployContract(net, contractcreator.DeployContractcreator)
			require.NoError(t, err)
			require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
			creator := receipt.ContractAddress

			creatorAbi, err := contractcreator.ContractcreatorMetaData.GetAbi()
			require.NoError(t, err)
			callCreator := func(method string) func(initCodeSize uint64) []byte {
				return func(initCodeSize uint64) []byte {
					data, err := creatorAbi.Pack(method, new(big.Int).SetUint64(initCodeSize))
					require.NoError(t, err)
					return data
				}
			}

			kinds := map[string]struct {
				to    *common.Address
				data  func(initCodeSize uint64) []byte
				limit uint64
			}{
				"transaction": {
					data:  func(initCodeSize uint64) []byte { return make([]byte, initCodeSize) },
					limit: opera.SonicBrioMaxInitCodeSize,
				},
				"CREATE": {
					to:    &creator,
					data:  callCreator("CreatetWith"),
					limit: opera.SonicBrioVMCreateMaxInitCodeSize,
				},
				"CREATE2": {
					to:    &creator,
					data:  callCreator("Create2With"),
					limit: opera.SonicBrioVMCreateMaxInitCodeSize,
				},
			}

			sender := net.GetSessionSponsor()
			const gas = 2_000_000
			for kindName, kind := range kinds {
				for _, initCodeSize := range []uint64{kind.limit, kind.limit + 1} {
					data := kind.data(initCodeSize)
					withinLimit := initCodeSize <= kind.limit

					t.Run(fmt.Sprintf("%s/initCodeSize=%d/processing", kindName, initCodeSize), func(t *testing.T) {
						tx := CreateTransaction(t, net, &types.LegacyTx{
							To:   kind.to,
							Gas:  gas,
							Data: data,
						}, sender)

						switch {
						case withinLimit:
							receipt, err := net.Run(tx)
							require.NoError(t, err)
							require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
						case kind.to == nil:
							_, err := net.Send(tx)
							require.NoError(t, err)
							_, err = net.TryGetReceipt(3*time.Second, tx.Hash())
							require.ErrorIs(t, err, context.DeadlineExceeded, "transaction should be skipped")

							// The skipped transaction stays in the pool, where it keeps
							// being re-emitted and blocks the later nonces of its sender.
							replacement := SignTransaction(t, net.GetChainId(), &types.LegacyTx{
								Nonce:    tx.Nonce(),
								To:       &common.Address{},
								Gas:      21_000,
								GasPrice: new(big.Int).Mul(tx.GasPrice(), big.NewInt(2)),
							}, sender)
							receipt, err := net.Run(replacement)
							require.NoError(t, err)
							require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
						default:
							receipt, err := net.Run(tx)
							require.NoError(t, err)
							require.Equal(t, types.ReceiptStatusFailed, receipt.Status)
						}
					})

					t.Run(fmt.Sprintf("%s/initCodeSize=%d/tracing", kindName, initCodeSize), func(t *testing.T) {
						var trace struct {
							Error string `json:"error"`
						}
						err := client.Client().Call(&trace, "debug_traceCall", ethapi.TransactionArgs{
							From: AsPointer(sender.Address()),
							To:   kind.to,
							Gas:  AsPointer(hexutil.Uint64(gas)),
							Data: AsPointer(hexutil.Bytes(data)),
						}, "latest", callTracerConfig())

						switch {
						case withinLimit:
							require.NoError(t, err)
							require.Empty(t, trace.Error)
						case kind.to == nil:
							require.ErrorContains(t, err, vm.ErrMaxInitCodeSizeExceeded.Error())
						default:
							require.NoError(t, err)
							require.Contains(t, trace.Error, vm.ErrMaxInitCodeSizeExceeded.Error())
						}
					})
				}
			}
		})
	}
}
