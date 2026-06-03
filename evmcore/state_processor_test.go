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

package evmcore

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/evmcore/core_types"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies/registry"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/opera/contracts/sfc"
	"github.com/0xsoniclabs/sonic/utils/signers/internaltx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// process_iteratively is an internal implementation of the StateProcessor's
// Process method using BeginBlock and an iterative transaction processing
// based on the TransactionProcessor. It is used to make sure that BeginBlock
// and the TransactionProcessor implementation behave the same way as the
// Process method.
func (p *StateProcessor) process_iteratively(
	block *EvmBlock, stateDb state.StateDB, cfg vm.Config, gasLimit uint64,
	usedGas *uint64, trueTxOffset int, onNewLog func(*core_types.Log), maxBlockSize uint64,
) ProcessSummary {
	// This implementation is a wrapper around the BeginBlock function, which
	// handles the actual transaction processing.
	txProcessor := p.BeginBlock(block, stateDb, cfg, gasLimit)
	summary := ProcessSummary{}
	for _, tx := range block.Transactions {
		cur := txProcessor.Run(trueTxOffset, tx)
		summary.ProcessedTransactions = append(summary.ProcessedTransactions, cur.ProcessedTransactions...)
		for _, processed := range cur.ProcessedTransactions {
			if processed.Receipt != nil {
				trueTxOffset++
			}
		}
	}

	// The used gas is the cumulative gas used reported by the last receipt.
	for _, tx := range summary.ProcessedTransactions {
		if tx.Receipt != nil {
			*usedGas = tx.Receipt.CumulativeGasUsed
			if onNewLog != nil {
				for _, log := range tx.Receipt.Logs {
					onNewLog(core_types.CoreLogFromGethLog(log))
				}
			}
		}
	}

	return summary
}

func TestProcess_ReportsReceiptsOfProcessedTransactions(t *testing.T) {
	ctrl := gomock.NewController(t)

	blockGasLimit := 2*21_000 + 10_000
	transactions := []*types.Transaction{
		types.NewTx(&types.LegacyTx{Nonce: 0, To: &common.Address{}, Gas: 21_000}), // passes
		types.NewTx(&types.LegacyTx{Nonce: 3, To: &common.Address{}, Gas: 21_000}), // skipped due to nonce
		types.NewTx(&types.LegacyTx{Nonce: 0, To: &common.Address{}, Gas: 21_000}), // passes (mock does not track nonces)
		types.NewTx(&types.LegacyTx{Nonce: 0, To: &common.Address{}, Gas: 21_000}), // skipped due to block gas limit
	}

	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	signer := types.FrontierSigner{}
	for i := range transactions {
		transactions[i], err = types.SignTx(transactions[i], signer, key)
		require.NoError(t, err)
	}

	state := getStateDbMockForTransactions(ctrl, transactions)

	chainConfig := params.ChainConfig{}
	chain := NewMockDummyChain(ctrl)
	processor := NewStateProcessorForHeadState(
		&chainConfig,
		chain,
		opera.Upgrades{},
		nil,
	)

	tests := map[string]processFunction{
		"bulk":        processor.Process,
		"incremental": processor.process_iteratively,
	}

	for name, process := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			block := &EvmBlock{
				EvmHeader: EvmHeader{
					Number:   big.NewInt(1),
					GasLimit: uint64(blockGasLimit),
				},
				Transactions: transactions,
			}

			reportedLogs := []*core_types.Log{}
			onLog := func(log *core_types.Log) {
				reportedLogs = append(reportedLogs, log)
			}

			vmConfig := vm.Config{}
			gasLimit := uint64(blockGasLimit)
			usedGas := new(uint64)

			summary := process(block, state, vmConfig, gasLimit, usedGas, 0, onLog, math.MaxUint64)
			processed := summary.ProcessedTransactions

			// Receipts should be set accordingly.
			require.Len(processed, len(transactions))
			require.Equal(transactions[0], processed[0].Transaction)
			require.Equal(transactions[1], processed[1].Transaction)
			require.Equal(transactions[2], processed[2].Transaction)
			require.Equal(transactions[3], processed[3].Transaction)

			logMsg0 := &types.Log{Address: common.Address{0}, TxIndex: 0}
			logMsg1 := &types.Log{Address: common.Address{1}, TxIndex: 1}

			require.NotNil(processed[0].Receipt)
			require.Equal(&types.Receipt{
				Status:            types.ReceiptStatusSuccessful,
				GasUsed:           21_000,
				CumulativeGasUsed: 21_000,
				BlockNumber:       block.Number,
				TransactionIndex:  0,
				TxHash:            transactions[0].Hash(),
				Bloom: types.CreateBloom(&types.Receipt{
					Logs: []*types.Log{logMsg0},
				}),
				Logs:              []*types.Log{logMsg0},
				EffectiveGasPrice: big.NewInt(0),
			}, processed[0].Receipt)

			require.Nil(processed[1].Receipt)

			require.NotNil(processed[2].Receipt)
			require.Equal(&types.Receipt{
				Status:            types.ReceiptStatusSuccessful,
				GasUsed:           21_000,
				CumulativeGasUsed: 42_000,
				BlockNumber:       block.Number,
				TransactionIndex:  1,
				TxHash:            transactions[2].Hash(),
				Bloom: types.CreateBloom(&types.Receipt{
					Logs: []*types.Log{logMsg1},
				}),
				Logs:              []*types.Log{logMsg1},
				EffectiveGasPrice: big.NewInt(0),
			}, processed[2].Receipt)

			require.Nil(processed[3].Receipt)

			coreLogMsg0 := core_types.CoreLogFromGethLog(logMsg0)
			coreLogMsg1 := core_types.CoreLogFromGethLog(logMsg1)
			require.Equal([]*core_types.Log{coreLogMsg0, coreLogMsg1}, reportedLogs)

			require.Equal(uint64(21_000+21_000), *usedGas)
		})
	}
}

func TestProcess_DetectsTransactionThatCanNotBeConvertedIntoAMessage(t *testing.T) {
	ctrl := gomock.NewController(t)

	chainConfig := params.ChainConfig{}
	chain := NewMockDummyChain(ctrl)

	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	signer := types.FrontierSigner{}

	logMsg0 := &types.Log{Address: common.Address{0}, TxIndex: 0}

	// The conversion into a evmcore Message depends on the ability to check
	// the signature and to derive the sender address. To stimulate a failure
	// in the conversion, a invalid signature is used.
	transactions := []*types.Transaction{
		types.NewTx(&types.LegacyTx{
			Nonce: 1, To: &common.Address{}, Gas: 21_000,
			R: big.NewInt(1), S: big.NewInt(2), V: big.NewInt(3),
		}),
		// Make sure that a transaction succeeding the failing one is processed
		// correctly.
		types.MustSignNewTx(key, signer, &types.LegacyTx{
			Nonce: 0, To: &common.Address{}, Gas: 21_000,
		}),
	}

	state := getStateDbMockForTransactions(ctrl, transactions)
	processor := NewStateProcessorForHeadState(&chainConfig, chain, opera.Upgrades{}, nil)
	tests := map[string]processFunction{
		"bulk":        processor.Process,
		"incremental": processor.process_iteratively,
	}

	for name, process := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			block := &EvmBlock{
				EvmHeader: EvmHeader{
					Number:   big.NewInt(1),
					GasLimit: 30_000,
				},
				Transactions: transactions,
			}

			vmConfig := vm.Config{}
			gasLimit := uint64(math.MaxUint64)
			usedGas := new(uint64)

			summary := process(block, state, vmConfig, gasLimit, usedGas, 0, nil, math.MaxUint64)
			processed := summary.ProcessedTransactions

			require.Len(processed, len(transactions))
			require.Equal(transactions[0], processed[0].Transaction)
			require.Equal(transactions[1], processed[1].Transaction)

			require.Nil(processed[0].Receipt)
			require.Equal(&types.Receipt{
				Status:            types.ReceiptStatusSuccessful,
				GasUsed:           21_000,
				CumulativeGasUsed: 21_000,
				BlockNumber:       block.Number,
				TransactionIndex:  0,
				TxHash:            transactions[1].Hash(),
				Bloom: types.CreateBloom(&types.Receipt{
					Logs: []*types.Log{logMsg0},
				}),
				Logs:              []*types.Log{logMsg0},
				EffectiveGasPrice: big.NewInt(0),
			}, processed[1].Receipt)
		})
	}
}

func TestProcess_TracksParentBlockHashIfPragueIsEnabled(t *testing.T) {
	for _, isPrague := range []bool{false, true} {
		ctrl := gomock.NewController(t)

		chainConfig := params.ChainConfig{}

		if isPrague {
			chainConfig = params.ChainConfig{
				ChainID:     big.NewInt(12),
				LondonBlock: new(big.Int).SetUint64(0),
				PragueTime:  new(uint64),
			}
		}
		chain := NewMockDummyChain(ctrl)

		processor := NewStateProcessorForHeadState(&chainConfig, chain, opera.Upgrades{}, nil)

		tests := map[string]processFunction{
			"bulk":        processor.Process,
			"incremental": processor.process_iteratively,
		}

		for name, process := range tests {
			t.Run(name, func(t *testing.T) {
				state := state.NewMockStateDB(ctrl)

				if isPrague {
					any := gomock.Any()
					gomock.InOrder(
						state.EXPECT().AddAddressToAccessList(params.HistoryStorageAddress),
						state.EXPECT().Snapshot().Return(0),
						state.EXPECT().Exist(params.HistoryStorageAddress).Return(true),
						state.EXPECT().GetCode(any).AnyTimes(),
						state.EXPECT().Finalise(any),
						state.EXPECT().EndTransaction(), // must be terminated
					)
				}

				require := require.New(t)
				block := &EvmBlock{
					EvmHeader: EvmHeader{
						Number:   big.NewInt(1),
						GasLimit: 30_000,
					},
				}
				require.Equal(isPrague, chainConfig.IsPrague(block.Number, uint64(block.Time)))

				vmConfig := vm.Config{}
				gasLimit := uint64(math.MaxUint64)
				usedGas := new(uint64)
				processed := process(block, state, vmConfig, gasLimit, usedGas, 0, nil, math.MaxUint64).ProcessedTransactions
				require.Empty(processed)
			})
		}
	}
}

func TestProcess_FailingTransactionAreSkippedButTheBlockIsNotTerminated(t *testing.T) {
	ctrl := gomock.NewController(t)
	state := state.NewMockStateDB(ctrl)

	chainConfig := params.ChainConfig{}
	chain := NewMockDummyChain(ctrl)
	processor := NewStateProcessorForHeadState(&chainConfig, chain, opera.Upgrades{}, nil)

	block := &EvmBlock{
		EvmHeader: EvmHeader{
			Number:   big.NewInt(1),
			GasLimit: 100_000,
		},
		Transactions: []*types.Transaction{
			// This transaction will fail due to an invalid signature.
			types.NewTx(&types.LegacyTx{
				Nonce:    0,
				To:       &common.Address{},
				Gas:      21_000,
				GasPrice: big.NewInt(1),
				V:        big.NewInt(1), // invalid signature
			}),
			// Valid transaction that will succeed.
			types.NewTx(&types.LegacyTx{
				Nonce:    0,
				To:       &common.Address{},
				Gas:      21_000,
				GasPrice: big.NewInt(1),
			}),
		},
	}

	// Mock the state database interactions for passing transaction.
	any := gomock.Any()
	state.EXPECT().SetTxContext(any, any).Times(1)
	state.EXPECT().GetBalance(any).Return(uint256.NewInt(1000000)).Times(1)
	state.EXPECT().SubBalance(any, any, any).Times(2)
	state.EXPECT().Prepare(any, any, any, any, any, any).Times(1)
	state.EXPECT().GetNonce(any).Return(uint64(0)).Times(1)
	state.EXPECT().SetNonce(any, any, any).Times(1)
	state.EXPECT().GetCode(any).Return(nil).Times(2)
	state.EXPECT().Snapshot().Return(0).Times(1)
	state.EXPECT().Exist(any).Return(true).Times(1)
	state.EXPECT().AddBalance(any, any, any).Times(3)
	state.EXPECT().GetRefund().Return(uint64(0)).Times(2)
	state.EXPECT().GetLogs(any, any).Return([]*types.Log{})
	state.EXPECT().EndTransaction().Times(1)
	state.EXPECT().TxIndex().Return(0).Times(1)

	// Process the block
	gasLimit := uint64(math.MaxUint64)
	usedGas := new(uint64)
	processed := processor.Process(block, state, vm.Config{}, gasLimit, usedGas, 0, nil, math.MaxUint64).ProcessedTransactions

	require.Len(t, processed, 2)
	require.Equal(t, processed[0].Transaction, block.Transactions[0])
	require.Nil(t, processed[0].Receipt)
	require.Equal(t, processed[1].Transaction, block.Transactions[1])
	require.NotNil(t, processed[1].Receipt)
}

func TestProcess_EnforcesGasLimitBySkippingExcessiveTransactions(t *testing.T) {
	ctrl := gomock.NewController(t)
	chainConfig := params.ChainConfig{}
	chain := NewMockDummyChain(ctrl)
	processor := NewStateProcessorForHeadState(&chainConfig, chain, opera.Upgrades{}, nil)

	tests := map[string]processFunction{
		"bulk":        processor.Process,
		"incremental": processor.process_iteratively,
	}

	zero := common.Address{}
	transactions := []*types.Transaction{
		types.NewTx(&types.LegacyTx{Nonce: 1, To: &zero, Gas: 21_000}),
		types.NewTx(&types.LegacyTx{Nonce: 2, To: &zero, Gas: 21_000}),
		types.NewTx(&types.LegacyTx{Nonce: 3, To: &zero, Gas: 21_000}),
	}
	state := getStateDbMockForTransactions(ctrl, transactions)

	for name, process := range tests {
		t.Run(name, func(t *testing.T) {
			block := &EvmBlock{
				EvmHeader: EvmHeader{
					Number:   big.NewInt(1),
					GasLimit: math.MaxUint64,
				},
				Transactions: transactions,
			}

			vmConfig := vm.Config{}
			usedGas := new(uint64)

			tests := map[string]struct {
				gasLimit uint64
				passing  int
			}{
				"no gas": {
					gasLimit: 0,
					passing:  0,
				},
				"not enough for one": {
					gasLimit: 21_000 - 1,
					passing:  0,
				},
				"enough for one": {
					gasLimit: 21_000,
					passing:  1,
				},
				"not enough for two": {
					gasLimit: 2*21_000 - 1,
					passing:  1,
				},
				"enough for two": {
					gasLimit: 2 * 21_000,
					passing:  2,
				},
				"enough for three": {
					gasLimit: 3 * 21_000,
					passing:  3,
				},
				"more than enough": {
					gasLimit: math.MaxUint64,
					passing:  3,
				},
			}

			for name, test := range tests {
				t.Run(name, func(t *testing.T) {
					require := require.New(t)
					gasLimit := test.gasLimit
					processed := process(block, state, vmConfig, gasLimit, usedGas, 0, nil, math.MaxUint64).ProcessedTransactions
					require.Len(processed, 3)

					for i, tx := range transactions {
						require.Equal(tx, processed[i].Transaction)
					}
					for i := range test.passing {
						require.NotNil(processed[i].Receipt)
					}
					for i := test.passing; i < 3; i++ {
						require.Nil(processed[i].Receipt)
					}
				})
			}

		})
	}
}

func TestProcess_UsesDifficultyOfOne(t *testing.T) {
	ctrl := gomock.NewController(t)
	processor := NewStateProcessorForHeadState(&params.ChainConfig{}, nil, opera.Upgrades{}, nil)

	state, block := createScenarioWithTxCheckingDifficulty(ctrl, big.NewInt(1))

	// Check that the difficulty of 1 is used.
	results := processor.Process(block, state, vm.Config{}, math.MaxUint64, new(uint64), 0, nil, math.MaxUint64).ProcessedTransactions
	require.Len(t, results, 1)
	require.NotNil(t, results[0].Receipt)
	require.Equal(t, types.ReceiptStatusSuccessful, results[0].Receipt.Status)

	// Check that an unexpected difficulty causes a revert.
	wrongDifficulty := big.NewInt(2)
	state, block = createScenarioWithTxCheckingDifficulty(ctrl, wrongDifficulty)
	results = processor.Process(block, state, vm.Config{}, math.MaxUint64, new(uint64), 0, nil, math.MaxUint64).ProcessedTransactions
	require.Len(t, results, 1)
	require.NotNil(t, results[0].Receipt)
	require.Equal(t, types.ReceiptStatusFailed, results[0].Receipt.Status)

}

func TestProcessWithDifficulty_UsesProvidedDifficulty(t *testing.T) {
	for _, difficulty := range []*big.Int{big.NewInt(0), big.NewInt(2), big.NewInt(42)} {
		t.Run(fmt.Sprintf("difficulty=%s", difficulty.String()), func(t *testing.T) {
			ctrl := gomock.NewController(t)
			processor := NewStateProcessorForHeadState(&params.ChainConfig{}, nil, opera.Upgrades{}, nil)

			state, block := createScenarioWithTxCheckingDifficulty(ctrl, difficulty)
			results := processor.ProcessWithDifficulty(
				block, state, vm.Config{}, math.MaxUint64,
				new(uint64), 0, nil, difficulty, math.MaxUint64,
			).ProcessedTransactions
			require.Len(t, results, 1)
			require.NotNil(t, results[0].Receipt)
			require.Equal(t, types.ReceiptStatusSuccessful, results[0].Receipt.Status)
		})
	}
}

func TestProcessWithDifficulty_onNewLog_CollectsLogsAccordingToLogsProduced(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	key2, err := crypto.GenerateKey()
	require.NoError(t, err)
	signer := types.FrontierSigner{}

	invalidTx := types.NewTx(&types.LegacyTx{
		Nonce: 0, To: &common.Address{}, Gas: 21_000,
		R: big.NewInt(1), S: big.NewInt(2), V: big.NewInt(3),
	})
	tx1 := types.MustSignNewTx(key, signer, &types.LegacyTx{
		Nonce: 0, To: &common.Address{}, Gas: 21_000,
	})
	tx2 := types.MustSignNewTx(key2, signer, &types.LegacyTx{
		Nonce: 0, To: &common.Address{}, Gas: 21_000,
	})

	log1 := &types.Log{Address: common.Address{1}, Topics: []common.Hash{{1}}, Data: []byte{1}}
	log2 := &types.Log{Address: common.Address{2}, Topics: []common.Hash{{2}}, Data: []byte{2}}
	log3 := &types.Log{Address: common.Address{3}, Topics: []common.Hash{{3}}, Data: []byte{3}}

	tests := map[string]struct {
		transactions      []*types.Transaction
		logsByTxIndex     map[common.Hash][]*types.Log
		expectedCallbacks []*core_types.Log
	}{
		"transaction without receipt does not emit": {
			transactions: []*types.Transaction{
				invalidTx,
			},
			logsByTxIndex: map[common.Hash][]*types.Log{
				// invalid tx shall not have logs, but if it would, they should not be emitted.
				invalidTx.Hash(): {
					log1,
				},
			},
			expectedCallbacks: nil,
		},
		"transaction without logs does not emit": {
			transactions: []*types.Transaction{
				tx1,
			},
			logsByTxIndex: map[common.Hash][]*types.Log{
				tx1.Hash(): {},
			},
			expectedCallbacks: nil,
		},
		"transaction with one log emits one callback": {
			transactions: []*types.Transaction{
				tx1,
			},
			logsByTxIndex: map[common.Hash][]*types.Log{
				tx1.Hash(): {
					log1,
				},
			},
			expectedCallbacks: []*core_types.Log{
				core_types.CoreLogFromGethLog(log1),
			},
		},
		"transaction with multiple logs emits all callbacks": {
			transactions: []*types.Transaction{
				tx1,
			},
			logsByTxIndex: map[common.Hash][]*types.Log{
				tx1.Hash(): {
					log1,
					log2,
					log3,
				},
			},
			expectedCallbacks: []*core_types.Log{
				core_types.CoreLogFromGethLog(log1),
				core_types.CoreLogFromGethLog(log2),
				core_types.CoreLogFromGethLog(log3),
			},
		},
		"multiple transactions with logs emit all callbacks": {
			transactions: []*types.Transaction{
				tx1,
				tx2,
			},
			logsByTxIndex: map[common.Hash][]*types.Log{
				tx1.Hash(): {
					log1,
					log2,
				},
				tx2.Hash(): {
					log3,
				},
			},
			expectedCallbacks: []*core_types.Log{
				core_types.CoreLogFromGethLog(log1),
				core_types.CoreLogFromGethLog(log2),
				core_types.CoreLogFromGethLog(log3),
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			stateDb := state.NewMockStateDB(ctrl)
			stateDb.EXPECT().SetTxContext(gomock.Any(), gomock.Any()).AnyTimes()
			stateDb.EXPECT().TxIndex().Return(0).AnyTimes()

			for _, tx := range test.transactions {
				stateDb.EXPECT().GetLogs(tx.Hash(), gomock.Any()).DoAndReturn(
					func(txHash, _ common.Hash) []*types.Log {
						logs := test.logsByTxIndex[txHash]
						copied := make([]*types.Log, len(logs))
						copy(copied, logs)
						return copied
					},
				).AnyTimes()
			}

			mockStateDbTransactionExecution(stateDb)

			processor := NewStateProcessorForHeadState(&params.ChainConfig{}, nil, opera.Upgrades{}, nil)
			block := &EvmBlock{
				EvmHeader: EvmHeader{
					Number:   big.NewInt(1),
					GasLimit: math.MaxUint64,
				},
				Transactions: test.transactions,
			}

			var emitted []*core_types.Log
			summary := processor.ProcessWithDifficulty(
				block,
				stateDb,
				vm.Config{},
				math.MaxUint64,
				new(uint64),
				0,
				func(log *core_types.Log) {
					emitted = append(emitted, log)
				},
				big.NewInt(1),
				math.MaxUint64,
			)

			require.Len(t, summary.ProcessedTransactions, len(test.transactions))
			require.Equal(t, test.expectedCallbacks, emitted)
		})
	}
}

func TestProcessWithDifficulty_onNewLog_ReportsLogsInOrder(t *testing.T) {
	ctrl := gomock.NewController(t)

	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	signer := types.FrontierSigner{}

	transactions := []*types.Transaction{
		types.MustSignNewTx(key, signer, &types.LegacyTx{Nonce: 0, To: &common.Address{1}, Gas: 21_000}),
		// Invalid signature -> skipped transaction -> no receipt/log callbacks.
		types.NewTx(&types.LegacyTx{
			Nonce: 0, To: &common.Address{}, Gas: 21_000,
		}),
		types.MustSignNewTx(key, signer, &types.LegacyTx{Nonce: 0, To: &common.Address{2}, Gas: 21_000}),
	}

	logsByTxIndex := map[common.Hash][]*types.Log{
		transactions[0].Hash(): {
			{Address: common.Address{10}},
		},
		transactions[1].Hash(): {},
		transactions[2].Hash(): {
			{Address: common.Address{20}},
			{Address: common.Address{30}},
		},
	}

	stateDb := state.NewMockStateDB(ctrl)
	stateDb.EXPECT().SetTxContext(gomock.Any(), gomock.Any()).AnyTimes()
	stateDb.EXPECT().TxIndex().Return(0).AnyTimes()

	getLogs := func(txHash, _ common.Hash) []*types.Log {
		logs := logsByTxIndex[txHash]
		copied := make([]*types.Log, len(logs))
		copy(copied, logs)
		return copied
	}
	stateDb.EXPECT().GetLogs(transactions[0].Hash(), gomock.Any()).DoAndReturn(getLogs)
	stateDb.EXPECT().GetLogs(transactions[1].Hash(), gomock.Any()).DoAndReturn(getLogs)
	stateDb.EXPECT().GetLogs(transactions[2].Hash(), gomock.Any()).DoAndReturn(getLogs)

	mockStateDbTransactionExecution(stateDb)

	processor := NewStateProcessorForHeadState(&params.ChainConfig{}, nil, opera.Upgrades{}, nil)
	block := &EvmBlock{
		EvmHeader: EvmHeader{
			Number:   big.NewInt(1),
			GasLimit: math.MaxUint64,
		},
		Transactions: transactions,
	}

	var emitted []core_types.Log
	summary := processor.ProcessWithDifficulty(
		block,
		stateDb,
		vm.Config{},
		math.MaxUint64,
		new(uint64),
		0,
		func(log *core_types.Log) {
			emitted = append(emitted, *log)
		},
		big.NewInt(1),
		math.MaxUint64,
	)

	require.Len(t, summary.ProcessedTransactions, len(transactions))
	require.Equal(t, []core_types.Log{
		*core_types.CoreLogFromGethLog(&types.Log{Address: common.Address{10}}),
		*core_types.CoreLogFromGethLog(&types.Log{Address: common.Address{20}}),
		*core_types.CoreLogFromGethLog(&types.Log{Address: common.Address{30}}),
	}, emitted)
}

func TestProcessWithDifficulty_onNewLog_SkipsCallbackWhenNil(t *testing.T) {
	ctrl := gomock.NewController(t)

	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	tx, err := types.SignTx(
		types.NewTx(&types.LegacyTx{Nonce: 0, To: &common.Address{}, Gas: 21_000}),
		types.FrontierSigner{},
		key,
	)
	require.NoError(t, err)

	stateDb := state.NewMockStateDB(ctrl)
	stateDb.EXPECT().SetTxContext(gomock.Any(), gomock.Any()).AnyTimes()
	stateDb.EXPECT().TxIndex().Return(0).AnyTimes()
	stateDb.EXPECT().GetLogs(gomock.Any(), gomock.Any()).
		Return([]*types.Log{{Address: common.Address{1}}}).MinTimes(1)

	mockStateDbTransactionExecution(stateDb)

	processor := NewStateProcessorForHeadState(&params.ChainConfig{}, nil, opera.Upgrades{}, nil)
	block := &EvmBlock{
		EvmHeader: EvmHeader{
			Number:   big.NewInt(1),
			GasLimit: math.MaxUint64,
		},
		Transactions: []*types.Transaction{tx},
	}

	require.NotPanics(t, func() {
		summary := processor.ProcessWithDifficulty(
			block,
			stateDb,
			vm.Config{},
			math.MaxUint64,
			new(uint64),
			0,
			nil,
			big.NewInt(1),
			math.MaxUint64,
		)
		require.Len(t, summary.ProcessedTransactions, 1)
		require.NotNil(t, summary.ProcessedTransactions[0].Receipt)
	})
}

