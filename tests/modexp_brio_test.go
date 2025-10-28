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

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestModExp_BrioFlagEnforcesFusakaUpperBounds(t *testing.T) {
	tests := map[string]struct {
		upgrades       opera.Upgrades
		size           int
		expectedStatus uint64
	}{
		"Sonic/1024": {
			upgrades:       opera.GetSonicUpgrades(),
			size:           1024,
			expectedStatus: types.ReceiptStatusSuccessful,
		},
		"Sonic/1025": {
			upgrades:       opera.GetSonicUpgrades(),
			size:           1025,
			expectedStatus: types.ReceiptStatusSuccessful,
		},
		"Allegro/1024": {
			upgrades:       opera.GetAllegroUpgrades(),
			size:           1024,
			expectedStatus: types.ReceiptStatusSuccessful,
		},
		"Allegro/1025": {
			upgrades:       opera.GetAllegroUpgrades(),
			size:           1025,
			expectedStatus: types.ReceiptStatusSuccessful,
		},
		"Brio/1024": {
			upgrades:       opera.GetBrioUpgrades(),
			size:           1024,
			expectedStatus: types.ReceiptStatusSuccessful,
		},
		"Brio/1025": {
			upgrades:       opera.GetBrioUpgrades(),
			size:           1025,
			expectedStatus: types.ReceiptStatusFailed,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			session := getIntegrationTestNetSession(t, test.upgrades)
			chainId := session.GetChainId()
			sender := session.GetSessionSponsor()

			oneBytes := uint256.NewInt(1).Bytes32()
			sizeBytes := uint256.NewInt(uint64(test.size)).Bytes32()
			input := sizeBytes[:]                             // base length
			input = append(input, oneBytes[:]...)             // exponent length
			input = append(input, oneBytes[:]...)             // modulus length
			input = append(input, make([]byte, test.size)...) // base
			input = append(input, 0x01)                       // exponent
			input = append(input, 0x01)                       // modulus

			modExpAddress := common.HexToAddress("0x05")
			txsPayload := &types.AccessListTx{
				ChainID:    chainId,
				Nonce:      0,
				Gas:        60_000,
				To:         &modExpAddress,
				Value:      big.NewInt(0),
				Data:       input,
				AccessList: types.AccessList{},
			}
			signedTx := CreateTransaction(t, session, txsPayload, sender)
			receipt, err := session.Run(signedTx)
			require.NoError(t, err)

			require.Equal(t, test.expectedStatus, receipt.Status)
		})
	}
}

func TestModExp_MinimumGasPriceIsUpdatedInBrio(t *testing.T) {
	// Prague added a floor data gas cost which depends on the size of the input data,
	// in order to test the modExp base cost the gas price needs to be higher than the
	// floor cost. A random address is added to the access list to increase the gas cost.

	gasLimit := uint64(21_000)
	gasLimit += 93 * 4 // zero input data cost
	gasLimit += 6 * 16 // non zero input data cost
	gasLimit += 2400   // access list cost

	tests := map[string]struct {
		upgrades opera.Upgrades
		gas      uint64
	}{
		"Sonic": {
			upgrades: opera.GetSonicUpgrades(),
			gas:      gasLimit + 200, // modExp original base cost
		},
		"Allegro": {
			upgrades: opera.GetAllegroUpgrades(),
			gas:      gasLimit + 200, // modExp original base cost
		},
		"Brio": {
			upgrades: opera.GetBrioUpgrades(),
			gas:      gasLimit + 500, // modExp updated base cost
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			session := getIntegrationTestNetSession(t, test.upgrades)
			chainId := session.GetChainId()
			sender := session.GetSessionSponsor()

			oneBytes := uint256.NewInt(1).Bytes32()
			input := oneBytes[:]                  // base length
			input = append(input, oneBytes[:]...) // exponent length
			input = append(input, oneBytes[:]...) // modulus length
			input = append(input, 0x01)           // base
			input = append(input, 0x01)           // exponent
			input = append(input, 0x01)           // modulus

			modExpAddress := common.HexToAddress("0x05")
			txsPayload := &types.AccessListTx{
				ChainID: chainId,
				Nonce:   0,
				Gas:     test.gas + 1, // +1 to ensure there was no error which consumed the gas
				To:      &modExpAddress,
				Value:   big.NewInt(0),
				Data:    input,
				AccessList: types.AccessList{
					{Address: common.HexToAddress("0x42")}, // Add random address to access list to increase gas cost
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

func TestModExp_GasPriceIsUpdatedInBrio(t *testing.T) {
	// Just like the previous minimum gas price test, the gas price needs to be higher than the
	// floor data gas cost which depends on the size of the input data. A random address is
	// added to the access list to increase the gas cost.

	gasLimit := uint64(21_000)
	gasLimit += 93 * 4  // zero input data cost
	gasLimit += 99 * 16 // non zero input data cost
	gasLimit += 2400    // access list cost

	tests := map[string]struct {
		upgrades opera.Upgrades
		gas      uint64
	}{
		"Sonic": {
			upgrades: opera.GetSonicUpgrades(),
			gas:      gasLimit + 1360, // modExp original cost
		},
		"Allegro": {
			upgrades: opera.GetAllegroUpgrades(),
			gas:      gasLimit + 1360, // modExp original cost
		},
		"Brio": {
			upgrades: opera.GetBrioUpgrades(),
			gas:      gasLimit + 4080, // modExp updated cost
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			session := getIntegrationTestNetSession(t, test.upgrades)
			chainId := session.GetChainId()
			sender := session.GetSessionSponsor()

			sizeBytes := uint256.NewInt(32).Bytes32()
			inputBytes := uint256.NewInt(0).Not(uint256.NewInt(0)).Bytes32() // max 32 byte value

			input := sizeBytes[:]                   // base length
			input = append(input, sizeBytes[:]...)  // exponent length
			input = append(input, sizeBytes[:]...)  // modulus length
			input = append(input, inputBytes[:]...) // base
			input = append(input, inputBytes[:]...) // exponent
			input = append(input, inputBytes[:]...) // modulus

			modExpAddress := common.HexToAddress("0x05")
			txsPayload := &types.AccessListTx{
				ChainID: chainId,
				Nonce:   0,
				Gas:     test.gas + 1, // +1 to ensure there was no error which consumed the gas
				To:      &modExpAddress,
				Value:   big.NewInt(0),
				Data:    input,
				AccessList: types.AccessList{
					{Address: common.HexToAddress("0x42")}, // Add random address to access list to increase gas cost
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
