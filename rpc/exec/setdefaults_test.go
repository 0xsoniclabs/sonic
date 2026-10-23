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

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/tests"
)

func newEnv(t *testing.T, opts ...tests.Option) (*exec.Env, *tests.FakeCore) {
	t.Helper()
	fc := tests.NewFakeCore(t, append([]tests.Option{tests.WithChainID(146), tests.WithBlocks(2)}, opts...)...)
	return tests.NewEnvHarness(t, fc, nil).Env, fc
}

func TestEnv_MaxGasLimitFollowsTheRulesOfTheNextBlock(t *testing.T) {
	// The head (block 2) seals an epoch: block 3 has a larger event gas limit.
	env, _ := newEnv(t, tests.WithRules(func(block uint64) opera.Rules {
		rules := opera.FakeNetRules(opera.GetBrioUpgrades())
		if block > 2 {
			rules.Economy.Gas.MaxEventGas += 1_000_000
		}
		return rules
	}))
	head := env.Chain.Head()
	require.EqualValues(t, 2, head.Number)
	old, err := env.Core.Rules(head.Number)
	require.NoError(t, err)
	next, err := env.Core.Rules(head.Number + 1)
	require.NoError(t, err)
	require.Equal(t, old.Economy.Gas.MaxEventGas+1_000_000, next.Economy.Gas.MaxEventGas)
	require.Equal(t, 1_000_000+maxGasLimitOf(old), env.MaxGasLimit(head))
}

func maxGasLimitOf(rules *opera.Rules) uint64 {
	empty := rules.Economy.Gas.EventGas +
		uint64(rules.Dag.MaxParents-rules.Dag.MaxFreeParents)*rules.Economy.Gas.ParentGas +
		uint64(rules.Dag.MaxExtraData)*rules.Economy.Gas.ExtraDataGas
	return rules.Economy.Gas.MaxEventGas - empty
}

func TestSetDefaults_RejectsGasPriceWithAuthorizationList(t *testing.T) {
	env, _ := newEnv(t)
	fee := (*hexutil.Big)(big.NewInt(1))
	args := exec.CallArgs{GasPrice: fee, AuthorizationList: []types.SetCodeAuthorization{{}}}
	err := args.SetDefaults(context.Background(), env, env.Chain.Head())
	require.EqualError(t, err, "both gasPrice and authorizationList specified")
}

func TestSetDefaults_RejectsBothFeeStyles(t *testing.T) {
	env, _ := newEnv(t)
	fee := (*hexutil.Big)(big.NewInt(1))
	args := exec.CallArgs{GasPrice: fee, MaxFeePerGas: fee}
	err := args.SetDefaults(context.Background(), env, env.Chain.Head())
	require.EqualError(t, err, "both gasPrice and (maxFeePerGas or maxPriorityFeePerGas) specified")
}

func TestSetDefaults_RejectsMaxFeeBelowTip(t *testing.T) {
	env, _ := newEnv(t)
	args := exec.CallArgs{MaxFeePerGas: (*hexutil.Big)(big.NewInt(1)), MaxPriorityFeePerGas: (*hexutil.Big)(big.NewInt(2))}
	err := args.SetDefaults(context.Background(), env, env.Chain.Head())
	require.EqualError(t, err, "maxFeePerGas (0x1) < maxPriorityFeePerGas (0x2)")
}

func complete(addr common.Address) exec.CallArgs {
	nonce := hexutil.Uint64(0)
	gas := hexutil.Uint64(21000)
	fee := (*hexutil.Big)(big.NewInt(100))
	return exec.CallArgs{From: &addr, To: &addr, Nonce: &nonce, Gas: &gas, MaxFeePerGas: fee, MaxPriorityFeePerGas: fee}
}

