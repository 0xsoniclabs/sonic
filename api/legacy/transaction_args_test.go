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

package legacy

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/0xsoniclabs/sonic/evmcore"
)

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
		"SetCodeTx: missing To": {
			args: TransactionArgs{
				From:                 &addr,
				Nonce:                &nonce,
				Gas:                  &gas,
				MaxFeePerGas:         validFee,
				MaxPriorityFeePerGas: validFee,
				AuthorizationList:    []types.SetCodeAuthorization{{}},
			},
			expectedErr: "missing To in SetCodeTx",
		},
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
		"BlobTx: missing To": {
			args: TransactionArgs{
				From:                 &addr,
				Nonce:                &nonce,
				Gas:                  &gas,
				MaxFeePerGas:         validFee,
				MaxPriorityFeePerGas: validFee,
				BlobFeeCap:           validFee,
				BlobHashes:           []common.Hash{{1}},
			},
			expectedErr: "missing To in BlobTx",
		},
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

// newSetDefaultsTestBackend builds a mock Backend with the expectations
// needed to reach the post-gas-pricing checks in setDefaults, i.e. up to and
// including the "to == nil" create checks and the chainID check.
func newSetDefaultsTestBackend(t *testing.T, chainID *big.Int) *MockBackend {
	t.Helper()
	ctrl := gomock.NewController(t)
	backend := NewMockBackend(ctrl)

	block := &evmcore.EvmBlock{}
	block.Number = big.NewInt(1)

	backend.EXPECT().CurrentBlock().Return(block).AnyTimes()
	backend.EXPECT().ChainConfig(gomock.Any()).Return(&params.ChainConfig{}).AnyTimes()
	backend.EXPECT().ChainID().Return(chainID).AnyTimes()
	return backend
}

func TestTransactionArgs_SetDefaults_RejectsGasPriceWithAuthorizationList(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	backend := NewMockBackend(ctrl)

	args := TransactionArgs{
		GasPrice:          (*hexutil.Big)(big.NewInt(1)),
		AuthorizationList: []types.SetCodeAuthorization{{}},
	}

	err := args.setDefaults(context.Background(), backend)
	require.EqualError(t, err, "both gasPrice and authorizationList specified")
}

func TestTransactionArgs_SetDefaults_RejectsMissingToWithBlobHashes(t *testing.T) {
	t.Parallel()

	addr := common.Address{1}
	nonce := hexutil.Uint64(0)
	gas := hexutil.Uint64(21000)
	fee := (*hexutil.Big)(big.NewInt(100))

	backend := newSetDefaultsTestBackend(t, big.NewInt(1))

	args := TransactionArgs{
		From:                 &addr,
		Nonce:                &nonce,
		Gas:                  &gas,
		MaxFeePerGas:         fee,
		MaxPriorityFeePerGas: fee,
		BlobHashes:           []common.Hash{{1}},
	}

	err := args.setDefaults(context.Background(), backend)
	require.EqualError(t, err, `missing "to" in blob transaction`)
}

func TestTransactionArgs_SetDefaults_RejectsMissingToWithBlobFeeCap(t *testing.T) {
	t.Parallel()

	addr := common.Address{1}
	nonce := hexutil.Uint64(0)
	gas := hexutil.Uint64(21000)
	fee := (*hexutil.Big)(big.NewInt(100))
	data := hexutil.Bytes{0x01}

	backend := newSetDefaultsTestBackend(t, big.NewInt(1))

	// BlobFeeCap alone makes ToTransaction build a BlobTx, which needs "to".
	args := TransactionArgs{
		From:                 &addr,
		Nonce:                &nonce,
		Gas:                  &gas,
		MaxFeePerGas:         fee,
		MaxPriorityFeePerGas: fee,
		Data:                 &data,
		BlobFeeCap:           fee,
	}

	err := args.setDefaults(context.Background(), backend)
	require.EqualError(t, err, `missing "to" in blob transaction`)
}

func TestTransactionArgs_SetDefaults_RejectsMissingToWithAuthorizationList(t *testing.T) {
	t.Parallel()

	addr := common.Address{1}
	nonce := hexutil.Uint64(0)
	gas := hexutil.Uint64(21000)
	fee := (*hexutil.Big)(big.NewInt(100))
	data := hexutil.Bytes{0x01}

	backend := newSetDefaultsTestBackend(t, big.NewInt(1))

	args := TransactionArgs{
		From:                 &addr,
		Nonce:                &nonce,
		Gas:                  &gas,
		MaxFeePerGas:         fee,
		MaxPriorityFeePerGas: fee,
		Data:                 &data,
		AuthorizationList:    []types.SetCodeAuthorization{{}},
	}

	err := args.setDefaults(context.Background(), backend)
	require.EqualError(t, err, `authorizationList provided for contract creation, but "to" field is missing`)
}

func TestTransactionArgs_SetDefaults_ChainID(t *testing.T) {
	t.Parallel()

	addr := common.Address{1}
	nonce := hexutil.Uint64(0)
	gas := hexutil.Uint64(21000)
	fee := (*hexutil.Big)(big.NewInt(100))
	nodeChainID := big.NewInt(146)

	tests := map[string]struct {
		chainID     *hexutil.Big
		expectedErr string
	}{
		"nil chainID defaults to node's chainID": {
			chainID: nil,
		},
		"matching chainID is accepted": {
			chainID: (*hexutil.Big)(nodeChainID),
		},
		"mismatching chainID is rejected": {
			chainID:     (*hexutil.Big)(big.NewInt(1337)),
			expectedErr: "chainId does not match node's (have=1337, want=146)",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			backend := newSetDefaultsTestBackend(t, nodeChainID)

			args := TransactionArgs{
				From:                 &addr,
				To:                   &addr,
				Nonce:                &nonce,
				Gas:                  &gas,
				MaxFeePerGas:         fee,
				MaxPriorityFeePerGas: fee,
				ChainID:              test.chainID,
			}

			err := args.setDefaults(context.Background(), backend)
			if test.expectedErr != "" {
				require.EqualError(t, err, test.expectedErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, (*hexutil.Big)(nodeChainID), args.ChainID)
		})
	}
}
