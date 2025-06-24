// Copyright 2025 Sonic Operations Ltd
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

package drivermodule_test

import (
	"github.com/0xsoniclabs/sonic/gossip/blockproc/drivermodule"
	"github.com/0xsoniclabs/sonic/inter/iblockproc"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"go.uber.org/mock/gomock"
	"math/big"
	"testing"
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
	module := drivermodule.NewDriverTxListenerModule()

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
		Validators: valsBuilder.Build(),
		Rules:      rules,
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
	module := drivermodule.NewDriverTxListenerModule()

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
		Validators: valsBuilder.Build(),
		Rules:      rules,
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
	module := drivermodule.NewDriverTxListenerModule()

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
		Validators: valsBuilder.Build(),
		Rules:      rules,
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
