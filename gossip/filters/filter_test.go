// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package filters

import (
	"context"
	"math/big"
	"path"
	"testing"

	"github.com/0xsoniclabs/sonic/utils/adapters/ethdb2kvdb"
	"github.com/Fantom-foundation/lachesis-base/kvdb/table"

	"github.com/0xsoniclabs/sonic/topicsdb"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethdb/leveldb"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/triedb"
)

func testConfig() Config {
	return Config{
		IndexedLogsBlockRangeLimit:   1000,
		UnindexedLogsBlockRangeLimit: 1000,
	}
}

func makeReceipt(addr common.Address) *types.Receipt {
	receipt := types.NewReceipt(nil, false, 0)
	receipt.Logs = []*types.Log{
		{Address: addr},
	}
	return receipt
}

func BenchmarkFilters(b *testing.B) {
	dir := b.TempDir()

	backend := newTestBackend()

	db, err := leveldb.New(path.Join(dir, "backend-db"), 100, 1000, "", false)
	if err != nil {
		b.Fatal(err)
	}
	ldb := rawdb.NewDatabase(db)

	if err != nil {
		b.Fatal(err)
	}
	backend.db = rawdb.NewTable(ldb, "a")
	backend.logIndex = topicsdb.NewWithThreadPool(table.New(ethdb2kvdb.Wrap(ldb), []byte("b")))

	var (
		key1, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		addr1   = crypto.PubkeyToAddress(key1.PublicKey)
		addr2   = common.BytesToAddress([]byte("jeff"))
		addr3   = common.BytesToAddress([]byte("ethereum"))
		addr4   = common.BytesToAddress([]byte("random addresses please"))
	)

	genesis := getGenesisBlockForTesting(backend.db, addr1, big.NewInt(1000000))
	chain, receipts := core.GenerateChain(params.TestChainConfig, genesis, ethash.NewFaker(), backend.db, 100010, func(i int, gen *core.BlockGen) {
		switch i {
		case 2403:
			receipt := makeReceipt(addr1)
			gen.AddUncheckedReceipt(receipt)
		case 1034:
			receipt := makeReceipt(addr2)
			gen.AddUncheckedReceipt(receipt)
		case 34:
			receipt := makeReceipt(addr3)
			gen.AddUncheckedReceipt(receipt)
		case 99999:
			receipt := makeReceipt(addr4)
			gen.AddUncheckedReceipt(receipt)
		}
	})
	for i, block := range chain {
		rawdb.WriteBlock(backend.db, block)
		rawdb.WriteCanonicalHash(backend.db, block.Hash(), block.NumberU64())
		rawdb.WriteHeadBlockHash(backend.db, block.Hash())
		rawdb.WriteReceipts(backend.db, block.Hash(), block.NumberU64(), receipts[i])
	}
	b.ResetTimer()

	filter := NewRangeFilter(backend, testConfig(), 0, -1, []common.Address{addr1, addr2, addr3, addr4}, nil)

	for i := 0; i < b.N; i++ {
		logs, _ := filter.Logs(context.Background())
		if len(logs) != 4 {
			// TODO: fix it
			b.Fatal("expected 4 logs, got", len(logs))
		}
	}
}