func TestProcessWithDifficulty_onNewLog_DoesNotLogRolledBackTransactions(t *testing.T) {
	ctrl := gomock.NewController(t)
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	chainId := big.NewInt(1)
	signer := types.LatestSignerForChainID(chainId)

	sender := crypto.PubkeyToAddress(key.PublicKey)

	tx, bundle, _ := bundle.NewBuilder().
		WithSigner(signer).
		AllOf(
			bundle.Step(key, types.AccessListTx{
				Gas:   21_000,
				Nonce: 0, // Correct nonce, will be processed.
			}),
			bundle.Step(key, types.AccessListTx{
				Gas:   21_000,
				Nonce: 2, // gapped nonce to cause a rollback.
			}),
		).BuildEnvelopeBundleAndPlan()
	successfulTxHash := bundle.GetTransactionsInReferencedOrder()[0].Hash()

	stateDb := state.NewMockStateDB(ctrl)
	stateDb.EXPECT().SetTxContext(gomock.Any(), gomock.Any()).AnyTimes()
	stateDb.EXPECT().TxIndex().Return(0).AnyTimes()

	// Transaction is expected to produce logs,
	stateDb.EXPECT().GetLogs(successfulTxHash, gomock.Any()).
		Return([]*types.Log{
			{Address: common.Address{1}, TxIndex: 0},
		}).MinTimes(1)

	stateDb.EXPECT().GetNonce(sender).Return(uint64(0))

	mockStateDbTransactionExecution(stateDb)

	any := gomock.Any()
	stateDb.EXPECT().HasBundleRecentlyBeenProcessed(gomock.Any()).Return(false)
	stateDb.EXPECT().InterTxSnapshot().AnyTimes()
	stateDb.EXPECT().RevertToInterTxSnapshot(0).MinTimes(1)
	stateDb.EXPECT().CreateContract(any)
	stateDb.EXPECT().AddProcessedBundle(any, any)

	upgrades := opera.Upgrades{
		Brio:               true,
		TransactionBundles: true,
	}

	processor := NewStateProcessorForHeadState(&params.ChainConfig{ChainID: chainId}, nil, upgrades, nil)
	block := &EvmBlock{
		EvmHeader: EvmHeader{
			Number:   big.NewInt(1),
			GasLimit: math.MaxUint64,
		},
		Transactions: []*types.Transaction{tx},
	}

	expectNoLogs := func(log *core_types.Log) {
		t.Fatal("logs shall not be collected")
	}

	summary := processor.ProcessWithDifficulty(
		block,
		stateDb,
		vm.Config{},
		math.MaxUint64,
		new(uint64),
		0,
		expectNoLogs,
		big.NewInt(1),
		math.MaxUint64,
	)

	require.Len(t, summary.ProcessedTransactions, 0)
}

// createScenarioWithTxCheckingDifficulty creates a test scenario where a single
// transaction checks that the block difficulty matches the expected value.
func createScenarioWithTxCheckingDifficulty(
	ctrl *gomock.Controller, expectedDifficulty *big.Int,
) (
	*state.MockStateDB,
	*EvmBlock,
) {
	// Prepare code that checks for the expected difficulty.
	code := []byte{
		byte(vm.DIFFICULTY),                              // fetch the difficulty
		byte(vm.PUSH1), byte(expectedDifficulty.Int64()), // push expected difficulty
		byte(vm.EQ),          // compare
		byte(vm.PUSH1), 0x09, // push jump-dest address
		byte(vm.JUMPI), 0x0a, // jump to success if equal
		byte(vm.REVERT),   // revert if difficulty is not expected
		byte(vm.JUMPDEST), // success
		byte(vm.STOP),     // stop execution without revert
	}

	// Create a state mock that is able to run a full transaction.
	any := gomock.Any()
	state := state.NewMockStateDB(ctrl)
	state.EXPECT().SetTxContext(any, any).AnyTimes()
	state.EXPECT().GetBalance(any).Return(uint256.NewInt(math.MaxInt64)).AnyTimes()
	state.EXPECT().SubBalance(any, any, any).AnyTimes()
	state.EXPECT().Prepare(any, any, any, any, any, any).AnyTimes()
	state.EXPECT().GetNonce(any).AnyTimes()
	state.EXPECT().SetNonce(any, any, any).AnyTimes()
	state.EXPECT().GetCode(any).Return(code).AnyTimes()
	state.EXPECT().Snapshot().AnyTimes()
	state.EXPECT().Exist(any).Return(true).AnyTimes()
	state.EXPECT().AddBalance(any, any, any).AnyTimes()
	state.EXPECT().GetCodeHash(any).Return(crypto.Keccak256Hash(code)).AnyTimes()
	state.EXPECT().RevertToSnapshot(any).AnyTimes()
	state.EXPECT().GetRefund().AnyTimes()
	state.EXPECT().AddRefund(any).AnyTimes()
	state.EXPECT().SubRefund(any).AnyTimes()
	state.EXPECT().GetLogs(any, any).AnyTimes()
	state.EXPECT().EndTransaction().AnyTimes()
	state.EXPECT().TxIndex().AnyTimes()

	// Create a block with a single transaction calling the smart contract
	// checking the difficulty.
	transactions := []*types.Transaction{
		types.NewTx(&types.LegacyTx{
			To:  &common.Address{1, 2, 3},
			Gas: 100_000,
		}),
	}

	block := &EvmBlock{
		EvmHeader: EvmHeader{
			Number: big.NewInt(1),
		},
		Transactions: transactions,
	}

	return state, block
}

func TestProcessWithDifficulty_ForwardsTimeToBundleProcessing(t *testing.T) {
	ctrl := gomock.NewController(t)
	stateDb := state.NewMockStateDB(ctrl)
	stateDb.EXPECT().HasBundleRecentlyBeenProcessed(gomock.Any()).AnyTimes()
	stateDb.EXPECT().InterTxSnapshot().AnyTimes()
	stateDb.EXPECT().AddProcessedBundle(gomock.Any(), gomock.Any()).AnyTimes()

	currentTime := inter.Timestamp(12_345)

	processableBundle := bundle.NewBuilder().AllOf().SetNotBefore(currentTime).Build()
	blockedBundle := bundle.NewBuilder().AllOf().SetNotBefore(currentTime + 1).Build()

	processor := NewStateProcessorForHeadState(&params.ChainConfig{}, nil, opera.Upgrades{
		Brio:               true,
		TransactionBundles: true,
	}, nil)

	block := &EvmBlock{
		EvmHeader: EvmHeader{
			Number: big.NewInt(1),
			Time:   currentTime,
		},
		Transactions: []*types.Transaction{
			processableBundle,
			blockedBundle,
		},
	}

	summary := processor.ProcessWithDifficulty(
		block, stateDb, vm.Config{}, math.MaxUint64,
		new(uint64), 0, nil, big.NewInt(1), math.MaxUint64,
	)

	// The rejected bundle should be listed in the summary, the accepted not,
	// as the envelope is not reported and the bundle itself is empty.
	require.Len(t, summary.ProcessedTransactions, 1)

	processed := summary.ProcessedTransactions[0]
	require.Equal(t, processed.Transaction, blockedBundle)
	require.Nil(t, processed.Receipt)
}

func TestProcess_ForwardsCorrectIndexToTransactionProcessor(t *testing.T) {
	for _, offset := range []int{0, 1, 42} {
		t.Run(fmt.Sprintf("offset=%d", offset), func(t *testing.T) {
			ctrl := gomock.NewController(t)
			upgrades := opera.Upgrades{Brio: true, TransactionBundles: true}
			processor := NewStateProcessorForHeadState(&params.ChainConfig{}, nil, upgrades, nil)

			any := gomock.Any()
			state := state.NewMockStateDB(ctrl)

			// The legacyTxIndex always starts counting at 0 for every call and
			// it is used to set up the transaction context in the StateDB.
			state.EXPECT().SetTxContext(any, 0).AnyTimes()

			// The trueTxIndex is the one that is passed to the function and
			// used to get the offset of the bundle in the block.
			state.EXPECT().AddProcessedBundle(any, bundle.PositionInBlock{
				Offset: uint32(offset),
				Count:  0,
			})

			// accept everything else that is needed to get the transaction to run
			state.EXPECT().GetBalance(any).Return(uint256.NewInt(1e18)).AnyTimes()
			state.EXPECT().SubBalance(any, any, any).AnyTimes()
			state.EXPECT().Prepare(any, any, any, any, any, any).AnyTimes()
			state.EXPECT().GetNonce(any).AnyTimes()
			state.EXPECT().SetNonce(any, any, any).AnyTimes()
			state.EXPECT().GetCode(any).AnyTimes()
			state.EXPECT().Snapshot().AnyTimes()
			state.EXPECT().Exist(any).Return(true).AnyTimes()
			state.EXPECT().AddBalance(any, any, any).AnyTimes()
			state.EXPECT().GetCodeHash(any).AnyTimes()
			state.EXPECT().RevertToSnapshot(any).AnyTimes()
			state.EXPECT().GetRefund().AnyTimes()
			state.EXPECT().AddRefund(any).AnyTimes()
			state.EXPECT().SubRefund(any).AnyTimes()
			state.EXPECT().GetLogs(any, any).AnyTimes()
			state.EXPECT().EndTransaction().AnyTimes()
			state.EXPECT().TxIndex().AnyTimes()
			state.EXPECT().HasBundleRecentlyBeenProcessed(any).AnyTimes()
			state.EXPECT().InterTxSnapshot().AnyTimes()

			// create a block with an empty bundle
			block := &EvmBlock{
				EvmHeader: EvmHeader{
					Number: big.NewInt(1),
				},
				Transactions: []*types.Transaction{
					bundle.NewBuilder().Build(),
				},
			}

			// run the block on the processor with the desired initial offset
			processor.Process(block, state, vm.Config{}, math.MaxUint64, new(uint64), offset, nil, math.MaxUint64)
		})
	}
}

func TestApplyTransaction_InternalTransactionsSkipBaseFeeCharges(t *testing.T) {
	for _, internal := range []bool{true, false} {
		t.Run("internal="+fmt.Sprint(internal), func(t *testing.T) {
			ctxt := gomock.NewController(t)
			state := state.NewMockStateDB(ctxt)

			any := gomock.Any()
			state.EXPECT().GetBalance(any).Return(uint256.NewInt(0))
			state.EXPECT().SubBalance(any, any, any)
			state.EXPECT().EndTransaction()
			if !internal {
				state.EXPECT().GetNonce(any)
				state.EXPECT().GetCode(any)
			}

			evm := vm.NewEVM(vm.BlockContext{}, state, &params.ChainConfig{}, vm.Config{})
			gp := core.NewGasPool(1000000)

			// The transaction will fail for various reasons, but for this test
			// this is not relevant. We just want to check if the base fee
			// configuration flag is updated to match the SkipAccountChecks flag.
			_, _, err := applyTransaction(&core.Message{
				SkipNonceChecks:       internal,
				SkipTransactionChecks: internal,
				GasPrice:              big.NewInt(0),
				Value:                 big.NewInt(0),
			}, gp, state, nil, nil, nil, evm)
			if err == nil {
				t.Errorf("expected transaction to fail")
			}

			if want, got := internal, evm.Config.NoBaseFee; want != got {
				t.Fatalf("want %v, got %v", want, got)
			}
		})
	}
}

func TestApplyTransaction_FailsForTransactionWithInvalidGasPrice(t *testing.T) {
	ctrl := gomock.NewController(t)
	stateDb := state.NewMockStateDB(ctrl)
	stateDb.EXPECT().EndTransaction()

	msg := &core.Message{
		GasPrice: big.NewInt(-1),
	}
	_, _, err := applyTransaction(msg, nil, stateDb, nil, nil, nil, nil)
	require.ErrorContains(t, err, "failed to create EVM transaction context")
}

func TestApplyTransaction_ApplyMessageError_RevertsSnapshotIfPrague(t *testing.T) {
	versions := map[string]bool{
		"pre prague": false,
		"prague":     true,
	}

	for name, isPrague := range versions {
		t.Run(name, func(t *testing.T) {
			pragueTime := uint64(1000)
			callToSnapshot := 0
			if isPrague {
				pragueTime = 0
				callToSnapshot = 1
			}
			any := gomock.Any()
			ctrl := gomock.NewController(t)
			state := state.NewMockStateDB(ctrl)
			evm := vm.NewEVM(vm.BlockContext{}, state, &params.ChainConfig{
				LondonBlock:        new(big.Int).SetUint64(0),
				MergeNetsplitBlock: new(big.Int).SetUint64(0),
				ShanghaiTime:       new(uint64),
				CancunTime:         new(uint64),
				PragueTime:         &pragueTime,
			}, vm.Config{})
			gp := core.NewGasPool(1000000)

			blockNumber := big.NewInt(100)
			evm.Context.Random = &common.Hash{0x01} // triggers isMerge
			evm.Context.BlockNumber = blockNumber   // triggers isMerge
			evm.Context.Time = 100                  // triggers IsPrague

			initCode := make([]byte, 50000) // large init code to trigger error
			msg := &core.Message{
				From:                  common.Address{1},
				To:                    nil, // contract creation
				GasLimit:              1000000,
				GasPrice:              big.NewInt(1),
				GasFeeCap:             big.NewInt(0),
				GasTipCap:             big.NewInt(0),
				Value:                 big.NewInt(0),
				Data:                  initCode,
				SkipNonceChecks:       true,
				SkipTransactionChecks: true,
			}

			gomock.InOrder(
				state.EXPECT().Snapshot().Return(42).Times(callToSnapshot),
				state.EXPECT().GetBalance(msg.From).Return(uint256.NewInt(1000000)),
				state.EXPECT().SubBalance(any, any, any),
				state.EXPECT().RevertToSnapshot(42).Times(callToSnapshot),
				state.EXPECT().EndTransaction(),
			)

			receipt, gasUsed, err :=
				applyTransaction(msg, gp, state, blockNumber, nil, new(uint64), evm)
			require.ErrorContains(t, err, "max initcode size exceeded")
			require.Nil(t, receipt)
			require.Equal(t, uint64(0), gasUsed)
		})
	}
}

func TestApplyTransaction_SetsEffectiveGasPriceInReceipt(t *testing.T) {
	gasPrices := []*big.Int{
		big.NewInt(0),
		big.NewInt(100),
		big.NewInt(math.MaxInt64),
		new(big.Int).Lsh(big.NewInt(1), 200),
	}

	for _, price := range gasPrices {
		t.Run(fmt.Sprintf("%v", price), func(t *testing.T) {
			require := require.New(t)
			ctrl := gomock.NewController(t)
			state := state.NewMockStateDB(ctrl)

			// -- setup to get a simple transaction to pass --

			blockContext := vm.BlockContext{
				BlockNumber: big.NewInt(123),
				BaseFee:     big.NewInt(0),
				Transfer: func(_ vm.StateDB, _ common.Address, _ common.Address, _ *uint256.Int, _ *params.Rules) {
					// do nothing
				},
				CanTransfer: func(_ vm.StateDB, _ common.Address, _ *uint256.Int) bool {
					return true
				},
				Random: &common.Hash{}, // < signals Revision >= Merge
			}

			evm := vm.NewEVM(blockContext, state, &params.ChainConfig{
				LondonBlock:        new(big.Int).SetUint64(0),
				MergeNetsplitBlock: new(big.Int).SetUint64(0),
				ShanghaiTime:       new(uint64),
				CancunTime:         new(uint64),
			}, vm.Config{})

			// accept everything else that is needed to get the transaction to run
			any := gomock.Any()
			balance := new(uint256.Int).Lsh(uint256.NewInt(1), 240)
			state.EXPECT().GetBalance(any).Return(balance).AnyTimes()
			state.EXPECT().SubBalance(any, any, any).AnyTimes()
			state.EXPECT().Prepare(any, any, any, any, any, any).AnyTimes()
			state.EXPECT().GetNonce(any).AnyTimes()
			state.EXPECT().SetNonce(any, any, any).AnyTimes()
			state.EXPECT().GetCode(any).AnyTimes()
			state.EXPECT().Snapshot().AnyTimes()
			state.EXPECT().Exist(any).Return(true).AnyTimes()
			state.EXPECT().AddBalance(any, any, any).AnyTimes()
			state.EXPECT().GetRefund().AnyTimes()
			state.EXPECT().AddRefund(any).AnyTimes()
			state.EXPECT().GetLogs(any, any).AnyTimes()
			state.EXPECT().EndTransaction().AnyTimes()
			state.EXPECT().TxIndex().AnyTimes()

			gp := core.NewGasPool(1000000)
			var usedGas uint64

			blockNum := big.NewInt(12)

			tx := types.NewTx(&types.LegacyTx{})

			// -- end of setup --

			// The only thing we really care of is that the GasPrice of the
			// message is stored in the receipt.
			msg := &core.Message{
				GasPrice:  price,
				GasFeeCap: new(big.Int).Add(price, big.NewInt(1000)),
				GasTipCap: big.NewInt(100),
				To:        &common.Address{},
				Value:     big.NewInt(0),
				GasLimit:  21_000,
			}

			receipt, _, err := applyTransaction(msg, gp, state, blockNum, tx, &usedGas, evm)
			require.NoError(err)

			require.Equal(price, receipt.EffectiveGasPrice)
		})
	}
}

func TestApplyTransaction_CollectsLogsFromStateDbIntoReceipt(t *testing.T) {
	// Verify that logs returned by statedb.GetLogs are placed in the receipt.
	ctrl := gomock.NewController(t)
	stateDb := state.NewMockStateDB(ctrl)

	blockContext := vm.BlockContext{
		BlockNumber: big.NewInt(1),
		BaseFee:     big.NewInt(0),
		Transfer:    func(_ vm.StateDB, _, _ common.Address, _ *uint256.Int, _ *params.Rules) {},
		CanTransfer: func(_ vm.StateDB, _ common.Address, _ *uint256.Int) bool { return true },
		Random:      &common.Hash{},
	}
	evmInstance := vm.NewEVM(blockContext, stateDb, &params.ChainConfig{
		LondonBlock:        new(big.Int),
		MergeNetsplitBlock: new(big.Int),
		ShanghaiTime:       new(uint64),
		CancunTime:         new(uint64),
	}, vm.Config{})

	any := gomock.Any()
	balance := new(uint256.Int).Lsh(uint256.NewInt(1), 200)
	stateDb.EXPECT().GetBalance(any).Return(balance).AnyTimes()
	stateDb.EXPECT().SubBalance(any, any, any).AnyTimes()
	stateDb.EXPECT().Prepare(any, any, any, any, any, any).AnyTimes()
	stateDb.EXPECT().GetNonce(any).AnyTimes()
	stateDb.EXPECT().SetNonce(any, any, any).AnyTimes()
	stateDb.EXPECT().GetCode(any).AnyTimes()
	stateDb.EXPECT().Snapshot().AnyTimes()
	stateDb.EXPECT().Exist(any).Return(true).AnyTimes()
	stateDb.EXPECT().AddBalance(any, any, any).AnyTimes()
	stateDb.EXPECT().GetRefund().AnyTimes()
	stateDb.EXPECT().AddRefund(any).AnyTimes()
	stateDb.EXPECT().EndTransaction().AnyTimes()
	stateDb.EXPECT().TxIndex().AnyTimes()

	tx := types.NewTx(&types.LegacyTx{To: &common.Address{2}, Gas: 21_000})
	expectedLogs := []*types.Log{{Address: common.Address{1}}}
	stateDb.EXPECT().GetLogs(tx.Hash(), common.Hash{}).Return(expectedLogs)

	gp := core.NewGasPool(1_000_000)
	var usedGas uint64
	msg := &core.Message{
		To:                    &common.Address{2},
		GasLimit:              21_000,
		GasPrice:              big.NewInt(0),
		GasFeeCap:             big.NewInt(0),
		GasTipCap:             big.NewInt(0),
		Value:                 big.NewInt(0),
		SkipNonceChecks:       true,
		SkipTransactionChecks: true,
	}

	receipt, _, err := applyTransaction(msg, gp, stateDb, big.NewInt(1), tx, &usedGas, evmInstance)
	require.NoError(t, err)
	require.Equal(t, expectedLogs, receipt.Logs)
}

// processFunction is a function type alias for the StateProcessor's Process
// function to allow side-by-side testing of different implementations.
type processFunction = func(
	block *EvmBlock,
	statedb state.StateDB,
	cfg vm.Config,
	gasLimit uint64,
	usedGas *uint64,
	trueTxOffset int,
	onNewLog func(*core_types.Log),
	maxBlockSize uint64,
) ProcessSummary

func getStateDbMockForTransactions(
	ctrl *gomock.Controller,
	transactions []*types.Transaction,
) *state.MockStateDB {
	// Allow basically everything, but expect the context to be set up for
	// the given transactions and their positions.
	state := state.NewMockStateDB(ctrl)
	txIndex := new(int)
	for _, tx := range transactions {
		state.EXPECT().SetTxContext(tx.Hash(), gomock.Any()).Do(
			func(hash common.Hash, index int) {
				*txIndex = index
			},
		).AnyTimes()
	}
	// When asked for the TxIndex, use the value that was set last.
	state.EXPECT().TxIndex().DoAndReturn(func() int {
		return *txIndex
	}).AnyTimes()

	any := gomock.Any()

	// Have transaction specific logs.
	state.EXPECT().GetLogs(any, any).DoAndReturn(
		func(_, _ common.Hash) []*types.Log {
			return []*types.Log{
				{
					Address: common.Address{byte(*txIndex)},
					TxIndex: uint(*txIndex),
				},
			}
		},
	).AnyTimes()

	state.EXPECT().GetBalance(any).Return(uint256.NewInt(math.MaxInt64)).AnyTimes()
	state.EXPECT().AddBalance(any, any, any).AnyTimes()
	state.EXPECT().SubBalance(any, any, any).AnyTimes()
	state.EXPECT().Prepare(any, any, any, any, any, any).AnyTimes()
	state.EXPECT().GetNonce(any).AnyTimes()
	state.EXPECT().SetNonce(any, any, any).AnyTimes()
	state.EXPECT().GetCodeHash(any).Return(types.EmptyCodeHash).AnyTimes()
	state.EXPECT().GetCode(any).AnyTimes()
	state.EXPECT().GetStorageRoot(any).Return(types.EmptyRootHash).AnyTimes()
	state.EXPECT().Snapshot().AnyTimes()
	state.EXPECT().Exist(any).Return(true).AnyTimes()
	state.EXPECT().GetRefund().AnyTimes()
	state.EXPECT().EndTransaction().AnyTimes()
	return state
}

func TestRunTransactions_RunsAllTransactionsAndCollectsProcessedTransactions(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := NewMock_transactionRunner(ctrl)

	txs := []*types.Transaction{
		getRegularTransaction(t),
		getSponsorshipRequest(t),
		getTransactionBundle(t),
	}

	regularTxResult := ProcessedTransaction{
		Transaction: types.NewTx(&types.LegacyTx{Nonce: 1}),
		Receipt:     &types.Receipt{GasUsed: 100},
	}

	sponsoredTxResult := []ProcessedTransaction{
		{
			Transaction: types.NewTx(&types.LegacyTx{Nonce: 2}),
			Receipt:     &types.Receipt{GasUsed: 200},
		},
		{ // to simulate the internal fee deduction transaction
			Transaction: types.NewTx(&types.LegacyTx{Nonce: 3}),
			Receipt:     &types.Receipt{GasUsed: 300},
		},
	}

	bundleTxResult := []ProcessedTransaction{
		{
			Transaction: types.NewTx(&types.LegacyTx{Nonce: 4}),
			Receipt:     &types.Receipt{GasUsed: 400},
		},
		{
			Transaction: types.NewTx(&types.LegacyTx{Nonce: 5}),
			Receipt:     &types.Receipt{GasUsed: 500},
		},
		{
			Transaction: types.NewTx(&types.LegacyTx{Nonce: 6}),
			Receipt:     &types.Receipt{GasUsed: 600},
		},
	}

	context := &runContext{
		runner:   runner,
		upgrades: opera.Upgrades{Brio: true, GasSubsidies: true, TransactionBundles: true},
		signer:   types.LatestSignerForChainID(big.NewInt(1)),
	}
	gomock.InOrder(
		runner.EXPECT().runRegularTransaction(context, txs[0], 4, gomock.Any()).Return(
			regularTxResult,
			core_types.TransactionResultSuccessful,
		),
		runner.EXPECT().runSponsoredTransaction(context, txs[1], 5, gomock.Any()).Return(
			sponsoredTxResult,
			core_types.TransactionResultSuccessful,
		),
		runner.EXPECT().runTransactionBundle(context, txs[2], 7, gomock.Any()).Return(
			bundleTxResult,
			core_types.TransactionResultSuccessful,
			core_types.ExecutionCost(0),
		),
	)

	// run the transactions; as a side-effect, check that the
	// transaction offset is correctly initialized and updated.
	summary := runTransactions(context, txs, 4, math.MaxUint64)
	got := summary.ProcessedTransactions
	require.Len(t, got, 6)

	want := []ProcessedTransaction{}
	want = append(want, regularTxResult)
	want = append(want, sponsoredTxResult...)
	want = append(want, bundleTxResult...)
	require.Equal(t, want, got)
}

func TestRunTransactions_ProvidesNextIndexAsOriginalIndexPlusNumberOfPreviouslyProcessedTransactions(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := NewMock_transactionRunner(ctrl)

	txs := []*types.Transaction{
		getSponsorshipRequest(t),
		getRegularTransaction(t),
		getRegularTransaction(t),
		getTransactionBundle(t),
		getRegularTransaction(t),
	}

	sponsoredTxResult1 := []ProcessedTransaction{
		{
			Transaction: types.NewTx(&types.LegacyTx{}),
			Receipt:     &types.Receipt{},
		},
		{ // to simulate the internal fee deduction transaction
			Transaction: types.NewTx(&types.LegacyTx{}),
			Receipt:     &types.Receipt{},
		},
	}

	regularTxResult2 := ProcessedTransaction{
		Transaction: types.NewTx(&types.LegacyTx{}),
		Receipt:     nil,
	}

	regularTxResult3 := ProcessedTransaction{
		Transaction: types.NewTx(&types.LegacyTx{}),
		Receipt:     &types.Receipt{},
	}

	bundleTxResult4 := []ProcessedTransaction{
		{
			Transaction: types.NewTx(&types.LegacyTx{}),
			Receipt:     &types.Receipt{},
		},
		{
			Transaction: types.NewTx(&types.LegacyTx{}),
			Receipt:     nil,
		},
		{
			Transaction: types.NewTx(&types.LegacyTx{}),
			Receipt:     &types.Receipt{},
		},
	}

	regularTxResult5 := ProcessedTransaction{
		Transaction: types.NewTx(&types.LegacyTx{}),
		Receipt:     &types.Receipt{},
	}

	context := &runContext{
		signer: types.LatestSignerForChainID(big.NewInt(1)),
		runner: runner,
		upgrades: opera.Upgrades{
			Brio:               true,
			GasSubsidies:       true,
			TransactionBundles: true,
		},
	}

	trueStartIndex := 7
	gomock.InOrder(
		runner.EXPECT().runSponsoredTransaction(context, txs[0], trueStartIndex, gomock.Any()).Return(
			sponsoredTxResult1,
			core_types.TransactionResultSuccessful,
		),
		// the sponsored transaction results in two processed transactions: the sponsored transaction itself and an internal fee deduction transaction, so the next transaction should have index startIndex + 2
		runner.EXPECT().runRegularTransaction(context, txs[1], trueStartIndex+2, gomock.Any()).Return(
			regularTxResult2,
			core_types.TransactionResultSuccessful,
		),
		// the regular transaction has a nil receipt, which does not increase
		// the transaction offset.
		runner.EXPECT().runRegularTransaction(context, txs[2], trueStartIndex+2, gomock.Any()).Return(
			regularTxResult3,
			core_types.TransactionResultSuccessful,
		),
		// the bundle transaction has 3 results, one with a nil receipt, but all
		// should be counted towards the legacy offset of the next transaction,
		// but only the accepted ones should be counted towards the true offset
		runner.EXPECT().runTransactionBundle(context, txs[3], trueStartIndex+3, gomock.Any()).Return(
			bundleTxResult4,
			core_types.TransactionResultSuccessful,
			core_types.ExecutionCost(0),
		),
		// the next regular transaction should see all transactions processed
		// for the bundle transaction, even the one with the nil receipt; the
		// true index ignores the nil receipts.
		runner.EXPECT().runRegularTransaction(context, txs[4], trueStartIndex+5, gomock.Any()).Return(
			regularTxResult5,
			core_types.TransactionResultSuccessful,
		),
	)

	summary := runTransactions(context, txs, trueStartIndex, math.MaxUint64)
	got := summary.ProcessedTransactions

	want := []ProcessedTransaction{}
	want = append(want, sponsoredTxResult1...)
	want = append(want, regularTxResult2)
	want = append(want, regularTxResult3)
	want = append(want, bundleTxResult4...)
	want = append(want, regularTxResult5)
	require.Equal(t, want, got)
}

