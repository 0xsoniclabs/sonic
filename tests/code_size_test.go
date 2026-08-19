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
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
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

func TestCodeSizeLimits_FailedDeploymentsReportTheFirstFailingCheck(t *testing.T) {
	// The 0xEF prefix is checked first. Before Canto, the code deposit is
	// charged before the code size is checked; from Canto on, the size is
	// checked first.
	const brioSize = opera.SonicBrioMaxCodeSize + 1
	const cantoSize = params.MaxCodeSizeAmsterdam + 1
	tests := map[string]struct {
		upgrades opera.Upgrades
		codeSize uint64
		prefixEF bool
		gas      uint64
		want     error
	}{
		"Brio oversized code starting with 0xEF": {
			upgrades: opera.GetBrioUpgrades(),
			codeSize: brioSize,
			prefixEF: true,
			gas:      20_000_000,
			want:     vm.ErrInvalidCode,
		},
		"Brio oversized code with unpayable deposit": {
			upgrades: opera.GetBrioUpgrades(),
			codeSize: brioSize,
			gas:      1_000_000,
			want:     vm.ErrCodeStoreOutOfGas,
		},
		"Brio oversized code with payable deposit": {
			upgrades: opera.GetBrioUpgrades(),
			codeSize: brioSize,
			gas:      20_000_000,
			want:     vm.ErrMaxCodeSizeExceeded,
		},
		"Canto oversized code starting with 0xEF": {
			upgrades: opera.GetCantoUpgrades(),
			codeSize: cantoSize,
			prefixEF: true,
			gas:      20_000_000,
			want:     vm.ErrInvalidCode,
		},
		"Canto oversized code with unpayable deposit": {
			upgrades: opera.GetCantoUpgrades(),
			codeSize: cantoSize,
			gas:      1_000_000,
			want:     vm.ErrMaxCodeSizeExceeded,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			session := getIntegrationTestNetSession(t, test.upgrades)
			client, err := session.GetClient()
			require.NoError(t, err)
			defer client.Close()

			var initCode []byte
			if test.prefixEF {
				initCode = []byte{
					byte(vm.PUSH1), 0xEF, // push prefix
					byte(vm.PUSH1), 0x00, // push memory offset
					byte(vm.MSTORE8), // store prefix
				}
			}
			initCode = append(initCode,
				byte(vm.PUSH3), byte(test.codeSize>>16), // push code size
				byte(test.codeSize>>8), byte(test.codeSize), // push code size
				byte(vm.PUSH1), 0x00, // push memory offset
				byte(vm.RETURN), // return
			)

			_, err = client.CallContract(t.Context(), ethereum.CallMsg{
				Gas:  test.gas,
				Data: initCode,
			}, nil)
			require.ErrorContains(t, err, test.want.Error())

			var trace struct {
				Error string `json:"error"`
			}
			err = client.Client().Call(&trace, "debug_traceCall",
				map[string]any{
					"gas":  hexutil.Uint64(test.gas),
					"data": hexutil.Bytes(initCode),
				},
				"latest",
				map[string]any{"tracer": "callTracer"},
			)
			require.NoError(t, err)
			require.Contains(t, trace.Error, test.want.Error())
		})
	}
}
