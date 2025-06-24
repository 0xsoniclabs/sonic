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

package evmmodule

import (
	"math"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/inter/iblockproc"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	tracing "github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	uint256 "github.com/holiman/uint256"
	"go.uber.org/mock/gomock"
)

func TestEvm_IgnoresGasPriceOfInternalTransactions(t *testing.T) {
	ctrl := gomock.NewController(t)
	stateDb := state.NewMockStateDB(ctrl)

	zero := uint256.NewInt(0)
	zeroAddress := common.Address{}
	targetAddress := common.Address{0x01}
	any := gomock.Any()

	stateDb.EXPECT().BeginBlock(any)
	stateDb.EXPECT().SetTxContext(any, any)
	stateDb.EXPECT().GetBalance(zeroAddress).Return(zero)
	stateDb.EXPECT().SubBalance(zeroAddress, zero, tracing.BalanceDecreaseGasBuy)
	stateDb.EXPECT().Prepare(any, any, any, any, any, any).AnyTimes()
	stateDb.EXPECT().GetNonce(zeroAddress).Return(uint64(14))
	stateDb.EXPECT().SetNonce(zeroAddress, uint64(15), any)
	stateDb.EXPECT().Snapshot().Return(1)
	stateDb.EXPECT().Exist(targetAddress).Return(true)
	stateDb.EXPECT().SubBalance(zeroAddress, zero, tracing.BalanceChangeTransfer)
	stateDb.EXPECT().AddBalance(targetAddress, zero, tracing.BalanceChangeTransfer)
	stateDb.EXPECT().GetCode(targetAddress).MinTimes(1)
	stateDb.EXPECT().GetRefund().AnyTimes().Return(uint64(0))
	stateDb.EXPECT().AddBalance(zeroAddress, zero, tracing.BalanceIncreaseGasReturn)
	stateDb.EXPECT().GetLogs(any, any)
	stateDb.EXPECT().EndTransaction()
	stateDb.EXPECT().TxIndex()

	evmModule := New()
	processor := evmModule.Start(
		iblockproc.BlockCtx{},
		stateDb,
		nil,
		nil,
		opera.Rules{
			Economy: opera.EconomyRules{
				MinGasPrice: big.NewInt(12), // > than 0 offered by the internal transactions
			},
			Upgrades: opera.Upgrades{
				London: true,
			},
			Blocks: opera.BlocksRules{
				MaxBlockGas: 1e12,
			},
		},
		&params.ChainConfig{
			ChainID:     big.NewInt(1),
			LondonBlock: big.NewInt(0),
		},
		common.Hash{},
	)

	// This inner transaction has a gas price of 0, which is less than the MinGasPrice
	// on the chain. However, since it is an internal transaction, the lower gas price
	// boundary should be ignored.
	nonce := uint64(15)
	inner := types.NewTransaction(nonce, targetAddress, common.Big0, 1e10, common.Big0, nil)

	receipts := processor.Execute([]*types.Transaction{inner}, math.MaxUint64)

	if len(receipts) != 1 {
		t.Fatalf("Expected 1 receipt, got %d", len(receipts))
	}
	if receipts[0] == nil {
		t.Fatalf("Transaction was skipped")
	}
	if want, got := types.ReceiptStatusSuccessful, receipts[0].Status; want != got {
		t.Errorf("Expected status %v, got %v", want, got)
	}
}
