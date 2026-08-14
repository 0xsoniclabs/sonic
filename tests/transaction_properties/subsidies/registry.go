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

package subsidies

import (
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/proxy"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies/registry"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/tests/contracts/network_sponsor"
	"github.com/0xsoniclabs/sonic/tests/contracts/network_sponsor_tracking"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// InstallRegistry points the registry proxy at an implementation answering in the given mode. The
// genesis of a network with gas subsidies enabled already carries the fund-backed reference registry,
// so only the network-sponsored modes need installing -- and each of those answers the same mode for
// every caller, which is what keeps them free of per-iteration setup.
func InstallRegistry(t *testing.T, net *tests.IntegrationTestNet, mode Mode) {
	t.Helper()

	var deploy func(*bind.TransactOpts, bind.ContractBackend) (*types.Transaction, error)
	switch mode {
	case ModeFundBacked:
		return // the reference registry deployed in genesis
	case ModeNetwork:
		deploy = func(opts *bind.TransactOpts, backend bind.ContractBackend) (*types.Transaction, error) {
			_, tx, _, err := network_sponsor.DeployNetworkSponsor(opts, backend)
			return tx, err
		}
	case ModeNetworkTracked:
		deploy = func(opts *bind.TransactOpts, backend bind.ContractBackend) (*types.Transaction, error) {
			_, tx, _, err := network_sponsor_tracking.DeployNetworkSponsorTracking(opts, backend)
			return tx, err
		}
	default:
		t.Fatalf("no registry implements mode %v", mode)
	}

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	receipt, err := net.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return deploy(opts, client)
	})
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status, "the registry must deploy")

	proxyContract, err := proxy.NewProxy(registry.GetAddress(), client)
	require.NoError(t, err)

	update, err := net.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return proxyContract.Update(opts, receipt.ContractAddress)
	})
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, update.Status, "the proxy must accept the update")
}
