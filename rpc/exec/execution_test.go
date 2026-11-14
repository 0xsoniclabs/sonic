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

	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/tests"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

var (
	funded  = common.Address{0x0f}
	writer  = common.Address{0xc0}
	spinner = common.Address{0xc1}
	slot0   = common.Hash{}
)

func newStateEnvironment(t *testing.T) *tests.Harness {
	t.Helper()
	fake := tests.NewCarmenCore(t, map[common.Address]tests.AccountState{
		funded:  {Balance: big.NewInt(1e18)},
		writer:  {Code: tests.WritesStorage},
		spinner: {Code: tests.Spins},
	})
	fake.AddBlock()
	fake.SetHead(1)
	fake.SetPoolNonce(funded, 3)
	return tests.NewHarness(t, fake, func(*tests.Harness) {})
}

func TestStateAt_WordsMissingBlocksAsTheLegacyStateReads(t *testing.T) {
	h := newStateEnvironment(t)
	head := h.Environment.Head()
	unknown := common.Hash{0x01}
	cases := map[string]struct {
		reference chain.BlockReference
		err       string
	}{
		"above the head":   {chain.ReferenceTo(2), "header not found"},
		"unknown hash":     {chain.BlockReference{Hash: &unknown}, "header not found"},
		"missing selector": {chain.BlockReference{}, "unknown header selector"},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := h.Environment.OpenState(head, test.reference)
			require.EqualError(t, err, test.err)
			_, _, err = h.Environment.StateAt(head, test.reference)
			require.EqualError(t, err, test.err)
		})
	}
}

// A node without an archive refuses a state read by tag as the legacy stack,
// which looks for the newest archived block first, and one by number with
// the archive's words.
func TestOpenState_WithoutAnArchive(t *testing.T) {
	h := newStateEnvironment(t)
	h.Core.SetMode("State", tests.NoArchive)
	head := h.Environment.Head()
	_, err := h.Environment.OpenState(head, chain.ReferenceTo(chain.LatestBlockNumber))
	require.EqualError(t, err, "failed to get latest block number; failed to get latest archive block; no archive")
	_, err = h.Environment.OpenState(head, chain.ReferenceTo(1))
	require.EqualError(t, err, "no archive")
	_, err = h.Environment.OpenState(head, chain.ReferenceTo(chain.EarliestBlockNumber))
	require.EqualError(t, err, "no archive")
}

func TestRules_WordMissingRules(t *testing.T) {
	h := newStateEnvironment(t)
	h.Core.SetMode("Rules", tests.NotFound)
	_, err := h.Environment.VMConfig(1)
	require.EqualError(t, err, "no network rules found for block height 1")
	_, err = h.Environment.MaxGasLimit(h.Environment.Head())
	require.EqualError(t, err, "no network rules found for block height 2")
}

func TestMaxGasLimit_IsTheGasOfAnEventLessAnEmptyOne(t *testing.T) {
	h := newStateEnvironment(t)
	rules, err := h.Core.Rules(2)
	require.NoError(t, err)
	gas := rules.Economy.Gas
	limit, err := h.Environment.MaxGasLimit(h.Environment.Head())
	require.NoError(t, err)
	require.Equal(t, gas.MaxEventGas-gas.EventGas-uint64(rules.Dag.MaxParents-rules.Dag.MaxFreeParents)*gas.ParentGas-
		uint64(rules.Dag.MaxExtraData)*gas.ExtraDataGas, limit)
}

func TestExecute_RunsTheCallsBeforeFirst(t *testing.T) {
	h := newStateEnvironment(t)
	head := h.Environment.Head()
	stateDB, header, err := h.Environment.StateAt(head, chain.ReferenceTo(1))
	require.NoError(t, err)
	defer stateDB.Release()
	to := writer
	result, err := h.Environment.Execute(context.Background(), stateDB, header, exec.Call{
		Arguments: exec.CallArguments{To: &to},
		Before:    []exec.CallArguments{{To: &to}, {GasPrice: (*hexutil.Big)(big.NewInt(1)), MaxFeePerGas: (*hexutil.Big)(big.NewInt(1))}},
	})
	require.NoError(t, err)
	require.False(t, result.Failed())
	require.Equal(t, common.BigToHash(big.NewInt(1)), stateDB.GetState(writer, slot0), "the call before wrote the slot")
}

