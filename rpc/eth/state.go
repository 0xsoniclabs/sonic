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
	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// maxGetProofKeys bounds the storage keys of one eth_getProof: each takes a
// walk of the trie.
const maxGetProofKeys = 1024

type accountParams struct {
	Address common.Address
	Block   *chain.BlockReference
}

type storageParams struct {
	Address common.Address
	Key     string
	Block   *chain.BlockReference
}

type proofParams struct {
	Address     common.Address
	StorageKeys []string
	Block       *chain.BlockReference
}

// orLatest is the block a state read names, the head if it names none.
func orLatest(reference *chain.BlockReference) chain.BlockReference {
	if reference == nil {
		return chain.ReferenceTo(chain.LatestBlockNumber)
	}
	return *reference
}

func (h *handlers) getBalance(ctx context.Context, params accountParams) (*hexutil.U256, error) {
	stateDB, err := h.environment.OpenState(h.environment.Head(), orLatest(params.Block))
	if err != nil {
		return nil, err
	}
	defer stateDB.Release()
	return (*hexutil.U256)(stateDB.GetBalance(params.Address)), stateDB.Error()
}

func (h *handlers) getCode(ctx context.Context, params accountParams) (hexutil.Bytes, error) {
	stateDB, err := h.environment.OpenState(h.environment.Head(), orLatest(params.Block))
	if err != nil {
		return nil, err
	}
	defer stateDB.Release()
	return stateDB.GetCode(params.Address), stateDB.Error()
}

// getStorageAt takes the key as hex of any length, as the legacy handler
// does.
func (h *handlers) getStorageAt(ctx context.Context, params storageParams) (hexutil.Bytes, error) {
	stateDB, err := h.environment.OpenState(h.environment.Head(), orLatest(params.Block))
	if err != nil {
		return nil, err
	}
	defer stateDB.Release()
	value := stateDB.GetState(params.Address, common.HexToHash(params.Key))
	return value[:], stateDB.Error()
}

// getTransactionCount answers the pool's nonce for the pending block, and
// otherwise the nonce in the state of the block.
func (h *handlers) getTransactionCount(ctx context.Context, params accountParams) (*hexutil.Uint64, error) {
	reference := orLatest(params.Block)
	if reference.Number != nil && *reference.Number == chain.PendingBlockNumber {
		nonce := h.node.PoolNonce(params.Address)
		return (*hexutil.Uint64)(&nonce), nil
	}
	stateDB, err := h.environment.OpenState(h.environment.Head(), reference)
	if err != nil {
		return nil, err
	}
	defer stateDB.Release()
	nonce := stateDB.GetNonce(params.Address)
	return (*hexutil.Uint64)(&nonce), stateDB.Error()
}

type accountResult struct {
	CodeHash    common.Hash    `json:"codeHash"`
	StorageRoot common.Hash    `json:"storageRoot"`
	Balance     *hexutil.U256  `json:"balance"`
	Nonce       hexutil.Uint64 `json:"nonce"`
}

// getAccount answers the fields of an account, read from a witness proof of
// the state.
func (h *handlers) getAccount(ctx context.Context, params accountParams) (*accountResult, error) {
	stateDB, header, err := h.environment.StateAt(h.environment.Head(), orLatest(params.Block))
	if err != nil {
		return nil, err
	}
	defer stateDB.Release()
	proof, err := stateDB.GetProof(params.Address, nil)
	if err != nil {
		return nil, err
	}
	root, address := cc.Hash(header.Root), cc.Address(params.Address)
	codeHash, _, err := proof.GetCodeHash(root, address)
	if err != nil {
		return nil, err
	}
	_, storageRoot, _ := proof.GetAccountElements(root, address)
	balance, _, err := proof.GetBalance(root, address)
	if err != nil {
		return nil, err
	}
	nonce, _, err := proof.GetNonce(root, address)
	if err != nil {
		return nil, err
	}
	balanceU256 := balance.Uint256()
	return &accountResult{
		CodeHash:    common.Hash(codeHash),
		StorageRoot: common.Hash(storageRoot),
		Balance:     (*hexutil.U256)(&balanceU256),
		Nonce:       hexutil.Uint64(nonce.ToUint64()),
	}, stateDB.Error()
}

