package gossip

import (
	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/sonic/inter/ibr"
	"github.com/0xsoniclabs/sonic/inter/ier"
	"github.com/ethereum/go-ethereum/core/types"
)

func (s *Store) GetFullBlockRecord(n consensus.BlockID) *ibr.LlrFullBlockRecord {
	block := s.GetBlock(n)
	if block == nil {
		return nil
	}
	txs := s.GetBlockTxs(n, block)
	receipts, _ := s.EvmStore().GetRawReceipts(n)
	if receipts == nil {
		receipts = []*types.ReceiptForStorage{}
	}
	return ibr.FullBlockRecordFor(block, txs, receipts)
}

func (s *Store) GetFullEpochRecord(epoch consensus.Epoch) *ier.LlrFullEpochRecord {
	// Use current state if current epoch is requested.
	if epoch == s.GetEpoch() {
		state := s.getBlockEpochState()
		return &ier.LlrFullEpochRecord{
			BlockState: *state.BlockState,
			EpochState: *state.EpochState,
		}
	}
	hbs, hes := s.GetHistoryBlockEpochState(epoch)
	if hbs == nil || hes == nil {
		return nil
	}
	return &ier.LlrFullEpochRecord{
		BlockState: *hbs,
		EpochState: *hes,
	}
}
