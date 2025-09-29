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
	"fmt"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

////////////////////////////////////////////////////////////////////////////////
// Static Validation

func TestValidateTxStatic_Value_RejectsTxWithNegativeValue(t *testing.T) {
	txs := []types.TxData{
		&types.LegacyTx{
			Value: big.NewInt(-1),
		},
		&types.AccessListTx{
			Value: big.NewInt(-1),
		},
		&types.DynamicFeeTx{
			Value: big.NewInt(-1),
		},
		// BlobTx value is unsigned
		// SetCodeTx value is unsigned
	}

	for _, tx := range txs {
		t.Run(transactionTypeName(tx), func(t *testing.T) {
			err := ValidateTxStatic(types.NewTx(tx))
			require.ErrorIs(t, err, ErrNegativeValue)
		})
	}
}

func TestValidateTxStatic_GasPriceAndTip_RejectsTxWith(t *testing.T) {
	extremelyLargeValue := new(big.Int).Lsh(big.NewInt(1), 256)

	t.Run("gas fee larger than 256 bits", func(t *testing.T) {
		txs := []types.TxData{
			&types.LegacyTx{
				GasPrice: extremelyLargeValue,
			},
			&types.AccessListTx{
				GasPrice: extremelyLargeValue,
			},
			&types.DynamicFeeTx{
				GasFeeCap: extremelyLargeValue,
			},
			// blob GasFeeCap is uint256, cannot overflow
			// SetCodeTx GasFeeCap is uint256, cannot overflow
		}

		for _, tx := range txs {
			t.Run(transactionTypeName(tx), func(t *testing.T) {
				err := ValidateTxStatic(types.NewTx(tx))
				require.ErrorIs(t, err, ErrFeeCapVeryHigh)
			})
		}
	})

	t.Run("gas tip larger than 256 bits", func(t *testing.T) {
		txs := []types.TxData{
			&types.DynamicFeeTx{
				GasTipCap: extremelyLargeValue,
			},
			// blob GasTipCap is uint256, cannot overflow
			// SetCodeTx GasTipCap is uint256, cannot overflow
		}

		for _, tx := range txs {
			t.Run(transactionTypeName(tx), func(t *testing.T) {
				err := ValidateTxStatic(types.NewTx(tx))
				require.ErrorIs(t, err, ErrTipVeryHigh)
			})
		}
	})

	t.Run("gas fee lower than gas tip", func(t *testing.T) {
		txs := []types.TxData{
			&types.DynamicFeeTx{
				GasFeeCap: big.NewInt(1),
				GasTipCap: big.NewInt(2),
			},
			&types.BlobTx{
				GasFeeCap: uint256.NewInt(1),
				GasTipCap: uint256.NewInt(2),
			},
			&types.SetCodeTx{
				GasFeeCap: uint256.NewInt(1),
				GasTipCap: uint256.NewInt(2),
			},
		}

		for _, tx := range txs {
			t.Run(transactionTypeName(tx), func(t *testing.T) {
				err := ValidateTxStatic(types.NewTx(tx))
				require.ErrorIs(t, err, ErrTipAboveFeeCap)
			})
		}
	})
}

func TestValidateTxStatic_AuthorizationList_RejectsTxWithEmptyAuthorization(t *testing.T) {
	err := ValidateTxStatic(types.NewTx(&types.SetCodeTx{}))
	require.ErrorIs(t, err, ErrEmptyAuthorizations)
}

func TestValidateTxStatic_AcceptsValidTransactions(t *testing.T) {

	tests := []types.TxData{
		&types.LegacyTx{},
		&types.AccessListTx{},
		&types.DynamicFeeTx{},
		&types.BlobTx{},
		&types.SetCodeTx{
			AuthList: []types.SetCodeAuthorization{{}},
		},
	}
	for _, tx := range tests {
		t.Run(transactionTypeName(tx), func(t *testing.T) {
			err := ValidateTxStatic(types.NewTx(tx))
			require.NoError(t, err)
		})
	}
}

////////////////////////////////////////////////////////////////////////////////
// Network Validation

func TestValidateTxForNetwork_BeforeEip2718_RejectsNonLegacyTransactions(t *testing.T) {

	tests := []types.TxData{
		&types.AccessListTx{},
		&types.DynamicFeeTx{},
		&types.BlobTx{},
		&types.SetCodeTx{
			AuthList: []types.SetCodeAuthorization{{}},
		},
	}

	for _, tx := range tests {
		t.Run(transactionTypeName(tx), func(t *testing.T) {

			ctrl := gomock.NewController(t)
			chain := NewMockStateReader(ctrl)
			signer := NewMockSigner(ctrl)

			rules := NetworkRules{eip2718: false}
			err := ValidateTxForNetwork(types.NewTx(tx), rules, chain, signer)
			require.ErrorIs(t, ErrTxTypeNotSupported, err)
		})
	}
}