func TestSetDefaults_RejectsContractCreationWithoutData(t *testing.T) {
	env, _ := newEnv(t)
	addr := common.Address{1}
	for name, test := range map[string]struct {
		mutate func(*exec.CallArgs)
		want   string
	}{
		"blob hashes":  {func(a *exec.CallArgs) { a.To = nil; a.BlobHashes = []common.Hash{{1}} }, `missing "to" in blob transaction`},
		"blob fee cap": {func(a *exec.CallArgs) { a.To = nil; a.BlobFeeCap = (*hexutil.Big)(big.NewInt(1)) }, `missing "to" in blob transaction`},
		"no data":      {func(a *exec.CallArgs) { a.To = nil }, `contract creation without any data provided`},
		"authorization list": {func(a *exec.CallArgs) {
			a.To = nil
			data := hexutil.Bytes{1}
			a.Data = &data
			a.AuthorizationList = []types.SetCodeAuthorization{{}}
		}, `authorizationList provided for contract creation, but "to" field is missing`},
		"data and input differ": {func(a *exec.CallArgs) {
			d1, d2 := hexutil.Bytes{1}, hexutil.Bytes{2}
			a.Data, a.Input = &d1, &d2
		}, `both "data" and "input" are set and not equal. Please use "input" to pass transaction call data`},
	} {
		t.Run(name, func(t *testing.T) {
			args := complete(addr)
			test.mutate(&args)
			require.EqualError(t, args.SetDefaults(context.Background(), env, env.Chain.Head()), test.want)
		})
	}
}

func TestSetDefaults_ChainID(t *testing.T) {
	env, _ := newEnv(t)
	addr := common.Address{1}
	for name, test := range map[string]struct {
		chainID *hexutil.Big
		wantErr string
	}{
		"nil chainID defaults to node's chainID": {chainID: nil},
		"matching chainID is accepted":           {chainID: (*hexutil.Big)(big.NewInt(146))},
		"mismatching chainID is rejected": {
			chainID: (*hexutil.Big)(big.NewInt(1337)),
			wantErr: "chainId does not match node's (have=1337, want=146)",
		},
	} {
		t.Run(name, func(t *testing.T) {
			args := complete(addr)
			args.ChainID = test.chainID
			err := args.SetDefaults(context.Background(), env, env.Chain.Head())
			if test.wantErr != "" {
				require.EqualError(t, err, test.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, (*hexutil.Big)(big.NewInt(146)), args.ChainID)
		})
	}
}

func TestSetDefaults_FillsNonceFeesAndValue(t *testing.T) {
	env, fc := newEnv(t)
	addr := common.Address{1}
	fc.SetPoolNonce(addr, 11)
	gas := hexutil.Uint64(21000)
	args := exec.CallArgs{From: &addr, To: &addr, Gas: &gas}
	require.NoError(t, args.SetDefaults(context.Background(), env, env.Chain.Head()))
	require.EqualValues(t, 11, *args.Nonce)
	require.Equal(t, big.NewInt(0), args.Value.ToInt())
	// Dynamic fee transaction: the cap is the tip over the minimal price.
	require.NotNil(t, args.MaxFeePerGas)
	require.NotNil(t, args.MaxPriorityFeePerGas)
	require.Nil(t, args.GasPrice)
	require.Equal(t, new(big.Int).Add(args.MaxPriorityFeePerGas.ToInt(), big.NewInt(1100)), args.MaxFeePerGas.ToInt())
}

func TestSetDefaults_LegacyPricingBeforeLondon(t *testing.T) {
	// A fee style the chain does not know refuses 1559 fields.
	env, _ := newEnv(t)
	addr := common.Address{1}
	gas := hexutil.Uint64(21000)
	nonce := hexutil.Uint64(0)
	args := exec.CallArgs{From: &addr, To: &addr, Gas: &gas, Nonce: &nonce, GasPrice: (*hexutil.Big)(big.NewInt(7))}
	require.NoError(t, args.SetDefaults(context.Background(), env, env.Chain.Head()))
	require.Nil(t, args.MaxFeePerGas)
	require.Equal(t, big.NewInt(7), args.GasPrice.ToInt())
}
