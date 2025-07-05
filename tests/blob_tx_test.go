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
	"bytes"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/tests/contracts/blobbasefee"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestBlobTransaction(t *testing.T) {

	ctxt := MakeTestContext(t)
	defer ctxt.Close()

	t.Run("blob tx with non-empty blobs is rejected", func(t *testing.T) {
		testBlobTx_WithBlobsIsRejected(t, ctxt)
	})

	t.Run("blob tx with empty blobs is executed", func(t *testing.T) {
		testBlobTx_WithEmptyBlobsIsExecuted(t, ctxt)
		checkBlocksSanity(t, ctxt.client)
	})

	t.Run("blob tx with nil sidecar is executed", func(t *testing.T) {
		testBlobTx_WithNilSidecarIsExecuted(t, ctxt)
		checkBlocksSanity(t, ctxt.client)
	})

	t.Run("blob base fee can be read from head, block and history", func(t *testing.T) {
		testBlobBaseFee_CanReadBlobBaseFeeFromHeadAndBlockAndHistory(t, ctxt)
		checkBlocksSanity(t, ctxt.client)
	})

	t.Run("blob gas used can be read from block header", func(t *testing.T) {
		testBlobBaseFee_CanReadBlobGasUsed(t, ctxt)
		checkBlocksSanity(t, ctxt.client)
	})
}

func testBlobTx_WithBlobsIsRejected(t *testing.T, ctxt *testContext) {
	require := require.New(t)
	nonZeroNumberOfBlobs := 2

	// Create a new transaction with blob data
	blobs := make([][]byte, nonZeroNumberOfBlobs)
	for i := 0; i < nonZeroNumberOfBlobs; i++ {
		var blob kzg4844.Blob
		copy(blobs[i], blob[:])
	}

	tx, err := createTestBlobTransaction(t, ctxt, blobs...)
	require.NoError(err)

	// attempt to run tx
	_, err = ctxt.net.Run(tx)
	require.ErrorContains(err, "non-empty blob transaction are not supported")

	// repeat same tx (regression against reported repeated tx issue)
	_, err = ctxt.net.Run(tx)
	require.ErrorContains(err, "non-empty blob transaction are not supported")
}

func testBlobTx_WithEmptyBlobsIsExecuted(t *testing.T, ctxt *testContext) {
	require := require.New(t)

	tx, err := createTestBlobTransaction(t, ctxt)
	require.NoError(err)

	// run tx
	receipt, err := ctxt.net.Run(tx)
	require.NoError(err, "transaction must be accepted")
	require.Equal(
		types.ReceiptStatusSuccessful,
		receipt.Status,
		"transaction must succeed",
	)
}

func testBlobTx_WithNilSidecarIsExecuted(t *testing.T, ctxt *testContext) {
	require := require.New(t)

	tx, err := createTestBlobTransactionWithNilSidecar(t, ctxt)
	require.NoError(err)

	// run tx
	receipt, err := ctxt.net.Run(tx)
	require.NoError(err, "transaction must be accepted")
	require.Equal(
		types.ReceiptStatusSuccessful,
		receipt.Status,
		"transaction must succeed",
	)
}

