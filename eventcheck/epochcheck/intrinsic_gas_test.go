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

package epochcheck

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestCheckTxs_RejectsTransactionsBelowTheIntrinsicGasOfTheEpoch(t *testing.T) {
	to := &common.Address{1}
	tests := map[string]struct {
		upgrades opera.Upgrades
		to       *common.Address
		value    *big.Int
		minGas   uint64
	}{
		"Brio call":             {upgrades: opera.GetBrioUpgrades(), to: to, minGas: params.TxGas},
		"Brio creation":         {upgrades: opera.GetBrioUpgrades(), minGas: params.TxGasContractCreation},
		"Canto call":            {upgrades: opera.GetCantoUpgrades(), to: to, minGas: params.TxBaseCost2780},
		"Canto call with value": {upgrades: opera.GetCantoUpgrades(), to: to, value: big.NewInt(1), minGas: params.TxBaseCost2780},
		"Canto creation":        {upgrades: opera.GetCantoUpgrades(), minGas: params.TxBaseCost2780 + params.CreateAccessAmsterdam},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			rules := opera.Rules{Upgrades: test.upgrades}
			withGas := func(gas uint64) types.Transactions {
				return types.Transactions{types.NewTx(&types.LegacyTx{To: test.to, Value: test.value, Gas: gas})}
			}
			require.NoError(t, CheckTxs(withGas(test.minGas), rules))
			require.ErrorIs(t, CheckTxs(withGas(test.minGas-1), rules), ErrIntrinsicGas)
		})
	}
}

func TestMinIntrinsicGas_NeverExceedsTheIntrinsicGasOfBlockProcessing(t *testing.T) {
	sender := common.Address{1}
	other := common.Address{2}
	accessList := types.AccessList{{Address: other, StorageKeys: []common.Hash{{1}}}}
	tests := map[string]struct {
		tx types.TxData
		// exactFromCanto is set for the transactions whose price does not
		// depend on the sender under EIP-2780.
		exactFromCanto bool
	}{
		"transfer to self": {tx: &types.LegacyTx{To: &sender, Value: big.NewInt(1)}, exactFromCanto: true},
		"call":             {tx: &types.LegacyTx{To: &other}},
		"transfer":         {tx: &types.LegacyTx{To: &other, Value: big.NewInt(1)}},
		"call with data":   {tx: &types.LegacyTx{To: &other, Data: []byte{0, 1, 2}}},
		"creation":         {tx: &types.LegacyTx{Data: make([]byte, 100)}, exactFromCanto: true},
		"access list":      {tx: &types.AccessListTx{To: &other, AccessList: accessList}},
		"authorizations":   {tx: &types.SetCodeTx{To: other, AuthList: []types.SetCodeAuthorization{{}}}},
	}

	for upgradesName, upgrades := range opera.GetAllHardForksInOrder() {
		chainConfig := opera.CreateTransientEvmChainConfig(0, []opera.UpgradeHeight{{Upgrades: upgrades}}, 1)
		chainRules := chainConfig.Rules(big.NewInt(1), true, 0)
		for name, test := range tests {
			t.Run(upgradesName+"/"+name, func(t *testing.T) {
				tx := types.NewTx(test.tx)
				want, err := core.IntrinsicGas(tx.Data(), tx.AccessList(), tx.SetCodeAuthorizations(),
					sender, tx.To(), uint256.MustFromBig(tx.Value()), chainRules)
				require.NoError(t, err)

				got, err := minIntrinsicGas(tx, opera.Rules{Upgrades: upgrades})
				require.NoError(t, err)
				if upgrades.Canto && test.exactFromCanto {
					require.Equal(t, want, got)
				} else {
					require.LessOrEqual(t, got, want)
				}
			})
		}
	}
}

func TestChecker_IntrinsicGas_LegacyCalculationDoesNotAccountForInitDataOrAuthList(t *testing.T) {

	tests := map[string]*types.Transaction{
		"legacyTx": types.NewTx(&types.LegacyTx{
			To:  nil,
			Gas: 21_000,
			// some data that takes
			Data: make([]byte, params.MaxInitCodeSize),
		}),
		"setCodeTx": types.NewTx(&types.SetCodeTx{
			To:       common.Address{},
			Gas:      21_000,
			AuthList: []types.SetCodeAuthorization{{}}}),
	}

	for name, tx := range tests {
		t.Run(name, func(t *testing.T) {
			costLegacy, err := intrinsicGasLegacy(tx.Data(), tx.AccessList(), tx.To() == nil)
			require.NoError(t, err)

			// in sonic, Homestead, Istanbul and Shanghai are always active
			costNew, err := core.IntrinsicGas(tx.Data(), tx.AccessList(),
				tx.SetCodeAuthorizations(), common.Address{}, tx.To(), uint256.NewInt(0),
				params.Rules{IsHomestead: true, IsIstanbul: true, IsShanghai: true})
			require.NoError(t, err)
			require.Greater(t, costNew, costLegacy)
		})
	}
}

func TestChecker_IntrinsicGas_LegacyIsCheaperOrSameForAllRevisionCombinations(t *testing.T) {

	trueFalse := []bool{true, false}

	for _, homestead := range trueFalse {
		for _, istanbul := range trueFalse {
			for _, shanghai := range trueFalse {
				t.Run(makeTestName(homestead, istanbul, shanghai), func(t *testing.T) {

					costLegacy, err := intrinsicGasLegacy([]byte("test"), nil, false)
					require.NoError(t, err)

					costNew, err := core.IntrinsicGas([]byte("test"), nil, nil,
						common.Address{}, &common.Address{}, uint256.NewInt(0),
						params.Rules{IsHomestead: homestead, IsIstanbul: istanbul, IsShanghai: shanghai})
					require.NoError(t, err)

					require.GreaterOrEqual(t, costNew, costLegacy)

				})
			}
		}
	}
}

func makeTestName(homestead, istanbul, shanghai bool) string {
	name := ""
	withWithout := func(fork bool) string {
		if fork {
			return "With"
		}
		return "Without"
	}
	name += withWithout(homestead) + "Homestead"
	name += withWithout(istanbul) + "Istanbul"
	name += withWithout(shanghai) + "Shanghai"
	return name
}
