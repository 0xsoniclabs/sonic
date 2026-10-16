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

	"github.com/0xsoniclabs/sonic/rpc/dispatch"
)

// maxGetProofKeys is the maximum number of storage keys accepted by a single
// eth_getProof request. Every key requires a separate trie walk.
const maxGetProofKeys = 1024

func registerState(d *dispatch.Dispatcher, a *API, m func(string) string) {
	dispatch.Add2(d, m("getBalance"), a.getBalance, "address", "block?")
	dispatch.Add2(d, m("getCode"), a.getCode, "address", "block?")
	dispatch.Add3(d, m("getStorageAt"), a.getStorageAt, "address", "key", "block?")
	dispatch.Add2(d, m("getTransactionCount"), a.getTransactionCount, "address", "block?")
	dispatch.Add2(d, m("getAccount"), a.getAccount, "address", "block?")
	dispatch.Add3(d, m("getProof"), a.getProof, "address", "storageKeys", "block?")
}

// getBalance returns the amount of wei for the address in the state of the
// block. When the block parameter is omitted, it defaults to the latest block.
func (a *API) getBalance(_ context.Context, address common.Address, ref *dispatch.BlockRef) (*hexutil.U256, error) {
	statedb, _, err := a.env.StateAt(a.chain.Head(), dispatch.OrLatest(ref))
	if err != nil {
		return nil, err
	}
	defer statedb.Release()
	return (*hexutil.U256)(statedb.GetBalance(address)), statedb.Error()
}

// getCode returns the code stored at the address in the state of the block.
func (a *API) getCode(_ context.Context, address common.Address, ref *dispatch.BlockRef) (hexutil.Bytes, error) {
	statedb, _, err := a.env.StateAt(a.chain.Head(), dispatch.OrLatest(ref))
	if err != nil {
		return nil, err
	}
	defer statedb.Release()
	code := statedb.GetCode(address)
	return code, statedb.Error()
}

// getStorageAt returns the storage at the address and key in the state of the block.
func (a *API) getStorageAt(_ context.Context, address common.Address, key string, ref *dispatch.BlockRef) (hexutil.Bytes, error) {
	statedb, _, err := a.env.StateAt(a.chain.Head(), dispatch.OrLatest(ref))
	if err != nil {
		return nil, err
	}
	defer statedb.Release()
	res := statedb.GetState(address, common.HexToHash(key))
	return res[:], statedb.Error()
}

// getTransactionCount returns the number of transactions the address has sent
// as of the block. The pending block asks the pool, which includes pending
// transactions.
func (a *API) getTransactionCount(_ context.Context, address common.Address, ref *dispatch.BlockRef) (*hexutil.Uint64, error) {
	resolved := dispatch.OrLatest(ref)
	if resolved.Number != nil && *resolved.Number == dispatch.PendingBlockNumber {
		nonce := a.core.PoolNonce(address)
		return (*hexutil.Uint64)(&nonce), nil
	}
	statedb, _, err := a.env.StateAt(a.chain.Head(), resolved)
	if err != nil {
		return nil, err
	}
	defer statedb.Release()
	nonce := statedb.GetNonce(address)
	return (*hexutil.Uint64)(&nonce), statedb.Error()
}

// GetAccountResult is the result of eth_getAccount.
type GetAccountResult struct {
	CodeHash    common.Hash    `json:"codeHash"`
	StorageRoot common.Hash    `json:"storageRoot"`
	Balance     *hexutil.U256  `json:"balance"`
	Nonce       hexutil.Uint64 `json:"nonce"`
}

