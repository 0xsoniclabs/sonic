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

package drivercall

import (
	"math"
	"math/big"
	"testing"
	"time"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/opera/contracts/driver"
	"github.com/0xsoniclabs/sonic/utils/signers/internaltx"
)

func TestParseSealEpochArgs_Success(t *testing.T) {
	tests := map[string][]ValidatorEpochMetric{
		"empty list": {},
		"single validator": {{
			Missed: opera.BlocksMissed{
				BlocksNum: 42,
				Period:    inter.FromUnix(100),
			},
			Uptime:          inter.FromUnix(900),
			OriginatedTxFee: big.NewInt(1_000_000),
		}},
		"multiple validators": {
			{
				Missed: opera.BlocksMissed{
					BlocksNum: 0,
					Period:    inter.FromUnix(0),
				},
				Uptime:          inter.FromUnix(0),
				OriginatedTxFee: big.NewInt(0),
			},
			{
				Missed: opera.BlocksMissed{
					BlocksNum: 7,
					Period:    inter.FromUnix(300),
				},
				Uptime:          inter.FromUnix(700),
				OriginatedTxFee: big.NewInt(500),
			},
			{
				Missed: opera.BlocksMissed{
					BlocksNum: 1000,
					Period:    inter.FromUnix(86400),
				},
				Uptime:          inter.FromUnix(86400),
				OriginatedTxFee: new(big.Int).Lsh(big.NewInt(1), 200),
			},
		},
	}

	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			tx := newInternalDriverTx(SealEpoch(input))
			got, err := ParseSealEpochArgs(tx)
			require.NoError(t, err)

			require.Len(t, got, len(input))
			for i := range input {
				require.Equal(t, input[i].Missed, got[i].Missed)
				require.Equal(t, input[i].Uptime, got[i].Uptime)

				want := input[i].OriginatedTxFee
				got := got[i].OriginatedTxFee
				require.True(t, want.Cmp(got) == 0, "want %v, got %v", want, got)
			}
		})
	}
}

func TestParseSealEpochArgs_Errors(t *testing.T) {
	validData := SealEpoch([]ValidatorEpochMetric{})

	t.Run("nil transaction", func(t *testing.T) {
		_, err := ParseSealEpochArgs(nil)
		require.ErrorContains(t, err, "transaction is nil or not internal")
	})

	t.Run("non-internal transaction", func(t *testing.T) {
		tx := types.NewTx(&types.LegacyTx{
			V: big.NewInt(1),
		})
		require.False(t, internaltx.IsInternal(tx))
		_, err := ParseSealEpochArgs(tx)
		require.ErrorContains(t, err, "transaction is nil or not internal")
	})

	t.Run("transaction with nil To", func(t *testing.T) {
		tx := types.NewTx(&types.LegacyTx{
			To:   nil,
			Data: validData,
		})
		_, err := ParseSealEpochArgs(tx)
		require.ErrorContains(t, err, "transaction does not target the node driver contract")
	})

	t.Run("transaction targeting wrong contract address", func(t *testing.T) {
		tx := types.NewTx(&types.LegacyTx{
			To:   &common.Address{1, 2, 3},
			Data: validData,
		})
		_, err := ParseSealEpochArgs(tx)
		require.ErrorContains(t, err, "transaction does not target the node driver contract")
	})

	t.Run("data too short to hold a selector", func(t *testing.T) {
		for _, data := range [][]byte{nil, {}, {1}, {1, 2}, {1, 2, 3}} {
			tx := newInternalDriverTx(data)
			_, err := ParseSealEpochArgs(tx)
			require.ErrorContains(t, err, "data too short to contain a function selector")
		}
	})

	t.Run("unknown four-byte selector", func(t *testing.T) {
		// 4-byte selector that does not match any method in the ABI.
		data := append([]byte{1, 2, 3, 4}, make([]byte, 128)...)
		tx := newInternalDriverTx(data)
		_, err := ParseSealEpochArgs(tx)
		require.ErrorContains(t, err, "unknown method")
	})

	t.Run("valid selector but wrong method", func(t *testing.T) {
		// sealEpochValidators is a different ABI method on the same contract.
		data := SealEpochValidators([]idx.ValidatorID{1, 2, 3})
		tx := newInternalDriverTx(data)
		_, err := ParseSealEpochArgs(tx)
		require.ErrorContains(t, err, "expected sealEpoch")
	})

	t.Run("truncated arguments", func(t *testing.T) {
		// Keep the sealEpoch 4-byte selector but truncate the ABI payload.
		full := SealEpoch([]ValidatorEpochMetric{{
			Missed:          opera.BlocksMissed{BlocksNum: 1, Period: inter.FromUnix(1)},
			Uptime:          inter.FromUnix(1),
			OriginatedTxFee: big.NewInt(1),
		}})
		tx := newInternalDriverTx(full[:6]) // selector (4 bytes) + 2 garbage bytes
		_, err := ParseSealEpochArgs(tx)
		require.ErrorContains(t, err, "failed to unpack sealEpoch arguments")
	})

	t.Run("array length mismatch", func(t *testing.T) {
		// Pack a sealEpoch call where offlineBlocks has a different length than
		// the other three arrays to trigger the length-mismatch check.
		twoElems := []*big.Int{big.NewInt(1), big.NewInt(2)}
		threeElems := []*big.Int{big.NewInt(1), big.NewInt(2), big.NewInt(3)}
		data, err := sAbi.Pack("sealEpoch", twoElems, threeElems, twoElems, twoElems)
		require.NoError(t, err)
		tx := newInternalDriverTx(data)
		_, err = ParseSealEpochArgs(tx)
		require.ErrorContains(t, err, "argument array lengths do not match")
	})
}

