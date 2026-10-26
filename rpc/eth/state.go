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

package eth

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"

	cc "github.com/0xsoniclabs/carmen/go/common"
	"github.com/0xsoniclabs/carmen/go/common/immutable"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
)

// maxGetProofKeys bounds the storage keys of one eth_getProof; every key is a
// trie walk of its own.
const maxGetProofKeys = 1024

type addressAtBlockArgs struct {
	Address common.Address  `rpc:"address"`
	Block   *chain.BlockRef `rpc:"block"`
}

type storageAtArgs struct {
	Address common.Address  `rpc:"address"`
	Key     string          `rpc:"key"`
	Block   *chain.BlockRef `rpc:"block"`
}

type proofArgs struct {
	Address     common.Address  `rpc:"address"`
	StorageKeys []string        `rpc:"storageKeys"`
	Block       *chain.BlockRef `rpc:"block"`
}

// withState opens the archive at the selected block, latest if none is
// selected, runs read on it and releases it. A missing block is "header not
// found", as for every state getter of the reference.
func withState[T any](ctx context.Context, a *api, ref *chain.BlockRef, read func(st state.StateDB) (T, error)) (any, error) {
	st, err := a.env.OpenState(a.env.Latest(ctx), chain.OrLatest(ref))
	if err != nil {
		return nil, exec.HeaderNotFound(err)
	}
	defer st.Release()
	return read(st)
}

func (a *api) getBalance(ctx context.Context, p addressAtBlockArgs) (any, error) {
	return withState(ctx, a, p.Block, func(st state.StateDB) (*hexutil.U256, error) {
		return (*hexutil.U256)(st.GetBalance(p.Address)), nil
	})
}

func (a *api) getCode(ctx context.Context, p addressAtBlockArgs) (any, error) {
	return withState(ctx, a, p.Block, func(st state.StateDB) (hexutil.Bytes, error) {
		return st.GetCode(p.Address), nil
	})
}

func (a *api) getStorageAt(ctx context.Context, p storageAtArgs) (any, error) {
	return withState(ctx, a, p.Block, func(st state.StateDB) (hexutil.Bytes, error) {
		value := st.GetState(p.Address, common.HexToHash(p.Key))
		return value[:], nil
	})
}

// getTransactionCount answers the nonce of an account at a block; at pending,
// the pool's next nonce, which counts the account's pending transactions.
func (a *api) getTransactionCount(ctx context.Context, p addressAtBlockArgs) (any, error) {
	if chain.OrLatest(p.Block).IsPending() {
		nonce := a.pool.PoolNonce(p.Address)
		return (*hexutil.Uint64)(&nonce), nil
	}
	return withState(ctx, a, p.Block, func(st state.StateDB) (*hexutil.Uint64, error) {
		nonce := st.GetNonce(p.Address)
		return (*hexutil.Uint64)(&nonce), nil
	})
}

// GetAccountResult is the answer of eth_getAccount.
type GetAccountResult struct {
	CodeHash    common.Hash    `json:"codeHash"`
	StorageRoot common.Hash    `json:"storageRoot"`
	Balance     *hexutil.U256  `json:"balance"`
	Nonce       hexutil.Uint64 `json:"nonce"`
}

func (a *api) getAccount(ctx context.Context, p addressAtBlockArgs) (any, error) {
	st, header, err := a.env.StateAt(a.env.Latest(ctx), chain.OrLatest(p.Block))
	if err != nil {
		return nil, exec.HeaderNotFound(err)
	}
	defer st.Release()
	proof, err := st.GetProof(p.Address, nil)
	if err != nil {
		return nil, err
	}
	root, addr := cc.Hash(header.Root), cc.Address(p.Address)
	codeHash, _, err := proof.GetCodeHash(root, addr)
	if err != nil {
		return nil, err
	}
	_, storageRoot, _ := proof.GetAccountElements(root, addr)
	balance, _, err := proof.GetBalance(root, addr)
	if err != nil {
		return nil, err
	}
	nonce, _, err := proof.GetNonce(root, addr)
	if err != nil {
		return nil, err
	}
	u256Balance := balance.Uint256()
	return &GetAccountResult{
		CodeHash:    common.Hash(codeHash),
		StorageRoot: common.Hash(storageRoot),
		Balance:     (*hexutil.U256)(&u256Balance),
		Nonce:       hexutil.Uint64(nonce.ToUint64()),
	}, nil
}

