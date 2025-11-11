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

package evmcore

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies/registry"
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
	usedGas *uint64, onNewLog func(*types.Log),
) []ProcessedTransaction {
	// This implementation is a wrapper around the BeginBlock function, which
	// handles the actual transaction processing.
	txProcessor := p.BeginBlock(block, stateDb, cfg, gasLimit, onNewLog)
	processed := make([]ProcessedTransaction, 0, len(block.Transactions))
	for i, tx := range block.Transactions {
		processed = append(processed, txProcessor.Run(i, tx)...)
	}

	// The used gas is the cumulative gas used reported by the last receipt.
	for _, tx := range processed {
		if tx.Receipt != nil {
			*usedGas = tx.Receipt.CumulativeGasUsed
		}
	}

	return processed
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
	processor := NewStateProcessor(&chainConfig, chain, opera.Upgrades{})

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

			reportedLogs := []*types.Log{}
			onLog := func(log *types.Log) {
				reportedLogs = append(reportedLogs, log)
			}

			vmConfig := vm.Config{}
			gasLimit := uint64(blockGasLimit)
			usedGas := new(uint64)
			processed := process(block, state, vmConfig, gasLimit, usedGas, onLog)

			// Receipts should be set accordingly.
			require.Len(processed, len(transactions))
			require.Equal(transactions[0], processed[0].Transaction)
			require.Equal(transactions[1], processed[1].Transaction)
			require.Equal(transactions[2], processed[2].Transaction)
			require.Equal(transactions[3], processed[3].Transaction)

			logMsg0 := &types.Log{Address: common.Address{0}, TxIndex: 0}
			logMsg2 := &types.Log{Address: common.Address{2}, TxIndex: 2}

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
				Logs: []*types.Log{logMsg0},
			}, processed[0].Receipt)

			require.Nil(processed[1].Receipt)

			require.NotNil(processed[2].Receipt)
			require.Equal(&types.Receipt{
				Status:            types.ReceiptStatusSuccessful,
				GasUsed:           21_000,
				CumulativeGasUsed: 42_000,
				BlockNumber:       block.Number,
				TransactionIndex:  2, // TODO: this should be 1; this needs to be investigated
				TxHash:            transactions[2].Hash(),
				Bloom: types.CreateBloom(&types.Receipt{
					Logs: []*types.Log{logMsg2},
				}),
				Logs: []*types.Log{logMsg2},
			}, processed[2].Receipt)

			require.Nil(processed[3].Receipt)

			require.Equal([]*types.Log{logMsg0, logMsg2}, reportedLogs)

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

	logMsg1 := &types.Log{Address: common.Address{1}, TxIndex: 1}

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
	processor := NewStateProcessor(&chainConfig, chain, opera.Upgrades{})
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
			processed := process(block, state, vmConfig, gasLimit, usedGas, nil)

			require.Len(processed, len(transactions))
			require.Equal(transactions[0], processed[0].Transaction)
			require.Equal(transactions[1], processed[1].Transaction)

			require.Nil(processed[0].Receipt)
			require.Equal(&types.Receipt{
				Status:            types.ReceiptStatusSuccessful,
				GasUsed:           21_000,
				CumulativeGasUsed: 21_000,
				BlockNumber:       block.Number,
				TransactionIndex:  1, // Even though the first tx is skipped, the index is still 1
				TxHash:            transactions[1].Hash(),
				Bloom: types.CreateBloom(&types.Receipt{
					Logs: []*types.Log{logMsg1},
				}),
				Logs: []*types.Log{logMsg1},
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

		processor := NewStateProcessor(&chainConfig, chain, opera.Upgrades{})

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
						state.EXPECT().SubBalance(any, any, any),
						state.EXPECT().AddBalance(any, any, any),
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
				processed := process(block, state, vmConfig, gasLimit, usedGas, nil)
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
	processor := NewStateProcessor(&chainConfig, chain, opera.Upgrades{})

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
	processed := processor.Process(block, state, vm.Config{}, gasLimit, usedGas, nil)

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
	processor := NewStateProcessor(&chainConfig, chain, opera.Upgrades{})

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
					processed := process(block, state, vmConfig, gasLimit, usedGas, nil)
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
			gp := new(core.GasPool).AddGas(1000000)

			// The transaction will fail for various reasons, but for this test
			// this is not relevant. We just want to check if the base fee
			// configuration flag is updated to match the SkipAccountChecks flag.
			_, _, err := applyTransaction(&core.Message{
				SkipNonceChecks:       internal,
				SkipTransactionChecks: internal,
				GasPrice:              big.NewInt(0),
				Value:                 big.NewInt(0),
			}, gp, state, nil, nil, nil, evm, nil)
			if err == nil {
				t.Errorf("expected transaction to fail")
			}

			if want, got := internal, evm.Config.NoBaseFee; want != got {
				t.Fatalf("want %v, got %v", want, got)
			}
		})
	}
}

