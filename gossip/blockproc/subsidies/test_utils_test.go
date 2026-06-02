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

package subsidies

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestGetMaxPostTxSizeForTests_ReturnsMaximumOverheadSize(t *testing.T) {
	max := uint64(0)
	for _, mode := range allModes() {
		s := Sponsorship{mode: mode}
		overhead := s.Overhead().Size
		if overhead > max {
			max = overhead
		}
	}

	require.Equal(t, max, GetMaxPostTxSizeForTests())
}

func TestPostTxBuilder_ForFundBacked_ProducesValidTransaction(t *testing.T) {
	tx := NewPostTxBuilder().ForFundBacked().BuildForTesting()
	require.NotNil(t, tx)
	require.True(t, IsFeeChargeTransaction(tx))

	amount, err := ParseFeeChargeAmount(tx)
	require.NoError(t, err)
	require.True(t, amount.Sign() > 0)
}

func TestPostTxBuilder_ForNetworkSponsoredWithTracking_ProducesValidTransaction(t *testing.T) {
	tx := NewPostTxBuilder().ForNetworkSponsoredWithTracking().BuildForTesting()
	require.NotNil(t, tx)
	require.True(t, IsTrackTransaction(tx))

	amount, err := ParseTrackAmount(tx)
	require.NoError(t, err)
	require.True(t, amount.Sign() > 0)
}

func TestPostTxBuilder_WithCustomFundId_ProducesTransactionWithCorrectFundId(t *testing.T) {
	fundId := Identifier(common.HexToHash("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"))
	tx := NewPostTxBuilder().ForFundBacked().WithId(fundId).BuildForTesting()
	require.NotNil(t, tx)
	require.True(t, IsFeeChargeTransaction(tx))

	got := Identifier(tx.Data()[4:36])
	require.Equal(t, got, fundId)
}

func TestPostTxBuilder_WithCustomTrackingId_ProducesTransactionWithCorrectTrackingId(t *testing.T) {
	trackingId := Identifier(common.HexToHash("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"))
	tx := NewPostTxBuilder().ForNetworkSponsoredWithTracking().WithId(trackingId).BuildForTesting()
	require.NotNil(t, tx)
	require.True(t, IsTrackTransaction(tx))

	got := Identifier(tx.Data()[4:36])
	require.Equal(t, got, trackingId)
}

func TestPostTxBuilder_WithCustomUsedGasAndPrice_ProducesTxWithCorrectAmount(t *testing.T) {
	overheads := []uint64{0, 50_000, 100_000}
	usedGases := []uint64{0, 100_000, 150_000}
	gasPrices := []uint64{0, 1, 10, 100}

	for _, overhead := range overheads {
		for _, usedGas := range usedGases {
			for _, gasPrice := range gasPrices {
				tx := NewPostTxBuilder().
					ForFundBacked().
					WithOverhead(overhead).
					WithUsedGas(usedGas).
					WithGasPrice(big.NewInt(int64(gasPrice))).
					BuildForTesting()

				require.NotNil(t, tx)
				require.True(t, IsFeeChargeTransaction(tx))

				amount, err := ParseFeeChargeAmount(tx)
				require.NoError(t, err)

				expectedAmount := uint256.NewInt(uint64((overhead + usedGas) * gasPrice))
				require.Equal(t, 0, amount.Cmp(expectedAmount), "expected amount to be %s but got %s", expectedAmount.String(), amount.String())
			}
		}
	}
}

func TestPostTxBuilder_WithGasLimit_ProducesTxWithGivenGasLimit(t *testing.T) {
	gasLimit := uint64(300_000)
	tx := NewPostTxBuilder().ForFundBacked().WithGasLimit(gasLimit).BuildForTesting()
	require.NotNil(t, tx)
	require.True(t, IsFeeChargeTransaction(tx))
	require.Equal(t, gasLimit, tx.Gas())
}

func TestPostTxBuilder_WithNonce_ProducesTxWithGivenNonce(t *testing.T) {
	nonce := uint64(42)
	tx := NewPostTxBuilder().ForFundBacked().WithNonce(nonce).BuildForTesting()
	require.NotNil(t, tx)
	require.True(t, IsFeeChargeTransaction(tx))
	require.Equal(t, nonce, tx.Nonce())
}

func TestPostTxBuilder_OverflowFees_Panics(t *testing.T) {
	require.PanicsWithValue(t,
		"failed to create post-sponsorship transaction for testing: fee calculation overflow",
		func() {
			NewPostTxBuilder().
				ForFundBacked().
				WithOverhead(1).
				WithUsedGas(1).
				WithGasPrice(new(big.Int).Lsh(big.NewInt(1), 256)).
				BuildForTesting()
		},
	)
}
