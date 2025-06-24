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
	"testing"

	"github.com/0xsoniclabs/sonic/tests/contracts/invalidstart"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestInvalidStart_IdentifiesInvalidStartContract(t *testing.T) {
	require := require.New(t)
	// byteccode source: https://eips.ethereum.org/EIPS/eip-3541#test-cases
	invalidCode := []byte{0x60, 0xef, 0x60, 0x00, 0x53, 0x60, 0x01, 0x60, 0x00, 0xf3}
	validCode := []byte{0x60, 0xfe, 0x60, 0x00, 0x53, 0x60, 0x01, 0x60, 0x00, 0xf3}

	net := StartIntegrationTestNet(t)

	// Deploy the invalid start contract.
	contract, _, err := DeployContract(net, invalidstart.DeployInvalidstart)
	require.NoError(err)

	// -- invalid codes

	// attempt to create a contract with code starting with 0xEF using CREATE
	receipt, err := net.Apply(contract.CreateContractWithInvalidCode)
	require.NoError(err)
	require.Equal(types.ReceiptStatusFailed, receipt.Status, "unexpected succeeded on invalid code with CREATE")

	// attempt to create a contract with code starting with 0xEF using CREATE2
	receipt, err = net.Apply(contract.Create2ContractWithInvalidCode)
	require.NoError(err)
	require.Equal(types.ReceiptStatusFailed, receipt.Status, "unexpected succeeded on invalid code with CREATE2")

	// attempt to run a transaction without receiver, with an invalid code.
	invalidTransaction, err := getTransactionWithCodeAndNoReceiver(t, invalidCode, net)
	require.NoError(err)
	receipt, err = net.Run(invalidTransaction)
	require.NoError(err)
	require.Equal(types.ReceiptStatusFailed, receipt.Status, "unexpected succeeded on transfer to empty receiver with invalid code")

	// -- valid codes

	// create a contract with valid code using CREATE
	receipt, err = net.Apply(contract.CreateContractWithValidCode)
	require.NoError(err)
	require.Equal(types.ReceiptStatusSuccessful, receipt.Status, "failed on valid code with CREATE")

	// create a contract with valid code using CREATE2
	receipt, err = net.Apply(contract.Create2ContractWithValidCode)
	require.NoError(err)
	require.Equal(types.ReceiptStatusSuccessful, receipt.Status, "failed on valid code with CREATE2")

	// run a transaction without receiver, with a valid code.
	validTransaction, err := getTransactionWithCodeAndNoReceiver(t, validCode, net)
	require.NoError(err)
	receipt, err = net.Run(validTransaction)
	require.NoError(err)
	require.Equal(types.ReceiptStatusSuccessful, receipt.Status, "failed on transfer to empty receiver with valid code")
}

func getTransactionWithCodeAndNoReceiver(t testing.TB, code []byte, net *IntegrationTestNet) (*types.Transaction, error) {
	// these values are needed for the transaction but are irrelevant for the test
	t.Helper()
	require := require.New(t)
	client, err := net.GetClient()
	require.NoError(err, "failed to connect to the network:")

	defer client.Close()
	chainId, err := client.ChainID(t.Context())
	require.NoError(err, "failed to get chain ID::")

	nonce, err := client.NonceAt(t.Context(), net.GetSessionSponsor().Address(), nil)
	require.NoError(err, "failed to get nonce:")

	price, err := client.SuggestGasPrice(t.Context())
	require.NoError(err, "failed to get gas price:")
	// ---------

	transaction, err := types.SignTx(types.NewTx(&types.AccessListTx{
		ChainID:  chainId,
		Gas:      500_000, // some gas that is big enough to run the code
		GasPrice: price,
		To:       nil,
		Nonce:    nonce,
		Data:     code,
	}), types.NewLondonSigner(chainId), net.GetSessionSponsor().PrivateKey)
	require.NoError(err, "failed to sign transaction:")

	return transaction, nil
}