func testBlobBaseFee_CanReadBlobBaseFeeFromHeadAndBlockAndHistory(t *testing.T, ctxt *testContext) {
	require := require.New(t)

	// Deploy the blob base fee contract.
	contract, _, err := DeployContract(ctxt.net, blobbasefee.DeployBlobbasefee)
	require.NoError(err, "failed to deploy contract; ", err)

	// Collect the current blob base fee from the head state.
	receipt, err := ctxt.net.Apply(contract.LogCurrentBlobBaseFee)
	require.NoError(err, "failed to log current blob base fee; ", err)
	require.Equal(len(receipt.Logs), 1, "unexpected number of logs; expected 1, got ", len(receipt.Logs))

	entry, err := contract.ParseCurrentBlobBaseFee(*receipt.Logs[0])
	require.NoError(err, "failed to parse log; ", err)
	fromLog := entry.Fee.Uint64()

	// Collect the blob base fee from the block header.
	block, err := ctxt.client.BlockByNumber(t.Context(), receipt.BlockNumber)
	require.NoError(err, "failed to get block header; ", err)
	fromBlock := getBlobBaseFeeFrom(block.Header())

	// Collect the blob base fee from the archive.
	fromArchive, err := contract.GetBlobBaseFee(&bind.CallOpts{BlockNumber: receipt.BlockNumber})
	require.NoError(err, "failed to get blob base fee from archive; ", err)

	// call the blob base fee rpc method
	fromRpc := new(hexutil.Uint64)
	err = ctxt.client.Client().Call(&fromRpc, "eth_blobBaseFee")
	require.NoError(err, "failed to get blob base fee from rpc; ", err)

	// we check blob base fee is one because it is not implemented yet. TODO issue #147
	require.Equal(fromLog, uint64(1), "invalid blob base fee from log; ", fromLog)
	require.Equal(fromLog, fromArchive.Uint64(), "blob base fee mismatch; from log %v, from archive %v", fromLog, fromArchive)
	require.Equal(fromLog, fromBlock, "blob base fee mismatch; from log %v, from block %v", fromLog, fromBlock)
	require.Equal(fromLog, uint64(*fromRpc), "blob base fee mismatch; from log %v, from rpc %v", fromLog, fromRpc)
}

func testBlobBaseFee_CanReadBlobGasUsed(t *testing.T, ctxt *testContext) {
	require := require.New(t)

	// Get blob gas used from the block header of the latest block.
	block, err := ctxt.client.BlockByNumber(t.Context(), nil)
	require.NoError(err, "failed to get block header; ", err)
	require.Empty(*block.BlobGasUsed(), "unexpected value in blob gas used")
	require.Empty(*block.Header().ExcessBlobGas, "unexpected excess blob gas value")

	// check value for blob gas used is rlp encoded and decoded
	buffer := bytes.NewBuffer(make([]byte, 0))
	err = block.EncodeRLP(buffer)
	require.NoError(err, "failed to encode block header; ", err)

	// decode block
	stream := rlp.NewStream(buffer, 0)
	err = block.DecodeRLP(stream)
	require.NoError(err, "failed to decode block header; ", err)

	// check blob gas used and excess blob gas are zero
	require.Empty(*block.BlobGasUsed(), "unexpected blob gas used value")
	require.Empty(*block.Header().ExcessBlobGas, "unexpected excess blob gas value")
}

////////////////////////////////////////////////////////////////////////////////
// Helper Functions
////////////////////////////////////////////////////////////////////////////////

func createTestBlobTransaction(t *testing.T, ctxt *testContext, data ...[]byte) (*types.Transaction, error) {
	require := require.New(t)

	chainId, err := ctxt.client.ChainID(t.Context())
	require.NoError(err, "failed to get chain ID::")

	nonce, err := ctxt.client.NonceAt(t.Context(), ctxt.net.GetSessionSponsor().Address(), nil)
	require.NoError(err, "failed to get nonce:")

	var sidecar *types.BlobTxSidecar
	var blobHashes []common.Hash

	if len(data) > 0 {
		sidecar = new(types.BlobTxSidecar)
	}

	for _, data := range data {
		var blob kzg4844.Blob // Define a blob array to hold the large data payload, blobs are 128kb in length
		copy(blob[:], data)

		blobCommitment, err := kzg4844.BlobToCommitment(&blob)
		require.NoError(err, "failed to compute blob commitment")

		blobProof, err := kzg4844.ComputeBlobProof(&blob, blobCommitment)
		require.NoError(err, "failed to compute blob proof")

		sidecar.Blobs = append(sidecar.Blobs, blob)
		sidecar.Commitments = append(sidecar.Commitments, blobCommitment)
		sidecar.Proofs = append(sidecar.Proofs, blobProof)
	}

	// Get blob hashes from the sidecar
	if len(data) > 0 {
		blobHashes = sidecar.BlobHashes()
	}

	// Create and return transaction with the blob data and cryptographic proofs
	tx := types.NewTx(&types.BlobTx{
		ChainID:    uint256.MustFromBig(chainId),
		Nonce:      nonce,
		GasTipCap:  uint256.NewInt(1e10),  // max priority fee per gas
		GasFeeCap:  uint256.NewInt(50e10), // max fee per gas
		Gas:        35000,                 // gas limit for the transaction
		To:         common.Address{},      // recipient's address
		Value:      uint256.NewInt(0),     // value transferred in the transaction
		BlobFeeCap: uint256.NewInt(3e10),  // fee cap for the blob data
		BlobHashes: blobHashes,            // blob hashes in the transaction
		Sidecar:    sidecar,               // sidecar data in the transaction
	})

	return types.SignTx(tx, types.NewCancunSigner(chainId), ctxt.net.GetSessionSponsor().PrivateKey)
}

