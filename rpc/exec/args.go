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
	"bytes"
	"context"
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

	"github.com/0xsoniclabs/sonic/rpc/chain"
	"github.com/0xsoniclabs/sonic/rpc/oracle"
	"github.com/0xsoniclabs/sonic/utils"
)

// CallArgs are the arguments of a message call or of a transaction to build.
type CallArgs struct {
	From                 *common.Address `json:"from"`
	To                   *common.Address `json:"to"`
	Gas                  *hexutil.Uint64 `json:"gas"`
	GasPrice             *hexutil.Big    `json:"gasPrice"`
	MaxFeePerGas         *hexutil.Big    `json:"maxFeePerGas"`
	MaxPriorityFeePerGas *hexutil.Big    `json:"maxPriorityFeePerGas"`
	Value                *hexutil.Big    `json:"value"`
	Nonce                *hexutil.Uint64 `json:"nonce"`

	// "data" and "input" are both accepted; "input" is the newer name and is
	// preferred.
	Data  *hexutil.Bytes `json:"data"`
	Input *hexutil.Bytes `json:"input"`

	AccessList *types.AccessList `json:"accessList,omitempty"`
	ChainID    *hexutil.Big      `json:"chainId,omitempty"`

	BlobFeeCap *hexutil.Big  `json:"maxFeePerBlobGas"`
	BlobHashes []common.Hash `json:"blobVersionedHashes,omitempty"`

	Blobs       []kzg4844.Blob       `json:"blobs"`
	Commitments []kzg4844.Commitment `json:"commitments"`
	Proofs      []kzg4844.Proof      `json:"proofs"`

	AuthorizationList []types.SetCodeAuthorization `json:"authorizationList"`
}

// Sender is the sender of the call, the zero address if none is given.
func (args *CallArgs) Sender() common.Address {
	if args.From == nil {
		return common.Address{}
	}
	return *args.From
}

// CallData is the call data; "input" is preferred over "data".
func (args *CallArgs) CallData() []byte {
	if args.Input != nil {
		return *args.Input
	}
	if args.Data != nil {
		return *args.Data
	}
	return nil
}

var errBothFeeKinds = errors.New("both gasPrice and (maxFeePerGas or maxPriorityFeePerGas) specified")

// SetDefaults fills in the fields of a transaction to build that the caller
// left out: fees from the oracle, the pool nonce, the gas from an estimate
// and the chain id. It keeps the reference's defects: the gas estimate sees
// "data" only, not "input".
func (e *Env) SetDefaults(ctx context.Context, head *Head, args *CallArgs) error {
	if args.GasPrice != nil && (args.MaxFeePerGas != nil || args.MaxPriorityFeePerGas != nil) {
		return errBothFeeKinds
	}
	// An EIP-7702 set-code transaction cannot be a legacy transaction.
	if args.GasPrice != nil && args.AuthorizationList != nil {
		return errors.New("both gasPrice and authorizationList specified")
	}
	chainConfig := e.ChainConfig(head.Number())
	if args.MaxPriorityFeePerGas == nil || args.MaxFeePerGas == nil {
		if chainConfig.IsLondon(head.Block.Number) && args.GasPrice == nil {
			if args.MaxPriorityFeePerGas == nil {
				tip, err := e.oracle.SuggestTip(ctx, oracle.AsDefaultCertainty)
				if err != nil {
					return err
				}
				args.MaxPriorityFeePerGas = (*hexutil.Big)(tip)
			}
			if args.MaxFeePerGas == nil {
				feeCap := new(big.Int).Add((*big.Int)(args.MaxPriorityFeePerGas), e.SuggestedGasPrice(head))
				args.MaxFeePerGas = (*hexutil.Big)(feeCap)
			}
			if args.MaxFeePerGas.ToInt().Cmp(args.MaxPriorityFeePerGas.ToInt()) < 0 {
				return fmt.Errorf("maxFeePerGas (%v) < maxPriorityFeePerGas (%v)", args.MaxFeePerGas, args.MaxPriorityFeePerGas)
			}
		} else {
			if args.MaxFeePerGas != nil || args.MaxPriorityFeePerGas != nil {
				return errors.New("maxFeePerGas or maxPriorityFeePerGas specified but london is not active yet")
			}
			if args.GasPrice == nil {
				price, err := e.oracle.SuggestTip(ctx, oracle.AsDefaultCertainty)
				if err != nil {
					return err
				}
				price.Add(price, e.SuggestedGasPrice(head))
				args.GasPrice = (*hexutil.Big)(price)
			}
		}
	} else if args.MaxFeePerGas.ToInt().Cmp(args.MaxPriorityFeePerGas.ToInt()) < 0 {
		return fmt.Errorf("maxFeePerGas (%v) < maxPriorityFeePerGas (%v)", args.MaxFeePerGas, args.MaxPriorityFeePerGas)
	}
	if args.Value == nil {
		args.Value = new(hexutil.Big)
	}
	if args.Nonce == nil {
		nonce := e.core.PoolNonce(args.Sender())
		args.Nonce = (*hexutil.Uint64)(&nonce)
	}
	if args.Data != nil && args.Input != nil && !bytes.Equal(*args.Data, *args.Input) {
		return errors.New(`both "data" and "input" are set and not equal. Please use "input" to pass transaction call data`)
	}
	if args.To == nil {
		// ToTransaction builds a blob transaction if either blob field is set.
		if args.BlobHashes != nil || args.BlobFeeCap != nil {
			return errors.New(`missing "to" in blob transaction`)
		}
		if len(args.CallData()) == 0 {
			return errors.New(`contract creation without any data provided`)
		}
		if args.AuthorizationList != nil {
			return errors.New(`authorizationList provided for contract creation, but "to" field is missing`)
		}
	}
	if args.Gas == nil {
		// The estimate copies "data" and not "input", as the reference does.
		callArgs := CallArgs{
			From:                 args.From,
			To:                   args.To,
			GasPrice:             args.GasPrice,
			MaxFeePerGas:         args.MaxFeePerGas,
			MaxPriorityFeePerGas: args.MaxPriorityFeePerGas,
			Value:                args.Value,
			Data:                 args.Data,
			AccessList:           args.AccessList,
			BlobFeeCap:           args.BlobFeeCap,
			BlobHashes:           args.BlobHashes,
			AuthorizationList:    args.AuthorizationList,
		}
		estimated, err := e.EstimateGas(ctx, head, callArgs, chain.NumberRef(chain.PendingBlockNumber), nil, nil, e.cfg.GasCap, nil)
		if err != nil {
			return err
		}
		args.Gas = &estimated
	}
	want := e.ChainID()
	if args.ChainID != nil {
		if have := (*big.Int)(args.ChainID); have.Cmp(want) != 0 {
			return fmt.Errorf("chainId does not match node's (have=%v, want=%v)", have, want)
		}
	} else {
		args.ChainID = (*hexutil.Big)(want)
	}
	return nil
}

