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

package sonicapi

import (
	"crypto/ecdsa"
	"encoding/json"
	"math/big"
	"strings"
	"testing"

	"github.com/0xsoniclabs/sonic/api/rpctest"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func Test_bundlesRPC_PrepareAndSubmit_PoolsValidBundles(t *testing.T) {

	// This test covers the integration of PrepareBundle and SubmitBundle,
	// ensuring that a bundle prepared by the API can be successfully submitted
	// and recognized by the transaction pool as a valid bundle envelope
	// with the correct execution plan hash.

	accounts := make(map[string]*rpctest.Wallet)
	accounts["ACCOUNT1"] = rpctest.NewWallet(t)
	accounts["ACCOUNT2"] = rpctest.NewWallet(t)
	addrToKey := make(map[common.Address]*ecdsa.PrivateKey)
	for _, acc := range accounts {
		addrToKey[*acc.Address()] = acc.PrivateKey
	}

	tests := map[string]struct {
		request        string
		expectedBundle string
	}{
		"one tx bundle": {
			request: `{
			   "steps": [	
					{
						"from": "ACCOUNT1",
						"to": "ACCOUNT2",
						"nonce": "0x0"
					}
			   ]
			}`,
			expectedBundle: "A",
		},
		"two independent steps": {
			request: `{
			   "steps": [	
					{
						"from": "ACCOUNT1",
						"to": "ACCOUNT2",
						"nonce": "0x0"
					},
					{
						"from": "ACCOUNT2",
						"to": "ACCOUNT1",
						"nonce": "0x0"
					}
			   ]
			}`,
			expectedBundle: "AllOf(A,B)",
		},
		"oneOf with two steps": {
			request: `{
				"oneOf": true,
				"steps": [
					{
						"from": "ACCOUNT1",
						"to": "ACCOUNT2",
						"nonce": "0x0",
						"gas": "0x5208"
					},
					{
						"from": "ACCOUNT2",
						"to": "ACCOUNT1",
						"nonce": "0x0",
						"gas": "0x5208"
					}
				]
			}`,
			expectedBundle: "OneOf(A,B)",
		},
		"execution flags": {
			request: `{
			   "steps": [	
					{
						"from": "ACCOUNT1",
						"to": "ACCOUNT2",
						"nonce": "0x0",
						"gas": "0x5208",
						"tolerateFailed": true
					},
					{
						"from": "ACCOUNT2",
						"to": "ACCOUNT1",
						"nonce": "0x0",
						"gas": "0x5208",
						"tolerateInvalid": true
					}
			   ]
			}`,
			expectedBundle: "AllOf(Step[TolerateFailed](A),Step[TolerateInvalid](B))",
		},
		"nested groups": {
			request: `{
				"oneOf": true,
				"steps": [
					{
						"from": "ACCOUNT1",
						"to": "ACCOUNT2",
						"nonce": "0x0",
						"gas": "0x5208"
					},
					{
						"oneOf": true,
						"steps": [
							{
								"from": "ACCOUNT2",
								"to": "ACCOUNT1",
								"nonce": "0x0",
								"gas": "0x5208"
							},
							{
								"from": "ACCOUNT1",
								"to": "ACCOUNT2",
								"nonce": "0x1",
								"gas": "0x5208"
							}
						]
					}
				]
			}`,
			expectedBundle: "OneOf(A,OneOf(B,C))",
		},
		"nested group with repeated transaction reference": {
			request: `{
				"oneOf": true,
				"steps": [
					{
						"from": "ACCOUNT1",
						"to": "ACCOUNT2",
						"nonce": "0x0",
						"gas": "0x5208"
					},
					{
						"oneOf": true,
						"steps": [
							{
								"from": "ACCOUNT2",
								"to": "ACCOUNT1",
								"nonce": "0x0",
								"gas": "0x5208"
							},
							{
								"from": "ACCOUNT1",
								"to": "ACCOUNT2",
								"nonce": "0x0",
								"gas": "0x5208"
							}
						]
					}
				]
			}`,
			expectedBundle: "OneOf(A,OneOf(B,A))",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			ctrl := gomock.NewController(t)
			pool := rpctest.NewMockTxPool(ctrl)

			beBuilder := rpctest.NewBackendBuilder(t).
				WithPool(pool)
			for _, acc := range accounts {
				beBuilder.WithAccount(*acc.Address(), rpctest.AccountState{
					Nonce:   0,
					Balance: big.NewInt(1e18), // 1 ETH
				})
			}
			be := beBuilder.Build()
			api := NewPublicBundleAPI(be)
			signer := be.GetSigner()

			// ============ fill-in templated inputs ==================

			input := test.request
			for placeholder, account := range accounts {
				input = strings.ReplaceAll(input, placeholder, account.Address().Hex())
			}

			// ============ PrepareBundle ==================

			var proposal RPCExecutionProposal
			err := json.Unmarshal([]byte(input), &proposal)
			require.NoError(t, err, "test input must be a valid RPCExecutionProposal")

			prepared, err := api.PrepareBundle(t.Context(), proposal)
			require.NoError(t, err)

			reconstructedPlan, err := ToBundleExecutionPlan(prepared.ExecutionPlan)
			require.NoError(t, err)

			require.Equal(t, test.expectedBundle, reconstructedPlan.Root.String(),
				"prepared execution plan does not match expected")

			// ============ SubmitBundle ==================

			submitArgs := SubmitBundleArgs{
				SignedTransactions: make([]hexutil.Bytes, len(prepared.Transactions)),
				ExecutionPlan:      prepared.ExecutionPlan,
			}
			// This loop replicates third-party wallets signing the bundled transactions
			for i, tx := range prepared.Transactions {
				// Note: all transactions at this stage have access list, therefore
				// toTransactionForBundles is not needed.
				txToSign, err := tx.ToTransaction()
				require.NoError(t, err)
				signedTx, err := types.SignTx(txToSign, signer, addrToKey[*tx.From])
				require.NoError(t, err)

				encodedTx, err := signedTx.MarshalBinary()
				require.NoError(t, err)

				submitArgs.SignedTransactions[i] = encodedTx
			}

			pool.EXPECT().AddLocal(isAnEnvelope{}).Do(func(tx *types.Transaction) {
				_, extractedPlan, err := bundle.ValidateEnvelope(signer, tx)
				require.NoError(t, err)
				require.Equal(t, reconstructedPlan.Hash(), extractedPlan.Hash())
			})

			submitted, err := api.SubmitBundle(t.Context(), submitArgs)
			require.NoError(t, err)
			require.Equal(t, reconstructedPlan.Hash(), submitted,
				"submit must return the same execution plan as reconstructed and pooled")
		})
	}
}

// isAnEnvelope is a gomock matcher that checks if a transaction is a bundle envelope
type isAnEnvelope struct{}

func (m isAnEnvelope) Matches(x interface{}) bool {

	tx, ok := x.(*types.Transaction)
	if !ok {
		return false
	}

	return bundle.IsEnvelope(tx)
}

func (m isAnEnvelope) String() string {
	return "is a bundle envelope"
}
