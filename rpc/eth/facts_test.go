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
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/eth"
	"github.com/0xsoniclabs/sonic/rpc/tests"
)

func TestFacts_ChainIdBlockNumberAndConstants(t *testing.T) {
	h := newEth(t, chainOf(3))
	var id hexutil.Big
	h.MustCall(&id, "eth_chainId")
	require.Equal(t, int64(tests.ChainID), id.ToInt().Int64())

	var number hexutil.Uint64
	h.MustCall(&number, "eth_blockNumber")
	require.Equal(t, hexutil.Uint64(3), number)

	var addr common.Address
	h.MustCall(&addr, "eth_coinbase")
	require.Equal(t, common.Address{}, addr)
	h.MustCall(&addr, "eth_etherbase")
	require.Equal(t, common.Address{}, addr)
	require.Equal(t, `"0x0"`, string(h.MustCall(nil, "eth_hashrate")))
}

func TestFacts_SyncingFollowsTheAgeOfTheNewestBlock(t *testing.T) {
	fc := chainOf(1)
	h := newEth(t, fc)
	fc.SetProgress(core.Progress{CurrentBlock: 1, CurrentEpoch: 7, CurrentBlockTime: inter.Timestamp(time.Now().UnixNano())})
	require.Equal(t, "false", string(h.MustCall(nil, "eth_syncing")))

	fc.SetProgress(core.Progress{CurrentBlock: 5, CurrentEpoch: 7, HighestBlock: 9, HighestEpoch: 8, CurrentBlockTime: inter.FromUnix(1)})
	var status map[string]any
	h.MustCall(&status, "eth_syncing")
	require.Equal(t, "0x5", status["currentBlock"])
	require.Equal(t, "0x9", status["highestBlock"])
	require.Equal(t, "0x8", status["highestEpoch"])
	require.Equal(t, "0x0", status["startingBlock"])

	var epoch hexutil.Uint64
	h.MustCall(&epoch, "eth_currentEpoch")
	require.Equal(t, hexutil.Uint64(7), epoch)
}

func TestFacts_GetRulesAnswersTheOpenEpochAtTheHead(t *testing.T) {
	fc := chainOf(2)
	open := opera.FakeNetRules(opera.GetBrioUpgrades())
	open.Name = "open epoch"
	fc.SetOpenRules(open)
	h := newEth(t, fc)

	var rules opera.Rules
	h.MustCall(&rules, "eth_getRules", "latest")
	require.Equal(t, "open epoch", rules.Name)
	h.MustCall(&rules, "eth_getRules", "0x1")
	require.NotEqual(t, "open epoch", rules.Name)
	require.Equal(t, "null", string(h.MustCall(nil, "eth_getRules", "0x3")), "a block above the head")
	require.Equal(t, -32602, h.Fails("eth_getRules", "0x8000000000000000").Code)
}

func TestFacts_ConfigAnswersTheCurrentAndTheLastUpgrade(t *testing.T) {
	fc := chainOf(3)
	rules := opera.FakeNetRules(opera.GetBrioUpgrades())
	fc.SetUpgradeHeights([]opera.UpgradeHeight{
		{Upgrades: opera.GetSonicUpgrades(), Height: 0},
		{Upgrades: rules.Upgrades, Height: 2},
	})
	h := newEth(t, fc)
	var cfg struct {
		Current, Next, Last *struct {
			BlockHeight     hexutil.Big       `json:"blockHeight"`
			ChainID         hexutil.Big       `json:"chainId"`
			ForkID          hexutil.Bytes     `json:"forkId"`
			Precompiles     map[string]string `json:"precompiles"`
			SystemContracts map[string]string `json:"systemContracts"`
		}
	}
	h.MustCall(&cfg, "eth_config")
	require.Nil(t, cfg.Next)
	require.Equal(t, int64(2), cfg.Current.BlockHeight.ToInt().Int64())
	require.Equal(t, int64(0), cfg.Last.BlockHeight.ToInt().Int64())
	require.Equal(t, int64(tests.ChainID), cfg.Current.ChainID.ToInt().Int64())
	want, err := eth.MakeForkId(opera.UpgradeHeight{Upgrades: rules.Upgrades, Height: 2}, common.HexToHash("0x6e57"))
	require.NoError(t, err)
	require.Equal(t, hexutil.Bytes(want[:]), cfg.Current.ForkID)
	require.Contains(t, cfg.Current.SystemContracts, "HISTORY_STORAGE_ADDRESS")
	require.NotContains(t, cfg.Last.SystemContracts, "HISTORY_STORAGE_ADDRESS")
	require.NotEmpty(t, cfg.Current.Precompiles)
}

