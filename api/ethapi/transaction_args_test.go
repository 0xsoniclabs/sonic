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

package ethapi

import (
	"math"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestTransactionArgs_ToMessage_GasCap(t *testing.T) {
	t.Parallel()

	var (
		gas10M       = hexutil.Uint64(10_000_000)
		gasMaxUint64 = hexutil.Uint64(math.MaxUint64)
	)

	tests := []struct {
		name         string
		argGas       *hexutil.Uint64
		globalGasCap uint64
		expectedGas  uint64
	}{
		{
			name:         "gas cap 0 and arg gas nil",
			globalGasCap: 0,
			argGas:       nil,
			expectedGas:  math.MaxInt64,
		}, {
			name:         "gas cap 0 and arg gas 10M",
			globalGasCap: 0,
			argGas:       &gas10M,
			expectedGas:  10_000_000,
		}, {
			name:         "gas cap 0 and arg gas maxUint64",
			globalGasCap: 0,
			argGas:       &gasMaxUint64,
			expectedGas:  math.MaxInt64,
		}, {
			name:         "gas cap 50M and arg gas nil",
			globalGasCap: 50_000_000,
			argGas:       nil,
			expectedGas:  50_000_000,
		}, {
			name:         "gas cap 50M and arg gas 10M",
			globalGasCap: 50_000_000,
			argGas:       &gas10M,
			expectedGas:  10_000_000,
		}, {
			name:         "gas cap 50M and arg gas maxUint64",
			globalGasCap: 50_000_000,
			argGas:       &gasMaxUint64,
			expectedGas:  50_000_000,
		}, {
			name:         "gas cap maxUint64 and arg gas 10M",
			globalGasCap: math.MaxUint64,
			argGas:       &gas10M,
			expectedGas:  10_000_000,
		}, {
			name:         "gas cap maxUint64 and arg gas maxUint64",
			globalGasCap: math.MaxUint64,
			argGas:       &gasMaxUint64,
			expectedGas:  math.MaxInt64,
		},
	}

	for _, test := range tests {
		args := TransactionArgs{Gas: test.argGas}
		msg, err := args.ToMessage(test.globalGasCap, nil, log.Root())
		require.Nil(t, err)
		require.Equal(t, test.expectedGas, msg.GasLimit, test.name)
	}
}

func TestTransactionArgs_ToMessage_CapsGasToProvidedGlobalAndLogsWarning(t *testing.T) {
	t.Parallel()

	args := TransactionArgs{
		Gas: new(hexutil.Uint64(150)),
	}

	ctrl := gomock.NewController(t)
	logger := logger.NewMockLogger(ctrl)
	logger.EXPECT().Warn("Caller gas above allowance, capping",
		gomock.Any(), gomock.Any(),
		gomock.Any(), gomock.Any(),
	)

	globalGasCap := uint64(100)

	msg, err := args.ToMessage(globalGasCap, nil, logger)
	require.NoError(t, err, "Failed to convert TransactionArgs to message")
	require.Equal(t, globalGasCap, msg.GasLimit, "Gas limit should be capped to the provided global gas cap")
}

func TestTransactionArgs_ToMessage_Empty(t *testing.T) {
	t.Parallel()

	empty := TransactionArgs{}

	gasCap := uint64(0x123)
	baseFee := big.NewInt(100)

	msg, err := empty.ToMessage(gasCap, baseFee, nil)
	require.NoError(t, err, "Failed to convert empty TransactionArgs to message")

	require.NotNil(t, msg)
	require.Nil(t, msg.To)
	require.Equal(t, gasCap, msg.GasLimit)
	require.Equal(t, big.NewInt(0), msg.GasPrice)
	require.Equal(t, big.NewInt(0), msg.Value)
	require.Nil(t, msg.BlobGasFeeCap)
	require.Equal(t, big.NewInt(0), msg.GasTipCap)
	require.Equal(t, uint64(0), msg.Nonce)
}

func TestTransactionArgs_ToMessage_TrivialFieldsAreCopied(t *testing.T) {
	t.Parallel()
	// this test checks that the trivial fields of TransactionArgs
	// are correctly converted to a core.Message,
	// Trivial fields are those which do not include any logic

	txArgs := TransactionArgs{
		To:    &common.Address{0x1},
		Nonce: new(hexutil.Uint64(0x2)),
		Value: (*hexutil.Big)(big.NewInt(0x3)),
		Data:  new(hexutil.Bytes([]byte{0x4})),
		Gas:   new(hexutil.Uint64(0x5)),
		AccessList: new(
			types.AccessList{
				{
					Address: common.Address{0x1},
					StorageKeys: []common.Hash{
						common.HexToHash("0x1234"),
						common.HexToHash("0x5678"),
					},
				},
			}),
		BlobFeeCap: (*hexutil.Big)(big.NewInt(0x6)),
		BlobHashes: []common.Hash{
			common.HexToHash("0x7"),
		},
		AuthorizationList: []types.SetCodeAuthorization{
			{
				Address: common.Address{0x1},
				Nonce:   0x2,
				ChainID: *uint256.NewInt(0x3),
			},
		},
	}
	msg, err := txArgs.ToMessage(0x4321, big.NewInt(100), nil)
	require.NoError(t, err)

	require.Equal(t, core.Message{
		To:       &common.Address{0x1},
		Nonce:    msg.Nonce,
		Value:    big.NewInt(0x3),
		GasLimit: 0x5,

		GasPrice:  big.NewInt(0), // not set, so it defaults to 0
		GasFeeCap: big.NewInt(0), // not set, so it defaults to 0
		GasTipCap: big.NewInt(0), // not set, so it defaults to 0

		Data: []byte{0x4},
		AccessList: types.AccessList{
			{
				Address: common.Address{0x1},
				StorageKeys: []common.Hash{
					common.HexToHash("0x1234"),
					common.HexToHash("0x5678"),
				},
			},
		},
		BlobGasFeeCap: big.NewInt(0x6),
		BlobHashes: []common.Hash{
			common.HexToHash("0x7"),
		},
		SetCodeAuthorizations: []types.SetCodeAuthorization{
			{
				Address: common.Address{0x1},
				Nonce:   0x2,
				ChainID: *uint256.NewInt(0x3),
			},
		},

		// Hardcoded values
		SkipNonceChecks:       true,
		SkipTransactionChecks: true,
	}, *msg)
}

func TestTransactionArgs_ToMessage_GasPriceFollowsEIP1559Rules(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		args        TransactionArgs
		expectedMsg core.Message
		baseFee     *big.Int
	}{
		"zero initialized and without basefee uses pre-eip1559 rules": {
			args: TransactionArgs{},
			expectedMsg: core.Message{
				GasLimit: math.MaxInt64,
				Value:    big.NewInt(0),

				GasPrice:  big.NewInt(0),
				GasFeeCap: big.NewInt(0),
				GasTipCap: big.NewInt(0),

				// Hardcoded values
				SkipNonceChecks:       true,
				SkipTransactionChecks: true,
			},
		},
		"zero initialized and basefee uses eip1559 rules": {
			args: TransactionArgs{},
			expectedMsg: core.Message{
				GasLimit: math.MaxInt64,
				Value:    big.NewInt(0),

				GasPrice:  big.NewInt(0),
				GasFeeCap: big.NewInt(0),
				GasTipCap: big.NewInt(0),

				// Hardcoded values
				SkipNonceChecks:       true,
				SkipTransactionChecks: true,
			},
			baseFee: big.NewInt(77),
		},
		"gasPrice and basefee promotes gas pricing to eip1559 rules": {
			args: TransactionArgs{
				GasPrice: (*hexutil.Big)(big.NewInt(10000000)),
			},
			expectedMsg: core.Message{
				GasLimit: math.MaxInt64,
				Value:    big.NewInt(0),

				GasPrice:  big.NewInt(10000000),
				GasFeeCap: big.NewInt(10000000),
				GasTipCap: big.NewInt(10000000),
				// Hardcoded values
				SkipNonceChecks:       true,
				SkipTransactionChecks: true,
			},
			baseFee: big.NewInt(77),
		},
		"gasPrice and no basefee promotes gas pricing to eip1559 rules": {
			args: TransactionArgs{
				GasPrice: (*hexutil.Big)(big.NewInt(10000000)),
			},
			expectedMsg: core.Message{
				GasLimit: math.MaxInt64,
				Value:    big.NewInt(0),

				GasPrice:  big.NewInt(10000000),
				GasFeeCap: big.NewInt(10000000),
				GasTipCap: big.NewInt(10000000),
				// Hardcoded values
				SkipNonceChecks:       true,
				SkipTransactionChecks: true,
			},
		},
		"maxFeePerGas and no basefee uses pre-eip1559 rules": {
			args: TransactionArgs{
				MaxFeePerGas: (*hexutil.Big)(big.NewInt(1234)),
			},
			expectedMsg: core.Message{
				GasLimit: math.MaxInt64,
				Value:    big.NewInt(0),

				GasPrice:  big.NewInt(0),
				GasFeeCap: big.NewInt(0),
				GasTipCap: big.NewInt(0),

				// Hardcoded values
				SkipNonceChecks:       true,
				SkipTransactionChecks: true,
			},
		},
		"maxFeePerGas and basefee uses eip1559 rules": {
			args: TransactionArgs{
				MaxFeePerGas: (*hexutil.Big)(big.NewInt(1234)),
			},
			expectedMsg: core.Message{
				GasLimit: math.MaxInt64,
				Value:    big.NewInt(0),

				GasPrice:  big.NewInt(77),
				GasFeeCap: big.NewInt(1234),
				GasTipCap: big.NewInt(0),

				// Hardcoded values
				SkipNonceChecks:       true,
				SkipTransactionChecks: true,
			},
			baseFee: big.NewInt(77),
		},
		"maxPriorityFeePerGas and no basefee uses pre-eip1559 rules": {
			args: TransactionArgs{
				MaxPriorityFeePerGas: (*hexutil.Big)(big.NewInt(1234)),
			},
			expectedMsg: core.Message{
				GasLimit: math.MaxInt64,
				Value:    big.NewInt(0),

				GasPrice:  big.NewInt(0),
				GasFeeCap: big.NewInt(0),
				GasTipCap: big.NewInt(0),

				// Hardcoded values
				SkipNonceChecks:       true,
				SkipTransactionChecks: true,
			},
		},
		"maxPriorityFeePerGas and basefee uses eip1559 rules": {
			args: TransactionArgs{
				MaxPriorityFeePerGas: (*hexutil.Big)(big.NewInt(1234)),
			},
			expectedMsg: core.Message{
				GasLimit: math.MaxInt64,
				Value:    big.NewInt(0),

				GasPrice:  big.NewInt(0),
				GasFeeCap: big.NewInt(0),
				GasTipCap: big.NewInt(1234),

				// Hardcoded values
				SkipNonceChecks:       true,
				SkipTransactionChecks: true,
			},
			baseFee: big.NewInt(77),
		},
		"maxFeePerGas, maxPriorityFeePerGas and no basefee uses pre-eip1559 rules": {
			args: TransactionArgs{
				MaxFeePerGas:         (*hexutil.Big)(big.NewInt(1234)),
				MaxPriorityFeePerGas: (*hexutil.Big)(big.NewInt(5678)),
			},
			expectedMsg: core.Message{
				GasLimit: math.MaxInt64,
				Value:    big.NewInt(0),

				GasPrice:  big.NewInt(0),
				GasFeeCap: big.NewInt(0),
				GasTipCap: big.NewInt(0),

				// Hardcoded values
				SkipNonceChecks:       true,
				SkipTransactionChecks: true,
			},
		},
		"maxFeePerGas, maxPriorityFeePerGas and basefee uses eip1559 rules": {
			args: TransactionArgs{
				MaxFeePerGas:         (*hexutil.Big)(big.NewInt(1234)),
				MaxPriorityFeePerGas: (*hexutil.Big)(big.NewInt(5678)),
			},
			expectedMsg: core.Message{
				GasLimit: math.MaxInt64,
				Value:    big.NewInt(0),

				GasPrice:  big.NewInt(1234),
				GasFeeCap: big.NewInt(1234),
				GasTipCap: big.NewInt(5678),

				// Hardcoded values
				SkipNonceChecks:       true,
				SkipTransactionChecks: true,
			},
			baseFee: big.NewInt(77),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			msg, err := test.args.ToMessage(0, test.baseFee, nil)
			require.NoError(t, err, "Failed to convert TransactionArgs to message")

			require.Equal(t, test.expectedMsg, *msg)
		})
	}
}