func TestValidateTxForNetwork_RejectsTxBasedOnTypeAndActiveRevision(t *testing.T) {

	tests := map[string]struct {
		tx    types.TxData
		rules NetworkRules
	}{
		"reject access list tx before eip2718": {
			tx:    &types.AccessListTx{},
			rules: NetworkRules{eip2718: false},
		},
		"reject dynamic fee tx before eip1559": {
			tx:    &types.DynamicFeeTx{},
			rules: NetworkRules{eip2718: true, eip1559: false},
		},
		"reject blob tx before eip4844": {
			tx:    &types.BlobTx{},
			rules: NetworkRules{eip2718: true, eip1559: true, eip4844: false},
		},
		"reject setCode tx before eip7702": {
			tx:    &types.SetCodeTx{AuthList: []types.SetCodeAuthorization{{}}},
			rules: NetworkRules{eip2718: true, eip1559: true, eip4844: false, eip7702: false},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			ctrl := gomock.NewController(t)
			chain := NewMockStateReader(ctrl)
			signer := NewMockSigner(ctrl)

			err := ValidateTxForNetwork(types.NewTx(test.tx), test.rules, chain, signer)
			require.ErrorIs(t, ErrTxTypeNotSupported, err)
		})
	}
}

func TestValidateTxForNetwork_Blobs_RejectsTxWith(t *testing.T) {
	//  only Blob Transactions with empty blob hash and no sidecar are accepted in sonic.

	t.Run("blob tx with non-empty blob hashes", func(t *testing.T) {
		tx := types.NewTx(&types.BlobTx{
			BlobHashes: []common.Hash{{0x01}},
		})

		rules := NetworkRules{eip2718: true, eip1559: true, eip4844: true}
		err := ValidateTxForNetwork(tx, rules, nil, nil)
		require.ErrorIs(t, err, ErrNonEmptyBlobTx)
	})

	t.Run("blob tx with non-empty sidecar", func(t *testing.T) {

		tx := types.NewTx(&types.BlobTx{
			Sidecar: &types.BlobTxSidecar{Commitments: []kzg4844.Commitment{{0x01}}},
		})

		rules := NetworkRules{eip2718: true, eip1559: true, eip4844: true}
		err := ValidateTxForNetwork(tx, rules, nil, nil)
		require.ErrorIs(t, err, ErrNonEmptyBlobTx)
	})
}

func TestValidateTxForNetwork_Gas_RejectsTx_IntrinsicGasTooLow(t *testing.T) {

	// 0 gas is always lower than any required intrinsic gas
	test := []types.TxData{
		&types.LegacyTx{},
		&types.AccessListTx{},
		&types.DynamicFeeTx{},
		&types.BlobTx{},
		&types.SetCodeTx{
			AuthList: []types.SetCodeAuthorization{{}},
		},
	}

	for _, tx := range test {
		t.Run(transactionTypeName(tx), func(t *testing.T) {
			rules := NetworkRules{eip2718: true, eip1559: true, eip4844: true, eip7702: true}
			err := ValidateTxForNetwork(types.NewTx(tx), rules, nil, nil)
			require.ErrorIs(t, err, ErrIntrinsicGas)
		})
	}
}

func TestValidateTxForNetwork_Gas_RejectsTx_GasLowerThanFloorDataGas(t *testing.T) {

	someData := make([]byte, 1024)
	floorDataGas, err := core.FloorDataGas(someData)
	require.NoError(t, err)

	// 0 gas is always lower than any required intrinsic gas
	test := []types.TxData{
		&types.LegacyTx{
			Data: someData,
			Gas:  floorDataGas - 1,
			To:   &common.Address{42}, // not a contract creation
		},
		&types.AccessListTx{
			Data: someData,
			Gas:  floorDataGas - 1,
			To:   &common.Address{42}, // not a contract creation
		},
		&types.DynamicFeeTx{
			Data: someData,
			Gas:  floorDataGas - 1,
			To:   &common.Address{42}, // not a contract creation
		},
		&types.BlobTx{
			Data: someData,
			Gas:  floorDataGas - 1,
		},
		&types.SetCodeTx{
			Data: someData,
			Gas:  floorDataGas - 1,
		},
	}

	for _, tx := range test {
		t.Run(transactionTypeName(tx), func(t *testing.T) {
			rules := NetworkRules{eip2718: true, eip1559: true, eip4844: true, eip7702: true, eip7623: true}
			err := ValidateTxForNetwork(types.NewTx(tx), rules, nil, nil)
			require.ErrorIs(t, err, ErrFloorDataGas)

			// check that the error is not happening with eip7623 disabled
			ctrl := gomock.NewController(t)
			signer := NewMockSigner(ctrl)
			signer.EXPECT().Sender(gomock.Any()).Return(common.Address{42}, nil)

			rules.eip7623 = false
			err = ValidateTxForNetwork(types.NewTx(tx), rules, nil, signer)
			require.NoError(t, err)

		})
	}
}

