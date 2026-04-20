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

package bundle

import (
	"bytes"
	"fmt"
	"math"
	"math/rand/v2"
	"testing"

	"github.com/0xsoniclabs/sonic/utils/checked"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

func Test_calculateEnvelopeGas_ComputesGasBasedOnProvidedParameters(t *testing.T) {
	const markerCosts = params.TxAccessListAddressGas + params.TxAccessListStorageKeyGas
	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	largePayload := append(bytes.Repeat([]byte{0}, 500), bytes.Repeat([]byte{0xFF}, 700)...)

	tests := map[string]struct {
		bundle     TransactionBundle
		payload    []byte
		accessList types.AccessList
		authList   []types.SetCodeAuthorization
		expected   uint64
	}{
		"empty": {
			expected: params.TxGas,
		},
		"transactions dominating gas requirements": {
			bundle: NewBuilder().AllOf(
				Step(key, types.AccessListTx{Gas: 100_000}),
				Step(key, types.AccessListTx{Gas: 250_000}),
			).BuildBundle(),
			expected: 100_000 + 250_000 + 2*markerCosts,
		},
		"payload dominating gas requirements": {
			payload: largePayload,
			expected: func() uint64 {
				res, err := core.FloorDataGas(largePayload)
				require.NoError(t, err)
				return res
			}(),
		},
		"access list dominating gas requirements": {
			accessList: make([]types.AccessTuple, 1000),
			expected:   1000*params.TxAccessListAddressGas + params.TxGas,
		},
		"access slot dominating gas requirements": {
			accessList: []types.AccessTuple{
				{StorageKeys: make([]common.Hash, 1000)},
			},
			expected: 1000*params.TxAccessListStorageKeyGas +
				params.TxAccessListAddressGas +
				params.TxGas,
		},
		"set code authorization dominating gas requirements": {
			authList: make([]types.SetCodeAuthorization, 1000),
			expected: 1000*params.CallNewAccountGas + params.TxGas,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)

			// cross-check expected gas limit
			want, err := calculateEnvelopeGasInternal(
				tc.bundle.GetTransactionsInReferencedOrder(),
				uint64(bytes.Count(tc.payload, []byte{0})),
				uint64(len(tc.payload)-bytes.Count(tc.payload, []byte{0})),
				uint64(len(tc.accessList)),
				uint64(tc.accessList.StorageKeys()),
				uint64(len(tc.authList)),
			)
			require.NoError(err)
			require.Equal(int(tc.expected), int(want))

			// test that function produces expected value
			have, err := calculateEnvelopeGas(
				tc.bundle,
				tc.payload,
				tc.accessList,
				tc.authList,
			)
			require.NoError(err)
			require.Equal(int(tc.expected), int(have))
		})
	}
}

func Test_calculateEnvelopeGas_ComputesNonZeroAndZeroBytesInPayloadCorrectly(t *testing.T) {
	tests := map[string][]byte{
		"empty":                         nil,
		"only zero bytes":               make([]byte, 10),
		"only non-zero bytes":           bytes.Repeat([]byte{0xFF}, 10),
		"mixed zero and non-zero bytes": append(bytes.Repeat([]byte{0}, 5), bytes.Repeat([]byte{0xFF}, 5)...),
	}

	for name, payload := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)

			zeros := bytes.Count(payload, []byte{0})
			nonZeros := len(payload) - zeros

			want, err := calculateEnvelopeGasInternal(
				nil, uint64(zeros), uint64(nonZeros), 0, 0, 0,
			)
			require.NoError(err)

			have, err := calculateEnvelopeGas(
				TransactionBundle{}, payload, nil, nil,
			)
			require.NoError(err)
			require.Equal(want, have)
		})
	}
}

