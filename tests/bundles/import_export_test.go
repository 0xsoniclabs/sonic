package bundles

import (
	"math/big"
	"os"
	"path/filepath"
	"testing"

	sonictool "github.com/0xsoniclabs/sonic/cmd/sonictool/app"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"

	"github.com/stretchr/testify/require"
)

func TestIntegrationTestNet_ExportGenesisToFixedLocation_WithoutBundles(t *testing.T) {
	upgrades := opera.GetBrioUpgrades()
	upgrades.TransactionBundles = true
	net := tests.StartIntegrationTestNet(t,
		tests.IntegrationTestNetOptions{Upgrades: &upgrades})

	net.Stop()

	// Use the first node's directory as the data directory
	dataDir := filepath.Join(net.GetDirectory(), "state")
	exportPath := "test_exported_genesis" // Fixed, non-temporary location

	// Remove the file if it already exists
	_ = os.Remove(exportPath)

	// Run the export command
	err := sonictool.RunWithArgs([]string{
		"sonictool",
		"--datadir", dataDir,
		"genesis", "export", exportPath,
	})
	require.NoError(t, err, "Failed to export genesis to fixed location")

	// Check that the file now exists
	_, err = os.Stat(exportPath)
	require.NoError(t, err, "Exported genesis file does not exist at fixed location")
}

func TestIntegrationTestNet_ExportGenesisToFixedLocation_WithBundles(t *testing.T) {
	upgrades := opera.GetBrioUpgrades()
	upgrades.TransactionBundles = true
	net := tests.StartIntegrationTestNet(t,
		tests.IntegrationTestNetOptions{Upgrades: &upgrades})

	// runBundle(t, net)
	net.Stop()

	// Use the first node's directory as the data directory
	dataDir := filepath.Join(net.GetDirectory(), "state")
	exportPath := "test_exported_genesis_with_bundles" // Fixed, non-temporary location

	// Remove the file if it already exists
	_ = os.Remove(exportPath)

	// Run the export command
	err := sonictool.RunWithArgs([]string{
		"sonictool",
		"--datadir", dataDir,
		"genesis", "export", exportPath,
	})
	require.NoError(t, err, "Failed to export genesis to fixed location")

	// Check that the file now exists
	_, err = os.Stat(exportPath)
	require.NoError(t, err, "Exported genesis file does not exist at fixed location")
}

func TestIntegrationTestNet_ImportGenesisFromFixedLocation_WithoutBundles(t *testing.T) {

	net := tests.StartIntegrationTestNet(t)
	net.Stop()

	dataDir := filepath.Join(net.GetDirectory(), "state")
	exportPath := "test_exported_genesis" // Fixed, non-temporary location

	// clean client state
	err := os.RemoveAll(dataDir)
	require.NoError(t, err)

	// import genesis file
	err = sonictool.RunWithArgs([]string{
		"sonictool",
		"--datadir", dataDir,
		"genesis", "--experimental", exportPath,
	})
	require.NoError(t, err, "Failed to import genesis from fixed location")
	require.NoError(t, net.Restart())
	_ = tests.MakeAccountWithBalance(t, net, big.NewInt(10))
}

func TestIntegrationTestNet_ImportGenesisFromFixedLocation_WithBundles(t *testing.T) {

	net := tests.StartIntegrationTestNet(t)
	net.Stop()

	dataDir := filepath.Join(net.GetDirectory(), "state")
	exportPath := "test_exported_genesis_with_bundles" // Fixed, non-temporary location

	// clean client state
	err := os.RemoveAll(dataDir)
	require.NoError(t, err)

	// import genesis file
	err = sonictool.RunWithArgs([]string{
		"sonictool",
		"--datadir", dataDir,
		"genesis", "--experimental", exportPath,
	})
	require.NoError(t, err, "Failed to import genesis from fixed location")
	require.NoError(t, net.Restart())

	_ = tests.MakeAccountWithBalance(t, net, big.NewInt(10))
}

// runBundle prepares and runs a bundle with a single transaction,
// waits for its execution and returns the envelop transaction hash and bundle info.
// func runBundle(t *testing.T, net *tests.IntegrationTestNet) (common.Hash, ethapi.RPCBundleInfo) {
// 	t.Helper()

// 	client, err := net.GetClient()
// 	require.NoError(t, err)
// 	defer client.Close()

// 	// prepare a bundle with a single transaction
// 	gasPrice, err := client.SuggestGasPrice(t.Context())
// 	require.NoError(t, err)

// 	sender := net.GetSessionSponsor()
// 	tx := ethereum.CallMsg{
// 		From:     sender.Address(),
// 		To:       &common.Address{0x42},
// 		GasPrice: gasPrice,
// 	}

// 	earliest, err := client.BlockNumber(t.Context())
// 	require.NoError(t, err)
// 	earliestBlock := int64(earliest)
// 	latest := earliestBlock + 100

// 	preparedBundle, err := PrepareBundle(t, client, []ethereum.CallMsg{tx}, &earliestBlock, &latest)
// 	require.NoError(t, err)
// 	signer := types.LatestSignerForChainID(net.GetChainId())

// 	txs := make([]*types.Transaction, len(preparedBundle.Transactions))
// 	for i, txArgs := range preparedBundle.Transactions {
// 		txs[i], err = types.SignTx(txArgs.ToTransaction(), signer, sender.PrivateKey)
// 		require.NoError(t, err)
// 	}

// 	// Submit the bundle
// 	bundleHash, err := SubmitBundle(client, txs, preparedBundle.ExecutionPlan)
// 	require.NoError(t, err)
// 	info, err := WaitForBundlesExecution(t.Context(), client.Client(), []common.Hash{bundleHash})
// 	require.NoError(t, err)

// 	return bundleHash, *info[0]
// }
