package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/0xsoniclabs/sonic/gossip"
	"github.com/0xsoniclabs/sonic/integration"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/utils/cachescale"
	"github.com/cockroachdb/pebble"
	"github.com/ethereum/go-ethereum/common"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

// Creation of a usable Sonic DB directory:
//
// > go run ./cmd/sonictool --datadir /media/herbert/WorkData/sonic/chain_data_with_fake_logs genesis fake 1
//
// Copy in the tables 't' and 'r' from a full sonic DB with logs using the code
// in this file (with the copyTable function and the adjusted file paths).
//
// > go run ./utils/leap/bench
//
// Run the Sonic client without network:
//
// > go run ./cmd/sonicd --datadir /media/herbert/WorkData/sonic/chain_data_with_fake_logs --maxpeers 0 --http --http.api eth --pprof

func main() {

	src := "/media/herbert/WorkData/fantom/datadir-logs-59-63M-only"
	dst := "/media/herbert/WorkData/sonic/chain_data_with_fake_logs"

	err := copyTables(
		filepath.Join(src, "chaindata", "gossip"),
		filepath.Join(dst, "chaindata", "gossip"),
		't',
		func(key []byte) bool {
			if len(key) != 82 {
				return false
			}
			// prefix+topic+topicN+(blockN+TxHash+logIndex)
			block := binary.BigEndian.Uint64(key[1+32+1 : 1+32+1+8])
			return 59_000_000 <= block && block <= 63_000_000
		},
	)
	if err != nil {
		fmt.Printf("Failed to copy 't' table: %v\n", err)
		return
	}

	err = copyTables(
		filepath.Join(src, "chaindata", "gossip"),
		filepath.Join(dst, "chaindata", "gossip"),
		'r',
		func(key []byte) bool {
			if len(key) != 49 {
				return false
			}
			// (blockN+TxHash+logIndex)
			block := binary.BigEndian.Uint64(key[1 : 1+8])
			return 59_000_000 <= block && block <= 63_000_000
		},
	)
	if err != nil {
		fmt.Printf("Failed to copy 'r' table: %v\n", err)
		return
	}

	return

	path := "/media/herbert/WorkData/chaindata/sonic_main_net/sonic-new-filter-datadir"
	listKeys(filepath.Join(path, "chaindata", "gossip"))
	return

	db, err := openDb(path)
	if err != nil {
		fmt.Printf("Failed to open DB: %v\n", err)
		return
	}
	defer func() {
		if err := db.Close(); err != nil {
			fmt.Printf("Failed to close DB: %v\n", err)
		}
	}()

	// TODO: check whether there are log entries in this Database.

	// [map[fromBlock:0x3bc2bd6 toBlock:0x3bc2bd6 topics:[0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef]]]

	block, err := strconv.ParseUint("3bc2bd6", 16, 64)
	if err != nil {
		fmt.Printf("Failed to parse block number: %v\n", err)
		return
	}

	topic := hash.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")

	logs, err := db.EvmStore().EvmLogs.FindInBlocks(
		context.Background(),
		idx.Block(block), idx.Block(block),
		[][]common.Hash{{}, {common.Hash(topic)}},
	)
	if err != nil {
		fmt.Printf("Failed to find logs: %v\n", err)
		return
	}

	fmt.Printf("Found %d logs\n", len(logs))

}

func openDb(
	dataDir string,
) (*gossip.Store, error) {
	chaindataDir := filepath.Join(dataDir, "chaindata")
	carmenDir := filepath.Join(dataDir, "carmen")

	if stat, err := os.Stat(chaindataDir); err != nil || !stat.IsDir() {
		return nil, fmt.Errorf("unable to validate: datadir does not contain chaindata")
	}
	if stat, err := os.Stat(carmenDir); err != nil || !stat.IsDir() {
		return nil, fmt.Errorf("unable to validate: datadir does not contain carmen")
	}

	dbs, err := integration.GetDbProducer(chaindataDir, integration.DBCacheConfig{
		Cache:   480 * opt.MiB,
		Fdlimit: 100,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to make DB producer: %v", err)
	}

	gdbConfig := gossip.DefaultStoreConfig(cachescale.Identity)
	gdbConfig.EVM.StateDb.Directory = carmenDir
	gdb, err := gossip.NewStore(dbs, gdbConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create gossip store: %w", err)
	}

	err = gdb.EvmStore().Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open EvmStore: %v", err)
	}

	return gdb, nil

}

