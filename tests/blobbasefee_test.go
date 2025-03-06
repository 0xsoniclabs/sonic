package tests

import (
	"bytes"
	"context"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/tests/contracts/blobbasefee"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/require"
)

func TestBlobBaseFee_CanReadBlobBaseFeeFromHeadAndBlockAndHistory(t *testing.T) {
	require := require.New(t)
	net := StartIntegrationTestNet(t)

	// Deploy the blob base fee contract.
	contract, _, err := DeployContract(net, blobbasefee.DeployBlobbasefee)
	require.NoError(err, "failed to deploy contract; ", err)

	// Collect the current blob base fee from the head state.
	receipt, err := net.Apply(contract.LogCurrentBlobBaseFee)
	require.NoError(err, "failed to log current blob base fee; ", err)
	require.Equal(len(receipt.Logs), 1, "unexpected number of logs; expected 1, got ", len(receipt.Logs))

	entry, err := contract.ParseCurrentBlobBaseFee(*receipt.Logs[0])
	require.NoError(err, "failed to parse log; ", err)
	fromLog := entry.Fee.Uint64()

	// Collect the blob base fee from the block header.
	client, err := net.GetClient()
	require.NoError(err, "failed to get client; ", err)
	defer client.Close()

	block, err := client.BlockByNumber(context.Background(), receipt.BlockNumber)
	require.NoError(err, "failed to get block header; ", err)
	fromBlock := getBlobBaseFeeFrom(block.Header())

	// Collect the blob base fee from the archive.
	fromArchive, err := contract.GetBlobBaseFee(&bind.CallOpts{BlockNumber: receipt.BlockNumber})
	require.NoError(err, "failed to get blob base fee from archive; ", err)

	// call the blob base fee rpc method
	fromRpc := new(hexutil.Uint64)
	err = client.Client().Call(&fromRpc, "eth_blobBaseFee")
	require.NoError(err, "failed to get blob base fee from rpc; ", err)

	// we check blob base fee is one because it is not implemented yet. TODO issue #147
	require.Equal(fromLog, uint64(1), "invalid blob base fee from log; ", fromLog)
	require.Equal(fromLog, fromArchive.Uint64(), "blob base fee mismatch; from log %v, from archive %v", fromLog, fromArchive)
	require.Equal(fromLog, fromBlock, "blob base fee mismatch; from log %v, from block %v", fromLog, fromBlock)
	require.Equal(fromLog, uint64(*fromRpc), "blob base fee mismatch; from log %v, from rpc %v", fromLog, fromRpc)
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

func TestBlobBaseFee_CanReadBlobGasUsed(t *testing.T) {
	require := require.New(t)
	net := StartIntegrationTestNet(t)

	client, err := net.GetClient()
	require.NoError(err, "failed to get client; ", err)
	defer client.Close()

	// Get blob gas used from the block header of the latest block.
	block, err := client.BlockByNumber(context.Background(), nil)
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