type proofResult struct {
	Address      common.Address  `json:"address"`
	AccountProof []string        `json:"accountProof"`
	Balance      *hexutil.U256   `json:"balance"`
	CodeHash     common.Hash     `json:"codeHash"`
	Nonce        hexutil.Uint64  `json:"nonce"`
	StorageHash  common.Hash     `json:"storageHash"`
	StorageProof []storageResult `json:"storageProof"`
}

type storageResult struct {
	Key   string       `json:"key"`
	Value *hexutil.Big `json:"value"`
	Proof []string     `json:"proof"`
}

// getProof answers the Merkle proof of an account and of storage keys. The
// keys are bounded and decoded before any state is read.
func (h *handlers) getProof(ctx context.Context, params proofParams) (*proofResult, error) {
	if len(params.StorageKeys) > maxGetProofKeys {
		return nil, dispatch.InvalidParams(fmt.Sprintf("too many storage keys requested (max %d, got %d)", maxGetProofKeys, len(params.StorageKeys)))
	}
	keys := make([]common.Hash, len(params.StorageKeys))
	for i, key := range params.StorageKeys {
		decoded, err := decodeStorageKey(key)
		if err != nil {
			return nil, dispatch.InvalidParams(fmt.Sprintf("%v: %q", err, key))
		}
		keys[i] = decoded
	}
	stateDB, header, err := h.environment.StateAt(h.environment.Head(), orLatest(params.Block))
	if err != nil {
		return nil, err
	}
	defer stateDB.Release()
	proof, err := stateDB.GetProof(params.Address, keys)
	if err != nil {
		return nil, fmt.Errorf("failed to generate proof: %w", err)
	}
	root, address := cc.Hash(header.Root), cc.Address(params.Address)
	storageProof := make([]storageResult, len(keys))
	for i, key := range keys {
		value, _, err := proof.GetState(root, address, cc.Key(key))
		if err != nil {
			return nil, err
		}
		elements, _ := proof.GetStorageElements(root, address, cc.Key(key))
		storageProof[i] = storageResult{Key: key.Hex(), Value: (*hexutil.Big)(new(big.Int).SetBytes(value[:])), Proof: hexStrings(elements)}
	}
	accountProof, storageHash, _ := proof.GetAccountElements(root, address)
	codeHash, _, err := proof.GetCodeHash(root, address)
	if err != nil {
		return nil, err
	}
	balance, _, err := proof.GetBalance(root, address)
	if err != nil {
		return nil, err
	}
	nonce, _, err := proof.GetNonce(root, address)
	if err != nil {
		return nil, err
	}
	balanceU256 := balance.Uint256()
	return &proofResult{
		Address:      params.Address,
		AccountProof: hexStrings(accountProof),
		Balance:      (*hexutil.U256)(&balanceU256),
		CodeHash:     common.Hash(codeHash),
		Nonce:        hexutil.Uint64(nonce.ToUint64()),
		StorageHash:  common.Hash(storageHash),
		StorageProof: storageProof,
	}, stateDB.Error()
}

// decodeStorageKey decodes a hex key of at most 32 bytes, with or without
// prefix, of an odd number of digits too, left-padded with zeros.
func decodeStorageKey(key string) (common.Hash, error) {
	if strings.HasPrefix(key, "0x") || strings.HasPrefix(key, "0X") {
		key = key[2:]
	}
	if len(key)%2 == 1 {
		key = "0" + key
	}
	if len(key) > 2*common.HashLength {
		return common.Hash{}, fmt.Errorf("storage key too long (want at most %d bytes)", common.HashLength)
	}
	decoded, err := hex.DecodeString(key)
	if err != nil {
		return common.Hash{}, errors.New("invalid hex in storage key")
	}
	return common.BytesToHash(decoded), nil
}

func hexStrings(elements []immutable.Bytes) []string {
	encoded := make([]string, len(elements))
	for i, element := range elements {
		encoded[i] = hexutil.Encode(element.ToBytes())
	}
	return encoded
}
