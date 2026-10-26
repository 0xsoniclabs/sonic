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

package eth

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/params"

	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/oracle"
)

// gasPrice suggests the price of a legacy transaction: the base fee of the
// head with a margin. Tips have no effect on Sonic, so none is added.
func (handler *handlers) gasPrice(ctx context.Context, _ dispatch.Arguments) (any, error) {
	head, err := handler.environment.Latest(ctx)
	if err != nil {
		return nil, err
	}
	return (*hexutil.Big)(exec.SuggestedGasPrice(head)), nil
}

func (handler *handlers) maxPriorityFeePerGas(ctx context.Context, _ dispatch.Arguments) (any, error) {
	tip, err := handler.environment.Oracle().SuggestTip(ctx, oracle.DefaultCertainty)
	if err != nil {
		return nil, err
	}
	return (*hexutil.Big)(tip), nil
}

// blobBaseFee is constant: Sonic supports no blobs, so the excess blob gas
// of every block is zero.
func (handler *handlers) blobBaseFee(context.Context, dispatch.Arguments) (any, error) {
	return (*hexutil.Big)(big.NewInt(params.BlobTxMinBlobGasprice)), nil
}

// feeHistoryResult is the answer of eth_feeHistory, as go-ethereum's: one
// reward and ratio per block, and one more base fee for the block after the
// last.
type feeHistoryResult struct {
	OldestBlock      *hexutil.Big     `json:"oldestBlock"`
	Reward           [][]*hexutil.Big `json:"reward,omitempty"`
	BaseFee          []*hexutil.Big   `json:"baseFeePerGas,omitempty"`
	GasUsedRatio     []float64        `json:"gasUsedRatio"`
	BlobBaseFee      []*hexutil.Big   `json:"baseFeePerBlobGas,omitempty"`
	BlobGasUsedRatio []float64        `json:"blobGasUsedRatio,omitempty"`
}

const (
	// maxFeeHistoryBlocks is the clamp of the block count, as in the
	// reference, which clamps rather than refuses.
	maxFeeHistoryBlocks = 1024
	// maxRewardPercentiles bounds the percentiles of a request, as in
	// go-ethereum.
	maxRewardPercentiles = 100
)

var errInvalidPercentile = errors.New("invalid reward percentile")

// feeHistory reports a history whose every block has the same base fee, the
// suggested price of a new transaction, and the same rewards, the tips the
// oracle suggests for the percentiles: on Sonic the base fee of the next
// block, not the history, is what a transaction pays.
func (handler *handlers) feeHistory(ctx context.Context, arguments dispatch.Arguments) (any, error) {
	blockCount := uint64(dispatch.Argument[math.HexOrDecimal64](arguments, 0))
	lastBlock := dispatch.Argument[chain.BlockNumber](arguments, 1)
	percentiles := dispatch.Argument[[]float64](arguments, 2)
	if blockCount == 0 {
		return &feeHistoryResult{OldestBlock: (*hexutil.Big)(new(big.Int))}, nil
	}
	blockCount = min(blockCount, maxFeeHistoryBlocks)
	if err := checkPercentiles(percentiles); err != nil {
		return nil, err
	}
	head, err := handler.environment.Latest(ctx)
	if err != nil {
		return nil, err
	}
	last, err := handler.lastHistoryBlock(head, lastBlock)
	if err != nil {
		return nil, err
	}
	oldest := uint64(0)
	if last >= blockCount {
		oldest = last - (blockCount - 1)
	}
	tips, err := handler.rewards(ctx, percentiles)
	if err != nil {
		return nil, err
	}
	return newFeeHistory(oldest, last-oldest+1, exec.SuggestedGasPrice(head), tips), nil
}

func checkPercentiles(percentiles []float64) error {
	if len(percentiles) > maxRewardPercentiles {
		return fmt.Errorf("%w: over the query limit %d", errInvalidPercentile, maxRewardPercentiles)
	}
	for index, percentile := range percentiles {
		if percentile < 0 || percentile > 100 {
			return fmt.Errorf("%w: %f", errInvalidPercentile, percentile)
		}
		if index > 0 && percentile < percentiles[index-1] {
			return fmt.Errorf("%w: #%d:%f > #%d:%f", errInvalidPercentile, index-1, percentiles[index-1], index, percentile)
		}
	}
	return nil
}

// lastHistoryBlock resolves the last block of the history; a number above
// the head is refused with the head named.
func (handler *handlers) lastHistoryBlock(head exec.Head, lastBlock chain.BlockNumber) (uint64, error) {
	if !lastBlock.IsTag() {
		if uint64(lastBlock) > head.Number {
			return 0, fmt.Errorf("block %d not found; latest block is %d", uint64(lastBlock), head.Number)
		}
		return uint64(lastBlock), nil
	}
	return handler.environment.Resolve(head, chain.ReferenceToNumber(lastBlock))
}

func (handler *handlers) rewards(ctx context.Context, percentiles []float64) ([]*hexutil.Big, error) {
	tips := make([]*hexutil.Big, 0, len(percentiles))
	for _, percentile := range percentiles {
		tip, err := handler.environment.Oracle().SuggestTip(ctx, uint64(oracle.DecimalUnit*percentile/100.0))
		if err != nil {
			return nil, err
		}
		tips = append(tips, (*hexutil.Big)(tip))
	}
	return tips, nil
}

func newFeeHistory(oldest, blocks uint64, baseFee *big.Int, tips []*hexutil.Big) *feeHistoryResult {
	blobBaseFee := (*hexutil.Big)(big.NewInt(params.BlobTxMinBlobGasprice))
	result := &feeHistoryResult{
		OldestBlock:      (*hexutil.Big)(new(big.Int).SetUint64(oldest)),
		Reward:           make([][]*hexutil.Big, 0, blocks),
		BaseFee:          make([]*hexutil.Big, 0, blocks+1),
		GasUsedRatio:     make([]float64, 0, blocks),
		BlobBaseFee:      make([]*hexutil.Big, 0, blocks+1),
		BlobGasUsedRatio: make([]float64, 0, blocks),
	}
	for range blocks {
		result.Reward = append(result.Reward, tips)
		result.BaseFee = append(result.BaseFee, (*hexutil.Big)(baseFee))
		result.GasUsedRatio = append(result.GasUsedRatio, 0.99)
		result.BlobBaseFee = append(result.BlobBaseFee, blobBaseFee)
		result.BlobGasUsedRatio = append(result.BlobGasUsedRatio, 0)
	}
	result.BaseFee = append(result.BaseFee, (*hexutil.Big)(baseFee))
	result.BlobBaseFee = append(result.BlobBaseFee, blobBaseFee)
	return result
}
