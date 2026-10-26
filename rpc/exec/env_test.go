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
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/core"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/tests"
)

func newEnv(t *testing.T, mutate ...tests.Mutate) (*tests.ReplayChain, *exec.Env) {
	t.Helper()
	r := tests.NewReplayChain(t)
	h := tests.NewHarness(t, r.Core, nil, mutate...)
	return r, h.Env
}

// TestLookups_AbsenceIsASentinelNeverNilNil is H1 and section 2: every lookup
// answers a value or an error, and absence reaches the caller as its sentinel.
func TestLookups_AbsenceIsASentinelNeverNilNil(t *testing.T) {
	r, env := newEnv(t)
	head := env.Latest(context.Background())
	invalid := chain.NumberRef(-9)
	unknown := chain.HashRef(common.Hash{1})
	above := chain.NumberRef(5)
	none := chain.BlockRef{}

	type lookup func(chain.BlockRef) (any, error)
	lookups := map[string]lookup{
		"FindHeader": func(ref chain.BlockRef) (any, error) { return env.FindHeader(head, ref) },
		"FindBlock":  func(ref chain.BlockRef) (any, error) { return env.FindBlock(head, ref) },
		"OpenState": func(ref chain.BlockRef) (any, error) {
			st, err := env.OpenState(head, ref)
			if st != nil {
				st.Release()
			}
			return st, err
		},
		"StateAt": func(ref chain.BlockRef) (any, error) {
			st, header, err := env.StateAt(head, ref)
			if st != nil {
				st.Release()
			}
			if err == nil && header == nil {
				return nil, nil
			}
			return st, err
		},
	}
	for name, find := range lookups {
		_, err := find(invalid)
		require.ErrorIs(t, err, chain.ErrInvalidTag, name)
		_, err = find(unknown)
		require.ErrorIs(t, err, chain.ErrUnknownHash, name)
		_, err = find(above)
		require.ErrorIs(t, err, core.ErrNotFound, name)
		_, err = find(none)
		require.ErrorIs(t, err, chain.ErrNoSelector, name)
		v, err := find(chain.HashRef(r.Block.Hash))
		require.NoError(t, err, name)
		require.NotNil(t, v, name)
	}

	for _, mode := range []tests.Mode{tests.Nil, tests.NotFound} {
		for _, method := range []string{"Header", "Block", "State", "TxByHash", "Rules", "Receipts", "BundleInfo"} {
			r.Core.SetMode(method, mode)
		}
		_, err := env.HeaderAt(head, 1)
		require.ErrorIs(t, err, core.ErrNotFound, mode.String())
		_, err = env.BlockAt(head, 1)
		require.ErrorIs(t, err, core.ErrNotFound, mode.String())
		_, err = env.StateAtNumber(head, 1)
		require.ErrorIs(t, err, core.ErrNotFound, mode.String())
		_, _, _, err = env.TxByHash(head, r.Call.Hash())
		require.ErrorIs(t, err, core.ErrNotFound, mode.String())
		_, err = env.Rules(context.Background(), 1)
		require.ErrorIs(t, err, core.ErrNotFound, mode.String())
		if mode == tests.NotFound {
			// A nil answer of a value is its zero value.
			_, err = env.BundleInfo(head, common.Hash{})
			require.ErrorIs(t, err, core.ErrNotFound, mode.String())
		}
		for _, method := range []string{"Header", "Block", "State", "TxByHash", "Rules", "Receipts", "BundleInfo"} {
			r.Core.SetMode(method, tests.Normal)
		}
	}
	receipts, err := env.ReceiptsAt(head, 1)
	require.NoError(t, err)
	require.Len(t, receipts, 2)
	_, err = env.ReceiptsAt(head, 2)
	require.ErrorIs(t, err, core.ErrNotFound)
	_, _, _, err = env.TxByHash(head, common.Hash{1})
	require.ErrorIs(t, err, core.ErrNotFound)

	opened, released := r.Core.TrackStates()
	require.Equal(t, opened, released)
}

