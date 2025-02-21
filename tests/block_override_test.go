package tests

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/ethapi"
	"github.com/0xsoniclabs/sonic/tests/contracts/block_hash"
	block_override "github.com/0xsoniclabs/sonic/tests/contracts/blockoverride"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	req "github.com/stretchr/testify/require"
)

func TestBlockOverride(t *testing.T) {
	require := req.New(t)
	net, err := StartIntegrationTestNet(t.TempDir())
	if err != nil {
		t.Fatalf("Failed to start the fake network: %v", err)
	}
	defer net.Stop()

	// Deploy the block hash observer contract.
	_, receipt, err := DeployContract(net, block_override.DeployBlockOverride)
	require.NoError(err, "failed to deploy contract; %v", err)
	contractAddress := receipt.ContractAddress
	//contractCreationBlock := receipt.BlockNumber.Uint64()

	netClient, err := net.GetClient()
	require.NoError(err, "failed to get client; %v", err)

	contract, err := block_hash.NewBlockHash(contractAddress, netClient)
	require.NoError(err, "failed to instantiate contract")

	receiptObserve, err := net.Apply(contract.Observe)
	require.NoError(err, "failed to observe block hash; %v", err)
	require.Equal(types.ReceiptStatusSuccessful, receiptObserve.Status,
		"failed to observe block hash; %v", err,
	)

	blockNumber := receiptObserve.BlockNumber.Uint64()

	rpcClient := netClient.Client()

	time := uint64(1234)
	gasLimit := uint64(567890)

	blockOverrides := &ethapi.BlockOverrides{
		Number:      (*hexutil.Big)(big.NewInt(42)),
		Difficulty:  (*hexutil.Big)(big.NewInt(1)),
		Time:        (*hexutil.Uint64)(&time),
		GasLimit:    (*hexutil.Uint64)(&gasLimit),
		Coinbase:    &common.Address{1},
		Random:      &common.Hash{2},
		BaseFee:     (*hexutil.Big)(big.NewInt(1_000)),
		BlobBaseFee: (*hexutil.Big)(big.NewInt(100)),
	}

	params, err := makeEthCall(t, rpcClient, contractAddress, blockNumber, nil)
	require.NoError(err, "failed to make eth_call; %v", err)

	paramsOverride, err := makeEthCall(t, rpcClient, contractAddress, blockNumber, blockOverrides)
	require.NoError(err, "failed to make eth_call; %v", err)

	t.Logf("params: %v", params)
	t.Logf("params: %v", paramsOverride)

	err = CompareBlockParameters(params, paramsOverride)
	require.NoError(err, "failed to compare block parameters; %v", err)

}

type BlockParameters struct {
	Number      *big.Int
	Difficulty  *big.Int
	Time        *big.Int
	GasLimit    *big.Int
	Coinbase    common.Address
	Random      common.Hash
	BaseFee     *big.Int
	BlobBaseFee *big.Int
}

func getBlockParameters(data []byte) (BlockParameters, error) {

	if len(data) != 256 {
		return BlockParameters{}, fmt.Errorf("invalid data length: %d, expected 256", len(data))
	}

	return BlockParameters{
		Number:      new(big.Int).SetBytes(data[:32]),
		Difficulty:  new(big.Int).SetBytes(data[32:64]),
		Time:        new(big.Int).SetBytes(data[64:96]),
		GasLimit:    new(big.Int).SetBytes(data[96:128]),
		Coinbase:    common.BytesToAddress(data[128:160]),
		Random:      common.BytesToHash(data[160:192]),
		BaseFee:     new(big.Int).SetBytes(data[192:224]),
		BlobBaseFee: new(big.Int).SetBytes(data[224:]),
	}, nil
}

func (bp *BlockParameters) String() string {
	return fmt.Sprintf(
		"number: %v difficulty: %v time: %v gasLimit: %v coinbase: %v random: %v baseFee: %v blobbasefee: %v",
		bp.Number, bp.Difficulty, bp.Time, bp.GasLimit, bp.Coinbase, bp.Random, bp.BaseFee, bp.BlobBaseFee,
	)
}

func makeEthCall(t *testing.T, rpcClient *rpc.Client, contractAddress common.Address, blockNumber uint64, blockOverrides *ethapi.BlockOverrides) (BlockParameters, error) {
	require := req.New(t)

	// function getBlockParameters on deployed contract
	params := map[string]interface{}{
		"to":   contractAddress.String(),
		"data": "0xa3289b77",
	}

	var res interface{}
	err := rpcClient.Call(&res, "eth_call", params, hexutil.EncodeUint64(blockNumber), nil, blockOverrides)
	require.NoError(err, "failed to call eth_call; %v", err)

	if s, ok := res.(string); ok {
		b, err := hexutil.Decode(s)
		require.NoError(err, "failed to decode result hex; %v", err)

		params, err := getBlockParameters(b)
		require.NoError(err, "failed to decode block parameters; %v", err)

		return params, nil
	} else {
		return BlockParameters{}, fmt.Errorf("invalid result type: %T", res)
	}
}

// CompareBlockParameters compares two BlockParameters objects and returns an error if any fields were not overridden
func CompareBlockParameters(params1, params2 BlockParameters) error {
	if params1.Number.Cmp(params2.Number) == 0 {
		return fmt.Errorf("Number field was not overridden: %v", params1.Number)
	}
	if params1.Difficulty.Cmp(params2.Difficulty) == 0 {
		return fmt.Errorf("Difficulty field was not overridden: %v", params1.Difficulty)
	}
	if params1.Time.Cmp(params2.Time) == 0 {
		return fmt.Errorf("Time field was not overridden: %v", params1.Time)
	}
	if params1.GasLimit.Cmp(params2.GasLimit) == 0 {
		return fmt.Errorf("GasLimit field was not overridden: %v", params1.GasLimit)
	}
	if params1.Coinbase == params2.Coinbase {
		return fmt.Errorf("Coinbase field was not overridden: %v", params1.Coinbase)
	}
	if params1.Random == params2.Random {
		return fmt.Errorf("Random field was not overridden: %v", params1.Random)
	}
	if params1.BaseFee.Cmp(params2.BaseFee) == 0 {
		return fmt.Errorf("BaseFee field was not overridden: %v", params1.BaseFee)
	}
	if params1.BlobBaseFee.Cmp(params2.BlobBaseFee) == 0 {
		return fmt.Errorf("BlobBaseFee field was not overridden: %v", params1.BlobBaseFee)
	}
	return nil
}
