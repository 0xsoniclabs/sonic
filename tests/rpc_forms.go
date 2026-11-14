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

package tests

import (
	"encoding/json"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/rpc"
)

// The JSON forms the tests exchange with a node's RPC, as clients write and
// read them, independent of the code that serves them.

// TransactionArgs are the arguments of a call or of a new transaction.
type TransactionArgs struct {
	From                 *common.Address              `json:"from"`
	To                   *common.Address              `json:"to"`
	Gas                  *hexutil.Uint64              `json:"gas"`
	GasPrice             *hexutil.Big                 `json:"gasPrice"`
	MaxFeePerGas         *hexutil.Big                 `json:"maxFeePerGas"`
	MaxPriorityFeePerGas *hexutil.Big                 `json:"maxPriorityFeePerGas"`
	Value                *hexutil.Big                 `json:"value"`
	Nonce                *hexutil.Uint64              `json:"nonce"`
	Data                 *hexutil.Bytes               `json:"data"`
	Input                *hexutil.Bytes               `json:"input"`
	AccessList           *types.AccessList            `json:"accessList,omitempty"`
	ChainID              *hexutil.Big                 `json:"chainId,omitempty"`
	AuthorizationList    []types.SetCodeAuthorization `json:"authorizationList,omitempty"`
}

// ToTransaction is the unsigned transaction of arguments that name their
// gas, nonce and price: a dynamic fee transaction with a fee cap, an access
// list transaction with a list, and a legacy one otherwise.
func (a *TransactionArgs) ToTransaction() (*types.Transaction, error) {
	if a.Gas == nil || a.Nonce == nil {
		return nil, errors.New("the arguments name no gas or nonce")
	}
	data := []byte{}
	switch {
	case a.Input != nil:
		data = *a.Input
	case a.Data != nil:
		data = *a.Data
	}
	switch {
	case a.MaxFeePerGas != nil:
		var accessList types.AccessList
		if a.AccessList != nil {
			accessList = *a.AccessList
		}
		return types.NewTx(&types.DynamicFeeTx{
			ChainID: a.ChainID.ToInt(), Nonce: uint64(*a.Nonce), GasTipCap: a.MaxPriorityFeePerGas.ToInt(),
			GasFeeCap: a.MaxFeePerGas.ToInt(), Gas: uint64(*a.Gas), To: a.To, Value: a.Value.ToInt(), Data: data, AccessList: accessList,
		}), nil
	case a.AccessList != nil:
		return types.NewTx(&types.AccessListTx{
			ChainID: a.ChainID.ToInt(), Nonce: uint64(*a.Nonce), GasPrice: a.GasPrice.ToInt(), Gas: uint64(*a.Gas),
			To: a.To, Value: a.Value.ToInt(), Data: data, AccessList: *a.AccessList,
		}), nil
	}
	return types.NewTx(&types.LegacyTx{
		Nonce: uint64(*a.Nonce), GasPrice: a.GasPrice.ToInt(), Gas: uint64(*a.Gas), To: a.To, Value: a.Value.ToInt(), Data: data,
	}), nil
}

// OverrideAccount replaces parts of an account for a call.
type OverrideAccount struct {
	Nonce     *hexutil.Uint64              `json:"nonce,omitempty"`
	Code      *hexutil.Bytes               `json:"code,omitempty"`
	Balance   *hexutil.U256                `json:"balance,omitempty"`
	State     *map[common.Hash]common.Hash `json:"state,omitempty"`
	StateDiff *map[common.Hash]common.Hash `json:"stateDiff,omitempty"`
}

// StateOverride replaces accounts for a call.
type StateOverride map[common.Address]OverrideAccount

// BlockOverrides replaces fields of the block context of a call.
type BlockOverrides struct {
	Number      *hexutil.Big    `json:"number,omitempty"`
	Difficulty  *hexutil.Big    `json:"difficulty,omitempty"`
	Time        *hexutil.Uint64 `json:"time,omitempty"`
	GasLimit    *hexutil.Uint64 `json:"gasLimit,omitempty"`
	Coinbase    *common.Address `json:"coinbase,omitempty"`
	Random      *common.Hash    `json:"random,omitempty"`
	BaseFee     *hexutil.Big    `json:"baseFee,omitempty"`
	BlobBaseFee *hexutil.Big    `json:"blobBaseFee,omitempty"`
}

// TraceCallConfig is how a traced call runs.
type TraceCallConfig struct {
	tracers.TraceConfig
	StateOverrides *StateOverride  `json:"stateOverrides,omitempty"`
	BlockOverrides *BlockOverrides `json:"blockOverrides,omitempty"`
	TxIndex        *hexutil.Uint   `json:"txIndex,omitempty"`
}

// RPCTransaction is a transaction as the RPC answers it.
type RPCTransaction struct {
	BlockHash           *common.Hash                 `json:"blockHash"`
	BlockNumber         *hexutil.Big                 `json:"blockNumber"`
	From                common.Address               `json:"from"`
	Gas                 hexutil.Uint64               `json:"gas"`
	GasPrice            *hexutil.Big                 `json:"gasPrice"`
	GasFeeCap           *hexutil.Big                 `json:"maxFeePerGas,omitempty"`
	GasTipCap           *hexutil.Big                 `json:"maxPriorityFeePerGas,omitempty"`
	Hash                common.Hash                  `json:"hash"`
	Input               hexutil.Bytes                `json:"input"`
	Nonce               hexutil.Uint64               `json:"nonce"`
	To                  *common.Address              `json:"to"`
	TransactionIndex    *hexutil.Uint64              `json:"transactionIndex"`
	Value               *hexutil.Big                 `json:"value"`
	Type                hexutil.Uint64               `json:"type"`
	Accesses            *types.AccessList            `json:"accessList,omitempty"`
	ChainID             *hexutil.Big                 `json:"chainId,omitempty"`
	V                   *hexutil.Big                 `json:"v"`
	R                   *hexutil.Big                 `json:"r"`
	S                   *hexutil.Big                 `json:"s"`
	MaxFeePerBlobGas    *hexutil.Big                 `json:"maxFeePerBlobGas"`
	BlobVersionedHashes []common.Hash                `json:"blobVersionedHashes"`
	AuthorizationList   []types.SetCodeAuthorization `json:"authorizationList,omitempty"`
}

// AccountResult is an account as eth_getAccount answers it.
type AccountResult struct {
	CodeHash    common.Hash    `json:"codeHash"`
	StorageRoot common.Hash    `json:"storageRoot"`
	Balance     *hexutil.U256  `json:"balance"`
	Nonce       hexutil.Uint64 `json:"nonce"`
}

// BundleInfo tells where the transactions of an executed bundle are.
type BundleInfo struct {
	Block    rpc.BlockNumber `json:"block,omitempty"`
	Position hexutil.Uint    `json:"position,omitempty"`
	Count    hexutil.Uint    `json:"count,omitempty"`
}

// PreparedBundle is a bundle's transactions to sign and its execution plan.
type PreparedBundle struct {
	Transactions  []TransactionArgs `json:"transactions"`
	ExecutionPlan json.RawMessage   `json:"executionPlan"`
}

// SubmittedBundle is a prepared bundle with its transactions signed.
type SubmittedBundle struct {
	SignedTransactions []hexutil.Bytes `json:"signedTransactions"`
	ExecutionPlan      json.RawMessage `json:"executionPlan,omitempty"`
}
