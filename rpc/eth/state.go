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

	carmen "github.com/0xsoniclabs/carmen/go/common"
	"github.com/0xsoniclabs/carmen/go/common/immutable"
	"github.com/0xsoniclabs/carmen/go/common/witness"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/dispatch"
	"github.com/0xsoniclabs/sonic/rpc/exec"
)

// maxProofKeys bounds the storage keys of one eth_getProof (V10).
const maxProofKeys = 1024

// valueOr returns an optional argument, or the given value when the client
// sent none.
func valueOr[Value any](arguments dispatch.Arguments, position int, absent Value) Value {
	if value := dispatch.Argument[*Value](arguments, position); value != nil {
		return *value
	}
	return absent
}

// findState opens the state of the block an optional argument names, the
// head's when there is none. The caller releases it.
func (handler *handlers) findState(ctx context.Context, arguments dispatch.Arguments, position int) (state.StateDB, error) {
	head, err := handler.environment.Latest(ctx)
	if err != nil {
		return nil, err
	}
	statedb, err := handler.environment.FindState(head, valueOr(arguments, position, chain.Latest()))
	return statedb, exec.StateLookupError(err)
}

// stateAt opens the state of the block a reference names with the block's
// header. The caller releases the state.
func (handler *handlers) stateAt(ctx context.Context, reference chain.BlockReference) (exec.Head, state.StateDB, *evmcore.EvmHeader, error) {
	head, err := handler.environment.Latest(ctx)
	if err != nil {
		return exec.Head{}, nil, nil, err
	}
	statedb, header, err := handler.environment.StateAt(head, reference)
	return head, statedb, header, exec.StateLookupError(err)
}

func (handler *handlers) getBalance(ctx context.Context, arguments dispatch.Arguments) (any, error) {
	statedb, err := handler.findState(ctx, arguments, 1)
	if err != nil {
		return nil, err
	}
	defer statedb.Release()
	return (*hexutil.U256)(statedb.GetBalance(dispatch.Argument[common.Address](arguments, 0))), statedb.Error()
}

func (handler *handlers) getCode(ctx context.Context, arguments dispatch.Arguments) (any, error) {
	statedb, err := handler.findState(ctx, arguments, 1)
	if err != nil {
		return nil, err
	}
	defer statedb.Release()
	return hexutil.Bytes(statedb.GetCode(dispatch.Argument[common.Address](arguments, 0))), statedb.Error()
}

// getStorageAt reads a slot whose key it converts leniently, as the
// reference did: the key is not checked.
func (handler *handlers) getStorageAt(ctx context.Context, arguments dispatch.Arguments) (any, error) {
	statedb, err := handler.findState(ctx, arguments, 2)
	if err != nil {
		return nil, err
	}
	defer statedb.Release()
	address := dispatch.Argument[common.Address](arguments, 0)
	value := statedb.GetState(address, common.HexToHash(dispatch.Argument[string](arguments, 1)))
	return hexutil.Bytes(value[:]), statedb.Error()
}

// getTransactionCount answers the nonce of the pool for pending, which
// counts the pending transactions of the account (I5).
func (handler *handlers) getTransactionCount(ctx context.Context, arguments dispatch.Arguments) (any, error) {
	address := dispatch.Argument[common.Address](arguments, 0)
	reference := valueOr(arguments, 1, chain.Latest())
	if reference.Number != nil && *reference.Number == chain.PendingBlockNumber {
		nonce := handler.node.PoolNonce(address)
		return (*hexutil.Uint64)(&nonce), nil
	}
	statedb, err := handler.findState(ctx, arguments, 1)
	if err != nil {
		return nil, err
	}
	defer statedb.Release()
	nonce := statedb.GetNonce(address)
	return (*hexutil.Uint64)(&nonce), statedb.Error()
}

// accountResult is the answer of eth_getAccount.
type accountResult struct {
	CodeHash    common.Hash    `json:"codeHash"`
	StorageRoot common.Hash    `json:"storageRoot"`
	Balance     *hexutil.U256  `json:"balance"`
	Nonce       hexutil.Uint64 `json:"nonce"`
}

func (handler *handlers) getAccount(ctx context.Context, arguments dispatch.Arguments) (any, error) {
	_, statedb, header, err := handler.stateAt(ctx, valueOr(arguments, 1, chain.Latest()))
	if err != nil {
		return nil, err
	}
	defer statedb.Release()
	address := dispatch.Argument[common.Address](arguments, 0)
	proof, err := statedb.GetProof(address, nil)
	if err != nil {
		return nil, err
	}
	fields, err := readAccount(proof, carmen.Hash(header.Root), carmen.Address(address))
	if err != nil {
		return nil, err
	}
	return &accountResult{
		CodeHash:    fields.codeHash,
		StorageRoot: fields.storageRoot,
		Balance:     fields.balance,
		Nonce:       fields.nonce,
	}, statedb.Error()
}

// accountFields are the fields of an account a witness proof proves.
type accountFields struct {
	elements    []string
	storageRoot common.Hash
	codeHash    common.Hash
	balance     *hexutil.U256
	nonce       hexutil.Uint64
}

