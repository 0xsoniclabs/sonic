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
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/tests"
)

var (
	richAddress = common.HexToAddress("0xa0")
	poorAddress = common.HexToAddress("0xa1")
	// needsData reverts when it is called without call data.
	needsData        = []byte{0x36, 0x60, 0x08, 0x57, 0x60, 0x00, 0x80, 0xfd, 0x5b, 0x00}
	needsDataAddress = common.HexToAddress("0xc4")
)

// estimationChain is a chain of twenty blocks, Osaka from block 10 on.
type estimationChain struct {
	environment *exec.Environment
	core        *tests.CarmenCore
	head        exec.Head
}

func newEstimationChain(t *testing.T, mutations ...tests.Mutate) estimationChain {
	t.Helper()
	carmenCore := tests.NewCarmenCore(t, map[common.Address]tests.Account{
		counterAddress:   {Code: counter},
		reverterAddress:  {Code: reverter},
		looperAddress:    {Code: looper},
		needsDataAddress: {Code: needsData},
		richAddress:      {Balance: new(big.Int).Lsh(big.NewInt(1), 100)},
		poorAddress:      {Balance: new(big.Int).Mul(big.NewInt(20_000), big.NewInt(1_000_000_000))},
	})
	for range 19 {
		carmenCore.AppendBlock(tests.BlockSpec{})
	}
	carmenCore.SetChainInfo(core.ChainInfo{ChainID: tests.FakeChainID, UpgradeHeights: []opera.UpgradeHeight{
		{Upgrades: opera.GetAllegroUpgrades(), Height: 0},
		{Upgrades: opera.GetBrioUpgrades(), Height: 10},
	}})
	carmenCore.PublishHead(19)
	harness := tests.NewHarness(t, carmenCore, func(*dispatch.Dispatcher, *exec.Environment) {}, mutations...)
	harness.WaitForHead(19)
	head, err := harness.Environment.Latest(context.Background())
	require.NoError(t, err)
	return estimationChain{environment: harness.Environment, core: carmenCore, head: head}
}

func gasCap(limit uint64) tests.Mutate {
	return func(configs *tests.Configs) { configs.Exec.RPCGasCap = limit }
}

func (chain estimationChain) estimate(arguments exec.CallArguments, block chain.BlockReference) (hexutil.Uint64, error) {
	return chain.environment.EstimateGas(chain.head, exec.Estimation{Arguments: arguments, Block: block})
}

func atBlock(number uint64) chain.BlockReference {
	return chain.ReferenceToNumber(chain.BlockNumber(number))
}

func addressOf(address common.Address) *common.Address {
	return &address
}

func TestEstimateGas_FindsTheLeastGasThatSucceeds(t *testing.T) {
	estimation := newEstimationChain(t)
	gas, err := estimation.estimate(exec.CallArguments{Recipient: addressOf(poorAddress)}, chain.Latest())
	require.NoError(t, err)
	require.Equal(t, hexutil.Uint64(21000), gas)

	gas, err = estimation.estimate(exec.CallArguments{Recipient: addressOf(counterAddress)}, atBlock(5))
	require.NoError(t, err)
	call := prepare(t, estimation.environment, estimation.head, 5, nil)
	statedb, err := estimation.environment.OpenState(estimation.head, 5)
	require.NoError(t, err)
	defer statedb.Release()
	for _, try := range []struct {
		gas    uint64
		failed bool
	}{{uint64(gas), false}, {uint64(gas) - 1, true}} {
		result, err := call.Run(context.Background(), statedb.Copy(), nil, message(t, counterAddress, try.gas), 0)
		require.NoError(t, err)
		require.Equal(t, try.failed, result.Failed(), "with %d gas", try.gas)
	}
}

// TestEstimateGas_OpensTheStateOnce holds I8 and H2: one state for the
// whole search, its copies for the tries.
func TestEstimateGas_OpensTheStateOnce(t *testing.T) {
	estimation := newEstimationChain(t)
	before := estimation.core.States()
	headers := estimation.core.Calls("Header")
	_, err := estimation.estimate(exec.CallArguments{Recipient: addressOf(counterAddress)}, atBlock(5))
	require.NoError(t, err)
	require.Equal(t, tests.StateCount{Opened: before.Opened + 1, Released: before.Released + 1}, estimation.core.States())
	require.Equal(t, headers+1, estimation.core.Calls("Header"), "the header of the block is read once")
}

func TestEstimateGas_RunsThePreviousMessagesFirst(t *testing.T) {
	estimation := newEstimationChain(t)
	alone, err := estimation.estimate(exec.CallArguments{Recipient: addressOf(counterAddress)}, atBlock(5))
	require.NoError(t, err)
	previous := hexutil.Uint64(100_000)
	after, err := estimation.environment.EstimateGas(estimation.head, exec.Estimation{
		Arguments: exec.CallArguments{Recipient: addressOf(counterAddress)},
		Block:     atBlock(5),
		Previous:  []exec.CallArguments{{Recipient: addressOf(counterAddress), Gas: &previous}},
	})
	require.NoError(t, err)
	require.Less(t, after, alone, "a slot the previous message set is cheaper to change")
}

