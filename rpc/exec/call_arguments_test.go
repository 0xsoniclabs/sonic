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
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// filledArguments are arguments with every default a transaction needs.
func filledArguments() exec.CallArguments {
	to := common.Address{0x42}
	nonce, gas := hexutil.Uint64(3), hexutil.Uint64(21000)
	input := hexutil.Bytes{0x01}
	return exec.CallArguments{
		To: &to, Nonce: &nonce, Gas: &gas, Input: &input,
		ChainID: (*hexutil.Big)(big.NewInt(4003)), Value: (*hexutil.Big)(big.NewInt(5)),
	}
}

func TestToTransaction_FormsTheTypeItsFieldsName(t *testing.T) {
	price := (*hexutil.Big)(big.NewInt(7))
	cases := map[string]struct {
		change func(*exec.CallArguments)
		kind   uint8
	}{
		"legacy": {func(a *exec.CallArguments) { a.GasPrice = price }, types.LegacyTxType},
		"access list": {func(a *exec.CallArguments) {
			a.GasPrice, a.AccessList = price, &types.AccessList{{Address: common.Address{0x01}}}
		}, types.AccessListTxType},
		"dynamic fee": {func(a *exec.CallArguments) { a.MaxFeePerGas, a.MaxPriorityFeePerGas = price, price }, types.DynamicFeeTxType},
		"blob": {func(a *exec.CallArguments) {
			a.MaxFeePerGas, a.MaxPriorityFeePerGas, a.BlobFeeCap, a.BlobHashes = price, price, price, []common.Hash{{0x01}}
		}, types.BlobTxType},
		"set code": {func(a *exec.CallArguments) {
			a.MaxFeePerGas, a.MaxPriorityFeePerGas, a.AuthorizationList = price, price, []types.SetCodeAuthorization{{Nonce: 1}}
		}, types.SetCodeTxType},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			arguments := filledArguments()
			test.change(&arguments)
			transaction, err := arguments.ToTransaction()
			require.NoError(t, err)
			require.Equal(t, test.kind, transaction.Type())
			require.Equal(t, common.Address{0x42}, *transaction.To())
			require.Equal(t, uint64(3), transaction.Nonce())
			require.Equal(t, uint64(21000), transaction.Gas())
			require.Equal(t, []byte{0x01}, transaction.Data())
			require.Equal(t, int64(5), transaction.Value().Int64())
		})
	}
}

func TestToTransaction_RefusesWhatATypeCannotHold(t *testing.T) {
	negative := (*hexutil.Big)(big.NewInt(-1))
	price := (*hexutil.Big)(big.NewInt(7))
	cases := map[string]struct {
		change func(*exec.CallArguments)
		err    string
	}{
		"set code without recipient": {func(a *exec.CallArguments) {
			a.To, a.MaxFeePerGas, a.MaxPriorityFeePerGas = nil, price, price
			a.AuthorizationList = []types.SetCodeAuthorization{{}}
		}, "missing To in SetCodeTx"},
		"blob without recipient": {func(a *exec.CallArguments) {
			a.To, a.MaxFeePerGas, a.MaxPriorityFeePerGas, a.BlobFeeCap = nil, price, price, price
		}, "missing To in BlobTx"},
		"negative chain id": {func(a *exec.CallArguments) { a.ChainID, a.BlobFeeCap = negative, price },
			"invalid ChainID: unable to convert negative big.Int to uint256"},
		"negative fee cap": {func(a *exec.CallArguments) { a.MaxFeePerGas, a.BlobFeeCap = negative, price },
			"invalid MaxFeePerGas: unable to convert negative big.Int to uint256"},
		"negative tip cap": {func(a *exec.CallArguments) {
			a.MaxFeePerGas, a.MaxPriorityFeePerGas, a.BlobFeeCap = price, negative, price
		},
			"invalid MaxPriorityFeePerGas: unable to convert negative big.Int to uint256"},
		"negative blob fee cap": {func(a *exec.CallArguments) { a.BlobFeeCap = negative },
			"invalid BlobFeeCap: unable to convert negative big.Int to uint256"},
		"negative value": {func(a *exec.CallArguments) { a.Value, a.BlobFeeCap = negative, price },
			"invalid Value: unable to convert negative big.Int to uint256"},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			arguments := filledArguments()
			test.change(&arguments)
			_, err := arguments.ToTransaction()
			require.EqualError(t, err, test.err)
		})
	}
}
