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

package exec_test

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/oracle"
	"github.com/0xsoniclabs/sonic/rpc/tests"
)

func poolNonce(common.Address) uint64 {
	return 42
}

func TestSetDefaults_FillsWhatTheClientLeftOut(t *testing.T) {
	estimation := newEstimationChain(t)
	arguments := exec.CallArguments{From: addressOf(richAddress), Recipient: addressOf(counterAddress)}
	require.NoError(t, estimation.environment.SetDefaults(estimation.head, &arguments, poolNonce))

	tip, err := estimation.environment.Oracle().SuggestTip(context.Background(), oracle.DefaultCertainty)
	require.NoError(t, err)
	require.Equal(t, tip, arguments.MaxPriorityFeePerGas.ToInt(), "the oracle's tip")
	feeCap := new(big.Int).Add(tip, exec.SuggestedGasPrice(estimation.head))
	require.Equal(t, feeCap, arguments.MaxFeePerGas.ToInt(), "the tip above the suggested price")
	require.Nil(t, arguments.GasPrice)
	require.Zero(t, arguments.Value.ToInt().Sign())
	require.Equal(t, hexutil.Uint64(42), *arguments.Nonce, "the pool's nonce")
	require.Equal(t, big.NewInt(tests.FakeChainID), arguments.ChainID.ToInt())
	estimated, err := estimation.estimate(exec.CallArguments{From: addressOf(richAddress), Recipient: addressOf(counterAddress), MaxFeePerGas: arguments.MaxFeePerGas, MaxPriorityFeePerGas: arguments.MaxPriorityFeePerGas, Value: arguments.Value}, atBlock(19))
	require.NoError(t, err)
	require.Equal(t, estimated, *arguments.Gas, "the estimated gas")

	transaction, err := arguments.ToTransaction()
	require.NoError(t, err)
	require.Equal(t, uint8(types.DynamicFeeTxType), transaction.Type())
}

func TestSetDefaults_KeepsWhatTheClientSet(t *testing.T) {
	estimation := newEstimationChain(t)
	price := (*hexutil.Big)(big.NewInt(5_000_000_000))
	gas, nonce, value := hexutil.Uint64(30_000), hexutil.Uint64(3), (*hexutil.Big)(big.NewInt(1))
	arguments := exec.CallArguments{Recipient: addressOf(counterAddress), GasPrice: price, Gas: &gas, Nonce: &nonce, Value: value}
	require.NoError(t, estimation.environment.SetDefaults(estimation.head, &arguments, poolNonce))
	require.Equal(t, price, arguments.GasPrice)
	require.Nil(t, arguments.MaxFeePerGas, "a legacy price stays legacy")
	require.Equal(t, gas, *arguments.Gas)
	require.Equal(t, nonce, *arguments.Nonce)
	require.Equal(t, value, arguments.Value)
}

// TestSetDefaults_RefusesWhatCannotBeATransaction holds V5 and F4: each
// check with the reference's wording.
func TestSetDefaults_RefusesWhatCannotBeATransaction(t *testing.T) {
	estimation := newEstimationChain(t)
	one, two := (*hexutil.Big)(big.NewInt(1)), (*hexutil.Big)(big.NewInt(2))
	code, other := hexutil.Bytes{0x01}, hexutil.Bytes{0x02}
	hashes := []common.Hash{{1}}
	authorizations := []types.SetCodeAuthorization{}
	wrongChain := (*hexutil.Big)(big.NewInt(1))
	gas := hexutil.Uint64(30_000)
	cases := map[string]struct {
		arguments exec.CallArguments
		message   string
	}{
		"both price styles":      {exec.CallArguments{GasPrice: one, MaxFeePerGas: two}, "both gasPrice and (maxFeePerGas or maxPriorityFeePerGas) specified"},
		"legacy set-code":        {exec.CallArguments{GasPrice: one, AuthorizationList: authorizations}, "both gasPrice and authorizationList specified"},
		"fee cap below tip":      {exec.CallArguments{MaxFeePerGas: one, MaxPriorityFeePerGas: two}, "maxFeePerGas (0x1) < maxPriorityFeePerGas (0x2)"},
		"data and input":         {exec.CallArguments{Recipient: addressOf(counterAddress), Data: &code, Input: &other}, `both "data" and "input" are set and not equal. Please use "input" to pass transaction call data`},
		"blob creation":          {exec.CallArguments{BlobHashes: hashes}, `missing "to" in blob transaction`},
		"empty creation":         {exec.CallArguments{}, `contract creation without any data provided`},
		"set-code creation":      {exec.CallArguments{Input: &code, AuthorizationList: authorizations}, `authorizationList provided for contract creation, but "to" field is missing`},
		"another chain":          {exec.CallArguments{Recipient: addressOf(counterAddress), Gas: &gas, ChainID: wrongChain}, "chainId does not match node's (have=1, want=4003)"},
		"a failing estimation":   {exec.CallArguments{From: addressOf(richAddress), Recipient: addressOf(reverterAddress)}, "execution reverted"},
		"a sender without funds": {exec.CallArguments{Recipient: addressOf(counterAddress)}, "insufficient funds for transfer"},
	}
	for name, test := range cases {
		arguments := test.arguments
		require.EqualError(t, estimation.environment.SetDefaults(estimation.head, &arguments, poolNonce), test.message, name)
	}
}

// TestSetDefaults_EstimatesWithoutInput reproduces the reference's bug 2 of
// section 6 (F8): the estimation takes data and not input.
func TestSetDefaults_EstimatesWithoutInput(t *testing.T) {
	estimation := newEstimationChain(t)
	calldata := hexutil.Bytes{0x01}
	withData := exec.CallArguments{From: addressOf(richAddress), Recipient: addressOf(needsDataAddress), Data: &calldata}
	require.NoError(t, estimation.environment.SetDefaults(estimation.head, &withData, poolNonce))
	withInput := exec.CallArguments{From: addressOf(richAddress), Recipient: addressOf(needsDataAddress), Input: &calldata}
	require.EqualError(t, estimation.environment.SetDefaults(estimation.head, &withInput, poolNonce), "execution reverted")
}
