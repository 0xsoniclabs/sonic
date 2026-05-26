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
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestBundle_NonExecutableBundlesAreRejected(t *testing.T) {
	net := GetIntegrationTestNetWithBundlesEnabled(t)

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	signer := types.LatestSignerForChainID(net.GetChainId())
	sender := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))
	recipient := common.Address{0x01}

	// Submit a bundle with nonce 0 and verify that it executes successfully.
	firstEnvelope, _, firstPlan := bundle.NewBuilder().
		WithSigner(signer).
		AllOf(Step(t, net, sender, &types.AccessListTx{
			To:    &recipient,
			Nonce: 0,
		})).
		BuildEnvelopeBundleAndPlan()

	_, err = net.Send(firstEnvelope)
	require.NoError(t, err)

	firstInfo, err := WaitForBundleExecution(t.Context(), client.Client(), firstPlan.Hash())
	require.NoError(t, err)
	require.EqualValues(t, 1, firstInfo.Count)

	// Build another bundle with nonce 2 while nonce 1 is still missing.
	// The bundle should be rejected as non-executable.
	secondEnvelope := bundle.NewBuilder().
		WithSigner(signer).
		AllOf(Step(t, net, sender, &types.AccessListTx{
			To:    &recipient,
			Nonce: 2,
		})).
		Build()

	_, err = net.Send(secondEnvelope)
	require.Error(t, err)
	require.ErrorContains(t, err, "bundle is not executable")
}