func TestTransactionArgs_ToMessage_RejectsConversionWithIncoherentGasPricing(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		args TransactionArgs
	}{
		"with maxFeePerGas": {
			args: TransactionArgs{
				GasPrice:     (*hexutil.Big)(big.NewInt(10000000)),
				MaxFeePerGas: (*hexutil.Big)(big.NewInt(20000000)),
			},
		},
		"with maxPriorityFeePerGas": {
			args: TransactionArgs{
				GasPrice:             (*hexutil.Big)(big.NewInt(10000000)),
				MaxPriorityFeePerGas: (*hexutil.Big)(big.NewInt(30000000)),
			},
		},
		"with both maxFeePerGas and maxPriorityFeePerGas": {
			args: TransactionArgs{
				GasPrice:             (*hexutil.Big)(big.NewInt(10000000)),
				MaxFeePerGas:         (*hexutil.Big)(big.NewInt(20000000)),
				MaxPriorityFeePerGas: (*hexutil.Big)(big.NewInt(30000000)),
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			msg, err := tc.args.ToMessage(0, nil, nil)
			require.Nil(t, msg)
			require.EqualError(t, err, "both gasPrice and (maxFeePerGas or maxPriorityFeePerGas) specified")

		})
	}
}

func TestTransactionArgs_ToTransaction(t *testing.T) {

	tests := map[string]struct {
		args     TransactionArgs
		expected *types.Transaction
	}{
		"legacy transaction": {
			args: TransactionArgs{
				To:       &common.Address{0x41},
				Nonce:    new(hexutil.Uint64(0x42)),
				Gas:      new(hexutil.Uint64(0x43)),
				GasPrice: (*hexutil.Big)(big.NewInt(0x44)),
				Value:    (*hexutil.Big)(big.NewInt(0x45)),
				Data:     new(hexutil.Bytes{0x46}),
			},
			expected: types.NewTx(&types.LegacyTx{
				To:       &common.Address{0x41},
				Nonce:    0x42,
				Gas:      0x43,
				GasPrice: big.NewInt(0x44),
				Value:    big.NewInt(0x45),
				Data:     []byte{0x46},
			}),
		},
		"accessList transaction": {
			args: TransactionArgs{
				To:       &common.Address{0x41},
				Nonce:    new(hexutil.Uint64(0x42)),
				Gas:      new(hexutil.Uint64(0x43)),
				GasPrice: (*hexutil.Big)(big.NewInt(0x44)),
				Value:    (*hexutil.Big)(big.NewInt(0x45)),
				Data:     new(hexutil.Bytes{0x46}),
				AccessList: new(types.AccessList{
					{
						Address: common.Address{0x01},
						StorageKeys: []common.Hash{
							common.HexToHash("0x1234"),
							common.HexToHash("0x5678"),
						},
					},
				}),
			},
			expected: types.NewTx(&types.AccessListTx{
				To:       &common.Address{0x41},
				Nonce:    0x42,
				Gas:      0x43,
				GasPrice: big.NewInt(0x44),
				Value:    big.NewInt(0x45),
				Data:     []byte{0x46},
				AccessList: types.AccessList{
					{
						Address: common.Address{0x01},
						StorageKeys: []common.Hash{
							common.HexToHash("0x1234"),
							common.HexToHash("0x5678"),
						},
					},
				},
			}),
		},
		"dynamicFee transaction": {
			args: TransactionArgs{
				To:                   &common.Address{0x41},
				Nonce:                new(hexutil.Uint64(0x42)),
				Gas:                  new(hexutil.Uint64(0x43)),
				MaxFeePerGas:         (*hexutil.Big)(big.NewInt(0x44)),
				MaxPriorityFeePerGas: (*hexutil.Big)(big.NewInt(0x45)),
				Value:                (*hexutil.Big)(big.NewInt(0x46)),
				Data:                 new(hexutil.Bytes{0x47}),
				AccessList: new(types.AccessList{
					{
						Address: common.Address{0x01},
					},
				}),
			},
			expected: types.NewTx(&types.DynamicFeeTx{
				To:        &common.Address{0x41},
				Nonce:     0x42,
				Gas:       0x43,
				GasFeeCap: big.NewInt(0x44),
				GasTipCap: big.NewInt(0x45),
				Value:     big.NewInt(0x46),
				Data:      []byte{0x47},
				AccessList: types.AccessList{
					{
						Address: common.Address{0x01},
					},
				},
			}),
		},
		"blob transaction": {
			args: TransactionArgs{
				To:                   &common.Address{0x41},
				Nonce:                new(hexutil.Uint64(0x42)),
				Gas:                  new(hexutil.Uint64(0x43)),
				MaxFeePerGas:         (*hexutil.Big)(big.NewInt(0x44)),
				MaxPriorityFeePerGas: (*hexutil.Big)(big.NewInt(0x45)),
				Value:                (*hexutil.Big)(big.NewInt(0x46)),
				Data:                 new(hexutil.Bytes{0x47}),
				AccessList: new(types.AccessList{
					{
						Address: common.Address{0x01},
					},
				}),
				BlobFeeCap: (*hexutil.Big)(big.NewInt(0x48)),
				BlobHashes: []common.Hash{
					common.HexToHash("0x49"),
					common.HexToHash("0x4a"),
				},
			},
			expected: types.NewTx(&types.BlobTx{
				To:        common.Address{0x41},
				Nonce:     0x42,
				Gas:       0x43,
				GasFeeCap: uint256.NewInt(0x44),
				GasTipCap: uint256.NewInt(0x45),
				Value:     uint256.NewInt(0x46),
				Data:      []byte{0x47},
				AccessList: types.AccessList{
					{
						Address: common.Address{0x01},
					},
				},
				BlobFeeCap: uint256.NewInt(0x48),
				BlobHashes: []common.Hash{
					common.HexToHash("0x49"),
					common.HexToHash("0x4a"),
				},
			}),
		},
		"setCode transaction": {
			args: TransactionArgs{
				To:                   &common.Address{0x41},
				Nonce:                new(hexutil.Uint64(0x42)),
				Gas:                  new(hexutil.Uint64(0x43)),
				MaxFeePerGas:         (*hexutil.Big)(big.NewInt(0x44)),
				MaxPriorityFeePerGas: (*hexutil.Big)(big.NewInt(0x45)),
				Value:                (*hexutil.Big)(big.NewInt(0x46)),
				Data:                 new(hexutil.Bytes{0x47}),
				AccessList: new(types.AccessList{
					{
						Address: common.Address{0x01},
					},
				}),
				AuthorizationList: []types.SetCodeAuthorization{
					{
						Address: common.Address{0x01},
						Nonce:   0x02,
						ChainID: *uint256.NewInt(0x03),
					},
				},
			},
			expected: types.NewTx(&types.SetCodeTx{
				To:        common.Address{0x41},
				Nonce:     0x42,
				Gas:       0x43,
				GasFeeCap: uint256.NewInt(0x44),
				GasTipCap: uint256.NewInt(0x45),
				Value:     uint256.NewInt(0x46),
				Data:      []byte{0x47},
				AccessList: types.AccessList{
					{
						Address: common.Address{0x01},
					},
				},
				AuthList: []types.SetCodeAuthorization{
					{
						Address: common.Address{0x01},
						Nonce:   0x02,
						ChainID: *uint256.NewInt(0x03),
					},
				},
			}),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			converted, err := test.args.ToTransaction()
			require.NoError(t, err)

			// expected type must match
			require.Equal(t, test.expected.Type(), converted.Type())

			// trivial fields from the legacy transaction type
			require.Equal(t, test.expected.ChainId(), converted.ChainId())
			require.Equal(t, test.expected.To(), converted.To())
			require.Equal(t, test.expected.Nonce(), converted.Nonce())
			require.Equal(t, test.expected.Gas(), converted.Gas())
			require.Equal(t, test.expected.Value(), converted.Value())
			require.Equal(t, test.expected.Data(), converted.Data())

			// access list
			require.Equal(t, test.expected.AccessList(), converted.AccessList())

			// dynamic gas fee - eip1559
			require.Equal(t, test.expected.GasFeeCap(), converted.GasFeeCap())
			require.Equal(t, test.expected.GasTipCap(), converted.GasTipCap())

			// blobs
			require.Equal(t, test.expected.BlobGasFeeCap(), converted.BlobGasFeeCap())
			require.Equal(t, test.expected.BlobHashes(), converted.BlobHashes())

			// set code authorizations
			require.Equal(t, test.expected.SetCodeAuthorizations(), converted.SetCodeAuthorizations())
		})
	}
}

