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

package drivermodule

import (
	"fmt"
	"math"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies"
	"github.com/0xsoniclabs/sonic/inter/iblockproc"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const OrigOriginated = 10_000
const GasUsed = 40
const GasFeeCap = 100
const GasTip = 3
const BaseFee = 50
const BlobGasUsed = 2 * params.BlobTxBlobGasPerBlob
const BlobFeeCap = 6
const BlobBaseFee = 4
const EffectiveGasPrice = 53

func TestReceiptRewardWithoutFixEnabled(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	module := NewDriverTxListenerModule()

	blockCtx := iblockproc.BlockCtx{}
	bs := iblockproc.BlockState{
		ValidatorStates: []iblockproc.ValidatorBlockState{{
			Originated: big.NewInt(OrigOriginated),
		}},
	}
	valsBuilder := pos.NewBuilder()
	valsBuilder.Set(1, 100)

	rules := opera.MainNetRules()
	rules.Upgrades.Allegro = false // disable fix

	es := iblockproc.EpochState{
		EpochStateV1: iblockproc.EpochStateV1{
			Validators: valsBuilder.Build(),
			Rules:      rules,
		},
	}
	stateDb := state.NewMockStateDB(ctrl)
	listener := module.Start(blockCtx, bs, es, stateDb)

	tx := types.NewTx(&types.DynamicFeeTx{
		GasTipCap: big.NewInt(GasTip),
		GasFeeCap: big.NewInt(GasFeeCap),
	})
	receipt := &types.Receipt{
		TxHash:  tx.Hash(),
		GasUsed: GasUsed,
	}
	listener.OnNewReceipt(tx, receipt, idx.ValidatorID(1), big.NewInt(BaseFee), big.NewInt(BlobBaseFee))

	originated := bs.ValidatorStates[es.Validators.GetIdx(1)].Originated.Uint64()
	if originated != OrigOriginated+GasUsed*GasFeeCap {
		t.Errorf("Originated increment not GasUsed*GasFeeCap: expected %d, actual %d",
			OrigOriginated+GasUsed*GasFeeCap, originated)
	}
}

func TestReceiptRewardWithFixEnabled(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	module := NewDriverTxListenerModule()

	blockCtx := iblockproc.BlockCtx{}
	bs := iblockproc.BlockState{
		ValidatorStates: []iblockproc.ValidatorBlockState{{
			Originated: big.NewInt(OrigOriginated),
		}},
	}
	valsBuilder := pos.NewBuilder()
	valsBuilder.Set(1, 100)

	rules := opera.MainNetRules()
	rules.Upgrades.Allegro = true // enable fix

	es := iblockproc.EpochState{
		EpochStateV1: iblockproc.EpochStateV1{
			Validators: valsBuilder.Build(),
			Rules:      rules,
		},
	}
	stateDb := state.NewMockStateDB(ctrl)
	listener := module.Start(blockCtx, bs, es, stateDb)

	tx := types.NewTx(&types.DynamicFeeTx{
		GasTipCap: big.NewInt(GasTip),
		GasFeeCap: big.NewInt(GasFeeCap),
	})
	receipt := &types.Receipt{
		TxHash:  tx.Hash(),
		GasUsed: GasUsed,
	}
	listener.OnNewReceipt(tx, receipt, idx.ValidatorID(1), big.NewInt(BaseFee), big.NewInt(BlobBaseFee))

	originated := bs.ValidatorStates[es.Validators.GetIdx(1)].Originated.Uint64()
	if originated != OrigOriginated+GasUsed*EffectiveGasPrice {
		t.Errorf("Originated increment not GasUsed*EffectiveGasPrice: expected %d, actual %d",
			OrigOriginated+GasUsed*EffectiveGasPrice, originated)
	}
}

func TestReceiptRewardWithBlobsAndFixEnabled(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	module := NewDriverTxListenerModule()

	blockCtx := iblockproc.BlockCtx{}
	bs := iblockproc.BlockState{
		ValidatorStates: []iblockproc.ValidatorBlockState{{
			Originated: big.NewInt(OrigOriginated),
		}},
	}
	valsBuilder := pos.NewBuilder()
	valsBuilder.Set(1, 100)

	rules := opera.MainNetRules()
	rules.Upgrades.Allegro = true // enable fix

	es := iblockproc.EpochState{
		EpochStateV1: iblockproc.EpochStateV1{
			Validators: valsBuilder.Build(),
			Rules:      rules,
		},
	}
	stateDb := state.NewMockStateDB(ctrl)
	listener := module.Start(blockCtx, bs, es, stateDb)

	tx := types.NewTx(&types.BlobTx{
		GasTipCap:  uint256.NewInt(GasTip),
		GasFeeCap:  uint256.NewInt(GasFeeCap),
		BlobFeeCap: uint256.NewInt(BlobFeeCap),
		BlobHashes: make([]common.Hash, 2),
	})
	receipt := &types.Receipt{
		TxHash:      tx.Hash(),
		GasUsed:     GasUsed,
		BlobGasUsed: BlobGasUsed,
	}
	listener.OnNewReceipt(tx, receipt, idx.ValidatorID(1), big.NewInt(BaseFee), big.NewInt(BlobBaseFee))

	originated := bs.ValidatorStates[es.Validators.GetIdx(1)].Originated.Uint64()
	if originated != OrigOriginated+GasUsed*EffectiveGasPrice+BlobGasUsed*BlobBaseFee {
		t.Errorf("Originated increment not GasUsed*EffectiveGasPrice+BlobGasUsed*BlobBaseFee: expected %d, actual %d",
			OrigOriginated+GasUsed*EffectiveGasPrice+BlobGasUsed*BlobBaseFee, originated)
	}
}

func TestComputeEffectiveFee_ComputesFeesBySummingGasAndBlobFees(t *testing.T) {
	prices := []*big.Int{
		big.NewInt(0),
		big.NewInt(1),
		big.NewInt(10),
		big.NewInt(1e18),
		new(big.Int).Lsh(big.NewInt(1), 100),
	}

	used := []uint64{
		0, 1, 1000, 1e9, math.MaxUint64,
	}

	for _, gasPrice := range prices {
		for _, gasUsed := range used {
			for _, blobGasPrice := range prices {
				for _, blobGasUsed := range used {
					receipt := &types.Receipt{
						EffectiveGasPrice: gasPrice,
						GasUsed:           gasUsed,
						BlobGasUsed:       blobGasUsed,
						BlobGasPrice:      blobGasPrice,
					}

					want := new(big.Int).Add(
						new(big.Int).Mul(
							new(big.Int).SetUint64(gasUsed),
							gasPrice,
						),
						new(big.Int).Mul(
							new(big.Int).SetUint64(blobGasUsed),
							blobGasPrice,
						),
					)

					got, err := ComputeEffectiveFee(nil, receipt)
					require.NoError(t, err)
					require.True(t,
						want.Cmp(got) == 0,
						"want %v, got %v", want, got,
					)
				}
			}
		}
	}
}

func TestComputeEffectiveFee_UsesChargedAmountForSponsorshipPayments(t *testing.T) {
	prices := []*big.Int{
		big.NewInt(0),
		big.NewInt(1),
		big.NewInt(10),
		big.NewInt(1e18),
		new(big.Int).Lsh(big.NewInt(1), 100),
	}

	uses := []uint64{
		0, 1, 1000, 1e9, math.MaxUint64,
	}

	for _, price := range prices {
		for _, used := range uses {
			t.Run(fmt.Sprintf("price=%v/used=%d", price, used), func(t *testing.T) {
				ctrl := gomock.NewController(t)
				nonceSource := subsidies.NewMockNonceSource(ctrl)
				nonceSource.EXPECT().GetNonce(gomock.Any()).AnyTimes()

				tx, err := subsidies.GetFeeChargeTransaction(
					nonceSource,
					subsidies.FundId{},
					subsidies.GasConfig{},
					used,
					price,
				)
				require.NoError(t, err)

				want := new(big.Int).Mul(
					new(big.Int).SetUint64(used),
					price,
				)

				got, err := ComputeEffectiveFee(tx, nil)
				require.NoError(t, err)
				require.True(t,
					want.Cmp(got) == 0,
					"want %v, got %v", want, got,
				)
			})
		}
	}
}

func TestComputeEffectiveFee_MissingReceipt_ReportsError(t *testing.T) {
	_, err := ComputeEffectiveFee(nil, nil)
	require.ErrorContains(t, err, "missing receipt")
}

func TestComputeEffectiveFee_MissingEffectiveGasPrice_ReportsError(t *testing.T) {
	receipt := &types.Receipt{}
	_, err := ComputeEffectiveFee(nil, receipt)
	require.ErrorContains(t, err, "missing effective gas price")
}

func TestComputeEffectiveFee_MissingBlobGasPrice_ReportsIfBlobGasIsUsed(t *testing.T) {
	receipt := &types.Receipt{
		EffectiveGasPrice: big.NewInt(0),
	}

	// without blob gas usage no error is reported
	_, err := ComputeEffectiveFee(nil, receipt)
	require.NoError(t, err)

	// with blob gas usage, missing blob gas price should cause an error
	receipt.BlobGasUsed = 1
	_, err = ComputeEffectiveFee(nil, receipt)
	require.ErrorContains(t, err, "missing blob gas price")
}
