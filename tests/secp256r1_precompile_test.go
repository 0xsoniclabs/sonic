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
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/integration/makefakegenesis"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/tosca/go/tosca/vm"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestSECP256r1_NewPrecompileHasCorrectGasCost(t *testing.T) {
	gasLimit := uint64(21_000)
	gasLimit += 160 * 4 // input intrinsic cost
	gasLimit += 2400    // access list cost

	tests := map[string]struct {
		upgrades opera.Upgrades
		gas      uint64
	}{
		"Sonic": {
			upgrades: opera.GetSonicUpgrades(),
			gas:      gasLimit,
		},
		"Allegro": {
			upgrades: opera.GetAllegroUpgrades(),
			gas:      gasLimit,
		},
		"Brio": {
			upgrades: opera.GetBrioUpgrades(),
			gas:      gasLimit + 6900, // gas cost of secp256r1 precompile
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			session := getIntegrationTestNetSession(t, test.upgrades)
			chainId := session.GetChainId()
			sender := session.GetSessionSponsor()

			precompileAddress := common.BytesToAddress([]byte{0x01, 0x00})
			txsPayload := &types.AccessListTx{
				ChainID: chainId,
				Nonce:   0,
				Gas:     test.gas + 1, // add 1 to ensure all the gas is not just consumed by an error
				To:      &precompileAddress,
				Data:    make([]byte, 160),

				// Prague added a floor data gas cost which depends on the size of the input data,
				// in order to test the secp256r1 gas cost, a random address is added to the access
				// list to increase the gas cost.
				AccessList: types.AccessList{
					{Address: common.Address{0x42}},
				},
			}

			signedTx := CreateTransaction(t, session, txsPayload, sender)
			receipt, err := session.Run(signedTx)
			require.NoError(t, err)

			require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
			require.Equal(t, test.gas, receipt.GasUsed)
		})
	}
}

func TestSECP256r1_VerifySignatureInBrio(t *testing.T) {
	// This test uses a contract that calls the secp256r1 precompile.
	// The precompile returns 1 if the signature is valid, and 0 if the signature is invalid.
	// In case the precompile does not exist, the call returns no return data.
	// The calling contract stops if the return value is 1, and reverts if the return value is 0 or empty.

	// validInput is a valid secp256r1 signature verification input.
	validInput := []byte{187, 90, 82, 244, 47, 156, 146, 97, 237, 67, 97, 245, 148, 34, 161, 227,
		0, 54, 231, 195, 43, 39, 12, 136, 7, 164, 25, 254, 202, 96, 80, 35, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 67, 25, 5, 83, 88, 232, 97, 123, 12, 70, 53, 61, 3, 156, 218,
		171, 255, 255, 255, 255, 0, 0, 0, 0, 255, 255, 255, 255, 255, 255, 255, 255, 188, 230, 250,
		173, 167, 23, 158, 132, 243, 185, 202, 194, 252, 99, 37, 78, 10, 217, 149, 0, 40, 141, 70,
		105, 64, 3, 29, 114, 169, 245, 68, 90, 77, 67, 120, 70, 64, 133, 91, 240, 166, 152, 116,
		210, 222, 95, 225, 3, 197, 1, 30, 110, 242, 196, 45, 205, 80, 213, 211, 210, 159, 153, 174,
		110, 186, 44, 128, 201, 36, 79, 76, 84, 34, 240, 151, 159, 240, 195, 186, 94}

	// all 0 is an invalid secp256r1 signature verification input.
	invalidInput := make([]byte, len(validInput))

	tests := map[string]struct {
		upgrades       opera.Upgrades
		input          []byte
		expectedStatus uint64
	}{
		"Sonic": {
			upgrades: opera.GetSonicUpgrades(),
			input:    validInput,
			// precompile doesn't exist yet, a call to an empty contract returns no return data.
			expectedStatus: types.ReceiptStatusFailed,
		},
		"Allegro": {
			upgrades: opera.GetAllegroUpgrades(),
			input:    validInput,
			// precompile doesn't exist yet, a call to an empty contract returns no return data.
			expectedStatus: types.ReceiptStatusFailed,
		},
		"Brio/Invalid": {
			upgrades:       opera.GetBrioUpgrades(),
			input:          invalidInput,
			expectedStatus: types.ReceiptStatusFailed,
		},
		"Brio/Valid": {
			upgrades:       opera.GetBrioUpgrades(),
			input:          validInput,
			expectedStatus: types.ReceiptStatusSuccessful,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			recipient := common.BytesToAddress([]byte{0x42})
			net := StartIntegrationTestNetWithJsonGenesis(t, IntegrationTestNetOptions{
				Upgrades: &test.upgrades,
				Accounts: []makefakegenesis.Account{
					{
						Address: recipient,
						Code:    secp256r1TestCode(),
					},
				},
			})

			client, err := net.GetClient()
			require.NoError(t, err)
			defer client.Close()

			sender := MakeAccountWithBalance(t, net, big.NewInt(1e18))
			gasPrice, err := client.SuggestGasPrice(t.Context())
			require.NoError(t, err)

			chainId, err := client.ChainID(t.Context())
			require.NoError(t, err)

			txData := &types.AccessListTx{
				ChainID:  chainId,
				Nonce:    0,
				GasPrice: gasPrice,
				Gas:      50_000,
				To:       &recipient,
				Data:     test.input,
			}
			tx := SignTransaction(t, chainId, txData, sender)

			receipt, err := net.Run(tx)
			require.NoError(t, err)

			require.Equal(t, test.expectedStatus, receipt.Status)
		})
	}
}

func secp256r1TestCode() []byte {
	// handle input data
	code := []byte{
		byte(vm.CALLDATASIZE), // input data size
		byte(vm.PUSH1), 0x00,  // offset from which the input data is copied
		byte(vm.PUSH1), 0x00, // dest offset
		byte(vm.CALLDATACOPY), // copy the input data to memory
	}

	// set up Call
	code = append(code, []byte{
		byte(vm.PUSH1), 0x01, // return data size
		byte(vm.PUSH1), 0x00, // return data offset
		byte(vm.PUSH1), 0xA0, // argument size
		byte(vm.PUSH1), 0x00, // argument offset
		byte(vm.PUSH1), 0x00, // value
		byte(vm.PUSH20), // push address of precompile
	}...)
	code = append(code, common.BytesToAddress([]byte{0x1, 0x00}).Bytes()...)
	code = append(code, []byte{
		byte(vm.PUSH2), 0x1a, 0xf4, // gas price of precompile, 6900
		byte(vm.CALL), // call to precompile
	}...)

	// handle return value
	code = append(code, []byte{
		byte(vm.PUSH1), 0x00, // return data offset
		byte(vm.PUSH1), 0x01, // return data size
		byte(vm.PUSH1), 0x00, // destination offset
		byte(vm.RETURNDATACOPY), // copy the return data to memory
		byte(vm.PUSH1), 0x00,    // offset
		byte(vm.MLOAD), // load return value
	}...)

	// jump based on return value
	// if return value is 1, jump to stop
	// if return value is 0, continue to revert
	code = append(code, []byte{
		byte(vm.PUSH1), 0x37, // push offset jump destination
		byte(vm.JUMPI),    // jump if success (1)
		byte(vm.REVERT),   // revert if failure (0)
		byte(vm.JUMPDEST), // jump destination
		byte(vm.STOP),     // stop execution
	}...)

	return code
}
