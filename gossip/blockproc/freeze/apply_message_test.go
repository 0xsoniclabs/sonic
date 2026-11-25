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
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/gossip/blockproc/freeze/registry"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestApplyMessage_RejectsFrozenSenders(t *testing.T) {
	var (
		frozenSender   = common.Address{0x1}
		unfrozenSender = common.Address{0x2}
		zeroAddress    = common.Address{}
	)
	tests := map[string]struct {
		from    common.Address
		enabled bool
		err     error
	}{
		"freeze enabled frozen sender returns error": {frozenSender, true, ErrFrozenAccount},
		"freeze enabled unfrozen sender runs":        {unfrozenSender, true, nil},
		"freeze enabled frozen zero address runs":    {zeroAddress, true, nil},
		"freeze disabled frozen sender runs":         {frozenSender, false, nil},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			statedb, err := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
			require.NoError(t, err)
			for _, account := range []common.Address{frozenSender, zeroAddress} {
				statedb.SetState(registry.GetAddress(), registry.GetFrozenSlot(account), common.Hash{31: 1})
			}
			evm := vm.NewEVM(vm.BlockContext{BlockNumber: big.NewInt(0), BaseFee: big.NewInt(0)}, statedb, params.MergedTestChainConfig, vm.Config{})
			zero := new(uint256.Int)
			_, err = ApplyMessage(evm, &core.Message{
				From: test.from, To: &common.Address{0x3}, Value: zero, GasLimit: 21_000,
				GasPrice: zero, GasFeeCap: zero, GasTipCap: zero,
			}, core.NewGasPool(21_000), NewRegistryChecker(test.enabled, statedb))
			require.ErrorIs(t, err, test.err)
			nonce := uint64(0)
			if test.err == nil {
				nonce = 1
			}
			require.Equal(t, nonce, statedb.GetNonce(test.from))
		})
	}
}
