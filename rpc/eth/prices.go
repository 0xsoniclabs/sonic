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

	"github.com/Fantom-foundation/lachesis-base/utils/piecefunc"
	"github.com/ethereum/go-ethereum/common/hexutil"
	geth_math "github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/params"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/oracle"
)

func registerPrices(d *dispatch.Dispatcher, a *API, m func(string) string) {
	dispatch.Add0(d, m("gasPrice"), a.gasPrice)
	dispatch.Add0(d, m("maxPriorityFeePerGas"), a.maxPriorityFeePerGas)
	dispatch.Add3(d, m("feeHistory"), a.feeHistory, "blockCount", "lastBlock", "rewardPercentiles")
	dispatch.Add0(d, m("blobBaseFee"), a.blobBaseFee)
}

// gasPrice returns a suggestion for a gas price for legacy transactions. There
// are no tip suggestions since tips have no real effect on the Sonic network:
// the suggested gas price is a slightly increased base fee to provide a buffer
// for short-term price fluctuations.
func (a *API) gasPrice(context.Context) (*hexutil.Big, error) {
	return (*hexutil.Big)(a.env.MinGasPrice(a.chain.Head())), nil
}

// maxPriorityFeePerGas returns a suggestion for a gas tip cap for dynamic fee
// transactions.
func (a *API) maxPriorityFeePerGas(context.Context) (*hexutil.Big, error) {
	return (*hexutil.Big)(a.env.Oracle.SuggestTip(oracle.AsDefaultCertainty)), nil
}

// FeeHistoryResult is the result of eth_feeHistory, matching go-ethereum.
// Reward and GasUsedRatio hold one entry per returned block, while BaseFee
// and BlobBaseFee hold one extra trailing entry for the block following the
// last returned one.
type FeeHistoryResult struct {
	OldestBlock      *hexutil.Big     `json:"oldestBlock"`
	Reward           [][]*hexutil.Big `json:"reward,omitempty"`
	BaseFee          []*hexutil.Big   `json:"baseFeePerGas,omitempty"`
	GasUsedRatio     []float64        `json:"gasUsedRatio"`
	BlobBaseFee      []*hexutil.Big   `json:"baseFeePerBlobGas,omitempty"`
	BlobGasUsedRatio []float64        `json:"blobGasUsedRatio,omitempty"`
}

// maxRewardPercentiles limits the number of requested percentiles, as in go-ethereum.
const maxRewardPercentiles = 100

var errInvalidPercentile = errors.New("invalid reward percentile")

func (a *API) feeHistory(_ context.Context, blockCount geth_math.HexOrDecimal64, lastBlock dispatch.BlockNumber, rewardPercentiles []float64) (*FeeHistoryResult, error) {
	// validate input parameters
	if blockCount == 0 {
		return &FeeHistoryResult{
			OldestBlock:  (*hexutil.Big)(new(big.Int)),
			GasUsedRatio: nil,
		}, nil
	}
	if blockCount > 1024 {
		blockCount = 1024
	}
	if len(rewardPercentiles) > maxRewardPercentiles {
		return nil, fmt.Errorf("%w: over the query limit %d", errInvalidPercentile, maxRewardPercentiles)
	}
	for i, p := range rewardPercentiles {
		if p < 0 || p > 100 {
			return nil, fmt.Errorf("%w: %f", errInvalidPercentile, p)
		}
		if i > 0 && p < rewardPercentiles[i-1] {
			return nil, fmt.Errorf("%w: #%d:%f > #%d:%f", errInvalidPercentile, i-1, rewardPercentiles[i-1], i, p)
		}
	}

	head := a.chain.Head()
	last, err := a.chain.Number(head, lastBlock)
	if err != nil {
		return nil, err
	}
	if last > head.Number {
		return nil, fmt.Errorf("block %d not found; latest block is %d", last, head.Number)
	}

	oldest := last
	if oldest >= uint64(blockCount) {
		oldest -= uint64(blockCount - 1)
	} else {
		oldest = 0
	}

	baseFee := a.env.MinGasPrice(head)

	tips := make([]*hexutil.Big, 0, len(rewardPercentiles))
	for _, p := range rewardPercentiles {
		tip := a.env.Oracle.SuggestTip(uint64(piecefunc.DecimalUnit * p / 100.0))
		tips = append(tips, (*hexutil.Big)(tip))
	}

	// Blobs are not supported, so the blob base fee is constant and no blob
	// gas is ever used.
	blobBaseFee := (*hexutil.Big)(big.NewInt(params.BlobTxMinBlobGasprice))

	numBlocks := last - oldest + 1
	res := &FeeHistoryResult{
		OldestBlock:      (*hexutil.Big)(new(big.Int).SetUint64(oldest)),
		Reward:           make([][]*hexutil.Big, 0, numBlocks),
		BaseFee:          make([]*hexutil.Big, 0, numBlocks+1),
		GasUsedRatio:     make([]float64, 0, numBlocks),
		BlobBaseFee:      make([]*hexutil.Big, 0, numBlocks+1),
		BlobGasUsedRatio: make([]float64, 0, numBlocks),
	}
	for i := uint64(0); i < numBlocks; i++ {
		res.Reward = append(res.Reward, tips)
		res.BaseFee = append(res.BaseFee, (*hexutil.Big)(baseFee))
		res.GasUsedRatio = append(res.GasUsedRatio, 0.99)
		res.BlobBaseFee = append(res.BlobBaseFee, blobBaseFee)
		res.BlobGasUsedRatio = append(res.BlobGasUsedRatio, 0)
	}
	// As in go-ethereum, the base fee arrays carry one extra entry for the
	// block following the last one. The base fee is constant, so it is the
	// same as for the returned blocks.
	res.BaseFee = append(res.BaseFee, (*hexutil.Big)(baseFee))
	res.BlobBaseFee = append(res.BlobBaseFee, blobBaseFee)
	return res, nil
}

// blobBaseFee returns the blob base fee. As blobs are not supported, it equals
// the minimum blob gas price: the formula depends on the blob gas price and the
// excess blob gas, which is always 0.
func (a *API) blobBaseFee(context.Context) (*hexutil.Big, error) {
	return (*hexutil.Big)(big.NewInt(params.BlobTxMinBlobGasprice)), nil
}
