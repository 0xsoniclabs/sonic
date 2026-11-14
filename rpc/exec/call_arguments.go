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

package exec

import (
	"errors"
	"fmt"
	"math"
	"math/big"

	"github.com/0xsoniclabs/sonic/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/log"
	"github.com/holiman/uint256"
)

// CallArguments are the arguments of a call or of a new transaction, as
// clients give them.
type CallArguments struct {
	From                 *common.Address `json:"from"`
	To                   *common.Address `json:"to"`
	Gas                  *hexutil.Uint64 `json:"gas"`
	GasPrice             *hexutil.Big    `json:"gasPrice"`
	MaxFeePerGas         *hexutil.Big    `json:"maxFeePerGas"`
	MaxPriorityFeePerGas *hexutil.Big    `json:"maxPriorityFeePerGas"`
	Value                *hexutil.Big    `json:"value"`
	Nonce                *hexutil.Uint64 `json:"nonce"`

	// Data and Input are both accepted; Input is the newer name and is
	// preferred.
	Data  *hexutil.Bytes `json:"data"`
	Input *hexutil.Bytes `json:"input"`

	AccessList *types.AccessList `json:"accessList,omitempty"`
	ChainID    *hexutil.Big      `json:"chainId,omitempty"`

	BlobFeeCap *hexutil.Big  `json:"maxFeePerBlobGas"`
	BlobHashes []common.Hash `json:"blobVersionedHashes,omitempty"`

	// The sidecar of a blob transaction, decoded and not used: the network
	// carries no blobs.
	Blobs       []kzg4844.Blob       `json:"blobs"`
	Commitments []kzg4844.Commitment `json:"commitments"`
	Proofs      []kzg4844.Proof      `json:"proofs"`

	AuthorizationList []types.SetCodeAuthorization `json:"authorizationList"`
}

var errBothPricings = errors.New("both gasPrice and (maxFeePerGas or maxPriorityFeePerGas) specified")

// Sender is the sender of the call, the zero address if none is given.
func (a *CallArguments) Sender() common.Address {
	if a.From == nil {
		return common.Address{}
	}
	return *a.From
}

// CallData is the call's input, preferring Input to Data.
func (a *CallArguments) CallData() []byte {
	switch {
	case a.Input != nil:
		return *a.Input
	case a.Data != nil:
		return *a.Data
	default:
		return nil
	}
}

// ToMessage turns the arguments into a message of the EVM, its gas capped by
// the gas cap, which Tosca's int64 gas bounds further. Without a base fee the
// message is priced as before EIP-1559, otherwise as after.
func (a *CallArguments) ToMessage(gasCap uint64, baseFee *big.Int) (*core.Message, error) {
	if a.GasPrice != nil && (a.MaxFeePerGas != nil || a.MaxPriorityFeePerGas != nil) {
		return nil, errBothPricings
	}
	if gasCap == 0 || gasCap > math.MaxInt64 {
		gasCap = math.MaxInt64
	}
	var nonce uint64
	if a.Nonce != nil {
		nonce = uint64(*a.Nonce)
	}
	gas := gasCap
	if a.Gas != nil {
		if requested := uint64(*a.Gas); requested > gas {
			log.Warn("Caller gas above allowance, capping", "requested", requested, "cap", gas)
		} else {
			gas = requested
		}
	}
	gasPrice, gasFeeCap, gasTipCap := a.prices(baseFee)
	value := new(big.Int)
	if a.Value != nil {
		value = a.Value.ToInt()
	}
	var accessList types.AccessList
	if a.AccessList != nil {
		accessList = *a.AccessList
	}
	blobHashes := a.BlobHashes
	if len(blobHashes) == 0 {
		blobHashes = nil
	}
	var blobGasFeeCap *big.Int
	if a.BlobFeeCap != nil {
		blobGasFeeCap = a.BlobFeeCap.ToInt()
	} else if blobHashes != nil {
		blobGasFeeCap = new(big.Int)
	}
	return &core.Message{
		From:                  a.Sender(),
		To:                    a.To,
		Nonce:                 nonce,
		Value:                 value,
		GasLimit:              gas,
		GasPrice:              gasPrice,
		GasFeeCap:             gasFeeCap,
		GasTipCap:             gasTipCap,
		Data:                  a.CallData(),
		AccessList:            accessList,
		BlobGasFeeCap:         blobGasFeeCap,
		BlobHashes:            blobHashes,
		SetCodeAuthorizations: a.AuthorizationList,
		SkipNonceChecks:       true,
		SkipTransactionChecks: true,
	}, nil
}

