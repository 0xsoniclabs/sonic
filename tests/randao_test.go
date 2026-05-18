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
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestRandao_randaoIntegrationTest(t *testing.T) {
	const NumNodes = 3

	modes := map[string]bool{
		"single proposer":      true,
		"distributed proposer": false,
	}

	for name, test := range opera.GetAllHardForksInOrder() {

		for modeName, singleProposer := range modes {
			test.SingleProposerBlockFormation = singleProposer

			t.Run(name+"/"+modeName, func(t *testing.T) {
				net := StartIntegrationTestNet(t,
					IntegrationTestNetOptions{
						NumNodes: NumNodes,
						Upgrades: &test,
					},
				)
				defer net.Stop()

				// issue one transaction to trigger one block
				receipt, err := net.EndowAccount(common.Address{0xFE}, big.NewInt(1))
				require.NoError(t, err)
				require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

				// Ensure that the block has been processed and randao is set
				randaoList := make([]common.Hash, NumNodes)
				for i := range NumNodes {
					client, err := net.GetClientConnectedToNode(i)
					require.NoError(t, err)
					defer client.Close()

					// The receipt was obtained from node 0, but other
					// nodes may not have synced the block yet.
					block := WaitForBlock(t, client, int(receipt.BlockNumber.Int64()))
					require.NotZero(t, block.Header().MixDigest)
					randaoList[i] = block.Header().MixDigest
				}

				// Verify that all nodes have the same randao value
				for i := range NumNodes - 1 {
					require.Equal(t, randaoList[i], randaoList[i+1], "Randao values should match across nodes")
				}

				// Verify that the randao value is different int the next block
				receipt, err = net.EndowAccount(common.Address{0xFE}, big.NewInt(1))
				require.NoError(t, err)
				require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
				client, err := net.GetClientConnectedToNode(0)
				require.NoError(t, err)
				defer client.Close()

				var block *types.Block
				timeout, cancel := context.WithTimeout(t.Context(), 3*time.Second)
				defer cancel()
				err = WaitFor(timeout, func(ctx context.Context) (bool, error) {
					block, err = client.BlockByNumber(ctx, receipt.BlockNumber)
					if err == ethereum.NotFound {
						return false, nil
					}
					if err != nil {
						return false, err
					}
					return true, nil
				})
				require.NoError(t, err)
				require.NotZero(t, block.Header().MixDigest)
				require.NotEqual(t, randaoList[0], block.Header().MixDigest, "Randao value should change in the next block")
			})
		}
	}
}
