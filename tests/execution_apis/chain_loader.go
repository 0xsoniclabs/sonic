package execution_apis

import (
	"fmt"
	"io"
	"os"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

// LoadChain reads a chain.rlp file and decodes it into a sequence of
// rpctest.Block structs suitable for the fake backend builder.
//
// chain.rlp contains a stream of RLP-encoded go-ethereum blocks (header + txs + uncles).
func LoadChain(path string) ([]*types.Block, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening chain.rlp: %w", err)
	}
	defer func() { _ = f.Close() }()

	stream := rlp.NewStream(f, 0)
	var rawBlocks []*types.Block
	for {
		var block types.Block
		err := stream.Decode(&block)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("decoding block at index %d: %w", len(rawBlocks), err)
		}

		rawBlocks = append(rawBlocks, &block)
	}

	return rawBlocks, nil
}
