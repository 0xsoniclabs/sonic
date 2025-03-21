package tests

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

func TestCollect(t *testing.T) {
	require := require.New(t)

	const URL = "https://rpc.soniclabs.com"

	client, err := ethclient.Dial(URL)
	require.NoError(err)
	defer client.Close()

	fmt.Printf("block, tx, limit, gas, hash\n")

	const S = uint64(15_000_000)
	const N = uint64(1000)
	for i := S; i < S+N; i++ {
		txs, err := getTransactions(client, i)
		require.NoError(err)
		for j, tx := range txs {
			limit, err := strconv.ParseUint(tx.Gas, 0, 64)
			require.NoError(err)
			gas, err := getUsedGas(client, tx.Hash)
			require.NoError(err)
			fmt.Printf("%d, %d, %d, %d, %s\n", i, j, limit, gas, tx.Hash)
		}
	}

	t.Fail()
}

type transaction struct {
	Hash string
	Gas  string
}

func getTransactions(
	client *ethclient.Client,
	blockNumber uint64,
) ([]transaction, error) {
	res := struct {
		Transactions []transaction
	}{}
	err := client.Client().Call(&res, "eth_getBlockByNumber", fmt.Sprintf("0x%x", blockNumber), true)
	if err != nil {
		return nil, err
	}
	return res.Transactions, nil
}

func getUsedGas(
	client *ethclient.Client,
	hash string,
) (uint64, error) {
	res := struct {
		GasUsed string
	}{}
	err := client.Client().Call(&res, "eth_getTransactionReceipt", hash)
	if err != nil {
		return 0, err
	}
	return strconv.ParseUint(res.GasUsed, 0, 64)
}
