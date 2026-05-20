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
	"time"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestBundle_EnvelopeAndBundleOnly_SemanticsEnabledByBrio_ExecutionEnabledByBundleFlag(t *testing.T) {
	cases := map[string]struct {
		sendEnvelope          bool // mutually exclusive with sendBundleOnly
		sendBundleOnly        bool // mutually exclusive with sendEnvelope
		brio                  bool
		transactionBundles    bool
		expectEnvelopeReceipt bool
		expectInnerReceipt    bool
	}{
		"Envelope/PreBrio/WithoutBundles": {
			sendEnvelope:          true,
			brio:                  false,
			transactionBundles:    false,
			expectEnvelopeReceipt: true, // envelope interpreted as normal transaction
		},
		"Envelope/PreBrio/WithBundles": {
			sendEnvelope:          true,
			brio:                  false,
			transactionBundles:    true,
			expectEnvelopeReceipt: true, // envelope interpreted as normal transaction
		},
		"Envelope/PostBrio/WithoutBundles": {
			sendEnvelope:       true,
			brio:               true,
			transactionBundles: false,
			expectInnerReceipt: false, // bundles disabled, so envelope should not be executed
		},
		"Envelope/PostBrio/WithBundles": {
			sendEnvelope:       true,
			brio:               true,
			transactionBundles: true,
			expectInnerReceipt: true, // envelope interpreted as bundle, inner transaction executed
		},
		"BundleOnly/PreBrio/WithoutBundles": {
			sendBundleOnly:     true,
			brio:               false,
			transactionBundles: false,
			expectInnerReceipt: true, // bundle-only marker ignored and executed as normal transaction
		},
		"BundleOnly/PreBrio/WithBundles": {
			sendBundleOnly:     true,
			brio:               false,
			transactionBundles: true,
			expectInnerReceipt: true, // bundle-only marker ignored and executed as normal transaction
		},
		"BundleOnly/PostBrio/WithoutBundles": {
			sendBundleOnly:     true,
			brio:               true,
			transactionBundles: false,
			expectInnerReceipt: false, // bundle-only transaction not executed
		},
		"BundleOnly/PostBrio/WithBundles": {
			sendBundleOnly:     true,
			brio:               true,
			transactionBundles: true,
			expectInnerReceipt: false, // bundle-only transaction not executed
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			upgrades := opera.GetAllegroUpgrades()
			upgrades.Brio = tc.brio
			upgrades.TransactionBundles = tc.transactionBundles

			for mode, net := range GetUnfilteredNetVariants(t, upgrades) {
				t.Run(mode, func(t *testing.T) {

					client, err := net.GetClient()
					require.NoError(t, err)
					defer client.Close()

					signer := types.LatestSignerForChainID(net.GetChainId())

					sender := tests.MakeAccountWithBalance(t, net, big.NewInt(1e18))

					txBundle := bundle.NewBuilder().
						WithSigner(signer).
						AllOf(Step(t, net, sender, &types.AccessListTx{})).
						BuildBundle()
					gasPrice, err := client.SuggestGasPrice(t.Context())
					require.NoError(t, err)
					envelope := bundle.NewEnvelope(signer, sender.PrivateKey, 0, gasPrice, &txBundle)
					innerTx := txBundle.GetTransactionsInReferencedOrder()[0]

					// Submit the transaction.
					if tc.sendEnvelope {
						_, err = net.Send(envelope)
					} else if tc.sendBundleOnly {
						_, err = net.Send(innerTx)
					}
					require.NoError(t, err)

					time.Sleep(1 * time.Second)

					envelopeReceipt, err := client.TransactionReceipt(t.Context(), envelope.Hash())
					if tc.expectEnvelopeReceipt {
						require.NoError(t, err)
						require.Equal(t, types.ReceiptStatusSuccessful, envelopeReceipt.Status)
					} else {
						require.ErrorContains(t, err, "not found")
					}

					innerReceipt, err := client.TransactionReceipt(t.Context(), innerTx.Hash())
					if tc.expectInnerReceipt {
						require.NoError(t, err)
						require.Equal(t, types.ReceiptStatusSuccessful, innerReceipt.Status)
					} else {
						require.ErrorContains(t, err, "not found")
					}
				})
			}
		})
	}
}
