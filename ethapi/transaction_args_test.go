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

package ethapi

import (
	"math"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestTransactionArgs_GasCap(t *testing.T) {
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
		msg, err := args.ToMessage(test.globalGasCap, nil)
		require.Nil(t, err)
		require.Equal(t, test.expectedGas, msg.GasLimit, test.name)
	}
}

func TestTransactionArgs_ToMessage_Empty(t *testing.T) {
	t.Parallel()

	empty := TransactionArgs{}
	msg, err := empty.ToMessage(0x123, big.NewInt(100))
	require.NoError(t, err, "Failed to convert empty TransactionArgs to message")

	require.NotNil(t, msg)
	require.Nil(t, msg.To)
	require.Equal(t, uint64(0x123), msg.GasLimit)
	require.Equal(t, big.NewInt(0), msg.GasPrice)
	require.Equal(t, big.NewInt(0), msg.Value)
	require.Nil(t, msg.BlobGasFeeCap)
	require.Equal(t, big.NewInt(0), msg.GasTipCap)
	require.Equal(t, uint64(0), msg.Nonce, "Nonce should be hardcoded to 0")
}

func TestTransactionArgs_ToMessage_TrivialFieldsAreCopied(t *testing.T) {
	t.Parallel()
	// this test checks that the trivial fields of TransactionArgs
	// are correctly converted to a core.Message,
	// Trivial fields are those which do not include any logic

	txArgs := TransactionArgs{
		To:    &common.Address{0x1},
		Nonce: asPointer(hexutil.Uint64(0x2)),
		Value: (*hexutil.Big)(big.NewInt(0x3)),
		Data:  asPointer(hexutil.Bytes([]byte{0x4})),
		Gas:   asPointer(hexutil.Uint64(0x5)),
		AccessList: asPointer(
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
	msg, err := txArgs.ToMessage(0x4321, big.NewInt(100))
	require.NoError(t, err)

	require.Equal(t, core.Message{
		To:       &common.Address{0x1},
		Nonce:    0, // hardcoded to 0
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
		SkipNonceChecks:  true,
		SkipFromEOACheck: true,
	}, *msg)

}

func TestTransactionArgs_ToMessage_GasPriceFollowsEIP1559Rules(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		args        TransactionArgs
		expectedMsg core.Message
		baseFee     *big.Int
	}{
		"zero initialized": {
			args: TransactionArgs{},
			expectedMsg: core.Message{
				GasLimit: math.MaxInt64,
				Value:    big.NewInt(0),

				GasPrice:  big.NewInt(0),
				GasFeeCap: big.NewInt(0),
				GasTipCap: big.NewInt(0),

				// Hardcoded values
				SkipNonceChecks:  true,
				SkipFromEOACheck: true,
			},
			baseFee: big.NewInt(0),
		},
		"legacy gas price model": {
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
				SkipNonceChecks:  true,
				SkipFromEOACheck: true,
			},
			baseFee: nil, // is ignored
		},
		"legacy gas price model with basefee": {
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
				SkipNonceChecks:  true,
				SkipFromEOACheck: true,
			},
			baseFee: big.NewInt(77), // is ignored
		},
		"eip1559 with gas fee cap": {
			args: TransactionArgs{
				MaxFeePerGas: (*hexutil.Big)(big.NewInt(1234)),
			},
			expectedMsg: core.Message{
				GasLimit: math.MaxInt64,
				Value:    big.NewInt(0),

				// nil basefee, the algorithm will treat the transaction
				// as legacy with 0 GasPrice
				GasPrice:  big.NewInt(0),
				GasFeeCap: big.NewInt(0),
				GasTipCap: big.NewInt(0),

				// Hardcoded values
				SkipNonceChecks:  true,
				SkipFromEOACheck: true,
			},
			baseFee: nil, // is ignored
		},
		"eip1559 with gas fee cap and basefee": {
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
				SkipNonceChecks:  true,
				SkipFromEOACheck: true,
			},
			baseFee: big.NewInt(77),
		},
		"eip1559 with gas tip cap": {
			args: TransactionArgs{
				MaxPriorityFeePerGas: (*hexutil.Big)(big.NewInt(1234)),
			},
			expectedMsg: core.Message{
				GasLimit: math.MaxInt64,
				Value:    big.NewInt(0),

				// nil basefee, the algorithm will treat the transaction
				// as legacy with 0 GasPrice
				GasPrice:  big.NewInt(0),
				GasFeeCap: big.NewInt(0),
				GasTipCap: big.NewInt(0),

				// Hardcoded values
				SkipNonceChecks:  true,
				SkipFromEOACheck: true,
			},
			baseFee: nil,
		},
		"eip1559 with gas tip cap and basefee": {
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
				SkipNonceChecks:  true,
				SkipFromEOACheck: true,
			},
			baseFee: big.NewInt(77),
		},
		"eip1559 with gas fee cap and gas tip cap": {
			args: TransactionArgs{
				MaxFeePerGas:         (*hexutil.Big)(big.NewInt(1234)),
				MaxPriorityFeePerGas: (*hexutil.Big)(big.NewInt(5678)),
			},
			expectedMsg: core.Message{
				GasLimit: math.MaxInt64,
				Value:    big.NewInt(0),

				// nil basefee, the algorithm will treat the transaction
				// as legacy with 0 GasPrice
				GasPrice:  big.NewInt(0),
				GasFeeCap: big.NewInt(0),
				GasTipCap: big.NewInt(0),

				// Hardcoded values
				SkipNonceChecks:  true,
				SkipFromEOACheck: true,
			},
		},
		"eip1559 with gas fee cap and gas tip cap and base fee": {
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
				SkipNonceChecks:  true,
				SkipFromEOACheck: true,
			},
			baseFee: big.NewInt(77),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			msg, err := test.args.ToMessage(0, test.baseFee)
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
			msg, err := tc.args.ToMessage(0, nil)
			require.Nil(t, msg)
			require.EqualError(t, err, "both gasPrice and (maxFeePerGas or maxPriorityFeePerGas) specified")
		})
	}
}

// asPointer is a helper function to convert a value to a pointer,
// useful to inline hexutil types in tests.
func asPointer[T any](v T) *T {
	return &v
}