func TestValidateTxForNetwork_GasLimitIsCheckedAfterOsaka(t *testing.T) {

	gasLimit := uint64(30_000_000)

	test := []types.TxData{
		&types.LegacyTx{
			To:  &common.Address{42}, // not a contract creation
			Gas: gasLimit + 1,
		},
		&types.AccessListTx{
			To:  &common.Address{42}, // not a contract creation
			Gas: gasLimit + 1,
		},
		&types.DynamicFeeTx{
			To:  &common.Address{42}, // not a contract creation
			Gas: gasLimit + 1,
		},
		&types.BlobTx{
			Gas: gasLimit + 1,
		},
		&types.SetCodeTx{
			Gas: gasLimit + 1,
		},
	}

	for _, tx := range test {
		t.Run(transactionTypeName(tx), func(t *testing.T) {
			ctrl := gomock.NewController(t)
			chain := NewMockStateReader(ctrl)
			chain.EXPECT().MaxGasLimit().Return(gasLimit).AnyTimes()
			signer := NewMockSigner(ctrl)
			signer.EXPECT().Sender(gomock.Any()).Return(common.Address{42}, nil).AnyTimes()

			rules := NetworkRules{
				eip2718: true,
				eip1559: true,
				eip4844: true,
				eip7702: true,
				eip7623: true,
				osaka:   true,
			}
			err := ValidateTxForNetwork(types.NewTx(tx), rules, chain, signer)
			require.ErrorIs(t, err, ErrGasLimitTooHigh)

			// check that the error is not reported with osaka disabled
			rules.osaka = false
			err = ValidateTxForNetwork(types.NewTx(tx), rules, chain, signer)
			require.NoError(t, err)
		})
	}
}

func TestValidateTxForNetwork_InitCodeTooLarge_ReturnsError(t *testing.T) {

	data := make([]byte, params.MaxInitCodeSize+1)
	gas, err := core.IntrinsicGas(data, nil, nil, true, true, true, true)
	require.NoError(t, err)

	tests := []types.TxData{
		&types.LegacyTx{
			To:   nil, // contract creation
			Gas:  gas,
			Data: data,
		},
		&types.AccessListTx{
			To:   nil, // contract creation
			Gas:  gas,
			Data: data,
		},
		&types.DynamicFeeTx{
			To:   nil, // contract creation
			Gas:  gas,
			Data: data,
		},
	}

	for _, tx := range tests {
		t.Run(transactionTypeName(tx), func(t *testing.T) {
			ctrl := gomock.NewController(t)
			chain := NewMockStateReader(ctrl)
			signer := NewMockSigner(ctrl)
			signer.EXPECT().Sender(gomock.Any()).Return(common.Address{42}, nil).AnyTimes()

			rules := NetworkRules{
				istanbul: true,
				shanghai: true,
				eip2718:  true,
				eip1559:  true,
				eip4844:  true,
			}

			err := ValidateTxForNetwork(types.NewTx(tx), rules, chain, signer)
			require.ErrorIs(t, err, ErrMaxInitCodeSizeExceeded)

			// check that the error is not happening with shanghai disabled
			rules.shanghai = false
			err = ValidateTxForNetwork(types.NewTx(tx), rules, chain, signer)
			require.NoError(t, err)
		})
	}
}

func TestValidateTxForNetwork_Signer_RejectsTxWithInvalidSigner(t *testing.T) {

	tests := []types.TxData{
		&types.LegacyTx{
			To:  &common.Address{42}, // not a contract creation
			Gas: 21000,
		},
		&types.AccessListTx{
			To:  &common.Address{42}, // not a contract creation
			Gas: 21000,
		},
		&types.DynamicFeeTx{
			To:  &common.Address{42}, // not a contract creation
			Gas: 21000,
		},
		&types.BlobTx{
			Gas: 21000,
		},
		&types.SetCodeTx{
			Gas:      50_000,
			AuthList: []types.SetCodeAuthorization{{}},
		},
	}

	for _, tx := range tests {
		t.Run(transactionTypeName(tx), func(t *testing.T) {
			ctrl := gomock.NewController(t)
			chain := NewMockStateReader(ctrl)
			signer := NewMockSigner(ctrl)
			signer.EXPECT().Sender(gomock.Any()).Return(common.Address{42}, fmt.Errorf("some error"))

			rules := NetworkRules{
				istanbul: true,
				shanghai: true,
				eip2718:  true,
				eip1559:  true,
				eip4844:  true,
				eip7702:  true,
			}

			err := ValidateTxForNetwork(types.NewTx(tx), rules, chain, signer)
			require.ErrorIs(t, err, ErrInvalidSender)

		})
	}
}

