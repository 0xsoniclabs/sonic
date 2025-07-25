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
	"github.com/0xsoniclabs/sonic/tests/contracts/data_reader"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEstimateGas(t *testing.T) {
	t.Run("Sonic", func(t *testing.T) {
		session := getIntegrationTestNetSession(t, opera.GetSonicUpgrades())
		t.Parallel()

		dataContract, receipt, err := DeployContract(session, data_reader.DeployDataReader)
		require.NoError(t, err, "failed to deploy contract; %v", err)
		require.Equal(t, receipt.Status, types.ReceiptStatusSuccessful)
		dataContractAddress := receipt.ContractAddress

		doTestEstimate(t, session, makeTestCases(t, session, dataContract, dataContractAddress))
	})
	t.Run("Allegro", func(t *testing.T) {
		session := getIntegrationTestNetSession(t, opera.GetAllegroUpgrades())
		t.Parallel()

		dataContract, receipt, err := DeployContract(session, data_reader.DeployDataReader)
		require.NoError(t, err, "failed to deploy contract; %v", err)
		require.Equal(t, receipt.Status, types.ReceiptStatusSuccessful)
		dataContractAddress := receipt.ContractAddress

		doTestEstimate(t, session, makeAllegroCases(t, session, dataContract, dataContractAddress))
	})
}

func makeTestCases(
	t *testing.T,
	session IntegrationTestNetSession,
	contract *data_reader.DataReader,
	contractAddress common.Address,
) map[string]types.TxData {

	largeCallDataZeros := getCallData(t, session, func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return contract.SendData(opts, make([]byte, 40_000))
	})
	largeCallDataOnes := getCallData(t, session, func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return contract.SendData(opts, []byte{0xFF, 40_000: 0xFF})
	})
	smallCallData := getCallData(t, session, func(opts *bind.TransactOpts) (*types.Transaction, error) {
		// although the contract function receives no parameters, call data will still
		// contain the function selector and the empty data, so it will not be empty.
		return contract.SendData(opts, nil)
	})

	return map[string]types.TxData{
		"do nothing": &types.LegacyTx{},
		"simple transfer": &types.LegacyTx{
			To:    &common.Address{0x42},
			Value: big.NewInt(1),
		},
		"create": &types.LegacyTx{
			To: nil,
			// one Stop instruction followed by a some data
			Data: []byte{0x0, 100: 0xFF},
		},
		"create with long data": &types.LegacyTx{
			To: nil,
			// one Stop instruction followed by a some more data
			Data: []byte{0x0, 40_000: 0xFF},
		},
		"access list": &types.AccessListTx{
			To:    &common.Address{0x42},
			Value: big.NewInt(1),
			AccessList: types.AccessList{
				{
					Address:     common.Address{0x42},
					StorageKeys: []common.Hash{{0x01}},
				},
			},
		},
		"call contract with small data": &types.LegacyTx{
			To:   &contractAddress,
			Data: smallCallData,
		},
		"call with large data with zeros": &types.LegacyTx{
			To:   &contractAddress,
			Data: largeCallDataZeros,
		},
		"call with large data with ones": &types.LegacyTx{
			To:   &contractAddress,
			Data: largeCallDataOnes,
		},
	}
}

func makeAllegroCases(
	t *testing.T,
	session IntegrationTestNetSession,
	contract *data_reader.DataReader,
	contractAddress common.Address,
) map[string]types.TxData {

	// Allegro executes all test cases for Sonic as well.
	cases := makeTestCases(t, session, contract, contractAddress)

	// create authorization, use new account to avoid altering session sponsor state
	account := makeAccountWithBalance(t, session, big.NewInt(1e18))
	auth, err := types.SignSetCode(account.PrivateKey,
		types.SetCodeAuthorization{
			ChainID: *uint256.MustFromBig(session.GetChainId()),
			Address: contractAddress,
			Nonce:   1,
		})
	require.NoError(t, err, "failed to create authorization; %v", err)

	callData := getCallData(t, session, func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return contract.SendData(opts, make([]byte, 128))
	})

	cases["authorizations"] = &types.SetCodeTx{
		To:       account.Address(),
		Data:     callData,
		AuthList: []types.SetCodeAuthorization{auth},
	}
	cases["multiple authorizations"] = &types.SetCodeTx{
		To:       account.Address(),
		Data:     callData,
		AuthList: []types.SetCodeAuthorization{auth, auth},
	}
	return cases
}

func doTestEstimate(
	t *testing.T,
	session IntegrationTestNetSession,
	tests map[string]types.TxData) {

	account := makeAccountWithBalance(t, session, big.NewInt(1e18))
	netUpgrades := session.GetUpgrades()

	client, err := session.GetClient()
	require.NoError(t, err, "Failed to get client")
	defer client.Close()

	for name, txData := range tests {
		t.Run(name, func(t *testing.T) {

			tmpTx := types.NewTx(txData)
			// first compute intrinsic gas: if intrinsic gas fails, estimate
			// will fail too
			intrinsicGas, err := core.IntrinsicGas(
				tmpTx.Data(),
				tmpTx.AccessList(),
				tmpTx.SetCodeAuthorizations(),
				tmpTx.To() == nil,
				true, true, netUpgrades.Allegro)
			require.NoError(t, err, "Failed to calculate intrinsic gas")

			// estimate gas used by the message
			gasEstimation, err := client.EstimateGas(t.Context(),
				ethereum.CallMsg{
					From:              account.Address(),
					To:                tmpTx.To(),
					Data:              tmpTx.Data(),
					AccessList:        tmpTx.AccessList(),
					AuthorizationList: tmpTx.SetCodeAuthorizations(),
				})
			require.NoError(t, err)
			assert.GreaterOrEqual(t, gasEstimation, intrinsicGas,
				"Gas estimation should be greater than or equal to intrinsic gas")

			// execute the transaction, with the estimated gas
			gasPrice, err := client.SuggestGasPrice(t.Context())
			require.NoError(t, err, "Failed to suggest gas price")

			nonce, err := client.PendingNonceAt(t.Context(), account.Address())
			require.NoError(t, err, "Failed to get pending nonce")

			switch txData := txData.(type) {
			case *types.LegacyTx:
				txData.Gas = gasEstimation
				txData.GasPrice = gasPrice
				txData.Nonce = nonce
			case *types.AccessListTx:
				txData.Gas = gasEstimation
				txData.GasPrice = gasPrice
				txData.Nonce = nonce
			case *types.SetCodeTx:
				txData.Gas = gasEstimation
				txData.GasFeeCap = uint256.MustFromBig(gasPrice)
				txData.Nonce = nonce
			default:
				t.Fatalf("Not implemented transaction type: %T", txData)
			}

			tx, err := types.SignNewTx(account.PrivateKey,
				types.LatestSignerForChainID(session.GetChainId()), txData)
			require.NoError(t, err, "Failed to sign transaction")

			receipt, err := session.Run(tx)
			require.NoError(t, err)
			require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status,
				"Transaction result shall be successful")

			gasUsed := receipt.GasUsed
			assert.LessOrEqual(t, gasUsed, gasEstimation,
				"Gas used shall be less than or equal to gas estimation")
		})
	}
}