// TestLookups_ABlockAboveTheRequestsHeadIsMissing: a request reads the chain
// at the head it took, also after the head advanced.
func TestLookups_ABlockAboveTheRequestsHeadIsMissing(t *testing.T) {
	r := tests.NewReplayChain(t)
	h := tests.NewHarness(t, r.Core, nil)
	head := h.Env.Latest(context.Background())
	tx := r.Wallet.Transfer(t, 2, r.Recipient, big.NewInt(1))
	r.Core.AddAndPublish(types.Transactions{tx})
	h.WaitHead(2)
	_, err := h.Env.HeaderAt(head, 2)
	require.ErrorIs(t, err, core.ErrNotFound)
	_, _, _, err = h.Env.TxByHash(head, tx.Hash())
	require.ErrorIs(t, err, core.ErrNotFound)
	r.Core.SetBundle(common.Hash{1}, core.BundleInfo{BlockNumber: 2})
	_, err = h.Env.BundleInfo(head, common.Hash{1})
	require.ErrorIs(t, err, core.ErrNotFound)
	_, _, _, err = h.Env.TxByHash(h.Env.Latest(context.Background()), tx.Hash())
	require.NoError(t, err)
}

// TestLookups_ACancelledRequestAsksNothing is Q2: a lookup checks the
// request's context before it asks the node and after.
func TestLookups_ACancelledRequestAsksNothing(t *testing.T) {
	r, env := newEnv(t)
	ctx, cancel := context.WithCancel(context.Background())
	head := env.Latest(ctx)
	cancel()
	headers, states := r.Core.Calls("Header"), r.Core.Calls("State")
	_, err := env.HeaderAt(head, 1)
	require.ErrorIs(t, err, context.Canceled)
	_, err = env.StateAtNumber(head, 1)
	require.ErrorIs(t, err, context.Canceled)
	_, err = env.Resolve(head, chain.LatestRef())
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, headers, r.Core.Calls("Header"))
	require.Equal(t, states, r.Core.Calls("State"))
	require.Equal(t, ctx, head.Context())

	// A state opened while the request was cancelled is released.
	ctx2, cancel2 := context.WithCancel(context.Background())
	release := r.Core.Gate("State")
	done := make(chan error)
	go func() {
		_, err := env.StateAtNumber(env.Latest(ctx2), 1)
		done <- err
	}()
	tests.WaitFor(t, func() bool { return r.Core.Calls("State") > states })
	cancel2()
	release()
	require.ErrorIs(t, <-done, context.Canceled)
	opened, released := r.Core.TrackStates()
	require.Equal(t, opened, released)
}

func TestFacts_ChainIdSignerConfigAndRules(t *testing.T) {
	r, env := newEnv(t)
	require.Equal(t, int64(tests.ChainID), env.ChainID().Int64())
	require.Equal(t, int64(tests.ChainID), env.Signer().ChainID().Int64())
	require.Equal(t, uint64(tests.ChainID), env.ChainInfo().NetworkID)
	require.Equal(t, int64(tests.ChainID), env.ChainConfig(1).ChainID.Int64())
	head := env.Latest(context.Background())
	rules, err := env.OpenRules(head)
	require.NoError(t, err)
	limit, err := env.MaxGasLimit(head)
	require.NoError(t, err)
	require.Positive(t, limit)
	require.Positive(t, rules.Blocks.MaxBlockGas)
	_, err = env.VMConfig(context.Background(), 1)
	require.NoError(t, err)
	r.Core.SetMode("Rules", tests.Fail)
	_, err = env.VMConfig(context.Background(), 1)
	require.EqualError(t, err, "no network rules found for block height 1: "+tests.ErrFake.Error())
	_, err = env.MaxGasLimit(head)
	require.ErrorIs(t, err, tests.ErrFake)
	require.Positive(t, env.SuggestedGasPrice(head).Sign())
}

// TestBlockContext_BlockHashIsOneLookup is V11 at the source.
func TestBlockContext_BlockHashIsOneLookup(t *testing.T) {
	r, env := newEnv(t)
	header, err := env.HeaderAt(env.Latest(context.Background()), 1)
	require.NoError(t, err)
	ctx := env.BlockContext(header)
	block0, err := r.Core.Header(0)
	require.NoError(t, err)
	before := r.Core.Calls("Header")
	require.Equal(t, block0.Hash, ctx.GetHash(0))
	require.Equal(t, common.Hash{}, ctx.GetHash(1), "the block of the call")
	require.Equal(t, common.Hash{}, ctx.GetHash(1<<40))
	require.Equal(t, before+1, r.Core.Calls("Header"))
	r.Core.SetMode("Header", tests.Fail)
	require.Equal(t, common.Hash{}, ctx.GetHash(0))
}