// newInternalDriverTx creates an unsigned (internal) transaction targeting
// the node driver contract with the given call data.
func newInternalDriverTx(data []byte) *types.Transaction {
	to := driver.ContractAddress
	return types.NewTx(&types.LegacyTx{
		To:   &to,
		Data: data,
	})
}

func TestConvertSealEpochArgs_Success(t *testing.T) {
	tests := map[string][]ValidatorEpochMetric{
		"empty list": {},
		"single validator": {{
			Missed: opera.BlocksMissed{
				BlocksNum: 42,
				Period:    inter.FromUnix(100),
			},
			Uptime:          inter.FromUnix(900),
			OriginatedTxFee: big.NewInt(1_000_000),
		}},
		"multiple validators": {
			{
				Missed: opera.BlocksMissed{
					BlocksNum: 0,
					Period:    inter.FromUnix(0),
				},
				Uptime:          inter.FromUnix(0),
				OriginatedTxFee: big.NewInt(0),
			},
			{
				Missed: opera.BlocksMissed{
					BlocksNum: 7,
					Period:    inter.FromUnix(300),
				},
				Uptime:          inter.FromUnix(700),
				OriginatedTxFee: big.NewInt(500),
			},
			{
				Missed: opera.BlocksMissed{
					BlocksNum: 1000,
					Period:    inter.FromUnix(86400),
				},
				Uptime:          inter.FromUnix(86400),
				OriginatedTxFee: new(big.Int).Lsh(big.NewInt(1), 200), // large fee, no range limit
			},
		},
	}

	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := convertSealEpochArgs(metricsToArgs(input))
			require.NoError(t, err)
			require.Len(t, got, len(input))
			for i := range input {
				require.Equal(t, input[i].Missed, got[i].Missed)
				require.Equal(t, input[i].Uptime, got[i].Uptime)
				require.Zero(t, input[i].OriginatedTxFee.Cmp(got[i].OriginatedTxFee))
			}
		})
	}
}

