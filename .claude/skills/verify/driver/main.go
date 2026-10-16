// Throwaway RPC driver for verifying the ledger change against a live fakenet.
package main

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

const keyHex = "163f5f0f9a621d72fedd85ffca3d08d131ab4e812181e0d30ffd1c885d20aac7"

var (
	ctx    = context.Background()
	key, _ = crypto.HexToECDSA(keyHex)
	from   = crypto.PubkeyToAddress(key.PublicKey)
)

func must(err error) {
	if err != nil {
		fmt.Println("ERROR:", err)
		os.Exit(1)
	}
}

func send(c *ethclient.Client, to *common.Address, value *big.Int, data []byte, gas uint64) *types.Receipt {
	chainID, err := c.ChainID(ctx)
	must(err)
	nonce, err := c.PendingNonceAt(ctx, from)
	must(err)
	price, err := c.SuggestGasPrice(ctx)
	must(err)
	price = new(big.Int).Mul(price, big.NewInt(2))
	tx := types.NewTx(&types.LegacyTx{Nonce: nonce, To: to, Value: value, Data: data, Gas: gas, GasPrice: price})
	signed, err := types.SignTx(tx, types.LatestSignerForChainID(chainID), key)
	must(err)
	must(c.SendTransaction(ctx, signed))
	for i := 0; i < 100; i++ {
		r, err := c.TransactionReceipt(ctx, signed.Hash())
		if err == nil {
			return r
		}
		time.Sleep(100 * time.Millisecond)
	}
	must(fmt.Errorf("no receipt for %s", signed.Hash()))
	return nil
}

func main() {
	c, err := ethclient.Dial(os.Args[1])
	must(err)
	switch os.Args[2] {
	case "info":
		n, err := c.BlockNumber(ctx)
		must(err)
		b, err := c.BalanceAt(ctx, from, nil)
		must(err)
		var epoch string
		must(c.Client().Call(&epoch, "eth_currentEpoch"))
		h, err := c.HeaderByNumber(ctx, nil)
		must(err)
		fmt.Printf("from=%s block=%d epoch=%s balance=%s headRoot=%s\n", from, n, epoch, b, h.Root)
	case "transfer":
		n, _ := strconv.Atoi(os.Args[3])
		to := common.HexToAddress("0x00000000000000000000000000000000000beef1")
		before, err := c.BalanceAt(ctx, to, nil)
		must(err)
		for i := 0; i < n; i++ {
			r := send(c, &to, big.NewInt(1000), nil, 21000)
			fmt.Printf("transfer %d: status=%d block=%d gasUsed=%d\n", i, r.Status, r.BlockNumber, r.GasUsed)
		}
		after, err := c.BalanceAt(ctx, to, nil)
		must(err)
		fmt.Printf("recipient balance %s -> %s (delta %s, expected %d)\n", before, after, new(big.Int).Sub(after, before), 1000*n)
	case "logs":
		// initcode returns runtime PUSH1 0x42 PUSH1 0 PUSH1 0 LOG1 STOP
		initcode := common.FromHex("0x67604260006000a10060005260086018f3")
		r := send(c, nil, nil, initcode, 200000)
		fmt.Printf("deploy: status=%d block=%d contract=%s\n", r.Status, r.BlockNumber, r.ContractAddress)
		addr := r.ContractAddress
		sub := make(chan types.Log, 16)
		var subscription ethereum.Subscription
		if len(os.Args) > 3 {
			ws, err := ethclient.Dial(os.Args[3])
			must(err)
			subscription, err = ws.SubscribeFilterLogs(ctx, ethereum.FilterQuery{Addresses: []common.Address{addr}}, sub)
			must(err)
		}
		for i := 0; i < 3; i++ {
			r := send(c, &addr, nil, nil, 50000)
			fmt.Printf("call %d: status=%d block=%d receiptLogs=%d", i, r.Status, r.BlockNumber, len(r.Logs))
			if len(r.Logs) > 0 {
				fmt.Printf(" topic=%s", r.Logs[0].Topics[0])
			}
			fmt.Println()
		}
		logs, err := c.FilterLogs(ctx, ethereum.FilterQuery{Addresses: []common.Address{addr}, FromBlock: r.BlockNumber})
		must(err)
		fmt.Printf("eth_getLogs: %d logs\n", len(logs))
		if subscription != nil {
			got := 0
			timeout := time.After(3 * time.Second)
		loop:
			for got < 3 {
				select {
				case l := <-sub:
					got++
					fmt.Printf("ws log: block=%d tx=%s removed=%v\n", l.BlockNumber, l.TxHash, l.Removed)
				case <-timeout:
					break loop
				}
			}
			fmt.Printf("ws subscription: %d logs\n", got)
		}
	case "rules":
		addr := common.HexToAddress("0xd100ae0000000000000000000000000000000000")
		diff := []byte(os.Args[3])
		data := crypto.Keccak256([]byte("updateNetworkRules(bytes)"))[:4]
		data = append(data, common.LeftPadBytes(big.NewInt(32).Bytes(), 32)...)
		data = append(data, common.LeftPadBytes(big.NewInt(int64(len(diff))).Bytes(), 32)...)
		data = append(data, common.RightPadBytes(diff, (len(diff)+31)/32*32)...)
		r := send(c, &addr, nil, data, 1_000_000)
		fmt.Printf("updateNetworkRules: status=%d block=%d\n", r.Status, r.BlockNumber)
	case "burst":
		chainID, err := c.ChainID(ctx)
		must(err)
		nonce, err := c.PendingNonceAt(ctx, from)
		must(err)
		price, err := c.SuggestGasPrice(ctx)
		must(err)
		price = new(big.Int).Mul(price, big.NewInt(2))
		to := common.HexToAddress("0x00000000000000000000000000000000000beef2")
		var hashes []common.Hash
		for i := 0; i < 20; i++ {
			var tx *types.Transaction
			if i == 10 {
				tx = types.NewTx(&types.LegacyTx{Nonce: nonce + uint64(i), Data: common.FromHex("0xfe"), Gas: 100000, GasPrice: price})
			} else {
				tx = types.NewTx(&types.LegacyTx{Nonce: nonce + uint64(i), To: &to, Value: big.NewInt(1), Gas: 21000, GasPrice: price})
			}
			signed, err := types.SignTx(tx, types.LatestSignerForChainID(chainID), key)
			must(err)
			must(c.SendTransaction(ctx, signed))
			hashes = append(hashes, signed.Hash())
		}
		blocks := map[uint64]int{}
		failed := 0
		for _, h := range hashes {
			for i := 0; i < 100; i++ {
				r, err := c.TransactionReceipt(ctx, h)
				if err == nil {
					blocks[r.BlockNumber.Uint64()]++
					if r.Status == 0 {
						failed++
					}
					break
				}
				time.Sleep(100 * time.Millisecond)
			}
		}
		bal, err := c.BalanceAt(ctx, to, nil)
		must(err)
		fmt.Printf("burst: txs per block=%v failed=%d recipient balance=%s (expected 19)\n", blocks, failed, bal)
	case "advance-epoch":
		addr := common.HexToAddress("0xd100ae0000000000000000000000000000000000")
		data := append(crypto.Keccak256([]byte("advanceEpochs(uint256)"))[:4], common.LeftPadBytes(big.NewInt(1).Bytes(), 32)...)
		r := send(c, &addr, nil, data, 1_000_000)
		fmt.Printf("advanceEpochs: status=%d block=%d\n", r.Status, r.BlockNumber)
	}
}
