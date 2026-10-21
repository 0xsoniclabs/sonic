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
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
	"github.com/0xsoniclabs/sonic/rpc/oracle"
	"github.com/0xsoniclabs/sonic/rpc/tests"
)

func TestEnv_TxByHash(t *testing.T) {
	env, fc := newEnv(t)
	tx := tests.NewAccount(t, nil, nil).Transfer(t, 0, common.Address{1}, 1)
	fc.AddBlock(tx)
	require.Eventually(t, func() bool { return env.Chain.Latest() == 3 }, time.Second, time.Millisecond)
	head := env.Chain.Head()

	got, block, index, err := env.TxByHash(head, tx.Hash())
	require.NoError(t, err)
	require.Equal(t, tx.Hash(), got.Hash())
	require.EqualValues(t, 3, block)
	require.EqualValues(t, 0, index)

	t.Run("unknown", func(t *testing.T) {
		got, _, _, err := env.TxByHash(head, common.Hash{9})
		require.NoError(t, err)
		require.Nil(t, got)
	})

	t.Run("not served yet", func(t *testing.T) {
		behind := dispatch.Head{Number: 2}
		got, block, index, err := env.TxByHash(behind, tx.Hash())
		require.NoError(t, err)
		require.Nil(t, got)
		require.Zero(t, block)
		require.Zero(t, index)
	})

	t.Run("failure of the node", func(t *testing.T) {
		fc.Inject("TxByHash", tests.Fail)
		defer fc.Inject("TxByHash", tests.Normal)
		got, _, _, err := env.TxByHash(head, tx.Hash())
		require.ErrorIs(t, err, tests.ErrInjected)
		require.Nil(t, got)
	})
}

func TestEnv_HeaderAt(t *testing.T) {
	env, _ := newEnv(t)
	head := env.Chain.Head()

	shared, err := env.HeaderAt(head, 2)
	require.NoError(t, err)
	require.Same(t, head.Header, shared, "the header of the head is the shared one")

	older, err := env.HeaderAt(head, 1)
	require.NoError(t, err)
	require.EqualValues(t, 1, older.Number.Uint64())

	above, err := env.HeaderAt(head, 3)
	require.NoError(t, err)
	require.Nil(t, above)
}

func TestEnv_FindBlock(t *testing.T) {
	env, fc := newEnv(t)
	head := env.Chain.Head()

	for name, ref := range map[string]dispatch.BlockRef{
		"latest": dispatch.RefLatest(),
		"number": dispatch.RefNumber(2),
		"hash":   dispatch.RefHash(fc.BlockAt(2).Hash),
	} {
		block, err := env.FindBlock(head, ref)
		require.NoError(t, err, name)
		require.EqualValues(t, 2, block.NumberU64(), name)
	}

	for name, ref := range map[string]dispatch.BlockRef{
		"above the head": dispatch.RefNumber(3),
		"unknown hash":   dispatch.RefHash(common.Hash{9}),
		"invalid tag":    dispatch.RefNumber(-9),
	} {
		block, err := env.FindBlock(head, ref)
		require.NoError(t, err, name)
		require.Nil(t, block, name)

		_, err = env.BlockByRef(head, ref)
		require.ErrorIs(t, err, exec.ErrHeaderNotFound, name)
	}

	_, err := env.FindBlock(head, dispatch.BlockRef{})
	require.ErrorIs(t, err, exec.ErrNoBlockSelector)
	_, err = env.BlockByRef(head, dispatch.BlockRef{})
	require.ErrorIs(t, err, exec.ErrNoBlockSelector)
}

func TestEnv_StateAt(t *testing.T) {
	fc, _ := tests.NewCarmenCore(t, tests.WithBlocks(2))
	d := dispatch.New(dispatch.DefaultConfig(), dispatch.NewMetrics(nil))
	chain := dispatch.NewChain(fc, dispatch.DefaultConfig(), d.Metrics())
	require.NoError(t, chain.Start(d))
	t.Cleanup(func() { d.Stop(0) })
	env := exec.NewEnv(chain, oracle.New(oracle.Config{}, fc))
	head := env.Chain.Head()

	t.Run("head", func(t *testing.T) {
		st, header, err := env.StateAt(head, dispatch.RefLatest())
		require.NoError(t, err)
		defer st.Release()
		require.Equal(t, head.Header, header)
		require.NotSame(t, head.Header, header, "the caller owns the header")
	})

	t.Run("older block by hash", func(t *testing.T) {
		st, header, err := env.StateAt(head, dispatch.RefHash(fc.BlockAt(1).Hash))
		require.NoError(t, err)
		defer st.Release()
		require.EqualValues(t, 1, header.Number.Uint64())
	})

	t.Run("no block", func(t *testing.T) {
		for name, ref := range map[string]dispatch.BlockRef{
			"above the head": dispatch.RefNumber(3),
			"unknown hash":   dispatch.RefHash(common.Hash{9}),
			"invalid tag":    dispatch.RefNumber(-9),
		} {
			_, _, err := env.StateAt(head, ref)
			require.ErrorIs(t, err, exec.ErrHeaderNotFound, name)
		}
	})

	t.Run("no selector", func(t *testing.T) {
		_, _, err := env.StateAt(head, dispatch.BlockRef{})
		require.ErrorIs(t, err, dispatch.ErrNoSelector)
	})
}

func TestEnv_OpenStateReadsNoHeader(t *testing.T) {
	fc, _ := tests.NewCarmenCore(t, tests.WithBlocks(2))
	d := dispatch.New(dispatch.DefaultConfig(), dispatch.NewMetrics(nil))
	chain := dispatch.NewChain(fc, dispatch.DefaultConfig(), d.Metrics())
	require.NoError(t, chain.Start(d))
	t.Cleanup(func() { d.Stop(0) })
	env := exec.NewEnv(chain, oracle.New(oracle.Config{}, fc))
	head := env.Chain.Head()

	fc.Inject("Header", tests.Fail)
	before := fc.Calls("Header")
	for name, ref := range map[string]dispatch.BlockRef{
		"latest": dispatch.RefLatest(),
		"number": dispatch.RefNumber(1),
		"hash":   dispatch.RefHash(fc.BlockAt(1).Hash),
	} {
		st, err := env.OpenState(head, ref)
		require.NoError(t, err, name)
		st.Release()
	}
	require.Equal(t, before, fc.Calls("Header"), "the header is not read")

	_, _, err := env.StateAt(head, dispatch.RefNumber(1))
	require.ErrorIs(t, err, tests.ErrInjected, "StateAt does read it")

	for name, ref := range map[string]dispatch.BlockRef{
		"above the head": dispatch.RefNumber(3),
		"unknown hash":   dispatch.RefHash(common.Hash{9}),
		"invalid tag":    dispatch.RefNumber(-9),
	} {
		_, err := env.OpenState(head, ref)
		require.ErrorIs(t, err, exec.ErrHeaderNotFound, name)
	}
	_, err = env.OpenState(head, dispatch.BlockRef{})
	require.ErrorIs(t, err, dispatch.ErrNoSelector)
}