// ToMessage converts the arguments to the message of a call. The gas is
// capped by the gas cap, and by math.MaxInt64 for Tosca; with no base fee
// the message is priced as before EIP-1559.
func (args *CallArgs) ToMessage(gasCap uint64, baseFee *big.Int) (*core.Message, error) {
	if args.GasPrice != nil && (args.MaxFeePerGas != nil || args.MaxPriorityFeePerGas != nil) {
		return nil, errBothFeeKinds
	}
	if gasCap == 0 || gasCap > math.MaxInt64 {
		gasCap = math.MaxInt64
	}
	var nonce uint64
	if args.Nonce != nil {
		nonce = uint64(*args.Nonce)
	}
	gas := gasCap
	if args.Gas != nil {
		if requested := uint64(*args.Gas); requested > gas {
			log.Warn("Caller gas above allowance, capping", "requested", requested, "cap", gas)
		} else {
			gas = requested
		}
	}

	var gasPrice, gasFeeCap, gasTipCap *big.Int
	if baseFee == nil {
		gasPrice = new(big.Int)
		if args.GasPrice != nil {
			gasPrice = args.GasPrice.ToInt()
		}
		gasFeeCap, gasTipCap = gasPrice, gasPrice
	} else if args.GasPrice != nil {
		gasPrice = args.GasPrice.ToInt()
		gasFeeCap, gasTipCap = gasPrice, gasPrice
	} else {
		gasFeeCap = new(big.Int)
		if args.MaxFeePerGas != nil {
			gasFeeCap = args.MaxFeePerGas.ToInt()
		}
		gasTipCap = new(big.Int)
		if args.MaxPriorityFeePerGas != nil {
			gasTipCap = args.MaxPriorityFeePerGas.ToInt()
		}
		// Backfill the legacy gas price for the EVM, unless all are zero.
		gasPrice = new(big.Int)
		if gasFeeCap.BitLen() > 0 || gasTipCap.BitLen() > 0 {
			gasPrice = utils.BigMin(new(big.Int).Add(gasTipCap, baseFee), gasFeeCap)
		}
	}
	value := new(big.Int)
	if args.Value != nil {
		value = args.Value.ToInt()
	}
	var accessList types.AccessList
	if args.AccessList != nil {
		accessList = *args.AccessList
	}
	var blobGasFeeCap *big.Int
	if args.BlobFeeCap != nil {
		blobGasFeeCap = args.BlobFeeCap.ToInt()
	}
	blobHashes := args.BlobHashes
	if len(blobHashes) == 0 {
		blobHashes = nil
	}
	return &core.Message{
		From:                  args.Sender(),
		To:                    args.To,
		Nonce:                 nonce,
		Value:                 value,
		GasLimit:              gas,
		GasPrice:              gasPrice,
		GasFeeCap:             gasFeeCap,
		GasTipCap:             gasTipCap,
		Data:                  args.CallData(),
		AccessList:            accessList,
		BlobGasFeeCap:         blobGasFeeCap,
		BlobHashes:            blobHashes,
		SetCodeAuthorizations: args.AuthorizationList,
		SkipNonceChecks:       true,
		SkipTransactionChecks: true,
	}, nil
}