// prices are the gas price, fee cap and tip cap of the message: all the gas
// price before EIP-1559 or when it is given, otherwise the caps given, and a
// gas price backfilled from them unless both are zero.
func (a *CallArguments) prices(baseFee *big.Int) (gasPrice, gasFeeCap, gasTipCap *big.Int) {
	if baseFee == nil || a.GasPrice != nil {
		gasPrice = new(big.Int)
		if a.GasPrice != nil {
			gasPrice = a.GasPrice.ToInt()
		}
		return gasPrice, gasPrice, gasPrice
	}
	gasFeeCap, gasTipCap = new(big.Int), new(big.Int)
	if a.MaxFeePerGas != nil {
		gasFeeCap = a.MaxFeePerGas.ToInt()
	}
	if a.MaxPriorityFeePerGas != nil {
		gasTipCap = a.MaxPriorityFeePerGas.ToInt()
	}
	gasPrice = new(big.Int)
	if gasFeeCap.BitLen() > 0 || gasTipCap.BitLen() > 0 {
		gasPrice = utils.BigMin(new(big.Int).Add(gasTipCap, baseFee), gasFeeCap)
	}
	return gasPrice, gasFeeCap, gasTipCap
}

// ToTransaction turns arguments, whose defaults are filled, into an unsigned
// transaction of the type its fields name: a set-code, blob, dynamic fee,
// access list or legacy transaction.
func (a *CallArguments) ToTransaction() (*types.Transaction, error) {
	accessList := types.AccessList{}
	if a.AccessList != nil {
		accessList = *a.AccessList
	}
	switch {
	case len(a.AuthorizationList) > 0:
		fields, err := a.uint256Fields("SetCodeTx", false)
		if err != nil {
			return nil, err
		}
		return types.NewTx(&types.SetCodeTx{
			To: *a.To, ChainID: fields.chainID, Nonce: uint64(*a.Nonce), Gas: uint64(*a.Gas),
			GasFeeCap: fields.feeCap, GasTipCap: fields.tipCap, Value: fields.value,
			Data: a.CallData(), AccessList: accessList, AuthList: a.AuthorizationList,
		}), nil
	case a.BlobFeeCap != nil || len(a.BlobHashes) > 0:
		fields, err := a.uint256Fields("BlobTx", true)
		if err != nil {
			return nil, err
		}
		return types.NewTx(&types.BlobTx{
			To: *a.To, ChainID: fields.chainID, Nonce: uint64(*a.Nonce), Gas: uint64(*a.Gas),
			GasFeeCap: fields.feeCap, GasTipCap: fields.tipCap, Value: fields.value,
			Data: a.CallData(), AccessList: accessList, BlobFeeCap: fields.blobFeeCap, BlobHashes: a.BlobHashes,
		}), nil
	case a.MaxFeePerGas != nil:
		return types.NewTx(&types.DynamicFeeTx{
			To: a.To, ChainID: a.ChainID.ToInt(), Nonce: uint64(*a.Nonce), Gas: uint64(*a.Gas),
			GasFeeCap: a.MaxFeePerGas.ToInt(), GasTipCap: a.MaxPriorityFeePerGas.ToInt(), Value: a.Value.ToInt(),
			Data: a.CallData(), AccessList: accessList,
		}), nil
	case a.AccessList != nil:
		return types.NewTx(&types.AccessListTx{
			To: a.To, ChainID: a.ChainID.ToInt(), Nonce: uint64(*a.Nonce), Gas: uint64(*a.Gas),
			GasPrice: a.GasPrice.ToInt(), Value: a.Value.ToInt(), Data: a.CallData(), AccessList: *a.AccessList,
		}), nil
	default:
		return types.NewTx(&types.LegacyTx{
			To: a.To, Nonce: uint64(*a.Nonce), Gas: uint64(*a.Gas),
			GasPrice: a.GasPrice.ToInt(), Value: a.Value.ToInt(), Data: a.CallData(),
		}), nil
	}
}

// uint256Fields are the fields a set-code or blob transaction holds as
// 256-bit integers.
type uint256Fields struct {
	chainID, feeCap, tipCap, blobFeeCap, value *uint256.Int
}

// uint256Fields converts the fields a set-code or blob transaction holds as
// 256-bit integers, the blob fee cap with it for a blob transaction, and
// checks its recipient, which such a transaction needs.
func (a *CallArguments) uint256Fields(kind string, withBlobFeeCap bool) (uint256Fields, error) {
	var fields uint256Fields
	var err error
	if fields.chainID, err = utils.BigIntToUint256(a.ChainID.ToInt()); err != nil {
		return fields, fmt.Errorf("invalid ChainID: %w", err)
	}
	if fields.feeCap, err = utils.BigIntToUint256(a.MaxFeePerGas.ToInt()); err != nil {
		return fields, fmt.Errorf("invalid MaxFeePerGas: %w", err)
	}
	if fields.tipCap, err = utils.BigIntToUint256(a.MaxPriorityFeePerGas.ToInt()); err != nil {
		return fields, fmt.Errorf("invalid MaxPriorityFeePerGas: %w", err)
	}
	if withBlobFeeCap {
		if fields.blobFeeCap, err = utils.BigIntToUint256(a.BlobFeeCap.ToInt()); err != nil {
			return fields, fmt.Errorf("invalid BlobFeeCap: %w", err)
		}
	}
	if fields.value, err = utils.BigIntToUint256(a.Value.ToInt()); err != nil {
		return fields, fmt.Errorf("invalid Value: %w", err)
	}
	if a.To == nil {
		return fields, fmt.Errorf("missing To in %s", kind)
	}
	return fields, nil
}
