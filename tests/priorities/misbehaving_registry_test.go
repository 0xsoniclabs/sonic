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
	modeMutating
)

func TestPriorities_MisbehavingRegistry_LeavesTransactionsUnprioritized(t *testing.T) {
	cases := map[string]struct {
		mode, configMode  uint8
		expectPrioritized bool
	}{
		"getPriority and getPriorityConfig correct":            {modeCorrect, modeCorrect, true},
		"getPriority needs more gas than it is granted":        {modeOutOfGas, modeCorrect, false},
		"getPriority returns values out of range":              {modeOutOfRange, modeCorrect, false},
		"getPriority returns fewer values than expected":       {modeTruncated, modeCorrect, false},
		"getPriority reverts":                                  {modeReverting, modeCorrect, false},
		"getPriorityConfig needs more gas than it is granted":  {modeCorrect, modeOutOfGas, false},
		"getPriorityConfig returns values out of range":        {modeCorrect, modeOutOfRange, false},
		"getPriorityConfig returns fewer values than expected": {modeCorrect, modeTruncated, false},
		"getPriorityConfig reverts":                            {modeCorrect, modeReverting, false},
	}

	net, client, _ := netClientSignerWithPriorities(t, nil)
	defer client.Close()

	reg := installMisbehavingRegistry(t, net, client)
	magicValue, err := reg.MAGICVALUE(nil)
	require.NoError(t, err)

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			setModes(t, net, reg, tc.mode, tc.configMode)

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

func TestPriorities_RegistryMutatesState_ChangesDoNotPersist(t *testing.T) {
	require := require.New(t)

	net, client, _ := netClientSignerWithPriorities(t, nil)
	defer client.Close()

	reg := installMisbehavingRegistry(t, net, client)
	setModes(t, net, reg, modeMutating, modeMutating)

	magicValue, err := reg.MAGICVALUE(nil)
	require.NoError(err)

	sender := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))
	txs := make([]*types.Transaction, 5)
	for i := range txs {
		txs[i] = newSignedTx(t, net, sender, uint64(i), magicValue.Uint64(), 21_000, nil)
	}

	txHasMagicValue := func(tx *types.Transaction) bool { return tx.Value().Cmp(magicValue) == 0 }
	requirePriorityHasEffect(t, net, txs, true, txHasMagicValue)

	// The writes were discarded with the snapshot wrapping each query.
	mutated, err := reg.Mutated(nil)
	require.NoError(err)
	require.False(mutated)

	// Running the very same query as a transaction does persist the write, so
	// the check above observes the rollback and not a registry that never wrote.
	receipt, err := net.Apply(reg.GetPriorityConfig)
	require.NoError(err)
	require.Equal(types.ReceiptStatusSuccessful, receipt.Status)
	mutated, err = reg.Mutated(nil)
	require.NoError(err)
	require.True(mutated)
}

// installMisbehavingRegistry deploys a MisbehavingPriorityRegistry and points
// the priority-registry proxy at it, returning a binding to the proxy.
func installMisbehavingRegistry(
	t *testing.T,
	net *tests.IntegrationTestNet,
	client *tests.PooledEhtClient,
) *misbehaving_priority_registry.MisbehavingPriorityRegistry {
	t.Helper()
	require := require.New(t)

	_, deployReceipt, err := tests.DeployContract(net,
		misbehaving_priority_registry.DeployMisbehavingPriorityRegistry)
	require.NoError(err)
	require.Equal(types.ReceiptStatusSuccessful, deployReceipt.Status)
	switchPriorityRegistry(t, net, deployReceipt.ContractAddress)

	reg, err := misbehaving_priority_registry.NewMisbehavingPriorityRegistry(
		registry.GetAddress(), client)
	require.NoError(err)
	return reg
}

// setModes selects how the registry answers the node's getPriority and
// getPriorityConfig queries.
func setModes(
	t *testing.T,
	net *tests.IntegrationTestNet,
	reg *misbehaving_priority_registry.MisbehavingPriorityRegistry,
	mode, configMode uint8,
) {
	t.Helper()
	require := require.New(t)

	for _, set := range []func(*bind.TransactOpts) (*types.Transaction, error){
		func(opts *bind.TransactOpts) (*types.Transaction, error) { return reg.SetMode(opts, mode) },
		func(opts *bind.TransactOpts) (*types.Transaction, error) { return reg.SetConfigMode(opts, configMode) },
	} {
		receipt, err := net.Apply(set)
		require.NoError(err)
		require.Equal(types.ReceiptStatusSuccessful, receipt.Status)
	}
}