func TestConvertSealEpochArgs_Errors(t *testing.T) {
	valid := []*big.Int{big.NewInt(1)}

	t.Run("wrong number of arguments", func(t *testing.T) {
		_, err := convertSealEpochArgs([]any{valid, valid, valid}) // only 3 instead of 4
		require.ErrorContains(t, err, "expected 4 arguments for sealEpoch, got 3")
	})

	t.Run("wrong type for offlineTimes", func(t *testing.T) {
		_, err := convertSealEpochArgs([]any{[]int{1}, valid, valid, valid})
		require.ErrorContains(t, err, "unexpected type for offlineTimes argument")
	})

	t.Run("wrong type for offlineBlocks", func(t *testing.T) {
		_, err := convertSealEpochArgs([]any{valid, []int{1}, valid, valid})
		require.ErrorContains(t, err, "unexpected type for offlineBlocks argument")
	})

	t.Run("wrong type for uptimes", func(t *testing.T) {
		_, err := convertSealEpochArgs([]any{valid, valid, []int{1}, valid})
		require.ErrorContains(t, err, "unexpected type for uptimes argument")
	})

	t.Run("wrong type for fees", func(t *testing.T) {
		_, err := convertSealEpochArgs([]any{valid, valid, valid, []int{1}})
		require.ErrorContains(t, err, "unexpected type for fees argument")
	})

	t.Run("array length mismatch in offlineBlocks", func(t *testing.T) {
		two := []*big.Int{big.NewInt(1), big.NewInt(2)}
		_, err := convertSealEpochArgs([]any{valid, two, valid, valid})
		require.ErrorContains(t, err, "argument array lengths do not match")
	})

	t.Run("array length mismatch in uptimes", func(t *testing.T) {
		two := []*big.Int{big.NewInt(1), big.NewInt(2)}
		_, err := convertSealEpochArgs([]any{valid, valid, two, valid})
		require.ErrorContains(t, err, "argument array lengths do not match")
	})

	t.Run("array length mismatch in fees", func(t *testing.T) {
		two := []*big.Int{big.NewInt(1), big.NewInt(2)}
		_, err := convertSealEpochArgs([]any{valid, valid, valid, two})
		require.ErrorContains(t, err, "argument array lengths do not match")
	})

	t.Run("offlineBlocks out of uint64 range", func(t *testing.T) {
		overflow := new(big.Int).Lsh(big.NewInt(1), 64) // > math.MaxUint64
		_, err := convertSealEpochArgs([]any{valid, []*big.Int{overflow}, valid, valid})
		require.ErrorContains(t, err, "offlineBlocks contains value that doesn't fit in uint64")
	})

	t.Run("offlineTimes out of int64 range", func(t *testing.T) {
		overflow := new(big.Int).Lsh(big.NewInt(1), 63) // > math.MaxInt64
		_, err := convertSealEpochArgs([]any{[]*big.Int{overflow}, valid, valid, valid})
		require.ErrorContains(t, err, "offlineTimes contains value that doesn't fit in int64")
	})

	t.Run("offlineTimes out of max-seconds range", func(t *testing.T) {
		overflow := big.NewInt(math.MaxInt64/int64(time.Second) + 1)
		_, err := convertSealEpochArgs([]any{[]*big.Int{overflow}, valid, valid, valid})
		require.ErrorContains(t, err, "offlineTimes contains value that is too large to be a valid offline time in seconds")
	})

	t.Run("uptimes out of int64 range", func(t *testing.T) {
		overflow := new(big.Int).Lsh(big.NewInt(1), 63) // > math.MaxInt64
		_, err := convertSealEpochArgs([]any{valid, valid, []*big.Int{overflow}, valid})
		require.ErrorContains(t, err, "uptimes contains value that doesn't fit in int64")
	})

	t.Run("uptimes out of max-seconds range", func(t *testing.T) {
		overflow := big.NewInt(math.MaxInt64/int64(time.Second) + 1)
		_, err := convertSealEpochArgs([]any{valid, valid, []*big.Int{overflow}, valid})
		require.ErrorContains(t, err, "uptimes contains value that is too large to be a valid uptime in seconds")
	})
}

// metricsToArgs converts a ValidatorEpochMetric slice into the []any format
// expected by convertSealEpochArgs, using the same encoding as SealEpoch.
func metricsToArgs(metrics []ValidatorEpochMetric) []any {
	offlineTimes := make([]*big.Int, len(metrics))
	offlineBlocks := make([]*big.Int, len(metrics))
	uptimes := make([]*big.Int, len(metrics))
	fees := make([]*big.Int, len(metrics))
	for i, m := range metrics {
		offlineTimes[i] = new(big.Int).SetUint64(uint64(m.Missed.Period.Unix()))
		offlineBlocks[i] = new(big.Int).SetUint64(uint64(m.Missed.BlocksNum))
		uptimes[i] = new(big.Int).SetUint64(uint64(m.Uptime.Unix()))
		fees[i] = m.OriginatedTxFee
	}
	return []any{offlineTimes, offlineBlocks, uptimes, fees}
}