func TestNullIfNotFound(t *testing.T) {
	v, err := exec.NullIfNotFound(5, core.ErrNotFound)
	require.Nil(t, v)
	require.NoError(t, err)
	v, err = exec.NullIfNotFound(5, chain.ErrUnknownHash)
	require.Nil(t, v)
	require.NoError(t, err)
	_, err = exec.NullIfNotFound(5, tests.ErrFake)
	require.ErrorIs(t, err, tests.ErrFake)
	v, err = exec.NullIfNotFound(5, nil)
	require.Equal(t, 5, v)
	require.NoError(t, err)
	require.Equal(t, exec.ErrHeaderNotFound, exec.HeaderNotFound(core.ErrNotFound))
	require.Equal(t, tests.ErrFake, exec.HeaderNotFound(tests.ErrFake))
	require.EqualError(t, exec.UnknownHashError(common.Hash{1}), "block with hash 0x0100000000000000000000000000000000000000000000000000000000000000 not found")
}

func TestSetDefaults_FillsWhatTheCallerLeftOut(t *testing.T) {
	r, env := newEnv(t)
	head := env.Latest(context.Background())
	r.Core.SetNonce(r.Wallet.Address, 4)
	args := exec.CallArgs{From: &r.Wallet.Address, To: &r.Recipient}
	require.NoError(t, env.SetDefaults(context.Background(), head, &args))
	require.Equal(t, hexutil.Uint64(4), *args.Nonce)
	require.Equal(t, hexutil.Uint64(21000), *args.Gas)
	require.Equal(t, int64(tests.ChainID), args.ChainID.ToInt().Int64())
	require.NotNil(t, args.MaxFeePerGas)
	require.NotNil(t, args.MaxPriorityFeePerGas)
	require.Zero(t, args.Value.ToInt().Sign())
	tx, err := args.ToTransaction()
	require.NoError(t, err)
	require.Equal(t, uint8(types.DynamicFeeTxType), tx.Type())
}

// TestSetDefaults_Refusals are the rows of V5.
func TestSetDefaults_Refusals(t *testing.T) {
	r, env := newEnv(t)
	head := env.Latest(context.Background())
	one := (*hexutil.Big)(big.NewInt(1))
	two := (*hexutil.Big)(big.NewInt(2))
	data, input := hexutil.Bytes{1}, hexutil.Bytes{2}
	gas := hexutil.Uint64(21000)
	other := (*hexutil.Big)(big.NewInt(1))
	tests := map[string]struct {
		args exec.CallArgs
		msg  string
	}{
		"both fee kinds":   {exec.CallArgs{GasPrice: one, MaxFeePerGas: one}, "both gasPrice and (maxFeePerGas or maxPriorityFeePerGas) specified"},
		"price with auths": {exec.CallArgs{GasPrice: one, AuthorizationList: []types.SetCodeAuthorization{{}}}, "both gasPrice and authorizationList specified"},
		"fee below tip":    {exec.CallArgs{MaxFeePerGas: one, MaxPriorityFeePerGas: two}, "maxFeePerGas (0x1) < maxPriorityFeePerGas (0x2)"},
		"data and input":   {exec.CallArgs{To: &r.Recipient, GasPrice: one, Data: &data, Input: &input}, `both "data" and "input" are set and not equal. Please use "input" to pass transaction call data`},
		"blob without to":  {exec.CallArgs{GasPrice: one, BlobHashes: []common.Hash{{}}}, `missing "to" in blob transaction`},
		"creation empty":   {exec.CallArgs{GasPrice: one}, "contract creation without any data provided"},
		"auths without to": {exec.CallArgs{MaxFeePerGas: two, MaxPriorityFeePerGas: one, Data: &data, AuthorizationList: []types.SetCodeAuthorization{{}}}, `authorizationList provided for contract creation, but "to" field is missing`},
		"another chain":    {exec.CallArgs{To: &r.Recipient, GasPrice: one, Gas: &gas, ChainID: (*hexutil.Big)(big.NewInt(1))}, "chainId does not match node's (have=1, want=4003)"},
		"estimate fails":   {exec.CallArgs{From: &r.Recipient, To: &r.Recipient, GasPrice: (*hexutil.Big)(big.NewInt(1e9)), Value: other}, "insufficient funds for transfer"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			args := tc.args
			require.EqualError(t, env.SetDefaults(context.Background(), head, &args), tc.msg)
		})
	}
}

