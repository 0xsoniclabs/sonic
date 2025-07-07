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

	"github.com/0xsoniclabs/sonic/config"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests/contracts/block_hash"
	"github.com/0xsoniclabs/sonic/tests/contracts/read_history_storage"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
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

///////////////////////////////////////////////////////////////////////////////
// History storage contract address and sender address
///////////////////////////////////////////////////////////////////////////////

var (
	historyStorageAddress = common.HexToAddress("0x0000F90827F1C53a10cb7A02335B175320002935")
	senderAddr            = common.HexToAddress("0x3462413Af4609098e1E27A490f554f260213D685")
)

func TestBlockHash_EIP2935_IsAutomaticallyDeployedWithFakeNet(t *testing.T) {

	tests := map[string]func(t *testing.T) *IntegrationTestNet{
		"json genesis": func(t *testing.T) *IntegrationTestNet {
			return StartIntegrationTestNetWithJsonGenesis(t,
				IntegrationTestNetOptions{
					Upgrades: AsPointer(opera.GetAllegroUpgrades()),
				})
		},
		"fake genesis": func(t *testing.T) *IntegrationTestNet {
			return StartIntegrationTestNetWithFakeGenesis(t,
				IntegrationTestNetOptions{
					Upgrades: AsPointer(opera.GetAllegroUpgrades()),
				})
		},
	}

	for name, netConstructor := range tests {
		t.Run(name, func(t *testing.T) {
			require := req.New(t)
			net := netConstructor(t)

			client, err := net.GetClient()
			require.NoError(err)
			defer client.Close()

			code, err := client.CodeAt(t.Context(), historyStorageAddress, nil)
			require.NoError(err)
			require.Equal(params.HistoryStorageCode, code)

			nonce, err := client.NonceAt(t.Context(), historyStorageAddress, nil)
			require.NoError(err)
			require.Equal(uint64(1), nonce)
		})
	}
}

func TestBlockHash_EIP2935_HistoryContractIsNotDeployedBeforePrague(t *testing.T) {

	tests := map[string]func(t *testing.T) *IntegrationTestNet{
		"json genesis": func(t *testing.T) *IntegrationTestNet {
			return StartIntegrationTestNetWithJsonGenesis(t)
		},
		"fake genesis": func(t *testing.T) *IntegrationTestNet {
			return StartIntegrationTestNetWithFakeGenesis(t)
		},
	}

	for name, netConstructor := range tests {
		t.Run(name, func(t *testing.T) {
			require := req.New(t)
			net := netConstructor(t)

			client, err := net.GetClient()
			require.NoError(err)
			defer client.Close()

			code, err := client.CodeAt(t.Context(), historyStorageAddress, nil)
			require.NoError(err)
			require.Empty(code)

			nonce, err := client.NonceAt(t.Context(), historyStorageAddress, nil)
			require.NoError(err)
			require.Equal(uint64(0), nonce)
		})
	}
}

