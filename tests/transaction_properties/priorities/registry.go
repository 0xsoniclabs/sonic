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
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/priorities/registry"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/proxy"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/tests/contracts/nonce_priority_registry"
	"github.com/0xsoniclabs/sonic/tests/transaction_properties/core"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// gasPerWindowRegistration is what a window is registered with, set rather than estimated: an
// estimate is one round trip against state the archive may not have ingested yet, and what the call
// costs is known -- one storage slot per nonce slot of the window.
const gasPerWindowRegistration = 300_000

// InstallRegistry points the priority registry proxy at an implementation that answers per
// (sender, nonce) rather than per sender. The genesis of a network with transaction priorities
// enabled carries the development registry, which keys a priority on the sender alone -- so with
// that one installed, every transaction of a sender would share one priority and the per-sender
// nonce rule could never be reached.
//
// Everything afterwards goes through the proxy at the well-known address, never through the
// implementation: the proxy delegates, so the storage a registration writes and the storage
// getPriority reads are the proxy's, and a registration sent to the implementation itself would be
// written where nothing ever looks.
func InstallRegistry(t *testing.T, net *tests.IntegrationTestNet) {
	t.Helper()

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	receipt, err := net.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
		_, tx, _, err := nonce_priority_registry.DeployNoncePriorityRegistry(opts, client)
		return tx, err
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

	implementation, err := client.StorageAt(
		t.Context(), registry.GetAddress(), proxy.GetSlotForImplementation(), nil)
	require.NoError(t, err)
	require.Equal(t, receipt.ContractAddress, common.BytesToAddress(implementation),
		"the proxy must point at the installed registry")

	// The rate limits are set once and never drawn -- see the note on perEntityBudget.
	config, err := net.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return bound(t, client).SetConfig(opts,
			new(big.Int).SetUint64(perEntityBudget), new(big.Int).SetUint64(maxPiggybackTxs))
	})
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, config.Status,
		"the registry must accept the rate limits")
}

// bound binds the registry through the proxy at the well-known address.
func bound(t *testing.T, client *tests.PooledEhtClient) *nonce_priority_registry.NoncePriorityRegistry {
	t.Helper()
	registry, err := nonce_priority_registry.NewNoncePriorityRegistry(registry.GetAddress(), client)
	require.NoError(t, err)
	return registry
}

// Window is the priorities of one sender's next nonce slots, the first of them at FirstNonce.
type Window struct {
	Account    common.Address
	FirstNonce uint64
	Slots      []Priority
}

// Registrar writes into the installed registry: the per-entity gas budget every block is formed
// against, and the priority of each nonce slot a batch can reach.
type Registrar struct {
	registry *nonce_priority_registry.NoncePriorityRegistry
	session  tests.IntegrationTestNetSession
	client   *tests.PooledEhtClient
	chainId  *big.Int

	// clerk pays for every registration. It is one account of the pool, claimed for the life of the
	// network and never released, so it is never an account under observation, and stocked on claim,
	// since the pool may hand over one that earlier iterations have drained -- and never the session
	// sponsor, which drives every block of the barrier and would be drained by a run of a few hundred
	// iterations registering several windows apiece.
	clerk core.PooledAccount
}

// NewRegistrar binds the registry through the proxy at the well-known address and claims the
// account that pays for the registrations.
func NewRegistrar(
	session tests.IntegrationTestNetSession,
	client *tests.PooledEhtClient,
	network *core.Network,
) (*Registrar, error) {

	bound, err := nonce_priority_registry.NewNoncePriorityRegistry(registry.GetAddress(), client)
	if err != nil {
		return nil, fmt.Errorf("failed to bind the priority registry: %w", err)
	}

	clerk, err := network.ClaimPayer()
	if err != nil {
		return nil, fmt.Errorf("failed to claim the account paying for the registrations: %w", err)
	}

	return &Registrar{
		registry: bound,
		session:  session,
		client:   client,
		chainId:  network.Cfg.ChainId,
		clerk:    clerk,
	}, nil
}

// Register writes every window, sending all of them before waiting for any, so an iteration's
// registrations cost about one block rather than one apiece.
//
// A window is written whether or not anything in it is prioritized: an account comes back from the
// pool at the nonce it left, so a slot this iteration means to leave alone may still hold what an
// earlier iteration registered for it, and only writing it again makes the chain agree with the
// model about what it holds.
func (r *Registrar) Register(ctx context.Context, windows []Window) error {
	nonce, err := r.client.PendingNonceAt(ctx, r.clerk.Address())
	if err != nil {
		return fmt.Errorf("failed to read the nonce of the registering account: %w", err)
	}

	opts := func() (*bind.TransactOpts, error) {
		opts, err := bind.NewKeyedTransactorWithChainID(r.clerk.PrivateKey, r.chainId)
		if err != nil {
			return nil, fmt.Errorf("failed to build a transactor for the registering account: %w", err)
		}
		opts.Nonce = new(big.Int).SetUint64(nonce)
		opts.GasLimit = gasPerWindowRegistration
		opts.NoSend = true
		nonce++
		return opts, nil
	}

	txs := make([]*types.Transaction, 0, len(windows))
	for _, window := range windows {
		windowOpts, err := opts()
		if err != nil {
			return err
		}
		slots := make([]nonce_priority_registry.NoncePriorityRegistryPriority, len(window.Slots))
		for i, slot := range window.Slots {
			slots[i] = nonce_priority_registry.NoncePriorityRegistryPriority{
				Level:  slot.Level,
				Weight: slot.Weight,
				Id:     new(big.Int).SetUint64(slot.Entity),
			}
		}
		tx, err := r.registry.SetPriorities(windowOpts,
			window.Account, new(big.Int).SetUint64(window.FirstNonce), slots)
		if err != nil {
			return fmt.Errorf("failed to build the registration of %v: %w", window.Account, err)
		}
		txs = append(txs, tx)
	}

	receipts, err := r.session.RunAll(txs)
	if err != nil {
		return fmt.Errorf("failed to register %d window(s): %w", len(windows), err)
	}
	for i, receipt := range receipts {
		if receipt.Status != types.ReceiptStatusSuccessful {
			return fmt.Errorf("registration %d of %d failed with status %d",
				i, len(receipts), receipt.Status)
		}
	}
	return nil
}
