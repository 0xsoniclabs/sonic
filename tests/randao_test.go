package tests

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestRandao_randaoIntegrationTest(t *testing.T) {
	const NumNodes = 3

	tests := map[string]opera.Upgrades{
		"dag proposal": opera.GetSonicUpgrades(),
		"single proposal": {
			Berlin:                       true,
			London:                       true,
			Llr:                          false,
			Allegro:                      true,
			Sonic:                        true,
			SingleProposerBlockFormation: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
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

				block, err := client.BlockByNumber(t.Context(), receipt.BlockNumber)
				require.NoError(t, err)
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
			block, err := client.BlockByNumber(t.Context(), receipt.BlockNumber)
			require.NoError(t, err)
			require.NotZero(t, block.Header().MixDigest)
			require.NotEqual(t, randaoList[0], block.Header().MixDigest, "Randao value should change in the next block")
		})
	}
}