func TestRunTransaction_GasSubsidiesDisabled_ProcessesRegularTransaction(t *testing.T) {
	tests := map[string]*types.Transaction{
		"regular": getRegularTransaction(t),
		"request": getSponsorshipRequest(t),
	}

	for name, tx := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			runner := NewMock_transactionRunner(ctrl)
			context := &runContext{
				runner:   runner,
				upgrades: opera.Upgrades{GasSubsidies: false},
			}
			sizeLimit := uint64(1234)
			runner.EXPECT().runRegularTransaction(context, tx, 0, sizeLimit)
			runTransaction(context, tx, 0, sizeLimit)
		})
	}
}

func TestRunTransaction_GasSubsidiesEnabled_RunsRegularTransactionWithoutSponsorship(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := NewMock_transactionRunner(ctrl)

	tx := getRegularTransaction(t)
	processed := ProcessedTransaction{
		Transaction: tx,
		Receipt:     &types.Receipt{GasUsed: 1},
	}

	context := &runContext{
		runner:   runner,
		upgrades: opera.Upgrades{GasSubsidies: true},
	}
	runner.EXPECT().runRegularTransaction(context, tx, 123, uint64(456)).Return(processed, core_types.TransactionResultSuccessful)
	got, status, execCost := runTransaction(context, tx, 123, 456)
	require.Equal(t, []ProcessedTransaction{processed}, got)
	require.Equal(t, core_types.TransactionResultSuccessful, status)
	require.EqualValues(t, processed.Receipt.GasUsed, execCost)
}

func TestRunTransaction_GasSubsidiesEnabled_RunsSponsorshipRequestWithSponsorship(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := NewMock_transactionRunner(ctrl)

	tx := getSponsorshipRequest(t)

	context := &runContext{
		runner:   runner,
		upgrades: opera.Upgrades{GasSubsidies: true},
	}
	runner.EXPECT().runSponsoredTransaction(context, tx, 123, uint64(456)).Return(
		[]ProcessedTransaction{{
			Transaction: tx,
			Receipt:     &types.Receipt{GasUsed: 1},
		}},
		core_types.TransactionResultSuccessful,
	)
	processed, status, execCost := runTransaction(context, tx, 123, 456)
	require.Equal(t, core_types.TransactionResultSuccessful, status)
	require.Len(t, processed, 1)
	require.Equal(t, tx, processed[0].Transaction)
	require.NotNil(t, processed[0].Receipt)
	require.EqualValues(t, 1, execCost)
}

func TestRunTransactions_GasSubsidiesDisabled_BundlesDisabled_ProcessesAsRegularTransaction(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := NewMock_transactionRunner(ctrl)

	tx := getTransactionBundle(t)

	context := &runContext{
		runner:   runner,
		upgrades: opera.Upgrades{TransactionBundles: false, GasSubsidies: false},
	}
	sizeLimit := uint64(math.MaxUint64)
	runner.EXPECT().runRegularTransaction(context, tx, 123, sizeLimit).Return(
		ProcessedTransaction{
			Transaction: tx,
			Receipt:     nil,
		},
		core_types.TransactionResultSuccessful,
	)
	summary := runTransactions(context, []*types.Transaction{tx}, 123, sizeLimit)
	processed := summary.ProcessedTransactions
	require.Len(t, processed, 1)
	require.Equal(t, tx, processed[0].Transaction)
	require.Nil(t, processed[0].Receipt)
}

func TestRunTransactions_GasSubsidiesEnabled_BundlesDisabled_ProcessesAsSponsorshipRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := NewMock_transactionRunner(ctrl)

	tx := getTransactionBundle(t)

	context := &runContext{
		runner:   runner,
		upgrades: opera.Upgrades{TransactionBundles: false, GasSubsidies: true},
	}
	sizeLimit := uint64(math.MaxUint64)
	runner.EXPECT().runSponsoredTransaction(context, tx, 123, sizeLimit).Return(
		[]ProcessedTransaction{{

			Transaction: tx,
			Receipt:     nil,
		}},
		core_types.TransactionResultSuccessful,
	)
	summary := runTransactions(context, []*types.Transaction{tx}, 123, sizeLimit)
	processed := summary.ProcessedTransactions
	require.Len(t, processed, 1)
	require.Equal(t, tx, processed[0].Transaction)
	require.Nil(t, processed[0].Receipt)
}

func TestRunTransactions_BundlesEnabled_RunsRegularTransactionOnItsOwn(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := NewMock_transactionRunner(ctrl)

	tx := getRegularTransaction(t)

	context := &runContext{
		runner:   runner,
		upgrades: opera.Upgrades{TransactionBundles: true},
	}
	sizeLimit := uint64(math.MaxUint64)
	runner.EXPECT().runRegularTransaction(context, tx, 123, sizeLimit).Return(
		ProcessedTransaction{
			Transaction: tx,
			Receipt:     nil,
		},
		core_types.TransactionResultSuccessful,
	)
	summary := runTransactions(context, []*types.Transaction{tx}, 123, sizeLimit)
	processed := summary.ProcessedTransactions
	require.Len(t, processed, 1)
	require.Equal(t, tx, processed[0].Transaction)
	require.Nil(t, processed[0].Receipt)
}

func TestRunTransactions_BundlesEnabled_RunsSponsorshipRequestWithSponsorship(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := NewMock_transactionRunner(ctrl)

	tx := getSponsorshipRequest(t)

	context := &runContext{
		runner:   runner,
		upgrades: opera.Upgrades{GasSubsidies: true, TransactionBundles: true},
	}
	sizeLimit := uint64(math.MaxUint64)
	runner.EXPECT().runSponsoredTransaction(context, tx, 123, sizeLimit).Return(
		[]ProcessedTransaction{{
			Transaction: tx,
			Receipt:     nil,
		}},
		core_types.TransactionResultSuccessful,
	)
	summary := runTransactions(context, []*types.Transaction{tx}, 123, sizeLimit)
	processed := summary.ProcessedTransactions
	require.Len(t, processed, 1)
	require.Equal(t, tx, processed[0].Transaction)
	require.Nil(t, processed[0].Receipt)
}

func TestRunTransactions_EnvelopeAndBundleOnly_SemanticsEnabledByBrio_ExecutionEnabledByBundleFlag(t *testing.T) {
	signer := types.LatestSignerForChainID(big.NewInt(1))

	envelopeTx := getTransactionBundle(t)
	txBundle, _, err := bundle.ValidateEnvelope(signer, envelopeTx)
	require.NoError(t, err)
	bundleOnlyTx := txBundle.GetTransactionsInReferencedOrder()[0]

	cases := map[string]struct {
		tx                 *types.Transaction
		brio               bool
		transactionBundles bool
		expectRunRegular   bool
		expectRunBundle    bool
		expectInvalid      bool
	}{
		"Envelope/PreBrio/WithoutBundles": {
			tx:                 envelopeTx,
			brio:               false,
			transactionBundles: false,
			expectRunRegular:   true,
		},
		"Envelope/PreBrio/WithBundles": {
			tx:                 envelopeTx,
			brio:               false,
			transactionBundles: true,
			expectRunRegular:   true,
		},
		"Envelope/PostBrio/WithoutBundles": {
			tx:                 envelopeTx,
			brio:               true,
			transactionBundles: false,
			expectInvalid:      true,
		},
		"Envelope/PostBrio/WithBundles": {
			tx:                 envelopeTx,
			brio:               true,
			transactionBundles: true,
			expectRunBundle:    true,
		},
		"BundleOnly/PreBrio/WithoutBundles": {
			tx:                 bundleOnlyTx,
			brio:               false,
			transactionBundles: false,
			expectRunRegular:   true,
		},
		"BundleOnly/PreBrio/WithBundles": {
			tx:                 bundleOnlyTx,
			brio:               false,
			transactionBundles: true,
			expectRunRegular:   true,
		},
		"BundleOnly/PostBrio/WithoutBundles": {
			tx:                 bundleOnlyTx,
			brio:               true,
			transactionBundles: false,
			expectInvalid:      true,
		},
		"BundleOnly/PostBrio/WithBundles": {
			tx:                 bundleOnlyTx,
			brio:               true,
			transactionBundles: true,
			expectInvalid:      true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			runner := NewMock_transactionRunner(ctrl)
			context := &runContext{
				runner: runner,
				upgrades: opera.Upgrades{
					Brio:               tc.brio,
					TransactionBundles: tc.transactionBundles,
				},
			}

			sizeLimit := uint64(math.MaxUint64)
			if tc.expectRunRegular {
				runner.EXPECT().runRegularTransaction(context, tc.tx, 0, sizeLimit).
					Return(ProcessedTransaction{Transaction: tc.tx, Receipt: &types.Receipt{}}, core_types.TransactionResultSuccessful)
			}
			if tc.expectRunBundle {
				runner.EXPECT().runTransactionBundle(context, tc.tx, 0, sizeLimit).
					Return([]ProcessedTransaction{{Transaction: tc.tx, Receipt: &types.Receipt{}}}, core_types.TransactionResultSuccessful, core_types.ExecutionCost(0))
			}

			summary := runTransactions(context, []*types.Transaction{tc.tx}, 0, sizeLimit)
			processed := summary.ProcessedTransactions
			require.Len(t, processed, 1)
			if tc.expectInvalid {
				require.Equal(t, ProcessedTransaction{tc.tx, nil}, processed[0])
			} else {
				require.Equal(t, ProcessedTransaction{tc.tx, &types.Receipt{}}, processed[0])
			}
		})
	}
}

func TestRunTransactions_BundleOnlyTxsAreNotFilteredDuringReplay(t *testing.T) {
	signer := types.LatestSignerForChainID(big.NewInt(1))

	envelopeTx := getTransactionBundle(t)
	txBundle, _, err := bundle.ValidateEnvelope(signer, envelopeTx)
	require.NoError(t, err)
	bundleOnlyTx := txBundle.GetTransactionsInReferencedOrder()[0]

	cases := map[string]struct {
		brio               bool
		transactionBundles bool
	}{
		"PreBrio/WithoutBundles": {
			brio:               false,
			transactionBundles: false,
		},
		"PreBrio/WithBundles": {
			brio:               false,
			transactionBundles: true,
		},
		"PostBrio/WithoutBundles": {
			brio:               true,
			transactionBundles: false,
		},
		"PostBrio/WithBundles": {
			brio:               true,
			transactionBundles: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			runner := NewMock_transactionRunner(ctrl)
			context := &runContext{
				runner: runner,
				upgrades: opera.Upgrades{
					Brio:               tc.brio,
					TransactionBundles: tc.transactionBundles,
				},
				forReplay: true, // this is the relevant part for this test
			}
			sizeLimit := uint64(math.MaxUint64)
			runner.EXPECT().runRegularTransaction(context, bundleOnlyTx, 0, sizeLimit).
				Return(ProcessedTransaction{Transaction: bundleOnlyTx, Receipt: &types.Receipt{}}, core_types.TransactionResultSuccessful)

			summary := runTransactions(context, []*types.Transaction{bundleOnlyTx}, 0, sizeLimit)
			processed := summary.ProcessedTransactions
			require.Len(t, processed, 1)
			require.Equal(t, ProcessedTransaction{bundleOnlyTx, &types.Receipt{}}, processed[0])
		})
	}
}

func TestRunSponsoredTransaction_InsufficientGas_SkipsTransaction(t *testing.T) {
	const overhead = 123_456 // overhead to be charged for sponsored transactions

	tests := map[string]struct {
		availableGas uint64
		shouldSkip   bool
	}{
		"no gas": {
			availableGas: 0,
			shouldSkip:   true,
		},
		"not enough for sponsored tx": {
			availableGas: 20_999,
			shouldSkip:   true,
		},
		"enough for sponsored tx, not enough for fee deduction": {
			availableGas: 21_000,
			shouldSkip:   true,
		},
		"just not enough for both": {
			availableGas: 21_000 + overhead - 1,
			shouldSkip:   true,
		},
		"enough for both": {
			availableGas: 21_000 + overhead,
			shouldSkip:   false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			state := state.NewMockStateDB(ctrl)
			evm := NewMock_evm(ctrl)

			tx := getSponsorshipRequest(t)

			gasPool := core.NewGasPool(test.availableGas)
			context := &runContext{
				gasPool:  gasPool,
				statedb:  state,
				signer:   types.LatestSignerForChainID(nil),
				baseFee:  big.NewInt(1),
				upgrades: opera.Upgrades{GasSubsidies: true},
			}

			// Snapshot for the IsCovered call
			state.EXPECT().Snapshot().Return(1)
			state.EXPECT().RevertToSnapshot(1)

			// Call to getConfig contract and return the expected overhead.
			any := gomock.Any()
			result := make([]byte, 3*32)
			binary.BigEndian.PutUint64(result[88:], overhead)
			evm.EXPECT().Call(any, any, any, any, any).
				Return(result, uint64(0), nil)

			// Call made by IsCovered
			evm.EXPECT().Call(any, any, any, any, any).
				Return([]byte{31: 1}, uint64(0), nil) // indicates "covered"

			if !test.shouldSkip {

				// Request for the nonce of the internal fee-deduction.
				state.EXPECT().GetNonce(common.Address{}).Return(uint64(123))

				evm.EXPECT().runWithoutBaseFeeCheck(any, tx, any).Return(ProcessedTransaction{
					Transaction: tx,
					Receipt: &types.Receipt{
						Status:  types.ReceiptStatusSuccessful,
						GasUsed: 21_000,
					},
				})

				// Expect the fee deduction transaction to be processed as well.
				evm.EXPECT().runWithoutBaseFeeCheck(any, any, any).Return(ProcessedTransaction{
					Transaction: &types.Transaction{},
					Receipt: &types.Receipt{
						Status:  types.ReceiptStatusSuccessful,
						GasUsed: 321, // arbitrary
					},
				})
			}

			runner := &transactionRunner{evm: evm}
			sizeLimit := uint64(math.MaxUint64)
			got, status := runner.runSponsoredTransaction(context, tx, 0, sizeLimit)
			if test.shouldSkip {
				require.Equal(t, core_types.TransactionResultInvalid, status)
			} else {
				require.Equal(t, core_types.TransactionResultSuccessful, status)
			}

			if test.shouldSkip {
				want := []ProcessedTransaction{{
					Transaction: tx,
					Receipt:     nil,
				}}
				require.Equal(t, want, got)
			} else {
				require.Len(t, got, 2)
				require.Equal(t, tx, got[0].Transaction)
				require.NotNil(t, got[0].Receipt)
				require.NotNil(t, got[1].Transaction)
				require.NotNil(t, got[1].Receipt)
			}
		})
	}
}

func TestRunSponsoredTransaction_SponsorshipNotCovered_ReturnsASkippedTransaction(t *testing.T) {
	ctrl := gomock.NewController(t)
	state := state.NewMockStateDB(ctrl)

	tx := types.NewTx(&types.LegacyTx{
		Nonce: 0, To: &common.Address{1}, Gas: 21_000,
	})

	// Snapshot for the IsCovered call
	state.EXPECT().Snapshot().Return(1)
	state.EXPECT().RevertToSnapshot(1)

	gasPool := core.NewGasPool(1_000_000)
	context := &runContext{
		statedb:  state,
		gasPool:  gasPool,
		upgrades: opera.Upgrades{GasSubsidies: false}, // < nothing is covered
	}

	runner := &transactionRunner{}
	sizeLimit := uint64(math.MaxUint64)
	got, status := runner.runSponsoredTransaction(context, tx, 0, sizeLimit)
	require.Equal(t, core_types.TransactionResultInvalid, status)
	want := []ProcessedTransaction{{
		Transaction: tx,
		Receipt:     nil,
	}}
	require.Equal(t, want, got)
}

func TestRunSponsoredTransaction_SponsorshipCoverageCheckFails_ReturnsASkippedTransaction(t *testing.T) {
	ctrl := gomock.NewController(t)
	state := state.NewMockStateDB(ctrl)
	evm := NewMock_evm(ctrl)

	tx := getSponsorshipRequest(t)

	// Snapshot for the IsCovered call
	state.EXPECT().Snapshot().Return(1)
	state.EXPECT().RevertToSnapshot(1)

	// Call made by IsCovered fails.
	any := gomock.Any()
	issue := fmt.Errorf("sponsorship check failed")
	evm.EXPECT().Call(any, any, any, any, any).Return(nil, uint64(0), issue)

	gasPool := core.NewGasPool(1_000_000)
	context := &runContext{
		statedb:  state,
		signer:   types.LatestSignerForChainID(nil),
		baseFee:  big.NewInt(1),
		gasPool:  gasPool,
		upgrades: opera.Upgrades{GasSubsidies: true},
	}

	runner := &transactionRunner{evm: evm}
	sizeLimit := uint64(math.MaxUint64)
	got, status := runner.runSponsoredTransaction(context, tx, 0, sizeLimit)
	require.Equal(t, core_types.TransactionResultInvalid, status)
	want := []ProcessedTransaction{{
		Transaction: tx,
		Receipt:     nil,
	}}
	require.Equal(t, want, got)
}

func TestRunSponsoredTransaction_SponsoredTransactionIsSkipped_NoFeeDeductionTxIsIssued(t *testing.T) {
	ctrl := gomock.NewController(t)
	state := state.NewMockStateDB(ctrl)
	evm := NewMock_evm(ctrl)

	tx := getSponsorshipRequest(t)

	// Snapshot for the IsCovered call
	state.EXPECT().Snapshot().Return(1)
	state.EXPECT().RevertToSnapshot(1)

	// Let the IsCovered call indicate that the transaction is covered,
	any := gomock.Any()
	evm.EXPECT().Call(any, any, any, any, any).
		Return([]byte{95: 0}, uint64(0), nil) // results of getGasConfig
	evm.EXPECT().Call(any, any, any, any, any).
		Return([]byte{31: 1}, uint64(0), nil) // indicates "covered"

	// Let the sponsored transaction be processed, but result in a skipped
	// transaction (e.g. due to a wrong nonce).
	evm.EXPECT().runWithoutBaseFeeCheck(any, tx, any).Return(ProcessedTransaction{
		Transaction: tx,
		Receipt:     nil,
	})

	gasPool := core.NewGasPool(1_000_000)
	context := &runContext{
		statedb:  state,
		signer:   types.LatestSignerForChainID(nil),
		baseFee:  big.NewInt(1),
		gasPool:  gasPool,
		upgrades: opera.Upgrades{GasSubsidies: true},
	}

	runner := &transactionRunner{evm: evm}
	sizeLimit := uint64(math.MaxUint64)
	got, status := runner.runSponsoredTransaction(context, tx, 0, sizeLimit)
	require.Equal(t, core_types.TransactionResultInvalid, status)
	want := []ProcessedTransaction{{
		Transaction: tx,
		Receipt:     nil,
	}}
	require.Equal(t, want, got)
}

func TestRunSponsoredTransaction_FailingCreationOfFeeDeduction_TransactionIsAcceptedWithoutFeeDeduction(t *testing.T) {
	const overhead = 255

	ctrl := gomock.NewController(t)
	state := state.NewMockStateDB(ctrl)
	evm := NewMock_evm(ctrl)

	tx := getSponsorshipRequest(t)

	// Snapshot for the IsCovered call
	state.EXPECT().Snapshot().Return(1)
	state.EXPECT().RevertToSnapshot(1)

	// Nonce request for the fee deduction transaction
	state.EXPECT().GetNonce(common.Address{}).Return(uint64(123))

	// Let the IsCovered call indicate that the transaction is covered,
	any := gomock.Any()
	evm.EXPECT().Call(any, any, any, any, any).
		Return([]byte{95: overhead}, uint64(0), nil) // results of getGasConfig
	evm.EXPECT().Call(any, any, any, any, any).
		Return([]byte{31: 1}, uint64(0), nil) // indicates "covered"

	// Simulate huge gas prices, that are still ok for the sponsored transaction
	// but that cause an overflow when the overhead gas is added.
	gasUsed := uint64(21_000_000_000) // < note: much more than the tx gas limit
	gasPrice := new(big.Int).Lsh(big.NewInt(1), 230)

	_, overflow := uint256.FromBig(
		new(big.Int).Mul(
			gasPrice,
			new(big.Int).SetUint64(tx.Gas()+overhead),
		),
	)
	require.False(t, overflow, "test setup invalid: gas price overflows maximum fees for sponsored transaction")

	_, overflow = uint256.FromBig(new(big.Int).Mul(gasPrice, new(big.Int).SetUint64(gasUsed)))
	require.True(t, overflow, "test setup invalid: gas price does not cause overflow for gas used")

	// The sponsored transaction is processed successfully, consuming huge
	// amounts of gas for some reason.
	processed := ProcessedTransaction{
		Transaction: tx,
		Receipt: &types.Receipt{
			Status:  types.ReceiptStatusSuccessful,
			GasUsed: gasUsed,
		},
	}
	evm.EXPECT().runWithoutBaseFeeCheck(any, tx, any).Return(processed)

	gasPool := core.NewGasPool(1_000_000)
	context := &runContext{
		statedb:  state,
		signer:   types.LatestSignerForChainID(nil),
		baseFee:  gasPrice,
		gasPool:  gasPool,
		upgrades: opera.Upgrades{GasSubsidies: true},
	}

	runner := &transactionRunner{evm: evm}
	sizeLimit := uint64(math.MaxUint64)
	got, status := runner.runSponsoredTransaction(context, tx, 0, sizeLimit)
	require.Equal(t, core_types.TransactionResultSuccessful, status)
	want := []ProcessedTransaction{processed}
	require.Equal(t, want, got)
}

func TestRunSponsoredTransaction_FeeDeductionTxIsSkipped_TransactionIsAcceptedWithoutFeeDeduction(t *testing.T) {
	ctrl := gomock.NewController(t)
	state := state.NewMockStateDB(ctrl)
	evm := NewMock_evm(ctrl)

	tx := getSponsorshipRequest(t)

	// Snapshot for the IsCovered call
	state.EXPECT().Snapshot().Return(1)
	state.EXPECT().RevertToSnapshot(1)

	// Nonce request for the fee deduction transaction
	state.EXPECT().GetNonce(common.Address{}).Return(uint64(123))

	// Let the IsCovered call indicate that the transaction is covered,
	any := gomock.Any()
	evm.EXPECT().Call(any, any, any, any, any).
		Return([]byte{95: 0}, uint64(0), nil) // results of getGasConfig
	evm.EXPECT().Call(any, any, any, any, any).
		Return([]byte{31: 1}, uint64(0), nil) // indicates "covered"

	// Expect the sponsored transaction to be processed successfully.
	processedSponsoredTransaction := ProcessedTransaction{
		Transaction: tx,
		Receipt: &types.Receipt{
			Status:  types.ReceiptStatusSuccessful,
			GasUsed: 21_000,
		},
	}
	evm.EXPECT().runWithoutBaseFeeCheck(any, tx, any).Return(processedSponsoredTransaction)

	skippedFeeDeductionTransaction := ProcessedTransaction{
		Transaction: &types.Transaction{},
		Receipt:     nil,
	}
	evm.EXPECT().runWithoutBaseFeeCheck(any, gomock.Not(tx), any).
		Return(skippedFeeDeductionTransaction)

	gasPool := core.NewGasPool(1_000_000)
	context := &runContext{
		statedb:  state,
		signer:   types.LatestSignerForChainID(nil),
		baseFee:  big.NewInt(1),
		gasPool:  gasPool,
		upgrades: opera.Upgrades{GasSubsidies: true},
	}

	sizeLimit := uint64(math.MaxUint64)
	runner := &transactionRunner{evm: evm}
	got, status := runner.runSponsoredTransaction(context, tx, 0, sizeLimit)
	require.Equal(t, core_types.TransactionResultSuccessful, status)
	want := []ProcessedTransaction{
		processedSponsoredTransaction,
		skippedFeeDeductionTransaction,
	}
	require.Equal(t, want, got)
}

func TestRunSponsoredTransaction_FeeDeductionTxFails_TransactionIsAcceptedWithoutFeeDeduction(t *testing.T) {
	ctrl := gomock.NewController(t)
	state := state.NewMockStateDB(ctrl)
	evm := NewMock_evm(ctrl)

	tx := getSponsorshipRequest(t)

	// Snapshot for the IsCovered call
	state.EXPECT().Snapshot().Return(1)
	state.EXPECT().RevertToSnapshot(1)

	// Nonce request for the fee deduction transaction
	state.EXPECT().GetNonce(common.Address{}).Return(uint64(123))

	// Let the IsCovered call indicate that the transaction is covered,
	any := gomock.Any()
	evm.EXPECT().Call(any, any, any, any, any).
		Return([]byte{95: 0}, uint64(0), nil) // results of getGasConfig
	evm.EXPECT().Call(any, any, any, any, any).
		Return([]byte{31: 1}, uint64(0), nil) // indicates "covered"

	// Expect the sponsored transaction to be processed successfully.
	processedSponsoredTransaction := ProcessedTransaction{
		Transaction: tx,
		Receipt: &types.Receipt{
			Status:  types.ReceiptStatusSuccessful,
			GasUsed: 21_000,
		},
	}
	evm.EXPECT().runWithoutBaseFeeCheck(any, tx, any).Return(processedSponsoredTransaction)

	skippedFeeDeductionTransaction := ProcessedTransaction{
		Transaction: &types.Transaction{},
		Receipt: &types.Receipt{
			Status: types.ReceiptStatusFailed,
		},
	}
	evm.EXPECT().runWithoutBaseFeeCheck(any, gomock.Not(tx), any).
		Return(skippedFeeDeductionTransaction)

	gasPool := core.NewGasPool(1_000_000)
	context := &runContext{
		statedb:  state,
		signer:   types.LatestSignerForChainID(nil),
		baseFee:  big.NewInt(1),
		gasPool:  gasPool,
		upgrades: opera.Upgrades{GasSubsidies: true},
	}
	sizeLimit := uint64(math.MaxUint64)
	runner := &transactionRunner{evm: evm}
	got, status := runner.runSponsoredTransaction(context, tx, 0, sizeLimit)
	require.Equal(t, core_types.TransactionResultSuccessful, status)
	want := []ProcessedTransaction{
		processedSponsoredTransaction,
		skippedFeeDeductionTransaction,
	}
	require.Equal(t, want, got)
}

func TestRunSponsoredTransaction_TxIsNetworkSponsored_TransactionIsAcceptedWithoutPostTransaction(t *testing.T) {
	ctrl := gomock.NewController(t)
	state := state.NewMockStateDB(ctrl)
	evm := NewMock_evm(ctrl)

	tx := getSponsorshipRequest(t)

	// Snapshot for the IsCovered call
	state.EXPECT().Snapshot().Return(1)
	state.EXPECT().RevertToSnapshot(1)

	// Let the IsCovered call indicate that the transaction is covered by a
	// network sponsorship without tracking.
	any := gomock.Any()
	evm.EXPECT().Call(any, any, any, any, any).
		Return([]byte{95: 0}, uint64(0), nil) // results of getGasConfig
	evm.EXPECT().Call(any, any, any, any, any).
		Return([]byte{31: 2, 63: 0}, uint64(0), nil) // indicates "network covered"

	// Expect the sponsored transaction to be processed successfully.
	processedSponsoredTransaction := ProcessedTransaction{
		Transaction: tx,
		Receipt: &types.Receipt{
			Status:  types.ReceiptStatusSuccessful,
			GasUsed: 21_000,
		},
	}
	evm.EXPECT().runWithoutBaseFeeCheck(any, tx, any).Return(processedSponsoredTransaction)

	// expect no additional post-transaction for fee deduction or tracking.

	gasPool := core.NewGasPool(1_000_000)
	context := &runContext{
		statedb:  state,
		signer:   types.LatestSignerForChainID(nil),
		baseFee:  big.NewInt(1),
		gasPool:  gasPool,
		upgrades: opera.Upgrades{GasSubsidies: true},
	}
	sizeLimit := uint64(math.MaxUint64)
	runner := &transactionRunner{evm: evm}
	got, status := runner.runSponsoredTransaction(context, tx, 0, sizeLimit)
	require.Equal(t, core_types.TransactionResultSuccessful, status)
	want := []ProcessedTransaction{
		processedSponsoredTransaction,
	}
	require.Equal(t, want, got)
}