func TestApplyTransaction_BlobHashesNotSupportedAndSkipped(t *testing.T) {
	ctrl := gomock.NewController(t)
	state := state.NewMockStateDB(ctrl)
	evm := vm.NewEVM(vm.BlockContext{}, state, &params.ChainConfig{}, vm.Config{})
	gp := new(core.GasPool).AddGas(1000000)

	state.EXPECT().EndTransaction()

	msg := &core.Message{
		From:       common.Address{1},
		To:         &common.Address{2},
		GasLimit:   21000,
		GasPrice:   big.NewInt(1),
		BlobHashes: []common.Hash{{0x01}},
	}
	usedGas := uint64(0)
	receipt, gasUsed, err :=
		applyTransaction(msg, gp, state, big.NewInt(1), nil, &usedGas, evm, nil)
	require.ErrorContains(t, err, "blob data is not supported")
	require.Nil(t, receipt)
	require.Equal(t, uint64(0), gasUsed)
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
			gp := new(core.GasPool).AddGas(1000000)

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
				applyTransaction(msg, gp, state, blockNumber, nil, new(uint64), evm, nil)
			require.ErrorContains(t, err, "max initcode size exceeded")
			require.Nil(t, receipt)
			require.Equal(t, uint64(0), gasUsed)
		})
	}
}

// processFunction is a function type alias for the StateProcessor's Process
// function to allow side-by-side testing of different implementations.
type processFunction = func(
	block *EvmBlock,
	statedb state.StateDB,
	cfg vm.Config,
	gasLimit uint64,
	usedGas *uint64,
	onNewLog func(*types.Log),
) []ProcessedTransaction

func getStateDbMockForTransactions(
	ctrl *gomock.Controller,
	transactions []*types.Transaction,
) *state.MockStateDB {
	// Allow basically everything, but expect the context to be set up for
	// the given transactions and their positions.
	state := state.NewMockStateDB(ctrl)
	txIndex := new(int)
	for i, tx := range transactions {
		state.EXPECT().SetTxContext(tx.Hash(), i).Do(
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

func TestRunTransactions_GasSubsidiesDisabled_ProcessesRegularTransaction(t *testing.T) {
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
			runner.EXPECT().runRegularTransaction(context, tx, 0)
			runTransactions(context, []*types.Transaction{tx}, 0)
		})
	}
}

func TestRunTransactions_GasSubsidiesEnabled_RunsRegularTransactionWithoutSponsorship(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := NewMock_transactionRunner(ctrl)

	tx := getRegularTransaction(t)
	processed := ProcessedTransaction{
		Transaction: tx,
	}

	context := &runContext{
		runner:   runner,
		upgrades: opera.Upgrades{GasSubsidies: true},
	}
	runner.EXPECT().runRegularTransaction(context, tx, 0).Return(processed)
	got := runTransactions(context, []*types.Transaction{tx}, 0)
	require.Equal(t, []ProcessedTransaction{processed}, got)
}

