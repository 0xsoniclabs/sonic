package ethapi

import (
	"context"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/rpc"
)

// BlockOverrides is a set of header fields to override.
type BlockOverrides struct {
	Number      *hexutil.Big
	Difficulty  *hexutil.Big
	Time        *hexutil.Uint64
	GasLimit    *hexutil.Uint64
	Coinbase    *common.Address
	Random      *common.Hash
	BaseFee     *hexutil.Big
	BlobBaseFee *hexutil.Big
}

// apply overrides the given header fields into the given block context.
func (diff *BlockOverrides) apply(blockCtx *vm.BlockContext) {
	if diff == nil {
		return
	}
	if diff.Number != nil {
		blockCtx.BlockNumber = diff.Number.ToInt()
	}
	if diff.Difficulty != nil {
		blockCtx.Difficulty = diff.Difficulty.ToInt()
	}
	if diff.Time != nil {
		blockCtx.Time = uint64(*diff.Time)
	}
	if diff.GasLimit != nil {
		blockCtx.GasLimit = uint64(*diff.GasLimit)
	}
	if diff.Coinbase != nil {
		blockCtx.Coinbase = *diff.Coinbase
	}
	if diff.Random != nil {
		blockCtx.Random = diff.Random
	}
	if diff.BaseFee != nil {
		blockCtx.BaseFee = diff.BaseFee.ToInt()
	}
	if diff.BlobBaseFee != nil {
		blockCtx.BlobBaseFee = diff.BlobBaseFee.ToInt()
	}
}

// getBlockContext returns a new vm.BlockContext based on the given header and backend
func getBlockContext(ctx context.Context, backend Backend, header *evmcore.EvmHeader) vm.BlockContext {
	chain := chainContext{
		ctx: ctx,
		b:   backend,
	}
	return evmcore.NewEVMBlockContext(header, &chain, nil)
}

// chainContextBackend provides methods required to implement ChainContext.
type chainContextBackend interface {
	HeaderByNumber(context.Context, rpc.BlockNumber) (*evmcore.EvmHeader, error)
}

// chainContext is an implementation of core.chainContext. It's main use-case
// is instantiating a vm.BlockContext without having access to the BlockChain object.
type chainContext struct {
	b   chainContextBackend
	ctx context.Context
}

func (context *chainContext) GetHeader(hash common.Hash, number uint64) *evmcore.EvmHeader {
	// This method is called to get the hash for a block number when executing the BLOCKHASH
	// opcode. Hence no need to search for non-canonical blocks.
	header, err := context.b.HeaderByNumber(context.ctx, rpc.BlockNumber(number))
	if header == nil || err != nil {
		return nil
	}
	if header.Hash != hash {
		return nil
	}
	return header
}