func TestRunSponsoredTransaction_TxIsNetworkSponsoredWithTracking_TransactionIsAcceptedWithTrackingTx(t *testing.T) {
	ctrl := gomock.NewController(t)
	state := state.NewMockStateDB(ctrl)
	evm := NewMock_evm(ctrl)

	tx := getSponsorshipRequest(t)

	// Snapshot for the IsCovered call
	state.EXPECT().Snapshot().Return(1)
	state.EXPECT().RevertToSnapshot(1)

	// Nonce request for the post-transaction
	state.EXPECT().GetNonce(common.Address{}).Return(uint64(123))

	// Let the IsCovered call indicate that the transaction is covered,
	any := gomock.Any()
	evm.EXPECT().Call(any, any, any, any, any).
		Return([]byte{95: 0}, uint64(0), nil) // results of getGasConfig
	evm.EXPECT().Call(any, any, any, any, any).
		Return([]byte{31: 3, 63: 12}, uint64(0), nil) // indicates network-sponsored with tracking

	// Expect the sponsored transaction to be processed successfully.
	processedSponsoredTransaction := ProcessedTransaction{
		Transaction: tx,
		Receipt: &types.Receipt{
			Status:  types.ReceiptStatusSuccessful,
			GasUsed: 21_000,
		},
	}
	evm.EXPECT().runWithoutBaseFeeCheck(any, tx, any).Return(processedSponsoredTransaction)

	baseFee := big.NewInt(0)

	trackingTx := subsidies.NewPostTxBuilder().
		ForNetworkSponsoredWithTracking().
		WithNonce(123).
		WithGasPrice(baseFee).
		WithGasLimit(0).
		WithId(subsidies.Identifier{31: 12}).
		BuildForTesting()

	processedTrackingTx := ProcessedTransaction{
		Transaction: trackingTx,
		Receipt: &types.Receipt{
			Status: types.ReceiptStatusSuccessful,
		},
	}

	evm.EXPECT().runWithoutBaseFeeCheck(any, any, any).
		DoAndReturn(func(_ *runContext, tx *types.Transaction, _ int) ProcessedTransaction {
			require.Equal(t, trackingTx.Hash(), tx.Hash())
			return processedTrackingTx
		})

	gasPool := core.NewGasPool(1_000_000)
	context := &runContext{
		statedb:  state,
		signer:   types.LatestSignerForChainID(nil),
		baseFee:  baseFee,
		gasPool:  gasPool,
		upgrades: opera.Upgrades{GasSubsidies: true},
	}
	sizeLimit := uint64(math.MaxUint64)
	runner := &transactionRunner{evm: evm}
	got, status := runner.runSponsoredTransaction(context, tx, 0, sizeLimit)
	require.Equal(t, core_types.TransactionResultSuccessful, status)
	want := []ProcessedTransaction{
		processedSponsoredTransaction,
		processedTrackingTx,
	}
	require.Equal(t, want, got)
}

func TestRunSponsoredTransaction_TxIndexIsIncrementedForFeeDeductionTx(t *testing.T) {
	ctrl := gomock.NewController(t)
	state := state.NewMockStateDB(ctrl)
	evm := NewMock_evm(ctrl)

	tx := getSponsorshipRequest(t)

	// Snapshot for the IsCovered call
	state.EXPECT().Snapshot().Return(1)
	state.EXPECT().RevertToSnapshot(1)

	// Nonce request for the fee deduction transaction
	state.EXPECT().GetNonce(common.Address{}).Return(uint64(123))

	any := gomock.Any()
	evm.EXPECT().Call(any, any, any, any, any).
		Return([]byte{95: 0}, uint64(0), nil) // results of getGasConfig
	evm.EXPECT().Call(any, any, any, any, any).
		Return([]byte{31: 1}, uint64(0), nil) // indicates "covered"

	txIndex := 7
	evm.EXPECT().runWithoutBaseFeeCheck(any, tx, txIndex).
		Return(ProcessedTransaction{
			Transaction: tx,
			Receipt:     &types.Receipt{},
		})

	evm.EXPECT().runWithoutBaseFeeCheck(any, gomock.Not(tx), txIndex+1).
		Return(ProcessedTransaction{
			Transaction: tx,
			Receipt:     &types.Receipt{},
		})

	gasPool := core.NewGasPool(1_000_000)
	context := &runContext{
		statedb:  state,
		signer:   types.LatestSignerForChainID(nil),
		baseFee:  big.NewInt(1),
		gasPool:  gasPool,
		upgrades: opera.Upgrades{GasSubsidies: true},
	}

	runner := &transactionRunner{evm: evm}
	sizeLimit := uint64(math.MaxUint64)
	got, status := runner.runSponsoredTransaction(context, tx, txIndex, sizeLimit)
	require.Equal(t, core_types.TransactionResultFailed, status)
	require.Len(t, got, 2)
	require.Equal(t, tx, got[0].Transaction)
	require.NotNil(t, got[0].Receipt)
	require.NotNil(t, got[1].Transaction)
	require.NotNil(t, got[1].Receipt)
}

func TestRunSponsoredTransaction_ForReplay_SkipsPostTransactions(t *testing.T) {
	ctrl := gomock.NewController(t)
	state := state.NewMockStateDB(ctrl)
	evm := NewMock_evm(ctrl)

	tx := getSponsorshipRequest(t)

	// Snapshot for the IsCovered call
	state.EXPECT().Snapshot().Return(1)
	state.EXPECT().RevertToSnapshot(1)

	any := gomock.Any()
	evm.EXPECT().Call(any, any, any, any, any).
		Return([]byte{95: 0}, uint64(0), nil) // results of getGasConfig
	evm.EXPECT().Call(any, any, any, any, any).
		Return([]byte{31: 1}, uint64(0), nil) // indicates "covered"

	txIndex := 7
	evm.EXPECT().runWithoutBaseFeeCheck(any, tx, txIndex).
		Return(ProcessedTransaction{
			Transaction: tx,
			Receipt:     &types.Receipt{Status: types.ReceiptStatusSuccessful},
		})

	gasPool := core.NewGasPool(1_000_000)
	context := &runContext{
		statedb:   state,
		signer:    types.LatestSignerForChainID(nil),
		baseFee:   big.NewInt(1),
		gasPool:   gasPool,
		upgrades:  opera.Upgrades{GasSubsidies: true},
		forReplay: true,
	}

	runner := &transactionRunner{evm: evm}
	sizeLimit := uint64(math.MaxUint64)
	got, status := runner.runSponsoredTransaction(context, tx, txIndex, sizeLimit)
	require.Equal(t, core_types.TransactionResultSuccessful, status)
	require.Len(t, got, 1)
	require.Equal(t, tx, got[0].Transaction)
	require.NotNil(t, got[0].Receipt)
}

func TestRunSponsoredTransaction_ProcessesAllPostTransactionsInOrder(t *testing.T) {
	tests := map[string][]ProcessedTransaction{
		"no post-tx": {},
		"one successful post-tx": {
			{
				Transaction: &types.Transaction{},
				Receipt: &types.Receipt{
					Status: types.ReceiptStatusSuccessful,
				},
			},
		},
		"one failed post-tx": {
			{
				Transaction: &types.Transaction{},
				Receipt: &types.Receipt{
					Status: types.ReceiptStatusFailed,
				},
			},
		},
		"one invalid post-tx": {
			{
				Transaction: &types.Transaction{},
				Receipt:     nil,
			},
		},
		"multiple successful post-txs": {
			{
				Transaction: &types.Transaction{},
				Receipt: &types.Receipt{
					Status: types.ReceiptStatusSuccessful,
				},
			},
			{
				Transaction: &types.Transaction{},
				Receipt: &types.Receipt{
					Status: types.ReceiptStatusSuccessful,
				},
			},
			{
				Transaction: &types.Transaction{},
				Receipt: &types.Receipt{
					Status: types.ReceiptStatusSuccessful,
				},
			},
		},
		"mix of successful, failed and invalid post-txs": {
			{
				Transaction: &types.Transaction{},
				Receipt: &types.Receipt{
					Status: types.ReceiptStatusSuccessful,
				},
			},
			{
				Transaction: &types.Transaction{},
				Receipt: &types.Receipt{
					Status: types.ReceiptStatusFailed,
				},
			},
			{
				Transaction: &types.Transaction{},
				Receipt:     nil,
			},
			{
				Transaction: &types.Transaction{},
				Receipt: &types.Receipt{
					Status: types.ReceiptStatusSuccessful,
				},
			},
			{
				Transaction: &types.Transaction{},
				Receipt: &types.Receipt{
					Status: types.ReceiptStatusFailed,
				},
			},
			{
				Transaction: &types.Transaction{},
				Receipt:     nil,
			},
			{
				Transaction: &types.Transaction{},
				Receipt: &types.Receipt{
					Status: types.ReceiptStatusSuccessful,
				},
			},
		},
	}

	for name, processedPostTxs := range tests {
		t.Run(name, func(t *testing.T) {

			ctrl := gomock.NewController(t)
			state := state.NewMockStateDB(ctrl)
			evm := NewMock_evm(ctrl)

			tx := getSponsorshipRequest(t)

			// Snapshot for the IsCovered call
			state.EXPECT().Snapshot().Return(1)
			state.EXPECT().RevertToSnapshot(1)

			any := gomock.Any()
			evm.EXPECT().Call(any, any, any, any, any).
				Return([]byte{95: 0}, uint64(0), nil) // results of getGasConfig
			evm.EXPECT().Call(any, any, any, any, any).
				Return([]byte{31: 1}, uint64(0), nil) // indicates "covered"

			// Run of the sponsored transaction.
			txIndex := 7
			evm.EXPECT().runWithoutBaseFeeCheck(any, tx, txIndex).
				Return(ProcessedTransaction{
					Transaction: tx,
					Receipt:     &types.Receipt{},
				})

			// Run of the post-transactions, also checking the tx-index.
			postTxIndex := txIndex + 1
			for _, postTx := range processedPostTxs {
				evm.EXPECT().runWithoutBaseFeeCheck(any, postTx.Transaction, postTxIndex).
					Return(postTx)
				if postTx.Receipt != nil {
					postTxIndex++
				}
			}

			gasPool := core.NewGasPool(1_000_000)
			context := &runContext{
				statedb:  state,
				signer:   types.LatestSignerForChainID(nil),
				baseFee:  big.NewInt(1),
				gasPool:  gasPool,
				upgrades: opera.Upgrades{GasSubsidies: true},
			}

			getPostTxs := func(subsidies.Sponsorship, subsidies.NonceSource, uint64, *big.Int) ([]*types.Transaction, error) {
				txs := []*types.Transaction{}
				for _, postTx := range processedPostTxs {
					txs = append(txs, postTx.Transaction)
				}
				return txs, nil
			}

			runner := &transactionRunner{evm: evm}
			sizeLimit := uint64(math.MaxUint64)
			got, status := runner.runSponsoredTransactionInternal(context, tx, txIndex, sizeLimit, getPostTxs)
			require.Equal(t, core_types.TransactionResultFailed, status)
			require.Len(t, got, len(processedPostTxs)+1)
			require.Equal(t, tx, got[0].Transaction)
			require.NotNil(t, got[0].Receipt)
			for i, postTx := range processedPostTxs {
				require.Equal(t, postTx.Transaction, got[i+1].Transaction)
				require.Equal(t, postTx.Receipt, got[i+1].Receipt)
			}
		})
	}
}

// TestRunSponsoredTransaction_PostTxOutcome_DeterminesRollbackUnderBrio covers
// the 2 × 3 matrix of [Brio on/off] × [post-tx skipped/failed/success].
// When Brio is enabled and the post-tx is not successful, the entire group is
// rolled back (sponsored tx disappears from the block, gas pool and usedGas
// are restored). In all other cases the full result list is returned.
func TestRunSponsoredTransaction_PostTxOutcome_DeterminesRollbackUnderBrio(t *testing.T) {
	postTx := &types.Transaction{}

	tests := map[string]struct {
		brio          bool
		postTxReceipt *types.Receipt
		wantRollback  bool
	}{
		"brio on / post-tx failed":   {true, &types.Receipt{Status: types.ReceiptStatusFailed}, true},
		"brio on / post-tx skipped":  {true, nil, true},
		"brio on / post-tx success":  {true, &types.Receipt{Status: types.ReceiptStatusSuccessful}, false},
		"brio off / post-tx failed":  {false, &types.Receipt{Status: types.ReceiptStatusFailed}, false},
		"brio off / post-tx skipped": {false, nil, false},
		"brio off / post-tx success": {false, &types.Receipt{Status: types.ReceiptStatusSuccessful}, false},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			stateDb := state.NewMockStateDB(ctrl)
			mockEvm := NewMock_evm(ctrl)

			tx := getSponsorshipRequest(t)
			any := gomock.Any()

			stateDb.EXPECT().Snapshot().Return(1)
			stateDb.EXPECT().RevertToSnapshot(1)
			mockEvm.EXPECT().Call(any, any, any, any, any).Return([]byte{95: 0}, uint64(0), nil) // getGasConfig
			mockEvm.EXPECT().Call(any, any, any, any, any).Return([]byte{31: 1}, uint64(0), nil) // chooseFund → mode 1

			const snapshotId = 42
			if test.brio {
				stateDb.EXPECT().InterTxSnapshot().Return(snapshotId)
			}
			if test.wantRollback {
				stateDb.EXPECT().RevertToInterTxSnapshot(snapshotId)
			}

			const initialGasInPool = uint64(1_000_000)
			gasPool := core.NewGasPool(initialGasInPool)
			usedGas := uint64(50_000)

			processedSponsored := ProcessedTransaction{
				Transaction: tx,
				Receipt:     &types.Receipt{Status: types.ReceiptStatusSuccessful, GasUsed: 21_000},
			}
			mockEvm.EXPECT().runWithoutBaseFeeCheck(any, tx, any).
				DoAndReturn(func(ctxt *runContext, _ *types.Transaction, _ int) ProcessedTransaction {
					_ = ctxt.gasPool.SubGas(21_000)
					*ctxt.usedGas += 21_000
					return processedSponsored
				})

			processedPost := ProcessedTransaction{Transaction: postTx, Receipt: test.postTxReceipt}
			mockEvm.EXPECT().runWithoutBaseFeeCheck(any, postTx, any).
				DoAndReturn(func(ctxt *runContext, _ *types.Transaction, _ int) ProcessedTransaction {
					_ = ctxt.gasPool.SubGas(1_000)
					*ctxt.usedGas += 1_000
					return processedPost
				})

			ctxt := &runContext{
				statedb:  stateDb,
				signer:   types.LatestSignerForChainID(nil),
				baseFee:  big.NewInt(1),
				gasPool:  gasPool,
				usedGas:  &usedGas,
				upgrades: opera.Upgrades{GasSubsidies: true, Brio: test.brio},
			}

			getPostTxs := func(subsidies.Sponsorship, subsidies.NonceSource, uint64, *big.Int) ([]*types.Transaction, error) {
				return []*types.Transaction{postTx}, nil
			}

			runner := &transactionRunner{evm: mockEvm}
			got, status := runner.runSponsoredTransactionInternal(ctxt, tx, 0, math.MaxUint64, getPostTxs)

			if test.wantRollback {
				require.Equal(t, core_types.TransactionResultInvalid, status)
				require.Equal(t, []ProcessedTransaction{{Transaction: tx}}, got)
				require.Equal(t, initialGasInPool, gasPool.Gas(), "gas pool must be restored")
				require.Equal(t, uint64(50_000), usedGas, "usedGas must be restored")
			} else {
				require.Equal(t, core_types.TransactionResultSuccessful, status)
				require.Equal(t, []ProcessedTransaction{processedSponsored, processedPost}, got)
			}
		})
	}
}

// TestRunSponsoredTransaction_PostTxBuildError_DeterminesRollbackUnderBrio
// covers the [Brio on/off] × build-error matrix: when Brio is enabled and the
// post-transaction cannot be constructed, the sponsored transaction is rolled
// back; pre-Brio the sponsored transaction is kept.
func TestRunSponsoredTransaction_PostTxBuildError_DeterminesRollbackUnderBrio(t *testing.T) {
	tests := map[string]struct {
		brio         bool
		wantRollback bool
	}{
		"brio on":  {brio: true, wantRollback: true},
		"brio off": {brio: false, wantRollback: false},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			stateDb := state.NewMockStateDB(ctrl)
			mockEvm := NewMock_evm(ctrl)

			tx := getSponsorshipRequest(t)
			any := gomock.Any()

			stateDb.EXPECT().Snapshot().Return(1)
			stateDb.EXPECT().RevertToSnapshot(1)
			mockEvm.EXPECT().Call(any, any, any, any, any).Return([]byte{95: 0}, uint64(0), nil)
			mockEvm.EXPECT().Call(any, any, any, any, any).Return([]byte{31: 1}, uint64(0), nil)

			const snapshotId = 42
			if test.brio {
				stateDb.EXPECT().InterTxSnapshot().Return(snapshotId)
				stateDb.EXPECT().RevertToInterTxSnapshot(snapshotId)
			}

			processedSponsored := ProcessedTransaction{
				Transaction: tx,
				Receipt:     &types.Receipt{Status: types.ReceiptStatusSuccessful, GasUsed: 21_000},
			}
			mockEvm.EXPECT().runWithoutBaseFeeCheck(any, tx, any).Return(processedSponsored)

			ctxt := &runContext{
				statedb:  stateDb,
				signer:   types.LatestSignerForChainID(nil),
				baseFee:  big.NewInt(1),
				gasPool:  core.NewGasPool(1_000_000),
				usedGas:  new(uint64),
				upgrades: opera.Upgrades{GasSubsidies: true, Brio: test.brio},
			}

			getPostTxs := func(subsidies.Sponsorship, subsidies.NonceSource, uint64, *big.Int) ([]*types.Transaction, error) {
				return nil, fmt.Errorf("cannot build post-tx")
			}

			runner := &transactionRunner{evm: mockEvm}
			got, status := runner.runSponsoredTransactionInternal(ctxt, tx, 0, math.MaxUint64, getPostTxs)

			if test.wantRollback {
				require.Equal(t, core_types.TransactionResultInvalid, status)
				require.Equal(t, []ProcessedTransaction{{Transaction: tx}}, got)
			} else {
				require.Equal(t, core_types.TransactionResultSuccessful, status)
				require.Equal(t, []ProcessedTransaction{processedSponsored}, got)
			}
		})
	}
}

func TestRunSponsoredTransaction_CoveredTransaction_ProcessesTwoTransactionsSuccessfully(t *testing.T) {
	// This test is an integration test covering the combination of the state
	// processor's runTransaction function, the subsidies package's utility
	// functions and the on-chain subsidies registry and SFC contracts.
	// The aim of this test is to provide a high-level coverage of the
	// interaction of these components, making sure that a sponsored
	// transaction that is covered by a fund is processed successfully,
	// resulting in two successful transactions: the sponsored transaction
	// itself and the subsequent fee deduction transaction.
	// This test is a smoke test for the overall functionality, and does not
	// aim to cover all edge cases or failure scenarios.
	require := require.New(t)
	ctrl := gomock.NewController(t)

	key, err := crypto.GenerateKey()
	require.NoError(err)
	signer := types.LatestSignerForChainID(nil)

	sender := crypto.PubkeyToAddress(key.PublicKey)
	target := common.Address{1, 2, 3}
	tx := types.MustSignNewTx(key, signer, &types.LegacyTx{
		Nonce: 0, To: &target, Gas: 21_000,
	})
	require.True(subsidies.IsSponsorshipRequest(tx))
	txIndex := 12

	// --- prepare state DB interactions ---
	state := state.NewMockStateDB(ctrl)

	any := gomock.Any()
	zeroAddress := common.Address{}
	sfcAddress := sfc.ContractAddress
	sfcCode := sfc.GetContractBin()
	registryAddress := registry.GetAddress()
	registryCodeHash := crypto.Keccak256Hash(registry.GetCode())

	// Define the expected sequence of calls to the StateDB, focusing on the
	// handling of snapshots and state modifications.
	gomock.InOrder(
		// --- The effects of the IsCovered call ---
		state.EXPECT().Snapshot().Return(1), // < added by runSponsoredTransaction
		state.EXPECT().Snapshot().Return(2), // < added for the getGasConfig call by the EVM (not reverted)
		state.EXPECT().Snapshot().Return(3), // < added for the chooseFund call by the EVM (not reverted)
		state.EXPECT().GetCode(registryAddress).Return(registry.GetCode()),
		// the effects of the IsCovered call in runSponsoredTransaction must be
		// reverted to avoid spilling side-effects into the actual transaction
		state.EXPECT().RevertToSnapshot(1),

		// --- The effects of the sponsored transaction itself ---
		state.EXPECT().SetTxContext(tx.Hash(), txIndex),
		state.EXPECT().SetNonce(sender, uint64(1), tracing.NonceChangeEoACall),
		state.EXPECT().Snapshot().Return(4), // < for the transaction processing
		state.EXPECT().EndTransaction(),
		state.EXPECT().TxIndex().Return(txIndex),

		// --- Preparation of the fee deduction transaction ---
		state.EXPECT().GetNonce(zeroAddress).Return(uint64(123)),

		// --- The effects of the fee deduction transaction ---
		state.EXPECT().SetTxContext(any, txIndex+1),
		state.EXPECT().SetNonce(zeroAddress, uint64(124), tracing.NonceChangeEoACall),
		state.EXPECT().Snapshot().Return(5),                           // < for the deductFees call
		state.EXPECT().Snapshot().Return(6),                           // < for the nested burnNativeToken call to SFC
		state.EXPECT().SetState(sfcAddress, any, any).AnyTimes(),      // < update of the total token supply
		state.EXPECT().Snapshot().Return(7),                           // < transfer to account 0
		state.EXPECT().SetState(registryAddress, any, any).AnyTimes(), // < update of the fund
		state.EXPECT().EndTransaction(),
		state.EXPECT().TxIndex().Return(txIndex+1),
	)

	// StateDB interactions that are occurring, and need to be accounted for,
	// but that are not relevant for this test. They basically set the execution
	// environment required for running sponsored transactions.

	state.EXPECT().Exist(zeroAddress).Return(true).AnyTimes()
	state.EXPECT().GetCode(zeroAddress).Return(nil).AnyTimes()
	state.EXPECT().GetNonce(zeroAddress).Return(uint64(123)).AnyTimes()
	state.EXPECT().GetBalance(zeroAddress).Return(uint256.NewInt(1e18)).AnyTimes()

	state.EXPECT().Exist(sfcAddress).Return(true).AnyTimes()
	state.EXPECT().GetCode(sfcAddress).Return(sfcCode).AnyTimes()
	state.EXPECT().GetCodeHash(sfcAddress).Return(crypto.Keccak256Hash(sfcCode)).AnyTimes()
	state.EXPECT().GetCodeSize(sfcAddress).Return(len(sfcCode)).AnyTimes()
	state.EXPECT().GetNonce(sfcAddress).Return(uint64(0)).AnyTimes()
	state.EXPECT().GetBalance(sfcAddress).Return(uint256.NewInt(1e18)).AnyTimes()
	state.EXPECT().GetState(sfcAddress, any).Return(common.Hash{1}).AnyTimes()
	state.EXPECT().GetStateAndCommittedState(sfcAddress, any).Return(common.Hash{1}, common.Hash{1}).AnyTimes()

	state.EXPECT().Exist(registryAddress).Return(true).AnyTimes()
	state.EXPECT().GetCode(registryAddress).Return(registry.GetCode()).AnyTimes()
	state.EXPECT().GetCodeHash(registryAddress).Return(registryCodeHash).AnyTimes()
	state.EXPECT().GetBalance(registryAddress).Return(uint256.NewInt(1e18)).AnyTimes()
	state.EXPECT().GetState(registryAddress, any).Return(common.Hash{1}).AnyTimes()
	state.EXPECT().GetStateAndCommittedState(registryAddress, any).Return(common.Hash{1}, common.Hash{1}).AnyTimes()

	state.EXPECT().Exist(target).Return(false).AnyTimes()
	state.EXPECT().GetCode(target).Return(nil).AnyTimes()

	state.EXPECT().GetNonce(sender).Return(uint64(0)).AnyTimes()
	state.EXPECT().GetCode(sender).Return(nil)
	state.EXPECT().GetBalance(sender).Return(uint256.NewInt(1_000_000))

	// the actual balance changes are not relevant for this test
	state.EXPECT().AddBalance(any, any, any).AnyTimes()
	state.EXPECT().SubBalance(any, any, any).AnyTimes()

	state.EXPECT().AddRefund(any).AnyTimes()
	state.EXPECT().GetRefund().Return(uint64(0)).AnyTimes()
	state.EXPECT().SubRefund(any).AnyTimes()

	state.EXPECT().Prepare(any, any, any, any, any, any).AnyTimes()
	state.EXPECT().AddressInAccessList(any).Return(true).AnyTimes()
	state.EXPECT().SlotInAccessList(any, any).Return(true, true).AnyTimes()
	state.EXPECT().AddAddressToAccessList(any).AnyTimes()
	state.EXPECT().AddSlotToAccessList(any, any).AnyTimes()

	// also logs are not relevant for this test
	state.EXPECT().AddLog(any).AnyTimes()
	state.EXPECT().GetLogs(any, any).Return(nil).AnyTimes()

	// --- Create an EVM instance capable of processing code ---

	rules := opera.FakeNetRules(opera.GetSonicUpgrades())
	vmConfig := opera.GetVmConfig(rules)

	var updateHeights []opera.UpgradeHeight
	chainConfig := opera.CreateTransientEvmChainConfig(
		rules.NetworkID,
		updateHeights,
		1,
	)

	baseFee := big.NewInt(1)
	blockContext := vm.BlockContext{
		BlockNumber: big.NewInt(123),
		BaseFee:     baseFee,
		Transfer: func(_ vm.StateDB, _ common.Address, _ common.Address, amount *uint256.Int, _ *params.Rules) {
			// do nothing
		},
		CanTransfer: func(_ vm.StateDB, _ common.Address, amount *uint256.Int) bool {
			return true
		},
		Random: &common.Hash{}, // < signals Revision >= Merge
	}
	vm := vm.NewEVM(blockContext, state, chainConfig, vmConfig)
	runner := &transactionRunner{evm{vm}}

	gasPool := core.NewGasPool(1_000_000)
	usedGas := new(uint64)
	context := &runContext{
		signer:   signer,
		baseFee:  baseFee,
		statedb:  state,
		gasPool:  gasPool,
		usedGas:  usedGas,
		runner:   runner,
		upgrades: opera.Upgrades{GasSubsidies: true},
	}

	// --- start of actual test ---

	processedTransactions, status := runner.runSponsoredTransaction(context, tx, txIndex, math.MaxUint64)
	require.Equal(core_types.TransactionResultSuccessful, status)

	// the transaction should be sponsored successfully
	require.Len(processedTransactions, 2)
	require.Equal(tx, processedTransactions[0].Transaction)
	require.NotNil(processedTransactions[0].Receipt)

	// the fee deduction transaction should be the second one
	require.NotNil(processedTransactions[1].Transaction)
	callData := processedTransactions[1].Transaction.Data()
	require.Equal(4+2*32, len(callData)) // chooseFund + deductFees

	fundId := subsidies.Identifier(callData[4:])
	gasUsed := processedTransactions[0].Receipt.GasUsed
	gasPrice := baseFee // gas price is base fee for sponsored tx
	paymentTx := subsidies.NewPostTxBuilder().
		WithNonce(123).
		WithId(fundId).
		WithOverhead(210_000). // < hard-coded in dev version of registry
		WithGasLimit(60_000).  // < hard-coded in dev version of registry
		WithUsedGas(gasUsed).
		WithGasPrice(gasPrice).
		BuildForTesting()

	got := processedTransactions[1].Transaction
	require.Equal(paymentTx.Hash(), got.Hash())
	require.NotNil(processedTransactions[1].Receipt)
	require.Equal(types.ReceiptStatusSuccessful, processedTransactions[1].Receipt.Status)

	// the total gas usage should be the sum of both transactions
	require.Equal(
		processedTransactions[0].Receipt.GasUsed+processedTransactions[1].Receipt.GasUsed,
		*usedGas,
	)
}