func createTestBlobTransactionWithNilSidecar(t *testing.T, ctxt *testContext) (*types.Transaction, error) {
	require := require.New(t)

	chainId, err := ctxt.client.ChainID(t.Context())
	require.NoError(err, "failed to get chain ID::")

	nonce, err := ctxt.client.NonceAt(t.Context(), ctxt.net.GetSessionSponsor().Address(), nil)
	require.NoError(err, "failed to get nonce:")

	// Create and return transaction with the blob data and cryptographic proofs
	tx := types.NewTx(&types.BlobTx{
		ChainID:    uint256.MustFromBig(chainId),
		Nonce:      nonce,
		GasTipCap:  uint256.NewInt(1e10),  // max priority fee per gas
		GasFeeCap:  uint256.NewInt(50e10), // max fee per gas
		Gas:        35000,                 // gas limit for the transaction
		To:         common.Address{},      // recipient's address
		Value:      uint256.NewInt(0),     // value transferred in the transaction
		BlobFeeCap: uint256.NewInt(3e10),  // fee cap for the blob data
		BlobHashes: nil,                   // blob hashes in the transaction
		Sidecar:    nil,                   // sidecar data in the transaction
	})

	return types.SignTx(tx, types.NewCancunSigner(chainId), ctxt.net.GetSessionSponsor().PrivateKey)
}

func checkBlocksSanity(t *testing.T, client *ethclient.Client) {
	// This check is a regression from an issue found while fetching a block by
	// number where the last block was not correctly serialized
	require := require.New(t)

	lastBlock, err := client.BlockByNumber(t.Context(), nil)
	require.NoError(err)

	for i := uint64(0); i < lastBlock.Number().Uint64(); i++ {
		_, err := client.BlockByNumber(t.Context(), big.NewInt(int64(i)))
		require.NoError(err)
	}
}

type testContext struct {
	net    *IntegrationTestNet
	client *ethclient.Client
}

func MakeTestContext(t *testing.T) *testContext {
	net := StartIntegrationTestNet(t)

	client, err := net.GetClient()
	require.NoError(t, err)

	return &testContext{net, client}
}

func (tc *testContext) Close() {
	tc.client.Close()
	tc.net.Stop()
}

// helper functions to calculate blob base fee based on https://eips.ethereum.org/EIPS/eip-4844#gas-accounting
func getBlobBaseFeeFrom(header *types.Header) uint64 {
	cancunTime := uint64(0)
	config := &params.ChainConfig{}
	config.LondonBlock = big.NewInt(0)
	config.CancunTime = &cancunTime
	config.BlobScheduleConfig = &params.BlobScheduleConfig{
		Cancun: params.DefaultCancunBlobConfig,
	}
	return eip4844.CalcBlobFee(config, header).Uint64()
}