func TestExecute_FailsACancelledRun(t *testing.T) {
	h := newStateEnvironment(t)
	stateDB, header, err := h.Environment.StateAt(h.Environment.Head(), chain.ReferenceTo(1))
	require.NoError(t, err)
	defer stateDB.Release()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	// The cancellation reaches the EVM in a goroutine of its own; a run long
	// enough sees it.
	to := spinner
	_, err = h.Environment.Execute(ctx, stateDB, header, exec.Call{Arguments: exec.CallArguments{To: &to}})
	require.EqualError(t, err, "execution aborted (timeout = 0s)")
}

func TestToMessage_PricesAsTheLegacyCalls(t *testing.T) {
	price := func(value int64) *hexutil.Big { return (*hexutil.Big)(big.NewInt(value)) }
	cases := map[string]struct {
		args               exec.CallArguments
		baseFee            *big.Int
		gasPrice, fee, tip int64
	}{
		"before London":           {exec.CallArguments{GasPrice: price(5)}, nil, 5, 5, 5},
		"before London, unpriced": {exec.CallArguments{}, nil, 0, 0, 0},
		"legacy price":            {exec.CallArguments{GasPrice: price(5)}, big.NewInt(3), 5, 5, 5},
		"dynamic fee":             {exec.CallArguments{MaxFeePerGas: price(10), MaxPriorityFeePerGas: price(2)}, big.NewInt(3), 5, 10, 2},
		"dynamic fee at its cap":  {exec.CallArguments{MaxFeePerGas: price(4), MaxPriorityFeePerGas: price(2)}, big.NewInt(3), 4, 4, 2},
		"unpriced":                {exec.CallArguments{}, big.NewInt(3), 0, 0, 0},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			message, err := test.args.ToMessage(0, test.baseFee)
			require.NoError(t, err)
			require.Equal(t, test.gasPrice, message.GasPrice.Int64())
			require.Equal(t, test.fee, message.GasFeeCap.Int64())
			require.Equal(t, test.tip, message.GasTipCap.Int64())
		})
	}
}

func TestToMessage_CapsTheGasAndTakesTheInput(t *testing.T) {
	gas := hexutil.Uint64(100)
	data, input := hexutil.Bytes{0x01}, hexutil.Bytes{0x02}
	message, err := (&exec.CallArguments{Gas: &gas, Data: &data, Input: &input, BlobHashes: []common.Hash{{}}}).ToMessage(50, nil)
	require.NoError(t, err)
	require.Equal(t, uint64(50), message.GasLimit)
	require.Equal(t, []byte{0x02}, message.Data)
	require.Equal(t, int64(0), message.BlobGasFeeCap.Int64(), "blob hashes default their fee cap to zero")
}

func TestStateOverride_ReplacesAccounts(t *testing.T) {
	h := newStateEnvironment(t)
	stateDB, err := h.Environment.OpenState(h.Environment.Head(), chain.ReferenceTo(1))
	require.NoError(t, err)
	defer stateDB.Release()
	nonce := hexutil.Uint64(5)
	code := hexutil.Bytes{0x00}
	balance := new(hexutil.U256)
	overrides := exec.StateOverride{
		funded: {Nonce: &nonce, Code: &code, Balance: &balance, StateDiff: &map[common.Hash]common.Hash{{0x01}: {0x02}}},
		writer: {State: &map[common.Hash]common.Hash{{0x03}: {0x04}}},
	}
	require.NoError(t, overrides.Apply(stateDB))
	require.Equal(t, uint64(5), stateDB.GetNonce(funded))
	require.Equal(t, []byte{0x00}, stateDB.GetCode(funded))
	require.True(t, stateDB.GetBalance(funded).IsZero())
	require.Equal(t, common.Hash{0x02}, stateDB.GetState(funded, common.Hash{0x01}))
	require.Equal(t, common.Hash{0x04}, stateDB.GetState(writer, common.Hash{0x03}))

	var none *exec.StateOverride
	require.NoError(t, none.Apply(stateDB))
}