func TestRunSponsoredTransaction_MatchesCoveredAndReceiptToStatus(t *testing.T) {
	tests := map[string]struct {
		subsidyRequestFail bool
		covered            bool
		poolNotEnough      bool
		receipt            *types.Receipt
		status             core_types.TransactionResult
	}{
		"subsidy request fail": {
			subsidyRequestFail: true,
			status:             core_types.TransactionResultInvalid,
		},
		"not covered": {
			covered: false,
			status:  core_types.TransactionResultInvalid,
		},
		"pool not enough": {
			covered:       true,
			poolNotEnough: true,
			status:        core_types.TransactionResultInvalid,
		},
		"sponsored tx nil receipt": {
			covered: true,
			receipt: nil,
			status:  core_types.TransactionResultInvalid,
		},
		"sponsored tx failed": {
			covered: true,
			receipt: &types.Receipt{Status: types.ReceiptStatusFailed},
			status:  core_types.TransactionResultFailed,
		},
		"sponsored tx success": {
			covered: true,
			receipt: &types.Receipt{Status: types.ReceiptStatusSuccessful},
			status:  core_types.TransactionResultSuccessful,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			state := state.NewMockStateDB(ctrl)
			evm := NewMock_evm(ctrl)

			var tx *types.Transaction
			if test.subsidyRequestFail {
				tx = getSponsorshipRequestUnsigned(t)
			} else {
				tx = getSponsorshipRequest(t)
			}

			var gasPool *core.GasPool
			if test.poolNotEnough {
				gasPool = core.NewGasPool(0)
			} else {
				gasPool = core.NewGasPool(1_000_000)
			}
			ctxt := &runContext{
				statedb:  state,
				signer:   types.LatestSignerForChainID(nil),
				baseFee:  big.NewInt(1),
				gasPool:  gasPool,
				upgrades: opera.Upgrades{GasSubsidies: true},
			}

			// Snapshot for the IsCovered call
			state.EXPECT().Snapshot().Return(1)
			state.EXPECT().RevertToSnapshot(1)

			txIndex := 7
			if !test.subsidyRequestFail {
				any := gomock.Any()
				evm.EXPECT().Call(any, any, any, any, any).
					Return([]byte{95: 0}, uint64(0), nil) // results of getGasConfig
				if test.covered {
					evm.EXPECT().Call(any, any, any, any, any).
						Return([]byte{31: 1}, uint64(0), nil) // indicates "covered"
				} else {
					evm.EXPECT().Call(any, any, any, any, any).
						Return([]byte{}, uint64(0), nil) // indicates "not covered"
				}

				if test.covered && !test.poolNotEnough {
					evm.EXPECT().runWithoutBaseFeeCheck(ctxt, tx, txIndex).
						Return(ProcessedTransaction{
							Transaction: tx,
							Receipt:     test.receipt,
						})

					if test.receipt != nil {
						// Nonce request for the fee deduction transaction
						state.EXPECT().GetNonce(common.Address{}).Return(uint64(123))

						evm.EXPECT().runWithoutBaseFeeCheck(ctxt, gomock.Not(tx), txIndex+1).
							Return(ProcessedTransaction{
								Transaction: tx,
								Receipt:     &types.Receipt{},
							})
					}
				}
			}

			runner := transactionRunner{evm}
			_, status := runner.runSponsoredTransaction(ctxt, tx, txIndex, math.MaxUint64)
			require.Equal(t, test.status, status)
		})
	}
}

func TestRunTransactionBundle_BundlesDisabled_ReturnsEnvelopeAndResultInvalid(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := NewMocklogger(ctrl)
	log.EXPECT().Warn("Transaction bundles are not enabled, bundle transaction skipped", gomock.Any())

	tx := types.NewTx(&types.LegacyTx{})

	context := &runContext{
		upgrades: opera.Upgrades{TransactionBundles: false},
	}

	runner := &transactionRunner{}

	processedTransactions, result, execCost := runner.runTransactionBundleInternal(context, tx, 0, log, math.MaxUint64)
	require.Len(t, processedTransactions, 1)
	require.Equal(t, tx, processedTransactions[0].Transaction)
	require.Nil(t, processedTransactions[0].Receipt)
	require.Equal(t, core_types.TransactionResultInvalid, result)
	require.Zero(t, execCost)
}

func TestRunTransactionBundle_InvalidEnvelope_ReturnsEnvelopeAndResultInvalid(t *testing.T) {
	ctrl := gomock.NewController(t)
	log := NewMocklogger(ctrl)
	log.EXPECT().Warn("Invalid bundle skipped", "tx", gomock.Any(), "err", gomock.Any())

	tx := types.NewTx(&types.LegacyTx{})
	require.False(t, bundle.IsEnvelope(tx))

	context := &runContext{
		signer:   types.LatestSignerForChainID(big.NewInt(1)),
		upgrades: opera.Upgrades{TransactionBundles: true},
	}

	runner := &transactionRunner{}

	processedTransactions, result, execCost := runner.runTransactionBundleInternal(context, tx, 0, log, math.MaxUint64)
	require.Len(t, processedTransactions, 1)
	require.Equal(t, tx, processedTransactions[0].Transaction)
	require.Nil(t, processedTransactions[0].Receipt)
	require.Equal(t, core_types.TransactionResultInvalid, result)
	require.Zero(t, execCost)
}

func TestRunTransactionBundle_BundleOutOfRange_ReturnsEnvelopeAndResultInvalid(t *testing.T) {
	signer := types.LatestSignerForChainID(big.NewInt(1))

	ctrl := gomock.NewController(t)

	log := NewMocklogger(ctrl)
	log.EXPECT().Warn("Bundle skipped due to out-of-range execution plan", gomock.Any())

	blockNumber := 10

	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	tx := bundle.NewBuilder().
		With(bundle.Step(key, &types.AccessListTx{
			Nonce: 0, To: &common.Address{1}, Gas: 21_000, GasPrice: big.NewInt(1),
		})).
		SetEarliest(11). // not ready for execution yet
		SetRangeLength(2).
		Build()

	_, _, err = bundle.ValidateEnvelope(signer, tx)
	require.NoError(t, err)

	context := &runContext{
		signer:      signer,
		upgrades:    opera.Upgrades{TransactionBundles: true},
		blockNumber: big.NewInt(int64(blockNumber)),
	}

	runner := &transactionRunner{}

	processedTransactions, result, execCost := runner.runTransactionBundleInternal(context, tx, 0, log, math.MaxUint64)
	require.Len(t, processedTransactions, 1)
	require.Equal(t, tx, processedTransactions[0].Transaction)
	require.Nil(t, processedTransactions[0].Receipt)
	require.Equal(t, core_types.TransactionResultInvalid, result)
	require.Zero(t, execCost)
}

func TestRunTransactionBundle_BundleOutOfTime_ReturnsEnvelopeAndResultInvalid(t *testing.T) {
	signer := types.LatestSignerForChainID(big.NewInt(1))

	ctrl := gomock.NewController(t)

	log := NewMocklogger(ctrl)
	log.EXPECT().Warn("Bundle skipped due to out-of-time execution plan", gomock.Any())

	blockNumber := 10
	blockTime := inter.Timestamp(1234)

	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	tx := bundle.NewBuilder().
		With(bundle.Step(key, &types.AccessListTx{
			Nonce: 0, To: &common.Address{1}, Gas: 21_000, GasPrice: big.NewInt(1),
		})).
		SetNotBefore(blockTime + 1).
		Build()

	_, _, err = bundle.ValidateEnvelope(signer, tx)
	require.NoError(t, err)

	context := &runContext{
		signer:      signer,
		upgrades:    opera.Upgrades{TransactionBundles: true},
		blockNumber: big.NewInt(int64(blockNumber)),
		blockTime:   blockTime,
	}

	runner := &transactionRunner{}

	processedTransactions, result, execCost := runner.runTransactionBundleInternal(context, tx, 0, log, math.MaxUint64)
	require.Len(t, processedTransactions, 1)
	require.Equal(t, tx, processedTransactions[0].Transaction)
	require.Nil(t, processedTransactions[0].Receipt)
	require.Equal(t, core_types.TransactionResultInvalid, result)
	require.Zero(t, execCost)
}

func TestRunTransactionBundle_PreviouslyProcessedBundle_ReturnsEnvelopeAndResultInvalid(t *testing.T) {
	signer := types.LatestSignerForChainID(big.NewInt(1))

	ctrl := gomock.NewController(t)

	tx, plan := bundle.NewBuilder().BuildEnvelopeAndPlan()
	_, _, err := bundle.ValidateEnvelope(signer, tx)
	require.NoError(t, err)

	log := NewMocklogger(ctrl)
	log.EXPECT().Warn("Rescheduled bundle skipped", "exec_plan_hash", plan.Hash())

	state := state.NewMockStateDB(ctrl)
	state.EXPECT().HasBundleRecentlyBeenProcessed(plan.Hash()).Return(true)

	context := &runContext{
		signer:      signer,
		statedb:     state,
		upgrades:    opera.Upgrades{TransactionBundles: true},
		blockNumber: big.NewInt(0),
	}

	runner := &transactionRunner{}

	processedTransactions, result, execCost := runner.runTransactionBundleInternal(context, tx, 0, log, math.MaxUint64)
	require.Len(t, processedTransactions, 1)
	require.Equal(t, tx, processedTransactions[0].Transaction)
	require.Nil(t, processedTransactions[0].Receipt)
	require.Equal(t, core_types.TransactionResultInvalid, result)
	require.Zero(t, execCost)
}

func TestRunTransactionBundle_RunBundleNotSuccessful_ReturnsNoTransactionAndResultFailed_AndMarksBundleAsProcessed(t *testing.T) {
	signer := types.LatestSignerForChainID(big.NewInt(1))
	ctrl := gomock.NewController(t)
	state := state.NewMockStateDB(ctrl)
	evm := NewMock_evm(ctrl)

	txOffset := 12

	tx := bundle.OneOf().Build() // an empty bundle with OneOf flag will fail
	_, plan, err := bundle.ValidateEnvelope(signer, tx)
	require.NoError(t, err)

	gomock.InOrder(
		state.EXPECT().HasBundleRecentlyBeenProcessed(plan.Hash()),
		state.EXPECT().InterTxSnapshot().Return(1),
		state.EXPECT().RevertToInterTxSnapshot(1),
		state.EXPECT().AddProcessedBundle(plan.Hash(), bundle.PositionInBlock{
			Offset: uint32(txOffset),
			Count:  0,
		}),
	)

	gasPool := core.NewGasPool(1_000_000)
	context := &runContext{
		statedb:     state,
		signer:      signer,
		baseFee:     big.NewInt(1),
		usedGas:     new(uint64),
		gasPool:     gasPool,
		upgrades:    opera.Upgrades{TransactionBundles: true},
		blockNumber: big.NewInt(0),
	}

	runner := &transactionRunner{evm: evm}

	processedTransactions, result, execCost := runner.runTransactionBundle(context, tx, txOffset, math.MaxUint64)
	require.Len(t, processedTransactions, 0)
	require.Equal(t, core_types.TransactionResultFailed, result)
	require.Zero(t, execCost)
}

func TestRunTransactionBundle_RunBundleSuccessful_ReturnsBundleOnlyTransactionAndResultSuccessful_AndMarksBundleAsProcessed(t *testing.T) {
	ctrl := gomock.NewController(t)
	state := state.NewMockStateDB(ctrl)
	evm := NewMock_evm(ctrl)

	signer := types.LatestSignerForChainID(big.NewInt(1))
	trueTxOffset := 14

	envelope := getTransactionBundle(t)
	txBundle, err := bundle.OpenEnvelope(signer, envelope)
	require.NoError(t, err)
	_, plan, err := bundle.ValidateEnvelope(signer, envelope)
	require.NoError(t, err)

	gomock.InOrder(
		state.EXPECT().HasBundleRecentlyBeenProcessed(plan.Hash()),
		state.EXPECT().InterTxSnapshot().Return(0), // < snapshot for size check
		state.EXPECT().InterTxSnapshot().Return(1), // < snapshot for execution
		state.EXPECT().AddProcessedBundle(plan.Hash(), bundle.PositionInBlock{
			Offset: uint32(trueTxOffset),
			Count:  1,
		}),
	)

	runner := &transactionRunner{evm: evm}

	context := &runContext{
		statedb:     state,
		signer:      signer,
		baseFee:     big.NewInt(1),
		usedGas:     new(uint64),
		gasPool:     core.NewGasPool(1_000_000),
		upgrades:    opera.Upgrades{TransactionBundles: true},
		blockNumber: big.NewInt(0),
		runner:      runner,
	}

	txs := txBundle.GetTransactionsInReferencedOrder()

	evm.EXPECT().runWithBaseFeeCheck(context, gomock.Any(), trueTxOffset).
		Return(ProcessedTransaction{
			Transaction: txs[0],
			Receipt:     &types.Receipt{Status: types.ReceiptStatusSuccessful, GasUsed: txs[0].Gas()},
		})

	processedTransactions, result, execCost := runner.runTransactionBundle(context, envelope, trueTxOffset, math.MaxUint64)
	require.Len(t, processedTransactions, 1)
	require.Equal(t, txs[0], processedTransactions[0].Transaction)
	require.NotNil(t, processedTransactions[0].Receipt)
	require.Equal(t, core_types.TransactionResultSuccessful, result)
	require.EqualValues(t, txs[0].Gas(), execCost)
}

func TestRunTransactionBundle_RunBundleSuccessful_ReportsCorrectOffsetAndCountToStateDB(t *testing.T) {
	tests := map[string]struct {
		offset      uint32
		validTxs    []bool
		wantedCount uint32
	}{
		"empty result": {
			offset:      12,
			validTxs:    []bool{},
			wantedCount: 0,
		},
		"one successful transaction": {
			offset:      5,
			validTxs:    []bool{true},
			wantedCount: 1,
		},
		"one invalid transaction": {
			offset:      8,
			validTxs:    []bool{false},
			wantedCount: 0,
		},
		"multiple transactions with mixed results": {
			offset:      3,
			validTxs:    []bool{true, false, true, true, false},
			wantedCount: 3,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			// The main goal of this test is to verify that the
			// runTransactionBundle function reports the correct number of
			// transactions accepted in the block. Transactions are accepted if
			// they are listed in the processed transactions with a non-nil
			// receipt.
			//
			// To simulate the execution of bundles with different number of
			// accepted transactions, the following bundle blueprint is used:
			//
			//      envelope = AllOf(A)
			//
			// where A is a transaction requesting sponsorship. The sponsored
			// inner transaction is needed since regular transactions are
			// restricted to only result into a single processed transaction.
			// Sponsored transactions, however, result in multiple transactions
			// (typically two), but by mocking their execution an arbitrary
			// list of transactions can be returned.
			//
			// Thus, to simulate the test cases defined above, the envelope with
			// a single sponsored transaction is used, to produce the list of
			// expected processed transactions for the test setup.

			ctrl := gomock.NewController(t)

			signer := types.LatestSignerForChainID(big.NewInt(1))
			trueTxOffset := test.offset

			key, err := crypto.GenerateKey()
			require.NoError(t, err)
			envelope := bundle.AllOf(
				bundle.Step(key, &types.AccessListTx{ // < sponsorship request
					To: &common.Address{1},
				}),
			).Build()

			state := state.NewMockStateDB(ctrl)
			gomock.InOrder(
				state.EXPECT().HasBundleRecentlyBeenProcessed(gomock.Any()),
				state.EXPECT().InterTxSnapshot().Return(0),
				state.EXPECT().InterTxSnapshot().Return(1),
				state.EXPECT().AddProcessedBundle(gomock.Any(), bundle.PositionInBlock{
					Offset: test.offset, // < these two fields are the main test targets
					Count:  test.wantedCount,
				}),
			)

			// Here we build the list of processed transactions resulting from
			// the sponsored transaction execution step in the bundle.
			var execResult []ProcessedTransaction
			for _, accepted := range test.validTxs {
				var receipt *types.Receipt
				if accepted {
					receipt = &types.Receipt{GasUsed: 1}
				}
				execResult = append(execResult, ProcessedTransaction{
					Transaction: types.NewTx(&types.LegacyTx{}),
					Receipt:     receipt,
				})
			}

			innerRunner := NewMock_transactionRunner(ctrl)
			innerRunner.EXPECT().runSponsoredTransaction(gomock.Any(), gomock.Any(), int(trueTxOffset), uint64(math.MaxUint64)).Return(
				execResult, core_types.TransactionResultSuccessful,
			)

			context := &runContext{
				statedb: state,
				signer:  signer,
				usedGas: new(uint64),
				gasPool: core.NewGasPool(1_000_000),
				upgrades: opera.Upgrades{
					GasSubsidies:       true,
					TransactionBundles: true,
				},
				blockNumber: big.NewInt(0),
				runner:      innerRunner,
			}

			runner := &transactionRunner{}
			processedTransactions, result, execCost := runner.runTransactionBundle(context, envelope, int(trueTxOffset), math.MaxUint64)
			require.Equal(t, execResult, processedTransactions)
			require.Equal(t, core_types.TransactionResultSuccessful, result)
			execCostExpected := uint64(0)
			for _, accepted := range test.validTxs {
				if accepted {
					execCostExpected += 1
				}
			}
			require.Equal(t, core_types.ExecutionCost(execCostExpected), execCost)
		})
	}
}

func TestRunRegularTransaction_InternalTransactions_SkipsTransactionChecksTrue(t *testing.T) {

	maxTxGas := uint64(1_500_000)

	// -- setup --
	ctrl := gomock.NewController(t)
	state := state.NewMockStateDB(ctrl)
	any := gomock.Any()
	state.EXPECT().SetTxContext(any, any).Times(2)
	state.EXPECT().GetBalance(any).Return(uint256.NewInt(math.MaxInt64))
	state.EXPECT().EndTransaction().Times(2)
	state.EXPECT().SubBalance(any, any, any)
	state.EXPECT().Prepare(any, any, any, any, any, any)
	state.EXPECT().GetNonce(any).Return(uint64(0)).Times(2)
	state.EXPECT().SetNonce(any, any, any)
	state.EXPECT().GetCode(any).Return([]byte{}).AnyTimes()
	state.EXPECT().Snapshot().Return(1)
	state.EXPECT().Exist(any).Return(true)
	state.EXPECT().GetRefund().Return(uint64(0)).Times(2)
	state.EXPECT().AddBalance(any, any, any)
	state.EXPECT().GetLogs(any, any)
	state.EXPECT().TxIndex().Return(0)
	rules := opera.FakeNetRules(opera.GetBrioUpgrades())
	rules.Economy.Gas.MaxEventGas = maxTxGas
	vmConfig := opera.GetVmConfig(rules)
	updateHeights := []opera.UpgradeHeight{
		{Upgrades: rules.Upgrades, Height: 0, Time: 0},
	}
	chainConfig := opera.CreateTransientEvmChainConfig(
		rules.NetworkID,
		updateHeights,
		1,
	)
	baseFee := big.NewInt(1)
	blockContext := vm.BlockContext{
		BlockNumber: big.NewInt(123),
		BaseFee:     baseFee,
		Transfer: func(_ vm.StateDB, _ common.Address, _ common.Address, amount *uint256.Int, _ *params.Rules) {
			// do nothing
		},
		CanTransfer: func(_ vm.StateDB, _ common.Address, amount *uint256.Int) bool {
			return true
		},
		Random: &common.Hash{}, // < signals Revision >= Merge
	}

	vm := vm.NewEVM(blockContext, state, chainConfig, vmConfig)
	runner := &transactionRunner{evm{vm}}
	// enough max gas per block to accommodate for the internal transaction.
	gasPool := core.NewGasPool(maxTxGas * 3)
	usedGas := new(uint64)
	context := &runContext{
		signer:   types.LatestSignerForChainID(nil),
		baseFee:  baseFee,
		statedb:  state,
		gasPool:  gasPool,
		usedGas:  usedGas,
		runner:   runner,
		upgrades: opera.Upgrades{Brio: true},
	}

	// -- end of setup --

	unsignedTx := types.NewTx(&types.LegacyTx{
		Nonce: 0, To: &common.Address{1}, Gas: maxTxGas * 2, GasPrice: big.NewInt(1),
	})
	require.True(t, internaltx.IsInternal(unsignedTx))

	// run an internal transaction with gas over the max tx gas limit.
	got, status := runner.runRegularTransaction(context, unsignedTx, 0, math.MaxUint64)
	require.Equal(t, core_types.TransactionResultSuccessful, status)

	require.Equal(t, unsignedTx, got.Transaction)
	require.NotNil(t, got.Receipt)
	require.Equal(t, types.ReceiptStatusSuccessful, got.Receipt.Status)

	// non internal transaction with the same gas limit is rejected.
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	signer := types.LatestSignerForChainID(nil)
	regularTx := types.MustSignNewTx(key, signer, &types.LegacyTx{
		Nonce: 0, To: &common.Address{1}, Gas: maxTxGas * 2, GasPrice: big.NewInt(1),
	})
	got, status = runner.runRegularTransaction(context, regularTx, 0, math.MaxUint64)
	require.Equal(t, core_types.TransactionResultInvalid, status)
	require.Equal(t, regularTx, got.Transaction)
	require.Nil(t, got.Receipt)
}

func TestRunRegularTransaction_RegularTransaction(t *testing.T) {

	tests := map[string]struct {
		rules      opera.Rules
		stateSetup func(state *state.MockStateDB)
		validation func(t *testing.T, got ProcessedTransaction)
		status     core_types.TransactionResult
	}{
		"Brio/Skipped": {
			rules: opera.FakeNetRules(opera.GetBrioUpgrades()),
			stateSetup: func(state *state.MockStateDB) {
				any := gomock.Any()
				state.EXPECT().SetTxContext(any, any)
				state.EXPECT().EndTransaction()
				state.EXPECT().GetNonce(any).Return(uint64(0)).Times(1)
			},
			validation: func(t *testing.T, got ProcessedTransaction) {
				require.Nil(t, got.Receipt, "expected no receipt for transaction with too high gas")
			},
			status: core_types.TransactionResultInvalid,
		},
		"Pre-Brio/Accepted": {
			rules: opera.FakeNetRules(opera.GetAllegroUpgrades()),
			stateSetup: func(state *state.MockStateDB) {
				any := gomock.Any()
				state.EXPECT().SetTxContext(any, any)
				state.EXPECT().GetBalance(any).Return(uint256.NewInt(math.MaxInt64))
				state.EXPECT().EndTransaction()
				state.EXPECT().SubBalance(any, any, any)
				state.EXPECT().Prepare(any, any, any, any, any, any)
				state.EXPECT().GetNonce(any).Return(uint64(0)).AnyTimes()
				state.EXPECT().SetNonce(any, any, any)
				state.EXPECT().GetCode(any).Return([]byte{}).AnyTimes()
				state.EXPECT().Snapshot().Return(1)
				state.EXPECT().Exist(any).Return(true)
				state.EXPECT().GetRefund().Return(uint64(0)).Times(2)
				state.EXPECT().AddBalance(any, any, any)
				state.EXPECT().GetLogs(any, any)
				state.EXPECT().TxIndex().Return(0)
			},
			validation: func(t *testing.T, got ProcessedTransaction) {
				require.NotNil(t, got.Receipt, "expected receipt for accepted transaction")
				require.Equal(t, types.ReceiptStatusSuccessful, got.Receipt.Status, "expected successful transaction")
			},
			status: core_types.TransactionResultSuccessful,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			maxTxGas := uint64(1_500_000)
			rules := test.rules
			rules.Economy.Gas.MaxEventGas = maxTxGas

			// -- setup --
			ctrl := gomock.NewController(t)
			state := state.NewMockStateDB(ctrl)

			test.stateSetup(state)

			vmConfig := opera.GetVmConfig(rules)
			updateHeights := []opera.UpgradeHeight{
				{Upgrades: rules.Upgrades, Height: 0, Time: 0},
			}
			chainConfig := opera.CreateTransientEvmChainConfig(
				rules.NetworkID,
				updateHeights,
				1,
			)
			baseFee := big.NewInt(1)
			blockContext := vm.BlockContext{
				BlockNumber: big.NewInt(123),
				BaseFee:     baseFee,
				Transfer: func(_ vm.StateDB, _ common.Address, _ common.Address, amount *uint256.Int, _ *params.Rules) {
					// do nothing
				},
				CanTransfer: func(_ vm.StateDB, _ common.Address, amount *uint256.Int) bool {
					return true
				},
				Random: &common.Hash{}, // < signals Revision >= Merge
			}

			vm := vm.NewEVM(blockContext, state, chainConfig, vmConfig)
			runner := &transactionRunner{evm{vm}}
			// enough max gas per block to accommodate for the internal transaction.
			gasPool := core.NewGasPool(maxTxGas * 3)
			usedGas := new(uint64)
			context := &runContext{
				signer:   types.LatestSignerForChainID(nil),
				baseFee:  baseFee,
				statedb:  state,
				gasPool:  gasPool,
				usedGas:  usedGas,
				runner:   runner,
				upgrades: opera.Upgrades{Brio: true},
			}

			// -- end of setup --

			key, err := crypto.GenerateKey()
			require.NoError(t, err)
			signer := types.LatestSignerForChainID(nil)
			regularTx := types.MustSignNewTx(key, signer, &types.LegacyTx{
				Nonce: 0, To: &common.Address{1}, Gas: maxTxGas + 1, GasPrice: big.NewInt(1),
			})

			got, status := runner.runRegularTransaction(context, regularTx, 0, math.MaxUint64)
			require.Equal(t, test.status, status)

			require.Equal(t, regularTx, got.Transaction)
			test.validation(t, got)
		})
	}
}

func TestRunRegularTransaction_MatchesReceiptToStatus(t *testing.T) {
	tests := map[string]struct {
		receipt *types.Receipt
		status  core_types.TransactionResult
	}{
		"nil receipt": {
			receipt: nil,
			status:  core_types.TransactionResultInvalid,
		},
		"failed": {
			receipt: &types.Receipt{Status: types.ReceiptStatusFailed},
			status:  core_types.TransactionResultFailed,
		},
		"success": {
			receipt: &types.Receipt{Status: types.ReceiptStatusSuccessful},
			status:  core_types.TransactionResultSuccessful,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			evm := NewMock_evm(ctrl)

			ctxt := &runContext{}
			tx := types.NewTx(&types.LegacyTx{})

			evm.EXPECT().runWithBaseFeeCheck(ctxt, tx, 12).Return(
				ProcessedTransaction{
					Transaction: tx,
					Receipt:     test.receipt,
				},
			)

			runner := transactionRunner{evm}
			_, status := runner.runRegularTransaction(ctxt, tx, 12, math.MaxUint64)
			require.Equal(t, test.status, status)
		})
	}
}

