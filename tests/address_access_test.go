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

	accessCost "github.com/0xsoniclabs/sonic/tests/contracts/access_cost"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestAddressAccess(t *testing.T) {
	someAccountAddress := common.Address{1}

	net := StartIntegrationTestNet(t)

	contract, receipt, err := DeployContract(net, accessCost.DeployAccessCost)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// Execute function on an address, cold access
	receipt, err = net.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return contract.TouchAddress(opts, someAccountAddress)
	})
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	txColdAccess, err := contract.ParseLogCost(*receipt.Logs[0])
	require.NoError(t, err)
	_, viewColdAccess, err := contract.GetAddressAccessCost(nil, someAccountAddress)
	require.NoError(t, err)

	t.Run("coinbase yields zero address", func(t *testing.T) {
		coinBaseAddress, err := contract.GetCoinBaseAddress(nil)
		require.NoError(t, err)
		require.Equal(t, common.Address{}, coinBaseAddress)
	})

	t.Run("tx access is warm", func(t *testing.T) {
		tests := map[string]func(*bind.TransactOpts) (*types.Transaction, error){
			"coinbase": contract.TouchCoinBase,
			"origin":   contract.TouchOrigin,
			"access list": func(ops *bind.TransactOpts) (*types.Transaction, error) {
				ops.GasPrice = nil // < transactions with gas price cannot have access list
				ops.GasFeeCap = big.NewInt(1e12)
				ops.GasTipCap = big.NewInt(1000)
				ops.AccessList = types.AccessList{
					{Address: someAccountAddress},
				}
				return contract.TouchAddress(ops, someAccountAddress)
			},
		}

		for name, access := range tests {
			t.Run(name, func(t *testing.T) {
				receipt, err = net.Apply(access)
				require.NoError(t, err)
				require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

				warmAccess, err := contract.ParseLogCost(*receipt.Logs[0])
				require.NoError(t, err)

				// Difference must be the extra cost of a cold access
				diff := new(big.Int).Sub(txColdAccess.Cost, warmAccess.Cost)
				require.Equal(t, big.NewInt(2500), diff, "Expected cost difference of 2500 for warm access")
			})
		}
	})

	t.Run("archive access is warm", func(t *testing.T) {

		tests := map[string]func(t *testing.T) (*big.Int, error){
			"origin": func(t *testing.T) (*big.Int, error) {
				originAddr, err := contract.GetOrigin(nil)
				require.NoError(t, err)
				_, cost, err := contract.GetAddressAccessCost(nil, originAddr)
				return cost, err
			},
			"coinbase": func(t *testing.T) (*big.Int, error) {
				coinbaseAddr, err := contract.GetCoinBaseAddress(nil)
				require.NoError(t, err)
				_, cost, err := contract.GetAddressAccessCost(nil, coinbaseAddr)
				return cost, err
			},
		}

		for name, access := range tests {
			t.Run(name, func(t *testing.T) {
				cost, err := access(t)
				require.NoError(t, err)
				diff := new(big.Int).Sub(viewColdAccess, cost)
				require.Equal(t, big.NewInt(2500), diff, "Expected cost difference of 2500 for warm access")
			})
		}
	})
}