func Test_calculateEnvelopeGasInternal_UsesMaximumOfIntrinsicFloorAndTransactionGas(t *testing.T) {
	tests := map[string]struct {
		transactions           []uint64
		numNonZeroBytesInData  uint64
		numAccessListAddresses uint64
		wantedMax              string
	}{
		"max intrinsic gas": {
			numAccessListAddresses: 100, // < only contributes to intrinsic gas
			wantedMax:              "intrinsic",
		},
		"max floor gas": {
			numNonZeroBytesInData: 100, // < more expensive in floor costs than in intrinsic costs
			wantedMax:             "floor",
		},
		"max transaction gas": {
			transactions: []uint64{100000}, // < more expensive than intrinsic and floor costs in this test
			wantedMax:    "transactions",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)

			intrinsic, err := calculateIntrinsicGas(
				0,
				tc.numNonZeroBytesInData,
				tc.numAccessListAddresses,
				0,
				0,
			)
			require.NoError(err)

			floorDataGas, err := calculateFloorDataGas(
				0, tc.numNonZeroBytesInData,
			)
			require.NoError(err)

			transactions := make([]*types.Transaction, len(tc.transactions))
			for i, gas := range tc.transactions {
				transactions[i] = types.NewTx(&types.LegacyTx{
					Gas: gas,
				})
			}

			txGasSum, err := calculateTxGasSum(transactions)
			require.NoError(err)

			switch tc.wantedMax {
			case "intrinsic":
				require.Greater(intrinsic, floorDataGas)
				require.Greater(intrinsic, txGasSum)
			case "floor":
				require.Greater(floorDataGas, intrinsic)
				require.Greater(floorDataGas, txGasSum)
			case "transactions":
				require.Greater(txGasSum, floorDataGas)
				require.Greater(txGasSum, intrinsic)
			default:
				require.True(false, "unsupported test case spec")
			}

			want := max(intrinsic, floorDataGas, txGasSum)
			have, err := calculateEnvelopeGasInternal(
				transactions, 0, tc.numNonZeroBytesInData,
				tc.numAccessListAddresses, 0, 0,
			)
			require.NoError(err)
			require.Equal(want, have)
		})
	}
}

func Test_calculateEnvelopeGasInternal_CombinationTests(t *testing.T) {

	txCosts := [][]uint64{
		{},
		{1},
		{21_000, 50_000},
		{100_000, 150_000},
		{21_000, 50_000, 10_000_000},
	}

	valueRanges := []uint64{
		0, 1, 10, 100, 1000,
	}

	for _, txCosts := range txCosts {
		for _, numZeroBytes := range valueRanges {
			for _, numNonZeroBytes := range valueRanges {
				for _, numAccessListAddresses := range valueRanges {
					for _, numAccessListStorageKeys := range valueRanges {
						for _, numAuthListEntries := range valueRanges {
							name := fmt.Sprintf(
								"txCosts=%v/zeroBytes=%d/nonZeroBytes=%d/accessListAddresses=%d/accessListStorageKeys=%d/authListEntries=%d",
								txCosts, numZeroBytes, numNonZeroBytes, numAccessListAddresses, numAccessListStorageKeys, numAuthListEntries,
							)
							t.Run(name, func(t *testing.T) {
								require := require.New(t)

								transactions := make([]*types.Transaction, len(txCosts))
								for i, gas := range txCosts {
									transactions[i] = types.NewTx(&types.LegacyTx{
										Gas: gas,
									})
								}

								intrinsic, err := calculateIntrinsicGas(
									numZeroBytes,
									numNonZeroBytes,
									numAccessListAddresses,
									numAccessListStorageKeys,
									numAuthListEntries,
								)
								require.NoError(err)

								floorDataGas, err := calculateFloorDataGas(
									numZeroBytes, numNonZeroBytes,
								)
								require.NoError(err)

								txGasSum, err := calculateTxGasSum(transactions)
								require.NoError(err)

								want := max(intrinsic, floorDataGas, txGasSum)
								have, err := calculateEnvelopeGasInternal(
									transactions, numZeroBytes, numNonZeroBytes,
									numAccessListAddresses, numAccessListStorageKeys,
									numAuthListEntries,
								)
								require.NoError(err)
								require.Equal(want, have)
							})
						}
					}
				}
			}
		}
	}
}

func Test_calculateEnvelopeGasInternal_DetectsIntrinsicGasOverflow(t *testing.T) {
	numAccessListSlots := uint64(math.MaxUint64)

	_, err := calculateEnvelopeGasInternal(nil, 0, 0, 0, numAccessListSlots, 0)
	require.ErrorContains(t, err, "intrinsic gas")
	require.ErrorIs(t, err, checked.ErrOverflow)
}

func Test_calculateEnvelopeGasInternal_DetectsFloorDataGasOverflow(t *testing.T) {
	// low enough to pass the intrinsic gas computation, but high enough to fail
	// in the floor data gas computation, where non-zero bytes are more expensive
	numNonZeroBytesInData := uint64(math.MaxUint64/40 + 1)

	_, err := calculateEnvelopeGasInternal(nil, 0, numNonZeroBytesInData, 0, 0, 0)
	require.ErrorContains(t, err, "floor data gas")
	require.ErrorIs(t, err, checked.ErrOverflow)
}

func Test_calculateEnvelopeGasInternal_DetectsTxGasSumOverflow(t *testing.T) {
	transactions := []*types.Transaction{
		types.NewTx(&types.LegacyTx{Gas: math.MaxUint64 - 1000}),
		types.NewTx(&types.LegacyTx{Gas: 2000}),
	}

	_, err := calculateEnvelopeGasInternal(transactions, 0, 0, 0, 0, 0)
	require.ErrorContains(t, err, "transaction gas sum")
	require.ErrorIs(t, err, checked.ErrOverflow)
}