func TestBundleTransactionRunner_Run_KeepsTrackOfProcessedTransactions(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := NewMock_transactionRunner(ctrl)

	stateDb := state.NewMockStateDB(ctrl)
	stateDb.EXPECT().InterTxSnapshot().Return(1).AnyTimes()
	stateDb.EXPECT().RevertToInterTxSnapshot(1).AnyTimes()

	ctxt := &runContext{
		runner:   runner,
		upgrades: opera.Upgrades{GasSubsidies: true},
		gasPool:  core.NewGasPool(math.MaxUint64),
		usedGas:  new(uint64),
		statedb:  stateDb,
	}
	sizeLimit := uint64(math.MaxUint64)
	bundleTransactionRunner := &bundleTransactionRunner{ctxt: ctxt, sizeLimit: sizeLimit}

	tx1 := getRegularTransaction(t)
	tx2 := getSponsorshipRequest(t)
	tx2payment := getSponsorshipRequest(t) // some dummy tx
	tx3 := getRegularTransaction(t)

	runner.EXPECT().runRegularTransaction(ctxt, tx1, gomock.Any(), sizeLimit).
		Return(ProcessedTransaction{Transaction: tx1}, core_types.TransactionResultSuccessful)
	runner.EXPECT().runSponsoredTransaction(ctxt, tx2, gomock.Any(), sizeLimit).
		Return([]ProcessedTransaction{{Transaction: tx2}, {Transaction: tx2payment}}, core_types.TransactionResultFailed)
	runner.EXPECT().runRegularTransaction(ctxt, tx3, gomock.Any(), sizeLimit).
		Return(ProcessedTransaction{Transaction: tx3}, core_types.TransactionResultInvalid)

	// one processed transaction with status successful
	bundleTransactionRunner.Run(tx1)
	require.Len(t, bundleTransactionRunner.processedTransactions, 1)
	require.Equal(t, tx1.Hash(), bundleTransactionRunner.processedTransactions[0].Transaction.Hash())

	// two processed transactions with status failed
	bundleTransactionRunner.Run(tx2)
	require.Len(t, bundleTransactionRunner.processedTransactions, 3)
	require.Equal(t, tx1.Hash(), bundleTransactionRunner.processedTransactions[0].Transaction.Hash())
	require.Equal(t, tx2.Hash(), bundleTransactionRunner.processedTransactions[1].Transaction.Hash())
	require.Equal(t, tx2payment.Hash(), bundleTransactionRunner.processedTransactions[2].Transaction.Hash())

	// one processed transaction with status invalid
	bundleTransactionRunner.Run(tx3)
	require.Len(t, bundleTransactionRunner.processedTransactions, 4)
	require.Equal(t, tx1.Hash(), bundleTransactionRunner.processedTransactions[0].Transaction.Hash())
	require.Equal(t, tx2.Hash(), bundleTransactionRunner.processedTransactions[1].Transaction.Hash())
	require.Equal(t, tx2payment.Hash(), bundleTransactionRunner.processedTransactions[2].Transaction.Hash())
	require.Equal(t, tx3.Hash(), bundleTransactionRunner.processedTransactions[3].Transaction.Hash())
}

func TestBundleTransactionRunner_Run_IncrementsTrueOffsetBasedOnProcessedTransactions(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := NewMock_transactionRunner(ctrl)

	stateDb := state.NewMockStateDB(ctrl)
	stateDb.EXPECT().InterTxSnapshot().Return(1).AnyTimes()
	stateDb.EXPECT().RevertToInterTxSnapshot(1).AnyTimes()

	trueStartOffset := 50

	ctxt := &runContext{
		runner:   runner,
		upgrades: opera.Upgrades{GasSubsidies: true},
		gasPool:  core.NewGasPool(math.MaxUint64),
		usedGas:  new(uint64),
		statedb:  stateDb,
	}
	bundleTransactionRunner := &bundleTransactionRunner{
		ctxt:         ctxt,
		trueTxOffset: trueStartOffset,
		sizeLimit:    math.MaxUint64,
	}

	tx1 := getRegularTransaction(t)
	tx2 := getSponsorshipRequest(t)
	tx3 := getSponsorshipRequest(t)
	tx4 := getRegularTransaction(t)

	runner.EXPECT().runRegularTransaction(ctxt, tx1, gomock.Any(), gomock.Any()).
		Return(
			ProcessedTransaction{Transaction: tx1, Receipt: &types.Receipt{}},
			core_types.TransactionResultSuccessful,
		)
	runner.EXPECT().runSponsoredTransaction(ctxt, tx2, gomock.Any(), gomock.Any()).
		Return(
			[]ProcessedTransaction{
				{Transaction: tx2, Receipt: &types.Receipt{}},
				{Transaction: tx2, Receipt: &types.Receipt{}},
			},
			core_types.TransactionResultFailed,
		)
	runner.EXPECT().runSponsoredTransaction(ctxt, tx3, gomock.Any(), gomock.Any()).
		Return(
			[]ProcessedTransaction{
				{Transaction: tx3, Receipt: &types.Receipt{}},
				{Transaction: tx3, Receipt: nil},
			},
			core_types.TransactionResultInvalid,
		)
	runner.EXPECT().runRegularTransaction(ctxt, tx4, gomock.Any(), gomock.Any()).
		Return(
			ProcessedTransaction{Transaction: tx4, Receipt: nil},
			core_types.TransactionResultSuccessful,
		)

	// one processed transaction with non-nil receipt:
	//  - offset should increment by 1
	bundleTransactionRunner.Run(tx1)
	require.Equal(t, bundleTransactionRunner.trueTxOffset, trueStartOffset+1)

	// two processed transactions with non-nil receipts:
	//  - offset should increment by 2
	bundleTransactionRunner.Run(tx2)
	require.Equal(t, bundleTransactionRunner.trueTxOffset, trueStartOffset+1+2)

	// two processed transactions, one with nil receipt and one with non-nil receipt:
	//  - offset should only increment by 1
	bundleTransactionRunner.Run(tx3)
	require.Equal(t, bundleTransactionRunner.trueTxOffset, trueStartOffset+1+2+1)

	// one processed transaction with nil receipt:
	//  - offset should not increment
	bundleTransactionRunner.Run(tx4)
	require.Equal(t, bundleTransactionRunner.trueTxOffset, trueStartOffset+1+2+1+0)
}

func TestBundleTransactionRunner_CreateSnapshot_CallsInterTxSnapshotOnStateDbAndRecordsLocalState(t *testing.T) {
	ctrl := gomock.NewController(t)
	state := state.NewMockStateDB(ctrl)

	state.EXPECT().InterTxSnapshot().Return(123)

	gasPool := core.NewGasPool(10)
	usedGas := uint64(11)
	ctxt := &runContext{statedb: state, usedGas: &usedGas, gasPool: gasPool}
	bundleTransactionRunner := &bundleTransactionRunner{
		ctxt:                  ctxt,
		trueTxOffset:          13,
		processedTransactions: make([]ProcessedTransaction, 14),
		sizeLimit:             15,
	}

	snapshotId := bundleTransactionRunner.CreateSnapshot()
	require.Equal(t, 0, snapshotId)
	require.Len(t, bundleTransactionRunner.snapshots, 1)
	require.Equal(t, 123, bundleTransactionRunner.snapshots[0].stateDbSnapshot)
	require.EqualValues(t, 10, bundleTransactionRunner.snapshots[0].gasPool.Gas())
	require.EqualValues(t, 11, bundleTransactionRunner.snapshots[0].usedGas)
	require.Equal(t, 13, bundleTransactionRunner.snapshots[0].trueTxOffset)
	require.Equal(t, 14, bundleTransactionRunner.snapshots[0].processedTransactionListLength)
	require.EqualValues(t, 15, bundleTransactionRunner.snapshots[0].sizeLimit)
}

func TestBundleTransactionRunner_RevertToSnapshot_CallsRevertToInterTxSnapshotOnStateDbAndResetsLocalState(t *testing.T) {
	ctrl := gomock.NewController(t)
	state := state.NewMockStateDB(ctrl)

	snapshotId := 123
	state.EXPECT().RevertToInterTxSnapshot(snapshotId)

	gasPool := core.NewGasPool(10)
	usedGas := uint64(11)
	ctxt := &runContext{statedb: state, usedGas: &usedGas, gasPool: gasPool}
	bundleTransactionRunner := &bundleTransactionRunner{
		ctxt:                  ctxt,
		trueTxOffset:          51,
		processedTransactions: make([]ProcessedTransaction, 100),
		sizeLimit:             999,
	}

	bundleTransactionRunner.snapshots = []bundleTransactionRunnerSnapshot{{
		stateDbSnapshot:                snapshotId,
		trueTxOffset:                   15,
		processedTransactionListLength: 5,
		usedGas:                        6,
		gasPool:                        core.NewGasPool(7),
		sizeLimit:                      42,
	}}
	bundleTransactionRunner.RevertToSnapshot(0)

	require.Len(t, bundleTransactionRunner.snapshots, 0)
	require.Equal(t, 15, bundleTransactionRunner.trueTxOffset)
	require.Len(t, bundleTransactionRunner.processedTransactions, 5)
	require.EqualValues(t, 6, *ctxt.usedGas)
	require.EqualValues(t, 7, ctxt.gasPool.Gas())
	require.EqualValues(t, 42, bundleTransactionRunner.sizeLimit)
}

func TestBundleTransactionRunner_CreatingAndRevertingSnapshotsDoesNotAlterUsedGasAndGasPoolAddressesOfContext(t *testing.T) {
	ctrl := gomock.NewController(t)
	state := state.NewMockStateDB(ctrl)

	snapshotId := 123
	state.EXPECT().InterTxSnapshot().Return(snapshotId)
	state.EXPECT().RevertToInterTxSnapshot(snapshotId)

	gasPool := core.NewGasPool(10)
	usedGas := uint64(11)
	ctxt := &runContext{statedb: state, usedGas: &usedGas, gasPool: gasPool}
	bundleTransactionRunner := &bundleTransactionRunner{
		ctxt:                  ctxt,
		trueTxOffset:          51,
		processedTransactions: make([]ProcessedTransaction, 100),
	}

	usedGasAddress := ctxt.usedGas
	gasPoolAddress := ctxt.gasPool

	snapshotId = bundleTransactionRunner.CreateSnapshot()
	require.Same(t, usedGasAddress, ctxt.usedGas)
	require.Same(t, gasPoolAddress, ctxt.gasPool)

	bundleTransactionRunner.RevertToSnapshot(snapshotId)
	require.Same(t, usedGasAddress, ctxt.usedGas)
	require.Same(t, gasPoolAddress, ctxt.gasPool)
}

func TestBundleTransactionRunner_RevertToSnapshot_InvalidId_TriggerInvalidRevertInStateDB(t *testing.T) {
	tests := []int{-10, -1, 5, 10}
	for _, id := range tests {
		t.Run(fmt.Sprintf("snapshot id %d", id), func(t *testing.T) {
			ctrl := gomock.NewController(t)
			db := state.NewMockStateDB(ctrl)

			runner := &bundleTransactionRunner{
				ctxt:      &runContext{statedb: db},
				snapshots: make([]bundleTransactionRunnerSnapshot, 5),
			}

			db.EXPECT().RevertToInterTxSnapshot(state.InvalidSnapshotID)

			runner.RevertToSnapshot(id)
		})
	}
}

func TestBundleTransactionRunner_Run_ForwardsLegacyTransactionOffsetToRegularTxCall(t *testing.T) {
	for _, offset := range []int{0, 5, 10} {
		t.Run(fmt.Sprintf("offset %d", offset), func(t *testing.T) {
			ctrl := gomock.NewController(t)
			runner := NewMock_transactionRunner(ctrl)

			stateDb := state.NewMockStateDB(ctrl)
			stateDb.EXPECT().InterTxSnapshot().Return(1).AnyTimes()
			stateDb.EXPECT().RevertToInterTxSnapshot(1).AnyTimes()

			sizeLimit := uint64(1_000_000)
			ctxt := &runContext{
				runner:  runner,
				gasPool: core.NewGasPool(math.MaxUint64),
				usedGas: new(uint64),
				statedb: stateDb,
			}
			bundleTransactionRunner := &bundleTransactionRunner{
				ctxt:         ctxt,
				trueTxOffset: offset + 1,
				sizeLimit:    sizeLimit,
			}

			tx := getRegularTransaction(t)

			runner.EXPECT().runRegularTransaction(ctxt, tx, offset+1, sizeLimit)
			bundleTransactionRunner.Run(tx)
		})
	}
}

func TestBundleTransactionRunner_Run_ForwardsLegacyTransactionOffsetToSponsoredTxCall(t *testing.T) {
	for _, offset := range []int{0, 5, 10} {
		t.Run(fmt.Sprintf("offset %d", offset), func(t *testing.T) {
			ctrl := gomock.NewController(t)
			runner := NewMock_transactionRunner(ctrl)

			stateDb := state.NewMockStateDB(ctrl)
			stateDb.EXPECT().InterTxSnapshot().Return(1).AnyTimes()
			stateDb.EXPECT().RevertToInterTxSnapshot(1).AnyTimes()

			sizeLimit := uint64(1_000_000)
			ctxt := &runContext{
				runner:   runner,
				upgrades: opera.Upgrades{GasSubsidies: true},
				gasPool:  core.NewGasPool(math.MaxUint64),
				usedGas:  new(uint64),
				statedb:  stateDb,
			}
			bundleTransactionRunner := &bundleTransactionRunner{
				ctxt:         ctxt,
				trueTxOffset: offset + 1,
				sizeLimit:    sizeLimit,
			}

			tx := getSponsorshipRequest(t)

			runner.EXPECT().runSponsoredTransaction(ctxt, tx, offset+1, sizeLimit)
			bundleTransactionRunner.Run(tx)
		})
	}
}

func TestBundleTransactionRunner_Run_ForwardsTrueTransactionOffsetToBundleTxCall(t *testing.T) {
	for _, trueOffset := range []int{0, 5, 10} {
		t.Run(fmt.Sprintf("true=%d", trueOffset), func(t *testing.T) {
			ctrl := gomock.NewController(t)
			runner := NewMock_transactionRunner(ctrl)

			stateDb := state.NewMockStateDB(ctrl)
			stateDb.EXPECT().InterTxSnapshot().Return(1).AnyTimes()
			stateDb.EXPECT().RevertToInterTxSnapshot(1).AnyTimes()

			sizeLimit := uint64(1_000_000)
			upgrades := opera.Upgrades{Brio: true, GasSubsidies: true, TransactionBundles: true}
			ctxt := &runContext{
				runner:   runner,
				upgrades: upgrades,
				gasPool:  core.NewGasPool(math.MaxUint64),
				usedGas:  new(uint64),
				statedb:  stateDb,
			}
			bundleTransactionRunner := &bundleTransactionRunner{
				ctxt:         ctxt,
				trueTxOffset: trueOffset,
				sizeLimit:    sizeLimit,
			}

			tx := getTransactionBundle(t)

			runner.EXPECT().runTransactionBundle(ctxt, tx, trueOffset, sizeLimit)
			bundleTransactionRunner.Run(tx)
		})
	}
}

func TestBundleTransactionRunner_Run_UpdatesTxIndexBasedOnNumberOfAcceptedTransactions(t *testing.T) {
	tests := map[string]struct {
		execResult []ProcessedTransaction
	}{
		"empty result": {
			execResult: []ProcessedTransaction{},
		},
		"one accepted transaction": {
			execResult: []ProcessedTransaction{
				{Receipt: &types.Receipt{}},
			},
		},
		"multiple transactions with mixed acceptance": {
			execResult: []ProcessedTransaction{
				{Receipt: &types.Receipt{}},
				{Receipt: nil},
				{Receipt: &types.Receipt{}},
				{Receipt: nil},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			runner := NewMock_transactionRunner(ctrl)

			stateDb := state.NewMockStateDB(ctrl)
			stateDb.EXPECT().InterTxSnapshot().Return(1).AnyTimes()
			stateDb.EXPECT().RevertToInterTxSnapshot(1).AnyTimes()

			trueStartOffset := 7

			ctxt := &runContext{
				runner:   runner,
				upgrades: opera.Upgrades{GasSubsidies: true},
				gasPool:  core.NewGasPool(math.MaxUint64),
				usedGas:  new(uint64),
				statedb:  stateDb,
			}
			bundleTransactionRunner := &bundleTransactionRunner{
				ctxt:         ctxt,
				trueTxOffset: trueStartOffset,
				sizeLimit:    math.MaxUint64,
			}

			tx := getSponsorshipRequest(t)

			for i := range test.execResult {
				test.execResult[i].Transaction = types.NewTx(&types.LegacyTx{})
			}

			runner.EXPECT().runSponsoredTransaction(ctxt, tx, trueStartOffset, gomock.Any()).Return(
				test.execResult,
				core_types.TransactionResultSuccessful,
			)

			bundleTransactionRunner.Run(tx)

			acceptedCount := 0
			for _, result := range test.execResult {
				if result.Receipt != nil {
					acceptedCount++
				}
			}
			require.Equal(t, trueStartOffset+acceptedCount, bundleTransactionRunner.trueTxOffset)
		})
	}
}

func TestBundleTransactionRunner_Snapshot_CoversTxOffset(t *testing.T) {
	ctrl := gomock.NewController(t)
	state := state.NewMockStateDB(ctrl)
	state.EXPECT().InterTxSnapshot().AnyTimes()
	state.EXPECT().RevertToInterTxSnapshot(gomock.Any()).AnyTimes()

	runner := &bundleTransactionRunner{
		ctxt:         &runContext{statedb: state, usedGas: new(uint64), gasPool: core.NewGasPool(1_000_000)},
		trueTxOffset: 7,
	}

	s1 := runner.CreateSnapshot()
	runner.trueTxOffset = 12
	_ = runner.CreateSnapshot()
	runner.trueTxOffset = 18
	s3 := runner.CreateSnapshot()
	runner.trueTxOffset = 21

	// revert a single snapshot to check that the transaction index is covered
	runner.RevertToSnapshot(s3)
	require.Equal(t, 18, runner.trueTxOffset)

	// revert two snapshots in one go and check the recovered state
	runner.RevertToSnapshot(s1)
	require.Equal(t, 7, runner.trueTxOffset)
}

func TestBundleTransactionRunner_Run_CollectsProcessedTransactionsInOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	txRunner := NewMock_transactionRunner(ctrl)

	signer := types.LatestSignerForChainID(big.NewInt(1))

	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	// We simulate 4 sponsored transactions with variable length lists of
	// processed transactions to check the correct aggregation of the processed
	// transaction list.
	txs := []*types.Transaction{
		types.MustSignNewTx(key, signer, &types.LegacyTx{Nonce: 0, To: &common.Address{}}),
		types.MustSignNewTx(key, signer, &types.LegacyTx{Nonce: 1, To: &common.Address{}}),
		types.MustSignNewTx(key, signer, &types.LegacyTx{Nonce: 2, To: &common.Address{}}),
		types.MustSignNewTx(key, signer, &types.LegacyTx{Nonce: 3, To: &common.Address{}}),
	}

	results := [][]ProcessedTransaction{
		{{Transaction: txs[0], Receipt: &types.Receipt{}}},
		{
			{Transaction: txs[1], Receipt: &types.Receipt{}},
			{Transaction: txs[2], Receipt: nil},
		},
		{},
		{{Transaction: txs[3], Receipt: &types.Receipt{}}},
	}

	for i, tx := range txs {
		require.True(t, subsidies.IsSponsorshipRequest(tx))
		txRunner.EXPECT().runSponsoredTransaction(gomock.Any(), tx, gomock.Any(), gomock.Any()).Return(
			results[i],
			core_types.TransactionResultSuccessful,
		)
	}

	stateDb := state.NewMockStateDB(ctrl)
	stateDb.EXPECT().InterTxSnapshot().Return(1).AnyTimes()
	stateDb.EXPECT().RevertToInterTxSnapshot(1).AnyTimes()

	ctxt := &runContext{
		runner:   txRunner,
		upgrades: opera.Upgrades{GasSubsidies: true},
		gasPool:  core.NewGasPool(math.MaxUint64),
		usedGas:  new(uint64),
		statedb:  stateDb,
	}
	runner := &bundleTransactionRunner{
		ctxt:      ctxt,
		sizeLimit: math.MaxUint64,
	}

	var want []ProcessedTransaction
	require.Equal(t, want, runner.processedTransactions)

	for i := range txs {
		runner.Run(txs[i])
		want = append(want, results[i]...)
		require.Equal(t, want, runner.processedTransactions)
	}
}

func TestBundleTransactionRunner_Snapshot_CoversProcessedTransactions(t *testing.T) {
	ctrl := gomock.NewController(t)
	state := state.NewMockStateDB(ctrl)
	state.EXPECT().InterTxSnapshot().AnyTimes()
	state.EXPECT().RevertToInterTxSnapshot(gomock.Any()).AnyTimes()

	processedTransactions := []ProcessedTransaction{
		{Receipt: &types.Receipt{CumulativeGasUsed: 1}},
		{Receipt: &types.Receipt{CumulativeGasUsed: 2}},
		{Receipt: &types.Receipt{CumulativeGasUsed: 3}},
		{Receipt: &types.Receipt{CumulativeGasUsed: 4}},
		{Receipt: &types.Receipt{CumulativeGasUsed: 5}},
	}

	runner := &bundleTransactionRunner{
		ctxt:                  &runContext{statedb: state, usedGas: new(uint64), gasPool: core.NewGasPool(1_000_000)},
		processedTransactions: processedTransactions[:1],
	}

	s1 := runner.CreateSnapshot()
	runner.processedTransactions = append(
		runner.processedTransactions,
		processedTransactions[1],
	)

	_ = runner.CreateSnapshot()
	runner.processedTransactions = append(
		runner.processedTransactions,
		processedTransactions[2],
	)

	s3 := runner.CreateSnapshot()
	runner.processedTransactions = append(
		runner.processedTransactions,
		processedTransactions[3],
		processedTransactions[4],
	)

	// revert a single snapshot to check that the list of processed transactions
	// is covered
	runner.RevertToSnapshot(s3)
	require.Equal(t, processedTransactions[:3], runner.processedTransactions)

	// revert two snapshots in one go and check the recovered state
	runner.RevertToSnapshot(s1)
	require.Equal(t, processedTransactions[:1], runner.processedTransactions)

}

// --- Utility functions for creating test transactions ---

func getRegularTransaction(t *testing.T) *types.Transaction {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	signer := types.LatestSignerForChainID(nil)
	return types.MustSignNewTx(key, signer, &types.LegacyTx{
		Nonce: 0, To: &common.Address{1}, Gas: 21_000, GasPrice: big.NewInt(1),
	})
}

func getSponsorshipRequestUnsigned(*testing.T) *types.Transaction {
	return types.NewTx(&types.LegacyTx{
		Nonce: 0, To: &common.Address{1}, Gas: 21_000,
	})
}

func getSponsorshipRequest(t *testing.T) *types.Transaction {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	signer := types.LatestSignerForChainID(nil)
	return types.MustSignNewTx(key, signer, &types.LegacyTx{
		Nonce: 0, To: &common.Address{1}, Gas: 21_000,
	})
}

func getTransactionBundle(t *testing.T) *types.Transaction {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	envelope := bundle.AllOf(bundle.Step(key, &types.AccessListTx{
		Nonce: 0, To: &common.Address{1}, Gas: 21_000, GasPrice: big.NewInt(1),
	})).Build()
	return envelope
}

func TestTransactionGenerationUtilities(t *testing.T) {
	regular := getRegularTransaction(t)
	request := getSponsorshipRequest(t)

	require.False(t, subsidies.IsSponsorshipRequest(regular))
	require.True(t, subsidies.IsSponsorshipRequest(request))
}

