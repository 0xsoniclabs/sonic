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
	"github.com/0xsoniclabs/sonic/rpc/oracle"
)

const (
	// maxFeeHistory is the most blocks eth_feeHistory answers; a larger count
	// is clamped, as in the reference.
	maxFeeHistory = 1024
	// maxRewardPercentiles bounds the percentiles of one request, as in
	// go-ethereum.
	maxRewardPercentiles = 100
)

var errInvalidPercentile = errors.New("invalid reward percentile")

// gasPrice suggests the price of a legacy transaction: the base fee of the
// head with a margin. Tips have no effect on Sonic and are not suggested.
func (a *api) gasPrice(ctx context.Context, _ dispatch.None) (any, error) {
	return (*hexutil.Big)(a.env.SuggestedGasPrice(a.env.Latest(ctx))), nil
}

func (a *api) maxPriorityFeePerGas(ctx context.Context, _ dispatch.None) (any, error) {
	tip, err := a.env.Oracle().SuggestTip(ctx, oracle.AsDefaultCertainty)
	if err != nil {
		return nil, err
	}
	return (*hexutil.Big)(tip), nil
}

func blobBaseFee(context.Context, dispatch.None) (any, error) {
	// Blobs are not supported: the excess blob gas is always zero, so the blob
	// base fee is the minimum blob gas price.
	return (*hexutil.Big)(big.NewInt(params.BlobTxMinBlobGasprice)), nil
}

// FeeHistoryResult is the answer of eth_feeHistory, as in go-ethereum: one
// entry per block in Reward and GasUsedRatio, and one more, for the block
// after the last, in BaseFee and BlobBaseFee.
type FeeHistoryResult struct {
	OldestBlock      *hexutil.Big     `json:"oldestBlock"`
	Reward           [][]*hexutil.Big `json:"reward,omitempty"`
	BaseFee          []*hexutil.Big   `json:"baseFeePerGas,omitempty"`
	GasUsedRatio     []float64        `json:"gasUsedRatio"`
	BlobBaseFee      []*hexutil.Big   `json:"baseFeePerBlobGas,omitempty"`
	BlobGasUsedRatio []float64        `json:"blobGasUsedRatio,omitempty"`
}

type feeHistoryArgs struct {
	BlockCount        math.HexOrDecimal64 `rpc:"blockCount"`
	LastBlock         chain.BlockNumber   `rpc:"lastBlock"`
	RewardPercentiles []float64           `rpc:"rewardPercentiles"`
}

// feeHistory answers a constant history: Sonic's base fee and tips do not vary
// by block in a way the history could show, so every block reports the
// current suggestion.
func (a *api) feeHistory(ctx context.Context, p feeHistoryArgs) (any, error) {
	if p.BlockCount == 0 {
		return &FeeHistoryResult{OldestBlock: (*hexutil.Big)(new(big.Int))}, nil
	}
	count := uint64(min(p.BlockCount, maxFeeHistory))
	if len(p.RewardPercentiles) > maxRewardPercentiles {
		return nil, fmt.Errorf("%w: over the query limit %d", errInvalidPercentile, maxRewardPercentiles)
	}
	for i, pc := range p.RewardPercentiles {
		if pc < 0 || pc > 100 {
			return nil, fmt.Errorf("%w: %f", errInvalidPercentile, pc)
		}
		if i > 0 && pc < p.RewardPercentiles[i-1] {
			return nil, fmt.Errorf("%w: #%d:%f > #%d:%f", errInvalidPercentile, i-1, p.RewardPercentiles[i-1], i, pc)
		}
	}

	head := a.env.Latest(ctx)
	last, err := a.env.Chain().ResolveNumber(head.Head, p.LastBlock)
	if err != nil {
		return nil, err
	}
	if latest := head.Number(); last > latest {
		return nil, fmt.Errorf("block %d not found; latest block is %d", last, latest)
	}
	oldest := uint64(0)
	if last >= count {
		oldest = last - (count - 1)
	}

	baseFee := a.env.SuggestedGasPrice(head)
	tips := make([]*hexutil.Big, 0, len(p.RewardPercentiles))
	for _, pc := range p.RewardPercentiles {
		tip, err := a.env.Oracle().SuggestTip(ctx, uint64(oracle.DecimalUnit*pc/100.0))
		if err != nil {
			return nil, err
		}
		tips = append(tips, (*hexutil.Big)(tip))
	}
	blobBaseFee := (*hexutil.Big)(big.NewInt(params.BlobTxMinBlobGasprice))

	blocks := last - oldest + 1
	res := &FeeHistoryResult{
		OldestBlock:      (*hexutil.Big)(new(big.Int).SetUint64(oldest)),
		Reward:           make([][]*hexutil.Big, 0, blocks),
		BaseFee:          make([]*hexutil.Big, 0, blocks+1),
		GasUsedRatio:     make([]float64, 0, blocks),
		BlobBaseFee:      make([]*hexutil.Big, 0, blocks+1),
		BlobGasUsedRatio: make([]float64, 0, blocks),
	}
	for range blocks {
		res.Reward = append(res.Reward, tips)
		res.BaseFee = append(res.BaseFee, (*hexutil.Big)(baseFee))
		res.GasUsedRatio = append(res.GasUsedRatio, 0.99)
		res.BlobBaseFee = append(res.BlobBaseFee, blobBaseFee)
		res.BlobGasUsedRatio = append(res.BlobGasUsedRatio, 0)
	}
	res.BaseFee = append(res.BaseFee, (*hexutil.Big)(baseFee))
	res.BlobBaseFee = append(res.BlobBaseFee, blobBaseFee)
	return res, nil
}
