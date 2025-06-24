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
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/tests/contracts/block_hash"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	req "github.com/stretchr/testify/require"
)

func TestBlockHash_CorrectBlockHashesAreAccessibleInContracts(t *testing.T) {
	require := req.New(t)
	net := StartIntegrationTestNet(t)

	// Deploy the block hash observer contract.
	_, receipt, err := DeployContract(net, block_hash.DeployBlockHash)
	require.NoError(err, "failed to deploy contract; %v", err)
	contractAddress := receipt.ContractAddress
	contractCreationBlock := receipt.BlockNumber.Uint64()

	runTest := func(t *testing.T) {
		t.Run("visible block hash on head", func(t *testing.T) {
			testVisibleBlockHashOnHead(t, net, contractAddress)
		})
		t.Run("visible block hash in archive", func(t *testing.T) {
			testVisibleBlockHashesInArchive(t, net, contractAddress, contractCreationBlock)
		})
	}

	t.Run("fresh network", runTest)
	err = net.Restart()
	require.NoError(err, "failed to restart network; %v", err)
	t.Run("restarted network", runTest)
	err = net.RestartWithExportImport()
	require.NoError(err, "failed to restart network with export/import; %v", err)
	t.Run("reinitialized network", runTest)
}

func testVisibleBlockHashOnHead(
	t *testing.T,
	net *IntegrationTestNet,
	observerContractAddress common.Address,
) {
	require := req.New(t)
	client, err := net.GetClient()
	require.NoError(err, "failed to get client; %v", err)
	defer client.Close()

	contract, err := block_hash.NewBlockHash(observerContractAddress, client)
	require.NoError(err, "failed to instantiate contract")

	for range 3 {
		receipt, err := net.Apply(contract.Observe)
		require.NoError(err, "failed to observe block hash; %v", err)
		require.Equal(types.ReceiptStatusSuccessful, receipt.Status,
			"failed to observe block hash; %v", err,
		)

		blockNumber := receipt.BlockNumber.Uint64()
		require.Len(receipt.Logs, int(blockNumber+6), "unexpected number of logs")

		for _, log := range receipt.Logs {
			entry, err := contract.ParseSeen(*log)
			require.NoError(err, "failed to parse log; %v", err)
			current := entry.CurrentBlock.Uint64()
			require.Equal(blockNumber, current, "unexpected block number")
			observed := entry.ObservedBlock.Uint64()
			seen := common.Hash(entry.BlockHash)

			want := common.Hash{}
			if observed < current {
				hash, err := client.BlockByNumber(t.Context(), entry.ObservedBlock)
				require.NoError(err, "failed to get block hash; %v", err)
				want = hash.Hash()
			}
			require.Equal(want, seen, "block hash mismatch, current: %d, observed: %d", current, observed)
		}
	}
}

func testVisibleBlockHashesInArchive(
	t *testing.T,
	net *IntegrationTestNet,
	observerContractAddress common.Address,
	observerCreationBlock uint64,
) {
	require := req.New(t)
	client, err := net.GetClient()
	require.NoError(err, "failed to get client; %v", err)
	defer client.Close()

	// Get list of all block hashes.
	numBlocks, err := client.BlockNumber(t.Context())
	require.NoError(err, "failed to get block number; %v", err)

	hashes := []common.Hash{}
	for i := uint64(0); i <= numBlocks; i++ {
		hash, err := client.BlockByNumber(t.Context(), big.NewInt(int64(i)))
		require.NoError(err, "failed to get block hash; %v", err)
		hashes = append(hashes, hash.Hash())
	}

	// Check that blocks are reported correctly by archive queries.
	numChecks := 0
	observer, err := block_hash.NewBlockHash(observerContractAddress, client)
	require.NoError(err, "failed to instantiate contract")
	for observationBlock := range numBlocks {
		if observationBlock < observerCreationBlock {
			continue
		}
		for observedBlock := range numBlocks {
			hash, err := observer.GetBlockHash(&bind.CallOpts{
				BlockNumber: big.NewInt(int64(observationBlock)),
			}, big.NewInt(int64(observedBlock)))
			require.NoError(err, "failed to get block hash; %v", err)

			want := common.Hash{}
			if observedBlock < observationBlock {
				want = hashes[observedBlock]
			}
			got := common.Hash(hash)
			require.Equal(want, got, "block hash mismatch, observation: %d, observed: %d", observationBlock, observedBlock)
			numChecks++
		}
	}
	require.Greater(numChecks, 0, "no checks performed")
}
