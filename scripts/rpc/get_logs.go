package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

type RPCRequest struct {
	Jsonrpc string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	ID      int         `json:"id"`
}

const rpcURL = "https://rpc.sonic.soniclabs.com"

type Log struct {
	Address        common.Address
	Topics         []common.Hash
	Data           hexutil.Bytes
	BlockNumber    hexutil.Uint64
	TxHash         common.Hash
	TxIndex        hexutil.Uint
	BlockHash      common.Hash
	BlockTimestamp uint64
	Index          hexutil.Uint
	Removed        bool
}

func main() {

	lastBlockNumber, blockHash, err := getLastBlockNumberAndHash("latest")
	if err != nil {
		fmt.Println("Error getting last block hash:", err)
		return
	}
	fmt.Printf("Latest Block Number: %d, Hash: %s\n", lastBlockNumber, blockHash)

	blockTxs := getBlockTxs(blockHash)
	fmt.Printf("Number of transactions in block: %d\n", len(blockTxs))

	logs := getLogsFromBlockHash(blockHash)

	validateBlockAndLogTxHashes(blockTxs, logs)

	// for i := range 1000 {
	// 	blockNumber, blockHash, err := getLastBlockNumberAndHash(string(lastBlockNumber - uint64(i)))
	// 	if err != nil {
	// 		fmt.Println("Error getting last block hash:", err)
	// 		return
	// 	}

	// 	// fmt.Printf("Latest Block Number: %d, Hash: %s\n", blockNumber, blockHash)

	// 	logs := getLogsFromBlockHash(blockHash)
	// }

}

func jsonRequest(method string, params interface{}) ([]byte, error) {
	reqBody := RPCRequest{
		Jsonrpc: "2.0",
		Method:  method,
		Params:  params,
		ID:      1,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(rpcURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

// getLastBlockNumberAndHash retrieves the latest block number and hash via RPC call
func getLastBlockNumberAndHash(blockNumber string) (uint64, string, error) {

	type BlockResult struct {
		Hash   string         `json:"hash"`
		Number hexutil.Uint64 `json:"number"`
	}

	reqBody := RPCRequest{
		Jsonrpc: "2.0",
		Method:  "eth_getBlockByNumber",
		Params:  []interface{}{blockNumber, false},
		ID:      1,
	}

	body, err := jsonRequest(reqBody.Method, reqBody.Params)
	if err != nil {
		return 0, "", err
	}

	type RPCResponse struct {
		Result BlockResult `json:"result"`
	}

	var rpcResp RPCResponse
	err = json.Unmarshal(body, &rpcResp)
	if err != nil {
		return 0, "", err
	}

	return uint64(rpcResp.Result.Number), rpcResp.Result.Hash, nil
}

func getBlockTxs(blockHash string) []*types.Transaction {

	type BlockResult struct {
		Transactions []*types.Transaction `json:"transactions"`
	}

	reqBody := RPCRequest{
		Jsonrpc: "2.0",
		Method:  "eth_getBlockByHash",
		Params:  []interface{}{blockHash, true},
		ID:      1,
	}

	body, err := jsonRequest(reqBody.Method, reqBody.Params)
	if err != nil {
		fmt.Println("Error getting block transactions:", err)
		return nil
	}

	type RPCResponse struct {
		Result BlockResult `json:"result"`
	}

	var rpcResp RPCResponse
	err = json.Unmarshal(body, &rpcResp)
	if err != nil {
		fmt.Println("Error unmarshalling block transactions response:", err)
		return nil
	}

	return rpcResp.Result.Transactions

}

// getLogsFromBlockHash retrieves logs for a given block hash via RPC call
func getLogsFromBlockHash(blockHash string) []Log {

	type FilterParams struct {
		BlockHash string `json:"blockHash,omitempty"`
	}

	filter := FilterParams{
		BlockHash: blockHash,
	}

	reqBody := RPCRequest{
		Jsonrpc: "2.0",
		Method:  "eth_getLogs",
		Params:  []FilterParams{filter},
		ID:      1,
	}

	body, err := jsonRequest(reqBody.Method, reqBody.Params)
	if err != nil {
		fmt.Println("Error getting logs:", err)
		return nil
	}

	var result struct {
		Result []Log `json:"result"`
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		fmt.Println("Error unmarshalling logs response:", err)
		return nil
	}

	return result.Result
}

func validateBlockAndLogTxHashes(blockTxs []*types.Transaction, logs []Log) bool {
	for _, log := range logs {
		txIndex := log.TxIndex
		txHash := log.TxHash

		if int(len(blockTxs)) <= int(txIndex) {
			fmt.Printf("TxIndex %d out of range for block transactions\n", txIndex)
			return false
		}

		tx := blockTxs[txIndex]
		if tx.Hash() != txHash {
			fmt.Printf("Mismatch for log TxIndex %d: expected %s, got %s\n", txIndex, txHash.Hex(), tx.Hash().Hex())
			return false
		}
	}
	return true
}
