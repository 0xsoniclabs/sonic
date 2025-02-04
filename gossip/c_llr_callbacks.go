package gossip

import (
	"bytes"
	"fmt"
	"math/big"
	"time"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/0xsoniclabs/sonic/gossip/evmstore"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/iblockproc"
	"github.com/0xsoniclabs/sonic/inter/ibr"
	"github.com/0xsoniclabs/sonic/inter/ier"
	"github.com/0xsoniclabs/sonic/opera"
)

// defaultBlobGasPrice Sonic does not support blobs, so this price is constant
var defaultBlobGasPrice = big.NewInt(1) // TODO issue #147

func indexRawReceipts(s *Store, receiptsForStorage []*types.ReceiptForStorage, txs types.Transactions, blockIdx idx.Block, blockHash common.Hash, config *params.ChainConfig, time uint64, baseFee *big.Int, blobGasPrice *big.Int) (types.Receipts, error) {
	s.evm.SetRawReceipts(blockIdx, receiptsForStorage)

	receipts, err := evmstore.UnwrapStorageReceipts(receiptsForStorage, blockIdx, config, blockHash, time, baseFee, blobGasPrice, txs)
	if err != nil {
		return nil, err
	}

	for _, r := range receipts {
		s.evm.IndexLogs(r.Logs...)
	}
	return receipts, nil
}

func (s *Store) WriteFullBlockRecord(br ibr.LlrIdxFullBlockRecord) (err error) {
	for _, tx := range br.Txs {
		s.EvmStore().SetTx(tx.Hash(), tx)
	}

	var decodedReceipts types.Receipts
	if len(br.Receipts) != 0 {
		// Note: it's possible for receipts to get indexed twice by BR and block processing
		decodedReceipts, err = indexRawReceipts(s, br.Receipts, br.Txs, br.Idx, common.Hash(br.BlockHash),
			s.GetEvmChainConfig(), uint64(br.Time.Unix()), br.BaseFee, defaultBlobGasPrice)
		if err != nil {
			return err
		}
	}

	for i, tx := range br.Txs {
		s.EvmStore().SetTx(tx.Hash(), tx)
		s.EvmStore().SetTxPosition(tx.Hash(), evmstore.TxPosition{
			Block:       br.Idx,
			BlockOffset: uint32(i),
		})
	}

	builder := inter.NewBlockBuilder().
		WithNumber(uint64(br.Idx)).
		WithParentHash(common.Hash(br.ParentHash)).
		WithStateRoot(common.Hash(br.StateRoot)).
		WithTime(br.Time).
		WithDuration(time.Duration(br.Duration)).
		WithDifficulty(br.Difficulty).
		WithGasLimit(br.GasLimit).
		WithGasUsed(br.GasUsed).
		WithBaseFee(br.BaseFee).
		WithPrevRandao(common.Hash(br.PrevRandao)).
		WithEpoch(br.Epoch)

	for i := range br.Txs {
		builder.AddTransaction(br.Txs[i], decodedReceipts[i])
	}

	block := builder.Build()
	if !bytes.Equal(block.Hash().Bytes(), br.BlockHash.Bytes()) {
		return fmt.Errorf("block #%d hash mismatch; expected %s, got %s",
			br.Idx,
			br.BlockHash.String(),
			block.Hash().String())
	}

	s.SetBlock(br.Idx, block)
	s.SetBlockIndex(block.Hash(), br.Idx)
	return nil
}

func (s *Store) WriteFullEpochRecord(er ier.LlrIdxFullEpochRecord) {
	s.SetHistoryBlockEpochState(er.Idx, er.BlockState, er.EpochState)
	s.SetEpochBlock(er.BlockState.LastBlock.Idx+1, er.Idx)
}

func (s *Store) WriteUpgradeHeight(bs iblockproc.BlockState, es iblockproc.EpochState, prevEs *iblockproc.EpochState) {
	if prevEs == nil || es.Rules.Upgrades != prevEs.Rules.Upgrades {
		s.AddUpgradeHeight(opera.UpgradeHeight{
			Upgrades: es.Rules.Upgrades,
			Height:   bs.LastBlock.Idx + 1,
			Time:     bs.LastBlock.Time + 1,
		})
	}
}
