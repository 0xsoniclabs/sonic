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
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	accessCost "github.com/0xsoniclabs/sonic/tests/contracts/access_cost"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/stretchr/testify/require"
)

// TestCreateAccessList_TreatsCalldataInInputAndDataAlike checks that
// eth_createAccessList gives the same answer whether the calldata is sent as
// "input" or as "data", also when the caller omits "gas" and the method has
// to estimate it.
func TestCreateAccessList_TreatsCalldataInInputAndDataAlike(t *testing.T) {
	session := getIntegrationTestNetSession(t, opera.GetSonicUpgrades())
	t.Parallel()

	client, err := session.GetClient()
	require.NoError(t, err)
	defer client.Close()

	_, receipt, err := DeployContract(session, accessCost.DeployAccessCost)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
	contract := receipt.ContractAddress

	external := common.Address{0x42}
	abi, err := accessCost.AccessCostMetaData.GetAbi()
	require.NoError(t, err)
	call, err := abi.Pack("getAddressAccessCost", external)
	require.NoError(t, err)

	// The init code reads the balance of an external account and deploys
	// a contract without code.
	creation := append(append(
		[]byte{byte(vm.PUSH20)}, external.Bytes()...),
		byte(vm.BALANCE), byte(vm.POP),
		byte(vm.STOP),
	)

	sender := MakeAccountWithBalance(t, session, big.NewInt(1e18)).Address()
	wantList := types.AccessList{{Address: external, StorageKeys: []common.Hash{}}}

	targets := map[string]struct {
		to       *common.Address
		calldata []byte
	}{
		"call":              {to: &contract, calldata: call},
		"contract creation": {calldata: creation},
	}

	fieldSets := map[string][]string{
		"input":          {"input"},
		"data":           {"data"},
		"input and data": {"input", "data"},
	}

	gasLimits := map[string]*hexutil.Uint64{
		"with gas":    AsPointer(hexutil.Uint64(1_000_000)),
		"without gas": nil,
	}

	for targetName, target := range targets {
		for fieldName, fields := range fieldSets {
			for gasName, gas := range gasLimits {
				t.Run(targetName+" with calldata in "+fieldName+" "+gasName, func(t *testing.T) {
					t.Parallel()

					args := map[string]any{"from": sender}
					if target.to != nil {
						args["to"] = *target.to
					}
					for _, field := range fields {
						args[field] = hexutil.Bytes(target.calldata)
					}
					if gas != nil {
						args["gas"] = *gas
					}

					var res struct {
						AccessList types.AccessList `json:"accessList"`
						Error      string           `json:"error,omitempty"`
					}
					err := client.Client().Call(&res, "eth_createAccessList", args, "latest")
					require.NoError(t, err, "eth_createAccessList failed")
					require.Empty(t, res.Error, "call reported an execution error")
					require.Equal(t, wantList, res.AccessList)
				})
			}
		}
	}

	t.Run("different calldata in input and data is rejected", func(t *testing.T) {
		t.Parallel()

		args := map[string]any{
			"from":  sender,
			"to":    contract,
			"input": hexutil.Bytes(call),
			"data":  hexutil.Bytes{0x01},
		}
		var res any
		err := client.Client().Call(&res, "eth_createAccessList", args, "latest")
		require.ErrorContains(t, err, `both "data" and "input" are set and not equal`)
	})
}
