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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/log"
	"github.com/holiman/uint256"

	"github.com/0xsoniclabs/sonic/utils"
)

// CallArguments are the fields of a transaction or of a message call as a
// client sends them, with the JSON names of go-ethereum's TransactionArgs.
type CallArguments struct {
	From                 *common.Address `json:"from"`
	Recipient            *common.Address `json:"to"`
	Gas                  *hexutil.Uint64 `json:"gas"`
	GasPrice             *hexutil.Big    `json:"gasPrice"`
	MaxFeePerGas         *hexutil.Big    `json:"maxFeePerGas"`
	MaxPriorityFeePerGas *hexutil.Big    `json:"maxPriorityFeePerGas"`
	Value                *hexutil.Big    `json:"value"`
	Nonce                *hexutil.Uint64 `json:"nonce"`

	// Data and Input are both accepted, for backwards compatibility; Input
	// is the newer name and wins.
	Data  *hexutil.Bytes `json:"data"`
	Input *hexutil.Bytes `json:"input"`

	AccessList *types.AccessList `json:"accessList,omitempty"`
	ChainID    *hexutil.Big      `json:"chainId,omitempty"`

	BlobFeeCap  *hexutil.Big         `json:"maxFeePerBlobGas"`
	BlobHashes  []common.Hash        `json:"blobVersionedHashes,omitempty"`
	Blobs       []kzg4844.Blob       `json:"blobs"`
	Commitments []kzg4844.Commitment `json:"commitments"`
	Proofs      []kzg4844.Proof      `json:"proofs"`

	AuthorizationList []types.SetCodeAuthorization `json:"authorizationList"`
}

// Sender is the sender of the call, the zero address if none is given.
func (arguments *CallArguments) Sender() common.Address {
	if arguments.From == nil {
		return common.Address{}
	}
	return *arguments.From
}

// CallData is the data of the call, input when both are given.
func (arguments *CallArguments) CallData() []byte {
	if arguments.Input != nil {
		return *arguments.Input
	}
	if arguments.Data != nil {
		return *arguments.Data
	}
	return nil
}

var errBothPriceStyles = errors.New("both gasPrice and (maxFeePerGas or maxPriorityFeePerGas) specified")

// ToMessage converts the arguments into a message for the EVM, with the gas
// capped at the global cap and the prices of a dynamic-fee transaction when
// a base fee is given. Nonce and transaction checks are skipped, as for a
// call.
func (arguments *CallArguments) ToMessage(globalGasCap uint64, baseFee *big.Int) (*core.Message, error) {
	if arguments.GasPrice != nil && (arguments.MaxFeePerGas != nil || arguments.MaxPriorityFeePerGas != nil) {
		return nil, errBothPriceStyles
	}
	// Tosca counts gas in int64.
	if globalGasCap == 0 || globalGasCap > math.MaxInt64 {
		globalGasCap = math.MaxInt64
	}
	var nonce uint64
	if arguments.Nonce != nil {
		nonce = uint64(*arguments.Nonce)
	}
	gas := globalGasCap
	if arguments.Gas != nil {
		requested := uint64(*arguments.Gas)
		if requested > gas {
			log.Warn("Caller gas above allowance, capping", "requested", requested, "cap", gas)
		} else {
			gas = requested
		}
	}
	gasPrice, gasFeeCap, gasTipCap := arguments.messagePrices(baseFee)
	value := new(big.Int)
	if arguments.Value != nil {
		value = arguments.Value.ToInt()
	}
	var accessList types.AccessList
	if arguments.AccessList != nil {
		accessList = *arguments.AccessList
	}
	var blobGasFeeCap *big.Int
	if arguments.BlobFeeCap != nil {
		blobGasFeeCap = arguments.BlobFeeCap.ToInt()
	}
	blobHashes := arguments.BlobHashes
	if len(blobHashes) == 0 {
		blobHashes = nil
	}
	return &core.Message{
		From:                  arguments.Sender(),
		To:                    arguments.Recipient,
		Nonce:                 nonce,
		Value:                 value,
		GasLimit:              gas,
		GasPrice:              gasPrice,
		GasFeeCap:             gasFeeCap,
		GasTipCap:             gasTipCap,
		Data:                  arguments.CallData(),
		AccessList:            accessList,
		BlobGasFeeCap:         blobGasFeeCap,
		BlobHashes:            blobHashes,
		SetCodeAuthorizations: arguments.AuthorizationList,
		SkipNonceChecks:       true,
		SkipTransactionChecks: true,
	}, nil
}

// messagePrices returns the gas price, fee cap and tip cap of the message.
// Without a base fee the execution is pre-London; with one, a legacy price
// becomes both caps and dynamic fees back-fill the legacy price.
func (arguments *CallArguments) messagePrices(baseFee *big.Int) (gasPrice, gasFeeCap, gasTipCap *big.Int) {
	if baseFee == nil {
		gasPrice = new(big.Int)
		if arguments.GasPrice != nil {
			gasPrice = arguments.GasPrice.ToInt()
		}
		return gasPrice, gasPrice, gasPrice
	}
	if arguments.GasPrice != nil {
		gasPrice = arguments.GasPrice.ToInt()
		return gasPrice, gasPrice, gasPrice
	}
	gasFeeCap = new(big.Int)
	if arguments.MaxFeePerGas != nil {
		gasFeeCap = arguments.MaxFeePerGas.ToInt()
	}
	gasTipCap = new(big.Int)
	if arguments.MaxPriorityFeePerGas != nil {
		gasTipCap = arguments.MaxPriorityFeePerGas.ToInt()
	}
	gasPrice = new(big.Int)
	if gasFeeCap.BitLen() > 0 || gasTipCap.BitLen() > 0 {
		gasPrice = utils.BigMin(new(big.Int).Add(gasTipCap, baseFee), gasFeeCap)
	}
	return gasPrice, gasFeeCap, gasTipCap
}

