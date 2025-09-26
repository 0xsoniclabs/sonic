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
	"fmt"
	"math"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/inter/iblockproc"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	tracing "github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	uint256 "github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

//go:generate mockgen -source=evm_test.go -destination=evm_test_mock.go -package=evmmodule

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

	processed := processor.Execute([]*types.Transaction{inner}, math.MaxUint64)

	if len(processed) != 1 {
		t.Fatalf("Expected 1 processed transaction, got %d", len(processed))
	}
	if processed[0].Receipt == nil {
		t.Fatalf("Transaction was skipped")
	}
	if want, got := types.ReceiptStatusSuccessful, processed[0].Receipt.Status; want != got {
		t.Errorf("Expected status %v, got %v", want, got)
	}
}

func TestOperaEVMProcessor_Execute_ProducesContinuousTxIndexesInLogsAndReceipts(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	stateDb := state.NewMockStateDB(ctrl)
	logConsumer := NewMock_onNewLog(ctrl)

	any := gomock.Any()
	stateDb.EXPECT().BeginBlock(any).AnyTimes()
	stateDb.EXPECT().GetNonce(any).AnyTimes().Return(uint64(0))
	stateDb.EXPECT().GetCode(any).AnyTimes().Return(nil)
	stateDb.EXPECT().GetBalance(any).AnyTimes().Return(uint256.NewInt(1e18))
	stateDb.EXPECT().SubBalance(any, any, any).AnyTimes()
	stateDb.EXPECT().Prepare(any, any, any, any, any, any).AnyTimes()
	stateDb.EXPECT().SetNonce(any, any, any).AnyTimes()
	stateDb.EXPECT().Snapshot().AnyTimes().Return(1)
	stateDb.EXPECT().Exist(any).AnyTimes().Return(true)
	stateDb.EXPECT().AddBalance(any, any, any).AnyTimes()
	stateDb.EXPECT().GetRefund().AnyTimes().Return(uint64(0))
	stateDb.EXPECT().EndTransaction().AnyTimes()

	// track the Tx index set in the state db
	currentTxIndex := 0
	stateDb.EXPECT().SetTxContext(any, any).AnyTimes().Do(
		func(_ common.Hash, txIndex int) {
			currentTxIndex = txIndex
		},
	)
	stateDb.EXPECT().TxIndex().AnyTimes().DoAndReturn(
		func() int {
			return currentTxIndex
		},
	)
	stateDb.EXPECT().GetLogs(any, any).AnyTimes().DoAndReturn(
		func(_, _ common.Hash) []*types.Log {
			return []*types.Log{{
				TxIndex: uint(currentTxIndex),
			}}
		},
	)

	// Logs should be reported in consecutive order, one per transaction.
	const N = 5
	for i := range N * 3 {
		logConsumer.EXPECT().OnNewLog(LogWithTxIndex(uint(i)))
	}

	evmModule := New()
	processor := evmModule.Start(
		iblockproc.BlockCtx{}, stateDb, nil, logConsumer.OnNewLog,
		opera.Rules{}, &params.ChainConfig{}, common.Hash{},
	)

	key, err := crypto.GenerateKey()
	require.NoError(err)

	signer := types.LatestSignerForChainID(nil)
	tx := types.MustSignNewTx(key, signer, &types.LegacyTx{
		To: &common.Address{}, Nonce: 0, Gas: 21_0000,
	})

	// Make sure that the transaction index in the receipts is continuous
	// across multiple Execute calls, even when some calls have multiple
	// transactions and some have just one.
	txIndex := uint(0)
	for range N {
		processed := processor.Execute(types.Transactions{tx, tx}, math.MaxUint64)
		require.Len(processed, 2)
		require.NotNil(processed[0].Receipt)
		require.NotNil(processed[1].Receipt)
		require.Equal(txIndex, processed[0].Receipt.TransactionIndex)
		txIndex++
		require.Equal(txIndex, processed[1].Receipt.TransactionIndex)
		txIndex++

		processed = processor.Execute(types.Transactions{tx}, math.MaxUint64)
		require.Len(processed, 1)
		require.NotNil(processed[0].Receipt)
		require.Equal(txIndex, processed[0].Receipt.TransactionIndex)
		txIndex++
	}
}