func TestTrackingOfTxIndicesInNestedAndComposedBundles(t *testing.T) {
	// This test is a kind of integration test of most of the functions in the
	// state processor involved in tracking the transaction index. It sets up a
	// few example bundles with expectations on the seen transaction indices
	// that are then verified during a test execution.
	//
	// The idea is to build bundles of transactions where the individual
	// transactions encode the expected transaction indices when being executed.
	// This encoding happens by placing the true transaction index into the
	// value field. A mock implementation of the transaction runner intercepts
	// those transactions, decodes the expected values for the transaction
	// indices, and checks that they match the actually encountered values.
	//
	// The following functions provide some utilities to improve the readability
	// of the test case specifications.
	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	wantTxIndex := func(trueOffset uint64) bundle.BuilderStep {
		// the two expected offsets are encoded in the Nonce and Value fields
		return bundle.Step(key, &types.AccessListTx{
			Value: big.NewInt(int64(trueOffset)),
		})
	}

	skippedWithTxIndex := func(trueOffset uint64) bundle.BuilderStep {
		// the test runner below skips all transactions with non-empty data
		return bundle.Step(key,
			&types.AccessListTx{
				Value: big.NewInt(int64(trueOffset)),
				Data:  []byte{0},
			},
		).WithFlags(bundle.EF_TolerateInvalid)
	}

	sponsoredWithTxIndex := func(trueOffset uint64) bundle.BuilderStep {
		return bundle.Step(key, &types.AccessListTx{
			Value: big.NewInt(int64(trueOffset)),
			To:    &common.Address{}, // < makes it a sponsorship request
		})
	}

	fail := bundle.OneOf()

	// The test runner below is set up to check that when running each of those
	// bundles the transaction indices are checked in each step.
	tests := map[string]*types.Transaction{
		"all-of sequence": bundle.AllOf(
			wantTxIndex(0),
			wantTxIndex(1),
			wantTxIndex(2),
		).Build(),

		"all-of sequence with skipped": bundle.AllOf(
			wantTxIndex(0),
			skippedWithTxIndex(1),
			wantTxIndex(1), // < skipped transactions are ignored
		).Build(),

		"all-of sequence with sponsored": bundle.AllOf(
			wantTxIndex(0),
			sponsoredWithTxIndex(1),
			wantTxIndex(3), // id 2 is the implicit payment of the sponsorship
		).Build(),

		"mix of sponsored and skipped": bundle.AllOf(
			wantTxIndex(0),
			sponsoredWithTxIndex(1),
			wantTxIndex(3),
			skippedWithTxIndex(4),
			skippedWithTxIndex(4),
			skippedWithTxIndex(4),
			wantTxIndex(4),
			sponsoredWithTxIndex(5),
			wantTxIndex(7),
		).Build(),

		"group rolled-back": bundle.OneOf(
			bundle.AllOf(
				wantTxIndex(0),
				wantTxIndex(1),
				wantTxIndex(2),
				fail,
			),
			// after the fail, the counting starts from 0 again
			bundle.AllOf(
				wantTxIndex(0),
				wantTxIndex(1),
			),
		).Build(),

		"partial roll-back": bundle.AllOf(
			wantTxIndex(0),
			skippedWithTxIndex(1),
			wantTxIndex(1),
			bundle.AllOf(
				wantTxIndex(2),
				skippedWithTxIndex(3),
				wantTxIndex(3),
				fail, // the group fails here, but the outer AllOf continues
			).WithFlags(bundle.EF_TolerateFailed),
			bundle.AllOf(
				wantTxIndex(2),
				wantTxIndex(3),
				skippedWithTxIndex(4),
				wantTxIndex(4),
			),
		).Build(),

		"nested bundles": bundle.AllOf(
			bundle.Step(key, bundle.AllOf(
				wantTxIndex(0),
				skippedWithTxIndex(1),
				wantTxIndex(1),
			).Build()),
			bundle.Step(key, bundle.AllOf(
				wantTxIndex(2),
				wantTxIndex(3),
			).Build()),
		).Build(),

		"nested bundle rolling back with outer failure tolerance": bundle.AllOf(
			bundle.Step(key, bundle.AllOf(
				wantTxIndex(0),
				wantTxIndex(1),
			).Build()),
			bundle.Step(key, bundle.AllOf(
				wantTxIndex(2),
				wantTxIndex(3),
				fail, // this one fails, but the failure is tolerated
			).WithFlags(bundle.EF_TolerateFailed).Build()),
			bundle.Step(key, bundle.AllOf(
				wantTxIndex(2),
			).Build()),
		).Build(),

		"nested bundle rolling back with inner failure tolerance": bundle.AllOf(
			bundle.Step(key, bundle.AllOf(
				wantTxIndex(0),
				wantTxIndex(1),
			).Build()),
			bundle.Step(key, bundle.AllOf(
				wantTxIndex(2),
				wantTxIndex(3),
				fail, // this one fails, but the failure is tolerated
			).Build()).WithFlags(bundle.EF_TolerateFailed),
			bundle.Step(key, bundle.AllOf(
				wantTxIndex(2),
			).Build()),
		).Build(),
	}

	for name, tx := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			ctrl := gomock.NewController(t)
			runner := NewMock_transactionRunner(ctrl)

			any := gomock.Any()

			// Keep track of seen transactions to make sure all transactions
			// in the bundle have actually been processed in the expected order.
			// It is also a neat, readable primitive for debugging this test.
			var seenTxNonces []int

			// regular transactions all pass, but we check whether the
			// given nonce matches the transaction index.
			runner.EXPECT().runRegularTransaction(any, any, any, any).DoAndReturn(
				func(
					ctxt *runContext, tx *types.Transaction,
					trueTxOffset int, sizeLimit uint64,
				) (ProcessedTransaction, core_types.TransactionResult) {
					require.Equal(int(tx.Value().Int64()), trueTxOffset)
					seenTxNonces = append(seenTxNonces, int(tx.Nonce()))

					// skip the transaction if it contains data
					if len(tx.Data()) > 0 {
						return ProcessedTransaction{
							Transaction: tx,
						}, core_types.TransactionResultInvalid
					}

					return ProcessedTransaction{
						Transaction: tx,
						Receipt:     &types.Receipt{},
					}, core_types.TransactionResultSuccessful
				},
			).AnyTimes()

			// sponsored transactions all pass and produce two successful
			// processed transactions, the sponsored and the payment tx
			runner.EXPECT().runSponsoredTransaction(any, any, any, any).DoAndReturn(
				func(
					ctxt *runContext, tx *types.Transaction,
					trueTxOffset int, sizeLimit uint64,
				) ([]ProcessedTransaction, core_types.TransactionResult) {
					require.Equal(int(tx.Value().Int64()), trueTxOffset)
					seenTxNonces = append(seenTxNonces, int(tx.Nonce()))
					return []ProcessedTransaction{
						{
							Transaction: tx,
							Receipt:     &types.Receipt{},
						},
						{
							Transaction: types.NewTx(&types.LegacyTx{}), // < the payment transaction, we don't care about the details here
							Receipt:     &types.Receipt{},
						},
					}, core_types.TransactionResultSuccessful
				},
			).AnyTimes()

			// bundle envelopes are opened and executed
			runner.EXPECT().runTransactionBundle(any, any, any, any).DoAndReturn(
				// we want to use the real implementation here to check the
				// correct forwarding of the transaction indices
				new(transactionRunner).runTransactionBundle,
			).AnyTimes()

			// The execution of the transactions does not really reach the
			// StateDB, but some operations are triggered while processing the
			// bundles. These are not the objective of this test.
			stateDb := state.NewMockStateDB(ctrl)
			stateDb.EXPECT().HasBundleRecentlyBeenProcessed(any).AnyTimes()
			stateDb.EXPECT().InterTxSnapshot().AnyTimes()
			stateDb.EXPECT().RevertToInterTxSnapshot(any).AnyTimes()
			stateDb.EXPECT().AddProcessedBundle(any, any).AnyTimes()

			signer := types.LatestSignerForChainID(big.NewInt(1))
			upgrades := opera.Upgrades{Brio: true, GasSubsidies: true, TransactionBundles: true}
			ctxt := &runContext{
				signer:      signer,
				usedGas:     new(uint64),
				gasPool:     core.NewGasPool(1_000_000),
				upgrades:    upgrades,
				blockNumber: big.NewInt(0),
				statedb:     stateDb,
				runner:      runner,
			}

			// the actual execution of the test case
			runTransaction(ctxt, tx, 0, math.MaxUint64)

			// make sure that all transactions have indeed been processed
			wantedTxNonces := collectAllReferencedTransactionNonces(t, signer, tx)
			require.Equal(wantedTxNonces, seenTxNonces)
		})
	}
}

func collectAllReferencedTransactionNonces(
	t *testing.T,
	signer types.Signer,
	tx *types.Transaction,
) []int {
	t.Helper()
	var res []int
	if bundle.IsEnvelope(tx) {
		bundle, err := bundle.OpenEnvelope(signer, tx)
		require.NoError(t, err)
		for _, tx := range bundle.GetTransactionsInReferencedOrder() {
			res = append(res, collectAllReferencedTransactionNonces(t, signer, tx)...)
		}
	} else {
		res = append(res, int(tx.Nonce()))
	}
	return res
}

func TestNewTransactionProcessorForBlock_ConfiguresTransactionProcessorWithValuesFromParameters(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)

	block := &EvmBlock{
		EvmHeader: EvmHeader{
			Number: big.NewInt(123),
		},
	}

	chainCfg := &params.ChainConfig{
		ChainID: big.NewInt(456),
	}
	currentRules := opera.Rules{
		Name: "unit-test-net",
		Upgrades: opera.Upgrades{
			Berlin: true,
			Brio:   true,
		},
	}

	chain := NewMockChainState(ctrl)
	chain.EXPECT().GetCurrentChainConfig().Return(chainCfg)
	chain.EXPECT().GetCurrentNetworkRules().Return(currentRules).AnyTimes()

	state := state.NewMockStateDB(ctrl)

	processor := NewTransactionProcessorForBlock(chain, state, block)

	require.Equal(block.Number, processor.blockNumber)
	require.NotNil(processor.gp)
	require.Equal(uint64(math.MaxUint64), processor.gp.Gas())
	require.Equal(block.Header(), processor.header)
	require.Equal(types.LatestSignerForChainID(chainCfg.ChainID), processor.signer)
	require.Equal(state, processor.stateDb)
	require.EqualValues(0, processor.usedGas)
	require.NotNil(processor.vmEnvironment)
	require.Equal(currentRules.Upgrades, processor.upgrades)
}

func TestTransactionProcessor_Run_ForwardsTimeToBundleProcessing(t *testing.T) {
	ctrl := gomock.NewController(t)
	stateDb := state.NewMockStateDB(ctrl)
	stateDb.EXPECT().HasBundleRecentlyBeenProcessed(gomock.Any()).AnyTimes()
	stateDb.EXPECT().InterTxSnapshot().AnyTimes()
	stateDb.EXPECT().AddProcessedBundle(gomock.Any(), gomock.Any()).AnyTimes()

	chainCfg := &params.ChainConfig{
		ChainID: big.NewInt(1),
	}
	currentRules := opera.Rules{
		Upgrades: opera.Upgrades{
			Brio:               true,
			TransactionBundles: true,
		},
	}

	chain := NewMockChainState(ctrl)
	chain.EXPECT().GetCurrentChainConfig().Return(chainCfg)
	chain.EXPECT().GetCurrentNetworkRules().Return(currentRules).AnyTimes()

	currentTime := inter.Timestamp(12_345)
	block := &EvmBlock{
		EvmHeader: EvmHeader{
			Number: big.NewInt(1),
			Time:   currentTime,
		},
	}

	txProcessor := NewTransactionProcessorForBlock(chain, stateDb, block)

	// Processable Bundles should be processed.
	processableBundle := bundle.NewBuilder().AllOf().SetNotBefore(currentTime).Build()
	summary := txProcessor.Run(0, processableBundle)
	require.Empty(t, summary.ProcessedTransactions) // < accepted, no result

	// Blocked bundles should be rejected.
	blockedBundle := bundle.NewBuilder().AllOf().SetNotBefore(currentTime + 1).Build()
	summary = txProcessor.Run(0, blockedBundle)
	require.Len(t, summary.ProcessedTransactions, 1)
	processedTx := summary.ProcessedTransactions[0]
	require.Equal(t, blockedBundle, processedTx.Transaction)
	require.Nil(t, processedTx.Receipt)
}

// --- ExecutionCost tests ---

func TestRunTransactions_AccumulatesExecutionCostFromAllTransactions(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := NewMock_transactionRunner(ctrl)

	context := &runContext{
		signer:   types.LatestSignerForChainID(big.NewInt(1)),
		runner:   runner,
		upgrades: opera.Upgrades{Brio: true, GasSubsidies: true, TransactionBundles: true},
	}

	txs := []*types.Transaction{
		getRegularTransaction(t),
		getSponsorshipRequest(t),
		getTransactionBundle(t),
	}

	gomock.InOrder(
		runner.EXPECT().runRegularTransaction(context, txs[0], 0, gomock.Any()).Return(
			ProcessedTransaction{Transaction: txs[0], Receipt: &types.Receipt{GasUsed: 100}},
			core_types.TransactionResultSuccessful,
		),
		runner.EXPECT().runSponsoredTransaction(context, txs[1], 1, gomock.Any()).Return(
			[]ProcessedTransaction{
				{Transaction: txs[1], Receipt: &types.Receipt{GasUsed: 200}},
				{Transaction: types.NewTx(&types.LegacyTx{}), Receipt: &types.Receipt{GasUsed: 300}},
			},
			core_types.TransactionResultSuccessful,
		),
		runner.EXPECT().runTransactionBundle(context, txs[2], 3, gomock.Any()).Return(
			[]ProcessedTransaction{{Transaction: txs[2], Receipt: &types.Receipt{GasUsed: 400}}},
			core_types.TransactionResultSuccessful,
			core_types.ExecutionCost(500), // higher than receipt gas
		),
	)

	summary := runTransactions(context, txs, 0, math.MaxUint64)
	require.EqualValues(t, 100+200+300+500, summary.ExecutionCost)
}

func TestRunTransaction_RegularTransaction_ReturnsExecutionCostFromReceipt(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := NewMock_transactionRunner(ctrl)

	context := &runContext{runner: runner, upgrades: opera.Upgrades{}}

	tx := getRegularTransaction(t)

	cases := map[string]struct {
		receipt        *types.Receipt
		expectExecCost core_types.ExecutionCost
	}{
		"receipt with gas used": {
			receipt:        &types.Receipt{GasUsed: 42_000},
			expectExecCost: 42_000,
		},
		"nil-receipt with zero gas used": {
			receipt:        nil,
			expectExecCost: 0,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			runner.EXPECT().runRegularTransaction(context, tx, 0, gomock.Any()).Return(
				ProcessedTransaction{Transaction: tx, Receipt: tc.receipt},
				core_types.TransactionResultSuccessful,
			)

			_, _, execCost := runTransaction(context, tx, 0, math.MaxUint64)
			require.Equal(t, tc.expectExecCost, execCost)
		})
	}
}

func TestRunTransaction_SponsoredTransaction_ReturnsExecutionCostFromReceipts(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := NewMock_transactionRunner(ctrl)

	context := &runContext{runner: runner, upgrades: opera.Upgrades{GasSubsidies: true}}

	tx := getSponsorshipRequest(t)

	cases := map[string]struct {
		receipts       [2]*types.Receipt
		expectExecCost core_types.ExecutionCost
	}{
		"receipts with gas used": {
			receipts:       [2]*types.Receipt{{GasUsed: 21_000}, {GasUsed: 5_000}},
			expectExecCost: 26_000,
		},
		"receipt with nil and one with gas used": {
			receipts:       [2]*types.Receipt{nil, {GasUsed: 5_000}},
			expectExecCost: 5_000,
		},
		"nil-receipts": {
			receipts:       [2]*types.Receipt{nil, nil},
			expectExecCost: 0,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			runner.EXPECT().runSponsoredTransaction(context, tx, 0, gomock.Any()).Return(
				[]ProcessedTransaction{
					{Transaction: tx, Receipt: tc.receipts[0]},
					{Transaction: &types.Transaction{}, Receipt: tc.receipts[1]},
				},
				core_types.TransactionResultSuccessful,
			)

			_, _, execCost := runTransaction(context, tx, 0, math.MaxUint64)
			require.Equal(t, tc.expectExecCost, execCost)
		})
	}
}

func TestRunTransaction_Bundle_ReturnsExecutionCostFromBundleExecution(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := NewMock_transactionRunner(ctrl)

	tx := getTransactionBundle(t)
	context := &runContext{
		runner: runner,
		upgrades: opera.Upgrades{
			Brio:               true,
			TransactionBundles: true,
		},
	}

	runner.EXPECT().runTransactionBundle(context, tx, 0, gomock.Any()).Return(
		[]ProcessedTransaction{{Transaction: tx, Receipt: &types.Receipt{GasUsed: 100_000}}},
		core_types.TransactionResultSuccessful,
		core_types.ExecutionCost(150_000), // higher than receipt gas due to rolled-back steps
	)

	_, _, execCost := runTransaction(context, tx, 0, math.MaxUint64)
	require.Equal(t, core_types.ExecutionCost(150_000), execCost)
}

func TestRunRegularTransaction_AcceptsTransactionIfItIsWithinSizeLimit(t *testing.T) {
	tx := types.NewTx(&types.LegacyTx{})

	tests := map[string]struct {
		sizeLimit  uint64
		successful bool
	}{
		"size limit smaller than transaction size": {
			sizeLimit: tx.Size() - 1,
		},
		"size limit equal to transaction size": {
			sizeLimit:  tx.Size(),
			successful: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			evm := NewMock_evm(ctrl)
			ctxt := &runContext{}

			if test.successful {
				evm.EXPECT().runWithBaseFeeCheck(ctxt, tx, 0).Return(
					ProcessedTransaction{Transaction: tx, Receipt: &types.Receipt{Status: types.ReceiptStatusSuccessful}},
				)
			}

			runner := transactionRunner{evm}
			got, status := runner.runRegularTransaction(ctxt, tx, 0, test.sizeLimit)
			require.Equal(t, tx, got.Transaction)

			if test.successful {
				require.Equal(t, core_types.TransactionResultSuccessful, status)
				require.NotNil(t, got.Receipt)
			} else {
				require.Equal(t, core_types.TransactionResultInvalid, status)
				require.Nil(t, got.Receipt)
			}
		})
	}
}

func TestRunSponsoredTransaction_SkipsTransactionExceedingSizeLimit(t *testing.T) {
	tx := getSponsorshipRequest(t)

	tests := map[string]struct {
		sizeLimit uint64
	}{
		"no space": {
			sizeLimit: 0,
		},
		"only space for tx, not payment": {
			sizeLimit: tx.Size(),
		},
		"just under the required size": {
			sizeLimit: tx.Size() + subsidies.GetMaxPostTxSizeForTests() - 1,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			evm := NewMock_evm(ctrl)
			stateDb := state.NewMockStateDB(ctrl)

			// IsCovered is called before the size check, since the extra
			// size needed depends on the sponsorship mode. So we need to
			// mock the EVM to report a fund-backed sponsorship mode, which
			// is one of the modes that requires the maximum extra size, to
			// trigger the size check in the tested code.
			stateDb.EXPECT().Snapshot().Return(1).AnyTimes()
			stateDb.EXPECT().RevertToSnapshot(1).AnyTimes()

			gasConfigResponse := make([]byte, 3*32)
			binary.BigEndian.PutUint64(gasConfigResponse[32*0+24:32*0+32], 100_000) // chooseFundGasLimit
			binary.BigEndian.PutUint64(gasConfigResponse[32*1+24:32*1+32], 60_000)  // deductFeesGasLimit
			binary.BigEndian.PutUint64(gasConfigResponse[32*2+24:32*2+32], 210_000) // overheadCharge (legacy shared overhead)
			fundIdResponse := [32]byte{0x01}                                        // non-zero fundId → mode 1
			gomock.InOrder(
				evm.EXPECT().Call(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(gasConfigResponse, uint64(0), nil),
				evm.EXPECT().Call(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(fundIdResponse[:], uint64(0), nil),
			)

			ctxt := &runContext{
				upgrades: opera.Upgrades{GasSubsidies: true},
				signer:   types.LatestSignerForChainID(nil),
				statedb:  stateDb,
				baseFee:  big.NewInt(1),
			}
			runner := &transactionRunner{evm: evm}
			got, status := runner.runSponsoredTransaction(ctxt, tx, 0, tc.sizeLimit)
			require.Equal(t, core_types.TransactionResultInvalid, status)
			require.Len(t, got, 1)
			require.Equal(t, tx, got[0].Transaction)
			require.Nil(t, got[0].Receipt)
		})
	}
}

func TestBundleTransactionRunner_Run_DeductsSizeLimitOnlyIfReceiptIsNotNil(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := NewMock_transactionRunner(ctrl)

	stateDb := state.NewMockStateDB(ctrl)
	stateDb.EXPECT().InterTxSnapshot().Return(1).AnyTimes()
	stateDb.EXPECT().RevertToInterTxSnapshot(1).AnyTimes()

	tx1 := getRegularTransaction(t)
	tx2 := getRegularTransaction(t)
	skippedTx := types.NewTx(&types.LegacyTx{
		Nonce: 0, To: &common.Address{1}, Gas: 21_000, GasPrice: big.NewInt(1),
	})

	ctxt := &runContext{
		runner:  runner,
		gasPool: core.NewGasPool(math.MaxUint64),
		usedGas: new(uint64),
		statedb: stateDb,
	}

	initialSizeLimit := tx1.Size() + tx2.Size()
	bundleRunner := &bundleTransactionRunner{
		ctxt:      ctxt,
		sizeLimit: initialSizeLimit,
	}

	runner.EXPECT().runRegularTransaction(ctxt, tx1, 0, initialSizeLimit).Return(
		ProcessedTransaction{Transaction: tx1, Receipt: &types.Receipt{}},
		core_types.TransactionResultSuccessful,
	)

	result := bundleRunner.Run(tx1)
	require.Equal(t, core_types.TransactionResultSuccessful, result)
	require.Equal(t, initialSizeLimit-tx1.Size(), bundleRunner.sizeLimit)

	// Running a transaction that is skipped (receipt is nil) should not reduce the size limit
	runner.EXPECT().runRegularTransaction(ctxt, skippedTx, 1, initialSizeLimit-tx1.Size()).Return(
		ProcessedTransaction{Transaction: skippedTx, Receipt: nil},
		core_types.TransactionResultInvalid,
	)

	result = bundleRunner.Run(skippedTx)
	require.Equal(t, core_types.TransactionResultInvalid, result)
	require.Equal(t, initialSizeLimit-tx1.Size(), bundleRunner.sizeLimit)

	// Second transaction should get the reduced sizeLimit
	runner.EXPECT().runRegularTransaction(ctxt, tx2, 1, initialSizeLimit-tx1.Size()).Return(
		ProcessedTransaction{Transaction: tx2, Receipt: &types.Receipt{}},
		core_types.TransactionResultSuccessful,
	)

	result = bundleRunner.Run(tx2)
	require.Equal(t, core_types.TransactionResultSuccessful, result)
	require.Equal(t, initialSizeLimit-tx1.Size()-tx2.Size(), bundleRunner.sizeLimit)
}

func TestBundleTransactionRunner_Run_RevertsWhenSizeLimitExceeded(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := NewMock_transactionRunner(ctrl)

	stateDb := state.NewMockStateDB(ctrl)

	interTxSnapshot := 42
	gomock.InOrder(
		stateDb.EXPECT().InterTxSnapshot().Return(interTxSnapshot),
		stateDb.EXPECT().RevertToInterTxSnapshot(interTxSnapshot),
	)

	tx := getRegularTransaction(t)

	ctxt := &runContext{
		runner:  runner,
		gasPool: core.NewGasPool(math.MaxUint64),
		usedGas: new(uint64),
		statedb: stateDb,
	}

	// sizeLimit is smaller than the transaction
	bundleRunner := &bundleTransactionRunner{
		ctxt:      ctxt,
		sizeLimit: tx.Size() - 1,
	}

	// The transaction will be run (runner is called with sizeLimit),
	// but since the runner returns a receipt and the processed tx size
	// exceeds the remaining sizeLimit, the bundle runner reverts.
	runner.EXPECT().runRegularTransaction(ctxt, tx, 0, tx.Size()-1).Return(
		ProcessedTransaction{Transaction: tx, Receipt: &types.Receipt{}},
		core_types.TransactionResultSuccessful,
	)

	result := bundleRunner.Run(tx)
	require.Equal(t, core_types.TransactionResultInvalid, result)
	// After revert, processed transactions should be empty
	require.Empty(t, bundleRunner.processedTransactions)
}

func TestBundleTransactionRunner_Snapshot_CoversSizeLimit(t *testing.T) {
	ctrl := gomock.NewController(t)
	stateDb := state.NewMockStateDB(ctrl)
	stateDb.EXPECT().InterTxSnapshot().AnyTimes()
	stateDb.EXPECT().RevertToInterTxSnapshot(gomock.Any()).AnyTimes()

	runner := &bundleTransactionRunner{
		ctxt:      &runContext{statedb: stateDb, usedGas: new(uint64), gasPool: core.NewGasPool(1_000_000)},
		sizeLimit: 1000,
	}

	s1 := runner.CreateSnapshot()
	require.EqualValues(t, 1000, runner.snapshots[0].sizeLimit)

	runner.sizeLimit = 500
	s2 := runner.CreateSnapshot()
	require.EqualValues(t, 500, runner.snapshots[1].sizeLimit)

	runner.sizeLimit = 100

	// revert to s2, should restore sizeLimit to 500
	runner.RevertToSnapshot(s2)
	require.EqualValues(t, 500, runner.sizeLimit)

	// revert to s1, should restore sizeLimit to 1000
	runner.RevertToSnapshot(s1)
	require.EqualValues(t, 1000, runner.sizeLimit)
}

func TestRunTransactions_ProcessesAllTransactionsWithinBlockSizeLimit(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	regular1 := getRegularTransaction(t)
	regular2 := getRegularTransaction(t)
	regular3 := getRegularTransaction(t)
	sponsored := getSponsorshipRequest(t)
	paymentTx := types.NewTx(&types.LegacyTx{Nonce: 99})

	tests := map[string]struct {
		txs           []*types.Transaction
		remainingSize uint64
		setupMock     func(*runContext, *Mock_transactionRunner)
		wantProcessed int
		wantIncluded  int
	}{
		"single transaction": {
			txs:           []*types.Transaction{regular1},
			remainingSize: regular1.Size(),
			setupMock: func(ctx *runContext, runner *Mock_transactionRunner) {
				runner.EXPECT().runRegularTransaction(ctx, regular1, 0, regular1.Size()).Return(
					ProcessedTransaction{Transaction: regular1, Receipt: &types.Receipt{}},
					core_types.TransactionResultSuccessful,
				)
			},
			wantProcessed: 1,
			wantIncluded:  1,
		},
		"multiple transactions": {
			txs:           []*types.Transaction{regular1, regular2, regular3},
			remainingSize: regular1.Size() + regular2.Size() + regular3.Size(),
			setupMock: func(ctx *runContext, runner *Mock_transactionRunner) {
				gomock.InOrder(
					runner.EXPECT().runRegularTransaction(ctx, regular1, 0, regular1.Size()+regular2.Size()+regular3.Size()).Return(
						ProcessedTransaction{Transaction: regular1, Receipt: &types.Receipt{}},
						core_types.TransactionResultSuccessful,
					),
					runner.EXPECT().runRegularTransaction(ctx, regular2, 1, regular2.Size()+regular3.Size()).Return(
						ProcessedTransaction{Transaction: regular2, Receipt: &types.Receipt{}},
						core_types.TransactionResultSuccessful,
					),
					runner.EXPECT().runRegularTransaction(ctx, regular3, 2, regular3.Size()).Return(
						ProcessedTransaction{Transaction: regular3, Receipt: &types.Receipt{}},
						core_types.TransactionResultSuccessful,
					),
				)
			},
			wantProcessed: 3,
			wantIncluded:  3,
		},
		"sponsored transaction with payment": {
			txs:           []*types.Transaction{sponsored},
			remainingSize: sponsored.Size() + subsidies.GetMaxPostTxSizeForTests(),
			setupMock: func(ctx *runContext, runner *Mock_transactionRunner) {
				runner.EXPECT().runSponsoredTransaction(ctx, sponsored, 0, gomock.Any()).Return(
					[]ProcessedTransaction{
						{Transaction: sponsored, Receipt: &types.Receipt{}},
						{Transaction: paymentTx, Receipt: &types.Receipt{}},
					},
					core_types.TransactionResultSuccessful,
				)
			},
			wantProcessed: 2,
			wantIncluded:  2,
		},
		"bundle transaction": {
			txs:           []*types.Transaction{bundle.AllOf(bundle.Step(key, regular1)).Build()},
			remainingSize: regular1.Size() + 1024, // Add some extra space for the bundle marker added by the builder
			setupMock: func(ctx *runContext, runner *Mock_transactionRunner) {
				runner.EXPECT().runTransactionBundle(ctx, gomock.Any(), 0, gomock.Any()).Return(
					[]ProcessedTransaction{
						{Transaction: regular1, Receipt: &types.Receipt{}},
					},
					core_types.TransactionResultSuccessful,
					core_types.ExecutionCost(0),
				)
			},
			wantProcessed: 1,
			wantIncluded:  1,
		},
		"combination of regular and skipped transactions": {
			txs:           []*types.Transaction{regular1, types.NewTx(&types.LegacyTx{Data: []byte{0}}), regular2},
			remainingSize: regular1.Size() + regular2.Size(),
			setupMock: func(ctx *runContext, runner *Mock_transactionRunner) {
				gomock.InOrder(
					runner.EXPECT().runRegularTransaction(ctx, regular1, 0, regular1.Size()+regular2.Size()).Return(
						ProcessedTransaction{Transaction: regular1, Receipt: &types.Receipt{}},
						core_types.TransactionResultSuccessful,
					),
					runner.EXPECT().runRegularTransaction(ctx, gomock.Any(), 1, regular2.Size()).Return(
						ProcessedTransaction{Transaction: types.NewTx(&types.LegacyTx{Data: []byte{0}}), Receipt: nil},
						core_types.TransactionResultInvalid,
					),
					runner.EXPECT().runRegularTransaction(ctx, regular2, 1, regular2.Size()).Return(
						ProcessedTransaction{Transaction: regular2, Receipt: &types.Receipt{}},
						core_types.TransactionResultSuccessful,
					),
				)
			},
			wantProcessed: 3,
			wantIncluded:  2,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			runner := NewMock_transactionRunner(ctrl)

			context := &runContext{
				upgrades: opera.Upgrades{Brio: true, GasSubsidies: true, TransactionBundles: true},
				signer:   types.LatestSignerForChainID(big.NewInt(1)),
				runner:   runner,
			}
			tc.setupMock(context, runner)

			summary := runTransactions(context, tc.txs, 0, tc.remainingSize)

			require.Len(t, summary.ProcessedTransactions, tc.wantProcessed)
			included := 0
			for _, processedTx := range summary.ProcessedTransactions {
				if processedTx.Receipt != nil {
					included++
				}
			}
			require.Equal(t, tc.wantIncluded, included)
		})
	}
}