func listKeys(
	path string,
) error {

	// topic+topicN+(blockN+TxHash+logIndex) -> topic_count (where topicN=0 is for address)
	//		Topic kvdb.Store `table:"t"`
	// (blockN+TxHash+logIndex) -> ordered topic_count topics, blockHash, address, data
	//		Logrec kvdb.Store `table:"r"`

	db, err := pebble.Open(path, &pebble.Options{})
	if err != nil {
		return fmt.Errorf("failed to open DB: %v", err)
	}
	defer db.Close()

	fmt.Printf("Listing some keys with 't' prefix:\n")
	iter, err := db.NewIter(&pebble.IterOptions{
		LowerBound: []byte{'t'},
		UpperBound: []byte{'t' + 1},
	})
	if err != nil {
		return fmt.Errorf("failed to create iterator: %v", err)
	}

	count := 0
	for iter.First(); iter.Valid(); iter.Next() {
		fmt.Printf("Key: %x\n", iter.Key())

		data := iter.Key()
		table := rune(data[0])
		data = data[1:]

		var topic [32]byte
		copy(topic[:], data[0:32])
		data = data[32:]

		pos := uint8(data[0])
		data = data[1:]

		block := binary.BigEndian.Uint64(data[0:8])
		data = data[8:]

		var txHash [32]byte
		copy(txHash[:], data[0:32])
		data = data[32:]

		index := binary.BigEndian.Uint32(data[0:4])

		fmt.Printf(
			"%c topic: %x pos: %d block: %d (%x) tx: %x index: %d\n",
			table,
			topic,
			pos,
			block, block,
			txHash,
			index,
		)

		if block > 60_000_000 {
			continue
		}

		count++
		if count > 10 {
			break
		}
	}
	if err := iter.Close(); err != nil {
		return fmt.Errorf("failed to close iterator: %v", err)
	}
	if count == 0 {
		fmt.Printf("  (no keys found)\n")
	}

	return nil
	// Look for key entries with 't' or 'r' prefix.
}

func copyTables(
	src, dst string,
	prefix byte,
	keyFilter func([]byte) bool,
) error {

	// topic+topicN+(blockN+TxHash+logIndex) -> topic_count (where topicN=0 is for address)
	//		Topic kvdb.Store `table:"t"`
	// (blockN+TxHash+logIndex) -> ordered topic_count topics, blockHash, address, data
	//		Logrec kvdb.Store `table:"r"`

	in, err := pebble.Open(src, &pebble.Options{
		ErrorIfNotExists: true,
		ReadOnly:         true,
	})
	if err != nil {
		return fmt.Errorf("failed to open DB: %v", err)
	}
	defer in.Close()

	out, err := pebble.Open(dst, &pebble.Options{})
	if err != nil {
		return fmt.Errorf("failed to open DB: %v", err)
	}
	defer out.Close()

	iter, err := in.NewIter(&pebble.IterOptions{
		LowerBound: []byte{prefix},
		UpperBound: []byte{prefix + 1},
	})
	if err != nil {
		return fmt.Errorf("failed to create iterator: %v", err)
	}

	count := 0
	batch := out.NewBatch()
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		if !keyFilter(key) {
			continue
		}
		batch.Set(iter.Key(), iter.Value(), nil)

		count++
		if count%100_000 == 0 {
			fmt.Printf("Copied %d entries\n", count)
			if err := batch.Commit(nil); err != nil {
				return fmt.Errorf("failed to commit batch: %v", err)
			}
			batch = out.NewBatch()
		}
	}
	if err := batch.Commit(nil); err != nil {
		return fmt.Errorf("failed to commit batch: %v", err)
	}
	if err := iter.Close(); err != nil {
		return fmt.Errorf("failed to close iterator: %v", err)
	}

	fmt.Printf("Copied total %d entries with prefix %c\n", count, prefix)

	// Copied total 14417849414 entries with prefix t (284 GiB)
	// Copied total 4204029363 entries with prefix r (649 GiB total)

	return nil
}
