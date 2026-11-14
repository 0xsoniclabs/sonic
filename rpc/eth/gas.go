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

	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/oracle"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/params"
)

// gasPrice suggests the price of a legacy transaction: the head's base fee
// with a margin. Tips have no effect on the network, so none is added.
func (h *handlers) gasPrice(context.Context, struct{}) (*hexutil.Big, error) {
	return (*hexutil.Big)(h.environment.SuggestedGasPrice(h.environment.Head())), nil
}

func (h *handlers) maxPriorityFeePerGas(context.Context, struct{}) (*hexutil.Big, error) {
	tip, err := h.environment.Oracle().SuggestTip(oracle.DefaultCertainty)
	return (*hexutil.Big)(tip), err
}

// blobBaseFee is the least blob gas price: no excess blob gas raises it.
func (h *handlers) blobBaseFee(context.Context, struct{}) (*hexutil.Big, error) {
	return (*hexutil.Big)(big.NewInt(blobTxMinBlobGasprice)), nil
}

type feeHistoryParams struct {
	BlockCount        math.HexOrDecimal64
	LastBlock         chain.BlockNumber
	RewardPercentiles []float64
}

// feeHistoryResult is the answer of eth_feeHistory, as go-ethereum's: the
// base fees hold one more entry, for the block after the last one.
type feeHistoryResult struct {
	OldestBlock      *hexutil.Big     `json:"oldestBlock"`
	Reward           [][]*hexutil.Big `json:"reward,omitempty"`
	BaseFee          []*hexutil.Big   `json:"baseFeePerGas,omitempty"`
	GasUsedRatio     []float64        `json:"gasUsedRatio"`
	BlobBaseFee      []*hexutil.Big   `json:"baseFeePerBlobGas,omitempty"`
	BlobGasUsedRatio []float64        `json:"blobGasUsedRatio,omitempty"`
}

// blobTxMinBlobGasprice is the blob base fee of every block: the network has
// no blobs.
const blobTxMinBlobGasprice = params.BlobTxMinBlobGasprice

const (
	// maxFeeHistoryBlocks is where a larger block count is clamped.
	maxFeeHistoryBlocks = 1024
	// maxRewardPercentiles bounds the percentiles of one request.
	maxRewardPercentiles = 100
)

var errInvalidPercentile = errors.New("invalid reward percentile")

// feeHistory answers, for each block of the range, the same fees: the
// suggested price as base fee, the oracle's tip at each percentile as reward
// and a full block, since the network prices by the head and not by the
// history.
func (h *handlers) feeHistory(ctx context.Context, params feeHistoryParams) (*feeHistoryResult, error) {
	if params.BlockCount == 0 {
		return &feeHistoryResult{OldestBlock: (*hexutil.Big)(new(big.Int))}, nil
	}
	count := min(uint64(params.BlockCount), maxFeeHistoryBlocks)
	if err := checkPercentiles(params.RewardPercentiles); err != nil {
		return nil, err
	}
	head := h.environment.Head()
	last, err := h.environment.Resolve(head, chain.ReferenceTo(params.LastBlock))
	if err != nil {
		return nil, err
	}
	if last > head.NumberU64() {
		return nil, fmt.Errorf("block %d not found; latest block is %d", last, head.NumberU64())
	}
	oldest := uint64(0)
	if last >= count {
		oldest = last - (count - 1)
	}
	baseFee := (*hexutil.Big)(h.environment.SuggestedGasPrice(head))
	tips := make([]*hexutil.Big, 0, len(params.RewardPercentiles))
	for _, percentile := range params.RewardPercentiles {
		tip, err := h.environment.Oracle().SuggestTip(uint64(oracle.DecimalUnit * percentile / 100.0))
		if err != nil {
			return nil, err
		}
		tips = append(tips, (*hexutil.Big)(tip))
	}
	blobBaseFee := (*hexutil.Big)(big.NewInt(blobTxMinBlobGasprice))
	blocks := last - oldest + 1
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
		result.BaseFee = append(result.BaseFee, baseFee)
		result.GasUsedRatio = append(result.GasUsedRatio, 0.99)
		result.BlobBaseFee = append(result.BlobBaseFee, blobBaseFee)
		result.BlobGasUsedRatio = append(result.BlobGasUsedRatio, 0)
	}
	result.BaseFee = append(result.BaseFee, baseFee)
	result.BlobBaseFee = append(result.BlobBaseFee, blobBaseFee)
	return result, nil
}

// checkPercentiles refuses more than the bound of percentiles, one out of
// [0, 100], and a decreasing one.
func checkPercentiles(percentiles []float64) error {
	if len(percentiles) > maxRewardPercentiles {
		return fmt.Errorf("%w: over the query limit %d", errInvalidPercentile, maxRewardPercentiles)
	}
	for i, percentile := range percentiles {
		if percentile < 0 || percentile > 100 {
			return fmt.Errorf("%w: %f", errInvalidPercentile, percentile)
		}
		if i > 0 && percentile < percentiles[i-1] {
			return fmt.Errorf("%w: #%d:%f > #%d:%f", errInvalidPercentile, i-1, percentiles[i-1], i, percentile)
		}
	}
	return nil
}
