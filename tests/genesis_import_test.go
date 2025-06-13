package tests

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestGenesis_NetworkCanCreateNewBlocksAfterExportImport(t *testing.T) {
	const numBlocks = 3
	require := require.New(t)

	net := StartIntegrationTestNet(t)

	// Produce a few blocks on the network.
	for range numBlocks {
		_, err := net.EndowAccount(common.Address{42}, big.NewInt(100))
		require.NoError(err, "failed to endow account")
	}

	// get headers for all blocks
	originalHeaders, err := net.GetHeaders()
	require.NoError(err)

	originalHashes := []common.Hash{}
	for _, header := range originalHeaders {
		originalHashes = append(originalHashes, header.Hash())
	}

	err = net.RestartWithExportImport()
	require.NoError(err)

	client, err := net.GetClient()
	require.NoError(err)
	defer client.Close()

	// check address 42 has balance
	balance42, err := client.BalanceAt(t.Context(), common.Address{42}, nil)
	require.NoError(err)
	require.Equal(int64(100*numBlocks), balance42.Int64(), "unexpected balance")

	// check headers are consistent with original hashes
	newHeaders, err := net.GetHeaders()
	require.NoError(err)
	require.LessOrEqual(len(originalHashes), len(newHeaders), "unexpected number of headers")
	for i := 0; i < len(originalHashes); i++ {
		require.Equal(originalHashes[i], newHeaders[i].Hash(), "unexpected header for block %d", i)
	}

	// Produce a few blocks on the network
	for range numBlocks {
		_, err := net.EndowAccount(common.Address{42}, big.NewInt(100))
		require.NoError(err, "failed to endow account")
	}

	// get headers for all blocks
	allHeaders, err := net.GetHeaders()
	require.NoError(err)

	// check headers from before the export are still reachable
	require.LessOrEqual(len(newHeaders), len(allHeaders), "unexpected number of headers")
	for i := 0; i < len(newHeaders); i++ {
		require.Equal(newHeaders[i].Hash(), newHeaders[i].Hash(), "unexpected header for block %d", i)
	}
}