func Test_calculateIntrinsicGas_MatchesGethImplementation(t *testing.T) {
	tests := map[string]struct {
		data       []byte
		accessList types.AccessList
		authList   []types.SetCodeAuthorization
	}{
		"empty transaction": {},
		"short data": {
			data: make([]byte, 1),
		},
		"data with only zero bytes": {
			data: make([]byte, 10),
		},
		"data with only non-zero bytes": {
			data: bytes.Repeat([]byte{0xFF}, 10),
		},
		"data with mixed zero and non-zero bytes": {
			data: append(bytes.Repeat([]byte{0}, 5), bytes.Repeat([]byte{0xFF}, 5)...),
		},
		"short access list": {
			accessList: types.AccessList{{}},
		},
		"long access list": {
			accessList: types.AccessList{{}, {}, {}, {}, {}, {}, {}, {}, {}, {}},
		},
		"few slots in access list": {
			accessList: types.AccessList{{
				StorageKeys: []common.Hash{{}, {}, {}},
			}},
		},
		"many slots in access list": {
			accessList: types.AccessList{{
				StorageKeys: []common.Hash{{}, {}, {}, {}, {}, {}, {}, {}, {}, {}},
			}},
		},
		"few set code authorizations": {
			authList: []types.SetCodeAuthorization{{}, {}},
		},
		"many set code authorizations": {
			authList: []types.SetCodeAuthorization{{}, {}, {}, {}, {}, {}, {}, {}, {}, {}},
		},
		"complex transaction": {
			data: append(bytes.Repeat([]byte{0}, 5), bytes.Repeat([]byte{0xFF}, 5)...),
			accessList: types.AccessList{
				{},
				{
					StorageKeys: []common.Hash{{}, {}, {}},
				},
			},
			authList: []types.SetCodeAuthorization{{}, {}, {}},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			want, err := core.IntrinsicGas(
				tc.data,
				tc.accessList,
				tc.authList,
				false, // envelopes are not a contract creations
				true,  // is homestead
				true,  // is istanbul
				true,  // is shanghai
			)
			require.NoError(err)

			zeros := bytes.Count(tc.data, []byte{0})
			nonZeros := len(tc.data) - zeros

			got, err := calculateIntrinsicGas(
				uint64(zeros),
				uint64(nonZeros),
				uint64(len(tc.accessList)),
				uint64(tc.accessList.StorageKeys()),
				uint64(len(tc.authList)),
			)
			require.NoError(err)
			require.Equal(want, got)
		})
	}
}

func Test_calculateIntrinsicGas_DetectsOverflows(t *testing.T) {
	tests := map[string]struct {
		numZeroBytes             uint64
		numNonZeroBytes          uint64
		numAccessListAddresses   uint64
		numAccessListStorageKeys uint64
		numAuthListEntries       uint64
	}{
		"close overflow from zero bytes": {
			numZeroBytes: math.MaxUint64/params.TxDataZeroGas + 1,
		},
		"big overflow from zero bytes": {
			numZeroBytes: math.MaxUint64,
		},
		"close overflow from non-zero bytes": {
			numNonZeroBytes: math.MaxUint64/params.TxDataNonZeroGasEIP2028 + 1,
		},
		"big overflow from non-zero bytes": {
			numNonZeroBytes: math.MaxUint64,
		},
		"close overflow from access list addresses": {
			numAccessListAddresses: math.MaxUint64/params.TxAccessListAddressGas + 1,
		},
		"big overflow from access list addresses": {
			numAccessListAddresses: math.MaxUint64,
		},
		"close overflow from access list storage keys": {
			numAccessListStorageKeys: math.MaxUint64/params.TxAccessListStorageKeyGas + 1,
		},
		"big overflow from access list storage keys": {
			numAccessListStorageKeys: math.MaxUint64,
		},
		"close overflow from set code authorizations": {
			numAuthListEntries: math.MaxUint64/params.CallNewAccountGas + 1,
		},
		"big overflow from set code authorizations": {
			numAuthListEntries: math.MaxUint64,
		},
		"combination overflow": {
			numZeroBytes:             math.MaxUint64 / 64,
			numNonZeroBytes:          math.MaxUint64 / 64,
			numAccessListAddresses:   math.MaxUint64 / 64,
			numAccessListStorageKeys: math.MaxUint64 / 64,
			numAuthListEntries:       math.MaxUint64 / 64,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			_, err := calculateIntrinsicGas(
				tc.numZeroBytes,
				tc.numNonZeroBytes,
				tc.numAccessListAddresses,
				tc.numAccessListStorageKeys,
				tc.numAuthListEntries,
			)
			require.ErrorIs(err, checked.ErrOverflow)
		})
	}
}