// TestEstimateGas_RejectsWhatNeverSucceeds: a revert with its data; a run
// that fails otherwise with its error, which Tosca's interpreter words
// "execution unsuccessful" for running out of gas; and gas too little for
// the intrinsic cost with the allowance.
func TestEstimateGas_RejectsWhatNeverSucceeds(t *testing.T) {
	estimation := newEstimationChain(t, gasCap(60_000))
	_, err := estimation.estimate(exec.CallArguments{Recipient: addressOf(reverterAddress)}, atBlock(5))
	require.Equal(t, &dispatch.Error{Code: 3, Message: "execution reverted", Data: "0xdeadbeef"}, dispatch.ToError(err))

	_, err = estimation.estimate(exec.CallArguments{Recipient: addressOf(looperAddress)}, atBlock(5))
	require.EqualError(t, err, "execution unsuccessful")

	estimation = newEstimationChain(t, gasCap(20_999))
	_, err = estimation.estimate(exec.CallArguments{Recipient: addressOf(richAddress)}, atBlock(5))
	require.EqualError(t, err, "gas required exceeds allowance (20999)")
}

// TestEstimateGas_IsCappedByTheTransactionGasLimitFromOsaka: the open
// epoch's limit bounds a block under Osaka, which an override may make.
func TestEstimateGas_IsCappedByTheTransactionGasLimitFromOsaka(t *testing.T) {
	estimation := newEstimationChain(t)
	open := opera.FakeNetRules(opera.GetAllegroUpgrades())
	open.Economy.Gas.MaxEventGas = 20_000 + (open.Economy.Gas.MaxEventGas - open.MaxTransactionGas())
	require.Equal(t, uint64(20_000), open.MaxTransactionGas(), "less than a transfer needs")
	estimation.core.SetOpenRules(open)
	allowance := hexutil.Uint64(1_000_000)
	transfer := exec.CallArguments{Recipient: addressOf(richAddress), Gas: &allowance}
	gas, err := estimation.estimate(transfer, atBlock(5))
	require.NoError(t, err)
	require.Equal(t, hexutil.Uint64(21000), gas)
	_, err = estimation.estimate(transfer, atBlock(15))
	require.EqualError(t, err, "gas required exceeds allowance (20000)")

	number := (*hexutil.Big)(big.NewInt(12))
	_, err = estimation.environment.EstimateGas(estimation.head, exec.Estimation{Arguments: transfer, Block: atBlock(5), BlockOverrides: &exec.BlockOverrides{Number: number}})
	require.EqualError(t, err, "gas required exceeds allowance (20000)")
}

func TestEstimateGas_IsCappedByTheFundsOfTheSender(t *testing.T) {
	estimation := newEstimationChain(t)
	price := (*hexutil.Big)(big.NewInt(1_000_000_000))
	_, err := estimation.estimate(exec.CallArguments{From: addressOf(poorAddress), Recipient: addressOf(richAddress), GasPrice: price}, atBlock(5))
	require.EqualError(t, err, "gas required exceeds allowance (20000)")

	value := (*hexutil.Big)(new(big.Int).Mul(big.NewInt(20_000), big.NewInt(1_000_000_000)))
	_, err = estimation.estimate(exec.CallArguments{From: addressOf(poorAddress), Recipient: addressOf(richAddress), GasPrice: price, Value: value}, atBlock(5))
	require.EqualError(t, err, "insufficient funds for transfer")
}

// TestEstimateGas_WordsAMissingBlockAsTheReference holds F3 and F2: the
// block is checked before the fees.
func TestEstimateGas_WordsAMissingBlockAsTheReference(t *testing.T) {
	estimation := newEstimationChain(t)
	unknown := common.Hash{0xee}
	badFees := exec.CallArguments{GasPrice: (*hexutil.Big)(big.NewInt(1)), MaxFeePerGas: (*hexutil.Big)(big.NewInt(1))}
	for _, arguments := range []exec.CallArguments{{}, badFees} {
		_, err := estimation.estimate(arguments, atBlock(50))
		require.EqualError(t, err, "block does not exists")
		_, err = estimation.estimate(arguments, chain.ReferenceToHash(unknown))
		require.EqualError(t, err, "block with hash "+unknown.String()+" not found")
		_, err = estimation.estimate(arguments, chain.BlockReference{})
		require.EqualError(t, err, "invalid block number or hash")
	}
	_, err := estimation.estimate(badFees, chain.Latest())
	require.EqualError(t, err, "both gasPrice and (maxFeePerGas or maxPriorityFeePerGas) specified")
}

func TestEstimateGas_AppliesTheStateOverrides(t *testing.T) {
	estimation := newEstimationChain(t)
	code := hexutil.Bytes(reverter)
	_, err := estimation.environment.EstimateGas(estimation.head, exec.Estimation{
		Arguments:      exec.CallArguments{Recipient: addressOf(counterAddress)},
		Block:          atBlock(5),
		StateOverrides: exec.StateOverride{counterAddress: {Code: &code}},
	})
	require.EqualError(t, err, "execution reverted")
	storage := map[common.Hash]common.Hash{}
	_, err = estimation.environment.EstimateGas(estimation.head, exec.Estimation{
		Arguments:      exec.CallArguments{Recipient: addressOf(counterAddress)},
		Block:          atBlock(5),
		StateOverrides: exec.StateOverride{counterAddress: {State: &storage, StateDiff: &storage}},
	})
	require.ErrorContains(t, err, "has both 'state' and 'stateDiff'")
}