func TestValidateTxForNetwork_AcceptsTransactions(t *testing.T) {

	tests := []types.TxData{
		&types.LegacyTx{
			To:  &common.Address{42}, // not a contract creation
			Gas: 21000,
		},
		&types.AccessListTx{
			To:  &common.Address{42}, // not a contract creation
			Gas: 21000,
		},
		&types.DynamicFeeTx{
			To:  &common.Address{42}, // not a contract creation
			Gas: 21000,
		},
		&types.BlobTx{
			Gas: 21000,
		},
		&types.SetCodeTx{
			Gas:      50_000,
			AuthList: []types.SetCodeAuthorization{{}},
		},
	}

	for _, tx := range tests {
		t.Run(transactionTypeName(tx), func(t *testing.T) {

			ctrl := gomock.NewController(t)
			chain := NewMockStateReader(ctrl)
			signer := NewMockSigner(ctrl)
			signer.EXPECT().Sender(gomock.Any()).Return(common.Address{42}, nil)

			rules := NetworkRules{
				istanbul: true,
				shanghai: true,
				eip2718:  true,
				eip1559:  true,
				eip4844:  true,
				eip7702:  true,
			}

			err := ValidateTxForNetwork(types.NewTx(tx), rules, chain, signer)
			require.NoError(t, err)
		})
	}
}

////////////////////////////////////////////////////////////////////////////////
// Block Validation

func TestValidateTxForBlock_MaxGas_RejectsTxWithGasOverMaxGas(t *testing.T) {

	tests := []types.TxData{
		&types.LegacyTx{
			To:       &common.Address{42}, // not a contract creation
			Gas:      100_000,
			GasPrice: big.NewInt(1),
		},
		&types.AccessListTx{
			To:       &common.Address{42}, // not a contract creation
			Gas:      100_000,
			GasPrice: big.NewInt(1),
		},
		&types.DynamicFeeTx{
			To:        &common.Address{42}, // not a contract creation
			Gas:       100_000,
			GasFeeCap: big.NewInt(1),
		},
		&types.BlobTx{
			Gas:       100_000,
			GasFeeCap: uint256.NewInt(1),
		},
		&types.SetCodeTx{
			Gas:       100_000,
			GasFeeCap: uint256.NewInt(1),
			AuthList:  []types.SetCodeAuthorization{{}},
		},
	}

	for _, tx := range tests {
		t.Run(transactionTypeName(tx), func(t *testing.T) {
			ctrl := gomock.NewController(t)
			chain := NewMockStateReader(ctrl)
			chain.EXPECT().MaxGasLimit().Return(uint64(99_999))
			chain.EXPECT().GetCurrentBaseFee().Return(big.NewInt(1)).AnyTimes()

			err := ValidateTxForBlock(types.NewTx(tx), chain)
			require.ErrorIs(t, err, ErrGasLimit)
		})
	}
}

func TestValidateTxForBlock_BaseFee_RejectsTxWithGasPriceLowerThanBaseFee(t *testing.T) {
	tests := []types.TxData{
		&types.LegacyTx{
			To:       &common.Address{42}, // not a contract creation
			Gas:      100_000,
			GasPrice: big.NewInt(1),
		},
		&types.AccessListTx{
			To:       &common.Address{42}, // not a contract creation
			Gas:      100_000,
			GasPrice: big.NewInt(1),
		},
		&types.DynamicFeeTx{
			To:        &common.Address{42}, // not a contract creation
			Gas:       100_000,
			GasFeeCap: big.NewInt(1),
		},
		&types.BlobTx{
			Gas:       100_000,
			GasFeeCap: uint256.NewInt(1),
		},
		&types.SetCodeTx{
			Gas:       100_000,
			GasFeeCap: uint256.NewInt(1),
			AuthList:  []types.SetCodeAuthorization{{}},
		},
	}

	for _, tx := range tests {
		t.Run(transactionTypeName(tx), func(t *testing.T) {
			ctrl := gomock.NewController(t)
			chain := NewMockStateReader(ctrl)
			chain.EXPECT().GetCurrentBaseFee().Return(big.NewInt(2))

			err := ValidateTxForBlock(types.NewTx(tx), chain)
			require.ErrorIs(t, err, ErrUnderpriced)
		})
	}
}

func TestValidateTxForBlock_AcceptsTransactions(t *testing.T) {

	tests := []types.TxData{
		&types.LegacyTx{
			To:       &common.Address{42}, // not a contract creation
			Gas:      100_000,
			GasPrice: big.NewInt(1),
		},
		&types.AccessListTx{
			To:       &common.Address{42}, // not a contract creation
			Gas:      100_000,
			GasPrice: big.NewInt(1),
		},
		&types.DynamicFeeTx{
			To:        &common.Address{42}, // not a contract creation
			Gas:       100_000,
			GasFeeCap: big.NewInt(1),
		},
		&types.BlobTx{
			Gas:       100_000,
			GasFeeCap: uint256.NewInt(1),
		},
		&types.SetCodeTx{
			Gas:       100_000,
			GasFeeCap: uint256.NewInt(1),
			AuthList:  []types.SetCodeAuthorization{{}},
		},
	}

	for _, tx := range tests {
		t.Run(transactionTypeName(tx), func(t *testing.T) {
			ctrl := gomock.NewController(t)
			chain := NewMockStateReader(ctrl)
			chain.EXPECT().MaxGasLimit().Return(uint64(100_000))
			chain.EXPECT().GetCurrentBaseFee().Return(big.NewInt(1))

			err := ValidateTxForBlock(types.NewTx(tx), chain)
			require.NoError(t, err)
		})
	}
}

