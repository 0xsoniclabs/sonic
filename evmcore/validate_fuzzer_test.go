package evmcore

import (
	"crypto/rand"
	"math"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// FuzzValidateTransaction fuzzes the validateTx function with randomly generated transactions.
func FuzzValidateTransaction(f *testing.F) {

	// Seed corpus with a few valid-looking values
	//    nonce: 0, gas: 21000, feeCap: 1_000_000_000, tip: 1_000_000_000, data: "hi", value 0
	f.Add(uint64(0), uint64(21000), int64(1_000_000_000), int64(1_000_000_000), []byte("hi"), int64(0))

	f.Fuzz(func(t *testing.T, nonce uint64, gas uint64, feeCap int64, tip int64, data []byte, value int64) {

		// a full persistent state is not need. ValidateTx needs to see the same state as the processor.
		ctxt := gomock.NewController(t)
		state := state.NewMockStateDB(ctxt)
		// expected calls to the state
		any := gomock.Any()
		state.EXPECT().Snapshot().AnyTimes()
		state.EXPECT().RevertToSnapshot(any).AnyTimes()

		// all accounts are unknown to a new state
		state.EXPECT().Exist(any).Return(false).AnyTimes()
		state.EXPECT().CreateAccount(any).AnyTimes()

		state.EXPECT().GetBalance(any).Return(uint256.NewInt(0)).AnyTimes()
		state.EXPECT().GetCode(any).Return(data).AnyTimes()
		state.EXPECT().GetCodeHash(any).Return(common.Hash{}).AnyTimes()
		state.EXPECT().GetCodeSize(any).Return(0).AnyTimes()
		state.EXPECT().GetState(any, any).Return(common.Hash{}).AnyTimes()

		state.EXPECT().HasSelfDestructed(any).Return(false).AnyTimes()
		state.EXPECT().SelfDestruct(any).AnyTimes()

		state.EXPECT().AddBalance(any, any, any).AnyTimes()
		state.EXPECT().AddLog(any).AnyTimes()
		state.EXPECT().AddRefund(any).AnyTimes()

		state.EXPECT().SetState(any, any, any).Return(common.Hash{}).AnyTimes()

		state.EXPECT().Witness().AnyTimes()

		random := make([]byte, 32)
		_, err := rand.Read(random)
		require.NoError(t, err)

		evm := vm.NewEVM(
			vm.BlockContext{
				BlockNumber: big.NewInt(1),
				Difficulty:  big.NewInt(1),
				BaseFee:     big.NewInt(1),
				BlobBaseFee: big.NewInt(0),
				Random:      (*common.Hash)(random),

				Transfer:    vm.TransferFunc(func(sd vm.StateDB, a1, a2 common.Address, i *uint256.Int) {}),
				CanTransfer: vm.CanTransferFunc(func(sd vm.StateDB, a1 common.Address, i *uint256.Int) bool { return true }),
				GetHash:     func(i uint64) common.Hash { return common.Hash{} },
			},
			state,
			&params.ChainConfig{},
			vm.Config{},
		)
		evm.GasPrice = big.NewInt(42)

		feeCapBig := big.NewInt(feeCap)
		tipBig := big.NewInt(tip)

		// To fields are 20% nil, so that some transactions are contract creations
		to := common.Address{0x42}
		toPtr := &to
		nilTo, err := rand.Int(rand.Reader, big.NewInt(5))
		require.NoError(t, err)
		if nilTo.Uint64() == 0 {
			// Set to nil
			toPtr = nil
		}

		// Randomly choose a transaction type
		var tx types.TxData
		randTxType, err := rand.Int(rand.Reader, big.NewInt(5))
		require.NoError(t, err)
		switch randTxType.Uint64() {
		case 0: // Legacy Tx
			tx = &types.LegacyTx{
				Nonce:    nonce,
				Gas:      gas,
				GasPrice: feeCapBig,
				To:       toPtr,
				Value:    big.NewInt(value),
				Data:     data,
			}
		case 1: // AccessList Tx
			tx = &types.AccessListTx{
				ChainID:    big.NewInt(1),
				Nonce:      nonce,
				Gas:        gas,
				GasPrice:   feeCapBig,
				To:         toPtr,
				Value:      big.NewInt(value),
				Data:       data,
				AccessList: types.AccessList{},
			}
		case 2: // DynamicFee Tx
			tx = &types.DynamicFeeTx{
				ChainID:    big.NewInt(1),
				Nonce:      nonce,
				Gas:        gas,
				GasFeeCap:  feeCapBig,
				GasTipCap:  tipBig,
				To:         toPtr,
				Value:      big.NewInt(value),
				Data:       data,
				AccessList: types.AccessList{},
			}
		case 3: // Blob Transaction
			tx = &types.BlobTx{
				ChainID:    uint256.NewInt(1),
				Nonce:      nonce,
				Gas:        gas,
				GasFeeCap:  uint256.MustFromBig(feeCapBig),
				GasTipCap:  uint256.MustFromBig(tipBig),
				To:         common.Address{}, // cannot be create
				Value:      uint256.NewInt(uint64(value)),
				Data:       data,
				AccessList: types.AccessList{},
			}
		case 4: // SetCode Transaction
			tx = &types.SetCodeTx{
				ChainID:    uint256.NewInt(1),
				Nonce:      nonce,
				Gas:        gas,
				GasFeeCap:  uint256.MustFromBig(feeCapBig),
				GasTipCap:  uint256.MustFromBig(tipBig),
				To:         common.Address{}, // cannot be create
				Value:      uint256.NewInt(uint64(value)),
				Data:       data,
				AccessList: types.AccessList{},
				AuthList:   []types.SetCodeAuthorization{{}},
			}
		}

		// Sign the transaction
		signer, from, signedTx := signTxForTest(t, tx)

		// Set up sufficient balance and nonce
		// TODO: fuzz on balance and nonce as well.
		state.EXPECT().GetBalance(from).Return(uint256.NewInt(math.MaxUint64)).AnyTimes()
		state.EXPECT().GetNonce(from).Return(uint64(0)).AnyTimes()

		// TODO: fuzz on validation options as well.
		opt := testTransactionsOption()
		opt.currentState = state

		// Validate the transaction
		validateErr := validateTx(signedTx, signer, opt)

		var callErr error
		if toPtr == nil {
			_, _, _, callErr = evm.Create(from, data, gas, uint256.NewInt(uint64(value)))
		} else {
			_, _, callErr = evm.Call(from, common.Address{}, data, gas, uint256.NewInt(uint64(value)))
		}

		if callErr != nil {
			if callErr != validateErr {
				t.Logf("validateTx = %v,\nevm.Call: %v", validateErr, callErr)
			}
		}
	})

}
