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

	priorityregistry "github.com/0xsoniclabs/sonic/gossip/blockproc/priorities/registry"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// newFundedPrioritizedAccount creates a funded account and registers it in the
// priority registry with (level, weight, id).
func newFundedPrioritizedAccount(
	t *testing.T,
	net *tests.IntegrationTestNet,
	level, weight, id uint64,
) *tests.Account {
	t.Helper()
	acc := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))
	setPrioritized(t, net, acc.Address(), level, weight, id)
	return acc
}

// setPrioritized registers `sender` in the priority registry with
// (level, weight, id).
func setPrioritized(
	t *testing.T,
	net *tests.IntegrationTestNet,
	sender common.Address,
	level, weight, id uint64,
) {
	t.Helper()
	require := require.New(t)

	client, err := net.GetClient()
	require.NoError(err)
	defer client.Close()

	reg, err := priorityregistry.NewRegistry(priorityregistry.GetAddress(), client)
	require.NoError(err)

	receipt, err := net.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return reg.SetSenderPriority(opts, sender, level, weight, new(big.Int).SetUint64(id))
	})
	require.NoError(err)
	require.Equal(types.ReceiptStatusSuccessful, receipt.Status)
}

// newSignedTx builds and signs transfer from `account`. If `gasPrice` is nil,
// the network's currently suggested gas price is used.
func newSignedTx(
	t *testing.T,
	net *tests.IntegrationTestNet,
	account *tests.Account,
	nonce uint64,
	gasLimit uint64,
	gasPrice *big.Int,
) *types.Transaction {
	t.Helper()
	require := require.New(t)

	if gasPrice == nil {
		client, err := net.GetClient()
		require.NoError(err)
		defer client.Close()

		gasPrice, err = client.SuggestGasPrice(t.Context())
		require.NoError(err)
	}

	signer := types.LatestSignerForChainID(net.GetChainId())
	return types.MustSignNewTx(account.PrivateKey, signer, &types.LegacyTx{
		Nonce:    nonce,
		To:       &common.Address{0x99},
		Value:    big.NewInt(1),
		Gas:      gasLimit,
		GasPrice: gasPrice,
	})
}