////////////////////////////////////////////////////////////////////////////////
// State Validation

func TestValidateTxForState_Signer_RejectsTxWithInvalidSigner(t *testing.T) {

	test := []types.TxData{
		&types.LegacyTx{},
		&types.AccessListTx{},
		&types.DynamicFeeTx{},
		&types.BlobTx{},
		&types.SetCodeTx{
			AuthList: []types.SetCodeAuthorization{{}},
		},
	}

	for _, tx := range test {
		t.Run(transactionTypeName(tx), func(t *testing.T) {
			signer := NewMockSigner(gomock.NewController(t))
			signer.EXPECT().Sender(gomock.Any()).Return(common.Address{42}, fmt.Errorf("some error"))
			err := ValidateTxForState(types.NewTx(tx), nil, signer)
			require.ErrorIs(t, err, ErrInvalidSender)
		})
	}
}

func TestValidateTxForState_Nonce_RejectsTxWithOlderNonce(t *testing.T) {
	tests := []types.TxData{
		&types.LegacyTx{},
		&types.AccessListTx{},
		&types.DynamicFeeTx{},
		&types.BlobTx{},
		&types.SetCodeTx{
			AuthList: []types.SetCodeAuthorization{{}},
		},
	}
	for _, tx := range tests {
		t.Run(transactionTypeName(tx), func(t *testing.T) {
			ctrl := gomock.NewController(t)
			state := state.NewMockStateDB(ctrl)
			state.EXPECT().GetNonce(gomock.Any()).Return(uint64(1))
			signer := NewMockSigner(ctrl)
			signer.EXPECT().Sender(gomock.Any()).Return(common.Address{42}, nil)

			err := ValidateTxForState(types.NewTx(tx), state, signer)
			require.ErrorIs(t, err, ErrNonceTooLow)
		})
	}
}

func TestValidateTxForState_Balance_RejectsTxWhenInsufficientBalance(t *testing.T) {
	tests := []types.TxData{
		&types.LegacyTx{
			Gas:      100,
			GasPrice: big.NewInt(1),
		},
		&types.AccessListTx{
			Gas:      100,
			GasPrice: big.NewInt(1),
		},
		&types.DynamicFeeTx{
			Gas:       100,
			GasFeeCap: big.NewInt(1),
		},
		&types.BlobTx{
			Gas:       100,
			GasFeeCap: uint256.NewInt(1),
		},
		&types.SetCodeTx{
			Gas:       100,
			GasFeeCap: uint256.NewInt(1),
			AuthList:  []types.SetCodeAuthorization{{}},
		},
	}
	for _, tx := range tests {
		t.Run(transactionTypeName(tx), func(t *testing.T) {
			ctrl := gomock.NewController(t)
			state := state.NewMockStateDB(ctrl)
			state.EXPECT().GetNonce(gomock.Any()).Return(uint64(0))
			state.EXPECT().GetBalance(gomock.Any()).Return(uint256.NewInt(99))
			signer := NewMockSigner(ctrl)
			signer.EXPECT().Sender(gomock.Any()).Return(common.Address{42}, nil)

			err := ValidateTxForState(types.NewTx(tx), state, signer)
			require.ErrorIs(t, err, ErrInsufficientFunds)
		})
	}
}

func TestValidateTxForState_AcceptsTransactions(t *testing.T) {

	tests := []types.TxData{
		&types.LegacyTx{
			Gas:      100,
			GasPrice: big.NewInt(1),
		},
		&types.AccessListTx{
			Gas:      100,
			GasPrice: big.NewInt(1),
		},
		&types.DynamicFeeTx{
			Gas:       100,
			GasFeeCap: big.NewInt(1),
		},
		&types.BlobTx{
			Gas:       100,
			GasFeeCap: uint256.NewInt(1),
		},
		&types.SetCodeTx{
			Gas:       100,
			GasFeeCap: uint256.NewInt(1),
			AuthList:  []types.SetCodeAuthorization{{}},
		},
	}
	for _, tx := range tests {
		t.Run(transactionTypeName(tx), func(t *testing.T) {
			ctrl := gomock.NewController(t)
			state := state.NewMockStateDB(ctrl)
			state.EXPECT().GetNonce(gomock.Any()).Return(uint64(0))
			state.EXPECT().GetBalance(gomock.Any()).Return(uint256.NewInt(100))
			signer := NewMockSigner(ctrl)
			signer.EXPECT().Sender(gomock.Any()).Return(common.Address{42}, nil)

			err := ValidateTxForState(types.NewTx(tx), state, signer)
			require.NoError(t, err)
		})
	}
}