// ToTransaction converts the arguments, whose defaults are set, into an
// unsigned transaction of the type the fields call for.
func (arguments *CallArguments) ToTransaction() (*types.Transaction, error) {
	switch {
	case len(arguments.AuthorizationList) > 0:
		return arguments.setCodeTransaction()
	case arguments.BlobFeeCap != nil || len(arguments.BlobHashes) > 0:
		return arguments.blobTransaction()
	case arguments.MaxFeePerGas != nil:
		return types.NewTx(&types.DynamicFeeTx{
			To:         arguments.Recipient,
			ChainID:    (*big.Int)(arguments.ChainID),
			Nonce:      uint64(*arguments.Nonce),
			Gas:        uint64(*arguments.Gas),
			GasFeeCap:  (*big.Int)(arguments.MaxFeePerGas),
			GasTipCap:  (*big.Int)(arguments.MaxPriorityFeePerGas),
			Value:      (*big.Int)(arguments.Value),
			Data:       arguments.CallData(),
			AccessList: arguments.accessListOrEmpty(),
		}), nil
	case arguments.AccessList != nil:
		return types.NewTx(&types.AccessListTx{
			To:         arguments.Recipient,
			ChainID:    (*big.Int)(arguments.ChainID),
			Nonce:      uint64(*arguments.Nonce),
			Gas:        uint64(*arguments.Gas),
			GasPrice:   (*big.Int)(arguments.GasPrice),
			Value:      (*big.Int)(arguments.Value),
			Data:       arguments.CallData(),
			AccessList: *arguments.AccessList,
		}), nil
	}
	return types.NewTx(&types.LegacyTx{
		To:       arguments.Recipient,
		Nonce:    uint64(*arguments.Nonce),
		Gas:      uint64(*arguments.Gas),
		GasPrice: (*big.Int)(arguments.GasPrice),
		Value:    (*big.Int)(arguments.Value),
		Data:     arguments.CallData(),
	}), nil
}

func (arguments *CallArguments) accessListOrEmpty() types.AccessList {
	if arguments.AccessList != nil {
		return *arguments.AccessList
	}
	return types.AccessList{}
}

// uint256Fields are the chain id and fee fields of a typed transaction.
type uint256Fields struct {
	chainID, gasFeeCap, gasTipCap *uint256.Int
}

func (arguments *CallArguments) uint256Fields() (uint256Fields, error) {
	var fields uint256Fields
	var err error
	if fields.chainID, err = utils.BigIntToUint256((*big.Int)(arguments.ChainID)); err != nil {
		return fields, fmt.Errorf("invalid ChainID: %w", err)
	}
	if fields.gasFeeCap, err = utils.BigIntToUint256((*big.Int)(arguments.MaxFeePerGas)); err != nil {
		return fields, fmt.Errorf("invalid MaxFeePerGas: %w", err)
	}
	if fields.gasTipCap, err = utils.BigIntToUint256((*big.Int)(arguments.MaxPriorityFeePerGas)); err != nil {
		return fields, fmt.Errorf("invalid MaxPriorityFeePerGas: %w", err)
	}
	return fields, nil
}

func (arguments *CallArguments) value() (*uint256.Int, error) {
	value, err := utils.BigIntToUint256((*big.Int)(arguments.Value))
	if err != nil {
		return nil, fmt.Errorf("invalid Value: %w", err)
	}
	return value, nil
}

func (arguments *CallArguments) setCodeTransaction() (*types.Transaction, error) {
	fields, err := arguments.uint256Fields()
	if err != nil {
		return nil, err
	}
	value, err := arguments.value()
	if err != nil {
		return nil, err
	}
	if arguments.Recipient == nil {
		return nil, fmt.Errorf("missing To in SetCodeTx")
	}
	return types.NewTx(&types.SetCodeTx{
		To:         *arguments.Recipient,
		ChainID:    fields.chainID,
		Nonce:      uint64(*arguments.Nonce),
		Gas:        uint64(*arguments.Gas),
		GasFeeCap:  fields.gasFeeCap,
		GasTipCap:  fields.gasTipCap,
		Value:      value,
		Data:       arguments.CallData(),
		AccessList: arguments.accessListOrEmpty(),
		AuthList:   arguments.AuthorizationList,
	}), nil
}

func (arguments *CallArguments) blobTransaction() (*types.Transaction, error) {
	fields, err := arguments.uint256Fields()
	if err != nil {
		return nil, err
	}
	blobFeeCap, err := utils.BigIntToUint256((*big.Int)(arguments.BlobFeeCap))
	if err != nil {
		return nil, fmt.Errorf("invalid BlobFeeCap: %w", err)
	}
	value, err := arguments.value()
	if err != nil {
		return nil, err
	}
	if arguments.Recipient == nil {
		return nil, fmt.Errorf("missing To in BlobTx")
	}
	return types.NewTx(&types.BlobTx{
		To:         *arguments.Recipient,
		ChainID:    fields.chainID,
		Nonce:      uint64(*arguments.Nonce),
		Gas:        uint64(*arguments.Gas),
		GasFeeCap:  fields.gasFeeCap,
		GasTipCap:  fields.gasTipCap,
		Value:      value,
		Data:       arguments.CallData(),
		AccessList: arguments.accessListOrEmpty(),
		BlobFeeCap: blobFeeCap,
		BlobHashes: arguments.BlobHashes,
	}), nil
}