func TestSetDefaults_FillsANewTransaction(t *testing.T) {
	h := newStateEnvironment(t)
	to := common.Address{0x42}
	args := exec.CallArguments{From: &funded, To: &to}
	require.NoError(t, h.Environment.SetDefaults(context.Background(), h.Environment.Head(), &args))
	require.Equal(t, hexutil.Uint64(3), *args.Nonce, "the pool's nonce")
	require.Equal(t, hexutil.Uint64(21000), *args.Gas, "the estimate")
	require.Equal(t, int64(tests.ChainID), args.ChainID.ToInt().Int64())
	require.NotNil(t, args.MaxPriorityFeePerGas)
	require.Equal(t, 1, args.MaxFeePerGas.ToInt().Cmp(args.MaxPriorityFeePerGas.ToInt()), "the fee cap adds the suggested price")
	require.Zero(t, args.Value.ToInt().Sign())
}

func TestSetDefaults_RefusesWhatATransactionCannotHold(t *testing.T) {
	h := newStateEnvironment(t)
	price := (*hexutil.Big)(big.NewInt(1))
	to := common.Address{0x42}
	data, input := hexutil.Bytes{0x01}, hexutil.Bytes{0x02}
	cases := map[string]struct {
		args exec.CallArguments
		err  string
	}{
		"both pricings":       {exec.CallArguments{GasPrice: price, MaxFeePerGas: price}, "both gasPrice and (maxFeePerGas or maxPriorityFeePerGas) specified"},
		"price and code list": {exec.CallArguments{GasPrice: price, AuthorizationList: []types.SetCodeAuthorization{}}, "both gasPrice and authorizationList specified"},
		"fee cap below tip": {exec.CallArguments{MaxFeePerGas: price, MaxPriorityFeePerGas: (*hexutil.Big)(big.NewInt(2))},
			"maxFeePerGas (0x1) < maxPriorityFeePerGas (0x2)"},
		"data and input":     {exec.CallArguments{To: &to, Data: &data, Input: &input}, `both "data" and "input" are set and not equal. Please use "input" to pass transaction call data`},
		"blob creation":      {exec.CallArguments{BlobHashes: []common.Hash{{}}}, `missing "to" in blob transaction`},
		"empty creation":     {exec.CallArguments{}, `contract creation without any data provided`},
		"code list creation": {exec.CallArguments{Input: &input, AuthorizationList: []types.SetCodeAuthorization{}}, `authorizationList provided for contract creation, but "to" field is missing`},
		"other chain":        {exec.CallArguments{To: &to, ChainID: price}, "chainId does not match node's (have=1, want=4003)"},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			args := test.args
			if name == "other chain" {
				gas := hexutil.Uint64(21000)
				args.Gas = &gas
			}
			require.EqualError(t, h.Environment.SetDefaults(context.Background(), h.Environment.Head(), &args), test.err)
		})
	}
}

func TestSetDefaults_PricesALegacyTransactionByItsGasPrice(t *testing.T) {
	h := newStateEnvironment(t)
	to := common.Address{0x42}
	price := (*hexutil.Big)(big.NewInt(7))
	gas := hexutil.Uint64(21000)
	args := exec.CallArguments{To: &to, GasPrice: price, Gas: &gas}
	require.NoError(t, h.Environment.SetDefaults(context.Background(), h.Environment.Head(), &args))
	require.Nil(t, args.MaxFeePerGas)
	require.Equal(t, price, args.GasPrice)
}
