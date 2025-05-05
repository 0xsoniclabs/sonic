package tests

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/ethclient"
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
	// because blobs are arrays of 128KB, transactions with blobs are
	// considered oversized and rejected
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

func createTestBlobTransaction(t *testing.T, ctxt *testContext, data ...[]byte) (*types.Transaction, error) {
	require := require.New(t)

	chainId, err := ctxt.client.ChainID(context.Background())
	require.NoError(err, "failed to get chain ID::")

	nonce, err := ctxt.client.NonceAt(context.Background(), ctxt.net.GetSessionSponsor().Address(), nil)
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

	chainId, err := ctxt.client.ChainID(context.Background())
	require.NoError(err, "failed to get chain ID::")

	nonce, err := ctxt.client.NonceAt(context.Background(), ctxt.net.GetSessionSponsor().Address(), nil)
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

	lastBlock, err := client.BlockByNumber(context.Background(), nil)
	require.NoError(err)

	for i := uint64(0); i < lastBlock.Number().Uint64(); i++ {
		_, err := client.BlockByNumber(context.Background(), big.NewInt(int64(i)))
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
