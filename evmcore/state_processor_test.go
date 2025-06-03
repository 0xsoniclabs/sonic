package evmcore

import (
	"fmt"
	"math"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
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
	block *EvmBlock, stateDb state.StateDB, cfg vm.Config, usedGas *uint64, onNewLog func(*types.Log),
) (
	types.Receipts, []*types.Log, []uint32, error,
) {
	// This implementation is a wrapper around the BeginBlock function, which
	// handles the actual transaction processing.
	txProcessor := p.BeginBlock(block, stateDb, cfg, onNewLog)
	receipts := make(types.Receipts, len(block.Transactions))
	skipped := make([]uint32, 0, len(block.Transactions))
	allLogs := make([]*types.Log, 0, len(block.Transactions))
	for i, tx := range block.Transactions {
		receipt, skip, err := txProcessor.Run(i, tx)
		if skip {
			skipped = append(skipped, uint32(i))
			receipts[i] = nil
			continue
		}
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to process transaction %d [%v]: %w", i, tx.Hash().Hex(), err)
		}
		receipts[i] = receipt
		allLogs = append(allLogs, receipt.Logs...)
		*usedGas = receipt.CumulativeGasUsed
	}

	return receipts, allLogs, skipped, nil
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
	processor := NewStateProcessor(&chainConfig, chain)

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
			usedGas := new(uint64)
			receipts, logs, skipped, err := process(block, state, vmConfig, usedGas, onLog)
			require.NoError(err)

			// Receipts should be set accordingly.
			require.Len(receipts, len(transactions))

			logMsg0 := &types.Log{Address: common.Address{0}}
			logMsg2 := &types.Log{Address: common.Address{2}}

			require.NotNil(receipts[0])
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
			}, receipts[0])

			require.Nil(receipts[1])

			require.NotNil(receipts[2])
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
			}, receipts[2])

			require.Nil(receipts[3])

			require.Equal([]*types.Log{logMsg0, logMsg2}, logs)
			require.Equal([]*types.Log{logMsg0, logMsg2}, reportedLogs)

			require.Equal([]uint32{1, 3}, skipped)

			require.Equal(uint64(21_000+21_000), *usedGas)
		})
	}
}

func TestProcess_DetectsTransactionThatCanNotBeConvertedIntoAMessage(t *testing.T) {
	ctrl := gomock.NewController(t)

	chainConfig := params.ChainConfig{}
	chain := NewMockDummyChain(ctrl)

	// The conversion into a evmcore Message depends on the ability to check
	// the signature and to derive the sender address. To stimulate a failure
	// in the conversion, a invalid signature is used.
	transactions := []*types.Transaction{
		types.NewTx(&types.LegacyTx{
			Nonce: 1, To: &common.Address{}, Gas: 21_000,
			R: big.NewInt(1), S: big.NewInt(2), V: big.NewInt(3),
		}),
	}

	state := getStateDbMockForTransactions(ctrl, transactions)
	processor := NewStateProcessor(&chainConfig, chain)
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
			usedGas := new(uint64)
			receipts, logs, skipped, err := process(block, state, vmConfig, usedGas, nil)
			require.ErrorContains(err, "invalid transaction v, r, s values")

			require.Nil(receipts)
			require.Nil(logs)
			require.Nil(skipped)
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

		state := state.NewMockStateDB(ctrl)

		if isPrague {
			any := gomock.Any()
			state.EXPECT().AddAddressToAccessList(params.HistoryStorageAddress).Times(2)
			state.EXPECT().Snapshot().Return(0).AnyTimes()
			state.EXPECT().Exist(params.HistoryStorageAddress).Return(true).AnyTimes()
			state.EXPECT().AddBalance(any, any, any).AnyTimes()
			state.EXPECT().SubBalance(any, any, any).AnyTimes()
			state.EXPECT().GetCode(any).AnyTimes()
			state.EXPECT().Finalise(any).AnyTimes()
		}

		processor := NewStateProcessor(&chainConfig, chain)

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
				}
				require.Equal(isPrague, chainConfig.IsPrague(block.Number, uint64(block.Time)))

				vmConfig := vm.Config{}
				usedGas := new(uint64)
				receipts, logs, skipped, err := process(block, state, vmConfig, usedGas, nil)
				require.NoError(err)
				require.Empty(receipts)
				require.Empty(logs)
				require.Empty(skipped)
			})
		}
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
			_, _, _, err := applyTransaction(&core.Message{
				SkipNonceChecks:  internal,
				SkipFromEOACheck: internal,
				GasPrice:         big.NewInt(0),
				Value:            big.NewInt(0),
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
	receipt, gasUsed, skipped, err :=
		applyTransaction(msg, gp, state, big.NewInt(1), nil, &usedGas, evm, nil)
	require.ErrorContains(t, err, "blob data is not supported")
	require.Nil(t, receipt)
	require.Equal(t, uint64(0), gasUsed)
	require.True(t, skipped)
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
				From:             common.Address{1},
				To:               nil, // contract creation
				GasLimit:         1000000,
				GasPrice:         big.NewInt(1),
				GasFeeCap:        big.NewInt(0),
				GasTipCap:        big.NewInt(0),
				Value:            big.NewInt(0),
				Data:             initCode,
				SkipNonceChecks:  true,
				SkipFromEOACheck: true,
			}

			gomock.InOrder(
				state.EXPECT().Snapshot().Return(42).Times(callToSnapshot),
				state.EXPECT().GetBalance(msg.From).Return(uint256.NewInt(1000000)),
				state.EXPECT().SubBalance(any, any, any),
				state.EXPECT().RevertToSnapshot(42).Times(callToSnapshot),
				state.EXPECT().EndTransaction(),
			)

			receipt, gasUsed, skipped, err :=
				applyTransaction(msg, gp, state, blockNumber, nil, new(uint64), evm, nil)
			require.ErrorContains(t, err, "max initcode size exceeded")
			require.Nil(t, receipt)
			require.Equal(t, uint64(0), gasUsed)
			require.True(t, skipped)
		})
	}
}

// processFunction is a function type alias for the StateProcessor's Process
// function to allow side-by-side testing of different implementations.
type processFunction = func(
	block *EvmBlock,
	statedb state.StateDB,
	cfg vm.Config,
	usedGas *uint64,
	onNewLog func(*types.Log),
) (
	receipts types.Receipts,
	allLogs []*types.Log,
	skipped []uint32,
	err error,
)

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
				{Address: common.Address{byte(*txIndex)}},
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