// AccountResult is the answer of eth_getProof.
type AccountResult struct {
	Address      common.Address  `json:"address"`
	AccountProof []string        `json:"accountProof"`
	Balance      *hexutil.U256   `json:"balance"`
	CodeHash     common.Hash     `json:"codeHash"`
	Nonce        hexutil.Uint64  `json:"nonce"`
	StorageHash  common.Hash     `json:"storageHash"`
	StorageProof []StorageResult `json:"storageProof"`
}

// StorageResult is the proof of one storage slot.
type StorageResult struct {
	Key   string       `json:"key"`
	Value *hexutil.Big `json:"value"`
	Proof []string     `json:"proof"`
}

// getProof answers the witness of an account and some of its slots. The
// number of keys and their encoding are checked before any state is read.
func (a *api) getProof(ctx context.Context, p proofArgs) (any, error) {
	if len(p.StorageKeys) > maxGetProofKeys {
		return nil, dispatch.InvalidParams("too many storage keys requested (max %d, got %d)", maxGetProofKeys, len(p.StorageKeys))
	}
	keys := make([]common.Hash, len(p.StorageKeys))
	for i, key := range p.StorageKeys {
		decoded, err := decodeStorageKey(key)
		if err != nil {
			return nil, dispatch.InvalidParams("%v: %q", err, key)
		}
		keys[i] = decoded
	}

	st, header, err := a.env.StateAt(a.env.Latest(ctx), chain.OrLatest(p.Block))
	if err != nil {
		return nil, exec.HeaderNotFound(err)
	}
	defer st.Release()

	proof, err := st.GetProof(p.Address, keys)
	if err != nil {
		return nil, fmt.Errorf("failed to generate proof: %w", err)
	}
	root, addr := cc.Hash(header.Root), cc.Address(p.Address)
	storageProof := make([]StorageResult, len(keys))
	for i, key := range keys {
		value, _, err := proof.GetState(root, addr, cc.Key(key))
		if err != nil {
			return nil, err
		}
		elements, _ := proof.GetStorageElements(root, addr, cc.Key(key))
		storageProof[i] = StorageResult{
			Key:   key.Hex(),
			Value: (*hexutil.Big)(new(big.Int).SetBytes(value[:])),
			Proof: toHexSlice(elements),
		}
	}
	accountProof, storageHash, _ := proof.GetAccountElements(root, addr)
	codeHash, _, err := proof.GetCodeHash(root, addr)
	if err != nil {
		return nil, err
	}
	balance, _, err := proof.GetBalance(root, addr)
	if err != nil {
		return nil, err
	}
	nonce, _, err := proof.GetNonce(root, addr)
	if err != nil {
		return nil, err
	}
	u256Balance := balance.Uint256()
	return &AccountResult{
		Address:      p.Address,
		AccountProof: toHexSlice(accountProof),
		Balance:      (*hexutil.U256)(&u256Balance),
		CodeHash:     common.Hash(codeHash),
		Nonce:        hexutil.Uint64(nonce.ToUint64()),
		StorageHash:  common.Hash(storageHash),
		StorageProof: storageProof,
	}, nil
}

// decodeStorageKey parses a hex storage key of at most 32 bytes; shorter keys
// are left-padded and an odd number of digits is accepted.
func decodeStorageKey(s string) (common.Hash, error) {
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		s = s[2:]
	}
	if len(s)%2 == 1 {
		s = "0" + s
	}
	if len(s) > 2*common.HashLength {
		return common.Hash{}, fmt.Errorf("storage key too long (want at most %d bytes)", common.HashLength)
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return common.Hash{}, errors.New("invalid hex in storage key")
	}
	return common.BytesToHash(b), nil
}

func toHexSlice(b []immutable.Bytes) []string {
	r := make([]string, len(b))
	for i := range b {
		r[i] = hexutil.Encode(b[i].ToBytes())
	}
	return r
}