func readAccount(proof witness.Proof, root carmen.Hash, address carmen.Address) (accountFields, error) {
	elements, storageRoot, _ := proof.GetAccountElements(root, address)
	codeHash, _, err := proof.GetCodeHash(root, address)
	if err != nil {
		return accountFields{}, err
	}
	balance, _, err := proof.GetBalance(root, address)
	if err != nil {
		return accountFields{}, err
	}
	nonce, _, err := proof.GetNonce(root, address)
	if err != nil {
		return accountFields{}, err
	}
	amount := balance.Uint256()
	return accountFields{
		elements:    hexStrings(elements),
		storageRoot: common.Hash(storageRoot),
		codeHash:    common.Hash(codeHash),
		balance:     (*hexutil.U256)(&amount),
		nonce:       hexutil.Uint64(nonce.ToUint64()),
	}, nil
}

// proofResult is the answer of eth_getProof.
type proofResult struct {
	Address      common.Address `json:"address"`
	AccountProof []string       `json:"accountProof"`
	Balance      *hexutil.U256  `json:"balance"`
	CodeHash     common.Hash    `json:"codeHash"`
	Nonce        hexutil.Uint64 `json:"nonce"`
	StorageHash  common.Hash    `json:"storageHash"`
	StorageProof []storageProof `json:"storageProof"`
}

type storageProof struct {
	Key   string       `json:"key"`
	Value *hexutil.Big `json:"value"`
	Proof []string     `json:"proof"`
}

// getProof checks and decodes every key before it reads any state (V10).
func (handler *handlers) getProof(ctx context.Context, arguments dispatch.Arguments) (any, error) {
	keys, err := decodeStorageKeys(dispatch.Argument[[]string](arguments, 1))
	if err != nil {
		return nil, err
	}
	_, statedb, header, err := handler.stateAt(ctx, valueOr(arguments, 2, chain.Latest()))
	if err != nil {
		return nil, err
	}
	defer statedb.Release()
	address := dispatch.Argument[common.Address](arguments, 0)
	proof, err := statedb.GetProof(address, keys)
	if err != nil {
		return nil, fmt.Errorf("failed to generate proof: %w", err)
	}
	root := carmen.Hash(header.Root)
	slots, err := storageProofs(proof, root, carmen.Address(address), keys)
	if err != nil {
		return nil, err
	}
	fields, err := readAccount(proof, root, carmen.Address(address))
	if err != nil {
		return nil, err
	}
	return &proofResult{
		Address:      address,
		AccountProof: fields.elements,
		Balance:      fields.balance,
		CodeHash:     fields.codeHash,
		Nonce:        fields.nonce,
		StorageHash:  fields.storageRoot,
		StorageProof: slots,
	}, statedb.Error()
}

func storageProofs(proof witness.Proof, root carmen.Hash, address carmen.Address, keys []common.Hash) ([]storageProof, error) {
	slots := make([]storageProof, len(keys))
	for index, key := range keys {
		value, _, err := proof.GetState(root, address, carmen.Key(key))
		if err != nil {
			return nil, err
		}
		elements, _ := proof.GetStorageElements(root, address, carmen.Key(key))
		slots[index] = storageProof{
			Key:   key.Hex(),
			Value: (*hexutil.Big)(new(big.Int).SetBytes(value[:])),
			Proof: hexStrings(elements),
		}
	}
	return slots, nil
}

func decodeStorageKeys(encoded []string) ([]common.Hash, error) {
	if len(encoded) > maxProofKeys {
		return nil, dispatch.InvalidParameters("too many storage keys requested (max %d, got %d)", maxProofKeys, len(encoded))
	}
	keys := make([]common.Hash, len(encoded))
	for index, key := range encoded {
		decoded, err := decodeStorageKey(key)
		if err != nil {
			return nil, dispatch.InvalidParameters("%v: %q", err, key)
		}
		keys[index] = decoded
	}
	return keys, nil
}

// decodeStorageKey parses a hex key of at most 32 bytes, with or without
// 0x and with an odd number of digits, left-padded with zeros.
func decodeStorageKey(encoded string) (common.Hash, error) {
	if strings.HasPrefix(encoded, "0x") || strings.HasPrefix(encoded, "0X") {
		encoded = encoded[2:]
	}
	if len(encoded)%2 == 1 {
		encoded = "0" + encoded
	}
	if len(encoded) > 2*common.HashLength {
		return common.Hash{}, fmt.Errorf("storage key too long (want at most %d bytes)", common.HashLength)
	}
	decoded, err := hex.DecodeString(encoded)
	if err != nil {
		return common.Hash{}, errors.New("invalid hex in storage key")
	}
	return common.BytesToHash(decoded), nil
}

func hexStrings(elements []immutable.Bytes) []string {
	encoded := make([]string, len(elements))
	for index, element := range elements {
		encoded[index] = hexutil.Encode(element.ToBytes())
	}
	return encoded
}