func TestRunTransactions_GasSubsidiesEnabled_RunsSponsorshipRequestWithSponsorship(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := NewMock_transactionRunner(ctrl)

	tx := getSponsorshipRequest(t)

	context := &runContext{
		runner:   runner,
		upgrades: opera.Upgrades{GasSubsidies: true},
	}
	runner.EXPECT().runSponsoredTransaction(context, tx, 0).Return([]ProcessedTransaction{{
		Transaction: tx,
		Receipt:     nil,
	}})
	processed := runTransactions(context, []*types.Transaction{tx}, 0)
	require.Len(t, processed, 1)
	require.Equal(t, tx, processed[0].Transaction)
	require.Nil(t, processed[0].Receipt)
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

			gasPool := new(core.GasPool).AddGas(test.availableGas)
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
			got := runner.runSponsoredTransaction(context, tx, 0)

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

	gasPool := new(core.GasPool).AddGas(1_000_000)
	context := &runContext{
		statedb:  state,
		gasPool:  gasPool,
		upgrades: opera.Upgrades{GasSubsidies: false}, // < nothing is covered
	}

	runner := &transactionRunner{}
	got := runner.runSponsoredTransaction(context, tx, 0)
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

	gasPool := new(core.GasPool).AddGas(1_000_000)
	context := &runContext{
		statedb:  state,
		signer:   types.LatestSignerForChainID(nil),
		baseFee:  big.NewInt(1),
		gasPool:  gasPool,
		upgrades: opera.Upgrades{GasSubsidies: true},
	}

	runner := &transactionRunner{evm: evm}
	got := runner.runSponsoredTransaction(context, tx, 0)
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

	gasPool := new(core.GasPool).AddGas(1_000_000)
	context := &runContext{
		statedb:  state,
		signer:   types.LatestSignerForChainID(nil),
		baseFee:  big.NewInt(1),
		gasPool:  gasPool,
		upgrades: opera.Upgrades{GasSubsidies: true},
	}

	runner := &transactionRunner{evm: evm}
	got := runner.runSponsoredTransaction(context, tx, 0)
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

	gasPool := new(core.GasPool).AddGas(1_000_000)
	context := &runContext{
		statedb:  state,
		signer:   types.LatestSignerForChainID(nil),
		baseFee:  gasPrice,
		gasPool:  gasPool,
		upgrades: opera.Upgrades{GasSubsidies: true},
	}

	runner := &transactionRunner{evm: evm}
	got := runner.runSponsoredTransaction(context, tx, 0)
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

	gasPool := new(core.GasPool).AddGas(1_000_000)
	context := &runContext{
		statedb:  state,
		signer:   types.LatestSignerForChainID(nil),
		baseFee:  big.NewInt(1),
		gasPool:  gasPool,
		upgrades: opera.Upgrades{GasSubsidies: true},
	}

	runner := &transactionRunner{evm: evm}
	got := runner.runSponsoredTransaction(context, tx, 0)
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

	gasPool := new(core.GasPool).AddGas(1_000_000)
	context := &runContext{
		statedb:  state,
		signer:   types.LatestSignerForChainID(nil),
		baseFee:  big.NewInt(1),
		gasPool:  gasPool,
		upgrades: opera.Upgrades{GasSubsidies: true},
	}

	runner := &transactionRunner{evm: evm}
	got := runner.runSponsoredTransaction(context, tx, 0)
	want := []ProcessedTransaction{
		processedSponsoredTransaction,
		skippedFeeDeductionTransaction,
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

	gasPool := new(core.GasPool).AddGas(1_000_000)
	context := &runContext{
		statedb:  state,
		signer:   types.LatestSignerForChainID(nil),
		baseFee:  big.NewInt(1),
		gasPool:  gasPool,
		upgrades: opera.Upgrades{GasSubsidies: true},
	}

	runner := &transactionRunner{evm: evm}
	got := runner.runSponsoredTransaction(context, tx, txIndex)
	require.Len(t, got, 2)
	require.Equal(t, tx, got[0].Transaction)
	require.NotNil(t, got[0].Receipt)
	require.NotNil(t, got[1].Transaction)
	require.NotNil(t, got[1].Receipt)
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
		Transfer: func(_ vm.StateDB, _ common.Address, _ common.Address, amount *uint256.Int) {
			// do nothing
		},
		CanTransfer: func(_ vm.StateDB, _ common.Address, amount *uint256.Int) bool {
			return true
		},
		Random: &common.Hash{}, // < signals Revision >= Merge
	}
	vm := vm.NewEVM(blockContext, state, chainConfig, vmConfig)
	runner := &transactionRunner{evm{vm}}

	gasPool := new(core.GasPool).AddGas(1_000_000)
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

	processedTransactions := runner.runSponsoredTransaction(context, tx, txIndex)

	// the transaction should be sponsored successfully
	require.Len(processedTransactions, 2)
	require.Equal(tx, processedTransactions[0].Transaction)
	require.NotNil(processedTransactions[0].Receipt)

	// the fee deduction transaction should be the second one
	require.NotNil(processedTransactions[1].Transaction)
	callData := processedTransactions[1].Transaction.Data()
	require.Equal(4+2*32, len(callData)) // chooseFund + deductFees

	fundId := subsidies.FundId(callData[4:])
	gasUsed := processedTransactions[0].Receipt.GasUsed
	gasPrice := baseFee // gas price is base fee for sponsored tx

	gasConfig := subsidies.GasConfig{ // < values hard-coded in dev version of the registry
		SponsorshipOverheadGasCost: 210_000,
		DeductFeesGasCost:          60_000,
	}
	feeDeductionTx, err := subsidies.GetFeeChargeTransaction(state, fundId, gasConfig, gasUsed, gasPrice)
	require.NoError(err)
	got := processedTransactions[1].Transaction
	require.Equal(feeDeductionTx.Hash(), got.Hash())
	require.NotNil(processedTransactions[1].Receipt)
	require.Equal(types.ReceiptStatusSuccessful, processedTransactions[1].Receipt.Status)

	// the total gas usage should be the sum of both transactions
	require.Equal(
		processedTransactions[0].Receipt.GasUsed+processedTransactions[1].Receipt.GasUsed,
		*usedGas,
	)
}

