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

package evmcore

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestNewEVMBlockContext_DifficultyIsOne(t *testing.T) {
	header := &EvmHeader{
		Number: big.NewInt(12),
	}
	context := NewEVMBlockContext(header, nil, nil)
	require.Equal(t, big.NewInt(1), context.Difficulty)
}

func TestNewEVMBlockContextWithDifficulty_UsesProvidedDifficulty(t *testing.T) {
	header := &EvmHeader{
		Number: big.NewInt(12),
	}
	for i := range int64(10) {
		difficulty := big.NewInt(i)
		context := NewEVMBlockContextWithDifficulty(header, nil, nil, difficulty)
		require.Equal(t, difficulty, context.Difficulty)
	}
}

func TestNewEVMTxContext_ReturnsErrorForNilMessage(t *testing.T) {
	_, err := NewEVMTxContext(nil)
	require.ErrorContains(t, err, "message cannot be nil")
}

func TestNewEVMTxContext_ReturnsErrorForInvalidGasPrice(t *testing.T) {
	tests := map[string]*big.Int{
		"negative":  big.NewInt(-1),
		"too large": new(big.Int).Lsh(big.NewInt(1), 256),
	}
	for name, gasPrice := range tests {
		t.Run(name, func(t *testing.T) {
			msg := &core.Message{
				GasPrice: gasPrice,
			}
			_, err := NewEVMTxContext(msg)
			require.ErrorContains(t, err, "invalid gas price")
		})
	}
}

func TestNewEVMTxContext_UsesEthereumCoreConversion(t *testing.T) {
	msg := &core.Message{
		From:     common.Address{1, 2, 3},
		GasPrice: big.NewInt(100),
		BlobHashes: []common.Hash{
			{4, 5, 6},
			{7, 8},
		},
	}
	txContext, err := NewEVMTxContext(msg)
	require.NoError(t, err)
	expected := core.NewEVMTxContext(msg)
	require.Equal(t, expected, txContext)
}

func TestNewEVMTxContext_OnlyCoversKnownFields(t *testing.T) {
	// This test should detect if the implementation in go-ethereum changes in a
	// way that requires us to update our wrapper.

	msg := &core.Message{
		From:      common.Address{1, 2, 3},
		To:        &common.Address{4, 5, 6},
		Value:     big.NewInt(1000),
		GasLimit:  21000,
		GasPrice:  big.NewInt(100),
		GasFeeCap: big.NewInt(200),
		GasTipCap: big.NewInt(50),
		Data:      []byte{0x1, 0x2},
		AccessList: types.AccessList{
			{
				Address:     common.Address{9, 10},
				StorageKeys: []common.Hash{{11}, {12}},
			},
		},
		BlobGasFeeCap: big.NewInt(300),
		BlobHashes: []common.Hash{
			{4, 5, 6},
			{7, 8},
		},
		SetCodeAuthorizations: []types.SetCodeAuthorization{
			{Address: common.Address{13, 14}},
		},
	}

	txContext, err := NewEVMTxContext(msg)
	require.NoError(t, err)

	expected := vm.TxContext{
		Origin:     msg.From,
		GasPrice:   uint256.MustFromBig(msg.GasPrice),
		BlobHashes: msg.BlobHashes,
	}
	require.Equal(t, expected, txContext)
}

func TestMustNewEVMTxContext_PanicsOnInvalidGasPrice(t *testing.T) {
	msg := &core.Message{
		GasPrice: big.NewInt(-1),
	}
	require.PanicsWithValue(
		t,
		"failed to create EVM transaction context: invalid gas price -1",
		func() { MustNewEVMTxContext(msg) },
	)
}