////////////////////////////////////////////////////////////////////////////////
// TxPool Policies Validation

func TestValidateTxForPool_Data_RejectsTxWithOversizedData(t *testing.T) {
	data := make([]byte, txMaxSize+1)
	tests := []types.TxData{
		&types.LegacyTx{
			Data: data,
		},
		&types.AccessListTx{
			Data: data,
		},
		&types.DynamicFeeTx{
			Data: data,
		},
		&types.BlobTx{
			Data: data,
		},
		&types.SetCodeTx{
			Data:     data,
			AuthList: []types.SetCodeAuthorization{{}},
		},
	}

	for _, tx := range tests {
		t.Run(transactionTypeName(tx), func(t *testing.T) {
			opts := poolOptions{}
			err := validateTxForPool(types.NewTx(tx), opts, nil)
			require.ErrorIs(t, err, ErrOversizedData)
		})
	}
}

func TestValidateTxForPool_Signer_RejectsTxWithInvalidSigner(t *testing.T) {
	tests := []types.TxData{
		&types.LegacyTx{},
		&types.AccessListTx{},
		&types.DynamicFeeTx{},
		&types.BlobTx{},
		&types.SetCodeTx{
			AuthList: []types.SetCodeAuthorization{{}},
		},
	}

	for _, tx := range tests {
		t.Run(transactionTypeName(tx), func(t *testing.T) {
			opts := poolOptions{}
			ctrl := gomock.NewController(t)
			signer := NewMockSigner(ctrl)
			signer.EXPECT().Sender(gomock.Any()).Return(common.Address{42}, fmt.Errorf("some error"))
			err := validateTxForPool(types.NewTx(tx), opts, signer)
			require.ErrorIs(t, err, ErrInvalidSender)
		})
	}
}

func TestValidateTxForPool_RejectsNonLocalTxWithTipLowerThanMinPool(t *testing.T) {
	tip := big.NewInt(1)
	tests := []types.TxData{
		&types.DynamicFeeTx{
			GasTipCap: tip,
		},
		&types.BlobTx{
			GasTipCap: uint256.MustFromBig(tip),
		},
		&types.SetCodeTx{
			GasTipCap: uint256.MustFromBig(tip),
			AuthList:  []types.SetCodeAuthorization{{}},
		},
	}

	for _, tx := range tests {
		t.Run(transactionTypeName(tx), func(t *testing.T) {
			opts := poolOptions{
				minTip: new(big.Int).Add(tip, big.NewInt(1)),
				locals: newAccountSet(nil),
			}

			ctrl := gomock.NewController(t)
			signer := NewMockSigner(ctrl)
			signer.EXPECT().Sender(gomock.Any()).Return(common.Address{42}, nil)

			err := validateTxForPool(types.NewTx(tx), opts, signer)
			require.ErrorIs(t, err, ErrUnderpriced)
		})
	}

}

func TestValidateTxForPool_AcceptsNonLocalTxWithTipBiggerThanMin(t *testing.T) {

	tip := big.NewInt(2)
	tests := []types.TxData{
		&types.DynamicFeeTx{
			GasTipCap: tip,
		},
		&types.BlobTx{
			GasTipCap: uint256.MustFromBig(tip),
		},
		&types.SetCodeTx{
			GasTipCap: uint256.MustFromBig(tip),
			AuthList:  []types.SetCodeAuthorization{{}},
		},
	}

	for _, tx := range tests {
		t.Run(transactionTypeName(tx), func(t *testing.T) {
			opts := poolOptions{
				minTip: big.NewInt(1),
				locals: newAccountSet(nil),
			}

			ctrl := gomock.NewController(t)
			signer := NewMockSigner(ctrl)
			signer.EXPECT().Sender(gomock.Any()).Return(common.Address{42}, nil)

			err := validateTxForPool(types.NewTx(tx), opts, signer)
			require.NoError(t, err)
		})
	}
}

////////////////////////////////////////////////////////////////////////////////
// ValidateTx

func TestValidateTx_RejectsTx_whenNetworkValidationFails(t *testing.T) {
	tests := []types.TxData{
		&types.LegacyTx{},
		&types.AccessListTx{},
		&types.DynamicFeeTx{},
		&types.BlobTx{},
		&types.SetCodeTx{
			AuthList: []types.SetCodeAuthorization{{}},
		},
	}

	for _, tx := range tests {
		t.Run(transactionTypeName(tx), func(t *testing.T) {
			opts := poolOptions{}

			rules := NetworkRules{}

			err := validateTx(types.NewTx(tx), opts, rules, nil, nil, nil)
			require.Error(t, err)
		})
	}
}

