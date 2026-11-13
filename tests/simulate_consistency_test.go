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
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/stretchr/testify/require"
)

func TestSimulateV1_MatchesOnChainExecution(t *testing.T) {
	deployCodeOfSize := func(size uint64) []byte {
		return []byte{
			byte(vm.PUSH3), byte(size >> 16), byte(size >> 8), byte(size),
			byte(vm.PUSH1), 0x00,
			byte(vm.RETURN),
		}
	}
	receiver := common.Address{0x42}

	tests := map[string]struct {
		to         *common.Address
		gas        uint64
		data       []byte
		wantStatus uint64
	}{
		"transfer with unused gas": {
			to:         &receiver,
			gas:        100_000,
			wantStatus: types.ReceiptStatusSuccessful,
		},
		"init code of maximum size": {
			gas:        20_000_000,
			data:       make([]byte, opera.SonicPostAllegroMaxInitCodeSize),
			wantStatus: types.ReceiptStatusSuccessful,
		},
		"deployed code of maximum size": {
			gas:        20_000_000,
			data:       deployCodeOfSize(opera.SonicPostAllegroMaxCodeSize),
			wantStatus: types.ReceiptStatusSuccessful,
		},
		"deployed code above maximum size": {
			gas:        20_000_000,
			data:       deployCodeOfSize(opera.SonicPostAllegroMaxCodeSize + 1),
			wantStatus: types.ReceiptStatusFailed,
		},
	}

	forks := map[string]opera.Upgrades{
		"Brio": opera.GetBrioUpgrades(),
	}
	for forkName, upgrades := range forks {
		t.Run(forkName, func(t *testing.T) {
			session := getIntegrationTestNetSession(t, upgrades)
			t.Parallel()

			sender := session.GetSessionSponsor()
			client, err := session.GetClient()
			require.NoError(t, err)
			defer client.Close()

			for name, test := range tests {
				t.Run(name, func(t *testing.T) {
					tx := CreateTransaction(t, session, &types.LegacyTx{
						To:   test.to,
						Gas:  test.gas,
						Data: test.data,
					}, sender)
					receipt, err := session.Run(tx)
					require.NoError(t, err)
					require.Equal(t, test.wantStatus, receipt.Status)

					call := map[string]any{
						"from":  sender.Address(),
						"gas":   hexutil.Uint64(test.gas),
						"input": hexutil.Bytes(test.data),
					}
					if test.to != nil {
						call["to"] = test.to
					}
					opts := map[string]any{
						"blockStateCalls": []any{map[string]any{"calls": []any{call}}},
					}
					var result []struct {
						Calls []struct {
							Status  hexutil.Uint64 `json:"status"`
							GasUsed hexutil.Uint64 `json:"gasUsed"`
						} `json:"calls"`
					}
					parent := hexutil.Uint64(receipt.BlockNumber.Uint64() - 1)
					err = client.Client().Call(&result, "eth_simulateV1", opts, parent)
					require.NoError(t, err)
					require.Len(t, result, 1)
					require.Len(t, result[0].Calls, 1)
					require.Equal(t, receipt.Status, uint64(result[0].Calls[0].Status))
					require.Equal(t, receipt.GasUsed, uint64(result[0].Calls[0].GasUsed))
				})
			}
		})
	}
}