func Test_calculateFloorDataGas_MatchesGethImplementation(t *testing.T) {
	tests := map[string][]byte{
		"empty data":                              {},
		"data with only zero bytes":               make([]byte, 10),
		"data with only non-zero bytes":           bytes.Repeat([]byte{0xFF}, 10),
		"data with mixed zero and non-zero bytes": append(bytes.Repeat([]byte{0}, 5), bytes.Repeat([]byte{0xFF}, 5)...),
	}

	for i := range 5 {
		length := rand.IntN(500)
		randomData := make([]byte, length)
		for i := range randomData {
			randomData[i] = byte(rand.IntN(3))
		}
		tests[fmt.Sprintf("random data %d", i)] = randomData
	}

	for name, data := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			want, err := core.FloorDataGas(data)
			require.NoError(err)

			zeros := bytes.Count(data, []byte{0})
			nonZeros := len(data) - zeros
			got, err := calculateFloorDataGas(uint64(zeros), uint64(nonZeros))
			require.NoError(err)
			require.Equal(want, got)
		})
	}
}

func Test_calculateFloorDataGas_DetectsOverflows(t *testing.T) {
	tests := map[string]struct {
		numZeroBytes    uint64
		numNonZeroBytes uint64
	}{
		"big overflow from zero bytes": {
			numZeroBytes:    math.MaxUint64,
			numNonZeroBytes: 0,
		},
		"close overflow from zero bytes": {
			numZeroBytes: (math.MaxUint64-params.TxGas)/params.TxCostFloorPerToken + 1,
		},
		"big overflow from non-zero bytes": {
			numZeroBytes:    0,
			numNonZeroBytes: math.MaxUint64,
		},
		"close overflow from non-zero bytes": {
			numZeroBytes:    0,
			numNonZeroBytes: (math.MaxUint64-params.TxGas)/params.TxCostFloorPerToken/params.TxTokenPerNonZeroByte + 1,
		},
		"overflow from combined zero and non-zero bytes": {
			numZeroBytes:    math.MaxUint64 / 2,
			numNonZeroBytes: math.MaxUint64 / 2,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			_, err := calculateFloorDataGas(tc.numZeroBytes, tc.numNonZeroBytes)
			require.ErrorIs(err, checked.ErrOverflow)
		})
	}
}

func Test_calculateTxGasSum_sumsGasLimits(t *testing.T) {
	tests := map[string][]uint64{
		"no transactions":       {},
		"one transaction":       {21000},
		"multiple transactions": {21000, 50000, 100000},
	}

	for name, gasLimits := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			var transactions []*types.Transaction
			var expectedSum uint64
			for _, gasLimit := range gasLimits {
				tx := types.NewTx(&types.LegacyTx{
					Gas: gasLimit,
				})
				transactions = append(transactions, tx)
				expectedSum += gasLimit
			}

			sum, err := calculateTxGasSum(transactions)
			require.NoError(err)
			require.Equal(expectedSum, sum)
		})
	}
}

func Test_calculateTxGasSum_detectsOverflows(t *testing.T) {
	tests := map[string][]uint64{
		"overflow from two transactions": {
			math.MaxUint64 - 1000, 2000,
		},
		"overflow from multiple transactions": {
			math.MaxUint64 - 1000, 500, 300, 200, 1,
		},
	}

	for name, gasLimits := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			var transactions []*types.Transaction
			for _, gasLimit := range gasLimits {
				tx := types.NewTx(&types.LegacyTx{
					Gas: gasLimit,
				})
				transactions = append(transactions, tx)
			}

			_, err := calculateTxGasSum(transactions)
			require.ErrorIs(err, checked.ErrOverflow)
		})
	}
}

func Test_calculateTxGasSum_ignoresNilTransactions(t *testing.T) {
	tests := map[string][]*types.Transaction{
		"all nil transactions": {
			nil, nil, nil,
		},
		"some nil transactions": {
			types.NewTx(&types.LegacyTx{Gas: 21000}), nil,
			types.NewTx(&types.LegacyTx{Gas: 50000}), nil,
		},
	}

	for name, transactions := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			expectedSum := uint64(0)
			for _, tx := range transactions {
				if tx != nil {
					expectedSum += tx.Gas()
				}
			}

			sum, err := calculateTxGasSum(transactions)
			require.NoError(err)
			require.Equal(expectedSum, sum)
		})
	}
}