func TestValidateTx_RejectsTx_WhenStaticValidationFails(t *testing.T) {
	tests := []types.TxData{
		&types.LegacyTx{
			Gas:      100_000,
			GasPrice: big.NewInt(-1), // invalid as it has negative gas price
		},
		&types.AccessListTx{
			Gas:      100_000,
			GasPrice: big.NewInt(-1), // invalid as it has negative gas price
		},
		&types.DynamicFeeTx{
			Gas:       100_000,
			GasFeeCap: big.NewInt(-1), // invalid as it has negative gas price
		},
		&types.BlobTx{
			Gas:       100_000,
			GasFeeCap: uint256.NewInt(1),
			GasTipCap: uint256.NewInt(2), // invalid as tip is above fee cap
		},
		&types.SetCodeTx{
			Gas: 100_000,
			// invalid as it has empty auth list
		},
	}

	for _, tx := range tests {
		t.Run(transactionTypeName(tx), func(t *testing.T) {
			opts := poolOptions{}

			rules := NetworkRules{
				eip2718: true,
				eip1559: true,
				eip4844: true,
				eip7702: true,
			}
			ctrl := gomock.NewController(t)
			signer := NewMockSigner(ctrl)
			signer.EXPECT().Sender(gomock.Any()).Return(common.Address{42}, nil).AnyTimes()

			chain := NewMockStateReader(ctrl)
			chain.EXPECT().GetCurrentBaseFee().Return(big.NewInt(5)).AnyTimes()
			state := state.NewMockStateDB(ctrl)

			err := validateTx(types.NewTx(tx), opts, rules, chain, state, signer)
			require.Error(t, err)
		})
	}
}

func TestValidateTx_RejectsTx_WhenBlockValidationFails(t *testing.T) {

	tests := []types.TxData{
		&types.LegacyTx{
			Gas:      100_000,
			GasPrice: big.NewInt(10),
		},
		&types.AccessListTx{
			Gas:      100_000,
			GasPrice: big.NewInt(10),
		},
		&types.DynamicFeeTx{
			Gas:       100_000,
			GasFeeCap: big.NewInt(10),
		},
		&types.BlobTx{
			Gas:       100_000,
			GasFeeCap: uint256.NewInt(10),
		},
		&types.SetCodeTx{
			Gas:       100_000,
			GasFeeCap: uint256.NewInt(10),
			AuthList:  []types.SetCodeAuthorization{{}},
		},
	}

	for _, tx := range tests {
		t.Run(transactionTypeName(tx), func(t *testing.T) {
			opts := poolOptions{}

			rules := NetworkRules{
				eip2718: true,
				eip1559: true,
				eip4844: true,
				eip7702: true,
			}
			ctrl := gomock.NewController(t)
			signer := NewMockSigner(ctrl)
			signer.EXPECT().Sender(gomock.Any()).Return(common.Address{42}, nil).AnyTimes()

			chain := NewMockStateReader(ctrl)
			chain.EXPECT().GetCurrentBaseFee().Return(big.NewInt(5)).AnyTimes()
			chain.EXPECT().MaxGasLimit().Return(uint64(50_000)).AnyTimes() // lower than tx gas
			state := state.NewMockStateDB(ctrl)

			err := validateTx(types.NewTx(tx), opts, rules, chain, state, signer)
			require.Error(t, err)
		})
	}
}

func TestValidateTx_RejectsTx_WhenPoolValidationFails(t *testing.T) {

	tests := []types.TxData{
		&types.LegacyTx{
			Gas:      100_000,
			GasPrice: big.NewInt(10),
		},
		&types.AccessListTx{
			Gas:      100_000,
			GasPrice: big.NewInt(10),
		},
		&types.DynamicFeeTx{
			Gas:       100_000,
			GasFeeCap: big.NewInt(10),
		},
		&types.BlobTx{
			Gas:       100_000,
			GasFeeCap: uint256.NewInt(10),
		},
		&types.SetCodeTx{
			Gas:       100_000,
			GasFeeCap: uint256.NewInt(10),
			AuthList:  []types.SetCodeAuthorization{{}},
		},
	}

	for _, tx := range tests {
		t.Run(transactionTypeName(tx), func(t *testing.T) {

			rules := NetworkRules{
				eip2718: true,
				eip1559: true,
				eip4844: true,
				eip7702: true,
			}
			ctrl := gomock.NewController(t)
			signer := NewMockSigner(ctrl)
			signer.EXPECT().Sender(gomock.Any()).Return(common.Address{42}, nil).AnyTimes()
			signer.EXPECT().Equal(gomock.Any()).Return(false).AnyTimes()

			chain := NewMockStateReader(ctrl)
			chain.EXPECT().GetCurrentBaseFee().Return(big.NewInt(5)).AnyTimes()
			chain.EXPECT().MaxGasLimit().Return(uint64(100_000)).AnyTimes()
			state := state.NewMockStateDB(ctrl)

			opts := poolOptions{
				minTip: big.NewInt(20), // transactions are tipping 0, so this will fail
				locals: newAccountSet(signer),
			}

			err := validateTx(types.NewTx(tx), opts, rules, chain, state, signer)
			require.Error(t, err)
		})
	}
}

