// Copyright 2026 Sonic Operations Ltd
// This file is part of the Sonic Client
//
// Sonic is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Sonic is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Sonic. If not, see <http://www.gnu.org/licenses/>.

package app

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/0xsoniclabs/sonic/config/flags"
	"github.com/ethereum/go-ethereum/ethdb/pebble"
	"gopkg.in/urfave/cli.v1"
)

type prefixStats struct {
	count     int
	keyLenSum int
	valLenSum int
}

func analyzeDbs(ctx *cli.Context) error {
	dataDir := ctx.GlobalString(flags.DataDirFlag.Name)
	if dataDir == "" {
		return fmt.Errorf("--%s needs to be set", flags.DataDirFlag.Name)
	}
	chaindataDir := filepath.Join(dataDir, "chaindata")

	dbDirs, err := os.ReadDir(chaindataDir)
	if err != nil {
		return fmt.Errorf("failed to read chaindata directory: %v", err)
	}
	sort.Slice(dbDirs, func(i, j int) bool {
		return dbDirs[i].Name() < dbDirs[j].Name()
	})

	for _, dbDir := range dbDirs {
		if !dbDir.IsDir() {
			continue
		}
		path := filepath.Join(chaindataDir, dbDir.Name())

		statsForPrefixes, err := analyzeDb(path)
		if err != nil {
			return fmt.Errorf("failed to analyze DB %s: %v", path, err)
		}

		printTableStats(dbDir.Name(), statsForPrefixes)
	}
	return nil
}

func analyzeDb(path string) (res map[string]*prefixStats, err error) {
	db, err := pebble.New(path, 0, 0, "pebble", true)
	if err != nil {
		return nil, fmt.Errorf("failed to open DB: %v", err)
	}
	defer func() {
		if cerr := db.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("failed to close DB: %v", cerr)
		}
	}()

	foundPrefixes := make(map[string]*prefixStats)

	iter := db.NewIterator(nil, nil)
	for iter.Next() {
		key := iter.Key()
		val := iter.Value()
		if len(key) > 0 {
			prefix := string(key[0])
			stats, ok := foundPrefixes[prefix]
			if !ok {
				stats = &prefixStats{}
				foundPrefixes[prefix] = stats
			}
			stats.count++
			stats.keyLenSum += len(key)
			stats.valLenSum += len(val)
		}
	}
	iter.Release()

	return foundPrefixes, nil
}

func getTableNames(dbName string) map[string]string {
	epochGossipStore := map[string]string{
		// gossip/store_epoch.go
		"H": "Heads",
		"v": "DagIndex",
		"t": "LastEvents",
	}
	gossipStore := map[string]string{
		// gossip/store.go
		"_": "Version",
		"D": "BlockEpochState",
		"h": "BlockEpochStateHistory",
		"e": "Events",
		"b": "Blocks",
		"P": "EpochBlocks",
		"g": "Genesis",
		"U": "UpgradeHeights",
		"C": "CommitteeCertificates",
		"c": "BlockCertificates",
		"l": "HighestLamport",
		"V": "NetworkVersion",
		"B": "BlockHashes",
		// gossip/evmstore/store.go
		// "r": "Receipts", // overloaded with Logrec
		"x": "TxPositions",
		"X": "Txs",
		// topicsdb/index.go
		"t": "Topic",
		"r": "Logrec / Receipts", // overloaded with Receipts
	}
	lachesisStore := map[string]string{
		// abft/store.go
		"c": "LastDecidedState",
		"e": "EpochState",
	}
	epochLachesisStore := map[string]string{
		// abft/store.go
		"r": "Roots",
		"v": "VectorIndex",
		"C": "ConfirmedEvent",
	}

	switch {
	case dbName == "gossip":
		return gossipStore
	case strings.HasPrefix(dbName, "gossip-"):
		return epochGossipStore
	case dbName == "lachesis":
		return lachesisStore
	case strings.HasPrefix(dbName, "lachesis-"):
		return epochLachesisStore
	default:
		return map[string]string{}
	}
}

func printTableStats(dbName string, statsForPrefixes map[string]*prefixStats) {
	prefixes := make([]string, 0, len(statsForPrefixes))
	for prefix := range statsForPrefixes {
		prefixes = append(prefixes, prefix)
	}
	sort.Strings(prefixes)

	tableNames := getTableNames(dbName)

	fmt.Printf("DB: %s\n", dbName)
	formatS := "%-24s %-6s %12s %13s %13s %9s\n"
	formatD := "%-24s %-6s %12d %13d %13d %9d\n"
	fmt.Printf(formatS, "Table Name", "Prefix", "Count", "Key Bytes", "Value Bytes", "Total GiB")
	fmt.Printf(formatS, strings.Repeat("-", 24), strings.Repeat("-", 6), strings.Repeat("-", 12), strings.Repeat("-", 13), strings.Repeat("-", 13), strings.Repeat("-", 9))
	for _, prefix := range prefixes {
		stats := statsForPrefixes[prefix]
		tableName := tableNames[prefix]
		if tableName == "" {
			tableName = "?"
		}
		if !unicode.IsGraphic(rune(prefix[0])) {
			prefix = " "
		}
		fmt.Printf(formatD,
			tableName, prefix, stats.count, stats.keyLenSum, stats.valLenSum, (stats.keyLenSum+stats.valLenSum)/1024/1024/1024)
	}
	fmt.Println()
}