func TestFacts_ConfigOfAnUpgradeAboveTheHeadFails(t *testing.T) {
	fc := chainOf(1)
	fc.SetUpgradeHeights([]opera.UpgradeHeight{{Upgrades: opera.GetSonicUpgrades(), Height: 5}})
	h := newEth(t, fc)
	require.Equal(t, "failed to get current config, block 5 not found to determine activation time", h.Fails("eth_config").Message)
}

func TestFees_GasPriceIsTheBaseFeeWithAMargin(t *testing.T) {
	fc := chainOf(1)
	h := newEth(t, fc)
	var price hexutil.Big
	h.MustCall(&price, "eth_gasPrice")
	head, err := fc.Header(1)
	require.NoError(t, err)
	require.Positive(t, price.ToInt().Cmp(head.BaseFee))
}

func TestFees_MaxPriorityFeeIsAtLeastTheMinimumTip(t *testing.T) {
	fc := chainOf(1)
	rules := opera.FakeNetRules(opera.GetBrioUpgrades())
	fc.SetOracleInputs(core.OracleInputs{PendingRules: rules, MinTip: big.NewInt(7_000)})
	h := newEth(t, fc)
	var tip hexutil.Big
	h.MustCall(&tip, "eth_maxPriorityFeePerGas")
	require.GreaterOrEqual(t, tip.ToInt().Int64(), int64(7_000))
	require.Equal(t, `"0x1"`, string(h.MustCall(nil, "eth_blobBaseFee")))
}

func TestFees_FeeHistory(t *testing.T) {
	h := newEth(t, chainOf(5))
	var res eth.FeeHistoryResult
	h.MustCall(&res, "eth_feeHistory", "0x2", "latest", []float64{10, 90})
	require.Equal(t, int64(4), res.OldestBlock.ToInt().Int64())
	require.Len(t, res.Reward, 2)
	require.Len(t, res.Reward[0], 2)
	require.Len(t, res.BaseFee, 3)
	require.Len(t, res.GasUsedRatio, 2)

	h.MustCall(&res, "eth_feeHistory", "0x2000", "0x3", nil)
	require.Zero(t, res.OldestBlock.ToInt().Int64(), "the count is clamped at the genesis")
	require.Len(t, res.GasUsedRatio, 4)

	raw := h.MustCall(nil, "eth_feeHistory", "0x0", "latest", nil)
	require.JSONEq(t, `{"oldestBlock":"0x0","gasUsedRatio":null}`, string(raw))

	tests := map[string]struct {
		params []any
		msg    string
	}{
		"above head":    {[]any{"0x1", "0x6", nil}, "block 6 not found; latest block is 5"},
		"negative":      {[]any{"0x1", "latest", []float64{-1}}, "invalid reward percentile: -1.000000"},
		"above 100":     {[]any{"0x1", "latest", []float64{101}}, "invalid reward percentile: 101.000000"},
		"not ascending": {[]any{"0x1", "latest", []float64{50, 10}}, "invalid reward percentile: #0:50.000000 > #1:10.000000"},
		"too many":      {[]any{"0x1", "latest", make([]float64, 101)}, "invalid reward percentile: over the query limit 100"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			e := h.Fails("eth_feeHistory", tc.params...)
			require.Equal(t, -32000, e.Code)
			require.Equal(t, tc.msg, e.Message)
		})
	}
	var decimal eth.FeeHistoryResult
	require.NoError(t, json.Unmarshal(h.MustCall(nil, "eth_feeHistory", 2, "latest", nil), &decimal), "a decimal count, as go-ethereum takes it")
	require.Len(t, decimal.GasUsedRatio, 2)
}