func TestValidateTx_RejectsTx_WhenStateValidationFails(t *testing.T) {

	tests := []types.TxData{
		&types.LegacyTx{
			Gas:      100_000,
			GasPrice: big.NewInt(10),
		},
		&types.AccessListTx{
			Gas:      100_000,
			GasPrice: big.NewInt(10),
		},
		&types.DynamicFeeTx{
			Gas:       100_000,
			GasFeeCap: big.NewInt(10),
		},
		&types.BlobTx{
			Gas:       100_000,
			GasFeeCap: uint256.NewInt(10),
		},
		&types.SetCodeTx{
			Gas:       100_000,
			GasFeeCap: uint256.NewInt(10),
			AuthList:  []types.SetCodeAuthorization{{}},
		},
	}

	for _, tx := range tests {
		t.Run(transactionTypeName(tx), func(t *testing.T) {

			rules := NetworkRules{
				eip2718: true,
				eip1559: true,
				eip4844: true,
				eip7702: true,
			}
			ctrl := gomock.NewController(t)
			signer := NewMockSigner(ctrl)
			signer.EXPECT().Sender(gomock.Any()).Return(common.Address{42}, nil).AnyTimes()
			signer.EXPECT().Equal(gomock.Any()).Return(false).AnyTimes()

			chain := NewMockStateReader(ctrl)
			chain.EXPECT().GetCurrentBaseFee().Return(big.NewInt(5)).AnyTimes()
			chain.EXPECT().MaxGasLimit().Return(uint64(100_000)).AnyTimes()
			state := state.NewMockStateDB(ctrl)
			state.EXPECT().GetNonce(gomock.Any()).Return(uint64(1)).AnyTimes() // higher than tx nonce 0

			opts := poolOptions{
				minTip:  big.NewInt(0),
				isLocal: true,
				locals:  newAccountSet(signer),
			}

			err := validateTx(types.NewTx(tx), opts, rules, chain, state, signer)
			require.Error(t, err)
		})
	}
}

func TestValidateTx_Success(t *testing.T) {

	tests := []types.TxData{
		&types.LegacyTx{
			Gas:      100_000,
			GasPrice: big.NewInt(10),
		},
		&types.AccessListTx{
			Gas:      100_000,
			GasPrice: big.NewInt(10),
		},
		&types.DynamicFeeTx{
			Gas:       100_000,
			GasFeeCap: big.NewInt(10),
			GasTipCap: big.NewInt(5),
		},
		&types.BlobTx{
			Gas:       100_000,
			GasFeeCap: uint256.NewInt(10),
			GasTipCap: uint256.NewInt(5),
		},
		&types.SetCodeTx{
			Gas:       100_000,
			GasFeeCap: uint256.NewInt(10),
			GasTipCap: uint256.NewInt(5),
			AuthList:  []types.SetCodeAuthorization{{}},
		},
	}

	for _, tx := range tests {
		t.Run(transactionTypeName(tx), func(t *testing.T) {

			rules := NetworkRules{
				eip2718: true,
				eip1559: true,
				eip4844: true,
				eip7702: true,
			}
			ctrl := gomock.NewController(t)
			signer := NewMockSigner(ctrl)
			signer.EXPECT().Sender(gomock.Any()).Return(common.Address{42}, nil).AnyTimes()
			signer.EXPECT().Equal(gomock.Any()).Return(false).AnyTimes()

			chain := NewMockStateReader(ctrl)
			chain.EXPECT().GetCurrentBaseFee().Return(big.NewInt(5)).AnyTimes()
			chain.EXPECT().MaxGasLimit().Return(uint64(100_000)).AnyTimes()
			state := state.NewMockStateDB(ctrl)
			state.EXPECT().GetNonce(gomock.Any()).Return(uint64(0)).AnyTimes()
			state.EXPECT().GetBalance(gomock.Any()).Return(uint256.NewInt(1_000_000)).AnyTimes()

			opts := poolOptions{
				minTip:  big.NewInt(0),
				isLocal: true,
				locals:  newAccountSet(signer),
			}

			err := validateTx(types.NewTx(tx), opts, rules, chain, state, signer)
			require.NoError(t, err)
		})
	}
}

// =============================================================================
// Helpers
// =============================================================================

func transactionTypeName(tx types.TxData) string {
	switch tx.(type) {
	case *types.LegacyTx:
		return "legacy"
	case *types.AccessListTx:
		return "access list"
	case *types.DynamicFeeTx:
		return "dynamic fee"
	case *types.BlobTx:
		return "blob"
	case *types.SetCodeTx:
		return "setCode"
	default:
		return "unknown"
	}
}