func TestToTransaction_ReturnsErrors(t *testing.T) {
	addr := common.Address{1}
	nonce := hexutil.Uint64(1)
	gas := hexutil.Uint64(21000)
	validFee := (*hexutil.Big)(big.NewInt(1000))

	negative := (*hexutil.Big)(big.NewInt(-1))
	overflow := (*hexutil.Big)(new(big.Int).Lsh(big.NewInt(1), 257))

	tests := map[string]struct {
		args        TransactionArgs
		expectedErr string
	}{
		// --- SetCodeTx errors (triggered by AuthorizationList) ---
		"SetCodeTx: negative MaxFeePerGas": {
			args: TransactionArgs{
				From:                 &addr,
				To:                   &addr,
				Nonce:                &nonce,
				Gas:                  &gas,
				MaxFeePerGas:         negative,
				MaxPriorityFeePerGas: validFee,
				AuthorizationList:    []types.SetCodeAuthorization{{}},
			},
			expectedErr: "invalid MaxFeePerGas",
		},
		"SetCodeTx: overflowing MaxFeePerGas": {
			args: TransactionArgs{
				From:                 &addr,
				To:                   &addr,
				Nonce:                &nonce,
				Gas:                  &gas,
				MaxFeePerGas:         overflow,
				MaxPriorityFeePerGas: validFee,
				AuthorizationList:    []types.SetCodeAuthorization{{}},
			},
			expectedErr: "invalid MaxFeePerGas",
		},
		"SetCodeTx: negative MaxPriorityFeePerGas": {
			args: TransactionArgs{
				From:                 &addr,
				To:                   &addr,
				Nonce:                &nonce,
				Gas:                  &gas,
				MaxFeePerGas:         validFee,
				MaxPriorityFeePerGas: negative,
				AuthorizationList:    []types.SetCodeAuthorization{{}},
			},
			expectedErr: "invalid MaxPriorityFeePerGas",
		},
		"SetCodeTx: overflowing MaxPriorityFeePerGas": {
			args: TransactionArgs{
				From:                 &addr,
				To:                   &addr,
				Nonce:                &nonce,
				Gas:                  &gas,
				MaxFeePerGas:         validFee,
				MaxPriorityFeePerGas: overflow,
				AuthorizationList:    []types.SetCodeAuthorization{{}},
			},
			expectedErr: "invalid MaxPriorityFeePerGas",
		},
		"SetCodeTx: negative ChainID": {
			args: TransactionArgs{
				From:                 &addr,
				To:                   &addr,
				Nonce:                &nonce,
				Gas:                  &gas,
				ChainID:              negative,
				MaxFeePerGas:         validFee,
				MaxPriorityFeePerGas: validFee,
				AuthorizationList:    []types.SetCodeAuthorization{{}},
			},
			expectedErr: "invalid ChainID",
		},
		"SetCodeTx: overflowing ChainID": {
			args: TransactionArgs{
				From:                 &addr,
				To:                   &addr,
				Nonce:                &nonce,
				Gas:                  &gas,
				ChainID:              overflow,
				MaxFeePerGas:         validFee,
				MaxPriorityFeePerGas: validFee,
				AuthorizationList:    []types.SetCodeAuthorization{{}},
			},
			expectedErr: "invalid ChainID",
		},
		"SetCodeTx: negative Value": {
			args: TransactionArgs{
				From:                 &addr,
				To:                   &addr,
				Nonce:                &nonce,
				Gas:                  &gas,
				Value:                negative,
				MaxFeePerGas:         validFee,
				MaxPriorityFeePerGas: validFee,
				AuthorizationList:    []types.SetCodeAuthorization{{}},
			},
			expectedErr: "invalid Value",
		},
		"SetCodeTx: overflowing Value": {
			args: TransactionArgs{
				From:                 &addr,
				To:                   &addr,
				Nonce:                &nonce,
				Gas:                  &gas,
				Value:                overflow,
				MaxFeePerGas:         validFee,
				MaxPriorityFeePerGas: validFee,
				AuthorizationList:    []types.SetCodeAuthorization{{}},
			},
			expectedErr: "invalid Value",
		},

		// --- BlobTx errors (triggered by BlobFeeCap or BlobHashes) ---
		"BlobTx: negative MaxFeePerGas": {
			args: TransactionArgs{
				From:                 &addr,
				To:                   &addr,
				Nonce:                &nonce,
				Gas:                  &gas,
				MaxFeePerGas:         negative,
				MaxPriorityFeePerGas: validFee,
				BlobFeeCap:           validFee,
				BlobHashes:           []common.Hash{{1}},
			},
			expectedErr: "invalid MaxFeePerGas",
		},
		"BlobTx: overflowing MaxFeePerGas": {
			args: TransactionArgs{
				From:                 &addr,
				To:                   &addr,
				Nonce:                &nonce,
				Gas:                  &gas,
				MaxFeePerGas:         overflow,
				MaxPriorityFeePerGas: validFee,
				BlobFeeCap:           validFee,
				BlobHashes:           []common.Hash{{1}},
			},
			expectedErr: "invalid MaxFeePerGas",
		},
		"BlobTx: negative MaxPriorityFeePerGas": {
			args: TransactionArgs{
				From:                 &addr,
				To:                   &addr,
				Nonce:                &nonce,
				Gas:                  &gas,
				MaxFeePerGas:         validFee,
				MaxPriorityFeePerGas: negative,
				BlobFeeCap:           validFee,
				BlobHashes:           []common.Hash{{1}},
			},
			expectedErr: "invalid MaxPriorityFeePerGas",
		},
		"BlobTx: overflowing MaxPriorityFeePerGas": {
			args: TransactionArgs{
				From:                 &addr,
				To:                   &addr,
				Nonce:                &nonce,
				Gas:                  &gas,
				MaxFeePerGas:         validFee,
				MaxPriorityFeePerGas: overflow,
				BlobFeeCap:           validFee,
				BlobHashes:           []common.Hash{{1}},
			},
			expectedErr: "invalid MaxPriorityFeePerGas",
		},
		"BlobTx: negative BlobFeeCap": {
			args: TransactionArgs{
				From:                 &addr,
				To:                   &addr,
				Nonce:                &nonce,
				Gas:                  &gas,
				MaxFeePerGas:         validFee,
				MaxPriorityFeePerGas: validFee,
				BlobFeeCap:           negative,
				BlobHashes:           []common.Hash{{1}},
			},
			expectedErr: "invalid BlobFeeCap",
		},
		"BlobTx: overflowing BlobFeeCap": {
			args: TransactionArgs{
				From:                 &addr,
				To:                   &addr,
				Nonce:                &nonce,
				Gas:                  &gas,
				MaxFeePerGas:         validFee,
				MaxPriorityFeePerGas: validFee,
				BlobFeeCap:           overflow,
				BlobHashes:           []common.Hash{{1}},
			},
			expectedErr: "invalid BlobFeeCap",
		},
		"BlobTx: negative ChainID": {
			args: TransactionArgs{
				From:                 &addr,
				To:                   &addr,
				Nonce:                &nonce,
				Gas:                  &gas,
				ChainID:              negative,
				MaxFeePerGas:         validFee,
				MaxPriorityFeePerGas: validFee,
				BlobFeeCap:           validFee,
				BlobHashes:           []common.Hash{{1}},
			},
			expectedErr: "invalid ChainID",
		},
		"BlobTx: overflowing ChainID": {
			args: TransactionArgs{
				From:                 &addr,
				To:                   &addr,
				Nonce:                &nonce,
				Gas:                  &gas,
				ChainID:              overflow,
				MaxFeePerGas:         validFee,
				MaxPriorityFeePerGas: validFee,
				BlobFeeCap:           validFee,
				BlobHashes:           []common.Hash{{1}},
			},
			expectedErr: "invalid ChainID",
		},
		"BlobTx: negative Value": {
			args: TransactionArgs{
				From:                 &addr,
				To:                   &addr,
				Nonce:                &nonce,
				Gas:                  &gas,
				Value:                negative,
				MaxFeePerGas:         validFee,
				MaxPriorityFeePerGas: validFee,
				BlobFeeCap:           validFee,
				BlobHashes:           []common.Hash{{1}},
			},
			expectedErr: "invalid Value",
		},
		"BlobTx: overflowing Value": {
			args: TransactionArgs{
				From:                 &addr,
				To:                   &addr,
				Nonce:                &nonce,
				Gas:                  &gas,
				Value:                overflow,
				MaxFeePerGas:         validFee,
				MaxPriorityFeePerGas: validFee,
				BlobFeeCap:           validFee,
				BlobHashes:           []common.Hash{{1}},
			},
			expectedErr: "invalid Value",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tx, err := tt.args.ToTransaction()
			require.Error(t, err)
			require.Nil(t, tx)
			require.ErrorContains(t, err, tt.expectedErr)
		})
	}
}