func TestRunTransaction_InternalTransactions_SkipsTransactionChecksTrue(t *testing.T) {

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
		Transfer: func(_ vm.StateDB, _ common.Address, _ common.Address, amount *uint256.Int) {
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
	gasPool := new(core.GasPool).AddGas(maxTxGas * 3)
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
	got := runner.runRegularTransaction(context, unsignedTx, 0)

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
	got = runner.runRegularTransaction(context, regularTx, 0)
	require.Equal(t, regularTx, got.Transaction)
	require.Nil(t, got.Receipt)
}

func TestRunTransaction_RegularTransaction(t *testing.T) {

	tests := map[string]struct {
		rules      opera.Rules
		stateSetup func(state *state.MockStateDB)
		validation func(t *testing.T, got ProcessedTransaction)
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
				Transfer: func(_ vm.StateDB, _ common.Address, _ common.Address, amount *uint256.Int) {
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
			gasPool := new(core.GasPool).AddGas(maxTxGas * 3)
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

			got := runner.runRegularTransaction(context, regularTx, 0)

			require.Equal(t, regularTx, got.Transaction)
			test.validation(t, got)
		})
	}
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

func getSponsorshipRequest(t *testing.T) *types.Transaction {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	signer := types.LatestSignerForChainID(nil)
	return types.MustSignNewTx(key, signer, &types.LegacyTx{
		Nonce: 0, To: &common.Address{1}, Gas: 21_000,
	})
}

func TestTransactionGenerationUtilities(t *testing.T) {
	regular := getRegularTransaction(t)
	request := getSponsorshipRequest(t)

	require.False(t, subsidies.IsSponsorshipRequest(regular))
	require.True(t, subsidies.IsSponsorshipRequest(request))
}