func TestFilters(t *testing.T) {
	var (
		backend = newTestBackend()
		key1, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		addr    = crypto.PubkeyToAddress(key1.PublicKey)

		hash1 = common.BytesToHash([]byte("topic1"))
		hash2 = common.BytesToHash([]byte("topic2"))
		hash3 = common.BytesToHash([]byte("topic3"))
		hash4 = common.BytesToHash([]byte("topic4"))
	)

	genesis := getGenesisBlockForTesting(backend.db, addr, big.NewInt(1000000))
	chain, receipts := core.GenerateChain(params.TestChainConfig, genesis, ethash.NewFaker(), backend.db, 1000, func(i int, gen *core.BlockGen) {
		switch i {
		case 1:
			receipt := types.NewReceipt(nil, false, 0)
			receipt.Logs = []*types.Log{
				{
					BlockNumber: 1,
					Address:     addr,
					Topics:      []common.Hash{hash1},
				},
			}
			gen.AddUncheckedReceipt(receipt)
			gen.AddUncheckedTx(types.NewTransaction(1, common.HexToAddress("0x1"), big.NewInt(1), 1, big.NewInt(1), nil))
			backend.MustPushLogs(receipt.Logs...)

		case 2:
			receipt := types.NewReceipt(nil, false, 0)
			receipt.Logs = []*types.Log{
				{
					BlockNumber: 2,
					Address:     addr,
					Topics:      []common.Hash{hash2},
				},
			}
			gen.AddUncheckedReceipt(receipt)
			gen.AddUncheckedTx(types.NewTransaction(2, common.HexToAddress("0x2"), big.NewInt(2), 2, big.NewInt(2), nil))
			backend.MustPushLogs(receipt.Logs...)

		case 998:
			receipt := types.NewReceipt(nil, false, 0)
			receipt.Logs = []*types.Log{
				{
					BlockNumber: 998,
					Address:     addr,
					Topics:      []common.Hash{hash3},
				},
			}
			gen.AddUncheckedReceipt(receipt)
			gen.AddUncheckedTx(types.NewTransaction(998, common.HexToAddress("0x998"), big.NewInt(998), 998, big.NewInt(998), nil))
			backend.MustPushLogs(receipt.Logs...)

		case 999:
			receipt := types.NewReceipt(nil, false, 0)
			receipt.Logs = []*types.Log{
				{
					Address: addr,
					Topics:  []common.Hash{hash4},
				},
			}
			gen.AddUncheckedReceipt(receipt)
			gen.AddUncheckedTx(types.NewTransaction(999, common.HexToAddress("0x999"), big.NewInt(999), 999, big.NewInt(999), nil))
			backend.MustPushLogs(receipt.Logs...)
		}
	})
	for i, block := range chain {
		rawdb.WriteBlock(backend.db, block)
		rawdb.WriteCanonicalHash(backend.db, block.Hash(), block.NumberU64())
		rawdb.WriteHeadBlockHash(backend.db, block.Hash())
		rawdb.WriteReceipts(backend.db, block.Hash(), block.NumberU64(), receipts[i])
	}

	var (
		filter *Filter
		logs   []*types.Log
		err    error
	)

	filter = NewRangeFilter(backend, testConfig(), 0, -1, []common.Address{addr}, [][]common.Hash{{hash1, hash2, hash3, hash4}})
	logs, err = filter.Logs(context.Background())
	if err != nil {
		t.Error(err)
	}
	if len(logs) != 4 {
		t.Error("expected 4 log, got", len(logs))
	}

	filter = NewRangeFilter(backend, testConfig(), 900, 999, []common.Address{addr}, [][]common.Hash{{hash3}})
	logs, err = filter.Logs(context.Background())
	if err != nil {
		t.Error(err)
	}
	if len(logs) != 1 {
		t.Error("expected 1 log, got", len(logs))
	}

	if len(logs) > 0 && logs[0].Topics[0] != hash3 {
		t.Errorf("expected log[0].Topics[0] to be %x, got %x", hash3, logs[0].Topics[0])
	}

	filter = NewRangeFilter(backend, testConfig(), 990, -1, []common.Address{addr}, [][]common.Hash{{hash3}})
	logs, err = filter.Logs(context.Background())
	if err != nil {
		t.Error(err)
	}
	if len(logs) != 1 {
		t.Error("expected 1 log, got", len(logs))
	}
	if len(logs) > 0 && logs[0].Topics[0] != hash3 {
		t.Errorf("expected log[0].Topics[0] to be %x, got %x", hash3, logs[0].Topics[0])
	}

	filter = NewRangeFilter(backend, testConfig(), 1, 10, nil, [][]common.Hash{{hash1, hash2}})
	logs, err = filter.Logs(context.Background())
	if err != nil {
		t.Error(err)
	}
	if len(logs) != 2 {
		t.Error("expected 2 log, got", len(logs))
	}

	failHash := common.BytesToHash([]byte("fail"))
	filter = NewRangeFilter(backend, testConfig(), 0, -1, nil, [][]common.Hash{{failHash}})
	logs, err = filter.Logs(context.Background())
	if err != nil {
		t.Error(err)
	}
	if len(logs) != 0 {
		t.Error("expected 0 log, got", len(logs))
	}

	failAddr := common.BytesToAddress([]byte("failmenow"))
	filter = NewRangeFilter(backend, testConfig(), 0, -1, []common.Address{failAddr}, nil)
	logs, err = filter.Logs(context.Background())
	if err != nil {
		t.Error(err)
	}
	if len(logs) != 0 {
		t.Error("expected 0 log, got", len(logs))
	}

	filter = NewRangeFilter(backend, testConfig(), 0, -1, nil, [][]common.Hash{{failHash}, {hash1}})
	logs, err = filter.Logs(context.Background())
	if err != nil {
		t.Error(err)
	}
	if len(logs) != 0 {
		t.Error("expected 0 log, got", len(logs))
	}

}

func getGenesisBlockForTesting(db ethdb.Database, address common.Address, balance *big.Int) *types.Block {
	genesis := core.Genesis{
		Alloc:   types.GenesisAlloc{address: {Balance: balance}},
		BaseFee: big.NewInt(params.InitialBaseFee),
		Config: &params.ChainConfig{
			BerlinBlock:         new(big.Int),
			LondonBlock:         new(big.Int),
			IstanbulBlock:       new(big.Int),
			PetersburgBlock:     new(big.Int),
			ConstantinopleBlock: new(big.Int),
			ByzantiumBlock:      new(big.Int),
			EIP158Block:         new(big.Int),
			EIP155Block:         new(big.Int),
			EIP150Block:         new(big.Int),
			HomesteadBlock:      new(big.Int),
		},
	}
	return genesis.MustCommit(db, triedb.NewDatabase(db, triedb.HashDefaults))
}

func TestSortLogsByBlockNumberAndLogIndex(t *testing.T) {
	logs := []*types.Log{
		{BlockNumber: 100, Index: 2},
		{BlockNumber: 200, Index: 1},
		{BlockNumber: 400, Index: 22},
		{BlockNumber: 100, Index: 1},
		{BlockNumber: 300, Index: 0},
		{BlockNumber: 400, Index: 20},
		{BlockNumber: 100, Index: 3},
		{BlockNumber: 200, Index: 0},
	}

	sortLogsByBlockNumberAndLogIndex(logs)

	expected := []struct {
		blockNumber uint64
		index       uint
	}{
		{100, 1},
		{100, 2},
		{100, 3},
		{200, 0},
		{200, 1},
		{300, 0},
		{400, 20},
		{400, 22},
	}

	for i, log := range logs {
		if log.BlockNumber != expected[i].blockNumber || log.Index != expected[i].index {
			t.Errorf("Unexpected log at position %d: got (BlockNumber: %d, Index: %d), want (BlockNumber: %d, Index: %d)",
				i, log.BlockNumber, log.Index, expected[i].blockNumber, expected[i].index)
		}
	}
}