func TestTransactionArgs_ToMessageIsEquivalentToEvmcoreToMessage(t *testing.T) {
	t.Parallel()

	// Verify tx args -> message conversion is equivalent to
	// tx args -> transaction -> message conversion.

	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	from := crypto.PubkeyToAddress(key.PublicKey)
	to := common.Address{0xaa}
	nonce := hexutil.Uint64(7)
	gas := hexutil.Uint64(123_456)
	chainID := (*hexutil.Big)(big.NewInt(1337))
	gasPrice := (*hexutil.Big)(big.NewInt(21))
	maxFeePerGas := (*hexutil.Big)(big.NewInt(100))
	maxPriorityFeePerGas := (*hexutil.Big)(big.NewInt(3))
	blobFeeCap := (*hexutil.Big)(big.NewInt(7))
	baseFee := big.NewInt(2)
	inputData := hexutil.Bytes{0x01, 0x02, 0x03}
	accessList := types.AccessList{{Address: common.Address{0x01}}}

	tests := map[string]TransactionArgs{
		"legacy tx": {
			From:     &from,
			To:       &to,
			Nonce:    &nonce,
			Gas:      &gas,
			GasPrice: gasPrice,
			Value:    (*hexutil.Big)(big.NewInt(11)),
			Input:    &inputData,
		},
		"access list tx": {
			From:       &from,
			To:         &to,
			Nonce:      &nonce,
			Gas:        &gas,
			GasPrice:   gasPrice,
			Value:      (*hexutil.Big)(big.NewInt(11)),
			Input:      &inputData,
			AccessList: &accessList,
			ChainID:    chainID,
		},
		"dynamic fee tx": {
			From:                 &from,
			To:                   &to,
			Nonce:                &nonce,
			Gas:                  &gas,
			Value:                (*hexutil.Big)(big.NewInt(11)),
			Input:                &inputData,
			AccessList:           &accessList,
			ChainID:              chainID,
			MaxFeePerGas:         maxFeePerGas,
			MaxPriorityFeePerGas: maxPriorityFeePerGas,
		},
		"blob tx with blob hashes": {
			From:                 &from,
			To:                   &to,
			Nonce:                &nonce,
			Gas:                  &gas,
			Value:                (*hexutil.Big)(big.NewInt(11)),
			Input:                &inputData,
			AccessList:           &accessList,
			ChainID:              chainID,
			MaxFeePerGas:         maxFeePerGas,
			MaxPriorityFeePerGas: maxPriorityFeePerGas,
			BlobFeeCap:           blobFeeCap,
			BlobHashes: []common.Hash{
				common.HexToHash("0x49"),
				common.HexToHash("0x4a"),
			},
		},
		"blob tx with empty blob hashes": {
			From:                 &from,
			To:                   &to,
			Nonce:                &nonce,
			Gas:                  &gas,
			Value:                (*hexutil.Big)(big.NewInt(11)),
			Input:                &inputData,
			AccessList:           &accessList,
			ChainID:              chainID,
			MaxFeePerGas:         maxFeePerGas,
			MaxPriorityFeePerGas: maxPriorityFeePerGas,
			BlobFeeCap:           blobFeeCap,
			BlobHashes:           []common.Hash{},
		},
		"set code tx": {
			From:                 &from,
			To:                   &to,
			Nonce:                &nonce,
			Gas:                  &gas,
			Value:                (*hexutil.Big)(big.NewInt(11)),
			Input:                &inputData,
			AccessList:           &accessList,
			ChainID:              chainID,
			MaxFeePerGas:         maxFeePerGas,
			MaxPriorityFeePerGas: maxPriorityFeePerGas,
			AuthorizationList: []types.SetCodeAuthorization{
				{
					Address: common.Address{0x01},
					Nonce:   0x02,
					ChainID: *uint256.NewInt(0x03),
				},
			},
		},
	}

	for name, args := range tests {
		t.Run(name, func(t *testing.T) {
			directMsg, err := args.ToMessage(0, baseFee, nil)
			require.NoError(t, err)

			tx, err := args.ToTransaction()
			require.NoError(t, err)

			signer := types.LatestSignerForChainID(args.ChainID.ToInt())
			signedTx, err := types.SignTx(tx, signer, key)
			require.NoError(t, err)

			msgFromTx, err := evmcore.TxAsMessage(signedTx, signer, baseFee)
			require.NoError(t, err)

			// RPC messages usually skip checks; normalize to compare payload fields.
			directMsg.SkipNonceChecks = false
			directMsg.SkipTransactionChecks = false

			require.Equal(t, directMsg, msgFromTx)
		})
	}
}
