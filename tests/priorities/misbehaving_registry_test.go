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

package priorities

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/priorities/registry"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/tests/contracts/misbehaving_priority_registry"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

const (
	modeCorrect uint8 = iota
	modeOutOfGas
	modeOutOfRange
	modeTruncated
	modeReverting
)

func TestPriorities_MisbehavingRegistryLeavesTransactionsUnprioritized(t *testing.T) {
	cases := map[string]struct {
		mode              uint8
		expectPrioritized bool
	}{
		"correct":                            {modeCorrect, true},
		"needs more gas than it is granted":  {modeOutOfGas, false},
		"returns values out of range":        {modeOutOfRange, false},
		"returns fewer values than expected": {modeTruncated, false},
		"reverts":                            {modeReverting, false},
	}

	net, client, _ := netClientSignerWithPriorities(t, nil)
	defer client.Close()

	_, deployReceipt, err := tests.DeployContract(net,
		misbehaving_priority_registry.DeployMisbehavingPriorityRegistry)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, deployReceipt.Status)
	switchPriorityRegistry(t, net, deployReceipt.ContractAddress)

	reg, err := misbehaving_priority_registry.NewMisbehavingPriorityRegistry(
		registry.GetAddress(), client)
	require.NoError(t, err)
	magicValue, err := reg.MAGICVALUE(nil)
	require.NoError(t, err)

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)

			receipt, err := net.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
				return reg.SetMode(opts, tc.mode)
			})
			require.NoError(err)
			require.Equal(types.ReceiptStatusSuccessful, receipt.Status)

			sender := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))
			txs := make([]*types.Transaction, 5)
			for i := range txs {
				txs[i] = newSignedTx(t, net, sender, uint64(i), magicValue.Uint64(), 21_000, nil)
			}

			txHasMagicValue := func(tx *types.Transaction) bool { return tx.Value().Cmp(magicValue) == 0 }
			// Whether or not the classification reaches the node, all
			// transactions are executed successfully.
			requirePriorityHasEffect(t, net, txs, tc.expectPrioritized, txHasMagicValue)
		})
	}
}