// getAccount returns the information about the account in the state of the block.
func (a *API) getAccount(_ context.Context, address common.Address, ref *dispatch.BlockRef) (*GetAccountResult, error) {
	statedb, header, err := a.env.StateAt(a.chain.Head(), dispatch.OrLatest(ref))
	if err != nil {
		return nil, err
	}
	defer statedb.Release()
	proof, err := statedb.GetProof(address, nil)
	if err != nil {
		return nil, err
	}
	if proof == nil {
		return nil, errors.New("no proof available")
	}
	root := cc.Hash(header.Root)
	codeHash, _, err := proof.GetCodeHash(root, cc.Address(address))
	if err != nil {
		return nil, err
	}
	_, storageRoot, _ := proof.GetAccountElements(root, cc.Address(address))
	balance, _, err := proof.GetBalance(root, cc.Address(address))
	if err != nil {
		return nil, err
	}
	nonce, _, err := proof.GetNonce(root, cc.Address(address))
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

// AccountResult is the result of eth_getProof.
type AccountResult struct {
	Address      common.Address  `json:"address"`
	AccountProof []string        `json:"accountProof"`
	Balance      *hexutil.U256   `json:"balance"`
	CodeHash     common.Hash     `json:"codeHash"`
	Nonce        hexutil.Uint64  `json:"nonce"`
	StorageHash  common.Hash     `json:"storageHash"`
	StorageProof []StorageResult `json:"storageProof"`
}

// StorageResult is the storage part of the result of eth_getProof.
type StorageResult struct {
	Key   string       `json:"key"`
	Value *hexutil.Big `json:"value"`
	Proof []string     `json:"proof"`
}

// getProof returns the Merkle-proof for an account and optionally some storage
// keys. At most 1024 keys are accepted and all are decoded before any state is
// read.
func (a *API) getProof(_ context.Context, address common.Address, storageKeys []string, ref *dispatch.BlockRef) (*AccountResult, error) {
	if len(storageKeys) > maxGetProofKeys {
		return nil, invalidParamsError(fmt.Sprintf("too many storage keys requested (max %d, got %d)", maxGetProofKeys, len(storageKeys)))
	}
	// Decode all keys up front so invalid input is rejected before any state access.
	keys := make([]common.Hash, len(storageKeys))
	for i, key := range storageKeys {
		decoded, err := decodeStorageKey(key)
		if err != nil {
			return nil, invalidParamsError(fmt.Sprintf("%v: %q", err, key))
		}
		keys[i] = decoded
	}

	statedb, header, err := a.env.StateAt(a.chain.Head(), dispatch.OrLatest(ref))
	if err != nil {
		return nil, err
	}
	defer statedb.Release()

	proof, err := statedb.GetProof(address, keys)
	if err != nil {
		return nil, fmt.Errorf("failed to generate proof: %w", err)
	}
	if proof == nil {
		return nil, errors.New("failed to generate proof: no proof available")
	}

	root := cc.Hash(header.Root)
	storageProof := make([]StorageResult, len(keys))
	for i, key := range keys {
		value, _, err := proof.GetState(root, cc.Address(address), cc.Key(key))
		if err != nil {
			return nil, err
		}
		elements, _ := proof.GetStorageElements(root, cc.Address(address), cc.Key(keys[i]))
		storageProof[i] = StorageResult{
			Key:   key.Hex(),
			Value: (*hexutil.Big)(new(big.Int).SetBytes(value[:])),
			Proof: toHexSlice(elements),
		}
	}

	accountProof, storageHash, _ := proof.GetAccountElements(root, cc.Address(address))

	codeHash, _, err := proof.GetCodeHash(root, cc.Address(address))
	if err != nil {
		return nil, err
	}
	balance, _, err := proof.GetBalance(root, cc.Address(address))
	if err != nil {
		return nil, err
	}
	nonce, _, err := proof.GetNonce(root, cc.Address(address))
	if err != nil {
		return nil, err
	}
	u256Balance := balance.Uint256()
	return &AccountResult{
		Address:      address,
		AccountProof: toHexSlice(accountProof),
		Balance:      (*hexutil.U256)(&u256Balance),
		CodeHash:     common.Hash(codeHash),
		Nonce:        hexutil.Uint64(nonce.ToUint64()),
		StorageHash:  common.Hash(storageHash),
		StorageProof: storageProof,
	}, nil
}

// decodeStorageKey parses a hex encoded storage key of at most 32 bytes.
// Shorter keys are left-padded with zeros; an odd number of hex digits is
// accepted. Invalid hex or keys longer than 32 bytes are rejected.
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