func TestRunTransaction_CollectsCausedByInformationForAllTransactionTypes(t *testing.T) {

	regularTx := getRegularTransaction(t)
	sponsoredTx := getSponsorshipRequest(t)
	bundleTx := getTransactionBundle(t)

	// Simulated output transactions from the runner.
	paymentTx := types.NewTx(&types.LegacyTx{Nonce: 99})
	bundleInner1 := types.NewTx(&types.LegacyTx{Nonce: 200})
	bundleInner2 := types.NewTx(&types.LegacyTx{Nonce: 201})

	tests := map[string]struct {
		tx           *types.Transaction
		setupMock    func(*runContext, *Mock_transactionRunner)
		wantCausedBy map[common.Hash]common.Hash
	}{
		"basic transaction": {
			tx: regularTx,
			setupMock: func(ctx *runContext, runner *Mock_transactionRunner) {
				runner.EXPECT().runRegularTransaction(ctx, regularTx, 0, gomock.Any()).Return(
					ProcessedTransaction{Transaction: regularTx, Receipt: &types.Receipt{}},
					core_types.TransactionResultSuccessful,
				)
			},
			wantCausedBy: map[common.Hash]common.Hash{
				regularTx.Hash(): regularTx.Hash(),
			},
		},
		"sponsored transaction": {
			tx: sponsoredTx,
			setupMock: func(ctx *runContext, runner *Mock_transactionRunner) {
				runner.EXPECT().runSponsoredTransaction(ctx, sponsoredTx, 0, gomock.Any()).Return(
					[]ProcessedTransaction{
						{Transaction: sponsoredTx, Receipt: &types.Receipt{}},
						{Transaction: paymentTx, Receipt: &types.Receipt{}},
					},
					core_types.TransactionResultSuccessful,
				)
			},
			wantCausedBy: map[common.Hash]common.Hash{
				sponsoredTx.Hash(): sponsoredTx.Hash(),
				paymentTx.Hash():   sponsoredTx.Hash(),
			},
		},
		"bundle transaction": {
			tx: bundleTx,
			setupMock: func(ctx *runContext, runner *Mock_transactionRunner) {
				runner.EXPECT().runTransactionBundle(ctx, bundleTx, 0, gomock.Any()).Return(
					[]ProcessedTransaction{
						{Transaction: bundleInner1, Receipt: &types.Receipt{}},
						{Transaction: bundleInner2, Receipt: &types.Receipt{}},
					},
					core_types.TransactionResultSuccessful,
					core_types.ExecutionCost(0),
				)
			},
			wantCausedBy: map[common.Hash]common.Hash{
				bundleInner1.Hash(): bundleTx.Hash(),
				bundleInner2.Hash(): bundleTx.Hash(),
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			runner := NewMock_transactionRunner(ctrl)

			context := &runContext{
				signer:   types.LatestSignerForChainID(big.NewInt(1)),
				upgrades: opera.Upgrades{Brio: true, GasSubsidies: true, TransactionBundles: true},
				runner:   runner,
			}
			tc.setupMock(context, runner)

			summary := runTransactions(context, []*types.Transaction{tc.tx}, 0, math.MaxUint64)
			require.Equal(t, tc.wantCausedBy, summary.CausedBy)
		})
	}
}

func TestRunTransactions_SmallerTransactionsAreProcessedAfterLargeTransactionExceedingSizeLimit(t *testing.T) {
	sizeLimit := uint64(1_000)

	// Create a tx that exceeds the size limit
	largeTx := types.NewTx(&types.LegacyTx{Data: make([]byte, sizeLimit+1)})

	// Following txs are not skipped
	tx0 := getRegularTransaction(t)
	tx1 := getRegularTransaction(t)

	transactions := []*types.Transaction{largeTx, tx0, tx1}

	ctrl := gomock.NewController(t)
	runner := NewMock_transactionRunner(ctrl)

	context := &runContext{
		signer:   types.LatestSignerForChainID(big.NewInt(1)),
		upgrades: opera.Upgrades{Brio: true, GasSubsidies: true, TransactionBundles: true},
		runner:   runner,
	}

	runner.EXPECT().runRegularTransaction(context, largeTx, 0, sizeLimit).Return(
		ProcessedTransaction{Transaction: largeTx, Receipt: nil},
		core_types.TransactionResultSuccessful,
	)
	runner.EXPECT().runRegularTransaction(context, tx0, 0, sizeLimit).Return(
		ProcessedTransaction{Transaction: tx0, Receipt: &types.Receipt{}},
		core_types.TransactionResultInvalid,
	)
	runner.EXPECT().runRegularTransaction(context, tx1, 1, sizeLimit-tx0.Size()).Return(
		ProcessedTransaction{Transaction: tx1, Receipt: &types.Receipt{}},
		core_types.TransactionResultInvalid,
	)

	summary := runTransactions(context, transactions, 0, sizeLimit)
	require.Equal(t, []ProcessedTransaction{
		{Transaction: largeTx, Receipt: nil},
		{Transaction: tx0, Receipt: &types.Receipt{}},
		{Transaction: tx1, Receipt: &types.Receipt{}},
	}, summary.ProcessedTransactions)
}

func TestRunTransactions_RemainingSizeIsSetToZeroWhenProcessedTxExceedsIt(t *testing.T) {
	tx0 := getRegularTransaction(t)
	tx1 := getRegularTransaction(t)

	// Set remaining size smaller than tx0's size. The mock will still return a
	// receipt (simulating a transaction that passed the runner's size check but
	// exceeds the tracked remaining budget).
	remainingSize := tx0.Size() - 1

	ctrl := gomock.NewController(t)
	runner := NewMock_transactionRunner(ctrl)

	context := &runContext{
		signer:   types.LatestSignerForChainID(big.NewInt(1)),
		upgrades: opera.Upgrades{Brio: true, GasSubsidies: true, TransactionBundles: true},
		runner:   runner,
	}

	gomock.InOrder(
		runner.EXPECT().runRegularTransaction(context, tx0, 0, remainingSize).Return(
			ProcessedTransaction{Transaction: tx0, Receipt: &types.Receipt{}},
			core_types.TransactionResultSuccessful,
		),

		// After tx0 exceeds remaining size, remainingSize becomes 0.
		// The next transaction should be called with sizeLimit = 0.
		runner.EXPECT().runRegularTransaction(context, tx1, 1, uint64(0)).Return(
			ProcessedTransaction{Transaction: tx1, Receipt: nil},
			core_types.TransactionResultInvalid,
		),
	)

	summary := runTransactions(context, []*types.Transaction{tx0, tx1}, 0, remainingSize)
	require.Len(t, summary.ProcessedTransactions, 2)
	require.NotNil(t, summary.ProcessedTransactions[0].Receipt)
	require.Nil(t, summary.ProcessedTransactions[1].Receipt)
}

func TestRunTransactions_AccumulatesMetricsForBundles(t *testing.T) {

	t.Run("executed bundles are counted", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		runner := NewMock_transactionRunner(ctrl)
		mockMetrics := NewMockBlockExecutionMetrics(ctrl)

		context := &runContext{
			signer:   types.LatestSignerForChainID(big.NewInt(1)),
			runner:   runner,
			upgrades: opera.Upgrades{Brio: true, TransactionBundles: true},
			metrics:  mockMetrics,
		}

		txs := []*types.Transaction{getTransactionBundle(t), getTransactionBundle(t)}

		runner.EXPECT().runTransactionBundle(context, txs[0], 0, gomock.Any()).Return(
			[]ProcessedTransaction{{Transaction: txs[0], Receipt: &types.Receipt{GasUsed: 100}}},
			core_types.TransactionResultSuccessful,
			core_types.ExecutionCost(100),
		)
		runner.EXPECT().runTransactionBundle(context, txs[1], 1, gomock.Any()).Return(
			[]ProcessedTransaction{{Transaction: txs[1], Receipt: &types.Receipt{GasUsed: 200}}},
			core_types.TransactionResultSuccessful,
			core_types.ExecutionCost(200),
		)
		mockMetrics.EXPECT().IncExecutedBundle().Times(2)
		mockMetrics.EXPECT().ObserveBundleEfficiency(uint64(100), uint64(100))
		mockMetrics.EXPECT().ObserveBundleEfficiency(uint64(200), uint64(200))

		runTransactions(context, txs, 0, math.MaxUint64)
	})

	t.Run("rolled back bundles are counted when result is TransactionResultFailed", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		runner := NewMock_transactionRunner(ctrl)
		mockMetrics := NewMockBlockExecutionMetrics(ctrl)

		context := &runContext{
			signer:   types.LatestSignerForChainID(big.NewInt(1)),
			runner:   runner,
			upgrades: opera.Upgrades{Brio: true, TransactionBundles: true},
			metrics:  mockMetrics,
		}

		txs := []*types.Transaction{getTransactionBundle(t)}

		runner.EXPECT().runTransactionBundle(context, txs[0], 0, gomock.Any()).Return(
			[]ProcessedTransaction{{Transaction: txs[0], Receipt: nil}},
			core_types.TransactionResultFailed,
			core_types.ExecutionCost(50),
		)
		mockMetrics.EXPECT().IncRolledBackBundle()
		mockMetrics.EXPECT().ObserveBundleEfficiency(uint64(0), uint64(50))

		runTransactions(context, txs, 0, math.MaxUint64)
	})

	t.Run("invalid bundles are counted when result is TransactionResultInvalid", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		runner := NewMock_transactionRunner(ctrl)
		mockMetrics := NewMockBlockExecutionMetrics(ctrl)

		context := &runContext{
			signer:   types.LatestSignerForChainID(big.NewInt(1)),
			runner:   runner,
			upgrades: opera.Upgrades{Brio: true, TransactionBundles: true},
			metrics:  mockMetrics,
		}

		txs := []*types.Transaction{getTransactionBundle(t)}

		runner.EXPECT().runTransactionBundle(context, txs[0], 0, gomock.Any()).Return(
			[]ProcessedTransaction{{Transaction: txs[0], Receipt: nil}},
			core_types.TransactionResultInvalid,
			core_types.ExecutionCost(50),
		)
		mockMetrics.EXPECT().IncInvalidBundle()
		mockMetrics.EXPECT().ObserveBundleEfficiency(uint64(0), uint64(50))

		runTransactions(context, txs, 0, math.MaxUint64)
	})

	t.Run("efficiency is reported for executed bundles", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		runner := NewMock_transactionRunner(ctrl)
		mockMetrics := NewMockBlockExecutionMetrics(ctrl)

		context := &runContext{
			signer:   types.LatestSignerForChainID(big.NewInt(1)),
			runner:   runner,
			upgrades: opera.Upgrades{Brio: true, TransactionBundles: true},
			metrics:  mockMetrics,
		}

		txs := []*types.Transaction{getTransactionBundle(t)}

		runner.EXPECT().runTransactionBundle(context, txs[0], 0, gomock.Any()).Return(
			[]ProcessedTransaction{{Transaction: txs[0], Receipt: &types.Receipt{GasUsed: 300}}},
			core_types.TransactionResultSuccessful,
			core_types.ExecutionCost(1000),
		)
		// efficiency = gasUsed / execCost = 300 / 1000
		mockMetrics.EXPECT().IncExecutedBundle()
		mockMetrics.EXPECT().ObserveBundleEfficiency(uint64(300), uint64(1000))

		runTransactions(context, txs, 0, math.MaxUint64)
	})
}

func TestRunTransactions_AccumulatesMetricsForSponsoredTx(t *testing.T) {

	t.Run("executed sponsorships are counted", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		runner := NewMock_transactionRunner(ctrl)
		mockMetrics := NewMockBlockExecutionMetrics(ctrl)

		context := &runContext{
			signer:   types.LatestSignerForChainID(big.NewInt(1)),
			runner:   runner,
			upgrades: opera.Upgrades{Brio: true, GasSubsidies: true},
			metrics:  mockMetrics,
		}

		tx := getSponsorshipRequest(t)
		txs := []*types.Transaction{tx}

		runner.EXPECT().runSponsoredTransaction(context, tx, 0, gomock.Any()).Return(
			[]ProcessedTransaction{
				{Transaction: tx, Receipt: &types.Receipt{GasUsed: 100}},
			},
			core_types.TransactionResultSuccessful,
		)
		mockMetrics.EXPECT().IncSponsoredTx()

		runTransactions(context, txs, 0, math.MaxUint64)
	})

	t.Run("skipped sponsorships are counted", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		runner := NewMock_transactionRunner(ctrl)
		mockMetrics := NewMockBlockExecutionMetrics(ctrl)

		context := &runContext{
			signer:   types.LatestSignerForChainID(big.NewInt(1)),
			runner:   runner,
			upgrades: opera.Upgrades{Brio: true, GasSubsidies: true},
			metrics:  mockMetrics,
		}

		tx := getSponsorshipRequest(t)
		txs := []*types.Transaction{tx}

		runner.EXPECT().runSponsoredTransaction(context, tx, 0, gomock.Any()).Return(
			[]ProcessedTransaction{
				{Transaction: tx, Receipt: nil},
			},
			core_types.TransactionResultFailed,
		)
		mockMetrics.EXPECT().IncSkippedSponsoredTx()

		runTransactions(context, txs, 0, math.MaxUint64)
	})

	t.Run("executed and skipped are exclusive", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		runner := NewMock_transactionRunner(ctrl)
		mockMetrics := NewMockBlockExecutionMetrics(ctrl)

		context := &runContext{
			signer:   types.LatestSignerForChainID(big.NewInt(1)),
			runner:   runner,
			upgrades: opera.Upgrades{Brio: true, GasSubsidies: true},
			metrics:  mockMetrics,
		}

		tx1 := getSponsorshipRequest(t)
		tx2 := getSponsorshipRequest(t)
		txs := []*types.Transaction{tx1, tx2}

		runner.EXPECT().runSponsoredTransaction(context, tx1, 0, gomock.Any()).Return(
			[]ProcessedTransaction{
				{Transaction: tx1, Receipt: &types.Receipt{GasUsed: 100}},
			},
			core_types.TransactionResultSuccessful,
		)
		runner.EXPECT().runSponsoredTransaction(context, tx2, 1, gomock.Any()).Return(
			[]ProcessedTransaction{
				{Transaction: tx2, Receipt: nil},
			},
			core_types.TransactionResultFailed,
		)
		// One goes to executed, the other to skipped - they are exclusive
		mockMetrics.EXPECT().IncSponsoredTx().Times(1)
		mockMetrics.EXPECT().IncSkippedSponsoredTx().Times(1)

		runTransactions(context, txs, 0, math.MaxUint64)
	})
}

func TestRunTransactions_AccumulatesMetricsForBundlesAndSponsoredTx(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := NewMock_transactionRunner(ctrl)
	mockMetrics := NewMockBlockExecutionMetrics(ctrl)

	context := &runContext{
		signer:   types.LatestSignerForChainID(big.NewInt(1)),
		runner:   runner,
		upgrades: opera.Upgrades{Brio: true, GasSubsidies: true, TransactionBundles: true},
		metrics:  mockMetrics,
	}

	sponsoredTx := getSponsorshipRequest(t)
	skippedSponsoredTx := getSponsorshipRequest(t)
	executedBundle := getTransactionBundle(t)
	rolledBackBundle := getTransactionBundle(t)
	txs := []*types.Transaction{sponsoredTx, executedBundle, skippedSponsoredTx, rolledBackBundle}

	// Inner transactions of bundles are regular txs (non-zero gas price),
	// not the envelope itself.
	bundleInnerTx1 := getRegularTransaction(t)
	bundleInnerTx2 := getRegularTransaction(t)

	gomock.InOrder(
		runner.EXPECT().runSponsoredTransaction(context, sponsoredTx, 0, gomock.Any()).Return(
			[]ProcessedTransaction{
				{Transaction: sponsoredTx, Receipt: &types.Receipt{GasUsed: 100}},
			},
			core_types.TransactionResultSuccessful,
		),
		runner.EXPECT().runTransactionBundle(context, executedBundle, 1, gomock.Any()).Return(
			[]ProcessedTransaction{{Transaction: bundleInnerTx1, Receipt: &types.Receipt{GasUsed: 500}}},
			core_types.TransactionResultSuccessful,
			core_types.ExecutionCost(800),
		),
		runner.EXPECT().runSponsoredTransaction(context, skippedSponsoredTx, 2, gomock.Any()).Return(
			[]ProcessedTransaction{
				{Transaction: skippedSponsoredTx, Receipt: nil},
			},
			core_types.TransactionResultFailed,
		),
		runner.EXPECT().runTransactionBundle(context, rolledBackBundle, 2, gomock.Any()).Return(
			[]ProcessedTransaction{{Transaction: bundleInnerTx2, Receipt: nil}},
			core_types.TransactionResultFailed,
			core_types.ExecutionCost(300),
		),
	)

	// Sponsored tx metrics
	mockMetrics.EXPECT().IncSponsoredTx()
	mockMetrics.EXPECT().IncSkippedSponsoredTx()

	// Bundle metrics
	mockMetrics.EXPECT().IncExecutedBundle()
	mockMetrics.EXPECT().IncRolledBackBundle()

	// Efficiency: executed bundle = 500/800, rolled-back bundle has gasUsed=0, execCost=300
	mockMetrics.EXPECT().ObserveBundleEfficiency(uint64(500), uint64(800))
	mockMetrics.EXPECT().ObserveBundleEfficiency(uint64(0), uint64(300))

	runTransactions(context, txs, 0, math.MaxUint64)
}

func TestRunTransactions_SkipsMetricsWithoutUpgrades(t *testing.T) {

	t.Run("sponsorship request before upgrade", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		runner := NewMock_transactionRunner(ctrl)
		mockMetrics := NewMockBlockExecutionMetrics(ctrl)

		context := &runContext{
			signer:   types.LatestSignerForChainID(big.NewInt(1)),
			runner:   runner,
			upgrades: opera.GetAllegroUpgrades(),
			metrics:  mockMetrics,
		}

		tx := getSponsorshipRequest(t)
		txs := []*types.Transaction{tx}

		runner.EXPECT().runRegularTransaction(context, tx, 0, gomock.Any()).Return(
			ProcessedTransaction{Transaction: tx},
			core_types.TransactionResultInvalid,
		)

		runTransactions(context, txs, 0, math.MaxUint64)
	})

	t.Run("bundle transaction before upgrade", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		runner := NewMock_transactionRunner(ctrl)
		mockMetrics := NewMockBlockExecutionMetrics(ctrl)

		context := &runContext{
			signer:   types.LatestSignerForChainID(big.NewInt(1)),
			runner:   runner,
			upgrades: opera.GetAllegroUpgrades(),
			metrics:  mockMetrics,
		}

		tx := getTransactionBundle(t)
		txs := []*types.Transaction{tx}

		runner.EXPECT().runRegularTransaction(context, tx, 0, gomock.Any()).Return(
			ProcessedTransaction{Transaction: tx, Receipt: &types.Receipt{GasUsed: 100}},
			core_types.TransactionResultSuccessful,
		)

		runTransactions(context, txs, 0, math.MaxUint64)
	})
}

func TestStateProcessor_MetricsAreEnabledInSingleBlockProposerMode(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockMetrics := NewMockBlockExecutionMetrics(ctrl)

	chain := NewMockDummyChain(ctrl)
	stateDb := state.NewMockStateDB(ctrl)

	processor := NewStateProcessorForHeadState(
		&params.ChainConfig{},
		chain,
		opera.Upgrades{},
		mockMetrics,
	)

	block := &EvmBlock{EvmHeader: EvmHeader{Number: big.NewInt(1)}}
	tp := processor.BeginBlock(block, stateDb, vm.Config{}, math.MaxUint64)

	require.Equal(t, mockMetrics, tp.metrics,
		"metrics must be forwarded from StateProcessor to TransactionProcessor via BeginBlock")
}

func TestTxAsMessage_ConvertsInternalTransactionToMessage(t *testing.T) {

	tx := types.NewTx(&types.LegacyTx{
		Nonce:    0,
		To:       &common.Address{1},
		Gas:      21_000,
		GasPrice: big.NewInt(1),
		Value:    big.NewInt(100),
		Data:     []byte{0x01, 0x02},
	})

	msg, err := TxAsMessage(tx, nil, big.NewInt(1))
	require.NoError(t, err)
	require.Equal(t, common.Address{}, msg.From)
	require.Equal(t, tx.To(), msg.To)
	require.Equal(t, tx.Gas(), msg.GasLimit)
	require.Equal(t, tx.GasPrice(), msg.GasPrice)
	require.Equal(t, tx.Value(), msg.Value)
	require.Equal(t, tx.Data(), msg.Data)
}

func TestTxAsMessage_ConvertsUserTransactionsToMessage(t *testing.T) {
	chainId := big.NewInt(1)
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	signer := types.LatestSignerForChainID(chainId)
	sender := crypto.PubkeyToAddress(key.PublicKey)

	accessList := types.AccessList{
		{
			Address:     common.Address{2},
			StorageKeys: []common.Hash{{1}, {2}},
		},
	}

	setCodeAuthorizations := []types.SetCodeAuthorization{
		{
			Address: common.Address{3},
			Nonce:   1,
		},
	}

	test := map[string]struct {
		tx           types.TxData
		expectations func(*testing.T, core.Message)
	}{
		"legacy transaction": {
			tx: &types.LegacyTx{
				Nonce:    0,
				To:       &common.Address{1},
				Gas:      21_000,
				GasPrice: big.NewInt(1),
				Value:    big.NewInt(100),
				Data:     []byte{0x01, 0x02},
			},
		},
		"access list transaction": {
			tx: &types.AccessListTx{
				Nonce:      0,
				To:         &common.Address{1},
				Gas:        21_000,
				GasPrice:   big.NewInt(1),
				Value:      big.NewInt(100),
				Data:       []byte{0x01, 0x02},
				AccessList: accessList,
			},
			expectations: func(t *testing.T, msg core.Message) {
				require.Equal(t, accessList, msg.AccessList)
			},
		},
		"dynamic fee transaction": {
			tx: &types.DynamicFeeTx{
				Nonce:      0,
				To:         &common.Address{1},
				Gas:        21_000,
				GasFeeCap:  big.NewInt(1),
				GasTipCap:  big.NewInt(2),
				Value:      big.NewInt(100),
				Data:       []byte{0x01, 0x02},
				AccessList: accessList,
			},
			expectations: func(t *testing.T, msg core.Message) {
				require.Equal(t, big.NewInt(1), msg.GasFeeCap)
				require.Equal(t, big.NewInt(2), msg.GasTipCap)
				require.Equal(t, accessList, msg.AccessList)
			},
		},
		"blob transaction with blob hashes": {
			tx: &types.BlobTx{
				Nonce:      0,
				To:         common.Address{1},
				Gas:        21_000,
				GasFeeCap:  uint256.NewInt(1),
				GasTipCap:  uint256.NewInt(2),
				Value:      uint256.NewInt(100),
				Data:       []byte{0x01, 0x02},
				AccessList: accessList,
				BlobHashes: []common.Hash{
					{1}, {2},
				},
			},
			expectations: func(t *testing.T, msg core.Message) {
				require.Equal(t, big.NewInt(1), msg.GasFeeCap)
				require.Equal(t, big.NewInt(2), msg.GasTipCap)
				require.Equal(t, accessList, msg.AccessList)
				require.Equal(t, msg.BlobHashes, []common.Hash{{1}, {2}},
					"Sonic rejects blobTxs with non-empty blob hashes, but this layer allows them for history replays")
			},
		},
		"blob transaction with nil blob hashes": {
			tx: &types.BlobTx{
				Nonce:      0,
				To:         common.Address{1},
				Gas:        21_000,
				GasFeeCap:  uint256.NewInt(1),
				GasTipCap:  uint256.NewInt(2),
				Value:      uint256.NewInt(100),
				AccessList: accessList,
				Data:       []byte{0x01, 0x02},
			},
			expectations: func(t *testing.T, msg core.Message) {
				require.Equal(t, big.NewInt(1), msg.GasFeeCap)
				require.Equal(t, big.NewInt(2), msg.GasTipCap)
				require.Equal(t, accessList, msg.AccessList)
				require.Nil(t, msg.BlobHashes, "Sonic allows empty blob txs, go-ethereum allows processing nil")
			},
		},
		"blob transaction with empty blob hashes": {
			tx: &types.BlobTx{
				Nonce:      0,
				To:         common.Address{1},
				Gas:        21_000,
				GasFeeCap:  uint256.NewInt(1),
				GasTipCap:  uint256.NewInt(2),
				Value:      uint256.NewInt(100),
				Data:       []byte{0x01, 0x02},
				AccessList: accessList,
				BlobHashes: []common.Hash{},
			},
			expectations: func(t *testing.T, msg core.Message) {
				require.Equal(t, big.NewInt(1), msg.GasFeeCap)
				require.Equal(t, big.NewInt(2), msg.GasTipCap)
				require.Equal(t, accessList, msg.AccessList)
				require.Nil(t, msg.BlobHashes, "Sonic allows empty blob txs, go-ethereum allows processing nil")
			},
		},
		"set code transaction": {
			tx: &types.SetCodeTx{
				Nonce:      0,
				To:         common.Address{1},
				Gas:        21_000,
				Value:      uint256.NewInt(100),
				Data:       []byte{0x01, 0x02},
				AccessList: accessList,
				AuthList:   setCodeAuthorizations,
			},
			expectations: func(t *testing.T, msg core.Message) {
				require.Equal(t, common.Address{1}, *msg.To)
				require.Equal(t, accessList, msg.AccessList)
				require.Equal(t, setCodeAuthorizations, msg.SetCodeAuthorizations)
			},
		},
		"set code transaction, empty auth list": {
			tx: &types.SetCodeTx{
				Nonce:      0,
				To:         common.Address{1},
				Gas:        21_000,
				Value:      uint256.NewInt(100),
				Data:       []byte{0x01, 0x02},
				AccessList: accessList,
			},
			expectations: func(t *testing.T, msg core.Message) {
				require.Equal(t, common.Address{1}, *msg.To)
				require.Equal(t, accessList, msg.AccessList)
				require.Empty(t, msg.SetCodeAuthorizations)
			},
		},
	}

	for name, tc := range test {
		t.Run(name, func(t *testing.T) {
			basefee := big.NewInt(1)
			tx := types.MustSignNewTx(key, signer, tc.tx)
			msg, err := TxAsMessage(tx, signer, basefee)
			require.NoError(t, err)
			require.NotNil(t, msg)

			require.Equal(t, sender, msg.From)
			require.Equal(t, tx.To(), msg.To)
			require.Equal(t, tx.GasPrice(), msg.GasPrice)
			require.Equal(t, tx.GasFeeCap(), msg.GasFeeCap)
			require.Equal(t, tx.GasTipCap(), msg.GasTipCap)
			require.Equal(t, tx.Value(), msg.Value)
			require.Equal(t, tx.Gas(), msg.GasLimit)
			require.Equal(t, tx.Data(), msg.Data)

			if tc.expectations != nil {
				tc.expectations(t, *msg)
			}
		})
	}
}

func mockStateDbTransactionExecution(stateDb *state.MockStateDB) {
	any := gomock.Any()
	stateDb.EXPECT().GetBalance(any).Return(uint256.NewInt(math.MaxInt64)).AnyTimes()
	stateDb.EXPECT().AddBalance(any, any, any).AnyTimes()
	stateDb.EXPECT().SubBalance(any, any, any).AnyTimes()
	stateDb.EXPECT().Prepare(any, any, any, any, any, any).AnyTimes()
	stateDb.EXPECT().GetNonce(any).AnyTimes()
	stateDb.EXPECT().SetNonce(any, any, any).AnyTimes()
	stateDb.EXPECT().GetCodeHash(any).Return(types.EmptyCodeHash).AnyTimes()
	stateDb.EXPECT().GetCode(any).AnyTimes()
	stateDb.EXPECT().GetStorageRoot(any).Return(types.EmptyRootHash).AnyTimes()
	stateDb.EXPECT().Snapshot().AnyTimes()
	stateDb.EXPECT().Exist(any).Return(true).AnyTimes()
	stateDb.EXPECT().GetRefund().AnyTimes()
	stateDb.EXPECT().EndTransaction().AnyTimes()
}