func TestOperaEVMProcessor_Finalize_ReportsAggregatedNumberOfSkippedTransactions(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	stateDb := state.NewMockStateDB(ctrl)
	logConsumer := NewMock_onNewLog(ctrl)

	// Create a state DB mock that allows to run transactions but does not keep
	// any state. Thus, the same valid transaction can be executed multiple times,
	// but any transaction with a nonce > 0 will always be skipped.
	any := gomock.Any()
	stateDb.EXPECT().BeginBlock(any).AnyTimes()
	stateDb.EXPECT().GetNonce(any).AnyTimes().Return(uint64(0))
	stateDb.EXPECT().GetCode(any).AnyTimes().Return(nil)
	stateDb.EXPECT().GetBalance(any).AnyTimes().Return(uint256.NewInt(1e18))
	stateDb.EXPECT().SubBalance(any, any, any).AnyTimes()
	stateDb.EXPECT().Prepare(any, any, any, any, any, any).AnyTimes()
	stateDb.EXPECT().SetNonce(any, any, any).AnyTimes()
	stateDb.EXPECT().Snapshot().AnyTimes().Return(1)
	stateDb.EXPECT().Exist(any).AnyTimes().Return(true)
	stateDb.EXPECT().AddBalance(any, any, any).AnyTimes()
	stateDb.EXPECT().GetRefund().AnyTimes().Return(uint64(0))
	stateDb.EXPECT().EndTransaction().AnyTimes()
	stateDb.EXPECT().SetTxContext(any, any).AnyTimes()
	stateDb.EXPECT().TxIndex().AnyTimes()
	stateDb.EXPECT().GetLogs(any, any).AnyTimes()
	stateDb.EXPECT().EndBlock(any).AnyTimes()
	stateDb.EXPECT().GetStateHash().AnyTimes()

	evmModule := New()
	processor := evmModule.Start(
		iblockproc.BlockCtx{}, stateDb, nil, logConsumer.OnNewLog,
		opera.Rules{}, &params.ChainConfig{}, common.Hash{},
	)

	key, err := crypto.GenerateKey()
	require.NoError(err)

	signer := types.LatestSignerForChainID(nil)

	// A valid transaction with a nonce of 0; since the state DB is stateless,
	// this transaction can be executed multiple times and will not be skipped.
	validTx := types.MustSignNewTx(key, signer, &types.LegacyTx{
		To: &common.Address{}, Nonce: 0, Gas: 21_0000,
	})

	// A transaction that will be skipped because of a nonce gap.
	// This transaction has a nonce of 1, but the state DB always returns
	// a nonce of 0, so this transaction will always be skipped.
	skippedTx := types.MustSignNewTx(key, signer, &types.LegacyTx{
		To: &common.Address{}, Nonce: 1, Gas: 21_0000,
	})

	processed := processor.Execute(types.Transactions{validTx}, math.MaxUint64)
	require.Len(processed, 1)
	require.Equal(validTx, processed[0].Transaction)
	require.NotNil(processed[0].Receipt)

	_, numSkipped, _ := processor.Finalize()
	require.Equal(0, numSkipped)

	processed = processor.Execute(types.Transactions{skippedTx}, math.MaxUint64)
	require.Len(processed, 1)
	require.Equal(skippedTx, processed[0].Transaction)
	require.Nil(processed[0].Receipt)

	_, numSkipped, _ = processor.Finalize()
	require.Equal(1, numSkipped)

	processed = processor.Execute(types.Transactions{skippedTx, validTx, skippedTx}, math.MaxUint64)
	require.Len(processed, 3)
	require.Equal(skippedTx, processed[0].Transaction)
	require.Nil(processed[0].Receipt)
	require.Equal(validTx, processed[1].Transaction)
	require.NotNil(processed[1].Receipt)
	require.Equal(skippedTx, processed[2].Transaction)
	require.Nil(processed[2].Receipt)

	_, numSkipped, _ = processor.Finalize()
	require.Equal(3, numSkipped)
}

// onNewLog is a helper interface to allow mocking the onNewLog function
// passed to the EVM processor.
type _onNewLog interface {
	OnNewLog(*types.Log)
}

// Added to avoid unused warning of onNewLog interface which is only used for
// generating the mock.
var _ _onNewLog = (*Mock_onNewLog)(nil)

// LogWithTxIndex creates a gomock matcher that matches a log message with the
// given transaction index.
func LogWithTxIndex(id any) gomock.Matcher {
	if matcher, ok := id.(gomock.Matcher); ok {
		return logWithTxIndex{txIndex: matcher}
	}
	return LogWithTxIndex(gomock.Eq(id))
}

type logWithTxIndex struct {
	txIndex gomock.Matcher
}

func (i logWithTxIndex) Matches(arg any) bool {
	log, ok := arg.(*types.Log)
	if !ok || log == nil {
		return false
	}
	return i.txIndex.Matches(log.TxIndex)
}

func (i logWithTxIndex) String() string {
	return fmt.Sprintf("Log with TxIndex: %s", i.txIndex.String())
}