func TestNewEVMBlockContextWithDifficulty_UsesHeaderParametersIfDefinedOrSonicDefaults(t *testing.T) {

	// This test verifies that NewEVMBlockContextWithDifficulty can both be used to create
	// an execution context which can process both recorded and live blocks,
	// by correctly using header parameters when defined.
	// The following parameters are tested:
	// - BaseFee is nil on Sonic (at this stage), but it uses the base fee from the header if defined.
	// - BlobBaseFee defaults to 1, but it uses the value from the header if defined.
	// - Coinbase is taken from the header if the author parameter is nil; otherwise it uses the author.
	// - PrevRandao is used if defined and difficulty is set to 0 in that case; otherwise difficulty remains the header value.

	const defaultBlobBaseFee int64 = 1

	someBaseFee := big.NewInt(7)
	someBlobBaseFee := big.NewInt(123)
	someDifficulty := big.NewInt(321)

	tests := map[string]struct {
		header      *EvmHeader
		author      *common.Address
		difficulty  *big.Int
		expectation func(t testing.TB, ctx vm.BlockContext)
	}{
		"uses header.BaseFee when defined": {
			header: &EvmHeader{
				Number:  big.NewInt(1),
				BaseFee: someBaseFee,
			},
			expectation: func(t testing.TB, ctx vm.BlockContext) {
				require.Equal(t, someBaseFee, ctx.BaseFee)
				require.NotSame(t, someBaseFee, ctx.BaseFee, "BaseFee should be copied, not aliased")
			},
		},
		"uses nil BaseFee when header.BaseFee is not defined": {
			header: &EvmHeader{
				Number: big.NewInt(1),
			},
			expectation: func(t testing.TB, ctx vm.BlockContext) {
				require.Nil(t, ctx.BaseFee)
			},
		},
		"header.blockBaseFee overrides default": {
			header: &EvmHeader{
				Number:      big.NewInt(1),
				BlobBaseFee: someBlobBaseFee,
			},
			expectation: func(t testing.TB, ctx vm.BlockContext) {
				require.Equal(t, someBlobBaseFee, ctx.BlobBaseFee)
				require.NotSame(t, someBlobBaseFee, ctx.BlobBaseFee, "BlobBaseFee should be copied, not aliased")
			},
		},
		"uses default blobBaseFee when header.blobBaseFee is not defined": {
			header: &EvmHeader{
				Number: big.NewInt(1),
			},
			expectation: func(t testing.TB, ctx vm.BlockContext) {
				require.Equal(t, defaultBlobBaseFee, ctx.BlobBaseFee.Int64())

			},
		},
		"uses header coinbase when author is nil": {
			header: &EvmHeader{
				Number:   big.NewInt(1),
				Coinbase: common.Address{0xAA},
			},
			author: nil,
			expectation: func(t testing.TB, ctx vm.BlockContext) {
				require.Equal(t, common.Address{0xAA}, ctx.Coinbase)
			},
		},
		"author overrides header coinbase": {
			header: &EvmHeader{
				Number:   big.NewInt(1),
				Coinbase: common.Address{0xAA},
			},
			author: &common.Address{0xBB},
			expectation: func(t testing.TB, ctx vm.BlockContext) {
				require.Equal(t, common.Address{0xBB}, ctx.Coinbase)
			},
		},
		"uses header.prevRandao and 0 difficulty when prevRandao is set": {
			header: &EvmHeader{
				Number:     big.NewInt(1),
				PrevRandao: common.Hash{0x01},
			},
			difficulty: someDifficulty,
			expectation: func(t testing.TB, ctx vm.BlockContext) {
				require.NotNil(t, ctx.Random)
				require.Equal(t, common.Hash{0x01}, *ctx.Random)
				require.Equal(t, int64(0), ctx.Difficulty.Int64())
			},
		},
		"keeps provided difficulty and nil random when prevRandao is zero": {
			header: &EvmHeader{
				Number:     big.NewInt(1),
				PrevRandao: common.Hash{},
			},
			difficulty: someDifficulty,
			expectation: func(t testing.TB, ctx vm.BlockContext) {
				require.Nil(t, ctx.Random)
				require.Equal(t, someDifficulty, ctx.Difficulty)
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := NewEVMBlockContextWithDifficulty(test.header, nil, test.author, test.difficulty)
			test.expectation(t, ctx)
		})
	}
}
