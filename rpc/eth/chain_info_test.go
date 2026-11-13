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

package eth_test

import (
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/tests"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

func TestChainFacts_AnswerConstantsAndTheServedHead(t *testing.T) {
	chain := newTestChain(t)
	h := newHarness(t, chain.fake)
	cases := map[string]struct {
		method string
		want   string
	}{
		"chain id":     {"eth_chainId", `"0xfa3"`},
		"block number": {"eth_blockNumber", `"0x2"`},
		"etherbase":    {"eth_etherbase", `"0x0000000000000000000000000000000000000000"`},
		"coinbase":     {"eth_coinbase", `"0x0000000000000000000000000000000000000000"`},
		"hashrate":     {"eth_hashrate", `"0x0"`},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, test.want, h.Result(t, test.method))
		})
	}
}

func TestBlockNumber_IsTheServedHeadNotTheStoredOne(t *testing.T) {
	chain := newTestChain(t)
	h := newHarness(t, chain.fake)
	require.Equal(t, `"0x2"`, h.Result(t, "eth_blockNumber"))
	h.SetHead(t, 3)
	require.Equal(t, `"0x3"`, h.Result(t, "eth_blockNumber"))
}

func TestSyncing_AnswersFalseForARecentBlock(t *testing.T) {
	chain := newTestChain(t)
	h := newHarness(t, chain.fake)
	chain.fake.SetProgress(core.Progress{CurrentBlockTime: inter.Timestamp(time.Now().UnixNano())})
	require.Equal(t, "false", h.Result(t, "eth_syncing"))

	chain.fake.SetProgress(core.Progress{
		CurrentEpoch: 4, CurrentBlock: 7, CurrentBlockHash: common.Hash{0x07},
		CurrentBlockTime: inter.Timestamp(time.Now().Add(-2 * time.Hour).UnixNano()),
		HighestBlock:     9, HighestEpoch: 5,
	})
	var progress map[string]any
	h.Call(t, &progress, "eth_syncing")
	require.Equal(t, "0x7", progress["currentBlock"])
	require.Equal(t, "0x9", progress["highestBlock"])
	require.Equal(t, "0x4", progress["currentEpoch"])
	require.Equal(t, "0x5", progress["highestEpoch"])
	require.Equal(t, common.Hash{0x07}.Hex(), progress["currentBlockHash"])
	require.Equal(t, "0x0", progress["startingBlock"])
}

func TestConfig_AnswersTheCurrentAndTheLastUpgrade(t *testing.T) {
	chain := newTestChain(t)
	chain.fake.SetUpgradeHeights([]opera.UpgradeHeight{
		opera.MakeUpgradeHeight(opera.GetSonicUpgrades(), 0),
		opera.MakeUpgradeHeight(opera.GetAllegroUpgrades(), 1),
		opera.MakeUpgradeHeight(opera.GetBrioUpgrades(), 3), // above the head
	})
	h := newHarness(t, chain.fake)
	var config struct {
		Current struct {
			BlockHeight     *hexutil.Big              `json:"blockHeight"`
			ChainID         *hexutil.Big              `json:"chainId"`
			ForkID          hexutil.Bytes             `json:"forkId"`
			SystemContracts map[string]common.Address `json:"systemContracts"`
			Precompiles     map[string]common.Address `json:"precompiles"`
		} `json:"current"`
		Next *struct{} `json:"next"`
		Last *struct {
			BlockHeight *hexutil.Big `json:"blockHeight"`
		} `json:"last"`
	}
	h.Call(t, &config, "eth_config")
	require.Equal(t, int64(1), config.Current.BlockHeight.ToInt().Int64())
	require.Equal(t, int64(tests.ChainID), config.Current.ChainID.ToInt().Int64())
	require.Len(t, config.Current.ForkID, 4)
	require.Contains(t, config.Current.SystemContracts, "HISTORY_STORAGE_ADDRESS")
	require.NotEmpty(t, config.Current.Precompiles)
	require.Nil(t, config.Next)
	require.Equal(t, int64(0), config.Last.BlockHeight.ToInt().Int64())
}

func TestConfig_FailsWithoutUpgrades(t *testing.T) {
	chain := newTestChain(t)
	chain.fake.SetUpgradeHeights(nil)
	h := newHarness(t, chain.fake)
	require.Equal(t, "no configs found", h.Fail(t, "eth_config").Message)
}

// Every method survives a panic of each node call it makes, and answers it
// as an internal error.
func TestMethods_RecoverAPanicOfTheNode(t *testing.T) {
	chain := newTestChain(t)
	h := newHarness(t, chain.fake)
	cases := []struct {
		coreMethod, method string
		params             []any
	}{
		{"Block", "eth_getBlockByNumber", []any{"0x1", false}},
		{"BlockNumberByHash", "eth_getBlockByHash", []any{chain.hashOf(t, 1), false}},
		{"Header", "eth_getHeaderByNumber", []any{"0x1"}},
		{"Receipts", "eth_getBlockReceipts", []any{"0x1"}},
		{"TransactionByHash", "eth_getTransactionByHash", []any{chain.transactions[0].Hash()}},
		{"PoolTransaction", "eth_getRawTransactionByHash", []any{common.Hash{0x01}}},
		{"Progress", "eth_syncing", nil},
		{"ChainInfo", "eth_config", nil},
	}
	for _, test := range cases {
		t.Run(test.coreMethod, func(t *testing.T) {
			chain.fake.SetMode(test.coreMethod, tests.Panic)
			defer chain.fake.SetMode(test.coreMethod, tests.Normal)
			before := h.Panics(t)
			require.Equal(t, dispatch.CodeInternal, h.Fail(t, test.method, test.params...).Code)
			require.Equal(t, before+1, h.Panics(t))
		})
	}
}