// ToTransaction converts arguments whose defaults are set to an unsigned
// transaction.
func (args *CallArgs) ToTransaction() (*types.Transaction, error) {
	accessList := func() types.AccessList {
		if args.AccessList != nil {
			return *args.AccessList
		}
		return types.AccessList{}
	}
	var data types.TxData
	switch {
	case len(args.AuthorizationList) > 0:
		chainID, err := utils.BigIntToUint256((*big.Int)(args.ChainID))
		if err != nil {
			return nil, fmt.Errorf("invalid ChainID: %w", err)
		}
		gasFeeCap, err := utils.BigIntToUint256((*big.Int)(args.MaxFeePerGas))
		if err != nil {
			return nil, fmt.Errorf("invalid MaxFeePerGas: %w", err)
		}
		gasTipCap, err := utils.BigIntToUint256((*big.Int)(args.MaxPriorityFeePerGas))
		if err != nil {
			return nil, fmt.Errorf("invalid MaxPriorityFeePerGas: %w", err)
		}
		value, err := utils.BigIntToUint256((*big.Int)(args.Value))
		if err != nil {
			return nil, fmt.Errorf("invalid Value: %w", err)
		}
		if args.To == nil {
			return nil, fmt.Errorf("missing To in SetCodeTx")
		}
		data = &types.SetCodeTx{
			To:         *args.To,
			ChainID:    chainID,
			Nonce:      uint64(*args.Nonce),
			Gas:        uint64(*args.Gas),
			GasFeeCap:  gasFeeCap,
			GasTipCap:  gasTipCap,
			Value:      value,
			Data:       args.CallData(),
			AccessList: accessList(),
			AuthList:   args.AuthorizationList,
		}

	case args.BlobFeeCap != nil || len(args.BlobHashes) > 0:
		chainID, err := utils.BigIntToUint256((*big.Int)(args.ChainID))
		if err != nil {
			return nil, fmt.Errorf("invalid ChainID: %w", err)
		}
		gasFeeCap, err := utils.BigIntToUint256((*big.Int)(args.MaxFeePerGas))
		if err != nil {
			return nil, fmt.Errorf("invalid MaxFeePerGas: %w", err)
		}
		gasTipCap, err := utils.BigIntToUint256((*big.Int)(args.MaxPriorityFeePerGas))
		if err != nil {
			return nil, fmt.Errorf("invalid MaxPriorityFeePerGas: %w", err)
		}
		blobFeeCap, err := utils.BigIntToUint256((*big.Int)(args.BlobFeeCap))
		if err != nil {
			return nil, fmt.Errorf("invalid BlobFeeCap: %w", err)
		}
		value, err := utils.BigIntToUint256((*big.Int)(args.Value))
		if err != nil {
			return nil, fmt.Errorf("invalid Value: %w", err)
		}
		if args.To == nil {
			return nil, fmt.Errorf("missing To in BlobTx")
		}
		data = &types.BlobTx{
			To:         *args.To,
			ChainID:    chainID,
			Nonce:      uint64(*args.Nonce),
			Gas:        uint64(*args.Gas),
			GasFeeCap:  gasFeeCap,
			GasTipCap:  gasTipCap,
			Value:      value,
			Data:       args.CallData(),
			AccessList: accessList(),
			BlobFeeCap: blobFeeCap,
			BlobHashes: args.BlobHashes,
		}

	case args.MaxFeePerGas != nil:
		data = &types.DynamicFeeTx{
			To:         args.To,
			ChainID:    (*big.Int)(args.ChainID),
			Nonce:      uint64(*args.Nonce),
			Gas:        uint64(*args.Gas),
			GasFeeCap:  (*big.Int)(args.MaxFeePerGas),
			GasTipCap:  (*big.Int)(args.MaxPriorityFeePerGas),
			Value:      (*big.Int)(args.Value),
			Data:       args.CallData(),
			AccessList: accessList(),
		}

	case args.AccessList != nil:
		data = &types.AccessListTx{
			To:         args.To,
			ChainID:    (*big.Int)(args.ChainID),
			Nonce:      uint64(*args.Nonce),
			Gas:        uint64(*args.Gas),
			GasPrice:   (*big.Int)(args.GasPrice),
			Value:      (*big.Int)(args.Value),
			Data:       args.CallData(),
			AccessList: *args.AccessList,
		}

	default:
		data = &types.LegacyTx{
			To:       args.To,
			Nonce:    uint64(*args.Nonce),
			Gas:      uint64(*args.Gas),
			GasPrice: (*big.Int)(args.GasPrice),
			Value:    (*big.Int)(args.Value),
			Data:     args.CallData(),
		}
	}
	return types.NewTx(data), nil
}