func TestBlockHash_EIP2935_DeployContract(t *testing.T) {
	require := req.New(t)

	net := StartIntegrationTestNet(t,
		IntegrationTestNetOptions{
			// < Allegro automatically deploys the history storage contract
			// < To test deployment, we need to use a feature set that does not already have the contract
			Upgrades: AsPointer(opera.GetSonicUpgrades()),
			ModifyConfig: func(config *config.Config) {
				// the transaction to deploy the contract is not replay protected
				// This has the benefit that the same tx will work in both ethereum and sonic.
				// Nevertheless the default RPC configuration rejects this sort of transaction.
				config.Opera.AllowUnprotectedTxs = true
			},
		},
	)

	client, err := net.GetClient()
	require.NoError(err)
	defer client.Close()

	// Deploy transaction as described in EIP-2935
	// https://eips.ethereum.org/EIPS/eip-2935
	// {
	// 	"type": "0x0",
	// 	"nonce": "0x0",
	// 	"to": null,
	// 	"gas": "0x3d090",
	// 	"gasPrice": "0xe8d4a51000",
	// 	"maxPriorityFeePerGas": null,
	// 	"maxFeePerGas": null,
	// 	"value": "0x0",
	// 	"input": "0x60538060095f395ff33373fffffffffffffffffffffffffffffffffffffffe14604657602036036042575f35600143038111604257611fff81430311604257611fff9006545f5260205ff35b5f5ffd5b5f35611fff60014303065500",
	// 	"v": "0x1b",
	// 	"r": "0x539",
	// 	"s": "0xaa12693182426612186309f02cfe8a80a0000",
	// 	"hash": "0x67139a552b0d3fffc30c0fa7d0c20d42144138c8fe07fc5691f09c1cce632e15"
	//   }

	v, ok := new(big.Int).SetString("0x1b", 0)
	require.True(ok)
	r, ok := new(big.Int).SetString("0x539", 0)
	require.True(ok)
	s, ok := new(big.Int).SetString("0xaa12693182426612186309f02cfe8a80a0000", 0)
	require.True(ok)

	payload := &types.LegacyTx{
		Nonce:    0,
		Gas:      0x3d090,
		GasPrice: new(big.Int).SetUint64(0xe8d4a51000),
		Value:    new(big.Int).SetUint64(0),
		Data:     common.Hex2Bytes("60538060095f395ff33373fffffffffffffffffffffffffffffffffffffffe14604657602036036042575f35600143038111604257611fff81430311604257611fff9006545f5260205ff35b5f5ffd5b5f35611fff60014303065500"),
		V:        v,
		R:        r,
		S:        s,
	}

	tx := types.NewTx(payload)

	// The transaction is pre EIP-155, (the chain ID is not included in the signature)
	sender, err := types.HomesteadSigner{}.Sender(tx)
	require.NoError(err)
	require.Equal(senderAddr, sender)

	_, err = net.EndowAccount(senderAddr, big.NewInt(1e18))
	require.NoError(err)

	receipt, err := net.Run(tx)
	require.NoError(err)
	require.Equal(types.ReceiptStatusSuccessful, receipt.Status)

	code, err := client.CodeAt(t.Context(), historyStorageAddress, nil)
	require.NoError(err)
	require.Equal(params.HistoryStorageCode, code)

	nonce, err := client.NonceAt(t.Context(), historyStorageAddress, nil)
	require.NoError(err)
	require.Equal(uint64(1), nonce)

	readHistoryStorageContract, receipt, err := DeployContract(net, read_history_storage.DeployReadHistoryStorage)
	require.NoError(err)
	require.Equal(types.ReceiptStatusSuccessful, receipt.Status)

	// Create one block and use the contract to read the block hash. because this
	// network is running in Sonic, no block hashes are stored in the history storage contract.
	receipt, err = net.EndowAccount(NewAccount().Address(), big.NewInt(1e18))
	require.NoError(err)
	blockNumber := receipt.BlockNumber

	receipt, err = net.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return readHistoryStorageContract.ReadHistoryStorage(opts, blockNumber)
	})
	require.NoError(err)
	require.Equal(types.ReceiptStatusSuccessful, receipt.Status)

	require.Len(receipt.Logs, 1)
	blockHash, err := readHistoryStorageContract.ParseBlockHash(*receipt.Logs[0])
	require.NoError(err)

	// read hash is null because the processor is not calling the history storage contract
	require.Equal(common.Hash{31: 0x00},
		common.BytesToHash(blockHash.BlockHash[:]))

}

func TestBlockHash_EIP2935_HistoryContractAccumulatesBlockHashes(t *testing.T) {

	require := req.New(t)

	net := StartIntegrationTestNetWithFakeGenesis(t,
		IntegrationTestNetOptions{
			Upgrades: AsPointer(opera.GetAllegroUpgrades()),
		})

	client, err := net.GetClient()
	require.NoError(err)
	defer client.Close()

	readHistoryStorageContract, receipt, err := DeployContract(net, read_history_storage.DeployReadHistoryStorage)
	require.NoError(err)
	require.Equal(types.ReceiptStatusSuccessful, receipt.Status)

	// eip-2935 describes a buffer-ring of 8191 block hashes.
	// testing this is impractical, this test checks a smaller range to ensure that contract
	// accumulates multiple block hashes.
	const testIterations = 10

	// Fist loop just issues synchronous transactions to create blocks
	hashes := make(map[uint64]common.Hash)
	for range testIterations {
		receipt, err := net.EndowAccount(NewAccount().Address(), big.NewInt(1e18))
		require.NoError(err)
		require.Equal(types.ReceiptStatusSuccessful, receipt.Status)
		hashes[receipt.BlockNumber.Uint64()] = receipt.BlockHash
	}

	// second loop queries block hashes of the blocks generated in the first loop
	for blockNumber, recordedHash := range hashes {
		receipt, err = net.Apply(func(opts *bind.TransactOpts) (*types.Transaction, error) {
			return readHistoryStorageContract.ReadHistoryStorage(opts, new(big.Int).SetUint64(blockNumber))
		})
		require.NoError(err)
		require.Equal(types.ReceiptStatusSuccessful, receipt.Status)
		require.Len(receipt.Logs, 1)

		blockHash, err := readHistoryStorageContract.ParseBlockHash(*receipt.Logs[0])
		require.NoError(err)
		require.Equal(0, blockHash.QueriedBlock.Cmp(big.NewInt(int64(blockNumber))))

		require.Equal(
			common.BytesToHash(blockHash.BlockHash[:]),
			common.BytesToHash(blockHash.BuiltinBlockHash[:]),
			"builtin blockhash does not match the block hash stored in the contract",
		)

		// read hash must be equal to the block hash retrieved from the client
		block, err := client.BlockByNumber(t.Context(), big.NewInt(int64(blockNumber)))
		require.NoError(err)
		require.Equal(common.BytesToHash(blockHash.BlockHash[:]), block.Hash(),
			"read hash does not match the block hash")

		// hash must be equal to the hash from the first loop receipt
		require.Equal(recordedHash, block.Hash(),
			"block hash does not match the hash from the receipt")
	}
}
