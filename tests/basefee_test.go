package tests

import (
	"testing"

	"github.com/0xsoniclabs/sonic/tests/contracts/basefee"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/stretchr/testify/require"
)

func TestBaseFee_CanReadBaseFeeFromHeadAndBlockAndHistory(t *testing.T) {
	net := StartIntegrationTestNet(t)

	// Deploy the base fee contract.
	contract, _, err := DeployContract(net, basefee.DeployBasefee)
	require.NoError(t, err)

	// Collect the current base fee from the head state.
	receipt, err := net.Apply(contract.LogCurrentBaseFee)
	require.NoError(t, err)
	require.Len(t, receipt.Logs, 1, "expected exactly one log entry for the base fee")

	entry, err := contract.ParseCurrentFee(*receipt.Logs[0])
	require.NoError(t, err)
	fromLog := entry.Fee

	// Collect the base fee from the block header.
	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	block, err := client.BlockByNumber(t.Context(), receipt.BlockNumber)
	require.NoError(t, err)
	fromBlock := block.BaseFee()

	// Collect the base fee from the archive.
	fromArchive, err := contract.GetBaseFee(&bind.CallOpts{BlockNumber: receipt.BlockNumber})
	require.NoError(t, err)

	require.GreaterOrEqual(t,
		fromLog.Sign(), 0,
		"base fee should be non-negative",
	)
	require.Equal(t,
		fromLog, fromBlock,
		"base fee from log should match base fee from block header",
	)
	require.Equal(t,
		fromLog, fromArchive,
		"base fee from log should match base fee from archive",
	)
}
