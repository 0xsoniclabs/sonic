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

package bundles

import (
	"context"
	"iter"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// GetUnfilteredNetVariants returns a sequence of different unfiltered network variants to test against.
// Each variant represents a different way of bypassing transaction validation, allowing testing of bundle execution
// in various scenarios where transactions may not be filtered by the TxPool or the emitter.
func GetUnfilteredNetVariants(t testing.TB, upgrades opera.Upgrades) iter.Seq2[string, UnfilteredNet] {
	return func(yield func(string, UnfilteredNet) bool) {
		if !yield("no-intake-validation net", NewNoIntakeValidationIntegrationTestNet(t, upgrades)) {
			return
		}
		if !yield("no-intake-and-emission-validation net", NewNoIntakeAndEmissionValidationTestNet(t, upgrades)) {
			return
		}
	}
}

// UnfilteredNet is an abstraction over different types of test networks
// that can be used to test bundle execution without the transactions
// being filtered by the TxPool or the emitter.
type UnfilteredNet interface {
	tests.IntegrationTestNetSession

	Send(*types.Transaction) (common.Hash, error)
	Require_BundleYieldsZeroTransactions(t testing.TB, planHash common.Hash)
}

type NoIntakeValidationTestNet struct {
	tests.IntegrationTestNetSession
}

// NewNoIntakeValidationIntegrationTestNet creates a test network where the TxPool
// validation is disabled, allowing transactions that would normally be synchronously
// rejected to be pooled. These transactions may not be promoted or emitted in events.
// Therefore execution of these transactions is not guaranteed and may still be
// filtered by the frontend of the system.
func NewNoIntakeValidationIntegrationTestNet(t testing.TB, upgrades opera.Upgrades) *NoIntakeValidationTestNet {
	return &NoIntakeValidationTestNet{
		tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
			Upgrades: &upgrades,
			ClientExtraArguments: []string{
				"--disable-txPool-validation",
			},
		}),
	}
}

func (n *NoIntakeValidationTestNet) Send(tx *types.Transaction) (common.Hash, error) {
	return n.IntegrationTestNetSession.Send(tx)
}

func (n *NoIntakeValidationTestNet) Require_BundleYieldsZeroTransactions(t testing.TB, planHash common.Hash) {

	client, err := n.GetClient()
	require.NoError(t, err)
	defer client.Close()

	// expect timeout
	timedCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err = WaitForBundleExecution(timedCtx, client.Client(), planHash)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

type NoIntakeAndEmissionValidationTestNet struct {
	tests.IntegrationTestNetSession
}

// NewNoIntakeAndEmissionValidationTestNet creates a test network where the TxPool is
// completely bypassed by sending envelopes directly to the consensus engine.
// This allows testing bundle execution in a scenario where transactions reach
// the block processor, with the intention of testing its resilience against
// invalid transactions.
func NewNoIntakeAndEmissionValidationTestNet(t testing.TB, upgrades opera.Upgrades) *NoIntakeAndEmissionValidationTestNet {
	return &NoIntakeAndEmissionValidationTestNet{
		tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{
			Upgrades: &upgrades,
		}),
	}
}

func (n *NoIntakeAndEmissionValidationTestNet) Send(tx *types.Transaction) (common.Hash, error) {
	return n.ForceEmit(context.Background(), tx)
}

func (n *NoIntakeAndEmissionValidationTestNet) Require_BundleYieldsZeroTransactions(t testing.TB, planHash common.Hash) {
	client, err := n.GetClient()
	require.NoError(t, err)
	defer client.Close()

	info, err := WaitForBundleExecution(t.Context(), client.Client(), planHash)
	require.NoError(t, err)
	require.NotNil(t, info)
	require.Zero(t, info.Count)
}