func TestOverrides_ApplyToStateAndBlock(t *testing.T) {
	r, env := newEnv(t)
	st, err := env.StateAtNumber(env.Latest(context.Background()), 1)
	require.NoError(t, err)
	defer st.Release()
	nonce := hexutil.Uint64(9)
	code := hexutil.Bytes{1, 2}
	balance := (*hexutil.U256)(uint256.NewInt(77))
	storage := map[common.Hash]common.Hash{{1}: {2}}
	diff := map[common.Hash]common.Hash{{3}: {4}}
	override := exec.StateOverride{
		r.Recipient: {Nonce: &nonce, Code: &code, Balance: &balance, State: &storage},
		r.Writer:    {StateDiff: &diff},
	}
	require.NoError(t, override.Apply(st))
	require.Equal(t, uint64(9), st.GetNonce(r.Recipient))
	require.Equal(t, []byte{1, 2}, st.GetCode(r.Recipient))
	require.Equal(t, uint64(77), st.GetBalance(r.Recipient).Uint64())
	require.Equal(t, common.Hash{2}, st.GetState(r.Recipient, common.Hash{1}))
	require.Equal(t, common.Hash{4}, st.GetState(r.Writer, common.Hash{3}))
	var none *exec.StateOverride
	require.NoError(t, none.Apply(st))
	both := exec.StateOverride{r.Writer: {State: &storage, StateDiff: &diff}}
	require.EqualError(t, both.Apply(st), "account 0x0000000000000000000000000000000000002222 has both 'state' and 'stateDiff'")

	blockCtx := vm.BlockContext{}
	random := common.Hash{5}
	coinbase := common.Address{6}
	time, limit := hexutil.Uint64(7), hexutil.Uint64(8)
	(&exec.BlockOverrides{
		Number: (*hexutil.Big)(big.NewInt(1)), Difficulty: (*hexutil.Big)(big.NewInt(2)), Time: &time, GasLimit: &limit,
		Coinbase: &coinbase, Random: &random, BaseFee: (*hexutil.Big)(big.NewInt(3)), BlobBaseFee: (*hexutil.Big)(big.NewInt(4)),
	}).Apply(&blockCtx)
	require.Equal(t, int64(1), blockCtx.BlockNumber.Int64())
	require.Equal(t, int64(2), blockCtx.Difficulty.Int64())
	require.Equal(t, uint64(7), blockCtx.Time)
	require.Equal(t, uint64(8), blockCtx.GasLimit)
	require.Equal(t, coinbase, blockCtx.Coinbase)
	require.Equal(t, &random, blockCtx.Random)
	require.Equal(t, int64(3), blockCtx.BaseFee.Int64())
	require.Equal(t, int64(4), blockCtx.BlobBaseFee.Int64())
	(*exec.BlockOverrides)(nil).Apply(&blockCtx)
}

func TestCall_PreArgsRunFirstOnTheSameState(t *testing.T) {
	r, env := newEnv(t)
	head := env.Latest(context.Background())
	pre := []exec.CallArgs{{From: &r.Wallet.Address, To: &r.Writer}, {From: &r.Wallet.Address, GasPrice: (*hexutil.Big)(big.NewInt(1)), MaxFeePerGas: (*hexutil.Big)(big.NewInt(1))}}
	res, err := env.Call(context.Background(), head, exec.CallArgs{From: &r.Wallet.Address, To: &r.Returner}, chain.LatestRef(), nil, nil, 0, 0, pre)
	require.NoError(t, err)
	require.Equal(t, common.BigToHash(big.NewInt(42)).Bytes(), res.Return())
	_, err = env.Call(context.Background(), head, exec.CallArgs{}, chain.NumberRef(9), nil, nil, 0, 0, nil)
	require.ErrorIs(t, err, exec.ErrHeaderNotFound)
}

