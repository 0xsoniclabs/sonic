package execution_apis

import (
	"fmt"
	"io"
	"math/big"
	"os"

	rpctest "github.com/0xsoniclabs/sonic/api/rpc_test"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

// LoadChain reads a chain.rlp file and decodes it into a sequence of
// rpctest.Block structs suitable for the fake backend builder.
//
// chain.rlp contains a stream of RLP-encoded go-ethereum blocks (header + txs + uncles).
func LoadChain(path string) ([]rpctest.Block, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening chain.rlp: %w", err)
	}
	defer func() { _ = f.Close() }()

	stream := rlp.NewStream(f, 0)
	var blocks []rpctest.Block

	for {
		var block types.Block
		err := stream.Decode(&block)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("decoding block at index %d: %w", len(blocks), err)
		}

		blocks = append(blocks, convertBlock(&block))
	}

	return blocks, nil
}

// convertBlock converts a go-ethereum types.Block to a rpctest.Block.
func convertBlock(block *types.Block) rpctest.Block {
	header := block.Header()

	result := rpctest.Block{
		Number:     header.Number.Uint64(),
		Hash:       block.Hash(),
		ParentHash: header.ParentHash,
		BaseFee:    header.BaseFee,
	}

	// Set PrevRandao (MixDigest in the header, used as PREVRANDAO post-merge)
	if header.MixDigest != (common.Hash{}) {
		result.PrevRandao = header.MixDigest
	}

	// Convert transactions
	if len(block.Transactions()) > 0 {
		result.Transactions = make(map[common.Hash]*rpctest.Transaction, len(block.Transactions()))
		for i, tx := range block.Transactions() {
			result.Transactions[tx.Hash()] = rpctest.NewTransaction(
				tx,
				header.Number.Uint64(),
				uint64(i),
				nil, // no receipt — no execution replay
			)
		}
	}

	return result
}

// ChainBlocks builds a full block history (genesis + chain blocks) from a genesis
// spec and chain.rlp data, suitable for WithBlockHistory().
func ChainBlocks(genesisBlock rpctest.Block, chainBlocks []rpctest.Block) []rpctest.Block {
	// If we don't have a genesis hash, derive it from the first chain block's parent
	if genesisBlock.Hash == (common.Hash{}) && len(chainBlocks) > 0 {
		genesisBlock.Hash = chainBlocks[0].ParentHash
	}

	all := make([]rpctest.Block, 0, 1+len(chainBlocks))
	all = append(all, genesisBlock)
	all = append(all, chainBlocks...)
	return all
}

// ChainHead returns the chain head block number as a *big.Int.
// Returns 0 if the chain is empty.
func ChainHead(blocks []rpctest.Block) *big.Int {
	if len(blocks) == 0 {
		return big.NewInt(0)
	}
	return big.NewInt(int64(blocks[len(blocks)-1].Number))
}
