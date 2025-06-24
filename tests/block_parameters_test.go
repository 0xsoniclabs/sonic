// Copyright 2025 Sonic Operations Ltd
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
	"fmt"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests/contracts/block_parameters"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestBlockParameters_BlockHeaderMatchesObservableBlockParameters(t *testing.T) {
	hardForks := map[string]opera.Upgrades{
		"sonic": {
			Berlin:  true,
			London:  true,
			Llr:     false,
			Sonic:   true,
			Allegro: false,
		},
		"allegro": {
			Berlin:  true,
			London:  true,
			Llr:     false,
			Sonic:   true,
			Allegro: true,
		},
	}

	for name, upgrades := range hardForks {
		t.Run(name, func(t *testing.T) {
			for _, singleProposer := range []bool{false, true} {
				t.Run(fmt.Sprintf("single_proposer=%t", singleProposer), func(t *testing.T) {
					upgrades := upgrades
					upgrades.SingleProposerBlockFormation = singleProposer

					net := StartIntegrationTestNetWithJsonGenesis(t,
						IntegrationTestNetOptions{
							Upgrades: &upgrades,
							NumNodes: 2,
						},
					)
					testBlockHeaderMatchesObservableBlockParameters(t, net)
				})
			}
		})
	}

}

func testBlockHeaderMatchesObservableBlockParameters(
	t *testing.T,
	net *IntegrationTestNet,
) {
	require := require.New(t)
	contract, receipt, err := DeployContract(net, block_parameters.DeployBlockParameters)
	require.NoError(err, "Failed to deploy BlockParameters contract")

	// Collect a few samples of block parameters from within transactions.
	fromBlocks := map[uint64]block_parameters.BlockParametersParameters{}
	blockHashesFromReceipts := map[uint64]common.Hash{}
	for range 10 {
		receipt, err := net.Apply(contract.LogBlockParameters)
		require.NoError(err, "Failed to apply LogBlockParameters transaction")
		require.Equal(types.ReceiptStatusSuccessful, receipt.Status)

		require.Len(receipt.Logs, 1, "Expected exactly one log entry")
		params, err := contract.ParseLog(*receipt.Logs[0])
		require.NoError(err, "Failed to parse log entry")

		number := receipt.BlockNumber.Uint64()
		fromBlocks[number] = params.Parameters
		blockHashesFromReceipts[number] = receipt.BlockHash
	}

	checkParameters := func() {
		for i := range net.NumNodes() {
			client, err := net.GetClientConnectedToNode(i)
			require.NoError(err, "Failed to get client")
			defer client.Close()

			// Verify those block parameters against the block headers.
			for blockNumber, fromTx := range fromBlocks {
				block, err := client.BlockByNumber(t.Context(), big.NewInt(int64(blockNumber)))
				require.NoError(err, "Failed to get block by number")

				require.Equal(fromTx.ChainId, net.GetChainId())
				require.Equal(fromTx.Number, block.Number())
				require.Equal(fromTx.BaseFee, block.BaseFee())
				// Note: BlobBaseFee is not available in the block header

				blockTime := new(big.Int).SetUint64(block.Time())
				require.Equal(fromTx.Time, blockTime)

				blockGasLimit := new(big.Int).SetUint64(block.GasLimit())
				require.Equal(fromTx.GasLimit, blockGasLimit)

				require.Equal(fromTx.Coinbase, block.Coinbase())

				mixDigest := block.MixDigest()
				require.Equal(fromTx.PrevRandao, new(big.Int).SetBytes(mixDigest[:]))

				// Check that the block hash in the receipt matches the actual block hash.
				require.Equal(blockHashesFromReceipts[blockNumber], block.Hash())
			}

			// Verify that the contract can retrieve the block parameters from the archive.
			contract, err := block_parameters.NewBlockParameters(
				receipt.ContractAddress, client,
			)
			require.NoError(err, "Failed to create contract instance")
			for blockNumber, fromTx := range fromBlocks {
				fromArchive, err := contract.GetBlockParameters(&bind.CallOpts{
					BlockNumber: big.NewInt(int64(blockNumber)),
				})
				require.NoError(err, "Failed to get block parameters from archive")

				require.Equal(fromTx, fromArchive)
			}
		}
	}

	// Check that the collected parameters match the block headers before and
	// after a restart. The restart is included to make sure the information is
	// not just cached in memory.
	checkParameters()
	require.NoError(net.Restart())
	checkParameters()
}