func TestReplay_StateAtTransaction(t *testing.T) {
	r, env := newEnv(t)
	head := env.Latest(context.Background())
	msg, st, err := env.StateAtTransaction(head, r.Block, 1)
	require.NoError(t, err)
	require.Equal(t, r.Writer, *msg.To)
	require.Equal(t, uint64(1), st.GetNonce(r.Wallet.Address), "the transfer before it ran")
	st.Release()

	_, st, err = env.StateAtTransaction(head, r.Block, 2)
	require.EqualError(t, err, "transaction index 2 out of range for block "+hexutil.Encode(r.Block.Hash[:]))
	require.Nil(t, st)
	_, _, err = env.StateAtTransaction(head, r.Block, 3)
	require.Error(t, err)
	genesis, err := r.Core.Block(0)
	require.NoError(t, err)
	_, _, err = env.StateAtTransaction(head, genesis, 0)
	require.EqualError(t, err, "no transaction in genesis")
	opened, released := r.Core.TrackStates()
	require.Equal(t, opened, released)

	_, err = env.TraceBlock(head, chain.BlockRef{})
	require.EqualError(t, err, "invalid arguments; neither block number nor hash specified")
	_, err = env.TraceBlock(head, chain.NumberRef(7))
	require.ErrorIs(t, err, exec.ErrHeaderNotFound)
	tx, msg, err := env.TxMessage(&exec.CallArgs{To: &r.Writer}, r.Block)
	require.NoError(t, err)
	require.Equal(t, uint8(types.LegacyTxType), tx.Type())
	require.Equal(t, r.Writer, *msg.To)
}

func TestSubmit_ChecksAndHandsOn(t *testing.T) {
	r, env := newEnv(t)
	tx := r.Wallet.Transfer(t, 2, r.Recipient, big.NewInt(1))
	hash, err := env.Submit(context.Background(), tx)
	require.NoError(t, err)
	require.Equal(t, tx.Hash(), hash)
	creation := r.Wallet.Sign(t, &types.DynamicFeeTx{ChainID: big.NewInt(tests.ChainID), Nonce: 3, Gas: 60_000, GasFeeCap: big.NewInt(1e12), Data: []byte{0}})
	_, err = env.Submit(context.Background(), creation)
	require.NoError(t, err)
	require.Len(t, r.Core.Sent(), 2)
	r.Core.SetSendError(errors.New("known transaction"))
	_, err = env.Submit(context.Background(), tx)
	require.EqualError(t, err, "known transaction")

	require.NoError(t, exec.CheckTxFee(big.NewInt(1e18), 1, 0))
	require.NoError(t, exec.CheckTxFee(big.NewInt(1e18), 1, 1))
	require.EqualError(t, exec.CheckTxFee(big.NewInt(1e18), 2, 1), "tx fee (2.00 FTM) exceeds the configured cap (1.00 FTM)")
}

func TestWithStateHooks_ReportsTheChanges(t *testing.T) {
	r, env := newEnv(t)
	st, err := env.StateAtNumber(env.Latest(context.Background()), 1)
	require.NoError(t, err)
	defer st.Release()
	var balances, nonces, codes, storage int
	hooked := exec.WithStateHooks(st, &tracing.Hooks{
		OnBalanceChange: func(common.Address, *big.Int, *big.Int, tracing.BalanceChangeReason) { balances++ },
		OnNonceChange:   func(common.Address, uint64, uint64) { nonces++ },
		OnCodeChange:    func(common.Address, common.Hash, []byte, common.Hash, []byte) { codes++ },
		OnStorageChange: func(common.Address, common.Hash, common.Hash, common.Hash) { storage++ },
	})
	hooked.AddBalance(r.Recipient, uint256.NewInt(1), tracing.BalanceChangeUnspecified)
	hooked.SubBalance(r.Wallet.Address, uint256.NewInt(1), tracing.BalanceChangeUnspecified)
	hooked.SetNonce(r.Recipient, 1, tracing.NonceChangeUnspecified)
	hooked.SetCode(r.Recipient, []byte{1}, tracing.CodeChangeUnspecified)
	hooked.SetState(r.Recipient, common.Hash{1}, common.Hash{2})
	require.Equal(t, 2, balances)
	require.Equal(t, 1, nonces)
	require.Equal(t, 1, codes)
	require.Equal(t, 1, storage)
}
