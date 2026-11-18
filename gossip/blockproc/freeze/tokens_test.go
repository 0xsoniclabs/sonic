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

package freeze

import (
	"slices"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/freeze/registry"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// tokenCall encodes a call of the function with selector, passing accounts
// and an amount of zero.
func tokenCall(selector []byte, accounts ...common.Address) []byte {
	input := slices.Clone(selector)
	for _, account := range accounts {
		input = append(input, common.LeftPadBytes(account.Bytes(), 32)...)
	}
	return append(input, make([]byte, 32)...)
}

func TestIsFrozenTokenCall_DetectsErc20TransfersNamingFrozenAccounts(t *testing.T) {
	db := newFakeState(frozenAccount)
	db.slots[registry.GetOptedOutSlot(optedOutToken)] = common.Hash{31: 1}

	tests := map[string]struct {
		token common.Address
		input []byte
		want  bool
	}{
		"transfer to frozen":          {token, tokenCall(transferSelector, frozenAccount), true},
		"transferFrom from frozen":    {token, tokenCall(transferFromSelector, frozenAccount, otherAccount), true},
		"transferFrom to frozen":      {token, tokenCall(transferFromSelector, otherAccount, frozenAccount), true},
		"transferFrom between others": {token, tokenCall(transferFromSelector, otherAccount, otherAccount), false},
		"opted-out token":             {optedOutToken, tokenCall(transferSelector, frozenAccount), false},
		"other function":              {token, tokenCall([]byte{1, 2, 3, 4}, frozenAccount), false},
		"truncated arguments":         {token, tokenCall(transferSelector, frozenAccount)[:4+31], false},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, test.want, isFrozenTokenCall(db, test.token, test.input))
		})
	}
}
