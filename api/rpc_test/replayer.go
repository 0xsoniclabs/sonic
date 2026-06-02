package rpctest

import (
	"fmt"
	"math/big"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

// ReplayChain replays all blocks through the EVM, applying transactions to the
// given state database. This updates the state to reflect head-of-chain and
// generates receipts for all transactions.
//
// The statedb should already have genesis alloc loaded before calling this.
// The chainConfig should come from the genesis specification.
// The rawBlocks are the decoded types.Block from chain.rlp.
func ReplayChain(
	statedb state.StateDB,
	chainConfig *params.ChainConfig,
	rawBlocks []*types.Block,
) ([]Block, error) {

	// Build a header index for DummyChain (needed by BLOCKHASH opcode)
	chain := newHeaderChain(rawBlocks)

	result := make([]Block, len(rawBlocks))
	for i, block := range rawBlocks {
		blockNum := block.Number().Uint64()
		header := block.Header()
		evmHeader := evmcore.ConvertFromEthHeader(header)

		statedb.BeginBlock(blockNum)

		if len(block.Transactions()) > 0 {
			receipts, err := replayBlockTransactions(
				statedb, chainConfig, evmHeader, chain, block,
			)
			if err != nil {
				return nil, fmt.Errorf("replaying block %d (index %d): %w", blockNum, i, err)
			}

			result[i] = convertBlock(block, receipts)
		}

		// Wait for block finalization
		done := statedb.EndBlock(blockNum)
		if done != nil {
			if err := <-done; err != nil {
				return nil, fmt.Errorf("ending block %d: %w", blockNum, err)
			}
		}
	}

	return result, nil
}

// replayBlockTransactions applies all transactions in a block and returns their receipts.
func replayBlockTransactions(
	statedb state.StateDB,
	chainConfig *params.ChainConfig,
	evmHeader *evmcore.EvmHeader,
	chain evmcore.DummyChain,
	block *types.Block,
) (map[common.Hash]*types.Receipt, error) {

	var (
		usedGas     uint64
		blockNumber = block.Number()
		blockHash   = block.Hash()
		baseFee     = block.BaseFee()
		signer      = types.LatestSignerForChainID(chainConfig.ChainID)
		gasPool     = core.NewGasPool(block.GasLimit())
	)

	blockContext := evmcore.NewEVMBlockContextWithDifficulty(
		evmHeader, chain, nil, new(big.Int).Set(block.Difficulty()),
	)
	evm := vm.NewEVM(blockContext, statedb, chainConfig, vm.Config{})

	receipts := make(map[common.Hash]*types.Receipt, len(block.Transactions()))

	for i, tx := range block.Transactions() {
		statedb.SetTxContext(tx.Hash(), i)

		msg, err := core.TransactionToMessage(tx, signer, baseFee)
		if err != nil {
			return nil, fmt.Errorf("tx %d (%s): converting to message: %w", i, tx.Hash().Hex(), err)
		}

		// Clear blob hashes — Sonic's ApplyTransactionWithEVM rejects non-empty
		// BlobHashes but blob data is not relevant for state execution.
		msg.BlobHashes = nil

		receipt, err := evmcore.ApplyTransactionWithEVM(
			msg, chainConfig, gasPool, statedb, blockNumber, blockHash, tx, &usedGas, evm,
		)
		if err != nil {
			return nil, fmt.Errorf("tx %d (%s): applying: %w", i, tx.Hash().Hex(), err)
		}

		receipts[tx.Hash()] = receipt
	}

	return receipts, nil
}

// headerChain implements evmcore.DummyChain for header lookups during replay.
type headerChain struct {
	headers map[uint64]*evmcore.EvmHeader
	hashes  map[common.Hash]*evmcore.EvmHeader
}

func newHeaderChain(blocks []*types.Block) *headerChain {
	hc := &headerChain{
		headers: make(map[uint64]*evmcore.EvmHeader, len(blocks)),
		hashes:  make(map[common.Hash]*evmcore.EvmHeader, len(blocks)),
	}
	for _, block := range blocks {
		evmH := evmcore.ConvertFromEthHeader(block.Header())
		// ConvertFromEthHeader puts hash in Extra, fix it
		evmH.Hash = block.Hash()
		hc.headers[block.Number().Uint64()] = evmH
		hc.hashes[block.Hash()] = evmH
	}
	return hc
}

// Header implements evmcore.DummyChain.
func (hc *headerChain) Header(hash common.Hash, number uint64) *evmcore.EvmHeader {
	h, ok := hc.headers[number]
	if !ok {
		return nil
	}
	// If hash is specified and doesn't match, return nil
	if hash != (common.Hash{}) && h.Hash != hash {
		return nil
	}
	return h
}

// convertBlock converts a go-ethereum types.Block to a Block.
func convertBlock(block *types.Block, receipts map[common.Hash]*types.Receipt) Block {
	header := block.Header()

	result := Block{
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
		result.Transactions = make(map[common.Hash]*Transaction, len(block.Transactions()))
		for i, tx := range block.Transactions() {
			result.Transactions[tx.Hash()] = NewTransaction(
				tx,
				header.Number.Uint64(),
				uint64(i),
				receipts[tx.Hash()],
			)
		}
	}

	return result
}
