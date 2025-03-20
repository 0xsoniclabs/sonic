package drivermodule_test

import (
	"github.com/0xsoniclabs/sonic/gossip/blockproc/drivermodule"
	"github.com/0xsoniclabs/sonic/inter/iblockproc"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
	"github.com/ethereum/go-ethereum/core/types"
	"go.uber.org/mock/gomock"
	"math/big"
	"testing"
)

const OrigOriginated = 10_000
const GasUsed = 40
const GasFeeCap = 100
const GasTip = 3
const BaseFee = 50
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
		TxHash:            tx.Hash(),
		GasUsed:           GasUsed,
		EffectiveGasPrice: big.NewInt(EffectiveGasPrice),
	}
	listener.OnNewReceipt(tx, receipt, idx.ValidatorID(1), big.NewInt(BaseFee))

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
		TxHash:            tx.Hash(),
		GasUsed:           GasUsed,
		EffectiveGasPrice: big.NewInt(EffectiveGasPrice),
	}
	listener.OnNewReceipt(tx, receipt, idx.ValidatorID(1), big.NewInt(BaseFee))

	originated := bs.ValidatorStates[es.Validators.GetIdx(1)].Originated.Uint64()
	if originated != OrigOriginated+GasUsed*EffectiveGasPrice {
		t.Errorf("Originated increment not GasUsed*EffectiveGasPrice: expected %d, actual %d",
			OrigOriginated+GasUsed*EffectiveGasPrice, originated)
	}
}
