package cmd

import (
	"fmt"
	"testing"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/cockroachdb/pebble"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/require"
)

func TestDumpBlocks(t *testing.T) {
	require := require.New(t)
	path := "<insert_your_pebble_db_path_here>"

	// Open the database.
	db, err := pebble.Open(path, &pebble.Options{
		ReadOnly: true,
	})
	require.NoError(err)
	defer db.Close()

	// Iterate through the blocks and print their hashes.
	iter, err := db.NewIter(&pebble.IterOptions{
		LowerBound: []byte("b"),
		UpperBound: []byte("c"),
	})
	require.NoError(err)
	defer iter.Close()

	for iter.First(); iter.Valid(); iter.Next() {
		//key := iter.Key()
		value := iter.Value()

		var block inter.Block
		require.NoError(rlp.DecodeBytes(value, &block))

		fmt.Printf("%d, %s\n", block.Number, block.Hash().String())
	}
}
